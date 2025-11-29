# Performance Benchmarks

## Overview

This document contains baseline performance benchmarks for the gzh-cli-quality project. Benchmarks measure the performance of critical operations across tools, detector, and executor packages.

## Test Environment

- **CPU**: Apple M1 Ultra (20 cores)
- **OS**: macOS (darwin arm64)
- **Go Version**: 1.24.0
- **Date**: 2025-11-29

## Benchmark Results

### Tools Package (11 benchmarks)

#### File Filtering Operations

| Benchmark | Operations/sec | ns/op | B/op | allocs/op |
|-----------|----------------|-------|------|-----------|
| FilterFilesByExtensions | 2,557,035 | 471.4 | 240 | 4 |
| FilterFilesByExtensions_LargeSet (1000 files) | 58,000 | 20,834 | 18,800 | 10 |

**Analysis**: File filtering is very fast for typical use cases (sub-microsecond). Large file sets (1000 files) still complete in ~21μs.

#### Command Building

| Benchmark | Operations/sec | ns/op | B/op | allocs/op |
|-----------|----------------|-------|------|-----------|
| GofumptTool_BuildCommand | 12,385 | 86,210 | 25,624 | 260 |
| BaseTool_Execute | 14,407 | 84,313 | 27,190 | 266 |

**Analysis**: Command building takes ~86μs with 260 allocations. Most time is spent on path operations and argument construction.

#### Output Parsing (JSON)

| Benchmark | Operations/sec | ns/op | B/op | allocs/op |
|-----------|----------------|-------|------|-----------|
| GolangciLintTool_ParseOutput | 347,692 | 3,429 | 1,432 | 19 |
| ESLintTool_ParseOutput | 329,967 | 3,662 | 1,440 | 21 |
| RuffTool_ParseOutput | 298,702 | 3,990 | 1,256 | 18 |
| ClippyTool_ParseOutput | 292,765 | 4,115 | 1,744 | 34 |

**Analysis**: JSON parsing is very efficient (~3-4μs per parse). All parsers perform similarly with minimal memory allocation.

#### Registry Operations

| Benchmark | Operations/sec | ns/op | B/op | allocs/op |
|-----------|----------------|-------|------|-----------|
| Registry_FindTool | 82,157,780 | 14.17 | 0 | 0 |
| Registry_GetToolsByLanguage | 5,503,344 | 221.0 | 112 | 3 |
| Registry_GetToolsByType | 4,695,355 | 261.6 | 240 | 4 |

**Analysis**: Tool lookup is extremely fast (14ns) with zero allocations. Language/type filtering is also very efficient (<300ns).

### Detector Package (9 benchmarks)

#### Language Detection

| Benchmark | Operations/sec | ns/op | B/op | allocs/op |
|-----------|----------------|-------|------|-----------|
| FileTypeDetector_DetectLanguages (10 files) | 17,251 | 69,895 | 13,008 | 160 |
| FileTypeDetector_DetectLanguages_LargeProject (100 files) | 2,432 | 484,908 | 82,416 | 1,292 |
| FileTypeDetector_GetFilesByLanguage | 29,356 | 40,454 | 6,280 | 61 |

**Analysis**:
- Small projects (10 files): ~70μs
- Large projects (100 files): ~485μs (~4.85μs per file)
- Language filtering: ~40μs
- Scales linearly with file count

#### Tool Detection

| Benchmark | Operations/sec | ns/op | B/op | allocs/op |
|-----------|----------------|-------|------|-----------|
| SystemToolDetector_IsToolAvailable | 168,014,928 | 7.139 | 0 | 0 |
| SystemToolDetector_GetToolVersion | 37 | 32,373,319 | 98,997 | 196 |

**Analysis**:
- Tool availability check: 7ns (cached, extremely fast)
- Version retrieval: ~32ms (calls actual `go version`, expected overhead)

#### Config File Discovery

| Benchmark | Operations/sec | ns/op | B/op | allocs/op |
|-----------|----------------|-------|------|-----------|
| ConfigFileDetector_FindConfigs | 27,130 | 43,502 | 10,672 | 82 |

**Analysis**: Config file discovery takes ~44μs for 4 config files. Dominated by filesystem operations.

#### Project Analysis

| Benchmark | Operations/sec | ns/op | B/op | allocs/op |
|-----------|----------------|-------|------|-----------|
| ProjectAnalyzer_AnalyzeProject | 7,802 | 149,333 | 27,259 | 268 |
| ProjectAnalyzer_GetOptimalToolSelection | 2,221,918 | 545.9 | 544 | 8 |
| RemoveDuplicates | 1,600,558 | 763.2 | 952 | 8 |

**Analysis**:
- Full project analysis: ~149μs (includes file scanning, language detection, config discovery)
- Tool selection: ~546ns (very fast)
- Duplicate removal: ~763ns

### Executor Package (7 benchmarks)

#### Execution Plan Creation

| Benchmark | Operations/sec | ns/op | B/op | allocs/op |
|-----------|----------------|-------|------|-----------|
| ExecutionPlan_Creation (5 tasks) | 6,218,366 | 189.6 | 320 | 10 |
| ExecutionPlan_LargeTaskSet (100 tasks) | 296,979 | 3,929 | 4,800 | 200 |

**Analysis**: Plan creation is very lightweight (~190ns for 5 tasks, ~4μs for 100 tasks). Scales linearly.

#### Parallel Execution

| Benchmark | Operations/sec | ns/op | B/op | allocs/op |
|-----------|----------------|-------|------|-----------|
| Executor_ExecuteParallel_1Worker (2 tasks) | 198,938 | 6,167 | 1,921 | 25 |
| Executor_ExecuteParallel_4Workers (8 tasks) | 59,943 | 18,961 | 4,690 | 36 |
| Executor_ExecuteParallel_8Workers (16 tasks) | 32,110 | 37,778 | 8,350 | 49 |

**Analysis**:
- 1 worker: 6.2μs per execution (minimal overhead)
- 4 workers: 19μs for 8 tasks (~2.4μs per task)
- 8 workers: 37.8μs for 16 tasks (~2.4μs per task)
- Parallelization overhead is well-managed
- Memory usage scales linearly with worker count

#### Tool Type Filtering

| Benchmark | Operations/sec | ns/op | B/op | allocs/op |
|-----------|----------------|-------|------|-----------|
| ToolTypeFilter_FormatOnly | 149,956,598 | 7.993 | 0 | 0 |
| ToolTypeFilter_LintOnly | 150,882,662 | 7.922 | 0 | 0 |

**Analysis**: Tool type filtering is extremely fast (~8ns) with zero allocations. Negligible overhead.

## Performance Characteristics

### Fast Operations (< 1μs)

- Registry lookups: 14-262ns
- Tool type filtering: 8ns
- Tool availability check: 7ns
- File filtering (small sets): 471ns

### Medium Operations (1-100μs)

- JSON parsing: 3-4μs
- Execution plan creation: 0.2-4μs
- Parallel execution: 6-38μs
- Language detection (small): 70μs
- Command building: 86μs

### Slow Operations (> 100μs)

- Project analysis: 149μs
- Language detection (100 files): 485μs
- Tool version retrieval: 32ms (external process)

## Optimization Insights

### Memory Efficiency

1. **Zero-allocation operations**:
   - Registry_FindTool
   - SystemToolDetector_IsToolAvailable
   - ToolTypeFilter operations

2. **Low-allocation operations** (< 100 B):
   - Most registry operations
   - Tool selection algorithms

3. **Higher allocations**:
   - Command building: 25KB (path operations)
   - Language detection: 13-82KB (file scanning)

### Scalability

1. **Linear scaling**:
   - File filtering: 471ns → 20.8μs (1000x files = 44x time)
   - Language detection: 70μs → 485μs (10x files = 7x time)
   - Plan creation: 190ns → 3.9μs (20x tasks = 20x time)

2. **Parallel efficiency**:
   - 1 worker: 6.2μs for 2 tasks = 3.1μs/task
   - 4 workers: 19μs for 8 tasks = 2.4μs/task
   - 8 workers: 37.8μs for 16 tasks = 2.4μs/task
   - **Result**: Good parallelization, ~20% overhead reduction

### Bottlenecks

1. **External processes**: Tool version detection (32ms) is slowest operation
2. **Filesystem I/O**: Config file discovery and language detection
3. **Command building**: Path resolution and argument construction

## Running Benchmarks

### Run all benchmarks

```bash
go test -bench=. -benchmem ./tools ./detector ./executor
```

### Run specific package benchmarks

```bash
go test -bench=. -benchmem ./tools
go test -bench=. -benchmem ./detector
go test -bench=. -benchmem ./executor
```

### Run specific benchmark

```bash
go test -bench=BenchmarkFilterFilesByExtensions ./tools
go test -bench=BenchmarkExecutor_ExecuteParallel ./executor
```

### Run with more iterations for accuracy

```bash
go test -bench=. -benchtime=10s ./tools
```

### Compare benchmarks (after changes)

```bash
# Save baseline
go test -bench=. -benchmem ./... > old.txt

# Make changes...

# Compare
go test -bench=. -benchmem ./... > new.txt
benchcmp old.txt new.txt
```

## Benchmark Maintenance

### When to update benchmarks

1. After significant algorithm changes
2. When adding new critical paths
3. Before performance optimization work
4. During code reviews for performance-sensitive code

### What to benchmark

✅ **Do benchmark**:
- Core algorithms (filtering, parsing, execution)
- Frequently called functions
- Operations with known performance requirements
- Parallel execution paths

❌ **Don't benchmark**:
- Simple getters/setters
- One-time initialization code
- External tool execution (too variable)
- UI/output formatting

## Optimization Guidelines

### Target Performance

Based on typical usage (project with 100 files, 5 tools):

- **Project analysis**: < 1ms ✅ (current: ~0.5ms)
- **Execution planning**: < 1ms ✅ (current: ~0.2ms)
- **Tool execution**: Limited by external tools (acceptable)
- **Report generation**: < 100ms (not benchmarked, I/O bound)

### Red Flags

- Any operation taking > 10ms that's not I/O or external process
- Memory allocations growing non-linearly with input size
- Parallel overhead > 50% of single-threaded time

---

**Last Updated**: 2025-11-29
**Total Benchmarks**: 27 (11 tools + 9 detector + 7 executor)
**Test Coverage**: 76.2%
