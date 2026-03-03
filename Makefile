.PHONY: help dev dev-lite build test lint docker-up docker-down docker-logs docker-build docker-prod-up docker-prod-down docker-prod-logs clean deps test-unit test-integration test-e2e test-e2e-frontend test-e2e-frontend-smoke test-coverage playwright-install test-load-tags reset-data

help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

dev: ## Run full-stack development mode (infra + API + worker)
	bash ./scripts/dev-full-stack.sh

dev-lite: ## Run lightweight development mode (API only)
	FLOWRA_DEV_MODE=lite go run ./cmd/api

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

test-e2e-frontend-smoke: ## Run board+sidebar frontend smoke regression test
	go test -tags=e2e -v -timeout=5m -run TestFrontend_BoardSidebarSmokeRegression ./tests/e2e/frontend/...

test-load-tags: ## Run k6 tag-system load test (requires k6 and AUTH_TOKEN)
	k6 run tests/load/tag-system/k6-tag-message-flow.js

reset-data: ## Reset local/dev data for Chat=SoT model (events/read models/outbox/repair queue)
	bash ./scripts/reset-data.sh

playwright-install: ## Install Playwright browsers for frontend E2E tests
	go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.1 install chromium

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

docker-build: ## Build production Docker image
	docker build -t flowra:latest .

docker-prod-up: ## Start production Docker stack
	docker compose -f docker-compose.prod.yml up -d --build

docker-prod-down: ## Stop production Docker stack
	docker compose -f docker-compose.prod.yml down

docker-prod-logs: ## Show production Docker logs
	docker compose -f docker-compose.prod.yml logs -f

clean: ## Clean build artifacts and test cache
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean -testcache

deps: ## Download and tidy dependencies
	go mod download
	go mod tidy

.DEFAULT_GOAL := help
