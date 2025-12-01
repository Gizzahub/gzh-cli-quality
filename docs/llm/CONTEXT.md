# gzh-cli-quality - Context for LLMs

## Overview

**gzh-cli-quality** is a multi-language code quality tool orchestrator written in Go.
It integrates 11+ formatters and linters into a single CLI, providing unified quality checks across Go, Python, JavaScript/TypeScript, and Rust projects.

**Repository**: https://github.com/Gizzahub/gzh-cli-quality
**Language**: Go 1.24+
**License**: MIT
**Version**: 0.1.1

---

## Core Capabilities

### 1. Multi-language Support
- **Go**: gofumpt, goimports, golangci-lint
- **Python**: black, ruff, pylint (optional)
- **JavaScript/TypeScript**: prettier, eslint, tsc
- **Rust**: rustfmt, cargo-fmt, clippy

### 2. Parallel Execution
- Worker pool pattern for efficient processing
- Configurable worker count (default: CPU cores)
- Priority-based task scheduling
- Context-based cancellation support

### 3. Git Integration
- `--staged`: Process only Git staged files
- `--changed`: Process staged + modified + untracked files
- `--since <ref>`: Process files changed since commit reference

### 4. Zero Configuration
- Auto-detect project languages
- Auto-discover installed tools
- Generate default configuration via `gz-quality init`

### 5. Multi-format Reports
- JSON: Machine-readable for CI/CD integration
- HTML: Visual report with charts
- Markdown: PR comments and documentation

---

## Architecture

### System Structure

```
CLI Layer (Cobra)
    ↓
QualityManager (Orchestration)
    ↓
┌─────────────┬─────────────┬─────────────┬─────────────┐
│  Detector   │  Registry   │  Executor   │  Reporter   │
│  Package    │  Package    │  Package    │  Package    │
└─────────────┴─────────────┴─────────────┴─────────────┘
                    ↓
            Tools Package
         (QualityTool Interface)
                    ↓
    ┌───────┬───────┬───────┬───────┐
    │  Go   │Python │ JS/TS │ Rust  │
    │ Tools │ Tools │ Tools │ Tools │
    └───────┴───────┴───────┴───────┘
```

### Key Components

#### QualityManager (`quality.go`)
- CLI command definitions
- Flag parsing and validation
- Execution flow orchestration
- Result aggregation and output

#### Tools Package (`tools/`)
- `QualityTool` interface definition
- `BaseTool` common implementation
- `ToolRegistry` for tool management
- Language-specific tool implementations

#### Executor Package (`executor/`)
- `ParallelExecutor`: Worker pool implementation
- `ExecutionPlanner`: Task planning and optimization
- Priority-based scheduling

#### Detector Package (`detector/`)
- `FileTypeDetector`: Language detection via file extensions and markers
- `SystemToolDetector`: Installed tool discovery via PATH

#### Report Package (`report/`)
- `ReportGenerator`: Multi-format report generation
- JSON, HTML, Markdown output support

---

## CLI Structure

### Commands

```
gz-quality run [flags]           # Run all formatters and linters
gz-quality check [flags]         # Run linters only (no modifications)
gz-quality tool <name> [flags]   # Run specific tool
gz-quality init [flags]          # Generate configuration file
gz-quality analyze [flags]       # Analyze project structure
gz-quality install [tool]        # Install quality tools
gz-quality upgrade [tool]        # Upgrade tools to latest version
gz-quality version [flags]       # Show version information
gz-quality list [flags]          # List available tools
```

### Key Flags

| Flag | Type | Description |
|------|------|-------------|
| `--staged` | bool | Process Git staged files only |
| `--changed` | bool | Process all changed files (staged + modified + untracked) |
| `--since <ref>` | string | Process files changed since commit reference |
| `--fix, -x` | bool | Apply auto-fixes from tools |
| `--format-only` | bool | Run formatters only |
| `--lint-only` | bool | Run linters only |
| `--workers, -w <n>` | int | Number of parallel workers (default: CPU count) |
| `--dry-run` | bool | Show execution plan without running |
| `--verbose, -v` | bool | Enable verbose output |
| `--report <format>` | string | Generate report (json, html, markdown) |
| `--output <path>` | string | Report output path |
| `--files <pattern>` | string | Specific file pattern to process |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success - no issues found |
| 1 | Issues found or partial failure |
| 2 | Execution error (config, tool issues, etc.) |

---

## Configuration

### File: `.gzquality.yml`

Located in project root. Generated via `gz-quality init`.

### Structure

```yaml
# Global settings
default_workers: 4          # Parallel worker count (default: CPU count)
timeout: "10m"              # Overall timeout

# Per-tool configuration
tools:
  <tool-name>:              # e.g., gofumpt, ruff, prettier
    enabled: true           # Enable/disable tool (default: true)
    config_file: ""         # Custom config file path
    args: []                # Additional CLI arguments
    env: {}                 # Environment variables
    priority: 10            # Execution priority (higher runs first)

# Per-language configuration
languages:
  <language>:               # e.g., Go, Python, JavaScript
    enabled: true           # Enable language processing
    preferred_tools: []     # Tool list in preferred order
    extensions: []          # File extensions

# File filtering
exclude:                    # Exclusion patterns (glob)
  - "node_modules/**"
  - "vendor/**"
  - ".git/**"

include: []                 # Inclusion patterns (overrides exclude)
```

### Default Tool Priorities

| Tool Type | Default Priority | Rationale |
|-----------|-----------------|-----------|
| Formatters | 10 | Run first (linters check formatted code) |
| Format+Lint | 7 | After formatters, before linters |
| Linters | 5 | Run last |
| Type checkers | 3 | Slowest, run last |

---

## Supported Tools

### Go Tools

#### gofumpt (Formatter)
- **Priority**: 10
- **Type**: FORMAT
- **Description**: Stricter gofmt variant
- **Binary**: `gofumpt`
- **Install**: `go install mvdan.cc/gofumpt@latest`
- **Config**: No config file, follows gofmt rules

#### goimports (Formatter)
- **Priority**: 9
- **Type**: FORMAT
- **Description**: Import statement formatter
- **Binary**: `goimports`
- **Install**: `go install golang.org/x/tools/cmd/goimports@latest`
- **Config**: No config file

#### golangci-lint (Linter)
- **Priority**: 5
- **Type**: LINT
- **Description**: Aggregates 40+ linters
- **Binary**: `golangci-lint`
- **Install**: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
- **Config**: `.golangci.yml`, `.golangci.yaml`

### Python Tools

#### black (Formatter)
- **Priority**: 10
- **Type**: FORMAT
- **Description**: Opinionated code formatter
- **Binary**: `black`
- **Install**: `pip install black`
- **Config**: `pyproject.toml`, `.black`

#### ruff (Formatter + Linter)
- **Priority**: 7
- **Type**: BOTH
- **Description**: Fast Python linter and formatter
- **Binary**: `ruff`
- **Install**: `pip install ruff`
- **Config**: `pyproject.toml`, `ruff.toml`, `.ruff.toml`

#### pylint (Linter)
- **Priority**: 5
- **Type**: LINT
- **Description**: Static code analyzer
- **Binary**: `pylint`
- **Install**: `pip install pylint`
- **Config**: `.pylintrc`, `pyproject.toml`
- **Note**: Disabled by default (slow)

### JavaScript/TypeScript Tools

#### prettier (Formatter)
- **Priority**: 10
- **Type**: FORMAT
- **Description**: Opinionated code formatter
- **Binary**: `prettier`
- **Install**: `npm install -g prettier`
- **Config**: `.prettierrc`, `.prettierrc.json`, `prettier.config.js`

#### eslint (Linter)
- **Priority**: 5
- **Type**: LINT
- **Description**: Pluggable linting utility
- **Binary**: `eslint`
- **Install**: `npm install -g eslint`
- **Config**: `.eslintrc.js`, `.eslintrc.json`, `.eslintrc.yml`

#### tsc (Type Checker)
- **Priority**: 3
- **Type**: LINT
- **Description**: TypeScript type checker
- **Binary**: `tsc`
- **Install**: `npm install -g typescript`
- **Config**: `tsconfig.json`
- **Note**: TypeScript projects only

### Rust Tools

#### rustfmt (Formatter)
- **Priority**: 10
- **Type**: FORMAT
- **Description**: Rust code formatter
- **Binary**: `rustfmt`
- **Install**: `rustup component add rustfmt`
- **Config**: `rustfmt.toml`, `.rustfmt.toml`

#### cargo-fmt (Formatter)
- **Priority**: 10
- **Type**: FORMAT
- **Description**: Cargo-integrated formatter
- **Binary**: `cargo-fmt`
- **Install**: Included with rustfmt
- **Config**: Same as rustfmt

#### clippy (Linter)
- **Priority**: 5
- **Type**: LINT
- **Description**: Rust linter
- **Binary**: `cargo-clippy`
- **Install**: `rustup component add clippy`
- **Config**: `clippy.toml`, `.clippy.toml`

---

## Data Flow

### Execution Flow: `gz-quality run --staged --fix`

```
1. CLI Parsing
   └─> Flags: {Staged: true, Fix: true}

2. Project Analysis
   ├─> FileTypeDetector → Languages: [Go, Python]
   ├─> SystemToolDetector → Tools: [gofumpt, black, ruff]
   └─> Git → StagedFiles: [main.go, utils.py]

3. Execution Planning
   └─> ExecutionPlan:
       - Task 1: gofumpt [main.go] (Priority: 10)
       - Task 2: black [utils.py] (Priority: 10)
       - Task 3: ruff [utils.py] (Priority: 7)

4. Parallel Execution (Worker Pool)
   ├─> Worker 1: gofumpt → Success (0.2s)
   ├─> Worker 2: black → Success (0.3s)
   └─> Worker 3: ruff → 2 issues found (0.5s)

5. Result Aggregation
   └─> Results: [
         {Tool: gofumpt, Success: true, Files: 1},
         {Tool: black, Success: true, Files: 1},
         {Tool: ruff, Success: false, Issues: 2}
       ]

6. Output
   └─> Console:
       ✅ gofumpt (0.2s) - 1 file
       ✅ black (0.3s) - 1 file
       ⚠️ ruff (0.5s) - 2 issues
       ✨ Complete: 0.5s, 2 files, 2 issues
```

---

## Integration Points

### Pre-commit Hooks

**Manual Hook** (`hooks/pre-commit`):
```bash
#!/bin/bash
gz-quality run --staged --fix
exit $?
```

**pre-commit Framework** (`.pre-commit-config.yaml`):
```yaml
repos:
  - repo: local
    hooks:
      - id: gz-quality
        name: Code Quality Check
        entry: gz-quality
        args: [run, --staged, --fix]
        language: system
        pass_filenames: false
```

### CI/CD Integration

**GitHub Actions**:
```yaml
- name: Quality Check
  run: |
    go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest
    gz-quality install
    gz-quality check --since origin/${{ github.base_ref }}
```

**GitLab CI**:
```yaml
quality:
  script:
    - go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest
    - gz-quality install
    - gz-quality check --since $CI_MERGE_REQUEST_DIFF_BASE_SHA
```

### Go Package API

**Programmatic Usage**:
```go
import (
    "github.com/Gizzahub/gzh-cli-quality/tools"
    "github.com/Gizzahub/gzh-cli-quality/executor"
)

// Create registry and register tools
registry := tools.NewRegistry()
registry.Register(NewGofumptTool())

// Create executor
exec := executor.NewParallelExecutor(4, 10*time.Minute)

// Execute tools
results, err := exec.ExecuteParallel(ctx, plan, 4)
```

---

## Performance Characteristics

### Benchmarks (Go 1.24, macOS M1)

| Operation | Time | Allocations |
|-----------|------|-------------|
| Registry Lookup | 14.2 ns/op | 0 B/op |
| File Filtering | 8.3 ns/op | 0 B/op |
| Tool Execution (gofumpt, 100 files) | 1.2 s | - |
| Parallel Execution (4 workers, 4 tools) | 2.4 s | - |

### Scaling Characteristics
- **Registry operations**: O(1) lookup via map
- **File filtering**: O(n) where n = file count
- **Parallel execution**: Scales linearly with worker count up to CPU cores
- **Memory usage**: ~50MB base + ~10MB per tool

### Optimization Tips
1. Use `--staged` or `--changed` for incremental checks
2. Increase `--workers` to match CPU cores
3. Disable slow tools locally (pylint, golangci-lint)
4. Use tool-specific caching (golangci-lint auto-caches)

---

## Testing

### Coverage: 76.2%

**Package Breakdown**:
- `tools/`: 82.4% (11 files, 1420 lines)
- `executor/`: 75.8% (1 file, 285 lines)
- `detector/`: 72.1% (2 files, 198 lines)
- `config/`: 68.9% (1 file, 156 lines)
- Root package: 71.3% (1 file, 789 lines)

### Test Types

**Unit Tests**:
- Tool interface implementations
- Registry operations
- Configuration parsing
- Language detection

**Integration Tests**:
- End-to-end CLI execution
- Multi-tool workflows
- Git integration

**Benchmark Tests**:
- Registry performance
- File filtering speed
- Execution overhead

### Running Tests

```bash
# All tests
go test ./... -v

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Benchmarks
go test -bench=. -benchmem ./tools ./detector ./executor
```

---

## Key Files Reference

### Entry Points
- `cmd/gz-quality/main.go`: CLI entry point
- `quality.go`: Main orchestration logic

### Core Packages
- `tools/interface.go`: QualityTool interface
- `tools/base.go`: BaseTool common implementation
- `tools/registry.go`: Tool registry
- `tools/go_tools.go`: Go tool implementations
- `tools/python_tools.go`: Python tool implementations
- `tools/javascript_tools.go`: JS/TS tool implementations
- `tools/rust_tools.go`: Rust tool implementations

### Supporting Packages
- `executor/runner.go`: Parallel execution engine
- `detector/language.go`: Language detection
- `detector/tools.go`: Tool discovery
- `config/config.go`: Configuration management
- `report/generator.go`: Report generation
- `git/utils.go`: Git utilities

### Documentation
- `README.md`: Project overview and quick start
- `docs/user/`: End-user guides
- `docs/llm/`: LLM-optimized documentation
- `docs/developer/`: Developer documentation
- `docs/integration/`: Integration guides

---

## Common Use Cases

### Use Case 1: Pre-commit Quality Check
```bash
# Check only staged files with auto-fix
gz-quality run --staged --fix

# Verify no issues remain
gz-quality check --staged
```

### Use Case 2: PR Validation
```bash
# Check files changed since main branch
gz-quality check --since main --report json --output pr-report.json
```

### Use Case 3: Full Project Audit
```bash
# Run all tools on entire project
gz-quality run --verbose

# Generate comprehensive report
gz-quality check --report html --output quality-report.html
```

### Use Case 4: Specific Language Check
```bash
# Go files only
gz-quality tool gofumpt && gz-quality tool golangci-lint

# Python files only
gz-quality tool ruff --fix && gz-quality check
```

### Use Case 5: CI/CD Integration
```bash
# Fast check for CI
gz-quality check --changed --workers 8 --timeout 5m
```

---

## Troubleshooting Quick Reference

### Command Not Found
- Add `$(go env GOPATH)/bin` to PATH
- Or use absolute path: `$(go env GOPATH)/bin/gz-quality`

### No Tools Found
- Run `gz-quality install` to install missing tools
- Check `gz-quality analyze` output

### Slow Execution
- Use `--changed` or `--staged` for incremental checks
- Increase `--workers` count
- Disable slow tools (pylint, golangci-lint) locally

### Git Integration Issues
- Verify Git is installed: `git --version`
- Check branch name: `git branch` (main vs master)
- Use commit hash: `--since abc1234`

### Configuration Not Applied
- Verify YAML syntax: `yamllint .gzquality.yml`
- Check indentation (2 spaces)
- Use `--dry-run --verbose` to debug

---

## References

- **Repository**: https://github.com/Gizzahub/gzh-cli-quality
- **Issues**: https://github.com/Gizzahub/gzh-cli-quality/issues
- **Documentation**: See `docs/` directory
- **API Reference**: `docs/developer/API.md`
- **Architecture**: `docs/developer/ARCHITECTURE.md`

---

**Last Updated**: 2025-12-01
**Version**: 0.1.1
**Go Version**: 1.24+
