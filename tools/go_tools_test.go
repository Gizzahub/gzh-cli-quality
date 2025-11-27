// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGofumptTool(t *testing.T) {
	tool := NewGofumptTool()

	assert.NotNil(t, tool)
	assert.Equal(t, "gofumpt", tool.Name())
	assert.Equal(t, "Go", tool.Language())
	assert.Equal(t, FORMAT, tool.Type())

	// Check config file patterns by creating a temp project with a config
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, ".gofumpt")
	err := os.WriteFile(configFile, []byte("# gofumpt config"), 0o644)
	assert.NoError(t, err)

	configs := tool.FindConfigFiles(tmpDir)
	assert.Contains(t, configs, configFile)
}

func TestGofumptTool_BuildCommand(t *testing.T) {
	tool := NewGofumptTool()

	tests := []struct {
		name        string
		files       []string
		options     ExecuteOptions
		expectedArgs []string
	}{
		{
			name:  "basic Go files",
			files: []string{"main.go", "utils.go"},
			options: ExecuteOptions{
				ProjectRoot: "/test/project",
			},
			expectedArgs: []string{"-w", "main.go", "utils.go"},
		},
		{
			name:  "with extra args",
			files: []string{"main.go"},
			options: ExecuteOptions{
				ExtraArgs: []string{"-l", "-s"},
			},
			expectedArgs: []string{"-w", "-l", "-s", "main.go"},
		},
		{
			name:  "filters non-Go files",
			files: []string{"main.go", "test.py", "README.md"},
			options: ExecuteOptions{},
			expectedArgs: []string{"-w", "main.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tool.BuildCommand(tt.files, tt.options)

			assert.Equal(t, "gofumpt", filepath.Base(cmd.Path))
			assert.Equal(t, tt.expectedArgs, cmd.Args[1:]) // Skip cmd.Args[0] which is the executable

			if tt.options.ProjectRoot != "" {
				assert.Equal(t, tt.options.ProjectRoot, cmd.Dir)
			}
		})
	}
}

func TestNewGoimportsTool(t *testing.T) {
	tool := NewGoimportsTool()

	assert.NotNil(t, tool)
	assert.Equal(t, "goimports", tool.Name())
	assert.Equal(t, "Go", tool.Language())
	assert.Equal(t, FORMAT, tool.Type())
}

func TestGoimportsTool_BuildCommand(t *testing.T) {
	tool := NewGoimportsTool()

	tests := []struct {
		name        string
		files       []string
		options     ExecuteOptions
		expectedArgs []string
		checkLocal   bool
	}{
		{
			name:  "basic Go files",
			files: []string{"main.go"},
			options: ExecuteOptions{},
			expectedArgs: []string{"-w", "main.go"},
		},
		{
			name:  "filters non-Go files",
			files: []string{"main.go", "test.js", "config.yml"},
			options: ExecuteOptions{},
			expectedArgs: []string{"-w", "main.go"},
		},
		{
			name:  "with extra args",
			files: []string{"main.go"},
			options: ExecuteOptions{
				ExtraArgs: []string{"-v"},
			},
			expectedArgs: []string{"-w", "-v", "main.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tool.BuildCommand(tt.files, tt.options)

			assert.Equal(t, "goimports", filepath.Base(cmd.Path))
			// Check that args contain expected elements (order may vary with -local flag)
			cmdArgs := cmd.Args[1:]
			for _, expected := range tt.expectedArgs {
				assert.Contains(t, cmdArgs, expected)
			}
		})
	}
}

func TestNewGolangciLintTool(t *testing.T) {
	tool := NewGolangciLintTool()

	assert.NotNil(t, tool)
	assert.Equal(t, "golangci-lint", tool.Name())
	assert.Equal(t, "Go", tool.Language())
	assert.Equal(t, LINT, tool.Type())

	// Check config file patterns by creating a temp project with a config
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, ".golangci.yml")
	err := os.WriteFile(configFile, []byte("# golangci-lint config"), 0o644)
	assert.NoError(t, err)

	configs := tool.FindConfigFiles(tmpDir)
	assert.Contains(t, configs, configFile)
}

func TestGolangciLintTool_BuildCommand(t *testing.T) {
	tool := NewGolangciLintTool()

	tests := []struct {
		name     string
		files    []string
		options  ExecuteOptions
		checkFix bool
		checkConfig bool
	}{
		{
			name:  "basic lint",
			files: []string{"main.go"},
			options: ExecuteOptions{
				ProjectRoot: "/test/project",
			},
		},
		{
			name:  "with fix flag",
			files: []string{"main.go"},
			options: ExecuteOptions{
				Fix: true,
			},
			checkFix: true,
		},
		{
			name:  "with config file",
			files: []string{"main.go"},
			options: ExecuteOptions{
				ConfigFile: ".golangci.yml",
			},
			checkConfig: true,
		},
		{
			name:  "multiple Go files",
			files: []string{"main.go", "utils.go", "config.go"},
			options: ExecuteOptions{},
		},
		{
			name:  "no files (all packages)",
			files: []string{},
			options: ExecuteOptions{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tool.BuildCommand(tt.files, tt.options)

			assert.Equal(t, "golangci-lint", filepath.Base(cmd.Path))
			cmdArgs := cmd.Args[1:]

			assert.Contains(t, cmdArgs, "run")
			assert.Contains(t, cmdArgs, "--out-format")
			assert.Contains(t, cmdArgs, "json")

			if tt.checkFix {
				assert.Contains(t, cmdArgs, "--fix")
			}

			if tt.checkConfig {
				assert.Contains(t, cmdArgs, "-c")
				assert.Contains(t, cmdArgs, ".golangci.yml")
			}

			if len(tt.files) == 0 {
				assert.Contains(t, cmdArgs, "./...")
			}

			if tt.options.ProjectRoot != "" {
				assert.Equal(t, tt.options.ProjectRoot, cmd.Dir)
			}
		})
	}
}

func TestGolangciLintTool_ParseOutput(t *testing.T) {
	tool := NewGolangciLintTool()

	tests := []struct {
		name     string
		output   string
		expected int
		checkIssue func(*testing.T, Issue)
	}{
		{
			name:   "empty output",
			output: "",
			expected: 0,
		},
		{
			name: "valid JSON output",
			output: `{
				"Issues": [
					{
						"FromLinter": "errcheck",
						"Text": "Error return value not checked",
						"Severity": "error",
						"Pos": {
							"Filename": "main.go",
							"Line": 42,
							"Column": 15
						}
					}
				]
			}`,
			expected: 1,
			checkIssue: func(t *testing.T, issue Issue) {
				assert.Equal(t, "main.go", issue.File)
				assert.Equal(t, 42, issue.Line)
				assert.Equal(t, 15, issue.Column)
				assert.Equal(t, "error", issue.Severity)
				assert.Equal(t, "errcheck", issue.Rule)
				assert.Equal(t, "Error return value not checked", issue.Message)
			},
		},
		{
			name: "multiple issues",
			output: `{
				"Issues": [
					{
						"FromLinter": "errcheck",
						"Text": "Error not checked",
						"Severity": "error",
						"Pos": {
							"Filename": "main.go",
							"Line": 10,
							"Column": 5
						}
					},
					{
						"FromLinter": "unused",
						"Text": "Unused variable 'x'",
						"Severity": "warning",
						"Pos": {
							"Filename": "utils.go",
							"Line": 20,
							"Column": 10
						}
					}
				]
			}`,
			expected: 2,
		},
		{
			name: "with replacement suggestion",
			output: `{
				"Issues": [
					{
						"FromLinter": "gofmt",
						"Text": "File is not formatted",
						"Severity": "error",
						"Pos": {
							"Filename": "main.go",
							"Line": 5,
							"Column": 1
						},
						"Replacement": {
							"NewLines": ["formatted line 1", "formatted line 2"]
						}
					}
				]
			}`,
			expected: 1,
			checkIssue: func(t *testing.T, issue Issue) {
				assert.NotEmpty(t, issue.Suggestion)
				assert.Contains(t, issue.Suggestion, "formatted line 1")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := tool.ParseOutput(tt.output)

			assert.Len(t, issues, tt.expected)

			if tt.checkIssue != nil && len(issues) > 0 {
				tt.checkIssue(t, issues[0])
			}
		})
	}
}

func TestGolangciLintTool_ParseTextOutput(t *testing.T) {
	tool := NewGolangciLintTool()

	tests := []struct {
		name     string
		output   string
		expected int
	}{
		{
			name:   "empty output",
			output: "",
			expected: 0,
		},
		{
			name:   "single text issue",
			output: "main.go:42:15: Error return value not checked (errcheck)",
			expected: 1,
		},
		{
			name: "multiple text issues",
			output: `main.go:10:5: Error not checked (errcheck)
utils.go:20:10: Unused variable 'x' (unused)
config.go:30:1: Missing comment (golint)`,
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := tool.parseTextOutput(tt.output)
			assert.Len(t, issues, tt.expected)
		})
	}
}

func TestGoTools_InterfaceCompliance(t *testing.T) {
	// Ensure all Go tools implement QualityTool interface
	var _ QualityTool = (*GofumptTool)(nil)
	var _ QualityTool = (*GoimportsTool)(nil)
	var _ QualityTool = (*GolangciLintTool)(nil)

	tools := []QualityTool{
		NewGofumptTool(),
		NewGoimportsTool(),
		NewGolangciLintTool(),
	}

	for _, tool := range tools {
		t.Run(tool.Name(), func(t *testing.T) {
			assert.NotEmpty(t, tool.Name())
			assert.NotEmpty(t, tool.Language())
			assert.NotNil(t, tool.Type())
		})
	}
}

func TestGoTools_Execute_NotAvailable(t *testing.T) {
	tool := NewGofumptTool()

	// Override executable to non-existent command
	tool.executable = "nonexistent-gofumpt-xyz"

	ctx := context.Background()
	result, err := tool.Execute(ctx, []string{"main.go"}, ExecuteOptions{})

	assert.NoError(t, err) // Execute returns error in result, not as error
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.NotNil(t, result.Error)
}
