// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package detector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Gizzahub/gzh-cli-quality/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: TestNewSystemToolDetector, TestIsToolAvailable, TestGetToolVersion,
// and TestCheckCommonLocations are already in detector_test.go

func TestNewConfigFileDetector(t *testing.T) {
	detector := NewConfigFileDetector()
	assert.NotNil(t, detector)
}

func TestFindConfigs(t *testing.T) {
	detector := NewConfigFileDetector()
	tmpDir := t.TempDir()

	// Create some config files
	configFiles := map[string]string{
		".golangci.yml": "# golangci-lint config",
		".eslintrc":     "{}",
		"pyproject.toml": "[tool.black]\nline-length = 88",
	}

	for name, content := range configFiles {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o644)
		require.NoError(t, err)
	}

	// Create mock tools
	toolList := []tools.QualityTool{
		tools.NewGolangciLintTool(),
		tools.NewESLintTool(),
		tools.NewBlackTool(),
	}

	// Find configs
	configs := detector.FindConfigs(tmpDir, toolList)

	// Check that configs were found
	assert.Greater(t, len(configs), 0)

	// golangci-lint should find its config
	if golangciConfig, ok := configs["golangci-lint"]; ok {
		assert.Contains(t, golangciConfig, ".golangci.yml")
	}
}

func TestFindConfigs_NoConfigs(t *testing.T) {
	detector := NewConfigFileDetector()
	tmpDir := t.TempDir()

	// Create mock tools but no config files
	toolList := []tools.QualityTool{
		tools.NewGofumptTool(),
	}

	configs := detector.FindConfigs(tmpDir, toolList)

	// Should return empty map
	assert.Equal(t, 0, len(configs))
}

func TestValidateConfig(t *testing.T) {
	detector := NewConfigFileDetector()
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		setupFile   bool
		fileName    string
		content     string
		expectError bool
	}{
		{
			name:        "valid config file",
			setupFile:   true,
			fileName:    ".golangci.yml",
			content:     "# valid config",
			expectError: false,
		},
		{
			name:        "non-existent file",
			setupFile:   false,
			fileName:    "nonexistent.yml",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configPath string

			if tt.setupFile {
				configPath = filepath.Join(tmpDir, tt.fileName)
				err := os.WriteFile(configPath, []byte(tt.content), 0o644)
				require.NoError(t, err)
			} else {
				configPath = filepath.Join(tmpDir, tt.fileName)
			}

			err := detector.ValidateConfig("test-tool", configPath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewProjectAnalyzer(t *testing.T) {
	analyzer := NewProjectAnalyzer()

	assert.NotNil(t, analyzer)
	assert.NotNil(t, analyzer.langDetector)
	assert.NotNil(t, analyzer.toolDetector)
	assert.NotNil(t, analyzer.configDetector)
}

func TestAnalyzeProject(t *testing.T) {
	analyzer := NewProjectAnalyzer()
	tmpDir := t.TempDir()

	// Create a mock project with Go files
	files := map[string]string{
		"main.go":        "package main\n\nfunc main() {}\n",
		"go.mod":         "module test\n\ngo 1.24\n",
		".golangci.yml":  "# config",
	}

	for name, content := range files {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o644)
		require.NoError(t, err)
	}

	// Create mock registry
	registry := tools.NewRegistry()
	registry.Register(tools.NewGofumptTool())
	registry.Register(tools.NewGoimportsTool())
	registry.Register(tools.NewGolangciLintTool())

	// Analyze project
	result, err := analyzer.AnalyzeProject(tmpDir, registry)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Check basic results
	assert.Equal(t, tmpDir, result.ProjectRoot)
	assert.Contains(t, result.Languages, "Go")
	assert.Greater(t, len(result.Languages["Go"]), 0)

	// Check that Go tools are detected
	assert.NotEmpty(t, result.AvailableTools)
}

func TestAnalyzeProject_MultiLanguage(t *testing.T) {
	analyzer := NewProjectAnalyzer()
	tmpDir := t.TempDir()

	// Create a multi-language project
	files := map[string]string{
		"main.go":    "package main\n",
		"main.py":    "def main():\n    pass\n",
		"index.js":   "console.log('hello');\n",
		"go.mod":     "module test\n",
	}

	for name, content := range files {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o644)
		require.NoError(t, err)
	}

	// Create registry with tools for multiple languages
	registry := tools.NewRegistry()
	registry.Register(tools.NewGofumptTool())
	registry.Register(tools.NewBlackTool())
	registry.Register(tools.NewPrettierTool())

	result, err := analyzer.AnalyzeProject(tmpDir, registry)
	require.NoError(t, err)

	// Should detect multiple languages
	assert.Contains(t, result.Languages, "Go")
	assert.Contains(t, result.Languages, "Python")
	assert.Contains(t, result.Languages, "JavaScript")
}

func TestAnalyzeProject_NoTools(t *testing.T) {
	analyzer := NewProjectAnalyzer()
	tmpDir := t.TempDir()

	// Create project with Go files
	err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n"), 0o644)
	require.NoError(t, err)

	// Empty registry - no tools available
	registry := tools.NewRegistry()

	result, err := analyzer.AnalyzeProject(tmpDir, registry)
	require.NoError(t, err)

	// Should have issues about missing tools
	assert.Greater(t, len(result.Issues), 0)
	assert.Contains(t, result.Issues[0], "No quality tools available")
}

func TestGetOptimalToolSelection(t *testing.T) {
	analyzer := NewProjectAnalyzer()

	// Create mock analysis result
	result := &AnalysisResult{
		ProjectRoot: "/test",
		Languages: map[string][]string{
			"Go": {"main.go", "utils.go"},
		},
		RecommendedTools: map[string][]string{
			"Go": {"gofumpt", "golangci-lint"},
		},
		ConfigFiles: map[string]string{
			"golangci-lint": ".golangci.yml",
		},
	}

	// Create registry
	registry := tools.NewRegistry()
	registry.Register(tools.NewGofumptTool())
	registry.Register(tools.NewGolangciLintTool())

	// Get optimal selection
	selection := analyzer.GetOptimalToolSelection(result, registry)

	// Should select Go tools
	assert.Contains(t, selection, "Go")
	goTools := selection["Go"]
	assert.Greater(t, len(goTools), 0)

	// golangci-lint should be first (has config)
	foundGolangci := false
	for _, tool := range goTools {
		if tool.Name() == "golangci-lint" {
			foundGolangci = true
			break
		}
	}
	assert.True(t, foundGolangci, "golangci-lint should be selected")
}

func TestGetOptimalToolSelection_PreferWithConfig(t *testing.T) {
	analyzer := NewProjectAnalyzer()

	result := &AnalysisResult{
		RecommendedTools: map[string][]string{
			"Go": {"gofumpt", "goimports", "golangci-lint"},
		},
		ConfigFiles: map[string]string{
			"gofumpt": ".gofumpt",
		},
	}

	registry := tools.NewRegistry()
	registry.Register(tools.NewGofumptTool())
	registry.Register(tools.NewGoimportsTool())
	registry.Register(tools.NewGolangciLintTool())

	selection := analyzer.GetOptimalToolSelection(result, registry)
	goTools := selection["Go"]

	// gofumpt should be included (has config)
	foundGofumpt := false
	for _, tool := range goTools {
		if tool.Name() == "gofumpt" {
			foundGofumpt = true
			break
		}
	}
	assert.True(t, foundGofumpt)
}

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "all duplicates",
			input:    []string{"a", "a", "a"},
			expected: []string{"a"},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeDuplicates(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSystemToolDetector_CacheIntegration(t *testing.T) {
	detector := NewSystemToolDetector()

	// Test a sequence of operations
	toolName := "go"

	// First check
	available1 := detector.IsToolAvailable(toolName)
	assert.True(t, available1)

	// Get version
	version := detector.GetToolVersion(toolName)
	assert.NotEmpty(t, version)
	// Version might be "unknown" if none of the version flags work
	if version != "unknown" {
		assert.Contains(t, version, "go")
	}

	// Second check should use cache
	available2 := detector.IsToolAvailable(toolName)
	assert.Equal(t, available1, available2)

	// Verify cache was used
	assert.Equal(t, 1, len(detector.pathCache))
}
