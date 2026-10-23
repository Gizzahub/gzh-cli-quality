package quality

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Gizzahub/gzh-cli-quality/tools"
	"github.com/stretchr/testify/require"
)

type releaseTool struct {
	mockTool
	result *tools.Result
	err    error
}

func (m *releaseTool) Execute(context.Context, []string, tools.ExecuteOptions) (*tools.Result, error) {
	return m.result, m.err
}

func captureReleaseOutput(t *testing.T, action func()) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stdout")
	require.NoError(t, err)
	previous := os.Stdout
	os.Stdout = f
	defer func() { os.Stdout = previous; require.NoError(t, f.Close()) }()
	action()
	_, err = f.Seek(0, io.SeekStart)
	require.NoError(t, err)
	data, err := io.ReadAll(f)
	require.NoError(t, err)
	return string(data)
}

func TestReleaseExecutionFailureAndReport(t *testing.T) {
	t.Chdir(t.TempDir())
	require.NoError(t, os.WriteFile("sample.go", []byte("package sample\n"), 0o600))
	manager := NewQualityManager()
	defer func() {
		if manager.cacheManager != nil {
			require.NoError(t, manager.cacheManager.Close())
		}
	}()
	manager.registry = tools.NewRegistry()
	manager.registry.Register(&releaseTool{
		mockTool: mockTool{name: "golangci-lint", language: "Go", toolType: tools.LINT},
		result:   &tools.Result{Tool: "golangci-lint", Language: "Go", Error: "exit status 2", FilesProcessed: 1},
	})
	output := captureReleaseOutput(t, func() {
		cmd := manager.newRunCmd()
		cmd.SetArgs([]string{"--no-cache", "--report=json", "--output=report.json"})
		require.ErrorContains(t, cmd.Execute(), "golangci-lint failed")
	})
	require.Contains(t, output, "0/1")
	require.FileExists(t, "report.json")
	require.NoError(t, executionResultError([]*tools.Result{{Success: true}}))
	require.ErrorContains(t, executionResultError([]*tools.Result{nil}), "no execution result")
}

func TestReleaseSummaryAndCacheCommands(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Chdir(t.TempDir())
	manager := NewQualityManager()
	if manager.cacheManager != nil {
		cacheManager := manager.cacheManager
		t.Cleanup(func() { require.NoError(t, cacheManager.Close()) })
	}
	output := captureReleaseOutput(t, func() {
		manager.displayResults([]*tools.Result{
			{Tool: "one", Success: true, Cached: true, FilesProcessed: 1},
			{Tool: "two", Error: "bad", Issues: []tools.Issue{{File: "a.go", Line: 2, Column: 3, Message: "broken", Rule: "rule"}}},
		}, time.Second, true)
		require.NoError(t, manager.newCacheStatsCmd().Execute())
		require.NoError(t, manager.newCacheClearCmd().Execute())
	})
	require.Contains(t, output, "1/2")
	require.Contains(t, output, "1개 캐시 히트")
	require.Contains(t, output, "a.go:2:3")
	manager.cacheManager = nil
	output = captureReleaseOutput(t, func() {
		require.NoError(t, manager.newCacheStatsCmd().Execute())
		require.NoError(t, manager.newCacheClearCmd().Execute())
	})
	require.Contains(t, output, "비활성화")
	for _, tc := range []struct {
		value int64
		want  string
	}{
		{1, "1 B"}, {1024, "1.00 KB"}, {1024 * 1024, "1.00 MB"}, {1024 * 1024 * 1024, "1.00 GB"},
	} {
		require.Equal(t, tc.want, formatBytes(tc.value))
	}
	for _, tc := range []struct {
		input string
		want  time.Duration
	}{
		{"", time.Hour}, {"invalid", time.Hour}, {"3d", 72 * time.Hour}, {"2m", 2 * time.Minute}, {"0d", time.Hour},
	} {
		require.Equal(t, tc.want, parseDuration(tc.input, time.Hour))
	}
	require.Error(t, manager.generateReport(nil, 0, 0, ".", "json", filepath.Join("missing", "\x00", "report.json")))
}
