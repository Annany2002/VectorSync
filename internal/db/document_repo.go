package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/Annany2002/vector-sync/internal/models"
	"github.com/lib/pq"
)

// distanceOperator returns the pgvector operator for a given distance metric.
// Cosine: <=> (cosine distance), Euclidean: <-> (L2 distance), Inner Product: <#> (negative inner product)
func distanceOperator(metric string) string {
	switch metric {
	case "euclidean":
		return "<->"
	case "inner_product":
		return "<#>"
	default: // "cosine"
		return "<=>"
	}
}

// similarityExpression returns the SQL expression to convert distance to similarity score.
// For cosine: 1 - distance. For euclidean: 1 / (1 + distance). For inner product: -1 * distance (pgvector returns negative).
func similarityExpression(metric, distExpr string) string {
	switch metric {
	case "euclidean":
		return fmt.Sprintf("1.0 / (1.0 + (%s))", distExpr)
	case "inner_product":
		return fmt.Sprintf("(%s) * -1", distExpr)
	default: // "cosine"
		return fmt.Sprintf("1 - (%s)", distExpr)
	}
}

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

// vectorBuilderPool reuses strings.Builder instances to reduce per-call allocations
// on the hot path (every insert, upsert, batch, and search calls vectorToString).
var vectorBuilderPool = sync.Pool{
	New: func() any { return &strings.Builder{} },
}

// vectorToString converts []float32 to PostgreSQL vector format: '[1.0,2.0,3.0]'
func vectorToString(vec []float32) string {
	b := vectorBuilderPool.Get().(*strings.Builder)
	b.Reset()
	// Pre-allocate: '[' + ~12 chars per float + ',' separators + ']'
	b.Grow(2 + len(vec)*12)
	b.WriteByte('[')
	for i, v := range vec {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(v), 'f', -1, 32))
	}
	b.WriteByte(']')
	s := b.String()
	vectorBuilderPool.Put(b)
	return s
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
// When includeVector is false, the vector column is excluded to reduce payload size
func (r *DocumentRepo) List(ctx context.Context, collectionId string, limit, offset int, includeVector bool) ([]models.Document, error) {
	vectorColumn := ""
	if includeVector {
		vectorColumn = "vector, "
	}

	selectQuery := fmt.Sprintf(`
		SELECT id, collection_id, %smetadata, content, created_at, updated_at
		FROM documents
		WHERE collection_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, vectorColumn)

	rows, err := r.db.QueryContext(ctx, selectQuery, collectionId, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var documents []models.Document

	// Scan all rows one by one
	for rows.Next() {
		var document models.Document
		var metadataBytes []byte

		if includeVector {
			var vectorStrReturned string
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
		} else {
			err = rows.Scan(
				&document.Id,
				&document.CollectionId,
				&metadataBytes,
				&document.Content,
				&document.CreatedAt,
				&document.UpdatedAt,
			)
			if err != nil {
				return nil, err
			}
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

// Search performs vector similarity search using the collection's configured distance metric
// Returns top-K results ordered by similarity (highest first)
// Supports optional metadata filtering and minimum similarity threshold
// When includeVector is false, the vector column is excluded from the SQL SELECT
func (r *DocumentRepo) Search(ctx context.Context, collectionId string, queryVector []float32, topK int, metadataFilter map[string]any, minThreshold float32, includeVector bool, distanceMetric string) ([]SearchResult, error) {
	// Convert query vector to PostgreSQL format
	vectorStr := vectorToString(queryVector)

	// Get the pgvector operator and similarity expression for this metric
	op := distanceOperator(distanceMetric)
	distExpr := fmt.Sprintf("vector %s $1::vector", op)
	simExpr := similarityExpression(distanceMetric, distExpr)

	vectorColumn := ""
	if includeVector {
		vectorColumn = "vector, "
	}
	query := fmt.Sprintf(`
		SELECT 
			id, collection_id, %smetadata, content, created_at, updated_at,
			%s AS similarity
		FROM documents
		WHERE collection_id = $2
	`, vectorColumn, simExpr)

	args := []any{vectorStr, collectionId}
	argIndex := 3

	// Add metadata filtering if provided
	if len(metadataFilter) > 0 {
		for key, value := range metadataFilter {
			query += fmt.Sprintf(" AND metadata @> $%d::jsonb", argIndex)
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
		query += fmt.Sprintf(" AND (%s) >= $%d", simExpr, argIndex)
		args = append(args, minThreshold)
		argIndex++
	}

	// Order by distance (ascending = most similar first) and limit to top-K
	query += fmt.Sprintf(" ORDER BY %s LIMIT $%d", distExpr, argIndex)
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

// BatchInsert inserts a group of documents inside a collection
// Uses an explicit transaction to reduce WAL overhead and ensure atomicity.
func (r *DocumentRepo) BatchInsert(ctx context.Context, collectionId string, documents []models.Document) (int, []models.Document, error) {
	// Start explicit transaction for atomic batch insert
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // no-op if tx.Commit() succeeds

	// Skip WAL fsync wait for this transaction. The data is written to the WAL
	// buffer and flushed asynchronously by the background writer, avoiding the
	// per-commit disk stall that dominates batch insert latency.
	if _, err = tx.ExecContext(ctx, "SET LOCAL synchronous_commit = OFF"); err != nil {
		return 0, nil, fmt.Errorf("failed to set synchronous_commit: %w", err)
	}

	n := len(documents)

	// Pre-serialize vectors and metadata in parallel across goroutines.
	// Each document's serialization is independent, so we fan out the CPU work
	// (vectorToString + json.Marshal) then assemble the query string sequentially.
	type serialized struct {
		vectorStr    string
		metadataJSON []byte
		err          error
	}
	prepped := make([]serialized, n)

	// Use a WaitGroup to fan out serialization. For small batches (<32 docs)
	// the overhead of goroutines isn't worth it, so we serialize inline.
	if n >= 32 {
		var wg sync.WaitGroup
		wg.Add(n)
		for i := range documents {
			go func(idx int) {
				defer wg.Done()
				prepped[idx].vectorStr = vectorToString(documents[idx].Vector)
				prepped[idx].metadataJSON, prepped[idx].err = json.Marshal(documents[idx].Metadata)
			}(i)
		}
		wg.Wait()
	} else {
		for i := range documents {
			prepped[i].vectorStr = vectorToString(documents[i].Vector)
			prepped[i].metadataJSON, prepped[i].err = json.Marshal(documents[i].Metadata)
		}
	}

	// Check for marshalling errors
	for i := range prepped {
		if prepped[i].err != nil {
			return 0, nil, fmt.Errorf("failed to marshal metadata for doc %d: %w", i, prepped[i].err)
		}
	}

	// Pre-allocate query builder and args slice to avoid incremental re-allocations.
	var query strings.Builder
	query.Grow(60 + n*26) // base header + ~26 chars per placeholder group
	query.WriteString("INSERT INTO documents (collection_id, vector, metadata, content) VALUES")

	// Pre-allocate: 4 args per document
	contents := make([]any, 0, n*4)

	// numBuf is a stack-allocated scratch buffer for integer formatting,
	// avoiding the heap allocations of fmt.Sprintf inside the hot loop.
	var numBuf [20]byte

	for i, v := range documents {
		// Write placeholder group directly without fmt.Sprintf
		base := i*4 + 1
		query.WriteString("($")
		query.Write(strconv.AppendInt(numBuf[:0], int64(base), 10))
		query.WriteString(",$")
		query.Write(strconv.AppendInt(numBuf[:0], int64(base+1), 10))
		query.WriteString("::vector,$")
		query.Write(strconv.AppendInt(numBuf[:0], int64(base+2), 10))
		query.WriteString(",$")
		query.Write(strconv.AppendInt(numBuf[:0], int64(base+3), 10))
		query.WriteByte(')')
		if i < n-1 {
			query.WriteByte(',')
		}

		contents = append(contents, v.CollectionId, prepped[i].vectorStr, prepped[i].metadataJSON, v.Content)
	}

	// add returning statement to return the document
	query.WriteString(" RETURNING id, collection_id, metadata, content, created_at, updated_at")

	// docs represent the documents that are successfully inserted and are returned by query
	var docs []models.Document

	rows, err := tx.QueryContext(ctx, query.String(), contents...)
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

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return 0, nil, fmt.Errorf("failed to commit batch insert: %w", err)
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
// Score = (textWeight * fts_rank) + (vectorWeight * vector_similarity)
// Supports optional metadata filtering via JSONB containment
func (r *DocumentRepo) HybridSearch(ctx context.Context, collectionId, queryText string, queryVector []float32, topK int, metadataFilter map[string]any, vectorWeight, textWeight float32, includeVector bool, distanceMetric string) ([]SearchResult, error) {
	// Convert query vector to PostgreSQL format
	vectorStr := vectorToString(queryVector)

	vectorColumn := ""
	if includeVector {
		vectorColumn = "vector, "
	}

	// Get the pgvector operator and similarity expression for this metric
	op := distanceOperator(distanceMetric)

	var hybrid string
	var args []any
	var argIndex int

	if queryText != "" {
		distExpr := fmt.Sprintf("vector %s $3::vector", op)
		simExpr := similarityExpression(distanceMetric, distExpr)

		hybrid = fmt.Sprintf(`
		WITH scored AS (
			SELECT id, collection_id, %smetadata, content, created_at, updated_at,
				ts_rank(to_tsvector('english', COALESCE(content, '')),
				plainto_tsquery('english', $1)) as fts_score,
				%s AS vector_score
			FROM documents
			WHERE collection_id = $2 
				AND to_tsvector('english', COALESCE(content, ''))
					@@ plainto_tsquery('english', $1)
		`, vectorColumn, simExpr)
		args = []any{queryText, collectionId, vectorStr}
		argIndex = 4
	} else {
		distExpr := fmt.Sprintf("vector %s $2::vector", op)
		simExpr := similarityExpression(distanceMetric, distExpr)

		hybrid = fmt.Sprintf(`
		WITH scored AS (
			SELECT id, collection_id, %smetadata, content, created_at, updated_at,
				0::float as fts_score,
				%s AS vector_score
			FROM documents
			WHERE collection_id = $1
		`, vectorColumn, simExpr)
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
