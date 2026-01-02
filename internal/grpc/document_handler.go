package grpc

import (
	pb "github.com/Annany2002/vector-sync/api/proto/v1"
	"github.com/Annany2002/vector-sync/internal/services"
)

// DocumentHandler implements the DocumentServiceServer interface
type DocumentHandler struct {
	documentService *services.DocumentService
	pb.UnimplementedDocumentServiceServer
}

// NewDocumentHandler creates a new collection handler
func NewDocumentHandler(svc *services.DocumentService) *DocumentHandler {
	return &DocumentHandler{documentService: svc}
}
