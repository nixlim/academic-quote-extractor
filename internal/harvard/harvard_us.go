package harvard

import (
	"fmt"
	"strings"

	"github.com/foundry-zero/aqe/internal/models"
)

// HarvardUS implements Harvard US citation style
// - Dates: "January 15, 2024" (month day, year)
// - Quotes: Double quotes for article titles
// - Italics: Asterisks for book/journal titles (markdown style)
type HarvardUS struct{}

// FormatFull returns the full bibliography reference
func (h *HarvardUS) FormatFull(ref Reference) string {
	switch ref.SourceType {
	case models.SourceTypeBook:
		return h.FormatBook(ref)
	case models.SourceTypeJournalArticle:
		return h.FormatJournalArticle(ref)
	case models.SourceTypeWebsite:
		return h.FormatWebsite(ref)
	case models.SourceTypeChapter:
		return h.FormatChapter(ref)
	default:
		return h.FormatBook(ref) // Default to book format
	}
}

// FormatReference is an alias for FormatFull
func (h *HarvardUS) FormatReference(ref Reference) string {
	return h.FormatFull(ref)
}

// FormatBook formats a book citation
// Format: Author, A. (Year) *Title*. Place: Publisher.
// With DOI: Author, A. (Year) *Title*. Place: Publisher. DOI: xxx
func (h *HarvardUS) FormatBook(ref Reference) string {
	var sb strings.Builder

	// Authors
	authors := FormatAuthors(ref.Authors)
	if authors != "" {
		sb.WriteString(authors)
		sb.WriteString(" ")
	}

	// Year
	sb.WriteString(fmt.Sprintf("(%d) ", ref.Year))

	// Title (italicized)
	sb.WriteString(fmt.Sprintf("*%s*", ref.Title))

	// Place and Publisher
	if ref.Place != "" && ref.Publisher != "" {
		sb.WriteString(fmt.Sprintf(". %s: %s", ref.Place, ref.Publisher))
	} else if ref.Publisher != "" {
		sb.WriteString(fmt.Sprintf(". %s", ref.Publisher))
	}

	// DOI
	if ref.DOI != "" {
		sb.WriteString(fmt.Sprintf(". DOI: %s", ref.DOI))
	}

	sb.WriteString(".")
	return sb.String()
}

// FormatJournalArticle formats a journal article citation
// Format: Author, A. (Year) "Title," *Journal*, Vol(Issue), pp. X-Y.
// With DOI: Author, A. (Year) "Title," *Journal*, Vol(Issue), pp. X-Y. DOI: xxx
func (h *HarvardUS) FormatJournalArticle(ref Reference) string {
	var sb strings.Builder

	// Authors
	authors := FormatAuthors(ref.Authors)
	if authors != "" {
		sb.WriteString(authors)
		sb.WriteString(" ")
	}

	// Year
	sb.WriteString(fmt.Sprintf("(%d) ", ref.Year))

	// Title (in quotes)
	sb.WriteString(fmt.Sprintf("\"%s,\" ", ref.Title))

	// Journal (italicized)
	sb.WriteString(fmt.Sprintf("*%s*", ref.Journal))

	// Volume and Issue
	if ref.Volume != "" {
		sb.WriteString(fmt.Sprintf(", %s", ref.Volume))
		if ref.Issue != "" {
			sb.WriteString(fmt.Sprintf("(%s)", ref.Issue))
		}
	}

	// Pages
	if ref.Pages != "" {
		sb.WriteString(fmt.Sprintf(", pp. %s", ref.Pages))
	}

	// DOI
	if ref.DOI != "" {
		sb.WriteString(fmt.Sprintf(". DOI: %s", ref.DOI))
	}

	sb.WriteString(".")
	return sb.String()
}

// FormatWebsite formats a website citation
// Format: Author, A. (Year) "Title," *Site Name*. Available at: URL (Accessed: Date).
func (h *HarvardUS) FormatWebsite(ref Reference) string {
	var sb strings.Builder

	// Authors
	authors := FormatAuthors(ref.Authors)
	if authors != "" {
		sb.WriteString(authors)
		sb.WriteString(" ")
	}

	// Year
	sb.WriteString(fmt.Sprintf("(%d) ", ref.Year))

	// Title (in quotes)
	sb.WriteString(fmt.Sprintf("\"%s,\" ", ref.Title))

	// Site name (italicized)
	if ref.SiteName != "" {
		sb.WriteString(fmt.Sprintf("*%s*", ref.SiteName))
	}

	// URL
	if ref.URL != "" {
		sb.WriteString(fmt.Sprintf(". Available at: %s", ref.URL))
	}

	// Access date
	if ref.AccessDate != "" {
		sb.WriteString(fmt.Sprintf(" (Accessed: %s)", ref.AccessDate))
	}

	sb.WriteString(".")
	return sb.String()
}

// FormatChapter formats a book chapter citation
// Format: Author, A. (Year) "Chapter," in Editor (ed.) *Book*. Place: Publisher, pp. X-Y.
// With DOI: Author, A. (Year) "Chapter," in Editor (ed.) *Book*. Place: Publisher, pp. X-Y. DOI: xxx
func (h *HarvardUS) FormatChapter(ref Reference) string {
	var sb strings.Builder

	// Authors
	authors := FormatAuthors(ref.Authors)
	if authors != "" {
		sb.WriteString(authors)
		sb.WriteString(" ")
	}

	// Year
	sb.WriteString(fmt.Sprintf("(%d) ", ref.Year))

	// Chapter title (in quotes)
	sb.WriteString(fmt.Sprintf("\"%s,\" ", ref.Title))

	// Editor
	if ref.Editor != "" {
		sb.WriteString(fmt.Sprintf("in %s (ed.) ", ref.Editor))
	}

	// Book title (italicized)
	if ref.BookTitle != "" {
		sb.WriteString(fmt.Sprintf("*%s*", ref.BookTitle))
	}

	// Place and Publisher
	if ref.Place != "" && ref.Publisher != "" {
		sb.WriteString(fmt.Sprintf(". %s: %s", ref.Place, ref.Publisher))
	} else if ref.Publisher != "" {
		sb.WriteString(fmt.Sprintf(". %s", ref.Publisher))
	}

	// Pages
	if ref.Pages != "" {
		sb.WriteString(fmt.Sprintf(", pp. %s", ref.Pages))
	}

	// DOI
	if ref.DOI != "" {
		sb.WriteString(fmt.Sprintf(". DOI: %s", ref.DOI))
	}

	sb.WriteString(".")
	return sb.String()
}

// FormatInText returns the in-text citation
// Format: (Author, Year, p. X) or (Author, Year) if no page
func (h *HarvardUS) FormatInText(ref Reference) string {
	authors := FormatAuthorsShort(ref.Authors)

	if ref.PageNum != nil {
		return fmt.Sprintf("(%s, %d, p. %d)", authors, ref.Year, *ref.PageNum)
	}
	return fmt.Sprintf("(%s, %d)", authors, ref.Year)
}
