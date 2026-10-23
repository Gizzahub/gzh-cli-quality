package cache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Gizzahub/gzh-cli-quality/tools"
	"github.com/stretchr/testify/require"
)

func TestCacheIdentifierIncludesCompleteIdentity(t *testing.T) {
	base := CacheKey{FilePath: "/a.go", FileHash: "12345678a", ToolName: "tool", ToolVersion: "1", ConfigHash: "12345678a", OptionsHash: "12345678a"}
	for _, change := range []func(*CacheKey){
		func(k *CacheKey) { k.FilePath = "/b.go" },
		func(k *CacheKey) { k.FileHash = "12345678b" },
		func(k *CacheKey) { k.ConfigHash = "12345678b" },
		func(k *CacheKey) { k.OptionsHash = "12345678b" },
		func(k *CacheKey) { k.ToolName = "another" },
		func(k *CacheKey) { k.ToolVersion = "2" },
	} {
		changed := base
		change(&changed)
		require.NotEqual(t, base.String(), changed.String())
	}
}

func TestCacheNeverReturnsAnotherFilesDiagnostics(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewCacheManager(t.TempDir(), 1<<20, time.Hour)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, manager.Close()) })
	tool := &mockTool{name: "tool", version: "1"}
	var keys []CacheKey
	for _, name := range []string{"a.go", "b.go"} {
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, []byte("package same\n"), 0o600))
		key, err := GenerateKey(path, tool, tools.ExecuteOptions{})
		require.NoError(t, err)
		keys = append(keys, key)
	}
	require.NoError(t, manager.Set(keys[0], &tools.Result{Success: true, Issues: []tools.Issue{{File: keys[0].FilePath, Message: "a warning"}}}))
	_, err = manager.Get(keys[1])
	require.Error(t, err, "same content in another path must miss")
	require.NoError(t, manager.Set(keys[1], &tools.Result{Success: true, Issues: []tools.Issue{{File: keys[1].FilePath, Message: "b warning"}}}))
	for i, key := range keys {
		cached, err := manager.Get(key)
		require.NoError(t, err)
		require.Equal(t, keys[i].FilePath, cached.Result.Issues[0].File)
	}
	// Legacy filenames are disposable cache entries: no collision-prone fallback
	// reads, and no destructive migration or deletion of the old data.
	oldName := "tool-1-" + keys[0].FileHash[:8] + "--" + keys[0].OptionsHash[:8]
	data, err := json.Marshal(CachedResult{Key: keys[0], Result: &tools.Result{Success: true}})
	require.NoError(t, err)
	require.NoError(t, manager.storage.Write(oldName, data))
	require.NoError(t, manager.Invalidate(keys[0]))
	_, err = manager.Get(keys[0])
	require.Error(t, err)
	_, err = manager.storage.Read(oldName)
	require.NoError(t, err, "migration must leave legacy entries untouched")
}

func TestCacheOptionsPreserveArgumentAndEnvironmentBoundaries(t *testing.T) {
	for _, pair := range [][2]tools.ExecuteOptions{
		{{ExtraArgs: []string{"--select", "A", "--select", "B"}}, {ExtraArgs: []string{"--select", "B", "--select", "A"}}},
		{{ExtraArgs: []string{"a,b", "c"}}, {ExtraArgs: []string{"a", "b,c"}}},
		{{Env: map[string]string{"A": "xenv:B=y"}}, {Env: map[string]string{"A": "x", "B": "y"}}},
		{{ProjectRoot: "/first"}, {ProjectRoot: "/second"}},
		{{ConfigFile: "first.cfg"}, {ConfigFile: "second.cfg"}},
	} {
		require.NotEqual(t, hashOptions(pair[0]), hashOptions(pair[1]))
	}
	require.Equal(t, hashOptions(tools.ExecuteOptions{Env: map[string]string{"A": "x", "B": "y"}}), hashOptions(tools.ExecuteOptions{Env: map[string]string{"B": "y", "A": "x"}}))
}

func TestCacheKeyTracksExplicitConfigAndContent(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(t.TempDir())
	tool := &mockTool{name: "tool", version: "1"}
	file := filepath.Join(dir, "a.go")
	require.NoError(t, os.WriteFile(file, []byte("package first\n"), 0o600))
	config := filepath.Join(dir, "custom.cfg")
	require.NoError(t, os.WriteFile(config, []byte("first"), 0o600))
	opts := tools.ExecuteOptions{ProjectRoot: dir, ConfigFile: "custom.cfg"}
	before, err := GenerateKey("a.go", tool, opts)
	require.NoError(t, err)
	require.Equal(t, file, before.FilePath)
	require.NoError(t, os.WriteFile(config, []byte("second"), 0o600))
	after, err := GenerateKey("a.go", tool, opts)
	require.NoError(t, err)
	require.NotEqual(t, before.ConfigHash, after.ConfigHash)
	require.NotEqual(t, before.String(), after.String())
	require.NoError(t, os.WriteFile(file, []byte("package second\n"), 0o600))
	changed, err := GenerateKey("a.go", tool, opts)
	require.NoError(t, err)
	require.NotEqual(t, after.FileHash, changed.FileHash)
	opts.ConfigFile = "missing.cfg"
	_, err = GenerateKey("a.go", tool, opts)
	require.ErrorContains(t, err, "missing.cfg")
}

type unavailableVersionTool struct {
	*mockTool
	err error
}

func (t unavailableVersionTool) GetVersion() (string, error) { return t.version, t.err }

func TestCacheKeyRequiresKnownVersion(t *testing.T) {
	file := filepath.Join(t.TempDir(), "a.go")
	require.NoError(t, os.WriteFile(file, []byte("package a"), 0o600))
	cause := errors.New("version lookup failed")
	_, err := GenerateKey(file, unavailableVersionTool{mockTool: &mockTool{name: "tool"}, err: cause}, tools.ExecuteOptions{})
	require.ErrorIs(t, err, cause)
	for _, version := range []string{"", "unknown"} {
		_, err := GenerateKey(file, &mockTool{name: "tool", version: version}, tools.ExecuteOptions{})
		require.ErrorContains(t, err, "version is unknown")
	}
}

func TestInvalidUTF8InputsCannotShareACacheEntry(t *testing.T) {
	file := filepath.Join(t.TempDir(), "a.go")
	require.NoError(t, os.WriteFile(file, []byte("package a"), 0o600))
	tool := &mockTool{name: "tool", version: "1"}
	first, second := string([]byte{0xff}), string([]byte{0xfe})
	// JSON normalizes both to the replacement character. Neither byte sequence
	// may reach storage as a successful cache identity.
	a, err := json.Marshal(first)
	require.NoError(t, err)
	b, err := json.Marshal(second)
	require.NoError(t, err)
	require.Equal(t, a, b)
	for _, raw := range []string{first, second} {
		for _, options := range []tools.ExecuteOptions{
			{ExtraArgs: []string{raw}},
			{Env: map[string]string{"MODE": raw}},
			{Env: map[string]string{raw: "value"}},
			{ProjectRoot: raw},
			{ConfigFile: raw},
		} {
			_, err := GenerateKey(file, tool, options)
			require.ErrorContains(t, err, "invalid UTF-8")
		}
		key := CacheKey{FilePath: "bad-" + raw, FileHash: "hash", ToolName: "tool", ToolVersion: "1"}
		require.ErrorContains(t, ValidateKey(key), "invalid UTF-8")
		manager, err := NewCacheManager(t.TempDir(), 1<<20, time.Hour)
		require.NoError(t, err)
		require.ErrorContains(t, manager.Set(key, &tools.Result{Success: true}), "invalid UTF-8")
		require.NoError(t, manager.Close())
	}
}
