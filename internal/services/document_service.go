package services

import (
	"github.com/Annany2002/vector-sync/internal/db"
)

// DocumentService is the service for collection operations
type DocumentService struct {
	repo db.DocumentRepo
}

// NewDocuementService creates a new collection service
func NewDocumentService(repo db.DocumentRepo) *DocumentService {
	return &DocumentService{repo: repo}
}
