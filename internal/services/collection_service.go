package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Annany2002/vector-sync/internal/models"
)

// CollectionService is the service for collection operations
type CollectionService struct {
	repo CollectionRepository
}

// NewCollectionService creates a new collection service
func NewCollectionService(repo CollectionRepository) *CollectionService {
	return &CollectionService{repo: repo}
}

// CreateCollection creates a new collection
func (s *CollectionService) CreateCollection(ctx context.Context, name string, vectorDimension int32, metadataSchema map[string]any, distanceMetric string, embeddingProvider, embeddingModel string) (*models.Collection, error) {
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

	// Validate and default distance metric
	if distanceMetric == "" {
		distanceMetric = "cosine"
	}
	switch distanceMetric {
	case "cosine", "euclidean", "inner_product":
		// valid
	default:
		return nil, fmt.Errorf("invalid distance_metric %q: must be cosine, euclidean, or inner_product", distanceMetric)
	}

	// Validate embedding provider/model consistency
	if embeddingProvider != "" {
		switch embeddingProvider {
		case "openai", "ollama", "cohere":
			// valid
		default:
			return nil, fmt.Errorf("unsupported embedding provider %q", embeddingProvider)
		}
		if embeddingModel == "" {
			return nil, errors.New("embedding_model is required when embedding_provider is set")
		}
	} else if embeddingModel != "" {
		return nil, errors.New("embedding_provider is required when embedding_model is set")
	}

	// Insert directly and let the UNIQUE constraint on collections.name
	// catch duplicates. This eliminates a DB round-trip (SELECT before INSERT)
	// and the TOCTOU race where two concurrent creates could both pass the
	// check and then one fails on insert anyway.
	collection, err := s.repo.Create(ctx, name, vectorDimension, metadataSchema, distanceMetric, embeddingProvider, embeddingModel)
	if err != nil {
		// PostgreSQL unique violation: code 23505
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return nil, errors.New("collection already exists")
		}
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
