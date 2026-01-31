package docling

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ProcessorOpts configures the local Docling processor
type ProcessorOpts struct {
	MaxTokens  int
	Overlap    int
	Debug      bool
	PythonPath string
}

// ProcessorOption is a functional option for Processor
type ProcessorOption func(*ProcessorOpts)

// WithMaxTokens sets the maximum tokens per chunk
func WithMaxTokens(n int) ProcessorOption {
	return func(o *ProcessorOpts) {
		o.MaxTokens = n
	}
}

// WithOverlap sets the overlap tokens between chunks
func WithOverlap(n int) ProcessorOption {
	return func(o *ProcessorOpts) {
		o.Overlap = n
	}
}

// WithProcessorDebug enables debug output
func WithProcessorDebug(debug bool) ProcessorOption {
	return func(o *ProcessorOpts) {
		o.Debug = debug
	}
}

// WithPythonPath sets a specific Python interpreter path
func WithPythonPath(path string) ProcessorOption {
	return func(o *ProcessorOpts) {
		o.PythonPath = path
	}
}

// ChunkResult represents a chunk output from the Python pipeline
type ChunkResult struct {
	ID          string   `json:"id"`             // e.g., "#/chunks/0"
	Text        string   `json:"text"`           // Chunk text (may include overlap)
	PageNum     *int     `json:"page_num"`       // Starting page number
	SectionPath []string `json:"section_path"`   // Section headings
	BBox        *BBox    `json:"bbox,omitempty"` // Bounding box (optional)
}

// Processor wraps the local Python Docling pipeline
type Processor struct {
	scriptPath string
	pythonPath string
	opts       ProcessorOpts
}

// NewProcessor creates a new local Docling processor.
// scriptPath is the path to process_document.py.
func NewProcessor(scriptPath string, options ...ProcessorOption) (*Processor, error) {
	opts := ProcessorOpts{
		MaxTokens: 800,
		Overlap:   200,
	}
	for _, opt := range options {
		opt(&opts)
	}

	// Resolve script path
	absScript, err := filepath.Abs(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("resolve script path: %w", err)
	}
	if _, err := os.Stat(absScript); os.IsNotExist(err) {
		return nil, fmt.Errorf("script not found: %s", absScript)
	}

	// Find Python
	pythonPath := opts.PythonPath
	if pythonPath == "" {
		pythonPath, err = findPython()
		if err != nil {
			return nil, fmt.Errorf("find python: %w", err)
		}
	}

	return &Processor{
		scriptPath: absScript,
		pythonPath: pythonPath,
		opts:       opts,
	}, nil
}

// findPython locates a Python 3 executable
func findPython() (string, error) {
	names := []string{"python3", "python"}
	for _, name := range names {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("python3 not found in PATH")
}

// ProcessFile parses and chunks a document file.
// Returns the chunks as a slice of ChunkResult.
func (p *Processor) ProcessFile(ctx context.Context, filePath string) ([]ChunkResult, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("resolve file path: %w", err)
	}

	args := []string{
		p.scriptPath,
		"--file", absPath,
		"--max-tokens", fmt.Sprintf("%d", p.opts.MaxTokens),
		"--overlap", fmt.Sprintf("%d", p.opts.Overlap),
	}
	if p.opts.Debug {
		args = append(args, "--debug")
	}

	cmd := exec.CommandContext(ctx, p.pythonPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if p.opts.Debug {
		fmt.Printf("[DEBUG] Running: %s %v\n", p.pythonPath, args)
	}

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("process document: %w\nstderr: %s", err, stderr.String())
	}

	if p.opts.Debug && stderr.Len() > 0 {
		fmt.Printf("[DEBUG] Python stderr:\n%s", stderr.String())
	}

	// Parse JSON lines output
	var chunks []ChunkResult
	scanner := bufio.NewScanner(&stdout)
	// Increase scanner buffer for large chunks
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var chunk ChunkResult
		if err := json.Unmarshal(line, &chunk); err != nil {
			return nil, fmt.Errorf("parse chunk output: %w\nline: %s", err, string(line))
		}
		chunks = append(chunks, chunk)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read output: %w", err)
	}

	if p.opts.Debug {
		fmt.Printf("[DEBUG] Received %d chunks from Python pipeline\n", len(chunks))
	}

	return chunks, nil
}

// CheckDependencies verifies that Python and required packages are available
func (p *Processor) CheckDependencies(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, p.pythonPath, "-c",
		"import docling; import docling_core; import transformers; print('ok')")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("python dependencies check failed: %w\nstderr: %s", err, stderr.String())
	}

	if string(bytes.TrimSpace(output)) != "ok" {
		return fmt.Errorf("unexpected output from dependency check: %s", string(output))
	}

	return nil
}
