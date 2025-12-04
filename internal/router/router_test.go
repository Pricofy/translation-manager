package router

import (
	"context"
	"testing"
)

func TestIsValidPair(t *testing.T) {
	r := &Router{}

	tests := []struct {
		source   string
		target   string
		expected bool
	}{
		// Basic Romance pairs
		{"es", "fr", true},
		{"fr", "es", true},
		{"es", "en", true},
		{"en", "es", true},
		{"de", "en", true},
		{"en", "de", true},
		{"es", "de", true},
		{"de", "fr", true},
		{"pt", "it", true},
		// Extended Romance languages
		{"ca", "en", true}, // Catalan
		{"ro", "en", true}, // Romanian
		{"gl", "es", true}, // Galician to Spanish
		{"oc", "fr", true}, // Occitan to French
		{"la", "en", true}, // Latin
		{"co", "it", true}, // Corsican to Italian
		// Spanish variants
		{"es_MX", "en", true},
		{"es_AR", "fr", true},
		{"en", "es_ES", true},
		// French variants
		{"fr_CA", "en", true},
		{"en", "fr_BE", true},
		// Portuguese variants
		{"pt_BR", "en", true},
		{"en", "pt_PT", true},
		// Invalid pairs
		{"es", "es", false}, // Same language
		{"xx", "yy", false}, // Unknown languages
		{"es", "", false},   // Empty target
		{"", "fr", false},   // Empty source
		{"ru", "es", false}, // Unsupported language (Russian)
		{"zh", "en", false}, // Unsupported language (Chinese)
		{"nl", "en", false}, // Unsupported language (Dutch)
		{"de", "de", false}, // Same language
	}

	for _, tt := range tests {
		t.Run(tt.source+"→"+tt.target, func(t *testing.T) {
			result := r.IsValidPair(tt.source, tt.target)
			if result != tt.expected {
				t.Errorf("IsValidPair(%q, %q) = %v, want %v",
					tt.source, tt.target, result, tt.expected)
			}
		})
	}
}

func TestGetRoute(t *testing.T) {
	r := &Router{}

	tests := []struct {
		source        string
		target        string
		expectedSteps int
		firstLambda   string
	}{
		// Direct to English (1 step)
		{"es", "en", 1, "pricofy-translator-romance-en"},
		{"fr", "en", 1, "pricofy-translator-romance-en"},
		{"pt", "en", 1, "pricofy-translator-romance-en"},
		{"de", "en", 1, "pricofy-translator-de-en"},
		// Extended Romance to English
		{"ca", "en", 1, "pricofy-translator-romance-en"},
		{"ro", "en", 1, "pricofy-translator-romance-en"},
		{"gl", "en", 1, "pricofy-translator-romance-en"},
		{"es_MX", "en", 1, "pricofy-translator-romance-en"},
		{"pt_BR", "en", 1, "pricofy-translator-romance-en"},
		// From English (1 step)
		{"en", "es", 1, "pricofy-translator-en-romance"},
		{"en", "fr", 1, "pricofy-translator-en-romance"},
		{"en", "de", 1, "pricofy-translator-en-de"},
		{"en", "ca", 1, "pricofy-translator-en-romance"},
		{"en", "ro", 1, "pricofy-translator-en-romance"},
		// Romance to Romance (2 steps via EN)
		{"es", "fr", 2, "pricofy-translator-romance-en"},
		{"fr", "it", 2, "pricofy-translator-romance-en"},
		{"pt", "es", 2, "pricofy-translator-romance-en"},
		{"ca", "es", 2, "pricofy-translator-romance-en"},
		{"ro", "fr", 2, "pricofy-translator-romance-en"},
		// Romance to German (2 steps via EN)
		{"es", "de", 2, "pricofy-translator-romance-en"},
		{"fr", "de", 2, "pricofy-translator-romance-en"},
		{"ca", "de", 2, "pricofy-translator-romance-en"},
		// German to Romance (2 steps via EN)
		{"de", "es", 2, "pricofy-translator-de-en"},
		{"de", "fr", 2, "pricofy-translator-de-en"},
		{"de", "ca", 2, "pricofy-translator-de-en"},
		{"de", "ro", 2, "pricofy-translator-de-en"},
	}

	for _, tt := range tests {
		t.Run(tt.source+"→"+tt.target, func(t *testing.T) {
			route := r.getRoute(tt.source, tt.target)
			if route == nil {
				t.Fatalf("getRoute(%q, %q) returned nil", tt.source, tt.target)
			}
			if len(route) != tt.expectedSteps {
				t.Errorf("getRoute(%q, %q) returned %d steps, want %d",
					tt.source, tt.target, len(route), tt.expectedSteps)
			}
			if route[0].lambdaName != tt.firstLambda {
				t.Errorf("getRoute(%q, %q) first lambda = %q, want %q",
					tt.source, tt.target, route[0].lambdaName, tt.firstLambda)
			}
		})
	}
}

func TestGetRoute_EnRomanceTargetLang(t *testing.T) {
	r := &Router{}

	tests := []struct {
		source     string
		target     string
		targetLang string
	}{
		{"en", "es", "es"},
		{"en", "fr", "fr"},
		{"en", "it", "it"},
		{"en", "pt", "pt"},
		{"en", "ca", "ca"},
		{"en", "ro", "ro"},
		{"en", "es_MX", "es_MX"},
		{"en", "pt_BR", "pt_BR"},
		{"es", "fr", "fr"},       // Second step of pivot
		{"ca", "ro", "ro"},       // Catalan to Romanian via English
		{"de", "es_AR", "es_AR"}, // German to Argentine Spanish
	}

	for _, tt := range tests {
		t.Run(tt.source+"→"+tt.target, func(t *testing.T) {
			route := r.getRoute(tt.source, tt.target)
			if route == nil {
				t.Fatalf("getRoute(%q, %q) returned nil", tt.source, tt.target)
			}
			// Check the last step (en-romance) has correct target_lang
			lastStep := route[len(route)-1]
			if lastStep.lambdaName == "pricofy-translator-en-romance" {
				if lastStep.targetLang != tt.targetLang {
					t.Errorf("en-romance step targetLang = %q, want %q",
						lastStep.targetLang, tt.targetLang)
				}
			}
		})
	}
}

func TestSupportedLanguages(t *testing.T) {
	// Verify core languages are supported
	coreLanguages := []string{"es", "it", "pt", "fr", "de", "en"}
	for _, lang := range coreLanguages {
		if !supportedLanguages[lang] {
			t.Errorf("Core language %q should be supported", lang)
		}
	}

	// Verify extended Romance languages
	extendedRomance := []string{"ca", "ro", "gl", "oc", "la", "co", "nap", "scn"}
	for _, lang := range extendedRomance {
		if !romanceLanguages[lang] {
			t.Errorf("Extended Romance language %q should be in romanceLanguages", lang)
		}
		if !supportedLanguages[lang] {
			t.Errorf("Extended Romance language %q should be supported", lang)
		}
	}

	// Verify language variants
	variants := []string{"es_MX", "es_AR", "fr_CA", "pt_BR", "pt_PT"}
	for _, lang := range variants {
		if !romanceLanguages[lang] {
			t.Errorf("Language variant %q should be in romanceLanguages", lang)
		}
	}

	// Verify unsupported languages
	unsupported := []string{"ru", "zh", "ja", "nl", "pl", ""}
	for _, lang := range unsupported {
		if supportedLanguages[lang] {
			t.Errorf("Language %q should not be supported", lang)
		}
	}

	// German and English should NOT be in romanceLanguages
	if romanceLanguages["de"] {
		t.Error("German should not be in romanceLanguages")
	}
	if romanceLanguages["en"] {
		t.Error("English should not be in romanceLanguages")
	}
}

func TestGetSupportedLanguages(t *testing.T) {
	langs := GetSupportedLanguages()

	if len(langs) < 40 {
		t.Errorf("Expected at least 40 supported languages, got %d", len(langs))
	}

	// Check that core languages are in the list
	coreFound := map[string]bool{"es": false, "fr": false, "de": false, "en": false}
	for _, lang := range langs {
		if _, ok := coreFound[lang]; ok {
			coreFound[lang] = true
		}
	}
	for lang, found := range coreFound {
		if !found {
			t.Errorf("Core language %q not found in GetSupportedLanguages()", lang)
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

func TestTranslateChunks_EmptyInput(t *testing.T) {
	r := &Router{}

	result, err := r.TranslateChunks(context.TODO(), "es", "en", [][]string{})

	if err != nil {
		t.Errorf("TranslateChunks with empty input should not error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("TranslateChunks with empty input should return empty slice, got %d items", len(result))
	}
}
