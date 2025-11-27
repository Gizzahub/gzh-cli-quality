// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 4, config.DefaultWorkers)
	assert.Equal(t, "10m", config.Timeout)
	assert.NotNil(t, config.Tools)
	assert.Greater(t, len(config.Tools), 0, "Should have default tools configured")

	// Check some expected tools
	expectedTools := []string{"gofumpt", "goimports", "golangci-lint", "black", "ruff", "prettier", "eslint"}
	for _, tool := range expectedTools {
		toolConfig, exists := config.Tools[tool]
		assert.True(t, exists, "Tool %s should exist in default config", tool)
		assert.True(t, toolConfig.Enabled, "Tool %s should be enabled by default", tool)
	}
}

func TestLoadConfig_NoFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()

	// Change to temp directory (no config file)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	config, err := LoadConfig("")
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Should return default config
	assert.Equal(t, 4, config.DefaultWorkers)
	assert.Equal(t, "10m", config.Timeout)
}

func TestLoadConfig_WithFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gzquality.yml")

	// Create test config file
	testConfig := `default_workers: 8
timeout: "5m"
tools:
  gofumpt:
    enabled: true
    priority: 10
  golangci-lint:
    enabled: false
    args: ["--fast"]
exclude:
  - "vendor/**"
  - "node_modules/**"
`
	err := os.WriteFile(configPath, []byte(testConfig), 0o644)
	require.NoError(t, err)

	// Load config
	config, err := LoadConfig(configPath)
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Check loaded values
	assert.Equal(t, 8, config.DefaultWorkers)
	assert.Equal(t, "5m", config.Timeout)

	// Check tool config
	gofumptConfig, exists := config.Tools["gofumpt"]
	assert.True(t, exists)
	assert.True(t, gofumptConfig.Enabled)
	assert.Equal(t, 10, gofumptConfig.Priority)

	lintConfig, exists := config.Tools["golangci-lint"]
	assert.True(t, exists)
	assert.False(t, lintConfig.Enabled)
	assert.Equal(t, []string{"--fast"}, lintConfig.Args)

	// Check exclude patterns
	assert.Equal(t, []string{"vendor/**", "node_modules/**"}, config.Exclude)
}

func TestLoadConfig_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gzquality.yml")

	// Create invalid YAML
	invalidYAML := `
default_workers: not_a_number
invalid: [unclosed
`
	err := os.WriteFile(configPath, []byte(invalidYAML), 0o644)
	require.NoError(t, err)

	// Should return error
	_, err = LoadConfig(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestLoadConfig_NonExistentFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/.gzquality.yml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gzquality.yml")

	// Create test config
	config := &Config{
		DefaultWorkers: 6,
		Timeout:        "15m",
		Tools: map[string]ToolConfig{
			"gofumpt": {
				Enabled:  true,
				Priority: 10,
				Args:     []string{"-l", "-w"},
			},
		},
		Exclude: []string{"vendor/**", "*.pb.go"},
	}

	// Save config
	err := SaveConfig(config, configPath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Load it back
	loadedConfig, err := LoadConfig(configPath)
	require.NoError(t, err)

	// Verify contents
	assert.Equal(t, 6, loadedConfig.DefaultWorkers)
	assert.Equal(t, "15m", loadedConfig.Timeout)
	assert.Equal(t, []string{"vendor/**", "*.pb.go"}, loadedConfig.Exclude)

	gofumptConfig := loadedConfig.Tools["gofumpt"]
	assert.True(t, gofumptConfig.Enabled)
	assert.Equal(t, 10, gofumptConfig.Priority)
	assert.Equal(t, []string{"-l", "-w"}, gofumptConfig.Args)
}

func TestSaveConfig_CreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "nested", ".gzquality.yml")

	config := DefaultConfig()

	// Should create directories
	err := SaveConfig(config, configPath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configPath)
	assert.NoError(t, err)
}

func TestFindConfigFile_CurrentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()

	// Create config in current directory
	configPath := filepath.Join(tmpDir, ".gzquality.yml")
	err = os.WriteFile(configPath, []byte("default_workers: 4"), 0o644)
	require.NoError(t, err)

	// Change to temp directory
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Should find config
	found := FindConfigFile()

	// Resolve symlinks for both paths (macOS has /var -> /private/var)
	expectedPath, err := filepath.EvalSymlinks(configPath)
	require.NoError(t, err)
	actualPath := found
	if found != "" {
		actualPath, err = filepath.EvalSymlinks(found)
		require.NoError(t, err)
	}

	assert.Equal(t, expectedPath, actualPath)
}

func TestFindConfigFile_ParentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()

	// Create config in parent directory
	configPath := filepath.Join(tmpDir, ".gzquality.yml")
	err = os.WriteFile(configPath, []byte("default_workers: 4"), 0o644)
	require.NoError(t, err)

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.MkdirAll(subDir, 0o755)
	require.NoError(t, err)

	// Change to subdirectory
	err = os.Chdir(subDir)
	require.NoError(t, err)

	// Should find config in parent
	found := FindConfigFile()

	// Resolve symlinks for both paths (macOS has /var -> /private/var)
	expectedPath, err := filepath.EvalSymlinks(configPath)
	require.NoError(t, err)
	actualPath := found
	if found != "" {
		actualPath, err = filepath.EvalSymlinks(found)
		require.NoError(t, err)
	}

	assert.Equal(t, expectedPath, actualPath)
}

func TestFindConfigFile_MultipleNames(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()

	configNames := []string{".gzquality.yml", ".gzquality.yaml", "gzquality.yml", "gzquality.yaml"}

	for _, name := range configNames {
		t.Run(name, func(t *testing.T) {
			testDir := filepath.Join(tmpDir, name)
			err := os.MkdirAll(testDir, 0o755)
			require.NoError(t, err)

			configPath := filepath.Join(testDir, name)
			err = os.WriteFile(configPath, []byte("default_workers: 4"), 0o644)
			require.NoError(t, err)

			err = os.Chdir(testDir)
			require.NoError(t, err)

			found := FindConfigFile()

			// Resolve symlinks for both paths (macOS has /var -> /private/var)
			expectedPath, err := filepath.EvalSymlinks(configPath)
			require.NoError(t, err)
			actualPath := found
			if found != "" {
				actualPath, err = filepath.EvalSymlinks(found)
				require.NoError(t, err)
			}

			assert.Equal(t, expectedPath, actualPath)
		})
	}
}

func TestFindConfigFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()

	// Change to empty directory
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Should return empty string
	found := FindConfigFile()
	assert.Equal(t, "", found)
}

func TestGetToolConfig_Exists(t *testing.T) {
	config := &Config{
		Tools: map[string]ToolConfig{
			"gofumpt": {
				Enabled:    true,
				Priority:   10,
				ConfigFile: "/path/to/config",
				Args:       []string{"-l"},
			},
		},
	}

	toolConfig := config.GetToolConfig("gofumpt")
	assert.True(t, toolConfig.Enabled)
	assert.Equal(t, 10, toolConfig.Priority)
	assert.Equal(t, "/path/to/config", toolConfig.ConfigFile)
	assert.Equal(t, []string{"-l"}, toolConfig.Args)
}

func TestGetToolConfig_NotExists(t *testing.T) {
	config := &Config{
		Tools: map[string]ToolConfig{},
	}

	// Should return default config
	toolConfig := config.GetToolConfig("nonexistent")
	assert.True(t, toolConfig.Enabled)
	assert.Equal(t, 5, toolConfig.Priority)
	assert.Empty(t, toolConfig.Args)
}

func TestGetLanguageConfig_Exists(t *testing.T) {
	config := &Config{
		Languages: map[string]LanguageConfig{
			"Go": {
				Enabled:        true,
				PreferredTools: []string{"gofumpt", "golangci-lint"},
				Extensions:     []string{".go"},
			},
		},
	}

	langConfig := config.GetLanguageConfig("Go")
	assert.True(t, langConfig.Enabled)
	assert.Equal(t, []string{"gofumpt", "golangci-lint"}, langConfig.PreferredTools)
	assert.Equal(t, []string{".go"}, langConfig.Extensions)
}

func TestGetLanguageConfig_NotExists(t *testing.T) {
	config := &Config{
		Languages: map[string]LanguageConfig{},
	}

	// Should return default config
	langConfig := config.GetLanguageConfig("Go")
	assert.True(t, langConfig.Enabled)
	assert.Empty(t, langConfig.PreferredTools)
}

func TestShouldInclude_NoPatterns(t *testing.T) {
	config := &Config{}

	// With no patterns, all files should be included
	assert.True(t, config.ShouldInclude("main.go"))
	assert.True(t, config.ShouldInclude("vendor/lib.go"))
}

func TestShouldInclude_ExcludePatterns(t *testing.T) {
	config := &Config{
		Exclude: []string{
			"vendor/*",
			"vendor",
			"node_modules/*",
			"node_modules",
			"*.pb.go",
			"dist/*",
			"dist",
		},
	}

	tests := []struct {
		path     string
		included bool
	}{
		{"main.go", true},
		{"src/main.go", true},
		{"vendor/lib.go", false},
		{"node_modules/pkg/index.js", false},
		{"api.pb.go", false},
		{"dist/bundle.js", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := config.ShouldInclude(tt.path)
			assert.Equal(t, tt.included, result, "File %s should be included: %v", tt.path, tt.included)
		})
	}
}

func TestShouldInclude_IncludePatterns(t *testing.T) {
	config := &Config{
		Include: []string{
			"*.go",
			"src/*.go",
			"pkg/*.go",
		},
	}

	tests := []struct {
		path     string
		included bool
	}{
		{"main.go", true},
		{"src/main.go", true},
		{"pkg/lib.go", true},
		{"test.js", false},
		{"deep/nested/file.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := config.ShouldInclude(tt.path)
			assert.Equal(t, tt.included, result, "File %s should be included: %v", tt.path, tt.included)
		})
	}
}

func TestShouldInclude_BothPatterns(t *testing.T) {
	config := &Config{
		Include: []string{"*.go", "src/*.go"},
		Exclude: []string{"*_test.go"},
	}

	tests := []struct {
		path     string
		included bool
	}{
		{"main.go", true},
		{"main_test.go", false},
		{"src/main.go", true},
		{"test.js", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := config.ShouldInclude(tt.path)
			assert.Equal(t, tt.included, result, "File %s should be included: %v", tt.path, tt.included)
		})
	}
}

func TestToolConfig_Environment(t *testing.T) {
	config := &ToolConfig{
		Enabled: true,
		Env: map[string]string{
			"GOFUMPT_OPTIONS": "-l -w",
			"GO111MODULE":     "on",
		},
	}

	assert.Equal(t, 2, len(config.Env))
	assert.Equal(t, "-l -w", config.Env["GOFUMPT_OPTIONS"])
	assert.Equal(t, "on", config.Env["GO111MODULE"])
}

func TestLoadConfigIntegration(t *testing.T) {
	// Create a realistic config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gzquality.yml")

	realisticConfig := `default_workers: 8
timeout: "30m"

tools:
  gofumpt:
    enabled: true
    priority: 10
  golangci-lint:
    enabled: true
    priority: 5
    args:
      - "--timeout=10m"
      - "--enable=gofmt,goimports,govet"
  black:
    enabled: true
    args: ["--line-length=100"]
  ruff:
    enabled: true
    args: ["--fix"]

languages:
  Go:
    enabled: true
    preferred_tools: ["gofumpt", "goimports", "golangci-lint"]
    extensions: [".go"]
  Python:
    enabled: true
    preferred_tools: ["black", "ruff"]
    extensions: [".py"]

exclude:
  - "vendor"
  - "vendor/*"
  - "node_modules"
  - "node_modules/*"
  - "*.pb.go"
  - "*.pb.gw.go"
  - "dist"
  - "dist/*"
  - "build"
  - "build/*"

include:
  - "*.go"
  - "*.py"
  - "*.js"
  - "*.ts"
`
	err := os.WriteFile(configPath, []byte(realisticConfig), 0o644)
	require.NoError(t, err)

	// Load and verify
	config, err := LoadConfig(configPath)
	require.NoError(t, err)

	assert.Equal(t, 8, config.DefaultWorkers)
	assert.Equal(t, "30m", config.Timeout)

	// Verify tools
	assert.True(t, config.Tools["gofumpt"].Enabled)
	assert.Equal(t, 10, config.Tools["gofumpt"].Priority)
	assert.True(t, config.Tools["golangci-lint"].Enabled)
	assert.Equal(t, []string{"--timeout=10m", "--enable=gofmt,goimports,govet"}, config.Tools["golangci-lint"].Args)

	// Verify languages
	goConfig := config.Languages["Go"]
	assert.True(t, goConfig.Enabled)
	assert.Equal(t, []string{"gofumpt", "goimports", "golangci-lint"}, goConfig.PreferredTools)

	// Verify patterns
	assert.Len(t, config.Exclude, 10)
	assert.Len(t, config.Include, 4)

	// Test ShouldInclude with realistic patterns
	assert.True(t, config.ShouldInclude("main.go"))
	assert.False(t, config.ShouldInclude("api.pb.go"))
}
