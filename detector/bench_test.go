// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package detector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Gizzahub/gzh-cli-quality/tools"
)

// BenchmarkFileTypeDetector_DetectLanguages benchmarks language detection
func BenchmarkFileTypeDetector_DetectLanguages(b *testing.B) {
	detector := NewFileTypeDetector()
	tmpDir := b.TempDir()

	// Create test files
	files := map[string]string{
		"main.go":       "package main\n",
		"utils.go":      "package utils\n",
		"test.py":       "def main(): pass\n",
		"app.js":        "console.log('hello');\n",
		"index.ts":      "const x: string = 'test';\n",
		"main.rs":       "fn main() {}\n",
		"README.md":     "# Test\n",
		"config.yaml":   "key: value\n",
		"Dockerfile":    "FROM alpine\n",
		"package.json":  "{}\n",
	}

	for name, content := range files {
		_ = os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o644)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.DetectLanguages(tmpDir)
	}
}

// BenchmarkFileTypeDetector_DetectLanguages_LargeProject benchmarks with many files
func BenchmarkFileTypeDetector_DetectLanguages_LargeProject(b *testing.B) {
	detector := NewFileTypeDetector()
	tmpDir := b.TempDir()

	// Create 100 files of different types
	for i := 0; i < 100; i++ {
		fileName := filepath.Join(tmpDir, "file_"+string(rune(i)))
		switch i % 4 {
		case 0:
			_ = os.WriteFile(fileName+".go", []byte("package main\n"), 0o644)
		case 1:
			_ = os.WriteFile(fileName+".py", []byte("def f(): pass\n"), 0o644)
		case 2:
			_ = os.WriteFile(fileName+".js", []byte("const x = 1;\n"), 0o644)
		case 3:
			_ = os.WriteFile(fileName+".rs", []byte("fn main() {}\n"), 0o644)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.DetectLanguages(tmpDir)
	}
}

// BenchmarkFileTypeDetector_GetFilesByLanguage benchmarks file collection by language
func BenchmarkFileTypeDetector_GetFilesByLanguage(b *testing.B) {
	detector := NewFileTypeDetector()
	tmpDir := b.TempDir()

	// Create test files
	files := map[string]string{
		"main.go":    "package main\n",
		"utils.go":   "package utils\n",
		"test.py":    "def main(): pass\n",
		"app.js":     "console.log('hello');\n",
	}

	for name, content := range files {
		_ = os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o644)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.GetFilesByLanguage(tmpDir, []string{"Go", "Python", "JavaScript"})
	}
}

// BenchmarkSystemToolDetector_IsToolAvailable benchmarks tool availability check
func BenchmarkSystemToolDetector_IsToolAvailable(b *testing.B) {
	detector := NewSystemToolDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detector.IsToolAvailable("go")
	}
}

// BenchmarkSystemToolDetector_GetToolVersion benchmarks version retrieval
func BenchmarkSystemToolDetector_GetToolVersion(b *testing.B) {
	detector := NewSystemToolDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detector.GetToolVersion("go")
	}
}

// BenchmarkConfigFileDetector_FindConfigs benchmarks config file discovery
func BenchmarkConfigFileDetector_FindConfigs(b *testing.B) {
	detector := NewConfigFileDetector()
	tmpDir := b.TempDir()

	// Create config files
	configs := map[string]string{
		".golangci.yml":  "# config\n",
		".eslintrc":      "{}\n",
		"pyproject.toml": "[tool.black]\n",
		"rustfmt.toml":   "max_width = 100\n",
	}

	for name, content := range configs {
		_ = os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o644)
	}

	toolList := []tools.QualityTool{
		tools.NewGolangciLintTool(),
		tools.NewESLintTool(),
		tools.NewBlackTool(),
		tools.NewRustfmtTool(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detector.FindConfigs(tmpDir, toolList)
	}
}

// BenchmarkProjectAnalyzer_AnalyzeProject benchmarks full project analysis
func BenchmarkProjectAnalyzer_AnalyzeProject(b *testing.B) {
	analyzer := NewProjectAnalyzer()
	tmpDir := b.TempDir()

	// Create a realistic project structure
	files := map[string]string{
		"main.go":        "package main\nfunc main() {}\n",
		"utils.go":       "package main\nfunc utils() {}\n",
		"go.mod":         "module test\ngo 1.24\n",
		".golangci.yml":  "# config\n",
		"test.py":        "def test(): pass\n",
		"pyproject.toml": "[tool.black]\n",
		"app.js":         "const x = 1;\n",
		".eslintrc":      "{}\n",
	}

	for name, content := range files {
		_ = os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o644)
	}

	registry := tools.NewRegistry()
	registry.Register(tools.NewGofumptTool())
	registry.Register(tools.NewGoimportsTool())
	registry.Register(tools.NewGolangciLintTool())
	registry.Register(tools.NewBlackTool())
	registry.Register(tools.NewPrettierTool())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = analyzer.AnalyzeProject(tmpDir, registry)
	}
}

// BenchmarkProjectAnalyzer_GetOptimalToolSelection benchmarks tool selection
func BenchmarkProjectAnalyzer_GetOptimalToolSelection(b *testing.B) {
	analyzer := NewProjectAnalyzer()

	result := &AnalysisResult{
		ProjectRoot: "/test",
		Languages: map[string][]string{
			"Go":         {"main.go", "utils.go"},
			"Python":     {"test.py"},
			"JavaScript": {"app.js"},
		},
		RecommendedTools: map[string][]string{
			"Go":         {"gofumpt", "golangci-lint"},
			"Python":     {"black", "ruff"},
			"JavaScript": {"prettier", "eslint"},
		},
		ConfigFiles: map[string]string{
			"golangci-lint": ".golangci.yml",
			"black":         "pyproject.toml",
		},
	}

	registry := tools.NewRegistry()
	registry.Register(tools.NewGofumptTool())
	registry.Register(tools.NewGoimportsTool())
	registry.Register(tools.NewGolangciLintTool())
	registry.Register(tools.NewBlackTool())
	registry.Register(tools.NewRuffTool())
	registry.Register(tools.NewPrettierTool())
	registry.Register(tools.NewESLintTool())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = analyzer.GetOptimalToolSelection(result, registry)
	}
}

// BenchmarkRemoveDuplicates benchmarks duplicate removal
func BenchmarkRemoveDuplicates(b *testing.B) {
	input := []string{
		"a", "b", "c", "a", "d", "b", "e", "f", "c", "g",
		"h", "i", "a", "j", "k", "b", "l", "m", "c", "n",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = removeDuplicates(input)
	}
}
