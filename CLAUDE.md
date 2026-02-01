# academic_quote_extractor Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-01-31

## Active Technologies

- Go 1.25.6+, Python 3.11+ (chunking script only) (001-quote-extractor-cli)

## Project Structure

```text
cmd/aqe/          # CLI entry point
internal/         # All application logic
  cli/            # Cobra commands
  docling/        # Local Python subprocess wrapper (processor.go)
  chunker/        # Legacy Python wrapper (deprecated)
  claude/         # Claude CLI wrapper (extraction, query expansion, batched scoring)
  search/         # Weaviate client
  store/          # SQLite operations
  harvard/        # Reference formatting (pure Go)
  models/         # Domain types
scripts/          # Python pipeline (process_document.py + lib/)
tests/            # contract/, integration/, unit/
```

## Commands

```bash
# Build
go build -o aqe ./cmd/aqe

# Test
go test ./...

# Lint & Format
go fmt ./...
go vet ./...

# Run all checks
go fmt ./... && go vet ./... && go test ./...
```

## Code Style

Go 1.25.6+, Python 3.11+ (chunking script only): Follow standard conventions.
See AGENTS.md for detailed import ordering, naming, error handling, and testing guidelines.

## Key References

- **AGENTS.md** — Build/test/lint commands, code style, architecture constraints, session workflow
- **KNOWLEDGE.md** — Detailed technical knowledge (Weaviate search pipeline, Claude CLI integration, chunking pipeline, query expansion + batched scoring)
- **STATUS.md** — Known limitations, technical debt, test coverage gaps
- **POSSIBLE_QUALITY_IMPROVEMENTS.md** — Future improvement options with trade-offs

## Recent Changes

- 001-quote-extractor-cli: All 138 tasks complete. Full validation passed.
- Post-completion: Local Docling pipeline, 800-token chunks, adjacent context, query expansion, batched scoring.

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
