# Possible Quality Improvements

Options for improving the number and quality of extracted quotes, ranked by impact.

---

## Implementing Now

### 1. Query Expansion via Claude (High Impact, Low Effort)

**Problem:** The user's topic string goes directly to Weaviate as-is. BM25 keyword
search misses synonyms and related concepts. For example, searching for
"programming languages and software engineering" won't match chunks that say
"coding paradigms" or "development methodology." The vector search side partially
compensates, but `nomic-embed-text` (137M params) won't catch every semantic
relationship.

**Solution:** Before searching Weaviate, make a quick Claude CLI call to expand
the topic into 3-4 search-optimized query variants. Run each as a separate
Weaviate hybrid search, union the results, and deduplicate by chunk ID. This
surfaces candidates that the original query would miss entirely.

**Example:**
```
User topic: "impact of social media on political polarization"

Claude generates:
  query 1: "social media political polarization partisan divide"
  query 2: "online platforms echo chambers political discourse"
  query 3: "digital media political opinion radicalization filter bubbles"
```

**Cost:** ~$0.01 per expansion call (small prompt, short response). 2-3 extra
Weaviate searches are free (local Docker).

**Files changed:**
- `internal/claude/wrapper.go` — New `ExpandQuery()` method
- `internal/claude/prompt.go` — New query expansion prompt template
- `internal/cli/extract.go` — Call `ExpandQuery()` before search, run multiple
  searches, deduplicate results

---

### 2. Batched Scoring (High Impact, Low Effort)

**Problem:** All candidate chunks are sent to Claude in a single prompt. With
~800-token chunks plus ~400 words of adjacent context each, 30 candidates
produces a ~40K token prompt. This hard-caps how many candidates we can evaluate.
Going above ~30 kills the Claude CLI subprocess (OOM/timeout). Additionally,
research on "lost in the middle" effects suggests LLMs score items in the middle
of long contexts less accurately.

**Solution:** Split candidates into batches of ~30 chunks, make one Claude CLI
call per batch, merge results, sort by relevance score.

**Benefits:**
- Raise `--candidates` default to 60-100 without OOM risk
- Smaller batches may improve scoring accuracy per chunk
- Combined with query expansion, evaluates a wider and more diverse candidate pool

**Trade-offs:**
- More Claude API calls = higher cost (~$0.04 per batch)
- Sequential batches add latency (~15-20s per batch)
- Scores aren't directly comparable across batches (different competition context)

**Files changed:**
- `internal/claude/wrapper.go` — New `ExtractQuotesBatched()` method, `splitChunks()` helper
- `internal/cli/extract.go` — Call batched method instead of single, raise `--candidates` default

---

## Future Improvements (Not Implementing Now)

### 3. Alpha Tuning + Fusion Type (Medium Impact, Trivial Effort)

The hybrid search alpha is hardcoded at 0.5 (equal BM25/vector weight). For
academic/conceptual queries, biasing toward vector search (alpha 0.6-0.7) may
improve retrieval. Additionally, switching from the default `rankedFusion` to
`relativeScoreFusion` normalizes scores before combining, which can improve
result ordering.

**Effort:** One-line changes each, but requires experimentation with different
topics to find optimal values. Should expose `--alpha` as a CLI flag.

### 4. Weaviate Reranker Module (Medium Impact, Medium Effort)

Add a cross-encoder reranker module to Weaviate (e.g., `reranker-transformers`)
that re-scores hybrid search results before returning them. This would improve
candidate ordering before Claude sees them.

**Why not now:** Requires changing the Docker setup and pulling another model.
Claude is already doing external reranking (relevance scoring), and doing it
better than a small cross-encoder would. The marginal improvement doesn't justify
the infrastructure complexity.

### 5. BM25 Parameter Tuning (Low Impact, Low Effort)

Weaviate uses default BM25 parameters (k1=1.2, b=0.75). Tuning these for
academic text with ~800-token chunks could marginally improve keyword search
quality. However, the vector search side already compensates for BM25
weaknesses, so gains would be small.

### 6. Document-Level Filtering (Low Impact, Low Effort)

Add a `--document` flag to `aqe extract` to restrict search to specific
documents. The Weaviate `Where` filter on `document_id` already exists for
deletion; it just needs to be wired into `HybridSearch`.

**Why not now:** Doesn't improve quality — just scoping. Useful feature but
orthogonal to quote quality.

---

## Recommended Implementation Order

1. **Batched scoring** first (unblocks higher candidate counts)
2. **Query expansion** second (surfaces better candidates to fill those higher counts)
3. Alpha tuning as a quick follow-up flag
4. Document filtering as a usability feature when needed
