package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// CLIEnvelope represents the JSON envelope from `claude --output-format json`.
// The CLI wraps responses in this structure; the actual content is a string in the Result field.
type CLIEnvelope struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype"`
	IsError   bool   `json:"is_error"`
	Result    string `json:"result"`
	SessionID string `json:"session_id"`
}

// ExtractionTask represents the input for a quote extraction request
type ExtractionTask struct {
	Topic  string       `json:"topic"`
	Chunks []ChunkInput `json:"chunks"`
}

// ChunkInput represents a candidate chunk for extraction
type ChunkInput struct {
	ID          string   `json:"id"`
	Text        string   `json:"text"`
	PageNum     *int     `json:"page_num,omitempty"`
	SectionPath []string `json:"section_path,omitempty"`
	DocumentID  int64    `json:"document_id"`
	PrevContext string   `json:"prev_context,omitempty"` // Truncated text from previous chunk (read-only context)
	NextContext string   `json:"next_context,omitempty"` // Truncated text from next chunk (read-only context)
}

// ExtractionResponse is the expected response from Claude
type ExtractionResponse struct {
	SelectedChunks []SelectedChunk `json:"selected_chunks"`
}

// SelectedChunk represents a single selected quote
type SelectedChunk struct {
	ChunkID     string `json:"chunk_id"`
	Relevance   int    `json:"relevance"`
	Explanation string `json:"explanation"`
}

// Wrapper provides access to the Claude CLI
type Wrapper struct {
	cliPath string
	timeout time.Duration
	debug   bool
}

// NewWrapper creates a new Claude CLI wrapper
func NewWrapper(timeout time.Duration) (*Wrapper, error) {
	// Find claude CLI
	cliPath, err := exec.LookPath("claude")
	if err != nil {
		return nil, fmt.Errorf("claude CLI not found in PATH: %w", err)
	}

	return &Wrapper{
		cliPath: cliPath,
		timeout: timeout,
		debug:   false,
	}, nil
}

// SetDebug enables or disables debug logging
func (w *Wrapper) SetDebug(debug bool) {
	w.debug = debug
}

// ExtractQuotes calls Claude to score and select relevant quotes
func (w *Wrapper) ExtractQuotes(ctx context.Context, task ExtractionTask) (*ExtractionResponse, error) {
	// Build the prompt
	prompt, err := w.buildPrompt(task)
	if err != nil {
		return nil, fmt.Errorf("build prompt: %w", err)
	}

	if w.debug {
		fmt.Printf("[DEBUG] Claude prompt length: %d characters\n", len(prompt))
	}

	// Create command with timeout context
	ctx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	// IMPORTANT: -p flag must be LAST
	cmd := exec.CommandContext(ctx, w.cliPath,
		"--print",
		"--output-format", "json",
		"-p", prompt,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run Claude
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run claude: %w\nstderr: %s", err, stderr.String())
	}

	// Parse the response.
	// Claude CLI with --output-format json wraps output in an envelope:
	//   {"type":"result","subtype":"success","is_error":false,"result":"<escaped JSON string>",...}
	// The actual content we need is inside the "result" field as a string.
	rawOutput := stdout.Bytes()
	if w.debug {
		fmt.Printf("[DEBUG] Raw Claude output (%d bytes): %s\n", len(rawOutput), string(rawOutput))
	}

	var response ExtractionResponse

	// Step 1: Try to unwrap the CLI JSON envelope
	var envelope CLIEnvelope
	if err := json.Unmarshal(rawOutput, &envelope); err == nil && envelope.Type == "result" {
		// Successfully parsed the envelope
		if envelope.IsError {
			return nil, fmt.Errorf("claude returned error: %s", envelope.Result)
		}

		if w.debug {
			fmt.Printf("[DEBUG] Unwrapped envelope, result string length: %d\n", len(envelope.Result))
		}

		// Parse the inner result string as ExtractionResponse
		if err := json.Unmarshal([]byte(envelope.Result), &response); err != nil {
			// The result string may contain non-JSON text wrapping the JSON
			extracted := extractJSON([]byte(envelope.Result))
			if extracted != nil {
				if err := json.Unmarshal(extracted, &response); err != nil {
					return nil, fmt.Errorf("parse inner result: %w\nresult string: %s", err, envelope.Result)
				}
			} else {
				return nil, fmt.Errorf("parse inner result: %w\nresult string: %s", err, envelope.Result)
			}
		}
	} else {
		// Fallback: try parsing as direct ExtractionResponse (no envelope)
		if err := json.Unmarshal(rawOutput, &response); err != nil {
			extracted := extractJSON(rawOutput)
			if extracted != nil {
				if err := json.Unmarshal(extracted, &response); err != nil {
					return nil, fmt.Errorf("parse response: %w\nraw output: %s", err, stdout.String())
				}
			} else {
				return nil, fmt.Errorf("parse response: %w\nraw output: %s", err, stdout.String())
			}
		}
	}

	// Validate the response
	if err := w.validateResponse(&response, task.Chunks); err != nil {
		return nil, fmt.Errorf("validate response: %w", err)
	}

	if w.debug {
		fmt.Printf("[DEBUG] Claude returned %d selected chunks\n", len(response.SelectedChunks))
	}

	return &response, nil
}

// validateResponse checks that the response is valid
func (w *Wrapper) validateResponse(resp *ExtractionResponse, chunks []ChunkInput) error {
	// Build a set of valid chunk IDs
	validIDs := make(map[string]bool)
	for _, c := range chunks {
		validIDs[c.ID] = true
	}

	for i, sc := range resp.SelectedChunks {
		// Check chunk_id exists
		if !validIDs[sc.ChunkID] {
			return fmt.Errorf("selected_chunks[%d]: unknown chunk_id %q", i, sc.ChunkID)
		}

		// Check score bounds
		if sc.Relevance < 0 || sc.Relevance > 100 {
			return fmt.Errorf("selected_chunks[%d]: relevance %d out of bounds (0-100)", i, sc.Relevance)
		}

		// Check explanation not empty
		if sc.Explanation == "" {
			return fmt.Errorf("selected_chunks[%d]: explanation is empty", i)
		}
	}

	return nil
}

// extractJSON attempts to find and extract JSON from a response that may contain other text
func extractJSON(data []byte) []byte {
	// Find the first { and last }
	start := bytes.IndexByte(data, '{')
	end := bytes.LastIndexByte(data, '}')

	if start == -1 || end == -1 || end <= start {
		return nil
	}

	return data[start : end+1]
}

// ValidateResponse is exported for testing
func ValidateResponse(resp *ExtractionResponse, chunks []ChunkInput) error {
	w := &Wrapper{}
	return w.validateResponse(resp, chunks)
}
