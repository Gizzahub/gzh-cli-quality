# Testing Guide

## Overview

This document describes the testing strategy, best practices, and guidelines for the gzh-cli-quality project.

## Test Coverage

### Current Coverage (As of 2025-11-29)

**Total Project Coverage: 76.2%** ✅

| Package | Coverage | Status | Notes |
|---------|----------|--------|-------|
| report | 95.3% | ✅ Excellent | Highest coverage |
| git | 92.0% | ✅ Excellent | |
| detector | 91.8% | ✅ Excellent | |
| config | 85.1% | ✅ Excellent | |
| executor | 80.0% | ✅ Excellent | |
| tools | 78.5% | ✅ Good | Exceeds 75% target |
| root | 57.4% | ⚠️ Moderate | Integration tests |
| cmd/gz-quality | 0.0% | ⚠️ Expected | main package |

### Coverage Goals

- **Minimum Target**: 70% total coverage
- **Good Target**: 75% package coverage
- **Excellent**: 85%+ package coverage
- **Exceptions**: main packages (cmd/*) are expected to have 0% coverage

## Test Organization

### Directory Structure

```
gzh-cli-quality/
├── quality_test.go              # Root package tests (392 lines)
├── cmd/
│   └── gz-quality/
│       └── main_test.go         # CLI integration tests (226 lines)
├── config/
│   └── config_test.go           # Config tests (modified)
├── detector/
│   ├── detector_test.go         # Core detector tests
│   └── tools_test.go            # Tool detection tests (352 lines)
└── tools/
    ├── base_test.go             # Base tool tests
    ├── go_tools_test.go         # Go tools (468 lines)
    ├── python_tools_test.go     # Python tools (393 lines)
    ├── javascript_tools_test.go # JS/TS tools (407 lines)
    └── rust_tools_test.go       # Rust tools (354 lines)
```

**Total Test Code**: 2,812 lines across 8 test files

## Test Types

### 1. Unit Tests

Test individual functions and methods in isolation.

**Example**: Tool initialization and configuration

```go
func TestNewGofumptTool(t *testing.T) {
    tool := NewGofumptTool()

    assert.NotNil(t, tool)
    assert.Equal(t, "gofumpt", tool.Name())
    assert.Equal(t, "Go", tool.Language())
    assert.Equal(t, FORMAT, tool.Type())
}
```

**Coverage**: ~80% of tests

### 2. Integration Tests

Test interaction between components.

**Example**: Command execution flow

```go
func TestRunQuality_DryRun(t *testing.T) {
    manager := NewQualityManager()
    cmd := manager.newRunCmd()

    tmpDir := t.TempDir()
    // ... setup test files

    cmd.SetArgs([]string{"--dry-run"})
    err := cmd.Execute()
    assert.NoError(t, err)
}
```

**Coverage**: ~15% of tests

### 3. Table-Driven Tests

Test multiple scenarios with the same logic.

**Example**: Command building variations

```go
func TestGofumptTool_BuildCommand(t *testing.T) {
    tool := NewGofumptTool()

    tests := []struct {
        name        string
        files       []string
        options     ExecuteOptions
        expectedArgs []string
    }{
        {
            name: "basic Go files",
            files: []string{"main.go", "utils.go"},
            options: ExecuteOptions{},
            expectedArgs: []string{"-w", "main.go", "utils.go"},
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := tool.BuildCommand(tt.files, tt.options)
            // ... assertions
        })
    }
}
```

**Coverage**: ~5% of tests, but tests many scenarios

## Testing Best Practices

### 1. Use Test Isolation

Always use `t.TempDir()` for file operations:

```go
func TestFileOperation(t *testing.T) {
    tmpDir := t.TempDir() // Automatically cleaned up

    testFile := filepath.Join(tmpDir, "test.go")
    err := os.WriteFile(testFile, []byte("package main"), 0o644)
    require.NoError(t, err)
}
```

### 2. Cleanup After Tests

Restore original state when modifying global state:

```go
func TestWithDirectoryChange(t *testing.T) {
    origDir, err := os.Getwd()
    require.NoError(t, err)
    defer func() {
        err := os.Chdir(origDir)
        require.NoError(t, err)
    }()

    err = os.Chdir(tmpDir)
    require.NoError(t, err)

    // ... test code
}
```

### 3. Test Error Paths

Don't just test happy paths:

```go
func TestGenerateReport_UnsupportedFormat(t *testing.T) {
    manager := NewQualityManager()

    err := manager.generateReport(results, time.Second, 0, tmpDir, "pdf", tmpDir+"/report.pdf")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "unsupported report format")
}
```

### 4. Use Mocks for External Dependencies

Create mock implementations for testing:

```go
type mockTool struct {
    name     string
    language string
    toolType tools.ToolType
}

func (m *mockTool) Name() string { return m.name }
func (m *mockTool) Language() string { return m.language }
// ... implement other QualityTool methods
```

### 5. Test Cross-Platform Compatibility

Handle platform-specific differences:

```go
func TestFindConfigFile(t *testing.T) {
    // Resolve symlinks for both paths (macOS has /var -> /private/var)
    expectedPath, err := filepath.EvalSymlinks(configPath)
    require.NoError(t, err)

    actualPath, err := filepath.EvalSymlinks(found)
    require.NoError(t, err)

    assert.Equal(t, expectedPath, actualPath)
}
```

## Running Tests

### Run All Tests

```bash
go test ./...
```

### Run Tests with Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out -covermode=set ./...

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

### Run Specific Package Tests

```bash
go test ./tools
go test ./detector
go test ./config
```

### Run Specific Test

```bash
go test -run TestNewGofumptTool ./tools
go test -run "TestRunQuality|TestRunCheck" .
```

### Run Tests with Verbose Output

```bash
go test -v ./...
```

### Run Tests with Race Detector

```bash
go test -race ./...
```

## Code Coverage Guidelines

### What to Test

✅ **Always Test**:
- Public API functions and methods
- Error handling paths
- Edge cases and boundary conditions
- Integration between components
- Configuration parsing and validation

⚠️ **Consider Testing**:
- Internal helper functions (if complex)
- Private methods with significant logic
- CLI command handlers (integration style)

❌ **Don't Test**:
- Simple getters/setters
- Trivial helper functions
- main() functions
- Generated code

### Coverage Targets by Package Type

| Package Type | Minimum | Target | Excellent |
|--------------|---------|--------|-----------|
| Core Logic | 70% | 80% | 90%+ |
| Tools/Utilities | 65% | 75% | 85%+ |
| CLI Commands | 40% | 50% | 60%+ |
| Main Packages | - | 0% | 0% |

## Test Maintenance

### When Adding New Features

1. **Write tests first** (TDD) or **immediately after** implementation
2. **Ensure coverage** doesn't decrease
3. **Add integration tests** for new commands/features
4. **Update this documentation** if testing strategy changes

### When Fixing Bugs

1. **Write a failing test** that reproduces the bug
2. **Fix the bug** to make the test pass
3. **Ensure no regressions** by running full test suite
4. **Commit test with fix** in same commit

### Regular Maintenance

- **Weekly**: Review coverage reports for gaps
- **Before Release**: Ensure all tests pass with `-race` flag
- **After Major Changes**: Update integration tests

## Troubleshooting

### Tests Fail on macOS but Pass on Linux

**Issue**: Path symlink differences (`/var` vs `/private/var`)

**Solution**: Use `filepath.EvalSymlinks()` to resolve symlinks before comparison

```go
expectedPath, err := filepath.EvalSymlinks(configPath)
actualPath, err := filepath.EvalSymlinks(found)
assert.Equal(t, expectedPath, actualPath)
```

### Tests are Flaky

**Possible Causes**:
- Race conditions (use `-race` flag)
- Timing issues (add appropriate waits)
- Shared global state (ensure isolation)
- File system issues (use `t.TempDir()`)

### Coverage is Lower Than Expected

**Check**:
1. Are you testing error paths?
2. Are table-driven tests covering all scenarios?
3. Are there untested helper functions?
4. Run `go tool cover -html=coverage.out` to see uncovered lines

## Continuous Integration

### GitHub Actions Workflow

Tests run automatically on:
- Every push to main branch
- Every pull request
- Weekly scheduled run

See `.github/workflows/coverage.yml` for configuration.

### Coverage Reporting

- Coverage report generated on every CI run
- HTML report available as artifact
- Badge updated in README.md

## Contributing

When contributing tests:

1. **Follow existing patterns**: Use table-driven tests where appropriate
2. **Test both success and failure**: Don't just test happy paths
3. **Use meaningful test names**: `TestFunctionName_Scenario`
4. **Keep tests focused**: One test should test one thing
5. **Document complex tests**: Add comments for non-obvious test logic

## References

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Package](https://github.com/stretchr/testify)
- [Table-Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Go Code Coverage](https://go.dev/blog/cover)

---

**Last Updated**: 2025-11-29
**Coverage**: 76.2% (Total Project)
