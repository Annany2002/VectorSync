package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/Annany2002/vector-sync/internal/db"
	"github.com/Annany2002/vector-sync/internal/models"
)

// DocumentService is the service for document operations
type DocumentService struct {
	documentRepo   db.DocumentRepo
	collectionRepo db.CollectionRepo
}

// NewDocumentService creates a new document service
func NewDocumentService(documentRepo db.DocumentRepo, collectionRepo db.CollectionRepo) *DocumentService {
	return &DocumentService{
		documentRepo:   documentRepo,
		collectionRepo: collectionRepo,
	}
}

// CreateDocument creates a new document
func (s *DocumentService) CreateDocument(ctx context.Context, collectionId, content string, vector []float32, metadata map[string]any) (*models.Document, error) {
	// Perform null checks
	if collectionId == "" {
		return nil, errors.New("collection_id is required")
	}
	if len(vector) == 0 {
		return nil, errors.New("vector cannot be empty")
	}
	if metadata == nil {
		metadata = make(map[string]any)
	}

	// Collection exists (and fetch it for dimension validation)
	collection, err := s.collectionRepo.ListById(ctx, collectionId)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("collection_id %s not found", collectionId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch collection: %w", err)
	}

	// Vector dimension matches collection's expected dimension
	if len(vector) != collection.VectorDimension {
		return nil, fmt.Errorf(
			"vector dimension mismatch: collection expects %d dimensions, got %d",
			collection.VectorDimension,
			len(vector),
		)
	}

	// Validate vector values (no NaN or Infinity)
	for i, val := range vector {
		if math.IsNaN(float64(val)) {
			return nil, fmt.Errorf("vector contains NaN at index %d", i)
		}
		if math.IsInf(float64(val), 0) {
			return nil, fmt.Errorf("vector contains Infinity at index %d", i)
		}
	}

	// Create the document
	document, err := s.documentRepo.Create(ctx, collectionId, content, vector, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	return document, nil
}

// Pagination Defaults
const (
	defaultLimit = 50
	maxLimit     = 1000
)

// ListDocuments returns documents from a collection with pagination
func (s *DocumentService) ListDocuments(ctx context.Context, collectionId string, limit, offset int32) ([]models.Document, error) {
	// check for empty collectionId
	if collectionId == "" {
		return nil, errors.New("collection_id is required")
	}

	// check if collection exists
	_, err := s.collectionRepo.ListById(ctx, collectionId)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("collection_id %s not found", collectionId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch collection: %w", err)
	}

	// Validate and normalize limit
	if limit < 0 {
		return nil, errors.New("limit cannot be negative")
	}
	if limit == 0 {
		limit = defaultLimit // Use default if not specified
	}
	if limit > maxLimit {
		return nil, fmt.Errorf("limit cannot exceed %d", maxLimit)
	}

	// Validate offset
	if offset < 0 {
		return nil, errors.New("offset cannot be negative")
	}
	// Note: We allow any non-negative offset for deep pagination

	// fetch documents from repository
	documents, err := s.documentRepo.List(ctx, collectionId, int(limit), int(offset))
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	return documents, nil
}
