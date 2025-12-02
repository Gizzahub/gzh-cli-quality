// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package executor

import (
	"context"
	"testing"
	"time"

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
