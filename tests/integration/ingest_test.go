//go:build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/foundry-zero/aqe/internal/chunker"
	"github.com/foundry-zero/aqe/internal/docling"
	"github.com/foundry-zero/aqe/internal/models"
	"github.com/foundry-zero/aqe/internal/search"
	"github.com/foundry-zero/aqe/internal/store"
)

const (
	doclingURL     = "http://localhost:5001"
	weaviateHost   = "localhost:8080"
	ollamaEndpoint = "http://ollama:11434"
)

// setupTestEnvironment creates a test database and returns cleanup function
func setupTestEnvironment(t *testing.T) (*store.Store, *docling.Client, *search.WeaviateClient, *chunker.Chunker, func()) {
	// Create temp database
	tmpDir, err := os.MkdirTemp("", "aqe-integration-*")
	require.NoError(t, err)

	dbPath := filepath.Join(tmpDir, "test.db")
	s, err := store.NewStore(dbPath)
	require.NoError(t, err)

	err = s.RunMigrations()
	require.NoError(t, err)

	// Create clients
	doclingClient := docling.NewClient(doclingURL)

	weaviateClient, err := search.NewWeaviateClient(weaviateHost, ollamaEndpoint)
	require.NoError(t, err)

	chunkerClient, err := chunker.NewChunker()
	require.NoError(t, err)

	cleanup := func() {
		s.Close()
		os.RemoveAll(tmpDir)
	}

	return s, doclingClient, weaviateClient, chunkerClient, cleanup
}

// findTestPDF locates the test PDF file
func findTestPDF(t *testing.T) string {
	patterns := []string{
		"../../Culturally-Responsive-Computing*.pdf",
		"../../../Culturally-Responsive-Computing*.pdf",
		"Culturally-Responsive-Computing*.pdf",
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		if len(matches) > 0 {
			absPath, err := filepath.Abs(matches[0])
			require.NoError(t, err)
			return absPath
		}
	}

	t.Skip("Test PDF not found")
	return ""
}

// TestIngestSinglePDF tests ingesting a single PDF file (T077)
func TestIngestSinglePDF(t *testing.T) {
	s, doclingClient, weaviateClient, chunkerClient, cleanup := setupTestEnvironment(t)
	defer cleanup()

	pdfPath := findTestPDF(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Step 1: Check Docling health
	err := doclingClient.Health(ctx)
	require.NoError(t, err, "Docling should be healthy")

	// Step 2: Create Weaviate schema
	err = weaviateClient.CreateSchema(ctx)
	require.NoError(t, err, "Should create Weaviate schema")

	// Step 3: Calculate checksum and check for duplicates
	checksum, err := store.CalculateChecksum(pdfPath)
	require.NoError(t, err, "Should calculate checksum")

	existing, err := s.GetDocumentByChecksum(checksum)
	require.NoError(t, err)
	require.Nil(t, existing, "Document should not exist yet")

	// Step 4: Convert document with Docling
	t.Log("Converting PDF with Docling...")
	doclingDoc, err := doclingClient.ConvertFile(ctx, pdfPath)
	require.NoError(t, err, "Docling conversion should succeed")
	require.NotNil(t, doclingDoc)
	require.NotEmpty(t, doclingDoc.Texts, "Should have text items")
	t.Logf("Docling extracted %d text items", len(doclingDoc.Texts))

	// Step 5: Chunk the document
	t.Log("Chunking document...")
	chunks, err := chunkerClient.ChunkDocument(ctx, doclingDoc)
	require.NoError(t, err, "Chunking should succeed")
	require.NotEmpty(t, chunks, "Should have chunks")
	t.Logf("Created %d chunks", len(chunks))

	// Step 6: Store document in SQLite
	doc := &models.Document{
		Filename:   filepath.Base(pdfPath),
		Filepath:   pdfPath,
		SourceType: models.SourceTypeBook,
		Checksum:   checksum,
		IngestedAt: time.Now(),
	}
	docID, err := s.InsertDocument(doc)
	require.NoError(t, err, "Should insert document")
	assert.Greater(t, docID, int64(0))

	// Step 7: Store chunks in SQLite and Weaviate
	var modelChunks []*models.Chunk
	for _, c := range chunks {
		mc := &models.Chunk{
			ID:          c.ID,
			DocumentID:  docID,
			Text:        c.Text,
			PageNum:     c.PageNum,
			SectionPath: c.SectionPath,
		}
		modelChunks = append(modelChunks, mc)
	}

	err = s.InsertChunks(modelChunks)
	require.NoError(t, err, "Should insert chunks to SQLite")

	// Insert to Weaviate
	for _, c := range modelChunks {
		_, err := weaviateClient.InsertChunk(ctx, c.ID, c.DocumentID, c.Text, c.PageNum, c.SectionPath)
		require.NoError(t, err, "Should insert chunk to Weaviate")
	}

	// Verify counts
	chunkCount, err := s.GetChunkCount()
	require.NoError(t, err)
	assert.Equal(t, len(chunks), chunkCount, "Chunk count should match")

	t.Logf("Successfully ingested document with %d chunks", chunkCount)

	// Cleanup Weaviate data
	err = weaviateClient.DeleteByDocumentID(ctx, docID)
	require.NoError(t, err)
}

// TestIngestDuplicateDetection tests duplicate detection (T078)
func TestIngestDuplicateDetection(t *testing.T) {
	s, _, _, _, cleanup := setupTestEnvironment(t)
	defer cleanup()

	pdfPath := findTestPDF(t)

	// Calculate checksum
	checksum, err := store.CalculateChecksum(pdfPath)
	require.NoError(t, err)

	// Insert a document with this checksum
	doc := &models.Document{
		Filename:   filepath.Base(pdfPath),
		Filepath:   pdfPath,
		SourceType: models.SourceTypeUnknown,
		Checksum:   checksum,
		IngestedAt: time.Now(),
	}
	_, err = s.InsertDocument(doc)
	require.NoError(t, err)

	// Try to check for duplicate
	existing, err := s.GetDocumentByChecksum(checksum)
	require.NoError(t, err)
	assert.NotNil(t, existing, "Should find existing document")
	assert.Equal(t, checksum, existing.Checksum)

	// Attempt to insert again should fail
	doc2 := &models.Document{
		Filename:   "duplicate.pdf",
		Filepath:   "/different/path/duplicate.pdf",
		SourceType: models.SourceTypeUnknown,
		Checksum:   checksum,
		IngestedAt: time.Now(),
	}
	_, err = s.InsertDocument(doc2)
	assert.Error(t, err, "Should fail due to duplicate checksum")
}
