# Data Model: Academic Quote Extractor CLI

**Feature**: 001-quote-extractor-cli
**Date**: 2026-01-29

## Entity Relationship Diagram

```
┌─────────────────┐       ┌─────────────────┐
│    Document     │       │   Extraction    │
├─────────────────┤       ├─────────────────┤
│ id (PK)         │       │ id (PK)         │
│ filename        │       │ topic           │
│ filepath        │       │ created_at      │
│ title           │       └────────┬────────┘
│ authors (JSON)  │                │
│ year            │                │ 1:N
│ publisher       │                │
│ source_type     │                ▼
│ checksum        │       ┌─────────────────┐
│ ingested_at     │       │ ExtractedQuote  │
└────────┬────────┘       ├─────────────────┤
         │                │ id (PK)         │
         │ 1:N            │ extraction_id   │──┐
         │                │ chunk_id        │──┼─┐
         ▼                │ relevance_score │  │ │
┌─────────────────┐       │ explanation     │  │ │
│     Chunk       │       └─────────────────┘  │ │
├─────────────────┤                            │ │
│ id (PK)         │◄───────────────────────────┼─┘
│ document_id     │──┐                         │
│ text            │  │                         │
│ page_num        │  │ FK to Document          │
│ section_path    │  │                         │
│ bbox (JSON)     │  │                         │
│ embedding_id    │  │                         │
└─────────────────┘  │                         │
                     │                         │
                     └─────────────────────────┘
                              FK to Extraction
```

## Entities

### Document

Represents an ingested source file.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | INTEGER | PRIMARY KEY, AUTOINCREMENT | Unique document identifier |
| filename | TEXT | NOT NULL | Original filename (e.g., "thesis.pdf") |
| filepath | TEXT | NOT NULL | Absolute path at ingestion time |
| title | TEXT | NULLABLE | Document title (auto-detected or manual) |
| authors | TEXT | NULLABLE, JSON array | Authors in format `["Smith, J.", "Jones, M."]` |
| year | INTEGER | NULLABLE | Publication year |
| publisher | TEXT | NULLABLE | Publisher name |
| source_type | TEXT | NOT NULL, DEFAULT 'unknown' | Enum: book, journal_article, website, chapter, unknown |
| checksum | TEXT | UNIQUE, NOT NULL | SHA-256 hash for duplicate detection |
| ingested_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | When document was ingested |

**Validation Rules:**
- `checksum` must be unique (prevents re-ingestion)
- `source_type` must be one of: book, journal_article, website, chapter, unknown
- `authors` must be valid JSON array when present

**Go Type:**
```go
type Document struct {
    ID         int64      `db:"id"`
    Filename   string     `db:"filename"`
    Filepath   string     `db:"filepath"`
    Title      *string    `db:"title"`
    Authors    []string   // Stored as JSON in authors column
    Year       *int       `db:"year"`
    Publisher  *string    `db:"publisher"`
    SourceType SourceType `db:"source_type"`
    Checksum   string     `db:"checksum"`
    IngestedAt time.Time  `db:"ingested_at"`
}

type SourceType string

const (
    SourceTypeBook           SourceType = "book"
    SourceTypeJournalArticle SourceType = "journal_article"
    SourceTypeWebsite        SourceType = "website"
    SourceTypeChapter        SourceType = "chapter"
    SourceTypeUnknown        SourceType = "unknown"
)
```

---

### Chunk

Represents a segment of text from a document. The `text` field is the **source of truth** for verbatim quotes.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | TEXT | PRIMARY KEY | Docling self_ref (e.g., "#/texts/42") |
| document_id | INTEGER | NOT NULL, FK → documents(id) | Parent document |
| text | TEXT | NOT NULL | Verbatim chunk text (source of truth) |
| page_num | INTEGER | NULLABLE | Starting page number |
| section_path | TEXT | NULLABLE, JSON array | Heading hierarchy `["Chapter 1", "Section 1.2"]` |
| bbox | TEXT | NULLABLE, JSON object | Bounding box `{"l":0,"t":0,"r":100,"b":50}` |
| embedding_id | TEXT | NULLABLE | Weaviate UUID reference |

**Validation Rules:**
- `id` format must match Docling self_ref pattern
- `text` must not be empty
- `section_path` must be valid JSON array when present
- `bbox` must be valid JSON object when present

**Go Type:**
```go
type Chunk struct {
    ID          string   `db:"id"`           // Docling self_ref
    DocumentID  int64    `db:"document_id"`
    Text        string   `db:"text"`         // Verbatim text - SOURCE OF TRUTH
    PageNum     *int     `db:"page_num"`
    SectionPath []string // Stored as JSON in section_path column
    BBox        *BBox    // Stored as JSON in bbox column
    EmbeddingID *string  `db:"embedding_id"` // Weaviate UUID
}

type BBox struct {
    L           float64 `json:"l"`
    T           float64 `json:"t"`
    R           float64 `json:"r"`
    B           float64 `json:"b"`
    CoordOrigin string  `json:"coord_origin,omitempty"`
}
```

---

### Extraction

Represents a quote extraction session.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | INTEGER | PRIMARY KEY, AUTOINCREMENT | Unique extraction identifier |
| topic | TEXT | NOT NULL | Research topic/query |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | When extraction was performed |

**Validation Rules:**
- `topic` must not be empty

**Go Type:**
```go
type Extraction struct {
    ID        int64     `db:"id"`
    Topic     string    `db:"topic"`
    CreatedAt time.Time `db:"created_at"`
}
```

---

### ExtractedQuote

Represents a single quote selected during extraction.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | INTEGER | PRIMARY KEY, AUTOINCREMENT | Unique quote identifier |
| extraction_id | INTEGER | NOT NULL, FK → extractions(id) | Parent extraction session |
| chunk_id | TEXT | NOT NULL, FK → chunks(id) | Source chunk reference |
| relevance_score | INTEGER | NOT NULL, CHECK(0-100) | LLM-assigned relevance (0-100) |
| explanation | TEXT | NOT NULL | Why this quote is relevant |

**Validation Rules:**
- `relevance_score` must be between 0 and 100
- `chunk_id` must exist in chunks table (validated before insert)
- `explanation` must not be empty

**Go Type:**
```go
type ExtractedQuote struct {
    ID             int64  `db:"id"`
    ExtractionID   int64  `db:"extraction_id"`
    ChunkID        string `db:"chunk_id"`
    RelevanceScore int    `db:"relevance_score"`
    Explanation    string `db:"explanation"`
}
```

---

## SQLite Schema

```sql
-- Enable foreign keys
PRAGMA foreign_keys = ON;

-- Documents table
CREATE TABLE IF NOT EXISTS documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename TEXT NOT NULL,
    filepath TEXT NOT NULL,
    title TEXT,
    authors TEXT,  -- JSON array: ["Smith, J.", "Jones, M."]
    year INTEGER,
    publisher TEXT,
    source_type TEXT NOT NULL DEFAULT 'unknown' 
        CHECK(source_type IN ('book', 'journal_article', 'website', 'chapter', 'unknown')),
    checksum TEXT UNIQUE NOT NULL,
    ingested_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Chunks table (source of truth for verbatim text)
CREATE TABLE IF NOT EXISTS chunks (
    id TEXT PRIMARY KEY,  -- Docling self_ref e.g., "#/texts/42"
    document_id INTEGER NOT NULL,
    text TEXT NOT NULL,
    page_num INTEGER,
    section_path TEXT,  -- JSON array: ["Chapter 1", "Section 1.2"]
    bbox TEXT,  -- JSON object: {"l":0,"t":0,"r":100,"b":50}
    embedding_id TEXT,  -- Weaviate UUID
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

-- Extractions table
CREATE TABLE IF NOT EXISTS extractions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    topic TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Extracted quotes table
CREATE TABLE IF NOT EXISTS extracted_quotes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    extraction_id INTEGER NOT NULL,
    chunk_id TEXT NOT NULL,
    relevance_score INTEGER NOT NULL CHECK(relevance_score >= 0 AND relevance_score <= 100),
    explanation TEXT NOT NULL,
    FOREIGN KEY (extraction_id) REFERENCES extractions(id) ON DELETE CASCADE,
    FOREIGN KEY (chunk_id) REFERENCES chunks(id)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_chunks_document ON chunks(document_id);
CREATE INDEX IF NOT EXISTS idx_chunks_page ON chunks(page_num);
CREATE INDEX IF NOT EXISTS idx_quotes_extraction ON extracted_quotes(extraction_id);
CREATE INDEX IF NOT EXISTS idx_quotes_chunk ON extracted_quotes(chunk_id);
CREATE INDEX IF NOT EXISTS idx_documents_checksum ON documents(checksum);
```

---

## Weaviate Schema

```json
{
  "class": "Chunk",
  "description": "Text chunk from academic document for semantic search",
  "vectorizer": "text2vec-ollama",
  "moduleConfig": {
    "text2vec-ollama": {
      "model": "nomic-embed-text",
      "apiEndpoint": "http://ollama:11434"
    }
  },
  "properties": [
    {
      "name": "text",
      "dataType": ["text"],
      "description": "Chunk text content (used for vectorization)",
      "moduleConfig": {
        "text2vec-ollama": {
          "skip": false,
          "vectorizePropertyName": false
        }
      }
    },
    {
      "name": "chunk_id",
      "dataType": ["string"],
      "description": "Docling self_ref - links to SQLite chunks.id",
      "moduleConfig": {
        "text2vec-ollama": {
          "skip": true
        }
      }
    },
    {
      "name": "document_id",
      "dataType": ["int"],
      "description": "SQLite document ID",
      "moduleConfig": {
        "text2vec-ollama": {
          "skip": true
        }
      }
    },
    {
      "name": "page_num",
      "dataType": ["int"],
      "description": "Page number in source document",
      "moduleConfig": {
        "text2vec-ollama": {
          "skip": true
        }
      }
    },
    {
      "name": "section_path",
      "dataType": ["string[]"],
      "description": "Section heading hierarchy",
      "moduleConfig": {
        "text2vec-ollama": {
          "skip": true
        }
      }
    }
  ]
}
```

---

## State Transitions

### Document Lifecycle

```
[Not Exists] ──ingest──▶ [Ingested] ──delete*──▶ [Not Exists]
                              │
                              │ (metadata incomplete)
                              ▼
                         [Needs Fix] ──meta fix──▶ [Ingested]
```

*Delete not implemented in MVP; future enhancement.

### Extraction Lifecycle

```
[Query Received] ──search──▶ [Chunks Retrieved] ──LLM scoring──▶ [Quotes Selected] ──save──▶ [Persisted]
                                                                                              │
                                                                                              ▼
                                                                                         [Exportable]
```

---

## Data Integrity Rules

1. **Verbatim Text Source**: All quote text MUST come from `chunks.text`, never from LLM responses
2. **Chunk ID Validation**: Before inserting `extracted_quotes`, verify `chunk_id` exists in `chunks`
3. **Duplicate Prevention**: `documents.checksum` enforces uniqueness
4. **Referential Integrity**: Foreign keys with CASCADE delete ensure orphan prevention
5. **Score Bounds**: `relevance_score` constrained to 0-100 at database level
