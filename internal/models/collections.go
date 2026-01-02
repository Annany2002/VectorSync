package models

import "time"

// Collection represents a vector collection
type Collection struct {
	ID              string         `json:"id" db:"id"`
	Name            string         `json:"name" db:"name"`
	VectorDimension int            `json:"vector_dimension" db:"vector_dim"`
	MetadataSchema  map[string]any `json:"metadata_schema,omitempty" db:"metadata_schema"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
	DocumentCount   int64          `json:"document_count" db:"document_count"`
}
