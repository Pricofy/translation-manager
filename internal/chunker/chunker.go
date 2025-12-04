// Package chunker provides text chunking by estimated token count.
package chunker

// DefaultMaxTokens is the default maximum tokens per chunk.
// With 384MB Lambda memory, ~3000 tokens is safe.
const DefaultMaxTokens = 3000

// EstimateTokens estimates the token count for a text.
// Uses a simple heuristic: ~4 characters per token for Latin languages.
func EstimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	// Rough estimate: 1 token ≈ 4 characters
	tokens := len(text) / 4
	if tokens == 0 {
		tokens = 1
	}
	return tokens
}

// ChunkByTokens splits texts into chunks that don't exceed maxTokens.
// Each text is kept whole - never split mid-text.
// Returns a slice of chunks, where each chunk is a slice of texts.
func ChunkByTokens(texts []string, maxTokens int) [][]string {
	if len(texts) == 0 {
		return nil
	}

	if maxTokens <= 0 {
		maxTokens = DefaultMaxTokens
	}

	var chunks [][]string
	var currentChunk []string
	currentTokens := 0

	for _, text := range texts {
		textTokens := EstimateTokens(text)

		// If a single text exceeds maxTokens, it gets its own chunk
		if textTokens > maxTokens {
			// Flush current chunk if not empty
			if len(currentChunk) > 0 {
				chunks = append(chunks, currentChunk)
				currentChunk = nil
				currentTokens = 0
			}
			// Add oversized text as its own chunk
			chunks = append(chunks, []string{text})
			continue
		}

		// If adding this text would exceed the limit, start a new chunk
		if currentTokens+textTokens > maxTokens && len(currentChunk) > 0 {
			chunks = append(chunks, currentChunk)
			currentChunk = nil
			currentTokens = 0
		}

		// Add text to current chunk
		currentChunk = append(currentChunk, text)
		currentTokens += textTokens
	}

	// Flush remaining chunk
	if len(currentChunk) > 0 {
		chunks = append(chunks, currentChunk)
	}

	return chunks
}
