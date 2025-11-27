# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2025-11-27

### Initial Release

This is the first release of gzh-cli-quality, a multi-language code quality tool orchestrator extracted from gzh-cli.

### Added

#### Core Features
- **Multi-language Support**: Go, Python, JavaScript/TypeScript, Rust
- **11+ Quality Tools**:
  - Go: gofumpt, goimports, golangci-lint
  - Python: black, ruff, pylint
  - JavaScript/TypeScript: prettier, eslint, tsc
  - Rust: rustfmt, clippy, cargo-fmt
- **Parallel Execution**: Worker pool pattern for fast execution
- **Git Integration**:
  - `--staged`: Check only staged files
  - `--changed`: Check all modified files
  - `--since <ref>`: Check files changed since specific commit/branch
- **Configuration System**: YAML-based configuration (`.gzquality.yml`)
- **Report Generation**: JSON, HTML, Markdown formats
- **Language Detection**: Automatic language and tool detection
- **Tool Management**: Install, upgrade, version checking commands

#### CLI Commands
- `gz-quality run`: Run all formatting and linting tools
- `gz-quality check`: Run linting only (no file modifications)
- `gz-quality init`: Generate project configuration
- `gz-quality analyze`: Analyze project and recommend tools
- `gz-quality tool <name>`: Run specific tool directly
- `gz-quality install`: Install quality tools
- `gz-quality upgrade`: Upgrade quality tools
- `gz-quality version`: Check installed tool versions
- `gz-quality list`: List available tools

#### Documentation
- **Design Documents** (72KB total):
  - README.md: Project overview and quick start
  - PRD.md: Product requirements document
  - REQUIREMENTS.md: Detailed functional/non-functional requirements
  - ARCHITECTURE.md: System architecture with diagrams
- **User Guides**:
  - docs/API.md: Complete API reference (14KB)
  - docs/ADDING_TOOLS.md: Guide for adding new tools (11KB)
  - docs/EXAMPLES.md: Comprehensive usage examples (50KB+)
  - docs/CI_INTEGRATION.md: CI/CD integration guide (20KB+)
- **Community Files**:
  - CONTRIBUTING.md: Contribution guidelines
  - CODE_OF_CONDUCT.md: Contributor Covenant v2.0
  - SECURITY.md: Security policy and vulnerability reporting

#### Testing
- **Unit Tests**: 20 tests, 34.4% coverage
- **Integration Tests**: 8 comprehensive CLI tests
- **Test Fixtures**: Sample files for Go, Python, JS, Rust
- **Makefile Targets**: `test`, `test-integration`, `test-all`

#### Quality Tools
- **Linting**: golangci-lint configuration with 15+ linters
- **Pre-commit Hooks**:
  - Pre-commit framework integration (`.pre-commit-hooks.yaml`)
  - Git hooks with automated installer (`hooks/install.sh`)
  - Multiple hook types (check, format, language-specific)
- **Makefile**: Comprehensive build system with quality targets

#### CI/CD
- **GitHub Actions Workflows**:
  - CI workflow: Multi-platform testing (Ubuntu, macOS, Windows)
  - Release workflow: Automated releases with GoReleaser
  - Coverage reporting to Codecov
- **GoReleaser**: Multi-platform binary distribution
  - Platforms: Linux, macOS, Windows
  - Architectures: amd64, arm64
  - Archive formats: tar.gz (Unix), zip (Windows)
- **Issue Templates**: Bug report, feature request
- **Pull Request Template**: Structured PR description

### Technical Details

#### Architecture
- **Language**: Go 1.24.0
- **CLI Framework**: Cobra v1.10.1
- **Testing**: testify v1.11.1
- **Configuration**: YAML v3.0.1
- **Design Pattern**: Worker Pool, Registry, Strategy

#### Project Structure
```
gzh-cli-quality/
├── cmd/gz-quality/           # CLI entry point
├── quality.go         # Quality manager and commands
├── tools/             # Quality tool implementations
├── config/            # Configuration management
├── detector/          # Language and tool detection
├── executor/          # Parallel execution engine
├── git/               # Git integration utilities
├── report/            # Report generation
├── tests/             # Integration tests
├── hooks/             # Git hooks
└── docs/              # Documentation
```

#### Dependencies
- `github.com/spf13/cobra` v1.10.1
- `github.com/stretchr/testify` v1.11.1
- `gopkg.in/yaml.v3` v3.0.1

### Migration from gzh-cli

This project was extracted from the quality module of gzh-cli with the following changes:

- **Removed Dependencies**:
  - No dependency on `internal/app` or `internal/logger`
  - Standalone binary with no external requirements
- **Simplified Interface**: Direct command execution without registry
- **Updated Import Paths**: `github.com/Gizzahub/gzh-cli-quality`

### Installation

#### From Source
```bash
git clone https://github.com/Gizzahub/gzh-cli-quality.git
cd gzh-cli-quality
make build
```

#### Using Go Install (after release)
```bash
go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.0
```

#### Pre-built Binaries
Download from [GitHub Releases](https://github.com/Gizzahub/gzh-cli-quality/releases)

### Quick Start

```bash
# Initialize configuration
gz-quality init

# Run quality checks
gz-quality run

# Check staged files before commit
gz-quality check --staged

# Generate report
gz-quality check --report json --output quality-report.json
```

### Known Limitations

- Quality tools must be installed separately (`gz-quality install`)
- Coverage reporting requires external tools
- Some tools may not be available on all platforms

### Contributors

- Archmagece (@archmagece)
- Claude AI Assistant (code generation and documentation)

### License

MIT License - see [LICENSE](LICENSE) for details

---

[Unreleased]: https://github.com/Gizzahub/gzh-cli-quality/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Gizzahub/gzh-cli-quality/releases/tag/v0.1.0
