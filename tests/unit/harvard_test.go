package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/foundry-zero/aqe/internal/harvard"
	"github.com/foundry-zero/aqe/internal/models"
)

func TestFormatAuthors(t *testing.T) {
	tests := []struct {
		name     string
		authors  []harvard.Author
		expected string
	}{
		{
			name:     "empty",
			authors:  nil,
			expected: "",
		},
		{
			name: "single author",
			authors: []harvard.Author{
				{LastName: "Smith", FirstName: "John"},
			},
			expected: "Smith, J.",
		},
		{
			name: "single author with initials",
			authors: []harvard.Author{
				{LastName: "Smith", Initials: "J.A."},
			},
			expected: "Smith, J.A.",
		},
		{
			name: "two authors",
			authors: []harvard.Author{
				{LastName: "Smith", FirstName: "John"},
				{LastName: "Jones", FirstName: "Mary"},
			},
			expected: "Smith, J. and Jones, M.",
		},
		{
			name: "three authors",
			authors: []harvard.Author{
				{LastName: "Smith", FirstName: "John"},
				{LastName: "Jones", FirstName: "Mary"},
				{LastName: "Williams", FirstName: "Kate"},
			},
			expected: "Smith, J., Jones, M. and Williams, K.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := harvard.FormatAuthors(tt.authors)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatAuthorsShort(t *testing.T) {
	tests := []struct {
		name     string
		authors  []harvard.Author
		expected string
	}{
		{
			name:     "empty",
			authors:  nil,
			expected: "",
		},
		{
			name: "single author",
			authors: []harvard.Author{
				{LastName: "Smith"},
			},
			expected: "Smith",
		},
		{
			name: "two authors",
			authors: []harvard.Author{
				{LastName: "Smith"},
				{LastName: "Jones"},
			},
			expected: "Smith and Jones",
		},
		{
			name: "three+ authors",
			authors: []harvard.Author{
				{LastName: "Smith"},
				{LastName: "Jones"},
				{LastName: "Williams"},
			},
			expected: "Smith et al.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := harvard.FormatAuthorsShort(tt.authors)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatInText(t *testing.T) {
	formatter := harvard.NewFormatter("harvard_us")
	page := 42

	tests := []struct {
		name     string
		ref      harvard.Reference
		expected string
	}{
		{
			name: "single author with page",
			ref: harvard.Reference{
				Authors: []harvard.Author{{LastName: "Smith"}},
				Year:    2023,
				PageNum: &page,
			},
			expected: "(Smith, 2023, p. 42)",
		},
		{
			name: "two authors without page",
			ref: harvard.Reference{
				Authors: []harvard.Author{{LastName: "Smith"}, {LastName: "Jones"}},
				Year:    2023,
			},
			expected: "(Smith and Jones, 2023)",
		},
		{
			name: "three+ authors",
			ref: harvard.Reference{
				Authors: []harvard.Author{{LastName: "Smith"}, {LastName: "Jones"}, {LastName: "Williams"}},
				Year:    2023,
			},
			expected: "(Smith et al., 2023)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatInText(tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatBook(t *testing.T) {
	formatter := harvard.NewFormatter("harvard_us")

	ref := harvard.Reference{
		Authors:    []harvard.Author{{LastName: "Smith", FirstName: "John"}},
		Year:       2023,
		Title:      "The Great Book",
		Place:      "New York",
		Publisher:  "Academic Press",
		SourceType: models.SourceTypeBook,
	}

	result := formatter.FormatFull(ref)
	assert.Contains(t, result, "Smith, J.")
	assert.Contains(t, result, "(2023)")
	assert.Contains(t, result, "*The Great Book*")
	assert.Contains(t, result, "New York: Academic Press")
}

func TestFormatBookWithDOI(t *testing.T) {
	formatter := harvard.NewFormatter("harvard_us")

	ref := harvard.Reference{
		Authors:    []harvard.Author{{LastName: "Smith", FirstName: "John"}},
		Year:       2023,
		Title:      "The Great Book",
		Publisher:  "Academic Press",
		DOI:        "10.1234/example",
		SourceType: models.SourceTypeBook,
	}

	result := formatter.FormatFull(ref)
	assert.Contains(t, result, "DOI: 10.1234/example")
}

func TestFormatJournalArticle(t *testing.T) {
	formatter := harvard.NewFormatter("harvard_us")

	ref := harvard.Reference{
		Authors:    []harvard.Author{{LastName: "Smith", FirstName: "John"}},
		Year:       2023,
		Title:      "An Important Study",
		Journal:    "Journal of Testing",
		Volume:     "45",
		Issue:      "2",
		Pages:      "100-120",
		SourceType: models.SourceTypeJournalArticle,
	}

	result := formatter.FormatFull(ref)
	assert.Contains(t, result, "Smith, J.")
	assert.Contains(t, result, "(2023)")
	assert.Contains(t, result, "\"An Important Study,\"")
	assert.Contains(t, result, "*Journal of Testing*")
	assert.Contains(t, result, "45(2)")
	assert.Contains(t, result, "pp. 100-120")
}

func TestFormatJournalArticleWithDOI(t *testing.T) {
	formatter := harvard.NewFormatter("harvard_us")

	ref := harvard.Reference{
		Authors:    []harvard.Author{{LastName: "Smith", FirstName: "John"}},
		Year:       2023,
		Title:      "An Important Study",
		Journal:    "Journal of Testing",
		DOI:        "10.1234/example",
		SourceType: models.SourceTypeJournalArticle,
	}

	result := formatter.FormatFull(ref)
	assert.Contains(t, result, "DOI: 10.1234/example")
}

func TestFormatWebsite(t *testing.T) {
	formatter := harvard.NewFormatter("harvard_us")

	ref := harvard.Reference{
		Authors:    []harvard.Author{{LastName: "Smith", FirstName: "John"}},
		Year:       2023,
		Title:      "Important Web Article",
		SiteName:   "Example Site",
		URL:        "https://example.com/article",
		AccessDate: "January 15, 2024",
		SourceType: models.SourceTypeWebsite,
	}

	result := formatter.FormatFull(ref)
	assert.Contains(t, result, "Smith, J.")
	assert.Contains(t, result, "(2023)")
	assert.Contains(t, result, "\"Important Web Article,\"")
	assert.Contains(t, result, "*Example Site*")
	assert.Contains(t, result, "Available at: https://example.com/article")
	assert.Contains(t, result, "(Accessed: January 15, 2024)")
}
