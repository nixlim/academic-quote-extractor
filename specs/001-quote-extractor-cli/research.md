# Research: Academic Quote Extractor CLI

**Feature**: 001-quote-extractor-cli
**Date**: 2026-01-29
**Status**: Complete

## 1. Document Parsing: Docling vs GROBID

### Decision: Docling-serve

### Rationale
Docling provides superior multi-format support and built-in chunking capabilities essential for academic quote extraction.

### Comparison

| Capability | Docling | GROBID |
|------------|---------|--------|
| Format Support | PDF, DOCX, PPTX, XLSX, HTML, images | PDF only |
| Output Format | Unified DoclingDocument → JSON/MD | TEI-XML |
| Chunking | Built-in HierarchicalChunker | Manual |
| Page/BBox Metadata | Native | Native |
| Go Integration | REST API (docling-serve) | REST API |

### Alternatives Considered
- **GROBID**: PDF-only limitation disqualifies it; students use DOCX frequently
- **PyMuPDF + custom chunking**: More work, less metadata preservation
- **LlamaIndex document loaders**: Adds Python dependency to core; violates Go-first principle

### Implementation Notes
- Use docling-serve Docker image on port 5001
- Call `/v1/convert/file` endpoint for document conversion
- Chunking via Python wrapper (scripts/chunk_helper.py) using HierarchicalChunker

---

## 2. Chunking Strategy: HierarchicalChunker

### Decision: Python wrapper with HierarchicalChunker, ~800 tokens, 25% overlap

### Rationale
HierarchicalChunker preserves document structure (sections, subsections, paragraphs) and maintains page/bbox metadata critical for accurate citations.

### Configuration

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| Target chunk size | ~800 tokens | Fits in 2K embedding context with query room |
| Overlap | 200 tokens (25%) | Prevents mid-sentence splits |
| Heading preservation | Yes | Required for section context in quotes |
| Page number tracking | Yes | Required for Harvard citations |

### Alternatives Considered
- **Fixed-size chunking**: Loses semantic boundaries, splits sentences
- **Sentence-based chunking**: Too granular for academic passages
- **Paragraph-based**: Inconsistent sizes, some paragraphs too long

### Implementation Notes
```python
# scripts/chunk_helper.py
from docling.chunking import HierarchicalChunker

chunker = HierarchicalChunker(
    max_tokens=800,
    overlap_tokens=200,
    include_metadata=True
)
```

---

## 3. Embedding Model: nomic-embed-text

### Decision: nomic-embed-text via Ollama (local)

### Rationale
Optimal balance of context length (2K tokens), quality (outperforms OpenAI ada-002), and local execution (no API costs, offline capable).

### Comparison

| Model | Size | Context | Dimensions | Quality | Speed |
|-------|------|---------|------------|---------|-------|
| **nomic-embed-text** | 274MB | 2K | 768 | High | Fast |
| mxbai-embed-large | 670MB | 512 | 1024 | SOTA | Medium |
| snowflake-arctic-embed:335m | 669MB | 512 | - | High | Medium |
| OpenAI text-embedding-3-small | API | 8K | 1536 | High | Fast |

### Alternatives Considered
- **mxbai-embed-large**: Higher quality but 512 token limit truncates academic chunks
- **OpenAI embeddings**: Violates offline operation requirement; API costs
- **snowflake-arctic-embed**: 512 context too limiting

### Implementation Notes
- Weaviate text2vec-ollama module auto-generates embeddings on insert
- No explicit embedding calls needed in Go code
- Pull model: `docker exec ollama ollama pull nomic-embed-text`

---

## 4. Vector Database: Weaviate

### Decision: Weaviate with hybrid search (BM25 + vector)

### Rationale
Native Go client, built-in hybrid search, seamless Ollama integration via text2vec-ollama module.

### Configuration

| Setting | Value |
|---------|-------|
| Class name | Chunk |
| Vectorizer | text2vec-ollama |
| Vector index | HNSW (default) |
| Hybrid alpha | 0.5 (balanced BM25/vector) |

### Alternatives Considered
- **Qdrant**: Good Go client but no built-in BM25 hybrid
- **Milvus**: More complex setup, overkill for single-user
- **Chroma**: Python-native, violates Go-first principle
- **pgvector**: Requires PostgreSQL; SQLite preferred per constitution

### Implementation Notes
- Weaviate schema created on first ingest
- Hybrid search with configurable alpha for BM25/vector balance
- Store chunk_id, document_id, page_num, section_path as properties

---

## 5. Persistence: SQLite

### Decision: SQLite for all persistent data

### Rationale
Constitution mandates SQLite for simplicity. Verbatim text stored here as source of truth (not in Weaviate) to ensure deterministic quote retrieval.

### Schema Design

```sql
-- Core tables
documents (id, filename, filepath, title, authors, year, publisher, source_type, checksum, ingested_at)
chunks (id, document_id, text, page_num, section_path, bbox, embedding_id)
extractions (id, topic, created_at)
extracted_quotes (id, extraction_id, chunk_id, relevance_score, explanation)
```

### Alternatives Considered
- **Dolt**: Version control overkill for single-user CLI
- **PostgreSQL**: External server; violates embedded requirement
- **BadgerDB**: Key-value only; need relational queries

### Implementation Notes
- Use github.com/mattn/go-sqlite3 (CGO required)
- Migrations in internal/store/migrations.go
- Default path: ./quotes.db (configurable via --db flag)

---

## 6. LLM Integration: Claude Code CLI

### Decision: Wrap claude CLI binary for relevance scoring

### Rationale
Constitution requires deterministic quote retrieval (LLM returns chunk IDs only). Claude Code CLI provides structured output parsing.

### Integration Pattern

```go
// internal/claude/wrapper.go
type ExtractionResult struct {
    SelectedChunks []struct {
        ChunkID     string `json:"chunk_id"`
        Relevance   int    `json:"relevance"`
        Explanation string `json:"explanation"`
    } `json:"selected_chunks"`
}
```

### Prompt Structure
1. System: "You are an academic research assistant"
2. User: Topic + chunks with IDs
3. Output: JSON with chunk_id references only (never quote text)

### Alternatives Considered
- **Direct Anthropic API**: More control but adds dependency
- **Local LLM (Ollama)**: Quality may be insufficient for relevance scoring
- **OpenAI API**: Constitution doesn't prohibit, but Claude CLI already available

### Implementation Notes
- Strict output validation: reject responses containing quote text
- Parse JSON response for chunk IDs
- Lookup verbatim text from SQLite (never from LLM response)

---

## 7. Reference Formatting: Harvard US Style

### Decision: Harvard US style with modular architecture

### Rationale
US style requested in clarifications. Modular design allows future addition of APA, MLA, Chicago.

### Format Rules

| Type | Full Reference Format |
|------|----------------------|
| Book | Author, A. (Year) *Title*. Place: Publisher. |
| Journal | Author, A. (Year) "Title," *Journal*, Vol(Issue), pp. X-Y. |
| Website | Author, A. (Year) "Title," *Site*. Available at: URL (Accessed: Date). |
| Chapter | Author, A. (Year) "Chapter," in Editor (ed.) *Book*. Place: Publisher, pp. X-Y. |

| In-text | Format |
|---------|--------|
| Single author | (Smith, 2023, p. 42) |
| Two authors | (Smith and Jones, 2023, p. 42) |
| 3+ authors | (Smith et al., 2023, p. 42) |

### Implementation Notes
- Interface: `Formatter` with `FormatFull()` and `FormatInText()` methods
- US date format: "January 15, 2024"
- Include DOI/URL when available in reference

---

## 8. CLI Framework: Cobra

### Decision: github.com/spf13/cobra

### Rationale
De facto standard for Go CLIs. Supports subcommands, flags, help generation, shell completion.

### Command Structure

```
aqe
├── ingest <path> [--title] [--author] [--year]
├── extract <topic> [--max-quotes] [--min-relevance]
├── export <id> [--format] [--output]
├── meta
│   └── fix
└── [global flags: --db]
```

### Alternatives Considered
- **urfave/cli**: Good but Cobra more widely used
- **Standard flag package**: Insufficient for subcommands
- **Kong**: Less ecosystem support

---

## 9. Testing Strategy

### Decision: Three-tier testing (unit, integration, contract)

### Test Distribution

| Type | Location | Dependencies | Purpose |
|------|----------|--------------|---------|
| Unit | tests/unit/ | None | Harvard formatting, model logic |
| Contract | tests/contract/ | Docker services | API compatibility |
| Integration | tests/integration/ | All services | End-to-end workflows |

### Contract Tests
- Docling: Verify /v1/convert/file response structure
- Weaviate: Verify schema creation, hybrid search
- Ollama: Verify embedding generation

### Implementation Notes
- Use testify/assert for assertions
- Integration tests tagged with `//go:build integration`
- CI runs unit tests always; integration tests require Docker

---

## 10. Error Handling Strategy

### Decision: Structured errors with user/system distinction

### Error Categories

| Category | Exit Code | Example |
|----------|-----------|---------|
| User error | 1 | Invalid file path, unsupported format |
| System error | 2 | Service unavailable, database corruption |
| Success | 0 | Operation completed |

### Implementation Notes
- Wrap errors with context: `fmt.Errorf("ingest: %w", err)`
- User-facing messages via stderr
- No stack traces in production (use --debug flag for verbose output)
