# Spec-Kit Instructions for Academic Quote Extractor (AQE)

## Project Overview

The Academic Quote Extractor is a Go CLI application that extracts quotes from academic sources with Harvard references and relevance explanations. It uses a hybrid RAG architecture with Docling for document processing, Weaviate for vector search, SQLite for persistence, and Claude Code CLI for relevance analysis.

---

## Step 1: Initialize Spec-Kit

```bash
# Install Specify CLI (if not already installed)
uv tool install specify-cli --from git+https://github.com/github/spec-kit.git

# Initialize in the current project directory
specify init . --ai claude --force

# Or create fresh project directory
specify init academic-quote-extractor --ai claude

# Verify installation
specify check
```

---

## Step 2: Create Project Constitution

Launch Claude Code in the project directory and run:

```
/speckit.constitution

**Prompt for constitution:**

Create principles for the Academic Quote Extractor project with the following governing guidelines:

CORE PRINCIPLES:
1. Deterministic Quote Retrieval: Never let the LLM generate quote text. Claude outputs chunk IDs only, system replaces with verbatim text from database. This eliminates citation hallucination entirely.

2. Hybrid RAG Architecture: Use semantic search (BM25 + vector) for retrieval, LLM only for relevance scoring and explanation generation. Optimize for cost-effectiveness over maximum accuracy.

3. Single Purpose Focus: Extract quotes with Harvard references and relevance explanations. No theme clustering, no argument mapping, no contradiction detection.

4. Go-First Development: All core logic in Go. Python only for Docling chunking wrapper. Minimize external dependencies.

5. Docker-Based Services: Docling-serve and Weaviate run in Docker. SQLite embedded for persistence.

CODE QUALITY STANDARDS:
- Test coverage for all CLI commands
- Integration tests with real Docker services
- Contract tests for all external APIs (Docling, Weaviate)
- Error handling with meaningful messages for CLI users

ARCHITECTURAL CONSTRAINTS:
- Maximum 3 main packages: cmd/aqe, internal/*, scripts/
- No custom abstractions over Weaviate or SQLite clients
- Harvard reference formatting in pure Go (no external libraries)
- Chunk IDs as primary key for quote lookup (never store formatted quotes)

OUTPUT FORMAT REQUIREMENTS:
- JSON output must be valid and parseable
- Markdown output must render correctly in common viewers
- In-text citations must follow strict Harvard format: (Author, Year, p. X)
- Full references must be complete and verifiable
```

---

## Step 3: Create Feature Specification

Run the specification command:

```
/speckit.specify

**Prompt for specification:**

Review the following files:
[Academic Quote Extractor v2 - Revised Specification](SPEC-v2)
architecture-v2.mermaid
data-flow-v2.mermaid
agent-roles.mermaid
DOCLING_HOWTO.md

Build AQE (Academic Quote Extractor), a Go CLI application for extracting relevant quotes from academic documents with Harvard-style citations.

USER JOURNEYS (in priority order):

P1 - INGEST: A student drops PDF/DOCX files into a sources folder and runs "aqe ingest ./sources/" to process all documents. The system parses documents via Docling, chunks them with metadata preservation (page numbers, sections), generates embeddings, and stores chunks in both Weaviate (for search) and SQLite (for verbatim text retrieval). Success: "Ingested 5 documents, 234 chunks"

P2 - EXTRACT: A student runs "aqe extract 'Impact of social media on political polarization'" to find relevant quotes. The system performs hybrid search (BM25 + vector), sends top chunks to Claude for relevance scoring, receives chunk IDs back (not generated text), looks up verbatim text from SQLite, formats Harvard references, and presents quotes with relevance explanations. Success: Displays 12 relevant quotes with proper citations.

P3 - EXPORT: A student runs "aqe export 1 --format markdown" to export a saved extraction. The system retrieves the extraction from SQLite, formats quotes with proper Harvard references, and outputs to stdout or file. Formats: JSON, Markdown, BibTeX bibliography.

P4 - METADATA FIX: A student runs "aqe meta fix" to interactively complete missing metadata (author, title, year) for documents where it couldn't be auto-detected.

FUNCTIONAL REQUIREMENTS:
- FR-001: System MUST parse PDFs, DOCX, and TXT files via Docling-serve API
- FR-002: System MUST preserve page numbers, section headings, and bounding boxes per chunk
- FR-003: System MUST store verbatim chunk text in SQLite as source of truth
- FR-004: System MUST use Weaviate hybrid search (BM25 + vector similarity)
- FR-005: System MUST call Claude Code CLI for relevance scoring, receiving only chunk IDs
- FR-006: System MUST format Harvard references in pure Go (books, journal articles, websites, chapters)
- FR-007: System MUST validate chunk IDs exist before outputting quotes
- FR-008: System MUST support configurable relevance threshold (default: 60/100)
- FR-009: System MUST support configurable max quotes (default: 20)
- FR-010: System MUST persist extraction sessions for later export

KEY ENTITIES:
- Document: filename, filepath, title, authors (JSON array), year, publisher, source_type, checksum
- Chunk: id (Docling self_ref), document_id, text (verbatim), page_num, section_path, bbox, embedding_id
- Extraction: topic, created_at
- ExtractedQuote: extraction_id, chunk_id, relevance_score, explanation

SUCCESS CRITERIA:
- SC-001: Ingest 100-page PDF in under 60 seconds
- SC-002: Extract quotes from 10 documents in under 10 seconds
- SC-003: Zero citation hallucinations (all quotes verbatim from source)
- SC-004: Harvard references pass manual verification 100% of time
- SC-005: CLI commands exit 0 on success, non-zero on error with meaningful message
```

---

## Step 4: Clarify Specification (Recommended)

Before planning, run the clarification workflow:

```
/speckit.clarify
```

This will ask structured questions about ambiguous areas. Key clarifications resolved:

1. **Embedding Model**: Local embeddings via Weaviate's text2vec-ollama module (offline operation)
2. **Docling Chunking**: Python wrapper script with HierarchicalChunker for metadata preservation
3. **Harvard Variants**: US style (month-day-year, double quotes), include DOI/URL when available, modular for future citation styles
4. **Error Recovery**: Resume capability - successfully ingested documents retained, re-run continues from failure point

---

## Step 5: Generate Implementation Plan

Run the plan command:

```
/speckit.plan

Create implementation plan for Academic Quote Extractor with these technical decisions:

LANGUAGE/RUNTIME:
- Go 1.25.6+ (latest stable)
- Python 3.11+ (only for chunk_helper.py script)

DEPENDENCIES:
- github.com/spf13/cobra (CLI framework)
- github.com/weaviate/weaviate-go-client/v4 (Weaviate client)
- github.com/mattn/go-sqlite3 (SQLite driver)

DOCKER SERVICES:
- Docling: quay.io/docling-project/docling-serve:latest on port 5001
- Weaviate: cr.weaviate.io/semitechnologies/weaviate:1.27.0 on port 8080 with text2vec-ollama module
- Ollama: ollama/ollama:latest on port 11434 with nomic-embed-text model

STORAGE:
- SQLite database at ./quotes.db (default, configurable via --db flag)
- Weaviate for vector storage with text2vec-ollama vectorizer (local embeddings)

TESTING:
- Go standard testing with testify/assert
- Integration tests require Docker services running
- Contract tests for Docling, Weaviate, and Ollama APIs

PROJECT STRUCTURE (single project):
cmd/aqe/main.go
internal/
├── cli/          # Command implementations
├── docling/      # Docling HTTP client
├── claude/       # Claude Code CLI wrapper
├── search/       # Weaviate client
├── store/        # SQLite operations
├── harvard/      # Reference formatter (modular for future citation styles)
└── models/       # Domain types
scripts/
└── chunk_helper.py  # Python wrapper for Docling HierarchicalChunker
tests/
├── contract/
├── integration/
└── unit/

CLAUDE CODE INTEGRATION:
- Wrap claude CLI binary
- Pass structured JSON prompt with topic + chunks
- Parse JSON response for chunk IDs + explanations
- Strict output format validation

CHUNKING CONFIGURATION:
- Use Docling HierarchicalChunker via Python wrapper (scripts/chunk_helper.py)
- Target chunk size: ~800 tokens with 25% overlap (~200 tokens)
- Preserves section headings, page numbers, bounding boxes
- Overlap ensures sentences/phrases not split at boundaries

EMBEDDING PIPELINE:
- Weaviate auto-generates embeddings via text2vec-ollama module on insert
- No explicit embedding calls needed in Go code
- Ollama runs nomic-embed-text model (768 dimensions, 2K context window)
- Chosen for: 2K context (ideal for academic chunks), outperforms OpenAI ada-002, 274MB footprint

REFERENCE FORMATTING:
- Harvard US style (month-day-year, double quotation marks)
- Include DOI/URL when available
- Support: Book, JournalArticle, Website, BookChapter
- In-text: (Author, Year, p. X) or (Author et al., Year) for 3+ authors
- Modular design to support APA, MLA, Chicago in future
```

---

## Step 6: Generate Task Breakdown

Run the tasks command:

```
/speckit.tasks
```

This will read plan.md, data-model.md, and contracts/ to generate tasks.md with:
- Proper phase ordering (Setup -> Foundational -> User Stories)
- Parallel markers [P] for independent tasks
- User story labels [US1], [US2], etc.
- Exact file paths for each task

---

## Step 7: Execute Implementation

Run the implement command:

```
/speckit.implement
```

This executes tasks from tasks.md in order, respecting dependencies and parallel markers.

---

## Expected Directory Structure After Spec-Kit Init

```
academic_quote_extractor/
├── .specify/
│   ├── memory/
│   │   └── constitution.md          # Project principles
│   ├── scripts/
│   │   ├── check-prerequisites.sh
│   │   ├── common.sh
│   │   ├── create-new-feature.sh
│   │   ├── setup-plan.sh
│   │   └── update-claude-md.sh
│   ├── specs/
│   │   └── 001-aqe-core/
│   │       ├── spec.md              # Feature specification
│   │       ├── plan.md              # Implementation plan
│   │       ├── research.md          # Technical research
│   │       ├── data-model.md        # Entity definitions
│   │       ├── quickstart.md        # Validation scenarios
│   │       ├── tasks.md             # Task breakdown
│   │       └── contracts/
│   │           ├── api-spec.json    # Docling API contract
│   │           └── weaviate-schema.json
│   └── templates/
│       ├── plan-template.md
│       ├── spec-template.md
│       └── tasks-template.md
├── cmd/
│   └── aqe/
│       └── main.go
├── internal/
│   ├── cli/
│   │   ├── init.go
│   │   ├── ingest.go
│   │   ├── extract.go
│   │   ├── export.go
│   │   └── meta.go
│   ├── docling/
│   │   ├── client.go
│   │   └── types.go
│   ├── claude/
│   │   └── wrapper.go
│   ├── search/
│   │   └── weaviate.go
│   ├── store/
│   │   └── sqlite.go
│   ├── harvard/
│   │   └── formatter.go
│   └── models/
│       ├── document.go
│       ├── chunk.go
│       └── quote.go
├── scripts/
│   └── chunk_helper.py
├── tests/
│   ├── contract/
│   ├── integration/
│   └── unit/
├── docker-compose.yml
├── go.mod
├── go.sum
├── CLAUDE.md                        # Auto-updated by spec-kit
├── SPEC-v2.md                       # Original specification (reference)
├── DOCLING_HOWTO.md                 # Docling documentation (reference)
└── README.md
```

---

## Key Commands Reference

| Command | Purpose |
|---------|---------|
| `specify init . --ai claude` | Initialize spec-kit in current directory |
| `specify check` | Verify required tools are installed |
| `/speckit.constitution` | Create/update project principles |
| `/speckit.specify <description>` | Create feature specification |
| `/speckit.clarify` | Structured Q&A to reduce ambiguity |
| `/speckit.plan <tech details>` | Generate implementation plan |
| `/speckit.tasks` | Generate task breakdown from plan |
| `/speckit.implement` | Execute tasks to build feature |
| `/speckit.analyze` | Cross-artifact consistency check |
| `/speckit.checklist` | Generate quality validation checklist |

---

## Environment Variables

```bash
# Optional: Override feature detection for non-Git repos
export SPECIFY_FEATURE="001-aqe-core"

# Optional: GitHub token for API requests
export GH_TOKEN="ghp_..."

# Note: No OPENAI_API_KEY needed - embeddings generated locally via Ollama
```

---

## Docker Services Setup

Before running ingest/extract commands, start the required services:

```bash
# Start Docling and Weaviate
docker-compose up -d

# Verify services are running
curl http://localhost:5001/health      # Docling
curl http://localhost:8080/v1/.well-known/ready  # Weaviate
```

**docker-compose.yml** (should be in project root):

```yaml
version: '3.8'

services:
  docling:
    image: quay.io/docling-project/docling-serve
    ports:
      - "5001:5001"
    environment:
      - DOCLING_SERVE_ENABLE_UI=false
    volumes:
      - docling-cache:/app/models

  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
    volumes:
      - ollama-models:/root/.ollama
    # Pull embedding model on first run:
    # docker exec -it ollama ollama pull nomic-embed-text
    # Model: 274MB, 768 dimensions, 2K context window - ideal for academic passages

  weaviate:
    image: cr.weaviate.io/semitechnologies/weaviate:1.27.0
    ports:
      - "8080:8080"
      - "50051:50051"
    environment:
      QUERY_DEFAULTS_LIMIT: 25
      AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED: 'true'
      PERSISTENCE_DATA_PATH: '/var/lib/weaviate'
      DEFAULT_VECTORIZER_MODULE: 'text2vec-ollama'
      ENABLE_MODULES: 'text2vec-ollama'
      OLLAMA_API_ENDPOINT: 'http://ollama:11434'
      CLUSTER_HOSTNAME: 'node1'
    volumes:
      - weaviate-data:/var/lib/weaviate
    depends_on:
      - ollama

volumes:
  docling-cache:
  ollama-models:
  weaviate-data:
```

---

## Critical Design Decisions (from SPEC-v2.md)

### 1. Deterministic Quote Retrieval

```
WRONG:  Claude generates: "The author said something like..."
RIGHT:  Claude outputs: [QUOTE:chunk_abc123]
        System replaces with verbatim text from database
```

### 2. Hybrid RAG Over Pure Agentic

| Factor | Hybrid RAG | Agentic Search |
|--------|------------|----------------|
| Latency | ~2-3 seconds | 5-15 seconds |
| API Cost | 1-2 LLM calls | 5-10 LLM calls |
| Accuracy | 85-90% | 90-95% |

The 5% accuracy gain doesn't justify 5-10x cost increase for structured academic documents.

### 3. SQLite Over Dolt

For a single-user CLI tool, SQLite's simplicity and speed outweigh Dolt's version control features. Schema designed to be Dolt-compatible for future migration.

---

## Validation Checklist

Before marking implementation complete:

- [ ] `aqe ingest ./test-sources/` processes PDF and DOCX without errors
- [ ] `aqe extract "test topic"` returns quotes with valid Harvard references
- [ ] All returned quotes match verbatim text in source documents (no hallucination)
- [ ] `aqe export 1 --format json` produces valid, parseable JSON
- [ ] `aqe export 1 --format markdown` renders correctly
- [ ] In-text citations follow format: (Author, Year, p. X)
- [ ] Full references are complete with all required fields
- [ ] CLI exits with code 0 on success, non-zero on error
- [ ] Error messages are actionable for users
- [ ] Integration tests pass with Docker services running
- [ ] Contract tests verify Docling and Weaviate API compatibility
