// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Gizzahub/gzh-cli-quality/cache"
	"github.com/Gizzahub/gzh-cli-quality/tools"
)

// mockToolForBench is a simple mock tool for benchmarking
type mockToolForBench struct {
	name     string
	language string
}

func (m *mockToolForBench) Name() string                { return m.name }
func (m *mockToolForBench) Language() string            { return m.language }
func (m *mockToolForBench) Type() tools.ToolType        { return tools.FORMAT }
func (m *mockToolForBench) IsAvailable() bool           { return true }
func (m *mockToolForBench) Install() error              { return nil }
func (m *mockToolForBench) Upgrade() error              { return nil }
func (m *mockToolForBench) GetVersion() (string, error) { return "1.0.0", nil }
func (m *mockToolForBench) Execute(ctx context.Context, files []string, options tools.ExecuteOptions) (*tools.Result, error) {
	return &tools.Result{
		Tool:           m.name,
		Language:       m.language,
		Success:        true,
		FilesProcessed: len(files),
		Duration:       time.Millisecond,
		Issues:         []tools.Issue{},
	}, nil
}
func (m *mockToolForBench) FindConfigFiles(projectRoot string) []string {
	return []string{}
}

// BenchmarkExecutionPlan_Creation benchmarks execution plan creation
func BenchmarkExecutionPlan_Creation(b *testing.B) {
	// Simple benchmark for plan structure creation
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tasks := make([]tools.Task, 5)
		for j := 0; j < 5; j++ {
			tasks[j] = tools.Task{
				Tool:  &mockToolForBench{name: "tool", language: "Go"},
				Files: []string{"main.go", "utils.go"},
			}
		}
		_ = &tools.ExecutionPlan{
			Tasks:             tasks,
			TotalFiles:        10,
			EstimatedDuration: "100ms",
		}
	}
}

// BenchmarkExecutionPlan_LargeTaskSet benchmarks with many tasks
func BenchmarkExecutionPlan_LargeTaskSet(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tasks := make([]tools.Task, 100)
		for j := 0; j < 100; j++ {
			tasks[j] = tools.Task{
				Tool:  &mockToolForBench{name: "tool", language: "Go"},
				Files: []string{"file.go"},
			}
		}
		_ = &tools.ExecutionPlan{
			Tasks:             tasks,
			TotalFiles:        100,
			EstimatedDuration: "1s",
		}
	}
}

// BenchmarkExecutor_ExecuteParallel_1Worker benchmarks single worker execution
func BenchmarkExecutor_ExecuteParallel_1Worker(b *testing.B) {
	executor := NewParallelExecutor(1, 5*time.Minute)
	ctx := context.Background()

	plan := &tools.ExecutionPlan{
		Tasks: []tools.Task{
			{
				Tool:  &mockToolForBench{name: "tool1", language: "Go"},
				Files: []string{"main.go", "utils.go"},
			},
			{
				Tool:  &mockToolForBench{name: "tool2", language: "Go"},
				Files: []string{"handlers.go", "models.go"},
			},
		},
		TotalFiles:        4,
		EstimatedDuration: "2ms",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecuteParallel(ctx, plan, 1)
	}
}

// BenchmarkExecutor_ExecuteParallel_4Workers benchmarks parallel execution
func BenchmarkExecutor_ExecuteParallel_4Workers(b *testing.B) {
	executor := NewParallelExecutor(4, 5*time.Minute)
	ctx := context.Background()

	plan := &tools.ExecutionPlan{
		Tasks: []tools.Task{
			{Tool: &mockToolForBench{name: "tool1", language: "Go"}, Files: []string{"file1.go"}},
			{Tool: &mockToolForBench{name: "tool2", language: "Go"}, Files: []string{"file2.go"}},
			{Tool: &mockToolForBench{name: "tool3", language: "Go"}, Files: []string{"file3.go"}},
			{Tool: &mockToolForBench{name: "tool4", language: "Go"}, Files: []string{"file4.go"}},
			{Tool: &mockToolForBench{name: "tool5", language: "Python"}, Files: []string{"file1.py"}},
			{Tool: &mockToolForBench{name: "tool6", language: "Python"}, Files: []string{"file2.py"}},
			{Tool: &mockToolForBench{name: "tool7", language: "Python"}, Files: []string{"file3.py"}},
			{Tool: &mockToolForBench{name: "tool8", language: "Python"}, Files: []string{"file4.py"}},
		},
		TotalFiles:        8,
		EstimatedDuration: "8ms",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecuteParallel(ctx, plan, 4)
	}
}

// BenchmarkExecutor_ExecuteParallel_8Workers benchmarks high parallelism
func BenchmarkExecutor_ExecuteParallel_8Workers(b *testing.B) {
	executor := NewParallelExecutor(8, 5*time.Minute)
	ctx := context.Background()

	// Create 16 tasks
	tasks := make([]tools.Task, 16)
	for i := 0; i < 16; i++ {
		tasks[i] = tools.Task{
			Tool:  &mockToolForBench{name: "tool", language: "Go"},
			Files: []string{"file.go"},
		}
	}

	plan := &tools.ExecutionPlan{
		Tasks:             tasks,
		TotalFiles:        16,
		EstimatedDuration: "16ms",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecuteParallel(ctx, plan, 8)
	}
}

// BenchmarkToolTypeFilter_FormatOnly benchmarks format-only filtering
func BenchmarkToolTypeFilter_FormatOnly(b *testing.B) {
	// Create mixed tool list
	formatTool := &mockToolForBench{name: "formatter", language: "Go"}
	lintTool := &mockToolForBench{name: "linter", language: "Go"}

	options := PlanOptions{
		FormatOnly: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = matchesToolType(formatTool, options)
		_ = matchesToolType(lintTool, options)
	}
}

// BenchmarkToolTypeFilter_LintOnly benchmarks lint-only filtering
func BenchmarkToolTypeFilter_LintOnly(b *testing.B) {
	// Create mixed tool list
	formatTool := &mockToolForBench{name: "formatter", language: "Go"}
	lintTool := &mockToolForBench{name: "linter", language: "Go"}

	options := PlanOptions{
		LintOnly: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = matchesToolType(formatTool, options)
		_ = matchesToolType(lintTool, options)
	}
}

// ============================================================================
// Cache Performance Benchmarks
// ============================================================================

// setupBenchCache creates a cache manager for benchmarking
func setupBenchCache(b *testing.B) (*cache.CacheManager, string, func()) {
	b.Helper()
	tmpDir, err := os.MkdirTemp("", "bench-cache-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}

	cacheDir := filepath.Join(tmpDir, "cache")
	cacheManager, err := cache.NewCacheManager(cacheDir, 100*1024*1024, 24*time.Hour)
	if err != nil {
		os.RemoveAll(tmpDir)
		b.Fatalf("Failed to create cache manager: %v", err)
	}

	cleanup := func() {
		cacheManager.Close()
		os.RemoveAll(tmpDir)
	}

	return cacheManager, tmpDir, cleanup
}

// setupBenchFile creates a test file for benchmarking
func setupBenchFile(b *testing.B, dir, content string) string {
	b.Helper()
	filePath := filepath.Join(dir, "test.go")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to write test file: %v", err)
	}
	return filePath
}

// BenchmarkCache_Miss benchmarks cache miss scenario (cold cache)
func BenchmarkCache_Miss(b *testing.B) {
	cacheManager, tmpDir, cleanup := setupBenchCache(b)
	defer cleanup()

	filesDir := filepath.Join(tmpDir, "files")
	os.MkdirAll(filesDir, 0755)

	executor := NewParallelExecutorWithCache(4, 5*time.Minute, cacheManager)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Create unique file for each iteration to ensure cache miss
		testFile := filepath.Join(filesDir, "test_"+string(rune('a'+i%26))+".go")
		os.WriteFile(testFile, []byte("package main\n// iteration "+string(rune('0'+i%10))), 0644)

		plan := &tools.ExecutionPlan{
			Tasks: []tools.Task{
				{
					Tool:    &mockToolForBench{name: "gofumpt", language: "Go"},
					Files:   []string{testFile},
					Options: tools.ExecuteOptions{ProjectRoot: filesDir},
				},
			},
			TotalFiles:        1,
			EstimatedDuration: "1ms",
		}
		b.StartTimer()

		_, _ = executor.ExecuteParallel(ctx, plan, 1)
	}
}

// BenchmarkCache_Hit benchmarks cache hit scenario (warm cache)
func BenchmarkCache_Hit(b *testing.B) {
	cacheManager, tmpDir, cleanup := setupBenchCache(b)
	defer cleanup()

	filesDir := filepath.Join(tmpDir, "files")
	os.MkdirAll(filesDir, 0755)
	testFile := setupBenchFile(b, filesDir, "package main\n")

	executor := NewParallelExecutorWithCache(4, 5*time.Minute, cacheManager)
	ctx := context.Background()

	plan := &tools.ExecutionPlan{
		Tasks: []tools.Task{
			{
				Tool:    &mockToolForBench{name: "gofumpt", language: "Go"},
				Files:   []string{testFile},
				Options: tools.ExecuteOptions{ProjectRoot: filesDir},
			},
		},
		TotalFiles:        1,
		EstimatedDuration: "1ms",
	}

	// Warm up cache with first execution
	_, _ = executor.ExecuteParallel(ctx, plan, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecuteParallel(ctx, plan, 1)
	}
}

// BenchmarkCache_MultiFile_AllHit benchmarks multi-file cache hit scenario
func BenchmarkCache_MultiFile_AllHit(b *testing.B) {
	cacheManager, tmpDir, cleanup := setupBenchCache(b)
	defer cleanup()

	filesDir := filepath.Join(tmpDir, "files")
	os.MkdirAll(filesDir, 0755)

	// Create multiple test files
	var testFiles []string
	for i := 0; i < 10; i++ {
		filePath := filepath.Join(filesDir, "test_"+string(rune('a'+i))+".go")
		os.WriteFile(filePath, []byte("package main\n// file "+string(rune('a'+i))), 0644)
		testFiles = append(testFiles, filePath)
	}

	executor := NewParallelExecutorWithCache(4, 5*time.Minute, cacheManager)
	ctx := context.Background()

	plan := &tools.ExecutionPlan{
		Tasks: []tools.Task{
			{
				Tool:    &mockToolForBench{name: "gofumpt", language: "Go"},
				Files:   testFiles,
				Options: tools.ExecuteOptions{ProjectRoot: filesDir},
			},
		},
		TotalFiles:        len(testFiles),
		EstimatedDuration: "10ms",
	}

	// Warm up cache
	_, _ = executor.ExecuteParallel(ctx, plan, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecuteParallel(ctx, plan, 1)
	}
}

// BenchmarkCache_MultiFile_PartialHit benchmarks partial cache hit scenario
func BenchmarkCache_MultiFile_PartialHit(b *testing.B) {
	cacheManager, tmpDir, cleanup := setupBenchCache(b)
	defer cleanup()

	filesDir := filepath.Join(tmpDir, "files")
	os.MkdirAll(filesDir, 0755)

	// Create test files (half will be cached, half won't)
	var testFiles []string
	for i := 0; i < 10; i++ {
		filePath := filepath.Join(filesDir, "test_"+string(rune('a'+i))+".go")
		os.WriteFile(filePath, []byte("package main\n// file "+string(rune('a'+i))), 0644)
		testFiles = append(testFiles, filePath)
	}

	executor := NewParallelExecutorWithCache(4, 5*time.Minute, cacheManager)
	ctx := context.Background()

	// Cache only first 5 files
	warmupPlan := &tools.ExecutionPlan{
		Tasks: []tools.Task{
			{
				Tool:    &mockToolForBench{name: "gofumpt", language: "Go"},
				Files:   testFiles[:5],
				Options: tools.ExecuteOptions{ProjectRoot: filesDir},
			},
		},
		TotalFiles:        5,
		EstimatedDuration: "5ms",
	}
	_, _ = executor.ExecuteParallel(ctx, warmupPlan, 1)

	// Full plan with all 10 files
	fullPlan := &tools.ExecutionPlan{
		Tasks: []tools.Task{
			{
				Tool:    &mockToolForBench{name: "gofumpt", language: "Go"},
				Files:   testFiles,
				Options: tools.ExecuteOptions{ProjectRoot: filesDir},
			},
		},
		TotalFiles:        10,
		EstimatedDuration: "10ms",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecuteParallel(ctx, fullPlan, 1)
	}
}

// BenchmarkCache_NoCache benchmarks execution without cache (baseline)
func BenchmarkCache_NoCache(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-nocache-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filesDir := filepath.Join(tmpDir, "files")
	os.MkdirAll(filesDir, 0755)
	testFile := filepath.Join(filesDir, "test.go")
	os.WriteFile(testFile, []byte("package main\n"), 0644)

	executor := NewParallelExecutor(4, 5*time.Minute)
	ctx := context.Background()

	plan := &tools.ExecutionPlan{
		Tasks: []tools.Task{
			{
				Tool:    &mockToolForBench{name: "gofumpt", language: "Go"},
				Files:   []string{testFile},
				Options: tools.ExecuteOptions{ProjectRoot: filesDir},
			},
		},
		TotalFiles:        1,
		EstimatedDuration: "1ms",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.ExecuteParallel(ctx, plan, 1)
	}
}

// BenchmarkFilterIssuesByFile benchmarks issue filtering performance
func BenchmarkFilterIssuesByFile(b *testing.B) {
	// Create issues from multiple files
	issues := make([]tools.Issue, 100)
	files := []string{"file1.go", "file2.go", "file3.go", "file4.go", "file5.go"}
	for i := 0; i < 100; i++ {
		issues[i] = tools.Issue{
			File:    files[i%len(files)],
			Line:    i + 1,
			Message: "test issue",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = filterIssuesByFile(issues, "file1.go")
	}
}

// BenchmarkFilterIssuesByFile_Large benchmarks filtering with many issues
func BenchmarkFilterIssuesByFile_Large(b *testing.B) {
	// Create 1000 issues
	issues := make([]tools.Issue, 1000)
	files := []string{"a.go", "b.go", "c.go", "d.go", "e.go", "f.go", "g.go", "h.go", "i.go", "j.go"}
	for i := 0; i < 1000; i++ {
		issues[i] = tools.Issue{
			File:    files[i%len(files)],
			Line:    i + 1,
			Message: "test issue " + string(rune('0'+i%10)),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = filterIssuesByFile(issues, "a.go")
	}
}
