package quality

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Gizzahub/gzh-cli-quality/executor"
	"github.com/Gizzahub/gzh-cli-quality/tools"
	"github.com/stretchr/testify/require"
)

type releaseTool struct {
	mockTool
	result *tools.Result
	err    error
	calls  int
}

func (m *releaseTool) Execute(context.Context, []string, tools.ExecuteOptions) (*tools.Result, error) {
	m.calls++
	return m.result, m.err
}

// Keep the real planner/executor while making its analysis independent of
// installed tools, user configuration and detector recommendation heuristics.
type releaseAnalyzer struct{ tool tools.QualityTool }

func (a releaseAnalyzer) AnalyzeProject(root string, _ tools.ToolRegistry) (*executor.AnalysisResult, error) {
	return &executor.AnalysisResult{ProjectRoot: root, Languages: map[string][]string{"Go": {"sample.go"}}}, nil
}

func (a releaseAnalyzer) GetOptimalToolSelection(*executor.AnalysisResult, tools.ToolRegistry) map[string][]tools.QualityTool {
	return map[string][]tools.QualityTool{"Go": {a.tool}}
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
	tool := &releaseTool{
		mockTool: mockTool{name: "golangci-lint", language: "Go", toolType: tools.LINT},
		result:   &tools.Result{Tool: "golangci-lint", Language: "Go", Error: "exit status 2", FilesProcessed: 1},
	}
	manager := &QualityManager{
		registry: tools.NewRegistry(),
		planner:  executor.NewExecutionPlanner(releaseAnalyzer{tool: tool}),
		executor: executor.NewParallelExecutor(1, time.Minute),
	}
	manager.registry.Register(tool)
	output := captureReleaseOutput(t, func() {
		cmd := manager.newRunCmd()
		cmd.SetArgs([]string{"--no-cache", "--report=json", "--output=report.json"})
		require.ErrorContains(t, cmd.Execute(), "golangci-lint failed")
	})
	require.Contains(t, output, "0/1")
	require.Equal(t, 1, tool.calls, "the failing tool must actually execute")
	require.FileExists(t, "report.json")
	report, err := os.ReadFile("report.json")
	require.NoError(t, err)
	require.Contains(t, string(report), "exit status 2")
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
