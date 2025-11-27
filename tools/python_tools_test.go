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

func TestNewBlackTool(t *testing.T) {
	tool := NewBlackTool()

	assert.NotNil(t, tool)
	assert.Equal(t, "black", tool.Name())
	assert.Equal(t, "Python", tool.Language())
	assert.Equal(t, FORMAT, tool.Type())

	// Check config file patterns
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "pyproject.toml")
	err := os.WriteFile(configFile, []byte("[tool.black]\nline-length = 88"), 0o644)
	assert.NoError(t, err)

	configs := tool.FindConfigFiles(tmpDir)
	assert.Contains(t, configs, configFile)
}

func TestBlackTool_BuildCommand(t *testing.T) {
	tool := NewBlackTool()

	tests := []struct {
		name         string
		files        []string
		options      ExecuteOptions
		expectedArgs []string
	}{
		{
			name:  "basic Python files",
			files: []string{"main.py", "utils.py"},
			options: ExecuteOptions{},
			expectedArgs: []string{"--line-length", "88", "main.py", "utils.py"},
		},
		{
			name:  "with config file",
			files: []string{"main.py"},
			options: ExecuteOptions{
				ConfigFile: "pyproject.toml",
			},
			expectedArgs: []string{"--config", "pyproject.toml"},
		},
		{
			name:  "filters non-Python files",
			files: []string{"main.py", "test.go", "README.md"},
			options: ExecuteOptions{},
			expectedArgs: []string{"main.py"},
		},
		{
			name:  "no files - formats current directory",
			files: []string{},
			options: ExecuteOptions{},
			expectedArgs: []string{"."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tool.BuildCommand(tt.files, tt.options)

			assert.Equal(t, "black", filepath.Base(cmd.Path))
			cmdArgs := cmd.Args[1:]

			for _, expected := range tt.expectedArgs {
				assert.Contains(t, cmdArgs, expected)
			}
		})
	}
}

func TestNewRuffTool(t *testing.T) {
	tool := NewRuffTool()

	assert.NotNil(t, tool)
	assert.Equal(t, "ruff", tool.Name())
	assert.Equal(t, "Python", tool.Language())
	assert.Equal(t, BOTH, tool.Type())

	// Check config file patterns
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "ruff.toml")
	err := os.WriteFile(configFile, []byte("line-length = 88"), 0o644)
	assert.NoError(t, err)

	configs := tool.FindConfigFiles(tmpDir)
	assert.Contains(t, configs, configFile)
}

func TestRuffTool_BuildCommand(t *testing.T) {
	tool := NewRuffTool()

	tests := []struct {
		name     string
		files    []string
		options  ExecuteOptions
		checkFix bool
		checkFormat bool
	}{
		{
			name:  "lint mode",
			files: []string{"main.py"},
			options: ExecuteOptions{},
		},
		{
			name:  "format mode",
			files: []string{"main.py"},
			options: ExecuteOptions{
				FormatOnly: true,
			},
			checkFormat: true,
		},
		{
			name:  "lint with fix",
			files: []string{"main.py"},
			options: ExecuteOptions{
				Fix: true,
			},
			checkFix: true,
		},
		{
			name:  "with config file",
			files: []string{"main.py"},
			options: ExecuteOptions{
				ConfigFile: "ruff.toml",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tool.BuildCommand(tt.files, tt.options)

			assert.Equal(t, "ruff", filepath.Base(cmd.Path))
			cmdArgs := cmd.Args[1:]

			if tt.checkFormat {
				assert.Contains(t, cmdArgs, "format")
			} else {
				assert.Contains(t, cmdArgs, "check")
			}

			if tt.checkFix && !tt.checkFormat {
				assert.Contains(t, cmdArgs, "--fix")
			}

			if tt.options.ConfigFile != "" {
				assert.Contains(t, cmdArgs, "--config")
			}
		})
	}
}

func TestRuffTool_ParseOutput(t *testing.T) {
	tool := NewRuffTool()

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
			output: `[
				{
					"code": "E501",
					"message": "Line too long",
					"filename": "main.py",
					"location": {
						"row": 10,
						"column": 80
					},
					"end_location": {
						"row": 10,
						"column": 100
					}
				}
			]`,
			expected: 1,
			checkIssue: func(t *testing.T, issue Issue) {
				assert.Equal(t, "main.py", issue.File)
				assert.Equal(t, 10, issue.Line)
				assert.Equal(t, 80, issue.Column)
				assert.Equal(t, "E501", issue.Rule)
				assert.Equal(t, "Line too long", issue.Message)
			},
		},
		{
			name: "with fix suggestion",
			output: `[
				{
					"code": "F401",
					"message": "Unused import",
					"filename": "main.py",
					"location": {
						"row": 5,
						"column": 1
					},
					"end_location": {
						"row": 5,
						"column": 20
					},
					"fix": {
						"content": ""
					}
				}
			]`,
			expected: 1,
			checkIssue: func(t *testing.T, issue Issue) {
				assert.Equal(t, "", issue.Suggestion)
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

func TestNewPylintTool(t *testing.T) {
	tool := NewPylintTool()

	assert.NotNil(t, tool)
	assert.Equal(t, "pylint", tool.Name())
	assert.Equal(t, "Python", tool.Language())
	assert.Equal(t, LINT, tool.Type())
}

func TestPylintTool_BuildCommand(t *testing.T) {
	tool := NewPylintTool()

	tests := []struct {
		name    string
		files   []string
		options ExecuteOptions
	}{
		{
			name:  "basic Python files",
			files: []string{"main.py"},
			options: ExecuteOptions{},
		},
		{
			name:  "with config file",
			files: []string{"main.py"},
			options: ExecuteOptions{
				ConfigFile: ".pylintrc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tool.BuildCommand(tt.files, tt.options)

			assert.Equal(t, "pylint", filepath.Base(cmd.Path))
			cmdArgs := cmd.Args[1:]

			assert.Contains(t, cmdArgs, "--output-format")
			assert.Contains(t, cmdArgs, "json")

			if tt.options.ConfigFile != "" {
				assert.Contains(t, cmdArgs, "--rcfile")
			}
		})
	}
}

func TestPylintTool_ParseOutput(t *testing.T) {
	tool := NewPylintTool()

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
			name: "valid JSON output",
			output: `[
				{
					"type": "error",
					"module": "main",
					"obj": "",
					"line": 10,
					"column": 5,
					"path": "main.py",
					"symbol": "undefined-variable",
					"message": "Undefined variable 'x'",
					"message-id": "E0602"
				}
			]`,
			expected: 1,
		},
		{
			name: "multiple severity levels",
			output: `[
				{
					"type": "error",
					"line": 10,
					"column": 5,
					"path": "main.py",
					"symbol": "error-symbol",
					"message": "Error message",
					"message-id": "E0001"
				},
				{
					"type": "warning",
					"line": 20,
					"column": 10,
					"path": "main.py",
					"symbol": "warning-symbol",
					"message": "Warning message",
					"message-id": "W0001"
				}
			]`,
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := tool.ParseOutput(tt.output)
			assert.Len(t, issues, tt.expected)
		})
	}
}

func TestPythonTools_InterfaceCompliance(t *testing.T) {
	// Ensure all Python tools implement QualityTool interface
	var _ QualityTool = (*BlackTool)(nil)
	var _ QualityTool = (*RuffTool)(nil)
	var _ QualityTool = (*PylintTool)(nil)

	tools := []QualityTool{
		NewBlackTool(),
		NewRuffTool(),
		NewPylintTool(),
	}

	for _, tool := range tools {
		t.Run(tool.Name(), func(t *testing.T) {
			assert.NotEmpty(t, tool.Name())
			assert.NotEmpty(t, tool.Language())
			assert.NotNil(t, tool.Type())
		})
	}
}

func TestPythonTools_Execute_NotAvailable(t *testing.T) {
	tool := NewBlackTool()

	// Override executable to non-existent command
	tool.executable = "nonexistent-black-xyz"

	ctx := context.Background()
	result, err := tool.Execute(ctx, []string{"main.py"}, ExecuteOptions{})

	assert.NoError(t, err) // Execute returns error in result, not as error
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.NotNil(t, result.Error)
}
