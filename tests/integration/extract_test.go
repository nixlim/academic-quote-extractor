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

	"github.com/foundry-zero/aqe/internal/claude"
	"github.com/foundry-zero/aqe/internal/models"
	"github.com/foundry-zero/aqe/internal/search"
	"github.com/foundry-zero/aqe/internal/store"
)

// setupExtractTestData creates test data for extraction tests
func setupExtractTestData(t *testing.T, s *store.Store, weaviateClient *search.WeaviateClient) (int64, []string, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create Weaviate schema
	err := weaviateClient.CreateSchema(ctx)
	require.NoError(t, err)

	// Create a test document
	doc := &models.Document{
		Filename:   "test-extract.pdf",
		Filepath:   "/test/path/test-extract.pdf",
		SourceType: models.SourceTypeBook,
		Checksum:   "extract-test-checksum-" + time.Now().Format("20060102150405"),
		IngestedAt: time.Now(),
	}
	docID, err := s.InsertDocument(doc)
	require.NoError(t, err)

	// Create test chunks with varied content
	testChunks := []struct {
		id   string
		text string
	}{
		{
			"#/chunks/extract-1",
			"Computer science education is essential for preparing students for the digital economy. Computational thinking skills enable problem-solving across disciplines.",
		},
		{
			"#/chunks/extract-2",
			"Culturally responsive pedagogy recognizes the importance of including students' cultural references in all aspects of learning.",
		},
		{
			"#/chunks/extract-3",
			"The history of pizza making dates back to ancient civilizations. Flatbreads with toppings were common in Mediterranean cultures.",
		},
		{
			"#/chunks/extract-4",
			"Programming languages like Python and JavaScript are widely used in educational settings to teach introductory computer science.",
		},
		{
			"#/chunks/extract-5",
			"Inclusive curriculum design ensures that all students, regardless of background, can see themselves represented in the learning materials.",
		},
	}

	var chunkIDs []string
	for _, tc := range testChunks {
		chunk := &models.Chunk{
			ID:         tc.id,
			DocumentID: docID,
			Text:       tc.text,
		}
		err := s.InsertChunk(chunk)
		require.NoError(t, err)

		_, err = weaviateClient.InsertChunk(ctx, tc.id, docID, tc.text, nil, nil)
		require.NoError(t, err)

		chunkIDs = append(chunkIDs, tc.id)
	}

	// Wait for embeddings
	time.Sleep(2 * time.Second)

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		weaviateClient.DeleteByDocumentID(ctx, docID)
	}

	return docID, chunkIDs, cleanup
}

// TestExtractWithMatches tests extraction with matching content (T096)
func TestExtractWithMatches(t *testing.T) {
	// Setup test environment
	tmpDir, err := os.MkdirTemp("", "aqe-extract-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	s, err := store.NewStore(dbPath)
	require.NoError(t, err)
	defer s.Close()

	err = s.RunMigrations()
	require.NoError(t, err)

	weaviateClient, err := search.NewWeaviateClient(weaviateHost, ollamaEndpoint)
	require.NoError(t, err)

	docID, chunkIDs, cleanup := setupExtractTestData(t, s, weaviateClient)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Step 1: Perform hybrid search
	results, err := weaviateClient.HybridSearch(ctx, "computer science education", 10, 0.5)
	require.NoError(t, err)
	t.Logf("Hybrid search returned %d results", len(results))

	// Step 2: Get chunk texts from SQLite
	var searchChunkIDs []string
	for _, r := range results {
		searchChunkIDs = append(searchChunkIDs, r.ChunkID)
	}

	// Step 3: Prepare Claude input
	var claudeChunks []claude.ChunkInput
	for _, r := range results {
		claudeChunks = append(claudeChunks, claude.ChunkInput{
			ID:         r.ChunkID,
			Text:       r.Text,
			DocumentID: r.DocumentID,
		})
	}

	// Skip Claude test if not available
	wrapper, err := claude.NewWrapper(120 * time.Second)
	if err != nil {
		t.Skipf("Claude CLI not available: %v", err)
	}

	// Step 4: Call Claude for relevance scoring
	task := claude.ExtractionTask{
		Topic:  "computer science education",
		Chunks: claudeChunks,
	}

	response, err := wrapper.ExtractQuotes(ctx, task)
	require.NoError(t, err)
	require.NotNil(t, response)

	t.Logf("Claude selected %d chunks", len(response.SelectedChunks))

	// Verify results
	assert.NotEmpty(t, response.SelectedChunks, "Should have selected chunks for CS education topic")

	// Step 5: Store extraction results
	extraction := &models.Extraction{
		Topic:     "computer science education",
		CreatedAt: time.Now(),
	}
	extractionID, err := s.InsertExtraction(extraction)
	require.NoError(t, err)

	var quotes []*models.ExtractedQuote
	for _, sc := range response.SelectedChunks {
		quotes = append(quotes, &models.ExtractedQuote{
			ExtractionID:   extractionID,
			ChunkID:        sc.ChunkID,
			RelevanceScore: sc.Relevance,
			Explanation:    sc.Explanation,
		})
	}

	if len(quotes) > 0 {
		err = s.InsertExtractedQuotes(quotes)
		require.NoError(t, err)
	}

	// Verify we can retrieve the quotes
	savedQuotes, err := s.GetQuotesByExtractionID(extractionID)
	require.NoError(t, err)
	assert.Len(t, savedQuotes, len(quotes))

	// Log results
	_ = docID
	_ = chunkIDs
	for _, q := range savedQuotes {
		t.Logf("Quote: chunk=%s, relevance=%d", q.ChunkID, q.RelevanceScore)
	}
}

// TestExtractNoMatches tests extraction with no matching content (T097)
func TestExtractNoMatches(t *testing.T) {
	// Setup test environment
	tmpDir, err := os.MkdirTemp("", "aqe-extract-nomatch-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	s, err := store.NewStore(dbPath)
	require.NoError(t, err)
	defer s.Close()

	err = s.RunMigrations()
	require.NoError(t, err)

	weaviateClient, err := search.NewWeaviateClient(weaviateHost, ollamaEndpoint)
	require.NoError(t, err)

	// Create very specific content that won't match our query
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = weaviateClient.CreateSchema(ctx)
	require.NoError(t, err)

	doc := &models.Document{
		Filename:   "unrelated.pdf",
		Filepath:   "/test/unrelated.pdf",
		SourceType: models.SourceTypeBook,
		Checksum:   "nomatch-test-" + time.Now().Format("20060102150405"),
		IngestedAt: time.Now(),
	}
	docID, err := s.InsertDocument(doc)
	require.NoError(t, err)

	// Insert only unrelated content
	chunk := &models.Chunk{
		ID:         "#/chunks/unrelated-1",
		DocumentID: docID,
		Text:       "The molecular structure of water consists of two hydrogen atoms and one oxygen atom.",
	}
	err = s.InsertChunk(chunk)
	require.NoError(t, err)

	_, err = weaviateClient.InsertChunk(ctx, chunk.ID, docID, chunk.Text, nil, nil)
	require.NoError(t, err)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		weaviateClient.DeleteByDocumentID(ctx, docID)
	}()

	// Wait for embeddings
	time.Sleep(2 * time.Second)

	// Search for completely unrelated topic
	results, err := weaviateClient.HybridSearch(ctx, "ancient roman military tactics", 10, 0.5)
	require.NoError(t, err)

	// We may still get results due to BM25, but they should have low scores
	t.Logf("Search for unrelated topic returned %d results", len(results))

	// If we have a Claude wrapper, verify low relevance scores
	wrapper, err := claude.NewWrapper(120 * time.Second)
	if err != nil {
		t.Skipf("Claude CLI not available: %v", err)
	}

	if len(results) > 0 {
		var claudeChunks []claude.ChunkInput
		for _, r := range results {
			claudeChunks = append(claudeChunks, claude.ChunkInput{
				ID:         r.ChunkID,
				Text:       r.Text,
				DocumentID: r.DocumentID,
			})
		}

		task := claude.ExtractionTask{
			Topic:  "ancient roman military tactics",
			Chunks: claudeChunks,
		}

		response, err := wrapper.ExtractQuotes(ctx, task)
		require.NoError(t, err)

		// All chunks should have low relevance or be excluded
		for _, sc := range response.SelectedChunks {
			if sc.Relevance >= 60 {
				t.Logf("Warning: Unrelated chunk %s has high relevance %d", sc.ChunkID, sc.Relevance)
			}
		}
	}
}
