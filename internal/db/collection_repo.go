// CollectionRepo is the interface for the collection repository
package db

import (
	"context"
	"database/sql"
	"encoding/json"

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

// Create creates a new collection in the database
func (r *CollectionRepo) Create(ctx context.Context, name string, vectorDimension int32, metadataSchema map[string]any) (*models.Collection, error) {
	// Only insert the fields we provide: name, vector_dim, metadata_schema
	insertQuery := `
		INSERT INTO collections (name, vector_dim, metadata_schema)
		VALUES ($1, $2, $3)
		RETURNING id, name, vector_dim, metadata_schema, created_at, updated_at
	`

	// Convert map to JSON for JSONB column
	metadataJSON, err := json.Marshal(metadataSchema)
	if err != nil {
		return nil, err
	}

	var collection models.Collection
	var metadataBytes []byte

	// Scan each field individually from the RETURNING clause
	err = r.db.QueryRowContext(ctx, insertQuery, name, vectorDimension, metadataJSON).Scan(
		&collection.ID,
		&collection.Name,
		&collection.VectorDimension,
		&metadataBytes,
		&collection.CreatedAt,
		&collection.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert JSON bytes back to map
	if len(metadataBytes) > 0 {
		err = json.Unmarshal(metadataBytes, &collection.MetadataSchema)
		if err != nil {
			return nil, err
		}
	}

	return &collection, nil
}

// List returns collections with pagination support
func (r *CollectionRepo) List(ctx context.Context, limit, offset int) ([]models.Collection, error) {
	selectQuery := `
		SELECT id, name, vector_dim, metadata_schema, created_at, updated_at
		FROM collections
		ORDER BY id
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, selectQuery, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []models.Collection
	for rows.Next() {
		var collection models.Collection
		var metadataBytes []byte

		err = rows.Scan(
			&collection.ID,
			&collection.Name,
			&collection.VectorDimension,
			&metadataBytes,
			&collection.CreatedAt,
			&collection.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Convert JSON bytes back to map
		if len(metadataBytes) > 0 {
			err = json.Unmarshal(metadataBytes, &collection.MetadataSchema)
			if err != nil {
				return nil, err
			}
		}

		collections = append(collections, collection)
	}

	return collections, nil
}

// GetCollectionByName gets a collection by name
func (r *CollectionRepo) GetCollectionByName(ctx context.Context, name string) (*models.Collection, error) {
	// Selecting the columns we need
	selectQuery := `
		SELECT id, name, vector_dim, metadata_schema, created_at, updated_at
		FROM collections
		WHERE name = $1
	`

	var collection models.Collection
	var metadataBytes []byte

	err := r.db.QueryRowContext(ctx, selectQuery, name).Scan(
		&collection.ID,
		&collection.Name,
		&collection.VectorDimension,
		&metadataBytes,
		&collection.CreatedAt,
		&collection.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert JSON bytes to map
	if len(metadataBytes) > 0 {
		err = json.Unmarshal(metadataBytes, &collection.MetadataSchema)
		if err != nil {
			return nil, err
		}
	}

	return &collection, nil
}
