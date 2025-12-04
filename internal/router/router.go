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

// Supported languages (without English)
var supportedLanguages = map[string]bool{
	"es": true,
	"it": true,
	"pt": true,
	"fr": true,
	"de": true,
}

// Router routes translation requests to the appropriate Lambda function.
type Router struct {
	lambdaClient *lambda.Client
	environment  string
}

// TranslatorRequest is the request format for translator Lambdas (chunked mode).
type TranslatorRequest struct {
	Chunks [][]string `json:"chunks"`
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

// HasDirectPair checks if a direct translation pair is available.
func (r *Router) HasDirectPair(source, target string) bool {
	return supportedLanguages[source] && supportedLanguages[target] && source != target
}

// GetFunctionName returns the Lambda function name for a language pair.
func (r *Router) GetFunctionName(source, target string) string {
	return fmt.Sprintf("pricofy-translator-%s-%s", source, target)
}

// TranslateChunks sends all chunks to the translator Lambda in a single invocation.
// The translator processes each chunk sequentially and returns all results.
// This avoids spawning multiple Lambda instances.
func (r *Router) TranslateChunks(ctx context.Context, source, target string, chunks [][]string) ([][]string, error) {
	if len(chunks) == 0 {
		return [][]string{}, nil
	}

	functionName := r.GetFunctionName(source, target)

	// Prepare request with all chunks
	req := TranslatorRequest{Chunks: chunks}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Single Lambda invocation for all chunks
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
// Wraps the batch in a single chunk and unwraps the result.
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
