package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/foundry-zero/aqe/internal/claude"
	"github.com/foundry-zero/aqe/internal/harvard"
	"github.com/foundry-zero/aqe/internal/models"
	"github.com/foundry-zero/aqe/internal/search"
	"github.com/foundry-zero/aqe/internal/store"
)

var (
	// Extract flags
	maxQuotes      int
	minRelevance   int
	candidateLimit int
)

// extractCmd represents the extract command
var extractCmd = &cobra.Command{
	Use:   "extract <topic>",
	Short: "Extract relevant quotes for a research topic",
	Long: `Extract relevant quotes from your ingested documents for a research topic.

The command will:
1. Search for semantically relevant passages using hybrid BM25+vector search
2. Score relevance using Claude LLM
3. Return quotes with Harvard-style in-text citations and explanations
4. Save the extraction for later export

Examples:
  aqe extract "impact of social media on political polarization"
  aqe extract "climate change policy effectiveness" --max-quotes 10 --min-relevance 80`,
	Args: cobra.ExactArgs(1),
	RunE: runExtract,
}

func init() {
	rootCmd.AddCommand(extractCmd)

	extractCmd.Flags().IntVar(&maxQuotes, "max-quotes", 20, "Maximum number of quotes to return")
	extractCmd.Flags().IntVar(&minRelevance, "min-relevance", 60, "Minimum relevance score (0-100)")
	extractCmd.Flags().IntVar(&candidateLimit, "candidates", 100, "Number of candidate chunks to retrieve for LLM scoring")
}

func runExtract(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	topic := args[0]

	db := GetStore()

	// Check if corpus exists
	chunkCount, err := db.GetChunkCount()
	if err != nil {
		return fmt.Errorf("check corpus: %w", err)
	}
	if chunkCount == 0 {
		fmt.Fprintln(os.Stderr, "Error: No documents in corpus")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Run 'aqe ingest <path>' first to add documents.")
		os.Exit(ExitUserError)
	}

	fmt.Println("Searching for relevant quotes...")

	// Initialize Weaviate client
	weaviateClient, err := search.NewWeaviateClient("localhost:8080", "http://ollama:11434")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Search service unavailable")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Could not connect to Weaviate at localhost:8080.")
		os.Exit(ExitSysError)
	}
	if IsDebug() {
		weaviateClient.SetDebug(true)
	}

	// Perform hybrid search - use configured candidate limit
	searchLimit := candidateLimit
	if maxQuotes > searchLimit/2 {
		searchLimit = maxQuotes * 2 // Ensure we get enough candidates
	}

	results, err := weaviateClient.HybridSearch(ctx, topic, searchLimit, 0.5)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "no such host") {
			fmt.Fprintln(os.Stderr, "Error: Search service unavailable")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "The Weaviate service is not responding at localhost:8080.")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "To fix:")
			fmt.Fprintln(os.Stderr, "  1. Check if Docker is running: docker ps")
			fmt.Fprintln(os.Stderr, "  2. Start services: docker-compose up -d")
			fmt.Fprintln(os.Stderr, "  3. Wait for Weaviate to be ready: curl http://localhost:8080/v1/.well-known/ready")
			os.Exit(ExitSysError)
		}
		return fmt.Errorf("search: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("\nNo relevant quotes found for this topic.")
		fmt.Println("Try:")
		fmt.Println("  - Using different keywords")
		fmt.Println("  - Ingesting more relevant documents")
		return nil
	}

	fmt.Printf("Found %d candidate chunks\n", len(results))
	fmt.Println("Scoring relevance with Claude...")

	// Build chunk inputs for Claude with adjacent context
	chunkInputs := make([]claude.ChunkInput, len(results))
	for i, r := range results {
		input := claude.ChunkInput{
			ID:          r.ChunkID,
			Text:        r.Text,
			PageNum:     r.PageNum,
			SectionPath: r.SectionPath,
			DocumentID:  r.DocumentID,
		}

		// Fetch adjacent chunks for context (if position data available)
		chunk, err := db.GetChunkByID(r.ChunkID)
		if err == nil && chunk != nil && chunk.Position != nil {
			adj, err := db.GetAdjacentChunks(chunk.ID, chunk.DocumentID, *chunk.Position)
			if err == nil && adj != nil {
				if adj.Prev != nil {
					input.PrevContext = truncateContext(adj.Prev.Text, 200)
				}
				if adj.Next != nil {
					input.NextContext = truncateContext(adj.Next.Text, 200)
				}
			}
		}

		chunkInputs[i] = input
	}

	// Call Claude for relevance scoring
	claudeWrapper, err := claude.NewWrapper(120 * time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Claude CLI not available")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "The claude CLI was not found in PATH.")
		fmt.Fprintln(os.Stderr, "Install it from: https://claude.ai/code")
		os.Exit(ExitSysError)
	}
	if IsDebug() {
		claudeWrapper.SetDebug(true)
	}

	task := claude.ExtractionTask{
		Topic:  topic,
		Chunks: chunkInputs,
	}

	response, err := claudeWrapper.ExtractQuotes(ctx, task)
	if err != nil {
		return fmt.Errorf("extract quotes: %w", err)
	}

	// Filter by relevance threshold and limit
	var selectedChunks []claude.SelectedChunk
	for _, sc := range response.SelectedChunks {
		if sc.Relevance >= minRelevance {
			selectedChunks = append(selectedChunks, sc)
		}
		if len(selectedChunks) >= maxQuotes {
			break
		}
	}

	if len(selectedChunks) == 0 {
		fmt.Printf("\nNo quotes met the minimum relevance threshold (%d).\n", minRelevance)
		fmt.Println("Try lowering --min-relevance or using different search terms.")
		return nil
	}

	// Save extraction
	extractionID, err := saveExtraction(db, topic, selectedChunks)
	if err != nil {
		return fmt.Errorf("save extraction: %w", err)
	}

	// Display results
	fmt.Printf("\nExtraction #%d: %q\n", extractionID, topic)
	fmt.Printf("Retrieved %d quotes (relevance >= %d)\n\n", len(selectedChunks), minRelevance)

	// Get document info for citations
	formatter := harvard.NewFormatter("harvard_us")

	for i, sc := range selectedChunks {
		// Get chunk details
		chunkDetails, err := getChunkWithDocument(db, sc.ChunkID)
		if err != nil {
			Debugf("Failed to get chunk details: %v", err)
			continue
		}

		fmt.Printf("Quote %d (Relevance: %d/100)\n", i+1, sc.Relevance)
		fmt.Println(strings.Repeat("━", 40))
		fmt.Printf("\"%s\"\n\n", chunkDetails.Text)

		// Format citation
		ref := buildReference(chunkDetails)
		inText := formatter.FormatInText(ref)
		fmt.Printf("— %s\n\n", inText)

		fmt.Printf("Why relevant: %s\n\n", sc.Explanation)
		fmt.Println(strings.Repeat("━", 40))
		fmt.Println()
	}

	fmt.Printf("Saved as extraction #%d. Export with: aqe export %d\n", extractionID, extractionID)

	return nil
}

// ChunkWithDocument holds chunk data along with its document metadata
type ChunkWithDocument struct {
	*models.Chunk
	Document *models.Document
}

func getChunkWithDocument(db *store.Store, chunkID string) (*ChunkWithDocument, error) {
	row := db.DB().QueryRow(`
		SELECT c.id, c.document_id, c.text, c.page_num, c.section_path, c.bbox,
		       d.id, d.filename, d.title, d.authors, d.year, d.publisher, d.source_type
		FROM chunks c
		JOIN documents d ON c.document_id = d.id
		WHERE c.id = ?`, chunkID)

	var chunk models.Chunk
	var doc models.Document
	var sectionPathJSON, bboxJSON sql.NullString
	var title, publisher, authorsJSON sql.NullString
	var year sql.NullInt64

	err := row.Scan(
		&chunk.ID, &chunk.DocumentID, &chunk.Text, &chunk.PageNum, &sectionPathJSON, &bboxJSON,
		&doc.ID, &doc.Filename, &title, &authorsJSON, &year, &publisher, &doc.SourceType,
	)
	if err != nil {
		return nil, fmt.Errorf("scan chunk: %w", err)
	}

	if sectionPathJSON.Valid {
		chunk.UnmarshalSectionPath(sectionPathJSON.String)
	}
	if bboxJSON.Valid {
		chunk.UnmarshalBBox(bboxJSON.String)
	}
	if title.Valid {
		doc.Title = &title.String
	}
	if publisher.Valid {
		doc.Publisher = &publisher.String
	}
	if year.Valid {
		y := int(year.Int64)
		doc.Year = &y
	}
	if authorsJSON.Valid {
		doc.UnmarshalAuthors(authorsJSON.String)
	}

	return &ChunkWithDocument{
		Chunk:    &chunk,
		Document: &doc,
	}, nil
}

func saveExtraction(db *store.Store, topic string, chunks []claude.SelectedChunk) (int64, error) {
	// Insert extraction
	result, err := db.DB().Exec(`INSERT INTO extractions (topic) VALUES (?)`, topic)
	if err != nil {
		return 0, fmt.Errorf("insert extraction: %w", err)
	}

	extractionID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get extraction id: %w", err)
	}

	// Insert quotes
	stmt, err := db.DB().Prepare(`
		INSERT INTO extracted_quotes (extraction_id, chunk_id, relevance_score, explanation)
		VALUES (?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("prepare quote insert: %w", err)
	}
	defer stmt.Close()

	for _, sc := range chunks {
		_, err := stmt.Exec(extractionID, sc.ChunkID, sc.Relevance, sc.Explanation)
		if err != nil {
			return 0, fmt.Errorf("insert quote: %w", err)
		}
	}

	return extractionID, nil
}

// truncateContext truncates text to approximately maxWords words for use as context
func truncateContext(text string, maxWords int) string {
	words := strings.Fields(text)
	if len(words) <= maxWords {
		return text
	}
	return strings.Join(words[:maxWords], " ") + "..."
}

func buildReference(cwd *ChunkWithDocument) harvard.Reference {
	ref := harvard.Reference{
		SourceType: cwd.Document.SourceType,
		PageNum:    cwd.Chunk.PageNum,
	}

	if cwd.Document.Title != nil {
		ref.Title = *cwd.Document.Title
	} else {
		ref.Title = cwd.Document.Filename
	}

	if cwd.Document.Year != nil {
		ref.Year = *cwd.Document.Year
	}

	if cwd.Document.Publisher != nil {
		ref.Publisher = *cwd.Document.Publisher
	}

	// Parse authors
	ref.Authors = harvard.ParseAuthorsString(cwd.Document.Authors)

	return ref
}
