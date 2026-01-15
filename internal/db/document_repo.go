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

// UpsertResult contains the upserted document and whether it was newly created
type UpsertResult struct {
	Document *models.Document
	IsNew    bool
}

// Upsert inserts a new document or updates an existing one
// Uses PostgreSQL ON CONFLICT for atomic upsert operation
func (r *DocumentRepo) Upsert(ctx context.Context, documentId, collectionId, content string, vector []float32, metadata map[string]any) (*UpsertResult, error) {
	// Convert vector to PostgreSQL format: '[0.1,0.2,0.3]'
	vectorStr := vectorToString(vector)

	// Convert metadata map to JSONB
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Use ON CONFLICT to insert or update atomically
	// xmax = 0 means INSERT, non-zero means UPDATE
	upsertQuery := `
		INSERT INTO documents (id, collection_id, vector, metadata, content)
		VALUES ($1, $2, $3::vector, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			vector = EXCLUDED.vector,
			metadata = EXCLUDED.metadata,
			content = EXCLUDED.content,
			updated_at = NOW()
		RETURNING id, collection_id, vector, metadata, content, created_at, updated_at, (xmax = 0) AS is_new
	`

	var document models.Document
	var vectorStrReturned string
	var metadataBytes []byte
	var isNew bool

	err = r.db.QueryRowContext(ctx, upsertQuery, documentId, collectionId, vectorStr, metadataJSON, content).Scan(
		&document.Id,
		&document.CollectionId,
		&vectorStrReturned,
		&metadataBytes,
		&document.Content,
		&document.CreatedAt,
		&document.UpdatedAt,
		&isNew,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert document: %w", err)
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

	return &UpsertResult{
		Document: &document,
		IsNew:    isNew,
	}, nil
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

// GetById returns a document with an id
func (r *DocumentRepo) GetById(ctx context.Context, documentId string) (*models.Document, error) {
	selectQuery := `
		SELECT id, collection_id, vector, metadata, content, created_at, updated_at
		FROM documents
		WHERE id = $1
	`

	var document models.Document
	var vectorStrReturned string
	var metadataBytes []byte

	// Query single row by id
	err := r.db.QueryRowContext(ctx, selectQuery, documentId).Scan(
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

	return &document, nil
}

// Delete deletes a document with an id
func (r *DocumentRepo) DeleteById(ctx context.Context, documentId string) error {
	deleteQuery := `
		DELETE FROM documents WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, deleteQuery, documentId)
	if err != nil {
		return err
	}

	// Check if document was actually deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// SearchResult represents a document with its similarity score
type SearchResult struct {
	Document models.Document
	Score    float32
}

// Search performs vector similarity search using cosine similarity
// Returns top-K results ordered by similarity (highest first)
// Supports optional metadata filtering and minimum similarity threshold
func (r *DocumentRepo) Search(ctx context.Context, collectionId string, queryVector []float32, topK int, metadataFilter map[string]any, minThreshold float32) ([]SearchResult, error) {
	// Convert query vector to PostgreSQL format
	vectorStr := vectorToString(queryVector)

	// Build the base query with cosine similarity
	// pgvector's <=> operator is cosine distance (0 = identical, 2 = opposite)
	// We convert to similarity: 1 - distance = similarity (1 = identical, -1 = opposite)
	query := `
		SELECT 
			id, collection_id, vector, metadata, content, created_at, updated_at,
			1 - (vector <=> $1::vector) AS similarity
		FROM documents
		WHERE collection_id = $2
	`

	args := []any{vectorStr, collectionId}
	argIndex := 3

	// Add metadata filtering if provided
	if len(metadataFilter) > 0 {
		for key, value := range metadataFilter {
			// Use JSONB containment operator @> for metadata filtering
			// metadata @> '{"key": "value"}' checks if metadata contains the key-value pair
			query += fmt.Sprintf(" AND metadata @> $%d::jsonb", argIndex)

			// Convert single key-value to JSONB format: {"key": "value"}
			filterJSON, err := json.Marshal(map[string]any{key: value})
			if err != nil {
				return nil, fmt.Errorf("failed to marshal metadata filter: %w", err)
			}
			args = append(args, filterJSON)
			argIndex++
		}
	}

	// Add minimum threshold filtering if provided
	if minThreshold > 0 {
		query += fmt.Sprintf(" AND (1 - (vector <=> $1::vector)) >= $%d", argIndex)
		args = append(args, minThreshold)
		argIndex++
	}

	// Order by similarity (highest first) and limit to top-K
	query += fmt.Sprintf(" ORDER BY vector <=> $1::vector LIMIT $%d", argIndex)
	args = append(args, topK)

	// Execute query
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	var results []SearchResult

	// Scan all rows
	for rows.Next() {
		var result SearchResult
		var vectorStrReturned string
		var metadataBytes []byte

		err = rows.Scan(
			&result.Document.Id,
			&result.Document.CollectionId,
			&vectorStrReturned,
			&metadataBytes,
			&result.Document.Content,
			&result.Document.CreatedAt,
			&result.Document.UpdatedAt,
			&result.Score,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}

		// Parse vector string back to []float32
		result.Document.Vector, err = stringToVector(vectorStrReturned)
		if err != nil {
			return nil, fmt.Errorf("failed to parse returned vector: %w", err)
		}

		// Parse metadata JSON back to map
		if len(metadataBytes) > 0 {
			err = json.Unmarshal(metadataBytes, &result.Document.Metadata)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		results = append(results, result)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	return results, nil
}
