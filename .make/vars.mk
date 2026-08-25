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
# `golangci-lint version` prints "has version 2.12.2 built with ...".
GOLANGCI_LINT_VERSION := v2.12.2
GOLANGCI_LINT_BARE := $(GOLANGCI_LINT_VERSION:v%=%)
