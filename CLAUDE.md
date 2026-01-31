# academic_quote_extractor Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-01-31

## Active Technologies

- Go 1.25.6+, Python 3.11+ (chunking script only) (001-quote-extractor-cli)

## Project Structure

```text
cmd/aqe/          # CLI entry point
internal/         # All application logic
  cli/            # Cobra commands
  docling/        # HTTP client for docling-serve
  chunker/        # Python wrapper for HierarchicalChunker
  claude/         # Claude CLI wrapper
  search/         # Weaviate client
  store/          # SQLite operations
  harvard/        # Reference formatting (pure Go)
  models/         # Domain types
scripts/          # Python chunking script only
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

## Recent Changes

- 001-quote-extractor-cli: All 138 tasks complete. Full validation passed.

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
