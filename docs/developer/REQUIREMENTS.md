# gzh-cli-quality 요구사항 명세서

## 1. 개요

본 문서는 gzh-cli-quality의 기능적/비기능적 요구사항을 정의합니다.
PRD.md에서 정의한 제품 목표를 구체적인 기술 요구사항으로 변환합니다.

---

## 2. 기능 요구사항 (Functional Requirements)

### FR-1: 멀티 언어 도구 실행

#### FR-1.1: Go 도구 지원
| ID | 도구 | 타입 | 필수 | 설명 |
|----|------|------|------|------|
| FR-1.1.1 | gofumpt | 포매터 | O | Go 코드 포매팅 (gofmt 상위 호환) |
| FR-1.1.2 | goimports | 포매터 | O | import 문 정리 및 포매팅 |
| FR-1.1.3 | golangci-lint | 린터 | O | 통합 Go 린터 (43+ 린터 포함) |

#### FR-1.2: Python 도구 지원
| ID | 도구 | 타입 | 필수 | 설명 |
|----|------|------|------|------|
| FR-1.2.1 | black | 포매터 | O | Python 코드 포매터 |
| FR-1.2.2 | ruff | 포매터+린터 | O | 빠른 Python 린터/포매터 |
| FR-1.2.3 | pylint | 린터 | X | Python 정적 분석기 |

#### FR-1.3: JavaScript/TypeScript 도구 지원
| ID | 도구 | 타입 | 필수 | 설명 |
|----|------|------|------|------|
| FR-1.3.1 | prettier | 포매터 | O | JS/TS 코드 포매터 |
| FR-1.3.2 | eslint | 린터 | O | JS/TS 린터 |
| FR-1.3.3 | tsc | 타입체커 | X | TypeScript 타입 검사 |

#### FR-1.4: Rust 도구 지원
| ID | 도구 | 타입 | 필수 | 설명 |
|----|------|------|------|------|
| FR-1.4.1 | rustfmt | 포매터 | O | Rust 코드 포매터 |
| FR-1.4.2 | cargo-fmt | 포매터 | X | Cargo 기반 포매터 |
| FR-1.4.3 | clippy | 린터 | O | Rust 린터 |

---

### FR-2: Git 통합

#### FR-2.1: Staged 파일 처리
```
조건: --staged 플래그 사용
동작: Git staging area의 파일만 처리
출력: 처리된 파일 목록과 결과
```

#### FR-2.2: Changed 파일 처리
```
조건: --changed 플래그 사용
동작: staged + modified + untracked 파일 처리
출력: 처리된 파일 목록과 결과
```

#### FR-2.3: 커밋 기준 처리
```
조건: --since <commit-ref> 플래그 사용
동작: 지정된 커밋 이후 변경된 파일만 처리
유효성: 커밋 레퍼런스 유효성 검사 필수
예시: --since HEAD~1, --since main, --since abc1234
```

#### FR-2.4: Git 플래그 상호 배타
```
규칙: --staged, --changed, --since 중 하나만 사용 가능
오류: 복수 지정 시 명확한 에러 메시지 출력
```

---

### FR-3: 실행 모드

#### FR-3.1: run 명령어
```
명령: gz-quality run [flags]
동작: 모든 포매터 및 린터 실행
기본: 프로젝트 전체 파일 대상
옵션: --format-only, --lint-only, --fix, --dry-run
```

#### FR-3.2: check 명령어
```
명령: gz-quality check [flags]
동작: 린팅만 실행 (파일 수정 없음)
용도: CI/CD 검증, PR 체크
```

#### FR-3.3: tool 명령어
```
명령: gz-quality tool <tool-name> [flags]
동작: 특정 도구만 직접 실행
예시: gz-quality tool ruff --staged
옵션: 모든 Git 플래그 지원
```

#### FR-3.4: Dry-run 모드
```
조건: --dry-run 플래그 사용
동작: 실행 계획만 표시, 실제 실행하지 않음
출력: 실행될 도구, 대상 파일, 예상 시간
```

---

### FR-4: 도구 관리

#### FR-4.1: 도구 감지
```
명령: gz-quality analyze
동작: 프로젝트 언어 감지 및 설치된 도구 확인
출력: 언어별 권장 도구, 설치 상태, 버전 정보
```

#### FR-4.2: 도구 설치
```
명령: gz-quality install [tool-name]
동작:
  - tool-name 지정: 특정 도구 설치
  - tool-name 미지정: 프로젝트에 필요한 모든 도구 설치
설치 방법: 언어별 패키지 매니저 사용
  - Go: go install
  - Python: pip/uv
  - JavaScript: npm
  - Rust: cargo/rustup
```

#### FR-4.3: 도구 업그레이드
```
명령: gz-quality upgrade [tool-name]
동작: 도구를 최신 버전으로 업그레이드
출력: 업그레이드 전/후 버전 표시
```

#### FR-4.4: 버전 확인
```
명령: gz-quality version
동작: 모든 설치된 도구의 버전 표시
출력: 테이블 형식으로 도구명, 버전, 경로 표시
```

#### FR-4.5: 도구 목록
```
명령: gz-quality list
동작: 지원되는 모든 도구 목록 표시
출력: 도구명, 언어, 타입, 설치 상태
```

---

### FR-5: 리포트 생성

#### FR-5.1: JSON 리포트
```
조건: --report json --output <path>
출력 형식:
{
  "summary": { "total": int, "passed": int, "failed": int },
  "results": [
    {
      "tool": "string",
      "language": "string",
      "success": bool,
      "issues": [...]
    }
  ]
}
```

#### FR-5.2: HTML 리포트
```
조건: --report html --output <path>
출력: 시각화된 HTML 리포트
포함: 요약, 도구별 결과, 이슈 목록
```

#### FR-5.3: Markdown 리포트
```
조건: --report markdown --output <path>
출력: GitHub/GitLab 호환 마크다운
용도: PR 코멘트, 문서화
```

#### FR-5.4: 이슈 형식 통합
```
통합 이슈 형식:
{
  "file": "string",
  "line": int,
  "column": int,
  "severity": "error|warning|info",
  "rule": "string",
  "message": "string",
  "suggestion": "string (optional)"
}
```

---

### FR-6: 설정 관리

#### FR-6.1: 설정 파일 형식
```
파일명: .gzquality.yml
위치: 프로젝트 루트
```

#### FR-6.2: 설정 스키마
```yaml
# 기본 설정
default_workers: 4          # 병렬 워커 수
timeout: "10m"              # 실행 타임아웃

# 도구별 설정
tools:
  gofumpt:
    enabled: true
    priority: 10            # 높을수록 먼저 실행
    config_file: ""         # 커스텀 설정 파일
    args: []                # 추가 인수
    env: {}                 # 환경 변수

# 언어별 설정
languages:
  Go:
    enabled: true
    preferred_tools: [gofumpt, goimports, golangci-lint]
    extensions: [.go]

# 제외 패턴
exclude:
  - node_modules/**
  - vendor/**
  - .git/**

# 포함 패턴 (지정시 exclude보다 우선)
include: []
```

#### FR-6.3: 설정 자동 생성
```
명령: gz-quality init
동작: 프로젝트 분석 후 .gzquality.yml 생성
옵션: --force (기존 파일 덮어쓰기)
```

#### FR-6.4: 설정 우선순위
```
1. CLI 플래그 (최우선)
2. 프로젝트 설정 (.gzquality.yml)
3. 사용자 설정 (~/.config/gz-quality/config.yml)
4. 기본값
```

---

## 3. 비기능 요구사항 (Non-Functional Requirements)

### NFR-1: 성능

#### NFR-1.1: 병렬 실행
| 항목 | 요구사항 |
|------|----------|
| 기본 워커 수 | CPU 코어 수 (runtime.NumCPU()) |
| 최대 워커 수 | 설정 가능 (--workers) |
| 워커 패턴 | Worker Pool |
| 태스크 우선순위 | priority 필드 기반 정렬 |

#### NFR-1.2: 타임아웃
| 항목 | 기본값 | 최대값 |
|------|--------|--------|
| 전체 실행 | 10분 | 설정 가능 |
| 개별 도구 | 5분 | - |

#### NFR-1.3: 성능 목표
- 순차 실행 대비 50% 이상 시간 단축 (4+ 코어 기준)
- 100 파일 프로젝트: 변경 파일 기준 10초 내 완료
- 메모리 사용량: 500MB 이하 (일반적 프로젝트)

---

### NFR-2: 안정성

#### NFR-2.1: 오류 처리
```
원칙: 개별 도구 실패는 전체 실행을 중단하지 않음
동작:
  - 도구 실패 시 에러 기록 후 다음 도구 실행
  - 모든 실행 완료 후 통합 결과 보고
  - 부분 실패 시 종료 코드 = 1
```

#### NFR-2.2: 컨텍스트 취소
```
지원: context.Context 기반 취소
동작: Ctrl+C 시 진행 중인 모든 도구 graceful 종료
타임아웃: context.WithTimeout 적용
```

#### NFR-2.3: 복구성
```
원칙: 도구 크래시 시에도 다른 도구 실행 계속
로깅: 모든 오류는 stderr로 출력
```

---

### NFR-3: 사용성

#### NFR-3.1: 제로 설정 시작
```
요구사항: 설정 파일 없이 즉시 사용 가능
동작: 언어 자동 감지 → 도구 자동 선택 → 실행
```

#### NFR-3.2: 진행 표시
```
출력 형식:
🔍 프로젝트 분석 중...
📋 실행 계획: 3개 도구, 15개 파일
⚡ 실행 중 (4 workers)...
  ✅ gofumpt (0.5s) - 5 files
  ✅ goimports (0.3s) - 5 files
  ⚠️ golangci-lint (2.1s) - 2 issues
✨ 완료: 2.9s, 15 files, 2 issues
```

#### NFR-3.3: 상세 출력
```
조건: --verbose 또는 -v 플래그
출력: 개별 파일 처리 상태, 도구 명령어, 원본 출력
```

#### NFR-3.4: 에러 메시지
```
형식: 명확하고 실행 가능한 메시지
예시:
  ❌ golangci-lint 미설치
  💡 설치: gz-quality install golangci-lint
```

---

### NFR-4: 확장성

#### NFR-4.1: 도구 인터페이스
```go
type QualityTool interface {
    Name() string
    Language() string
    Type() ToolType  // FORMAT, LINT, BOTH
    IsAvailable() bool
    Install() error
    GetVersion() (string, error)
    Upgrade() error
    FindConfigFiles(projectRoot string) []string
    Execute(ctx context.Context, files []string, options ExecuteOptions) (*Result, error)
}
```

#### NFR-4.2: 도구 레지스트리
```go
type ToolRegistry interface {
    Register(tool QualityTool)
    GetTools() []QualityTool
    GetToolsByLanguage(language string) []QualityTool
    GetToolsByType(toolType ToolType) []QualityTool
    FindTool(name string) QualityTool
}
```

#### NFR-4.3: 새 도구 추가 용이성
- 인터페이스 구현만으로 새 도구 추가 가능
- 레지스트리에 등록하면 자동으로 CLI에 노출
- BaseTool 제공으로 공통 로직 재사용

---

### NFR-5: 호환성

#### NFR-5.1: 운영체제
| OS | 지원 수준 |
|----|----------|
| Linux (amd64, arm64) | 완전 지원 |
| macOS (amd64, arm64) | 완전 지원 |
| Windows (amd64) | 완전 지원 |

#### NFR-5.2: 런타임 의존성
| 의존성 | 최소 버전 |
|--------|----------|
| Go | 1.24.0 |
| Git | 2.0 |

#### NFR-5.3: 도구 버전 호환성
- 각 도구의 주요 버전과 호환성 유지
- 버전 불일치 시 경고 표시

---

## 4. 인터페이스 요구사항

### IR-1: CLI 인터페이스

#### IR-1.1: 명령어 형식
```
gz-quality <command> [subcommand] [flags] [args]
```

#### IR-1.2: 플래그 규칙
- 단축형: -x (단일 문자)
- 장문형: --fix (전체 이름)
- Boolean: --verbose (true), --no-color (false)
- 값 전달: --workers=4 또는 --workers 4

#### IR-1.3: 종료 코드
| 코드 | 의미 |
|------|------|
| 0 | 성공 (이슈 없음) |
| 1 | 이슈 발견 또는 부분 실패 |
| 2 | 실행 오류 (설정, 도구 미설치 등) |

---

### IR-2: 파일 인터페이스

#### IR-2.1: 설정 파일
- 형식: YAML
- 인코딩: UTF-8
- 파일명: `.gzquality.yml`

#### IR-2.2: 리포트 파일
- JSON: RFC 8259 준수
- HTML: HTML5 표준
- Markdown: CommonMark 규격

---

## 5. 제약사항

### C-1: 기술 제약
- 외부 도구 의존: 품질 도구는 시스템에 설치되어 있어야 함
- 네트워크 불필요: 오프라인 환경에서도 동작 (도구 설치 제외)

### C-2: 보안 제약
- 임의 명령어 실행 금지: 사전 정의된 도구만 실행
- 환경 변수 제한: 허용된 환경 변수만 전달

### C-3: 성능 제약
- 대용량 파일 (>10MB) 처리 시 경고
- 1000+ 파일 프로젝트에서 메모리 사용량 모니터링

---

*최종 수정: 2025-11-27*
*참조 문서: [PRD.md](./PRD.md)*
