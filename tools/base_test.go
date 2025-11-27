// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBaseTool(t *testing.T) {
	tool := NewBaseTool("gofmt", "Go", "gofmt", FORMAT)

	assert.NotNil(t, tool)
	assert.Equal(t, "gofmt", tool.name)
	assert.Equal(t, "Go", tool.language)
	assert.Equal(t, "gofmt", tool.executable)
	assert.Equal(t, FORMAT, tool.toolType)
}

func TestBaseTool_Name(t *testing.T) {
	tool := NewBaseTool("gofmt", "Go", "gofmt", FORMAT)
	assert.Equal(t, "gofmt", tool.Name())
}

func TestBaseTool_Language(t *testing.T) {
	tool := NewBaseTool("gofmt", "Go", "gofmt", FORMAT)
	assert.Equal(t, "Go", tool.Language())
}

func TestBaseTool_Type(t *testing.T) {
	tests := []struct {
		name     string
		toolType ToolType
	}{
		{"FORMAT tool", FORMAT},
		{"LINT tool", LINT},
		{"BOTH tool", BOTH},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewBaseTool("test", "Go", "test", tt.toolType)
			assert.Equal(t, tt.toolType, tool.Type())
		})
	}
}

func TestBaseTool_IsAvailable(t *testing.T) {
	t.Run("Available tool", func(t *testing.T) {
		// Use 'go' which should always be available in test environment
		tool := NewBaseTool("go", "Go", "go", FORMAT)
		assert.True(t, tool.IsAvailable())
	})

	t.Run("Unavailable tool", func(t *testing.T) {
		tool := NewBaseTool("nonexistent-tool-12345", "Go", "nonexistent-tool-12345", FORMAT)
		assert.False(t, tool.IsAvailable())
	})
}

func TestBaseTool_SetInstallCommand(t *testing.T) {
	tool := NewBaseTool("test", "Go", "test", FORMAT)

	installCmd := []string{"go", "install", "test@latest"}
	tool.SetInstallCommand(installCmd)

	assert.Equal(t, installCmd, tool.installCmd)
}

func TestBaseTool_Install_NoCommand(t *testing.T) {
	tool := NewBaseTool("test", "Go", "test", FORMAT)

	err := tool.Install()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no install command configured")
}

func TestBaseTool_Install_WithCommand(t *testing.T) {
	tool := NewBaseTool("test", "Go", "test", FORMAT)

	// Use a harmless command that will succeed
	tool.SetInstallCommand([]string{"echo", "test install"})

	err := tool.Install()

	assert.NoError(t, err)
}

func TestBaseTool_Install_FailedCommand(t *testing.T) {
	tool := NewBaseTool("test", "Go", "test", FORMAT)

	// Use a command that will fail
	tool.SetInstallCommand([]string{"false"})

	err := tool.Install()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to install")
}

func TestBaseTool_GetVersion_NotInstalled(t *testing.T) {
	tool := NewBaseTool("nonexistent", "Go", "nonexistent-tool-12345", FORMAT)

	version, err := tool.GetVersion()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not installed")
	assert.Equal(t, "", version)
}

func TestBaseTool_GetVersion_Installed(t *testing.T) {
	// Use 'go' which has --version flag
	tool := NewBaseTool("go", "Go", "go", FORMAT)

	version, err := tool.GetVersion()

	assert.NoError(t, err)
	assert.NotEmpty(t, version)
	assert.Contains(t, version, "go version")
}

func TestBaseTool_Upgrade_NoCommand(t *testing.T) {
	tool := NewBaseTool("nonexistent-test", "Go", "nonexistent-tool-99999", FORMAT)

	err := tool.Upgrade()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not installed")
}

func TestBaseTool_Upgrade_WithCommand(t *testing.T) {
	tool := NewBaseTool("test", "Go", "test", FORMAT)

	// Set both install and upgrade commands
	tool.SetInstallCommand([]string{"echo", "upgrade"})

	err := tool.Upgrade()

	assert.NoError(t, err)
}

func TestBaseTool_SetConfigPatterns(t *testing.T) {
	tool := NewBaseTool("test", "Go", "test", FORMAT)

	patterns := []string{".testrc", "test.config.json"}
	tool.SetConfigPatterns(patterns)

	assert.Equal(t, patterns, tool.configPatterns)
}

func TestBaseTool_FindConfigFiles_NoPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewBaseTool("test", "Go", "test", FORMAT)

	configs := tool.FindConfigFiles(tmpDir)

	assert.Empty(t, configs)
}

func TestBaseTool_FindConfigFiles_WithPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewBaseTool("test", "Go", "test", FORMAT)

	// Create test config files
	configFile1 := filepath.Join(tmpDir, ".testrc")
	configFile2 := filepath.Join(tmpDir, "test.config.json")
	nonConfigFile := filepath.Join(tmpDir, "other.txt")

	err := os.WriteFile(configFile1, []byte("test config"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(configFile2, []byte("{}"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(nonConfigFile, []byte("other"), 0o644)
	require.NoError(t, err)

	tool.SetConfigPatterns([]string{".testrc", "test.config.json"})

	configs := tool.FindConfigFiles(tmpDir)

	assert.Equal(t, 2, len(configs))

	// Convert to map for easier checking
	configMap := make(map[string]bool)
	for _, config := range configs {
		configMap[config] = true
	}

	assert.True(t, configMap[configFile1])
	assert.True(t, configMap[configFile2])
	assert.False(t, configMap[nonConfigFile])
}

func TestBaseTool_FindConfigFiles_NonExistentDir(t *testing.T) {
	tool := NewBaseTool("test", "Go", "test", FORMAT)
	tool.SetConfigPatterns([]string{".testrc"})

	configs := tool.FindConfigFiles("/nonexistent/directory/12345")

	assert.Empty(t, configs)
}

func TestBaseTool_FindConfigFiles_MultiplePatterns(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewBaseTool("test", "Go", "test", FORMAT)

	// Create multiple config files
	config1 := filepath.Join(tmpDir, ".testrc")
	config2 := filepath.Join(tmpDir, "test.config.js")

	err := os.WriteFile(config1, []byte("config1"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(config2, []byte("config2"), 0o644)
	require.NoError(t, err)

	tool.SetConfigPatterns([]string{".testrc", "test.config.js"})

	configs := tool.FindConfigFiles(tmpDir)

	// Should find both configs in root directory
	assert.Equal(t, 2, len(configs))

	configMap := make(map[string]bool)
	for _, config := range configs {
		configMap[config] = true
	}

	assert.True(t, configMap[config1])
	assert.True(t, configMap[config2])
}

func TestBaseTool_ParseOutput_EmptyOutput(t *testing.T) {
	tool := NewBaseTool("test", "Go", "test", LINT)

	issues := tool.ParseOutput("")

	assert.Empty(t, issues)
}

func TestBaseTool_ParseOutput_NoMatch(t *testing.T) {
	tool := NewBaseTool("test", "Go", "test", LINT)

	output := "This is some random output\nwith no issues\n"
	issues := tool.ParseOutput(output)

	assert.Empty(t, issues)
}

func TestBaseTool_BuildCommand_BasicArgs(t *testing.T) {
	tool := NewBaseTool("test", "Go", "test", FORMAT)

	files := []string{"file1.go", "file2.go"}
	options := ExecuteOptions{
		ProjectRoot: "/project",
	}

	cmd := tool.BuildCommand(files, options)

	assert.Contains(t, cmd.Path, "test")
	assert.Contains(t, cmd.Args, "file1.go")
	assert.Contains(t, cmd.Args, "file2.go")
	assert.Equal(t, "/project", cmd.Dir)
}

func TestBaseTool_BuildCommand_WithExtraArgs(t *testing.T) {
	tool := NewBaseTool("test", "Go", "test", LINT)

	files := []string{"file1.go"}
	options := ExecuteOptions{
		ProjectRoot: "/project",
		ExtraArgs:   []string{"--verbose", "--strict"},
	}

	cmd := tool.BuildCommand(files, options)

	assert.Contains(t, cmd.Args, "--verbose")
	assert.Contains(t, cmd.Args, "--strict")
}

func TestBaseTool_BuildCommand_WithEnv(t *testing.T) {
	tool := NewBaseTool("test", "Go", "test", FORMAT)

	files := []string{"file1.go"}
	options := ExecuteOptions{
		ProjectRoot: "/project",
		Env: map[string]string{
			"TEST_VAR": "test_value",
			"DEBUG":    "true",
		},
	}

	cmd := tool.BuildCommand(files, options)

	// Check environment variables
	envMap := make(map[string]string)
	for _, env := range cmd.Env {
		parts := splitEnv(env)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	assert.Equal(t, "test_value", envMap["TEST_VAR"])
	assert.Equal(t, "true", envMap["DEBUG"])
}

// Helper to split "KEY=VALUE" env strings
func splitEnv(env string) []string {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return []string{env[:i], env[i+1:]}
		}
	}
	return []string{env}
}
