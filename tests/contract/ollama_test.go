//go:build integration

package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const ollamaURL = "http://localhost:11434"

// OllamaTagsResponse represents the response from /api/tags
type OllamaTagsResponse struct {
	Models []OllamaModel `json:"models"`
}

// OllamaModel represents a model in Ollama
type OllamaModel struct {
	Name       string `json:"name"`
	Model      string `json:"model"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
}

// TestOllamaModelAvailable verifies nomic-embed-text is available (T059)
func TestOllamaModelAvailable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check if Ollama is running
	req, err := http.NewRequestWithContext(ctx, "GET", ollamaURL+"/api/tags", nil)
	require.NoError(t, err, "Should create request")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err, "Ollama should be reachable")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Ollama should return 200")

	// Parse response
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Should read response body")

	var tagsResp OllamaTagsResponse
	err = json.Unmarshal(body, &tagsResp)
	require.NoError(t, err, "Should parse JSON response")

	// Check for nomic-embed-text model
	var foundModel bool
	var modelNames []string
	for _, model := range tagsResp.Models {
		modelNames = append(modelNames, model.Name)
		if model.Name == "nomic-embed-text:latest" || model.Name == "nomic-embed-text" {
			foundModel = true
		}
	}

	if !foundModel {
		t.Logf("Available models: %v", modelNames)
		t.Log("To pull the required model, run:")
		t.Log("  docker exec -it ollama ollama pull nomic-embed-text")
	}

	assert.True(t, foundModel, "nomic-embed-text model should be available")
}

// TestOllamaHealth verifies Ollama is running
func TestOllamaHealth(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ollamaURL+"/", nil)
	require.NoError(t, err, "Should create request")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err, "Ollama should be reachable")
	defer resp.Body.Close()

	// Ollama returns "Ollama is running" on the root endpoint
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Should read response body")

	assert.Contains(t, string(body), "Ollama is running", "Ollama should indicate it's running")
}

// TestOllamaEmbeddings verifies that embeddings can be generated
func TestOllamaEmbeddings(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First verify the model is available
	TestOllamaModelAvailable(t)
	if t.Failed() {
		t.Skip("Skipping embeddings test - model not available")
	}

	// Try to generate an embedding
	payload := `{"model": "nomic-embed-text", "prompt": "This is a test sentence for embedding generation."}`
	req, err := http.NewRequestWithContext(ctx, "POST", ollamaURL+"/api/embeddings",
		io.NopCloser(jsonReader(payload)))
	require.NoError(t, err, "Should create request")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err, "Embedding request should succeed")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Should read response body")

	if resp.StatusCode != http.StatusOK {
		t.Logf("Response: %s", string(body))
	}
	require.Equal(t, http.StatusOK, resp.StatusCode, "Should return 200")

	// Parse and verify embedding
	var embeddingResp struct {
		Embedding []float64 `json:"embedding"`
	}
	err = json.Unmarshal(body, &embeddingResp)
	require.NoError(t, err, "Should parse embedding response")

	assert.NotEmpty(t, embeddingResp.Embedding, "Should return non-empty embedding")
	t.Logf("Generated embedding with %d dimensions", len(embeddingResp.Embedding))
}

// jsonReader converts a string to an io.Reader
func jsonReader(s string) io.Reader {
	return &jsonStringReader{s: s}
}

type jsonStringReader struct {
	s string
	i int
}

func (r *jsonStringReader) Read(p []byte) (n int, err error) {
	if r.i >= len(r.s) {
		return 0, io.EOF
	}
	n = copy(p, r.s[r.i:])
	r.i += n
	return n, nil
}

func (r *jsonStringReader) Close() error {
	return nil
}

// Helper to print available instructions
func init() {
	// Print instructions if env var is set
	if testing.Verbose() {
		fmt.Println("Ollama Contract Tests")
		fmt.Println("====================")
		fmt.Println("Ensure Ollama is running and has nomic-embed-text model:")
		fmt.Println("  docker compose up -d ollama")
		fmt.Println("  docker exec -it ollama ollama pull nomic-embed-text")
		fmt.Println()
	}
}
