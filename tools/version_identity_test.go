package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRustComponentVersionRejectsNonVersions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("RUSTUP_TOOLCHAIN", "")
	path := fakeExecutable(t, dir, "cargo", "component-version")
	for _, tail := range []string{"unknown", "(devel)", "project1", "1.8", "1.8.0-", "01.8.0"} {
		require.NoError(t, os.WriteFile(path+".versions", []byte(`{"clippy":"clippy `+tail+`", "fmt":"rustfmt `+tail+`"}`), 0o600))
		for _, tool := range []QualityTool{NewClippyTool(), NewCargoFmtTool()} {
			version, err := tool.GetVersion()
			require.NoError(t, err)
			require.Equal(t, "unknown", version, tail)
		}
	}
	for _, tail := range []string{"0.1.90 (hash 2025-09-14)", "1.8.0-stable (hash 2025-09-14)", "1.8.0-nightly (hash 2025-09-14)"} {
		require.NoError(t, os.WriteFile(path+".versions", []byte(`{"clippy":"clippy `+tail+`", "fmt":"rustfmt `+tail+`"}`), 0o600))
		for _, tool := range []QualityTool{NewClippyTool(), NewCargoFmtTool()} {
			version, err := tool.GetVersion()
			require.NoError(t, err)
			require.NotEqual(t, "unknown", version, tail)
		}
	}
}

func TestRustComponentInflightCancelBoundsInheritedPipe(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	path := fakeExecutable(t, dir, "cargo", "hold-version-pipe")
	t.Cleanup(func() {
		require.NoError(t, os.WriteFile(path+".release", []byte("release"), 0o600))
		if _, err := os.Stat(path + ".child.pid"); os.IsNotExist(err) {
			return
		}
		require.Eventually(t, func() bool { _, err := os.Stat(path + ".child.done"); return err == nil }, 3*time.Second, 20*time.Millisecond, "owned pipe holder must exit")
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := NewClippyTool().GetVersionForExecution(ctx, ExecuteOptions{}); done <- err }()
	require.Eventually(t, func() bool { _, err := os.Stat(path + ".child.pid"); return err == nil }, 5*time.Second, 20*time.Millisecond)
	start := time.Now()
	cancel()
	select {
	case err := <-done:
		require.True(t, errors.Is(err, context.Canceled), "cancellation cause lost: %v", err)
		require.Less(t, time.Since(start), 2500*time.Millisecond, "pipe holder must not retain probe beyond WaitDelay")
	case <-time.After(12 * time.Second):
		t.Fatal("canceled probe did not return")
	}
}

func TestCargoComponentProbeUsesSubcommand(t *testing.T) {
	t.Setenv("RUSTUP_TOOLCHAIN", "")
	dir := t.TempDir()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	path := fakeExecutable(t, dir, "cargo", "component-version")
	require.NoError(t, os.WriteFile(path+".versions", []byte(`{"clippy":"clippy 0.1.90", "fmt":"rustfmt 1.8.0"}`), 0o600))
	for _, test := range []struct {
		tool QualityTool
		want string
	}{{NewClippyTool(), "clippy 0.1.90"}, {NewCargoFmtTool(), "rustfmt 1.8.0"}} {
		version, err := test.tool.GetVersion()
		require.NoError(t, err)
		require.Equal(t, test.want, version)
		require.False(t, test.tool.(VersionProber).VersionCacheable())
	}
	log, err := os.ReadFile(path + ".calls")
	require.NoError(t, err)
	require.Equal(t, "clippy --version\nfmt --version\n", string(log))
	for _, output := range []string{`{"clippy":"cargo 1.90", "fmt":"cargo 1.90"}`, `{}`} {
		require.NoError(t, os.WriteFile(path+".versions", []byte(output), 0o600))
		for _, tool := range []QualityTool{NewClippyTool(), NewCargoFmtTool()} {
			version, err := tool.GetVersion()
			require.NoError(t, err)
			require.Equal(t, "unknown", version)
		}
	}
	require.NoError(t, os.WriteFile(path+".versions", []byte("invalid"), 0o600))
	_, err = NewClippyTool().GetVersion()
	require.Error(t, err)
}

func TestRustComponentProbeExecutionContext(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("RUSTUP_TOOLCHAIN", "ambient")
	path := fakeExecutable(t, dir, "cargo", "component-version")
	require.NoError(t, os.WriteFile(path+".versions", []byte(`{"clippy@ambient":"clippy 0.1.90-stable", "fmt@ambient":"rustfmt 1.8.0-stable"}`), 0o600))
	project := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(project, ".quality-component-versions"), []byte(`{"clippy@explicit":"clippy 0.1.91-nightly", "fmt@explicit":"rustfmt 1.9.0-nightly"}`), 0o600))
	rustfmtPath := fakeExecutable(t, dir, "rustfmt", "component-version")
	// Both launchers share the controlled component versions, independent of cwd.
	require.NoError(t, os.WriteFile(rustfmtPath+".versions", []byte(`{"fmt@ambient":"rustfmt 1.8.0-stable"}`), 0o600))
	for _, tool := range []QualityTool{NewClippyTool(), NewCargoFmtTool(), NewRustfmtTool()} {
		version, err := tool.GetVersion()
		require.NoError(t, err)
		require.Contains(t, version, "stable")
		prober, ok := tool.(interface {
			GetVersionForExecution(context.Context, ExecuteOptions) (string, error)
		})
		require.True(t, ok, "%s must probe the component selected for execution", tool.Name())
		version, err = prober.GetVersionForExecution(context.Background(), ExecuteOptions{ProjectRoot: project, Env: map[string]string{"RUSTUP_TOOLCHAIN": "explicit"}})
		require.NoError(t, err)
		require.Contains(t, version, "nightly")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = prober.GetVersionForExecution(ctx, ExecuteOptions{})
		require.Error(t, err)
	}
}

func TestRustComponentHonorsConfiguredVersionCommand(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("RUSTUP_TOOLCHAIN", "")
	paths := map[string]string{}
	for _, name := range []string{"cargo", "rustfmt"} {
		paths[name] = fakeExecutable(t, dir, name, "component-version")
		require.NoError(t, os.WriteFile(paths[name]+".versions", []byte(`{"clippy":"clippy 0.1.90", "fmt":"rustfmt 1.8.0-nightly"}`), 0o600))
	}
	type configurableProbe interface {
		QualityTool
		VersionProber
		SetVersionCommand(...string)
		GetVersionForExecution(context.Context, ExecuteOptions) (string, error)
	}
	for _, tool := range []configurableProbe{NewClippyTool(), NewCargoFmtTool(), NewRustfmtTool()} {
		executable, args := tool.VersionProbe()
		expected := append(args, "--verbose")
		configured := append([]string(nil), expected...)
		tool.SetVersionCommand(configured...)
		configured[0] = "mutated"
		_, observed := tool.VersionProbe()
		require.Equal(t, expected, observed)
		observed[0] = "mutated"
		_, observed = tool.VersionProbe()
		require.Equal(t, expected, observed, "probe argv must not alias configuration")
		version, err := tool.GetVersionForExecution(context.Background(), ExecuteOptions{ProjectRoot: dir})
		require.NoError(t, err)
		require.NotEqual(t, "unknown", version)
		calls, err := os.ReadFile(paths[executable] + ".calls")
		require.NoError(t, err)
		require.Contains(t, string(calls), strings.Join(expected, " ")+"\n")
		require.False(t, tool.VersionCacheable())
	}
}
