# gzh-cli-quality 시스템 아키텍처

## 1. 개요

gzh-cli-quality는 **멀티 언어 코드 품질 도구 오케스트레이터**로, 다양한 포매터와 린터를 통합 실행하는 CLI 애플리케이션입니다.

### 1.1 설계 원칙
- **단일 책임**: 각 컴포넌트는 하나의 역할만 수행
- **인터페이스 분리**: 도구 확장을 위한 명확한 계약
- **의존성 역전**: 구체 구현이 아닌 인터페이스에 의존
- **병렬 처리**: Worker Pool 패턴으로 효율적 실행

---

## 2. 시스템 개요

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLI Layer (Cobra)                        │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐   │
│  │   run   │ │  check  │ │  init   │ │  tool   │ │ install │   │
│  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘   │
└───────┼──────────┼──────────┼──────────┼──────────┼────────────┘
        │          │          │          │          │
        └──────────┴──────────┴──────────┴──────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      QualityManager                              │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Orchestration                         │    │
│  │  - 플래그 파싱 및 옵션 구성                              │    │
│  │  - 실행 계획 생성 및 조정                                │    │
│  │  - 결과 집계 및 리포트 출력                              │    │
│  └─────────────────────────────────────────────────────────┘    │
└───────┬─────────────┬─────────────┬─────────────┬───────────────┘
        │             │             │             │
        ▼             ▼             ▼             ▼
┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐
│  Detector │ │ ToolReg   │ │ Executor  │ │  Report   │
│  Package  │ │ Package   │ │ Package   │ │ Package   │
├───────────┤ ├───────────┤ ├───────────┤ ├───────────┤
│ • Language│ │ • Registry│ │ • Parallel│ │ • JSON    │
│   Detect  │ │ • Tool    │ │   Exec    │ │ • HTML    │
│ • Tool    │ │   Lookup  │ │ • Worker  │ │ • Markdown│
│   Detect  │ │           │ │   Pool    │ │           │
└───────────┘ └─────┬─────┘ └───────────┘ └───────────┘
                    │
                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Tools Package                               │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                  QualityTool Interface                   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│    ┌─────────────────────────┼─────────────────────────┐        │
│    │           │             │             │           │        │
│    ▼           ▼             ▼             ▼           ▼        │
│ ┌──────┐  ┌──────┐      ┌──────┐      ┌──────┐   ┌──────┐      │
│ │ Go   │  │Python│      │ JS/TS│      │ Rust │   │ Base │      │
│ │Tools │  │Tools │      │Tools │      │Tools │   │ Tool │      │
│ └──────┘  └──────┘      └──────┘      └──────┘   └──────┘      │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. 컴포넌트 상세

### 3.1 QualityManager (오케스트레이션)

**위치**: `quality.go`

**책임**:
- CLI 명령어 정의 및 플래그 처리
- 실행 흐름 조정
- 결과 집계 및 출력

**주요 구조체**:
```go
type QualityManager struct {
    registry tools.ToolRegistry       // 도구 레지스트리
    analyzer *detector.ProjectAnalyzer // 프로젝트 분석기
    executor *executor.ParallelExecutor // 병렬 실행기
    planner  *executor.ExecutionPlanner // 실행 계획 생성기
}
```

**실행 흐름**:
```
1. 플래그 파싱 → PlanOptions 생성
2. 프로젝트 분석 (언어, 파일 감지)
3. 실행 계획 생성 (ExecutionPlan)
4. 병렬 실행 (ParallelExecutor)
5. 결과 집계 → 리포트 생성
```

---

### 3.2 Tools Package (도구 시스템)

**위치**: `tools/`

#### 3.2.1 인터페이스 정의 (`interface.go`)

```go
// ToolType: 도구 타입 분류
type ToolType int
const (
    FORMAT ToolType = iota  // 포매터
    LINT                    // 린터
    BOTH                    // 포매터+린터
)

// QualityTool: 품질 도구 계약
type QualityTool interface {
    Name() string                    // 도구 이름
    Language() string                // 대상 언어
    Type() ToolType                  // 도구 타입
    IsAvailable() bool               // 설치 확인
    Install() error                  // 도구 설치
    GetVersion() (string, error)     // 버전 조회
    Upgrade() error                  // 업그레이드
    FindConfigFiles(root string) []string  // 설정 파일 탐색
    Execute(ctx, files, options) (*Result, error)  // 실행
}

// ExecuteOptions: 실행 옵션
type ExecuteOptions struct {
    ProjectRoot string
    ConfigFile  string
    Fix         bool
    FormatOnly  bool
    LintOnly    bool
    ExtraArgs   []string
    Env         map[string]string
}

// Result: 실행 결과
type Result struct {
    Tool           string
    Language       string
    Success        bool
    Error          error
    FilesProcessed int
    Duration       string
    Issues         []Issue
    Output         string
}

// Issue: 코드 이슈
type Issue struct {
    File       string
    Line       int
    Column     int
    Severity   string  // error, warning, info
    Rule       string
    Message    string
    Suggestion string
}
```

#### 3.2.2 도구 레지스트리 (`registry.go`)

```go
type ToolRegistry interface {
    Register(tool QualityTool)
    GetTools() []QualityTool
    GetToolsByLanguage(language string) []QualityTool
    GetToolsByType(toolType ToolType) []QualityTool
    FindTool(name string) QualityTool
}
```

**구현 특성**:
- Thread-safe (sync.RWMutex)
- 이름 기반 조회 (O(1))
- 언어/타입별 필터링

#### 3.2.3 BaseTool (`base.go`)

공통 기능을 제공하는 기본 구현체:

```go
type BaseTool struct {
    name       string
    language   string
    toolType   ToolType
    extensions []string
    // ...
}

// 공통 메서드
func (b *BaseTool) Name() string
func (b *BaseTool) Language() string
func (b *BaseTool) Type() ToolType
func (b *BaseTool) IsAvailable() bool  // PATH에서 실행 파일 확인
func (b *BaseTool) GetVersion() (string, error)  // 버전 명령 실행
```

#### 3.2.4 언어별 도구 구현

| 파일 | 도구 | 특이사항 |
|------|------|----------|
| `go_tools.go` | gofumpt, goimports, golangci-lint | go.mod 감지 |
| `python_tools.go` | black, ruff, pylint | venv/uv 지원 |
| `javascript_tools.go` | prettier, eslint, tsc | node_modules 탐색 |
| `rust_tools.go` | rustfmt, clippy, cargo-fmt | Cargo.toml 감지 |

---

### 3.3 Executor Package (실행 엔진)

**위치**: `executor/`

#### 3.3.1 ParallelExecutor (`runner.go`)

**Worker Pool 패턴**:
```
┌─────────────────────────────────────────────────────────────┐
│                     ParallelExecutor                         │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                    Task Queue                        │    │
│  │  [gofumpt] [black] [prettier] [eslint] [golangci]   │    │
│  └────────────────────────┬────────────────────────────┘    │
│                           │                                  │
│           ┌───────────────┼───────────────┐                 │
│           ▼               ▼               ▼                 │
│      ┌─────────┐    ┌─────────┐    ┌─────────┐             │
│      │ Worker  │    │ Worker  │    │ Worker  │             │
│      │   #1    │    │   #2    │    │   #3    │             │
│      └────┬────┘    └────┬────┘    └────┬────┘             │
│           │              │              │                   │
│           └──────────────┼──────────────┘                   │
│                          ▼                                  │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                  Result Channel                      │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

**구현**:
```go
type ParallelExecutor struct {
    maxWorkers int           // 최대 워커 수
    timeout    time.Duration // 전체 타임아웃
}

func (e *ParallelExecutor) ExecuteParallel(
    ctx context.Context,
    plan *ExecutionPlan,
    workers int,
) ([]*Result, error)
```

**특성**:
- Context 기반 취소 지원
- Priority 기반 태스크 정렬
- 개별 도구 실패는 전체 실행을 중단하지 않음
- 결과 수집 후 일괄 반환

#### 3.3.2 ExecutionPlanner

```go
type ExecutionPlanner struct {
    analyzer ProjectAnalyzer
}

type PlanOptions struct {
    Files      []string  // 대상 파일
    Fix        bool      // 자동 수정
    FormatOnly bool      // 포매팅만
    LintOnly   bool      // 린팅만
    ExtraArgs  []string  // 추가 인수
    Since      string    // Git 커밋 기준
    Staged     bool      // Staged 파일만
    Changed    bool      // 변경 파일만
}

type ExecutionPlan struct {
    Tasks             []Task
    TotalFiles        int
    EstimatedDuration string
}

type Task struct {
    Tool     QualityTool
    Files    []string
    Options  ExecuteOptions
    Priority int  // 높을수록 먼저 실행
}
```

---

### 3.4 Detector Package (감지 시스템)

**위치**: `detector/`

#### 3.4.1 FileTypeDetector (`language.go`)

**언어 감지 규칙**:
```go
type LanguageRule struct {
    Name       string   // 언어 이름
    Extensions []string // 파일 확장자
    Indicators []string // 프로젝트 지표 (go.mod, package.json 등)
    MinFiles   int      // 최소 파일 수
    Weight     float64  // 신뢰도 가중치
}
```

**감지 결과**:
```go
type LanguageInfo struct {
    Name       string            // Go, Python, JavaScript 등
    Extensions []string          // [.go, .mod]
    Files      []string          // 감지된 파일 목록
    Indicators []string          // 발견된 지표
    Confidence float64           // 신뢰도 (0.0 - 1.0)
}
```

#### 3.4.2 SystemToolDetector (`tools.go`)

**도구 감지**:
- PATH에서 실행 파일 탐색
- 버전 명령 실행으로 설치 확인
- 설정 파일 위치 탐색

---

### 3.5 Config Package (설정 시스템)

**위치**: `config/`

#### 3.5.1 설정 구조 (`config.go`)

```go
type Config struct {
    DefaultWorkers int                       // 기본 워커 수
    Timeout        string                    // 타임아웃
    Tools          map[string]ToolConfig     // 도구별 설정
    Languages      map[string]LanguageConfig // 언어별 설정
    Exclude        []string                  // 제외 패턴
    Include        []string                  // 포함 패턴
}

type ToolConfig struct {
    Enabled    bool              // 활성화 여부
    ConfigFile string            // 설정 파일 경로
    Args       []string          // 추가 인수
    Env        map[string]string // 환경 변수
    Priority   int               // 실행 우선순위
}

type LanguageConfig struct {
    Enabled        bool     // 활성화 여부
    PreferredTools []string // 선호 도구
    Extensions     []string // 파일 확장자
}
```

#### 3.5.2 설정 로드 순서

```
1. 기본값 (DefaultConfig)
2. 사용자 전역 설정 (~/.config/gzq/config.yml)
3. 프로젝트 설정 (.gzquality.yml)
4. CLI 플래그 (최우선)
```

---

### 3.6 Report Package (리포트 시스템)

**위치**: `report/`

#### 3.6.1 리포트 구조 (`generator.go`)

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

#### 3.6.2 출력 형식

| 형식 | 용도 | 메서드 |
|------|------|--------|
| JSON | CI/CD 통합, 자동화 | `WriteJSON()` |
| HTML | 시각화 리포트 | `WriteHTML()` |
| Markdown | PR 코멘트, 문서화 | `WriteMarkdown()` |

---

## 4. 데이터 플로우

### 4.1 `gzq run` 실행 흐름

```
┌──────────────────────────────────────────────────────────────┐
│ 1. CLI 명령 파싱                                              │
│    gzq run --staged --fix --workers 4                         │
└──────────────────────────┬───────────────────────────────────┘
                           ▼
┌──────────────────────────────────────────────────────────────┐
│ 2. PlanOptions 생성                                           │
│    {Staged: true, Fix: true, Workers: 4}                      │
└──────────────────────────┬───────────────────────────────────┘
                           ▼
┌──────────────────────────────────────────────────────────────┐
│ 3. 프로젝트 분석                                              │
│    FileTypeDetector → Languages: [Go, Python]                 │
│    SystemToolDetector → Tools: [gofumpt, black, ruff]         │
│    Git → StagedFiles: [main.go, utils.py]                     │
└──────────────────────────┬───────────────────────────────────┘
                           ▼
┌──────────────────────────────────────────────────────────────┐
│ 4. 실행 계획 생성                                             │
│    ExecutionPlan {                                            │
│      Tasks: [                                                 │
│        {Tool: gofumpt, Files: [main.go], Priority: 10},       │
│        {Tool: black, Files: [utils.py], Priority: 10},        │
│        {Tool: ruff, Files: [utils.py], Priority: 7}           │
│      ],                                                       │
│      TotalFiles: 2                                            │
│    }                                                          │
└──────────────────────────┬───────────────────────────────────┘
                           ▼
┌──────────────────────────────────────────────────────────────┐
│ 5. 병렬 실행                                                  │
│    Worker Pool (4 workers)                                    │
│    ┌─────────────┬─────────────┬─────────────┐               │
│    │  gofumpt    │    black    │    ruff     │               │
│    │  (0.2s)     │   (0.3s)    │   (0.5s)    │               │
│    └─────────────┴─────────────┴─────────────┘               │
└──────────────────────────┬───────────────────────────────────┘
                           ▼
┌──────────────────────────────────────────────────────────────┐
│ 6. 결과 집계                                                  │
│    [Result{gofumpt, Success}, Result{black, Success},         │
│     Result{ruff, Issues: 2}]                                  │
└──────────────────────────┬───────────────────────────────────┘
                           ▼
┌──────────────────────────────────────────────────────────────┐
│ 7. 리포트 출력                                                │
│    ✅ gofumpt (0.2s) - 1 file                                 │
│    ✅ black (0.3s) - 1 file                                   │
│    ⚠️ ruff (0.5s) - 2 issues                                  │
│    ✨ 완료: 0.5s, 2 files, 2 issues                           │
└──────────────────────────────────────────────────────────────┘
```

---

## 5. 디렉토리 구조

```
gzh-cli-quality/
├── main.go                 # 진입점
├── quality.go              # QualityManager, 명령어 정의
├── register.go             # 도구 등록
│
├── config/
│   └── config.go           # 설정 로드/파싱
│
├── detector/
│   ├── language.go         # 언어 감지
│   └── tools.go            # 도구 감지
│
├── executor/
│   └── runner.go           # 병렬 실행 엔진
│
├── git/
│   └── utils.go            # Git 유틸리티
│
├── report/
│   └── generator.go        # 리포트 생성
│
├── tools/
│   ├── interface.go        # 인터페이스 정의
│   ├── base.go             # BaseTool 구현
│   ├── registry.go         # ToolRegistry 구현
│   ├── go_tools.go         # Go 도구들
│   ├── python_tools.go     # Python 도구들
│   ├── javascript_tools.go # JS/TS 도구들
│   └── rust_tools.go       # Rust 도구들
│
└── docs/
    ├── API.md              # API 레퍼런스
    └── ADDING_TOOLS.md     # 도구 추가 가이드
```

---

## 6. 설계 결정 사항

### D-1: Worker Pool 패턴 선택

**결정**: 고정 크기 Worker Pool + 채널 기반 태스크 분배

**근거**:
- 도구 실행은 I/O 바운드 작업
- 워커 수 제한으로 리소스 관리 용이
- Context 취소를 통한 graceful shutdown

**대안 검토**:
- Goroutine per Task: 리소스 제어 어려움
- Fan-out/Fan-in: 구현 복잡성 증가

---

### D-2: 도구 우선순위 시스템

**결정**: 정수형 Priority 필드 (높을수록 먼저 실행)

**기본 우선순위**:
| 타입 | 기본 Priority | 근거 |
|------|--------------|------|
| 포매터 | 10 | 포매팅 먼저 (린터가 포매팅된 코드 검사) |
| 포매터+린터 | 7 | 포매팅 후, 린터 전 |
| 린터 | 5 | 마지막 실행 |
| 타입체커 | 3 | 가장 느림, 마지막 |

---

### D-3: Git 통합 방식

**결정**: Git CLI 직접 호출

**근거**:
- go-git 라이브러리 의존성 최소화
- 모든 Git 기능 완전 지원
- PATH에 git 있으면 동작

**명령어**:
```bash
git diff --name-only --staged        # --staged
git diff --name-only                 # --changed (modified)
git ls-files --others --exclude-standard  # --changed (untracked)
git diff --name-only <ref>...HEAD   # --since
```

---

### D-4: 오류 처리 전략

**결정**: 개별 도구 실패는 전체 실행 중단 안함

**근거**:
- 부분 결과라도 사용자에게 유용
- CI/CD에서 모든 도구 결과 수집 필요
- 실패한 도구 정보는 결과에 포함

**구현**:
```go
for result := range resultChan {
    results = append(results, result)  // 에러 포함해도 수집
}
// 최종 반환시 부분 실패 표시
```

---

## 7. 확장 가이드

### 7.1 새 도구 추가

1. `QualityTool` 인터페이스 구현
2. `BaseTool` 임베딩으로 공통 기능 재사용
3. `registerAllTools()`에서 레지스트리 등록

상세 가이드: [docs/ADDING_TOOLS.md](docs/ADDING_TOOLS.md)

### 7.2 새 리포트 형식 추가

1. `ReportGenerator`에 `WriteXXX()` 메서드 추가
2. CLI에 `--report xxx` 옵션 추가
3. 템플릿 기반 생성 권장

---

*최종 수정: 2025-11-27*
*참조 문서: [PRD.md](./PRD.md), [REQUIREMENTS.md](./REQUIREMENTS.md)*
