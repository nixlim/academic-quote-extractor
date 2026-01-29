# Academic Quote Extractor v2 - Revised Specification

## Tightened Scope + Docling Integration

---

## 1. Simplified Scope

**Single Purpose**: Extract quotes from academic sources with Harvard references and relevance explanations.

**Output per quote**:
```json
{
  "quote": "Exact verbatim text from source",
  "reference": "Smith, J. (2023) Title. Publisher, p. 142",
  "in_text": "(Smith, 2023, p. 142)",
  "relevance": "Supports the argument that X because Y"
}
```

**That's it.** No theme clustering, no argument mapping, no contradiction detection.

---

## 2. Why Docling is the Right Choice

### 2.1 Docling vs GROBID Comparison

| Capability | Docling | GROBID |
|------------|---------|--------|
| **Format Support** | PDF, DOCX, PPTX, XLSX, HTML, images | PDF only |
| **Output Format** | Unified DoclingDocument → JSON/MD | TEI-XML |
| **Chunking** | Built-in HierarchicalChunker | Manual |
| **Page/BBox Metadata** | ✅ Native | ✅ Native |
| **Table Extraction** | ✅ TableFormer model | ✅ Good |
| **OCR** | ✅ Multiple backends | ✅ |
| **Local Execution** | ✅ Air-gapped capable | ✅ |
| **RAG Integrations** | LlamaIndex, LangChain, Weaviate | Manual |
| **Maintenance** | Active (51k+ stars, IBM/LF AI) | Active |
| **Go Integration** | REST API (docling-serve) | REST API |

**Verdict**: Docling wins for multi-format support, built-in chunking with metadata preservation, and native RAG framework integrations.

### 2.2 Docling's Key Advantages for Quote Extraction

1. **Preserved Page Numbers & Bounding Boxes**: Every chunk retains `page_no`, `bbox`, and `charspan` metadata—essential for accurate citations
2. **Hierarchical Chunking**: `HierarchicalChunker` preserves document structure (sections → subsections → paragraphs)
3. **Rich Metadata per Chunk**: Section headings, document origin, and provenance travel with each chunk
4. **Multi-Format**: Students upload PDFs, DOCXs, textbook chapters—Docling handles all

---

## 3. Architecture Decision: Hybrid RAG (Not Agentic)

### 3.1 Why Hybrid RAG Over Pure Agentic Search

For academic quote extraction:

| Factor | Hybrid RAG | Agentic Search |
|--------|------------|----------------|
| **Latency** | ~2-3 seconds | 5-15 seconds |
| **API Cost** | 1-2 LLM calls | 5-10 LLM calls |
| **Accuracy** | 85-90% | 90-95% |
| **Complexity** | Moderate | High |
| **Go Ecosystem** | Excellent | Limited |

**The 5% accuracy gain doesn't justify 5-10x cost increase for our use case.** Academic documents are well-structured—semantic search with metadata filtering is highly effective.

### 3.2 The Critical Design: Deterministic Quote Retrieval

**Never let Claude generate quote text.**

```
❌ BAD:  Claude generates: "The author said something like..."
✅ GOOD: Claude outputs: [QUOTE:chunk_abc123]
         System replaces with verbatim text from database
```

This eliminates citation hallucination entirely.

---

## 4. Persistence Decision: SQLite (Not Dolt)

### 4.1 Why Not Dolt for This Project

Dolt's version control is overkill for a CLI tool. The overhead isn't justified:

| Factor | Dolt | SQLite |
|--------|------|--------|
| **Setup** | Server or embedded driver | Zero config |
| **Size** | Heavy (~100MB+) | Tiny (~2MB) |
| **Go Integration** | Good but extra dependency | Native `database/sql` |
| **Version Control** | ✅ Full Git-like | ❌ None |
| **Speed** | ~10-30% slower | Fastest embedded |
| **Use Case Fit** | Collaborative, audit trails | Single-user CLI |

**For a CLI tool used by individual students**, SQLite is simpler and faster.

### 4.2 When Dolt Would Make Sense

- Multi-user server deployment
- Need to audit extraction algorithm changes
- Research reproducibility requirements
- Collaborative quote library

**Recommendation**: Start with SQLite, design schema to be Dolt-compatible for future migration.

---

## 5. Revised Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Go CLI Application                          │
│                                                                     │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────┐  │
│  │   Ingest     │ →  │   Search     │ →  │   Extract & Format   │  │
│  │   Command    │    │   Command    │    │      Command         │  │
│  └──────────────┘    └──────────────┘    └──────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
         │                    │                       │
         ▼                    ▼                       ▼
┌─────────────────┐  ┌─────────────────┐   ┌─────────────────────────┐
│  Docling-Serve  │  │    Weaviate     │   │   Claude Code CLI       │
│  (Docker)       │  │  (Hybrid BM25   │   │   (via wrapper)         │
│                 │  │   + Vector)     │   │                         │
│  - Parse docs   │  │                 │   │   - Relevance scoring   │
│  - Chunk        │  │  - Store chunks │   │   - Reference format    │
│  - Extract meta │  │  - Search       │   │   - Explanation gen     │
└─────────────────┘  └─────────────────┘   └─────────────────────────┘
         │                    │
         └────────────────────┘
                    │
                    ▼
           ┌───────────────┐
           │    SQLite     │
           │               │
           │  - Documents  │
           │  - Chunks     │
           │  - Extracts   │
           └───────────────┘
```

---

## 6. Data Flow

### 6.1 Ingestion Pipeline

```
User drops PDFs/DOCXs into ./sources/
                │
                ▼
┌─────────────────────────────────────┐
│         Go: Ingest Command          │
│  aqe ingest --sources ./sources     │
└─────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────┐
│          Docling-Serve API          │
│  POST /v1/convert/source            │
│                                     │
│  Returns: DoclingDocument JSON      │
│  - texts[] with page_no, bbox       │
│  - tables[] with structure          │
│  - metadata (title, if detectable)  │
└─────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────┐
│         Go: Chunk Processor         │
│                                     │
│  For each DoclingDocument:          │
│  1. Call HierarchicalChunker        │
│  2. Generate embeddings (OpenAI)    │
│  3. Store in Weaviate + SQLite      │
└─────────────────────────────────────┘
```

### 6.2 Extraction Pipeline

```
User: aqe extract --topic "Impact of social media on democracy"
                │
                ▼
┌─────────────────────────────────────┐
│       Go: Extract Command           │
│                                     │
│  1. Hybrid search in Weaviate       │
│     (BM25 + vector similarity)      │
│  2. Retrieve top-K chunks           │
│  3. Include metadata (page, section)│
└─────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────┐
│      Claude Code CLI Wrapper        │
│                                     │
│  Input: Topic + Retrieved chunks    │
│  Task:                              │
│  - Score relevance (0-100)          │
│  - Output chunk IDs to use          │
│  - Generate relevance explanation   │
│                                     │
│  Output format:                     │
│  [QUOTE:chunk_id] - explanation     │
└─────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────┐
│      Go: Post-Processor             │
│                                     │
│  1. Lookup chunk_id in SQLite       │
│  2. Retrieve verbatim text          │
│  3. Format Harvard reference        │
│  4. Validate chunk exists           │
│  5. Output final JSON/Markdown      │
└─────────────────────────────────────┘
```

---

## 7. Docling Integration Details

### 7.1 Docling-Serve Setup

```bash
# Start Docling server
docker run -d \
  --name docling \
  -p 5001:5001 \
  -v ./models:/app/models \
  quay.io/docling-project/docling-serve
```

### 7.2 Go Client for Docling

```go
type DoclingClient struct {
    baseURL    string
    httpClient *http.Client
}

type ConvertRequest struct {
    Sources []Source `json:"sources"`
    Options Options  `json:"options,omitempty"`
}

type Source struct {
    Kind string `json:"kind"` // "file" or "http"
    Path string `json:"path,omitempty"`
    URL  string `json:"url,omitempty"`
}

// Convert a local file
func (c *DoclingClient) ConvertFile(path string) (*DoclingDocument, error) {
    // Multipart upload to /v1/convert/file
}

// Response contains the full DoclingDocument
type DoclingDocument struct {
    Name     string           `json:"name"`
    Texts    []TextItem       `json:"texts"`
    Tables   []TableItem      `json:"tables"`
    Pictures []PictureItem    `json:"pictures"`
    Body     NodeItem         `json:"body"`
}

type TextItem struct {
    SelfRef  string     `json:"self_ref"`  // e.g., "#/texts/42"
    Text     string     `json:"text"`
    Label    string     `json:"label"`     // "paragraph", "section_header", etc.
    Prov     []Provenance `json:"prov"`
}

type Provenance struct {
    PageNo  int     `json:"page_no"`
    BBox    BBox    `json:"bbox"`
    Charspan []int  `json:"charspan"`
}

type BBox struct {
    L           float64 `json:"l"`
    T           float64 `json:"t"`
    R           float64 `json:"r"`
    B           float64 `json:"b"`
    CoordOrigin string  `json:"coord_origin"`
}
```

### 7.3 Using Docling's HierarchicalChunker

Since chunking happens in Python/Docling, we have two options:

**Option A: Use docling-serve chunking endpoint** (if available)

**Option B: Wrap Python chunker**

```go
// Call Python script for chunking
func (c *DoclingClient) ChunkDocument(doc *DoclingDocument) ([]Chunk, error) {
    // Serialize doc to JSON
    // Call: python3 -m chunk_helper --input doc.json
    // Parse output chunks
}
```

```python
# chunk_helper.py
from docling.chunking import HierarchicalChunker
from docling_core.types.doc import DoclingDocument
import json
import sys

doc = DoclingDocument.model_validate_json(sys.stdin.read())
chunker = HierarchicalChunker()
chunks = list(chunker.chunk(doc))

# Output with metadata
for chunk in chunks:
    print(json.dumps({
        "id": chunk.meta.doc_items[0].self_ref,
        "text": chunk.text,
        "page": chunk.meta.doc_items[0].prov[0].page_no if chunk.meta.doc_items[0].prov else None,
        "headings": chunk.meta.headings,
        "origin": {
            "filename": chunk.meta.origin.filename,
        }
    }))
```

---

## 8. Claude Code CLI Wrapper

### 8.1 Wrapper Design

```go
type ClaudeCodeWrapper struct {
    binPath string // Path to claude-code CLI
}

type ExtractionTask struct {
    Topic    string   `json:"topic"`
    Chunks   []Chunk  `json:"chunks"`
}

type ExtractionResult struct {
    SelectedChunks []SelectedChunk `json:"selected_chunks"`
}

type SelectedChunk struct {
    ChunkID     string `json:"chunk_id"`
    Relevance   int    `json:"relevance"`    // 0-100
    Explanation string `json:"explanation"`
}

func (w *ClaudeCodeWrapper) ExtractQuotes(task ExtractionTask) (*ExtractionResult, error) {
    prompt := buildExtractionPrompt(task)
    
    // Call Claude Code CLI
    cmd := exec.Command(w.binPath, "--prompt", prompt)
    output, err := cmd.Output()
    
    // Parse structured output
    return parseExtractionResult(output)
}
```

### 8.2 Extraction Prompt

```
You are an academic research assistant. Your task is to identify quotes relevant to an essay topic.

ESSAY TOPIC: {{.Topic}}

AVAILABLE CHUNKS (with IDs):
{{range .Chunks}}
[{{.ID}}] (Page {{.Page}}, Section: {{.Headings | join " > "}})
"{{.Text}}"

{{end}}

INSTRUCTIONS:
1. Review each chunk for relevance to the essay topic
2. Select chunks that would make good quotes (direct evidence, key arguments, important statistics)
3. For each selected chunk, explain WHY it's relevant

OUTPUT FORMAT (JSON):
{
  "selected_chunks": [
    {
      "chunk_id": "[exact chunk ID from above]",
      "relevance": [0-100 score],
      "explanation": "[2-3 sentences explaining how this quote supports/opposes/provides evidence for the topic]"
    }
  ]
}

RULES:
- Only select chunks that are genuinely quotable (not just background)
- Prefer direct statements over summaries
- Score 80+ = essential quote, 60-79 = useful, 40-59 = marginal
- Return empty array if no relevant quotes found
```

---

## 9. Harvard Reference Generation

### 9.1 Reference Formatter (Pure Go)

```go
type ReferenceFormatter struct{}

type DocumentMeta struct {
    Title     string
    Authors   []Author
    Year      int
    Publisher string
    Journal   string
    Volume    string
    Issue     string
    Pages     string
    DOI       string
    URL       string
    SourceType SourceType
}

type SourceType int
const (
    Book SourceType = iota
    JournalArticle
    Website
    Chapter
)

func (f *ReferenceFormatter) FormatHarvard(meta DocumentMeta, pageNum int) Reference {
    var full, inText string
    
    switch meta.SourceType {
    case Book:
        // Smith, J. and Jones, M. (2023) Title of Book. Place: Publisher.
        full = fmt.Sprintf("%s (%d) %s. %s.",
            formatAuthors(meta.Authors),
            meta.Year,
            meta.Title,
            meta.Publisher,
        )
        inText = fmt.Sprintf("(%s, %d, p. %d)",
            formatAuthorsShort(meta.Authors),
            meta.Year,
            pageNum,
        )
        
    case JournalArticle:
        // Smith, J. (2023) 'Title', Journal, 45(2), pp. 12-34.
        full = fmt.Sprintf("%s (%d) '%s', %s, %s(%s), pp. %s.",
            formatAuthors(meta.Authors),
            meta.Year,
            meta.Title,
            meta.Journal,
            meta.Volume,
            meta.Issue,
            meta.Pages,
        )
        inText = fmt.Sprintf("(%s, %d, p. %d)",
            formatAuthorsShort(meta.Authors),
            meta.Year,
            pageNum,
        )
    // ... other types
    }
    
    return Reference{Full: full, InText: inText}
}

func formatAuthors(authors []Author) string {
    if len(authors) == 0 {
        return "Unknown"
    }
    if len(authors) == 1 {
        return fmt.Sprintf("%s, %s.", authors[0].LastName, authors[0].FirstInitial())
    }
    if len(authors) == 2 {
        return fmt.Sprintf("%s, %s. and %s, %s.",
            authors[0].LastName, authors[0].FirstInitial(),
            authors[1].LastName, authors[1].FirstInitial(),
        )
    }
    // 3+ authors: first author et al.
    return fmt.Sprintf("%s, %s. et al.", authors[0].LastName, authors[0].FirstInitial())
}
```

---

## 10. SQLite Schema

```sql
-- Documents table
CREATE TABLE documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename TEXT NOT NULL,
    filepath TEXT NOT NULL,
    title TEXT,
    authors TEXT,  -- JSON array
    year INTEGER,
    publisher TEXT,
    source_type TEXT,
    checksum TEXT UNIQUE,
    ingested_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Chunks table (source of truth for verbatim text)
CREATE TABLE chunks (
    id TEXT PRIMARY KEY,  -- Docling self_ref e.g., "#/texts/42"
    document_id INTEGER NOT NULL,
    text TEXT NOT NULL,
    page_num INTEGER,
    section_path TEXT,  -- JSON array of heading hierarchy
    bbox TEXT,  -- JSON for PDF highlighting
    embedding_id TEXT,  -- Reference to Weaviate UUID
    FOREIGN KEY (document_id) REFERENCES documents(id)
);

-- Extractions table (saved extraction sessions)
CREATE TABLE extractions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    topic TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Extracted quotes
CREATE TABLE extracted_quotes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    extraction_id INTEGER NOT NULL,
    chunk_id TEXT NOT NULL,
    relevance_score INTEGER,
    explanation TEXT,
    FOREIGN KEY (extraction_id) REFERENCES extractions(id),
    FOREIGN KEY (chunk_id) REFERENCES chunks(id)
);

-- Indexes
CREATE INDEX idx_chunks_document ON chunks(document_id);
CREATE INDEX idx_quotes_extraction ON extracted_quotes(extraction_id);
```

---

## 11. CLI Design

```bash
# Initialize workspace
aqe init --db ./quotes.db --weaviate localhost:8080

# Ingest documents
aqe ingest ./sources/
aqe ingest ./textbook.pdf --title "Economics 101" --author "Smith, J." --year 2023

# Extract quotes for a topic
aqe extract "The impact of social media on political polarization"
aqe extract "Climate policy effectiveness" --max-quotes 20 --min-relevance 60

# List saved extractions
aqe list

# Export extraction
aqe export 1 --format markdown > quotes.md
aqe export 1 --format json > quotes.json
aqe export 1 --format bibtex > references.bib

# Interactive metadata completion
aqe meta fix  # Prompts for missing author/title/year
```

---

## 12. Output Formats

### 12.1 JSON Output

```json
{
  "topic": "The impact of social media on political polarization",
  "extracted_at": "2024-01-15T10:30:00Z",
  "quotes": [
    {
      "quote": "Social media algorithms have been shown to increase exposure to ideologically congruent content by 34% compared to chronological feeds.",
      "reference": "Johnson, M. (2023) 'Filter Bubbles and Democracy', Political Communication, 41(2), pp. 156-178.",
      "in_text": "(Johnson, 2023, p. 162)",
      "page": 162,
      "relevance": 92,
      "explanation": "Provides quantitative evidence that social media algorithms contribute to polarization by limiting exposure to diverse viewpoints."
    }
  ],
  "bibliography": [
    "Johnson, M. (2023) 'Filter Bubbles and Democracy', Political Communication, 41(2), pp. 156-178."
  ]
}
```

### 12.2 Markdown Output

```markdown
# Quotes: The impact of social media on political polarization

*Extracted: 15 January 2024*

---

> "Social media algorithms have been shown to increase exposure to ideologically 
> congruent content by 34% compared to chronological feeds."
>
> — Johnson (2023, p. 162)
>
> **Relevance (92/100):** Provides quantitative evidence that social media 
> algorithms contribute to polarization by limiting exposure to diverse viewpoints.

---

## Bibliography

Johnson, M. (2023) 'Filter Bubbles and Democracy', *Political Communication*, 
41(2), pp. 156-178.
```

---

## 13. Project Structure

```
aqe/
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
│   │   ├── client.go      # HTTP client for docling-serve
│   │   └── chunker.go     # Python wrapper for chunking
│   ├── claude/
│   │   └── wrapper.go     # Claude Code CLI wrapper
│   ├── search/
│   │   └── weaviate.go    # Weaviate client
│   ├── store/
│   │   └── sqlite.go      # SQLite operations
│   ├── harvard/
│   │   └── formatter.go   # Reference formatting
│   └── models/
│       ├── document.go
│       ├── chunk.go
│       └── quote.go
├── scripts/
│   └── chunk_helper.py    # Python chunking helper
├── docker-compose.yml     # Docling + Weaviate
├── go.mod
└── README.md
```

---

## 14. Docker Compose for Dependencies

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

  weaviate:
    image: cr.weaviate.io/semitechnologies/weaviate:1.27.0
    ports:
      - "8080:8080"
      - "50051:50051"
    environment:
      QUERY_DEFAULTS_LIMIT: 25
      AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED: 'true'
      PERSISTENCE_DATA_PATH: '/var/lib/weaviate'
      DEFAULT_VECTORIZER_MODULE: 'text2vec-openai'
      ENABLE_MODULES: 'text2vec-openai,generative-openai'
      CLUSTER_HOSTNAME: 'node1'
    volumes:
      - weaviate-data:/var/lib/weaviate

volumes:
  docling-cache:
  weaviate-data:
```

---

## 15. Summary: Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Document Parsing | **Docling** | Multi-format, built-in chunking, metadata preservation |
| Search Strategy | **Hybrid RAG** | Cost-effective, sufficient accuracy for academic docs |
| Vector DB | **Weaviate** | Native Go client, built-in hybrid search |
| Persistence | **SQLite** | Simple CLI tool, zero-config |
| LLM Interface | **Claude Code CLI** | Per requirements, wrapper pattern |
| Quote Retrieval | **Deterministic** | Chunk IDs only, no LLM-generated text |
| Reference Format | **Harvard** | Go implementation, template-based |

---

## 16. Implementation Order

1. Docling client + SQLite schema + basic ingest
2. Weaviate integration + embedding pipeline
3. Claude Code wrapper + extraction logic
4. Harvard formatter + export formats
5. CLI polish + testing + documentation
