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
func (s *DocumentService) CreateDocument(ctx context.Context, collectionID, content string, vector []float32, metadata map[string]any) (*models.Document, error) {
	// Check #1: Collection ID not empty
	if collectionID == "" {
		return nil, errors.New("collection ID is required")
	}

	// Check #2: Vector not empty
	if len(vector) == 0 {
		return nil, errors.New("vector cannot be empty")
	}

	// Check #3: Initialize metadata if nil
	if metadata == nil {
		metadata = make(map[string]any)
	}

	// Check #4: Collection exists (and fetch it for dimension validation)
	collection, err := s.collectionRepo.ListById(ctx, collectionID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("collection with ID %s not found", collectionID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch collection: %w", err)
	}

	// Check #5: Vector dimension matches collection's expected dimension
	if len(vector) != collection.VectorDimension {
		return nil, fmt.Errorf(
			"vector dimension mismatch: collection expects %d dimensions, got %d",
			collection.VectorDimension,
			len(vector),
		)
	}

	// Check #6: Validate vector values (no NaN or Infinity)
	for i, val := range vector {
		if math.IsNaN(float64(val)) {
			return nil, fmt.Errorf("vector contains NaN at index %d", i)
		}
		if math.IsInf(float64(val), 0) {
			return nil, fmt.Errorf("vector contains Infinity at index %d", i)
		}
	}

	// All validations passed - create the document
	document, err := s.documentRepo.Create(ctx, collectionID, content, vector, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	return document, nil
}
