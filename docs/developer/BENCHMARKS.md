# Performance Benchmarks

## Overview

This document contains baseline performance benchmarks for the gzh-cli-quality project. Benchmarks measure the performance of critical operations across tools, detector, and executor packages.

## G2 캐시 판정 기준 — 2026-10-25

PRODUCT.md G2는 비캐시 대비 60배에서 "모두 적중 시 도구 실행·버전 조회 subprocess
0회, quality 자체 처리 시간 파일당 100µs 이하"로 바뀌었다. 판정값은 아래 workload(source
`d7cf583`의 35파일)로 `BenchmarkCache_InstalledGofumpt`를 `-count=6` 실행한 WarmHit
중앙값 ÷ 파일 수다. 배율은 비캐시 실행 비용 ÷ warm hit 비용이라 quality가 소유하지 않는
도구 비용에 좌우된다. 아래 계열의 비캐시 약 10~11ms에서 60배는 WarmHit 약 0.17~0.19ms(파일당
약 5µs) 이하를 요구하는데, 이는 적중 35회의 `CacheMetadataReads` component 0.902ms보다도
작다. 그래서 quality가 소유한 비용만 판정한다. 배율은 참고값으로 계속 기록하고, 60배 미충족
이력은 아래에 그대로 보존한다. no-op mock benchmark는 도구 실행이 없어 이 판정에 쓰지 않는다.

100µs는 아래 6회 계열 값의 약 2.2~2.3배다. 오버헤드가 그 이상 늘면 잡아내지만 2.2배
미만의 퇴행은 놓친다. warm hit는 프로세스를 띄우지 않으므로 변동 원인은 디스크
경합으로 추정하며(분리 측정 없음), 같은 workload에서 WarmHit 중앙값이 약 3.2배(1.522 → 4.824ms) 움직인 기록이 있다.

| 계열 (모두 source `d7cf583` 35파일)               | WarmHit 중앙값 | 파일당   | 도구 실행 / 버전 조회 | 참고 배율 | 새 G2  |
| ------------------------------------------------- | -------------- | -------- | --------------------- | --------- | ------ |
| 버전 캐시 추가 후 `-count=6` (아래 절)            | 1.522 ms       | 43.5 µs  | 0 / 0                 | 약 6.8배  | 충족   |
| devbox TASK-356 `-count=6` (quality `42c5c0f`)    | 1.607675 ms    | 45.9 µs  | 0 / 0                 | 약 6.9배  | 충족   |
| devbox ISSUE-362 동시 세션 `-count=3` (`42c5c0f`) | 4.824 ms       | 137.8 µs | 0 / 0                 | 약 7.9배  | 미충족 |

판정 조건(`-count=6`)을 갖춘 두 계열은 충족하고 측정 전후 load average를 기록했다.
다른 세션의 `-count=3` 계열은 미충족이며 부하를 기록하지 않았다. 그 계열은 NoCache도
38.089ms로 높아 경합을 받은 것으로 추정하지만 원인을 분리하지 않았다. 이 판정은 보존된
계열을 새 기준으로 읽은 것이며 2026-10-25에 새로 측정하지 않았다. 미충족 계열의 원인을
분리하거나 부하를 기록한 반복 측정으로 다시 판정하기 전에는 G2 충족을 확정하지 않는다. 이 workload는 `_test.go`와 `testdata/`만 제외하므로 testutil 같은
보조 Go 파일도 포함한다.

## 실제 formatter 캐시 측정 — 2026-10-24

이 절은 2026-10-24 관측이며, 아래 2025년 수치는 당시 기록으로 유지한다. 당시 PRODUCT의
캐시 60배 목표를 낮추거나 no-op mock benchmark로 충족했다고 판정하지 않았다.

측정 기간은 2026-10-23~24, 환경은 Apple M5 Max / darwin arm64 / Go 1.27.3이다.
기존 `$(go env GOPATH)/bin/gofumpt` v0.10.0 (go1.26.3)을 사용했고 새 formatter는
설치하지 않았다. workload는 source `d7cf583210f2174d2c58746f6c939277546a5bbe`의
tracked Go 파일 35개(`_test.go`·`testdata/` 제외, testutil·fixture 포함), 224,789 bytes를 임시 디렉터리에 복사한 것이다.
테스트·testdata는 이 **benchmark 입력**에서 제외하며 제품 테스트 커버리지의
package/파일 선택이나 하한은 변경하지 않는다. 경로와 content를 차례로 결합한
workload SHA-256은 `ec3ce86dbe97d934fe390f18379ba672378cfb791c0049c8dd06f579d7be8435`다.

`BenchmarkCache_InstalledGofumpt`는 고정 ExecutionPlan과 실제 adapter / executor /
disk cache를 사용한다. 각 반복의 도구 결과·처리 파일 수·Cached 여부를 검증하고 실제 실행·
버전 조회 호출 수를 출력한다. check만 실행하며 workload bytes를 수정하지 않는다.
ColdMiss는 **애플리케이션 캐시만** 비우고 그 삭제 시간은 측정에서 제외한다.
WarmHit는 사전 한 번 실제 도구를 실행한 뒤 전부 적중하는 상태다. OS page cache를
강제로 비우지 않았으며, 도구 설치·Go 빌드·workload 복사는 시간에 포함하지 않는다.

`-benchtime=300ms -count=3`, task 1개 / worker 1개 결과이며 시간 단위는 ms/op다.
수정 후 계열은 version snapshot과 cache identity 수정 후, 후속 invalid UTF-8
cache 우회 방어를 추가하기 전 candidate를 측정했다. 최종 UTF-8 방어 코드는
이 3회 성능 계열에서 다시 측정하지 않았다.

| 경로     | 수정 전 3회                 | 수정 후 3회                 | 실제 tool 실행 / version 조회 (수정 전 → 후) |
| -------- | --------------------------- | --------------------------- | -------------------------------------------- |
| NoCache  | 22.116 / 33.900 / 24.373    | 20.759 / 19.031 / 20.539    | 1 / 0 → 1 / 0                                |
| ColdMiss | 711.141 / 737.657 / 787.728 | 122.061 / 134.127 / 155.195 | 1 / 70 → 1 / 1                               |
| WarmHit  | 507.485 / 442.358 / 311.095 | 30.199 / 30.094 / 28.793    | 0 / 35 → 0 / 1                               |

확인된 병목은 파일마다 버전을 조회하고 miss 저장 때 또 조회한 subprocess였다.
버전은 task 안에서만 재사용하고 다음 실행에서는 새로 조회하도록 수정했다.
동시에 경로·전체 hash·argv 순서·명시적 config와 환경이 다른 결과를 재사용하지
않도록 cache identity를 고쳤다. 버전 조회 단독 component는 6.749 / 7.412 /
7.154ms/op여서 수정 후에도 한 번의 probe 비용이 크다. `CacheMetadataReads` component는
실제 formatter 결과를 캐시에 채운 뒤 35개 `Get`만 측정하여 28.932 / 34.848 /
26.875ms/op를 관측했다. `Get`은 적중마다 access metadata를 동기적으로 다시 쓰므로
파일 read/JSON 처리/write 비용이 남는다. CPU profile에도 syscall 경로가 보이지만
setup도 포함하므로 wall time 비율로 해석하지 않는다.
안전한 버전 변경 감지를 없애거나 metadata 정책을 임의로 바꾸지는 않았다.

최종 수정 후 중앙값은 비캐시 20.539ms, warm hit 30.094ms로 캐시가 더 느리다.
앞선 수정 후 계열은 비캐시 15.120 / 20.905 / 18.253ms, cold miss 76.397 /
54.751 / 54.868ms, warm hit 10.174 / 10.708 / 10.816ms로 중앙값 약 1.70배였다.
동일 입력의 계열 간 차이 원인을 분리 측정하지 않았으므로 어느 계열도 보편적인
개선 배율로 주장하지 않는다. **60배는 두 계열 모두 미충족**이며 캐시 효과도 일관되지 않았다.
더 비싼 도구·더 큰 입력을 골라 목표를 달성한 것처럼 일반화하지 않는다.
이 측정은 파일 formatter workload이며 package 전체를 분석하는 linter에는 일반화하지 않는다.

### 실행 간 버전 캐시 추가 후 — 2026-10-24

[CACHING.md](CACHING.md)의 "도구 버전 캐시"를 추가한 뒤 같은 환경·workload로 다시
측정했다(workload SHA-256 위와 동일). benchmark wrapper가 `VersionProbe`/`VersionCacheable`을
전달하므로 WarmHit는 실제 도구와 같은 경로를 탄다. `WarmHitNoVersionCache`는 같은
WarmHit에서 wrapper가 `tools.VersionProber`를 구현하지 않게 하여 버전 캐시만 끈
변형이다. 결과 캐시는 그대로 적중하고 버전은 실행마다 조회한다. ColdMiss는
`InvalidateAll`이 버전 entry도 지우므로 조회가 1회 남는다.

방법: `-benchtime=300ms -count=6`, task 1개 / worker 1개, 실행 전후 load average
2.7~3.8(다른 VM 1개 동작 중, 그 밖의 무거운 작업 없음). 시간은 ms/op이고 6회의
중앙값과 최소~최대 범위다. 같은 날 앞서 실행한 `-count=3` 계열은 CacheMetadataReads가
37~61ms/op로 아래보다 40배 이상 커서 디스크 경합에 오염된 것으로 보고 폐기했다.

| 경로                  | 중앙값 | 범위 (6회)      | 실제 tool 실행 / version 조회 |
| --------------------- | ------ | --------------- | ----------------------------- |
| NoCache               | 10.366 | 10.289 ~ 10.579 | 1 / 0                         |
| ColdMiss              | 53.089 | 52.900 ~ 54.192 | 1 / 1                         |
| WarmHit               | 1.522  | 1.508 ~ 1.551   | 0 / 0                         |
| WarmHitNoVersionCache | 4.866  | 4.770 ~ 5.757   | 0 / 1                         |

component: `VersionProbe` 2.871 (2.848 ~ 2.913), `CacheMetadataReads` 0.902
(0.890 ~ 0.909)ms/op.

버전 캐시는 warm hit에서 조회 1회를 없애 중앙값을 4.866 → 1.522ms로 줄였다. 비캐시
대비 warm hit는 버전 캐시를 끄면 약 2.1배, 켜면 약 6.8배 빠르다. **60배는 미충족**이다.
이 측정은 파일 formatter workload이며 package 전체를 분석하는 linter에는 일반화하지 않는다.

재현에는 이미 설치된 gofumpt와 Git source revision이 필요하다. 저장소 root에서
`QUALITY_BENCH_GOFUMPT`를 PATH가 해석하는 실제 gofumpt 절대 경로로,
`QUALITY_BENCH_SOURCE_REF`를 위 SHA로 지정하고 다음 benchmark를 실행한다.
도구가 설치되지 않았으면 이 opt-in benchmark는 설치하지 않고 그 이유를 알린다.

```bash
go test ./executor -run '^$' -bench '^BenchmarkCache_InstalledGofumpt$' -benchtime=300ms -count=3
```

버전 캐시 on/off 비교(위 "실행 간 버전 캐시 추가 후" 표)는 같은 benchmark를 `-count=6`으로 실행했다.

```bash
GOWORK=off go test ./executor -run '^$' -bench '^BenchmarkCache_InstalledGofumpt$' -benchtime=300ms -count=6 -v
```

`VersionProbe`와 `CacheMetadataReads`는 별도 component 진단이며 제품 처리량 결과가 아니다. 각 실행의
tool version / source revision / 파일 수 / bytes / workload hash는 `-v`로 확인한다.
명시적 config, content, 버전, 작업 디렉터리, argv, Env 변경과 cached 진단 경로는
일반 회귀 테스트로 검증하며 benchmark를 정확성 검사의 대용으로 쓰지 않는다.

## Test Environment

- **CPU**: Apple M1 Ultra (20 cores)
- **OS**: macOS (darwin arm64)
- **Go Version**: 1.24.0
- **Date**: 2025-11-29

## Benchmark Results

### Tools Package (11 benchmarks)

#### File Filtering Operations

| Benchmark                                     | Operations/sec | ns/op  | B/op   | allocs/op |
| --------------------------------------------- | -------------- | ------ | ------ | --------- |
| FilterFilesByExtensions                       | 2,557,035      | 471.4  | 240    | 4         |
| FilterFilesByExtensions_LargeSet (1000 files) | 58,000         | 20,834 | 18,800 | 10        |

**Analysis**: File filtering is very fast for typical use cases (sub-microsecond). Large file sets (1000 files) still complete in ~21μs.

#### Command Building

| Benchmark                | Operations/sec | ns/op  | B/op   | allocs/op |
| ------------------------ | -------------- | ------ | ------ | --------- |
| GofumptTool_BuildCommand | 12,385         | 86,210 | 25,624 | 260       |
| BaseTool_Execute         | 14,407         | 84,313 | 27,190 | 266       |

**Analysis**: Command building takes ~86μs with 260 allocations. Most time is spent on path operations and argument construction.

#### Output Parsing (JSON)

| Benchmark                    | Operations/sec | ns/op | B/op  | allocs/op |
| ---------------------------- | -------------- | ----- | ----- | --------- |
| GolangciLintTool_ParseOutput | 347,692        | 3,429 | 1,432 | 19        |
| ESLintTool_ParseOutput       | 329,967        | 3,662 | 1,440 | 21        |
| RuffTool_ParseOutput         | 298,702        | 3,990 | 1,256 | 18        |
| ClippyTool_ParseOutput       | 292,765        | 4,115 | 1,744 | 34        |

**Analysis**: JSON parsing is very efficient (~3-4μs per parse). All parsers perform similarly with minimal memory allocation.

#### Registry Operations

| Benchmark                   | Operations/sec | ns/op | B/op | allocs/op |
| --------------------------- | -------------- | ----- | ---- | --------- |
| Registry_FindTool           | 82,157,780     | 14.17 | 0    | 0         |
| Registry_GetToolsByLanguage | 5,503,344      | 221.0 | 112  | 3         |
| Registry_GetToolsByType     | 4,695,355      | 261.6 | 240  | 4         |

**Analysis**: Tool lookup is extremely fast (14ns) with zero allocations. Language/type filtering is also very efficient (\<300ns).

### Detector Package (9 benchmarks)

#### Language Detection

| Benchmark                                                 | Operations/sec | ns/op   | B/op   | allocs/op |
| --------------------------------------------------------- | -------------- | ------- | ------ | --------- |
| FileTypeDetector_DetectLanguages (10 files)               | 17,251         | 69,895  | 13,008 | 160       |
| FileTypeDetector_DetectLanguages_LargeProject (100 files) | 2,432          | 484,908 | 82,416 | 1,292     |
| FileTypeDetector_GetFilesByLanguage                       | 29,356         | 40,454  | 6,280  | 61        |

**Analysis**:

- Small projects (10 files): ~70μs
- Large projects (100 files): ~485μs (~4.85μs per file)
- Language filtering: ~40μs
- Scales linearly with file count

#### Tool Detection

| Benchmark                          | Operations/sec | ns/op      | B/op   | allocs/op |
| ---------------------------------- | -------------- | ---------- | ------ | --------- |
| SystemToolDetector_IsToolAvailable | 168,014,928    | 7.139      | 0      | 0         |
| SystemToolDetector_GetToolVersion  | 37             | 32,373,319 | 98,997 | 196       |

**Analysis**:

- Tool availability check: 7ns (cached, extremely fast)
- Version retrieval: ~32ms (calls actual `go version`, expected overhead)

#### Config File Discovery

| Benchmark                      | Operations/sec | ns/op  | B/op   | allocs/op |
| ------------------------------ | -------------- | ------ | ------ | --------- |
| ConfigFileDetector_FindConfigs | 27,130         | 43,502 | 10,672 | 82        |

**Analysis**: Config file discovery takes ~44μs for 4 config files. Dominated by filesystem operations.

#### Project Analysis

| Benchmark                               | Operations/sec | ns/op   | B/op   | allocs/op |
| --------------------------------------- | -------------- | ------- | ------ | --------- |
| ProjectAnalyzer_AnalyzeProject          | 7,802          | 149,333 | 27,259 | 268       |
| ProjectAnalyzer_GetOptimalToolSelection | 2,221,918      | 545.9   | 544    | 8         |
| RemoveDuplicates                        | 1,600,558      | 763.2   | 952    | 8         |

**Analysis**:

- Full project analysis: ~149μs (includes file scanning, language detection, config discovery)
- Tool selection: ~546ns (very fast)
- Duplicate removal: ~763ns

### Executor Package (7 benchmarks)

#### Execution Plan Creation

| Benchmark                              | Operations/sec | ns/op | B/op  | allocs/op |
| -------------------------------------- | -------------- | ----- | ----- | --------- |
| ExecutionPlan_Creation (5 tasks)       | 6,218,366      | 189.6 | 320   | 10        |
| ExecutionPlan_LargeTaskSet (100 tasks) | 296,979        | 3,929 | 4,800 | 200       |

**Analysis**: Plan creation is very lightweight (~190ns for 5 tasks, ~4μs for 100 tasks). Scales linearly.

#### Parallel Execution

| Benchmark                                    | Operations/sec | ns/op  | B/op  | allocs/op |
| -------------------------------------------- | -------------- | ------ | ----- | --------- |
| Executor_ExecuteParallel_1Worker (2 tasks)   | 198,938        | 6,167  | 1,921 | 25        |
| Executor_ExecuteParallel_4Workers (8 tasks)  | 59,943         | 18,961 | 4,690 | 36        |
| Executor_ExecuteParallel_8Workers (16 tasks) | 32,110         | 37,778 | 8,350 | 49        |

**Analysis**:

- 1 worker: 6.2μs per execution (minimal overhead)
- 4 workers: 19μs for 8 tasks (~2.4μs per task)
- 8 workers: 37.8μs for 16 tasks (~2.4μs per task)
- Parallelization overhead is well-managed
- Memory usage scales linearly with worker count

#### Tool Type Filtering

| Benchmark                 | Operations/sec | ns/op | B/op | allocs/op |
| ------------------------- | -------------- | ----- | ---- | --------- |
| ToolTypeFilter_FormatOnly | 149,956,598    | 7.993 | 0    | 0         |
| ToolTypeFilter_LintOnly   | 150,882,662    | 7.922 | 0    | 0         |

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

1. **Low-allocation operations** (< 100 B):

   - Most registry operations
   - Tool selection algorithms

1. **Higher allocations**:

   - Command building: 25KB (path operations)
   - Language detection: 13-82KB (file scanning)

### Scalability

1. **Linear scaling**:

   - File filtering: 471ns → 20.8μs (1000x files = 44x time)
   - Language detection: 70μs → 485μs (10x files = 7x time)
   - Plan creation: 190ns → 3.9μs (20x tasks = 20x time)

1. **Parallel efficiency**:

   - 1 worker: 6.2μs for 2 tasks = 3.1μs/task
   - 4 workers: 19μs for 8 tasks = 2.4μs/task
   - 8 workers: 37.8μs for 16 tasks = 2.4μs/task
   - **Result**: Good parallelization, ~20% overhead reduction

### Bottlenecks

1. **External processes**: Tool version detection (32ms) is slowest operation
1. **Filesystem I/O**: Config file discovery and language detection
1. **Command building**: Path resolution and argument construction

## Running Benchmarks

### Using Make (Recommended)

```bash
# Run all benchmarks
make bench

# Save baseline for comparison
make bench-save

# Compare with baseline (detects regressions)
make bench-compare
```

### Using Go Test Directly

```bash
# Run all benchmarks
go test -bench=. -benchmem ./tools ./detector ./executor

# Run specific package benchmarks
go test -bench=. -benchmem ./tools
go test -bench=. -benchmem ./detector
go test -bench=. -benchmem ./executor

# Run specific benchmark
go test -bench=BenchmarkFilterFilesByExtensions ./tools
go test -bench=BenchmarkExecutor_ExecuteParallel ./executor

# Run with more iterations for accuracy
go test -bench=. -benchtime=10s ./tools
```

### Statistical Comparison with benchstat

```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Save baseline
go test -bench=. -benchmem ./... > old.txt

# Make changes...

# Save new results
go test -bench=. -benchmem ./... > new.txt

# Compare with statistical analysis
benchstat old.txt new.txt
```

### CI/CD Integration

Benchmarks run automatically on:

- Every pull request (with comparison to base branch)
- Pushes to master/main branches
- Manual workflow dispatch

**Features**:

- Automatic baseline comparison
- PR comments with performance analysis
- Performance regression detection (>10% slower)
- Benchmark results stored as artifacts (30 days)

**Workflow**: `.github/workflows/benchmarks.yml`

**Interpreting CI Results**:

```
name        old time/op  new time/op  delta
Filter-20   453ns ± 2%   412ns ± 1%  -9.05%  (p=0.000 n=10+10)

Positive delta (+X%): Performance regression (slower) ⚠️
Negative delta (-X%): Performance improvement (faster) ✅
~: No significant change
```

## Benchmark Maintenance

### When to update benchmarks

1. After significant algorithm changes
1. When adding new critical paths
1. Before performance optimization work
1. During code reviews for performance-sensitive code

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

______________________________________________________________________

**Last Updated**: 2025-11-29
**Total Benchmarks**: 27 (11 tools + 9 detector + 7 executor)
**Test Coverage**: 76.2%
