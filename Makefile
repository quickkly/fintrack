# FinTrack Makefile

.PHONY: build clean install test lint fmt dev help

# Build configuration
BINARY_NAME=fintrack
BUILD_DIR=./bin
VERSION?=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✓ Built $(BUILD_DIR)/$(BINARY_NAME)"

# Install to system
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "✓ Installed to /usr/local/bin/$(BINARY_NAME)"

# Development build (faster, no optimizations)
dev:
	@echo "Building for development..."
	@go build -o $(BINARY_NAME) .
	@echo "✓ Built $(BINARY_NAME)"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@echo "✓ Cleaned"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies updated"

# Initialize project
init:
	@echo "Initializing project..."
	@go mod tidy
	@mkdir -p configs staging
	@echo "✓ Project initialized"

# Run development server (for testing)
run: dev
	@echo "Running $(BINARY_NAME)..."
	@./$(BINARY_NAME)

# Show help
help:
	@echo "Available targets:"
	@echo "  build    - Build the binary"
	@echo "  install  - Install to system"
	@echo "  dev      - Fast development build"
	@echo "  test     - Run tests"
	@echo "  lint     - Run linter"
	@echo "  fmt      - Format code"
	@echo "  clean    - Clean build artifacts"
	@echo "  deps     - Update dependencies"
	@echo "  init     - Initialize project"
	@echo "  run      - Build and run"
	@echo "  help     - Show this help"