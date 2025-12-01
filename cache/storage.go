// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FilesystemStorage implements Storage interface using the filesystem.
type FilesystemStorage struct {
	basePath string
	mu       sync.RWMutex
}

// NewFilesystemStorage creates a new filesystem storage backend.
func NewFilesystemStorage(basePath string) (*FilesystemStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &FilesystemStorage{
		basePath: basePath,
	}, nil
}

// Read reads data from storage.
func (fs *FilesystemStorage) Read(key string) ([]byte, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	path := fs.keyToPath(key)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("cache miss: %s not found", key)
		}
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	return data, nil
}

// Write writes data to storage using atomic write.
func (fs *FilesystemStorage) Write(key string, data []byte) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.keyToPath(key)

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Atomic write: write to temp file then rename
	tempPath := path + ".tmp"

	// Write to temp file
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		// Clean up temp file on failure
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Delete deletes data from storage.
func (fs *FilesystemStorage) Delete(key string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.keyToPath(key)

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			// Already deleted, not an error
			return nil
		}
		return fmt.Errorf("failed to delete cache file: %w", err)
	}

	// Try to remove empty parent directories
	dir := filepath.Dir(path)
	_ = os.Remove(dir) // Ignore errors (directory may not be empty)

	return nil
}

// List returns all keys in storage.
func (fs *FilesystemStorage) List() ([]string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var keys []string

	// Walk the cache directory
	err := filepath.Walk(fs.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip temp files
		if filepath.Ext(path) == ".tmp" {
			return nil
		}

		// Convert path back to key
		relPath, err := filepath.Rel(fs.basePath, path)
		if err != nil {
			return err
		}

		keys = append(keys, relPath)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list cache entries: %w", err)
	}

	return keys, nil
}

// Size returns the total size of storage in bytes.
func (fs *FilesystemStorage) Size() (int64, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var totalSize int64

	err := filepath.Walk(fs.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) != ".tmp" {
			totalSize += info.Size()
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to calculate cache size: %w", err)
	}

	return totalSize, nil
}

// Close closes the storage backend.
func (fs *FilesystemStorage) Close() error {
	// Filesystem storage doesn't need explicit closing
	return nil
}

// keyToPath converts a cache key to a filesystem path.
// Format: basePath/results/{tool}/{key[:2]}/{key}.json
// The first 2 chars of key are used for sharding to avoid too many files in one directory.
func (fs *FilesystemStorage) keyToPath(key string) string {
	// Extract tool name from key (format: tool-version-...)
	// Split by dash and take first part
	tool := "unknown"
	if idx := filepath.ToSlash(key); idx != "" {
		parts := splitByDash(key)
		if len(parts) > 0 {
			tool = parts[0]
		}
	}

	// Use first 2 characters for sharding
	shard := "00"
	if len(key) >= 2 {
		shard = key[:2]
	}

	return filepath.Join(fs.basePath, "results", tool, shard, key+".json")
}

// splitByDash splits a string by dash character.
func splitByDash(s string) []string {
	var parts []string
	var current string

	for _, ch := range s {
		if ch == '-' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// CleanupCorrupted removes corrupted cache entries.
// Returns the number of entries cleaned up.
func (fs *FilesystemStorage) CleanupCorrupted() (int, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	count := 0

	err := filepath.Walk(fs.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and temp files
		if info.IsDir() || filepath.Ext(path) == ".tmp" {
			return nil
		}

		// Try to read file
		data, err := os.ReadFile(path)
		if err != nil {
			// Can't read: corrupted
			_ = os.Remove(path)
			count++
			return nil
		}

		// Check if it's valid JSON (basic validation)
		if len(data) == 0 || (data[0] != '{' && data[0] != '[') {
			// Invalid JSON: corrupted
			_ = os.Remove(path)
			count++
			return nil
		}

		return nil
	})

	if err != nil {
		return count, fmt.Errorf("cleanup failed: %w", err)
	}

	return count, nil
}
