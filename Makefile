# go2postgres Makefile

.PHONY: all build run test clean lint tidy deps help

# Build variables
BINARY_NAME=go2postgres
BUILD_DIR=./build
CMD_DIR=./cmd/go2postgres
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Go commands
GO=go
GOTEST=$(GO) test
GOBUILD=$(GO) build
GOCLEAN=$(GO) clean
GOMOD=$(GO) mod

# Default target
all: deps lint test build

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Tidy go.mod
tidy:
	$(GOMOD) tidy

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all: build-linux build-windows build-darwin

build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)

build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)

build-darwin:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)

# Run the application
run: build
	$(BUILD_DIR)/$(BINARY_NAME)

# Run in development mode
dev:
	$(GO) run $(CMD_DIR)

# Run tests
test:
	$(GOTEST) -v -race -cover ./...

# Run tests with coverage report
test-coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Lint the code
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, running go vet instead"; \
		$(GO) vet ./...; \
	fi

# Format code
fmt:
	$(GO) fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Generate .env file from template
env:
	@if [ ! -f .env ]; then \
		cp .env.example .env 2>/dev/null || echo "No .env.example found"; \
		echo "Created .env file"; \
	else \
		echo ".env already exists"; \
	fi

# Database migrations (placeholder)
migrate:
	@echo "Migrations are handled automatically at startup"

# Docker build
docker-build:
	docker build -t $(BINARY_NAME):$(VERSION) .

# Docker run
docker-run:
	docker run -p 8443:8443 --env-file .env $(BINARY_NAME):$(VERSION)

# Show help
help:
	@echo "go2postgres Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all          - Run deps, lint, test, and build"
	@echo "  deps         - Download dependencies"
	@echo "  tidy         - Tidy go.mod"
	@echo "  build        - Build the binary"
	@echo "  build-all    - Build for all platforms"
	@echo "  run          - Build and run"
	@echo "  dev          - Run in development mode"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  lint         - Lint the code"
	@echo "  fmt          - Format code"
	@echo "  clean        - Clean build artifacts"
	@echo "  env          - Create .env from template"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  help         - Show this help"
