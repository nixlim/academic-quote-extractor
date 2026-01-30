package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/foundry-zero/aqe/internal/harvard"
	"github.com/foundry-zero/aqe/internal/models"
	"github.com/foundry-zero/aqe/internal/store"
)

var (
	// Export flags
	exportFormat string
	exportOutput string
)

// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export <extraction-id>",
	Short: "Export an extraction in various formats",
	Long: `Export a saved extraction in Markdown, JSON, or BibTeX format.

Examples:
  aqe export 1                          # Export to stdout as markdown
  aqe export 1 --format json            # Export as JSON
  aqe export 1 --format bibtex          # Export as BibTeX
  aqe export 1 --output quotes.md       # Export to file`,
	Args: cobra.ExactArgs(1),
	RunE: runExport,
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVar(&exportFormat, "format", "markdown", "Output format: markdown, json, bibtex")
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "Output file (default: stdout)")
}

func runExport(cmd *cobra.Command, args []string) error {
	db := GetStore()

	// Parse extraction ID
	var extractionID int64
	if _, err := fmt.Sscanf(args[0], "%d", &extractionID); err != nil {
		return showExtractionError(db, args[0])
	}

	// Get extraction
	extraction, err := getExtraction(db, extractionID)
	if err != nil {
		return showExtractionError(db, args[0])
	}
	if extraction == nil {
		return showExtractionError(db, args[0])
	}

	// Get quotes with details
	quotes, err := getQuotesWithDetails(db, extractionID)
	if err != nil {
		return fmt.Errorf("get quotes: %w", err)
	}

	// Format output
	var output string
	switch exportFormat {
	case "markdown", "md":
		output = formatMarkdown(extraction, quotes)
	case "json":
		output, err = formatJSON(extraction, quotes)
		if err != nil {
			return fmt.Errorf("format json: %w", err)
		}
	case "bibtex", "bib":
		output = formatBibTeX(extraction, quotes)
	default:
		return fmt.Errorf("unknown format: %s (use markdown, json, or bibtex)", exportFormat)
	}

	// Write output
	if exportOutput != "" {
		if err := os.WriteFile(exportOutput, []byte(output), 0644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
		fmt.Printf("Exported to %s\n", exportOutput)
	} else {
		fmt.Print(output)
	}

	return nil
}

func showExtractionError(db *store.Store, arg string) error {
	// List available extractions
	extractions, err := listExtractions(db)
	if err != nil {
		return fmt.Errorf("list extractions: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Error: Invalid extraction ID: %s\n\n", arg)

	if len(extractions) == 0 {
		fmt.Fprintln(os.Stderr, "No extractions available. Run 'aqe extract <topic>' first.")
	} else {
		fmt.Fprintln(os.Stderr, "Available extractions:")
		for _, e := range extractions {
			fmt.Fprintf(os.Stderr, "  %d: %q (%s)\n", e.ID, e.Topic, e.CreatedAt.Format("2006-01-02"))
		}
	}

	os.Exit(ExitUserError)
	return nil
}

func getExtraction(db *store.Store, id int64) (*models.Extraction, error) {
	row := db.DB().QueryRow(`SELECT id, topic, created_at FROM extractions WHERE id = ?`, id)

	var extraction models.Extraction
	err := row.Scan(&extraction.ID, &extraction.Topic, &extraction.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &extraction, nil
}

func listExtractions(db *store.Store) ([]models.Extraction, error) {
	rows, err := db.DB().Query(`SELECT id, topic, created_at FROM extractions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var extractions []models.Extraction
	for rows.Next() {
		var e models.Extraction
		if err := rows.Scan(&e.ID, &e.Topic, &e.CreatedAt); err != nil {
			return nil, err
		}
		extractions = append(extractions, e)
	}

	return extractions, rows.Err()
}

// QuoteDetail holds a quote with full chunk and document data
type QuoteDetail struct {
	Quote    *models.ExtractedQuote
	Chunk    *models.Chunk
	Document *models.Document
}

func getQuotesWithDetails(db *store.Store, extractionID int64) ([]QuoteDetail, error) {
	rows, err := db.DB().Query(`
		SELECT 
			eq.id, eq.extraction_id, eq.chunk_id, eq.relevance_score, eq.explanation,
			c.id, c.document_id, c.text, c.page_num, c.section_path, c.bbox,
			d.id, d.filename, d.filepath, d.title, d.authors, d.year, d.publisher, d.source_type
		FROM extracted_quotes eq
		JOIN chunks c ON eq.chunk_id = c.id
		JOIN documents d ON c.document_id = d.id
		WHERE eq.extraction_id = ?
		ORDER BY eq.relevance_score DESC`, extractionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []QuoteDetail
	for rows.Next() {
		var qd QuoteDetail
		qd.Quote = &models.ExtractedQuote{}
		qd.Chunk = &models.Chunk{}
		qd.Document = &models.Document{}

		var sectionPathJSON, bboxJSON sql.NullString
		var title, publisher, authorsJSON sql.NullString
		var year sql.NullInt64

		err := rows.Scan(
			&qd.Quote.ID, &qd.Quote.ExtractionID, &qd.Quote.ChunkID, &qd.Quote.RelevanceScore, &qd.Quote.Explanation,
			&qd.Chunk.ID, &qd.Chunk.DocumentID, &qd.Chunk.Text, &qd.Chunk.PageNum, &sectionPathJSON, &bboxJSON,
			&qd.Document.ID, &qd.Document.Filename, &qd.Document.Filepath, &title, &authorsJSON, &year, &publisher, &qd.Document.SourceType,
		)
		if err != nil {
			return nil, err
		}

		if sectionPathJSON.Valid {
			qd.Chunk.UnmarshalSectionPath(sectionPathJSON.String)
		}
		if bboxJSON.Valid {
			qd.Chunk.UnmarshalBBox(bboxJSON.String)
		}
		if title.Valid {
			qd.Document.Title = &title.String
		}
		if publisher.Valid {
			qd.Document.Publisher = &publisher.String
		}
		if year.Valid {
			y := int(year.Int64)
			qd.Document.Year = &y
		}
		if authorsJSON.Valid {
			qd.Document.UnmarshalAuthors(authorsJSON.String)
		}

		results = append(results, qd)
	}

	return results, rows.Err()
}

func formatMarkdown(extraction *models.Extraction, quotes []QuoteDetail) string {
	var sb strings.Builder
	formatter := harvard.NewFormatter("harvard_us")

	// Header
	sb.WriteString(fmt.Sprintf("# Quotes: %s\n\n", extraction.Topic))
	sb.WriteString(fmt.Sprintf("*Extracted: %s*\n\n", extraction.CreatedAt.Format("January 2, 2006")))
	sb.WriteString("---\n\n")

	// Quotes
	for _, qd := range quotes {
		ref := buildReferenceFromQuoteDetail(&qd)
		inText := formatter.FormatInText(ref)

		sb.WriteString(fmt.Sprintf("> \"%s\"\n>\n", qd.Chunk.Text))
		sb.WriteString(fmt.Sprintf("> — %s\n>\n", inText))
		sb.WriteString(fmt.Sprintf("> **Relevance (%d/100):** %s\n\n", qd.Quote.RelevanceScore, qd.Quote.Explanation))
		sb.WriteString("---\n\n")
	}

	// Bibliography
	sb.WriteString("## Bibliography\n\n")
	seenDocs := make(map[int64]bool)
	for _, qd := range quotes {
		if seenDocs[qd.Document.ID] {
			continue
		}
		seenDocs[qd.Document.ID] = true

		ref := buildReferenceFromQuoteDetail(&qd)
		fullRef := formatter.FormatFull(ref)
		sb.WriteString(fullRef + "\n\n")
	}

	return sb.String()
}

// JSONOutput represents the JSON export format
type JSONOutput struct {
	Extraction struct {
		ID        int64     `json:"id"`
		Topic     string    `json:"topic"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"extraction"`
	Quotes []JSONQuote `json:"quotes"`
}

// JSONQuote represents a quote in JSON format
type JSONQuote struct {
	Text           string `json:"text"`
	RelevanceScore int    `json:"relevance_score"`
	Explanation    string `json:"explanation"`
	InText         string `json:"in_text"`
	FullReference  string `json:"full_reference"`
	PageNum        *int   `json:"page_num,omitempty"`
	Document       struct {
		ID       int64    `json:"id"`
		Filename string   `json:"filename"`
		Title    *string  `json:"title,omitempty"`
		Authors  []string `json:"authors,omitempty"`
		Year     *int     `json:"year,omitempty"`
	} `json:"document"`
}

func formatJSON(extraction *models.Extraction, quotes []QuoteDetail) (string, error) {
	formatter := harvard.NewFormatter("harvard_us")

	output := JSONOutput{}
	output.Extraction.ID = extraction.ID
	output.Extraction.Topic = extraction.Topic
	output.Extraction.CreatedAt = extraction.CreatedAt

	for _, qd := range quotes {
		ref := buildReferenceFromQuoteDetail(&qd)

		jq := JSONQuote{
			Text:           qd.Chunk.Text,
			RelevanceScore: qd.Quote.RelevanceScore,
			Explanation:    qd.Quote.Explanation,
			InText:         formatter.FormatInText(ref),
			FullReference:  formatter.FormatFull(ref),
			PageNum:        qd.Chunk.PageNum,
		}

		jq.Document.ID = qd.Document.ID
		jq.Document.Filename = qd.Document.Filename
		jq.Document.Title = qd.Document.Title
		jq.Document.Authors = qd.Document.Authors
		jq.Document.Year = qd.Document.Year

		output.Quotes = append(output.Quotes, jq)
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func formatBibTeX(extraction *models.Extraction, quotes []QuoteDetail) string {
	var sb strings.Builder

	// Comment header
	sb.WriteString(fmt.Sprintf("%% Bibliography for: %s\n", extraction.Topic))
	sb.WriteString(fmt.Sprintf("%% Extracted: %s\n\n", extraction.CreatedAt.Format("2006-01-02")))

	seenDocs := make(map[int64]bool)
	for _, qd := range quotes {
		if seenDocs[qd.Document.ID] {
			continue
		}
		seenDocs[qd.Document.ID] = true

		// Generate citation key
		key := generateCitationKey(qd.Document)

		// Determine entry type
		entryType := getBibTeXType(qd.Document.SourceType)

		sb.WriteString(fmt.Sprintf("@%s{%s,\n", entryType, key))

		// Author
		if len(qd.Document.Authors) > 0 {
			sb.WriteString(fmt.Sprintf("  author = {%s},\n", strings.Join(qd.Document.Authors, " and ")))
		}

		// Title
		if qd.Document.Title != nil {
			sb.WriteString(fmt.Sprintf("  title = {%s},\n", *qd.Document.Title))
		} else {
			sb.WriteString(fmt.Sprintf("  title = {%s},\n", qd.Document.Filename))
		}

		// Year
		if qd.Document.Year != nil {
			sb.WriteString(fmt.Sprintf("  year = {%d},\n", *qd.Document.Year))
		}

		// Publisher
		if qd.Document.Publisher != nil {
			sb.WriteString(fmt.Sprintf("  publisher = {%s},\n", *qd.Document.Publisher))
		}

		sb.WriteString("}\n\n")
	}

	return sb.String()
}

func buildReferenceFromQuoteDetail(qd *QuoteDetail) harvard.Reference {
	ref := harvard.Reference{
		SourceType: qd.Document.SourceType,
		PageNum:    qd.Chunk.PageNum,
	}

	if qd.Document.Title != nil {
		ref.Title = *qd.Document.Title
	} else {
		ref.Title = qd.Document.Filename
	}

	if qd.Document.Year != nil {
		ref.Year = *qd.Document.Year
	}

	if qd.Document.Publisher != nil {
		ref.Publisher = *qd.Document.Publisher
	}

	ref.Authors = harvard.ParseAuthorsString(qd.Document.Authors)

	return ref
}

func generateCitationKey(doc *models.Document) string {
	// Use first author's last name + year
	key := "unknown"
	if len(doc.Authors) > 0 {
		// Extract last name (before comma)
		parts := strings.Split(doc.Authors[0], ",")
		key = strings.ToLower(strings.TrimSpace(parts[0]))
		key = strings.ReplaceAll(key, " ", "")
	}

	if doc.Year != nil {
		key += fmt.Sprintf("%d", *doc.Year)
	}

	return key
}

func getBibTeXType(sourceType models.SourceType) string {
	switch sourceType {
	case models.SourceTypeBook:
		return "book"
	case models.SourceTypeJournalArticle:
		return "article"
	case models.SourceTypeWebsite:
		return "misc"
	case models.SourceTypeChapter:
		return "incollection"
	default:
		return "misc"
	}
}
