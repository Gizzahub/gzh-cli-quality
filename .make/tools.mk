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
#
# Installing to the shared $(go env GOPATH)/bin used to mean every Go project
# on the machine raced for one binary: whichever repo's install-lint ran last
# won, and any sibling repo pinning a different version broke without being
# touched (TASK-159). GOLANGCI_LINT_BIN now points at this repo's own
# bin/tools, so the check and the install both target the exact binary `lint`
# runs — no other repo can own or overwrite it.
install-lint: ## Install golangci-lint (pinned, v2)
	@if [ -x "$(GOLANGCI_LINT_BIN)" ] && "$(GOLANGCI_LINT_BIN)" version 2>/dev/null | grep -qF "has version $(GOLANGCI_LINT_BARE) "; then \
		echo "✅ golangci-lint $(GOLANGCI_LINT_VERSION) already installed: $(GOLANGCI_LINT_BIN)"; \
	else \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION) to $(GOLANGCI_LINT_BIN)..."; \
		mkdir -p "$(GOLANGCI_LINT_DIR)"; \
		GOBIN="$(GOLANGCI_LINT_DIR)" $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
		"$(GOLANGCI_LINT_BIN)" version 2>/dev/null | grep -qF "has version $(GOLANGCI_LINT_BARE) " || { \
			echo "⚠️  golangci-lint installation did not produce $(GOLANGCI_LINT_VERSION): $(GOLANGCI_LINT_BIN)" >&2; \
			exit 1; \
		}; \
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
