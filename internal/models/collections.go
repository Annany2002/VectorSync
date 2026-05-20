package models

import "time"

// Collection represents a vector collection
type Collection struct {
	Id              string         `json:"id" db:"id"`
	Name            string         `json:"name" db:"name"`
	VectorDimension int            `json:"vector_dimension" db:"vector_dim"`
	DistanceMetric  string         `json:"distance_metric" db:"distance_metric"`
	MetadataSchema  map[string]any `json:"metadata_schema,omitempty" db:"metadata_schema"`
	EmbeddingProvider string       `json:"embedding_provider,omitempty" db:"embedding_provider"`
	EmbeddingModel    string       `json:"embedding_model,omitempty" db:"embedding_model"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
	DocumentCount   int64          `json:"document_count" db:"document_count"`
}
