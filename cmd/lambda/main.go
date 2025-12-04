// Package main is the entry point for the translation manager Lambda function.
package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/pricofy/translation-manager/internal/handler"
)

func main() {
	lambda.Start(handler.Handle)
}
