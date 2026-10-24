package quality

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

type lazyFileState struct {
	mode     fs.FileMode
	modified time.Time
	content  string
}

func lazySnapshot(t *testing.T, root string) map[string]lazyFileState {
	t.Helper()
	state := make(map[string]lazyFileState)
	require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		item := lazyFileState{mode: info.Mode(), modified: info.ModTime()}
		if !entry.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			item.content = string(data)
		}
		state[path] = item
		return nil
	}))
	return state
}

func isolateLazyHome(t *testing.T, home string) {
	t.Helper()
	for _, name := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(name, home)
	}
}

func TestQualityInformationalPathsDoNotInitializeManager(t *testing.T) {
	for _, settings := range []string{"absent", "malformed", "enabled"} {
		t.Run(settings, func(t *testing.T) {
			directory := t.TempDir()
			home := filepath.Join(directory, "home")
			project := filepath.Join(directory, "project")
			require.NoError(t, os.Mkdir(home, 0o700))
			require.NoError(t, os.Mkdir(project, 0o700))
			isolateLazyHome(t, home)
			t.Chdir(project)
			if settings != "absent" {
				data := "cache:\n  enabled: true\n  directory: " + filepath.ToSlash(filepath.Join(directory, "configured-cache")) + "\n"
				if settings == "malformed" {
					data = "cache: [broken\n"
				}
				require.NoError(t, os.WriteFile(".gzquality.yaml", []byte(data), 0o600))
			}
			for _, args := range [][]string{nil, {"--help"}, {"help"}, {"--version"}, {"nosuch"}, {"--invalid"}, {"run", "--invalid"}, {"run", "--help"}, {"tool"}, {"tool", "nosuch"}, {"tool", "gofumpt", "--help"}} {
				for _, embedded := range []bool{false, true} {
					wantError := false
					for _, arg := range args {
						wantError = wantError || arg == "nosuch" || arg == "--invalid"
					}
					before := lazySnapshot(t, directory)
					command := NewQualityCmd()
					command.Version = "fixture-version"
					if embedded {
						parent := &cobra.Command{Use: "gz"}
						parent.AddCommand(command)
						command = parent
						if len(args) == 1 && args[0] == "help" {
							args = []string{"help", "quality"}
						} else {
							args = append([]string{"quality"}, args...)
						}
					}
					var output bytes.Buffer
					command.SetOut(&output)
					command.SetErr(&output)
					command.SetArgs(args)
					err := command.Execute()
					if wantError {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
						require.NotEmpty(t, output.String())
					}
					require.Equal(t, before, lazySnapshot(t, directory), "args=%v embedded=%v", args, embedded)
				}
			}
		})
	}
}

func TestQualityLazyOperationRetainsCacheAndErrors(t *testing.T) {
	// Reuse the native subprocess fixture on every OS; never invoke an installed formatter.
	repo, err := os.Getwd()
	require.NoError(t, err)
	bin := t.TempDir()
	tool := filepath.Join(bin, "gofumpt")
	if runtime.GOOS == "windows" {
		tool += ".exe"
	}
	build := exec.Command("go", "build", "-o", tool, filepath.Join(repo, "tools/testdata/fake-tool/main.go"))
	data, err := build.CombinedOutput()
	require.NoError(t, err, "%s", data)
	require.NoError(t, os.WriteFile(tool+".mode", []byte("component-version"), 0o600))
	require.NoError(t, os.WriteFile(tool+".versions", []byte(`{"fmt":"gofumpt v0.0.1"}`), 0o600))
	t.Setenv("PATH", bin)
	t.Setenv("RUSTUP_TOOLCHAIN", "") // The shared fixture can select component versions by this variable.
	project := t.TempDir()
	isolateLazyHome(t, project)
	t.Chdir(project)
	project, err = os.Getwd()
	require.NoError(t, err)
	cacheDir := filepath.Join(project, "result-cache")
	require.NoError(t, os.WriteFile(".gzquality.yaml", []byte("cache:\n  enabled: true\n  directory: "+filepath.ToSlash(cacheDir)+"\n"), 0o600))
	require.NoError(t, os.WriteFile("sample.go", []byte("package sample\n"), 0o600))
	command := NewQualityCmd()
	parent := &cobra.Command{Use: "gz"}
	parentCalls := 0
	parent.PersistentPreRun = func(*cobra.Command, []string) { parentCalls++ }
	parent.AddCommand(command)
	for i := 0; i < 2; i++ {
		parent.SetArgs([]string{"quality", "tool", "gofumpt", "--files", filepath.Join(project, "sample.go")})
		output := captureReleaseOutput(t, func() { require.NoError(t, parent.Execute()) })
		require.Contains(t, output, "🚀", "the real command must execute a task")
	}
	require.Equal(t, 2, parentCalls, "embedding hooks must remain intact")
	require.DirExists(t, cacheDir)
	calls, err := os.ReadFile(tool + ".calls")
	require.NoError(t, err)
	require.Equal(t, 2, bytes.Count(calls, []byte("\n")), "exactly one successful probe and one execution: %s", calls)
	require.Equal(t, 1, bytes.Count(calls, []byte("sample.go")), "the second operation must hit the result cache: %s", calls)
	parent.SetArgs([]string{"quality", "cache-stats"})
	output := captureReleaseOutput(t, func() { require.NoError(t, parent.Execute()) })
	require.Contains(t, output, "캐시 히트: 1회", "the command lifetime must retain its initialized manager")
	require.NoError(t, os.WriteFile("sample.go", []byte("package changed\n"), 0o600))
	require.NoError(t, os.WriteFile(tool+".mode", []byte("fail"), 0o600))
	parent.SetArgs([]string{"quality", "tool", "gofumpt", "--files", filepath.Join(project, "sample.go")})
	captureReleaseOutput(t, func() { require.ErrorContains(t, parent.Execute(), "gofumpt failed") })
	command = NewQualityCmd()
	command.SetArgs([]string{"tool", "gofumpt", "--files", filepath.Join(project, "sample.go")})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	captureReleaseOutput(t, func() { require.ErrorIs(t, command.ExecuteContext(ctx), context.Canceled) })
}
