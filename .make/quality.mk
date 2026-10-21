# .make/quality.mk - Code quality targets
# Included by main Makefile

.PHONY: lint fmt vet quality check
.PHONY: fmt-diff lint-diff fmt-check lint-check
.PHONY: fmt-md security

# ==============================================================================
# Standard Quality Checks
# ==============================================================================

lint: install-lint ## Run golangci-lint (self-installs the pinned version first)
	@echo "Running golangci-lint..."
	@"$(GOLANGCI_LINT_BIN)" run ./...

fmt: ## Format code with gofmt and gofumpt
	@echo "Formatting code..."
	$(GOFMT) ./...
	@if command -v gofumpt >/dev/null 2>&1; then \
		gofumpt -w .; \
	else \
		echo "gofumpt not installed, using go fmt only"; \
	fi

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...

quality: fmt lint test ## Run all quality checks (fmt + lint + test)
	@echo "✅ All quality checks passed"

check: vet lint ## Quick check (vet + lint, no tests)
	@echo "✅ Quick checks passed"

# ==============================================================================
# Diff-Aware Checks (faster for incremental development)
# ==============================================================================

fmt-diff: ## Format only changed Go files
	@echo "Formatting changed files..."
	@git diff --name-only --diff-filter=ACMR HEAD | grep '\.go$$' | xargs -r gofmt -w
	@if command -v gofumpt >/dev/null 2>&1; then \
		git diff --name-only --diff-filter=ACMR HEAD | grep '\.go$$' | xargs -r gofumpt -w; \
	fi
	@echo "✅ Changed files formatted"

lint-diff: install-lint ## Lint only changed Go files
	@echo "Linting changed files..."
	@"$(GOLANGCI_LINT_BIN)" run --new-from-rev=HEAD~1 ./...

fmt-check: ## Check if code is formatted (for CI)
	@echo "Checking code format..."
	@test -z "$$(gofmt -l .)" || { echo "Code is not formatted. Run: make fmt"; exit 1; }
	@echo "✅ Code is properly formatted"

lint-check: install-lint ## Run lint without fixing (for CI)
	@echo "Checking lint..."
	@"$(GOLANGCI_LINT_BIN)" run ./...

# ==============================================================================
# Additional Quality Tools
# ==============================================================================

# Git 조회·필터·formatter 실패를 전파하고 공백/개행 경로를 NUL로 보존한다.
# ignored venv와 symlink는 저장소 소유 Markdown 입력에서 제외한다.
# ./ 접두사는 '-'로 시작하는 파일명이 formatter 옵션이 되는 것을 막는다.
MDFORMAT_RUN = bash -o pipefail -c 'mode=$$1; shift; { if [ "$$mode" = changed ]; then git diff --name-only -z --diff-filter=d HEAD -- "*.md"; else git ls-files -z --cached --others --exclude-standard -- "*.md"; fi; } | while IFS= read -r -d "" f; do if [ -f "$$f" ] && [ ! -L "$$f" ]; then printf "%s\0" "./$$f"; fi; done | xargs -0 -r "$$@"' _

fmt-md: install-mdformat ## Format markdown files with the pinned mdformat bundle
	@echo "Formatting markdown..."
	@$(MDFORMAT_RUN) all "$(MDFORMAT)"
	@echo "✅ Markdown formatted"

security: ## Run security scan (gosec)
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -exclude-generated ./...; \
	else \
		echo "⚠️  gosec not installed. Install: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi
