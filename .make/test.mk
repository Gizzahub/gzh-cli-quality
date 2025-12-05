# .make/test.mk - Testing targets
# Included by main Makefile

.PHONY: test test-unit test-integration test-coverage test-verbose bench bench-save bench-compare

test: ## Run all tests with race detection
	@echo "Running tests..."
	$(GOTEST) $(RACE_FLAG) -timeout $(TEST_TIMEOUT) -coverprofile=$(COVERAGE_OUT) ./...

test-unit: ## Run unit tests only (skip integration)
	@echo "Running unit tests..."
	$(GOTEST) -v -short -timeout $(TEST_TIMEOUT) ./...

test-integration: build ## Run integration tests only
	@echo "Running integration tests..."
	$(GOTEST) -v -tags=integration -timeout $(TEST_TIMEOUT) ./tests/integration/...

test-coverage: test ## Generate HTML coverage report
	@echo "Generating coverage report..."
	$(GO) tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	@echo "Coverage report: $(COVERAGE_HTML)"
	@$(GO) tool cover -func=$(COVERAGE_OUT) | tail -1
	@command -v open >/dev/null 2>&1 && open $(COVERAGE_HTML) || \
	 command -v xdg-open >/dev/null 2>&1 && xdg-open $(COVERAGE_HTML) || \
	 echo "Please open $(COVERAGE_HTML) manually"

test-verbose: ## Run tests with verbose output
	@echo "Running tests (verbose)..."
	$(GOTEST) -v $(RACE_FLAG) -timeout $(TEST_TIMEOUT) ./...

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
