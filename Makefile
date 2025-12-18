# =============================================================================
# Makefile - Translation Manager (Go)
# =============================================================================
#
# Orchestrator for translation requests. Routes to translator Lambdas.
#
# Usage:
#   make <target> ENV=<dev|prod>
#
# =============================================================================

ENV ?= dev
AWS_REGION ?= eu-west-1

AWS_ACCOUNT_ID_DEV = 948976367203
AWS_ACCOUNT_ID_PROD = 380283541715

ifeq ($(ENV),prod)
  AWS_PROFILE = pricofy-prod
  AWS_ACCOUNT_ID = $(AWS_ACCOUNT_ID_PROD)
else
  AWS_PROFILE = pricofy-dev
  AWS_ACCOUNT_ID = $(AWS_ACCOUNT_ID_DEV)
endif

PROJECT_ROOT = $(shell pwd)
INFRA_DIR = $(PROJECT_ROOT)/infrastructure

.DEFAULT_GOAL := help

# -----------------------------------------------------------------------------
# Help
# -----------------------------------------------------------------------------

.PHONY: help
help: ## Show this help
	@echo ""
	@echo "Translation Manager (Go)"
	@echo "========================"
	@echo ""
	@echo "Usage: make <target> ENV=<dev|prod>"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
	@echo ""

# -----------------------------------------------------------------------------
# Development
# -----------------------------------------------------------------------------

.PHONY: install
install: ## Install dependencies
	go mod tidy
	cd $(INFRA_DIR) && npm install
	cd test/e2e && npm install

.PHONY: fmt
fmt: ## Format code
	go fmt ./...

.PHONY: lint
lint: ## Run linter
	golangci-lint run

.PHONY: vet
vet: ## Run go vet
	go vet ./...

# -----------------------------------------------------------------------------
# Build
# -----------------------------------------------------------------------------

.PHONY: build
build: ## Build Go binary for Lambda (ARM64)
	@echo "Building Go binary..."
	@mkdir -p dist
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o dist/bootstrap cmd/lambda/main.go
	@echo "Binary built: dist/bootstrap"

.PHONY: build-local
build-local: ## Build for local testing
	go build -o dist/translation-manager cmd/lambda/main.go

# -----------------------------------------------------------------------------
# Test
# -----------------------------------------------------------------------------

.PHONY: test
test: ## Run unit tests
	go test ./... -v

.PHONY: test-cover
test-cover: ## Run tests with coverage
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: test-e2e
test-e2e: ## Run E2E tests (requires deployed Lambda)
	@echo "Running E2E tests against $(ENV)..."
	cd test/e2e && AWS_PROFILE=$(AWS_PROFILE) npm test

# -----------------------------------------------------------------------------
# Deploy
# -----------------------------------------------------------------------------

.PHONY: deploy
deploy: clean install build lint test deploy-cdk ## Safe deployment (clean + install + build + lint + test + deploy)
	@echo "✅ Deployed to $(ENV)!"

.PHONY: deploy-quick
deploy-quick: build deploy-cdk ## Quick deploy (skips clean/lint/test - use with caution)
	@echo "⚡ Quick deployment to $(ENV)!"

.PHONY: deploy-cdk
deploy-cdk: ## Deploy CDK stack
	@echo "Deploying to $(ENV)..."
	cd $(INFRA_DIR) && \
		AWS_PROFILE=$(AWS_PROFILE) \
		CDK_DEFAULT_ACCOUNT=$(AWS_ACCOUNT_ID) \
		CDK_DEFAULT_REGION=$(AWS_REGION) \
		npx cdk deploy --context environment=$(ENV) --require-approval never

.PHONY: cdk-diff
cdk-diff: build ## Show CDK diff
	cd $(INFRA_DIR) && \
		AWS_PROFILE=$(AWS_PROFILE) \
		CDK_DEFAULT_ACCOUNT=$(AWS_ACCOUNT_ID) \
		CDK_DEFAULT_REGION=$(AWS_REGION) \
		npx cdk diff --context environment=$(ENV)

.PHONY: cdk-synth
cdk-synth: ## Synthesize CDK stack
	cd $(INFRA_DIR) && \
		AWS_PROFILE=$(AWS_PROFILE) \
		npx cdk synth --context environment=$(ENV)

# -----------------------------------------------------------------------------
# Test Invocation
# -----------------------------------------------------------------------------

.PHONY: test-invoke
test-invoke: ## Test deployed Lambda
	@echo "Testing translation-manager..."
	@aws lambda invoke \
		--function-name pricofy-translation-manager \
		--payload '{"texts": ["Hola mundo", "iPhone en perfecto estado"], "sourceLang": "es", "targetLang": "fr"}' \
		--cli-binary-format raw-in-base64-out \
		--profile $(AWS_PROFILE) \
		--region $(AWS_REGION) \
		/dev/stdout 2>/dev/null | jq .

.PHONY: test-batch
test-batch: ## Test with larger batch
	@echo "Testing with 10 texts..."
	@aws lambda invoke \
		--function-name pricofy-translation-manager \
		--payload '{"texts": ["Texto 1", "Texto 2", "Texto 3", "Texto 4", "Texto 5", "Texto 6", "Texto 7", "Texto 8", "Texto 9", "Texto 10"], "sourceLang": "es", "targetLang": "it"}' \
		--cli-binary-format raw-in-base64-out \
		--profile $(AWS_PROFILE) \
		--region $(AWS_REGION) \
		/dev/stdout 2>/dev/null | jq .

# -----------------------------------------------------------------------------
# Cleanup
# -----------------------------------------------------------------------------

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf dist/
	rm -rf coverage.out coverage.html
	rm -rf $(INFRA_DIR)/cdk.out/
	rm -rf test/e2e/node_modules/

.PHONY: destroy
destroy: ## Destroy CDK stack
	cd $(INFRA_DIR) && \
		AWS_PROFILE=$(AWS_PROFILE) \
		npx cdk destroy --context environment=$(ENV) --force

# -----------------------------------------------------------------------------
# CI/CD
# -----------------------------------------------------------------------------

.PHONY: ci
ci: fmt vet lint test ## Run all CI checks
	@echo "All CI checks passed!"
