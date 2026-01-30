# Developer Quickstart

Set up the development environment, understand the architecture, and start contributing.

## Prerequisites

| Requirement | Version | Purpose |
|-------------|---------|---------|
| Go | 1.25+ | Core application language |
| Docker + Compose | 20.10+ / 2.0+ | Docling, Weaviate, Ollama services |
| Python 3 | 3.9+ | Docling chunker wrapper |
| C compiler | Any | CGO required for go-sqlite3 |
| Claude CLI | Latest | Relevance scoring during extraction |

### Install Python dependencies

```bash
pip3 install "docling>=2.70.0" "docling-core>=2.0.0"
```

### Install C compiler (for SQLite CGO)

- **macOS**: `xcode-select --install`
- **Linux (Debian/Ubuntu)**: `sudo apt-get install gcc build-essential`

## Environment Setup

### 1. Clone and build

```bash
git clone <repository-url>
cd academic_quote_extractor
go build -o aqe ./cmd/aqe
```

### 2. Start Docker services

```bash
docker-compose up -d
```

This starts three containers:

| Service | Port | Purpose |
|---------|------|---------|
| Docling | 5001 | Document parsing (PDF, DOCX, TXT) |
| Weaviate | 8080, 50051 | Vector DB with hybrid BM25 + semantic search |
| Ollama | 11434 | Local embedding model (nomic-embed-text) |

### 3. Pull the embedding model (first time only)

```bash
docker exec -it ollama ollama pull nomic-embed-text
```

### 4. Verify services

```bash
./aqe status
```

**Expected output:**

```
AQE Infrastructure Status
============================================================

  Docling      [OK] http://localhost:5001/health  (27ms)
               status: ok
  Weaviate     [OK] http://localhost:8080/v1/.well-known/ready  (4ms)
               v1.27.0
  Ollama       [OK] http://localhost:11434/api/version  (11ms)
               v0.15.2
  Embeddings   [OK] nomic-embed-text
               model loaded (137M, 262MB)
  Claude CLI   [OK] claude
               2.1.25 (Claude Code)

Docker Containers
------------------------------------------------------------
docling     Up 29 minutes (healthy)     0.0.0.0:5001->5001/tcp
weaviate    Up 45 minutes               0.0.0.0:8080->8080/tcp
ollama      Up 45 minutes               0.0.0.0:11434->11434/tcp

Database
------------------------------------------------------------
  Path:        ./quotes.db
  Documents:   0
  Chunks:      0
  Extractions: 0
  Quotes:      0
```

All services should show `[OK]`. If any show `[FAIL]`, check that `docker-compose up -d` completed and containers are running.

Or check manually:

```bash
curl http://localhost:5001/health
curl http://localhost:8080/v1/.well-known/ready
curl http://localhost:11434/api/version
```

## Project Structure

```
cmd/aqe/main.go           Entry point (minimal -- delegates to cli package)

internal/
  cli/
    root.go                Root command, global flags (--db, --debug)
    ingest.go              aqe ingest -- document parsing and indexing
    extract.go             aqe extract -- topic-based quote extraction
    export.go              aqe export -- format and output results
    list.go                aqe list -- show saved extractions
    meta.go                aqe meta fix -- interactive metadata correction
    status.go              aqe status -- infrastructure health checks

  docling/
    client.go              HTTP client for docling-serve (POST /v1/convert)
    types.go               DoclingDocument, TextItem, Provenance types

  chunker/
    chunker.go             Python wrapper -- pipes JSON to chunk_helper.py

  claude/
    wrapper.go             Claude CLI exec wrapper + JSON envelope parsing
    prompt.go              Go text/template prompt templates

  search/
    weaviate.go            Weaviate client: insert, hybrid search, delete
    schema.go              "Chunk" class definition with text2vec-ollama

  store/
    sqlite.go              All SQLite CRUD (documents, chunks, extractions, quotes)
    migrations.go          DDL schema: 4 tables, 5 indexes

  harvard/
    formatter.go           Formatter interface + factory pattern
    harvard_us.go          US Harvard style: Book, Article, Website, Chapter
    types.go               Author struct, FormatAuthors, ParseAuthorString

  models/
    document.go            Document struct, SourceType enum
    chunk.go               Chunk struct, BBox, JSON marshal helpers
    extraction.go          Extraction struct
    quote.go               ExtractedQuote struct

scripts/
  chunk_helper.py          Python HierarchicalChunker (stdin JSON -> stdout JSONL)
  requirements.txt         docling>=2.70.0, docling-core>=2.0.0

tests/
  unit/                    No Docker required
  contract/                Require Docker (build tag: integration)
  integration/             Require Docker (build tag: integration)
```

## Architecture

### Data Flow

**Ingest phase:**
```
PDF/DOCX/TXT -> Docling (HTTP) -> DoclingDocument JSON
  -> Python HierarchicalChunker (stdin/stdout)
  -> Weaviate (chunks + auto-embeddings via nomic-embed-text)
  -> SQLite (verbatim text, metadata, document records)
```

**Extract phase:**
```
Topic string -> Weaviate hybrid search (top 50, alpha=0.5)
  -> Claude CLI (chunk text + topic -> chunk IDs + relevance scores)
  -> SQLite lookup (verbatim text by chunk ID)
  -> Harvard formatting -> Save extraction to SQLite
```

**Export phase:**
```
Extraction ID -> SQLite (extraction + quotes + chunks + documents)
  -> Format as Markdown / JSON / BibTeX -> stdout or file
```

### Key Design Decisions

1. **Zero hallucination**: Claude returns chunk IDs only. Quote text always comes from SQLite. The application layer rejects any LLM-generated quote text.

2. **Hybrid search**: Weaviate combines BM25 keyword matching with vector similarity (nomic-embed-text, 768 dimensions). Alpha=0.5 balances both approaches.

3. **Chunking fallback**: If the Python chunker fails, the system falls back to Docling's raw text items as chunks.

4. **Claude CLI JSON envelope**: The `--output-format json` flag wraps output in an envelope where `result` is a string, not an object. The wrapper parses the outer envelope first, then extracts the inner JSON from the `result` string.

### Database Schema

```sql
documents (id INTEGER PK, filename, filepath, title, authors JSON,
           year, publisher, source_type CHECK, checksum UNIQUE, ingested_at)

chunks (id TEXT PK, document_id FK, text, page_num, section_path JSON,
        bbox JSON, embedding_id)

extractions (id INTEGER PK, topic, created_at)

extracted_quotes (id INTEGER PK, extraction_id FK, chunk_id FK,
                  relevance_score CHECK 0-100, explanation)
```

### Weaviate Schema

Class `Chunk` with properties:
- `text` (vectorized via text2vec-ollama)
- `chunk_id`, `document_id`, `page_num`, `section_path` (not vectorized)

## Smoke-Test the Full Pipeline

After verifying services, run through the pipeline to confirm everything works:

```bash
# Ingest the test PDF
./aqe ingest "Culturally-Responsive-Computing-An-Introduction-into-Computer-Science-Security-and-Technology-Updated-122024-1735660702.pdf" \
  --title "Culturally Responsive Computing: An Introduction into Computer Science, Security, and Technology" \
  --author "Walton, Devan J." \
  --year 2024
# => Processing: Culturally-Responsive-Computing-...pdf
# => Ingested 1 documents, 2920 chunks

# Extract quotes
./aqe extract "cultural bias in technology and algorithms" --max-quotes 3
# => Searching for relevant quotes...
# => Found 50 candidate chunks
# => Scoring relevance with Claude...
# =>
# => Extraction #1: "cultural bias in technology and algorithms"
# => Retrieved 3 quotes (relevance >= 60)
# => ...
# => Saved as extraction #1. Export with: aqe export 1

# List and export
./aqe list
# => Saved Extractions:
# =>
# =>   #1: "cultural bias in technology and algorithms"
# =>       3 quotes | 2026-01-30
# =>       Export: aqe export 1

./aqe export 1 --format markdown
# => (markdown output to stdout with blockquotes, citations, bibliography)
```

## Running Tests

### Unit tests (no Docker required)

```bash
go test ./...
go test -v ./tests/unit/...
```

### Contract tests (require Docker services)

```bash
go test -v -tags=integration ./tests/contract/...
```

Tests API contracts for:
- Docling health, PDF/DOCX conversion
- Weaviate schema creation, chunk insert, hybrid search
- Ollama health, model availability, embeddings
- Claude CLI availability, output format, response validation

### Integration tests (require Docker services)

```bash
go test -v -tags=integration ./tests/integration/...
```

End-to-end tests for:
- Single PDF ingest, duplicate detection
- Extract with matches, extract with no matches
- Markdown export, JSON export

### Run a specific test

```bash
go test -v -run TestFormatAuthors ./tests/unit/...
go test -v -run TestDoclingHealth -tags=integration ./tests/contract/...
```

### Race detection

```bash
go test -race ./...
```

### All checks (pre-commit)

```bash
go fmt ./... && go vet ./... && go test ./...
```

## Lint and Format

```bash
# Format all Go code
go fmt ./...
gofmt -s -w .

# Static analysis
go vet ./...
```

## Code Style

### Import ordering

Group imports with blank lines between groups:

```go
import (
    "context"
    "fmt"

    "github.com/spf13/cobra"
    "github.com/weaviate/weaviate-go-client/v4/weaviate"

    "github.com/foundry-zero/aqe/internal/models"
    "github.com/foundry-zero/aqe/internal/store"
)
```

### Error handling

Always wrap errors with context:

```go
if err != nil {
    return fmt.Errorf("failed to parse document %s: %w", filename, err)
}
```

Distinguish user errors (exit code 1) from system errors (exit code 2). Never expose stack traces to CLI users.

### Testing conventions

Use testify for assertions and table-driven tests:

```go
func TestFormatInText(t *testing.T) {
    tests := []struct {
        name     string
        authors  []Author
        year     int
        page     int
        expected string
    }{
        {"single author", []Author{{LastName: "Smith"}}, 2023, 42, "(Smith, 2023, p. 42)"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            assert := assert.New(t)
            result := FormatInText(tt.authors, tt.year, tt.page)
            assert.Equal(tt.expected, result)
        })
    }
}
```

## Constitution (Non-Negotiable)

These principles govern all development decisions:

1. **Deterministic Quote Retrieval** -- LLM outputs chunk IDs only; verbatim text from SQLite
2. **Hybrid RAG Architecture** -- Weaviate for search; LLM only for scoring/explanation
3. **Single Purpose Focus** -- Quote extraction with citations only. No clustering, argument mapping, or contradiction detection.
4. **Go-First Development** -- Python only in `scripts/` for Docling wrapper
5. **Docker-Based Services** -- Weaviate and Docling in Docker; SQLite embedded

See `.specify/memory/constitution.md` for full details.

## Key Files Reference

| File | Purpose |
|------|---------|
| `specs/001-quote-extractor-cli/spec.md` | Feature specification (user stories, FRs) |
| `specs/001-quote-extractor-cli/plan.md` | Implementation plan |
| `specs/001-quote-extractor-cli/tasks.md` | Task breakdown (138 tasks) |
| `.specify/memory/constitution.md` | Project constitution |
| `architecture-v2.mermaid` | System architecture diagram |
| `data-flow-v2.mermaid` | Data flow sequence diagram |
| `AGENTS.md` | AI agent instructions |
