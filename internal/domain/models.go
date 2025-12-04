// Package domain contains the core domain types for the translation manager.
package domain

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

// TranslatorRequest is the request format for translator Lambdas.
type TranslatorRequest struct {
	Texts []string `json:"texts"`
}

// TranslatorResponse is the response format from translator Lambdas.
type TranslatorResponse struct {
	Translations []string `json:"translations"`
	Error        string   `json:"error,omitempty"`
}
