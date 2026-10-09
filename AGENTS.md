# CLAUDE.md

This file provides LLM-optimized guidance for Claude Code when working with this repository.

---

## Project Context

**Binary**: `gz-quality`
**Module**: `github.com/Gizzahub/gzh-cli-quality`
**Go Version**: 1.24+
**Architecture**: Quality analysis CLI (Cobra-based)

### Core Principles

- **Multi-language support**: Go, Python, Rust, shell scripts
- **Interface-driven design**: Use Go interfaces for tool abstraction
- **Plugin architecture**: Easy to add new linters and tools
- **Caching strategy**: Efficient result caching to avoid redundant checks
- **Direct constructors**: No DI containers, simple factory pattern
- **Quality-first**: Focus on code quality, linting, and testing

---

## Module-Specific Guides (AGENTS.md)

**Read these before modifying code:**

| Guide | Location | Purpose |
|-------|----------|---------|
| Common Rules | `cmd/AGENTS_COMMON.md` | Project-wide conventions |
| Quality CLI | `cmd/quality/AGENTS.md` | CLI-specific rules |

---

## Internal Packages

| Package | Purpose | Key Features |
|---------|---------|--------------|
| `cache/` | Result caching | Storage, manager, key generation |
| `config/` | Configuration | YAML/JSON config parsing |
| `detector/` | Language detection | File type identification |
| `executor/` | Tool execution | Command runner, output parsing |
| `report/` | Result reporting | Formatters (JSON, table, markdown) |
| `tools/` | Tool implementations | Go, Python, Rust, shell linters |
| `internal/errors` | Custom errors | Quality-specific error types |
| `internal/logger` | Structured logging | Leveled logging with context |
| `internal/testutil` | Test utilities | Mock builders, assertions |

---

## Development Workflow

### Before Code Modification

1. **Read AGENTS.md** for the module you're modifying
2. Check existing patterns in `internal/`, `cache/`, `tools/`
3. Review CONTRIBUTING.md for guidelines

### Code Modification Process

```bash
# 1. Write code + tests
# 2. Quality checks (CRITICAL)
make quality    # runs fmt + lint + test

# Quick development cycle
make dev-fast   # format + unit tests only

# Pre-PR verification
make pr-check
```

---

## Essential Commands Reference

### Development Workflow

```bash
# One-time setup
make deps
make install-tools

# Before every commit (CRITICAL)
make quality

# Build & install
make build
make install

# Quick development
make dev-fast   # format + unit tests
make dev        # format + lint + test
```

### Testing

```bash
make test           # All tests
make test-unit      # Unit tests only
make test-coverage  # With coverage report
make bench          # Benchmarks
```

### Code Quality

```bash
make fmt            # Format code
make lint           # Run linters
make fmt-diff       # Format changed files only
make lint-diff      # Lint changed files only
```

---

## Project Structure

```
.
├── cmd/
│   └── quality/
│       ├── AGENTS.md           # Module-specific guide
│       ├── main.go             # Entry point
│       ├── root.go             # Root command
│       └── *.go                # Subcommands
├── cache/                       # Caching system
│   ├── cache.go                # Cache interface
│   ├── manager.go              # Cache manager
│   ├── storage.go              # File storage
│   └── key.go                  # Cache key generation
├── config/                      # Configuration
│   ├── config.go               # Config types
│   └── loader.go               # Config loading
├── detector/                    # Language detection
│   ├── detector.go             # File detector
│   └── languages.go            # Language definitions
├── executor/                    # Tool execution
│   ├── executor.go             # Command executor
│   └── runner.go               # Execution runner
├── report/                      # Result reporting
│   ├── formatter.go            # Formatter interface
│   ├── json.go                 # JSON formatter
│   ├── table.go                # Table formatter
│   └── markdown.go             # Markdown formatter
├── tools/                       # Tool implementations
│   ├── go_tools.go             # Go linters (golangci-lint)
│   ├── python_tools.go         # Python tools (ruff, mypy)
│   ├── rust_tools.go           # Rust tools (clippy, fmt)
│   └── shell_tools.go          # Shell checkers (shellcheck)
├── internal/                    # Internal packages
│   ├── errors/                 # Custom error types
│   │   └── errors.go
│   ├── logger/                 # Structured logging
│   │   └── logger.go
│   └── testutil/               # Test utilities
│       └── testutil.go
├── benchmarks/                  # Performance benchmarks
├── tests/                       # Integration tests
├── .make/                       # Modular Makefile
├── .golangci.yml               # Linter config (30+ linters)
├── CLAUDE.md                   # This file
├── go.mod                      # Go module
├── Makefile                    # Build automation
└── README.md                   # Project documentation
```

---

## Important Rules

### Critical Requirements

- **Read AGENTS.md** before modifying any module
- Always run `make quality` before commit
- Test coverage: 80%+ for core logic
- **Sanitize command inputs** - prevent command injection
- **Cache invalidation** - proper cache key generation

### Code Style

- **Binary name**: `gz-quality`
- **Interface-driven**: Use interfaces for tools and executors
- **Error handling**: Use `internal/errors` package
- **Logging**: Use `internal/logger` package
- **Testing**: Use `internal/testutil` for test helpers

### Commit Format

```
{type}({scope}): {description}

{body}

Model: claude-{model}
Co-Authored-By: Claude <noreply@anthropic.com>
```

**Types**: feat, fix, docs, refactor, test, chore
**Scope**: REQUIRED (e.g., cmd, cache, tools, config, detector)

---

## FAQ

**Q: Where to add new linters?**
A: `tools/` - create or update `{language}_tools.go`

**Q: Where to add new language support?**
A: `detector/languages.go` - add language definition

**Q: Where to add caching logic?**
A: `cache/` - use existing cache manager

**Q: Where to add output formatters?**
A: `report/` - implement formatter interface

**Q: How to handle errors?**
A: Use `internal/errors` - `errors.Wrap()`, `errors.ErrToolNotFound`

**Q: How to add logging?**
A: Use `internal/logger` - `log := logger.New("component")`

**Q: What files should AI not modify?**
A: See `.claudeignore`

---

**Last Updated**: 2025-12-05
