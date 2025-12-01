// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilesystemStorage_ReadWrite(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewFilesystemStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	key := "test-key"
	data := []byte("test data")

	// Write
	if err := storage.Write(key, data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read
	readData, err := storage.Read(key)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if string(readData) != string(data) {
		t.Errorf("Read data = %s, want %s", readData, data)
	}
}

func TestFilesystemStorage_Delete(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewFilesystemStorage(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	key := "test-key"
	data := []byte("test data")

	// Write then delete
	storage.Write(key, data)

	if err := storage.Delete(key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Try to read deleted key
	_, err = storage.Read(key)
	if err == nil {
		t.Error("Expected error reading deleted key")
	}

	// Delete non-existent key should not error
	if err := storage.Delete("nonexistent"); err != nil {
		t.Errorf("Delete non-existent key should not error: %v", err)
	}
}

func TestFilesystemStorage_List(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewFilesystemStorage(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	// Write multiple keys
	keys := []string{"key1", "key2", "key3"}
	for _, key := range keys {
		if err := storage.Write(key, []byte("data")); err != nil {
			t.Fatal(err)
		}
	}

	// List
	listedKeys, err := storage.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(listedKeys) != len(keys) {
		t.Errorf("List returned %d keys, want %d", len(listedKeys), len(keys))
	}
}

func TestFilesystemStorage_Size(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewFilesystemStorage(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	// Initially empty
	size, err := storage.Size()
	if err != nil {
		t.Fatalf("Size failed: %v", err)
	}

	if size != 0 {
		t.Errorf("Initial size = %d, want 0", size)
	}

	// Write data
	data := []byte("test data that is exactly 30!!")
	storage.Write("key1", data)

	// Size should increase
	size, err = storage.Size()
	if err != nil {
		t.Fatal(err)
	}

	if size == 0 {
		t.Error("Size should be > 0 after writing")
	}
}

func TestFilesystemStorage_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewFilesystemStorage(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	key := "test-key"

	// Write initial data
	storage.Write(key, []byte("data1"))

	// Overwrite (should be atomic)
	storage.Write(key, []byte("data2"))

	// Read should get latest data
	data, _ := storage.Read(key)
	if string(data) != "data2" {
		t.Errorf("Read data = %s, want data2", data)
	}

	// No .tmp files should remain
	matches, _ := filepath.Glob(filepath.Join(tmpDir, "**/*.tmp"))
	if len(matches) > 0 {
		t.Error("Temporary files should be cleaned up")
	}
}

func TestFilesystemStorage_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewFilesystemStorage(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	// Write concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			key := filepath.Join("key", string(rune('0'+n)))
			data := []byte("data")
			storage.Write(key, data)
			storage.Read(key)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// All writes should succeed
	keys, _ := storage.List()
	if len(keys) < 10 {
		t.Errorf("Expected at least 10 keys, got %d", len(keys))
	}
}

func TestFilesystemStorage_CleanupCorrupted(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewFilesystemStorage(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	// Write valid entry
	storage.Write("valid", []byte(`{"key": "value"}`))

	// Write corrupted entry (empty file)
	corruptedPath := filepath.Join(tmpDir, "results", "tool", "00", "corrupted.json")
	os.MkdirAll(filepath.Dir(corruptedPath), 0755)
	os.WriteFile(corruptedPath, []byte(""), 0644)

	// Write invalid JSON
	invalidPath := filepath.Join(tmpDir, "results", "tool", "00", "invalid.json")
	os.WriteFile(invalidPath, []byte("not json"), 0644)

	// Cleanup
	count, err := storage.CleanupCorrupted()
	if err != nil {
		t.Fatalf("CleanupCorrupted failed: %v", err)
	}

	if count < 2 {
		t.Errorf("Expected at least 2 corrupted entries cleaned, got %d", count)
	}

	// Valid entry should still exist
	_, err = storage.Read("valid")
	if err != nil {
		t.Error("Valid entry should still exist after cleanup")
	}
}

func TestFilesystemStorage_CreateBasePath(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "nested", "cache", "dir")

	// Should create directory
	storage, err := NewFilesystemStorage(nonExistentPath)
	if err != nil {
		t.Fatalf("Failed to create storage with non-existent path: %v", err)
	}
	defer storage.Close()

	// Directory should exist
	if _, err := os.Stat(nonExistentPath); os.IsNotExist(err) {
		t.Error("Base path should be created")
	}
}
