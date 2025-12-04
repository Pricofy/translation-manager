// Package router handles routing translation requests to the correct Lambda.
package router

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

// Language groups
var (
	// Romance languages supported by opus-mt-ROMANCE-en / opus-mt-en-ROMANCE
	// All these languages can translate to/from English via the romance Lambdas
	romanceLanguages = map[string]bool{
		// Spanish variants
		"es": true, "es_AR": true, "es_CL": true, "es_CO": true, "es_CR": true,
		"es_DO": true, "es_EC": true, "es_ES": true, "es_GT": true, "es_HN": true,
		"es_MX": true, "es_NI": true, "es_PA": true, "es_PE": true, "es_PR": true,
		"es_SV": true, "es_UY": true, "es_VE": true,
		// French variants
		"fr": true, "fr_BE": true, "fr_CA": true, "fr_FR": true,
		"wa":  true, // Walloon
		"frp": true, // Franco-Provençal
		"oc":  true, // Occitan
		// Italian variants
		"it":  true,
		"co":  true, // Corsican
		"nap": true, // Neapolitan
		"scn": true, // Sicilian
		"vec": true, // Venetian
		// Portuguese variants
		"pt": true, "pt_BR": true, "pt_PT": true,
		"gl":  true, // Galician
		"mwl": true, // Mirandese
		// Catalan and related
		"ca":  true, // Catalan
		"an":  true, // Aragonese
		"lad": true, // Ladino
		// Romanian
		"ro": true,
		// Other Romance
		"la":  true, // Latin
		"rm":  true, // Romansh
		"lld": true, // Ladin
		"fur": true, // Friulian
		"lij": true, // Ligurian
		"lmo": true, // Lombard
		"sc":  true, // Sardinian
	}

	// All supported languages (romance + german + english)
	supportedLanguages = map[string]bool{}
)

// Initialize supportedLanguages from romanceLanguages + de + en
func init() {
	for lang := range romanceLanguages {
		supportedLanguages[lang] = true
	}
	supportedLanguages["de"] = true
	supportedLanguages["en"] = true
}

// Router routes translation requests to the appropriate Lambda function.
type Router struct {
	lambdaClient *lambda.Client
	environment  string
}

// TranslatorRequest is the request format for translator Lambdas (chunked mode).
type TranslatorRequest struct {
	Chunks     [][]string `json:"chunks"`
	TargetLang string     `json:"target_lang,omitempty"` // Required for en-romance
}

// TranslatorResponse is the response format from translator Lambdas (chunked mode).
type TranslatorResponse struct {
	Translations [][]string `json:"translations"`
	Error        string     `json:"error,omitempty"`
}

// New creates a new Router.
func New(ctx context.Context) (*Router, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "dev"
	}

	return &Router{
		lambdaClient: lambda.NewFromConfig(cfg),
		environment:  env,
	}, nil
}

// IsValidPair checks if a language pair can be translated.
func (r *Router) IsValidPair(source, target string) bool {
	return supportedLanguages[source] && supportedLanguages[target] && source != target
}

// GetSupportedLanguages returns a list of all supported language codes.
func GetSupportedLanguages() []string {
	langs := make([]string, 0, len(supportedLanguages))
	for lang := range supportedLanguages {
		langs = append(langs, lang)
	}
	return langs
}

// getRoute determines which Lambda(s) to call for a translation.
// Returns a list of (lambdaName, targetLang) pairs to execute in sequence.
// targetLang is only set for en-romance Lambda.
func (r *Router) getRoute(source, target string) []struct {
	lambdaName string
	targetLang string
} {
	// Direct to English
	if target == "en" {
		if romanceLanguages[source] {
			return []struct {
				lambdaName string
				targetLang string
			}{
				{lambdaName: "pricofy-translator-romance-en", targetLang: ""},
			}
		}
		if source == "de" {
			return []struct {
				lambdaName string
				targetLang string
			}{
				{lambdaName: "pricofy-translator-de-en", targetLang: ""},
			}
		}
	}

	// From English
	if source == "en" {
		if romanceLanguages[target] {
			return []struct {
				lambdaName string
				targetLang string
			}{
				{lambdaName: "pricofy-translator-en-romance", targetLang: target},
			}
		}
		if target == "de" {
			return []struct {
				lambdaName string
				targetLang string
			}{
				{lambdaName: "pricofy-translator-en-de", targetLang: ""},
			}
		}
	}

	// Romance to Romance (pivot through EN)
	if romanceLanguages[source] && romanceLanguages[target] {
		return []struct {
			lambdaName string
			targetLang string
		}{
			{lambdaName: "pricofy-translator-romance-en", targetLang: ""},
			{lambdaName: "pricofy-translator-en-romance", targetLang: target},
		}
	}

	// Romance to German (pivot through EN)
	if romanceLanguages[source] && target == "de" {
		return []struct {
			lambdaName string
			targetLang string
		}{
			{lambdaName: "pricofy-translator-romance-en", targetLang: ""},
			{lambdaName: "pricofy-translator-en-de", targetLang: ""},
		}
	}

	// German to Romance (pivot through EN)
	if source == "de" && romanceLanguages[target] {
		return []struct {
			lambdaName string
			targetLang string
		}{
			{lambdaName: "pricofy-translator-de-en", targetLang: ""},
			{lambdaName: "pricofy-translator-en-romance", targetLang: target},
		}
	}

	return nil
}

// TranslateChunks translates all chunks using the appropriate Lambda(s).
// For pairs that don't involve English, chains two Lambda calls.
func (r *Router) TranslateChunks(ctx context.Context, source, target string, chunks [][]string) ([][]string, error) {
	if len(chunks) == 0 {
		return [][]string{}, nil
	}

	route := r.getRoute(source, target)
	if route == nil {
		return nil, fmt.Errorf("unsupported language pair: %s-%s", source, target)
	}

	// Execute each step in the route
	currentChunks := chunks
	for i, step := range route {
		result, err := r.invokeLambda(ctx, step.lambdaName, step.targetLang, currentChunks)
		if err != nil {
			return nil, fmt.Errorf("step %d (%s) failed: %w", i+1, step.lambdaName, err)
		}
		currentChunks = result
	}

	return currentChunks, nil
}

// invokeLambda calls a translator Lambda with the given chunks.
func (r *Router) invokeLambda(ctx context.Context, functionName, targetLang string, chunks [][]string) ([][]string, error) {
	// Prepare request
	req := TranslatorRequest{
		Chunks:     chunks,
		TargetLang: targetLang,
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Invoke Lambda
	result, err := r.lambdaClient.Invoke(ctx, &lambda.InvokeInput{
		FunctionName: &functionName,
		Payload:      payload,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to invoke %s: %w", functionName, err)
	}

	// Check for Lambda errors
	if result.FunctionError != nil {
		return nil, fmt.Errorf("lambda error: %s", *result.FunctionError)
	}

	// Parse response
	var resp TranslatorResponse
	if err := json.Unmarshal(result.Payload, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("translator error: %s", resp.Error)
	}

	return resp.Translations, nil
}

// Translate is a convenience method for translating a single batch (no chunking).
func (r *Router) Translate(ctx context.Context, source, target string, texts []string) ([]string, error) {
	if len(texts) == 0 {
		return []string{}, nil
	}

	results, err := r.TranslateChunks(ctx, source, target, [][]string{texts})
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return []string{}, nil
	}

	return results[0], nil
}
