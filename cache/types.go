// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

// Package cache provides a file-based caching system for quality tool results.
package cache

import (
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
// Format: {tool}-{version}-{file_hash[:8]}-{config_hash[:8]}-{options_hash[:8]}
func (ck CacheKey) String() string {
	fileHashShort := ck.FileHash
	if len(fileHashShort) > 8 {
		fileHashShort = fileHashShort[:8]
	}

	configHashShort := ck.ConfigHash
	if len(configHashShort) > 8 {
		configHashShort = configHashShort[:8]
	}

	optionsHashShort := ck.OptionsHash
	if len(optionsHashShort) > 8 {
		optionsHashShort = optionsHashShort[:8]
	}

	return ck.ToolName + "-" + ck.ToolVersion + "-" + fileHashShort + "-" + configHashShort + "-" + optionsHashShort
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

// CacheMetadata contains metadata about a cache entry.
type CacheMetadata struct {
	// CreatedAt is when the entry was created
	CreatedAt time.Time `json:"created_at"`

	// LastAccessed is when the entry was last accessed
	LastAccessed time.Time `json:"last_accessed"`

	// AccessCount is the number of times accessed
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

	// HitCount is the number of cache hits
	HitCount int64

	// MissCount is the number of cache misses
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

	// List returns all keys in storage
	List() ([]string, error)

	// Size returns the total size of storage in bytes
	Size() (int64, error)

	// Close closes the storage backend
	Close() error
}

// Manager is the interface for cache management operations.
type Manager interface {
	// Get retrieves a cached result
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
