package grpc

import (
	"context"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/Annany2002/vector-sync/api/proto/v1"
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

	// Convert protobuf map[string]*Struct to map[string]any
	var metadataSchema map[string]any
	if req.GetMetadataSchema() != nil {
		metadataSchema = make(map[string]any)
		for key, value := range req.GetMetadataSchema() {
			metadataSchema[key] = value.AsMap()
		}
	}

	// Call service layer to create the collection
	collection, err := h.collectionService.CreateCollection(ctx, name, vectorDimension, metadataSchema)
	if err != nil {
		return nil, err
	}

	// Convert the metadata schema back to protobuf map[string]*Struct
	metadataSchemaProto := make(map[string]*structpb.Struct)
	for key, value := range collection.MetadataSchema {
		valueMap, ok := value.(map[string]any)
		if !ok {
			continue
		}
		structValue, err := structpb.NewStruct(valueMap)
		if err != nil {
			continue
		}
		metadataSchemaProto[key] = structValue
	}

	// Build and return the gRPC response
	return &pb.CreateCollectionResponse{
		Collection: &pb.Collection{
			Common: &pb.Common{
				Id:        collection.ID,
				CreatedAt: timestamppb.New(collection.CreatedAt),
				UpdatedAt: timestamppb.New(collection.UpdatedAt),
			},
			Name:            collection.Name,
			VectorDimension: int32(collection.VectorDimension),
			MetadataSchema:  metadataSchemaProto,
		},
	}, nil
}
