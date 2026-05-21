package chunker

import (
	"regexp"
	"strings"
)

// Chunker defines the interface for text chunking strategies
type Chunker interface {
	Chunk(text string, size, overlap int) []string
}

// FixedSizeChunker splits text into chunks of character size with overlap.
type FixedSizeChunker struct{}

func (c *FixedSizeChunker) Chunk(text string, size, overlap int) []string {
	if size <= 0 {
		size = 500
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= size {
		overlap = size / 2
	}

	var chunks []string
	textLen := len(text)
	if textLen == 0 {
		return chunks
	}

	start := 0
	for start < textLen {
		end := start + size
		if end > textLen {
			end = textLen
		}
		chunks = append(chunks, text[start:end])
		if end == textLen {
			break
		}
		start = end - overlap
	}
	return chunks
}

// SentenceChunker splits text into sentences and groups them.
type SentenceChunker struct{}

func (c *SentenceChunker) Chunk(text string, size, overlap int) []string {
	if size <= 0 {
		size = 500
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= size {
		overlap = size / 2
	}

	// Split by sentences using a regex that handles common punctuation followed by space or end-of-string
	re := regexp.MustCompile(`[^.!?]+([.!?]+(\s+|$)|$)`)
	matches := re.FindAllString(text, -1)
	if len(matches) == 0 {
		if len(text) > 0 {
			return []string{text}
		}
		return nil
	}

	var chunks []string
	var currentChunk strings.Builder

	for _, sentence := range matches {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		if currentChunk.Len()+len(sentence) > size && currentChunk.Len() > 0 {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()

			// Simple sentence overlap: we can carry over the last sentence if it is within overlap size
			// and is not the same sentence we're currently processing.
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(sentence)
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// RecursiveCharacterChunker splits text recursively using separators.
type RecursiveCharacterChunker struct {
	separators []string
}

func NewRecursiveCharacterChunker() *RecursiveCharacterChunker {
	return &RecursiveCharacterChunker{
		separators: []string{"\n\n", "\n", " ", ""},
	}
}

func (c *RecursiveCharacterChunker) Chunk(text string, size, overlap int) []string {
	if size <= 0 {
		size = 500
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= size {
		overlap = size / 2
	}

	return c.splitText(text, c.separators, size, overlap)
}

func (c *RecursiveCharacterChunker) splitText(text string, separators []string, size, overlap int) []string {
	// If text is already small enough, return it
	if len(text) <= size {
		return []string{text}
	}

	// Find the first separator that splits the text
	separator := ""
	var nextSeparators []string
	for i, sep := range separators {
		if strings.Contains(text, sep) {
			separator = sep
			nextSeparators = separators[i+1:]
			break
		}
	}

	// If no separators found, split by fixed size
	if separator == "" {
		fc := &FixedSizeChunker{}
		return fc.Chunk(text, size, overlap)
	}

	// Split text by separator
	parts := strings.Split(text, separator)
	var finalChunks []string
	var currentDocs []string

	for _, part := range parts {
		if len(part) > size {
			// If we have accumulated some docs, flush them first
			if len(currentDocs) > 0 {
				merged := c.mergeDocs(currentDocs, separator, size, overlap)
				finalChunks = append(finalChunks, merged...)
				currentDocs = nil
			}

			// Recursively split the large part using next separators
			recursiveChunks := c.splitText(part, nextSeparators, size, overlap)
			finalChunks = append(finalChunks, recursiveChunks...)
		} else {
			currentDocs = append(currentDocs, part)
		}
	}

	// Flush any remaining accumulated docs
	if len(currentDocs) > 0 {
		merged := c.mergeDocs(currentDocs, separator, size, overlap)
		finalChunks = append(finalChunks, merged...)
	}

	return finalChunks
}

func (c *RecursiveCharacterChunker) mergeDocs(docs []string, separator string, size, overlap int) []string {
	var merged []string
	var currentBlock []string
	currentLen := 0

	for _, doc := range docs {
		docLen := len(doc)
		// If adding this doc exceeds size
		if currentLen+docLen+len(separator) > size && len(currentBlock) > 0 {
			merged = append(merged, strings.Join(currentBlock, separator))

			// Retain elements for overlap
			// Keep removing from the front until the block size is within overlap
			for currentLen > overlap && len(currentBlock) > 0 {
				removed := currentBlock[0]
				currentBlock = currentBlock[1:]
				currentLen -= (len(removed) + len(separator))
			}
		}

		currentBlock = append(currentBlock, doc)
		if currentLen == 0 {
			currentLen = docLen
		} else {
			currentLen += docLen + len(separator)
		}
	}

	if len(currentBlock) > 0 {
		merged = append(merged, strings.Join(currentBlock, separator))
	}

	return merged
}

// GetChunker returns a chunker implementation based on the strategy name
func GetChunker(strategy string) Chunker {
	switch strings.ToLower(strategy) {
	case "fixed":
		return &FixedSizeChunker{}
	case "sentence":
		return &SentenceChunker{}
	case "recursive":
		fallthrough
	default:
		return NewRecursiveCharacterChunker()
	}
}
