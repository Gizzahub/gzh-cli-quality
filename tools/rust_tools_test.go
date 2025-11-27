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

func TestNewRustfmtTool(t *testing.T) {
	tool := NewRustfmtTool()

	assert.NotNil(t, tool)
	assert.Equal(t, "rustfmt", tool.Name())
	assert.Equal(t, "Rust", tool.Language())
	assert.Equal(t, FORMAT, tool.Type())

	// Check config file patterns
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "rustfmt.toml")
	err := os.WriteFile(configFile, []byte("max_width = 100"), 0o644)
	assert.NoError(t, err)

	configs := tool.FindConfigFiles(tmpDir)
	assert.Contains(t, configs, configFile)
}

func TestRustfmtTool_BuildCommand(t *testing.T) {
	tool := NewRustfmtTool()

	tests := []struct {
		name         string
		files        []string
		options      ExecuteOptions
		expectedArgs []string
	}{
		{
			name:  "basic Rust files",
			files: []string{"main.rs", "lib.rs"},
			options: ExecuteOptions{},
			expectedArgs: []string{"main.rs", "lib.rs"},
		},
		{
			name:  "with config file",
			files: []string{"main.rs"},
			options: ExecuteOptions{
				ConfigFile: "rustfmt.toml",
			},
			expectedArgs: []string{"--config-path", "rustfmt.toml"},
		},
		{
			name:  "filters non-Rust files",
			files: []string{"main.rs", "test.go", "README.md"},
			options: ExecuteOptions{},
			expectedArgs: []string{"main.rs"},
		},
		{
			name:  "with extra args",
			files: []string{"main.rs"},
			options: ExecuteOptions{
				ExtraArgs: []string{"--check", "--verbose"},
			},
			expectedArgs: []string{"--check", "--verbose", "main.rs"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tool.BuildCommand(tt.files, tt.options)

			assert.Equal(t, "rustfmt", filepath.Base(cmd.Path))
			cmdArgs := cmd.Args[1:]

			for _, expected := range tt.expectedArgs {
				assert.Contains(t, cmdArgs, expected)
			}
		})
	}
}

func TestNewClippyTool(t *testing.T) {
	tool := NewClippyTool()

	assert.NotNil(t, tool)
	assert.Equal(t, "clippy", tool.Name())
	assert.Equal(t, "Rust", tool.Language())
	assert.Equal(t, LINT, tool.Type())

	// Check config file patterns
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "Cargo.toml")
	err := os.WriteFile(configFile, []byte("[package]\nname = \"test\""), 0o644)
	assert.NoError(t, err)

	configs := tool.FindConfigFiles(tmpDir)
	assert.Contains(t, configs, configFile)
}

func TestClippyTool_BuildCommand(t *testing.T) {
	tool := NewClippyTool()

	tests := []struct {
		name     string
		files    []string
		options  ExecuteOptions
		checkFix bool
	}{
		{
			name:  "basic lint",
			files: []string{"main.rs"},
			options: ExecuteOptions{},
		},
		{
			name:  "with fix flag",
			files: []string{"main.rs"},
			options: ExecuteOptions{
				Fix: true,
			},
			checkFix: true,
		},
		{
			name:  "with extra args",
			files: []string{"main.rs"},
			options: ExecuteOptions{
				ExtraArgs: []string{"--all-features"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tool.BuildCommand(tt.files, tt.options)

			assert.Equal(t, "cargo", filepath.Base(cmd.Path))
			cmdArgs := cmd.Args[1:]

			assert.Contains(t, cmdArgs, "clippy")
			assert.Contains(t, cmdArgs, "--message-format")
			assert.Contains(t, cmdArgs, "json")
			assert.Contains(t, cmdArgs, "-D")
			assert.Contains(t, cmdArgs, "warnings")

			if tt.checkFix {
				assert.Contains(t, cmdArgs, "--fix")
			}
		})
	}
}

func TestClippyTool_ParseOutput(t *testing.T) {
	tool := NewClippyTool()

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
			name: "valid JSON output with warning",
			output: `{"reason":"compiler-message","message":{"message":"unused variable: 'x'","code":{"code":"unused_variables"},"level":"warning","spans":[{"file_name":"src/main.rs","line_start":10,"column_start":9}]},"target":{"name":"test"}}`,
			expected: 1,
			checkIssue: func(t *testing.T, issue Issue) {
				assert.Equal(t, "src/main.rs", issue.File)
				assert.Equal(t, 10, issue.Line)
				assert.Equal(t, 9, issue.Column)
				assert.Equal(t, "unused_variables", issue.Rule)
				assert.Equal(t, "warning", issue.Severity)
				assert.Equal(t, "unused variable: 'x'", issue.Message)
			},
		},
		{
			name: "error severity",
			output: `{"reason":"compiler-message","message":{"message":"cannot find value 'foo' in this scope","code":{"code":"E0425"},"level":"error","spans":[{"file_name":"src/lib.rs","line_start":5,"column_start":15}]}}`,
			expected: 1,
			checkIssue: func(t *testing.T, issue Issue) {
				assert.Equal(t, "error", issue.Severity)
				assert.Equal(t, "E0425", issue.Rule)
			},
		},
		{
			name: "no code specified",
			output: `{"reason":"compiler-message","message":{"message":"some lint message","level":"warning","spans":[{"file_name":"src/main.rs","line_start":20,"column_start":5}]}}`,
			expected: 1,
			checkIssue: func(t *testing.T, issue Issue) {
				assert.Equal(t, "", issue.Rule)
			},
		},
		{
			name: "no spans - should be skipped",
			output: `{"reason":"compiler-message","message":{"message":"building project","level":"info","spans":[]}}`,
			expected: 0,
		},
		{
			name: "multiple issues on separate lines",
			output: `{"reason":"compiler-message","message":{"message":"unused variable: 'x'","code":{"code":"unused_variables"},"level":"warning","spans":[{"file_name":"src/main.rs","line_start":10,"column_start":9}]}}
{"reason":"compiler-message","message":{"message":"unused variable: 'y'","code":{"code":"unused_variables"},"level":"warning","spans":[{"file_name":"src/main.rs","line_start":11,"column_start":9}]}}`,
			expected: 2,
		},
		{
			name: "invalid JSON - should be skipped",
			output: `not valid json
{"reason":"compiler-message","message":{"message":"valid","level":"warning","spans":[{"file_name":"src/main.rs","line_start":1,"column_start":1}]}}`,
			expected: 1,
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

func TestNewCargoFmtTool(t *testing.T) {
	tool := NewCargoFmtTool()

	assert.NotNil(t, tool)
	assert.Equal(t, "cargo-fmt", tool.Name())
	assert.Equal(t, "Rust", tool.Language())
	assert.Equal(t, FORMAT, tool.Type())

	// Check config file patterns
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, ".rustfmt.toml")
	err := os.WriteFile(configFile, []byte("max_width = 100"), 0o644)
	assert.NoError(t, err)

	configs := tool.FindConfigFiles(tmpDir)
	assert.Contains(t, configs, configFile)
}

func TestCargoFmtTool_BuildCommand(t *testing.T) {
	tool := NewCargoFmtTool()

	tests := []struct {
		name    string
		files   []string
		options ExecuteOptions
	}{
		{
			name:  "basic format",
			files: []string{}, // cargo fmt doesn't use individual files
			options: ExecuteOptions{},
		},
		{
			name:  "with extra args",
			files: []string{},
			options: ExecuteOptions{
				ExtraArgs: []string{"--check", "--verbose"},
			},
		},
		{
			name:  "with project root",
			files: []string{},
			options: ExecuteOptions{
				ProjectRoot: "/test/project",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tool.BuildCommand(tt.files, tt.options)

			assert.Equal(t, "cargo", filepath.Base(cmd.Path))
			cmdArgs := cmd.Args[1:]

			assert.Contains(t, cmdArgs, "fmt")

			if len(tt.options.ExtraArgs) > 0 {
				for _, arg := range tt.options.ExtraArgs {
					assert.Contains(t, cmdArgs, arg)
				}
			}

			if tt.options.ProjectRoot != "" {
				assert.Equal(t, tt.options.ProjectRoot, cmd.Dir)
			}
		})
	}
}

func TestRustTools_InterfaceCompliance(t *testing.T) {
	// Ensure all Rust tools implement QualityTool interface
	var _ QualityTool = (*RustfmtTool)(nil)
	var _ QualityTool = (*ClippyTool)(nil)
	var _ QualityTool = (*CargoFmtTool)(nil)

	tools := []QualityTool{
		NewRustfmtTool(),
		NewClippyTool(),
		NewCargoFmtTool(),
	}

	for _, tool := range tools {
		t.Run(tool.Name(), func(t *testing.T) {
			assert.NotEmpty(t, tool.Name())
			assert.NotEmpty(t, tool.Language())
			assert.NotNil(t, tool.Type())
		})
	}
}

func TestRustTools_Execute_NotAvailable(t *testing.T) {
	tool := NewRustfmtTool()

	// Override executable to non-existent command
	tool.executable = "nonexistent-rustfmt-xyz"

	ctx := context.Background()
	result, err := tool.Execute(ctx, []string{"main.rs"}, ExecuteOptions{})

	assert.NoError(t, err) // Execute returns error in result, not as error
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.NotNil(t, result.Error)
}

func TestRustfmtTool_BuildCommand_ProjectRoot(t *testing.T) {
	tool := NewRustfmtTool()

	cmd := tool.BuildCommand([]string{"main.rs"}, ExecuteOptions{
		ProjectRoot: "/test/project",
	})

	assert.Equal(t, "/test/project", cmd.Dir)
}

func TestClippyTool_BuildCommand_ProjectRoot(t *testing.T) {
	tool := NewClippyTool()

	cmd := tool.BuildCommand([]string{"main.rs"}, ExecuteOptions{
		ProjectRoot: "/test/project",
	})

	assert.Equal(t, "/test/project", cmd.Dir)
}
