# AGENTS.md - Academic Quote Extractor

Instructions for AI coding agents operating in this repository.

## Project Overview

Go CLI application (`aqe`) for extracting quotes from academic documents with Harvard citations.
Uses hybrid RAG: local Docling (Python subprocess) for parsing, Weaviate for search, Claude CLI for relevance scoring.

## Build & Run Commands

```bash
# Build the CLI
go build -o aqe ./cmd/aqe

# Run the CLI
./aqe --help
./aqe ingest ./sources/
./aqe extract "research topic"
./aqe export 1 --format markdown
```

## Test Commands

```bash
# Run all tests
go test ./...

# Run single test by name
go test -v -run TestFunctionName ./internal/store/...

# Run tests in specific package
go test -v ./internal/harvard/...
go test -v ./internal/store/...

# Run with race detection
go test -race ./...

# Run integration tests (requires Docker services)
go test -v -tags=integration ./tests/integration/...

# Run contract tests (requires Docker services)
go test -v -tags=integration ./tests/contract/...

# Run unit tests only
go test -v ./tests/unit/...
```

## Lint & Format Commands

```bash
# Format code
go fmt ./...
gofmt -s -w .

# Vet for common issues
go vet ./...

# Run all checks before commit
go fmt ./... && go vet ./... && go test ./...
```

## Python Pipeline Setup

```bash
# Install Python dependencies for local Docling parsing
pip install -r scripts/requirements.txt

# Test the pipeline
python scripts/process_document.py --file input/sample.pdf --debug
```

## Docker Services

Weaviate and Ollama run in Docker. Docling runs locally as a Python subprocess.

```bash
# Start Docker services (Weaviate + Ollama only)
docker-compose up -d

# Check service health
curl http://localhost:8080/v1/.well-known/ready  # Weaviate

# Pull embedding model (first time setup)
docker exec -it ollama ollama pull nomic-embed-text
```

## Code Style Guidelines

### Imports

Group imports in this order with blank lines between groups:
1. Standard library
2. External dependencies
3. Internal packages

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/spf13/cobra"
    "github.com/weaviate/weaviate-go-client/v4/weaviate"

    "github.com/org/aqe/internal/models"
    "github.com/org/aqe/internal/store"
)
```

### Naming Conventions

- **Packages**: lowercase, single word (`store`, `harvard`, `docling`)
- **Files**: lowercase with underscores (`harvard_us.go`, `chunk_helper.py`)
- **Types**: PascalCase (`Document`, `ExtractedQuote`, `ChunkResult`)
- **Functions**: PascalCase for exported, camelCase for internal
- **Constants**: PascalCase for exported (`SourceTypeBook`), camelCase for internal
- **Interfaces**: end with `-er` when appropriate (`Formatter`, `Chunker`)

### Error Handling

```go
// Always wrap errors with context
if err != nil {
    return fmt.Errorf("failed to parse document %s: %w", filename, err)
}

// Distinguish user errors from system errors
// User error: invalid input, file not found
// System error: service unavailable, database corruption

// Never expose stack traces to CLI users
// Log detailed errors when --debug flag is set
```

### Struct Tags

```go
type Document struct {
    ID       int64  `db:"id" json:"id"`
    Filename string `db:"filename" json:"filename"`
    Authors  []string // JSON marshaled - add helper methods
}
```

### Testing

```go
// Use testify for assertions
func TestFormatAuthors(t *testing.T) {
    assert := assert.New(t)
    
    result := FormatAuthors([]Author{{LastName: "Smith", FirstName: "John"}})
    assert.Equal("Smith, J.", result)
}

// Use table-driven tests for multiple cases
func TestFormatInText(t *testing.T) {
    tests := []struct {
        name     string
        authors  []Author
        year     int
        page     int
        expected string
    }{
        {"single author", []Author{{...}}, 2023, 42, "(Smith, 2023, p. 42)"},
        // more cases...
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

## Architecture Constraints

### Constitution Principles (Non-Negotiable)

1. **Deterministic Quotes**: LLM returns chunk IDs only. Quote text from SQLite, never generated.
2. **Hybrid RAG**: Weaviate for BM25+vector search. LLM only for scoring/explanations.
3. **Single Purpose**: Quote extraction with citations. No clustering/mapping features.
4. **Go-First**: All core logic in Go. Python only in `scripts/` for Docling wrapper.
5. **Service Architecture**: Weaviate and Ollama in Docker. Docling runs as local Python subprocess. SQLite embedded.

### Package Structure

```
cmd/aqe/          # CLI entry point only
internal/         # All application logic
  cli/            # Cobra commands
  docling/        # Local Python subprocess wrapper (processor.go)
  chunker/        # Legacy Python wrapper (deprecated)
  claude/         # Claude CLI wrapper
  search/         # Weaviate client
  store/          # SQLite operations
  harvard/        # Reference formatting (pure Go)
  models/         # Domain types
scripts/          # Python pipeline (process_document.py + lib/)
tests/            # contract/, integration/, unit/
```

### Key Constraints

- Use official clients directly (no abstraction layers over Weaviate/SQLite)
- Chunk IDs are primary keys for quote lookup
- Never store formatted quotes; reconstruct from chunk ID + metadata
- Harvard formatting must be pure Go (no external libraries)
- Exit codes: 0 = success, 1 = user error, 2 = system error

## Claude CLI Integration

**IMPORTANT**: The `-p` flag must be LAST on the command line:

```go
cmd := exec.CommandContext(ctx, "claude",
    "--print",
    "--output-format", "json",
    "-p", prompt,  // -p MUST be last
)
```

## File References

- **Spec**: `specs/001-quote-extractor-cli/spec.md`
- **Plan**: `specs/001-quote-extractor-cli/plan.md`
- **Tasks**: `specs/001-quote-extractor-cli/tasks.md`
- **Constitution**: `.specify/memory/constitution.md`
- **Architecture (design)**: `architecture-v2.mermaid`, `data-flow-v2.mermaid`
- **Architecture (as-built)**: `implementation-architecture.mermaid`, `implemented-flow.mermaid`
- **Status**: `STATUS.md`
- **Knowledge**: `KNOWLEDGE.md` — detailed technical knowledge (Weaviate pipeline, Claude CLI, chunking)
- **Quality improvements**: `POSSIBLE_QUALITY_IMPROVEMENTS.md`

## Project Status

**All 138 tasks complete.** The project is fully implemented and validated.
See `STATUS.md` for known limitations, technical debt, and test coverage gaps.

## KNOWLEDGE AND CODEBASE INSIGHTS

See `KNOWLEDGE.md` for detailed technical knowledge including:
- Complete Weaviate search pipeline analysis (schema, query construction, Docker config)
- Claude CLI JSON envelope format and parsing
- Chunking pipeline details (HybridChunker, 800 tokens, 200-word overlap)
- Query expansion + batched scoring implementation details

**CRITICAL RULES**

Test your own work - run the program, test it with pdf files present in `input/` directory.

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:ca08a54f -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->
