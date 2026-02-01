.PHONY: help sqlc-generate test test-verbose test-coverage build build-all clean fmt lint

# Default target
help:
	@echo "Available targets:"
	@echo "  sqlc-generate    - Generate Go code from SQL queries"
	@echo "  test             - Run all tests"
	@echo "  test-verbose     - Run tests with verbose output"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  build            - Build all binaries"
	@echo "  build-api        - Build mroki-api binary"
	@echo "  build-agent      - Build mroki-agent binary"
	@echo "  clean            - Remove build artifacts"
	@echo "  fmt              - Format Go code"
	@echo "  lint             - Run golangci-lint"

# Generate code from SQL using sqlc
sqlc-generate:
	@echo "Generating Go code from SQL..."
	go tool sqlc generate

# Run all tests
test:
	@echo "Running tests..."
	go test ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "Coverage report generated: coverage.out"
	@echo "View in browser: go tool cover -html=coverage.out"

# Build all binaries
build: build-api build-agent

# Build mroki-api
build-api:
	@echo "Building mroki-api..."
	@mkdir -p bin
	go build -o bin/mroki-api ./cmd/mroki-api

# Build mroki-agent
build-agent:
	@echo "Building mroki-agent..."
	@mkdir -p bin
	go build -o bin/mroki-agent ./cmd/mroki-agent

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out

# Format Go code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...

# Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	golangci-lint run
