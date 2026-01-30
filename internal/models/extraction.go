package models

import "time"

// Extraction represents a quote extraction session
type Extraction struct {
	ID         int64     `db:"id"`
	Topic      string    `db:"topic"`
	CreatedAt  time.Time `db:"created_at"`
	QuoteCount int       `db:"-"` // Not stored in DB, computed from join
}
