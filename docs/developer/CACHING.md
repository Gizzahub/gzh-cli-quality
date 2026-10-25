# Caching System Design

Design document for implementing a file-based caching system to improve gz-quality performance by avoiding redundant tool executions.

## 현재 구현 계약 — 2026-10-24

아래 설계 예시의 시간·속도 향상은 측정 결과가 아니다. 실제 설치된 gofumpt로
측정한 cold miss / warm hit / 비캐시 비교는 [BENCHMARKS.md](BENCHMARKS.md)의
2026-10-24 기록을 따른다. 이 workload에서 당시 60배 목표를 충족하지 못했다.
2026-10-25 바뀐 PRODUCT.md G2(적중 시 subprocess 0회·파일당 100µs 이하)의 계열별 판정은
BENCHMARKS.md의 "G2 캐시 판정 기준" 절을 따른다.

- 도구 버전은 파일마다 subprocess를 호출하지 않고 **한 task 안에서 한 번**
  얻는다. 실행 사이에는 아래 "도구 버전 캐시" 절의 캐시가 실행 파일이
  바뀌지 않은 동안에만 이전 조회 결과를 재사용하고, 그 밖에는 다시 조회하므로
  도구 업그레이드를 숨기지 않는다. 버전 조회 오류나 빈 값/`unknown`이면 캐시를
  읽거나 쓰지 않고 도구를 실행한다.
- `CacheKey`의 공개 필드는 유지한다. `String()`은 `ValidateKey`를 통과한 UTF-8
  identity 전체 필드의 JSON을 SHA-256으로
  해시한 64자리 소문자 hex인 **불투명 storage 식별자**다. 파일 절대 경로와 전체
  content/config/options hash가 포함되며, 기존 8자리 hash 접두사 충돌이나
  동일 content의 다른 파일 진단 재사용을 막는다. 식별자의 문자열 형식을 파싱하지 않는다.
- 기존 접두사 기반 이름의 캐시는 새 이름에서 miss로 취급한다. 충돌 가능한
  legacy fallback을 읽거나 migration을 이유로 기존 파일을 삭제하지 않는다.
  사용자가 요청한 cache-clear와 기존 크기/수명 cleanup 정책은 그대로다.
- 상대 파일 경로는 실행의 `ProjectRoot`를 기준으로 해석한다. options hash는
  작업 디렉터리·명시한 config 경로·fix/format/lint mode·**순서가 있는 argv**·
  명시적 `Env`를 JSON으로 구분하여 저장한다. 환경 map의 key 순서는 의미가 없다.
- 기본 config 탐색 결과에 더해 명시한 `ConfigFile`의 실제 content도 검사한다.
  명시한 config를 읽지 못하면 캐시를 사용하지 않는다.
- JSON이 서로 다른 잘못된 UTF-8 bytes를 같은 replacement character로 바꾸므로,
  identity 필드나 options(argv/Env/root/config 경로)에 잘못된 UTF-8이 있으면
  캐시를 우회한다. 도구의 실제 입력을 정규화하거나 실행을 거부하지 않는다.

캐시의 task별 probe와 파일별 진단·content/config/version/options 변경은 회귀
테스트로 확인한다. 다만 현재 키는 파일 단위이며 다른 소스 파일, package dependency,
탐색 목록 밖의 상위/사용자 설정이나 암묵적으로 상속한 환경 전체를 추적하지 않는다.
따라서 이 파일 formatter 측정을 package 전체를 분석하는 linter의 정확성·성능
보증으로 일반화하지 않는다. package 단위 cache 설계 변경은 이번 수정에 포함하지 않는다.

### 도구 버전 캐시

도구 버전은 결과 cache key의 일부이므로, 오래된 버전을 재사용하면 다른 버전의
도구가 만든 결과를 돌려주게 된다. 그래서 버전은 시간(TTL)이 아니라 **실행 파일
자체의 identity**에 묶는다 (`cache/version.go`).

- 대상: `tools.VersionProber`를 구현한 도구(`BaseTool` 기반 도구 전부)만.
  `VersionProbe()`는 `GetVersion`이 실행하는 실행 파일 이름과 순서가 있는
  version flag 목록(또는 `SetVersionCommand`로 지정한 단일 인자 목록)을 돌려준다.
  `GetVersion`을 다른 방식으로 재정의하는 도구는 `VersionProbe`도 함께 재정의해야 한다.
- flag가 모두 버전을 출력하지 않으면 `GetVersion`은 Go 실행 파일에 내장된 build
  info(`main module path`, 버전, Go release, 예: `golang.org/x/tools v0.38.0 go1.26.7`)를 쓴다. 같은 파일에서 읽으므로 stat identity 캐시와 맞물린다.
  수정된 checkout 빌드(`vcs.modified=true`)와 VCS revision 없는 `(devel)` 빌드는
  `unknown`이 되어 결과 캐시를 우회한다. 로컬 `replace`, `-tags`, vendor 차이는
  build info 문자열에 드러나지 않는다.
- opt-out: `VersionCacheable()`이 false인 도구는 캐시를 읽거나 쓰지 않고 매번
  조회한다. version 출력이 실행 파일 외의 환경·네트워크·설정에도 의존하는 native
  binary는 반드시 생성자에서 `SetVersionUncacheable()`을 호출해야 한다. 예:
  `govulncheck --version`은 PATH의 `go` toolchain(`Go: go1.x`)과 원격 취약점
  DB 갱신일(`DB updated:`)을 출력하므로 opt-out한다.
- 도구별 버전 식별:
  - `clippy`는 `cargo clippy --version`, `cargo-fmt`는 `cargo fmt --version`으로 각각
    clippy·rustfmt 버전을 얻는다. cargo 버전으로 대신하지 않으며 빈 값·다른 component
    출력·조회 실패는 결과 캐시를 우회한다.
    독립 `rustfmt`도 `rustfmt --version`으로 선택된 component를 확인한다.
    component 이름과 숫자 `MAJOR.MINOR.PATCH` token을 확인하며 stable/nightly 등의
    suffix는 허용한다. `unknown`·`(devel)`·비버전 문자열은 prefix가 맞아도 캐시하지 않는다.
  - Rust 세 adapter는 실행 파일 자체로 component 선택을 증명할 수 없어
    `VersionCacheable=false`다. `VersionProbe`는 `SetVersionCommand`를 포함한 실제
    argv를 돌려준다. task마다 직접 조회하며 같은 task의 파일들은 조회 결과를
    공유한다. cargo 파일이 그대로여도 component만 갱신되면 다음 task의 결과 key는
    달라져야 한다.
  - Rust adapter의 선택적 `GetVersionForExecution(ctx, options)`는 실제 실행과 같은
    `ProjectRoot`·환경 덮어쓰기를 적용한다. executor는 task당 한 번 이 probe를 우선하므로
    cwd별 toolchain이나 `RUSTUP_TOOLCHAIN` 선택도 결과 key에 반영된다. 이 선택적 메서드가
    없는 사용자 정의 도구는 기존 `GetVersion`·버전 캐시 경로를 유지한다. 필수 공개
    `QualityTool`·`Storage`·`VersionProber` 인터페이스는 바뀌지 않는다.
    실행 중 취소 원인은 `errors.Is`로 보존하고, 자식이 출력 pipe를 잡고 있어도
    기존 실행 경로와 같은 1초 `WaitDelay`로 취소 후 대기를 제한한다.
- key(`cache.VersionKey`, 모든 필드가 같을 때만 hit):
  `ToolName`, `ExecutablePath`(PATH 탐색 → 절대 경로 → symlink 해석 후 경로),
  `ProbeFlags`(순서 포함), `Device`, `Inode`, `Size`, `ModTimeNs`, `ChangeTimeNs`.
  ctime은 사용자가 되돌릴 수 없으므로 `cp -p`나 mtime을 복원한 제자리 덮어쓰기,
  `chmod`도 다시 조회하게 만든다. 교체 설치(rename)는 inode로, 경로 변경(PATH,
  symlink 대상 변경)은 경로로 miss가 된다.
- 다음 경우에는 캐시하지 않고 매번 조회한다. 실행 파일만으로 버전이 결정된다고
  보장할 수 없기 때문이다.
  - inode/ctime을 읽을 수 없는 플랫폼: `syscall.Stat_t` 필드가 OS마다 달라
    build tag로 나눈다(`version_stat_darwin.go`의 `Ctimespec`,
    `version_stat_linux.go`의 `Ctim`). 그 밖의 OS(Windows 등)는
    `version_stat_other.go`가 항상 조회를 선택한다. 추측한 대체 key나 TTL은 없다.
  - symlink가 **다른 이름**의 파일로 해석될 때: mise·volta shim처럼 하나의
    binary가 설정에 따라 버전을 고르는 경우다. 같은 이름의 symlink(Homebrew
    `bin/` → `Cellar/`, 대상 이름이 같은 `update-alternatives` link 등)는
    해석된 대상 경로로 key를 만들므로, 대상이 바뀌면 경로가 달라져 miss가 된다.
  - hard link가 2개 이상일 때: rustup proxy(`~/.cargo/bin/rustfmt`)처럼
    `rust-toolchain.toml`에 따라 버전이 바뀌는 경우다.
  - `#!` script일 때: asdf·pyenv shim, Python console script처럼 버전이 script
    밖의 파일에 있는 경우다.
- 저장: 결과 cache 디렉터리 아래 `versions/{sha256(tool, path, flags)}.json`에
  key와 버전을 저장한다. 같은 디렉터리의 임시 파일에 쓴 뒤 rename하므로 동시
  실행이 쓰다 만 entry를 읽지 않는다. 해석된 경로가 같은 채로 파일만 바뀌면
  같은 slot을 덮어쓴다. 경로가 바뀌면(brew·mise 업그레이드로 버전별 디렉터리가
  달라지는 경우 등) 새 slot에 기록하며, 이전 entry는 `InvalidateAll` 전까지
  남는다. 쓰는 도중 프로세스가 죽으면 `*.tmp`가 남을 수 있고, 이것도
  `InvalidateAll`이 지운다. 조회 전후 stat이 다르면(조회 도중 교체) 기록하지
  않는다. 잘못된 UTF-8 버전은 JSON이 다른 bytes로 바꾸므로 기록하지 않는다.
- 오류: entry가 없거나 읽을 수 없거나 손상됐거나 key가 다르면 다시 조회하고
  덮어쓴다. 저장 실패는 실행을 실패시키지 않는다. 조회 자체의 오류와
  `unknown`은 지금처럼 그대로 돌려주며 기록하지 않는다.
- 결과 storage의 `List`/`Size`/`CleanupCorrupted`는 `versions/`를 건너뛰므로
  버전 entry는 결과 entry 수·크기·LRU에 포함되지 않는다. 사용자 cache-clear
  (`InvalidateAll`)는 버전 entry도 지운다. 캐시가 비활성이면 매번 조회한다.
- 남는 한계: 위 판별을 통과하면서 cwd나 환경 변수로 버전을 고르는 단일 binary는
  도구가 opt-out하지 않으면 감지하지 못한다. 같은 inode를 같은 크기로 같은 coarse clock tick 안에 두 번
  덮어쓰면 mtime/ctime이 같을 수 있다(Linux는 tick이 수 ms).

## v0.1.3 → v0.2.0 마이그레이션

v0.2.0은 Go 캐시 API의 호환되지 않는 계약 변경을 포함한다. CLI 명령과
`gz-quality` 배포 실행 파일 이름은 바뀌지 않는다.

- 공개 `Storage` 인터페이스는 기존 여섯 메서드에 `Touch(key string) error`와
  `AccessedAt(key string) (time.Time, error)`를 추가한 여덟 메서드를 요구한다.
  사용자 정의 storage 구현과 mock도 두 메서드를 구현해야 한다.
- 캐시 hit는 JSON을 다시 쓰지 않는다. `CacheMetadata.LastAccessed`와
  `AccessCount`는 저장된 스냅샷이며 hit마다 갱신되는 시각·누적 횟수가 아니다.
  기존 JSON 필드 형식을 유지한다는 사실은 v0.1.3의 갱신 의미를 보장하지 않는다.
- `FilesystemStorage.AccessedAt`은 캐시 파일의 마지막 write 또는 touch 시각을
  반환한다. 최근 접근에 따른 cleanup은 이 시각을 사용한다. 사용자 정의 storage도
  write/touch 시각을 추적하고 `AccessedAt`으로 제공해야 한다.
- `CacheStats.HitCount`는 해당 manager 인스턴스 세션에서 집계한 hit 수다.
  프로세스를 넘어 저장되는 entry별 접근 횟수의 대체 지표가 아니다.

기존 metadata를 hit별 접근 추적에 사용한 호출자는 시간 조회를 `AccessedAt`으로
옮기고, 영속적인 entry별 누적 횟수가 필요하면 별도로 관리해야 한다.

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

______________________________________________________________________

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

______________________________________________________________________

## Requirements

### Functional Requirements

1. **FR1**: Cache tool execution results per file
1. **FR2**: Invalidate cache when file content changes
1. **FR3**: Invalidate cache when tool version changes
1. **FR4**: Invalidate cache when tool configuration changes
1. **FR5**: Support cache cleanup (size limits, age limits)
1. **FR6**: Provide cache statistics (hit rate, size)
1. **FR7**: Allow cache to be disabled via flag
1. **FR8**: Support cache prewarming for CI/CD

### Non-Functional Requirements

1. **NFR1**: Cache lookup must be < 1ms per file
1. **NFR2**: Cache storage must be < 100MB for typical projects
1. **NFR3**: Cache must be thread-safe (concurrent access)
1. **NFR4**: Cache corruption must not break tool execution
1. **NFR5**: Cache must be portable (macOS, Linux, Windows)

______________________________________________________________________

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
    Touch(key string) error                   // mark accessed without rewriting
    AccessedAt(key string) (time.Time, error) // last write or touch
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

______________________________________________________________________

## Cache Key Design

### Key Components

A cache entry is valid only if **ALL** of the following match:

1. **File Content**: Same file hash (SHA256)
1. **Tool Version**: Same tool version
1. **Tool Config**: Same configuration files
1. **Execution Options**: Same flags (--fix, --format-only, etc.)

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

______________________________________________________________________

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
    "last_accessed": "2025-12-02T00:00:00Z",
    "access_count": 0,
    "size_bytes": 1024
  }
}
```

A cache hit does not rewrite this file. `last_accessed` and `access_count` keep the values
written by `Set`; the filesystem backend records each hit by updating the file's modification
time (`Storage.Touch`), and size-based eviction orders entries by that time
(`Storage.AccessedAt`). Hit totals are in `CacheStats.HitCount`.

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

______________________________________________________________________

## Cache Invalidation

### Invalidation Triggers

| Trigger            | Action                           | Reason                   |
| ------------------ | -------------------------------- | ------------------------ |
| **File modified**  | Delete entries for that file     | Content changed          |
| **Tool upgraded**  | Delete entries for that tool     | Different version        |
| **Config changed** | Delete entries using that config | Different rules          |
| **Manual clear**   | Delete all entries               | User request             |
| **Age limit**      | Delete old entries               | Prevent unbounded growth |
| **Size limit**     | Delete oldest entries            | Prevent disk exhaustion  |

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

______________________________________________________________________

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

______________________________________________________________________

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

| Metric                 | Target  | Measurement               |
| ---------------------- | ------- | ------------------------- |
| **Lookup time**        | < 1ms   | Indexed SQLite query      |
| **Write time**         | < 5ms   | JSON write + DB insert    |
| **Hit rate (typical)** | > 80%   | Based on change frequency |
| **Hit rate (CI/CD)**   | > 95%   | Small PRs, cached base    |
| **Storage overhead**   | < 100MB | For 1000-file project     |
| **Memory overhead**    | < 10MB  | In-memory index only      |

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

______________________________________________________________________

## Implementation Plan

### Phase 1: Core Infrastructure ✅ COMPLETED

**Tasks:**

1. ✅ Create `cache/` package structure
1. ✅ Implement `CacheKey` generation
1. ✅ Implement `FilesystemStorage`
1. ✅ Implement `CacheManager` (basic operations)
1. ✅ Add unit tests (>80% coverage)

**Deliverables:**

- ✅ `cache/key.go` - Cache key generation with SHA256 hashing
- ✅ `cache/storage.go` - Filesystem storage with atomic writes
- ✅ `cache/manager.go` - Cache manager with hit/miss tracking
- ✅ `cache/types.go` - Type definitions
- ✅ `cache/*_test.go` - Comprehensive test suite

### Phase 2: Executor Integration ✅ COMPLETED

**Tasks:**

1. ✅ Add cache parameter to `ParallelExecutor`
1. ✅ Implement cache lookup before tool execution
1. ✅ Implement cache storage after successful execution
1. ✅ Add integration tests
1. ✅ Add file-specific result filtering for multi-file caching

**Deliverables:**

- ✅ Modified `executor/runner.go` with `NewParallelExecutorWithCache()`
- ✅ `executor/cache_integration_test.go` - Integration tests
- ✅ `filterIssuesByFile()` - Per-file issue extraction

### Phase 3: CLI Integration ✅ COMPLETED

**Tasks:**

1. ✅ Add `--cache` and `--no-cache` flags to commands
1. ✅ Implement `cache clear` command
1. ✅ Implement `cache stats` command
1. ✅ Add configuration file support (`CacheConfig`)
1. ✅ Cache state display in results

**Deliverables:**

- ✅ Modified `quality.go` with cache commands
- ✅ Modified `config/config.go` with `CacheConfig` struct
- ✅ CLI help text updates
- ✅ Cache hit indicator in results output

### Phase 4: Optimization & Production 🔄 IN PROGRESS

**Tasks:**

1. ⏳ Add SQLite index for fast lookups (using filesystem for now)
1. ✅ Implement LRU eviction policy (`Cleanup()` method)
1. ⏳ Add cache prewarming for CI/CD
1. ✅ Performance testing with benchmarks
1. ✅ Edge case handling (concurrent access via mutex)

**Deliverables:**

- ✅ `executor/bench_test.go` - Cache performance benchmarks
- ✅ LRU-based cleanup in `manager.go`
- ⏳ SQLite integration (optional enhancement)
- ⏳ CI/CD prewarming documentation

**Benchmark Results (Apple M1 Ultra):**

| Scenario            | Time      | Notes                   |
| ------------------- | --------- | ----------------------- |
| Cache Hit           | ~335µs/op | Single file, warm cache |
| Cache Miss          | ~350µs/op | Single file, cold cache |
| Multi-file All Hit  | ~3.1ms/op | 10 files, all cached    |
| No Cache (baseline) | ~5µs/op   | Mock tool execution     |

______________________________________________________________________

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

______________________________________________________________________

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

______________________________________________________________________

## Future Enhancements

### Phase 5: Advanced Features (Future)

1. **Distributed Cache**: Share cache across team (Redis, S3)
1. **Cache Compression**: Reduce storage overhead
1. **Smart Prewarming**: Predict cache needs
1. **Cache Analytics**: Dashboard for cache performance
1. **Incremental Results**: Cache partial results (per-function)

______________________________________________________________________

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

______________________________________________________________________

**Document Version**: 1.1
**Last Updated**: 2025-12-02
**Status**: Implementation Complete (Phase 1-3), Optimization In Progress (Phase 4)
**Authors**: Claude (claude-sonnet-4-5, claude-opus-4-5), archmagece

### Implementation Summary

| Phase                         | Status         | Commits              |
| ----------------------------- | -------------- | -------------------- |
| Phase 1: Core Infrastructure  | ✅ Complete    | `b799237`, `a5c6680` |
| Phase 2: Executor Integration | ✅ Complete    | `208c9f9`, `afd9463` |
| Phase 3: CLI Integration      | ✅ Complete    | `f532bc1`, `9908056` |
| Phase 4: Optimization         | 🔄 In Progress | `563ffe8`            |

**Total Lines Added**: ~2000+ lines of production code and tests
