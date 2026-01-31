<!--
SYNC IMPACT REPORT
==================
Version change: 1.0.0 → 1.1.0
Modified principles: V. Docker-Based Services → V. Service Architecture
Rationale: Docling-serve in Docker requires 16GB RAM and is slower than local Python
  Docling (1.8GB, already installed). Local subprocess replaces HTTP API for parsing.
Migration plan:
  - Replace internal/docling HTTP client usage with local subprocess wrapper
  - Remove Docling from docker-compose.yml
  - Keep HTTP client code marked deprecated for reference
  - Weaviate and Ollama remain in Docker (unchanged)
Added sections: N/A
Removed sections: N/A
Templates requiring updates: None (no principle-specific refs in templates)
Follow-up TODOs: None
-->

# Academic Quote Extractor Constitution

## Core Principles

### I. Deterministic Quote Retrieval

The system MUST never allow the LLM to generate quote text. Claude outputs chunk IDs only;
the system replaces these IDs with verbatim text from the database. This eliminates citation
hallucination entirely.

**Non-negotiable rules:**
- Quote text MUST be retrieved from SQLite by chunk ID, never generated
- LLM responses containing quote text MUST be rejected at the application layer
- Chunk IDs serve as the single source of truth for all quote content

**Rationale:** Academic integrity depends on accurate citations. LLM hallucination of quotes
is unacceptable in scholarly work.

### II. Hybrid RAG Architecture

Use semantic search (BM25 + vector) for retrieval; LLM only for relevance scoring and
explanation generation. Optimize for cost-effectiveness over maximum accuracy.

**Non-negotiable rules:**
- Retrieval MUST use hybrid BM25 + vector search via Weaviate
- LLM usage MUST be limited to: (1) relevance scoring of retrieved chunks, (2) generating
  explanations for why a quote is relevant
- System MUST NOT use LLM for embedding generation when cheaper alternatives exist

**Rationale:** Balancing retrieval quality with operational costs ensures the tool remains
practical for regular academic use.

### III. Single Purpose Focus

Extract quotes with Harvard references and relevance explanations. No theme clustering,
no argument mapping, no contradiction detection.

**Non-negotiable rules:**
- Feature requests for theme clustering MUST be rejected
- Feature requests for argument mapping MUST be rejected
- Feature requests for contradiction detection MUST be rejected
- Every feature MUST directly serve quote extraction with citations

**Rationale:** Scope creep destroys projects. A focused tool that does one thing well
outperforms a bloated tool that does many things poorly.

### IV. Go-First Development

All core logic MUST be implemented in Go. Python is permitted only for the Docling chunking
wrapper. Minimize external dependencies.

**Non-negotiable rules:**
- CLI, retrieval logic, reference formatting, and database operations MUST be in Go
- Python usage MUST be limited to `scripts/` directory for Docling integration only
- New dependencies MUST be justified; prefer standard library solutions

**Rationale:** Go provides a single binary deployment, strong typing, and excellent
concurrency—ideal for CLI tools and services.

### V. Service Architecture

Weaviate and Ollama run in Docker. Docling runs as a local Python subprocess.
SQLite is embedded for persistence.

**Non-negotiable rules:**
- Weaviate MUST run as a Docker container (not embedded)
- Docling MUST run as a local Python subprocess (not Docker) for parsing and chunking
- Ollama MUST run as a Docker container for embedding generation
- SQLite MUST be used for persistent storage (no external database servers)
- Docker Compose MUST be provided for Weaviate and Ollama setup

**Rationale:** Weaviate and Ollama benefit from Docker isolation for their complex
dependencies. Docling as a local subprocess uses 1.8GB RAM (vs 16GB in Docker),
starts instantly, and avoids HTTP serialization overhead. SQLite keeps the core
application simple and portable.

## Code Quality Standards

**Test coverage requirements:**
- All CLI commands MUST have test coverage
- Integration tests MUST use real Docker services (Weaviate, Docling-serve)
- Contract tests MUST exist for all external APIs (Docling, Weaviate, Anthropic)

**Error handling requirements:**
- All errors MUST include meaningful messages suitable for CLI users
- Errors MUST distinguish between user errors (invalid input) and system errors (service
  unavailable)
- Stack traces MUST NOT be shown to users in production mode

**Code review gates:**
- All changes MUST pass `go test ./...` before merge
- All changes MUST pass `go vet` and configured linters
- Integration tests MAY be skipped for pure refactoring PRs (with justification)

## Architectural Constraints & Output Requirements

**Package structure:**
- Maximum 3 main packages: `cmd/aqe`, `internal/*`, `scripts/`
- No custom abstractions over Weaviate or SQLite clients (use official clients directly)
- Harvard reference formatting MUST be implemented in pure Go (no external libraries)

**Data model constraints:**
- Chunk IDs MUST be the primary key for quote lookup
- Formatted quotes MUST NOT be stored; always reconstruct from chunk ID + metadata
- Document metadata (author, year, title) MUST be stored separately from chunk content

**Output format requirements:**
- JSON output MUST be valid and parseable by standard tools (`jq`, etc.)
- Markdown output MUST render correctly in common viewers (GitHub, VS Code, Obsidian)
- In-text citations MUST follow strict Harvard format: `(Author, Year, p. X)`
- Full references MUST be complete and verifiable (no placeholder data)

## Governance

This constitution supersedes all other development practices for the Academic Quote
Extractor project.

**Amendment procedure:**
1. Proposed amendments MUST be documented with rationale
2. Amendments MUST include a migration plan for existing code
3. Version MUST be incremented according to semantic versioning:
   - MAJOR: Principle removal or backward-incompatible redefinition
   - MINOR: New principle added or existing principle materially expanded
   - PATCH: Clarifications, wording fixes, non-semantic refinements

**Compliance:**
- All PRs MUST verify compliance with relevant principles
- Complexity that violates principles MUST be justified in the PR description
- Unjustified violations MUST block merge

**Version**: 1.1.0 | **Ratified**: 2026-01-29 | **Last Amended**: 2026-01-31
