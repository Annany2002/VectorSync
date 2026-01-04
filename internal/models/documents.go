package models

import "time"

// Document represents a document in a collection
type Document struct {
	Id           string         `json:"id" db:"id"`
	CollectionId string         `json:"collection_id" db:"collection_id"`
	Vector       []float32      `json:"vector" db:"vector"`
	Metadata     map[string]any `json:"metadata,omitempty" db:"metadata"`
	Content      string         `json:"content,omitempty" db:"content"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
}
