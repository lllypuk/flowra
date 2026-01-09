.PHONY: help dev build test lint docker-up docker-down docker-logs clean deps test-unit test-integration test-e2e test-e2e-frontend test-coverage playwright-install

help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

dev: ## Run in development mode
	go run ./cmd/api

build: ## Build binaries
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

test: ## Run all tests with coverage
	go test -v -race -coverprofile=coverage.out ./...

test-unit: ## Run unit tests only (fast)
	go test -v -race ./internal/...

test-integration: ## Run integration tests (with testcontainers)
	go test -tags=integration -v -timeout=10m ./tests/integration/...

test-e2e: ## Run E2E tests (with testcontainers)
	go test -tags=e2e -v -timeout=10m ./tests/e2e/...

test-e2e-frontend: ## Run frontend E2E tests (set HEADLESS=false for visible browser)
	go test -tags=e2e -v -timeout=5m ./tests/e2e/frontend/...

playwright-install: ## Install Playwright browsers for frontend E2E tests
	go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium

test-coverage: ## Generate HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint: ## Run linter and format code
	@go fmt ./...
	@golangci-lint run --fix

docker-up: ## Start Docker services
	docker-compose up -d

docker-down: ## Stop Docker services
	docker-compose down

docker-logs: ## Show Docker logs
	docker-compose logs -f

clean: ## Clean build artifacts and test cache
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean -testcache

deps: ## Download and tidy dependencies
	go mod download
	go mod tidy

.DEFAULT_GOAL := help
