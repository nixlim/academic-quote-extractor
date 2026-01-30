package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/foundry-zero/aqe/internal/store"
)

var (
	// Global flags
	dbPath string
	debug  bool

	// Shared resources
	db *store.Store

	// Exit codes
	ExitSuccess   = 0
	ExitUserError = 1
	ExitSysError  = 2
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "aqe",
	Short: "Academic Quote Extractor",
	Long: `aqe (Academic Quote Extractor) is a CLI tool for extracting relevant quotes 
from academic documents with Harvard-style citations.

It uses hybrid RAG (Retrieval-Augmented Generation) to:
1. Parse and chunk academic documents (PDF, DOCX, TXT)
2. Search for semantically relevant passages
3. Score relevance with Claude LLM
4. Generate properly formatted Harvard citations`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip database initialization for help commands
		if cmd.Name() == "help" || cmd.Name() == "version" {
			return nil
		}

		// Initialize database
		var err error
		db, err = store.NewStore(dbPath)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}

		// Run migrations
		if err := db.RunMigrations(); err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}

		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if db != nil {
			db.Close()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(ExitUserError)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "./quotes.db", "Path to SQLite database")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug output")
}

// GetStore returns the database store (for use by subcommands)
func GetStore() *store.Store {
	return db
}

// IsDebug returns whether debug mode is enabled
func IsDebug() bool {
	return debug
}

// UserError prints an error message and exits with user error code
func UserError(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", a...)
	os.Exit(ExitUserError)
}

// SystemError prints an error message and exits with system error code
func SystemError(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", a...)
	os.Exit(ExitSysError)
}

// Debugf prints debug messages when debug mode is enabled
func Debugf(format string, a ...interface{}) {
	if debug {
		fmt.Printf("[DEBUG] "+format+"\n", a...)
	}
}
