// Package chunker provides text chunking for translation batches.
package chunker

// DefaultMaxTextsPerChunk limits texts per chunk.
// 50 texts is optimal for 512MB Lambda with CTranslate2 beam search.
const DefaultMaxTextsPerChunk = 50

// ChunkTexts splits texts into chunks of maxTexts each.
// Each chunk will have at most maxTexts texts.
// Returns a slice of chunks, where each chunk is a slice of texts.
func ChunkTexts(texts []string, maxTexts int) [][]string {
	if len(texts) == 0 {
		return nil
	}

	if maxTexts <= 0 {
		maxTexts = DefaultMaxTextsPerChunk
	}

	var chunks [][]string

	for i := 0; i < len(texts); i += maxTexts {
		end := i + maxTexts
		if end > len(texts) {
			end = len(texts)
		}
		chunks = append(chunks, texts[i:end])
	}

	return chunks
}
