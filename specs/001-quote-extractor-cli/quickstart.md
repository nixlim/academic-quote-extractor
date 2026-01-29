# Quickstart: Academic Quote Extractor CLI

**Feature**: 001-quote-extractor-cli
**Purpose**: Validation scenarios to verify the implementation works end-to-end

## Prerequisites

### 1. Install Go 1.25.6+

```bash
# macOS with Homebrew
brew install go

# Verify installation
go version
# Expected: go version go1.25.6 darwin/arm64 (or later)
```

### 2. Install Python 3.11+

```bash
# macOS with Homebrew
brew install python@3.11

# Verify installation
python3 --version
# Expected: Python 3.11.x or later
```

### 3. Install Claude Code CLI

```bash
# Follow installation instructions at https://claude.ai/code
# Verify installation
claude --version
```

### 4. Start Docker Services

```bash
# From project root
docker-compose up -d

# Wait for services to be ready (~30 seconds for first start)
# Docling needs to download models on first run

# Verify services
curl http://localhost:5001/health          # Docling
curl http://localhost:8080/v1/.well-known/ready  # Weaviate
curl http://localhost:11434/api/version    # Ollama

# Pull embedding model (first time only)
docker exec -it ollama ollama pull nomic-embed-text
```

### 5. Build the CLI

```bash
# From project root
go build -o aqe ./cmd/aqe

# Or install to $GOPATH/bin
go install ./cmd/aqe
```

---

## Scenario 1: Basic Document Ingestion

**Goal**: Verify PDF ingestion with automatic metadata detection

### Setup

Create a test directory with sample PDF:
```bash
mkdir -p ./test-sources
# Add a sample academic PDF to ./test-sources/
```

### Execute

```bash
./aqe ingest ./test-sources/
```

### Expected Output

```
Processing: test-sources/sample-paper.pdf
  Detected: "The Impact of Social Media" by Smith, J. (2023)
  Chunks: 47
  
Ingested 1 document, 47 chunks
```

### Verify

```bash
# Check SQLite database
sqlite3 quotes.db "SELECT filename, title, authors FROM documents;"
# Expected: sample-paper.pdf|The Impact of Social Media|["Smith, J."]

sqlite3 quotes.db "SELECT COUNT(*) FROM chunks;"
# Expected: 47
```

---

## Scenario 2: Ingestion with Manual Metadata

**Goal**: Verify metadata override via CLI flags

### Execute

```bash
./aqe ingest ./test-sources/textbook.pdf \
  --title "Economics 101" \
  --author "Johnson, M." \
  --year 2022
```

### Expected Output

```
Processing: test-sources/textbook.pdf
  Using provided metadata: "Economics 101" by Johnson, M. (2022)
  Chunks: 156
  
Ingested 1 document, 156 chunks
```

---

## Scenario 3: Duplicate Detection

**Goal**: Verify system prevents re-ingestion of same file

### Execute

```bash
# Ingest same file again
./aqe ingest ./test-sources/sample-paper.pdf
```

### Expected Output

```
Skipping: test-sources/sample-paper.pdf (already ingested)

Ingested 0 documents, 0 chunks
```

---

## Scenario 4: Quote Extraction

**Goal**: Verify hybrid search and LLM relevance scoring

### Execute

```bash
./aqe extract "Impact of social media on political polarization"
```

### Expected Output

```
Searching for relevant quotes...
Found 47 candidate chunks
Scoring relevance with Claude...

Extraction #1: "Impact of social media on political polarization"
Retrieved 12 quotes (relevance >= 60)

Quote 1 (Relevance: 92/100)
━━━━━━━━━━━━━━━━━━━━━━━━━━━
"Social media algorithms have been shown to increase exposure to 
ideologically congruent content by 34% compared to chronological feeds."

— Smith (2023, p. 42)

Why relevant: Provides quantitative evidence that social media algorithms 
contribute to polarization by limiting exposure to diverse viewpoints.

━━━━━━━━━━━━━━━━━━━━━━━━━━━

[... more quotes ...]

Saved as extraction #1. Export with: aqe export 1
```

---

## Scenario 5: Custom Extraction Parameters

**Goal**: Verify configurable thresholds

### Execute

```bash
./aqe extract "Climate change policy effectiveness" \
  --max-quotes 5 \
  --min-relevance 80
```

### Expected Output

```
Searching for relevant quotes...
Found 89 candidate chunks
Scoring relevance with Claude...

Extraction #2: "Climate change policy effectiveness"
Retrieved 5 quotes (relevance >= 80)

[... 5 quotes with scores >= 80 ...]

Saved as extraction #2. Export with: aqe export 2
```

---

## Scenario 6: Export to Markdown

**Goal**: Verify Markdown formatting with Harvard references

### Execute

```bash
./aqe export 1 --format markdown
```

### Expected Output

```markdown
# Quotes: Impact of social media on political polarization

*Extracted: January 29, 2026*

---

> "Social media algorithms have been shown to increase exposure to 
> ideologically congruent content by 34% compared to chronological feeds."
>
> — Smith (2023, p. 42)
>
> **Relevance (92/100):** Provides quantitative evidence that social media 
> algorithms contribute to polarization by limiting exposure to diverse viewpoints.

---

[... more quotes ...]

## Bibliography

Smith, J. (2023) "The Impact of Social Media on Political Discourse," 
*Journal of Communication Studies*, 45(2), pp. 38-56.
```

---

## Scenario 7: Export to JSON

**Goal**: Verify valid JSON output

### Execute

```bash
./aqe export 1 --format json > quotes.json
```

### Verify

```bash
# Validate JSON
cat quotes.json | jq .

# Check structure
cat quotes.json | jq '.quotes | length'
# Expected: 12

cat quotes.json | jq '.quotes[0].in_text'
# Expected: "(Smith, 2023, p. 42)"
```

---

## Scenario 8: Export to BibTeX

**Goal**: Verify BibTeX bibliography generation

### Execute

```bash
./aqe export 1 --format bibtex
```

### Expected Output

```bibtex
@article{smith2023,
  author = {Smith, J.},
  title = {The Impact of Social Media on Political Discourse},
  journal = {Journal of Communication Studies},
  year = {2023},
  volume = {45},
  number = {2},
  pages = {38--56}
}
```

---

## Scenario 9: Metadata Fix

**Goal**: Verify interactive metadata completion

### Setup

Ingest a document with missing metadata:
```bash
./aqe ingest ./test-sources/unknown-doc.pdf
# System detects missing author/year
```

### Execute

```bash
./aqe meta fix
```

### Expected Interaction

```
Document: unknown-doc.pdf
Current metadata:
  Title: Research Methods Overview
  Author: (missing)
  Year: (missing)

Enter author (or press Enter to skip): Williams, K.
Enter year (or press Enter to skip): 2021

Updated: unknown-doc.pdf
  Author: Williams, K.
  Year: 2021

All documents now have complete metadata.
```

---

## Scenario 10: Error Handling - Service Unavailable

**Goal**: Verify meaningful error messages

### Setup

Stop Weaviate:
```bash
docker stop weaviate
```

### Execute

```bash
./aqe extract "Any topic"
```

### Expected Output

```
Error: Search service unavailable

The Weaviate service is not responding at localhost:8080.

To fix:
  1. Check if Docker is running: docker ps
  2. Start services: docker-compose up -d
  3. Wait for Weaviate to be ready: curl http://localhost:8080/v1/.well-known/ready

Exit code: 2
```

### Cleanup

```bash
docker start weaviate
```

---

## Validation Checklist

| Scenario | Validates | Pass Criteria |
|----------|-----------|---------------|
| 1 | FR-001, FR-002, SC-001 | PDF parsed, chunks created with metadata |
| 2 | FR-001 | Manual metadata overrides work |
| 3 | FR-011 | Duplicate detection via checksum |
| 4 | FR-004, FR-005, SC-003 | Hybrid search + LLM scoring, verbatim quotes |
| 5 | FR-008, FR-009 | Configurable thresholds respected |
| 6 | FR-006, SC-004, SC-007 | Valid Harvard references, renders correctly |
| 7 | SC-006 | Valid JSON output |
| 8 | FR-006 | Valid BibTeX output |
| 9 | US4 | Interactive metadata completion |
| 10 | FR-012, SC-005 | Meaningful error message, non-zero exit |

---

## Performance Validation

### SC-001: Ingest 100-page PDF in <60 seconds

```bash
time ./aqe ingest ./test-sources/long-document.pdf
# Expected: real < 60s
```

### SC-002: Extract from 10 documents in <10 seconds

```bash
# After ingesting 10 documents
time ./aqe extract "test topic"
# Expected: real < 10s
```
