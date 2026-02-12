package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Annany2002/vector-sync/internal/models"
	"github.com/lib/pq"
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
		RETURNING id, collection_id, metadata, content, created_at, updated_at
	`

	var document models.Document
	var metadataBytes []byte

	err = r.db.QueryRowContext(ctx, insertQuery, collectionId, vectorStr, metadataJSON, content).Scan(
		&document.Id,
		&document.CollectionId,
		&metadataBytes,
		&document.Content,
		&document.CreatedAt,
		&document.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert document: %w", err)
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
		RETURNING id, collection_id, metadata, content, created_at, updated_at, (xmax = 0) AS is_new
	`

	var document models.Document
	var metadataBytes []byte
	var isNew bool

	err = r.db.QueryRowContext(ctx, upsertQuery, documentId, collectionId, vectorStr, metadataJSON, content).Scan(
		&document.Id,
		&document.CollectionId,
		&metadataBytes,
		&document.Content,
		&document.CreatedAt,
		&document.UpdatedAt,
		&isNew,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert document: %w", err)
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
	var b strings.Builder
	// Pre-allocate: '[' + ~12 chars per float + ',' separators + ']'
	b.Grow(1 + len(vec)*12 + 1)
	b.WriteByte('[')
	for i, v := range vec {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(v), 'f', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
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
		val, err := strconv.ParseFloat(strings.TrimSpace(part), 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse vector value at index %d: %w", i, err)
		}
		vec[i] = float32(val)
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
// When includeVector is false, the vector column is excluded from the SQL SELECT
// to avoid transferring ~3KB per result and skipping deserialization
func (r *DocumentRepo) Search(ctx context.Context, collectionId string, queryVector []float32, topK int, metadataFilter map[string]any, minThreshold float32, includeVector bool) ([]SearchResult, error) {
	// Convert query vector to PostgreSQL format
	vectorStr := vectorToString(queryVector)

	// Build the base query with cosine similarity
	// pgvector's <=> operator is cosine distance (0 = identical, 2 = opposite)
	// We convert to similarity: 1 - distance = similarity (1 = identical, -1 = opposite)
	vectorColumn := ""
	if includeVector {
		vectorColumn = "vector, "
	}
	query := fmt.Sprintf(`
		SELECT 
			id, collection_id, %smetadata, content, created_at, updated_at,
			1 - (vector <=> $1::vector) AS similarity
		FROM documents
		WHERE collection_id = $2
	`, vectorColumn)

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
		var metadataBytes []byte

		if includeVector {
			var vectorStrReturned string
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
			result.Document.Vector, err = stringToVector(vectorStrReturned)
			if err != nil {
				return nil, fmt.Errorf("failed to parse returned vector: %w", err)
			}
		} else {
			err = rows.Scan(
				&result.Document.Id,
				&result.Document.CollectionId,
				&metadataBytes,
				&result.Document.Content,
				&result.Document.CreatedAt,
				&result.Document.UpdatedAt,
				&result.Score,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to scan search result: %w", err)
			}
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

// FullTextSearch performs full-text search on document content
// Uses PostgreSQL tsvector/tsquery for text matching and ts_rank for relevance scoring
// When includeVector is false, the vector column is excluded from SQL to reduce payload
func (r *DocumentRepo) FullTextSearch(ctx context.Context, collectionId, query string, limit int32, minRank float32, includeVector bool) ([]SearchResult, error) {
	// Note:
	// Query is built dynamically because minRank is optional.
	// When minRank is 0, we skip the threshold filter for better performance.

	vectorColumn := ""
	if includeVector {
		vectorColumn = "vector, "
	}

	// base query
	searchQuery := fmt.Sprintf(`
		SELECT id, collection_id, %smetadata, content, created_at, updated_at,
			ts_rank(to_tsvector('english', COALESCE(content, '')),
			plainto_tsquery('english', $1)) as rank
		FROM documents
		WHERE collection_id = $2 
			AND to_tsvector('english', COALESCE(content, ''))
				@@ plainto_tsquery('english', $1)
	`, vectorColumn)

	// args holds the ACTUAL VALUES that replace $1, $2, $3, etc.
	// argIndex tracks the NEXT placeholder number to use
	args := []any{query, collectionId}
	argIndex := 3

	// Add minimum rank threshold filter if specified
	if minRank > 0 {
		searchQuery += fmt.Sprintf(" AND ts_rank(to_tsvector('english', COALESCE(content, '')), plainto_tsquery('english', $1)) >= $%d", argIndex)

		// Append minRank value to args slice
		// args becomes: [query, collectionId, minRank]
		args = append(args, minRank)
		argIndex++
	}
	// add ORDER BY and LIMIT since it will be needed always
	searchQuery += fmt.Sprintf(" ORDER BY rank DESC LIMIT $%d", argIndex)
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, searchQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	var results []SearchResult

	// Scan all rows
	for rows.Next() {
		var result SearchResult
		var metadataBytes []byte

		if includeVector {
			var vectorStrReturned string
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
			result.Document.Vector, err = stringToVector(vectorStrReturned)
			if err != nil {
				return nil, fmt.Errorf("failed to parse returned vector: %w", err)
			}
		} else {
			err = rows.Scan(
				&result.Document.Id,
				&result.Document.CollectionId,
				&metadataBytes,
				&result.Document.Content,
				&result.Document.CreatedAt,
				&result.Document.UpdatedAt,
				&result.Score,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to scan search result: %w", err)
			}
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

// BatchInsert inserts a group of document inside a colletion
// For now we will just use one atomic insert operation
// Later on, we will modify this to handle errors, retries etc
func (r *DocumentRepo) BatchInsert(ctx context.Context, collectionId string, documents []models.Document) (int, []models.Document, error) {
	// we will use string builder to efficiently create
	// strings from many smaller strings and a custom array
	// of contents to insert each document
	var (
		query    strings.Builder
		contents []any
	)

	// build the initial query
	query.WriteString("INSERT INTO documents (collection_id, vector, metadata, content) VALUES")

	for i, v := range documents {
		idx := i * 4
		// Convert the vector of the document to string
		vectorStr := vectorToString(v.Vector)

		// Convert the metadata to []byte
		metadataJSON, err := json.Marshal(v.Metadata)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}

		// create a single insert line by line
		query.WriteString(fmt.Sprintf("($%d, $%d::vector, $%d, $%d)", idx+1, idx+2, idx+3, idx+4))

		// append "," after every line insert
		if i < len(documents)-1 {
			query.WriteString(",")
		}

		// we append each document in the values content to later pass in execCtx function
		contents = append(contents, v.CollectionId, vectorStr, metadataJSON, v.Content)
	}

	// add returning statement to return the document
	query.WriteString(" RETURNING id, collection_id, metadata, content, created_at, updated_at")

	// docs represent the documents that are successfully inserted and are returned by query
	var docs []models.Document

	rows, err := r.db.QueryContext(ctx, query.String(), contents...)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	// scan the rows to extract individual docs
	for rows.Next() {
		var doc models.Document
		var metadata []byte

		err := rows.Scan(
			&doc.Id,
			&doc.CollectionId,
			&metadata,
			&doc.Content,
			&doc.CreatedAt,
			&doc.UpdatedAt,
		)

		if err != nil {
			return 0, nil, err
		}

		// Parse metadata JSON back to map
		if len(metadata) > 0 {
			err = json.Unmarshal(metadata, &doc.Metadata)
			if err != nil {
				return 0, nil, err
			}
		}

		docs = append(docs, doc)
	}
	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		return 0, nil, err
	}

	return len(docs), docs, nil
}

// BatchDelete deletes a group of documents inside a collection
// Uses atomic operation with RETURNING to get deleted documents
func (r *DocumentRepo) BatchDelete(ctx context.Context, collectionId string, documentIds []string) (int, []models.Document, error) {
	// to efficiently delete multiple rows we use `ANY` to speed up
	// delete operations compared to using `IN`
	query := `
		DELETE FROM documents 
		WHERE ID = ANY($1::uuid[]) 
		AND 
		collection_id = $2
		RETURNING id, collection_id, metadata, content, created_at, updated_at
	`

	var docs []models.Document

	rows, err := r.db.QueryContext(ctx, query, pq.Array(documentIds), collectionId)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var doc models.Document
		var metadata []byte

		err := rows.Scan(
			&doc.Id,
			&doc.CollectionId,
			&metadata,
			&doc.Content,
			&doc.CreatedAt,
			&doc.UpdatedAt,
		)

		if err != nil {
			return 0, nil, err
		}

		// Parse metadata JSON back to map
		if len(metadata) > 0 {
			err = json.Unmarshal(metadata, &doc.Metadata)
			if err != nil {
				return 0, nil, err
			}
		}

		docs = append(docs, doc)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		return 0, nil, err
	}

	return len(docs), docs, nil
}

// HybridSearch performs a combined vector similarity and full-text search
// Returns top-K results ordered by weighted combined score (highest first)
// Score = (textWeight * fts_rank) + (vectorWeight * cosine_similarity)
// Supports optional metadata filtering via JSONB containment
func (r *DocumentRepo) HybridSearch(ctx context.Context, collectionId, queryText string, queryVector []float32, topK int, metadataFilter map[string]any, vectorWeight, textWeight float32, includeVector bool) ([]SearchResult, error) {
	// Convert query vector to PostgreSQL format
	vectorStr := vectorToString(queryVector)

	vectorColumn := ""
	if includeVector {
		vectorColumn = "vector, "
	}

	// Build CTE that computes both scores for each matching document:
	// - fts_score: PostgreSQL ts_rank for full-text relevance (0 if no text query)
	// - vector_score: cosine similarity (1 - cosine_distance)
	// Text filter is only applied when queryText is provided
	var hybrid string
	var args []any
	var argIndex int

	if queryText != "" {
		// Both text and vector search
		hybrid = fmt.Sprintf(`
		WITH scored AS (
			SELECT id, collection_id, %smetadata, content, created_at, updated_at,
				ts_rank(to_tsvector('english', COALESCE(content, '')),
				plainto_tsquery('english', $1)) as fts_score,
				1 - (vector <=> $3::vector) AS vector_score
			FROM documents
			WHERE collection_id = $2 
				AND to_tsvector('english', COALESCE(content, ''))
					@@ plainto_tsquery('english', $1)
		`, vectorColumn)
		args = []any{queryText, collectionId, vectorStr}
		argIndex = 4
	} else {
		// Vector-only search (no text filter)
		hybrid = fmt.Sprintf(`
		WITH scored AS (
			SELECT id, collection_id, %smetadata, content, created_at, updated_at,
				0::float as fts_score,
				1 - (vector <=> $2::vector) AS vector_score
			FROM documents
			WHERE collection_id = $1
		`, vectorColumn)
		args = []any{collectionId, vectorStr}
		argIndex = 3
	}

	// Add metadata filtering if provided
	if len(metadataFilter) > 0 {
		for key, value := range metadataFilter {
			// Use JSONB containment operator @> for metadata filtering
			hybrid += fmt.Sprintf(" AND metadata @> $%d::jsonb", argIndex)

			// Convert single key-value to JSONB format
			filterJSON, err := json.Marshal(map[string]any{key: value})
			if err != nil {
				return nil, fmt.Errorf("failed to marshal metadata filter: %w", err)
			}
			args = append(args, filterJSON)
			argIndex++
		}
	}

	// Close CTE and compute weighted combined score for final ordering
	hybrid += fmt.Sprintf(`
)
SELECT id, collection_id, %smetadata, content, created_at, updated_at,
    ($%d * fts_score) + ($%d * vector_score) AS combined_score
FROM scored
ORDER BY combined_score DESC
LIMIT $%d
`, vectorColumn, argIndex, argIndex+1, argIndex+2)

	args = append(args, textWeight, vectorWeight, topK)

	// Execute query
	rows, err := r.db.QueryContext(ctx, hybrid, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	var results []SearchResult

	// Scan all rows
	for rows.Next() {
		var result SearchResult
		var metadataBytes []byte

		if includeVector {
			var vectorStrReturned string
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
			result.Document.Vector, err = stringToVector(vectorStrReturned)
			if err != nil {
				return nil, fmt.Errorf("failed to parse returned vector: %w", err)
			}
		} else {
			err = rows.Scan(
				&result.Document.Id,
				&result.Document.CollectionId,
				&metadataBytes,
				&result.Document.Content,
				&result.Document.CreatedAt,
				&result.Document.UpdatedAt,
				&result.Score,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to scan search result: %w", err)
			}
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
