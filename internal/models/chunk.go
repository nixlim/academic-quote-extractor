package models

import (
	"encoding/json"
)

// BBox represents a bounding box in the document
type BBox struct {
	L           float64 `json:"l"`
	T           float64 `json:"t"`
	R           float64 `json:"r"`
	B           float64 `json:"b"`
	CoordOrigin string  `json:"coord_origin,omitempty"`
}

// Chunk represents a segment of text from a document.
// The Text field is the source of truth for verbatim quotes.
type Chunk struct {
	ID          string   `db:"id"` // Docling chunk ref e.g., "doc1:#/chunks/0"
	DocumentID  int64    `db:"document_id"`
	Text        string   `db:"text"` // Verbatim text - SOURCE OF TRUTH
	PageNum     *int     `db:"page_num"`
	SectionPath []string // Stored as JSON in section_path column
	BBox        *BBox    // Stored as JSON in bbox column
	EmbeddingID *string  `db:"embedding_id"` // Weaviate UUID
	Position    *int     `db:"position"`     // Sequential order within document (0-based)
}

// MarshalSectionPath converts SectionPath to JSON for database storage
func (c *Chunk) MarshalSectionPath() (string, error) {
	if c.SectionPath == nil {
		return "[]", nil
	}
	data, err := json.Marshal(c.SectionPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UnmarshalSectionPath populates SectionPath from JSON stored in database
func (c *Chunk) UnmarshalSectionPath(data string) error {
	if data == "" || data == "null" {
		c.SectionPath = nil
		return nil
	}
	return json.Unmarshal([]byte(data), &c.SectionPath)
}

// MarshalBBox converts BBox to JSON for database storage
func (c *Chunk) MarshalBBox() (string, error) {
	if c.BBox == nil {
		return "", nil
	}
	data, err := json.Marshal(c.BBox)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UnmarshalBBox populates BBox from JSON stored in database
func (c *Chunk) UnmarshalBBox(data string) error {
	if data == "" || data == "null" {
		c.BBox = nil
		return nil
	}
	return json.Unmarshal([]byte(data), &c.BBox)
}
