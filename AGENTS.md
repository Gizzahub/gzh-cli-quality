# gzh-cli-quality

## Development workflow — user decision 2026-11-18

- Use the CE task runtime for worktree lifecycle and integration:
  `ce task run-doctor`, `run-start <card-id> --type <type>`, work and push in
  the task worktree, `run-status`, then `run-finish` from the primary checkout.
  `.ce/task-runtime.yaml` is the tracked declaration; worktree placement comes
  from the machine's CE defaults. This replaces the 2026-11-06 plain-Git rule.
- This repository keeps no task board. Its cards live only in the
  gzh-cli-devbox board; name the run after that card's ID. `tasks/` is ignored
  here; do not create cards or receipts in this repository.
- Run the repository's applicable source checks and keep failures visible.
  Documentation and workflow-only changes require diff checks and relevant
  Markdown/configuration or workflow checks; do not require unrelated Go test
  suites for prose edits.
- GitLab CI is not required to build, test, or contribute. GitHub Actions and
  server-side branch rules are separate and must still be respected where present.
- Preserve in-file locked regions. Do not change machine-global installed tools
  or unrelated repositories as part of this repository's workflow.

This file provides LLM-optimized guidance for Claude Code when working with this repository.

______________________________________________________________________

## Project Context

**Binary**: `gz-quality`
**Module**: `github.com/Gizzahub/gzh-cli-quality`
**Go Version**: 1.26+ (development toolchain: 1.27.4; see `go.mod`)
**Architecture**: Quality analysis CLI (Cobra-based)

### Core Principles

- **Multi-language support**: Go, Python, Rust, shell scripts
- **Interface-driven design**: Use Go interfaces for tool abstraction
- **Plugin architecture**: Easy to add new linters and tools
- **Caching strategy**: Efficient result caching to avoid redundant checks
- **Direct constructors**: No DI containers, simple factory pattern
- **Quality-first**: Focus on code quality, linting, and testing

______________________________________________________________________

## Development Guides

**Read these before modifying code:**

| Guide        | Location          | Purpose                  |
| ------------ | ----------------- | ------------------------ |
| Common Rules | `CONTRIBUTING.md` | Project-wide conventions |

______________________________________________________________________

## Internal Packages

| Package             | Purpose              | Key Features                       |
| ------------------- | -------------------- | ---------------------------------- |
| `cache/`            | Result caching       | Storage, manager, key generation   |
| `config/`           | Configuration        | YAML/JSON config parsing           |
| `detector/`         | Language detection   | File type identification           |
| `executor/`         | Tool execution       | Command runner, output parsing     |
| `report/`           | Result reporting     | Formatters (JSON, table, markdown) |
| `tools/`            | Tool implementations | Go, Python, Rust, shell linters    |
| `internal/errors`   | Custom errors        | Quality-specific error types       |
| `internal/logger`   | Structured logging   | Leveled logging with context       |
| `internal/testutil` | Test utilities       | Mock builders, assertions          |

______________________________________________________________________

## Development Workflow

### Before Code Modification

1. **Read applicable AGENTS.md guidance** for the module you're modifying
1. Check existing patterns in `internal/`, `cache/`, `tools/`
1. Review CONTRIBUTING.md for guidelines

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

______________________________________________________________________

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

______________________________________________________________________

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
```

**Types**: feat, fix, docs, refactor, test, chore
**Scope**: REQUIRED (e.g., cmd, cache, tools, config, detector)

______________________________________________________________________

## FAQ

**Q: Where to add new linters?**
A: `tools/` - create or update `{language}_tools.go`

**Q: Where to add new language support?**
A: `detector/language.go` - add language definition

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

______________________________________________________________________

**Last Updated**: 2026-10-27
