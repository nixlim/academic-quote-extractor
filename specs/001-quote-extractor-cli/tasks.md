# Tasks: Academic Quote Extractor CLI

**Input**: Design documents from `/specs/001-quote-extractor-cli/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md
**Additional Context**: SPEC-v2.md and DOCLING_HOWTO.md at repository root

**Architecture Diagrams** (at repository root):
- `architecture-v2.mermaid` - System component layout
- `data-flow-v2.mermaid` - Ingest/Extract/Export sequence flows
- `agent-roles.mermaid` - Conceptual agent responsibilities

**Tests**: Test tasks are included for contract and integration testing as specified in plan.md and research.md.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Create directory structure per plan.md: cmd/aqe/, internal/{cli,docling,chunker,claude,search,store,harvard,models}/, scripts/, tests/{contract,integration,unit}/

- [X] T002 Initialize Go module: `go mod init github.com/[org]/aqe`

- [X] T003 Install Go dependencies: `go get github.com/spf13/cobra github.com/weaviate/weaviate-go-client/v4 github.com/mattn/go-sqlite3 github.com/stretchr/testify`

- [X] T004 [P] Create docker-compose.yml with Docling service (quay.io/docling-project/docling-serve:latest on port 5001)

- [X] T005 [P] Add Weaviate service to docker-compose.yml (cr.weaviate.io/semitechnologies/weaviate:1.27.0 on port 8080 with text2vec-ollama module)

- [X] T006 [P] Add Ollama service to docker-compose.yml (ollama/ollama:latest on port 11434)

- [X] T007 [P] Create scripts/requirements.txt with docling>=2.70.0 and docling-core dependencies

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

### Database Layer

- [X] T008 Create internal/store/migrations.go with documents table schema (id, filename, filepath, title, authors JSON, year, publisher, source_type, checksum UNIQUE, ingested_at)

- [X] T009 Add chunks table schema to migrations.go (id TEXT PK, document_id FK, text, page_num, section_path JSON, bbox JSON, embedding_id)

- [X] T010 Add extractions table schema to migrations.go (id, topic, created_at)

- [X] T011 Add extracted_quotes table schema to migrations.go (id, extraction_id FK, chunk_id FK, relevance_score CHECK 0-100, explanation)

- [X] T012 Add indexes to migrations.go: idx_chunks_document, idx_chunks_page, idx_quotes_extraction, idx_quotes_chunk, idx_documents_checksum

- [X] T013 Implement Store struct and NewStore() in internal/store/sqlite.go (open database, create if not exists)

- [X] T014 Implement Store.Close() and Store.RunMigrations() in internal/store/sqlite.go

- [X] T015 Implement Store.WithTx() transaction helper in internal/store/sqlite.go (rollback on error)

### Domain Models

- [X] T016 [P] Create SourceType enum in internal/models/document.go (book, journal_article, website, chapter, unknown)

- [X] T017 [P] Create Document struct in internal/models/document.go with JSON marshal/unmarshal for Authors field

- [X] T018 [P] Create BBox struct in internal/models/chunk.go (L, T, R, B, CoordOrigin)

- [X] T019 [P] Create Chunk struct in internal/models/chunk.go with JSON helpers for SectionPath and BBox

- [X] T020 [P] Create Extraction struct in internal/models/extraction.go (ID, Topic, CreatedAt)

- [X] T021 [P] Create ExtractedQuote struct in internal/models/quote.go (ID, ExtractionID, ChunkID, RelevanceScore, Explanation)

### Docling Integration

- [X] T022 [P] Create Docling types in internal/docling/types.go: DoclingDocument, Origin, TextItem structs

- [X] T023 [P] Create Provenance and BBox types in internal/docling/types.go per contracts/docling-api.md

- [X] T024 Create Docling Client struct and NewClient() in internal/docling/client.go (baseURL, httpClient with 60s timeout)

- [X] T025 Implement Client.Health() in internal/docling/client.go (GET /health)

- [X] T026 Implement Client.ConvertFile() in internal/docling/client.go (POST /v1/convert/file multipart upload)

### Weaviate Integration

- [X] T027 [P] Create ChunkResult struct in internal/search/weaviate.go (Text, ChunkID, DocumentID, PageNum, SectionPath, Score)

- [X] T028 [P] Implement GetChunkClass() in internal/search/schema.go returning Weaviate class definition with text2vec-ollama vectorizer

- [X] T029 Create WeaviateClient struct and NewWeaviateClient() in internal/search/weaviate.go

- [X] T030 Implement WeaviateClient.CreateSchema() in internal/search/weaviate.go

- [X] T031 Implement WeaviateClient.InsertChunk() in internal/search/weaviate.go (returns UUID)

- [X] T032 Implement WeaviateClient.DeleteByDocumentID() in internal/search/weaviate.go

### Python Chunker

- [X] T033 Create Chunker struct and ChunkOutput type in internal/chunker/chunker.go

- [X] T034 Implement NewChunker() in internal/chunker/chunker.go (verify Python and script exist)

- [X] T035 Implement Chunker.ChunkDocument() in internal/chunker/chunker.go (exec Python, stdin JSON, parse stdout)

- [X] T036 Create scripts/chunk_helper.py with HierarchicalChunker (800 tokens, 200 overlap) - read stdin, write JSON lines to stdout

### Claude Integration

- [X] T037 [P] Create Claude types in internal/claude/wrapper.go: ExtractionTask, ChunkInput, ExtractionResponse, SelectedChunk structs

- [X] T038 Create Wrapper struct and NewWrapper() in internal/claude/wrapper.go (cliPath, timeout)

- [X] T039 Implement Wrapper.buildPrompt() in internal/claude/prompt.go using text/template per contracts/claude-prompt.md

- [X] T040 Implement Wrapper.ExtractQuotes() in internal/claude/wrapper.go:
  **IMPORTANT**: Use `claude --print --output-format json -p "<prompt>"` - the `-p` flag MUST be last

- [X] T041 Implement ValidateResponse() in internal/claude/wrapper.go (check chunk_id format, existence, score bounds)

### Harvard Formatting

- [X] T042 [P] Create Author struct and FormatAuthors() helper in internal/harvard/types.go (handles 1, 2, 3+ authors)

- [X] T043 [P] Create FormatAuthorsShort() helper in internal/harvard/types.go for in-text citations

- [X] T044 Create Formatter interface in internal/harvard/formatter.go (FormatFull, FormatInText, FormatReference)

- [X] T045 Create NewFormatter() factory in internal/harvard/formatter.go (returns harvard_us implementation)

- [X] T046 Implement HarvardUS.FormatBook() in internal/harvard/harvard_us.go (include DOI when available per FR-006a)

- [X] T047 Implement HarvardUS.FormatJournalArticle() in internal/harvard/harvard_us.go (include DOI when available per FR-006a)

- [X] T048 Implement HarvardUS.FormatWebsite() in internal/harvard/harvard_us.go (with US date format, include URL per FR-006a)

- [X] T049 Implement HarvardUS.FormatChapter() in internal/harvard/harvard_us.go (include DOI when available per FR-006a)

- [X] T050 Implement HarvardUS.FormatInText() in internal/harvard/harvard_us.go

### CLI Framework

- [X] T051 Create root command in internal/cli/root.go with --db flag (default ./quotes.db)

- [X] T052 Add --debug flag to root command in internal/cli/root.go

- [X] T053 Create cmd/aqe/main.go entry point calling cli.Execute()

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Document Ingestion (Priority: P1)

**Goal**: Parse PDF, DOCX, and TXT files, chunk with metadata preservation, generate embeddings, and store for retrieval

**Acceptance Criteria from spec.md**:
- AC1: `aqe ingest ./sources/` processes all files, displays "Ingested N documents, M chunks"
- AC2: `aqe ingest doc.pdf --title '...' --author '...' --year 2023` uses provided metadata
- AC3: Mixed formats processed, unsupported skipped with warning
- AC4: Duplicate detection via checksum, skip with message
- AC5: Corrupted file error reported, continue with remaining files

### Contract Tests for User Story 1

- [X] T054 [P] [US1] Contract test TestDoclingHealth() in tests/contract/docling_test.go

- [X] T055 [P] [US1] Contract test TestDoclingConvertPDF() in tests/contract/docling_test.go

- [X] T056 [P] [US1] Contract test TestDoclingConvertDOCX() in tests/contract/docling_test.go

- [X] T057 [P] [US1] Contract test TestWeaviateSchemaCreation() in tests/contract/weaviate_test.go

- [X] T058 [P] [US1] Contract test TestWeaviateInsertChunk() in tests/contract/weaviate_test.go

- [X] T059 [P] [US1] Contract test TestOllamaModelAvailable() in tests/contract/ollama_test.go (verify nomic-embed-text)

### Store Operations for User Story 1

- [X] T060 [US1] Implement CalculateChecksum() in internal/store/sqlite.go (SHA-256 of file)

- [X] T061 [US1] Implement Store.InsertDocument() in internal/store/sqlite.go (marshal authors JSON, return ID)

- [X] T062 [US1] Implement Store.GetDocumentByChecksum() in internal/store/sqlite.go (return nil if not found)

- [X] T063 [US1] Implement Store.InsertChunk() in internal/store/sqlite.go

- [X] T064 [US1] Implement Store.InsertChunks() batch insert in internal/store/sqlite.go (use transaction)

### CLI Implementation for User Story 1

- [X] T065 [US1] Create ingest command skeleton in internal/cli/ingest.go with --title, --author, --year flags

- [X] T066 [US1] Implement file/directory detection and glob for *.pdf, *.docx, *.txt in ingest.go

- [X] T067 [US1] Implement duplicate check in runIngest() - calculate checksum, skip if exists

- [X] T068 [US1] Implement batch resume detection per FR-011a - display "Resuming batch ingestion, X of Y files already processed" when re-running

- [X] T069 [US1] Implement document parsing in runIngest() - call Docling ConvertFile()

- [X] T070 [US1] Implement chunking in runIngest() - call Chunker.ChunkDocument()

- [X] T071 [US1] Implement storage in runIngest() - insert Document, Chunks to SQLite

- [X] T072 [US1] Implement Weaviate insertion in runIngest() - insert chunks, store embedding_id

- [X] T073 [US1] Add progress output: "Processing: filename.pdf", "Chunks: N"

- [X] T074 [US1] Add summary output: "Ingested N documents, M chunks"

- [X] T075 [US1] Add error handling for service unavailable (Docling not running)

- [X] T076 [US1] Add error handling for corrupted/unsupported files (continue with remaining)

### Integration Test for User Story 1

- [X] T077 [US1] Integration test TestIngestSinglePDF() in tests/integration/ingest_test.go

- [X] T078 [US1] Integration test TestIngestDuplicateDetection() in tests/integration/ingest_test.go

**Checkpoint**: User Story 1 complete - can ingest documents and verify in storage

---

## Phase 4: User Story 2 - Quote Extraction (Priority: P2)

**Goal**: Search for semantically relevant passages, score with LLM, return quotes with Harvard citations

**Acceptance Criteria from spec.md**:
- AC1: `aqe extract "topic"` returns quotes with Harvard citations and explanations
- AC2: No relevant quotes returns informative message
- AC3: Default: up to 20 quotes with relevance >= 60
- AC4: `--max-quotes 10 --min-relevance 80` respected

### Contract Tests for User Story 2

- [X] T079 [P] [US2] Contract test TestWeaviateHybridSearch() in tests/contract/weaviate_test.go

- [X] T080 [P] [US2] Contract test TestClaudeOutputFormat() in tests/contract/claude_test.go

### Store Operations for User Story 2

- [X] T081 [US2] Implement WeaviateClient.HybridSearch() in internal/search/weaviate.go (BM25 + vector, alpha=0.5)

- [X] T082 [US2] Implement Store.GetChunksByIDs() in internal/store/sqlite.go

- [X] T083 [US2] Implement Store.GetChunksWithDocuments() join query in internal/store/sqlite.go

- [X] T084 [US2] Implement Store.InsertExtraction() in internal/store/sqlite.go

- [X] T085 [US2] Implement Store.InsertExtractedQuotes() batch insert in internal/store/sqlite.go

### CLI Implementation for User Story 2

- [X] T086 [US2] Create extract command skeleton in internal/cli/extract.go with --max-quotes, --min-relevance flags

- [X] T087 [US2] Implement hybrid search call in runExtract() - get top 50 candidates from Weaviate

- [X] T088 [US2] Implement chunk lookup in runExtract() - get verbatim text from SQLite

- [X] T089 [US2] Implement Claude call in runExtract() - build task, call ExtractQuotes()

- [X] T090 [US2] Implement response validation in runExtract() - verify chunk_ids exist

- [X] T091 [US2] Implement filtering in runExtract() - apply min-relevance threshold, limit to max-quotes

- [X] T092 [US2] Implement persistence in runExtract() - save Extraction and ExtractedQuotes to SQLite

- [X] T093 [US2] Implement quote display output with Harvard in-text citations

- [X] T094 [US2] Add error handling for empty corpus ("Run 'aqe ingest' first")

- [X] T095 [US2] Add error handling for no relevant quotes found

### Integration Test for User Story 2

- [X] T096 [US2] Integration test TestExtractWithMatches() in tests/integration/extract_test.go

- [X] T097 [US2] Integration test TestExtractNoMatches() in tests/integration/extract_test.go

**Checkpoint**: User Story 2 complete - can extract relevant quotes with citations

---

## Phase 5: User Story 3 - Extraction Export (Priority: P3)

**Goal**: Output saved extractions in Markdown, JSON, or BibTeX format

**Acceptance Criteria from spec.md**:
- AC1: `aqe export 1 --format markdown` outputs blockquotes, citations, bibliography
- AC2: `aqe export 1 --format json` outputs valid JSON
- AC3: `aqe export 1 --format bibtex` outputs valid BibTeX
- AC4: Invalid extraction ID shows error with available IDs

### Store Operations for User Story 3

- [X] T098 [US3] Implement Store.GetExtractionByID() in internal/store/sqlite.go

- [X] T099 [US3] Implement Store.GetQuotesByExtractionID() with chunk/document join in internal/store/sqlite.go

- [X] T100 [US3] Implement Store.ListExtractions() in internal/store/sqlite.go

### CLI Implementation for User Story 3

- [X] T101 [US3] Create export command skeleton in internal/cli/export.go with --format, --output flags

- [X] T102 [US3] Implement extraction lookup in runExport() with error for invalid ID

- [X] T103 [P] [US3] Implement formatMarkdown() in internal/cli/export.go (blockquotes, citations, bibliography)

- [X] T104 [P] [US3] Implement formatJSON() in internal/cli/export.go per spec.md JSON output format

- [X] T105 [P] [US3] Implement formatBibTeX() in internal/cli/export.go (generate citation keys, map source types)

- [X] T106 [US3] Implement file output in runExport() (--output flag writes to file, otherwise stdout)

- [X] T107 [US3] Add error message for invalid extraction ID with list of available IDs

### Integration Test for User Story 3

- [X] T108 [US3] Integration test TestExportMarkdown() in tests/integration/export_test.go (verify renders correctly per SC-007)

- [X] T109 [US3] Integration test TestExportJSON() in tests/integration/export_test.go (verify valid parseable JSON per SC-006)

**Checkpoint**: User Story 3 complete - can export extractions in multiple formats

---

## Phase 6: User Story 4 - Metadata Correction (Priority: P4)

**Goal**: Interactive completion of missing metadata for documents

**Acceptance Criteria from spec.md**:
- AC1: `aqe meta fix` prompts for missing author, title, year
- AC2: All complete shows "All documents have complete metadata."

### Store Operations for User Story 4

- [X] T110 [US4] Implement Store.GetDocumentsWithIncompleteMetadata() in internal/store/sqlite.go

- [X] T111 [US4] Implement Store.UpdateDocumentMetadata() in internal/store/sqlite.go

### CLI Implementation for User Story 4

- [X] T112 [US4] Create meta command group in internal/cli/meta.go

- [X] T113 [US4] Create meta fix subcommand skeleton in internal/cli/meta.go

- [X] T114 [US4] Implement document iteration and prompting in runMetaFix()

- [X] T115 [US4] Implement metadata update and confirmation output in runMetaFix()

### Unit Test for User Story 4

- [X] T116 [US4] Unit test for metadata update logic in tests/unit/store_test.go

**Checkpoint**: User Story 4 complete - can fix missing metadata interactively

---

## Phase 7: List Command & Polish

**Purpose**: Add list command and quality improvements

### List Command

- [X] T117 Implement list command in internal/cli/list.go (show saved extractions with quote counts)

### Unit Tests

- [X] T118 [P] Unit test TestFormatAuthors() in tests/unit/harvard_test.go (1, 2, 3+ authors)

- [X] T119 [P] Unit test TestFormatInText() in tests/unit/harvard_test.go

- [X] T120 [P] Unit test TestFormatBook() in tests/unit/harvard_test.go (include DOI/URL per FR-006a)

- [X] T121 [P] Unit test TestFormatJournalArticle() in tests/unit/harvard_test.go (include DOI per FR-006a)

- [X] T122 [P] Unit test TestDocumentCRUD() in tests/unit/store_test.go

- [X] T123 [P] Unit test TestChunkCRUD() in tests/unit/store_test.go

- [X] T124 [P] Unit test TestChecksumUniqueness() in tests/unit/store_test.go

### Quality & Validation

- [X] T125 Implement debug logging for Docling requests in internal/docling/client.go (when --debug)

- [X] T126 Implement debug logging for Weaviate queries in internal/search/weaviate.go (when --debug)

- [X] T127 Implement debug logging for Claude prompts in internal/claude/wrapper.go (when --debug)

- [X] T128 Implement exit codes: 0 success, 1 user error, 2 system error across all commands

- [ ] T129 Validate quickstart.md scenarios 1-3 (ingestion) - includes SC-008 usability check

- [ ] T130 Validate quickstart.md scenarios 4-5 (extraction)

- [ ] T131 Validate quickstart.md scenarios 6-8 (export)

- [ ] T132 Validate quickstart.md scenario 9 (metadata fix)

- [ ] T133 Validate quickstart.md scenario 10 (error handling)

- [ ] T134 Verify SC-003: all output quotes match verbatim source text (zero hallucination)

- [ ] T135 Verify SC-004: Harvard references pass manual verification (author format, year, page, punctuation)

- [ ] T136 Verify FR-006b: Formatter interface supports adding new citation styles (extensibility check)

- [ ] T137 Performance test: ingest 100-page PDF in <60 seconds on standard hardware (SC-001) - baseline: Apple M1/M2 or equivalent

- [ ] T138 Performance test: extract from 10 documents in <10 seconds with warm cache (SC-002)

---

## Dependencies & Execution Order

### Phase Dependencies

```
Phase 1: Setup (7 tasks)
    └── Phase 2: Foundational (46 tasks)
            ├── Phase 3: User Story 1 (25 tasks)
            │       └── Phase 4: User Story 2 (19 tasks)
            │               └── Phase 5: User Story 3 (12 tasks)
            └── Phase 6: User Story 4 (7 tasks)
                    └── Phase 7: Polish (22 tasks)
```

### Parallel Opportunities by Phase

**Phase 1 (Setup)**:
- T004 || T005 || T006 || T007 (docker services + requirements.txt)

**Phase 2 (Models)**:
- T016 || T017 || T018 || T019 || T020 || T021 (all model files)
- T022 || T023 (docling types)
- T027 || T028 (weaviate types)
- T037 (claude types)
- T042 || T043 (harvard helpers)

**Phase 3 (US1 contract tests)**:
- T054 || T055 || T056 || T057 || T058 || T059 (all contract tests)

**Phase 4 (US2 contract tests)**:
- T079 || T080 (contract tests)

**Phase 5 (US3 export formats)**:
- T103 || T104 || T105 (markdown, json, bibtex)

**Phase 7 (Unit tests)**:
- T118 || T119 || T120 || T121 || T122 || T123 || T124 (all unit tests)

---

## Summary

| Metric | Count |
|--------|-------|
| **Total Tasks** | 138 |
| **Setup Tasks** | 7 |
| **Foundational Tasks** | 46 |
| **User Story 1 Tasks** | 25 |
| **User Story 2 Tasks** | 19 |
| **User Story 3 Tasks** | 12 |
| **User Story 4 Tasks** | 7 |
| **Polish Tasks** | 22 |
| **Parallel Opportunities** | 35+ tasks marked [P] |

### Key Design Constraints

1. **Verbatim Quotes Only**: All quote text comes from SQLite chunks.text, never from LLM
2. **Chunk ID Validation**: Claude response chunk_ids must exist in corpus
3. **Harvard US Style**: Dates as "January 15, 2024", double quotes for article titles
4. **Local Embeddings**: nomic-embed-text via Ollama, no external API calls
5. **Exit Codes**: 0 = success, 1 = user error, 2 = system error
6. **Claude CLI**: Use `-p` flag LAST: `claude --print --output-format json -p "<prompt>"`
