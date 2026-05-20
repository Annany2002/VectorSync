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

type OpenAIEmbeddingClient struct {
	apiKey string
	model  string
	client *http.Client
}

type openaiRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type openaiResponseData struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type openaiResponse struct {
	Data []openaiResponseData `json:"data"`
}

type openaiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func NewOpenAIEmbeddingClient(model string) (*OpenAIEmbeddingClient, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY environment variable is not set")
	}

	return &OpenAIEmbeddingClient{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *OpenAIEmbeddingClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := c.GenerateEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, errors.New("no embedding returned from OpenAI")
	}
	return embeddings[0], nil
}

func (c *OpenAIEmbeddingClient) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	reqBody := openaiRequest{
		Input: texts,
		Model: c.model,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	apiBase := os.Getenv("OPENAI_API_BASE")
	if apiBase == "" {
		apiBase = "https://api.openai.com"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiBase+"/v1/embeddings", bytes.NewReader(jsonBytes))
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
		var errResp openaiErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("openai API error: %s (status: %d)", errResp.Error.Message, resp.StatusCode)
		}
		return nil, fmt.Errorf("openai API request failed with status code %d", resp.StatusCode)
	}

	var apiResp openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	// OpenAI returns data elements that might not be in order (though they usually are).
	// We sort/place them by index to guarantee ordering matches the input.
	results := make([][]float32, len(texts))
	for _, item := range apiResp.Data {
		if item.Index >= 0 && item.Index < len(results) {
			results[item.Index] = item.Embedding
		}
	}

	// Verify all items are populated
	for i, emb := range results {
		if len(emb) == 0 {
			return nil, fmt.Errorf("missing embedding for input index %d", i)
		}
	}

	return results, nil
}
