package embedding

import "context"

// Client defines the interface for generating vector embeddings from text
type Client interface {
	// GenerateEmbedding creates a single vector embedding for the input text
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)

	// GenerateEmbeddings creates multiple vector embeddings for the input texts
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}
