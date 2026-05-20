package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Annany2002/vector-sync/internal/embedding"
)

func TestOpenAIEmbeddingClient(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "sk-test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		// Mock response body structure matching the format expected by the client
		response := struct {
			Data []struct {
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			} `json:"data"`
		}{
			Data: []struct {
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Embedding: []float32{0.1, 0.2, 0.3},
					Index:     0,
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	os.Setenv("OPENAI_API_BASE", server.URL)
	defer os.Unsetenv("OPENAI_API_BASE")

	client, err := embedding.NewOpenAIEmbeddingClient("text-embedding-3-small")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	emb, err := client.GenerateEmbedding(context.Background(), "hello")
	if err != nil {
		t.Fatalf("failed to generate embedding: %v", err)
	}

	if len(emb) != 3 || emb[0] != 0.1 || emb[1] != 0.2 || emb[2] != 0.3 {
		t.Errorf("unexpected embedding values: %v", emb)
	}
}

func TestOllamaEmbeddingClient(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/embed" {
			response := struct {
				Embeddings [][]float32 `json:"embeddings"`
			}{
				Embeddings: [][]float32{
					{0.4, 0.5, 0.6},
				},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	os.Setenv("OLLAMA_HOST", server.URL)
	defer os.Unsetenv("OLLAMA_HOST")

	client, err := embedding.NewOllamaEmbeddingClient("nomic-embed-text")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	emb, err := client.GenerateEmbedding(context.Background(), "hello")
	if err != nil {
		t.Fatalf("failed to generate embedding: %v", err)
	}

	if len(emb) != 3 || emb[0] != 0.4 || emb[1] != 0.5 || emb[2] != 0.6 {
		t.Errorf("unexpected embedding values: %v", emb)
	}
}

func TestCohereEmbeddingClient(t *testing.T) {
	os.Setenv("COHERE_API_KEY", "co-test-key")
	defer os.Unsetenv("COHERE_API_KEY")

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := struct {
			Embeddings [][]float32 `json:"embeddings"`
		}{
			Embeddings: [][]float32{
				{0.7, 0.8, 0.9},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	os.Setenv("COHERE_API_BASE", server.URL)
	defer os.Unsetenv("COHERE_API_BASE")

	client, err := embedding.NewCohereEmbeddingClient("embed-english-v3.0")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	emb, err := client.GenerateEmbedding(context.Background(), "hello")
	if err != nil {
		t.Fatalf("failed to generate embedding: %v", err)
	}

	if len(emb) != 3 || emb[0] != 0.7 || emb[1] != 0.8 || emb[2] != 0.9 {
		t.Errorf("unexpected embedding values: %v", emb)
	}
}
