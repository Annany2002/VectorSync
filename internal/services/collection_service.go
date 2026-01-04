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

// Pagination defaults
const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// ListCollections retrieves collections with pagination
func (s *CollectionService) ListCollections(ctx context.Context, limit, offset int) ([]models.Collection, error) {
	// Apply default limit if not specified or invalid
	if limit <= 0 {
		limit = DefaultLimit
	}

	// Cap limit to prevent database overload
	if limit > MaxLimit {
		limit = MaxLimit
	}

	// Ensure offset is non-negative
	if offset < 0 {
		offset = 0
	}

	// Call the repository
	collections, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return collections, nil
}

// ListCollection retrieves a collection with an id
func (s *CollectionService) ListCollection(ctx context.Context, collectionId string) (*models.Collection, error) {
	// Check if id is not empty
	if collectionId == "" {
		return nil, errors.New("collectionId cannot be empty")
	}

	// Call the repository
	collection, err := s.repo.ListById(ctx, collectionId)
	if err != nil {
		return nil, err
	}
	return collection, nil
}

// DeleteCollection deletes a collection with an id
func (s *CollectionService) DeleteCollection(ctx context.Context, collectionId string) (int64, error) {
	// Check if id is not empty
	if collectionId == "" {
		return 0, errors.New("collectionId cannot be empty")
	}

	// Call the repository
	documentCount, err := s.repo.DeleteById(ctx, collectionId)
	if err != nil {
		return 0, err
	}
	return documentCount, nil
}
