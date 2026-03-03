package grpc

import (
	"context"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Annany2002/vector-sync/api/proto/v1/generated"
	"github.com/Annany2002/vector-sync/internal/models"
	"github.com/Annany2002/vector-sync/internal/services"
)

// CollectionHandler implements the CollectionServiceServer interface
type CollectionHandler struct {
	collectionService *services.CollectionService
	pb.UnimplementedCollectionServiceServer
}

// NewCollectionHandler creates a new collection handler
func NewCollectionHandler(svc *services.CollectionService) *CollectionHandler {
	return &CollectionHandler{collectionService: svc}
}

// CreateCollection implements the CreateCollection method
func (h *CollectionHandler) CreateCollection(ctx context.Context, req *pb.CreateCollectionRequest) (*pb.CreateCollectionResponse, error) {
	// Extract data from gRPC request
	name := req.GetName()
	vectorDimension := req.GetVectorDimension()
	distanceMetric := req.GetDistanceMetric()

	// Convert protobuf map[string]*Struct to map[string]any
	var metadataSchema map[string]any
	if req.GetMetadataSchema() != nil {
		metadataSchema = make(map[string]any)
		for key, value := range req.GetMetadataSchema() {
			metadataSchema[key] = value.AsMap()
		}
	}

	// Call service layer to create the collection
	collection, err := h.collectionService.CreateCollection(ctx, name, vectorDimension, metadataSchema, distanceMetric)
	if err != nil {
		return nil, err
	}

	// Build and return the gRPC response using helper function
	return &pb.CreateCollectionResponse{
		Collection: convertToProtoCollection(collection),
	}, nil
}

// ListCollections retrieves the collections with pagination
func (h *CollectionHandler) ListCollections(ctx context.Context, req *pb.ListCollectionsRequest) (*pb.ListCollectionsResponse, error) {
	// Extract pagination parameters - service layer handles defaults and caps
	limit := int(req.GetLimit())
	offset := int(req.GetOffset())

	// Call the service layer for listing collections
	collections, err := h.collectionService.ListCollections(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	// Convert []models.Collection to []*pb.Collection
	pbCollections := make([]*pb.Collection, 0, len(collections))
	for _, c := range collections {
		pbCollections = append(pbCollections, convertToProtoCollection(&c))
	}

	// Return ListCollectionsResponse (plural) with repeated collections field
	return &pb.ListCollectionsResponse{Collections: pbCollections}, nil
}

// ListCollection retrieves a collection with an id
func (h *CollectionHandler) ListCollection(ctx context.Context, req *pb.ListCollectionRequest) (*pb.ListCollectionResponse, error) {
	// Extract the collectionId
	collectionId := req.GetId()

	// Call the service layer for listing collection
	collection, err := h.collectionService.ListCollection(ctx, collectionId)
	if err != nil {
		return nil, err
	}

	// Convert models.Collection to *pb.Collection
	pbCollection := convertToProtoCollection(collection)

	// Return ListCollectionResponse with id `collectionId`
	return &pb.ListCollectionResponse{Collection: pbCollection}, nil
}

// DeleteCollection retrieves a collection with a id
func (h *CollectionHandler) DeleteCollection(ctx context.Context, req *pb.DeleteCollectionRequest) (*pb.DeleteCollectionResponse, error) {
	// Extract the collectionId
	collectionId := req.GetId()

	// Call the service layer for collection deletion
	document_count, err := h.collectionService.DeleteCollection(ctx, collectionId)
	if err != nil {
		return nil, err
	}

	// Return ListCollectionResponse with id `collectionId`
	return &pb.DeleteCollectionResponse{Id: collectionId, DeletedAt: timestamppb.Now(), DocumentDeleted: document_count}, nil
}

// convertToProtoCollection converts a models.Collection to a pb.Collection
func convertToProtoCollection(c *models.Collection) *pb.Collection {
	// Convert metadata schema from map[string]any to map[string]*structpb.Struct
	// This is needed because gRPC uses protobuf types, not Go native types
	metadataSchemaProto := make(map[string]*structpb.Struct)
	for key, value := range c.MetadataSchema {
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
		metadataSchemaProto[key] = structValue
	}

	// Build and return the protobuf Collection message
	return &pb.Collection{
		Id:              c.Id,
		CreatedAt:       timestamppb.New(c.CreatedAt),
		UpdatedAt:       timestamppb.New(c.UpdatedAt),
		Name:            c.Name,
		VectorDimension: int32(c.VectorDimension),
		MetadataSchema:  metadataSchemaProto,
		DocumentCount:   c.DocumentCount,
		DistanceMetric:  c.DistanceMetric,
	}
}
