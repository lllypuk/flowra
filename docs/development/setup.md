# Development Environment Setup Guide

## Overview

This document contains instructions for setting up the local development environment for the Flowra project.

## System Requirements

### Required Components

- **Go**: version 1.19 or higher
- **MongoDB**: version 8 or higher
- **Redis**: version 6 or higher
- **Docker**: version 20.10 or higher
- **Docker Compose**: version 2.0 or higher
- **Git**: version 2.30 or higher

### Recommended Tools

- **Make**: for task automation
- **golangci-lint**: for static code analysis
- **Air**: for hot reload in development
- **Postman** or **Insomnia**: for API testing
- **MongoDB Compass**: for working with the database

## Installing Dependencies

### macOS (using Homebrew)

```bash
# Go
brew install go

# MongoDB
brew tap mongodb/brew
brew install mongodb-community@8.0
brew services start mongodb-community@8.0

# Redis
brew install redis
brew services start redis

# Docker
brew install --cask docker

# Additional tools
brew install make
brew install golangci/tap/golangci-lint
```

### Ubuntu/Debian

```bash
# Go
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# MongoDB
curl -fsSL https://www.mongodb.org/static/pgp/server-8.0.asc | sudo gpg -o /usr/share/keyrings/mongodb-server-8.0.gpg --dearmor
echo "deb [ signed-by=/usr/share/keyrings/mongodb-server-8.0.gpg ] https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/8.0 multiverse" | sudo tee /etc/apt/sources.list.d/mongodb-org-8.0.list
sudo apt update
sudo apt install -y mongodb-org
sudo systemctl start mongod
sudo systemctl enable mongod

# Redis
sudo apt install redis-server
sudo systemctl start redis-server
sudo systemctl enable redis-server

# Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Additional tools
sudo apt install make
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

### Windows

1. Install Go from the official website: https://golang.org/dl/
2. Install MongoDB: https://www.mongodb.com/try/download/community
3. Install Redis: https://redis.io/download
4. Install Docker Desktop: https://www.docker.com/products/docker-desktop

## Project Setup

### 1. Clone the Repository

```bash
git clone https://github.com/lllypuk/flowra.git
cd flowra
```

### 2. Configure Environment Variables

Create a `.env` file in the project root:

```bash
cp .env.example .env
```

Edit the `.env` file:

```env
# Database
MONGODB_URI=mongodb://admin:admin123@localhost:27017
MONGODB_DATABASE=flowra

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# JWT
JWT_SECRET=your-super-secret-jwt-key-here
JWT_EXPIRES_IN=24h

# Server
SERVER_HOST=localhost
SERVER_PORT=8080

# Environment
ENV=development
LOG_LEVEL=debug

# External Services
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password

# OAuth
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
```

### 3. Install Go Dependencies

```bash
go mod download
go mod tidy
```

### 4. Run MongoDB via Docker (recommended)

```bash
# Start MongoDB via docker-compose
docker-compose up -d mongodb

# Verify connection
mongosh mongodb://admin:admin123@localhost:27017
```

### 5. Populate with Test Data (optional)

```bash
make seed
```

## Running the Application

### Development Mode

```bash
# Install Air for hot reload
go install github.com/cosmtrek/air@latest

# Run in development mode
make dev
```

### Production Mode

```bash
# Build the application
make build

# Run the compiled binary
make run
```

### Docker Compose

```bash
# Start all services in Docker
make docker-up

# Stop services
make docker-down
```

## Available Make Commands

```bash
# Development
make dev          # Run in development mode with hot reload
make build        # Build the application
make run          # Run the application
make clean        # Clean build files

# Testing
make test         # Run all tests
make test-unit    # Run unit tests
make test-integration # Run integration tests
make coverage     # Generate test coverage report

# Code Quality
make lint         # Check code with linter
make fmt          # Format code
make vet          # Check code with vet

# Database
make seed         # Populate DB with test data
make db-reset     # Clear and recreate DB

# Docker
make docker-build # Build Docker image
make docker-up    # Start services in Docker
make docker-down  # Stop Docker services

# Documentation
make docs         # Generate API documentation
make swagger      # Run Swagger UI
```

## Project Structure

After setup, you should have the following structure:

```
new-flowra/
├── .env                    # Environment variables (not committed)
├── .air.toml              # Air configuration
├── docker-compose.yml     # Docker Compose configuration
├── Makefile              # Task automation
├── cmd/                  # Application entry points
├── internal/             # Internal application code
├── pkg/                  # Reusable packages
├── migrations/           # Database migrations
├── configs/              # Configuration files
├── scripts/              # Helper scripts
└── docs/                 # Documentation
```

## IDE Setup

### VS Code

Recommended extensions:

```json
{
  "recommendations": [
    "golang.go",
    "ms-vscode-remote.remote-containers",
    "ms-azuretools.vscode-docker",
    "ms-vscode.vscode-json",
    "redhat.vscode-yaml",
    "bradlc.vscode-tailwindcss"
  ]
}
```

VS Code settings (`.vscode/settings.json`):

```json
{
  "go.formatTool": "gofmt",
  "go.lintTool": "golangci-lint",
  "go.vetOnSave": "package",
  "go.buildOnSave": "package",
  "go.testFlags": ["-v", "-race"],
  "go.coverOnSave": true,
  "go.coverageOptions": "showUncoveredCodeOnly",
  "files.exclude": {
    "**/.git": true,
    "**/node_modules": true,
    "**/vendor": true
  }
}
```

### GoLand

1. Open the project in GoLand
2. Configure Go Modules in Settings → Go → Go Modules
3. Configure Database connection in the Database panel
4. Install plugins: Docker, Database Tools and SQL

## Debugging

### Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Run with debugger
dlv debug ./cmd/api
```

### VS Code Debugging

Configuration `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch API",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/api",
      "env": {
        "ENV": "development"
      },
      "args": []
    }
  ]
}
```

## Testing

### Unit Tests

```bash
# Run all unit tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests

```bash
# Run integration tests
go test -tags=integration ./...
```

### Benchmarks

```bash
# Run benchmarks
go test -bench=. ./...
```

## Troubleshooting

### Common Issues

**1. Cannot connect to MongoDB**
```bash
# Check service status
sudo systemctl status mongod

# Check if MongoDB is running via Docker
docker ps | grep mongodb

# Verify connection
mongosh mongodb://admin:admin123@localhost:27017
```

**2. Go modules not loading**
```bash
# Clear module cache
go clean -modcache

# Reinstall dependencies
go mod download
```

**3. Port already in use**
```bash
# Find process using the port
lsof -i :8080

# Terminate the process
kill -9 <PID>
```

**4. Docker issues**
```bash
# Clean Docker containers and images
docker system prune -a

# Restart Docker daemon
sudo systemctl restart docker
```

## Next Steps

After successfully setting up the development environment:

1. Review the [coding standards](coding-standards.md)
2. Familiarize yourself with the [testing strategy](testing.md)
3. Read the [architecture description](../../ARCHITECTURE.md)
4. Start by exploring the [API documentation](../api/)

## Help

If you encounter problems:

1. Check the [FAQ](../faq.md)
2. Search for a solution in [Issues](https://github.com/lllypuk/new-flowra/issues)
3. Create a new Issue with the `setup` tag
4. Reach out in the Slack channel `#development`

---

*Last updated: [Current date]*
*Maintained by: Development Team*