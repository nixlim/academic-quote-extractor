package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/foundry-zero/aqe/internal/models"
)

// Store provides SQLite database operations
type Store struct {
	db *sql.DB
}

// NewStore creates a new Store with the given database path.
// Creates the database file if it doesn't exist.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// RunMigrations creates the database schema
func (s *Store) RunMigrations() error {
	_, err := s.db.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// DocumentWithChunkCount is a Document with its chunk count for listing
type DocumentWithChunkCount struct {
	models.Document
	ChunkCount int
}

// ListDocumentsWithChunkCounts returns all documents with their chunk counts
func (s *Store) ListDocumentsWithChunkCounts() ([]DocumentWithChunkCount, error) {
	rows, err := s.db.Query(`
		SELECT d.id, d.filename, d.filepath, d.title, d.authors, d.year, d.publisher, d.source_type, d.checksum, d.ingested_at,
			   COUNT(c.id) AS chunk_count
		FROM documents d
		LEFT JOIN chunks c ON c.document_id = d.id
		GROUP BY d.id
		ORDER BY d.id`)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	var results []DocumentWithChunkCount
	for rows.Next() {
		var dwc DocumentWithChunkCount
		var authorsJSON sql.NullString
		var title, publisher sql.NullString
		var year sql.NullInt64

		err := rows.Scan(
			&dwc.ID, &dwc.Filename, &dwc.Filepath, &title, &authorsJSON,
			&year, &publisher, &dwc.SourceType, &dwc.Checksum, &dwc.IngestedAt,
			&dwc.ChunkCount,
		)
		if err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}

		if title.Valid {
			dwc.Title = &title.String
		}
		if publisher.Valid {
			dwc.Publisher = &publisher.String
		}
		if year.Valid {
			y := int(year.Int64)
			dwc.Year = &y
		}
		if authorsJSON.Valid {
			if err := dwc.UnmarshalAuthors(authorsJSON.String); err != nil {
				return nil, fmt.Errorf("unmarshal authors: %w", err)
			}
		}

		results = append(results, dwc)
	}

	return results, rows.Err()
}

// GetQuoteCount returns the total number of extracted quotes
func (s *Store) GetQuoteCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM extracted_quotes").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count quotes: %w", err)
	}
	return count, nil
}

// WithTx executes a function within a transaction.
// Rolls back on error, commits on success.
func (s *Store) WithTx(fn func(tx *sql.Tx) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// DB returns the underlying database connection for direct queries
func (s *Store) DB() *sql.DB {
	return s.db
}

// CalculateChecksum computes SHA-256 hash of a file
func CalculateChecksum(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// InsertDocument inserts a new document and returns its ID
func (s *Store) InsertDocument(doc *models.Document) (int64, error) {
	authorsJSON, err := doc.MarshalAuthors()
	if err != nil {
		return 0, fmt.Errorf("marshal authors: %w", err)
	}

	result, err := s.db.Exec(`
		INSERT INTO documents (filename, filepath, title, authors, year, publisher, source_type, checksum, ingested_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		doc.Filename, doc.Filepath, doc.Title, authorsJSON, doc.Year, doc.Publisher,
		string(doc.SourceType), doc.Checksum, time.Now(),
	)
	if err != nil {
		return 0, fmt.Errorf("insert document: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}

	return id, nil
}

// GetDocumentByChecksum returns a document by its checksum, or nil if not found
func (s *Store) GetDocumentByChecksum(checksum string) (*models.Document, error) {
	row := s.db.QueryRow(`
		SELECT id, filename, filepath, title, authors, year, publisher, source_type, checksum, ingested_at
		FROM documents WHERE checksum = ?`, checksum)

	doc := &models.Document{}
	var authorsJSON sql.NullString
	var title, publisher sql.NullString
	var year sql.NullInt64

	err := row.Scan(
		&doc.ID, &doc.Filename, &doc.Filepath, &title, &authorsJSON,
		&year, &publisher, &doc.SourceType, &doc.Checksum, &doc.IngestedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan document: %w", err)
	}

	if title.Valid {
		doc.Title = &title.String
	}
	if publisher.Valid {
		doc.Publisher = &publisher.String
	}
	if year.Valid {
		y := int(year.Int64)
		doc.Year = &y
	}
	if authorsJSON.Valid {
		if err := doc.UnmarshalAuthors(authorsJSON.String); err != nil {
			return nil, fmt.Errorf("unmarshal authors: %w", err)
		}
	}

	return doc, nil
}

// InsertChunk inserts a single chunk
func (s *Store) InsertChunk(chunk *models.Chunk) error {
	sectionPathJSON, err := chunk.MarshalSectionPath()
	if err != nil {
		return fmt.Errorf("marshal section path: %w", err)
	}

	bboxJSON, err := chunk.MarshalBBox()
	if err != nil {
		return fmt.Errorf("marshal bbox: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO chunks (id, document_id, text, page_num, section_path, bbox, embedding_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		chunk.ID, chunk.DocumentID, chunk.Text, chunk.PageNum,
		sectionPathJSON, bboxJSON, chunk.EmbeddingID,
	)
	if err != nil {
		return fmt.Errorf("insert chunk: %w", err)
	}

	return nil
}

// InsertChunks inserts multiple chunks in a transaction
func (s *Store) InsertChunks(chunks []*models.Chunk) error {
	return s.WithTx(func(tx *sql.Tx) error {
		stmt, err := tx.Prepare(`
			INSERT INTO chunks (id, document_id, text, page_num, section_path, bbox, embedding_id)
			VALUES (?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return fmt.Errorf("prepare statement: %w", err)
		}
		defer stmt.Close()

		for _, chunk := range chunks {
			sectionPathJSON, err := chunk.MarshalSectionPath()
			if err != nil {
				return fmt.Errorf("marshal section path: %w", err)
			}

			bboxJSON, err := chunk.MarshalBBox()
			if err != nil {
				return fmt.Errorf("marshal bbox: %w", err)
			}

			_, err = stmt.Exec(
				chunk.ID, chunk.DocumentID, chunk.Text, chunk.PageNum,
				sectionPathJSON, bboxJSON, chunk.EmbeddingID,
			)
			if err != nil {
				return fmt.Errorf("insert chunk %s: %w", chunk.ID, err)
			}
		}

		return nil
	})
}

// GetDocumentCount returns the number of documents in the database
func (s *Store) GetDocumentCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM documents").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count documents: %w", err)
	}
	return count, nil
}

// GetChunkCount returns the number of chunks in the database
func (s *Store) GetChunkCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM chunks").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count chunks: %w", err)
	}
	return count, nil
}

// GetDocumentByID returns a document by its ID
func (s *Store) GetDocumentByID(id int64) (*models.Document, error) {
	row := s.db.QueryRow(`
		SELECT id, filename, filepath, title, authors, year, publisher, source_type, checksum, ingested_at
		FROM documents WHERE id = ?`, id)

	doc := &models.Document{}
	var authorsJSON sql.NullString
	var title, publisher sql.NullString
	var year sql.NullInt64

	err := row.Scan(
		&doc.ID, &doc.Filename, &doc.Filepath, &title, &authorsJSON,
		&year, &publisher, &doc.SourceType, &doc.Checksum, &doc.IngestedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan document: %w", err)
	}

	if title.Valid {
		doc.Title = &title.String
	}
	if publisher.Valid {
		doc.Publisher = &publisher.String
	}
	if year.Valid {
		y := int(year.Int64)
		doc.Year = &y
	}
	if authorsJSON.Valid {
		if err := doc.UnmarshalAuthors(authorsJSON.String); err != nil {
			return nil, fmt.Errorf("unmarshal authors: %w", err)
		}
	}

	return doc, nil
}

// GetChunkByID returns a chunk by its ID
func (s *Store) GetChunkByID(id string) (*models.Chunk, error) {
	row := s.db.QueryRow(`
		SELECT id, document_id, text, page_num, section_path, bbox, embedding_id
		FROM chunks WHERE id = ?`, id)

	chunk := &models.Chunk{}
	var sectionPathJSON, bboxJSON sql.NullString
	var pageNum sql.NullInt64
	var embeddingID sql.NullString

	err := row.Scan(
		&chunk.ID, &chunk.DocumentID, &chunk.Text, &pageNum,
		&sectionPathJSON, &bboxJSON, &embeddingID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan chunk: %w", err)
	}

	if pageNum.Valid {
		pn := int(pageNum.Int64)
		chunk.PageNum = &pn
	}
	if sectionPathJSON.Valid {
		if err := chunk.UnmarshalSectionPath(sectionPathJSON.String); err != nil {
			return nil, fmt.Errorf("unmarshal section path: %w", err)
		}
	}
	if bboxJSON.Valid {
		if err := chunk.UnmarshalBBox(bboxJSON.String); err != nil {
			return nil, fmt.Errorf("unmarshal bbox: %w", err)
		}
	}
	if embeddingID.Valid {
		chunk.EmbeddingID = &embeddingID.String
	}

	return chunk, nil
}

// GetChunksByIDs returns chunks by their IDs
func (s *Store) GetChunksByIDs(ids []string) ([]*models.Chunk, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Build query with placeholders
	placeholders := ""
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = id
	}

	rows, err := s.db.Query(`
		SELECT id, document_id, text, page_num, section_path, bbox, embedding_id
		FROM chunks WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("query chunks: %w", err)
	}
	defer rows.Close()

	var chunks []*models.Chunk
	for rows.Next() {
		chunk := &models.Chunk{}
		var sectionPathJSON, bboxJSON sql.NullString
		var pageNum sql.NullInt64
		var embeddingID sql.NullString

		err := rows.Scan(
			&chunk.ID, &chunk.DocumentID, &chunk.Text, &pageNum,
			&sectionPathJSON, &bboxJSON, &embeddingID,
		)
		if err != nil {
			return nil, fmt.Errorf("scan chunk: %w", err)
		}

		if pageNum.Valid {
			pn := int(pageNum.Int64)
			chunk.PageNum = &pn
		}
		if sectionPathJSON.Valid {
			if err := chunk.UnmarshalSectionPath(sectionPathJSON.String); err != nil {
				return nil, fmt.Errorf("unmarshal section path: %w", err)
			}
		}
		if bboxJSON.Valid {
			if err := chunk.UnmarshalBBox(bboxJSON.String); err != nil {
				return nil, fmt.Errorf("unmarshal bbox: %w", err)
			}
		}
		if embeddingID.Valid {
			chunk.EmbeddingID = &embeddingID.String
		}

		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// InsertExtraction inserts a new extraction and returns its ID
func (s *Store) InsertExtraction(extraction *models.Extraction) (int64, error) {
	result, err := s.db.Exec(`
		INSERT INTO extractions (topic, created_at)
		VALUES (?, ?)`,
		extraction.Topic, extraction.CreatedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("insert extraction: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}

	return id, nil
}

// GetExtractionByID returns an extraction by its ID
func (s *Store) GetExtractionByID(id int64) (*models.Extraction, error) {
	row := s.db.QueryRow(`
		SELECT id, topic, created_at
		FROM extractions WHERE id = ?`, id)

	extraction := &models.Extraction{}
	err := row.Scan(&extraction.ID, &extraction.Topic, &extraction.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan extraction: %w", err)
	}

	return extraction, nil
}

// ListExtractions returns all extractions with quote counts
func (s *Store) ListExtractions() ([]*models.Extraction, error) {
	rows, err := s.db.Query(`
		SELECT e.id, e.topic, e.created_at, COUNT(eq.id) as quote_count
		FROM extractions e
		LEFT JOIN extracted_quotes eq ON e.id = eq.extraction_id
		GROUP BY e.id
		ORDER BY e.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query extractions: %w", err)
	}
	defer rows.Close()

	var extractions []*models.Extraction
	for rows.Next() {
		extraction := &models.Extraction{}
		var quoteCount int
		err := rows.Scan(&extraction.ID, &extraction.Topic, &extraction.CreatedAt, &quoteCount)
		if err != nil {
			return nil, fmt.Errorf("scan extraction: %w", err)
		}
		extraction.QuoteCount = quoteCount
		extractions = append(extractions, extraction)
	}

	return extractions, nil
}

// InsertExtractedQuotes inserts multiple extracted quotes in a transaction
func (s *Store) InsertExtractedQuotes(quotes []*models.ExtractedQuote) error {
	return s.WithTx(func(tx *sql.Tx) error {
		stmt, err := tx.Prepare(`
			INSERT INTO extracted_quotes (extraction_id, chunk_id, relevance_score, explanation)
			VALUES (?, ?, ?, ?)`)
		if err != nil {
			return fmt.Errorf("prepare statement: %w", err)
		}
		defer stmt.Close()

		for _, quote := range quotes {
			_, err = stmt.Exec(quote.ExtractionID, quote.ChunkID, quote.RelevanceScore, quote.Explanation)
			if err != nil {
				return fmt.Errorf("insert quote: %w", err)
			}
		}

		return nil
	})
}

// GetQuotesByExtractionID returns all quotes for an extraction with chunk and document data
func (s *Store) GetQuotesByExtractionID(extractionID int64) ([]*models.ExtractedQuote, error) {
	rows, err := s.db.Query(`
		SELECT eq.id, eq.extraction_id, eq.chunk_id, eq.relevance_score, eq.explanation
		FROM extracted_quotes eq
		WHERE eq.extraction_id = ?
		ORDER BY eq.relevance_score DESC`, extractionID)
	if err != nil {
		return nil, fmt.Errorf("query quotes: %w", err)
	}
	defer rows.Close()

	var quotes []*models.ExtractedQuote
	for rows.Next() {
		quote := &models.ExtractedQuote{}
		err := rows.Scan(&quote.ID, &quote.ExtractionID, &quote.ChunkID, &quote.RelevanceScore, &quote.Explanation)
		if err != nil {
			return nil, fmt.Errorf("scan quote: %w", err)
		}
		quotes = append(quotes, quote)
	}

	return quotes, nil
}

// GetDocumentsWithIncompleteMetadata returns documents missing title, authors, or year
func (s *Store) GetDocumentsWithIncompleteMetadata() ([]*models.Document, error) {
	rows, err := s.db.Query(`
		SELECT id, filename, filepath, title, authors, year, publisher, source_type, checksum, ingested_at
		FROM documents
		WHERE title IS NULL OR authors IS NULL OR authors = '[]' OR year IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("query documents: %w", err)
	}
	defer rows.Close()

	var docs []*models.Document
	for rows.Next() {
		doc := &models.Document{}
		var authorsJSON sql.NullString
		var title, publisher sql.NullString
		var year sql.NullInt64

		err := rows.Scan(
			&doc.ID, &doc.Filename, &doc.Filepath, &title, &authorsJSON,
			&year, &publisher, &doc.SourceType, &doc.Checksum, &doc.IngestedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}

		if title.Valid {
			doc.Title = &title.String
		}
		if publisher.Valid {
			doc.Publisher = &publisher.String
		}
		if year.Valid {
			y := int(year.Int64)
			doc.Year = &y
		}
		if authorsJSON.Valid {
			if err := doc.UnmarshalAuthors(authorsJSON.String); err != nil {
				return nil, fmt.Errorf("unmarshal authors: %w", err)
			}
		}

		docs = append(docs, doc)
	}

	return docs, nil
}

// UpdateDocumentMetadata updates the metadata for a document
func (s *Store) UpdateDocumentMetadata(id int64, title string, authors []string, year int) error {
	// Marshal authors to JSON
	doc := &models.Document{Authors: authors}
	authorsJSON, err := doc.MarshalAuthors()
	if err != nil {
		return fmt.Errorf("marshal authors: %w", err)
	}

	_, err = s.db.Exec(`
		UPDATE documents SET title = ?, authors = ?, year = ?
		WHERE id = ?`,
		title, authorsJSON, year, id,
	)
	if err != nil {
		return fmt.Errorf("update document: %w", err)
	}

	return nil
}
