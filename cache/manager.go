// Copyright (c) 2025 Archmagece
// SPDX-License-Identifier: MIT

package cache

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Gizzahub/gzh-cli-quality/tools"
)

// CacheManager implements the Manager interface.
type CacheManager struct {
	storage    Storage
	enabled    bool
	maxSize    int64
	maxAge     time.Duration
	hitCount   atomic.Int64
	missCount  atomic.Int64
	mu         sync.RWMutex
}

// NewCacheManager creates a new cache manager.
func NewCacheManager(basePath string, maxSize int64, maxAge time.Duration) (*CacheManager, error) {
	storage, err := NewFilesystemStorage(basePath)
	if err != nil {
		return nil, err
	}

	return &CacheManager{
		storage: storage,
		enabled: true,
		maxSize: maxSize,
		maxAge:  maxAge,
	}, nil
}

// NewDisabledCacheManager creates a cache manager with caching disabled.
func NewDisabledCacheManager() *CacheManager {
	return &CacheManager{
		enabled: false,
	}
}

// Get retrieves a cached result.
func (cm *CacheManager) Get(key CacheKey) (*CachedResult, error) {
	if !cm.enabled {
		return nil, fmt.Errorf("cache disabled")
	}

	// Validate key
	if err := ValidateKey(key); err != nil {
		return nil, fmt.Errorf("invalid cache key: %w", err)
	}

	// Read from storage
	data, err := cm.storage.Read(key.String())
	if err != nil {
		cm.missCount.Add(1)
		return nil, err
	}

	// Deserialize
	var cached CachedResult
	if err := json.Unmarshal(data, &cached); err != nil {
		cm.missCount.Add(1)
		return nil, fmt.Errorf("failed to deserialize cached result: %w", err)
	}

	// Check if expired
	if cm.maxAge > 0 && time.Since(cached.Metadata.CreatedAt) > cm.maxAge {
		cm.missCount.Add(1)
		// Delete expired entry
		_ = cm.Invalidate(key)
		return nil, fmt.Errorf("cache entry expired")
	}

	// Update access metadata
	cached.Metadata.LastAccessed = time.Now()
	cached.Metadata.AccessCount++

	// Write updated metadata back (synchronously to avoid race conditions in tests)
	updatedData, _ := json.MarshalIndent(cached, "", "  ")
	_ = cm.storage.Write(key.String(), updatedData)

	cm.hitCount.Add(1)
	return &cached, nil
}

// Set stores a result in cache.
func (cm *CacheManager) Set(key CacheKey, result *tools.Result) error {
	if !cm.enabled {
		return nil // Silent no-op when disabled
	}

	// Validate key
	if err := ValidateKey(key); err != nil {
		return fmt.Errorf("invalid cache key: %w", err)
	}

	// Only cache successful results
	if !result.Success {
		return nil
	}

	// Create cached result
	cached := CachedResult{
		Version: "1.0",
		Key:     key,
		Result:  result,
		Metadata: CacheMetadata{
			CreatedAt:    time.Now(),
			LastAccessed: time.Now(),
			AccessCount:  0,
		},
	}

	// Serialize
	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize result: %w", err)
	}

	// Update size metadata
	cached.Metadata.SizeBytes = int64(len(data))

	// Re-serialize with updated metadata
	data, _ = json.MarshalIndent(cached, "", "  ")

	// Write to storage
	if err := cm.storage.Write(key.String(), data); err != nil {
		return fmt.Errorf("failed to write to cache: %w", err)
	}

	// Trigger cleanup if cache is getting large
	go func() {
		size, _ := cm.storage.Size()
		if size > int64(float64(cm.maxSize)*0.9) { // 90% threshold
			_ = cm.Cleanup()
		}
	}()

	return nil
}

// Invalidate removes a cache entry.
func (cm *CacheManager) Invalidate(key CacheKey) error {
	if !cm.enabled {
		return nil
	}

	return cm.storage.Delete(key.String())
}

// InvalidateAll removes all cache entries.
func (cm *CacheManager) InvalidateAll() error {
	if !cm.enabled {
		return nil
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	keys, err := cm.storage.List()
	if err != nil {
		return err
	}

	for _, key := range keys {
		_ = cm.storage.Delete(key)
	}

	// Reset counters
	cm.hitCount.Store(0)
	cm.missCount.Store(0)

	return nil
}

// Stats returns cache statistics.
func (cm *CacheManager) Stats() CacheStats {
	if !cm.enabled {
		return CacheStats{}
	}

	cm.mu.RLock()
	defer cm.mu.RUnlock()

	keys, _ := cm.storage.List()
	size, _ := cm.storage.Size()

	hits := cm.hitCount.Load()
	misses := cm.missCount.Load()
	total := hits + misses

	hitRate := 0.0
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	stats := CacheStats{
		Entries:   int64(len(keys)),
		SizeBytes: size,
		HitCount:  hits,
		MissCount: misses,
		HitRate:   hitRate,
	}

	// Find oldest and newest entries
	var oldest, newest time.Time
	for _, key := range keys {
		data, err := cm.storage.Read(key)
		if err != nil {
			continue
		}

		var cached CachedResult
		if err := json.Unmarshal(data, &cached); err != nil {
			continue
		}

		if oldest.IsZero() || cached.Metadata.CreatedAt.Before(oldest) {
			oldest = cached.Metadata.CreatedAt
		}

		if newest.IsZero() || cached.Metadata.CreatedAt.After(newest) {
			newest = cached.Metadata.CreatedAt
		}
	}

	stats.OldestEntry = oldest
	stats.NewestEntry = newest

	return stats
}

// Cleanup performs cache cleanup based on age and size limits.
func (cm *CacheManager) Cleanup() error {
	if !cm.enabled {
		return nil
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 1. Delete entries older than maxAge
	if cm.maxAge > 0 {
		cutoffTime := time.Now().Add(-cm.maxAge)
		keys, _ := cm.storage.List()

		for _, key := range keys {
			data, err := cm.storage.Read(key)
			if err != nil {
				continue
			}

			var cached CachedResult
			if err := json.Unmarshal(data, &cached); err != nil {
				// Corrupted entry: delete it
				_ = cm.storage.Delete(key)
				continue
			}

			if cached.Metadata.CreatedAt.Before(cutoffTime) {
				_ = cm.storage.Delete(key)
			}
		}
	}

	// 2. If still over size limit, delete least recently accessed
	size, _ := cm.storage.Size()
	if cm.maxSize > 0 && size > cm.maxSize {
		// Get all entries with metadata
		type entry struct {
			key          string
			lastAccessed time.Time
		}

		var entries []entry
		keys, _ := cm.storage.List()

		for _, key := range keys {
			data, err := cm.storage.Read(key)
			if err != nil {
				continue
			}

			var cached CachedResult
			if err := json.Unmarshal(data, &cached); err != nil {
				_ = cm.storage.Delete(key)
				continue
			}

			entries = append(entries, entry{
				key:          key,
				lastAccessed: cached.Metadata.LastAccessed,
			})
		}

		// Sort by last accessed (oldest first) - O(n log n)
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].lastAccessed.Before(entries[j].lastAccessed)
		})

		// Delete oldest entries until under limit
		for _, e := range entries {
			if size <= cm.maxSize {
				break
			}

			_ = cm.storage.Delete(e.key)
			size, _ = cm.storage.Size()
		}
	}

	return nil
}

// Close closes the cache manager.
func (cm *CacheManager) Close() error {
	if cm.storage != nil {
		return cm.storage.Close()
	}
	return nil
}

// Enabled returns whether caching is enabled.
func (cm *CacheManager) Enabled() bool {
	return cm.enabled
}

// SetEnabled enables or disables caching.
func (cm *CacheManager) SetEnabled(enabled bool) {
	cm.enabled = enabled
}
