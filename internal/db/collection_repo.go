// CollectionRepo is the interface for the collection repository
package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Annany2002/vector-sync/internal/metrics"
	"github.com/Annany2002/vector-sync/internal/models"
)

// CollectionRepo is the interface for the collection repository
type CollectionRepo struct {
	db *sql.DB
}

// NewCollectionRepo creates a new collection repository
func NewCollectionRepo(db *sql.DB) *CollectionRepo {
	return &CollectionRepo{db: db}
}

// distanceMetricOpsClass maps a distance metric to the pgvector HNSW operator class
func distanceMetricOpsClass(metric string) string {
	switch metric {
	case "euclidean":
		return "vector_l2_ops"
	case "inner_product":
		return "vector_ip_ops"
	default: // "cosine"
		return "vector_cosine_ops"
	}
}

// Create creates a new collection and builds a partial HNSW index for it
func (r *CollectionRepo) Create(ctx context.Context, name string, vectorDimension int32, metadataSchema map[string]any, distanceMetric string, embeddingProvider, embeddingModel string) (*models.Collection, error) {
	// Default distance metric to cosine
	if distanceMetric == "" {
		distanceMetric = "cosine"
	}

	insertQuery := `
		INSERT INTO collections (name, vector_dim, metadata_schema, distance_metric, embedding_provider, embedding_model)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, vector_dim, distance_metric, metadata_schema, embedding_provider, embedding_model, created_at, updated_at, document_count
	`

	// Convert map to JSON for JSONB column
	metadataJSON, err := json.Marshal(metadataSchema)
	if err != nil {
		return nil, err
	}

	var ep, em sql.NullString
	if embeddingProvider != "" {
		ep = sql.NullString{String: embeddingProvider, Valid: true}
	}
	if embeddingModel != "" {
		em = sql.NullString{String: embeddingModel, Valid: true}
	}

	var collection models.Collection
	var metadataBytes []byte
	var retEp, retEm sql.NullString

	err = r.db.QueryRowContext(ctx, insertQuery, name, vectorDimension, metadataJSON, distanceMetric, ep, em).Scan(
		&collection.Id,
		&collection.Name,
		&collection.VectorDimension,
		&collection.DistanceMetric,
		&metadataBytes,
		&retEp,
		&retEm,
		&collection.CreatedAt,
		&collection.UpdatedAt,
		&collection.DocumentCount,
	)
	if err != nil {
		return nil, err
	}

	if retEp.Valid {
		collection.EmbeddingProvider = retEp.String
	}
	if retEm.Valid {
		collection.EmbeddingModel = retEm.String
	}

	// Convert JSON bytes back to map
	if len(metadataBytes) > 0 {
		err = json.Unmarshal(metadataBytes, &collection.MetadataSchema)
		if err != nil {
			return nil, err
		}
	}

	// Create per-collection partial HNSW index for vector search acceleration
	opsClass := distanceMetricOpsClass(distanceMetric)
	indexQuery := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_hnsw_%s
		ON documents USING hnsw (vector %s)
		WITH (m = 16, ef_construction = 64)
		WHERE collection_id = '%s'
	`, collection.Id, opsClass, collection.Id)

	// Index creation runs in a background goroutine so collection creates return
	// immediately without blocking on the DDL statement.
	// context.Background() is intentional: the caller's context may be cancelled
	// before the goroutine executes, but we still want the index to be created.
	go func() {
		start := time.Now()
		_, err := r.db.ExecContext(context.Background(), indexQuery)
		metrics.HNSWBuildDuration.Observe(time.Since(start).Seconds())
		if err != nil {
			metrics.HNSWBuildErrors.Inc()
		}
	}()

	return &collection, nil
}

// scanCollection scans a single collection row (shared by List, ListById, GetCollectionByName)
func scanCollection(row interface{ Scan(dest ...any) error }) (*models.Collection, error) {
	var collection models.Collection
	var metadataBytes []byte
	var ep, em sql.NullString

	err := row.Scan(
		&collection.Id,
		&collection.Name,
		&collection.VectorDimension,
		&collection.DistanceMetric,
		&metadataBytes,
		&ep,
		&em,
		&collection.CreatedAt,
		&collection.UpdatedAt,
		&collection.DocumentCount,
	)
	if err != nil {
		return nil, err
	}

	if ep.Valid {
		collection.EmbeddingProvider = ep.String
	}
	if em.Valid {
		collection.EmbeddingModel = em.String
	}

	if len(metadataBytes) > 0 {
		err = json.Unmarshal(metadataBytes, &collection.MetadataSchema)
		if err != nil {
			return nil, err
		}
	}

	return &collection, nil
}

const collectionColumns = "id, name, vector_dim, distance_metric, metadata_schema, embedding_provider, embedding_model, created_at, updated_at, document_count"

// List returns collections with pagination support
func (r *CollectionRepo) List(ctx context.Context, limit, offset int) ([]models.Collection, error) {
	selectQuery := fmt.Sprintf(`
		SELECT %s FROM collections ORDER BY id LIMIT $1 OFFSET $2
	`, collectionColumns)

	rows, err := r.db.QueryContext(ctx, selectQuery, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []models.Collection
	for rows.Next() {
		collection, err := scanCollection(rows)
		if err != nil {
			return nil, err
		}
		collections = append(collections, *collection)
	}

	return collections, nil
}

// ListById returns a collection with an id
func (r *CollectionRepo) ListById(ctx context.Context, collectionId string) (*models.Collection, error) {
	selectQuery := fmt.Sprintf(`SELECT %s FROM collections WHERE id = $1`, collectionColumns)
	return scanCollection(r.db.QueryRowContext(ctx, selectQuery, collectionId))
}

// DeleteById deletes a collection with an id and drops its HNSW index
func (r *CollectionRepo) DeleteById(ctx context.Context, collectionId string) (int64, error) {
	deleteQuery := `
		DELETE FROM collections WHERE id = $1
		RETURNING document_count
	`

	var documentCount int64
	err := r.db.QueryRowContext(ctx, deleteQuery, collectionId).Scan(&documentCount)
	if err != nil {
		return 0, err
	}

	// Drop the per-collection HNSW index in the background. The DELETE above
	// cascade-deletes all documents, so the index is already orphaned.
	// context.Background() ensures the DROP completes even if the caller's
	// context is cancelled.
	go func() {
		dropIndexQuery := fmt.Sprintf(`DROP INDEX IF EXISTS idx_hnsw_%s`, collectionId)
		_, _ = r.db.ExecContext(context.Background(), dropIndexQuery)
	}()

	return documentCount, nil
}

// GetCollectionByName gets a collection by name
func (r *CollectionRepo) GetCollectionByName(ctx context.Context, name string) (*models.Collection, error) {
	selectQuery := fmt.Sprintf(`SELECT %s FROM collections WHERE name = $1`, collectionColumns)
	return scanCollection(r.db.QueryRowContext(ctx, selectQuery, name))
}
