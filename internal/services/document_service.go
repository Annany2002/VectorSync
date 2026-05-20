package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/Annany2002/vector-sync/internal/chunker"
	"github.com/Annany2002/vector-sync/internal/db"
	"github.com/Annany2002/vector-sync/internal/embedding"
	"github.com/Annany2002/vector-sync/internal/models"
)

// DocumentService is the service for document operations
type DocumentService struct {
	documentRepo    DocumentRepository
	collectionRepo  CollectionRepository
	collectionCache *db.CollectionCache
}

// NewDocumentService creates a new document service
func NewDocumentService(documentRepo DocumentRepository, collectionRepo CollectionRepository, collectionCache *db.CollectionCache) *DocumentService {
	return &DocumentService{
		documentRepo:    documentRepo,
		collectionRepo:  collectionRepo,
		collectionCache: collectionCache,
	}
}

// ChunkingConfig defines parameters for document ingestion and text splitting
type ChunkingConfig struct {
	Strategy     string
	ChunkSize    int
	ChunkOverlap int
}

// getCollectionInfo returns the vector dimension and distance metric for a collection
// under a single cache read-lock. On a cache miss it queries the DB once and
// populates both fields. This replaces the previous two-helper pattern that
// acquired the read-lock twice per request on the hot search path.
func (s *DocumentService) getCollectionInfo(ctx context.Context, collectionId string) (int, string, error) {
	if dim, metric, _, _, ok := s.collectionCache.GetInfo(collectionId); ok {
		return int(dim), metric, nil
	}

	// Cache miss: query DB and populate cache
	collection, err := s.collectionRepo.ListById(ctx, collectionId)
	if err == sql.ErrNoRows {
		return 0, "", fmt.Errorf("collection_id %s not found", collectionId)
	}
	if err != nil {
		return 0, "", fmt.Errorf("failed to fetch collection: %w", err)
	}

	s.collectionCache.Insert(collectionId, int32(collection.VectorDimension), collection.DistanceMetric, collection.EmbeddingProvider, collection.EmbeddingModel)
	return collection.VectorDimension, collection.DistanceMetric, nil
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

	// Validate collection exists and get dimension (uses cache)
	dimension, _, err := s.getCollectionInfo(ctx, collectionId)
	if err != nil {
		return nil, err
	}

	// Vector dimension matches collection's expected dimension
	if len(vector) != dimension {
		return nil, fmt.Errorf(
			"vector dimension mismatch: collection expects %d dimensions, got %d",
			dimension,
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

	// Validate collection exists and get dimension (uses cache)
	dimension, _, err := s.getCollectionInfo(ctx, collectionId)
	if err != nil {
		return nil, err
	}

	// Vector dimension matches collection's expected dimension
	if len(vector) != dimension {
		return nil, fmt.Errorf(
			"vector dimension mismatch: collection expects %d dimensions, got %d",
			dimension,
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
func (s *DocumentService) ListDocuments(ctx context.Context, collectionId string, limit, offset int32, includeVector bool) ([]models.Document, error) {
	// check for empty collectionId
	if collectionId == "" {
		return nil, errors.New("collection_id is required")
	}

	// Check if collection exists (uses cache to avoid DB round-trip)
	_, _, err := s.getCollectionInfo(ctx, collectionId)
	if err != nil {
		return nil, err
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
	documents, err := s.documentRepo.List(ctx, collectionId, int(limit), int(offset), includeVector)
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

// Batch size
const (
	batchSize = 1000
)

// SearchDocuments performs vector similarity search
// Returns top-K most similar documents with similarity scores
func (s *DocumentService) SearchDocuments(ctx context.Context, collectionId string, queryVector []float32, topK int32, metadataFilter map[string]any, minThreshold float32, includeVector bool) ([]db.SearchResult, error) {
	// Validate collection_id
	if collectionId == "" {
		return nil, errors.New("collection_id is required")
	}

	// Validate query vector
	if len(queryVector) == 0 {
		return nil, errors.New("query_vector cannot be empty")
	}

	// Validate collection exists; get dimension + distance metric in one cache lookup
	dimension, distanceMetric, err := s.getCollectionInfo(ctx, collectionId)
	if err != nil {
		return nil, err
	}

	// Validate vector dimension matches collection
	if len(queryVector) != dimension {
		return nil, fmt.Errorf(
			"query vector dimension mismatch: collection expects %d dimensions, got %d",
			dimension,
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
	results, err := s.documentRepo.Search(ctx, collectionId, queryVector, int(topK), metadataFilter, minThreshold, includeVector, distanceMetric)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	return results, nil
}

// FullTextSearchDocuments performs full-text search on document content
// Returns documents ranked by relevance, filtered by optional minRank threshold
func (s *DocumentService) FullTextSearchDocuments(ctx context.Context, collectionId, query string, limit int32, minRank float32, includeVector bool) ([]db.SearchResult, error) {
	// Validate collection_id
	if collectionId == "" {
		return nil, errors.New("collection_id is required")
	}

	// Validate query
	if query == "" {
		return nil, errors.New("query cannot be empty")
	}

	// Check if collection exists (uses cache)
	_, _, err := s.getCollectionInfo(ctx, collectionId)
	if err != nil {
		return nil, err
	}

	// Validate and normalize limit
	if limit < 0 {
		return nil, errors.New("limit cannot be negative")
	}
	if limit == 0 {
		limit = defaultLimit // Use default if not specified
	}

	// Perform search via repository
	results, err := s.documentRepo.FullTextSearch(ctx, collectionId, query, limit, minRank, includeVector)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	return results, nil
}

// BatchInsert inserts a list of documents inside a collection
// Returns a list of successfull inserted counts, with the docs and error
func (s *DocumentService) BatchInsert(ctx context.Context, collectionId string, documents []models.Document) (int, []models.Document, error) {
	// Validate collection_id
	if collectionId == "" {
		return 0, nil, errors.New("collection_id cannot be empty")
	}
	// Validate documents not empty
	if len(documents) == 0 {
		return 0, nil, errors.New("documents cannot be empty, size must be greater than zero")
	}

	// Check that the current batch size should be less than or equal to the limit
	if len(documents) > batchSize {
		return 0, nil, fmt.Errorf("batch size exceeds limit, got:%d, allowed:%d", len(documents), batchSize)
	}

	// Validate collection exists and get dimension (uses cache)
	dimension, _, err := s.getCollectionInfo(ctx, collectionId)
	if err != nil {
		return 0, nil, err
	}

	// Since this is an atomic insert, if vector length of any document does not match
	// the collection's vector dimension, then we reject the whole batch
	for _, v := range documents {
		if dimension != len(v.Vector) {
			return 0, nil, fmt.Errorf("mismatch between vector dimensions of document and collection, expected %d got %d", dimension, len(v.Vector))
		}
	}

	// Batch insert the documents
	insertCount, resultDocs, err := s.documentRepo.BatchInsert(ctx, collectionId, documents)
	if err != nil {
		return 0, nil, err
	}

	return insertCount, resultDocs, nil
}

// BatchDelete deletes a list of documents inside a collection
// Returns the deleted count, the deleted docs, and error
func (s *DocumentService) BatchDelete(ctx context.Context, collectionId string, documentIds []string) (int, []models.Document, error) {
	// Validate collection_id
	if collectionId == "" {
		return 0, nil, errors.New("collection_id cannot be empty")
	}
	// Validate documentIds not empty
	if len(documentIds) == 0 {
		return 0, nil, errors.New("document's ids cannot be empty, size must be greater than zero")
	}

	// Check that the current batch size should be less than
	// or equal to the batch size
	if len(documentIds) > batchSize {
		return 0, nil, fmt.Errorf("batch size exceeds limit, got:%d, allowed:%d", len(documentIds), batchSize)
	}

	// Check if collection exists (uses cache to avoid unnecessary DB round-trip)
	_, _, err := s.getCollectionInfo(ctx, collectionId)
	if err != nil {
		return 0, nil, err
	}

	// Batch delete the documents
	deletedCount, resultDocs, err := s.documentRepo.BatchDelete(ctx, collectionId, documentIds)
	if err != nil {
		return 0, nil, err
	}

	return deletedCount, resultDocs, nil
}

// HybridSearchDocuments performs combined vector similarity and full-text search
// Returns top-K results ordered by weighted combined score
func (s *DocumentService) HybridSearchDocuments(ctx context.Context, collectionId, queryText string, queryVector []float32, topK int32, metadataFilter map[string]any, vectorWeight, textWeight float32, includeVector bool) ([]db.SearchResult, error) {
	// Validate collection_id
	if collectionId == "" {
		return nil, errors.New("collection_id is required")
	}

	// Validate at least one search method is provided
	if len(queryVector) == 0 && queryText == "" {
		return nil, errors.New("at least one of query_vector or query_text is required")
	}

	// Validate collection exists; get dimension + distance metric in one cache lookup
	dimension, distanceMetric, err := s.getCollectionInfo(ctx, collectionId)
	if err != nil {
		return nil, err
	}

	// Validate vector dimension matches collection (if vector provided)
	if len(queryVector) > 0 && len(queryVector) != dimension {
		return nil, fmt.Errorf(
			"query vector dimension mismatch: collection expects %d dimensions, got %d",
			dimension,
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
		topK = defaultTopK
	}
	if topK > maxTopK {
		return nil, fmt.Errorf("top_k cannot exceed %d", maxTopK)
	}

	// Validate weights (must be between 0.0 and 1.0)
	if vectorWeight < 0.0 || vectorWeight > 1.0 {
		return nil, errors.New("vector_weight must be between 0.0 and 1.0")
	}
	if textWeight < 0.0 || textWeight > 1.0 {
		return nil, errors.New("text_weight must be between 0.0 and 1.0")
	}

	// Normalize weights to sum to 1.0
	if vectorWeight == 0 && textWeight == 0 {
		vectorWeight = 0.5
		textWeight = 0.5
	} else {
		sum := vectorWeight + textWeight
		if sum > 0 {
			vectorWeight = vectorWeight / sum
			textWeight = textWeight / sum
		}
	}

	// Initialize metadata filter if nil
	if metadataFilter == nil {
		metadataFilter = make(map[string]any)
	}

	// Perform hybrid search via repository
	results, err := s.documentRepo.HybridSearch(ctx, collectionId, queryText, queryVector, int(topK), metadataFilter, vectorWeight, textWeight, includeVector, distanceMetric)
	if err != nil {
		return nil, fmt.Errorf("failed to perform hybrid search: %w", err)
	}

	return results, nil
}

// IngestDocument ingests a raw document by chunking it, generating embeddings, and storing them
func (s *DocumentService) IngestDocument(ctx context.Context, collectionId, content string, metadata map[string]any, config ChunkingConfig) (int, []string, error) {
	if collectionId == "" {
		return 0, nil, errors.New("collection_id is required")
	}
	if content == "" {
		return 0, nil, errors.New("content is required")
	}
	if metadata == nil {
		metadata = make(map[string]any)
	}

	// 1. Retrieve the collection's embedding provider and model
	provider, model, err := s.getCollectionEmbeddingInfo(ctx, collectionId)
	if err != nil {
		return 0, nil, err
	}
	if provider == "" {
		return 0, nil, errors.New("collection is not configured for automatic embedding generation (no embedding_provider set)")
	}

	// 2. Perform chunking
	c := chunker.GetChunker(config.Strategy)
	chunks := c.Chunk(content, config.ChunkSize, config.ChunkOverlap)
	if len(chunks) == 0 {
		return 0, nil, nil
	}

	// 3. Generate embeddings
	client, err := embedding.GetClient(provider, model)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to initialize embedding client: %w", err)
	}

	embeddings, err := client.GenerateEmbeddings(ctx, chunks)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// 4. Validate embedding dimension
	dimension, _, err := s.getCollectionInfo(ctx, collectionId)
	if err != nil {
		return 0, nil, err
	}
	if len(embeddings) > 0 && len(embeddings[0]) != dimension {
		return 0, nil, fmt.Errorf("generated embedding dimension %d does not match collection expectation %d", len(embeddings[0]), dimension)
	}

	// 5. Prepare documents for BatchInsert
	docsToInsert := make([]models.Document, len(chunks))
	for i, chunk := range chunks {
		docsToInsert[i] = models.Document{
			CollectionId: collectionId,
			Content:      chunk,
			Vector:       embeddings[i],
			Metadata:     metadata,
		}
	}

	// 6. BatchInsert
	count, insertedDocs, err := s.BatchInsert(ctx, collectionId, docsToInsert)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to batch insert chunks: %w", err)
	}

	insertedIds := make([]string, len(insertedDocs))
	for i, doc := range insertedDocs {
		insertedIds[i] = doc.Id
	}

	return count, insertedIds, nil
}

func (s *DocumentService) getCollectionEmbeddingInfo(ctx context.Context, collectionId string) (string, string, error) {
	if _, _, provider, model, ok := s.collectionCache.GetInfo(collectionId); ok {
		return provider, model, nil
	}

	// Cache miss: query DB and populate cache
	collection, err := s.collectionRepo.ListById(ctx, collectionId)
	if err == sql.ErrNoRows {
		return "", "", fmt.Errorf("collection_id %s not found", collectionId)
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch collection: %w", err)
	}

	s.collectionCache.Insert(collectionId, int32(collection.VectorDimension), collection.DistanceMetric, collection.EmbeddingProvider, collection.EmbeddingModel)
	return collection.EmbeddingProvider, collection.EmbeddingModel, nil
}
