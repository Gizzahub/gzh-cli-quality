# 새로운 도구 추가 가이드

gzh-cli-quality에 새로운 품질 도구를 추가하는 방법을 설명합니다.

## 1. 개요

### 1.1 도구 추가 단계

1. `QualityTool` 인터페이스 구현
2. `BaseTool` 임베딩으로 공통 기능 재사용
3. 레지스트리에 등록
4. 테스트 작성

### 1.2 파일 구조

```
tools/
├── interface.go        # 인터페이스 정의 (수정 불필요)
├── base.go             # BaseTool (수정 불필요)
├── registry.go         # 레지스트리 (수정 불필요)
├── go_tools.go         # Go 도구 (새 Go 도구 추가)
├── python_tools.go     # Python 도구
├── javascript_tools.go # JS/TS 도구
├── rust_tools.go       # Rust 도구
└── <language>_tools.go # 새 언어 추가시
```

---

## 2. QualityTool 인터페이스

모든 품질 도구는 다음 인터페이스를 구현해야 합니다:

```go
type QualityTool interface {
    Name() string                    // 도구 이름
    Language() string                // 대상 언어
    Type() ToolType                  // FORMAT, LINT, BOTH
    IsAvailable() bool               // 설치 확인
    Install() error                  // 설치
    GetVersion() (string, error)     // 버전 조회
    Upgrade() error                  // 업그레이드
    FindConfigFiles(root string) []string  // 설정 파일 탐색
    Execute(ctx context.Context, files []string, options ExecuteOptions) (*Result, error)
}
```

---

## 3. BaseTool 활용

`BaseTool`은 공통 기능을 제공합니다. 새 도구는 이를 임베딩하여 재사용하세요.

### 3.1 BaseTool이 제공하는 기능

| 메서드 | 설명 |
|--------|------|
| `Name()` | 도구 이름 반환 |
| `Language()` | 언어 반환 |
| `Type()` | 도구 타입 반환 |
| `IsAvailable()` | PATH에서 실행 파일 확인 |
| `GetVersion()` | --version 명령 실행 |
| `Install()` | 설정된 설치 명령 실행 |
| `Upgrade()` | Install과 동일 (대부분의 경우) |
| `FindConfigFiles()` | 설정 패턴으로 파일 탐색 |
| `ExecuteCommand()` | 명령 실행 및 Result 생성 |

### 3.2 BaseTool 사용 예시

```go
type MyNewTool struct {
    *BaseTool  // 임베딩
}

func NewMyNewTool() *MyNewTool {
    tool := &MyNewTool{
        BaseTool: NewBaseTool(
            "mytool",      // 이름
            "Go",          // 언어
            "mytool",      // 실행 파일명
            LINT,          // 타입
        ),
    }

    // 설치 명령 설정
    tool.SetInstallCommand([]string{"go", "install", "example.com/mytool@latest"})

    // 설정 파일 패턴
    tool.SetConfigPatterns([]string{".mytool.yml", "mytool.config.json"})

    return tool
}
```

---

## 4. Execute 메서드 구현

핵심인 `Execute` 메서드를 구현해야 합니다.

### 4.1 기본 구조

```go
func (t *MyNewTool) Execute(ctx context.Context, files []string, options ExecuteOptions) (*Result, error) {
    // 1. 파일이 없으면 조기 반환
    if len(files) == 0 {
        return &Result{
            Tool:     t.Name(),
            Language: t.Language(),
            Success:  true,
        }, nil
    }

    // 2. 명령어 구성
    args := t.buildArgs(files, options)
    cmd := exec.CommandContext(ctx, t.executable, args...)
    cmd.Dir = options.ProjectRoot

    // 3. 환경 변수 설정
    if len(options.Env) > 0 {
        cmd.Env = append(os.Environ(), t.envToSlice(options.Env)...)
    }

    // 4. 실행 및 결과 수집
    result, err := t.ExecuteCommand(ctx, cmd, files)
    if err != nil {
        return result, err
    }

    // 5. 출력 파싱하여 이슈 추출
    result.Issues = t.parseOutput(result.Output)
    result.Success = len(result.Issues) == 0 && result.Error == nil

    return result, nil
}
```

### 4.2 인수 빌드 예시

```go
func (t *MyNewTool) buildArgs(files []string, options ExecuteOptions) []string {
    args := []string{}

    // 설정 파일 지정
    if options.ConfigFile != "" {
        args = append(args, "--config", options.ConfigFile)
    }

    // 자동 수정 옵션
    if options.Fix {
        args = append(args, "--fix")
    }

    // 추가 인수
    args = append(args, options.ExtraArgs...)

    // 대상 파일
    args = append(args, files...)

    return args
}
```

### 4.3 출력 파싱 예시

```go
func (t *MyNewTool) parseOutput(output string) []Issue {
    var issues []Issue

    // 예시: "file.go:10:5: error message"
    re := regexp.MustCompile(`^(.+):(\d+):(\d+):\s*(.+)$`)

    for _, line := range strings.Split(output, "\n") {
        matches := re.FindStringSubmatch(line)
        if len(matches) == 5 {
            lineNum, _ := strconv.Atoi(matches[2])
            colNum, _ := strconv.Atoi(matches[3])

            issues = append(issues, Issue{
                File:     matches[1],
                Line:     lineNum,
                Column:   colNum,
                Severity: "error",
                Message:  matches[4],
                Rule:     "unknown",
            })
        }
    }

    return issues
}
```

---

## 5. 레지스트리 등록

`quality.go`의 `registerAllTools()` 함수에 등록합니다.

```go
func registerAllTools(registry tools.ToolRegistry) {
    // 기존 도구들...

    // 새 도구 등록
    registry.Register(tools.NewMyNewTool())
}
```

---

## 6. 전체 예시: Biome 추가

Biome (JavaScript/TypeScript 포매터+린터) 추가 예시입니다.

### 6.1 도구 구현

```go
// javascript_tools.go에 추가

type BiomeTool struct {
    *BaseTool
}

func NewBiomeTool() *BiomeTool {
    tool := &BiomeTool{
        BaseTool: NewBaseTool("biome", "JavaScript", "biome", BOTH),
    }

    tool.SetInstallCommand([]string{"npm", "install", "-g", "@biomejs/biome"})
    tool.SetConfigPatterns([]string{
        "biome.json",
        "biome.jsonc",
    })

    return tool
}

func (t *BiomeTool) Execute(ctx context.Context, files []string, options ExecuteOptions) (*Result, error) {
    if len(files) == 0 {
        return &Result{Tool: t.Name(), Language: t.Language(), Success: true}, nil
    }

    args := t.buildArgs(files, options)
    cmd := exec.CommandContext(ctx, "biome", args...)
    cmd.Dir = options.ProjectRoot

    result, err := t.ExecuteCommand(ctx, cmd, files)
    if err != nil {
        return result, err
    }

    result.Issues = t.parseOutput(result.Output)
    result.Success = result.Error == nil

    return result, nil
}

func (t *BiomeTool) buildArgs(files []string, options ExecuteOptions) []string {
    var args []string

    // 포매팅 + 린팅
    if options.FormatOnly {
        args = append(args, "format")
    } else if options.LintOnly {
        args = append(args, "lint")
    } else {
        args = append(args, "check")
    }

    // 자동 수정
    if options.Fix {
        args = append(args, "--apply")
    }

    // 설정 파일
    if options.ConfigFile != "" {
        args = append(args, "--config-path", options.ConfigFile)
    }

    args = append(args, options.ExtraArgs...)
    args = append(args, files...)

    return args
}

func (t *BiomeTool) parseOutput(output string) []Issue {
    var issues []Issue

    // Biome JSON 출력 파싱 로직
    // 실제 구현시 Biome의 출력 형식에 맞게 조정

    return issues
}
```

### 6.2 레지스트리 등록

```go
// quality.go

func registerAllTools(registry tools.ToolRegistry) {
    // JavaScript tools
    registry.Register(tools.NewPrettierTool())
    registry.Register(tools.NewESLintTool())
    registry.Register(tools.NewBiomeTool())  // 추가
    // ...
}
```

### 6.3 기본 설정 추가

```go
// config/config.go

func DefaultConfig() *Config {
    return &Config{
        Tools: map[string]ToolConfig{
            // ...
            "biome": {
                Enabled:  false,  // prettier/eslint과 중복이므로 기본 비활성
                Priority: 8,
            },
        },
        // ...
    }
}
```

---

## 7. 테스트 작성

### 7.1 단위 테스트

```go
// tools/biome_test.go

func TestBiomeTool_Name(t *testing.T) {
    tool := NewBiomeTool()
    assert.Equal(t, "biome", tool.Name())
}

func TestBiomeTool_Language(t *testing.T) {
    tool := NewBiomeTool()
    assert.Equal(t, "JavaScript", tool.Language())
}

func TestBiomeTool_Type(t *testing.T) {
    tool := NewBiomeTool()
    assert.Equal(t, BOTH, tool.Type())
}

func TestBiomeTool_BuildArgs(t *testing.T) {
    tool := NewBiomeTool()

    tests := []struct {
        name     string
        options  ExecuteOptions
        files    []string
        expected []string
    }{
        {
            name:     "check mode",
            options:  ExecuteOptions{},
            files:    []string{"src/index.ts"},
            expected: []string{"check", "src/index.ts"},
        },
        {
            name:     "format only with fix",
            options:  ExecuteOptions{FormatOnly: true, Fix: true},
            files:    []string{"src/index.ts"},
            expected: []string{"format", "--apply", "src/index.ts"},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            args := tool.buildArgs(tt.files, tt.options)
            assert.Equal(t, tt.expected, args)
        })
    }
}
```

### 7.2 통합 테스트

```go
func TestBiomeTool_Execute_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    tool := NewBiomeTool()
    if !tool.IsAvailable() {
        t.Skip("biome not installed")
    }

    // 임시 프로젝트 생성
    tmpDir := t.TempDir()
    testFile := filepath.Join(tmpDir, "test.js")
    os.WriteFile(testFile, []byte("const x=1"), 0644)

    // 실행
    result, err := tool.Execute(context.Background(), []string{testFile}, ExecuteOptions{
        ProjectRoot: tmpDir,
    })

    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "biome", result.Tool)
}
```

---

## 8. 체크리스트

새 도구 추가 시 확인 사항:

- [ ] `QualityTool` 인터페이스 완전 구현
- [ ] `BaseTool` 임베딩 및 설정
- [ ] `SetInstallCommand()` 호출
- [ ] `SetConfigPatterns()` 호출
- [ ] `Execute()` 메서드 구현
- [ ] 출력 파싱 로직 구현
- [ ] 레지스트리 등록 (`registerAllTools`)
- [ ] 기본 설정 추가 (`DefaultConfig`)
- [ ] 단위 테스트 작성
- [ ] 통합 테스트 작성 (선택)
- [ ] README에 도구 추가

---

## 9. 팁

### 9.1 출력 파싱

- 도구가 JSON 출력을 지원하면 `--output-format json` 같은 옵션 활용
- 정규식보다 JSON 파싱이 안정적

### 9.2 설정 파일

- 도구별 설정 파일 위치 패턴을 정확히 파악
- 여러 가능한 위치 모두 등록

### 9.3 오류 처리

- 도구가 이슈 발견 시 비정상 종료 코드를 반환하는 경우가 많음
- `cmd.Run()` 에러와 실제 실행 오류 구분 필요

### 9.4 타임아웃

- 느린 도구는 Context 타임아웃 고려
- 대용량 파일/많은 파일 처리 시 주의

---

*최종 수정: 2025-11-27*
*참조: [ARCHITECTURE.md](./ARCHITECTURE.md), [API.md](./API.md)*
