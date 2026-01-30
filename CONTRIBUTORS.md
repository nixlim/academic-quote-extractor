# Contributors

## Project

Academic Quote Extractor (`aqe`) -- a Go CLI for extracting relevant quotes from
academic documents with Harvard-style citations.

## Authors

- **NiXLiM** -- Project creator, architecture, specification, and implementation

## AI Contributions

This project was developed with substantial AI assistance:

- **Claude Code (Anthropic)** -- Architecture design, Go implementation, test
  scaffolding, Python chunker script, documentation, and code review

All AI-generated code was reviewed, tested against live infrastructure, and
validated by the AI with human oversight.

## How to Contribute

1. Read [DEV_QUICKSTART.md](DEV_QUICKSTART.md) to set up the development environment
2. Review [STATUS.md](STATUS.md) for known issues, incomplete features, and areas
   needing work
3. Check the project [constitution](.specify/memory/constitution.md) for
   non-negotiable design principles
4. Run `./aqe status` to verify your local environment
5. Run `go fmt ./... && go vet ./... && go test ./...` before submitting changes

### Areas Welcoming Contributions

- **Test coverage** -- Many internal packages lack unit tests (see STATUS.md)
- **Configurable service URLs** -- Currently hardcoded to localhost
- **Additional citation styles** -- The `Formatter` interface supports extension
  (APA, MLA, Chicago)
- **Performance validation** -- Benchmarks for ingest and extract operations
- **DOCX and TXT ingestion** -- Less tested than PDF path
- **Error handling** -- Several locations silently discard errors (see STATUS.md)

### Contribution Guidelines

- Follow the code style documented in [AGENTS.md](AGENTS.md) and
  [DEV_QUICKSTART.md](DEV_QUICKSTART.md)
- All Go core logic must stay in Go (Python only in `scripts/`)
- Do not add features outside the scope of quote extraction with citations
- Use table-driven tests with `testify` assertions
- Wrap all errors with context: `fmt.Errorf("failed to X: %w", err)`
- Exit code 1 for user errors, exit code 2 for system errors

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
