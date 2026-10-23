# gzh-cli-quality API 레퍼런스

## 1. CLI 명령어 레퍼런스

### 1.1 gz-quality run

모든 포매팅 및 린팅 도구를 실행합니다.

```bash
gz-quality run [flags]
```

**플래그**:

| 플래그          | 단축 | 타입     | 기본값 | 설명                               |
| --------------- | ---- | -------- | ------ | ---------------------------------- |
| `--files`       | `-f` | []string | []     | 특정 파일들만 처리                 |
| `--fix`         | `-x` | bool     | false  | 자동 수정 적용                     |
| `--format-only` | -    | bool     | false  | 포매팅만 실행                      |
| `--lint-only`   | -    | bool     | false  | 린팅만 실행                        |
| `--workers`     | `-w` | int      | CPU 수 | 병렬 워커 수                       |
| `--extra-args`  | -    | []string | []     | 도구에 전달할 추가 인수            |
| `--dry-run`     | -    | bool     | false  | 실행하지 않고 계획만 표시          |
| `--verbose`     | `-v` | bool     | false  | 상세 출력                          |
| `--report`      | -    | string   | ""     | 리포트 형식 (json, html, markdown) |
| `--output`      | -    | string   | ""     | 리포트 출력 경로                   |
| `--staged`      | -    | bool     | false  | Git staged 파일만                  |
| `--changed`     | -    | bool     | false  | 변경된 파일만                      |
| `--since`       | -    | string   | ""     | 특정 커밋 이후 파일                |

**예시**:

```bash
gz-quality run --staged --fix
gz-quality run --since main --report json --output report.json
gz-quality run --format-only --workers 8
```

______________________________________________________________________

### 1.2 gz-quality check

린팅만 실행합니다 (파일 수정 없음).

```bash
gz-quality check [flags]
```

**플래그**: `gz-quality run`과 동일 (단, `--fix` 무시됨)

**예시**:

```bash
gz-quality check --staged
gz-quality check --since HEAD~5
```

______________________________________________________________________

### 1.3 gz-quality init

프로젝트 설정 파일(.gzquality.yml)을 생성합니다.

```bash
gz-quality init [flags]
```

**플래그**:

| 플래그     | 타입   | 기본값           | 설명               |
| ---------- | ------ | ---------------- | ------------------ |
| `--force`  | bool   | false            | 기존 파일 덮어쓰기 |
| `--output` | string | ".gzquality.yml" | 출력 파일 경로     |

**예시**:

```bash
gz-quality init
gz-quality init --force
gz-quality init --output custom-quality.yml
```

______________________________________________________________________

### 1.4 gz-quality analyze

프로젝트를 분석하고 권장 도구를 표시합니다.

```bash
gz-quality analyze [flags]
```

**플래그**:

| 플래그      | 타입 | 기본값 | 설명           |
| ----------- | ---- | ------ | -------------- |
| `--verbose` | bool | false  | 상세 분석 결과 |
| `--json`    | bool | false  | JSON 형식 출력 |

**출력 예시**:

```
📊 프로젝트 분석 결과

감지된 언어:
  ✓ Go (15 files)
  ✓ Python (8 files)

권장 도구:
  Go:
    ✓ gofumpt (설치됨, v0.9.1)
    ✓ goimports (설치됨)
    ✓ golangci-lint (설치됨, v1.55.2)
  Python:
    ✓ black (설치됨, v24.1.0)
    ✓ ruff (설치됨, v0.1.14)
    ✗ pylint (미설치)
```

______________________________________________________________________

### 1.5 gz-quality tool

특정 도구를 직접 실행합니다.

```bash
gz-quality tool <tool-name> [flags]
```

**지원 도구**: gofumpt, goimports, golangci-lint, black, ruff, pylint, prettier, eslint, tsc, rustfmt, cargo-fmt, clippy

**플래그**: `gz-quality run`과 동일한 Git/실행 플래그 지원

**예시**:

```bash
gz-quality tool ruff --staged --fix
gz-quality tool golangci-lint --since main
gz-quality tool prettier --files "src/**/*.ts"
```

______________________________________________________________________

### 1.6 gz-quality install

품질 도구를 설치합니다.

```bash
gz-quality install [tool-name] [flags]
```

**플래그**:

| 플래그       | 타입   | 기본값 | 설명                  |
| ------------ | ------ | ------ | --------------------- |
| `--all`      | bool   | false  | 모든 도구 설치        |
| `--language` | string | ""     | 특정 언어 도구만 설치 |

**예시**:

```bash
gz-quality install                    # 프로젝트에 필요한 도구
gz-quality install golangci-lint      # 특정 도구
gz-quality install --language Python  # Python 도구만
gz-quality install --all              # 모든 지원 도구
```

______________________________________________________________________

### 1.7 gz-quality upgrade

도구를 최신 버전으로 업그레이드합니다.

```bash
gz-quality upgrade [tool-name] [flags]
```

**예시**:

```bash
gz-quality upgrade                # 모든 도구
gz-quality upgrade golangci-lint  # 특정 도구
```

______________________________________________________________________

### 1.8 gz-quality version

설치된 도구 버전을 표시합니다.

```bash
gz-quality version [flags]
```

**플래그**:

| 플래그   | 타입 | 기본값 | 설명           |
| -------- | ---- | ------ | -------------- |
| `--json` | bool | false  | JSON 형식 출력 |

**출력 예시**:

```
gzh-cli-quality v1.0.0

설치된 도구:
  gofumpt       v0.9.1      /home/user/go/bin/gofumpt
  goimports     v0.16.1     /home/user/go/bin/goimports
  golangci-lint v1.55.2     /home/user/go/bin/golangci-lint
  black         24.1.0      /home/user/.local/bin/black
  ruff          0.1.14      /home/user/.local/bin/ruff
```

______________________________________________________________________

### 1.9 gz-quality list

사용 가능한 도구 목록을 표시합니다.

```bash
gz-quality list [flags]
```

**플래그**:

| 플래그        | 타입   | 기본값 | 설명                          |
| ------------- | ------ | ------ | ----------------------------- |
| `--language`  | string | ""     | 특정 언어만 필터              |
| `--type`      | string | ""     | 도구 타입 (formatter, linter) |
| `--installed` | bool   | false  | 설치된 도구만                 |

**출력 예시**:

```
사용 가능한 도구:

Go:
  ✓ gofumpt       formatter   Go 코드 포매터
  ✓ goimports     formatter   import 정리
  ✓ golangci-lint linter      통합 린터

Python:
  ✓ black         formatter   Python 포매터
  ✓ ruff          both        빠른 린터/포매터
  ✗ pylint        linter      정적 분석기
```

______________________________________________________________________

## 2. Go 패키지 API

### 2.1 tools 패키지

#### QualityTool 인터페이스

```go
package tools

type QualityTool interface {
    // 도구 정보
    Name() string                    // 도구 이름 (gofumpt, ruff 등)
    Language() string                // 대상 언어 (Go, Python 등)
    Type() ToolType                  // FORMAT, LINT, BOTH

    // 상태 확인
    IsAvailable() bool               // 설치 여부
    GetVersion() (string, error)     // 버전 문자열

    // 관리
    Install() error                  // 도구 설치
    Upgrade() error                  // 최신 버전으로 업그레이드
    FindConfigFiles(root string) []string  // 설정 파일 탐색

    // 실행
    Execute(ctx context.Context, files []string, options ExecuteOptions) (*Result, error)
}
```

#### ToolType

```go
type ToolType int

const (
    FORMAT ToolType = iota  // 포매터
    LINT                    // 린터
    BOTH                    // 포매터+린터
)

func (t ToolType) String() string  // "formatter", "linter", "formatter+linter"
```

#### ExecuteOptions

```go
type ExecuteOptions struct {
    ProjectRoot string            // 프로젝트 루트 경로
    ConfigFile  string            // 도구 설정 파일
    Fix         bool              // 자동 수정 여부
    FormatOnly  bool              // 포매팅만 (BOTH 타입용)
    LintOnly    bool              // 린팅만 (BOTH 타입용)
    ExtraArgs   []string          // 추가 CLI 인수
    Env         map[string]string // 환경 변수
}
```

#### Result

```go
type Result struct {
    Tool           string  // 도구 이름
    Language       string  // 언어
    Success        bool    // 성공 여부
    Error          error   // 에러 (있는 경우)
    FilesProcessed int     // 처리된 파일 수
    Duration       string  // 실행 시간 (예: "1.5s")
    Issues         []Issue // 발견된 이슈
    Output         string  // 원본 출력
}
```

#### Issue

```go
type Issue struct {
    File       string  // 파일 경로
    Line       int     // 라인 번호 (1-based)
    Column     int     // 컬럼 번호 (1-based)
    Severity   string  // "error", "warning", "info"
    Rule       string  // 규칙 이름
    Message    string  // 설명
    Suggestion string  // 수정 제안 (선택)
}
```

______________________________________________________________________

#### ToolRegistry 인터페이스

```go
type ToolRegistry interface {
    // 등록
    Register(tool QualityTool)

    // 조회
    GetTools() []QualityTool
    GetToolsByLanguage(language string) []QualityTool
    GetToolsByType(toolType ToolType) []QualityTool
    FindTool(name string) QualityTool
}
```

**사용 예시**:

```go
registry := tools.NewRegistry()
registry.Register(NewGofumptTool())

// 언어별 조회
goTools := registry.GetToolsByLanguage("Go")

// 이름으로 찾기
ruff := registry.FindTool("ruff")
if ruff != nil && ruff.IsAvailable() {
    result, err := ruff.Execute(ctx, files, options)
}
```

______________________________________________________________________

### 2.2 executor 패키지

#### ParallelExecutor

```go
package executor

type ParallelExecutor struct {
    maxWorkers int
    timeout    time.Duration
}

func NewParallelExecutor(maxWorkers int, timeout time.Duration) *ParallelExecutor

// 순차 실행
func (e *ParallelExecutor) Execute(ctx context.Context, plan *tools.ExecutionPlan) ([]*tools.Result, error)

// 병렬 실행
func (e *ParallelExecutor) ExecuteParallel(ctx context.Context, plan *tools.ExecutionPlan, workers int) ([]*tools.Result, error)
```

#### ExecutionPlanner

```go
type ExecutionPlanner struct {
    analyzer ProjectAnalyzer
}

func NewExecutionPlanner(analyzer ProjectAnalyzer) *ExecutionPlanner

func (p *ExecutionPlanner) CreatePlan(projectRoot string, registry tools.ToolRegistry, options PlanOptions) (*tools.ExecutionPlan, error)
```

#### PlanOptions

```go
type PlanOptions struct {
    Files      []string  // 대상 파일 (빈 배열: 전체)
    Fix        bool      // 자동 수정
    FormatOnly bool      // 포매팅만
    LintOnly   bool      // 린팅만
    ExtraArgs  []string  // 추가 인수
    Since      string    // Git 커밋 레퍼런스
    Staged     bool      // staged 파일만
    Changed    bool      // 변경 파일만
}
```

**사용 예시**:

```go
executor := executor.NewParallelExecutor(4, 10*time.Minute)
planner := executor.NewExecutionPlanner(analyzer)

plan, err := planner.CreatePlan(projectRoot, registry, PlanOptions{
    Staged: true,
    Fix:    true,
})

results, err := executor.ExecuteParallel(ctx, plan, 4)
```

______________________________________________________________________

### 2.3 config 패키지

#### Config

```go
package config

type Config struct {
    DefaultWorkers int
    Timeout        string
    Tools          map[string]ToolConfig
    Languages      map[string]LanguageConfig
    Exclude        []string
    Include        []string
}

func DefaultConfig() *Config
func LoadConfig(path string) (*Config, error)
func FindConfigFile() string
```

#### ToolConfig

```go
type ToolConfig struct {
    Enabled    bool
    ConfigFile string
    Args       []string
    Env        map[string]string
    Priority   int
}
```

#### LanguageConfig

```go
type LanguageConfig struct {
    Enabled        bool
    PreferredTools []string
    Extensions     []string
}
```

______________________________________________________________________

### 2.4 report 패키지

#### ReportGenerator

```go
package report

type ReportGenerator struct {
    projectRoot string
}

func NewReportGenerator(projectRoot string) *ReportGenerator

func (g *ReportGenerator) GenerateReport(results []*tools.Result, duration time.Duration, totalFiles int) *Report

func (g *ReportGenerator) WriteJSON(report *Report, path string) error
func (g *ReportGenerator) WriteHTML(report *Report, path string) error
func (g *ReportGenerator) WriteMarkdown(report *Report, path string) error
```

#### Report

```go
type Report struct {
    Timestamp    time.Time
    ProjectRoot  string
    TotalFiles   int
    Duration     time.Duration
    Summary      Summary
    ToolResults  []ToolResult
    IssuesByFile map[string][]Issue
}

type Summary struct {
    TotalTools      int
    SuccessfulTools int
    FailedTools     int
    TotalIssues     int
    ErrorIssues     int
    WarningIssues   int
    InfoIssues      int
    FilesWithIssues int
}
```

______________________________________________________________________

## 3. 설정 스키마

### .gzquality.yml

```yaml
# 전역 설정
default_workers: 4          # 병렬 워커 수 (기본: CPU 수)
timeout: "10m"              # 전체 타임아웃

# 도구별 설정
tools:
  <tool-name>:              # gofumpt, ruff, prettier 등
    enabled: true           # 활성화 (기본: true)
    config_file: ""         # 커스텀 설정 파일 경로
    args: []                # 추가 CLI 인수
    env: {}                 # 환경 변수
    priority: 10            # 실행 순서 (높을수록 먼저)

# 언어별 설정
languages:
  <language>:               # Go, Python, JavaScript 등
    enabled: true           # 언어 처리 여부
    preferred_tools: []     # 사용할 도구 목록 (순서대로)
    extensions: []          # 파일 확장자

# 파일 필터
exclude:                    # 제외 패턴 (glob)
  - "node_modules/**"
  - "vendor/**"
  - ".git/**"

include: []                 # 포함 패턴 (exclude보다 우선)
```

### 기본값

```yaml
default_workers: 4
timeout: "10m"

tools:
  gofumpt:      {enabled: true, priority: 10}
  goimports:    {enabled: true, priority: 9}
  golangci-lint: {enabled: true, priority: 5}
  black:        {enabled: true, priority: 10}
  ruff:         {enabled: true, priority: 7}
  pylint:       {enabled: false, priority: 5}
  prettier:     {enabled: true, priority: 10}
  eslint:       {enabled: true, priority: 5}
  tsc:          {enabled: true, priority: 3}
  rustfmt:      {enabled: true, priority: 10}
  clippy:       {enabled: true, priority: 5}

languages:
  Go:
    enabled: true
    preferred_tools: [gofumpt, goimports, golangci-lint]
    extensions: [.go]
  Python:
    enabled: true
    preferred_tools: [black, ruff]
    extensions: [.py, .pyi]
  JavaScript:
    enabled: true
    preferred_tools: [prettier, eslint]
    extensions: [.js, .jsx]
  TypeScript:
    enabled: true
    preferred_tools: [prettier, eslint, tsc]
    extensions: [.ts, .tsx]
  Rust:
    enabled: true
    preferred_tools: [rustfmt, clippy]
    extensions: [.rs]

exclude:
  - node_modules/**
  - vendor/**
  - .git/**
  - dist/**
  - build/**
```

______________________________________________________________________

## 4. 종료 코드

| 코드 | 의미                           |
| ---- | ------------------------------ |
| 0    | 성공 (이슈 없음)               |
| 1    | 이슈 발견 또는 부분 실패       |
| 2    | 실행 오류 (설정, 도구 문제 등) |

______________________________________________________________________

*최종 수정: 2025-11-27*
*참조: [ARCHITECTURE.md](./ARCHITECTURE.md), [REQUIREMENTS.md](./REQUIREMENTS.md)*
