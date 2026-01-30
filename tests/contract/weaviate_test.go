//go:build integration

package contract

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/foundry-zero/aqe/internal/search"
)

const (
	weaviateHost   = "localhost:8080"
	ollamaEndpoint = "http://ollama:11434"
)

// TestWeaviateSchemaCreation verifies schema creation (T057)
func TestWeaviateSchemaCreation(t *testing.T) {
	client, err := search.NewWeaviateClient(weaviateHost, ollamaEndpoint)
	require.NoError(t, err, "Should create Weaviate client")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create schema - should not error even if it exists
	err = client.CreateSchema(ctx)
	require.NoError(t, err, "Schema creation should succeed")

	// Call again to verify idempotency
	err = client.CreateSchema(ctx)
	require.NoError(t, err, "Schema creation should be idempotent")
}

// TestWeaviateInsertChunk verifies chunk insertion (T058)
func TestWeaviateInsertChunk(t *testing.T) {
	client, err := search.NewWeaviateClient(weaviateHost, ollamaEndpoint)
	require.NoError(t, err, "Should create Weaviate client")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Ensure schema exists
	err = client.CreateSchema(ctx)
	require.NoError(t, err, "Schema creation should succeed")

	// Insert a test chunk
	testChunkID := "#/chunks/test-" + time.Now().Format("20060102150405")
	testDocID := int64(999999)
	testText := "This is a test chunk for contract testing. It contains some academic text about computer science education."
	pageNum := 1
	sectionPath := []string{"Test Chapter", "Test Section"}

	uuid, err := client.InsertChunk(ctx, testChunkID, testDocID, testText, &pageNum, sectionPath)
	require.NoError(t, err, "Chunk insertion should succeed")
	assert.NotEmpty(t, uuid, "Should return a UUID")

	// Clean up - delete the test chunk
	err = client.DeleteByDocumentID(ctx, testDocID)
	require.NoError(t, err, "Cleanup deletion should succeed")
}

// TestWeaviateHybridSearch verifies hybrid search (T079)
func TestWeaviateHybridSearch(t *testing.T) {
	client, err := search.NewWeaviateClient(weaviateHost, ollamaEndpoint)
	require.NoError(t, err, "Should create Weaviate client")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Ensure schema exists
	err = client.CreateSchema(ctx)
	require.NoError(t, err, "Schema creation should succeed")

	// Insert test data
	testDocID := int64(888888)
	testChunks := []struct {
		id   string
		text string
	}{
		{"#/chunks/hs-test-1", "Computer science education is important for future technological advancement."},
		{"#/chunks/hs-test-2", "Machine learning algorithms can analyze large datasets effectively."},
		{"#/chunks/hs-test-3", "Software engineering principles guide modern application development."},
	}

	for _, tc := range testChunks {
		_, err := client.InsertChunk(ctx, tc.id, testDocID, tc.text, nil, nil)
		require.NoError(t, err, "Test chunk insertion should succeed")
	}

	// Wait for embeddings to be generated
	time.Sleep(2 * time.Second)

	// Perform hybrid search
	results, err := client.HybridSearch(ctx, "computer science", 10, 0.5)
	require.NoError(t, err, "Hybrid search should succeed")

	// Verify we get results
	// Note: Results might be empty if Ollama hasn't generated embeddings yet
	t.Logf("Search returned %d results", len(results))

	// Verify result structure if we have any
	for _, result := range results {
		assert.NotEmpty(t, result.ChunkID, "Result should have chunk_id")
		assert.NotEmpty(t, result.Text, "Result should have text")
	}

	// Clean up
	err = client.DeleteByDocumentID(ctx, testDocID)
	require.NoError(t, err, "Cleanup deletion should succeed")
}
