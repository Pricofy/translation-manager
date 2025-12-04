package chunker

import (
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "empty string",
			text:     "",
			expected: 0,
		},
		{
			name:     "short text",
			text:     "Hi",
			expected: 1, // 2/4 = 0, min 1
		},
		{
			name:     "typical title",
			text:     "iPhone 12 Pro en buen estado",
			expected: 7, // 28/4 = 7
		},
		{
			name:     "long description",
			text:     "Este es un artículo de alta calidad con muchas características increíbles",
			expected: 19, // 75/4 = 18.75, rounded to 19
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateTokens(tt.text)
			if result != tt.expected {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tt.text, result, tt.expected)
			}
		})
	}
}

func TestChunkByTokens(t *testing.T) {
	tests := []struct {
		name           string
		texts          []string
		maxTokens      int
		expectedChunks int
	}{
		{
			name:           "empty input",
			texts:          []string{},
			maxTokens:      100,
			expectedChunks: 0,
		},
		{
			name:           "nil input",
			texts:          nil,
			maxTokens:      100,
			expectedChunks: 0,
		},
		{
			name:           "single text fits",
			texts:          []string{"Hello world"},
			maxTokens:      100,
			expectedChunks: 1,
		},
		{
			name:           "multiple texts fit in one chunk",
			texts:          []string{"Hello", "World", "Test"},
			maxTokens:      100,
			expectedChunks: 1,
		},
		{
			name: "texts split into multiple chunks",
			texts: []string{
				strings.Repeat("a", 40), // 10 tokens
				strings.Repeat("b", 40), // 10 tokens
				strings.Repeat("c", 40), // 10 tokens
			},
			maxTokens:      15, // Each text is 10 tokens, so 3 chunks
			expectedChunks: 3,
		},
		{
			name: "each text in own chunk",
			texts: []string{
				strings.Repeat("a", 40), // 10 tokens
				strings.Repeat("b", 40), // 10 tokens
			},
			maxTokens:      10, // Exactly fits one
			expectedChunks: 2,
		},
		{
			name: "oversized text gets own chunk",
			texts: []string{
				"small",
				strings.Repeat("x", 200), // 50 tokens, exceeds max
				"another",
			},
			maxTokens:      20,
			expectedChunks: 3, // small+another could fit, but oversized breaks it
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := ChunkByTokens(tt.texts, tt.maxTokens)

			if len(chunks) != tt.expectedChunks {
				t.Errorf("ChunkByTokens() returned %d chunks, want %d", len(chunks), tt.expectedChunks)
			}

			// Verify all texts are preserved
			var allTexts []string
			for _, chunk := range chunks {
				allTexts = append(allTexts, chunk...)
			}

			if len(allTexts) != len(tt.texts) {
				t.Errorf("ChunkByTokens() lost texts: got %d, want %d", len(allTexts), len(tt.texts))
			}

			for i, text := range tt.texts {
				if i < len(allTexts) && allTexts[i] != text {
					t.Errorf("ChunkByTokens() text[%d] = %q, want %q", i, allTexts[i], text)
				}
			}
		})
	}
}

func TestChunkByTokens_PreservesOrder(t *testing.T) {
	texts := []string{"first", "second", "third", "fourth", "fifth"}
	chunks := ChunkByTokens(texts, 10)

	var result []string
	for _, chunk := range chunks {
		result = append(result, chunk...)
	}

	for i, text := range texts {
		if result[i] != text {
			t.Errorf("Order not preserved: got %q at position %d, want %q", result[i], i, text)
		}
	}
}

func TestChunkByTokens_DefaultMaxTokens(t *testing.T) {
	texts := []string{"test"}
	chunks := ChunkByTokens(texts, 0) // Should use default

	if len(chunks) != 1 {
		t.Errorf("ChunkByTokens with 0 maxTokens should use default, got %d chunks", len(chunks))
	}
}
