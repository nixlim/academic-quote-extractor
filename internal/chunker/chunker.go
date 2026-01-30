package chunker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ChunkOutput represents a chunk from the Python chunker
type ChunkOutput struct {
	ID          string   `json:"id"`             // Docling self_ref
	Text        string   `json:"text"`           // Chunk text content
	PageNum     *int     `json:"page_num"`       // Starting page number
	SectionPath []string `json:"section_path"`   // Section headings
	BBox        *BBox    `json:"bbox,omitempty"` // Bounding box
}

// BBox represents a bounding box
type BBox struct {
	L           float64 `json:"l"`
	T           float64 `json:"t"`
	R           float64 `json:"r"`
	B           float64 `json:"b"`
	CoordOrigin string  `json:"coord_origin,omitempty"`
}

// Chunker wraps the Python chunking script
type Chunker struct {
	pythonPath string
	scriptPath string
}

// NewChunker creates a new Chunker instance
func NewChunker(scriptPath string) (*Chunker, error) {
	// Find Python executable
	pythonPath, err := findPython()
	if err != nil {
		return nil, fmt.Errorf("find python: %w", err)
	}

	// Verify script exists
	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("resolve script path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("chunker script not found: %s", absPath)
	}

	return &Chunker{
		pythonPath: pythonPath,
		scriptPath: absPath,
	}, nil
}

// findPython locates a Python 3 executable
func findPython() (string, error) {
	// Try common Python 3 names
	names := []string{"python3", "python"}
	for _, name := range names {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("python3 not found in PATH")
}

// ChunkDocument processes a Docling document and returns chunks
func (c *Chunker) ChunkDocument(ctx context.Context, docJSON []byte) ([]ChunkOutput, error) {
	// Prepare command
	cmd := exec.CommandContext(ctx, c.pythonPath, c.scriptPath)
	cmd.Stdin = bytes.NewReader(docJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the script
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run chunker: %w\nstderr: %s", err, stderr.String())
	}

	// Parse JSON lines output
	var chunks []ChunkOutput
	decoder := json.NewDecoder(&stdout)
	for decoder.More() {
		var chunk ChunkOutput
		if err := decoder.Decode(&chunk); err != nil {
			return nil, fmt.Errorf("parse chunk output: %w", err)
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}
