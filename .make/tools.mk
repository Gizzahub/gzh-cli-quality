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
GOLANGCI_LINT_MODULE := github.com/golangci/golangci-lint/v2

# Exits 0 only when the binary really is the pinned module version AND was built
# with the Go toolchain that is active right now.
#
# The version half alone is not enough, and that gap was measured rather than
# guessed. `go install` builds with whatever toolchain is active and ignores
# go.mod's `toolchain` directive, and task worktrees live outside this tree so
# mise resolves the *global* Go there, not the repo's pin. A binary built by an
# older Go then reads a stdlib its go/types cannot parse and dies mid-analysis:
# "panic: file requires newer Go version go1.27 (application built with
# go1.26)". A `version`-string check waves that binary through, because the
# pinned release number is correct -- it is the compiler that is wrong. The
# integration gate hit exactly this in gzh-cli-shellforge on 2026-09-04 with a
# correctly-pinned v2.13.1 binary sitting in bin/tools. This repo carried the
# same string-only check and was one toolchain change away from the same
# failure; it passed only because the v2.13.1 bump happened to force a
# reinstall under the gate's own Go.
#
# `go version -m` is what makes the check honest: it reports the module version
# recorded inside the binary, so no wrapper script or --version string can
# satisfy it, and it names the building toolchain in the same output.
GOLANGCI_LINT_VERSION_OK = $(GO) version -m "$(GOLANGCI_LINT_BIN)" 2>/dev/null | \
	awk -v want="$$($(GO) env GOVERSION)" 'NR == 1 { built = $$NF } $$1 == "mod" && $$2 == "$(GOLANGCI_LINT_MODULE)" && $$3 == "$(GOLANGCI_LINT_VERSION)" { found = 1 } END { exit !(found && built == want) }'

install-lint: ## Install golangci-lint (pinned, v2)
	@if $(GOLANGCI_LINT_VERSION_OK); then \
		echo "✅ golangci-lint $(GOLANGCI_LINT_VERSION) already installed: $(GOLANGCI_LINT_BIN)"; \
	else \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION) to $(GOLANGCI_LINT_BIN)..."; \
		mkdir -p "$(GOLANGCI_LINT_DIR)"; \
		rm -f "$(GOLANGCI_LINT_BIN)"; \
		GOBIN="$(GOLANGCI_LINT_DIR)" $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
		$(GOLANGCI_LINT_VERSION_OK) || { \
			echo "⚠️  golangci-lint at $(GOLANGCI_LINT_BIN) is not $(GOLANGCI_LINT_MODULE) $(GOLANGCI_LINT_VERSION) built with $$($(GO) env GOVERSION)" >&2; \
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
