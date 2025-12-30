.PHONY: help build dev test fmt lint clean docker-build docker-up docker-down

help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	go build -o bin/vectorsync cmd/server/main.go

dev: ## Run the server in development mode
	air

test: ## Run tests
	go test -v ./...

format: ## Format code with goimports
	@echo "Formatting code..."
	goimports -w .
	@echo "Done!"

lint: ## Run golangci-lint
	@echo "Running linters..."
	golangci-lint run
	@echo "Done!"

fix: ## Auto-fix linting issues
	golangci-lint run --fix

clean: ## Clean build artifacts
	rm -rf bin/
	go clean

docker-build: ## Build Docker image
	docker compose build

docker-up: ## Start Docker container
	docker compose up

docker-down: ## Stop Docker container
	docker compose down

proto: ## Generate protobuf code
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/v1/*.proto

install-tools: ## Install development tools
	@echo "Installing goimports..."
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "Installing golangci-lint..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin
	@echo "Done! Tools installed."
