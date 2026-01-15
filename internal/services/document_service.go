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

// UpsertDocument inserts or updates a document
func (s *DocumentService) UpsertDocument(ctx context.Context, documentId, collectionId, content string, vector []float32, metadata map[string]any) (*db.UpsertResult, error) {
	// Perform null checks
	if documentId == "" {
		return nil, errors.New("document_id is required for upsert")
	}
	if collectionId == "" {
		return nil, errors.New("collection_id is required")
	}
	if len(vector) == 0 {
		return nil, errors.New("vector cannot be empty")
	}
	if metadata == nil {
		metadata = make(map[string]any)
	}

	// Check if collection exists (and fetch it for dimension validation)
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

	// Upsert the document
	result, err := s.documentRepo.Upsert(ctx, documentId, collectionId, content, vector, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert document: %w", err)
	}

	return result, nil
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

// GetDocument returns a document with an id
func (s *DocumentService) GetDocument(ctx context.Context, documentId string) (*models.Document, error) {
	// check for empty documentId
	if documentId == "" {
		return nil, errors.New("document_id is required")
	}

	// fetch document from repository
	document, err := s.documentRepo.GetById(ctx, documentId)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("document_id %s not found", documentId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch document: %w", err)
	}

	return document, nil
}

// DeleteDocument deletes a document with an id
func (s *DocumentService) DeleteDocument(ctx context.Context, documentId string) error {
	// check for empty documentId
	if documentId == "" {
		return errors.New("document_id is required")
	}

	// delete document from repository
	err := s.documentRepo.DeleteById(ctx, documentId)
	if err == sql.ErrNoRows {
		return fmt.Errorf("document_id %s not found", documentId)
	}
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// Search Defaults
const (
	defaultTopK = 10
	maxTopK     = 1000
)

// SearchDocuments performs vector similarity search
// Returns top-K most similar documents with similarity scores
func (s *DocumentService) SearchDocuments(ctx context.Context, collectionId string, queryVector []float32, topK int32, metadataFilter map[string]any, minThreshold float32) ([]db.SearchResult, error) {
	// Validate collection_id
	if collectionId == "" {
		return nil, errors.New("collection_id is required")
	}

	// Validate query vector
	if len(queryVector) == 0 {
		return nil, errors.New("query_vector cannot be empty")
	}

	// Check if collection exists and validate vector dimension
	collection, err := s.collectionRepo.ListById(ctx, collectionId)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("collection_id %s not found", collectionId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch collection: %w", err)
	}

	// Validate vector dimension matches collection
	if len(queryVector) != collection.VectorDimension {
		return nil, fmt.Errorf(
			"query vector dimension mismatch: collection expects %d dimensions, got %d",
			collection.VectorDimension,
			len(queryVector),
		)
	}

	// Validate query vector values (no NaN or Infinity)
	for i, val := range queryVector {
		if math.IsNaN(float64(val)) {
			return nil, fmt.Errorf("query vector contains NaN at index %d", i)
		}
		if math.IsInf(float64(val), 0) {
			return nil, fmt.Errorf("query vector contains Infinity at index %d", i)
		}
	}

	// Validate and normalize top_k
	if topK < 0 {
		return nil, errors.New("top_k cannot be negative")
	}
	if topK == 0 {
		topK = defaultTopK // Use default if not specified
	}
	if topK > maxTopK {
		return nil, fmt.Errorf("top_k cannot exceed %d", maxTopK)
	}

	// Validate min_threshold (must be between 0.0 and 1.0)
	if minThreshold < 0.0 || minThreshold > 1.0 {
		return nil, errors.New("min_threshold must be between 0.0 and 1.0")
	}

	// Initialize metadata filter if nil
	if metadataFilter == nil {
		metadataFilter = make(map[string]any)
	}

	// Perform search via repository
	results, err := s.documentRepo.Search(ctx, collectionId, queryVector, int(topK), metadataFilter, minThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	return results, nil
}
