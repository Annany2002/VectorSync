package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Annany2002/vector-sync/internal/db"
	"github.com/Annany2002/vector-sync/internal/models"
	"github.com/Annany2002/vector-sync/internal/services"
)

type mockCollectionRepo struct {
	ListByIdFn func(ctx context.Context, collectionId string) (*models.Collection, error)
}

func (m *mockCollectionRepo) Create(ctx context.Context, name string, vectorDimension int32, metadataSchema map[string]any, distanceMetric, provider, model string) (*models.Collection, error) {
	return nil, nil
}
func (m *mockCollectionRepo) List(ctx context.Context, limit, offset int) ([]models.Collection, error) {
	return nil, nil
}
func (m *mockCollectionRepo) ListById(ctx context.Context, collectionId string) (*models.Collection, error) {
	if m.ListByIdFn != nil {
		return m.ListByIdFn(ctx, collectionId)
	}
	return nil, nil
}
func (m *mockCollectionRepo) DeleteById(ctx context.Context, collectionId string) (int64, error) {
	return 0, nil
}

type mockDocumentRepo struct {
	BatchInsertFn func(ctx context.Context, collectionId string, documents []models.Document) (int, []models.Document, error)
}

func (m *mockDocumentRepo) Create(ctx context.Context, collectionId, content string, vector []float32, metadata map[string]any) (*models.Document, error) {
	return nil, nil
}
func (m *mockDocumentRepo) Upsert(ctx context.Context, documentId, collectionId, content string, vector []float32, metadata map[string]any) (*db.UpsertResult, error) {
	return nil, nil
}
func (m *mockDocumentRepo) List(ctx context.Context, collectionId string, limit, offset int, includeVector bool) ([]models.Document, error) {
	return nil, nil
}
func (m *mockDocumentRepo) GetById(ctx context.Context, documentId string) (*models.Document, error) {
	return nil, nil
}
func (m *mockDocumentRepo) DeleteById(ctx context.Context, documentId string) error {
	return nil
}
func (m *mockDocumentRepo) Search(ctx context.Context, collectionId string, queryVector []float32, topK int, metadataFilter map[string]any, minThreshold float32, includeVector bool, distanceMetric string) ([]db.SearchResult, error) {
	return nil, nil
}
func (m *mockDocumentRepo) FullTextSearch(ctx context.Context, collectionId, query string, limit int32, minRank float32, includeVector bool) ([]db.SearchResult, error) {
	return nil, nil
}
func (m *mockDocumentRepo) BatchInsert(ctx context.Context, collectionId string, documents []models.Document) (int, []models.Document, error) {
	if m.BatchInsertFn != nil {
		return m.BatchInsertFn(ctx, collectionId, documents)
	}
	return 0, nil, nil
}
func (m *mockDocumentRepo) BatchDelete(ctx context.Context, collectionId string, documentIds []string) (int, []models.Document, error) {
	return 0, nil, nil
}
func (m *mockDocumentRepo) HybridSearch(ctx context.Context, collectionId, queryText string, queryVector []float32, topK int, metadataFilter map[string]any, vectorWeight, textWeight float32, includeVector bool, distanceMetric string) ([]db.SearchResult, error) {
	return nil, nil
}

func TestIngestDocument(t *testing.T) {
	ctx := context.Background()

	// Spin up a mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := struct {
			Embeddings [][]float32 `json:"embeddings"`
		}{
			Embeddings: [][]float32{
				{0.1, 0.2, 0.3},
				{0.4, 0.5, 0.6},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	os.Setenv("OLLAMA_HOST", server.URL)
	defer os.Unsetenv("OLLAMA_HOST")

	colRepo := &mockCollectionRepo{
		ListByIdFn: func(ctx context.Context, collectionId string) (*models.Collection, error) {
			return &models.Collection{
				Id:                collectionId,
				VectorDimension:   3,
				DistanceMetric:    "cosine",
				EmbeddingProvider: "ollama",
				EmbeddingModel:    "nomic-embed-text",
			}, nil
		},
	}

	var capturedDocs []models.Document
	docRepo := &mockDocumentRepo{
		BatchInsertFn: func(ctx context.Context, collectionId string, documents []models.Document) (int, []models.Document, error) {
			capturedDocs = documents
			res := make([]models.Document, len(documents))
			for i, d := range documents {
				res[i] = d
				res[i].Id = fmt.Sprintf("doc-%d", i)
			}
			return len(res), res, nil
		},
	}

	cache := db.NewCollectionCache()
	cache.Insert("col-1", 3, "cosine", "ollama", "nomic-embed-text")
	svc := services.NewDocumentService(docRepo, colRepo, cache)

	count, ids, err := svc.IngestDocument(ctx, "col-1", "Sentence one. Sentence two.", nil, services.ChunkingConfig{
		Strategy:     "sentence",
		ChunkSize:    20,
		ChunkOverlap: 0,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
	if len(ids) != 2 || ids[0] != "doc-0" || ids[1] != "doc-1" {
		t.Errorf("unexpected document IDs: %v", ids)
	}

	if len(capturedDocs) != 2 {
		t.Fatalf("expected 2 documents inserted, got %d", len(capturedDocs))
	}
	if capturedDocs[0].Content != "Sentence one." || capturedDocs[1].Content != "Sentence two." {
		t.Errorf("unexpected content of inserted docs: %v", capturedDocs)
	}
}
