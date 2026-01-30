package cli

import (
	"database/sql"
	"fmt"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved extractions",
	Long: `List all saved quote extractions with their IDs and quote counts.

Examples:
  aqe list`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	db := GetStore()

	rows, err := db.DB().Query(`
		SELECT e.id, e.topic, e.created_at, COUNT(eq.id) as quote_count
		FROM extractions e
		LEFT JOIN extracted_quotes eq ON e.id = eq.extraction_id
		GROUP BY e.id
		ORDER BY e.created_at DESC`)
	if err != nil {
		return fmt.Errorf("query extractions: %w", err)
	}
	defer rows.Close()

	type ExtractionSummary struct {
		ID         int64
		Topic      string
		CreatedAt  sql.NullString
		QuoteCount int
	}

	var extractions []ExtractionSummary
	for rows.Next() {
		var e ExtractionSummary
		if err := rows.Scan(&e.ID, &e.Topic, &e.CreatedAt, &e.QuoteCount); err != nil {
			return fmt.Errorf("scan extraction: %w", err)
		}
		extractions = append(extractions, e)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate extractions: %w", err)
	}

	if len(extractions) == 0 {
		fmt.Println("No extractions found.")
		fmt.Println("Run 'aqe extract <topic>' to create your first extraction.")
		return nil
	}

	fmt.Println("Saved Extractions:")
	fmt.Println()

	for _, e := range extractions {
		date := "unknown date"
		if e.CreatedAt.Valid {
			date = e.CreatedAt.String[:10]
		}
		fmt.Printf("  #%d: %q\n", e.ID, e.Topic)
		fmt.Printf("      %d quotes | %s\n", e.QuoteCount, date)
		fmt.Printf("      Export: aqe export %d\n\n", e.ID)
	}

	return nil
}
