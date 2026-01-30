package models

import (
	"encoding/json"
	"time"
)

// SourceType represents the type of academic source
type SourceType string

const (
	SourceTypeBook           SourceType = "book"
	SourceTypeJournalArticle SourceType = "journal_article"
	SourceTypeWebsite        SourceType = "website"
	SourceTypeChapter        SourceType = "chapter"
	SourceTypeUnknown        SourceType = "unknown"
)

// Document represents an ingested source file
type Document struct {
	ID         int64      `db:"id"`
	Filename   string     `db:"filename"`
	Filepath   string     `db:"filepath"`
	Title      *string    `db:"title"`
	Authors    []string   // Stored as JSON in authors column
	Year       *int       `db:"year"`
	Publisher  *string    `db:"publisher"`
	SourceType SourceType `db:"source_type"`
	Checksum   string     `db:"checksum"`
	IngestedAt time.Time  `db:"ingested_at"`
}

// MarshalAuthors converts the Authors slice to JSON for database storage
func (d *Document) MarshalAuthors() (string, error) {
	if d.Authors == nil {
		return "[]", nil
	}
	data, err := json.Marshal(d.Authors)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UnmarshalAuthors populates Authors from JSON stored in database
func (d *Document) UnmarshalAuthors(data string) error {
	if data == "" || data == "null" {
		d.Authors = nil
		return nil
	}
	return json.Unmarshal([]byte(data), &d.Authors)
}

// ValidSourceTypes returns all valid source type values
func ValidSourceTypes() []SourceType {
	return []SourceType{
		SourceTypeBook,
		SourceTypeJournalArticle,
		SourceTypeWebsite,
		SourceTypeChapter,
		SourceTypeUnknown,
	}
}

// IsValidSourceType checks if a source type is valid
func IsValidSourceType(st SourceType) bool {
	for _, valid := range ValidSourceTypes() {
		if st == valid {
			return true
		}
	}
	return false
}
