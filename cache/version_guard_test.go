// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

//go:build darwin || linux

package cache

// Tests for the guards that refuse to key or record a version: files that do
// not pin it, opt-out, change during the probe, malformed entries and values.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/Gizzahub/gzh-cli-quality/tools"
)

func TestVersionCache_RefusesFilesThatDoNotPinTheVersion(t *testing.T) {
	f := newFakeVersionTool(t, "fakefmt v1")
	dir := filepath.Dir(f.exe)

	script := filepath.Join(dir, "shimfmt")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexec "+f.exe+" \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	renamed := filepath.Join(t.TempDir(), "otherfmt")
	if err := os.Symlink(f.exe, renamed); err != nil {
		t.Fatal(err)
	}
	hardlinkDir := t.TempDir()
	multiLinked := filepath.Join(hardlinkDir, "linkedfmt")
	f.install(multiLinked)
	if err := os.Link(multiLinked, filepath.Join(hardlinkDir, "linkedfmt2")); err != nil {
		t.Fatal(err)
	}

	for name, exe := range map[string]string{"script": script, "renaming symlink": renamed, "hard link": multiLinked} {
		t.Run(name, func(t *testing.T) {
			if _, ok := currentVersionKey("fakefmt", exe, []string{"--version"}); ok {
				t.Fatalf("%s must not be cached", exe)
			}
		})
	}

	// Same-name symlinks (Homebrew, go install into a linked dir) resolve to
	// the target and are cached under its path.
	linkDir := t.TempDir()
	if err := os.Symlink(f.exe, filepath.Join(linkDir, "fakefmt")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", linkDir)
	key, ok := currentVersionKey("fakefmt", "fakefmt", []string{"--version"})
	if !ok {
		t.Fatal("same-name symlink should be cacheable")
	}
	want, err := filepath.EvalSymlinks(f.exe)
	if err != nil {
		t.Fatal(err)
	}
	if key.ExecutablePath != want {
		t.Fatalf("ExecutablePath = %q, want %q", key.ExecutablePath, want)
	}
	vc := NewVersionCache(t.TempDir())
	viaPath := tools.NewBaseTool("fakefmt", "Go", "fakefmt", tools.FORMAT)
	requireVersion(t, vc, viaPath, "fakefmt v1")
	requireVersion(t, vc, viaPath, "fakefmt v1")
	requireProbes(t, f, 1)
}

func TestVersionKey_EveryFieldInvalidates(t *testing.T) {
	base := VersionKey{
		ToolName: "fakefmt", ExecutablePath: "/bin/fakefmt", ProbeFlags: []string{"--version", "-v"},
		Device: 1, Inode: 2, Size: 3, ModTimeNs: 4, ChangeTimeNs: 5,
	}
	mutations := map[string]func(*VersionKey){
		"ToolName":       func(k *VersionKey) { k.ToolName = "other" },
		"ExecutablePath": func(k *VersionKey) { k.ExecutablePath = "/usr/bin/fakefmt" },
		"ProbeFlags":     func(k *VersionKey) { k.ProbeFlags = []string{"-v", "--version"} },
		"Device":         func(k *VersionKey) { k.Device++ },
		"Inode":          func(k *VersionKey) { k.Inode++ },
		"Size":           func(k *VersionKey) { k.Size++ },
		"ModTimeNs":      func(k *VersionKey) { k.ModTimeNs++ },
		"ChangeTimeNs":   func(k *VersionKey) { k.ChangeTimeNs++ },
	}
	vc := NewVersionCache(t.TempDir())
	if err := vc.store(base, "v1"); err != nil {
		t.Fatal(err)
	}
	if v, ok := vc.lookup(base); !ok || v != "v1" {
		t.Fatalf("lookup(base) = %q, %v", v, ok)
	}
	for field, mutate := range mutations {
		key := base
		key.ProbeFlags = append([]string(nil), base.ProbeFlags...)
		mutate(&key)
		if _, ok := vc.lookup(key); ok {
			t.Errorf("%s change still hits", field)
		}
	}
}

func TestStatIdentity_ReadsStatT(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	id, ok := statIdentity(info)
	st := info.Sys().(*syscall.Stat_t)
	if !ok || id.inode != st.Ino || id.links != 1 || id.modTimeNs != info.ModTime().UnixNano() || id.changeTimeNs == 0 {
		t.Fatalf("statIdentity = %+v, %v", id, ok)
	}
}

func TestCacheManager_VersionEntriesAreNotResults(t *testing.T) {
	f := newFakeVersionTool(t, "fakefmt v1")
	base := t.TempDir()
	cm, err := NewCacheManager(base, 1<<20, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer cm.Close()

	got, err := cm.ToolVersion(f.tool)
	if err != nil || got != "fakefmt v1" {
		t.Fatalf("ToolVersion = %q, %v", got, err)
	}
	if _, err := cm.ToolVersion(f.tool); err != nil {
		t.Fatal(err)
	}
	requireProbes(t, f, 1)

	storage := cm.storage.(*FilesystemStorage)
	if keys, err := storage.List(); err != nil || len(keys) != 0 {
		t.Fatalf("List = %v, %v; version entries must not be listed", keys, err)
	}
	if size, err := storage.Size(); err != nil || size != 0 {
		t.Fatalf("Size = %d, %v; version entries must not count", size, err)
	}
	if n, err := storage.CleanupCorrupted(); err != nil || n != 0 {
		t.Fatalf("CleanupCorrupted = %d, %v", n, err)
	}

	if err := cm.InvalidateAll(); err != nil {
		t.Fatal(err)
	}
	if entries, _ := filepath.Glob(filepath.Join(base, versionsDirName, "*")); len(entries) != 0 {
		t.Fatalf("InvalidateAll left %v", entries)
	}
	if _, err := cm.ToolVersion(f.tool); err != nil {
		t.Fatal(err)
	}
	requireProbes(t, f, 2)

	cm.SetEnabled(false)
	for range 2 {
		if _, err := cm.ToolVersion(f.tool); err != nil {
			t.Fatal(err)
		}
	}
	requireProbes(t, f, 4)
}

func requireNoEntries(t *testing.T, dir string) {
	t.Helper()
	entries, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("entries = %v, want none", entries)
	}
}

func TestVersionCache_OptedOutToolProbesEveryTime(t *testing.T) {
	f := newFakeVersionTool(t, "fakefmt v1")
	f.tool.SetVersionUncacheable()
	dir := t.TempDir()
	vc := NewVersionCache(dir)
	for range 3 {
		requireVersion(t, vc, f.tool, "fakefmt v1")
	}
	requireProbes(t, f, 3)
	requireNoEntries(t, dir)
}

// swapDuringProbe replaces the executable with a different file after the
// probe ran but before GetVersion returns, once.
type swapDuringProbe struct {
	*tools.BaseTool
	swap func()
}

func (t *swapDuringProbe) GetVersion() (string, error) {
	version, err := t.BaseTool.GetVersion()
	if t.swap != nil {
		t.swap()
		t.swap = nil
	}
	return version, err
}

func TestVersionCache_ChangeDuringProbeIsNotRecorded(t *testing.T) {
	f := newFakeVersionTool(t, "fakefmt v1")
	dir := t.TempDir()
	vc := NewVersionCache(dir)
	tool := &swapDuringProbe{BaseTool: f.tool, swap: func() {
		staged := f.exe + ".new"
		f.install(staged)
		if err := os.Rename(staged, f.exe); err != nil {
			f.t.Fatal(err)
		}
	}}

	requireVersion(t, vc, tool, "fakefmt v1")
	requireProbes(t, f, 1)
	requireNoEntries(t, dir)

	f.setVersion("fakefmt v2")
	requireVersion(t, vc, tool, "fakefmt v2")
	requireProbes(t, f, 2)
}

func TestVersionCache_RejectsStaleFormatAndEmptyVersion(t *testing.T) {
	f := newFakeVersionTool(t, "fakefmt v1")
	dir := t.TempDir()
	vc := NewVersionCache(dir)
	requireVersion(t, vc, f.tool, "fakefmt v1")
	entries, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries = %v, %v", entries, err)
	}
	data, err := os.ReadFile(entries[0])
	if err != nil {
		t.Fatal(err)
	}
	var stored versionEntry
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}

	for name, edit := range map[string]func(*versionEntry){
		"other format":  func(e *versionEntry) { e.Format = "0"; e.Version = "stale" },
		"empty version": func(e *versionEntry) { e.Version = "" },
	} {
		entry := stored
		edit(&entry)
		data, err := json.Marshal(entry)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(entries[0], data, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, hit := vc.lookup(entry.Key); hit {
			t.Fatalf("%s: entry was accepted", name)
		}
	}
	requireVersion(t, vc, f.tool, "fakefmt v1")
	requireProbes(t, f, 2)
}

func TestVersionCache_InvalidUTF8VersionIsNotRecorded(t *testing.T) {
	invalid := "fakefmt \xff"
	f := newFakeVersionTool(t, invalid)
	dir := t.TempDir()
	vc := NewVersionCache(dir)
	requireVersion(t, vc, f.tool, invalid)
	requireVersion(t, vc, f.tool, invalid)
	requireProbes(t, f, 2)
	requireNoEntries(t, dir)
}

func TestCurrentVersionKey_RefusesNonRegularFile(t *testing.T) {
	fifo := filepath.Join(t.TempDir(), "fifofmt")
	if err := syscall.Mkfifo(fifo, 0o755); err != nil {
		t.Skipf("mkfifo: %v", err)
	}
	if err := os.Chmod(fifo, 0o755); err != nil { // Undo the umask.
		t.Fatal(err)
	}
	if _, err := exec.LookPath(fifo); err != nil {
		t.Fatalf("LookPath must accept the fifo for this test to reach the stat check: %v", err)
	}
	// A broken guard would open the fifo and block; fail instead of hanging.
	keyed := make(chan bool, 1)
	go func() {
		_, ok := currentVersionKey("fifofmt", fifo, []string{"--version"})
		keyed <- ok
	}()
	select {
	case ok := <-keyed:
		if ok {
			t.Fatal("a non-regular file was keyed")
		}
	case <-time.After(5 * time.Second):
		// Release the blocked reader so the goroutine can exit.
		if writer, err := os.OpenFile(fifo, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			_ = writer.Close()
		}
		t.Fatal("currentVersionKey blocked on a fifo: the regular-file check must run before the file is opened")
	}
}
