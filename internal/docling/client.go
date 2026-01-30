package docling

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Client provides HTTP access to docling-serve
type Client struct {
	baseURL    string
	httpClient *http.Client
	debug      bool
}

// NewClient creates a new Docling client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
		debug: false,
	}
}

// SetDebug enables or disables debug logging
func (c *Client) SetDebug(debug bool) {
	c.debug = debug
}

// Health checks if the Docling service is healthy
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// convertResponse wraps the Docling API response envelope
type convertResponse struct {
	Document struct {
		Filename    string           `json:"filename"`
		JSONContent *DoclingDocument `json:"json_content"`
	} `json:"document"`
}

// ConvertFile uploads a file to Docling for conversion
func (c *Client) ConvertFile(ctx context.Context, filePath string) (*DoclingDocument, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add file part - Docling API expects "files" (plural)
	filename := filepath.Base(filePath)
	part, err := writer.CreateFormFile("files", filename)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("copy file to form: %w", err)
	}

	// Request JSON output format
	if err := writer.WriteField("to_formats", "json"); err != nil {
		return nil, fmt.Errorf("write to_formats field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/convert/file", &body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	if c.debug {
		fmt.Printf("[DEBUG] Docling request: POST %s/v1/convert/file (file: %s)\n", c.baseURL, filename)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("convert request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if c.debug {
		fmt.Printf("[DEBUG] Docling response status: %d\n", resp.StatusCode)
		if len(respBody) < 500 {
			fmt.Printf("[DEBUG] Docling response body: %s\n", string(respBody))
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("convert failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	// Try parsing the wrapped response format first (newer Docling API)
	var wrapped convertResponse
	if err := json.Unmarshal(respBody, &wrapped); err == nil && wrapped.Document.JSONContent != nil {
		doc := wrapped.Document.JSONContent
		if doc.Name == "" {
			doc.Name = wrapped.Document.Filename
		}
		return doc, nil
	}

	// Fall back to direct document format (older API)
	var doc DoclingDocument
	if err := json.Unmarshal(respBody, &doc); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &doc, nil
}
