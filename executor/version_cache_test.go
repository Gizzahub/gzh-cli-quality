// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

//go:build darwin || linux

package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Gizzahub/gzh-cli-quality/cache"
	"github.com/Gizzahub/gzh-cli-quality/tools"
	"github.com/stretchr/testify/require"
)

// probedTool reports a regular, non-script executable file as its version
// probe, so the cache manager may reuse its version across executions.
type probedTool struct {
	*versionObservedTool
	executable string
}

func (t *probedTool) VersionProbe() (string, []string) { return t.executable, []string{"--version"} }
func (t *probedTool) VersionCacheable() bool           { return true }

func TestCacheVersionReusedAcrossRunsUntilExecutableChanges(t *testing.T) {
	manager, err := cache.NewCacheManager(t.TempDir(), 1<<20, time.Hour)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	dir := t.TempDir()
	file := filepath.Join(dir, "a.go")
	require.NoError(t, os.WriteFile(file, []byte("package a\n"), 0o600))
	executable := filepath.Join(t.TempDir(), "formatter")
	require.NoError(t, os.WriteFile(executable, []byte("\x7fELF stand-in"), 0o755))

	tool := &probedTool{versionObservedTool: &versionObservedTool{mockCacheableTool: newMockCacheableTool("formatter", "Go")}, executable: executable}
	executor := NewParallelExecutorWithCache(1, time.Minute, manager)
	plan := &tools.ExecutionPlan{Tasks: []tools.Task{{Tool: tool, Files: []string{file}, Options: tools.ExecuteOptions{ProjectRoot: dir}}}}
	run := func(wantCalls, wantExecutions int, wantCached bool) {
		t.Helper()
		results, err := executor.Execute(context.Background(), plan)
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.True(t, results[0].Success)
		require.Equal(t, wantCached, results[0].Cached)
		require.Equal(t, wantCalls, tool.versionCalls)
		require.Equal(t, wantExecutions, tool.execCount)
	}
	run(1, 1, false)
	run(1, 1, true) // Warm hit: no version probe.

	// An upgrade replaces the file; the next run probes and misses.
	tool.version = "2.0.0"
	replacement := executable + ".new"
	require.NoError(t, os.WriteFile(replacement, []byte("\x7fELF stand-in v2"), 0o755))
	require.NoError(t, os.Rename(replacement, executable))
	run(2, 2, false)
	run(2, 2, true)
}
