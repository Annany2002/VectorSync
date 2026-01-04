package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Annany2002/vector-sync/internal/models"
)

type DocumentRepo struct {
	db *sql.DB
}

func NewDocumentRepo(db *sql.DB) *DocumentRepo {
	return &DocumentRepo{db: db}
}

// Create inserts a new document into the database
func (r *DocumentRepo) Create(ctx context.Context, collectionId, content string, vector []float32, metadata map[string]any) (*models.Document, error) {
	// Convert vector to PostgreSQL format: '[0.1,0.2,0.3]'
	vectorStr := vectorToString(vector)

	// Convert metadata map to JSONB
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	insertQuery := `
		INSERT INTO documents (collection_id, vector, metadata, content)
		VALUES ($1, $2::vector, $3, $4)
		RETURNING id, collection_id, vector, metadata, content, created_at, updated_at
	`

	var document models.Document
	var vectorStrReturned string
	var metadataBytes []byte

	err = r.db.QueryRowContext(ctx, insertQuery, collectionId, vectorStr, metadataJSON, content).Scan(
		&document.Id,
		&document.CollectionId,
		&vectorStrReturned,
		&metadataBytes,
		&document.Content,
		&document.CreatedAt,
		&document.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert document: %w", err)
	}

	// Parse vector string back to []float32
	document.Vector, err = stringToVector(vectorStrReturned)
	if err != nil {
		return nil, fmt.Errorf("failed to parse returned vector: %w", err)
	}

	// Parse metadata JSON back to map
	if len(metadataBytes) > 0 {
		err = json.Unmarshal(metadataBytes, &document.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &document, nil
}

// vectorToString converts []float32 to PostgreSQL vector format: '[1.0,2.0,3.0]'
func vectorToString(vec []float32) string {
	strVals := make([]string, len(vec))
	for i, v := range vec {
		strVals[i] = fmt.Sprintf("%f", v)
	}
	return "[" + strings.Join(strVals, ",") + "]"
}

// stringToVector parses PostgreSQL vector string '[1.0,2.0,3.0]' to []float32
func stringToVector(s string) ([]float32, error) {
	// Remove brackets
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")

	if s == "" {
		return []float32{}, nil
	}

	// Split by comma
	parts := strings.Split(s, ",")
	vec := make([]float32, len(parts))

	for i, part := range parts {
		var val float32
		_, err := fmt.Sscanf(strings.TrimSpace(part), "%f", &val)
		if err != nil {
			return nil, fmt.Errorf("failed to parse vector value at index %d: %w", i, err)
		}
		vec[i] = val
	}

	return vec, nil
}

// List returns documents from a specific collection with pagination support
func (r *DocumentRepo) List(ctx context.Context, collectionId string, limit, offset int) ([]models.Document, error) {
	selectQuery := `
		SELECT id, collection_id, vector, metadata, content, created_at, updated_at
		FROM documents
		WHERE collection_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, selectQuery, collectionId, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var documents []models.Document

	// Scan all rows one by one
	for rows.Next() {
		var document models.Document
		var vectorStrReturned string
		var metadataBytes []byte

		err = rows.Scan(
			&document.Id,
			&document.CollectionId,
			&vectorStrReturned,
			&metadataBytes,
			&document.Content,
			&document.CreatedAt,
			&document.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Parse vector string back to []float32
		document.Vector, err = stringToVector(vectorStrReturned)
		if err != nil {
			return nil, err
		}

		// Parse metadata JSON back to map
		if len(metadataBytes) > 0 {
			err = json.Unmarshal(metadataBytes, &document.Metadata)
			if err != nil {
				return nil, err
			}
		}

		documents = append(documents, document)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return documents, nil
}
