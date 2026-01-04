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
		Id:           c.Id,
		CreatedAt:    timestamppb.New(c.CreatedAt),
		UpdatedAt:    timestamppb.New(c.UpdatedAt),
		CollectionId: c.CollectionId,
		Content:      c.Content,
		Vector:       []float32(c.Vector),
		Metadata:     metadataProto,
	}
}

// ListDocuments retrieves documents from a collection with pagination
func (h *DocumentHandler) ListDocuments(ctx context.Context, req *pb.ListDocumentsRequest) (*pb.ListDocumentsResponse, error) {
	// Extract parameters from request
	collectionId := req.GetCollectionId()
	limit := req.GetLimit()
	offset := req.GetOffset()

	// Call service layer to fetch documents
	documents, err := h.documentService.ListDocuments(ctx, collectionId, limit, offset)
	if err != nil {
		return nil, err
	}

	// Convert models.Document slice to pb.Document slice
	pbDocuments := make([]*pb.Document, len(documents))
	for i, doc := range documents {
		pbDocuments[i] = convertToProtoDocument(&doc)
	}

	return &pb.ListDocumentsResponse{
		Documents: pbDocuments,
	}, nil
}

// ListDocument retrieves a document with an id
func (h *DocumentHandler) ListDocument(ctx context.Context, req *pb.ListDocumentRequest) (*pb.ListDocumentResponse, error) {
	// Extract the documentId
	documentId := req.GetId()

	// Call service layer to fetch document
	document, err := h.documentService.GetDocument(ctx, documentId)
	if err != nil {
		return nil, err
	}

	// Convert to proto and return
	return &pb.ListDocumentResponse{
		Document: convertToProtoDocument(document),
	}, nil
}

// DeleteDocument deletes a document with an id
func (h *DocumentHandler) DeleteDocument(ctx context.Context, req *pb.DeleteDocumentRequest) (*pb.DeleteDocumentResponse, error) {
	// Extract the documentId
	documentId := req.GetId()

	// Call service layer to delete document
	err := h.documentService.DeleteDocument(ctx, documentId)
	if err != nil {
		return nil, err
	}

	// Return success response with id and timestamp
	return &pb.DeleteDocumentResponse{
		Id:        documentId,
		DeletedAt: timestamppb.Now(),
	}, nil
}
