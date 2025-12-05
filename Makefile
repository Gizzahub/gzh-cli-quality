# Makefile for gzh-cli-quality
# Modular structure with includes from .make/

.DEFAULT_GOAL := help

# Include modular Makefiles
include .make/vars.mk
include .make/build.mk
include .make/test.mk
include .make/quality.mk
include .make/deps.mk
include .make/tools.mk
include .make/dev.mk
include .make/docker.mk

# ==============================================================================
# Help
# ==============================================================================

.PHONY: help

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort
	@echo ''
	@echo 'Modular Makefile structure:'
	@echo '  .make/vars.mk      - Common variables'
	@echo '  .make/build.mk     - Build targets'
	@echo '  .make/test.mk      - Test targets'
	@echo '  .make/quality.mk   - Quality/lint targets'
	@echo '  .make/deps.mk      - Dependency management'
	@echo '  .make/tools.mk     - Tool installation'
	@echo '  .make/dev.mk       - Development workflows'
	@echo '  .make/docker.mk    - Docker targets'
