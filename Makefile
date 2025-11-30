.PHONY: help build install test test-integration test-all bench bench-compare bench-save lint clean

# Variables
BINARY_NAME=gz-quality
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o build/$(BINARY_NAME) ./cmd/gz-quality

install: ## Install the binary to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) ./cmd/gz-quality

test: ## Run unit tests
	@echo "Running unit tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-integration: build ## Run integration tests
	@echo "Running integration tests..."
	go test -v -tags=integration ./tests/integration/...

test-all: test test-integration ## Run all tests (unit + integration)
	@echo "✅ All tests passed"

test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

bench: ## Run performance benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./tools ./detector ./executor

bench-save: ## Run benchmarks and save results to bench.txt
	@echo "Running benchmarks and saving to bench.txt..."
	go test -bench=. -benchmem ./tools ./detector ./executor | tee bench.txt
	@echo "✅ Benchmark results saved to bench.txt"

bench-compare: ## Compare current benchmarks with saved baseline (requires bench.txt)
	@echo "Running benchmarks for comparison..."
	@if [ ! -f bench.txt ]; then \
		echo "❌ No baseline found. Run 'make bench-save' first to create bench.txt"; \
		exit 1; \
	fi
	@echo "Comparing with baseline (bench.txt)..."
	@go test -bench=. -benchmem ./tools ./detector ./executor > bench-new.txt 2>&1
	@echo ""
	@echo "=== Benchmark Comparison ==="
	@echo "Baseline: bench.txt"
	@echo "Current:  bench-new.txt"
	@echo ""
	@echo "Use 'benchstat bench.txt bench-new.txt' for detailed comparison"
	@echo "(Install benchstat: go install golang.org/x/perf/cmd/benchstat@latest)"
	@rm -f bench-new.txt

lint: ## Run linters
	@echo "Running linters..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	golangci-lint run --timeout 5m

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	@command -v gofumpt >/dev/null 2>&1 && gofumpt -w . || echo "gofumpt not installed, using go fmt only"

quality: fmt lint test ## Run all quality checks (format, lint, test)
	@echo "✅ All quality checks passed"

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf build/ coverage.out coverage.html

.DEFAULT_GOAL := help
