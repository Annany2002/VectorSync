package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestOpenAIEmbeddingClient(t *testing.T) {
	// Set dummy API key
	os.Setenv("OPENAI_API_KEY", "sk-test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response := openaiResponse{
			Data: []openaiResponseData{
				{
					Embedding: []float32{0.1, 0.2, 0.3},
					Index:     0,
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewOpenAIEmbeddingClient("text-embedding-3-small")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Override client URL to point to mock server (we will hijack the http client transport to rewrite URL)
	client.client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = server.Listener.Addr().String()
		return http.DefaultTransport.RoundTrip(req)
	})

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
			response := ollamaEmbedResponse{
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

	client, err := NewOllamaEmbeddingClient("nomic-embed-text")
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
		response := cohereResponse{
			Embeddings: [][]float32{
				{0.7, 0.8, 0.9},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewCohereEmbeddingClient("embed-english-v3.0")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Hijack transport to rewrite URL
	client.client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Scheme = "http"
		req.URL.Host = server.Listener.Addr().String()
		return http.DefaultTransport.RoundTrip(req)
	})

	emb, err := client.GenerateEmbedding(context.Background(), "hello")
	if err != nil {
		t.Fatalf("failed to generate embedding: %v", err)
	}

	if len(emb) != 3 || emb[0] != 0.7 || emb[1] != 0.8 || emb[2] != 0.9 {
		t.Errorf("unexpected embedding values: %v", emb)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
