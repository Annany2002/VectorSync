.PHONY: help all build run dev test format lint fix clean docker-build docker-up docker-down proto migrate-up install-tools

# Show this help message
help:
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Run all checks and build 
all: format lint test build

# Build the binary
build:
	@echo "Building Vectorsync for production..."
	@go build -o bin/vectorsync cmd/server/main.go

# Run the server in production
run: build
	@./bin/vectorsync

# Run the server in development mode with air
dev:
	@air

# Run tests
test:
	go test -v ./...

# Format code with goimports
format:
	@echo "Formatting code..."
	goimports -w .
	@echo "Done!"

# Run golangci-lint
lint:
	@echo "Running linters..."
	golangci-lint run
	@echo "Done!"

# Auto-fix linting issues
fix:
	golangci-lint run --fix

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Build Docker image
docker-build:
	docker compose build

# Start Docker container
docker-up:
	docker compose up

# Stop Docker container
docker-down:
	docker compose down

# Generate protobuf code
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/v1/*.proto

# Run database migrations
migrate-up:
	psql $$DB_URL -f migrations/001_enable_pgvector.up.sql
	psql $$DB_URL -f migrations/002_create_collections.up.sql
	psql $$DB_URL -f migrations/003_create_documents.up.sql

# Install development tools
install-tools:
	@echo "Installing goimports..."
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "Installing golangci-lint..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin
	@echo "Installing air for hot reload..."
	go install github.com/air-verse/air@latest
	@echo "Done! Tools installed."