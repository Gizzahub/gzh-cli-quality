package executor

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Gizzahub/gzh-cli-quality/cache"
	"github.com/Gizzahub/gzh-cli-quality/tools"
	"github.com/stretchr/testify/require"
)

func TestRustComponentOnlyUpgradeInvalidatesResultCache(t *testing.T) {
	t.Setenv("RUSTUP_TOOLCHAIN", "")
	dir := t.TempDir()
	name := "cargo"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, name)
	output, err := exec.Command("go", "build", "-buildvcs=false", "-o", path, "../tools/testdata/fake-tool").CombinedOutput()
	require.NoError(t, err, "%s", output)
	require.NoError(t, os.WriteFile(path+".mode", []byte("component-version"), 0o600))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	binary, err := os.ReadFile(path)
	require.NoError(t, err)
	rustfmtPath := filepath.Join(dir, "rustfmt"+filepath.Ext(path))
	require.NoError(t, os.WriteFile(rustfmtPath, binary, 0o755))
	require.NoError(t, os.WriteFile(rustfmtPath+".mode", []byte("component-version"), 0o600))
	before, err := os.Stat(path)
	require.NoError(t, err)
	file := filepath.Join(dir, "source.rs")
	require.NoError(t, os.WriteFile(file, []byte("fn main() {}\n"), 0o600))
	for _, tool := range []tools.QualityTool{tools.NewClippyTool(), tools.NewCargoFmtTool(), tools.NewRustfmtTool()} {
		t.Run(tool.Name(), func(t *testing.T) {
			componentPath := path
			if tool.Name() == "rustfmt" {
				componentPath = rustfmtPath
			}
			cacheDirectory := t.TempDir()
			manager, err := cache.NewCacheManager(cacheDirectory, 1<<20, time.Hour)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, manager.Close()) })
			executor := NewParallelExecutorWithCache(1, time.Minute, manager)
			plan := &tools.ExecutionPlan{Tasks: []tools.Task{{Tool: tool, Files: []string{file}, Options: tools.ExecuteOptions{ProjectRoot: dir}}}}
			versions := func(suffix string) {
				t.Helper()
				data, err := json.Marshal(map[string]string{"clippy": "clippy " + suffix, "fmt": "rustfmt " + suffix})
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(componentPath+".versions", data, 0o600))
			}
			run := func(cached bool) {
				t.Helper()
				results, err := executor.Execute(context.Background(), plan)
				require.NoError(t, err)
				require.Len(t, results, 1)
				require.True(t, results[0].Success)
				require.Equal(t, cached, results[0].Cached)
			}
			versions("1.0.0")
			run(false)
			run(true)
			versions("2.0.0") // Only component metadata changes; cargo remains byte-for-byte identical.
			run(false)
			run(true)
			require.NoError(t, os.WriteFile(componentPath+".versions", []byte(`{}`), 0o600))
			run(false)
			run(false) // Unknown must not read or populate result cache.
			for _, invalid := range []string{"unknown", "(devel)", "not-a-version", "1.8"} {
				versions(invalid)
				run(false)
				run(false) // A component prefix must not make a sentinel cacheable.
			}
			require.EqualValues(t, 2, manager.Stats().Entries, "invalid versions must not populate entries")
			require.NoError(t, os.WriteFile(componentPath+".versions", []byte(`invalid`), 0o600))
			run(false) // A failed probe still runs the tool without caching.
			versions("2.0.0")
			run(true)
			// The task cwd selects its component independently from the process cwd.
			project := t.TempDir()
			plan.Tasks[0].Options.ProjectRoot = project
			local := filepath.Join(project, ".quality-component-versions")
			require.NoError(t, os.WriteFile(local, []byte(`{"clippy":"clippy 0.1.90", "fmt":"rustfmt 1.8.0"}`), 0o600))
			run(false)
			run(true)
			require.NoError(t, os.WriteFile(local, []byte(`{"clippy":"clippy 0.1.91", "fmt":"rustfmt 1.9.0", "clippy@nightly":"clippy 0.1.92-nightly", "fmt@nightly":"rustfmt 1.10.0-nightly"}`), 0o600))
			run(false)
			run(true) // Same task cwd/options, component selected there changed.
			plan.Tasks[0].Options.Env = map[string]string{"RUSTUP_TOOLCHAIN": "nightly"}
			run(false)
			run(true)
			storage, err := cache.NewFilesystemStorage(cacheDirectory)
			require.NoError(t, err)
			keys, err := storage.List()
			require.NoError(t, err)
			observed := map[string]bool{}
			for _, key := range keys {
				data, err := storage.Read(key)
				require.NoError(t, err)
				var entry cache.CachedResult
				require.NoError(t, json.Unmarshal(data, &entry))
				observed[entry.Key.ToolVersion] = true
			}
			prefix := "clippy "
			if tool.Name() != "clippy" {
				prefix = "rustfmt "
			}
			suffixes := []string{"0.1.90", "0.1.91", "0.1.92-nightly"}
			if tool.Name() != "clippy" {
				suffixes = []string{"1.8.0", "1.9.0", "1.10.0-nightly"}
			}
			for _, suffix := range suffixes {
				require.True(t, observed[prefix+suffix], "stored version must describe selected context: %v", observed)
			}
		})
	}
	t.Run("unknown goimports provenance bypasses result cache", func(t *testing.T) {
		name := "goimports"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, binary, 0o755))
		require.NoError(t, os.WriteFile(path+".mode", []byte("component-version"), 0o600))
		manager, err := cache.NewCacheManager(t.TempDir(), 1<<20, time.Hour)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, manager.Close()) })
		executor := NewParallelExecutorWithCache(1, time.Minute, manager)
		goFile := filepath.Join(dir, "source.go")
		require.NoError(t, os.WriteFile(goFile, []byte("package example\n"), 0o600))
		plan := &tools.ExecutionPlan{Tasks: []tools.Task{{Tool: tools.NewGoimportsTool(), Files: []string{goFile}}}}
		for range 2 {
			results, err := executor.Execute(context.Background(), plan)
			require.NoError(t, err)
			require.Len(t, results, 1)
			require.True(t, results[0].Success)
			require.False(t, results[0].Cached)
		}
		require.Zero(t, manager.Stats().Entries)
		calls, err := os.ReadFile(path + ".calls")
		require.NoError(t, err)
		require.Equal(t, 2, strings.Count(string(calls), "-l "+goFile+"\n"))
		require.Equal(t, 2, strings.Count(string(calls), "--version\n"))
	})
	afterBytes, err := os.ReadFile(path)
	require.NoError(t, err)
	after, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, sha256.Sum256(binary), sha256.Sum256(afterBytes))
	require.True(t, before.ModTime().Equal(after.ModTime()), "cargo launcher must not change during component-only upgrade")
}
