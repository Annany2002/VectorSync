package grpc

import (
	"context"

	client "github.com/Annany2002/vector-sync/api/proto/v1"
)

type DocumentHandler struct {
	document client.DocumentServiceClient
}

func (d *DocumentHandler) CreateDocument(ctx context.Context, req *client.CreateDocumentRequest) (*client.CreateDocumentResponse, error) {
	// TODO: Implement document creation
	return nil, nil
}
