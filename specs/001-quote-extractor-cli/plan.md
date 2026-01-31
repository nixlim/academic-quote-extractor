# Implementation Plan: Academic Quote Extractor CLI

**Branch**: `001-quote-extractor-cli` | **Date**: 2026-01-29 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-quote-extractor-cli/spec.md`

## Architecture Diagrams

- **System Architecture**: [architecture-v2.mermaid](../../architecture-v2.mermaid) - Component layout showing Go CLI, Docker services, and data stores
- **Data Flow**: [data-flow-v2.mermaid](../../data-flow-v2.mermaid) - Sequence diagram for ingest/extract/export phases
- **Agent Roles**: [agent-roles.mermaid](../../agent-roles.mermaid) - Conceptual agent responsibilities (implemented as Go packages)

## Summary

Build a Go CLI application (aqe) that extracts relevant quotes from academic documents with Harvard-style citations. The system uses a hybrid RAG architecture: Docling-serve for document parsing, a Python wrapper for hierarchical chunking, Weaviate with local Ollama embeddings for hybrid search, and Claude Code CLI for relevance scoring. SQLite stores verbatim chunk text as the authoritative source, ensuring zero citation hallucination.

## Technical Context

**Language/Version**: Go 1.25.6+, Python 3.11+ (chunking script only)
**Primary Dependencies**:
- github.com/spf13/cobra (CLI framework)
- github.com/weaviate/weaviate-go-client/v4 (Weaviate client)
- github.com/mattn/go-sqlite3 (SQLite driver)
- github.com/stretchr/testify (testing assertions)

**External Services (Docker)**:
- Docling-serve: quay.io/docling-project/docling-serve:latest (port 5001)
- Weaviate: cr.weaviate.io/semitechnologies/weaviate:1.27.0 (port 8080)
- Ollama: ollama/ollama:latest (port 11434) with nomic-embed-text model

**Storage**:
- SQLite: ./quotes.db (default, configurable via --db flag) - verbatim text source of truth
- Weaviate: Vector storage with text2vec-ollama vectorizer for hybrid BM25+vector search

**Testing**: Go standard testing with testify/assert; contract tests for external APIs
**Target Platform**: macOS/Linux CLI (single binary)
**Project Type**: Single project (Go CLI)

**Performance Goals**:
- SC-001: Ingest 100-page PDF in <60 seconds
- SC-002: Extract quotes from 10 documents in <10 seconds

**Constraints**:
- Offline embedding generation (no external API calls for embeddings)
- Zero citation hallucination (chunk IDs only from LLM)
- Single-user local tool

**Scale/Scope**: Individual student use; corpus of 10-100 academic documents

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. Deterministic Quote Retrieval | ✅ PASS | FR-003, FR-005, FR-007: Claude returns chunk IDs only; verbatim text from SQLite |
| II. Hybrid RAG Architecture | ✅ PASS | FR-004, FR-004a: Weaviate BM25+vector; LLM for scoring/explanation only |
| III. Single Purpose Focus | ✅ PASS | Spec limited to quote extraction with citations; no clustering/mapping |
| IV. Go-First Development | ✅ PASS | All core logic in Go; Python only in scripts/chunk_helper.py |
| V. Docker-Based Services | ✅ PASS | Docling, Weaviate, Ollama in Docker; SQLite embedded |

**Code Quality Standards**:
| Requirement | Status | Plan |
|-------------|--------|------|
| CLI command test coverage | ✅ DONE | tests/unit/ for store/harvard; CLI tested via live validation |
| Integration tests with Docker | ✅ DONE | tests/integration/ requires running services |
| Contract tests for APIs | ✅ DONE | tests/contract/ for Docling, Weaviate, Ollama |
| Meaningful error messages | ✅ DONE | FR-012: user vs system error distinction, exit codes |

**Architectural Constraints**:
| Constraint | Status | Evidence |
|------------|--------|----------|
| Max 3 packages: cmd/aqe, internal/*, scripts/ | ✅ PASS | Project structure follows this |
| No custom abstractions over clients | ✅ PASS | Direct use of official Weaviate/SQLite clients |
| Harvard formatting in pure Go | ✅ PASS | internal/harvard/formatter.go |
| Chunk IDs as primary key | ✅ PASS | Chunk.ID is Docling self_ref |

## Project Structure

### Documentation (this feature)

```text
specs/001-quote-extractor-cli/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   ├── docling-api.md
│   ├── weaviate-schema.json
│   └── claude-prompt.md
└── tasks.md             # Phase 2 output (via /speckit.tasks)
```

### Source Code (repository root)

```text
cmd/aqe/
└── main.go              # CLI entry point

internal/
├── cli/                 # Command implementations
│   ├── root.go          # Root command, global flags
│   ├── ingest.go        # aqe ingest command
│   ├── extract.go       # aqe extract command
│   ├── export.go        # aqe export command
│   ├── list.go          # aqe list command
│   └── meta.go          # aqe meta fix command
├── docling/             # Docling HTTP client
│   ├── client.go        # HTTP client for docling-serve
│   └── types.go         # DoclingDocument, TextItem, Provenance
├── chunker/             # Python wrapper interface
│   └── chunker.go       # Calls scripts/chunk_helper.py
├── claude/              # Claude Code CLI wrapper
│   ├── wrapper.go       # Exec wrapper for claude CLI
│   └── prompt.go        # Prompt templates
├── search/              # Weaviate client
│   ├── weaviate.go      # Search operations
│   └── schema.go        # Schema setup
├── store/               # SQLite operations
│   ├── sqlite.go        # Database operations
│   └── migrations.go    # Schema migrations
├── harvard/             # Reference formatter
│   ├── formatter.go     # Citation formatting interface
│   ├── harvard_us.go    # US Harvard style implementation
│   └── types.go         # Reference types (Book, Article, etc.)
└── models/              # Domain types
    ├── document.go      # Document entity
    ├── chunk.go         # Chunk entity
    ├── extraction.go    # Extraction entity
    └── quote.go         # ExtractedQuote entity

scripts/
└── chunk_helper.py      # Python wrapper for Docling HierarchicalChunker

tests/
├── contract/            # API contract tests
│   ├── docling_test.go
│   ├── weaviate_test.go
│   └── ollama_test.go
├── integration/         # End-to-end tests (require Docker)
│   ├── ingest_test.go
│   ├── extract_test.go
│   └── export_test.go
└── unit/                # Unit tests
    ├── harvard_test.go
    ├── store_test.go
    └── cli_test.go

docker-compose.yml       # Docling, Weaviate, Ollama services
go.mod
go.sum
```

**Structure Decision**: Single Go project with standard cmd/internal layout. Python limited to scripts/ per constitution. Tests organized by type (contract/integration/unit).

## Complexity Tracking

> No constitution violations requiring justification.

| Aspect | Decision | Rationale |
|--------|----------|-----------|
| Python wrapper | scripts/chunk_helper.py | Required for HierarchicalChunker; constitution explicitly allows Python in scripts/ |
| Three Docker services | Docling + Weaviate + Ollama | Each serves distinct purpose; constitution requires Docker for Docling/Weaviate |

## Chunking Configuration

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| Chunk size | ~800 tokens | Fits within nomic-embed-text 2K context with room for query |
| Overlap | 200 tokens (25%) | Prevents mid-sentence splits, improves boundary recall |
| Chunker | HierarchicalChunker | Preserves section headings, page numbers, hierarchy |

## Embedding Configuration

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| Model | nomic-embed-text | 768 dims, 2K context, outperforms ada-002, 274MB footprint |
| Vectorizer | text2vec-ollama | Local generation, no API costs |
| Hybrid search | BM25 + vector | Best of keyword and semantic matching |

## Reference Formatting

| Aspect | Decision |
|--------|----------|
| Style | Harvard US (month-day-year, double quotes) |
| DOI/URL | Include when available |
| Multiple authors | 2: "Smith and Jones"; 3+: "Smith et al." |
| In-text | (Author, Year, p. X) |
| Extensibility | Modular interface for future APA/MLA/Chicago |
