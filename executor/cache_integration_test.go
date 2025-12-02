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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCacheableTool is a mock tool that counts executions
type mockCacheableTool struct {
	name          string
	language      string
	execCount     int
	available     bool
	version       string
	configFiles   []string
	successResult bool
}

func newMockCacheableTool(name, language string) *mockCacheableTool {
	return &mockCacheableTool{
		name:          name,
		language:      language,
		available:     true,
		version:       "1.0.0",
		configFiles:   []string{},
		successResult: true,
	}
}

func (m *mockCacheableTool) Name() string            { return m.name }
func (m *mockCacheableTool) Language() string        { return m.language }
func (m *mockCacheableTool) Type() tools.ToolType    { return tools.FORMAT }
func (m *mockCacheableTool) IsAvailable() bool       { return m.available }
func (m *mockCacheableTool) Install() error          { return nil }
func (m *mockCacheableTool) Upgrade() error          { return nil }
func (m *mockCacheableTool) GetVersion() (string, error) { return m.version, nil }
func (m *mockCacheableTool) FindConfigFiles(projectRoot string) []string { return m.configFiles }

func (m *mockCacheableTool) Execute(ctx context.Context, files []string, options tools.ExecuteOptions) (*tools.Result, error) {
	m.execCount++
	return &tools.Result{
		Tool:           m.name,
		Language:       m.language,
		Success:        m.successResult,
		FilesProcessed: len(files),
		Duration:       time.Millisecond,
		Issues:         []tools.Issue{},
	}, nil
}

func TestExecutor_WithCache_SingleFile(t *testing.T) {
	// Setup temp directory for cache
	cacheDir := t.TempDir()
	cacheManager, err := cache.NewCacheManager(filepath.Join(cacheDir, "cache"), 100*1024*1024, 24*time.Hour)
	require.NoError(t, err)
	defer cacheManager.Close()

	// Create temp file to process
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	err = os.WriteFile(testFile, []byte("package main\n"), 0644)
	require.NoError(t, err)

	// Create executor with cache
	executor := NewParallelExecutorWithCache(4, 5*time.Minute, cacheManager)
	assert.True(t, executor.CacheEnabled())

	// Create mock tool
	tool := newMockCacheableTool("gofumpt", "Go")

	// Create execution plan
	plan := &tools.ExecutionPlan{
		Tasks: []tools.Task{
			{
				Tool:     tool,
				Files:    []string{testFile},
				Options:  tools.ExecuteOptions{ProjectRoot: tmpDir},
				Priority: 10,
			},
		},
		TotalFiles:        1,
		EstimatedDuration: "1s",
	}

	// First execution: cache miss
	results, err := executor.ExecuteParallel(context.Background(), plan, 1)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Success)
	assert.Equal(t, 1, tool.execCount, "Tool should be executed once on cache miss")
	assert.False(t, results[0].Cached, "Result should not be cached on first execution")

	// Second execution: cache hit
	results, err = executor.ExecuteParallel(context.Background(), plan, 1)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Success)
	assert.Equal(t, 1, tool.execCount, "Tool should not be executed on cache hit")
	assert.True(t, results[0].Cached, "Result should be cached on second execution")

	// Third execution: still cache hit
	results, err = executor.ExecuteParallel(context.Background(), plan, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, tool.execCount, "Tool should still not be executed")
	assert.True(t, results[0].Cached)

	// Verify cache stats
	stats := cacheManager.Stats()
	assert.Equal(t, int64(2), stats.HitCount, "Should have 2 cache hits")
	assert.Equal(t, int64(1), stats.MissCount, "Should have 1 cache miss")
}

func TestExecutor_WithCache_FileModification(t *testing.T) {
	// Setup temp directory for cache
	cacheDir := t.TempDir()
	cacheManager, err := cache.NewCacheManager(filepath.Join(cacheDir, "cache"), 100*1024*1024, 24*time.Hour)
	require.NoError(t, err)
	defer cacheManager.Close()

	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	err = os.WriteFile(testFile, []byte("package main\n"), 0644)
	require.NoError(t, err)

	// Create executor with cache
	executor := NewParallelExecutorWithCache(4, 5*time.Minute, cacheManager)

	// Create mock tool
	tool := newMockCacheableTool("gofumpt", "Go")

	plan := &tools.ExecutionPlan{
		Tasks: []tools.Task{
			{
				Tool:     tool,
				Files:    []string{testFile},
				Options:  tools.ExecuteOptions{ProjectRoot: tmpDir},
				Priority: 10,
			},
		},
		TotalFiles:        1,
		EstimatedDuration: "1s",
	}

	// First execution: cache miss
	results, err := executor.ExecuteParallel(context.Background(), plan, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, tool.execCount)
	assert.False(t, results[0].Cached)

	// Second execution: cache hit
	results, err = executor.ExecuteParallel(context.Background(), plan, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, tool.execCount)
	assert.True(t, results[0].Cached)

	// Modify file content
	err = os.WriteFile(testFile, []byte("package main\n\nfunc foo() {}\n"), 0644)
	require.NoError(t, err)

	// Third execution: cache miss (file changed)
	results, err = executor.ExecuteParallel(context.Background(), plan, 1)
	require.NoError(t, err)
	assert.Equal(t, 2, tool.execCount, "Tool should be executed again after file modification")
	assert.False(t, results[0].Cached, "Result should not be cached after file modification")
}

func TestExecutor_WithCache_MultipleFiles(t *testing.T) {
	// Setup temp directory for cache
	cacheDir := t.TempDir()
	cacheManager, err := cache.NewCacheManager(filepath.Join(cacheDir, "cache"), 100*1024*1024, 24*time.Hour)
	require.NoError(t, err)
	defer cacheManager.Close()

	// Create temp files
	tmpDir := t.TempDir()
	testFile1 := filepath.Join(tmpDir, "test1.go")
	testFile2 := filepath.Join(tmpDir, "test2.go")
	testFile3 := filepath.Join(tmpDir, "test3.go")
	require.NoError(t, os.WriteFile(testFile1, []byte("package main\n// file 1\n"), 0644))
	require.NoError(t, os.WriteFile(testFile2, []byte("package main\n// file 2\n"), 0644))
	require.NoError(t, os.WriteFile(testFile3, []byte("package main\n// file 3\n"), 0644))

	// Create executor with cache
	executor := NewParallelExecutorWithCache(4, 5*time.Minute, cacheManager)

	// Create mock tool
	tool := newMockCacheableTool("gofumpt", "Go")

	plan := &tools.ExecutionPlan{
		Tasks: []tools.Task{
			{
				Tool:     tool,
				Files:    []string{testFile1, testFile2, testFile3},
				Options:  tools.ExecuteOptions{ProjectRoot: tmpDir},
				Priority: 10,
			},
		},
		TotalFiles:        3,
		EstimatedDuration: "1s",
	}

	// First execution: cache miss for all files
	results, err := executor.ExecuteParallel(context.Background(), plan, 1)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Success)
	// Note: execution count depends on implementation (per-file or batch)

	// Second execution: cache hit for all files
	tool.execCount = 0 // Reset counter
	results, err = executor.ExecuteParallel(context.Background(), plan, 1)
	require.NoError(t, err)
	assert.Equal(t, 0, tool.execCount, "Tool should not be executed when all files are cached")
	assert.True(t, results[0].Cached, "Result should be cached")

	// Modify one file
	require.NoError(t, os.WriteFile(testFile2, []byte("package main\n// file 2 modified\n"), 0644))

	// Third execution: partial cache hit (only file2 misses)
	tool.execCount = 0
	results, err = executor.ExecuteParallel(context.Background(), plan, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, tool.execCount, "Tool should be executed once for the modified file")
}

func TestExecutor_WithoutCache(t *testing.T) {
	// Create executor without cache
	executor := NewParallelExecutor(4, 5*time.Minute)
	assert.False(t, executor.CacheEnabled())

	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	require.NoError(t, os.WriteFile(testFile, []byte("package main\n"), 0644))

	// Create mock tool
	tool := newMockCacheableTool("gofumpt", "Go")

	plan := &tools.ExecutionPlan{
		Tasks: []tools.Task{
			{
				Tool:     tool,
				Files:    []string{testFile},
				Options:  tools.ExecuteOptions{ProjectRoot: tmpDir},
				Priority: 10,
			},
		},
		TotalFiles:        1,
		EstimatedDuration: "1s",
	}

	// First execution
	results, err := executor.ExecuteParallel(context.Background(), plan, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, tool.execCount)
	assert.False(t, results[0].Cached)

	// Second execution: still executes (no cache)
	results, err = executor.ExecuteParallel(context.Background(), plan, 1)
	require.NoError(t, err)
	assert.Equal(t, 2, tool.execCount, "Tool should be executed again without cache")
	assert.False(t, results[0].Cached)
}

func TestExecutor_CacheDisabled(t *testing.T) {
	// Setup cache but disabled
	cacheDir := t.TempDir()
	cacheManager, err := cache.NewCacheManager(filepath.Join(cacheDir, "cache"), 100*1024*1024, 24*time.Hour)
	require.NoError(t, err)
	defer cacheManager.Close()

	cacheManager.SetEnabled(false)

	// Create executor with disabled cache
	executor := NewParallelExecutorWithCache(4, 5*time.Minute, cacheManager)
	assert.False(t, executor.CacheEnabled())

	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	require.NoError(t, os.WriteFile(testFile, []byte("package main\n"), 0644))

	// Create mock tool
	tool := newMockCacheableTool("gofumpt", "Go")

	plan := &tools.ExecutionPlan{
		Tasks: []tools.Task{
			{
				Tool:     tool,
				Files:    []string{testFile},
				Options:  tools.ExecuteOptions{ProjectRoot: tmpDir},
				Priority: 10,
			},
		},
		TotalFiles:        1,
		EstimatedDuration: "1s",
	}

	// First execution
	results, err := executor.ExecuteParallel(context.Background(), plan, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, tool.execCount)

	// Second execution: still executes (cache disabled)
	results, err = executor.ExecuteParallel(context.Background(), plan, 1)
	require.NoError(t, err)
	assert.Equal(t, 2, tool.execCount)
	assert.False(t, results[0].Cached)
}

func TestExecutor_FailedResultNotCached(t *testing.T) {
	// Setup cache
	cacheDir := t.TempDir()
	cacheManager, err := cache.NewCacheManager(filepath.Join(cacheDir, "cache"), 100*1024*1024, 24*time.Hour)
	require.NoError(t, err)
	defer cacheManager.Close()

	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	require.NoError(t, os.WriteFile(testFile, []byte("package main\n"), 0644))

	// Create executor with cache
	executor := NewParallelExecutorWithCache(4, 5*time.Minute, cacheManager)

	// Create mock tool that fails
	tool := newMockCacheableTool("gofumpt", "Go")
	tool.successResult = false

	plan := &tools.ExecutionPlan{
		Tasks: []tools.Task{
			{
				Tool:     tool,
				Files:    []string{testFile},
				Options:  tools.ExecuteOptions{ProjectRoot: tmpDir},
				Priority: 10,
			},
		},
		TotalFiles:        1,
		EstimatedDuration: "1s",
	}

	// First execution: fails
	results, err := executor.ExecuteParallel(context.Background(), plan, 1)
	require.NoError(t, err)
	assert.False(t, results[0].Success)
	assert.Equal(t, 1, tool.execCount)

	// Second execution: still executes (failed results not cached)
	results, err = executor.ExecuteParallel(context.Background(), plan, 1)
	require.NoError(t, err)
	assert.Equal(t, 2, tool.execCount, "Failed results should not be cached")
	assert.False(t, results[0].Cached)
}
