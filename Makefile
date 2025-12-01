.PHONY: help build build-all install test test-integration test-all test-coverage bench bench-compare bench-save lint fmt vet quality clean deps run

# Variables
BINARY_NAME=gzh-quality
INSTALL_NAME=gz-quality
BUILD_DIR=build
MAIN_PATH=cmd/quality/main.go
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOINSTALL=$(GOCMD) install
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

build-all: ## Build for multiple platforms
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	@echo "Built binaries:"
	@ls -lh $(BUILD_DIR)/

install: build ## Install the binary to $GOPATH/bin
	@echo "Installing $(INSTALL_NAME)..."
	@mkdir -p $(GOPATH)/bin
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(INSTALL_NAME)
	@echo "✅ Installed $(INSTALL_NAME) to $(GOPATH)/bin/$(INSTALL_NAME)"

test: ## Run unit tests
	@echo "Running unit tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

test-integration: build ## Run integration tests
	@echo "Running integration tests..."
	$(GOTEST) -v -tags=integration ./tests/integration/...

test-all: test test-integration ## Run all tests (unit + integration)
	@echo "✅ All tests passed"

test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
	@command -v open >/dev/null 2>&1 && open coverage.html || \
	 command -v xdg-open >/dev/null 2>&1 && xdg-open coverage.html || \
	 echo "Please open coverage.html manually"

bench: ## Run performance benchmarks
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./tools ./detector ./executor

bench-save: ## Run benchmarks and save results to bench.txt
	@echo "Running benchmarks and saving to bench.txt..."
	$(GOTEST) -bench=. -benchmem ./tools ./detector ./executor | tee bench.txt
	@echo "✅ Benchmark results saved to bench.txt"

bench-compare: ## Compare current benchmarks with saved baseline (requires bench.txt)
	@echo "Running benchmarks for comparison..."
	@if [ ! -f bench.txt ]; then \
		echo "❌ No baseline found. Run 'make bench-save' first to create bench.txt"; \
		exit 1; \
	fi
	@echo "Comparing with baseline (bench.txt)..."
	@$(GOTEST) -bench=. -benchmem ./tools ./detector ./executor > bench-new.txt 2>&1
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
	$(GOFMT) ./...
	@command -v gofumpt >/dev/null 2>&1 && gofumpt -w . || echo "gofumpt not installed, using go fmt only"

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...

quality: fmt lint test ## Run all quality checks (format, lint, test)
	@echo "✅ All quality checks passed"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)/ coverage.out coverage.html bench.txt bench-new.txt
	@echo "Cleaned"

deps: ## Download and tidy dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "✅ Dependencies updated"

run: ## Run the application
	@echo "Running $(BINARY_NAME)..."
	$(GOCMD) run $(MAIN_PATH)

.DEFAULT_GOAL := help
