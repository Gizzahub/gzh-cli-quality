# Common Tasks - gzh-cli-quality

## Adding a New Language Tool

### QualityTool interface (tools/interface.go)

```go
type QualityTool interface {
    Name() string           // "gofumpt", "eslint"
    Language() string       // "Go", "Python"
    Type() ToolType         // FORMAT, LINT, BOTH
    IsAvailable() bool      // check if binary exists in PATH
    Run(ctx context.Context, files []string) (Result, error)
}
```

### Steps

1. Create `tools/<lang>_tools.go` implementing `QualityTool`
2. Register in `tools/registry.go` default initialization
3. Add test file `tools/<lang>_tools_test.go`

### Example: Go tools (tools/go_tools.go)

Implements `gofumpt` (FORMAT), `golangci-lint` (LINT), `goimports` (FORMAT).

Each tool follows the same pattern: detect binary → build args → run → parse output.

________________________________________________________________________

## Using the Tool Registry

```go
registry := tools.NewRegistry()
registry.Register(go.NewGofumpt())
registry.Register(python.NewBlack())

allTools := registry.GetTools()
goTools := registry.GetToolsByLanguage("Go")
linters := registry.GetToolsByType(tools.LINT)
```

________________________________________________________________________

## CLI Commands

| Command | Purpose |
|---------|---------|
| `gz-quality check` | Run all available tools on a directory |
| `gz-quality check --lang go` | Run only Go tools |
| `gz-quality check --format` | Run only formatters |
| `gz-quality check --fix` | Auto-fix where supported |

________________________________________________________________________

## Supported Languages

| Language | Formatter | Linter |
|----------|-----------|--------|
| Go | gofumpt, goimports | golangci-lint |
| Python | black, isort | ruff, flake8 |
| Rust | rustfmt | clippy |
| Shell | shfmt | shellcheck |
| JavaScript | prettier | eslint |
| C++ | clang-format | clang-tidy |
| Java | google-java-format | checkstyle |
| Kotlin | ktlint | ktlint |
| CSS | prettier | stylelint |
| Dockerfile | - | hadolint |
| Markdown | prettier | markdownlint |
| Protobuf | - | protolint |
| SQL | - | sqlfluff |
| YAML | prettier | yamllint |
| TOML | - | taplo |

________________________________________________________________________

## Tool Result Structure

```go
type Result struct {
    ToolName  string
    Files     []string
    Issues    []Issue
    Duration  time.Duration
    ExitCode  int
}
```
