package services

import (
	"context"

	"github.com/Annany2002/vector-sync/internal/db"
	"github.com/Annany2002/vector-sync/internal/models"
)

// DocumentRepository defines the interface for document persistence operations.
// Concrete implementation: db.DocumentRepo
type DocumentRepository interface {
	Create(ctx context.Context, collectionId, content string, vector []float32, metadata map[string]any) (*models.Document, error)
	Upsert(ctx context.Context, documentId, collectionId, content string, vector []float32, metadata map[string]any) (*db.UpsertResult, error)
	List(ctx context.Context, collectionId string, limit, offset int, includeVector bool) ([]models.Document, error)
	GetById(ctx context.Context, documentId string) (*models.Document, error)
	DeleteById(ctx context.Context, documentId string) error
	Search(ctx context.Context, collectionId string, queryVector []float32, topK int, metadataFilter map[string]any, minThreshold float32, includeVector bool, distanceMetric string) ([]db.SearchResult, error)
	FullTextSearch(ctx context.Context, collectionId, query string, limit int32, minRank float32, includeVector bool) ([]db.SearchResult, error)
	BatchInsert(ctx context.Context, collectionId string, documents []models.Document) (int, []models.Document, error)
	BatchDelete(ctx context.Context, collectionId string, documentIds []string) (int, []models.Document, error)
	HybridSearch(ctx context.Context, collectionId, queryText string, queryVector []float32, topK int, metadataFilter map[string]any, vectorWeight, textWeight float32, includeVector bool, distanceMetric string) ([]db.SearchResult, error)
}

// CollectionRepository defines the interface for collection persistence operations.
// Concrete implementation: db.CollectionRepo
type CollectionRepository interface {
	Create(ctx context.Context, name string, vectorDimension int32, metadataSchema map[string]any, distanceMetric string) (*models.Collection, error)
	List(ctx context.Context, limit, offset int) ([]models.Collection, error)
	ListById(ctx context.Context, collectionId string) (*models.Collection, error)
	DeleteById(ctx context.Context, collectionId string) (int64, error)
}
