package store

// migrationSQL contains the complete database schema
const migrationSQL = `
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
    id TEXT PRIMARY KEY,  -- Docling self_ref e.g., "#/chunks/0"
    document_id INTEGER NOT NULL,
    text TEXT NOT NULL,
    page_num INTEGER,
    section_path TEXT,  -- JSON array: ["Chapter 1", "Section 1.2"]
    bbox TEXT,  -- JSON object: {"l":0,"t":0,"r":100,"b":50}
    embedding_id TEXT,  -- Weaviate UUID
    position INTEGER,  -- Sequential order within document (0-based), used for adjacency queries
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
`

// migrationV2AddColumn adds the position column to existing databases
const migrationV2AddColumn = `ALTER TABLE chunks ADD COLUMN position INTEGER;`

// migrationV2Index creates the position index (safe to re-run via IF NOT EXISTS)
const migrationV2Index = `CREATE INDEX IF NOT EXISTS idx_chunks_position ON chunks(document_id, position);`
