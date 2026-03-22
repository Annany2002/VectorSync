package services

import (
	"context"
	"database/sql"
	"math"
	"strings"
	"testing"

	"github.com/Annany2002/vector-sync/internal/db"
	"github.com/Annany2002/vector-sync/internal/models"
)

// ---------------------------------------------------------------------------
// Mock document repository
// ---------------------------------------------------------------------------

type mockDocumentRepo struct {
	CreateFn         func(ctx context.Context, collectionId, content string, vector []float32, metadata map[string]any) (*models.Document, error)
	UpsertFn         func(ctx context.Context, documentId, collectionId, content string, vector []float32, metadata map[string]any) (*db.UpsertResult, error)
	ListFn           func(ctx context.Context, collectionId string, limit, offset int, includeVector bool) ([]models.Document, error)
	GetByIdFn        func(ctx context.Context, documentId string) (*models.Document, error)
	DeleteByIdFn     func(ctx context.Context, documentId string) error
	SearchFn         func(ctx context.Context, collectionId string, queryVector []float32, topK int, metadataFilter map[string]any, minThreshold float32, includeVector bool, distanceMetric string) ([]db.SearchResult, error)
	FullTextSearchFn func(ctx context.Context, collectionId, query string, limit int32, minRank float32, includeVector bool) ([]db.SearchResult, error)
	BatchInsertFn    func(ctx context.Context, collectionId string, documents []models.Document) (int, []models.Document, error)
	BatchDeleteFn    func(ctx context.Context, collectionId string, documentIds []string) (int, []models.Document, error)
	HybridSearchFn   func(ctx context.Context, collectionId, queryText string, queryVector []float32, topK int, metadataFilter map[string]any, vectorWeight, textWeight float32, includeVector bool, distanceMetric string) ([]db.SearchResult, error)
}

func (m *mockDocumentRepo) Create(ctx context.Context, collectionId, content string, vector []float32, metadata map[string]any) (*models.Document, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, collectionId, content, vector, metadata)
	}
	return nil, nil
}

func (m *mockDocumentRepo) Upsert(ctx context.Context, documentId, collectionId, content string, vector []float32, metadata map[string]any) (*db.UpsertResult, error) {
	if m.UpsertFn != nil {
		return m.UpsertFn(ctx, documentId, collectionId, content, vector, metadata)
	}
	return nil, nil
}

func (m *mockDocumentRepo) List(ctx context.Context, collectionId string, limit, offset int, includeVector bool) ([]models.Document, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, collectionId, limit, offset, includeVector)
	}
	return nil, nil
}

func (m *mockDocumentRepo) GetById(ctx context.Context, documentId string) (*models.Document, error) {
	if m.GetByIdFn != nil {
		return m.GetByIdFn(ctx, documentId)
	}
	return nil, nil
}

func (m *mockDocumentRepo) DeleteById(ctx context.Context, documentId string) error {
	if m.DeleteByIdFn != nil {
		return m.DeleteByIdFn(ctx, documentId)
	}
	return nil
}

func (m *mockDocumentRepo) Search(ctx context.Context, collectionId string, queryVector []float32, topK int, metadataFilter map[string]any, minThreshold float32, includeVector bool, distanceMetric string) ([]db.SearchResult, error) {
	if m.SearchFn != nil {
		return m.SearchFn(ctx, collectionId, queryVector, topK, metadataFilter, minThreshold, includeVector, distanceMetric)
	}
	return nil, nil
}

func (m *mockDocumentRepo) FullTextSearch(ctx context.Context, collectionId, query string, limit int32, minRank float32, includeVector bool) ([]db.SearchResult, error) {
	if m.FullTextSearchFn != nil {
		return m.FullTextSearchFn(ctx, collectionId, query, limit, minRank, includeVector)
	}
	return nil, nil
}

func (m *mockDocumentRepo) BatchInsert(ctx context.Context, collectionId string, documents []models.Document) (int, []models.Document, error) {
	if m.BatchInsertFn != nil {
		return m.BatchInsertFn(ctx, collectionId, documents)
	}
	return 0, nil, nil
}

func (m *mockDocumentRepo) BatchDelete(ctx context.Context, collectionId string, documentIds []string) (int, []models.Document, error) {
	if m.BatchDeleteFn != nil {
		return m.BatchDeleteFn(ctx, collectionId, documentIds)
	}
	return 0, nil, nil
}

func (m *mockDocumentRepo) HybridSearch(ctx context.Context, collectionId, queryText string, queryVector []float32, topK int, metadataFilter map[string]any, vectorWeight, textWeight float32, includeVector bool, distanceMetric string) ([]db.SearchResult, error) {
	if m.HybridSearchFn != nil {
		return m.HybridSearchFn(ctx, collectionId, queryText, queryVector, topK, metadataFilter, vectorWeight, textWeight, includeVector, distanceMetric)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newDocService builds a DocumentService with a pre-populated cache
// (collection "col-1" → dim=3, metric="cosine").
func newDocService(docRepo *mockDocumentRepo, colRepo *mockCollectionRepo) *DocumentService {
	cache := db.NewCollectionCache()
	cache.Insert("col-1", 3, "cosine")
	return NewDocumentService(docRepo, colRepo, cache)
}

// newDocServiceNoCache builds a DocumentService with an empty cache.
func newDocServiceNoCache(docRepo *mockDocumentRepo, colRepo *mockCollectionRepo) *DocumentService {
	cache := db.NewCollectionCache()
	return NewDocumentService(docRepo, colRepo, cache)
}

func requireErr(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", substr)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("expected error containing %q, got %q", substr, err.Error())
	}
}

func requireNilErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CreateDocument
// ---------------------------------------------------------------------------

func TestCreateDocument(t *testing.T) {
	ctx := context.Background()

	t.Run("empty collection_id", func(t *testing.T) {
		svc := newDocService(&mockDocumentRepo{}, &mockCollectionRepo{})
		_, err := svc.CreateDocument(ctx, "", "hello", []float32{1, 2, 3}, nil)
		requireErr(t, err, "collection_id")
	})

	t.Run("empty vector", func(t *testing.T) {
		svc := newDocService(&mockDocumentRepo{}, &mockCollectionRepo{})
		_, err := svc.CreateDocument(ctx, "col-1", "hello", []float32{}, nil)
		requireErr(t, err, "vector cannot be empty")
	})

	t.Run("collection not found on cache miss", func(t *testing.T) {
		colRepo := &mockCollectionRepo{
			ListByIdFn: func(_ context.Context, _ string) (*models.Collection, error) {
				return nil, sql.ErrNoRows
			},
		}
		svc := newDocServiceNoCache(&mockDocumentRepo{}, colRepo)
		_, err := svc.CreateDocument(ctx, "missing-col", "hello", []float32{1, 2, 3}, nil)
		requireErr(t, err, "not found")
	})

	t.Run("dimension mismatch", func(t *testing.T) {
		// cache has col-1 with dim=3; send a 4-element vector
		svc := newDocService(&mockDocumentRepo{}, &mockCollectionRepo{})
		_, err := svc.CreateDocument(ctx, "col-1", "hello", []float32{1, 2, 3, 4}, nil)
		requireErr(t, err, "dimension mismatch")
	})

	t.Run("NaN in vector", func(t *testing.T) {
		svc := newDocService(&mockDocumentRepo{}, &mockCollectionRepo{})
		vec := []float32{1, float32(math.NaN()), 3}
		_, err := svc.CreateDocument(ctx, "col-1", "hello", vec, nil)
		requireErr(t, err, "NaN")
	})

	t.Run("Inf in vector", func(t *testing.T) {
		svc := newDocService(&mockDocumentRepo{}, &mockCollectionRepo{})
		vec := []float32{1, float32(math.Inf(1)), 3}
		_, err := svc.CreateDocument(ctx, "col-1", "hello", vec, nil)
		requireErr(t, err, "Infinity")
	})

	t.Run("valid create returns document", func(t *testing.T) {
		docRepo := &mockDocumentRepo{
			CreateFn: func(_ context.Context, colId, content string, vector []float32, metadata map[string]any) (*models.Document, error) {
				return &models.Document{
					Id:           "doc-1",
					CollectionId: colId,
					Content:      content,
					Vector:       vector,
					Metadata:     metadata,
				}, nil
			},
		}
		svc := newDocService(docRepo, &mockCollectionRepo{})
		doc, err := svc.CreateDocument(ctx, "col-1", "hello", []float32{1, 2, 3}, nil)
		requireNilErr(t, err)
		if doc == nil {
			t.Fatal("expected non-nil document")
		}
		if doc.Id != "doc-1" {
			t.Errorf("expected doc id %q, got %q", "doc-1", doc.Id)
		}
		if doc.CollectionId != "col-1" {
			t.Errorf("expected collection_id %q, got %q", "col-1", doc.CollectionId)
		}
	})
}

// ---------------------------------------------------------------------------
// SearchDocuments
// ---------------------------------------------------------------------------

func TestSearchDocuments(t *testing.T) {
	ctx := context.Background()

	t.Run("negative top_k", func(t *testing.T) {
		svc := newDocService(&mockDocumentRepo{}, &mockCollectionRepo{})
		_, err := svc.SearchDocuments(ctx, "col-1", []float32{1, 2, 3}, -1, nil, 0, false)
		requireErr(t, err, "top_k cannot be negative")
	})

	t.Run("top_k exceeding max", func(t *testing.T) {
		svc := newDocService(&mockDocumentRepo{}, &mockCollectionRepo{})
		_, err := svc.SearchDocuments(ctx, "col-1", []float32{1, 2, 3}, 1001, nil, 0, false)
		requireErr(t, err, "cannot exceed")
	})

	t.Run("invalid min_threshold", func(t *testing.T) {
		svc := newDocService(&mockDocumentRepo{}, &mockCollectionRepo{})
		_, err := svc.SearchDocuments(ctx, "col-1", []float32{1, 2, 3}, 5, nil, 1.5, false)
		requireErr(t, err, "between 0.0 and 1.0")
	})

	t.Run("top_k zero uses default of 10", func(t *testing.T) {
		var capturedTopK int
		docRepo := &mockDocumentRepo{
			SearchFn: func(_ context.Context, _ string, _ []float32, topK int, _ map[string]any, _ float32, _ bool, _ string) ([]db.SearchResult, error) {
				capturedTopK = topK
				return nil, nil
			},
		}
		svc := newDocService(docRepo, &mockCollectionRepo{})
		_, err := svc.SearchDocuments(ctx, "col-1", []float32{1, 2, 3}, 0, nil, 0, false)
		requireNilErr(t, err)
		if capturedTopK != 10 {
			t.Errorf("expected default top_k=10, got %d", capturedTopK)
		}
	})
}

// ---------------------------------------------------------------------------
// BatchInsert
// ---------------------------------------------------------------------------

func TestBatchInsert(t *testing.T) {
	ctx := context.Background()

	t.Run("empty documents", func(t *testing.T) {
		svc := newDocService(&mockDocumentRepo{}, &mockCollectionRepo{})
		_, _, err := svc.BatchInsert(ctx, "col-1", nil)
		requireErr(t, err, "cannot be empty")
	})

	t.Run("exceeds batch size", func(t *testing.T) {
		docs := make([]models.Document, 1001)
		for i := range docs {
			docs[i] = models.Document{Vector: []float32{1, 2, 3}}
		}
		svc := newDocService(&mockDocumentRepo{}, &mockCollectionRepo{})
		_, _, err := svc.BatchInsert(ctx, "col-1", docs)
		requireErr(t, err, "exceeds limit")
	})

	t.Run("dimension mismatch in batch", func(t *testing.T) {
		docs := []models.Document{
			{Vector: []float32{1, 2, 3}},
			{Vector: []float32{1, 2}}, // wrong dimension
		}
		svc := newDocService(&mockDocumentRepo{}, &mockCollectionRepo{})
		_, _, err := svc.BatchInsert(ctx, "col-1", docs)
		requireErr(t, err, "mismatch")
	})
}

// ---------------------------------------------------------------------------
// HybridSearchDocuments
// ---------------------------------------------------------------------------

func TestHybridSearchDocuments(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid vector_weight", func(t *testing.T) {
		svc := newDocService(&mockDocumentRepo{}, &mockCollectionRepo{})
		_, err := svc.HybridSearchDocuments(ctx, "col-1", "query", []float32{1, 2, 3}, 5, nil, 1.5, 0.5, false)
		requireErr(t, err, "between 0.0 and 1.0")
	})

	t.Run("zero weights normalize to 0.5 0.5", func(t *testing.T) {
		var capturedVW, capturedTW float32
		docRepo := &mockDocumentRepo{
			HybridSearchFn: func(_ context.Context, _, _ string, _ []float32, _ int, _ map[string]any, vw, tw float32, _ bool, _ string) ([]db.SearchResult, error) {
				capturedVW = vw
				capturedTW = tw
				return nil, nil
			},
		}
		svc := newDocService(docRepo, &mockCollectionRepo{})
		_, err := svc.HybridSearchDocuments(ctx, "col-1", "query", []float32{1, 2, 3}, 5, nil, 0, 0, false)
		requireNilErr(t, err)
		if capturedVW != 0.5 {
			t.Errorf("expected vector_weight=0.5, got %f", capturedVW)
		}
		if capturedTW != 0.5 {
			t.Errorf("expected text_weight=0.5, got %f", capturedTW)
		}
	})
}
