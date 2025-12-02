# Caching System Design

Design document for implementing a file-based caching system to improve gz-quality performance by avoiding redundant tool executions.

## Table of Contents

- [Goals and Motivation](#goals-and-motivation)
- [Requirements](#requirements)
- [Architecture](#architecture)
- [Cache Key Design](#cache-key-design)
- [Cache Storage](#cache-storage)
- [Cache Invalidation](#cache-invalidation)
- [Integration Points](#integration-points)
- [Performance Impact](#performance-impact)
- [Implementation Plan](#implementation-plan)
- [Testing Strategy](#testing-strategy)
- [Risks and Mitigations](#risks-and-mitigations)

---

## Goals and Motivation

### Primary Goal

**Reduce execution time for repeated quality checks on unchanged files**

Target: 50-80% time reduction when checking files with no changes

### Current Problem

```bash
# First run: 10 files, 5 tools = 50 tool executions
$ gz-quality run
# ✓ gofumpt: 10 files (2.5s)
# ✓ golangci-lint: 10 files (15.3s)
# ✓ black: 10 files (1.2s)
# Total: 18.0s

# Second run: Same files, no changes = 50 tool executions again!
$ gz-quality run
# ✓ gofumpt: 10 files (2.5s)
# ✓ golangci-lint: 10 files (15.3s)
# ✓ black: 10 files (1.2s)
# Total: 18.0s (redundant!)
```

### Expected Behavior with Cache

```bash
# First run: Cache miss, full execution
$ gz-quality run
# Total: 18.0s

# Second run: Cache hit, skip execution
$ gz-quality run --use-cache
# ✓ gofumpt: 10 files (cached)
# ✓ golangci-lint: 10 files (cached)
# ✓ black: 10 files (cached)
# Total: 0.5s (96% faster!)

# Third run: 2 files changed
$ gz-quality run --use-cache
# ✓ gofumpt: 2 files (0.5s), 8 files (cached)
# ✓ golangci-lint: 2 files (3.1s), 8 files (cached)
# ✓ black: 2 files (0.2s), 8 files (cached)
# Total: 3.8s (79% faster!)
```

---

## Requirements

### Functional Requirements

1. **FR1**: Cache tool execution results per file
2. **FR2**: Invalidate cache when file content changes
3. **FR3**: Invalidate cache when tool version changes
4. **FR4**: Invalidate cache when tool configuration changes
5. **FR5**: Support cache cleanup (size limits, age limits)
6. **FR6**: Provide cache statistics (hit rate, size)
7. **FR7**: Allow cache to be disabled via flag
8. **FR8**: Support cache prewarming for CI/CD

### Non-Functional Requirements

1. **NFR1**: Cache lookup must be < 1ms per file
2. **NFR2**: Cache storage must be < 100MB for typical projects
3. **NFR3**: Cache must be thread-safe (concurrent access)
4. **NFR4**: Cache corruption must not break tool execution
5. **NFR5**: Cache must be portable (macOS, Linux, Windows)

---

## Architecture

### High-Level Design

```
┌─────────────────────────────────────────────────────────────┐
│                       gz-quality CLI                        │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
          ┌──────────────────────┐
          │   Executor           │
          │   (runner.go)        │
          └──────┬───────────────┘
                 │
                 ▼
     ┌──────────────────────────────┐
     │   Cache Layer (NEW)          │
     │                              │
     │  ┌────────────────────────┐  │
     │  │  CacheManager          │  │
     │  │  - Get(key) Result     │  │
     │  │  - Set(key, Result)    │  │
     │  │  - Invalidate(key)     │  │
     │  │  - Stats()             │  │
     │  └────────────────────────┘  │
     │           │                  │
     │           ▼                  │
     │  ┌────────────────────────┐  │
     │  │  CacheStorage          │  │
     │  │  - Read(path) []byte   │  │
     │  │  - Write(path, []byte) │  │
     │  │  - Delete(path)        │  │
     │  └────────────────────────┘  │
     └──────────┬───────────────────┘
                │
                ▼
    ┌─────────────────────────┐
    │  Filesystem             │
    │  ~/.cache/gz-quality/   │
    │  ├── index.db           │
    │  └── results/           │
    │      ├── abc123.json    │
    │      └── def456.json    │
    └─────────────────────────┘
```

### Component Responsibilities

#### 1. CacheManager

```go
type CacheManager struct {
    storage    CacheStorage
    enabled    bool
    maxSize    int64
    maxAge     time.Duration
    hitCount   int64
    missCount  int64
}

func (cm *CacheManager) Get(ctx context.Context, key CacheKey) (*CachedResult, error)
func (cm *CacheManager) Set(ctx context.Context, key CacheKey, result *tools.Result) error
func (cm *CacheManager) Invalidate(ctx context.Context, key CacheKey) error
func (cm *CacheManager) InvalidateAll(ctx context.Context) error
func (cm *CacheManager) Stats() CacheStats
func (cm *CacheManager) Cleanup(ctx context.Context) error
```

**Responsibilities:**
- Cache hit/miss logic
- Cache eviction policies
- Statistics tracking
- Thread-safe operations

#### 2. CacheStorage

```go
type CacheStorage interface {
    Read(key string) ([]byte, error)
    Write(key string, data []byte) error
    Delete(key string) error
    List() ([]string, error)
    Size() (int64, error)
}

// Implementation: FilesystemStorage
type FilesystemStorage struct {
    basePath string
}
```

**Responsibilities:**
- Low-level file I/O
- Directory management
- Atomic writes
- Error handling

#### 3. CacheKey

```go
type CacheKey struct {
    FilePath       string
    FileHash       string // SHA256 of file content
    ToolName       string
    ToolVersion    string
    ConfigHash     string // SHA256 of config file(s)
    OptionsHash    string // SHA256 of ExecuteOptions
}

func (ck CacheKey) String() string {
    // Returns: "gofumpt-v0.7.0-abc123-def456-ghi789"
}
```

**Responsibilities:**
- Unique identification of cache entries
- Deterministic key generation
- Collision avoidance

---

## Cache Key Design

### Key Components

A cache entry is valid only if **ALL** of the following match:

1. **File Content**: Same file hash (SHA256)
2. **Tool Version**: Same tool version
3. **Tool Config**: Same configuration files
4. **Execution Options**: Same flags (--fix, --format-only, etc.)

### Key Generation Algorithm

```go
func GenerateCacheKey(file string, tool tools.QualityTool, options tools.ExecuteOptions) (CacheKey, error) {
    // 1. File hash
    fileContent, err := os.ReadFile(file)
    if err != nil {
        return CacheKey{}, err
    }
    fileHash := sha256.Sum256(fileContent)

    // 2. Tool version
    toolVersion, _ := tool.GetVersion()

    // 3. Config hash
    configFiles := tool.FindConfigFiles(options.ProjectRoot)
    configHash := hashFiles(configFiles)

    // 4. Options hash
    optionsData := fmt.Sprintf("%v", options)
    optionsHashSum := sha256.Sum256([]byte(optionsData))

    return CacheKey{
        FilePath:    file,
        FileHash:    hex.EncodeToString(fileHash[:]),
        ToolName:    tool.Name(),
        ToolVersion: toolVersion,
        ConfigHash:  hex.EncodeToString(configHash[:]),
        OptionsHash: hex.EncodeToString(optionsHashSum[:]),
    }, nil
}
```

### Key String Format

```
{tool}-{version}-{file_hash[:8]}-{config_hash[:8]}-{options_hash[:8]}

Example:
gofumpt-v0.7.0-a1b2c3d4-e5f6g7h8-i9j0k1l2
```

### Why This Design?

✅ **Content-based**: File hash ensures accuracy
✅ **Tool-aware**: Version changes invalidate cache
✅ **Config-aware**: Config changes invalidate cache
✅ **Option-aware**: Different options = different results
✅ **Collision-resistant**: SHA256 provides strong guarantees

---

## Cache Storage

### Directory Structure

```
~/.cache/gz-quality/
├── index.db              # SQLite index (fast lookup)
├── results/              # Cached results (JSON)
│   ├── go/
│   │   ├── gofumpt/
│   │   │   ├── a1b2c3d4.json
│   │   │   └── e5f6g7h8.json
│   │   └── golangci-lint/
│   │       └── i9j0k1l2.json
│   ├── python/
│   │   └── black/
│   │       └── m3n4o5p6.json
│   └── javascript/
│       └── eslint/
│           └── q7r8s9t0.json
└── metadata.json         # Cache metadata (version, stats)
```

### Cache Entry Format

```json
{
  "version": "1.0",
  "key": {
    "file_path": "/project/src/main.go",
    "file_hash": "a1b2c3d4e5f6...",
    "tool_name": "gofumpt",
    "tool_version": "v0.7.0",
    "config_hash": "e5f6g7h8i9j0...",
    "options_hash": "i9j0k1l2m3n4..."
  },
  "result": {
    "tool": "gofumpt",
    "language": "Go",
    "success": true,
    "files_processed": 1,
    "duration": "0.5s",
    "issues": [],
    "output": ""
  },
  "metadata": {
    "created_at": "2025-12-02T00:00:00Z",
    "last_accessed": "2025-12-02T01:00:00Z",
    "access_count": 5,
    "size_bytes": 1024
  }
}
```

### Index Database (SQLite)

```sql
CREATE TABLE cache_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cache_key TEXT UNIQUE NOT NULL,
    file_path TEXT NOT NULL,
    tool_name TEXT NOT NULL,
    tool_version TEXT NOT NULL,
    result_path TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    last_accessed INTEGER NOT NULL,
    access_count INTEGER DEFAULT 0,
    size_bytes INTEGER DEFAULT 0,
    INDEX idx_file_tool (file_path, tool_name),
    INDEX idx_created_at (created_at),
    INDEX idx_last_accessed (last_accessed)
);

CREATE TABLE cache_metadata (
    key TEXT PRIMARY KEY,
    value TEXT
);

INSERT INTO cache_metadata VALUES ('version', '1.0');
INSERT INTO cache_metadata VALUES ('created_at', '2025-12-02T00:00:00Z');
```

**Why SQLite?**

✅ Fast lookups (indexed queries)
✅ ACID guarantees (data integrity)
✅ Concurrent reads (good for parallel execution)
✅ Built-in (no external dependencies)
✅ Lightweight (embedded database)

---

## Cache Invalidation

### Invalidation Triggers

| Trigger | Action | Reason |
|---------|--------|--------|
| **File modified** | Delete entries for that file | Content changed |
| **Tool upgraded** | Delete entries for that tool | Different version |
| **Config changed** | Delete entries using that config | Different rules |
| **Manual clear** | Delete all entries | User request |
| **Age limit** | Delete old entries | Prevent unbounded growth |
| **Size limit** | Delete oldest entries | Prevent disk exhaustion |

### LRU Eviction Policy

```go
func (cm *CacheManager) Cleanup(ctx context.Context) error {
    // 1. Delete entries older than maxAge
    cutoffTime := time.Now().Add(-cm.maxAge)
    cm.storage.DeleteWhere("last_accessed < ?", cutoffTime.Unix())

    // 2. Check total size
    totalSize, _ := cm.storage.Size()
    if totalSize > cm.maxSize {
        // Delete oldest entries until under limit
        entriesToDelete := cm.storage.Query(`
            SELECT cache_key
            FROM cache_entries
            ORDER BY last_accessed ASC
            LIMIT ?
        `, calculateDeleteCount(totalSize, cm.maxSize))

        for _, key := range entriesToDelete {
            cm.Invalidate(ctx, key)
        }
    }

    return nil
}
```

### Automatic Cleanup Schedule

- **On startup**: Quick cleanup (delete corrupt entries)
- **After execution**: If cache size > 90% of limit
- **Background**: Every 1 hour (async goroutine)
- **On exit**: Final cleanup (graceful shutdown)

---

## Integration Points

### 1. Executor Integration

```go
// executor/runner.go

func (e *ParallelExecutor) worker(
    ctx context.Context,
    wg *sync.WaitGroup,
    taskChan <-chan tools.Task,
    resultChan chan<- *tools.Result,
    errorChan chan<- error,
    cache *cache.CacheManager, // NEW
) {
    defer wg.Done()

    for task := range taskChan {
        // Check cache first
        if cache != nil && cache.Enabled() {
            cacheKey, _ := cache.GenerateKey(task.Files[0], task.Tool, task.Options)
            if cachedResult, err := cache.Get(ctx, cacheKey); err == nil {
                // Cache hit!
                resultChan <- cachedResult.ToToolResult()
                continue
            }
        }

        // Cache miss: execute tool
        result, err := task.Tool.Execute(ctx, task.Files, task.Options)

        // Store in cache
        if cache != nil && cache.Enabled() && result.Success {
            cacheKey, _ := cache.GenerateKey(task.Files[0], task.Tool, task.Options)
            cache.Set(ctx, cacheKey, result)
        }

        resultChan <- result
        errorChan <- err
    }
}
```

### 2. CLI Integration

```go
// quality.go

var (
    cacheEnabled bool
    cacheDir     string
    cacheMaxSize int64
    cacheMaxAge  time.Duration
)

func init() {
    runCmd.Flags().BoolVar(&cacheEnabled, "cache", true, "Enable result caching")
    runCmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Cache directory (default: ~/.cache/gz-quality)")
    runCmd.Flags().Int64Var(&cacheMaxSize, "cache-max-size", 100*1024*1024, "Max cache size in bytes")
    runCmd.Flags().DurationVar(&cacheMaxAge, "cache-max-age", 7*24*time.Hour, "Max cache age")

    // Cache management commands
    rootCmd.AddCommand(cacheClearCmd)
    rootCmd.AddCommand(cacheStatsCmd)
}

var cacheClearCmd = &cobra.Command{
    Use:   "cache-clear",
    Short: "Clear the result cache",
    Run: func(cmd *cobra.Command, args []string) {
        cache := cache.NewCacheManager(cacheDir, cacheMaxSize, cacheMaxAge)
        cache.InvalidateAll(context.Background())
    },
}

var cacheStatsCmd = &cobra.Command{
    Use:   "cache-stats",
    Short: "Show cache statistics",
    Run: func(cmd *cobra.Command, args []string) {
        cache := cache.NewCacheManager(cacheDir, cacheMaxSize, cacheMaxAge)
        stats := cache.Stats()
        fmt.Printf("Cache Statistics:\n")
        fmt.Printf("  Entries: %d\n", stats.Entries)
        fmt.Printf("  Size: %s\n", humanize.Bytes(stats.SizeBytes))
        fmt.Printf("  Hit Rate: %.2f%%\n", stats.HitRate*100)
    },
}
```

### 3. Configuration Integration

```yaml
# .gzquality.yml

cache:
  enabled: true
  directory: "~/.cache/gz-quality"
  max_size: "100MB"
  max_age: "7d"
  cleanup_interval: "1h"
```

---

## Performance Impact

### Expected Improvements

#### Scenario 1: Repeated Full Checks (No Changes)

```bash
# Without cache
$ time gz-quality run
# Total: 18.0s

# With cache (cache hit)
$ time gz-quality run --cache
# Total: 0.5s
# Improvement: 97% faster
```

#### Scenario 2: Incremental Changes (2/10 files changed)

```bash
# Without cache
$ time gz-quality run
# Total: 18.0s

# With cache (partial hit)
$ time gz-quality run --cache
# Total: 3.8s (2 files executed, 8 files cached)
# Improvement: 79% faster
```

#### Scenario 3: Large Codebase (1000 files, 10 changed)

```bash
# Without cache
$ time gz-quality run
# Total: 5m 30s

# With cache (99% hit rate)
$ time gz-quality run --cache
# Total: 15s
# Improvement: 95% faster
```

### Cache Performance Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Lookup time** | < 1ms | Indexed SQLite query |
| **Write time** | < 5ms | JSON write + DB insert |
| **Hit rate (typical)** | > 80% | Based on change frequency |
| **Hit rate (CI/CD)** | > 95% | Small PRs, cached base |
| **Storage overhead** | < 100MB | For 1000-file project |
| **Memory overhead** | < 10MB | In-memory index only |

### Benchmark Targets

```go
// Benchmark: Cache operations
func BenchmarkCacheGet(b *testing.B) {
    // Target: < 1ms per operation
}

func BenchmarkCacheSet(b *testing.B) {
    // Target: < 5ms per operation
}

func BenchmarkCacheLookup1000Files(b *testing.B) {
    // Target: < 1s for 1000 lookups
}
```

---

## Implementation Plan

### Phase 1: Core Infrastructure ✅ COMPLETED

**Tasks:**
1. ✅ Create `cache/` package structure
2. ✅ Implement `CacheKey` generation
3. ✅ Implement `FilesystemStorage`
4. ✅ Implement `CacheManager` (basic operations)
5. ✅ Add unit tests (>80% coverage)

**Deliverables:**
- ✅ `cache/key.go` - Cache key generation with SHA256 hashing
- ✅ `cache/storage.go` - Filesystem storage with atomic writes
- ✅ `cache/manager.go` - Cache manager with hit/miss tracking
- ✅ `cache/types.go` - Type definitions
- ✅ `cache/*_test.go` - Comprehensive test suite

### Phase 2: Executor Integration ✅ COMPLETED

**Tasks:**
1. ✅ Add cache parameter to `ParallelExecutor`
2. ✅ Implement cache lookup before tool execution
3. ✅ Implement cache storage after successful execution
4. ✅ Add integration tests
5. ✅ Add file-specific result filtering for multi-file caching

**Deliverables:**
- ✅ Modified `executor/runner.go` with `NewParallelExecutorWithCache()`
- ✅ `executor/cache_integration_test.go` - Integration tests
- ✅ `filterIssuesByFile()` - Per-file issue extraction

### Phase 3: CLI Integration ✅ COMPLETED

**Tasks:**
1. ✅ Add `--cache` and `--no-cache` flags to commands
2. ✅ Implement `cache clear` command
3. ✅ Implement `cache stats` command
4. ✅ Add configuration file support (`CacheConfig`)
5. ✅ Cache state display in results

**Deliverables:**
- ✅ Modified `quality.go` with cache commands
- ✅ Modified `config/config.go` with `CacheConfig` struct
- ✅ CLI help text updates
- ✅ Cache hit indicator in results output

### Phase 4: Optimization & Production 🔄 IN PROGRESS

**Tasks:**
1. ⏳ Add SQLite index for fast lookups (using filesystem for now)
2. ✅ Implement LRU eviction policy (`Cleanup()` method)
3. ⏳ Add cache prewarming for CI/CD
4. ✅ Performance testing with benchmarks
5. ✅ Edge case handling (concurrent access via mutex)

**Deliverables:**
- ✅ `executor/bench_test.go` - Cache performance benchmarks
- ✅ LRU-based cleanup in `manager.go`
- ⏳ SQLite integration (optional enhancement)
- ⏳ CI/CD prewarming documentation

**Benchmark Results (Apple M1 Ultra):**
| Scenario | Time | Notes |
|----------|------|-------|
| Cache Hit | ~335µs/op | Single file, warm cache |
| Cache Miss | ~350µs/op | Single file, cold cache |
| Multi-file All Hit | ~3.1ms/op | 10 files, all cached |
| No Cache (baseline) | ~5µs/op | Mock tool execution |

---

## Testing Strategy

### Unit Tests

```go
// cache/key_test.go
func TestGenerateCacheKey(t *testing.T)
func TestCacheKey_String(t *testing.T)
func TestCacheKey_Collision(t *testing.T)

// cache/storage_test.go
func TestFilesystemStorage_ReadWrite(t *testing.T)
func TestFilesystemStorage_Delete(t *testing.T)
func TestFilesystemStorage_ConcurrentAccess(t *testing.T)

// cache/manager_test.go
func TestCacheManager_GetSet(t *testing.T)
func TestCacheManager_Invalidation(t *testing.T)
func TestCacheManager_Eviction(t *testing.T)
func TestCacheManager_Stats(t *testing.T)
```

### Integration Tests

```go
// executor/cache_integration_test.go
func TestExecutor_WithCache_FullHit(t *testing.T)
func TestExecutor_WithCache_PartialHit(t *testing.T)
func TestExecutor_WithCache_Miss(t *testing.T)
func TestExecutor_WithCache_Invalidation(t *testing.T)
```

### Performance Tests

```go
// cache/bench_test.go
func BenchmarkCacheGet(b *testing.B)
func BenchmarkCacheSet(b *testing.B)
func BenchmarkCacheKey_Generation(b *testing.B)
func BenchmarkCache_1000Files(b *testing.B)
```

### End-to-End Tests

```bash
# Test cache behavior with real tools
$ gz-quality run --cache  # First run (cache miss)
$ gz-quality run --cache  # Second run (cache hit)
$ echo "change" >> file.go
$ gz-quality run --cache  # Partial cache hit

# Test cache management
$ gz-quality cache-stats
$ gz-quality cache-clear
```

---

## Risks and Mitigations

### Risk 1: Cache Corruption

**Impact**: High - Broken cache could cause incorrect results
**Probability**: Medium
**Mitigation**:
- Validate cache entries on read (checksum verification)
- Graceful degradation (cache miss on corruption)
- Automatic cleanup of corrupt entries
- Unit tests for edge cases

### Risk 2: Stale Cache

**Impact**: High - Outdated results could be served
**Probability**: Low (with proper invalidation)
**Mitigation**:
- Strong cache key (includes file hash, tool version, config)
- Conservative invalidation (invalidate on any doubt)
- Cache version in metadata (invalidate on upgrade)
- Manual clear command (`gz-quality cache-clear`)

### Risk 3: Disk Space Exhaustion

**Impact**: Medium - Large cache could fill disk
**Probability**: Low (with size limits)
**Mitigation**:
- Default size limit (100MB)
- LRU eviction policy
- Automatic cleanup
- User-configurable limits

### Risk 4: Performance Regression

**Impact**: Medium - Cache overhead could slow down small projects
**Probability**: Low
**Mitigation**:
- Cache lookup must be < 1ms (SQLite index)
- Only cache successful results
- Disable cache for small projects (< 10 files)
- Performance benchmarks in CI

### Risk 5: Concurrent Access Issues

**Impact**: Medium - Race conditions in parallel execution
**Probability**: Medium
**Mitigation**:
- Use SQLite (built-in concurrency control)
- Atomic file writes (write-then-rename)
- File locks for critical sections
- Concurrency tests

---

## Future Enhancements

### Phase 5: Advanced Features (Future)

1. **Distributed Cache**: Share cache across team (Redis, S3)
2. **Cache Compression**: Reduce storage overhead
3. **Smart Prewarming**: Predict cache needs
4. **Cache Analytics**: Dashboard for cache performance
5. **Incremental Results**: Cache partial results (per-function)

---

## Appendix

### A. Cache Key Examples

```
File: src/main.go (modified)
Tool: gofumpt v0.7.0
Config: .gofumpt.yml (unchanged)
Options: --fix

Key: gofumpt-v0.7.0-a1b2c3d4-e5f6g7h8-i9j0k1l2
     [tool]  [ver]  [file]   [config]  [opts]
```

### B. Cache Storage Math

```
Assumptions:
- 1000 files in project
- 5 tools per language
- Average result size: 2KB

Total cache size:
= 1000 files × 5 tools × 2KB
= 10,000 KB
= ~10 MB

With overhead (index, metadata):
= 10 MB × 1.5
= ~15 MB

Conclusion: Well under 100MB limit
```

### C. Performance Calculation

```
Scenario: 100 files, 1 file changed, 5 tools

Without cache:
= 100 files × 5 tools × 2s/tool
= 1000s
= ~16.7 minutes

With cache (99% hit rate):
= 1 file × 5 tools × 2s/tool + 99 lookups × 0.001s
= 10s + 0.099s
= ~10s

Speedup: 1000s / 10s = 100x faster!
```

---

**Document Version**: 1.1
**Last Updated**: 2025-12-02
**Status**: Implementation Complete (Phase 1-3), Optimization In Progress (Phase 4)
**Authors**: Claude (claude-sonnet-4-5, claude-opus-4-5), archmagece

### Implementation Summary

| Phase | Status | Commits |
|-------|--------|---------|
| Phase 1: Core Infrastructure | ✅ Complete | `b799237`, `a5c6680` |
| Phase 2: Executor Integration | ✅ Complete | `208c9f9`, `afd9463` |
| Phase 3: CLI Integration | ✅ Complete | `f532bc1`, `9908056` |
| Phase 4: Optimization | 🔄 In Progress | `563ffe8` |

**Total Lines Added**: ~2000+ lines of production code and tests
