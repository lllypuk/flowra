.PHONY: help dev build test lint docker-up docker-down docker-logs migrate-up migrate-down clean deps test-e2e test-e2e-docker test-e2e-short test-all test-repository test-integration test-integration-keycloak test-e2e-frontend test-e2e-frontend-headed playwright-install

help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

dev: ## Run in development mode
	go run cmd/api/main.go

build: ## Build binaries
	go build -o bin/api cmd/api/main.go
	go build -o bin/worker cmd/worker/main.go
	go build -o bin/migrator cmd/migrator/main.go

test: ## Run all tests
	go test -v -race -coverprofile=coverage.out ./...

test-unit: ## Run unit tests only
	go test -v -race ./internal/...

test-integration: ## Run integration tests (with testcontainers)
	go test -tags=integration -v -timeout=10m ./tests/integration/...

test-integration-keycloak: ## Run Keycloak integration tests only
	go test -tags=integration -v -count=1 -timeout=10m -run TestKeycloak ./tests/integration/...
	go test -tags=integration -v -count=1 -timeout=10m -run TestJWT ./tests/integration/...
	go test -tags=integration -v -count=1 -timeout=10m -run TestOAuth ./tests/integration/...
	go test -tags=integration -v -count=1 -timeout=10m -run TestAdmin ./tests/integration/...
	go test -tags=integration -v -count=1 -timeout=10m -run TestGroup ./tests/integration/...
	go test -tags=integration -v -count=1 -timeout=10m -run TestUser ./tests/integration/...
	go test -tags=integration -v -count=1 -timeout=10m -run TestFull ./tests/integration/...
	go test -tags=integration -v -count=1 -timeout=10m -run TestWorkspace ./tests/integration/...

test-e2e: ## Run E2E tests (with testcontainers)
	go test -tags=e2e -v -timeout=10m ./tests/e2e/...

test-e2e-docker: ## Run E2E tests with docker-compose MongoDB
	docker-compose up -d mongodb redis
	@sleep 2
	TEST_MONGODB_URI="mongodb://admin:admin123@localhost:27017/test_db" go test -tags=e2e -v ./tests/e2e/...
	docker-compose down

test-e2e-short: ## Run E2E tests in short mode
	go test -tags=e2e -v -short -timeout=5m ./tests/e2e/...

test-e2e-frontend: ## Run frontend E2E tests (requires running server on localhost:8080)
	go test -tags=e2e -v -timeout=5m ./tests/e2e/frontend/...

test-e2e-frontend-headed: ## Run frontend E2E tests with visible browser (for debugging)
	HEADLESS=false go test -tags=e2e -v -timeout=5m ./tests/e2e/frontend/...

playwright-install: ## Install Playwright browsers for frontend E2E tests
	go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium

test-all: ## Run all tests (unit + integration + e2e)
	go test -v -race -coverprofile=coverage.out ./internal/...
	go test -tags=integration -v -timeout=10m ./tests/integration/...
	go test -tags=e2e -v -timeout=10m ./tests/e2e/...

test-repository: ## Run only repository integration tests
	go test -v -race -timeout=5m ./internal/infrastructure/repository/...

test-coverage: ## Generate test coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-coverage-check: ## Check test coverage threshold (80%)
	@go test -coverprofile=coverage.out ./... > /dev/null 2>&1 || true
	@coverage=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ -z "$$coverage" ]; then \
		echo "❌ Failed to calculate coverage"; \
		exit 1; \
	fi; \
	if [ $$(echo "$$coverage < 80" | bc -l 2>/dev/null || echo 0) -eq 1 ]; then \
		echo "❌ Coverage $$coverage% is below 80%"; \
		exit 1; \
	else \
		echo "✅ Coverage $$coverage% meets threshold"; \
	fi

test-verbose: ## Run tests with verbose output
	go test -v -race -coverprofile=coverage.out ./...

test-clean: ## Clean test cache and coverage files
	go clean -testcache
	rm -f coverage.out coverage.html

# Linter
lint: ## Run linter and format code
	@echo "Running linter..."
	@go fmt ./...
	@golangci-lint run --fix

docker-up: ## Start Docker services
	docker-compose up -d

docker-down: ## Stop Docker services
	docker-compose down

docker-logs: ## Show Docker logs
	docker-compose logs -f

migrate-up: ## Run migrations up
	go run cmd/migrator/main.go up

migrate-down: ## Run migrations down
	go run cmd/migrator/main.go down

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

deps: ## Download dependencies
	go mod download
	go mod tidy

.DEFAULT_GOAL := help
