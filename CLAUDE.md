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
