package chunker

import (
	"testing"
)

func TestChunkTexts(t *testing.T) {
	tests := []struct {
		name           string
		texts          []string
		maxTexts       int
		expectedChunks int
		expectedSizes  []int
	}{
		{
			name:           "empty input",
			texts:          []string{},
			maxTexts:       50,
			expectedChunks: 0,
			expectedSizes:  nil,
		},
		{
			name:           "nil input",
			texts:          nil,
			maxTexts:       50,
			expectedChunks: 0,
			expectedSizes:  nil,
		},
		{
			name:           "single text",
			texts:          []string{"Hello"},
			maxTexts:       50,
			expectedChunks: 1,
			expectedSizes:  []int{1},
		},
		{
			name:           "exactly max texts",
			texts:          makeTexts(50),
			maxTexts:       50,
			expectedChunks: 1,
			expectedSizes:  []int{50},
		},
		{
			name:           "one over max",
			texts:          makeTexts(51),
			maxTexts:       50,
			expectedChunks: 2,
			expectedSizes:  []int{50, 1},
		},
		{
			name:           "100 texts in 2 chunks",
			texts:          makeTexts(100),
			maxTexts:       50,
			expectedChunks: 2,
			expectedSizes:  []int{50, 50},
		},
		{
			name:           "150 texts in 3 chunks",
			texts:          makeTexts(150),
			maxTexts:       50,
			expectedChunks: 3,
			expectedSizes:  []int{50, 50, 50},
		},
		{
			name:           "uses default when maxTexts is 0",
			texts:          makeTexts(60),
			maxTexts:       0,
			expectedChunks: 2,
			expectedSizes:  []int{50, 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := ChunkTexts(tt.texts, tt.maxTexts)

			if len(chunks) != tt.expectedChunks {
				t.Errorf("ChunkTexts() returned %d chunks, want %d", len(chunks), tt.expectedChunks)
			}

			for i, expectedSize := range tt.expectedSizes {
				if i < len(chunks) && len(chunks[i]) != expectedSize {
					t.Errorf("chunk[%d] has %d texts, want %d", i, len(chunks[i]), expectedSize)
				}
			}

			// Verify all texts are preserved in order
			var allTexts []string
			for _, chunk := range chunks {
				allTexts = append(allTexts, chunk...)
			}

			if len(allTexts) != len(tt.texts) {
				t.Errorf("ChunkTexts() lost texts: got %d, want %d", len(allTexts), len(tt.texts))
			}

			for i, text := range tt.texts {
				if i < len(allTexts) && allTexts[i] != text {
					t.Errorf("ChunkTexts() text[%d] = %q, want %q", i, allTexts[i], text)
				}
			}
		})
	}
}

func TestChunkTexts_PreservesOrder(t *testing.T) {
	texts := []string{"first", "second", "third", "fourth", "fifth"}
	chunks := ChunkTexts(texts, 2)

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

// Helper to create N texts
func makeTexts(n int) []string {
	texts := make([]string, n)
	for i := range texts {
		texts[i] = "text"
	}
	return texts
}
