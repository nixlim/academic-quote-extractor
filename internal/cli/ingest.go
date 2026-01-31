package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/foundry-zero/aqe/internal/docling"
	"github.com/foundry-zero/aqe/internal/models"
	"github.com/foundry-zero/aqe/internal/search"
	"github.com/foundry-zero/aqe/internal/store"
)

var (
	// Ingest flags
	ingestTitle  string
	ingestAuthor string
	ingestYear   int
)

// ingestCmd represents the ingest command
var ingestCmd = &cobra.Command{
	Use:   "ingest <path>",
	Short: "Ingest documents for quote extraction",
	Long: `Ingest academic documents (PDF, DOCX, TXT) for quote extraction.

The command will:
1. Parse document content using local Docling
2. Split into semantic chunks with HybridChunker
3. Generate embeddings via Ollama
4. Store in SQLite and Weaviate for retrieval

Examples:
  aqe ingest ./sources/              # Ingest all supported files in directory
  aqe ingest paper.pdf               # Ingest a single file
  aqe ingest doc.pdf --title "My Paper" --author "Smith, J." --year 2023`,
	Args: cobra.ExactArgs(1),
	RunE: runIngest,
}

func init() {
	rootCmd.AddCommand(ingestCmd)

	ingestCmd.Flags().StringVar(&ingestTitle, "title", "", "Document title (overrides auto-detection)")
	ingestCmd.Flags().StringVar(&ingestAuthor, "author", "", "Document author (overrides auto-detection)")
	ingestCmd.Flags().IntVar(&ingestYear, "year", 0, "Publication year (overrides auto-detection)")
}

func runIngest(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	path := args[0]

	// Get list of files to process
	files, err := getFilesToIngest(path)
	if err != nil {
		return fmt.Errorf("get files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No supported files found to ingest.")
		return nil
	}

	// Initialize local Docling processor
	scriptPath := filepath.Join(getScriptsDir(), "process_document.py")
	processorOpts := []docling.ProcessorOption{
		docling.WithMaxTokens(800),
		docling.WithOverlap(200),
	}
	if IsDebug() {
		processorOpts = append(processorOpts, docling.WithProcessorDebug(true))
	}

	processor, err := docling.NewProcessor(scriptPath, processorOpts...)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Document processing pipeline unavailable")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Could not initialize the local Docling processor.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "To fix:")
		fmt.Fprintln(os.Stderr, "  1. Ensure Python 3 is installed: python3 --version")
		fmt.Fprintln(os.Stderr, "  2. Install dependencies: pip install -r scripts/requirements.txt")
		fmt.Fprintf(os.Stderr, "\nDetails: %v\n", err)
		os.Exit(ExitSysError)
	}

	// Check Python dependencies
	if err := processor.CheckDependencies(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "Error: Missing Python dependencies")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Required Python packages are not installed.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "To fix:")
		fmt.Fprintln(os.Stderr, "  pip install -r scripts/requirements.txt")
		fmt.Fprintf(os.Stderr, "\nDetails: %v\n", err)
		os.Exit(ExitSysError)
	}

	// Initialize Weaviate
	weaviateClient, err := search.NewWeaviateClient("localhost:8080", "http://ollama:11434")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Search service unavailable")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Could not connect to Weaviate at localhost:8080.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "To fix:")
		fmt.Fprintln(os.Stderr, "  1. Start services: docker-compose up -d")
		fmt.Fprintln(os.Stderr, "  2. Wait for Weaviate to be ready: curl http://localhost:8080/v1/.well-known/ready")
		os.Exit(ExitSysError)
	}
	if IsDebug() {
		weaviateClient.SetDebug(true)
	}

	// Create Weaviate schema if needed
	if err := weaviateClient.CreateSchema(ctx); err != nil {
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
		return fmt.Errorf("create weaviate schema: %w", err)
	}

	// Check for already ingested files (batch resume)
	db := GetStore()
	alreadyIngested := 0
	for _, file := range files {
		checksum, err := store.CalculateChecksum(file)
		if err != nil {
			continue
		}
		existing, _ := db.GetDocumentByChecksum(checksum)
		if existing != nil {
			alreadyIngested++
		}
	}

	if alreadyIngested > 0 && alreadyIngested < len(files) {
		fmt.Printf("Resuming batch ingestion, %d of %d files already processed\n", alreadyIngested, len(files))
	}

	// Process each file
	var totalDocs, totalChunks int
	for _, file := range files {
		docs, chunks, err := processFile(ctx, file, db, processor, weaviateClient)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %s - %v\n", filepath.Base(file), err)
			continue
		}
		totalDocs += docs
		totalChunks += chunks
	}

	// Summary
	fmt.Printf("\nIngested %d documents, %d chunks\n", totalDocs, totalChunks)

	return nil
}

func processFile(ctx context.Context, filePath string, db *store.Store, processor *docling.Processor, weaviateClient *search.WeaviateClient) (int, int, error) {
	filename := filepath.Base(filePath)
	fmt.Printf("Processing: %s\n", filename)

	// Calculate checksum for duplicate detection
	checksum, err := store.CalculateChecksum(filePath)
	if err != nil {
		return 0, 0, fmt.Errorf("calculate checksum: %w", err)
	}

	// Check for duplicates
	existing, err := db.GetDocumentByChecksum(checksum)
	if err != nil {
		return 0, 0, fmt.Errorf("check duplicate: %w", err)
	}
	if existing != nil {
		fmt.Printf("  Skipping: already ingested\n")
		return 0, 0, nil
	}

	// Process document with local Docling pipeline (parse + chunk + overlap)
	fmt.Printf("  Parsing and chunking...\n")
	chunkResults, err := processor.ProcessFile(ctx, filePath)
	if err != nil {
		return 0, 0, fmt.Errorf("process document: %w", err)
	}

	// Extract metadata
	absPath, _ := filepath.Abs(filePath)
	doc := &models.Document{
		Filename:   filename,
		Filepath:   absPath,
		SourceType: detectSourceType(filename),
		Checksum:   checksum,
		IngestedAt: time.Now(),
	}

	// Apply CLI overrides
	if ingestTitle != "" {
		doc.Title = &ingestTitle
	}
	if ingestAuthor != "" {
		doc.Authors = []string{ingestAuthor}
	}
	if ingestYear > 0 {
		doc.Year = &ingestYear
	}

	// Log metadata
	if doc.Title != nil {
		fmt.Printf("  Title: %q", *doc.Title)
		if len(doc.Authors) > 0 {
			fmt.Printf(" by %s", strings.Join(doc.Authors, ", "))
		}
		if doc.Year != nil {
			fmt.Printf(" (%d)", *doc.Year)
		}
		fmt.Println()
	}

	// Insert document
	docID, err := db.InsertDocument(doc)
	if err != nil {
		return 0, 0, fmt.Errorf("insert document: %w", err)
	}
	doc.ID = docID

	// Convert ChunkResults to model Chunks with position
	chunks := chunkResultsToChunks(chunkResults, docID)

	if len(chunks) == 0 {
		fmt.Printf("  Warning: No chunks extracted\n")
		return 1, 0, nil
	}

	// Insert chunks into Weaviate and get embedding IDs
	for _, chunk := range chunks {
		embeddingID, err := weaviateClient.InsertChunk(ctx, chunk.ID, chunk.DocumentID, chunk.Text, chunk.PageNum, chunk.SectionPath)
		if err != nil {
			Debugf("Weaviate insert failed for chunk %s: %v", chunk.ID, err)
			// Continue without embedding ID
		} else {
			chunk.EmbeddingID = &embeddingID
		}
	}

	// Insert chunks into SQLite
	if err := db.InsertChunks(chunks); err != nil {
		return 0, 0, fmt.Errorf("insert chunks: %w", err)
	}

	fmt.Printf("  Chunks: %d\n", len(chunks))

	return 1, len(chunks), nil
}

func getFilesToIngest(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat path: %w", err)
	}

	supportedExts := map[string]bool{
		".pdf":  true,
		".docx": true,
		".txt":  true,
		".md":   true,
	}

	if !info.IsDir() {
		ext := strings.ToLower(filepath.Ext(path))
		if !supportedExts[ext] {
			fmt.Fprintf(os.Stderr, "Warning: Unsupported file format: %s\n", ext)
			return nil, nil
		}
		return []string{path}, nil
	}

	var files []string
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if supportedExts[ext] {
			files = append(files, filepath.Join(path, entry.Name()))
		}
	}

	return files, nil
}

func detectSourceType(filename string) models.SourceType {
	lower := strings.ToLower(filename)
	switch {
	case strings.Contains(lower, "article") || strings.Contains(lower, "journal"):
		return models.SourceTypeJournalArticle
	case strings.Contains(lower, "chapter"):
		return models.SourceTypeChapter
	case strings.Contains(lower, "web") || strings.Contains(lower, "online"):
		return models.SourceTypeWebsite
	default:
		return models.SourceTypeUnknown
	}
}

// chunkResultsToChunks converts Processor output to model Chunks with position set
func chunkResultsToChunks(results []docling.ChunkResult, docID int64) []*models.Chunk {
	chunks := make([]*models.Chunk, len(results))

	for i, r := range results {
		pos := i
		chunk := &models.Chunk{
			ID:          fmt.Sprintf("doc%d:%s", docID, r.ID),
			DocumentID:  docID,
			Text:        r.Text,
			PageNum:     r.PageNum,
			SectionPath: r.SectionPath,
			Position:    &pos,
		}

		if r.BBox != nil {
			chunk.BBox = &models.BBox{
				L:           r.BBox.L,
				T:           r.BBox.T,
				R:           r.BBox.R,
				B:           r.BBox.B,
				CoordOrigin: r.BBox.CoordOrigin,
			}
		}

		chunks[i] = chunk
	}

	return chunks
}

func getScriptsDir() string {
	// Try relative to current working directory
	if _, err := os.Stat("scripts"); err == nil {
		return "scripts"
	}
	// Try relative to executable
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(dir, "scripts")); err == nil {
			return filepath.Join(dir, "scripts")
		}
	}
	return "scripts"
}
