# Project Status

Implementation status, known limitations, and areas needing work.

**All 138 original tasks completed.** Post-completion improvement (Phases A-E) also done:
local Docling pipeline, larger chunks (~800 tokens), adjacent context for LLM scoring.

---

## Architecture (Current)

| Component | How It Runs | Notes |
|-----------|------------|-------|
| **Docling** | Local Python subprocess | `scripts/process_document.py` via `internal/docling/processor.go` |
| **Chunking** | Local Python (HybridChunker) | ~800 tokens/chunk with 200-word overlap |
| **Weaviate** | Docker container | BM25 + vector search via `nomic-embed-text` |
| **Ollama** | Docker container | Hosts the embedding model |
| **SQLite** | Embedded (CGO) | Schema v2: chunks table has `position` column |
| **Claude CLI** | Local subprocess | Relevance scoring with adjacent context |

### Chunking Improvements

Old pipeline: Docker Docling + separate `chunk_helper.py` = ~138 chars/chunk average, 16,660 chunks for 3 PDFs.

New pipeline: Local Docling + HybridChunker (800 tokens) + 200-word overlap = ~800 tokens/chunk, **830 chunks for 3 PDFs** (95% reduction).

---

## What Works (verified by live CLI testing)

| Feature | Status | Notes |
|---------|--------|-------|
| `aqe ingest` (PDF) | Working | Local Docling pipeline, ~800-token chunks |
| `aqe ingest` (Markdown) | Working | Falls back to text items when Python chunker unavailable |
| `aqe ingest` duplicate detection | Working | SHA-256 checksum, skips with message |
| `aqe ingest` metadata overrides | Working | `--title`, `--author`, `--year` flags |
| `aqe ingest` batch resume | Working | Re-run skips already-ingested files |
| `aqe extract` | Working | Hybrid search + Claude scoring with adjacent context |
| `aqe extract` filtering | Working | `--max-quotes`, `--min-relevance`, `--candidates` flags |
| `aqe export` markdown | Working | Blockquotes, in-text citations, bibliography |
| `aqe export` JSON | Working | Valid parseable JSON with full metadata |
| `aqe export` BibTeX | Working | Standard BibTeX with citation keys |
| `aqe export` to file | Working | `--output` flag writes file |
| `aqe list` | Working | Shows IDs, topics, quote counts, dates |
| `aqe meta fix` | Working | Interactive prompts for missing fields |
| `aqe status` | Working | All health checks, DB stats, Docker status |
| Error handling | Working | Exit code 1 for user errors, descriptive messages |
| `--debug` flag | Working | Full pipeline trace output |
| Harvard US citations | Working | In-text and full reference formatting |

## What Has NOT Been Tested

These features exist in code but were not exercised during live testing:

| Feature | Why Untested |
|---------|-------------|
| `aqe ingest` with DOCX files | No DOCX test file available |
| `aqe ingest` with directory of mixed formats | Timeout on large directory scan |
| `aqe ingest` with unsupported file types | Not exercised (code exists at `ingest.go:294`) |
| `aqe extract` on empty corpus | Not triggered (corpus already populated) |
| `aqe extract` with no relevant results | Not triggered (topic was relevant) |
| `aqe version` command | Does not exist (referenced in `root.go:40` but never implemented) |
| Multiple-author Harvard formatting | Only single-author tested in live run |
| Harvard Chapter formatting | `FormatChapter()` exists but no chapter-type documents ingested |
| Harvard Website formatting with access dates | No website-type documents ingested |

---

## Completed Validation (Phase 7)

All validation and performance tasks (T129-T138) have been completed:

| Task | Description | Result |
|------|-------------|--------|
| T129 | Quickstart scenarios 1-3 (ingestion) | Pass -- basic ingest, duplicate detection, batch resume |
| T130 | Quickstart scenarios 4-5 (extraction) | Pass -- 19 quotes default, 5 with custom params |
| T131 | Quickstart scenarios 6-8 (export) | Pass -- Markdown, JSON (valid), BibTeX |
| T132 | Quickstart scenario 9 (metadata fix) | Pass -- interactive prompts, completion detection |
| T133 | Quickstart scenario 10 (error handling) | Pass -- user-friendly messages, correct exit codes |
| T134 | SC-003: verbatim quotes | Pass -- all text from SQLite, never LLM-generated |
| T135 | SC-004: Harvard references | Pass -- format verified manually |
| T136 | FR-006b: Formatter extensibility | Pass -- clean interface, factory pattern |
| T137 | Performance: ingest | Bottleneck is Docling parsing, Go code is fast |
| T138 | Performance: extract | ~17s for 3 docs/830 chunks; Claude API is bottleneck |

---

## Hardcoded Values

Service URLs and operational parameters are hardcoded throughout the CLI
commands. These should be configurable via flags or environment variables.

### Service URLs

| File | Line | Value |
|------|------|-------|
| `internal/cli/ingest.go` | ~91 | `localhost:8080` (Weaviate) |
| `internal/cli/ingest.go` | ~91 | `http://ollama:11434` (Ollama -- Docker internal URL) |
| `internal/cli/extract.go` | ~73 | `localhost:8080` (Weaviate) |
| `internal/cli/extract.go` | ~73 | `http://ollama:11434` (Ollama) |
| `internal/cli/status.go` | ~105 | `http://localhost:8080/...` |
| `internal/cli/status.go` | ~123 | `http://localhost:11434/...` |

### Operational Parameters

| File | Line | Value | Description |
|------|------|-------|-------------|
| `internal/cli/extract.go` | ~51 | `30` | Default candidate limit |
| `internal/cli/extract.go` | ~85 | `50` | Weaviate candidate limit |
| `internal/cli/extract.go` | ~90 | `0.5` | Hybrid search alpha |
| `internal/cli/extract.go` | ~119 | `120 * time.Second` | Claude timeout |
| `internal/cli/status.go` | ~145 | `5 * time.Second` | Health check timeout |

---

## Silently Discarded Errors

These locations ignore error return values. The code continues without
logging or returning the error.

| File | Line | What's Ignored |
|------|------|----------------|
| `internal/cli/extract.go` | ~227 | `chunk.UnmarshalSectionPath()` error |
| `internal/cli/extract.go` | ~230 | `chunk.UnmarshalBBox()` error |
| `internal/cli/extract.go` | ~243 | `doc.UnmarshalAuthors()` error |
| `internal/cli/export.go` | ~199 | `chunk.UnmarshalSectionPath()` error |
| `internal/cli/export.go` | ~202 | `chunk.UnmarshalBBox()` error |
| `internal/cli/export.go` | ~215 | `doc.UnmarshalAuthors()` error |
| `internal/cli/meta.go` | ~134 | `getDocumentsWithIncompleteMetadata()` error |
| `internal/cli/meta.go` | ~180 | `doc.UnmarshalAuthors()` error |
| `internal/cli/ingest.go` | ~128 | `db.GetDocumentByChecksum()` error in resume check |
| `internal/cli/ingest.go` | ~183 | `filepath.Abs()` error |

---

## Store Layer Bypass

The `internal/store/sqlite.go` defines proper CRUD methods, but CLI commands
bypass them with raw SQL via `db.DB().QueryRow(...)` and `db.DB().Exec(...)`.
This duplicates query logic and misses any validation in the store methods.

| CLI File | Bypassed Store Method |
|----------|----------------------|
| `extract.go` `getChunkWithDocument()` | `Store.GetChunkByID()` + `Store.GetDocumentByID()` |
| `extract.go` `saveExtraction()` | `Store.InsertExtraction()` + `Store.InsertExtractedQuotes()` |
| `export.go` `getExtraction()` | `Store.GetExtractionByID()` |
| `export.go` `listExtractions()` | `Store.ListExtractions()` |
| `export.go` `getQuotesWithDetails()` | `Store.GetQuotesByExtractionID()` |
| `list.go` `runList()` | `Store.ListExtractions()` |
| `meta.go` `getDocumentsWithIncompleteMetadata()` | `Store.GetDocumentsWithIncompleteMetadata()` |
| `meta.go` `updateDocumentMetadata()` | `Store.UpdateDocumentMetadata()` |

---

## Dead / Unused Code

Exported symbols that are never called from non-test code:

| File | Symbol | Issue |
|------|--------|-------|
| `internal/cli/root.go` | `ExitSuccess` | Defined but never referenced |
| `internal/cli/root.go` | `UserError()` | Defined but never called |
| `internal/cli/root.go` | `SystemError()` | Defined but never called |
| `internal/models/document.go` | `ValidSourceTypes()` | Never called |
| `internal/models/document.go` | `IsValidSourceType()` | Never called |
| `internal/docling/types.go` | `ConvertResponse` (exported) | Unused; unexported `convertResponse` is used |
| `internal/docling/types.go` | `HealthResponse` | Never used |
| `internal/claude/prompt.go` | `BuildPromptJSON()` | Never called |
| `internal/harvard/formatter.go` | `FormatReference()` | Interface method, alias for `FormatFull()` |
| `internal/store/sqlite.go` | `GetDocumentCount()` | Never called |

### Deprecated Code (kept for reference)

| File | Notes |
|------|-------|
| `internal/docling/client.go` | Old Docker HTTP client — replaced by `processor.go` |
| `internal/chunker/chunker.go` | Old Python chunker wrapper — replaced by `scripts/lib/` |
| `scripts/chunk_helper.py` | Old chunking script — replaced by `scripts/process_document.py` |

---

## Test Coverage Gaps

### Packages with zero unit tests

| Package | What's untested |
|---------|----------------|
| `internal/cli/` | All command logic: `runIngest`, `runExtract`, `runExport`, `runList`, `runMetaFix`, `runStatus`, and all helper functions |
| `internal/docling/` | `Client.Health()`, `Client.ConvertFile()`, `Processor.ProcessFile()`, type parsing |
| `internal/chunker/` | `NewChunker()`, `ChunkDocument()`, `findPython()` |
| `internal/claude/` | `NewWrapper()`, `ExtractQuotes()`, `buildPrompt()`, `extractJSON()` |
| `internal/search/` | All `WeaviateClient` methods, `parseChunkResults()` |

These packages have **contract tests** (require Docker) and **integration tests**
(require Docker + Claude CLI), but no isolated unit tests.

### Specific untested functions in partially-tested packages

| File | Function |
|------|----------|
| `internal/harvard/types.go` | `ParseAuthorString()` |
| `internal/harvard/types.go` | `ParseAuthorsString()` |
| `internal/harvard/harvard_us.go` | `FormatChapter()` |
| `internal/store/sqlite.go` | `ListDocumentsWithChunkCounts()` |
| `internal/store/sqlite.go` | `GetChunkByID()` |
| `internal/store/sqlite.go` | `GetChunksByIDs()` |
| `internal/store/sqlite.go` | `GetAdjacentChunks()` |
| `internal/models/chunk.go` | All Chunk marshal/unmarshal methods |

### Integration tests that won't compile

| File | Issue |
|------|-------|
| `tests/integration/ingest_test.go:47` | `chunker.NewChunker()` called with no arguments; actual signature requires `scriptPath string` |
| `tests/integration/ingest_test.go:115` | `ChunkDocument(ctx, doclingDoc)` passes `*DoclingDocument` but method expects `[]byte` |

---

## Missing Feature: `aqe version`

`internal/cli/root.go` checks `cmd.Name() == "version"` to skip database
initialization, but no `version` subcommand is registered. Running
`./aqe version` produces "unknown command".

---

## Silent Degradation Paths

These situations silently degrade functionality without user-visible warnings
(only visible with `--debug`):

1. **Python chunker unavailable** -- Falls back to raw text items from Docling.
   Chunks will lack hierarchical section paths and may have worse overlap
   properties. (`ingest.go`)

2. **Weaviate chunk insertion failure** -- Individual chunk insert errors are
   swallowed; the chunk will exist in SQLite but have no embedding in Weaviate,
   making it unsearchable. (`ingest.go`)

3. **Unknown Harvard formatter style** -- `NewFormatter("anything")` silently
   returns `HarvardUS{}` with no error or warning. (`formatter.go`)

---

## Undocumented Behaviour

- `.md` (Markdown) is accepted as an ingest format (`ingest.go`), but the
  spec (`spec.md`) only mentions PDF, DOCX, and TXT.

- Source type detection (`ingest.go`) is based on filename keyword
  matching ("article", "journal", "chapter", "web", "online"), which is
  fragile and undocumented to users.

- The Ollama URL `http://ollama:11434` used in `ingest.go` and
  `extract.go` is a Docker internal hostname. This works when Weaviate
  calls Ollama inside Docker, but the CLI itself connects to Weaviate at
  `localhost:8080`. The Ollama URL is passed to Weaviate's vectorizer config,
  not called directly by the Go code.
