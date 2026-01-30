package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/foundry-zero/aqe/internal/chunker"
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
1. Parse document content using Docling
2. Split into semantic chunks
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

	// Initialize services
	doclingClient := docling.NewClient("http://localhost:5001")
	if IsDebug() {
		doclingClient.SetDebug(true)
	}

	// Check Docling health
	if err := doclingClient.Health(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "Error: Document parsing service unavailable")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "The Docling service is not responding at localhost:5001.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "To fix:")
		fmt.Fprintln(os.Stderr, "  1. Check if Docker is running: docker ps")
		fmt.Fprintln(os.Stderr, "  2. Start services: docker-compose up -d")
		fmt.Fprintln(os.Stderr, "  3. Wait for Docling to be ready: curl http://localhost:5001/health")
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
		return fmt.Errorf("create weaviate schema: %w", err)
	}

	// Initialize chunker
	scriptPath := filepath.Join(getScriptsDir(), "chunk_helper.py")
	chunkerInstance, err := chunker.NewChunker(scriptPath)
	if err != nil {
		Debugf("Chunker init failed: %v", err)
		// Continue without chunking for now - we'll use text items directly
		chunkerInstance = nil
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
		docs, chunks, err := processFile(ctx, file, db, doclingClient, weaviateClient, chunkerInstance)
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

func processFile(ctx context.Context, filePath string, db *store.Store, doclingClient *docling.Client, weaviateClient *search.WeaviateClient, chunkerInstance *chunker.Chunker) (int, int, error) {
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

	// Parse document with Docling
	doclingDoc, err := doclingClient.ConvertFile(ctx, filePath)
	if err != nil {
		return 0, 0, fmt.Errorf("parse document: %w", err)
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

	// Apply CLI overrides or use detected metadata
	if ingestTitle != "" {
		doc.Title = &ingestTitle
	} else if doclingDoc.Name != "" {
		doc.Title = &doclingDoc.Name
	}

	if ingestAuthor != "" {
		doc.Authors = []string{ingestAuthor}
	}

	if ingestYear > 0 {
		doc.Year = &ingestYear
	}

	// Log metadata
	if doc.Title != nil {
		if ingestTitle != "" || ingestAuthor != "" || ingestYear > 0 {
			fmt.Printf("  Using provided metadata: %q", *doc.Title)
		} else {
			fmt.Printf("  Detected: %q", *doc.Title)
		}
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

	// Generate chunks
	var chunks []*models.Chunk

	if chunkerInstance != nil {
		// Use Python chunker
		docJSON, err := json.Marshal(doclingDoc)
		if err != nil {
			return 0, 0, fmt.Errorf("marshal document: %w", err)
		}

		chunkOutputs, err := chunkerInstance.ChunkDocument(ctx, docJSON)
		if err != nil {
			Debugf("Chunker failed, falling back to text items: %v", err)
			// Fall back to text items
			chunks = textItemsToChunks(doclingDoc, docID)
		} else {
			chunks = chunkOutputsToChunks(chunkOutputs, docID)
		}
	} else {
		// Fall back to using text items directly
		chunks = textItemsToChunks(doclingDoc, docID)
	}

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

func textItemsToChunks(doc *docling.DoclingDocument, docID int64) []*models.Chunk {
	var chunks []*models.Chunk

	for i, item := range doc.Texts {
		if item.Text == "" {
			continue
		}

		chunk := &models.Chunk{
			ID:         fmt.Sprintf("#/texts/%d", i),
			DocumentID: docID,
			Text:       item.Text,
		}

		// Extract page number
		if len(item.Prov) > 0 {
			pn := item.Prov[0].PageNo
			chunk.PageNum = &pn

			// Extract bbox
			if item.Prov[0].BBox != nil {
				chunk.BBox = &models.BBox{
					L:           item.Prov[0].BBox.L,
					T:           item.Prov[0].BBox.T,
					R:           item.Prov[0].BBox.R,
					B:           item.Prov[0].BBox.B,
					CoordOrigin: item.Prov[0].BBox.CoordOrigin,
				}
			}
		}

		chunks = append(chunks, chunk)
	}

	return chunks
}

func chunkOutputsToChunks(outputs []chunker.ChunkOutput, docID int64) []*models.Chunk {
	chunks := make([]*models.Chunk, len(outputs))

	for i, out := range outputs {
		chunk := &models.Chunk{
			ID:          out.ID,
			DocumentID:  docID,
			Text:        out.Text,
			PageNum:     out.PageNum,
			SectionPath: out.SectionPath,
		}

		if out.BBox != nil {
			chunk.BBox = &models.BBox{
				L:           out.BBox.L,
				T:           out.BBox.T,
				R:           out.BBox.R,
				B:           out.BBox.B,
				CoordOrigin: out.BBox.CoordOrigin,
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
