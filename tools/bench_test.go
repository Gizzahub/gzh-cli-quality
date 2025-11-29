// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package tools

import (
	"context"
	"testing"
)

// BenchmarkFilterFilesByExtensions benchmarks file filtering performance
func BenchmarkFilterFilesByExtensions(b *testing.B) {
	files := []string{
		"main.go", "utils.go", "types.go", "handlers.go",
		"test.py", "config.yaml", "README.md", "script.sh",
		"app.js", "index.ts", "styles.css", "package.json",
		"main.rs", "lib.rs", "Cargo.toml", "Dockerfile",
	}
	extensions := []string{".go", ".ts", ".rs"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FilterFilesByExtensions(files, extensions)
	}
}

// BenchmarkFilterFilesByExtensions_LargeSet benchmarks with many files
func BenchmarkFilterFilesByExtensions_LargeSet(b *testing.B) {
	// Simulate a large project with 1000 files
	files := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		switch i % 5 {
		case 0:
			files[i] = "file" + string(rune(i)) + ".go"
		case 1:
			files[i] = "file" + string(rune(i)) + ".py"
		case 2:
			files[i] = "file" + string(rune(i)) + ".js"
		case 3:
			files[i] = "file" + string(rune(i)) + ".rs"
		case 4:
			files[i] = "file" + string(rune(i)) + ".txt"
		}
	}
	extensions := []string{".go", ".py"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FilterFilesByExtensions(files, extensions)
	}
}

// BenchmarkGofumptTool_BuildCommand benchmarks command building
func BenchmarkGofumptTool_BuildCommand(b *testing.B) {
	tool := NewGofumptTool()
	files := []string{"main.go", "utils.go", "handlers.go"}
	options := ExecuteOptions{
		ProjectRoot: "/test/project",
		Fix:         true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tool.BuildCommand(files, options)
	}
}

// BenchmarkGolangciLintTool_ParseOutput benchmarks JSON parsing
func BenchmarkGolangciLintTool_ParseOutput(b *testing.B) {
	tool := NewGolangciLintTool()
	output := `{
		"Issues": [
			{
				"FromLinter": "errcheck",
				"Text": "Error return value not checked",
				"Pos": {
					"Filename": "main.go",
					"Line": 10,
					"Column": 5
				}
			},
			{
				"FromLinter": "ineffassign",
				"Text": "ineffectual assignment to err",
				"Pos": {
					"Filename": "utils.go",
					"Line": 20,
					"Column": 3
				}
			}
		]
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tool.ParseOutput(output)
	}
}

// BenchmarkESLintTool_ParseOutput benchmarks ESLint JSON parsing
func BenchmarkESLintTool_ParseOutput(b *testing.B) {
	tool := NewESLintTool()
	output := `[
		{
			"filePath": "main.js",
			"messages": [
				{
					"ruleId": "no-unused-vars",
					"severity": 2,
					"message": "Variable 'x' is not used",
					"line": 10,
					"column": 5
				},
				{
					"ruleId": "semi",
					"severity": 1,
					"message": "Missing semicolon",
					"line": 15,
					"column": 20
				}
			],
			"errorCount": 1,
			"warningCount": 1
		}
	]`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tool.ParseOutput(output)
	}
}

// BenchmarkRuffTool_ParseOutput benchmarks Ruff JSON parsing
func BenchmarkRuffTool_ParseOutput(b *testing.B) {
	tool := NewRuffTool()
	output := `[
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
		},
		{
			"code": "F401",
			"message": "Unused import",
			"filename": "utils.py",
			"location": {
				"row": 5,
				"column": 1
			},
			"end_location": {
				"row": 5,
				"column": 20
			}
		}
	]`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tool.ParseOutput(output)
	}
}

// BenchmarkClippyTool_ParseOutput benchmarks Clippy JSON parsing
func BenchmarkClippyTool_ParseOutput(b *testing.B) {
	tool := NewClippyTool()
	output := `{"reason":"compiler-message","message":{"message":"unused variable: 'x'","code":{"code":"unused_variables"},"level":"warning","spans":[{"file_name":"src/main.rs","line_start":10,"column_start":9}]}}
{"reason":"compiler-message","message":{"message":"unused variable: 'y'","code":{"code":"unused_variables"},"level":"warning","spans":[{"file_name":"src/main.rs","line_start":11,"column_start":9}]}}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tool.ParseOutput(output)
	}
}

// BenchmarkBaseTool_Execute benchmarks tool execution
func BenchmarkBaseTool_Execute(b *testing.B) {
	tool := NewGofumptTool()
	ctx := context.Background()
	files := []string{"main.go"}
	options := ExecuteOptions{
		ProjectRoot: "/test/project",
	}

	// Override to non-existent command to avoid actual execution
	tool.executable = "nonexistent-tool"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tool.Execute(ctx, files, options)
	}
}

// BenchmarkRegistry_GetToolsByLanguage benchmarks tool lookup
func BenchmarkRegistry_GetToolsByLanguage(b *testing.B) {
	registry := NewRegistry()

	// Register all tools
	registry.Register(NewGofumptTool())
	registry.Register(NewGoimportsTool())
	registry.Register(NewGolangciLintTool())
	registry.Register(NewBlackTool())
	registry.Register(NewRuffTool())
	registry.Register(NewPylintTool())
	registry.Register(NewPrettierTool())
	registry.Register(NewESLintTool())
	registry.Register(NewTSCTool())
	registry.Register(NewRustfmtTool())
	registry.Register(NewClippyTool())
	registry.Register(NewCargoFmtTool())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.GetToolsByLanguage("Go")
	}
}

// BenchmarkRegistry_GetToolsByType benchmarks tool type filtering
func BenchmarkRegistry_GetToolsByType(b *testing.B) {
	registry := NewRegistry()

	// Register all tools
	registry.Register(NewGofumptTool())
	registry.Register(NewGoimportsTool())
	registry.Register(NewGolangciLintTool())
	registry.Register(NewBlackTool())
	registry.Register(NewRuffTool())
	registry.Register(NewPylintTool())
	registry.Register(NewPrettierTool())
	registry.Register(NewESLintTool())
	registry.Register(NewTSCTool())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.GetToolsByType(FORMAT)
	}
}

// BenchmarkRegistry_FindTool benchmarks specific tool lookup
func BenchmarkRegistry_FindTool(b *testing.B) {
	registry := NewRegistry()

	// Register all tools
	registry.Register(NewGofumptTool())
	registry.Register(NewGoimportsTool())
	registry.Register(NewGolangciLintTool())
	registry.Register(NewBlackTool())
	registry.Register(NewRuffTool())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.FindTool("ruff")
	}
}
