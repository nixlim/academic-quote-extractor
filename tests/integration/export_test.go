//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/foundry-zero/aqe/internal/harvard"
	"github.com/foundry-zero/aqe/internal/models"
	"github.com/foundry-zero/aqe/internal/store"
)

// setupExportTestData creates test data for export tests
func setupExportTestData(t *testing.T, s *store.Store) int64 {
	// Create a test document with full metadata
	title := "Introduction to Computer Science"
	year := 2023
	publisher := "Academic Press"
	doc := &models.Document{
		Filename:   "cs-intro.pdf",
		Filepath:   "/test/cs-intro.pdf",
		Title:      &title,
		Authors:    []string{"Smith, John", "Jones, Mary"},
		Year:       &year,
		Publisher:  &publisher,
		SourceType: models.SourceTypeBook,
		Checksum:   "export-test-" + time.Now().Format("20060102150405"),
		IngestedAt: time.Now(),
	}
	docID, err := s.InsertDocument(doc)
	require.NoError(t, err)

	// Create test chunks
	pageNum := 42
	chunks := []*models.Chunk{
		{
			ID:          "#/chunks/export-1",
			DocumentID:  docID,
			Text:        "Computer science education is fundamental to modern technological literacy.",
			PageNum:     &pageNum,
			SectionPath: []string{"Chapter 1", "Introduction"},
		},
		{
			ID:          "#/chunks/export-2",
			DocumentID:  docID,
			Text:        "Programming skills enable creative problem-solving across disciplines.",
			PageNum:     &pageNum,
			SectionPath: []string{"Chapter 1", "Skills"},
		},
	}
	err = s.InsertChunks(chunks)
	require.NoError(t, err)

	// Create an extraction
	extraction := &models.Extraction{
		Topic:     "computer science education",
		CreatedAt: time.Now(),
	}
	extractionID, err := s.InsertExtraction(extraction)
	require.NoError(t, err)

	// Create extracted quotes
	quotes := []*models.ExtractedQuote{
		{
			ExtractionID:   extractionID,
			ChunkID:        "#/chunks/export-1",
			RelevanceScore: 95,
			Explanation:    "Directly addresses the importance of CS education.",
		},
		{
			ExtractionID:   extractionID,
			ChunkID:        "#/chunks/export-2",
			RelevanceScore: 85,
			Explanation:    "Discusses skills developed through programming education.",
		},
	}
	err = s.InsertExtractedQuotes(quotes)
	require.NoError(t, err)

	return extractionID
}

// TestExportMarkdown tests markdown export (T108)
func TestExportMarkdown(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "aqe-export-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	s, err := store.NewStore(dbPath)
	require.NoError(t, err)
	defer s.Close()

	err = s.RunMigrations()
	require.NoError(t, err)

	extractionID := setupExportTestData(t, s)

	// Get extraction data
	extraction, err := s.GetExtractionByID(extractionID)
	require.NoError(t, err)
	require.NotNil(t, extraction)

	quotes, err := s.GetQuotesByExtractionID(extractionID)
	require.NoError(t, err)
	require.Len(t, quotes, 2)

	// Build markdown output
	var sb strings.Builder
	formatter := harvard.NewFormatter("harvard_us")

	sb.WriteString("# Extraction: ")
	sb.WriteString(extraction.Topic)
	sb.WriteString("\n\n")
	sb.WriteString("## Quotes\n\n")

	seenDocs := make(map[int64]*models.Document)

	for _, q := range quotes {
		chunk, err := s.GetChunkByID(q.ChunkID)
		require.NoError(t, err)

		doc, err := s.GetDocumentByID(chunk.DocumentID)
		require.NoError(t, err)
		seenDocs[doc.ID] = doc

		// Format as blockquote
		sb.WriteString("> ")
		sb.WriteString(chunk.Text)
		sb.WriteString("\n>\n> ")

		// Add in-text citation
		ref := buildReference(doc, chunk.PageNum)
		sb.WriteString(formatter.FormatInText(ref))
		sb.WriteString("\n\n")

		sb.WriteString("**Relevance:** ")
		sb.WriteString(string(rune('0' + q.RelevanceScore/10)))
		sb.WriteString(string(rune('0' + q.RelevanceScore%10)))
		sb.WriteString("%\n\n")

		sb.WriteString("**Explanation:** ")
		sb.WriteString(q.Explanation)
		sb.WriteString("\n\n---\n\n")
	}

	// Add bibliography
	sb.WriteString("## Bibliography\n\n")
	for _, doc := range seenDocs {
		ref := buildReference(doc, nil)
		sb.WriteString("- ")
		sb.WriteString(formatter.FormatFull(ref))
		sb.WriteString("\n")
	}

	markdown := sb.String()

	// Verify markdown content (SC-007)
	assert.Contains(t, markdown, "# Extraction: computer science education")
	assert.Contains(t, markdown, "> Computer science education")
	assert.Contains(t, markdown, "(Smith and Jones, 2023")
	assert.Contains(t, markdown, "## Bibliography")
	assert.Contains(t, markdown, "Smith, J. and Jones, M.")

	t.Log("Markdown output:")
	t.Log(markdown)
}

// TestExportJSON tests JSON export (T109)
func TestExportJSON(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "aqe-export-json-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	s, err := store.NewStore(dbPath)
	require.NoError(t, err)
	defer s.Close()

	err = s.RunMigrations()
	require.NoError(t, err)

	extractionID := setupExportTestData(t, s)

	// Get extraction data
	extraction, err := s.GetExtractionByID(extractionID)
	require.NoError(t, err)

	quotes, err := s.GetQuotesByExtractionID(extractionID)
	require.NoError(t, err)

	// Build JSON output structure
	type QuoteJSON struct {
		Text        string   `json:"text"`
		Relevance   int      `json:"relevance"`
		Explanation string   `json:"explanation"`
		Citation    string   `json:"citation"`
		PageNum     *int     `json:"page_num,omitempty"`
		SectionPath []string `json:"section_path,omitempty"`
	}

	type DocumentJSON struct {
		Title    string   `json:"title"`
		Authors  []string `json:"authors"`
		Year     int      `json:"year"`
		Citation string   `json:"citation"`
	}

	type ExportJSON struct {
		Topic      string         `json:"topic"`
		CreatedAt  string         `json:"created_at"`
		Quotes     []QuoteJSON    `json:"quotes"`
		References []DocumentJSON `json:"references"`
	}

	formatter := harvard.NewFormatter("harvard_us")
	seenDocs := make(map[int64]*models.Document)

	var quotesJSON []QuoteJSON
	for _, q := range quotes {
		chunk, err := s.GetChunkByID(q.ChunkID)
		require.NoError(t, err)

		doc, err := s.GetDocumentByID(chunk.DocumentID)
		require.NoError(t, err)
		seenDocs[doc.ID] = doc

		ref := buildReference(doc, chunk.PageNum)
		quotesJSON = append(quotesJSON, QuoteJSON{
			Text:        chunk.Text,
			Relevance:   q.RelevanceScore,
			Explanation: q.Explanation,
			Citation:    formatter.FormatInText(ref),
			PageNum:     chunk.PageNum,
			SectionPath: chunk.SectionPath,
		})
	}

	var refsJSON []DocumentJSON
	for _, doc := range seenDocs {
		ref := buildReference(doc, nil)
		refsJSON = append(refsJSON, DocumentJSON{
			Title:    *doc.Title,
			Authors:  doc.Authors,
			Year:     *doc.Year,
			Citation: formatter.FormatFull(ref),
		})
	}

	export := ExportJSON{
		Topic:      extraction.Topic,
		CreatedAt:  extraction.CreatedAt.Format(time.RFC3339),
		Quotes:     quotesJSON,
		References: refsJSON,
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(export, "", "  ")
	require.NoError(t, err)

	jsonStr := string(jsonBytes)

	// Verify JSON is valid and parseable (SC-006)
	var parsed ExportJSON
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err, "JSON should be valid and parseable")

	assert.Equal(t, "computer science education", parsed.Topic)
	assert.Len(t, parsed.Quotes, 2)
	assert.Len(t, parsed.References, 1)

	t.Log("JSON output:")
	t.Log(jsonStr)
}

// buildReference constructs a Harvard reference from a document
func buildReference(doc *models.Document, pageNum *int) harvard.Reference {
	var authors []harvard.Author
	for _, a := range doc.Authors {
		// Parse "LastName, FirstName" format
		parts := strings.SplitN(a, ", ", 2)
		author := harvard.Author{LastName: parts[0]}
		if len(parts) > 1 {
			author.FirstName = parts[1]
		}
		authors = append(authors, author)
	}

	ref := harvard.Reference{
		Authors:    authors,
		SourceType: doc.SourceType,
	}

	if doc.Title != nil {
		ref.Title = *doc.Title
	}
	if doc.Year != nil {
		ref.Year = *doc.Year
	}
	if doc.Publisher != nil {
		ref.Publisher = *doc.Publisher
	}
	if pageNum != nil {
		ref.PageNum = pageNum
	}

	return ref
}
