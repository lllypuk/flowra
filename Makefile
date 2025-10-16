.PHONY: help dev build test lint docker-up docker-down docker-logs migrate-up migrate-down clean deps

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

dev: ## Run in development mode
	go run cmd/api/main.go

build: ## Build binaries
	go build -o bin/api cmd/api/main.go
	go build -o bin/worker cmd/worker/main.go
	go build -o bin/migrator cmd/migrator/main.go

test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...

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
