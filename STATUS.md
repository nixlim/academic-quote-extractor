# Project Status

Implementation status, known limitations, and areas needing work.

**128 of 138 tasks completed.** The 10 remaining tasks (T129-T138) are all
validation and performance testing in Phase 7.

---

## What Works (verified by live CLI testing)

| Feature | Status | Notes |
|---------|--------|-------|
| `aqe ingest` (PDF) | Working | Tested with 100+ page PDF, produced 2920 chunks |
| `aqe ingest` (Markdown) | Working | Falls back to text items when Python chunker unavailable |
| `aqe ingest` duplicate detection | Working | SHA-256 checksum, skips with message |
| `aqe ingest` metadata overrides | Working | `--title`, `--author`, `--year` flags |
| `aqe ingest` batch resume | Working | Re-run skips already-ingested files |
| `aqe extract` | Working | Hybrid search + Claude scoring, saves extraction |
| `aqe extract` filtering | Working | `--max-quotes` and `--min-relevance` applied correctly |
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
| Python HierarchicalChunker with token/overlap config | Chunker script uses defaults (see below) |
| Multiple-author Harvard formatting | Only single-author tested in live run |
| Harvard Chapter formatting | `FormatChapter()` exists but no chapter-type documents ingested |
| Harvard Website formatting with access dates | No website-type documents ingested |

---

## Incomplete Tasks (from tasks.md)

All in Phase 7 -- Quality & Validation:

| Task | Description |
|------|-------------|
| T129 | Validate quickstart scenarios 1-3 (ingestion) including SC-008 usability |
| T130 | Validate quickstart scenarios 4-5 (extraction) |
| T131 | Validate quickstart scenarios 6-8 (export) |
| T132 | Validate quickstart scenario 9 (metadata fix) |
| T133 | Validate quickstart scenario 10 (error handling) |
| T134 | Verify SC-003: all output quotes match verbatim source text (zero hallucination) |
| T135 | Verify SC-004: Harvard references pass manual verification |
| T136 | Verify FR-006b: Formatter interface supports adding new citation styles |
| T137 | Performance test: ingest 100-page PDF in <60s (SC-001) |
| T138 | Performance test: extract from 10 docs in <10s (SC-002) |

---

## Hardcoded Values

Service URLs and operational parameters are hardcoded throughout the CLI
commands. These should be configurable via flags or environment variables.

### Service URLs (9 locations)

| File | Line | Value |
|------|------|-------|
| `internal/cli/ingest.go` | 72 | `http://localhost:5001` (Docling) |
| `internal/cli/ingest.go` | 91 | `localhost:8080` (Weaviate) |
| `internal/cli/ingest.go` | 91 | `http://ollama:11434` (Ollama -- Docker internal URL) |
| `internal/cli/extract.go` | 73 | `localhost:8080` (Weaviate) |
| `internal/cli/extract.go` | 73 | `http://ollama:11434` (Ollama) |
| `internal/cli/status.go` | 94 | `http://localhost:5001/health` |
| `internal/cli/status.go` | 105 | `http://localhost:8080/...` |
| `internal/cli/status.go` | 123 | `http://localhost:11434/...` |
| `internal/cli/status.go` | 179, 325 | additional localhost URLs |

### Operational Parameters

| File | Line | Value | Description |
|------|------|-------|-------------|
| `internal/docling/client.go` | 28 | `10 * time.Minute` | Docling HTTP timeout |
| `internal/cli/extract.go` | 85 | `50` | Weaviate candidate limit |
| `internal/cli/extract.go` | 90 | `0.5` | Hybrid search alpha |
| `internal/cli/extract.go` | 119 | `120 * time.Second` | Claude timeout |
| `internal/cli/status.go` | 145 | `5 * time.Second` | Health check timeout |

---

## Silently Discarded Errors

These locations ignore error return values. The code continues without
logging or returning the error.

| File | Line | What's Ignored |
|------|------|----------------|
| `internal/cli/extract.go` | 227 | `chunk.UnmarshalSectionPath()` error |
| `internal/cli/extract.go` | 230 | `chunk.UnmarshalBBox()` error |
| `internal/cli/extract.go` | 243 | `doc.UnmarshalAuthors()` error |
| `internal/cli/export.go` | 199 | `chunk.UnmarshalSectionPath()` error |
| `internal/cli/export.go` | 202 | `chunk.UnmarshalBBox()` error |
| `internal/cli/export.go` | 215 | `doc.UnmarshalAuthors()` error |
| `internal/cli/meta.go` | 134 | `getDocumentsWithIncompleteMetadata()` error |
| `internal/cli/meta.go` | 180 | `doc.UnmarshalAuthors()` error |
| `internal/cli/ingest.go` | 128 | `db.GetDocumentByChecksum()` error in resume check |
| `internal/cli/ingest.go` | 183 | `filepath.Abs()` error |

---

## Store Layer Bypass

The `internal/store/sqlite.go` defines proper CRUD methods, but CLI commands
bypass them with raw SQL via `db.DB().QueryRow(...)` and `db.DB().Exec(...)`.
This duplicates query logic and misses any validation in the store methods.

| CLI File | Bypassed Store Method |
|----------|----------------------|
| `extract.go:204` `getChunkWithDocument()` | `Store.GetChunkByID()` + `Store.GetDocumentByID()` |
| `extract.go:252` `saveExtraction()` | `Store.InsertExtraction()` + `Store.InsertExtractedQuotes()` |
| `export.go:121` `getExtraction()` | `Store.GetExtractionByID()` |
| `export.go:136` `listExtractions()` | `Store.ListExtractions()` |
| `export.go:162` `getQuotesWithDetails()` | `Store.GetQuotesByExtractionID()` |
| `list.go:28` `runList()` | `Store.ListExtractions()` |
| `meta.go:144` `getDocumentsWithIncompleteMetadata()` | `Store.GetDocumentsWithIncompleteMetadata()` |
| `meta.go:189` `updateDocumentMetadata()` | `Store.UpdateDocumentMetadata()` |

---

## Dead / Unused Code

Exported symbols that are never called from non-test code:

| File | Symbol | Issue |
|------|--------|-------|
| `internal/cli/root.go:21` | `ExitSuccess` | Defined but never referenced |
| `internal/cli/root.go:89` | `UserError()` | Defined but never called |
| `internal/cli/root.go:95` | `SystemError()` | Defined but never called |
| `internal/models/document.go:55` | `ValidSourceTypes()` | Never called |
| `internal/models/document.go:66` | `IsValidSourceType()` | Never called |
| `internal/docling/types.go:84` | `ConvertResponse` (exported) | Unused; unexported `convertResponse` is used |
| `internal/docling/types.go:91` | `HealthResponse` | Never used |
| `internal/claude/prompt.go:71` | `BuildPromptJSON()` | Never called |
| `internal/harvard/formatter.go:38` | `FormatReference()` | Interface method, alias for `FormatFull()` |
| `internal/store/sqlite.go:293` | `GetDocumentCount()` | Never called |

---

## Test Coverage Gaps

### Packages with zero unit tests

| Package | What's untested |
|---------|----------------|
| `internal/cli/` | All command logic: `runIngest`, `runExtract`, `runExport`, `runList`, `runMetaFix`, `runStatus`, and all helper functions |
| `internal/docling/` | `Client.Health()`, `Client.ConvertFile()`, type parsing |
| `internal/chunker/` | `NewChunker()`, `ChunkDocument()`, `findPython()` |
| `internal/claude/` | `NewWrapper()`, `ExtractQuotes()`, `buildPrompt()`, `extractJSON()` |
| `internal/search/` | All `WeaviateClient` methods, `parseChunkResults()` |

These packages have **contract tests** (require Docker) and **integration tests**
(require Docker + Claude CLI), but no isolated unit tests.

### Specific untested functions in partially-tested packages

| File | Function |
|------|----------|
| `internal/harvard/types.go:75` | `ParseAuthorString()` |
| `internal/harvard/types.go:88` | `ParseAuthorsString()` |
| `internal/harvard/harvard_us.go:156` | `FormatChapter()` |
| `internal/store/sqlite.go:63` | `ListDocumentsWithChunkCounts()` |
| `internal/store/sqlite.go:354` | `GetChunkByID()` |
| `internal/store/sqlite.go:398` | `GetChunksByIDs()` |
| `internal/models/chunk.go` | All Chunk marshal/unmarshal methods |

### Integration tests that won't compile

| File | Issue |
|------|-------|
| `tests/integration/ingest_test.go:47` | `chunker.NewChunker()` called with no arguments; actual signature requires `scriptPath string` |
| `tests/integration/ingest_test.go:115` | `ChunkDocument(ctx, doclingDoc)` passes `*DoclingDocument` but method expects `[]byte` |

---

## Python Chunker Gap

`scripts/chunk_helper.py:27` creates `HierarchicalChunker()` with **no
parameters**. Per task T036, it should be configured with 800 tokens and 200
overlap. The `HierarchicalChunker` constructor in the current docling version
may not accept these parameters directly -- this needs investigation.

---

## Missing Feature: `aqe version`

`internal/cli/root.go:40` checks `cmd.Name() == "version"` to skip database
initialization, but no `version` subcommand is registered. Running
`./aqe version` produces "unknown command".

---

## Silent Degradation Paths

These situations silently degrade functionality without user-visible warnings
(only visible with `--debug`):

1. **Python chunker unavailable** -- Falls back to raw text items from Docling.
   Chunks will lack hierarchical section paths and may have worse overlap
   properties. (`ingest.go:113-118`)

2. **Weaviate chunk insertion failure** -- Individual chunk insert errors are
   swallowed; the chunk will exist in SQLite but have no embedding in Weaviate,
   making it unsearchable. (`ingest.go:260-267`)

3. **Unknown Harvard formatter style** -- `NewFormatter("anything")` silently
   returns `HarvardUS{}` with no error or warning. (`formatter.go:44-52`)

---

## Undocumented Behaviour

- `.md` (Markdown) is accepted as an ingest format (`ingest.go:289`), but the
  spec (`spec.md`) only mentions PDF, DOCX, and TXT.

- Source type detection (`ingest.go:320-332`) is based on filename keyword
  matching ("article", "journal", "chapter", "web", "online"), which is
  fragile and undocumented to users.

- The Ollama URL `http://ollama:11434` used in `ingest.go:91` and
  `extract.go:73` is a Docker internal hostname. This works when Weaviate
  calls Ollama inside Docker, but the CLI itself connects to Weaviate at
  `localhost:8080`. The Ollama URL is passed to Weaviate's vectorizer config,
  not called directly by the Go code.
