.PHONY: help dev build test lint docker-up docker-down docker-logs migrate-up migrate-down clean deps test-integration test-integration-docker test-integration-short test-all test-repository

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

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
	go test -tags=integration -v -race -timeout=10m ./tests/integration/...

test-integration-docker: ## Run integration tests with docker-compose MongoDB
	docker-compose up -d mongodb
	@sleep 2
	TEST_MONGODB_URI="mongodb://admin:admin123@localhost:27017/test_db" go test -tags=integration -v ./tests/integration/...
	docker-compose down

test-integration-short: ## Run integration tests in short mode
	go test -tags=integration -v -short -timeout=5m ./tests/integration/...

test-all: ## Run all tests (unit + integration)
	go test -v -race -coverprofile=coverage.out ./internal/...
	go test -tags=integration -v -race -timeout=10m ./tests/integration/...

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

# Линтер
lint:
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
