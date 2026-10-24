// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

// Package cache provides a file-based caching system for quality tool results.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/Gizzahub/gzh-cli-quality/tools"
)

// CacheKey uniquely identifies a cached result.
// A cache entry is valid only if ALL components match.
type CacheKey struct {
	// FilePath is the absolute path to the file
	FilePath string

	// FileHash is SHA256 hash of file content
	FileHash string

	// ToolName is the name of the tool (e.g., "gofumpt")
	ToolName string

	// ToolVersion is the version of the tool
	ToolVersion string

	// ConfigHash is SHA256 hash of configuration file(s)
	ConfigHash string

	// OptionsHash is SHA256 hash of execution options
	OptionsHash string
}

// String returns a string representation of the cache key.
// The storage name hashes the full identity, including the source path, without
// embedding filesystem-unsafe tool version strings or truncating content hashes.
func (ck CacheKey) String() string {
	data, _ := json.Marshal(ck) // CacheKey contains only strings.
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// CachedResult represents a cached tool execution result.
type CachedResult struct {
	// Version is the cache format version
	Version string `json:"version"`

	// Key is the cache key
	Key CacheKey `json:"key"`

	// Result is the tool execution result
	Result *tools.Result `json:"result"`

	// Metadata contains cache metadata
	Metadata CacheMetadata `json:"metadata"`
}

// CacheMetadata contains the entry's stored metadata snapshot. Get returns these
// values unchanged, including legacy values; hits do not persist updated metadata.
type CacheMetadata struct {
	// CreatedAt is when the entry was created
	CreatedAt time.Time `json:"created_at"`

	// LastAccessed is the stored access timestamp (initialized by Set).
	// Get does not refresh it; Storage.AccessedAt tracks writes and successful touches.
	LastAccessed time.Time `json:"last_accessed"`

	// AccessCount is kept for format compatibility and is no longer
	// incremented on hits; CacheStats.HitCount counts this manager's session hits.
	AccessCount int64 `json:"access_count"`

	// SizeBytes is the size of the cached result in bytes
	SizeBytes int64 `json:"size_bytes"`
}

// CacheStats contains cache statistics.
type CacheStats struct {
	// Entries is the total number of cache entries
	Entries int64

	// SizeBytes is the total size of the cache in bytes
	SizeBytes int64

	// HitCount counts successful Gets in this manager instance, not persisted
	// per-entry accesses. InvalidateAll resets the session counters.
	HitCount int64

	// MissCount counts cache misses in this manager instance and is not persisted.
	// InvalidateAll resets the session counters.
	MissCount int64

	// HitRate is the cache hit rate (0.0 to 1.0)
	HitRate float64

	// OldestEntry is the timestamp of the oldest entry
	OldestEntry time.Time

	// NewestEntry is the timestamp of the newest entry
	NewestEntry time.Time
}

// Storage is the interface for cache storage backends.
type Storage interface {
	// Read reads data from storage
	Read(key string) ([]byte, error)

	// Write writes data to storage
	Write(key string, data []byte) error

	// Delete deletes data from storage
	Delete(key string) error

	// Touch marks an entry as accessed now without rewriting its data.
	// Manager.Get attempts Touch on hits but does not fail a hit if Touch fails.
	Touch(key string) error

	// AccessedAt returns when an entry was last written or touched
	AccessedAt(key string) (time.Time, error)

	// List returns all keys in storage
	List() ([]string, error)

	// Size returns the total size of storage in bytes
	Size() (int64, error)

	// Close closes the storage backend
	Close() error
}

// Manager is the interface for cache management operations.
type Manager interface {
	// Get retrieves a cached result with its stored metadata snapshot unchanged.
	// Successful hits count in this manager's session and attempt a best-effort Touch.
	Get(key CacheKey) (*CachedResult, error)

	// Set stores a result in cache
	Set(key CacheKey, result *tools.Result) error

	// Invalidate removes a cache entry
	Invalidate(key CacheKey) error

	// InvalidateAll removes all cache entries
	InvalidateAll() error

	// Stats returns cache statistics
	Stats() CacheStats

	// Cleanup performs cache cleanup (age, size limits)
	Cleanup() error

	// Close closes the cache manager
	Close() error

	// Enabled returns whether caching is enabled
	Enabled() bool
}
