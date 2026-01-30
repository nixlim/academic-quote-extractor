# Feature Specification: Academic Quote Extractor CLI

**Feature Branch**: `001-quote-extractor-cli`  
**Created**: 2026-01-29  
**Status**: Draft  
**Input**: Build AQE (Academic Quote Extractor), a Go CLI application for extracting relevant quotes from academic documents with Harvard-style citations.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Document Ingestion (Priority: P1)

A student drops PDF, DOCX, or TXT files into a sources folder and runs the ingest command to process all documents. The system parses documents, chunks them with metadata preservation (page numbers, sections), generates embeddings, and stores chunks for later search and retrieval. Upon completion, the system displays a summary of ingested documents and chunks.

**Why this priority**: Without ingested documents, no other feature can function. This is the foundational capability that enables all downstream quote extraction.

**Independent Test**: Can be fully tested by ingesting a set of sample documents and verifying they appear in storage with correct metadata. Delivers value as a document indexing system even before extraction is implemented.

**Acceptance Scenarios**:

1. **Given** a folder containing 5 PDF files, **When** the user runs "aqe ingest ./sources/", **Then** the system processes all files and displays "Ingested 5 documents, N chunks" where N reflects actual chunk count.

2. **Given** a single PDF file, **When** the user runs "aqe ingest document.pdf --title 'Economics 101' --author 'Smith, J.' --year 2023", **Then** the system ingests the document with the provided metadata overrides.

3. **Given** a folder with mixed file types (PDF, DOCX, TXT), **When** the user runs "aqe ingest ./sources/", **Then** the system processes all supported formats and skips unsupported files with a warning.

4. **Given** a document that was previously ingested, **When** the user runs ingest again on the same file, **Then** the system detects the duplicate (via checksum) and skips it with a message.

5. **Given** a corrupted or unreadable file, **When** the user runs ingest, **Then** the system reports an error for that specific file and continues processing remaining files.

---

### User Story 2 - Quote Extraction (Priority: P2)

A student runs the extract command with a research topic to find relevant quotes from ingested documents. The system searches for semantically relevant passages, scores them for relevance, and presents quotes with proper Harvard citations and explanations of why each quote is relevant to the topic.

**Why this priority**: This is the core value proposition of the tool. Once documents are ingested, users need to extract relevant quotes for their essays.

**Independent Test**: Can be tested by ingesting sample documents, then running extraction with a known topic and verifying relevant quotes are returned with valid citations.

**Acceptance Scenarios**:

1. **Given** ingested documents about social media, **When** the user runs "aqe extract 'Impact of social media on political polarization'", **Then** the system returns relevant quotes with Harvard in-text citations and relevance explanations.

2. **Given** a topic with no relevant quotes in the corpus, **When** the user runs extract, **Then** the system returns an empty result with an informative message.

3. **Given** the default settings, **When** the user runs extract, **Then** the system returns up to 20 quotes with relevance scores of 60 or higher.

4. **Given** custom parameters, **When** the user runs "aqe extract 'topic' --max-quotes 10 --min-relevance 80", **Then** the system returns at most 10 quotes, all with relevance scores of 80 or higher.

5. **Given** multiple documents from different sources, **When** the user runs extract, **Then** all returned quotes include accurate page numbers and properly formatted author/year citations.

---

### User Story 3 - Extraction Export (Priority: P3)

A student runs the export command to output a saved extraction session in various formats. The system retrieves the extraction, formats quotes with proper Harvard references, and outputs to the terminal or a file in JSON, Markdown, or BibTeX format.

**Why this priority**: After extracting quotes, users need to export them for use in essays, reference managers, or further processing.

**Independent Test**: Can be tested by creating an extraction, then exporting it in each format and verifying the output is valid and complete.

**Acceptance Scenarios**:

1. **Given** a saved extraction with ID 1, **When** the user runs "aqe export 1 --format markdown", **Then** the system outputs properly formatted Markdown with blockquotes, citations, and a bibliography section.

2. **Given** a saved extraction, **When** the user runs "aqe export 1 --format json", **Then** the system outputs valid, parseable JSON containing quotes, references, and relevance explanations.

3. **Given** a saved extraction, **When** the user runs "aqe export 1 --format bibtex", **Then** the system outputs a valid BibTeX bibliography file with entries for all cited sources.

4. **Given** an invalid extraction ID, **When** the user runs export, **Then** the system displays an error message with available extraction IDs.

5. **Given** an extraction, **When** the user runs "aqe export 1 --format markdown --output quotes.md", **Then** the system writes the output to the specified file.

---

### User Story 4 - Metadata Correction (Priority: P4)

A student runs the metadata fix command to interactively complete missing metadata for documents where author, title, or year could not be auto-detected during ingestion.

**Why this priority**: Complete metadata is necessary for accurate citations, but this is a correction workflow that can be deferred until core functionality works.

**Independent Test**: Can be tested by ingesting a document with no detectable metadata, then running the fix command and verifying the metadata is updated.

**Acceptance Scenarios**:

1. **Given** documents with incomplete metadata, **When** the user runs "aqe meta fix", **Then** the system prompts for each missing field (author, title, year) interactively.

2. **Given** all documents have complete metadata, **When** the user runs "aqe meta fix", **Then** the system displays "All documents have complete metadata."

3. **Given** a document with missing author, **When** the user provides the author name during the interactive prompt, **Then** the system updates the document metadata and confirms the change.

---

### Edge Cases

- What happens when the document parsing service is unavailable? System displays a clear error message indicating the service is down and how to start it.
- What happens when a PDF has no extractable text (scanned image)? Docling-serve handles OCR automatically when configured; system reports "no text extracted" if OCR is not available or fails. (Note: OCR configuration is handled by Docling-serve, not this application.)
- What happens when the search service is unavailable during extraction? System displays a clear error message indicating the service is down.
- What happens when a quote spans multiple pages? System includes the starting page number in the citation. The chunk's page_num field stores the first page where the chunk begins; page ranges are not tracked as chunks are sized to typically fit within a single page.
- What happens when author names contain non-ASCII characters? System preserves the original characters in citations.
- What happens when a document has multiple authors? System formats according to Harvard style (two authors: "Smith and Jones"; three or more: "Smith et al.").

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST parse PDF, DOCX, and TXT files through a document processing service.
- **FR-001a**: System MUST use a Python wrapper script with Docling's HierarchicalChunker for chunking, preserving section hierarchy and page metadata.
- **FR-001b**: Chunking MUST use sufficient overlap (minimum 25%) to ensure sentences and key phrases are not split across chunk boundaries.
- **FR-002**: System MUST preserve page numbers, section headings, and position metadata for each chunk.
- **FR-003**: System MUST store verbatim chunk text as the authoritative source for all quote output.
- **FR-004**: System MUST use hybrid search combining keyword matching and semantic similarity for retrieval.
- **FR-004a**: System MUST generate embeddings locally without external API calls, using a local embedding model via Weaviate's text2vec-ollama module.
- **FR-005**: System MUST obtain relevance scores and explanations from an LLM, receiving only chunk identifiers (never LLM-generated quote text).
- **FR-006**: System MUST format Harvard references for books, journal articles, websites, and book chapters using US style (e.g., "January 15, 2024" date format, double quotation marks).
- **FR-006a**: System MUST include DOI or URL in references when available.
- **FR-006b**: Reference formatting MUST be modular to support adding other citation styles (e.g., APA, MLA, Chicago) in the future.
- **FR-007**: System MUST validate that all chunk identifiers exist before including quotes in output.
- **FR-008**: System MUST support configurable relevance threshold with a default of 60 out of 100.
- **FR-009**: System MUST support configurable maximum quote count with a default of 20.
- **FR-010**: System MUST persist extraction sessions for later retrieval and export.
- **FR-011**: System MUST detect duplicate documents via checksum and skip re-ingestion.
- **FR-011a**: System MUST support resumable batch ingestion: if processing fails mid-batch, successfully ingested documents are retained and re-running ingest continues from where it left off.
- **FR-012**: System MUST provide clear error messages for all failure scenarios, distinguishing user errors from system errors.
- **FR-013**: System MUST exit with code 0 on success and non-zero on error.

### Key Entities

- **Document**: Represents an ingested source file. Contains filename, filepath, title, authors (multiple allowed), publication year, publisher, source type (book, journal article, website, chapter), and checksum for duplicate detection.

- **Chunk**: Represents a segment of text from a document. Contains a unique identifier, reference to parent document, verbatim text content, page number, section path (hierarchy of headings), position coordinates, and reference to its embedding in the search index.

- **Extraction**: Represents a quote extraction session. Contains the research topic and creation timestamp. Groups extracted quotes for later export.

- **ExtractedQuote**: Represents a single quote selected during extraction. Contains reference to the extraction session, reference to the source chunk, relevance score (0-100), and explanation of why the quote is relevant.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can ingest a 100-page document in under 60 seconds. (Baseline: Apple M1/M2 or equivalent x86_64 with 16GB RAM, SSD storage, Docker services running locally)
- **SC-002**: Users can extract quotes from a corpus of 10 documents in under 10 seconds. (Measured with warm cache; initial extraction may be slower due to model loading)
- **SC-003**: 100% of output quotes match verbatim text from source documents (zero citation hallucination).
- **SC-004**: Harvard references pass manual verification in 100% of cases (correct author format, year, page number, punctuation).
- **SC-005**: All CLI commands exit with code 0 on success and non-zero on error, with meaningful error messages.
- **SC-006**: JSON output is valid and parseable by standard tools.
- **SC-007**: Markdown output renders correctly in common viewers (GitHub, VS Code, Obsidian).
- **SC-008**: Users can complete the full workflow (ingest, extract, export) with no prior training.

## Assumptions

- The document parsing service (Docling-serve) is running and accessible.
- The search service (Weaviate) is running and accessible with text2vec-ollama module configured.
- A local Ollama instance is running with an embedding model (e.g., nomic-embed-text or mxbai-embed-large).
- Users have Claude Code CLI installed for relevance scoring.
- Documents are in readable format (not password-protected).
- Users are working with academic or research documents that contain quotable text passages.

## Clarifications

### Session 2026-01-29

- Q: Should embeddings use external APIs or local models? → A: Local embeddings via Weaviate's text2vec-ollama module for offline operation.
- Q: Use docling-serve endpoint or Python wrapper for chunking? → A: Python wrapper script with HierarchicalChunker for metadata preservation.
- Q: Harvard reference style - UK or US format? → A: US style (dates as "January 15, 2024", double quotes). Include DOI/URL when available. Make formatting modular for future citation styles.
- Q: Error recovery for batch ingestion failures? → A: Resume capability - successfully ingested documents retained, re-run continues from where it left off.
- Q: Embedding model selection? → A: nomic-embed-text (768 dimensions, 2K context window) - balances speed and quality, outperforms OpenAI ada-002, ideal context length for academic chunks.
- Q: Chunk overlap for retrieval quality? → A: Minimum 25% overlap to avoid splitting sentences/phrases at boundaries and improve recall.
