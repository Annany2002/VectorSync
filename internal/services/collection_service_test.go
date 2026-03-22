package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Annany2002/vector-sync/internal/models"
)

// mockCollectionRepo implements CollectionRepository for testing.
type mockCollectionRepo struct {
	CreateFn     func(ctx context.Context, name string, vectorDimension int32, metadataSchema map[string]any, distanceMetric string) (*models.Collection, error)
	ListFn       func(ctx context.Context, limit, offset int) ([]models.Collection, error)
	ListByIdFn   func(ctx context.Context, collectionId string) (*models.Collection, error)
	DeleteByIdFn func(ctx context.Context, collectionId string) (int64, error)
}

func (m *mockCollectionRepo) Create(ctx context.Context, name string, vectorDimension int32, metadataSchema map[string]any, distanceMetric string) (*models.Collection, error) {
	return m.CreateFn(ctx, name, vectorDimension, metadataSchema, distanceMetric)
}

func (m *mockCollectionRepo) List(ctx context.Context, limit, offset int) ([]models.Collection, error) {
	return m.ListFn(ctx, limit, offset)
}

func (m *mockCollectionRepo) ListById(ctx context.Context, collectionId string) (*models.Collection, error) {
	return m.ListByIdFn(ctx, collectionId)
}

func (m *mockCollectionRepo) DeleteById(ctx context.Context, collectionId string) (int64, error) {
	return m.DeleteByIdFn(ctx, collectionId)
}

func TestCreateCollection(t *testing.T) {
	t.Run("empty name returns error", func(t *testing.T) {
		svc := NewCollectionService(&mockCollectionRepo{})
		_, err := svc.CreateCollection(context.Background(), "", 128, nil, "cosine")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "name is required") {
			t.Errorf("expected error containing %q, got %q", "name is required", err.Error())
		}
	})

	t.Run("zero dimension returns error", func(t *testing.T) {
		svc := NewCollectionService(&mockCollectionRepo{})
		_, err := svc.CreateCollection(context.Background(), "test", 0, nil, "cosine")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "must be greater than 0") {
			t.Errorf("expected error containing %q, got %q", "must be greater than 0", err.Error())
		}
	})

	t.Run("negative dimension returns error", func(t *testing.T) {
		svc := NewCollectionService(&mockCollectionRepo{})
		_, err := svc.CreateCollection(context.Background(), "test", -5, nil, "cosine")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "must be greater than 0") {
			t.Errorf("expected error containing %q, got %q", "must be greater than 0", err.Error())
		}
	})

	t.Run("invalid distance metric returns error", func(t *testing.T) {
		svc := NewCollectionService(&mockCollectionRepo{})
		_, err := svc.CreateCollection(context.Background(), "test", 128, nil, "manhattan")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid distance_metric") {
			t.Errorf("expected error containing %q, got %q", "invalid distance_metric", err.Error())
		}
	})

	t.Run("empty distance metric defaults to cosine", func(t *testing.T) {
		var capturedMetric string
		repo := &mockCollectionRepo{
			CreateFn: func(ctx context.Context, name string, vectorDimension int32, metadataSchema map[string]any, distanceMetric string) (*models.Collection, error) {
				capturedMetric = distanceMetric
				return &models.Collection{Name: name, DistanceMetric: distanceMetric}, nil
			},
		}
		svc := NewCollectionService(repo)
		_, err := svc.CreateCollection(context.Background(), "test", 128, nil, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedMetric != "cosine" {
			t.Errorf("expected distance metric %q, got %q", "cosine", capturedMetric)
		}
	})

	t.Run("duplicate name returns collection already exists", func(t *testing.T) {
		repo := &mockCollectionRepo{
			CreateFn: func(ctx context.Context, name string, vectorDimension int32, metadataSchema map[string]any, distanceMetric string) (*models.Collection, error) {
				return nil, errors.New("duplicate key value violates unique constraint")
			},
		}
		svc := NewCollectionService(repo)
		_, err := svc.CreateCollection(context.Background(), "existing", 128, nil, "cosine")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "collection already exists" {
			t.Errorf("expected error %q, got %q", "collection already exists", err.Error())
		}
	})

	t.Run("valid create returns collection", func(t *testing.T) {
		expected := &models.Collection{
			Id:              "col-123",
			Name:            "my-collection",
			VectorDimension: 256,
			DistanceMetric:  "euclidean",
		}
		repo := &mockCollectionRepo{
			CreateFn: func(ctx context.Context, name string, vectorDimension int32, metadataSchema map[string]any, distanceMetric string) (*models.Collection, error) {
				return expected, nil
			},
		}
		svc := NewCollectionService(repo)
		got, err := svc.CreateCollection(context.Background(), "my-collection", 256, nil, "euclidean")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Id != expected.Id {
			t.Errorf("expected Id %q, got %q", expected.Id, got.Id)
		}
		if got.Name != expected.Name {
			t.Errorf("expected Name %q, got %q", expected.Name, got.Name)
		}
	})
}

func TestListCollections(t *testing.T) {
	t.Run("limit zero defaults to 20", func(t *testing.T) {
		var capturedLimit int
		repo := &mockCollectionRepo{
			ListFn: func(ctx context.Context, limit, offset int) ([]models.Collection, error) {
				capturedLimit = limit
				return []models.Collection{}, nil
			},
		}
		svc := NewCollectionService(repo)
		_, err := svc.ListCollections(context.Background(), 0, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedLimit != 20 {
			t.Errorf("expected limit 20, got %d", capturedLimit)
		}
	})

	t.Run("limit over 100 caps to 100", func(t *testing.T) {
		var capturedLimit int
		repo := &mockCollectionRepo{
			ListFn: func(ctx context.Context, limit, offset int) ([]models.Collection, error) {
				capturedLimit = limit
				return []models.Collection{}, nil
			},
		}
		svc := NewCollectionService(repo)
		_, err := svc.ListCollections(context.Background(), 200, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedLimit != 100 {
			t.Errorf("expected limit 100, got %d", capturedLimit)
		}
	})

	t.Run("negative offset defaults to 0", func(t *testing.T) {
		var capturedOffset int
		repo := &mockCollectionRepo{
			ListFn: func(ctx context.Context, limit, offset int) ([]models.Collection, error) {
				capturedOffset = offset
				return []models.Collection{}, nil
			},
		}
		svc := NewCollectionService(repo)
		_, err := svc.ListCollections(context.Background(), 10, -5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedOffset != 0 {
			t.Errorf("expected offset 0, got %d", capturedOffset)
		}
	})
}

func TestListCollection(t *testing.T) {
	t.Run("empty id returns error", func(t *testing.T) {
		svc := NewCollectionService(&mockCollectionRepo{})
		_, err := svc.ListCollection(context.Background(), "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "cannot be empty") {
			t.Errorf("expected error containing %q, got %q", "cannot be empty", err.Error())
		}
	})
}

func TestDeleteCollection(t *testing.T) {
	t.Run("empty id returns error", func(t *testing.T) {
		svc := NewCollectionService(&mockCollectionRepo{})
		_, err := svc.DeleteCollection(context.Background(), "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "cannot be empty") {
			t.Errorf("expected error containing %q, got %q", "cannot be empty", err.Error())
		}
	})
}
