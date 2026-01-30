# User Quickstart

Get from zero to extracting academic quotes in under 10 minutes.

## Prerequisites

You need the following installed on your machine:

| Requirement | Minimum Version | Check Command |
|-------------|----------------|---------------|
| Docker | 20.10+ | `docker --version` |
| Docker Compose | 2.0+ | `docker compose version` |
| Go | 1.25+ | `go version` |
| Python 3 | 3.9+ | `python3 --version` |
| Claude CLI | Latest | `claude --version` |

**Python packages** (for document chunking):

```bash
pip3 install "docling>=2.70.0" "docling-core>=2.0.0"
```

## Step 1: Start Services

AQE depends on three Docker services: Docling (document parsing), Weaviate (vector search), and Ollama (embeddings).

```bash
docker-compose up -d
```

Wait for services to become healthy, then pull the embedding model (first time only):

```bash
docker exec -it ollama ollama pull nomic-embed-text
```

Verify everything is running:

```bash
./aqe status
```

Or check manually:

```bash
curl http://localhost:5001/health           # Docling
curl http://localhost:8080/v1/.well-known/ready  # Weaviate
curl http://localhost:11434/api/version      # Ollama
```

## Step 2: Build the CLI

```bash
go build -o aqe ./cmd/aqe
```

This produces a single binary `aqe` in your project directory.

## Step 3: Ingest Documents

Place your academic documents (PDF, DOCX, TXT, or Markdown) in a folder and ingest them:

```bash
# Ingest an entire directory
./aqe ingest ./sources/

# Ingest a single file
./aqe ingest paper.pdf

# Ingest with metadata overrides (if auto-detection doesn't work)
./aqe ingest paper.pdf --title "Economics 101" --author "Smith, J." --year 2023
```

**Expected output:**

```
Processing: paper.pdf
Ingested 1 documents, 2920 chunks
```

If you re-ingest the same file, duplicates are detected automatically:

```
Processing: paper.pdf
  Skipping (already ingested): paper.pdf
Ingested 0 documents, 0 chunks
```

The system will:
- Parse each document through Docling
- Split text into chunks preserving page numbers and section headings
- Generate vector embeddings via Ollama
- Store everything in SQLite and Weaviate

If ingestion is interrupted, re-running the command continues where it left off.

**Supported formats**: `.pdf`, `.docx`, `.txt`, `.md`

**Note**: PDF ingestion can take several minutes for large documents. The tool
blocks until Docling finishes parsing -- there is no progress bar.

## Step 4: Extract Quotes

Run an extraction by providing your research topic:

```bash
./aqe extract "cultural bias in technology and algorithms"
```

**Expected output:**

```
Searching for relevant quotes...
Found 50 candidate chunks
Scoring relevance with Claude...

Extraction #1: "cultural bias in technology and algorithms"
Retrieved 20 quotes (relevance >= 60)

Quote 1 (Relevance: 92/100)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
"Despite their seemingly objective nature, algorithms can, and often do,
reflect the biases of their creators and the data they are trained on..."

— (Walton, 2024)

Why relevant: Highly quotable passage explaining how algorithms reflect creator
biases and training data biases, with specific domains where biased algorithms
cause real-world harm.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Quote 2 (Relevance: 90/100)
...

Saved as extraction #1. Export with: aqe export 1
```

Each extraction is **automatically saved** with an ID (here `#1`). You'll use
this ID to export the results in the next step.

### Customizing Extraction

```bash
# Limit to 5 quotes with high relevance
./aqe extract "climate change policy" --max-quotes 5 --min-relevance 80
```

| Flag | Default | Description |
|------|---------|-------------|
| `--max-quotes` | 20 | Maximum number of quotes to return |
| `--min-relevance` | 60 | Minimum relevance score (0-100) |

## Step 5: Export Results

Each extraction is saved with an auto-incrementing ID. To see all your
extractions, run:

```bash
./aqe list
```

**Expected output:**

```
Saved Extractions:

  #1: "cultural bias in technology and algorithms"
      20 quotes | 2026-01-30
      Export: aqe export 1
```

The listing shows each extraction's **ID**, topic, quote count, and date, plus
a ready-to-copy export command.

Now export using the extraction ID:

```bash
# Markdown (default) -- blockquotes with citations and bibliography
./aqe export 1

# JSON -- structured data for further processing
./aqe export 1 --format json

# BibTeX -- for reference managers
./aqe export 1 --format bibtex

# Write to a file instead of stdout
./aqe export 1 --format markdown --output quotes.md
# => Exported to quotes.md
```

### Markdown Output (actual)

```markdown
# Quotes: cultural bias in technology and algorithms

*Extracted: January 30, 2026*

---

> "Despite their seemingly objective nature, algorithms can, and often do,
> reflect the biases of their creators and the data they are trained on..."
>
> — (Walton, 2024)
>
> **Relevance (92/100):** Highly quotable passage explaining how algorithms
> reflect creator biases and training data biases.

---

## Bibliography

Walton, Devan J. (2024) *Culturally Responsive Computing: An Introduction
into Computer Science, Security, and Technology*.
```

### JSON Output (actual)

```json
{
  "extraction": {
    "id": 1,
    "topic": "cultural bias in technology and algorithms",
    "created_at": "2026-01-30T00:12:43Z"
  },
  "quotes": [
    {
      "text": "Despite their seemingly objective nature...",
      "relevance_score": 92,
      "explanation": "Highly quotable passage explaining...",
      "in_text": "(Walton, 2024)",
      "full_reference": "Walton, Devan J. (2024) *Culturally Responsive Computing...*.",
      "document": {
        "id": 1,
        "filename": "Culturally-Responsive-Computing-...md",
        "title": "Culturally Responsive Computing...",
        "authors": ["Walton, Devan J."],
        "year": 2024
      }
    }
  ]
}
```

### BibTeX Output (actual)

```bibtex
% Bibliography for: cultural bias in technology and algorithms
% Extracted: 2026-01-30

@misc{walton2024,
  author = {Walton, Devan J.},
  title = {Culturally Responsive Computing: An Introduction into Computer Science, Security, and Technology},
  year = {2024},
}
```

## Step 6: Fix Missing Metadata

If auto-detection missed metadata for any documents, fix it interactively:

```bash
./aqe meta fix
```

**Expected output (missing metadata):**

```
Document: my-paper.pdf
Current metadata:
  Title: my-paper
  Author: (missing)
  Year: (missing)

Enter author (or press Enter to skip): Smith, J.
Enter year (or press Enter to skip): 2023

Updated metadata for my-paper.pdf
```

**Expected output (all complete):**

```
All documents have complete metadata.
```

Complete metadata ensures accurate Harvard citations in your exports.

## Full Workflow Summary

```bash
# 1. Start services (once)
docker-compose up -d
docker exec -it ollama ollama pull nomic-embed-text  # first time only

# 2. Build
go build -o aqe ./cmd/aqe

# 3. Ingest
./aqe ingest paper.pdf --title "My Paper" --author "Smith, J." --year 2024
# => Ingested 1 documents, 2920 chunks

# 4. Fix metadata if needed
./aqe meta fix

# 5. Extract quotes (note the extraction ID in the output)
./aqe extract "your research topic"
# => ...
# => Saved as extraction #1. Export with: aqe export 1

# 6. List extractions to see IDs
./aqe list

# 7. Export using the extraction ID
./aqe export 1 --format markdown --output quotes.md
# => Exported to quotes.md
```

## Global Flags

These flags apply to all commands:

| Flag | Default | Description |
|------|---------|-------------|
| `--db` | `./quotes.db` | Path to SQLite database |
| `--debug` | `false` | Enable debug output |

## Troubleshooting

### "Service unavailable" errors

Make sure Docker services are running:

```bash
docker-compose up -d
./aqe status
```

### No quotes returned

- Check that documents were ingested successfully: `./aqe list` should show extractions, and ingestion should have reported chunk counts.
- Try a broader topic or lower the minimum relevance: `--min-relevance 40`
- Ensure the topic is related to your ingested documents.

### Missing or incorrect citations

- Run `./aqe meta fix` to fill in missing author, title, or year fields.
- Use `--author`, `--title`, and `--year` flags during ingestion for documents where auto-detection is unreliable.

### Build fails with CGO errors

AQE uses SQLite via CGO. Ensure you have a C compiler installed:

- **macOS**: `xcode-select --install`
- **Linux**: `sudo apt-get install gcc` (or equivalent)

### Embedding model not found

Pull the model into Ollama:

```bash
docker exec -it ollama ollama pull nomic-embed-text
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | User error (invalid input, file not found) |
| 2 | System error (service unavailable, database issue) |
