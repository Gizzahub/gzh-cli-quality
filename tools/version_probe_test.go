// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package tools

import (
	"debug/buildinfo"
	"os"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildInfoVersion(t *testing.T) {
	vcs := func(revision, modified string) []debug.BuildSetting {
		return []debug.BuildSetting{{Key: "vcs.revision", Value: revision}, {Key: "vcs.modified", Value: modified}}
	}
	tests := []struct {
		name string
		info buildinfo.BuildInfo
		want string
	}{
		{
			name: "installed module version",
			info: buildinfo.BuildInfo{GoVersion: "go1.26.7", Main: debug.Module{Path: "golang.org/x/tools", Version: "v0.38.0"}},
			want: "golang.org/x/tools v0.38.0 go1.26.7",
		},
		{
			name: "clean devel build",
			info: buildinfo.BuildInfo{GoVersion: "go1.26.7", Main: debug.Module{Path: "example.com/tool", Version: "(devel)"}, Settings: vcs("abc123", "false")},
			want: "example.com/tool (devel) abc123 go1.26.7",
		},
		{
			name: "modified devel build",
			info: buildinfo.BuildInfo{GoVersion: "go1.26.7", Main: debug.Module{Path: "example.com/tool", Version: "(devel)"}, Settings: vcs("abc123", "true")},
		},
		{
			name: "modified build stamped with a version",
			info: buildinfo.BuildInfo{GoVersion: "go1.26.7", Main: debug.Module{Path: "example.com/tool", Version: "v1.2.4-0.20261024000000-abc123+dirty"}, Settings: vcs("abc123", "true")},
		},
		{
			name: "devel build without VCS stamp",
			info: buildinfo.BuildInfo{GoVersion: "go1.26.7", Main: debug.Module{Path: "example.com/tool", Version: "(devel)"}},
		},
		{
			name: "no main module",
			info: buildinfo.BuildInfo{GoVersion: "go1.26.7"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, buildInfoVersion(&tt.info))
		})
	}
}

// A Go program that answers no version flag (goimports) is named by the build
// information in its executable rather than reported as "unknown".
func TestBaseTool_GetVersion_BuildInfoFallback(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	path := fakeExecutable(t, dir, "no-version-flag", "fail")

	info, err := buildinfo.ReadFile(path)
	require.NoError(t, err)
	require.NotEmpty(t, info.Main.Path)
	want := buildInfoVersion(info)
	if want == "" {
		// The fixture is built from this checkout, so a modified tree yields
		// no usable version and GetVersion keeps reporting "unknown".
		want = "unknown"
	}

	version, err := NewBaseTool("no-version-flag", "Go", "no-version-flag", FORMAT).GetVersion()
	require.NoError(t, err)
	assert.Equal(t, want, version)
}

// clippy and cargo fmt run through cargo, whose own version names neither.
func TestRustSubcommandVersionProbe(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	for _, tc := range []struct {
		tool interface {
			QualityTool
			VersionProber
		}
		args []string
	}{
		{NewClippyTool(), []string{"clippy", "--version"}},
		{NewCargoFmtTool(), []string{"fmt", "--version"}},
	} {
		t.Run(tc.tool.Name(), func(t *testing.T) {
			fakeExecutable(t, dir, "cargo", "version")
			version, err := tc.tool.GetVersion()
			require.NoError(t, err)
			assert.Equal(t, tc.args[0]+" --version", version)

			executable, flags := tc.tool.VersionProbe()
			assert.Equal(t, "cargo", executable)
			assert.Equal(t, tc.args, flags)
			assert.False(t, tc.tool.VersionCacheable())

			fakeExecutable(t, dir, "cargo", "fail")
			_, err = tc.tool.GetVersion()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to get "+tc.tool.Name()+" version")
			assert.Contains(t, err.Error(), "component not installed", "stderr explains the failure")
		})
	}
}
