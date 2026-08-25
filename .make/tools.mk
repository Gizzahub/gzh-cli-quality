# .make/tools.mk - Tool installation
# Included by main Makefile

.PHONY: install-tools install-lint install-fumpt install-goreleaser
.PHONY: print-golangci-version

install-tools: install-lint install-fumpt ## Install all development tools
	@echo "✅ All tools installed"

# The old form asked `command -v` whether *a* golangci-lint existed and skipped
# the install if one did. That check can never fail on a developer machine, so
# the pin was never exercised — and the pin was v1, which cannot read this
# repo's v2 config. Compare the version, not the presence.
install-lint: ## Install golangci-lint (pinned, v2)
	@if golangci-lint version 2>/dev/null | grep -qF "has version $(GOLANGCI_LINT_BARE) "; then \
		echo "✅ golangci-lint $(GOLANGCI_LINT_VERSION) already on PATH: $$(command -v golangci-lint)"; \
	else \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION) into $$(go env GOPATH)/bin..."; \
		GOBIN=$$(go env GOPATH)/bin $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	fi

# Single source of the version for CI — the workflow reads this rather than
# carrying its own literal, so the two cannot drift apart.
print-golangci-version: ## Print the pinned golangci-lint version
	@echo $(GOLANGCI_LINT_VERSION)

install-fumpt: ## Install gofumpt
	@echo "Installing gofumpt..."
	@if ! command -v gofumpt >/dev/null 2>&1; then \
		$(GO) install mvdan.cc/gofumpt@latest; \
	else \
		echo "gofumpt already installed"; \
	fi

install-goreleaser: ## Install goreleaser
	@echo "Installing goreleaser..."
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		$(GO) install github.com/goreleaser/goreleaser@latest; \
	else \
		echo "goreleaser already installed"; \
	fi
