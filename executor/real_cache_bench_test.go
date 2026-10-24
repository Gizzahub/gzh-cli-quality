package executor

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Gizzahub/gzh-cli-quality/cache"
	"github.com/Gizzahub/gzh-cli-quality/tools"
	"github.com/stretchr/testify/require"
)

type measuredFormatter struct {
	tools.QualityTool
	versions atomic.Int64
	runs     atomic.Int64
}

func (t *measuredFormatter) GetVersion() (string, error) {
	t.versions.Add(1)
	return t.QualityTool.GetVersion()
}

// probedFormatter forwards the wrapped tool's version probe, so the version
// cache treats it exactly like the unwrapped tool. A bare measuredFormatter is
// not a tools.VersionProber and so is probed on every run, which measures the
// same path with the version cache off.
type probedFormatter struct {
	*measuredFormatter
}

func (t probedFormatter) VersionProbe() (executable string, flags []string) {
	return t.QualityTool.(tools.VersionProber).VersionProbe()
}

func (t probedFormatter) VersionCacheable() bool {
	return t.QualityTool.(tools.VersionProber).VersionCacheable()
}

func (t *measuredFormatter) Execute(ctx context.Context, files []string, options tools.ExecuteOptions) (*tools.Result, error) {
	t.runs.Add(1)
	return t.QualityTool.Execute(ctx, files, options)
}

// Opt-in measurement of a real already-installed formatter, never a sleeping or
// no-op substitute. The source revision fixes the workload for before/after runs.
func BenchmarkCache_InstalledGofumpt(b *testing.B) {
	requested := os.Getenv("QUALITY_BENCH_GOFUMPT")
	if requested == "" {
		b.Skip("set QUALITY_BENCH_GOFUMPT to an already-installed gofumpt executable")
	}
	resolved, err := exec.LookPath("gofumpt")
	require.NoError(b, err)
	resolved, err = filepath.EvalSymlinks(resolved)
	require.NoError(b, err)
	requested, err = filepath.EvalSymlinks(requested)
	require.NoError(b, err)
	require.Equal(b, requested, resolved, "measure exactly the selected installed executable")
	ref := os.Getenv("QUALITY_BENCH_SOURCE_REF")
	if ref == "" {
		ref = "HEAD"
	}
	git := func(args ...string) []byte {
		cmd := exec.Command("git", args...)
		cmd.Dir = ".."
		output, err := cmd.CombinedOutput()
		require.NoError(b, err, "%s", output)
		return output
	}
	revision := strings.TrimSpace(string(git("rev-parse", ref)))
	project := filepath.Join(b.TempDir(), "project")
	var files []string
	var bytes int
	digest := sha256.New()
	contents := make(map[string][sha256.Size]byte)
	for _, name := range strings.Split(string(git("ls-tree", "-rz", "--name-only", revision)), "\x00") {
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || strings.Contains(name, "testdata/") {
			continue
		}
		data := git("show", revision+":"+name)
		path := filepath.Join(project, filepath.FromSlash(name))
		require.NoError(b, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(b, os.WriteFile(path, data, 0o600))
		files = append(files, path)
		contents[path] = sha256.Sum256(data)
		bytes += len(data)
		fmt.Fprintf(digest, "%s\x00%s\x00", name, data)
	}
	require.Greater(b, len(files), 0)
	b.Cleanup(func() {
		for path, expected := range contents {
			data, err := os.ReadFile(path)
			require.NoError(b, err)
			require.Equal(b, expected, sha256.Sum256(data), "formatter checks must preserve bytes")
		}
	})
	version, err := tools.NewGofumptTool().GetVersion()
	require.NoError(b, err)
	b.Logf("tool=%s version=%s revision=%s files=%d bytes=%d workload_sha256=%x", resolved, version, revision, len(files), bytes, digest.Sum(nil))
	b.Run("VersionProbe", func(b *testing.B) {
		tool := tools.NewGofumptTool()
		for range b.N {
			got, err := tool.GetVersion()
			require.NoError(b, err)
			require.Equal(b, version, got)
		}
	})
	b.Run("CacheMetadataReads", func(b *testing.B) {
		tool := tools.NewGofumptTool()
		manager, err := cache.NewCacheManager(b.TempDir(), 100<<20, time.Hour)
		require.NoError(b, err)
		b.Cleanup(func() { require.NoError(b, manager.Close()) })
		executor := NewParallelExecutorWithCache(1, time.Minute, manager)
		plan := &tools.ExecutionPlan{Tasks: []tools.Task{{Tool: tool, Files: files, Options: tools.ExecuteOptions{ProjectRoot: project}}}}
		results, err := executor.Execute(context.Background(), plan)
		require.NoError(b, err)
		require.Len(b, results, 1)
		require.True(b, results[0].Success)
		var keys []cache.CacheKey
		for _, file := range files {
			key, err := cache.GenerateKey(file, &cacheVersionTool{QualityTool: tool, version: version}, plan.Tasks[0].Options)
			require.NoError(b, err)
			keys = append(keys, key)
		}
		b.ResetTimer()
		for range b.N {
			for _, key := range keys {
				cached, err := manager.Get(key)
				require.NoError(b, err)
				require.True(b, cached.Result.Success)
			}
		}
		b.StopTimer()
		b.ReportMetric(float64(len(keys)), "cache_reads/op")
	})
	// WarmHitNoVersionCache is WarmHit with the version cache off: the result
	// cache still hits, but the tool version is probed on every run.
	for _, mode := range []string{"NoCache", "ColdMiss", "WarmHit", "WarmHitNoVersionCache"} {
		b.Run(mode, func(b *testing.B) {
			measured := &measuredFormatter{QualityTool: tools.NewGofumptTool()}
			var tool tools.QualityTool = probedFormatter{measured}
			if mode == "WarmHitNoVersionCache" {
				tool = measured
			}
			warm := mode == "WarmHit" || mode == "WarmHitNoVersionCache"
			manager, err := cache.NewCacheManager(b.TempDir(), 100<<20, time.Hour)
			require.NoError(b, err)
			b.Cleanup(func() { require.NoError(b, manager.Close()) })
			executor := NewParallelExecutor(1, time.Minute)
			if mode != "NoCache" {
				executor.SetCache(manager)
			}
			plan := &tools.ExecutionPlan{Tasks: []tools.Task{{Tool: tool, Files: files, Options: tools.ExecuteOptions{ProjectRoot: project}}}}
			run := func(wantCached bool) {
				results, err := executor.Execute(context.Background(), plan)
				require.NoError(b, err)
				require.Len(b, results, 1)
				require.True(b, results[0].Success, "%+v", results[0])
				require.Equal(b, len(files), results[0].FilesProcessed)
				require.Equal(b, wantCached, results[0].Cached)
			}
			if warm {
				run(false)
			}
			measured.versions.Store(0)
			measured.runs.Store(0)
			b.ResetTimer()
			for range b.N {
				if mode == "ColdMiss" {
					b.StopTimer()
					require.NoError(b, manager.InvalidateAll())
					b.StartTimer()
				}
				run(warm)
			}
			b.StopTimer()
			b.ReportMetric(float64(measured.versions.Load())/float64(b.N), "version_calls/op")
			b.ReportMetric(float64(measured.runs.Load())/float64(b.N), "tool_runs/op")
		})
	}
}
