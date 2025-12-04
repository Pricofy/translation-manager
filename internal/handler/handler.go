// Package handler provides the Lambda handler for the translation manager.
package handler

import (
	"context"
	"fmt"

	"github.com/pricofy/translation-manager/internal/chunker"
	"github.com/pricofy/translation-manager/internal/router"
)

// Request is the input to the translation manager.
type Request struct {
	Texts      []string `json:"texts"`
	SourceLang string   `json:"sourceLang"`
	TargetLang string   `json:"targetLang"`
}

// Response is the output from the translation manager.
type Response struct {
	Translations    []string `json:"translations,omitempty"`
	ChunksProcessed int      `json:"chunksProcessed,omitempty"`
	Error           string   `json:"error,omitempty"`
}

// Handle processes a translation request.
// It chunks the input texts and sends ALL chunks in a single Lambda invocation.
// The translator Lambda processes each chunk sequentially internally.
func Handle(ctx context.Context, req Request) (*Response, error) {
	// Validate request
	if err := validateRequest(req); err != nil {
		return &Response{Error: err.Error()}, nil
	}

	// Empty input - return immediately
	if len(req.Texts) == 0 {
		return &Response{Translations: []string{}, ChunksProcessed: 0}, nil
	}

	// Create router
	r, err := router.New(ctx)
	if err != nil {
		return &Response{Error: fmt.Sprintf("failed to create router: %v", err)}, nil
	}

	// Check if direct translation is available
	if !r.HasDirectPair(req.SourceLang, req.TargetLang) {
		return &Response{
			Error: fmt.Sprintf("no translator for %s→%s", req.SourceLang, req.TargetLang),
		}, nil
	}

	// Chunk the texts by token count
	chunks := chunker.ChunkByTokens(req.Texts, chunker.DefaultMaxTokens)

	// Send ALL chunks in a single Lambda invocation
	// The translator processes them sequentially internally
	chunkResults, err := r.TranslateChunks(ctx, req.SourceLang, req.TargetLang, chunks)
	if err != nil {
		return &Response{Error: fmt.Sprintf("translation failed: %v", err)}, nil
	}

	// Flatten results back to single list
	allTranslations := make([]string, 0, len(req.Texts))
	for _, chunkResult := range chunkResults {
		allTranslations = append(allTranslations, chunkResult...)
	}

	return &Response{
		Translations:    allTranslations,
		ChunksProcessed: len(chunks),
	}, nil
}

// validateRequest checks the request is valid.
func validateRequest(req Request) error {
	if req.SourceLang == "" {
		return fmt.Errorf("sourceLang is required")
	}
	if req.TargetLang == "" {
		return fmt.Errorf("targetLang is required")
	}
	if req.SourceLang == req.TargetLang {
		return fmt.Errorf("sourceLang and targetLang must be different")
	}
	if req.Texts == nil {
		return fmt.Errorf("texts is required")
	}
	return nil
}
