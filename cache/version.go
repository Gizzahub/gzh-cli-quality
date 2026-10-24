// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package cache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Gizzahub/gzh-cli-quality/tools"
)

// versionsDirName is the cache subdirectory that holds tool version probes.
// Result storage skips it, so version entries are not result entries.
const versionsDirName = "versions"

// versionEntryFormat is bumped whenever VersionKey or the entry layout changes.
const versionEntryFormat = "1"

// VersionKey identifies one version probe of one executable file. A recorded
// version is reused only when every field equals the current probe.
type VersionKey struct {
	// ToolName is the quality tool whose GetVersion produced the version.
	ToolName string `json:"tool_name"`

	// ExecutablePath is the absolute path of the file PATH resolves the tool
	// to, after following symlinks.
	ExecutablePath string `json:"executable_path"`

	// ProbeFlags are the version flags, in the order GetVersion tries them.
	ProbeFlags []string `json:"probe_flags"`

	// Device and Inode identify the file, so replacing it by rename misses.
	Device uint64 `json:"device"`
	Inode  uint64 `json:"inode"`

	// Size and ModTimeNs catch in-place rewrites.
	Size      int64 `json:"size"`
	ModTimeNs int64 `json:"mod_time_ns"`

	// ChangeTimeNs is the inode change time. Users cannot set it back, so a
	// copy that preserves mtime (cp -p) or a restored mtime still misses.
	ChangeTimeNs int64 `json:"change_time_ns"`
}

func (k VersionKey) equal(other VersionKey) bool {
	return k.ToolName == other.ToolName &&
		k.ExecutablePath == other.ExecutablePath &&
		slices.Equal(k.ProbeFlags, other.ProbeFlags) &&
		k.Device == other.Device &&
		k.Inode == other.Inode &&
		k.Size == other.Size &&
		k.ModTimeNs == other.ModTimeNs &&
		k.ChangeTimeNs == other.ChangeTimeNs
}

// slot names the file holding the latest probe for one tool, executable path
// and flag list; a changed file overwrites its slot instead of adding one.
func (k VersionKey) slot() string {
	data, _ := json.Marshal(struct {
		ToolName       string
		ExecutablePath string
		ProbeFlags     []string
	}{k.ToolName, k.ExecutablePath, k.ProbeFlags}) // Only strings: cannot fail.
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

type versionEntry struct {
	Format  string     `json:"format"`
	Key     VersionKey `json:"key"`
	Version string     `json:"version"`
}

// VersionCache records tool versions against the executable file's identity,
// so a warm run skips the version subprocess without trusting a stale version.
type VersionCache struct {
	dir string
}

// NewVersionCache returns a version cache that stores entries under dir.
func NewVersionCache(dir string) *VersionCache {
	return &VersionCache{dir: dir}
}

// ToolVersion returns tool.GetVersion(), reusing a version recorded for the
// same executable file and probe. It probes whenever the tool opts out
// (VersionCacheable false) or the file cannot be identified on this
// platform, and returns probe errors unchanged and unrecorded. A missing,
// unreadable or corrupt entry is probed and rewritten.
func (vc *VersionCache) ToolVersion(tool tools.QualityTool) (string, error) {
	prober, ok := tool.(tools.VersionProber)
	if !ok || !prober.VersionCacheable() {
		return tool.GetVersion()
	}
	executable, flags := prober.VersionProbe()
	key, ok := currentVersionKey(tool.Name(), executable, flags)
	if !ok {
		return tool.GetVersion()
	}
	if version, hit := vc.lookup(key); hit {
		return version, nil
	}

	version, err := tool.GetVersion()
	if err != nil || version == "" || version == "unknown" {
		return version, err
	}
	// Record only if the file did not change while it was probed; otherwise
	// the version may belong to either file.
	if after, ok := currentVersionKey(tool.Name(), executable, flags); ok && after.equal(key) {
		_ = vc.store(key, version) // Best effort: the next run probes again.
	}
	return version, nil
}

// Clear removes every recorded version.
func (vc *VersionCache) Clear() error {
	entries, err := os.ReadDir(vc.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to list version cache: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.Type().IsRegular() || (!strings.HasSuffix(name, ".json") && !strings.HasSuffix(name, ".tmp")) {
			continue
		}
		if err := os.Remove(filepath.Join(vc.dir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to remove version cache entry: %w", err)
		}
	}
	return nil
}

func (vc *VersionCache) path(key VersionKey) string {
	return filepath.Join(vc.dir, key.slot()+".json")
}

func (vc *VersionCache) lookup(key VersionKey) (string, bool) {
	data, err := os.ReadFile(vc.path(key))
	if err != nil {
		return "", false
	}
	var entry versionEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return "", false
	}
	if entry.Format != versionEntryFormat || entry.Version == "" || !entry.Key.equal(key) {
		return "", false
	}
	return entry.Version, true
}

// store writes the entry atomically: a temp file in the same directory, then
// a rename, so concurrent runs never read a partial entry.
func (vc *VersionCache) store(key VersionKey, version string) error {
	if !utf8.ValidString(version) {
		// JSON would replace the invalid bytes, so a hit would return a
		// version that differs from the probed one.
		return errors.New("version is not valid UTF-8")
	}
	data, err := json.Marshal(versionEntry{Format: versionEntryFormat, Key: key, Version: version})
	if err != nil {
		return fmt.Errorf("failed to serialize version entry: %w", err)
	}
	if err := os.MkdirAll(vc.dir, 0o755); err != nil {
		return fmt.Errorf("failed to create version cache directory: %w", err)
	}
	temp, err := os.CreateTemp(vc.dir, "*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := temp.Name()
	_, writeErr := temp.Write(data)
	closeErr := temp.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := os.Rename(tempPath, vc.path(key)); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}
	return nil
}

// currentVersionKey identifies the file exec.Command would run for
// executable. It reports false, meaning "always probe", when the file cannot
// stand for the version by itself:
//   - stat identity (device, inode, ctime) is unavailable on this platform;
//   - the path resolves through a symlink to a different name, as version
//     manager shims do (mise, volta, update-alternatives);
//   - the file has several hard links, as rustup proxies do;
//   - the file is a "#!" script, as asdf/pyenv shims and Python console
//     entry points are, whose version lives in files the key does not cover.
func currentVersionKey(toolName, executable string, flags []string) (VersionKey, bool) {
	found, err := exec.LookPath(executable)
	if err != nil {
		return VersionKey{}, false
	}
	abs, err := filepath.Abs(found)
	if err != nil {
		return VersionKey{}, false
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil || filepath.Base(resolved) != filepath.Base(abs) {
		return VersionKey{}, false
	}
	info, err := os.Stat(resolved)
	// The regular-file check must precede isScript: opening a FIFO blocks
	// until a writer appears.
	if err != nil || !info.Mode().IsRegular() {
		return VersionKey{}, false
	}
	id, ok := statIdentity(info)
	if !ok || id.links != 1 || isScript(resolved) {
		return VersionKey{}, false
	}
	return VersionKey{
		ToolName:       toolName,
		ExecutablePath: resolved,
		ProbeFlags:     slices.Clone(flags),
		Device:         id.device,
		Inode:          id.inode,
		Size:           info.Size(),
		ModTimeNs:      id.modTimeNs,
		ChangeTimeNs:   id.changeTimeNs,
	}, true
}

// fileIdentity holds the stat fields that only syscall.Stat_t provides.
type fileIdentity struct {
	device       uint64
	inode        uint64
	links        uint64
	modTimeNs    int64
	changeTimeNs int64
}

func isScript(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return true // Unreadable: do not vouch for it.
	}
	defer file.Close()
	header := make([]byte, 2)
	if _, err := io.ReadFull(file, header); err != nil {
		return !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF)
	}
	return bytes.Equal(header, []byte("#!"))
}
