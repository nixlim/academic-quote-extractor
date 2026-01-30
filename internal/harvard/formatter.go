package harvard

import (
	"github.com/foundry-zero/aqe/internal/models"
)

// Reference contains all possible fields for citation formatting
type Reference struct {
	Authors    []Author
	Year       int
	Title      string
	Journal    string // For journal articles
	Volume     string // For journal articles
	Issue      string // For journal articles
	Pages      string // Page range (e.g., "42-56")
	Publisher  string // For books
	Place      string // Publisher location
	URL        string // For websites
	DOI        string // Digital Object Identifier
	AccessDate string // For websites (formatted: "January 15, 2024")
	Editor     string // For chapters in edited volumes
	BookTitle  string // For chapters in edited volumes
	SiteName   string // For websites
	SourceType models.SourceType

	// For in-text citations
	PageNum *int // Specific page being cited
}

// Formatter defines the interface for citation formatting
type Formatter interface {
	// FormatFull returns the full bibliography reference
	FormatFull(ref Reference) string

	// FormatInText returns the in-text citation (e.g., "(Smith, 2023, p. 42)")
	FormatInText(ref Reference) string

	// FormatReference is an alias for FormatFull
	FormatReference(ref Reference) string
}

// NewFormatter creates a formatter for the specified style
// Currently only "harvard_us" is supported
func NewFormatter(style string) Formatter {
	switch style {
	case "harvard_us", "":
		return &HarvardUS{}
	default:
		// Default to Harvard US
		return &HarvardUS{}
	}
}
