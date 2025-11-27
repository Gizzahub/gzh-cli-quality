# gzh-cli-quality

멀티 언어 코드 품질 도구 오케스트레이터 - Go, Python, JavaScript/TypeScript, Rust 프로젝트의 포매팅과 린팅을 하나의 명령어로 통합 실행합니다.

## 주요 기능

- **통합 실행**: 11+ 품질 도구를 단일 CLI로 제공
- **병렬 처리**: Worker Pool 패턴으로 빠른 실행
- **Git 통합**: staged/changed 파일만 선택적 처리
- **무설정 시작**: 언어/도구 자동 감지로 즉시 사용
- **멀티 리포트**: JSON, HTML, Markdown 출력 지원

## 지원 도구

| 언어 | 포매터 | 린터 |
|------|--------|------|
| Go | gofumpt, goimports | golangci-lint |
| Python | black | ruff, pylint |
| JavaScript/TypeScript | prettier | eslint, tsc |
| Rust | rustfmt, cargo-fmt | clippy |

## 빠른 시작

### 설치

#### 소스에서 빌드

```bash
# 리포지토리 클론
git clone https://github.com/Gizzahub/gzh-cli-quality.git
cd gzh-cli-quality

# 빌드 (build/ 디렉토리에 gz-quality 바이너리 생성)
make build

# 또는 $GOPATH/bin에 직접 설치
make install
```

#### Go Install (향후 릴리스 후)

```bash
go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest
```

### 기본 사용법

```bash
# 전체 품질 검사 실행
gz-quality run

# staged 파일만 검사 (커밋 전 체크)
gz-quality run --staged

# 린팅만 실행 (파일 수정 없음)
gz-quality check

# 자동 수정 적용
gz-quality run --fix

# 특정 도구만 실행
gz-quality tool ruff --staged
gz-quality tool golangci-lint --since main
```

## CLI 명령어

| 명령어 | 설명 |
|--------|------|
| `gz-quality run` | 모든 포매팅 및 린팅 도구 실행 |
| `gz-quality check` | 린팅만 실행 (변경 없이 검사) |
| `gz-quality init` | 프로젝트 설정 파일 생성 |
| `gz-quality analyze` | 프로젝트 분석 및 권장 도구 표시 |
| `gz-quality tool <name>` | 특정 도구 직접 실행 |
| `gz-quality install` | 품질 도구 설치 |
| `gz-quality upgrade` | 품질 도구 업그레이드 |
| `gz-quality version` | 도구 버전 확인 |
| `gz-quality list` | 사용 가능한 도구 목록 |

### 주요 옵션

```bash
# Git 기반 필터링
--staged              # staged 파일만
--changed             # 변경된 모든 파일 (staged + modified + untracked)
--since <ref>         # 특정 커밋 이후 변경 파일

# 실행 모드
--format-only         # 포매팅만 실행
--lint-only           # 린팅만 실행
--fix, -x             # 자동 수정 적용
--dry-run             # 실행 계획만 표시

# 성능
--workers, -w <n>     # 병렬 워커 수 (기본: CPU 코어 수)

# 리포트
--report <format>     # 리포트 형식 (json, html, markdown)
--output <path>       # 리포트 출력 경로

# 출력
--verbose, -v         # 상세 출력
```

## 설정

### 자동 설정 생성

```bash
gz-quality init
```

### `.gzquality.yml` 예시

```yaml
default_workers: 4
timeout: "10m"

tools:
  gofumpt:
    enabled: true
    priority: 10
  golangci-lint:
    enabled: true
    priority: 5
    config_file: ".golangci.yml"
  ruff:
    enabled: true
    priority: 7
    args: ["--fix"]

languages:
  Go:
    enabled: true
    preferred_tools: [gofumpt, goimports, golangci-lint]
  Python:
    enabled: true
    preferred_tools: [black, ruff]

exclude:
  - node_modules/**
  - vendor/**
  - .git/**
  - dist/**
```

## 사용 예시

### 커밋 전 검사

```bash
# staged 파일에 포매팅 적용 후 린팅
gz-quality run --staged --fix

# 린팅 이슈만 확인 (수정 없이)
gz-quality check --staged
```

### PR 검사 (CI/CD)

```bash
# main 브랜치 이후 변경 파일 검사
gz-quality check --since main

# JSON 리포트 생성
gz-quality check --since main --report json --output quality-report.json
```

### 특정 언어만 검사

```bash
# Go 도구만 실행
gz-quality tool gofumpt && gz-quality tool golangci-lint

# Python 도구만 실행
gz-quality tool ruff --fix
```

## 출력 예시

```
🔍 프로젝트 분석 중...
📋 실행 계획: 3개 도구, 15개 파일
⚡ 실행 중 (4 workers)...
  ✅ gofumpt (0.5s) - 5 files
  ✅ goimports (0.3s) - 5 files
  ⚠️ golangci-lint (2.1s) - 2 issues
    main.go:42:15 warning: unused variable 'x' (deadcode)
    utils.go:18:1 error: missing return (typecheck)
✨ 완료: 2.9s, 15 files, 2 issues
```

## 문서

- [PRD.md](./PRD.md) - 제품 요구사항
- [REQUIREMENTS.md](./REQUIREMENTS.md) - 상세 요구사항
- [ARCHITECTURE.md](./ARCHITECTURE.md) - 시스템 아키텍처
- [docs/API.md](./docs/API.md) - API 레퍼런스
- [docs/ADDING_TOOLS.md](./docs/ADDING_TOOLS.md) - 도구 추가 가이드

## 개발

### 프로젝트 빌드

```bash
# 의존성 다운로드
go mod download

# 빌드
make build

# 테스트 실행
make test

# 커버리지 리포트 생성
make test-coverage

# 린트 실행
make lint

# 전체 품질 검사 (포매팅 + 린트 + 테스트)
make quality
```

### 바이너리 실행

```bash
# 빌드 후
./build/gz-quality version
./build/gz-quality list
./build/gz-quality run --help

# 또는 설치 후
gz-quality version
```

## 요구사항

### 런타임
- Go 1.24.0+
- Git 2.0+
- 각 언어별 품질 도구 (자동 설치 지원: `gz-quality install`)

### 개발 환경
- Go 1.24.0+
- Make
- golangci-lint (선택사항, `make lint` 실행시 필요)

## 라이선스

MIT License - [LICENSE](./LICENSE) 참조

---

*gzh-cli에서 분리된 품질 도구 모듈*
