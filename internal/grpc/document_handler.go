package grpc

import (
	"context"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Annany2002/vector-sync/api/proto/v1"
	"github.com/Annany2002/vector-sync/internal/models"
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

// CreateDocument creates a new document with a `collectionId`
func (h *DocumentHandler) CreateDocument(ctx context.Context, req *pb.CreateDocumentRequest) (*pb.CreateDocumentResponse, error) {
	// Extract the collectionId, content(if present), vector and the metadata
	collectionId := req.GetCollectionId()
	content := req.GetContent()
	vector := req.GetVector()

	// Convert protobuf map[string]*Struct to map[string]any
	var metadata map[string]any
	if req.GetMetadata() != nil {
		metadata = make(map[string]any)
		for key, value := range req.GetMetadata() {
			metadata[key] = value.AsMap()
		}
	}

	// Call service layer to create the document
	document, err := h.documentService.CreateDocument(ctx, collectionId, content, vector, metadata)
	if err != nil {
		return nil, err
	}

	// Build and return the gRPC response using helper function
	return &pb.CreateDocumentResponse{
		Document: convertToProtoDocument(document),
	}, nil
}

// convertToProtoDocument converts a models.Document to a pb.Document
func convertToProtoDocument(c *models.Document) *pb.Document {
	// Convert metadata schema from map[string]any to map[string]*structpb.Struct
	// This is needed because gRPC uses protobuf types, not Go native types
	metadataProto := make(map[string]*structpb.Struct)
	for key, value := range c.Metadata {
		// Type assertion: check if value is a map[string]any
		valueMap, ok := value.(map[string]any)
		if !ok {
			continue // Skip non-map values
		}
		// Convert Go map to protobuf Struct
		structValue, err := structpb.NewStruct(valueMap)
		if err != nil {
			continue // Skip on conversion error
		}
		metadataProto[key] = structValue
	}

	// Build and return the protobuf Collection message
	return &pb.Document{
		Id:           c.ID,
		CreatedAt:    timestamppb.New(c.CreatedAt),
		UpdatedAt:    timestamppb.New(c.UpdatedAt),
		CollectionId: c.CollectionID,
		Content:      c.Content,
		Vector:       []float32(c.Vector),
		Metadata:     metadataProto,
	}
}
