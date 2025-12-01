# Capabilities Reference

Comprehensive reference of all capabilities provided by gzh-cli-quality.

---

## Execution Modes

### 1. Full Quality Check

**Command**: `gz-quality run`

**Scope**: All files in project

**Actions**:
- Run all formatters (gofumpt, black, prettier, rustfmt)
- Run all linters (golangci-lint, ruff, eslint, clippy)

**Use Cases**:
- Full project audit
- Pre-commit verification
- Initial project setup

**Example**:
```bash
gz-quality run
gz-quality run --verbose
gz-quality run --dry-run  # Preview without executing
```

**Expected Output**:
```
🔍 프로젝트 분석 중...
📋 실행 계획: 5개 도구, 150개 파일
⚡ 실행 중 (4 workers)...
  ✅ gofumpt (1.2s) - 45 files
  ✅ black (0.8s) - 30 files
  ✅ prettier (1.5s) - 75 files
  ⚠️ golangci-lint (3.2s) - 5 issues
  ⚠️ eslint (2.1s) - 3 issues
✨ 완료: 3.5s, 150 files, 8 issues
```

---

### 2. Staged Files Check

**Command**: `gz-quality run --staged`

**Scope**: Git staged files only

**Actions**:
- Analyze files in Git staging area
- Run formatters and linters on staged files only
- Respect Git index state

**Use Cases**:
- Pre-commit hook integration
- Quick feedback during development
- Avoid processing entire codebase

**Example**:
```bash
# Stage files
git add main.go utils.py

# Check staged files
gz-quality run --staged

# With auto-fix
gz-quality run --staged --fix

# Re-stage fixed files
git add .
```

**Git Command Used Internally**:
```bash
git diff --name-only --staged
```

**Workflow**:
```
1. User: git add file.go
2. Tool: Detect staged files via Git
3. Tool: Run gofumpt on file.go only
4. User: Review changes
5. User: git commit
```

---

### 3. Changed Files Check

**Command**: `gz-quality run --changed`

**Scope**: Staged + modified + untracked files

**Actions**:
- Process all changes in working directory
- Include unstaged modifications
- Include untracked files

**Use Cases**:
- Work-in-progress validation
- Broader check than --staged
- Local development workflow

**Example**:
```bash
# Check all changed files
gz-quality run --changed

# With specific tool
gz-quality tool golangci-lint --changed
```

**Git Commands Used Internally**:
```bash
git diff --name-only                      # Modified files
git diff --name-only --staged            # Staged files
git ls-files --others --exclude-standard # Untracked files
```

---

### 4. Diff-based Check

**Command**: `gz-quality run --since <ref>`

**Scope**: Files changed since commit reference

**Actions**:
- Compare working tree with specified commit
- Process only differing files
- Support commit hash, branch name, tag

**Use Cases**:
- PR validation
- Feature branch review
- Incremental quality checks

**Example**:
```bash
# Since main branch
gz-quality run --since main

# Since specific commit
gz-quality run --since abc1234

# Since HEAD~5
gz-quality run --since HEAD~5

# Since tag
gz-quality run --since v1.0.0
```

**Git Command Used Internally**:
```bash
git diff --name-only <ref>...HEAD
```

**Validation Workflow**:
```
1. PR created: feature-branch → main
2. CI runs: gz-quality check --since main
3. Only changed files processed
4. Report generated for PR review
```

---

### 5. Lint-only Check

**Command**: `gz-quality check`

**Scope**: All files (or with Git flags)

**Actions**:
- Run linters only (golangci-lint, ruff, eslint, clippy)
- Skip formatters (no file modifications)
- Read-only validation

**Use Cases**:
- CI/CD validation
- Code review
- Quality gate enforcement

**Example**:
```bash
# Check all files
gz-quality check

# Check staged files
gz-quality check --staged

# Check since main
gz-quality check --since main

# Generate report
gz-quality check --report json --output report.json
```

**Exit Codes**:
- 0: No issues found
- 1: Issues found
- 2: Execution error

---

### 6. Format-only Mode

**Command**: `gz-quality run --format-only`

**Scope**: All files (or with Git flags)

**Actions**:
- Run formatters only (gofumpt, black, prettier, rustfmt)
- Skip linters
- Apply code style enforcement

**Use Cases**:
- Quick code formatting
- Style consistency enforcement
- Pre-commit formatting

**Example**:
```bash
# Format all files
gz-quality run --format-only

# Format with auto-fix
gz-quality run --format-only --fix

# Format staged files
gz-quality run --format-only --staged --fix
```

**Tools Executed**:
- Go: gofumpt, goimports
- Python: black
- JavaScript/TypeScript: prettier
- Rust: rustfmt

---

### 7. Specific Tool Execution

**Command**: `gz-quality tool <name> [flags]`

**Scope**: Single tool execution

**Actions**:
- Execute specified tool only
- Support all Git integration flags
- Pass additional arguments to tool

**Use Cases**:
- Debugging tool behavior
- Targeted quality checks
- Tool-specific configuration testing

**Example**:
```bash
# Run gofumpt
gz-quality tool gofumpt

# Run ruff with auto-fix
gz-quality tool ruff --fix

# Run golangci-lint on staged files
gz-quality tool golangci-lint --staged

# Pass additional arguments
gz-quality tool golangci-lint -- --enable-all --max-issues-per-linter 0
```

**Supported Tools**:
- Go: `gofumpt`, `goimports`, `golangci-lint`
- Python: `black`, `ruff`, `pylint`
- JavaScript/TypeScript: `prettier`, `eslint`, `tsc`
- Rust: `rustfmt`, `cargo-fmt`, `clippy`

---

### 8. Project Analysis

**Command**: `gz-quality analyze`

**Scope**: Project metadata

**Actions**:
- Detect languages via file extensions and markers
- Discover installed tools via PATH
- Recommend missing tools
- Show project statistics

**Use Cases**:
- Initial project assessment
- Troubleshooting tool detection
- Documentation of project structure

**Example**:
```bash
# Basic analysis
gz-quality analyze

# Verbose output
gz-quality analyze --verbose

# JSON output
gz-quality analyze --json
```

**Output**:
```
📊 프로젝트 분석 결과

감지된 언어:
  ✓ Go (45 files)
    Indicators: go.mod, go.sum
    Extensions: .go
  ✓ Python (30 files)
    Indicators: pyproject.toml, requirements.txt
    Extensions: .py
  ✓ TypeScript (75 files)
    Indicators: package.json, tsconfig.json
    Extensions: .ts, .tsx

권장 도구:
  Go:
    ✓ gofumpt (설치됨, v0.6.0)
    ✓ goimports (설치됨, v0.16.1)
    ✗ golangci-lint (미설치)
  Python:
    ✓ black (설치됨, 24.1.0)
    ✓ ruff (설치됨, 0.1.14)
    ✗ pylint (미설치)
  TypeScript:
    ✓ prettier (설치됨, 3.1.0)
    ✓ eslint (설치됨, 8.56.0)
    ✓ tsc (설치됨, 5.3.3)
```

---

### 9. Configuration Initialization

**Command**: `gz-quality init`

**Scope**: Configuration file

**Actions**:
- Generate `.gzquality.yml` in project root
- Include detected languages and tools
- Set default values

**Use Cases**:
- Initial project setup
- Configuration template generation
- Reset configuration to defaults

**Example**:
```bash
# Generate config
gz-quality init

# Overwrite existing
gz-quality init --force

# Custom output path
gz-quality init --output custom-config.yml
```

**Generated File** (`.gzquality.yml`):
```yaml
default_workers: 4
timeout: "10m"

tools:
  gofumpt:
    enabled: true
    priority: 10
  goimports:
    enabled: true
    priority: 9
  golangci-lint:
    enabled: true
    priority: 5
    config_file: ".golangci.yml"
  black:
    enabled: true
    priority: 10
  ruff:
    enabled: true
    priority: 7
    args: ["--fix"]

languages:
  Go:
    enabled: true
    preferred_tools: [gofumpt, goimports, golangci-lint]
  Python:
    enabled: true
    preferred_tools: [black, ruff]

exclude:
  - "vendor/**"
  - "node_modules/**"
  - ".git/**"
  - "dist/**"
  - "build/**"
```

---

### 10. Tool Installation

**Command**: `gz-quality install [tool]`

**Scope**: External tool installation

**Actions**:
- Install quality tools to system
- Verify PATH and permissions
- Support individual or batch installation

**Use Cases**:
- New developer onboarding
- CI/CD environment setup
- Missing tool resolution

**Example**:
```bash
# Install all required tools
gz-quality install

# Install specific tool
gz-quality install golangci-lint
gz-quality install ruff

# Install by language
gz-quality install --language Go
gz-quality install --language Python

# Install all supported tools
gz-quality install --all
```

**Installation Methods**:
- Go tools: `go install <package>@latest`
- Python tools: `pip install <package>`
- Node tools: `npm install -g <package>`
- Rust tools: `rustup component add <package>`

---

### 11. Tool Upgrade

**Command**: `gz-quality upgrade [tool]`

**Scope**: Tool version management

**Actions**:
- Upgrade tools to latest version
- Support individual or batch upgrade
- Verify upgrade success

**Example**:
```bash
# Upgrade all tools
gz-quality upgrade

# Upgrade specific tool
gz-quality upgrade golangci-lint
gz-quality upgrade ruff
```

---

### 12. Version Information

**Command**: `gz-quality version`

**Scope**: Version metadata

**Actions**:
- Show gz-quality version
- Show installed tool versions
- Show tool locations

**Example**:
```bash
# Basic version
gz-quality version

# JSON output
gz-quality version --json
```

**Output**:
```
gzh-cli-quality v0.1.1

설치된 도구:
  gofumpt       v0.6.0      /Users/user/go/bin/gofumpt
  goimports     v0.16.1     /Users/user/go/bin/goimports
  golangci-lint v1.55.2     /Users/user/go/bin/golangci-lint
  black         24.1.0      /Users/user/.local/bin/black
  ruff          0.1.14      /Users/user/.local/bin/ruff
  prettier      3.1.0       /usr/local/bin/prettier
  eslint        8.56.0      /usr/local/bin/eslint
```

---

### 13. List Available Tools

**Command**: `gz-quality list`

**Scope**: Tool catalog

**Actions**:
- List all supported tools
- Show tool type (formatter/linter)
- Show installation status
- Support filtering

**Example**:
```bash
# List all tools
gz-quality list

# Filter by language
gz-quality list --language Go
gz-quality list --language Python

# Filter by type
gz-quality list --type formatter
gz-quality list --type linter

# Show only installed tools
gz-quality list --installed
```

**Output**:
```
사용 가능한 도구:

Go:
  ✓ gofumpt       formatter   Go 코드 포매터 (gofmt 상위 호환)
  ✓ goimports     formatter   import 문 정리 및 포매팅
  ✓ golangci-lint linter      통합 린터 (43+ 린터 포함)

Python:
  ✓ black         formatter   opinionated Python 포매터
  ✓ ruff          both        빠른 Python 린터/포매터
  ✗ pylint        linter      Python 정적 분석기 (미설치)

JavaScript/TypeScript:
  ✓ prettier      formatter   JS/TS 코드 포매터
  ✓ eslint        linter      JS/TS 린터
  ✓ tsc           linter      TypeScript 타입 체커

Rust:
  ✗ rustfmt       formatter   Rust 코드 포매터 (미설치)
  ✗ cargo-fmt     formatter   Cargo 기반 포매터 (미설치)
  ✗ clippy        linter      Rust 린터 (미설치)
```

---

## Advanced Capabilities

### 14. Parallel Execution Control

**Flag**: `--workers <n>`

**Description**: Control number of parallel workers

**Default**: CPU core count

**Example**:
```bash
# Use 8 workers
gz-quality run --workers 8

# Single worker (sequential)
gz-quality run --workers 1

# Auto-detect CPU cores (default)
gz-quality run
```

**Performance Impact**:
- More workers = faster execution (up to CPU limit)
- Too many workers = resource contention
- Optimal: CPU core count or slightly less

---

### 15. Dry Run Mode

**Flag**: `--dry-run`

**Description**: Show execution plan without running tools

**Use Cases**:
- Preview what will be executed
- Validate configuration
- Debug tool selection

**Example**:
```bash
# Preview execution plan
gz-quality run --dry-run

# With verbose output
gz-quality run --dry-run --verbose
```

**Output**:
```
📋 실행 계획 (dry-run):

실행할 도구: 5개
처리할 파일: 150개

Task 1: gofumpt
  - Priority: 10
  - Files: 45 (*.go)
  - Command: gofumpt -w file1.go file2.go ...

Task 2: black
  - Priority: 10
  - Files: 30 (*.py)
  - Command: black file1.py file2.py ...

Task 3: prettier
  - Priority: 10
  - Files: 75 (*.ts, *.tsx, *.js)
  - Command: prettier --write file1.ts file2.tsx ...

Task 4: ruff
  - Priority: 7
  - Files: 30 (*.py)
  - Command: ruff check --fix file1.py file2.py ...

Task 5: golangci-lint
  - Priority: 5
  - Files: 45 (*.go)
  - Command: golangci-lint run file1.go file2.go ...

예상 실행 시간: ~3.5s
```

---

### 16. Report Generation

**Flag**: `--report <format>`

**Formats**: `json`, `html`, `markdown`

**Output Flag**: `--output <path>`

**Use Cases**:
- CI/CD integration (JSON)
- Visual review (HTML)
- PR comments (Markdown)

**Example**:
```bash
# JSON report
gz-quality check --report json --output quality-report.json

# HTML report
gz-quality check --report html --output quality-report.html

# Markdown report
gz-quality check --report markdown --output quality-report.md
```

**JSON Report Structure**:
```json
{
  "timestamp": "2025-12-01T10:30:00Z",
  "project_root": "/path/to/project",
  "total_files": 150,
  "duration": "3.5s",
  "summary": {
    "total_tools": 5,
    "successful_tools": 3,
    "failed_tools": 2,
    "total_issues": 8,
    "error_issues": 2,
    "warning_issues": 6,
    "info_issues": 0,
    "files_with_issues": 5
  },
  "tool_results": [
    {
      "tool": "gofumpt",
      "language": "Go",
      "success": true,
      "files_processed": 45,
      "duration": "1.2s",
      "issues": []
    },
    {
      "tool": "golangci-lint",
      "language": "Go",
      "success": false,
      "files_processed": 45,
      "duration": "3.2s",
      "issues": [
        {
          "file": "main.go",
          "line": 42,
          "column": 15,
          "severity": "warning",
          "rule": "unused",
          "message": "variable 'x' is unused"
        }
      ]
    }
  ],
  "issues_by_file": {
    "main.go": [...],
    "utils.go": [...]
  }
}
```

---

### 17. Verbose Output

**Flag**: `--verbose, -v`

**Description**: Enable detailed logging

**Use Cases**:
- Debugging tool execution
- Understanding performance
- Troubleshooting failures

**Example**:
```bash
gz-quality run --verbose
gz-quality run -v
```

**Verbose Output**:
```
[DEBUG] Loading configuration from .gzquality.yml
[DEBUG] Detected languages: Go, Python, TypeScript
[DEBUG] Registered tools: gofumpt, black, prettier, golangci-lint, ruff
[DEBUG] Git staged files: 5 files
[INFO] Execution plan: 3 tools, 5 files
[DEBUG] Starting worker pool with 4 workers
[DEBUG] Task 1: gofumpt [main.go, utils.go]
[DEBUG] Running: gofumpt -w main.go utils.go
[DEBUG] gofumpt completed in 0.2s
[DEBUG] Task 2: black [script.py]
[DEBUG] Running: black script.py
[DEBUG] black completed in 0.3s
[INFO] All tasks completed successfully
```

---

### 18. Timeout Control

**Flag**: `--timeout <duration>`

**Config**: `timeout` in `.gzquality.yml`

**Default**: 10 minutes

**Example**:
```bash
# 5 minute timeout
gz-quality run --timeout 5m

# 30 second timeout
gz-quality run --timeout 30s
```

**Config**:
```yaml
timeout: "15m"  # 15 minutes
```

---

### 19. File Pattern Filtering

**Flag**: `--files <pattern>`

**Description**: Process specific file patterns

**Use Cases**:
- Test specific directories
- Debug single file
- Selective quality checks

**Example**:
```bash
# Single file
gz-quality run --files "main.go"

# Glob pattern
gz-quality run --files "src/**/*.go"

# Multiple patterns
gz-quality run --files "**/*.py,**/*.go"

# Specific directory
gz-quality run --files "internal/**"
```

---

### 20. Auto-fix Mode

**Flag**: `--fix, -x`

**Description**: Apply automatic fixes from tools

**Use Cases**:
- Quick formatting
- Auto-correct linting issues
- Batch code cleanup

**Example**:
```bash
# Auto-fix all issues
gz-quality run --fix

# Auto-fix staged files
gz-quality run --staged --fix

# Short form
gz-quality run -x
```

**Behavior**:
- Formatters always modify files (gofumpt, black, prettier)
- Linters apply fixes when available (ruff --fix, eslint --fix)
- Some linters have no auto-fix (golangci-lint)

---

## Integration Capabilities

### 21. Pre-commit Hook Integration

**Manual Hook**:
```bash
#!/bin/bash
# .git/hooks/pre-commit
gz-quality run --staged --fix
exit $?
```

**pre-commit Framework**:
```yaml
# .pre-commit-config.yaml
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

---

### 22. CI/CD Integration

**GitHub Actions**:
```yaml
name: Quality Check
on: [push, pull_request]

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history for --since

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install gz-quality
        run: |
          go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest
          echo "$(go env GOPATH)/bin" >> $GITHUB_PATH

      - name: Install tools
        run: gz-quality install

      - name: Quality Check
        run: |
          if [ "${{ github.event_name }}" == "pull_request" ]; then
            gz-quality check --since origin/${{ github.base_ref }}
          else
            gz-quality check
          fi

      - name: Generate Report
        if: always()
        run: |
          gz-quality check --report json --output quality-report.json

      - name: Upload Report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: quality-report
          path: quality-report.json
```

---

### 23. Docker Integration

**Dockerfile**:
```dockerfile
FROM golang:1.24-alpine AS builder

# Install gz-quality
RUN go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest

FROM alpine:latest
COPY --from=builder /go/bin/gz-quality /usr/local/bin/

# Install runtime dependencies
RUN apk add --no-cache git

WORKDIR /workspace
ENTRYPOINT ["gz-quality"]
CMD ["run"]
```

**Usage**:
```bash
# Build image
docker build -t gz-quality:latest .

# Run check
docker run --rm -v $(pwd):/workspace gz-quality:latest run

# CI usage
docker run --rm -v $(pwd):/workspace gz-quality:latest check --since main
```

---

## Error Handling

### Exit Codes

| Code | Meaning | Example Scenario |
|------|---------|------------------|
| 0 | Success | No issues found, all tools passed |
| 1 | Issues found | Linting errors detected |
| 2 | Execution error | Tool not found, config error, Git error |

### Example Usage in Scripts

```bash
#!/bin/bash

# Run quality check
gz-quality check --staged

# Capture exit code
EXIT_CODE=$?

# Handle result
if [ $EXIT_CODE -eq 0 ]; then
  echo "✅ Quality check passed"
  exit 0
elif [ $EXIT_CODE -eq 1 ]; then
  echo "⚠️ Quality issues found"
  echo "Run 'gz-quality run --staged --fix' to auto-fix"
  exit 1
else
  echo "❌ Quality check failed with error"
  exit 2
fi
```

---

## Performance Optimization

### Best Practices

1. **Use Incremental Checks**:
   ```bash
   # Faster: Check only changed files
   gz-quality check --changed

   # Slower: Check entire project
   gz-quality check
   ```

2. **Optimize Worker Count**:
   ```bash
   # Match CPU cores
   gz-quality run --workers $(nproc)

   # Low-resource environments
   gz-quality run --workers 2
   ```

3. **Disable Slow Tools Locally**:
   ```yaml
   # .gzquality.yml
   tools:
     pylint:
       enabled: false  # Slow, disable locally
     golangci-lint:
       enabled: true
       timeout: "2m"   # Add timeout
   ```

4. **Use Caching in CI**:
   ```yaml
   # GitHub Actions
   - uses: actions/cache@v3
     with:
       path: |
         ~/.cache/golangci-lint
         ~/.cache/pip
       key: ${{ runner.os }}-quality-${{ hashFiles('**/go.sum', '**/requirements.txt') }}
   ```

---

**Last Updated**: 2025-12-01
**Related**: See `CONTEXT.md` for architecture details
