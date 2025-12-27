// Package main is the entry point for the translation manager Lambda function.
package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/pricofy/translation-manager/internal/handler"
)

func main() {
	lambda.Start(handleRequest)
}

func handleRequest(ctx context.Context, event json.RawMessage) (interface{}, error) {
	// Warmup detection (MUST be first - before any other processing)
	if warmup, ok := IsWarmupEvent(event); ok {
		return HandleWarmup(ctx, warmup)
	}

	// Parse the request and delegate to the handler
	var req handler.Request
	if err := json.Unmarshal(event, &req); err != nil {
		return nil, err
	}

	return handler.Handle(ctx, req)
}
