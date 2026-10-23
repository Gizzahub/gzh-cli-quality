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

type versionObservedTool struct {
	*mockCacheableTool
	versionCalls int
	versionErr   error
}

func (t *versionObservedTool) GetVersion() (string, error) {
	t.versionCalls++
	return t.version, t.versionErr
}

func TestCacheVersionIsObservedOncePerTaskAndRefreshedNextRun(t *testing.T) {
	manager, err := cache.NewCacheManager(t.TempDir(), 1<<20, time.Hour)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	dir := t.TempDir()
	files := []string{filepath.Join(dir, "a.go"), filepath.Join(dir, "b.go")}
	for _, path := range files {
		require.NoError(t, os.WriteFile(path, []byte("package same\n"), 0o600))
	}
	tool := &versionObservedTool{mockCacheableTool: newMockCacheableTool("formatter", "Go")}
	executor := NewParallelExecutorWithCache(1, time.Minute, manager)
	plan := &tools.ExecutionPlan{Tasks: []tools.Task{{Tool: tool, Files: files, Options: tools.ExecuteOptions{ProjectRoot: dir}}}}
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
	run(2, 1, true)
	tool.version = "2.0.0"
	run(3, 2, false)
	run(4, 2, true)
	tool.versionErr = errors.New("cannot determine version")
	run(5, 3, false)
	run(6, 4, false)
	tool.versionErr = nil
	require.NoError(t, os.WriteFile(files[1], []byte("package changed\n"), 0o600))
	run(7, 5, false)
	run(8, 5, true)
	plan.Tasks[0].Options.ExtraArgs = []string{string([]byte{0xff})}
	run(9, 6, false)
	run(10, 7, false)
}
