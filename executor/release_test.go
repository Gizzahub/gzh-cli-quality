package executor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Gizzahub/gzh-cli-quality/cache"
	"github.com/Gizzahub/gzh-cli-quality/tools"
	"github.com/stretchr/testify/require"
)

func TestReleaseExecutorNilResult(t *testing.T) {
	tool := &mockTool{name: "broken", language: "Go", executeFunc: func(context.Context, []string, tools.ExecuteOptions) (*tools.Result, error) { return nil, nil }}
	results, err := NewParallelExecutor(1, time.Second).Execute(context.Background(), &tools.ExecutionPlan{Tasks: []tools.Task{{Tool: tool}}})
	require.ErrorContains(t, err, "no execution result")
	require.Len(t, results, 1)
	require.False(t, results[0].Success)
}

func TestReleaseExecutorCancellation(t *testing.T) {
	started, stopped := make(chan struct{}), make(chan struct{})
	tool := &mockTool{name: "blocking", language: "Go", executeFunc: func(ctx context.Context, _ []string, _ tools.ExecuteOptions) (*tools.Result, error) {
		close(started)
		<-ctx.Done()
		close(stopped)
		return nil, ctx.Err()
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { <-started; cancel() }()
	_, err := NewParallelExecutor(1, time.Second).Execute(ctx, &tools.ExecutionPlan{Tasks: []tools.Task{{Tool: tool}}})
	require.ErrorIs(t, err, context.Canceled)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("tool did not stop on cancellation")
	}
}

func TestReleasePartialCacheFailure(t *testing.T) {
	manager, err := cache.NewCacheManager(t.TempDir(), 1<<20, time.Hour)
	require.NoError(t, err)
	defer func() { require.NoError(t, manager.Close()) }()
	cachedFile := filepath.Join(t.TempDir(), "cached.go")
	require.NoError(t, os.WriteFile(cachedFile, []byte("package cached"), 0o600))
	cause := errors.New("runner failed")
	tool := &mockTool{name: "linter", language: "Go", executeFunc: func(context.Context, []string, tools.ExecuteOptions) (*tools.Result, error) { return nil, cause }}
	key, err := cache.GenerateKey(cachedFile, tool, tools.ExecuteOptions{})
	require.NoError(t, err)
	require.NoError(t, manager.Set(key, &tools.Result{Tool: tool.Name(), Success: true, FilesProcessed: 1}))
	executor := NewParallelExecutorWithCache(1, time.Second, manager)
	result, err := executor.executeMultiFileWithCache(context.Background(), tools.Task{Tool: tool, Files: []string{cachedFile, filepath.Join(t.TempDir(), "missing.go")}})
	require.ErrorIs(t, err, cause)
	require.NotNil(t, result)
	require.False(t, result.Success)
	require.Equal(t, "runner failed", result.Error)
	require.Equal(t, 1, result.FilesProcessed)
	tool.executeFunc = func(context.Context, []string, tools.ExecuteOptions) (*tools.Result, error) { return nil, nil }
	result, err = executor.executeMultiFileWithCache(context.Background(), tools.Task{Tool: tool, Files: []string{cachedFile, filepath.Join(t.TempDir(), "missing.go")}})
	require.ErrorContains(t, err, "no execution result")
	require.NotNil(t, result)
	require.False(t, result.Success)
}

func TestReleaseMergeResults(t *testing.T) {
	tool := newMockCacheableTool("linter", "Go")
	result := mergeResults([]*tools.Result{
		{Success: true, FilesProcessed: 2, Cached: true, Output: "first", Duration: time.Second},
		{Success: false, FilesProcessed: 1, Error: "failed", Output: "second", Duration: 2 * time.Second, Issues: []tools.Issue{{File: "a.go"}}},
		nil,
	}, tool)
	require.False(t, result.Success)
	require.False(t, result.Cached)
	require.Equal(t, 3, result.FilesProcessed)
	require.Equal(t, 3*time.Second, result.Duration)
	require.Equal(t, "first\nsecond", result.Output)
	require.Equal(t, "failed", result.Error)
	require.Len(t, result.Issues, 1)
	require.True(t, mergeResults(nil, tool).Success)
}
