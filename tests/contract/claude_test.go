//go:build integration

package contract

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/foundry-zero/aqe/internal/claude"
)

// TestClaudeCLIAvailable verifies the Claude CLI is installed
func TestClaudeCLIAvailable(t *testing.T) {
	_, err := exec.LookPath("claude")
	require.NoError(t, err, "Claude CLI should be available in PATH")
}

// TestClaudeOutputFormat verifies Claude returns valid JSON (T080)
func TestClaudeOutputFormat(t *testing.T) {
	// Create wrapper
	wrapper, err := claude.NewWrapper(120 * time.Second)
	if err != nil {
		t.Skipf("Claude CLI not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Create a minimal test task
	task := claude.ExtractionTask{
		Topic: "computer science education",
		Chunks: []claude.ChunkInput{
			{
				ID:         "#/chunks/test-1",
				Text:       "Computer science education is crucial for developing computational thinking skills in students.",
				DocumentID: 1,
			},
			{
				ID:         "#/chunks/test-2",
				Text:       "The history of pizza making dates back to ancient civilizations in the Mediterranean.",
				DocumentID: 1,
			},
			{
				ID:         "#/chunks/test-3",
				Text:       "Modern programming languages like Python and JavaScript are widely taught in universities.",
				DocumentID: 1,
			},
		},
	}

	// Call Claude
	response, err := wrapper.ExtractQuotes(ctx, task)
	require.NoError(t, err, "ExtractQuotes should succeed")
	require.NotNil(t, response, "Should return a response")

	// Verify response structure
	t.Logf("Claude returned %d selected chunks", len(response.SelectedChunks))

	for i, chunk := range response.SelectedChunks {
		// Verify chunk_id is one of the input chunks
		validID := chunk.ChunkID == "#/chunks/test-1" ||
			chunk.ChunkID == "#/chunks/test-2" ||
			chunk.ChunkID == "#/chunks/test-3"
		assert.True(t, validID, "Selected chunk[%d] should have valid chunk_id, got %q", i, chunk.ChunkID)

		// Verify relevance is in bounds
		assert.GreaterOrEqual(t, chunk.Relevance, 0, "Relevance should be >= 0")
		assert.LessOrEqual(t, chunk.Relevance, 100, "Relevance should be <= 100")

		// Verify explanation is present
		assert.NotEmpty(t, chunk.Explanation, "Explanation should not be empty")

		t.Logf("  Chunk %s: relevance=%d, explanation=%q", chunk.ChunkID, chunk.Relevance, truncate(chunk.Explanation, 50))
	}

	// Verify that the relevant chunks were identified
	// The pizza chunk should NOT be selected (or have low relevance) for "computer science education"
	for _, chunk := range response.SelectedChunks {
		if chunk.ChunkID == "#/chunks/test-2" {
			assert.Less(t, chunk.Relevance, 50, "Pizza chunk should have low relevance for CS education topic")
		}
	}
}

// TestClaudeValidateResponse verifies response validation logic
func TestClaudeValidateResponse(t *testing.T) {
	chunks := []claude.ChunkInput{
		{ID: "#/chunks/1", Text: "Test text 1", DocumentID: 1},
		{ID: "#/chunks/2", Text: "Test text 2", DocumentID: 1},
	}

	tests := []struct {
		name      string
		response  *claude.ExtractionResponse
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid response",
			response: &claude.ExtractionResponse{
				SelectedChunks: []claude.SelectedChunk{
					{ChunkID: "#/chunks/1", Relevance: 85, Explanation: "Highly relevant"},
				},
			},
			wantError: false,
		},
		{
			name: "unknown chunk_id",
			response: &claude.ExtractionResponse{
				SelectedChunks: []claude.SelectedChunk{
					{ChunkID: "#/chunks/999", Relevance: 85, Explanation: "Test"},
				},
			},
			wantError: true,
			errorMsg:  "unknown chunk_id",
		},
		{
			name: "relevance too high",
			response: &claude.ExtractionResponse{
				SelectedChunks: []claude.SelectedChunk{
					{ChunkID: "#/chunks/1", Relevance: 150, Explanation: "Test"},
				},
			},
			wantError: true,
			errorMsg:  "out of bounds",
		},
		{
			name: "relevance negative",
			response: &claude.ExtractionResponse{
				SelectedChunks: []claude.SelectedChunk{
					{ChunkID: "#/chunks/1", Relevance: -10, Explanation: "Test"},
				},
			},
			wantError: true,
			errorMsg:  "out of bounds",
		},
		{
			name: "empty explanation",
			response: &claude.ExtractionResponse{
				SelectedChunks: []claude.SelectedChunk{
					{ChunkID: "#/chunks/1", Relevance: 85, Explanation: ""},
				},
			},
			wantError: true,
			errorMsg:  "explanation is empty",
		},
		{
			name: "empty response is valid",
			response: &claude.ExtractionResponse{
				SelectedChunks: []claude.SelectedChunk{},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := claude.ValidateResponse(tt.response, chunks)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
