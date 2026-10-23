# .make/quality.mk - Code quality targets
# Included by main Makefile

.PHONY: lint fmt vet quality check
.PHONY: fmt-diff lint-diff fmt-check lint-check
.PHONY: fmt-md format-md-check format-md-all-check markdown-selector-test mdformat-install-test mdformat-native-test security

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

quality: fmt format-md-check mdformat-install-test markdown-selector-test lint test ## Run all quality checks (fmt + Markdown check + installer/selector regression + lint + test)
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
MDFORMAT_BASE ?= origin/master
MDFORMAT_RUN = "$(MDFORMAT_PYTHON)" -I scripts/mdformat-files.py

fmt-md: install-mdformat ## Format markdown files with the pinned mdformat bundle
	@echo "Formatting markdown..."
	@$(MDFORMAT_RUN) all --formatter "$(MDFORMAT)"
	@echo "✅ Markdown formatted"

format-md-check: install-mdformat ## Check changed Markdown without modifying files
	@$(MDFORMAT_RUN) changed --base "$(MDFORMAT_BASE)" --formatter "$(MDFORMAT)" --check

format-md-all-check: install-mdformat ## Check all repository-owned Markdown explicitly
	@$(MDFORMAT_RUN) all --formatter "$(MDFORMAT)" --check

markdown-selector-test: install-mdformat ## Verify Markdown selector fixtures
	@"$(MDFORMAT_PYTHON)" -I scripts/test-mdformat-files.py

mdformat-install-test: install-mdformat ## Verify isolated mdformat installer fixtures
	@/bin/bash scripts/test-mdformat-install.sh "$(MDFORMAT_PYTHON)"

mdformat-native-test: install-mdformat ## Verify actual installer and native selector
	@"$(MDFORMAT_PYTHON)" -I scripts/test-mdformat-native.py

security: ## Run security scan (gosec)
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -exclude-generated ./...; \
	else \
		echo "⚠️  gosec not installed. Install: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi
