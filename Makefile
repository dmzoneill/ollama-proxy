.PHONY: all proto build run clean install-tools help test-coverage coverage security bench verify ci docker-build docker-run

# Variables
BINARY_NAME=ollama-proxy
VERSION?=dev
GIT_COMMIT=$(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-w -s -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# Default target
all: proto build

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Install required tools
install-tools:
	@echo "Installing protobuf compiler and plugins..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
	@echo "Tools installed! Make sure $(HOME)/go/bin is in your PATH"

# Generate proto code
proto:
	@echo "Generating gRPC code from proto files..."
	mkdir -p api/gen/go
	protoc \
		--proto_path=api/proto \
		--go_out=api/gen/go \
		--go_opt=paths=source_relative \
		--go-grpc_out=api/gen/go \
		--go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=api/gen/go \
		--grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=api/gen/openapiv2 \
		api/proto/*.proto
	@echo "Proto generation complete!"

# Build the binary
build: ## Build the binary with version info
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/proxy
	@echo "Build complete! Binary at bin/$(BINARY_NAME)"

# Run the proxy
run: build ## Run the proxy locally
	./bin/$(BINARY_NAME) --config config/config.yaml

# Development run (with live reload if you have air installed)
dev: ## Run in development mode with live reload
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Install air for live reload: go install github.com/air-verse/air@latest"; \
		go run ./cmd/proxy --config config/config.yaml; \
	fi

# Clean generated files
clean: ## Clean generated files and build artifacts
	rm -rf api/gen bin
	rm -f coverage.out coverage.html

# Format code
fmt: ## Format code with gofmt and goimports
	go fmt ./...
	@if command -v goimports > /dev/null; then \
		goimports -w .; \
	else \
		echo "Install goimports: go install golang.org/x/tools/cmd/goimports@latest"; \
	fi

# Run tests
test: ## Run all tests
	go test -v -race ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

coverage: test-coverage ## Alias for test-coverage

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Run linter
lint: ## Run golangci-lint
	golangci-lint run

security: ## Run security checks with gosec
	@if command -v gosec > /dev/null; then \
		gosec ./...; \
	else \
		echo "Install gosec: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
		exit 1; \
	fi

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

# Generate dependencies
deps: ## Download and tidy dependencies
	go mod download
	go mod tidy

verify: fmt vet lint test ## Run all verification steps

ci: verify security ## Run all CI checks

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(BINARY_NAME):$(VERSION) \
		-t $(BINARY_NAME):latest \
		.

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 -p 50051:50051 -p 9090:9090 $(BINARY_NAME):latest

docker-compose-up: ## Start services with docker-compose
	@echo "Starting services..."
	docker-compose up --build

docker-compose-down: ## Stop services with docker-compose
	@echo "Stopping services..."
	docker-compose down

# Full setup for new developers
setup: install-tools deps proto ## Full setup for new developers
	@echo "Setup complete! Run 'make run' to start the proxy"

.DEFAULT_GOAL := help
