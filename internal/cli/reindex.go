package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/foundry-zero/aqe/internal/search"
)

// reindexCmd represents the reindex command
var reindexCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Rebuild the Weaviate search index from SQLite",
	Long: `Rebuild the Weaviate vector search index from the chunks stored in SQLite.

This drops and recreates the Weaviate Chunk class, then re-inserts all chunks
with fresh embeddings. Use this when the Weaviate index is out of sync with
the SQLite database.`,
	RunE: runReindex,
}

func init() {
	rootCmd.AddCommand(reindexCmd)
}

func runReindex(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	db := GetStore()

	// Initialize Weaviate
	weaviateClient, err := search.NewWeaviateClient("localhost:8080", "http://ollama:11434")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Search service unavailable")
		os.Exit(ExitSysError)
	}
	if IsDebug() {
		weaviateClient.SetDebug(true)
	}

	// Get documents and chunk counts
	docs, err := db.ListDocumentsWithChunkCounts()
	if err != nil {
		return fmt.Errorf("list documents: %w", err)
	}

	totalChunks := 0
	for _, doc := range docs {
		totalChunks += doc.ChunkCount
	}

	if totalChunks == 0 {
		fmt.Println("No chunks to index.")
		return nil
	}

	fmt.Printf("Reindexing %d chunks from %d documents into Weaviate...\n\n", totalChunks, len(docs))

	// Drop existing Weaviate class
	fmt.Print("  Dropping existing index... ")
	err = weaviateClient.DeleteSchema(ctx)
	if err != nil {
		// Class might not exist, that's fine
		Debugf("Delete schema: %v", err)
	}
	fmt.Println("done")

	// Recreate schema
	fmt.Print("  Creating schema... ")
	if err := weaviateClient.CreateSchema(ctx); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	fmt.Println("done")

	// Re-insert chunks document by document
	var indexed, failed int
	for _, doc := range docs {
		if doc.ChunkCount == 0 {
			continue
		}

		title := doc.Filename
		if doc.Title != nil {
			title = *doc.Title
		}
		// Truncate long titles
		if len(title) > 60 {
			title = title[:57] + "..."
		}

		fmt.Printf("  [%d/%d] %s (%d chunks)... ", doc.ID, len(docs), title, doc.ChunkCount)

		chunks, err := db.GetChunksByDocumentID(doc.ID)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			failed += doc.ChunkCount
			continue
		}

		docFailed := 0
		for _, chunk := range chunks {
			_, err := weaviateClient.InsertChunk(ctx, chunk.ID, chunk.DocumentID, chunk.Text, chunk.PageNum, chunk.SectionPath)
			if err != nil {
				Debugf("Insert chunk %s: %v", chunk.ID, err)
				docFailed++
			}
		}

		docIndexed := len(chunks) - docFailed
		indexed += docIndexed
		failed += docFailed

		if docFailed > 0 {
			fmt.Printf("%d indexed, %d failed\n", docIndexed, docFailed)
		} else {
			fmt.Printf("%d indexed\n", docIndexed)
		}
	}

	fmt.Printf("\nReindex complete: %d indexed, %d failed (of %d total)\n", indexed, failed, totalChunks)

	// Verify count
	count, err := weaviateClient.GetObjectCount(ctx)
	if err != nil {
		Debugf("Count verification failed: %v", err)
	} else {
		fmt.Printf("Weaviate objects: %d\n", count)
	}

	return nil
}
