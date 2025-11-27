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

func TestNewPrettierTool(t *testing.T) {
	tool := NewPrettierTool()

	assert.NotNil(t, tool)
	assert.Equal(t, "prettier", tool.Name())
	assert.Equal(t, "JavaScript", tool.Language())
	assert.Equal(t, FORMAT, tool.Type())

	// Check config file patterns
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, ".prettierrc")
	err := os.WriteFile(configFile, []byte(`{"semi": false}`), 0o644)
	assert.NoError(t, err)

	configs := tool.FindConfigFiles(tmpDir)
	assert.Contains(t, configs, configFile)
}

func TestPrettierTool_BuildCommand(t *testing.T) {
	tool := NewPrettierTool()

	tests := []struct {
		name         string
		files        []string
		options      ExecuteOptions
		expectedArgs []string
	}{
		{
			name:  "basic JS files",
			files: []string{"main.js", "utils.js"},
			options: ExecuteOptions{},
			expectedArgs: []string{"--write", "main.js", "utils.js"},
		},
		{
			name:  "TypeScript files",
			files: []string{"main.ts", "types.tsx"},
			options: ExecuteOptions{},
			expectedArgs: []string{"--write", "main.ts", "types.tsx"},
		},
		{
			name:  "with config file",
			files: []string{"main.js"},
			options: ExecuteOptions{
				ConfigFile: ".prettierrc",
			},
			expectedArgs: []string{"--write", "--config", ".prettierrc"},
		},
		{
			name:  "filters unsupported files",
			files: []string{"main.js", "test.go", "README.txt"},
			options: ExecuteOptions{},
			expectedArgs: []string{"main.js"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tool.BuildCommand(tt.files, tt.options)

			assert.Equal(t, "prettier", filepath.Base(cmd.Path))
			cmdArgs := cmd.Args[1:]

			for _, expected := range tt.expectedArgs {
				assert.Contains(t, cmdArgs, expected)
			}
		})
	}
}

func TestNewESLintTool(t *testing.T) {
	tool := NewESLintTool()

	assert.NotNil(t, tool)
	assert.Equal(t, "eslint", tool.Name())
	assert.Equal(t, "JavaScript", tool.Language())
	assert.Equal(t, LINT, tool.Type())

	// Check config file patterns
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, ".eslintrc.json")
	err := os.WriteFile(configFile, []byte(`{"rules": {}}`), 0o644)
	assert.NoError(t, err)

	configs := tool.FindConfigFiles(tmpDir)
	assert.Contains(t, configs, configFile)
}

func TestESLintTool_BuildCommand(t *testing.T) {
	tool := NewESLintTool()

	tests := []struct {
		name     string
		files    []string
		options  ExecuteOptions
		checkFix bool
		checkConfig bool
	}{
		{
			name:  "basic lint",
			files: []string{"main.js"},
			options: ExecuteOptions{},
		},
		{
			name:  "with fix flag",
			files: []string{"main.js"},
			options: ExecuteOptions{
				Fix: true,
			},
			checkFix: true,
		},
		{
			name:  "with config file",
			files: []string{"main.js"},
			options: ExecuteOptions{
				ConfigFile: ".eslintrc.json",
			},
			checkConfig: true,
		},
		{
			name:  "TypeScript and JSX files",
			files: []string{"App.tsx", "Button.jsx"},
			options: ExecuteOptions{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tool.BuildCommand(tt.files, tt.options)

			assert.Equal(t, "eslint", filepath.Base(cmd.Path))
			cmdArgs := cmd.Args[1:]

			assert.Contains(t, cmdArgs, "--format")
			assert.Contains(t, cmdArgs, "json")

			if tt.checkFix {
				assert.Contains(t, cmdArgs, "--fix")
			}

			if tt.checkConfig {
				assert.Contains(t, cmdArgs, "--config")
				assert.Contains(t, cmdArgs, ".eslintrc.json")
			}
		})
	}
}

func TestESLintTool_ParseOutput(t *testing.T) {
	tool := NewESLintTool()

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
			name: "valid JSON output with errors",
			output: `[
				{
					"filePath": "main.js",
					"messages": [
						{
							"ruleId": "no-unused-vars",
							"severity": 2,
							"message": "Variable 'x' is not used",
							"line": 10,
							"column": 5
						}
					],
					"errorCount": 1,
					"warningCount": 0
				}
			]`,
			expected: 1,
			checkIssue: func(t *testing.T, issue Issue) {
				assert.Equal(t, "main.js", issue.File)
				assert.Equal(t, 10, issue.Line)
				assert.Equal(t, 5, issue.Column)
				assert.Equal(t, "no-unused-vars", issue.Rule)
				assert.Equal(t, "error", issue.Severity)
			},
		},
		{
			name: "warnings",
			output: `[
				{
					"filePath": "utils.js",
					"messages": [
						{
							"ruleId": "no-console",
							"severity": 1,
							"message": "Unexpected console statement",
							"line": 5,
							"column": 3
						}
					],
					"errorCount": 0,
					"warningCount": 1
				}
			]`,
			expected: 1,
			checkIssue: func(t *testing.T, issue Issue) {
				assert.Equal(t, "warning", issue.Severity)
			},
		},
		{
			name: "with fix suggestion",
			output: `[
				{
					"filePath": "main.js",
					"messages": [
						{
							"ruleId": "semi",
							"severity": 2,
							"message": "Missing semicolon",
							"line": 10,
							"column": 15,
							"fix": {
								"range": [100, 100],
								"text": ";"
							}
						}
					],
					"errorCount": 1,
					"warningCount": 0
				}
			]`,
			expected: 1,
			checkIssue: func(t *testing.T, issue Issue) {
				assert.Equal(t, ";", issue.Suggestion)
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

func TestNewTSCTool(t *testing.T) {
	tool := NewTSCTool()

	assert.NotNil(t, tool)
	assert.Equal(t, "tsc", tool.Name())
	assert.Equal(t, "TypeScript", tool.Language())
	assert.Equal(t, LINT, tool.Type())

	// Check config file patterns
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "tsconfig.json")
	err := os.WriteFile(configFile, []byte(`{"compilerOptions": {}}`), 0o644)
	assert.NoError(t, err)

	configs := tool.FindConfigFiles(tmpDir)
	assert.Contains(t, configs, configFile)
}

func TestTSCTool_BuildCommand(t *testing.T) {
	tool := NewTSCTool()

	tests := []struct {
		name    string
		files   []string
		options ExecuteOptions
	}{
		{
			name:  "basic TypeScript files",
			files: []string{"main.ts", "types.d.ts"},
			options: ExecuteOptions{},
		},
		{
			name:  "with config file",
			files: []string{"main.ts"},
			options: ExecuteOptions{
				ConfigFile: "tsconfig.json",
			},
		},
		{
			name:  "no files - use tsconfig",
			files: []string{},
			options: ExecuteOptions{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tool.BuildCommand(tt.files, tt.options)

			assert.Equal(t, "tsc", filepath.Base(cmd.Path))
			cmdArgs := cmd.Args[1:]

			assert.Contains(t, cmdArgs, "--noEmit")
			assert.Contains(t, cmdArgs, "--pretty")
			assert.Contains(t, cmdArgs, "false")

			if tt.options.ConfigFile != "" {
				assert.Contains(t, cmdArgs, "--project")
				assert.Contains(t, cmdArgs, "tsconfig.json")
			}
		})
	}
}

func TestTSCTool_ParseOutput(t *testing.T) {
	tool := NewTSCTool()

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
			name:   "single type error",
			output: "main.ts(10,5): error TS2304: Cannot find name 'x'.",
			expected: 1,
		},
		{
			name: "multiple errors",
			output: `main.ts(10,5): error TS2304: Cannot find name 'x'.
utils.ts(20,10): error TS2322: Type 'string' is not assignable to type 'number'.
types.d.ts(5,1): error TS1005: ';' expected.`,
			expected: 3,
		},
		{
			name:   "warnings",
			output: "main.ts(15,3): warning TS6133: 'foo' is declared but never used.",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := tool.ParseOutput(tt.output)
			assert.Len(t, issues, tt.expected)
		})
	}
}

func TestJavaScriptTools_InterfaceCompliance(t *testing.T) {
	// Ensure all JavaScript/TypeScript tools implement QualityTool interface
	var _ QualityTool = (*PrettierTool)(nil)
	var _ QualityTool = (*ESLintTool)(nil)
	var _ QualityTool = (*TSCTool)(nil)

	tools := []QualityTool{
		NewPrettierTool(),
		NewESLintTool(),
		NewTSCTool(),
	}

	for _, tool := range tools {
		t.Run(tool.Name(), func(t *testing.T) {
			assert.NotEmpty(t, tool.Name())
			assert.NotEmpty(t, tool.Language())
			assert.NotNil(t, tool.Type())
		})
	}
}

func TestJavaScriptTools_Execute_NotAvailable(t *testing.T) {
	tool := NewESLintTool()

	// Override executable to non-existent command
	tool.executable = "nonexistent-eslint-xyz"

	ctx := context.Background()
	result, err := tool.Execute(ctx, []string{"main.js"}, ExecuteOptions{})

	assert.NoError(t, err) // Execute returns error in result, not as error
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.NotNil(t, result.Error)
}
