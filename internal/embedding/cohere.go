package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

type CohereEmbeddingClient struct {
	apiKey string
	model  string
	client *http.Client
}

type cohereRequest struct {
	Texts     []string `json:"texts"`
	Model     string   `json:"model"`
	InputType string   `json:"input_type"` // e.g. "search_document"
}

type cohereResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

type cohereErrorResponse struct {
	Message string `json:"message"`
}

func NewCohereEmbeddingClient(model string) (*CohereEmbeddingClient, error) {
	apiKey := os.Getenv("COHERE_API_KEY")
	if apiKey == "" {
		return nil, errors.New("COHERE_API_KEY environment variable is not set")
	}

	return &CohereEmbeddingClient{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *CohereEmbeddingClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := c.GenerateEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, errors.New("no embedding returned from Cohere")
	}
	return embeddings[0], nil
}

func (c *CohereEmbeddingClient) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	reqBody := cohereRequest{
		Texts:     texts,
		Model:     c.model,
		InputType: "search_document",
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.cohere.com/v1/embed", bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp cohereErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Message != "" {
			return nil, fmt.Errorf("cohere API error: %s (status: %d)", errResp.Message, resp.StatusCode)
		}
		return nil, fmt.Errorf("cohere API request failed with status code %d", resp.StatusCode)
	}

	var apiResp cohereResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if len(apiResp.Embeddings) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(apiResp.Embeddings))
	}

	return apiResp.Embeddings, nil
}
