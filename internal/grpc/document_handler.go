package grpc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Annany2002/vector-sync/api/proto/v1/generated"
	"github.com/Annany2002/vector-sync/internal/models"
	"github.com/Annany2002/vector-sync/internal/services"
)

// DocumentHandler implements the DocumentService and CollectionService interface
type DocumentHandler struct {
	documentService   *services.DocumentService
	collectionService *services.CollectionService
	pb.UnimplementedDocumentServiceServer
}

// NewDocumentHandler creates a new collection handler
func NewDocumentHandler(doc_svc *services.DocumentService, col_svc *services.CollectionService) *DocumentHandler {
	return &DocumentHandler{documentService: doc_svc, collectionService: col_svc}
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

// UpsertDocument creates or updates a document with a specific ID
func (h *DocumentHandler) UpsertDocument(ctx context.Context, req *pb.UpsertDocumentRequest) (*pb.UpsertDocumentResponse, error) {
	// Extract the fields from request
	documentId := req.GetId()
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

	// Call service layer to upsert the document
	result, err := h.documentService.UpsertDocument(ctx, documentId, collectionId, content, vector, metadata)
	if err != nil {
		return nil, err
	}

	// Build and return the gRPC response
	return &pb.UpsertDocumentResponse{
		Document: convertToProtoDocument(result.Document),
		IsNew:    result.IsNew,
	}, nil
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

// SearchDocuments searches similar documents in collection using vector search
func (h *DocumentHandler) SearchDocuments(ctx context.Context, req *pb.SearchDocumentRequest) (*pb.SearchDocumentResponse, error) {
	// Extract the fields
	collectionId := req.GetCollectionId()
	queryVector := req.GetQueryVector()
	includeVector := req.GetIncludeVector()
	metadataFilter := req.GetMetadataFilter()
	minThreshold := req.GetMinThreshold()

	// basic validtions
	if collectionId == "" {
		return nil, errors.New("collection_id cannot be empty")
	}
	if len(queryVector) == 0 {
		return nil, errors.New("length of query vector must be greater than zero")
	}

	// check if collection exists or not
	collection, err := h.collectionService.ListCollection(ctx, collectionId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("collection with id %s not found", collectionId)
		}
		return nil, err
	}

	if collection.VectorDimension != len(queryVector) {
		return nil, errors.New("dimensions of query vectors and result vectors does not match")
	}

	// Extract top_k (default to 10 if not provided)
	topK := req.GetTopK()
	if topK == 0 {
		topK = 10
	}

	// Convert protobuf metadata filter to map[string]any
	var metadataFilterMap map[string]any
	if len(metadataFilter) > 0 {
		metadataFilterMap = make(map[string]any)
		for key, value := range metadataFilter {
			metadataFilterMap[key] = value.AsMap()
		}
	}

	// Call service layer to perform search
	searchResults, err := h.documentService.SearchDocuments(
		ctx,
		collectionId,
		queryVector,
		topK,
		metadataFilterMap,
		minThreshold,
	)
	if err != nil {
		return nil, err
	}

	// Convert db.SearchResult slice to pb.SearchResult slice
	pbResults := make([]*pb.SearchResult, len(searchResults))
	for i, result := range searchResults {
		// Convert document to proto format
		pbDoc := convertToProtoDocument(&result.Document)

		// Optionally exclude vector from response (for smaller payload size)
		if !includeVector {
			pbDoc.Vector = nil
		}

		pbResults[i] = &pb.SearchResult{
			Document: pbDoc,
			Score:    result.Score,
		}
	}

	return &pb.SearchDocumentResponse{
		Results: pbResults,
	}, nil
}

// FullTextSearch searches documents in a collection using PostgreSQL full-text search
func (h *DocumentHandler) FullTextSearch(ctx context.Context, req *pb.FullTextSearchRequest) (*pb.FullTextSearchResponse, error) {
	// Extract the fields
	collectionId := req.GetCollectionId()
	query := req.GetQuery()

	// Basic validations
	if collectionId == "" {
		return nil, errors.New("collection_id cannot be empty")
	}
	if query == "" {
		return nil, errors.New("query cannot be empty")
	}

	// check if collection exists or not
	_, err := h.collectionService.ListCollection(ctx, collectionId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("collection with id %s not found", collectionId)
		}
		return nil, err
	}

	// Extract limit (default: 10 if not specified)
	limit := req.GetLimit()
	if limit == 0 {
		limit = 10
	}

	// Validate rank
	minRank := req.GetMinRank()
	if minRank < 0.0 || minRank > 1.0 {
		return nil, fmt.Errorf("invalid value of %f for rank, should be between 0.0 and 1.0", minRank)
	}

	// Call service layer to perform search
	searchResults, err := h.documentService.FullTextSearchDocuments(
		ctx,
		collectionId,
		query,
		limit,
		minRank,
	)
	if err != nil {
		return nil, err
	}

	// Convert db.SearchResult slice to pb.SearchResult slice
	pbResults := make([]*pb.SearchResult, len(searchResults))
	for i, result := range searchResults {
		// Convert document to proto format
		pbDoc := convertToProtoDocument(&result.Document)

		pbResults[i] = &pb.SearchResult{
			Document: pbDoc,
			Score:    result.Score,
		}
	}

	return &pb.FullTextSearchResponse{
		Result: pbResults,
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
