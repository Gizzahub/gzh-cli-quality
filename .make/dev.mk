# .make/dev.mk - Development workflow targets
# Included by main Makefile

.PHONY: dev dev-fast verify pr-check ci-local
.PHONY: watch pre-commit release-dry
.PHONY: comments todo deps-graph
.PHONY: release-quality release-coverage-check

# ==============================================================================
# Development Workflows
# ==============================================================================

dev: fmt lint test ## Standard development workflow (format, lint, test)
	@echo "✅ Development workflow completed"

dev-fast: fmt test-unit ## Quick development cycle (format + unit tests only)
	@echo "✅ Fast development cycle completed"

verify: fmt lint test test-coverage ## Complete verification before PR
	@echo "✅ Verification completed"

pr-check: fmt lint test ## Pre-PR submission check
	@echo "✅ Pre-PR check passed - ready for submission"

ci-local: clean quality build ## Run full CI pipeline locally
	@echo "✅ Local CI pipeline completed"

# ==============================================================================
# Development Helpers
# ==============================================================================

watch: ## Watch for changes and run tests (requires entr)
	@echo "Watching for changes..."
	@if command -v entr >/dev/null 2>&1; then \
		find . -name "*.go" | entr -c make test-unit; \
	else \
		echo "⚠️  entr not installed. Install with: brew install entr (macOS) or apt install entr (Linux)"; \
	fi

pre-commit: quality ## Run pre-commit checks
	@echo "✅ Pre-commit checks passed"

release-quality: quality ## Enforce PRODUCT.md release coverage after quality checks
	@$(MAKE) release-coverage-check

release-coverage-check: ## Check an existing coverage.out against the 80% release floor
	@$(GO) tool cover -func="$(COVERAGE_OUT)" >/dev/null || exit 1
	@awk 'NR > 1 { total += $$2; if ($$3 > 0) covered += $$2 } END { if (total == 0) exit 1; printf "Release coverage: %.4f%% (required: 80%%)\n", 100 * covered / total; if (covered * 100 < total * 80) exit 1 }' "$(COVERAGE_OUT)"

release-dry: install-goreleaser ## Build release archives and cask without publishing
	@echo "Running release dry run..."
	@"$(GORELEASER_BIN)" check
	@"$(GORELEASER_BIN)" release --snapshot --clean

# ==============================================================================
# Code Analysis
# ==============================================================================

comments: ## Show all TODO/FIXME/NOTE comments
	@echo "=== TODO comments ==="
	@grep -rn "TODO" --include="*.go" . 2>/dev/null | grep -v vendor | grep -v .git || echo "No TODOs found"
	@echo ""
	@echo "=== FIXME comments ==="
	@grep -rn "FIXME" --include="*.go" . 2>/dev/null | grep -v vendor | grep -v .git || echo "No FIXMEs found"
	@echo ""
	@echo "=== NOTE comments ==="
	@grep -rn "NOTE" --include="*.go" . 2>/dev/null | grep -v vendor | grep -v .git || echo "No NOTEs found"

todo: comments ## Alias for comments

deps-graph: ## Show module dependency graph
	@echo "Module dependency graph:"
	@go mod graph
