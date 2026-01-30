package harvard

import (
	"fmt"
	"strings"
)

// Author represents a single author
type Author struct {
	LastName  string
	FirstName string
	Initials  string // Optional: if provided, used instead of FirstName
}

// String returns the author in "LastName, F." format
func (a Author) String() string {
	if a.Initials != "" {
		return fmt.Sprintf("%s, %s", a.LastName, a.Initials)
	}
	if a.FirstName != "" {
		// Extract initials from first name
		parts := strings.Fields(a.FirstName)
		initials := ""
		for _, p := range parts {
			if len(p) > 0 {
				initials += string(p[0]) + "."
			}
		}
		return fmt.Sprintf("%s, %s", a.LastName, initials)
	}
	return a.LastName
}

// FormatAuthors formats a list of authors according to Harvard style
// - 1 author: "Smith, J."
// - 2 authors: "Smith, J. and Jones, M."
// - 3+ authors: "Smith, J., Jones, M. and Williams, K."
func FormatAuthors(authors []Author) string {
	if len(authors) == 0 {
		return ""
	}
	if len(authors) == 1 {
		return authors[0].String()
	}
	if len(authors) == 2 {
		return fmt.Sprintf("%s and %s", authors[0].String(), authors[1].String())
	}

	// 3+ authors
	parts := make([]string, len(authors)-1)
	for i := 0; i < len(authors)-1; i++ {
		parts[i] = authors[i].String()
	}
	return fmt.Sprintf("%s and %s", strings.Join(parts, ", "), authors[len(authors)-1].String())
}

// FormatAuthorsShort formats authors for in-text citations
// - 1 author: "Smith"
// - 2 authors: "Smith and Jones"
// - 3+ authors: "Smith et al."
func FormatAuthorsShort(authors []Author) string {
	if len(authors) == 0 {
		return ""
	}
	if len(authors) == 1 {
		return authors[0].LastName
	}
	if len(authors) == 2 {
		return fmt.Sprintf("%s and %s", authors[0].LastName, authors[1].LastName)
	}
	return fmt.Sprintf("%s et al.", authors[0].LastName)
}

// ParseAuthorString parses an author string like "Smith, J." into an Author
func ParseAuthorString(s string) Author {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ",")
	if len(parts) < 2 {
		return Author{LastName: s}
	}
	return Author{
		LastName: strings.TrimSpace(parts[0]),
		Initials: strings.TrimSpace(parts[1]),
	}
}

// ParseAuthorsString parses a JSON-style list of authors
func ParseAuthorsString(authors []string) []Author {
	result := make([]Author, len(authors))
	for i, s := range authors {
		result[i] = ParseAuthorString(s)
	}
	return result
}
