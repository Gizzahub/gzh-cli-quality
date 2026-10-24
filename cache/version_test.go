// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

//go:build darwin || linux

package cache

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Gizzahub/gzh-cli-quality/tools"
)

// The fake tool is testdata/fakeversiontool built once per test run: a real
// native executable, because the version cache refuses "#!" scripts. Copying
// this test binary instead would cost about a second per spawn under -race.
const (
	fakeToolLogEnv     = "GZQ_FAKE_VERSION_TOOL_LOG"
	fakeToolVersionEnv = "GZQ_FAKE_VERSION_TOOL_VERSION"
)

var fakeToolBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "gzq-fake-version-tool-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fakeToolBinary = filepath.Join(dir, "fakeversiontool")
	build := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-o", fakeToolBinary, "./testdata/fakeversiontool/main.go")
	if output, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build fake version tool: %v\n%s", err, output)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

type fakeVersionTool struct {
	t       *testing.T
	exe     string
	logPath string
	tool    *tools.BaseTool
}

// newFakeVersionTool installs a fake tool named fakefmt in a new directory and
// reports version for every probe until setVersion changes it.
func newFakeVersionTool(t *testing.T, version string) *fakeVersionTool {
	t.Helper()
	dir := t.TempDir()
	f := &fakeVersionTool{t: t, exe: filepath.Join(dir, "fakefmt"), logPath: filepath.Join(dir, "probes.log")}
	f.install(f.exe)
	f.tool = tools.NewBaseTool("fakefmt", "Go", f.exe, tools.FORMAT)
	t.Setenv(fakeToolLogEnv, f.logPath)
	f.setVersion(version)
	return f
}

func (f *fakeVersionTool) setVersion(version string) { f.t.Setenv(fakeToolVersionEnv, version) }

// install copies the fake tool binary to path.
func (f *fakeVersionTool) install(path string) {
	f.t.Helper()
	src, err := os.Open(fakeToolBinary)
	if err != nil {
		f.t.Fatal(err)
	}
	defer src.Close()
	dst, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o755)
	if err != nil {
		f.t.Fatal(err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		f.t.Fatal(err)
	}
	if err := dst.Close(); err != nil {
		f.t.Fatal(err)
	}
}

// probes counts how many times the fake executable has been spawned.
func (f *fakeVersionTool) probes() int {
	f.t.Helper()
	data, err := os.ReadFile(f.logPath)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		f.t.Fatal(err)
	}
	return strings.Count(string(data), "\n")
}

func requireVersion(t *testing.T, vc *VersionCache, tool tools.QualityTool, want string) {
	t.Helper()
	got, err := vc.ToolVersion(tool)
	if err != nil {
		t.Fatalf("ToolVersion: %v", err)
	}
	if got != want {
		t.Fatalf("ToolVersion = %q, want %q", got, want)
	}
}

func requireProbes(t *testing.T, f *fakeVersionTool, want int) {
	t.Helper()
	if got := f.probes(); got != want {
		t.Fatalf("probes = %d, want %d", got, want)
	}
}

func ctimeNs(t *testing.T, path string) int64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	id, ok := statIdentity(info)
	if !ok {
		t.Fatal("stat identity unavailable")
	}
	return id.changeTimeNs
}

// changeCtime runs change, which must update path's ctime, and repeats it
// until the ctime actually differs from before. Linux stamps ctime from a
// coarse clock (one tick can be several ms), so a change made right after the
// file was written can carry the same ctime; the 1ms waits only cover that.
func changeCtime(t *testing.T, path string, before int64, change func(attempt int)) {
	t.Helper()
	for attempt := range 200 {
		change(attempt)
		if ctimeNs(t, path) != before {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("ctime did not change")
}

func TestVersionCache_HitSkipsProbe(t *testing.T) {
	f := newFakeVersionTool(t, "fakefmt v1")
	dir := t.TempDir()
	vc := NewVersionCache(dir)

	requireVersion(t, vc, f.tool, "fakefmt v1")
	requireProbes(t, f, 1)
	requireVersion(t, vc, f.tool, "fakefmt v1")
	requireProbes(t, f, 1)

	// A new process reads the persisted entry.
	requireVersion(t, NewVersionCache(dir), f.tool, "fakefmt v1")
	requireProbes(t, f, 1)
}

func TestVersionCache_ReplacedBinaryReprobes(t *testing.T) {
	f := newFakeVersionTool(t, "fakefmt v1")
	vc := NewVersionCache(t.TempDir())
	requireVersion(t, vc, f.tool, "fakefmt v1")

	before, err := os.Stat(f.exe)
	if err != nil {
		t.Fatal(err)
	}
	replacement := filepath.Join(filepath.Dir(f.exe), "fakefmt.new")
	// Same bytes, size and (restored) mtime; the inode differs, and so does the
	// ctime, which no user call can set. VersionKey_EveryFieldInvalidates covers
	// the inode field on its own.
	f.install(replacement)
	if err := os.Chtimes(replacement, before.ModTime(), before.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, f.exe); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(f.exe)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(before, after) {
		t.Fatal("replacement kept the inode")
	}
	f.setVersion("fakefmt v2")

	requireVersion(t, vc, f.tool, "fakefmt v2")
	requireProbes(t, f, 2)
	requireVersion(t, vc, f.tool, "fakefmt v2")
	requireProbes(t, f, 2)
}

func TestVersionCache_InPlaceRewriteReprobes(t *testing.T) {
	f := newFakeVersionTool(t, "fakefmt v1")
	vc := NewVersionCache(t.TempDir())
	requireVersion(t, vc, f.tool, "fakefmt v1")

	before, err := os.Stat(f.exe)
	if err != nil {
		t.Fatal(err)
	}
	beforeCtime := ctimeNs(t, f.exe)
	// Rewrite the last byte through the same inode, then restore size and
	// mtime as cp -p would: only the ctime is left to tell the files apart.
	// The byte is written back unchanged so the executable stays runnable
	// (macOS kills a signed binary whose pages were altered); the new
	// version comes from the environment instead.
	changeCtime(t, f.exe, beforeCtime, func(int) {
		file, err := os.OpenFile(f.exe, os.O_RDWR, 0)
		if err != nil {
			t.Fatal(err)
		}
		last := make([]byte, 1)
		if _, err := file.ReadAt(last, before.Size()-1); err != nil {
			t.Fatal(err)
		}
		if _, err := file.WriteAt(last, before.Size()-1); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(f.exe, before.ModTime(), before.ModTime()); err != nil {
			t.Fatal(err)
		}
	})
	after, err := os.Stat(f.exe)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("rewrite changed more than the ctime")
	}
	f.setVersion("fakefmt v1-patched")

	requireVersion(t, vc, f.tool, "fakefmt v1-patched")
	requireProbes(t, f, 2)
}

func TestVersionCache_MtimeTouchReprobes(t *testing.T) {
	f := newFakeVersionTool(t, "fakefmt v1")
	vc := NewVersionCache(t.TempDir())
	requireVersion(t, vc, f.tool, "fakefmt v1")

	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(f.exe, future, future); err != nil {
		t.Fatal(err)
	}

	requireVersion(t, vc, f.tool, "fakefmt v1")
	requireProbes(t, f, 2)
	requireVersion(t, vc, f.tool, "fakefmt v1")
	requireProbes(t, f, 2)
}

func TestVersionCache_CtimeOnlyChangeReprobes(t *testing.T) {
	f := newFakeVersionTool(t, "fakefmt v1")
	vc := NewVersionCache(t.TempDir())
	requireVersion(t, vc, f.tool, "fakefmt v1")

	before, err := os.Stat(f.exe)
	if err != nil {
		t.Fatal(err)
	}
	changeCtime(t, f.exe, ctimeNs(t, f.exe), func(attempt int) {
		mode := os.FileMode(0o755)
		if attempt%2 == 0 {
			mode = 0o775
		}
		if err := os.Chmod(f.exe, mode); err != nil {
			t.Fatal(err)
		}
	})
	after, err := os.Stat(f.exe)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("chmod changed more than the ctime")
	}

	requireVersion(t, vc, f.tool, "fakefmt v1")
	requireProbes(t, f, 2)
}

func TestVersionCache_CorruptEntryRecovers(t *testing.T) {
	f := newFakeVersionTool(t, "fakefmt v1")
	dir := t.TempDir()
	vc := NewVersionCache(dir)
	requireVersion(t, vc, f.tool, "fakefmt v1")

	entries, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries = %v, %v", entries, err)
	}
	for _, corrupt := range [][]byte{[]byte("{not json"), {}, []byte(`{"format":"1","key":{},"version":"stale"}`)} {
		if err := os.WriteFile(entries[0], corrupt, 0o600); err != nil {
			t.Fatal(err)
		}
		requireVersion(t, vc, f.tool, "fakefmt v1")
	}
	requireProbes(t, f, 4)
	requireVersion(t, vc, f.tool, "fakefmt v1") // Rewritten entry hits.
	requireProbes(t, f, 4)
}

func TestVersionCache_UnreadableDirectoryStillProbes(t *testing.T) {
	f := newFakeVersionTool(t, "fakefmt v1")
	notDir := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(notDir, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	vc := NewVersionCache(filepath.Join(notDir, "versions"))
	requireVersion(t, vc, f.tool, "fakefmt v1")
	requireVersion(t, vc, f.tool, "fakefmt v1")
	requireProbes(t, f, 2)
}

type flagsTool struct {
	*tools.BaseTool
	flags []string
}

func (t *flagsTool) VersionProbe() (string, []string) {
	executable, _ := t.BaseTool.VersionProbe()
	return executable, t.flags
}

func TestVersionCache_ProbeFlagsAreKeyed(t *testing.T) {
	f := newFakeVersionTool(t, "fakefmt v1")
	vc := NewVersionCache(t.TempDir())
	_, defaults := f.tool.VersionProbe()
	tool := &flagsTool{BaseTool: f.tool, flags: defaults}
	requireVersion(t, vc, tool, "fakefmt v1")
	requireVersion(t, vc, tool, "fakefmt v1")
	requireProbes(t, f, 1)

	tool.flags = []string{"--version"}
	requireVersion(t, vc, tool, "fakefmt v1")
	requireProbes(t, f, 2)
}

func TestVersionCache_ProbeErrorSurfacesAndIsNotRecorded(t *testing.T) {
	f := newFakeVersionTool(t, "fakefmt v1")
	dir := t.TempDir()
	vc := NewVersionCache(dir)
	missing := tools.NewBaseTool("fakefmt", "Go", filepath.Join(t.TempDir(), "fakefmt"), tools.FORMAT)
	if _, err := vc.ToolVersion(missing); err == nil {
		t.Fatal("missing tool: want the GetVersion error")
	}

	// Empty output on every flag is "unknown": returned, never recorded.
	f.setVersion("")
	for range 2 {
		requireVersion(t, vc, f.tool, "unknown")
	}
	if entries, _ := filepath.Glob(filepath.Join(dir, "*.json")); len(entries) != 0 {
		t.Fatalf("recorded entries = %v", entries)
	}
}
