# CLI Reference

Complete reference for all `aqe` commands, flags, output formats, and error handling.

> This document was produced by exercising every command against a live system
> with Docker services running (Docling, Weaviate, Ollama) and a real ingested PDF.

---

## Global Flags

These flags apply to **all** commands and must appear before the subcommand:

```
--db string   Path to SQLite database (default "./quotes.db")
--debug       Enable debug output
-h, --help    Help for aqe
```

**Examples:**

```bash
# Use a different database
./aqe --db /path/to/other.db list

# Enable debug output (shows HTTP calls, Weaviate queries, Claude prompts)
./aqe --debug extract "my topic"

# Combine both
./aqe --db ./test.db --debug ingest paper.pdf
```

**Note on `--debug`:** Debug output includes Docling HTTP request/response details, Weaviate
insert operations with UUIDs, Claude prompt lengths, raw Claude JSON envelope output, and
chunk counts at each stage. Very useful for diagnosing pipeline issues.

---

## Commands

### `aqe status`

Check health of all infrastructure services and display database statistics.

```bash
./aqe status
```

**Checks performed:**
- Docling health (`http://localhost:5001/health`)
- Weaviate readiness (`http://localhost:8080/v1/.well-known/ready`)
- Ollama version (`http://localhost:11434/api/version`)
- Embedding model availability (nomic-embed-text)
- Claude CLI availability
- Docker container states
- Database statistics (documents, chunks, extractions, quotes)
- Weaviate index node/shard status

**Sample output:**

```
AQE Infrastructure Status
============================================================

  Docling      [OK] http://localhost:5001/health  (27ms)
               status: ok
  Weaviate     [OK] http://localhost:8080/v1/.well-known/ready  (4ms)
               v1.27.0
  Ollama       [OK] http://localhost:11434/api/version  (11ms)
               v0.15.2
  Embeddings   [OK] nomic-embed-text
               model loaded (137M, 262MB)
  Claude CLI   [OK] claude
               2.1.25 (Claude Code)

Docker Containers
------------------------------------------------------------
docling     Up 29 minutes (healthy)     0.0.0.0:5001->5001/tcp
weaviate    Up 45 minutes               0.0.0.0:8080->8080/tcp
ollama      Up 45 minutes               0.0.0.0:11434->11434/tcp

Database
------------------------------------------------------------
  Path:        ./quotes.db
  Documents:   1
    #1   Culturally Responsive Computing: An Introductio...  Walton, Devan J. (2024) [2920 chunks]
  Chunks:      2920
  Extractions: 1
  Quotes:      20
    #1   cultural bias in technology and algorithms          [20 quotes]

Weaviate Index
------------------------------------------------------------
  Node:   node1 (HEALTHY)
  Shards: 1 shard(s)
```

**When to use:** Run this first to verify your environment is ready before ingesting
or extracting. Also useful for debugging when other commands fail.

---

### `aqe ingest <path>`

Parse and index academic documents for quote extraction.

```
Usage:  aqe ingest <path> [flags]

Flags:
  --author string   Document author (overrides auto-detection)
  --title string    Document title (overrides auto-detection)
  --year int        Publication year (overrides auto-detection)
  -h, --help        Help for ingest
```

**Accepts:** Exactly 1 argument -- a file path or directory path.

**Supported formats:** `.pdf`, `.docx`, `.txt`, `.md`

**Pipeline:**
1. Docling parses the document (HTTP POST to docling-serve)
2. Python HierarchicalChunker splits into semantic chunks (falls back to Docling text items if Python chunker is unavailable)
3. Chunks are inserted into Weaviate (auto-embedded via nomic-embed-text)
4. Document record and chunk text stored in SQLite

**Examples:**

```bash
# Ingest a single PDF
./aqe ingest paper.pdf

# Ingest with full metadata overrides
./aqe ingest "Culturally-Responsive-Computing.pdf" \
  --title "Culturally Responsive Computing: An Introduction into Computer Science, Security, and Technology" \
  --author "Walton, Devan J." \
  --year 2024

# Ingest all supported files in a directory
./aqe ingest ./sources/

# Ingest with debug to see the full pipeline
./aqe ingest paper.pdf --debug
```

**Sample output (success):**

```
Processing: paper.pdf
Ingested 1 documents, 2920 chunks
```

**Sample output (debug mode):**

```
[DEBUG] Weaviate class Chunk already exists
Processing: crc_test_text.md
[DEBUG] Docling request: POST http://localhost:5001/v1/convert/file (file: crc_test_text.md)
[DEBUG] Docling response status: 200
  Using provided metadata: "CRC Test" by Test, A. (2025)
[DEBUG] Chunker failed, falling back to text items: run chunker: exit status 1
[DEBUG] Weaviate inserted chunk #/texts/0 with UUID 007d017a-c593-4752-abfc-13b5c4df01ff
...
Ingested 1 documents, 48 chunks
```

**Duplicate detection:** Documents are checksummed (SHA-256). Re-ingesting the same file
produces:

```
Processing: paper.pdf
  Skipping (already ingested): paper.pdf
Ingested 0 documents, 0 chunks
```

**Error: file not found:**

```
$ ./aqe ingest nonexistent.pdf
Error: get files: stat path: stat nonexistent.pdf: no such file or directory
```

**Error: Docling parse failure:**

```
Warning: file.txt - parse document: convert failed: status 404, body: {"detail":"..."}
Ingested 0 documents, 0 chunks
```

**Important notes:**
- PDF ingestion through Docling can be slow (minutes for large documents). The tool does not
  show a progress bar -- it blocks until Docling returns.
- If the Python chunker is not installed (`docling` / `docling-core` packages), the system
  falls back to Docling's raw text items. Debug mode shows this:
  `[DEBUG] Chunker failed, falling back to text items: run chunker: exit status 1`
- Metadata overrides (`--title`, `--author`, `--year`) apply to all files when ingesting a
  directory. Use single-file ingestion if documents have different metadata.
- Resumable: if batch ingestion is interrupted, re-running skips already-ingested files.

---

### `aqe extract <topic>`

Find relevant quotes from ingested documents for a research topic.

```
Usage:  aqe extract <topic> [flags]

Flags:
  --max-quotes int      Maximum number of quotes to return (default 20)
  --min-relevance int   Minimum relevance score 0-100 (default 60)
  -h, --help            Help for extract
```

**Accepts:** Exactly 1 argument -- the research topic as a quoted string.

**Pipeline:**
1. Weaviate hybrid search (BM25 + vector, alpha=0.5, top 50 candidates)
2. All 50 candidates sent to Claude CLI for relevance scoring
3. Claude returns chunk IDs with relevance scores (0-100) and explanations
4. Quotes above `--min-relevance` are kept, capped at `--max-quotes`
5. Extraction saved to SQLite with an auto-incrementing ID

**Examples:**

```bash
# Default extraction (up to 20 quotes, relevance >= 60)
./aqe extract "impact of social media on political polarization"

# Narrow extraction (fewer, higher-quality quotes)
./aqe extract "digital divide and internet access inequality" --max-quotes 3 --min-relevance 70

# Debug mode to see Claude's raw response
./aqe extract "cultural bias" --debug
```

**Sample output:**

```
Searching for relevant quotes...
Found 50 candidate chunks
Scoring relevance with Claude...

Extraction #2: "digital divide and internet access inequality"
Retrieved 3 quotes (relevance >= 70)

Quote 1 (Relevance: 95/100)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
"The digital divide refers to the gap between those with access to modern
information and communication technology and those without..."

— (Walton, 2024)

Why relevant: Directly defines the digital divide and explains how disparities
in network access create cultural fragmentation.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Quote 2 (Relevance: 93/100)
...

Saved as extraction #2. Export with: aqe export 2
```

**Debug output includes:**
- Weaviate query parameters (query, limit, alpha)
- Number of candidate chunks found
- Claude prompt length in characters
- Raw Claude JSON envelope (full `--output-format json` response)
- Number of selected chunks returned by Claude

**Cost note:** Each extraction calls Claude once. Debug output shows the cost:
`"total_cost_usd": 0.10` for a typical extraction with 50 candidate chunks.

---

### `aqe list`

List all saved extractions with IDs, topics, quote counts, and dates.

```
Usage:  aqe list [flags]
```

**Examples:**

```bash
./aqe list
```

**Sample output (with extractions):**

```
Saved Extractions:

  #2: "digital divide and internet access inequality"
      3 quotes | 2026-01-30
      Export: aqe export 2

  #1: "cultural bias in technology and algorithms"
      20 quotes | 2026-01-30
      Export: aqe export 1
```

**Sample output (empty database):**

```
No extractions found.
Run 'aqe extract <topic>' to create your first extraction.
```

**Note:** Extractions are listed in reverse chronological order (newest first).
The output includes a ready-to-copy export command for each extraction.

---

### `aqe export <extraction-id>`

Export a saved extraction in Markdown, JSON, or BibTeX format.

```
Usage:  aqe export <extraction-id> [flags]

Flags:
  --format string   Output format: markdown, json, bibtex (default "markdown")
  --output string   Output file (default: stdout)
  -h, --help        Help for export
```

**Accepts:** Exactly 1 argument -- the extraction ID (integer from `aqe list`).

#### Markdown Format (default)

```bash
./aqe export 1
./aqe export 1 --format markdown
```

Produces:

```markdown
# Quotes: cultural bias in technology and algorithms

*Extracted: January 30, 2026*

---

> "Despite their seemingly objective nature, algorithms can, and often do,
> reflect the biases of their creators..."
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

#### JSON Format

```bash
./aqe export 1 --format json
```

Produces valid, parseable JSON:

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

#### BibTeX Format

```bash
./aqe export 1 --format bibtex
```

Produces:

```bibtex
% Bibliography for: cultural bias in technology and algorithms
% Extracted: 2026-01-30

@misc{walton2024,
  author = {Walton, Devan J.},
  title = {Culturally Responsive Computing: An Introduction into Computer Science, Security, and Technology},
  year = {2024},
}
```

#### Writing to a File

```bash
./aqe export 1 --output quotes.md
./aqe export 1 --format json --output quotes.json
```

Produces: `Exported to quotes.md`

#### Error: invalid extraction ID

```
$ ./aqe export 999
Error: Invalid extraction ID: 999

Available extractions:
  1: "cultural bias in technology and algorithms" (2026-01-30)
```

#### Error: invalid format

```
$ ./aqe export 1 --format invalid
Error: unknown format: invalid (use markdown, json, or bibtex)
```

---

### `aqe meta fix`

Interactively fix missing document metadata (author, title, year).

```
Usage:  aqe meta fix [flags]
```

**Example:**

```bash
./aqe meta fix
```

**Sample output (documents with missing metadata):**

```
Document: AGENTS.md
Current metadata:
  Title: AGENTS
  Author: (missing)
  Year: (missing)

Enter author (or press Enter to skip): Smith, J.
Enter year (or press Enter to skip): 2024

Updated metadata for AGENTS.md

1 documents still have incomplete metadata.
```

**Sample output (all metadata complete):**

```
All documents have complete metadata.
```

**Note:** This command is interactive -- it reads from stdin. It only prompts for
fields that are missing, not fields that are already set.

---

### `aqe meta`

Parent command for metadata management.

```bash
./aqe meta --help
```

Currently has one subcommand: `fix`.

---

## Exit Codes

| Code | Meaning | Example Triggers |
|------|---------|------------------|
| `0` | Success | Command completed normally |
| `1` | User error | Missing arguments, invalid file path, invalid format, invalid extraction ID |
| `2` | System error | Service unavailable, database corruption |

All error cases print to stderr and show the command usage when applicable.

---

## Typical Workflows

### First-Time Setup

```bash
# 1. Start Docker services
docker-compose up -d

# 2. Pull embedding model (one-time)
docker exec -it ollama ollama pull nomic-embed-text

# 3. Build
go build -o aqe ./cmd/aqe

# 4. Verify everything works
./aqe status
```

### Ingest, Extract, Export

```bash
# Ingest a document with metadata
./aqe ingest "paper.pdf" \
  --title "Culturally Responsive Computing" \
  --author "Walton, Devan J." \
  --year 2024
# => Processing: paper.pdf
# => Ingested 1 documents, 2920 chunks

# Fix any auto-detection gaps
./aqe meta fix

# Extract quotes (note the extraction ID in the output)
./aqe extract "cultural bias in technology and algorithms"
# => ...
# => Saved as extraction #1. Export with: aqe export 1

# Check what's available (shows IDs you need for export)
./aqe list
# => Saved Extractions:
# =>
# =>   #1: "cultural bias in technology and algorithms"
# =>       20 quotes | 2026-01-30
# =>       Export: aqe export 1

# Export using the extraction ID from list/extract output
./aqe export 1 --format markdown --output quotes.md
# => Exported to quotes.md
./aqe export 1 --format json --output quotes.json
# => Exported to quotes.json
./aqe export 1 --format bibtex --output refs.bib
# => Exported to refs.bib
```

### Multiple Extractions from Same Corpus

```bash
# Each extraction creates a new saved session with its own ID
./aqe extract "digital divide"
# => Saved as extraction #1. Export with: aqe export 1
./aqe extract "algorithmic bias in hiring"
# => Saved as extraction #2. Export with: aqe export 2
./aqe extract "privacy concerns in facial recognition"
# => Saved as extraction #3. Export with: aqe export 3

# List all to see IDs and quote counts
./aqe list
# => Saved Extractions:
# =>
# =>   #3: "privacy concerns in facial recognition"
# =>       12 quotes | 2026-01-30
# =>       Export: aqe export 3
# =>   ...

# Export each by its ID
./aqe export 1 --output topic1.md
./aqe export 2 --output topic2.md
./aqe export 3 --output topic3.md
```

### Debugging a Failed Pipeline

```bash
# Check infrastructure first
./aqe status

# Run with debug to see exactly where things fail
./aqe --debug ingest problematic.pdf 2>&1 | tee debug.log

# Common debug output patterns:
#   [DEBUG] Docling request: POST ...          -- Docling is being called
#   [DEBUG] Docling response status: 404       -- Docling failed
#   [DEBUG] Chunker failed, falling back ...   -- Python chunker not installed
#   [DEBUG] Weaviate inserted chunk ...        -- Successful chunk indexing
#   [DEBUG] Claude prompt length: N chars      -- Claude is being called
#   [DEBUG] Raw Claude output (N bytes): ...   -- Full Claude response
```

---

## Performance Characteristics

Based on live testing:

| Operation | Typical Duration | Notes |
|-----------|-----------------|-------|
| Ingest (small .md file) | 5-15 seconds | Docling parse + Weaviate insert |
| Ingest (100-page PDF) | 1-5 minutes | Dominated by Docling PDF parsing |
| Extract (50 candidates) | 20-30 seconds | Dominated by Claude API call |
| Export | < 1 second | Pure database reads + formatting |
| List | < 1 second | Database query only |
| Status | 1-2 seconds | Health check HTTP calls |

---

## Service Ports

| Service | Port | Protocol |
|---------|------|----------|
| Docling | 5001 | HTTP |
| Weaviate | 8080 | HTTP |
| Weaviate | 50051 | gRPC |
| Ollama | 11434 | HTTP |

These are hardcoded in the application. The SQLite database path is configurable
via `--db` (default: `./quotes.db`).
