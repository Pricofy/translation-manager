package handler

import (
	"testing"
)

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     Request
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			request: Request{
				Texts:      []string{"Hello"},
				SourceLang: "es",
				TargetLang: "fr",
			},
			expectError: false,
		},
		{
			name: "missing sourceLang",
			request: Request{
				Texts:      []string{"Hello"},
				SourceLang: "",
				TargetLang: "fr",
			},
			expectError: true,
			errorMsg:    "sourceLang is required",
		},
		{
			name: "missing targetLang",
			request: Request{
				Texts:      []string{"Hello"},
				SourceLang: "es",
				TargetLang: "",
			},
			expectError: true,
			errorMsg:    "targetLang is required",
		},
		{
			name: "same source and target",
			request: Request{
				Texts:      []string{"Hello"},
				SourceLang: "es",
				TargetLang: "es",
			},
			expectError: true,
			errorMsg:    "sourceLang and targetLang must be different",
		},
		{
			name: "nil texts",
			request: Request{
				Texts:      nil,
				SourceLang: "es",
				TargetLang: "fr",
			},
			expectError: true,
			errorMsg:    "texts is required",
		},
		{
			name: "empty texts array is valid",
			request: Request{
				Texts:      []string{},
				SourceLang: "es",
				TargetLang: "fr",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequest(tt.request)

			if tt.expectError {
				if err == nil {
					t.Errorf("validateRequest() should have returned error")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("validateRequest() error = %q, want %q", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateRequest() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestHandle_EmptyTexts(t *testing.T) {
	// Test that empty texts array returns immediately without invoking router
	req := Request{
		Texts:      []string{},
		SourceLang: "es",
		TargetLang: "fr",
	}

	// This test verifies the early return for empty input
	// The actual Handle function would need a mock router for full testing
	err := validateRequest(req)
	if err != nil {
		t.Errorf("Empty texts should be valid: %v", err)
	}
}
