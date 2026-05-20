package embedding

import (
	"fmt"
	"strings"
)

// GetClient returns an embedding client based on the provider and model names
func GetClient(provider, model string) (Client, error) {
	switch strings.ToLower(provider) {
	case "openai":
		return NewOpenAIEmbeddingClient(model)
	case "ollama":
		return NewOllamaEmbeddingClient(model)
	case "cohere":
		return NewCohereEmbeddingClient(model)
	default:
		return nil, fmt.Errorf("unknown embedding provider: %s", provider)
	}
}
