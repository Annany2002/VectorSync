package services

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Annany2002/vector-sync/internal/db"
	"github.com/Annany2002/vector-sync/internal/models"
)

// CollectionService is the service for collection operations
type CollectionService struct {
	repo db.CollectionRepo
}

// NewCollectionService creates a new collection service
func NewCollectionService(repo db.CollectionRepo) *CollectionService {
	return &CollectionService{repo: repo}
}

// CreateCollection creates a new collection
func (s *CollectionService) CreateCollection(ctx context.Context, name string, vectorDimension int32, metadataSchema map[string]any) (*models.Collection, error) {
	// validate the input
	if name == "" {
		return nil, errors.New("name is required")
	}
	if vectorDimension <= 0 {
		return nil, errors.New("vector dimension must be greater than 0")
	}
	if metadataSchema == nil {
		metadataSchema = make(map[string]any)
	}

	// check if the collection already exists
	existing, err := s.repo.GetCollectionByName(ctx, name)

	// If we found a collection (no error), it's a duplicate
	if err == nil && existing != nil {
		return nil, errors.New("collection already exists")
	}

	// If there's a database error (not just "not found"), return it
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Collection doesn't exist - safe to create
	collection, err := s.repo.Create(ctx, name, vectorDimension, metadataSchema)
	if err != nil {
		return nil, err
	}
	return collection, nil
}
