package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type OllamaEmbeddingClient struct {
	host  string
	model string
	client *http.Client
}

type ollamaEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type ollamaEmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

type ollamaEmbeddingsRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaEmbeddingsResponse struct {
	Embedding []float32 `json:"embedding"`
}

func NewOllamaEmbeddingClient(model string) (*OllamaEmbeddingClient, error) {
	host := os.Getenv("OLLAMA_HOST")
	if host == "" {
		host = "http://localhost:11434"
	}
	// Clean up trailing slash
	host = strings.TrimSuffix(host, "/")

	return &OllamaEmbeddingClient{
		host:  host,
		model: model,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *OllamaEmbeddingClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	// First try batch endpoint /api/embed (with single text)
	embeddings, err := c.tryBatchEmbed(ctx, []string{text})
	if err == nil && len(embeddings) > 0 {
		return embeddings[0], nil
	}

	// Fallback to /api/embeddings for older Ollama
	return c.callLegacyEmbeddings(ctx, text)
}

func (c *OllamaEmbeddingClient) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// First try batch endpoint /api/embed
	embeddings, err := c.tryBatchEmbed(ctx, texts)
	if err == nil {
		return embeddings, nil
	}

	// Fallback to sequential /api/embeddings for older Ollama versions
	results := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := c.callLegacyEmbeddings(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding at index %d: %w", i, err)
		}
		results[i] = emb
	}
	return results, nil
}

func (c *OllamaEmbeddingClient) tryBatchEmbed(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := ollamaEmbedRequest{
		Model: c.model,
		Input: texts,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/embed", c.host)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

	var apiResp ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	if len(apiResp.Embeddings) != len(texts) {
		return nil, errors.New("mismatch in returned embedding count")
	}

	return apiResp.Embeddings, nil
}

func (c *OllamaEmbeddingClient) callLegacyEmbeddings(ctx context.Context, text string) ([]float32, error) {
	reqBody := ollamaEmbeddingsRequest{
		Model:  c.model,
		Prompt: text,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/embeddings", c.host)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status code from legacy embeddings: %d", resp.StatusCode)
	}

	var apiResp ollamaEmbeddingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	if len(apiResp.Embedding) == 0 {
		return nil, errors.New("empty embedding returned")
	}

	return apiResp.Embedding, nil
}
