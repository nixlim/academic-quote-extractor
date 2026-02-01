# Knowledge Base

Detailed technical knowledge about the AQE codebase and its components.
This file is the authoritative reference for how subsystems work internally.

---

## Complete Weaviate Search Pipeline Analysis

### 1. Schema Definition (`schema.go`)

**Class name:** `Chunk`

**Vectorizer:** `text2vec-ollama` using the `nomic-embed-text` model, with the API
endpoint passed dynamically (configured as `http://ollama:11434` from the Docker
network perspective).

**Properties and their indexing/vectorization configuration:**

| Property | DataType | Vectorized? | `vectorizePropertyName` | Notes |
|---|---|---|---|---|
| `text` | `text` | **YES** (`skip: false`) | `false` (only the value is embedded, not the property name "text") | The sole property used for vector embedding |
| `chunk_id` | `string` | NO (`skip: true`) | -- | Docling `self_ref`, links to SQLite `chunks.id` |
| `document_id` | `int` | NO (`skip: true`) | -- | SQLite document foreign key |
| `page_num` | `int` | NO (`skip: true`) | -- | Source page number |
| `section_path` | `string[]` | NO (`skip: true`) | -- | Section heading hierarchy |

**Key observations about the schema:**

- No explicit `Tokenization` setting is specified on any property. Weaviate defaults
  apply: `text` datatype defaults to `word` tokenization (lowercased, split on
  whitespace/punctuation), which means the BM25 side of hybrid search operates on
  word-tokenized `text`. The `string` datatype (`chunk_id`) defaults to `field`
  tokenization (exact match).
- No `InvertedIndexConfig` is explicitly set at the class level, so Weaviate uses
  its defaults (BM25 with default k1=1.2, b=0.75).
- No `VectorIndexConfig` overrides -- defaults to HNSW with default settings.
- No explicit distance metric is set -- Weaviate defaults to cosine distance for HNSW.
- There is no `stopwordsConfig` override.

---

### 2. The Search Query (`weaviate.go` - `HybridSearch`)

**Query type:** Hybrid (BM25 + vector)

**Exact query construction (lines 197-213):**

```go
hybrid := w.client.GraphQL().HybridArgumentBuilder().
    WithQuery(query).
    WithAlpha(alpha)

result, err := w.client.GraphQL().Get().
    WithClassName(ChunkClassName).
    WithHybrid(hybrid).
    WithLimit(limit).
    WithFields(fields...).
    Do(ctx)
```

**Alpha value:** `0.5` (hardcoded at the call site in `extract.go`). The
`HybridSearch` method accepts `alpha` as a parameter, but the only caller always
passes `0.5`.

The meaning of alpha=0.5:
- `alpha = 0.0` = pure BM25 (keyword only)
- `alpha = 1.0` = pure vector (semantic only)
- `alpha = 0.5` = **equal weight** between BM25 and vector search

**What properties are searched:**

- The `HybridArgumentBuilder` does **not** call `WithProperties(...)` to restrict
  which properties the BM25 side searches. This means Weaviate's default behavior
  applies: **BM25 searches all `text`-type properties** (which is only `text` in
  this schema, since `chunk_id` is `string` type and `document_id`/`page_num` are
  `int`).
- The vector side uses the vectorized property (only `text`, since all others have
  `skip: true`).
- There is no `WithFusionType(...)` call, so Weaviate defaults to **`rankedFusion`**
  (reciprocal rank fusion).

**What properties are returned (via `ChunkFields()` plus `_additional`):**

1. `text` -- the chunk text
2. `chunk_id` -- the Docling self_ref / SQLite ID
3. `document_id` -- the source document ID
4. `page_num` -- page number
5. `section_path` -- heading hierarchy
6. `_additional { score }` -- the hybrid fusion score

**Filters applied:** None. The `HybridSearch` method does **not** accept or apply
any `Where` filter (e.g., no filtering by document ID). The search is always against
the entire corpus.

Note: `DeleteByDocumentID` uses a `Where` filter on `document_id` with `Equal`
operator, but that is for deletion, not search.

**Limit/candidate count:** Determined by the CLI, passed through to `WithLimit(limit)`.

---

### 3. CLI Orchestration (`extract.go`)

**CLI flags with defaults:**

| Flag | Default | Purpose |
|---|---|---|
| `--max-quotes` | 20 | Max quotes to return after LLM scoring |
| `--min-relevance` | 60 | Minimum relevance score (0-100) from LLM |
| `--candidates` | 60 | Number of candidate chunks to retrieve per search query |
| `--no-expand` | false | Disable query expansion |

**Search limit calculation:**

```go
searchLimit := candidateLimit           // default: 60
if maxQuotes > searchLimit/2 {
    searchLimit = maxQuotes * 2         // ensure at least 2x candidates vs desired quotes
}
```

So:
- Default: `candidateLimit=60`, `maxQuotes=20`. `20 > 60/2=30` is false, so
  `searchLimit = 60`.
- If `--candidates 100 --max-quotes 20`: `20 > 100/2=50` is false, so
  `searchLimit = 100`.
- If `--candidates 30 --max-quotes 30`: `30 > 30/2=15` is true, so
  `searchLimit = 30 * 2 = 60`.

The intent is that Weaviate always returns at least `2x maxQuotes` candidates so the
LLM has enough to score and filter.

**Query expansion (enabled by default):**

Before searching, Claude generates 3 alternative search queries with different
vocabulary/synonyms. Each query runs as a separate Weaviate hybrid search. Results
are deduplicated by chunk ID. Disable with `--no-expand`.

**The actual call (per query):**

```go
results, err := weaviateClient.HybridSearch(ctx, query, searchLimit, 0.5)
```

- `query` is either the original topic or an expanded variant.
- Alpha is hardcoded at `0.5`.

**Post-search pipeline:**

1. For each Weaviate result, the adjacent chunks (prev/next) are fetched from
   **SQLite** (not Weaviate) using position data, truncated to 200 words each.
2. The chunks + adjacent context are split into batches of 30 and sent to Claude CLI
   for relevance scoring (one call per batch).
3. Claude returns scored chunks with explanations from each batch.
4. Results are merged, sorted by relevance score descending.
5. Chunks that don't exist in SQLite are filtered out (handles stale Weaviate data).
6. Results are filtered by `--min-relevance` threshold and capped at `--max-quotes`.
7. Display with Harvard citations, save extraction to SQLite.

---

### 4. Docker Compose Configuration (`docker-compose.yml`)

**Weaviate image:** `cr.weaviate.io/semitechnologies/weaviate:1.27.0`

**Weaviate environment variables:**

| Variable | Value | Effect |
|---|---|---|
| `QUERY_DEFAULTS_LIMIT` | `25` | Default query limit when none specified (overridden by explicit `WithLimit` in all search calls) |
| `AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED` | `true` | No auth required |
| `PERSISTENCE_DATA_PATH` | `/var/lib/weaviate` | Persistent storage path |
| `DEFAULT_VECTORIZER_MODULE` | `text2vec-ollama` | Default vectorizer for new classes |
| `ENABLE_MODULES` | `text2vec-ollama` | Only this module is enabled (no generative modules, no rerankers) |
| `OLLAMA_API_ENDPOINT` | `http://ollama:11434` | Ollama endpoint for vectorization (Docker-internal DNS) |
| `CLUSTER_HOSTNAME` | `node1` | Single-node cluster |

**Ports exposed:**
- `8080:8080` -- REST/GraphQL API
- `50051:50051` -- gRPC API

**Ollama image:** `ollama/ollama:latest` on port `11434:11434`.

**Notable:** There are no Weaviate environment variables for:
- `QUERY_MAXIMUM_RESULTS` (no upper cap override)
- Any reranker module
- Any generative module
- Custom BM25 tuning (k1, b)
- Vector index configuration overrides

---

### 5. Summary of the Complete Search Pipeline

```
User types: aqe extract "impact of social media on political polarization"
                              |
                              v
              Claude expands topic into 3 alternative queries
              (disable with --no-expand)
                              |
                              v
              4 queries total (original + 3 expanded)
                              |
                              v
              For each query:
                Weaviate HybridSearch(query, limit=60, alpha=0.5)
                  - BM25 side: word-tokenized search on "text" property only
                  - Vector side: nomic-embed-text embedding of query vs text embeddings
                  - Fusion: rankedFusion (default, not explicitly set)
                  - No filters (searches entire corpus)
                  - No property restrictions on BM25
                              |
                              v
              Deduplicate results by chunk_id across all queries
              (typically 100-150 unique candidates from 4 x 60 results)
                              |
                              v
              For each result, fetch adjacent chunks from SQLite
                (prev + next, truncated to 200 words each)
                              |
                              v
              Split into batches of 30, score each batch with Claude CLI
                              |
                              v
              Merge batch results, sort by relevance descending
                              |
                              v
              Filter out stale chunks not in SQLite
                              |
                              v
              Filter: relevance >= 60 (default), cap at 20 quotes (default)
                              |
                              v
              Display with Harvard citations, save extraction to SQLite
```

**Key design decisions / limitations to note:**

1. **Alpha is not user-configurable** -- hardcoded at 0.5. The method signature
   accepts it as a parameter, but no CLI flag exposes it.
2. **No document-level filtering** -- you cannot search within a specific document.
   Every search hits the full corpus.
3. **No reranker module** -- Weaviate only has `text2vec-ollama` enabled. The
   reranking is done externally by Claude.
4. **Default BM25 parameters** -- k1=1.2, b=0.75 (Weaviate defaults, not overridden).
5. **Default fusion algorithm** -- `rankedFusion` (reciprocal rank fusion), not
   `relativeScoreFusion`, since `WithFusionType` is not called.
6. **Only `text` is vectorized** -- metadata fields (`section_path`, `chunk_id`, etc.)
   contribute nothing to either BM25 or vector search.
7. **Embedding model:** `nomic-embed-text` via Ollama -- a 137M parameter model
   producing 768-dimensional embeddings.

---

## Claude CLI Integration

When using `claude --print --output-format json -p "<prompt>"`, the output is a JSON envelope:

```json
{
  "type": "result",
  "subtype": "success",
  "is_error": false,
  "duration_ms": 1815,
  "num_turns": 1,
  "result": "{\"selected_chunks\": []}",
  "session_id": "...",
  "total_cost_usd": 0.04,
  "usage": {}
}
```

**Key insight:** The `result` field is a **string** containing the actual response
text. When we ask Claude to return JSON, the inner JSON is string-escaped inside
`result`. The code unwraps this envelope in `wrapper.go` before parsing the inner
content.

**IMPORTANT:** The `-p` flag must be **LAST** on the command line:

```go
cmd := exec.CommandContext(ctx, w.cliPath,
    "--print",
    "--output-format", "json",
    "-p", prompt,  // -p MUST be last
)
```

---

## Chunking Pipeline

The local Docling pipeline (`scripts/process_document.py`) uses `HybridChunker` with
800-token max and 200-word overlap between adjacent chunks. Position tracking enables
adjacent context lookup during extraction.

**Pipeline flow:**

```
PDF/DOCX/TXT → Docling DocumentConverter → HybridChunker (800 tokens) → 
Overlap (200 words) → Position assignment → SQLite + Weaviate storage
```

**Key metrics (3 test PDFs):**
- Old pipeline (Docker Docling + `chunk_helper.py`): ~138 chars/chunk, 16,660 chunks
- New pipeline (local Docling + HybridChunker): ~800 tokens/chunk, 830 chunks (95% reduction)

---

## Query Expansion + Batched Scoring

`aqe extract` uses two quality improvements:

1. **Query expansion**: Before searching, Claude generates 3 alternative search
   queries with different vocabulary/synonyms. Each query runs as a separate Weaviate
   hybrid search, results are deduplicated by chunk ID. Disable with `--no-expand`.
   Cost: ~$0.01 per expansion call.

2. **Batched scoring**: Candidates are split into batches of 30 and scored separately,
   then merged and sorted by relevance. This removes the OOM ceiling on candidate count.

**Combined effect:** 4x more unique candidates evaluated (126 vs 30), significantly
higher top relevance scores (92 vs 78 in testing), and better coverage across the corpus.
