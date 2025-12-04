package router

import (
	"context"
	"testing"
)

func TestHasDirectPair(t *testing.T) {
	r := &Router{}

	tests := []struct {
		source   string
		target   string
		expected bool
	}{
		{"es", "fr", true},
		{"fr", "es", true},
		{"es", "it", true},
		{"de", "pt", true},
		{"es", "es", false}, // Same language
		{"es", "en", false}, // English not supported
		{"en", "es", false},
		{"xx", "yy", false}, // Unknown languages
		{"es", "", false},
		{"", "fr", false},
	}

	for _, tt := range tests {
		t.Run(tt.source+"→"+tt.target, func(t *testing.T) {
			result := r.HasDirectPair(tt.source, tt.target)
			if result != tt.expected {
				t.Errorf("HasDirectPair(%q, %q) = %v, want %v",
					tt.source, tt.target, result, tt.expected)
			}
		})
	}
}

func TestGetFunctionName(t *testing.T) {
	r := &Router{}

	tests := []struct {
		source   string
		target   string
		expected string
	}{
		{"es", "fr", "pricofy-translator-es-fr"},
		{"fr", "es", "pricofy-translator-fr-es"},
		{"it", "de", "pricofy-translator-it-de"},
	}

	for _, tt := range tests {
		t.Run(tt.source+"→"+tt.target, func(t *testing.T) {
			result := r.GetFunctionName(tt.source, tt.target)
			if result != tt.expected {
				t.Errorf("GetFunctionName(%q, %q) = %q, want %q",
					tt.source, tt.target, result, tt.expected)
			}
		})
	}
}

func TestSupportedLanguages(t *testing.T) {
	// Verify all expected languages are supported
	expected := []string{"es", "it", "pt", "fr", "de"}

	for _, lang := range expected {
		if !supportedLanguages[lang] {
			t.Errorf("Language %q should be supported", lang)
		}
	}

	// Verify unsupported languages
	unsupported := []string{"en", "ru", "zh", "ja", ""}
	for _, lang := range unsupported {
		if supportedLanguages[lang] {
			t.Errorf("Language %q should not be supported", lang)
		}
	}
}

func TestTranslate_EmptyInput(t *testing.T) {
	r := &Router{}

	// Empty input should return empty output without invoking Lambda
	result, err := r.Translate(context.TODO(), "es", "fr", []string{})

	if err != nil {
		t.Errorf("Translate with empty input should not error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Translate with empty input should return empty slice, got %d items", len(result))
	}
}
