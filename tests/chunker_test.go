package tests

import (
	"testing"

	"github.com/Annany2002/vector-sync/internal/chunker"
)

func TestFixedSizeChunker(t *testing.T) {
	t.Run("chunk_size_3_overlap_0", func(t *testing.T) {
		c := chunker.GetChunker("fixed")
		chunks := c.Chunk("abcdefghij", 3, 0)
		t.Logf("Generated chunks: %q", chunks)
		expected := []string{"abc", "def", "ghi", "j"}
		if len(chunks) != len(expected) {
			t.Fatalf("expected %d chunks, got %d", len(expected), len(chunks))
		}
		for i, v := range expected {
			if chunks[i] != v {
				t.Errorf("chunk %d expected %q, got %q", i, v, chunks[i])
			}
		}
	})

	t.Run("chunk_size_4_overlap_2", func(t *testing.T) {
		c := chunker.GetChunker("fixed")
		chunks := c.Chunk("abcdefgh", 4, 2)
		expected := []string{"abcd", "cdef", "efgh"}
		if len(chunks) != len(expected) {
			t.Fatalf("expected %d chunks, got %d", len(expected), len(chunks))
		}
		for i, v := range expected {
			if chunks[i] != v {
				t.Errorf("chunk %d expected %q, got %q", i, v, chunks[i])
			}
		}
	})
}

func TestSentenceChunker(t *testing.T) {
	c := chunker.GetChunker("sentence")
	text := "This is the first sentence. And this is the second! Is there a third? Yes."
	chunks := c.Chunk(text, 20, 0)
	expected := []string{
		"This is the first sentence.",
		"And this is the second!",
		"Is there a third?",
		"Yes.",
	}
	if len(chunks) != len(expected) {
		t.Fatalf("expected %d chunks, got %d", len(expected), len(chunks))
	}
	for i, v := range expected {
		if chunks[i] != v {
			t.Errorf("chunk %d expected %q, got %q", i, v, chunks[i])
		}
	}
}

func TestRecursiveCharacterChunker(t *testing.T) {
	c := chunker.GetChunker("recursive")
	text := "This is paragraph one.\n\nThis is paragraph two, sentence one. Sentence two."
	chunks := c.Chunk(text, 40, 10)
	if len(chunks) == 0 {
		t.Fatal("expected chunks, got none")
	}
}

func TestGetChunker(t *testing.T) {
	if c := chunker.GetChunker("fixed"); c == nil {
		t.Error("expected fixed chunker")
	}
	if c := chunker.GetChunker("sentence"); c == nil {
		t.Error("expected sentence chunker")
	}
	if c := chunker.GetChunker("recursive"); c == nil {
		t.Error("expected recursive chunker")
	}
	// Default fallback
	if c := chunker.GetChunker("unknown"); c == nil {
		t.Error("expected default fallback chunker")
	}
}
