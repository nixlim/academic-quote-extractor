package unit

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/foundry-zero/aqe/internal/models"
	"github.com/foundry-zero/aqe/internal/store"
)

func setupTestDB(t *testing.T) (*store.Store, func()) {
	// Create temp file for test database
	f, err := os.CreateTemp("", "aqe-test-*.db")
	require.NoError(t, err)
	dbPath := f.Name()
	f.Close()

	// Create store
	s, err := store.NewStore(dbPath)
	require.NoError(t, err)

	// Run migrations
	err = s.RunMigrations()
	require.NoError(t, err)

	// Return cleanup function
	cleanup := func() {
		s.Close()
		os.Remove(dbPath)
	}

	return s, cleanup
}

func TestDocumentCRUD(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a document
	title := "Test Document"
	year := 2023
	doc := &models.Document{
		Filename:   "test.pdf",
		Filepath:   "/path/to/test.pdf",
		Title:      &title,
		Authors:    []string{"Smith, J.", "Jones, M."},
		Year:       &year,
		SourceType: models.SourceTypeBook,
		Checksum:   "abc123",
		IngestedAt: time.Now(),
	}

	// Insert
	id, err := s.InsertDocument(doc)
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	// Retrieve by checksum
	retrieved, err := s.GetDocumentByChecksum("abc123")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, id, retrieved.ID)
	assert.Equal(t, "test.pdf", retrieved.Filename)
	assert.Equal(t, "Test Document", *retrieved.Title)
	assert.Equal(t, []string{"Smith, J.", "Jones, M."}, retrieved.Authors)
	assert.Equal(t, 2023, *retrieved.Year)
	assert.Equal(t, models.SourceTypeBook, retrieved.SourceType)
}

func TestChunkCRUD(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a document first
	doc := &models.Document{
		Filename:   "test.pdf",
		Filepath:   "/path/to/test.pdf",
		SourceType: models.SourceTypeUnknown,
		Checksum:   "abc123",
		IngestedAt: time.Now(),
	}
	docID, err := s.InsertDocument(doc)
	require.NoError(t, err)

	// Create a chunk
	pageNum := 5
	chunk := &models.Chunk{
		ID:          "#/chunks/0",
		DocumentID:  docID,
		Text:        "This is test text for the chunk.",
		PageNum:     &pageNum,
		SectionPath: []string{"Chapter 1", "Section 1.1"},
	}

	// Insert
	err = s.InsertChunk(chunk)
	require.NoError(t, err)

	// Verify via count
	count, err := s.GetChunkCount()
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestChecksumUniqueness(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	// Create first document
	doc1 := &models.Document{
		Filename:   "doc1.pdf",
		Filepath:   "/path/to/doc1.pdf",
		SourceType: models.SourceTypeUnknown,
		Checksum:   "same_checksum",
		IngestedAt: time.Now(),
	}
	_, err := s.InsertDocument(doc1)
	require.NoError(t, err)

	// Try to insert document with same checksum
	doc2 := &models.Document{
		Filename:   "doc2.pdf",
		Filepath:   "/path/to/doc2.pdf",
		SourceType: models.SourceTypeUnknown,
		Checksum:   "same_checksum",
		IngestedAt: time.Now(),
	}
	_, err = s.InsertDocument(doc2)
	assert.Error(t, err, "Should fail due to unique checksum constraint")
}

func TestInsertChunksBatch(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a document first
	doc := &models.Document{
		Filename:   "test.pdf",
		Filepath:   "/path/to/test.pdf",
		SourceType: models.SourceTypeUnknown,
		Checksum:   "abc123",
		IngestedAt: time.Now(),
	}
	docID, err := s.InsertDocument(doc)
	require.NoError(t, err)

	// Create multiple chunks
	chunks := []*models.Chunk{
		{ID: "#/chunks/0", DocumentID: docID, Text: "Chunk 1 text"},
		{ID: "#/chunks/1", DocumentID: docID, Text: "Chunk 2 text"},
		{ID: "#/chunks/2", DocumentID: docID, Text: "Chunk 3 text"},
	}

	// Batch insert
	err = s.InsertChunks(chunks)
	require.NoError(t, err)

	// Verify count
	count, err := s.GetChunkCount()
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

// TestUpdateDocumentMetadata tests metadata update functionality (T116)
func TestUpdateDocumentMetadata(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a document with incomplete metadata
	doc := &models.Document{
		Filename:   "incomplete.pdf",
		Filepath:   "/path/to/incomplete.pdf",
		SourceType: models.SourceTypeUnknown,
		Checksum:   "incomplete123",
		IngestedAt: time.Now(),
	}
	docID, err := s.InsertDocument(doc)
	require.NoError(t, err)

	// Verify document has no metadata
	retrieved, err := s.GetDocumentByID(docID)
	require.NoError(t, err)
	assert.Nil(t, retrieved.Title)
	assert.Nil(t, retrieved.Year)
	assert.Empty(t, retrieved.Authors)

	// Update metadata
	err = s.UpdateDocumentMetadata(docID, "Complete Title", []string{"Author, A.", "Writer, B."}, 2024)
	require.NoError(t, err)

	// Verify update
	updated, err := s.GetDocumentByID(docID)
	require.NoError(t, err)
	require.NotNil(t, updated)

	assert.NotNil(t, updated.Title)
	assert.Equal(t, "Complete Title", *updated.Title)
	assert.NotNil(t, updated.Year)
	assert.Equal(t, 2024, *updated.Year)
	assert.Equal(t, []string{"Author, A.", "Writer, B."}, updated.Authors)
}

// TestGetDocumentsWithIncompleteMetadata tests finding documents with missing metadata
func TestGetDocumentsWithIncompleteMetadata(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a complete document
	title := "Complete Document"
	year := 2023
	completeDoc := &models.Document{
		Filename:   "complete.pdf",
		Filepath:   "/path/to/complete.pdf",
		Title:      &title,
		Authors:    []string{"Smith, J."},
		Year:       &year,
		SourceType: models.SourceTypeBook,
		Checksum:   "complete123",
		IngestedAt: time.Now(),
	}
	_, err := s.InsertDocument(completeDoc)
	require.NoError(t, err)

	// Create documents with incomplete metadata
	incompleteDoc1 := &models.Document{
		Filename:   "no-title.pdf",
		Filepath:   "/path/to/no-title.pdf",
		SourceType: models.SourceTypeUnknown,
		Checksum:   "notitle123",
		IngestedAt: time.Now(),
	}
	_, err = s.InsertDocument(incompleteDoc1)
	require.NoError(t, err)

	incompleteTitle := "Has Title"
	incompleteDoc2 := &models.Document{
		Filename:   "no-year.pdf",
		Filepath:   "/path/to/no-year.pdf",
		Title:      &incompleteTitle,
		Authors:    []string{"Author, A."},
		SourceType: models.SourceTypeUnknown,
		Checksum:   "noyear123",
		IngestedAt: time.Now(),
	}
	_, err = s.InsertDocument(incompleteDoc2)
	require.NoError(t, err)

	// Get documents with incomplete metadata
	incomplete, err := s.GetDocumentsWithIncompleteMetadata()
	require.NoError(t, err)

	// Should find 2 incomplete documents
	assert.Len(t, incomplete, 2)

	// Verify the complete document is not in the list
	for _, doc := range incomplete {
		assert.NotEqual(t, "complete.pdf", doc.Filename)
	}
}

// TestExtractionCRUD tests extraction and quote operations
func TestExtractionCRUD(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a document and chunk first
	doc := &models.Document{
		Filename:   "test.pdf",
		Filepath:   "/path/to/test.pdf",
		SourceType: models.SourceTypeUnknown,
		Checksum:   "extraction-test",
		IngestedAt: time.Now(),
	}
	docID, err := s.InsertDocument(doc)
	require.NoError(t, err)

	chunk := &models.Chunk{
		ID:         "#/chunks/test",
		DocumentID: docID,
		Text:       "Test text for extraction.",
	}
	err = s.InsertChunk(chunk)
	require.NoError(t, err)

	// Create an extraction
	extraction := &models.Extraction{
		Topic:     "test topic",
		CreatedAt: time.Now(),
	}
	extractionID, err := s.InsertExtraction(extraction)
	require.NoError(t, err)
	assert.Greater(t, extractionID, int64(0))

	// Retrieve extraction
	retrieved, err := s.GetExtractionByID(extractionID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, "test topic", retrieved.Topic)

	// Insert quotes
	quotes := []*models.ExtractedQuote{
		{
			ExtractionID:   extractionID,
			ChunkID:        "#/chunks/test",
			RelevanceScore: 85,
			Explanation:    "Highly relevant quote.",
		},
	}
	err = s.InsertExtractedQuotes(quotes)
	require.NoError(t, err)

	// Retrieve quotes
	retrievedQuotes, err := s.GetQuotesByExtractionID(extractionID)
	require.NoError(t, err)
	assert.Len(t, retrievedQuotes, 1)
	assert.Equal(t, 85, retrievedQuotes[0].RelevanceScore)
	assert.Equal(t, "#/chunks/test", retrievedQuotes[0].ChunkID)

	// List extractions with quote counts
	extractions, err := s.ListExtractions()
	require.NoError(t, err)
	assert.Len(t, extractions, 1)
	assert.Equal(t, 1, extractions[0].QuoteCount)
}
