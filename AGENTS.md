# AGENTS.md - Academic Quote Extractor

Instructions for AI coding agents operating in this repository.

## Project Overview

Go CLI application (`aqe`) for extracting quotes from academic documents with Harvard citations.
Uses hybrid RAG: Docling for parsing, Weaviate for search, Claude CLI for relevance scoring.

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

## Docker Services

```bash
# Start all services
docker-compose up -d

# Check service health
curl http://localhost:5001/health  # Docling
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
5. **Docker Services**: Weaviate, Docling, Ollama in Docker. SQLite embedded.

### Package Structure

```
cmd/aqe/          # CLI entry point only
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
- **Architecture**: `architecture-v2.mermaid`, `data-flow-v2.mermaid`

**CRITICAL RULES**

Once implementation has commenced it has to be completed to the end. Leave no ticket unfinished!

Test your own work - run the program, test it with pdf file present in directory:
Culturally-Responsive-Computing-An-Introduction-into-Computer-Science-Security-and-Technology-Updated-122024-1735660702.pdf

And processed file:
output/Culturally-Responsive-Computing-An-Introduction-into-Computer-Science-Security-and-Technology-Updated-122024-1735660702.md

Research Findings - Claude CLI --output-format json:
When using claude --print --output-format json -p "<prompt>", the output is a JSON envelope:
{
  type: result,
  subtype: success,
  is_error: false,
  duration_ms: 1815,
  num_turns: 1,
  result: {"selected_chunks": []},   // <-- STRING, not object
  session_id: ...,
  total_cost_usd: 0.04,
  usage: {...}
}

## KNOWLEDGE AND CODEBASE INSIGHTS

### Research Findings - Claude CLI --output-format json:
When using claude --print --output-format json -p "<prompt>", the output is a JSON envelope:
{
  type: result,
  subtype: success,
  is_error: false,
  duration_ms: 1815,
  num_turns: 1,
  result: {"selected_chunks": []},   // <-- STRING, not object
  session_id: ...,
  total_cost_usd: 0.04,
  usage: {...}
}
Key insight: The result field is a string containing the actual response text. When we ask Claude to return JSON, the inner JSON is string-escaped inside result. The current code tries json.Unmarshal on the whole envelope as ExtractionResponse, fails, then extractJSON grabs the outermost {...} which is the envelope itself — not the inner content.

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
