package chunker

import (
	"strings"
	"testing"
)

func TestFixedSizeChunker(t *testing.T) {
	chunker := &FixedSizeChunker{}
	text := "abcdefghij" // 10 chars

	t.Run("chunk size 3 overlap 0", func(t *testing.T) {
		chunks := chunker.Chunk(text, 3, 0)
		t.Logf("Generated chunks: %q", chunks)
		expected := []string{"abc", "def", "ghi", "j"}
		if len(chunks) != len(expected) {
			t.Fatalf("expected %d chunks, got %d", len(expected), len(chunks))
		}
		for i, v := range chunks {
			if v != expected[i] {
				t.Errorf("expected chunk %d to be %q, got %q", i, expected[i], v)
			}
		}
	})

	t.Run("chunk size 4 overlap 2", func(t *testing.T) {
		chunks := chunker.Chunk(text, 4, 2)
		expected := []string{"abcd", "cdef", "efgh", "ghij"}
		if len(chunks) != len(expected) {
			t.Fatalf("expected %d chunks, got %d: %v", len(expected), len(chunks), chunks)
		}
		for i, v := range chunks {
			if v != expected[i] {
				t.Errorf("expected chunk %d to be %q, got %q", i, expected[i], v)
			}
		}
	})
}

func TestSentenceChunker(t *testing.T) {
	chunker := &SentenceChunker{}
	text := "This is sentence one. Sentence two! And three?"

	chunks := chunker.Chunk(text, 30, 0)
	if len(chunks) == 0 {
		t.Fatal("expected chunks, got none")
	}

	for _, chunk := range chunks {
		if !strings.Contains(chunk, "sentence one") && !strings.Contains(chunk, "Sentence two") && !strings.Contains(chunk, "three") {
			t.Errorf("unexpected chunk content: %q", chunk)
		}
	}
}

func TestRecursiveCharacterChunker(t *testing.T) {
	chunker := NewRecursiveCharacterChunker()
	text := "Paragraph one.\n\nParagraph two.\nParagraph two line two.\nParagraph two line three."

	chunks := chunker.Chunk(text, 50, 10)
	if len(chunks) == 0 {
		t.Fatal("expected chunks, got none")
	}

	// Verify recursive splitting results in chunks smaller than/equal to size limit
	for i, chunk := range chunks {
		if len(chunk) > 50 {
			t.Errorf("chunk %d has length %d which exceeds limit 50: %q", i, len(chunk), chunk)
		}
	}
}

func TestGetChunker(t *testing.T) {
	if _, ok := GetChunker("fixed").(*FixedSizeChunker); !ok {
		t.Error("expected FixedSizeChunker")
	}
	if _, ok := GetChunker("sentence").(*SentenceChunker); !ok {
		t.Error("expected SentenceChunker")
	}
	if _, ok := GetChunker("recursive").(*RecursiveCharacterChunker); !ok {
		t.Error("expected RecursiveCharacterChunker")
	}
	if _, ok := GetChunker("unknown").(*RecursiveCharacterChunker); !ok {
		t.Error("expected fallback to RecursiveCharacterChunker")
	}
}
