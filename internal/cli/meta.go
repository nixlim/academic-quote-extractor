package cli

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/foundry-zero/aqe/internal/models"
	"github.com/foundry-zero/aqe/internal/store"
)

// metaCmd represents the meta command group
var metaCmd = &cobra.Command{
	Use:   "meta",
	Short: "Manage document metadata",
	Long:  `Commands for managing document metadata.`,
}

// metaFixCmd represents the meta fix subcommand
var metaFixCmd = &cobra.Command{
	Use:   "fix",
	Short: "Interactively fix missing document metadata",
	Long: `Interactively prompt for missing author, title, and year metadata.

Examples:
  aqe meta fix`,
	RunE: runMetaFix,
}

func init() {
	rootCmd.AddCommand(metaCmd)
	metaCmd.AddCommand(metaFixCmd)
}

func runMetaFix(cmd *cobra.Command, args []string) error {
	db := GetStore()

	// Get documents with incomplete metadata
	docs, err := getDocumentsWithIncompleteMetadata(db)
	if err != nil {
		return fmt.Errorf("get incomplete documents: %w", err)
	}

	if len(docs) == 0 {
		fmt.Println("All documents have complete metadata.")
		return nil
	}

	reader := bufio.NewReader(os.Stdin)

	for _, doc := range docs {
		fmt.Printf("\nDocument: %s\n", doc.Filename)
		fmt.Println("Current metadata:")

		if doc.Title != nil {
			fmt.Printf("  Title: %s\n", *doc.Title)
		} else {
			fmt.Println("  Title: (missing)")
		}

		if len(doc.Authors) > 0 {
			fmt.Printf("  Author: %s\n", strings.Join(doc.Authors, ", "))
		} else {
			fmt.Println("  Author: (missing)")
		}

		if doc.Year != nil {
			fmt.Printf("  Year: %d\n", *doc.Year)
		} else {
			fmt.Println("  Year: (missing)")
		}

		fmt.Println()

		// Prompt for missing fields
		updated := false

		if doc.Title == nil {
			fmt.Print("Enter title (or press Enter to skip): ")
			title, _ := reader.ReadString('\n')
			title = strings.TrimSpace(title)
			if title != "" {
				doc.Title = &title
				updated = true
			}
		}

		if len(doc.Authors) == 0 {
			fmt.Print("Enter author (or press Enter to skip): ")
			author, _ := reader.ReadString('\n')
			author = strings.TrimSpace(author)
			if author != "" {
				doc.Authors = []string{author}
				updated = true
			}
		}

		if doc.Year == nil {
			fmt.Print("Enter year (or press Enter to skip): ")
			yearStr, _ := reader.ReadString('\n')
			yearStr = strings.TrimSpace(yearStr)
			if yearStr != "" {
				if year, err := strconv.Atoi(yearStr); err == nil {
					doc.Year = &year
					updated = true
				}
			}
		}

		if updated {
			if err := updateDocumentMetadata(db, doc); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to update %s: %v\n", doc.Filename, err)
			} else {
				fmt.Printf("\nUpdated: %s\n", doc.Filename)
				if doc.Title != nil {
					fmt.Printf("  Title: %s\n", *doc.Title)
				}
				if len(doc.Authors) > 0 {
					fmt.Printf("  Author: %s\n", strings.Join(doc.Authors, ", "))
				}
				if doc.Year != nil {
					fmt.Printf("  Year: %d\n", *doc.Year)
				}
			}
		}
	}

	// Check if all complete now
	remaining, _ := getDocumentsWithIncompleteMetadata(db)
	if len(remaining) == 0 {
		fmt.Println("\nAll documents now have complete metadata.")
	} else {
		fmt.Printf("\n%d documents still have incomplete metadata.\n", len(remaining))
	}

	return nil
}

func getDocumentsWithIncompleteMetadata(db *store.Store) ([]*models.Document, error) {
	rows, err := db.DB().Query(`
		SELECT id, filename, filepath, title, authors, year, publisher, source_type, checksum, ingested_at
		FROM documents
		WHERE title IS NULL OR authors IS NULL OR authors = '[]' OR year IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*models.Document
	for rows.Next() {
		doc := &models.Document{}
		var authorsJSON sql.NullString
		var title, publisher sql.NullString
		var year sql.NullInt64

		err := rows.Scan(
			&doc.ID, &doc.Filename, &doc.Filepath, &title, &authorsJSON,
			&year, &publisher, &doc.SourceType, &doc.Checksum, &doc.IngestedAt,
		)
		if err != nil {
			return nil, err
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

		docs = append(docs, doc)
	}

	return docs, rows.Err()
}

func updateDocumentMetadata(db *store.Store, doc *models.Document) error {
	authorsJSON, err := doc.MarshalAuthors()
	if err != nil {
		return fmt.Errorf("marshal authors: %w", err)
	}

	_, err = db.DB().Exec(`
		UPDATE documents 
		SET title = ?, authors = ?, year = ?
		WHERE id = ?`,
		doc.Title, authorsJSON, doc.Year, doc.ID,
	)
	return err
}
