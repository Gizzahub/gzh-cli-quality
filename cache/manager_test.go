// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Gizzahub/gzh-cli-quality/tools"
)

func TestCacheManager_GetSet(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	filesDir := filepath.Join(tmpDir, "files")
	os.MkdirAll(filesDir, 0755)

	manager, err := NewCacheManager(cacheDir, 100*1024*1024, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.Close()

	// Create test file for key generation
	testFile := filepath.Join(filesDir, "test.go")
	os.WriteFile(testFile, []byte("package main"), 0644)

	tool := &mockTool{name: "gofumpt", version: "v0.7.0"}
	options := tools.ExecuteOptions{Fix: true}

	key, _ := GenerateKey(testFile, tool, options)

	// Initially cache miss
	_, err = manager.Get(key)
	if err == nil {
		t.Error("Expected cache miss on first get")
	}

	// Set result
	result := &tools.Result{
		Tool:     "gofumpt",
		Language: "Go",
		Success:  true,
		Duration: "1s",
	}

	if err := manager.Set(key, result); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Now cache hit
	cached, err := manager.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if cached.Result.Tool != "gofumpt" {
		t.Errorf("Cached result tool = %s, want gofumpt", cached.Result.Tool)
	}
}

func TestCacheManager_Invalidate(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	filesDir := filepath.Join(tmpDir, "files")
	os.MkdirAll(filesDir, 0755)

	manager, err := NewCacheManager(cacheDir, 100*1024*1024, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	testFile := filepath.Join(filesDir, "test.go")
	os.WriteFile(testFile, []byte("package main"), 0644)

	tool := &mockTool{name: "gofumpt", version: "v0.7.0"}
	key, _ := GenerateKey(testFile, tool, tools.ExecuteOptions{})

	result := &tools.Result{Success: true}

	// Set and verify
	manager.Set(key, result)
	if _, err := manager.Get(key); err != nil {
		t.Fatal("Should have cache hit")
	}

	// Invalidate
	if err := manager.Invalidate(key); err != nil {
		t.Fatalf("Invalidate failed: %v", err)
	}

	// Should be cache miss now
	if _, err := manager.Get(key); err == nil {
		t.Error("Expected cache miss after invalidation")
	}
}

func TestCacheManager_Stats(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	filesDir := filepath.Join(tmpDir, "files")
	os.MkdirAll(filesDir, 0755)

	manager, err := NewCacheManager(cacheDir, 100*1024*1024, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	testFile := filepath.Join(filesDir, "test.go")
	os.WriteFile(testFile, []byte("package main"), 0644)

	tool := &mockTool{name: "gofumpt", version: "v0.7.0"}
	key, _ := GenerateKey(testFile, tool, tools.ExecuteOptions{})

	// Initial stats
	stats := manager.Stats()
	if stats.Entries != 0 {
		t.Errorf("Initial entries = %d, want 0", stats.Entries)
	}

	// Add entry
	result := &tools.Result{Success: true}
	manager.Set(key, result)

	// Wait for async operations to complete
	time.Sleep(50 * time.Millisecond)

	// Stats should update
	stats = manager.Stats()
	if stats.Entries != 1 {
		t.Errorf("Entries = %d, want 1", stats.Entries)
	}

	// Generate cache hit
	manager.Get(key)

	// Hit count should increase
	stats = manager.Stats()
	if stats.HitCount != 1 {
		t.Errorf("HitCount = %d, want 1", stats.HitCount)
	}

	// Hit rate should be 1.0 (100%)
	if stats.HitRate != 1.0 {
		t.Errorf("HitRate = %f, want 1.0", stats.HitRate)
	}
}

func TestCacheManager_Cleanup_Age(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	filesDir := filepath.Join(tmpDir, "files")
	os.MkdirAll(filesDir, 0755)

	// Set maxAge to 1 nanosecond for testing
	manager, err := NewCacheManager(cacheDir, 100*1024*1024, 1*time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	testFile := filepath.Join(filesDir, "test.go")
	os.WriteFile(testFile, []byte("package main"), 0644)

	tool := &mockTool{name: "gofumpt", version: "v0.7.0"}
	key, _ := GenerateKey(testFile, tool, tools.ExecuteOptions{})

	result := &tools.Result{Success: true}
	manager.Set(key, result)

	// Wait for entry to expire
	time.Sleep(10 * time.Millisecond)

	// Cleanup should remove expired entry
	if err := manager.Cleanup(); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Entry should be gone
	stats := manager.Stats()
	if stats.Entries != 0 {
		t.Errorf("After cleanup, entries = %d, want 0", stats.Entries)
	}
}

func TestCacheManager_Cleanup_Size(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	filesDir := filepath.Join(tmpDir, "files")
	os.MkdirAll(filesDir, 0755)

	// Set very small maxSize for testing
	manager, err := NewCacheManager(cacheDir, 100, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	testFile := filepath.Join(filesDir, "test.go")
	os.WriteFile(testFile, []byte("package main"), 0644)

	tool := &mockTool{name: "gofumpt", version: "v0.7.0"}

	// Add multiple entries
	for i := 0; i < 5; i++ {
		options := tools.ExecuteOptions{ExtraArgs: []string{string(rune('a' + i))}}
		key, _ := GenerateKey(testFile, tool, options)
		result := &tools.Result{Success: true}
		manager.Set(key, result)
		time.Sleep(10 * time.Millisecond) // Ensure different access times
	}

	// Wait for async cleanup from Set() to complete
	time.Sleep(100 * time.Millisecond)

	// Force cleanup
	if err := manager.Cleanup(); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Should have evicted some entries
	stats := manager.Stats()
	if stats.Entries >= 5 {
		t.Errorf("Expected entries < 5 after cleanup, got %d", stats.Entries)
	}
}

func TestCacheManager_InvalidateAll(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	filesDir := filepath.Join(tmpDir, "files")
	os.MkdirAll(filesDir, 0755)

	manager, err := NewCacheManager(cacheDir, 100*1024*1024, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	testFile := filepath.Join(filesDir, "test.go")
	os.WriteFile(testFile, []byte("package main"), 0644)

	tool := &mockTool{name: "gofumpt", version: "v0.7.0"}

	// Add multiple entries
	for i := 0; i < 3; i++ {
		options := tools.ExecuteOptions{ExtraArgs: []string{string(rune('a' + i))}}
		key, _ := GenerateKey(testFile, tool, options)
		result := &tools.Result{Success: true}
		manager.Set(key, result)
	}

	// Wait for async operations to complete
	time.Sleep(50 * time.Millisecond)

	stats := manager.Stats()
	if stats.Entries != 3 {
		t.Errorf("Before invalidate, entries = %d, want 3", stats.Entries)
	}

	// Invalidate all
	if err := manager.InvalidateAll(); err != nil {
		t.Fatalf("InvalidateAll failed: %v", err)
	}

	// All entries should be gone
	stats = manager.Stats()
	if stats.Entries != 0 {
		t.Errorf("After invalidate all, entries = %d, want 0", stats.Entries)
	}

	// Counters should be reset
	if stats.HitCount != 0 || stats.MissCount != 0 {
		t.Error("Hit/miss counters should be reset")
	}
}

func TestCacheManager_OnlySuccessful(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	filesDir := filepath.Join(tmpDir, "files")
	os.MkdirAll(filesDir, 0755)

	manager, err := NewCacheManager(cacheDir, 100*1024*1024, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	testFile := filepath.Join(filesDir, "test.go")
	os.WriteFile(testFile, []byte("package main"), 0644)

	tool := &mockTool{name: "gofumpt", version: "v0.7.0"}
	key, _ := GenerateKey(testFile, tool, tools.ExecuteOptions{})

	// Try to cache failed result
	failedResult := &tools.Result{Success: false}
	if err := manager.Set(key, failedResult); err != nil {
		t.Fatal(err)
	}

	// Should not be cached
	if _, err := manager.Get(key); err == nil {
		t.Error("Failed results should not be cached")
	}
}

func TestCacheManager_Disabled(t *testing.T) {
	manager := NewDisabledCacheManager()

	if manager.Enabled() {
		t.Error("Cache should be disabled")
	}

	key := CacheKey{
		FilePath:    "/tmp/test.go",
		FileHash:    "abc123",
		ToolName:    "gofumpt",
		ToolVersion: "v0.7.0",
	}

	// Operations should not error but do nothing
	result := &tools.Result{Success: true}
	if err := manager.Set(key, result); err != nil {
		t.Error("Set on disabled cache should not error")
	}

	if _, err := manager.Get(key); err == nil {
		t.Error("Get on disabled cache should return error")
	}
}

func TestCacheManager_AccessCount(t *testing.T) {
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	filesDir := filepath.Join(tmpDir, "files")
	os.MkdirAll(filesDir, 0755)

	manager, err := NewCacheManager(cacheDir, 100*1024*1024, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	testFile := filepath.Join(filesDir, "test.go")
	os.WriteFile(testFile, []byte("package main"), 0644)

	tool := &mockTool{name: "gofumpt", version: "v0.7.0"}
	key, _ := GenerateKey(testFile, tool, tools.ExecuteOptions{})

	result := &tools.Result{Success: true}
	manager.Set(key, result)

	// Access multiple times
	for i := 0; i < 5; i++ {
		manager.Get(key)
		time.Sleep(10 * time.Millisecond) // Allow async metadata update
	}

	// Hit count should be 5
	stats := manager.Stats()
	if stats.HitCount != 5 {
		t.Errorf("HitCount = %d, want 5", stats.HitCount)
	}
}
