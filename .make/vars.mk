# .make/vars.mk - Common variables
# Included by main Makefile

# Project settings
BINARY_NAME := gz-quality
BUILD_DIR := build
MAIN_PKG := ./cmd/quality

# Version information
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(GIT_COMMIT) -X main.date=$(BUILD_DATE)"

# Go commands
GO := go
GOBUILD := $(GO) build
GOTEST := $(GO) test
GOINSTALL := $(GO) install
GOMOD := $(GO) mod
GOFMT := $(GO) fmt
GOVET := $(GO) vet

# Test settings
COVERAGE_OUT := coverage.out
COVERAGE_HTML := coverage.html
TEST_TIMEOUT := 5m
RACE_FLAG := -race

# Linter settings
# .golangci.yml is the v2 schema, so this must be a v2 release — v1 cannot
# parse it. GOLANGCI_LINT_BARE drops the leading `v` because
# `golangci-lint version` prints "has version 2.13.1 built with ...".
#
# v2.13.1 rather than v2.12.2: task worktrees live outside the devbox tree, so
# the devbox mise.toml pin does not reach them and mise resolves the global
# `go = "1.27.0"` there. v2.12.2 is built with go1.26 and panics on go1.27
# packages ("file requires newer Go version go1.27"), so the integration gate
# could never run this repo's linter -- reproduced, and the same panic is on
# master. gzh-cli and gzh-cli-package-manager already pin v2.13.1 and pass;
# this is aligning with them, not choosing a new version. (TASK-159)
GOLANGCI_LINT_VERSION := v2.13.1
GOLANGCI_LINT_BARE := $(GOLANGCI_LINT_VERSION:v%=%)
# All local lint paths use this exact managed binary. The directory is
# repo-owned rather than the shared $(GOPATH)/bin -- every Go project on a
# machine competed for one binary there, so two repos pinning different
# golangci-lint versions could not both be green (TASK-159). It stays
# overridable for isolated verification, while the filename remains fixed so
# install-lint and lint always agree on the destination.
GOLANGCI_LINT_DIR ?= $(CURDIR)/bin/tools
GOLANGCI_LINT_BIN := $(GOLANGCI_LINT_DIR)/golangci-lint
