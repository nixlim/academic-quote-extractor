package models

// ExtractedQuote represents a single quote selected during extraction
type ExtractedQuote struct {
	ID             int64  `db:"id"`
	ExtractionID   int64  `db:"extraction_id"`
	ChunkID        string `db:"chunk_id"`
	RelevanceScore int    `db:"relevance_score"` // 0-100
	Explanation    string `db:"explanation"`
}
