// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

package cache_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/Gizzahub/gzh-cli-quality/cache"
	"github.com/Gizzahub/gzh-cli-quality/tools"
)

// Explicit forwarding methods model an external backend implementing all eight
// required methods, without embedding Storage and inheriting future additions.
type externalStorage struct{ backend *cache.FilesystemStorage }

var _ cache.Storage = (*externalStorage)(nil)

func (s *externalStorage) Read(key string) ([]byte, error)          { return s.backend.Read(key) }
func (s *externalStorage) Write(key string, data []byte) error      { return s.backend.Write(key, data) }
func (s *externalStorage) Delete(key string) error                  { return s.backend.Delete(key) }
func (s *externalStorage) Touch(key string) error                   { return s.backend.Touch(key) }
func (s *externalStorage) AccessedAt(key string) (time.Time, error) { return s.backend.AccessedAt(key) }
func (s *externalStorage) List() ([]string, error)                  { return s.backend.List() }
func (s *externalStorage) Size() (int64, error)                     { return s.backend.Size() }
func (s *externalStorage) Close() error                             { return s.backend.Close() }

func TestExternalStoredMetadataAndSessionStatistics(t *testing.T) {
	directory := t.TempDir()
	backend, err := cache.NewFilesystemStorage(directory)
	if err != nil {
		t.Fatal(err)
	}
	storage := &externalStorage{backend: backend}
	t.Cleanup(func() {
		if err := storage.Close(); err != nil {
			t.Error(err)
		}
	})
	key := cache.CacheKey{
		FilePath: filepath.Join(directory, "source.go"), FileHash: "source-hash",
		ToolName: "external-tool", ToolVersion: "1.0",
	}
	legacy := cache.CacheMetadata{
		CreatedAt:    time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		LastAccessed: time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC), AccessCount: 41, SizeBytes: 123,
	}
	data, err := json.Marshal(cache.CachedResult{
		Version: "1.0", Key: key,
		Result: &tools.Result{Success: true}, Metadata: legacy,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Write(key.String(), data); err != nil {
		t.Fatal(err)
	}
	var last *cache.CacheManager
	for session := 0; session < 2; session++ {
		manager, err := cache.NewCacheManager(directory, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		last = manager
		t.Cleanup(func() {
			if err := manager.Close(); err != nil {
				t.Error(err)
			}
		})
		if got := manager.Stats().HitCount; got != 0 {
			t.Fatalf("new session hits = %d", got)
		}
		for hits := int64(1); hits <= 2; hits++ {
			cached, err := manager.Get(key)
			if err != nil {
				t.Fatal(err)
			}
			if cached.Metadata != legacy {
				t.Fatalf("stored metadata changed: %+v", cached.Metadata)
			}
			if got := manager.Stats().HitCount; got != hits {
				t.Fatalf("session hits = %d, want %d", got, hits)
			}
		}
		persisted, err := storage.Read(key.String())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(persisted, data) {
			t.Fatal("Get rewrote persisted entry metadata")
		}
	}
	if err := last.InvalidateAll(); err != nil {
		t.Fatal(err)
	}
	if got := last.Stats().HitCount; got != 0 {
		t.Fatalf("reset session hits = %d", got)
	}
}
