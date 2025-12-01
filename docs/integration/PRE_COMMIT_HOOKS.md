# Pre-commit Hooks Guide

Complete guide for integrating gz-quality with pre-commit hooks for automatic code quality checks.

## Table of Contents

- [Quick Start](#quick-start)
- [Installation Methods](#installation-methods)
- [Configuration Options](#configuration-options)
- [Hook Types](#hook-types)
- [Advanced Usage](#advanced-usage)
- [Real-World Scenarios](#real-world-scenarios)
- [Performance Optimization](#performance-optimization)
- [Troubleshooting](#troubleshooting)

---

## Quick Start

### 1. Install gz-quality

```bash
go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1
```

### 2. Choose Installation Method

**Option A: Git Hooks (Simple)**
```bash
bash hooks/install.sh
```

**Option B: pre-commit Framework (Recommended)**
```bash
pip install pre-commit
cp .pre-commit-config.example.yaml .pre-commit-config.yaml
pre-commit install
```

### 3. Test It

```bash
# Make a change
echo "package main" > test.go

# Stage and commit
git add test.go
git commit -m "test: verify pre-commit hook"
# → Hook runs automatically
```

---

## Installation Methods

### Method 1: Git Hooks (Native)

**Pros**: No dependencies, fast, simple
**Cons**: Manual installation per repository, limited flexibility

#### Automatic Installation

```bash
# From project root
bash hooks/install.sh
```

The install script will:
- Check if gz-quality is installed
- Backup existing pre-commit hook
- Install new hook with correct permissions
- Display configuration instructions

#### Manual Installation

```bash
# Copy hook
cp hooks/pre-commit .git/hooks/pre-commit

# Make executable
chmod +x .git/hooks/pre-commit

# Test
.git/hooks/pre-commit
```

#### Verification

```bash
# Check installation
ls -la .git/hooks/pre-commit

# Test hook
git add .
git commit -m "test" --dry-run
```

---

### Method 2: pre-commit Framework

**Pros**: Rich ecosystem, automatic updates, language-agnostic
**Cons**: Python dependency, slightly slower

#### Setup

```bash
# Install framework
pip install pre-commit

# Create config from example
cp .pre-commit-config.example.yaml .pre-commit-config.yaml

# Install git hooks
pre-commit install

# Run on all files (first time)
pre-commit run --all-files
```

#### Configuration

`.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/Gizzahub/gzh-cli-quality
    rev: v0.1.1
    hooks:
      - id: gz-quality-check
        stages: [commit]
```

#### Available Hook IDs

| Hook ID | Description | Use Case |
|---------|-------------|----------|
| `gz-quality-check` | Full check (format + lint) | Default, comprehensive |
| `gz-quality-format` | Format only, auto-fix | Fast feedback loop |
| `gz-quality-check-go` | Go files only | Go-only projects |
| `gz-quality-check-python` | Python files only | Python-only projects |
| `gz-quality-check-javascript` | JS/TS files only | Frontend projects |
| `gz-quality-check-rust` | Rust files only | Rust projects |

#### Commands

```bash
# Run manually on all files
pre-commit run --all-files

# Run on staged files
pre-commit run

# Run specific hook
pre-commit run gz-quality-check

# Update hooks to latest versions
pre-commit autoupdate

# Skip hooks for one commit
git commit --no-verify
```

---

### Method 3: Husky (Node.js Projects)

**Pros**: Node.js native, simple for JS projects
**Cons**: Node.js only

#### Setup

```bash
npm install --save-dev husky
npx husky install
npx husky add .husky/pre-commit "gz-quality check --staged"
```

#### package.json

```json
{
  "scripts": {
    "prepare": "husky install"
  },
  "devDependencies": {
    "husky": "^8.0.0"
  }
}
```

---

## Configuration Options

### Environment Variables

Control hook behavior with environment variables:

```bash
# Command to run (default: gz-quality)
export GZ_QUALITY_CMD="gz-quality"

# Mode: check, format, or run
export GZ_QUALITY_MODE="check"

# Additional flags
export GZ_QUALITY_FLAGS="--staged --verbose"
```

### Per-Repository Configuration

`.gzquality.yml`:

```yaml
# Files to exclude
exclude:
  - "vendor/**"
  - "node_modules/**"
  - "**/*_gen.go"
  - "**/*.pb.go"
  - "**/dist/**"

# Timeout settings
timeout: "5m"

# Enable auto-fix
autofix: true

# Tools to run
tools:
  golangci-lint:
    enabled: true
    timeout: "3m"
  ruff:
    enabled: true
    args: ["--fix"]
```

---

## Hook Types

### 1. Check-Only Hook (Default)

**Purpose**: Validate code quality without modifications
**Use Case**: CI/CD, strict validation

```bash
# Git hook
export GZ_QUALITY_MODE=check
git commit

# pre-commit framework
- id: gz-quality-check
```

**Behavior**:
- ✅ Reports issues
- ❌ Does not modify files
- 🚫 Blocks commit if issues found

---

### 2. Format Hook

**Purpose**: Auto-fix formatting issues
**Use Case**: Development workflow, rapid iteration

```bash
# Git hook
export GZ_QUALITY_MODE=format
git commit

# pre-commit framework
- id: gz-quality-format
```

**Behavior**:
- ✅ Fixes formatting automatically
- ⚠️ Does not check linting
- ✅ Allows commit after fixes

---

### 3. Full Hook (Format + Lint)

**Purpose**: Comprehensive quality check with auto-fix
**Use Case**: High-quality codebases, thorough validation

```bash
# Git hook
export GZ_QUALITY_MODE=run
git commit

# pre-commit framework
- id: gz-quality-check
  args: [--fix]
```

**Behavior**:
- ✅ Fixes formatting
- ✅ Reports linting issues
- 🚫 Blocks if linting fails

---

### 4. Language-Specific Hooks

**Purpose**: Run checks only for specific languages
**Use Case**: Monorepos, multi-language projects

```yaml
repos:
  - repo: https://github.com/Gizzahub/gzh-cli-quality
    rev: v0.1.1
    hooks:
      - id: gz-quality-check-go
        files: \.go$

      - id: gz-quality-check-python
        files: \.py$

      - id: gz-quality-check-javascript
        files: \.(js|ts|jsx|tsx)$
```

**Behavior**:
- ✅ Runs only on matching file types
- ⚡ Faster for specific changes
- 📦 Parallel execution per language

---

## Advanced Usage

### Conditional Execution

Run hooks only when specific files change:

```bash
#!/bin/bash
# .git/hooks/pre-commit

# Only run if Go files changed
if git diff --cached --name-only | grep -q '\.go$'; then
    gz-quality tool golangci-lint --staged
fi

# Only run if Python files changed
if git diff --cached --name-only | grep -q '\.py$'; then
    gz-quality tool ruff --staged --fix
fi
```

---

### Staged + Unstaged Files

Check both staged and unstaged changes:

```bash
# Check only staged (default)
gz-quality check --staged

# Check all changed files
gz-quality check --changed

# Check specific paths
gz-quality check --paths src/ pkg/
```

---

### Multi-Hook Chain

Combine gz-quality with other hooks:

```yaml
repos:
  # gz-quality first
  - repo: https://github.com/Gizzahub/gzh-cli-quality
    rev: v0.1.1
    hooks:
      - id: gz-quality-format

  # Then standard checks
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.5.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: check-merge-conflict

  # Finally, run tests
  - repo: local
    hooks:
      - id: go-test
        name: Run Go tests
        entry: go test ./...
        language: system
        pass_filenames: false
```

---

### Custom Hook Script

Create a custom hook with additional logic:

```bash
#!/bin/bash
# .git/hooks/pre-commit

set -e

echo "🔍 Running quality checks..."

# 1. Format code
gz-quality run --staged --format-only --fix

# 2. Re-stage formatted files
git diff --name-only | xargs git add

# 3. Run linting
gz-quality check --staged

# 4. Run tests on changed packages
CHANGED_PKGS=$(git diff --cached --name-only | grep '\.go$' | xargs -I {} dirname {} | sort -u)
if [ -n "$CHANGED_PKGS" ]; then
    echo "Running tests on changed packages..."
    for pkg in $CHANGED_PKGS; do
        go test ./$pkg/...
    done
fi

echo "✅ All checks passed!"
```

---

## Real-World Scenarios

### Scenario 1: Monorepo with Multiple Languages

**Requirement**: Go, Python, TypeScript in one repository

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/Gizzahub/gzh-cli-quality
    rev: v0.1.1
    hooks:
      # Go backend
      - id: gz-quality-check-go
        files: ^backend/.*\.go$

      # Python services
      - id: gz-quality-check-python
        files: ^services/.*\.py$

      # TypeScript frontend
      - id: gz-quality-check-javascript
        files: ^frontend/.*\.(ts|tsx)$
```

**Benefits**:
- ⚡ Only runs relevant checks per language
- 🎯 Fast feedback for focused changes
- 📊 Parallel execution across languages

---

### Scenario 2: Fast Feedback Loop (Format First)

**Requirement**: Quick iteration, defer linting to CI

```yaml
repos:
  - repo: https://github.com/Gizzahub/gzh-cli-quality
    rev: v0.1.1
    hooks:
      # Fast: Format only (auto-fix)
      - id: gz-quality-format
        stages: [commit]

      # Thorough: Full check (CI only)
      # - id: gz-quality-check
      #   stages: [push]
```

**Workflow**:
```bash
# Local: Fast formatting (< 1s)
git commit -m "feat: add feature"

# CI: Full check (slower, but comprehensive)
gz-quality check --since origin/main
```

---

### Scenario 3: Strict Quality Gate

**Requirement**: No issues allowed in any commit

```yaml
repos:
  - repo: https://github.com/Gizzahub/gzh-cli-quality
    rev: v0.1.1
    hooks:
      - id: gz-quality-check
        args: [--strict]  # Fail on warnings
        stages: [commit]

      - id: gz-quality-check
        args: [--report, json, --output, quality-report.json]
        stages: [push]

fail_fast: true  # Stop on first failure
```

---

### Scenario 4: Gradual Adoption (Warnings Only)

**Requirement**: Introduce quality checks without blocking commits

```bash
#!/bin/bash
# .git/hooks/pre-commit

set +e  # Don't exit on error

echo "🔍 Running quality checks (warnings only)..."

# Run check but don't block
gz-quality check --staged || true

# Show results but always succeed
echo ""
echo "⚠️  Quality issues found (not blocking commit)"
echo "Run 'gz-quality run --staged --fix' to auto-fix"
echo ""

exit 0  # Always allow commit
```

---

### Scenario 5: Auto-Fix Everything

**Requirement**: Automatically fix all fixable issues

```bash
#!/bin/bash
# .git/hooks/pre-commit

set -e

echo "🔧 Auto-fixing quality issues..."

# Run with auto-fix
gz-quality run --staged --fix

# Re-stage fixed files
CHANGED=$(git diff --name-only)
if [ -n "$CHANGED" ]; then
    echo "📝 Re-staging fixed files..."
    git add $CHANGED
fi

# Final check (should pass now)
gz-quality check --staged

echo "✅ All issues fixed and verified!"
```

---

## Performance Optimization

### 1. Check Only Staged Files

**Default behavior** - fastest option:

```bash
gz-quality check --staged
```

**Benchmark**:
- 1,000 files in repo
- 10 files staged
- ⏱️ **< 1 second** (only checks 10 files)

---

### 2. Parallel Execution

Leverage worker pool for large changes:

```bash
# Auto-detect CPU cores (default)
gz-quality run --staged

# Manual tuning
gz-quality run --staged --workers 4
```

**Benchmark**:
- 100 files staged
- 8 CPU cores
- ⏱️ **3 seconds** (vs 24s sequential)

---

### 3. Skip Slow Tools

Run fast tools in hooks, slow tools in CI:

```bash
# Local hook: Fast tools only (< 2s)
gz-quality tool gofumpt --staged
gz-quality tool ruff --staged --fix

# CI: All tools including slow ones (golangci-lint)
gz-quality check --since origin/main
```

---

### 4. Cache Tool Installations

Cache Go tools to avoid reinstallation:

```bash
# .pre-commit-config.yaml
default_language_version:
  golang: "1.24"

default_install_hook_types: [pre-commit, commit-msg]

# Cached after first run
repos:
  - repo: https://github.com/Gizzahub/gzh-cli-quality
    rev: v0.1.1
    hooks:
      - id: gz-quality-check
```

---

### 5. Incremental Checks

Use `--since` for branch workflows:

```bash
# Check only commits since branching
gz-quality check --since origin/main

# Check only recent commits
gz-quality check --since HEAD~3
```

---

## Troubleshooting

### Hook Not Running

**Symptom**: Commits succeed without running quality checks

**Diagnosis**:
```bash
# Check if hook exists
ls -la .git/hooks/pre-commit

# Check if executable
[ -x .git/hooks/pre-commit ] && echo "Executable" || echo "Not executable"

# Test hook manually
.git/hooks/pre-commit
```

**Fix**:
```bash
# Set execute permission
chmod +x .git/hooks/pre-commit

# Verify installation
bash hooks/install.sh
```

---

### Command Not Found

**Symptom**: `gz-quality: command not found`

**Diagnosis**:
```bash
# Check if installed
which gz-quality

# Check Go bin path
echo $GOPATH/bin
ls -la $GOPATH/bin/gz-quality
```

**Fix**:
```bash
# Add Go bin to PATH
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
source ~/.bashrc

# Or use absolute path
export GZ_QUALITY_CMD="$HOME/go/bin/gz-quality"

# Or reinstall
go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1
```

---

### Hook Too Slow

**Symptom**: Hook takes > 10 seconds

**Diagnosis**:
```bash
# Time the hook
time gz-quality check --staged

# Check what's being scanned
gz-quality check --staged --verbose
```

**Fix**:
```bash
# 1. Use --staged (should be default)
export GZ_QUALITY_FLAGS="--staged"

# 2. Reduce workers for small changes
export GZ_QUALITY_FLAGS="--staged --workers 2"

# 3. Skip slow tools locally
gz-quality tool gofumpt --staged  # Fast format only

# 4. Exclude large directories
# Add to .gzquality.yml:
exclude:
  - "vendor/**"
  - "node_modules/**"
  - "**/dist/**"
```

---

### False Positives

**Symptom**: Hook reports errors in generated files

**Fix**:
```yaml
# .gzquality.yml
exclude:
  - "**/*_gen.go"
  - "**/*.pb.go"
  - "**/*.pb.gw.go"
  - "**/mocks/**"
  - "**/testdata/**"
  - "docs/generated/**"
```

---

### Permission Denied

**Symptom**: `permission denied: .gzquality.yml`

**Fix**:
```bash
# Fix file permissions
chmod 644 .gzquality.yml

# Fix directory permissions
chmod 755 .git/hooks
```

---

### Pre-commit Framework Issues

**Symptom**: `pre-commit` command not found

**Fix**:
```bash
# Install pre-commit
pip install --user pre-commit

# Or with system package manager
brew install pre-commit  # macOS
apt install pre-commit   # Ubuntu
```

**Symptom**: Hook fails with Go installation error

**Fix**:
```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/Gizzahub/gzh-cli-quality
    rev: v0.1.1
    hooks:
      - id: gz-quality-check
        language_version: "1.24"  # Specify Go version
```

---

### Skip Hook for Emergency Commits

**Temporary bypass**:
```bash
# Skip all hooks once
git commit --no-verify -m "hotfix: emergency fix"

# Or disable temporarily
mv .git/hooks/pre-commit .git/hooks/pre-commit.disabled
git commit -m "WIP: work in progress"
mv .git/hooks/pre-commit.disabled .git/hooks/pre-commit
```

---

## Best Practices

### 1. Start with Format-Only

Begin with formatting hooks, add linting later:

```yaml
# Week 1: Format only
- id: gz-quality-format

# Week 2: Add check to push stage
- id: gz-quality-check
  stages: [push]

# Week 3: Move check to commit stage
- id: gz-quality-check
  stages: [commit]
```

---

### 2. Layer Hooks by Speed

Run fast checks first:

```yaml
repos:
  # Fast: < 1s
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.5.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer

  # Medium: 1-5s
  - repo: https://github.com/Gizzahub/gzh-cli-quality
    rev: v0.1.1
    hooks:
      - id: gz-quality-format

  # Slow: 5-30s
  - repo: local
    hooks:
      - id: go-test
        name: Run tests
        entry: go test -short ./...
        language: system
```

---

### 3. Align Local and CI

Keep local hooks in sync with CI:

```yaml
# .pre-commit-config.yaml (local)
- id: gz-quality-check
  stages: [commit]

# .github/workflows/ci.yml (CI)
- run: gz-quality check --since origin/main
```

---

### 4. Document Bypass Procedure

Add to CONTRIBUTING.md:

```markdown
## Pre-commit Hooks

Hooks run automatically on commit. To bypass (use sparingly):

git commit --no-verify

Only use for:
- Emergency hotfixes
- WIP commits on feature branches
- Generated files you can't fix
```

---

### 5. Monitor Hook Performance

Track hook execution time:

```bash
# Add to hook
START=$(date +%s)
gz-quality check --staged
END=$(date +%s)
echo "Hook took $((END - START)) seconds"
```

---

## Related Documentation

- [CI Integration Guide](./CI_INTEGRATION.md) - Integrate with CI/CD
- [Usage Examples](../user/02-examples.md) - Complete usage examples
- [Configuration Reference](../developer/API.md) - Configuration options
- [Multi-Repo Workflows](./MULTI_REPO_WORKFLOWS.md) - Large-scale project patterns

---

**Last Updated**: 2025-11-27
**gz-quality Version**: v0.1.1
