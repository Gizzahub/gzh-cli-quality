# 설정 가이드

`.gzquality.yml` 설정 파일을 사용하여 프로젝트에 맞게 gzh-cli-quality를 커스터마이징하는 방법을 설명합니다.

## 목차

- [설정 파일 개요](#설정-파일-개요)
- [설정 파일 생성](#설정-파일-생성)
- [전역 설정](#전역-설정)
- [도구별 설정](#도구별-설정)
- [언어별 설정](#언어별-설정)
- [파일 필터링](#파일-필터링)
- [실전 예제](#실전-예제)
- [고급 설정](#고급-설정)

---

## 설정 파일 개요

### 설정 파일 위치

gzh-cli-quality는 다음 순서로 설정 파일을 찾습니다:

1. **프로젝트 설정** (최우선): `.gzquality.yml` (프로젝트 루트)
2. **사용자 전역 설정**: `~/.config/gz-quality/config.yml`
3. **기본 설정**: 내장 기본값

**우선순위**: 프로젝트 설정 > 전역 설정 > 기본값

### 설정 파일 형식

- **파일명**: `.gzquality.yml` 또는 `.gzquality.yaml`
- **형식**: YAML
- **인코딩**: UTF-8
- **들여쓰기**: 스페이스 2칸

---

## 설정 파일 생성

### 자동 생성

```bash
# 프로젝트 루트에서 실행
gz-quality init

# 기존 파일 덮어쓰기
gz-quality init --force

# 커스텀 경로에 생성
gz-quality init --output custom-config.yml
```

**생성되는 내용**:
- 감지된 언어 기반 도구 설정
- 프로젝트 구조에 맞는 제외 패턴
- 권장 기본값

### 수동 생성

```bash
# 텍스트 에디터로 생성
vim .gzquality.yml

# 또는
touch .gzquality.yml
```

---

## 전역 설정

프로젝트 전체에 적용되는 설정입니다.

### default_workers

병렬 실행할 워커 수를 지정합니다.

```yaml
# CPU 코어 수만큼 (기본값)
default_workers: 4

# 최대 성능
default_workers: 8

# 저성능 환경
default_workers: 2

# 순차 실행
default_workers: 1
```

**권장값**:
- 개발 환경: CPU 코어 수
- CI/CD: 4-8
- 저사양: 2

### timeout

전체 실행 시간 제한을 설정합니다.

```yaml
# 기본값: 10분
timeout: "10m"

# 대규모 프로젝트
timeout: "30m"

# 빠른 체크
timeout: "2m"
```

**형식**: `"<숫자><단위>"` (단위: `s`, `m`, `h`)

**권장값**:
- 로컬 개발: 5-10분
- CI/CD: 15-30분
- Pre-commit: 1-2분

---

## 도구별 설정

각 도구의 동작을 세밀하게 제어합니다.

### 기본 구조

```yaml
tools:
  <도구명>:
    enabled: true|false        # 활성화 여부
    priority: <숫자>           # 실행 우선순위 (높을수록 먼저)
    config_file: "<경로>"      # 도구 설정 파일
    args: [<인수목록>]         # 추가 CLI 인수
    env:                       # 환경 변수
      KEY: "value"
    timeout: "<시간>"          # 도구별 타임아웃
```

### enabled (활성화)

```yaml
tools:
  # 활성화 (기본값)
  gofumpt:
    enabled: true

  # 비활성화 (느린 도구 로컬에서 끄기)
  pylint:
    enabled: false

  # 조건부 활성화
  golangci-lint:
    enabled: true
    # CI에서만 전체 검사
```

### priority (우선순위)

실행 순서를 제어합니다. 높은 숫자가 먼저 실행됩니다.

```yaml
tools:
  # 포매터: 우선순위 10 (먼저 실행)
  gofumpt:
    priority: 10
  black:
    priority: 10

  # 포매터+린터: 우선순위 7
  ruff:
    priority: 7

  # 린터: 우선순위 5 (나중 실행)
  golangci-lint:
    priority: 5
  eslint:
    priority: 5

  # 타입체커: 우선순위 3 (가장 나중)
  tsc:
    priority: 3
```

**권장 우선순위**:
- 10: 포매터 (gofumpt, black, prettier, rustfmt)
- 7: 포매터+린터 (ruff)
- 5: 린터 (golangci-lint, eslint, clippy)
- 3: 타입체커 (tsc)

### config_file (설정 파일)

도구별 설정 파일 경로를 지정합니다.

```yaml
tools:
  golangci-lint:
    config_file: ".golangci.yml"  # 상대 경로

  prettier:
    config_file: "/absolute/path/to/.prettierrc"  # 절대 경로

  eslint:
    config_file: "config/.eslintrc.json"  # 하위 디렉토리
```

**자동 감지**: 설정하지 않으면 도구의 기본 위치에서 자동 탐색
- golangci-lint: `.golangci.yml`, `.golangci.yaml`
- prettier: `.prettierrc`, `.prettierrc.json`, `prettier.config.js`
- eslint: `.eslintrc.js`, `.eslintrc.json`

### args (CLI 인수)

도구 실행 시 추가 인수를 전달합니다.

```yaml
tools:
  ruff:
    args: ["--fix", "--exit-zero"]

  golangci-lint:
    args: ["--enable-all", "--max-issues-per-linter", "0"]

  black:
    args: ["--line-length=100", "--target-version=py311"]

  prettier:
    args: ["--single-quote", "--trailing-comma=all"]
```

### env (환경 변수)

도구 실행 시 환경 변수를 설정합니다.

```yaml
tools:
  golangci-lint:
    env:
      GOLANGCI_LINT_CACHE: "/tmp/golangci-cache"
      GOPROXY: "https://proxy.golang.org"

  ruff:
    env:
      RUFF_CACHE_DIR: "/tmp/ruff-cache"
```

### timeout (도구별 타임아웃)

```yaml
tools:
  # 느린 도구에 더 긴 시간 부여
  golangci-lint:
    timeout: "5m"

  pylint:
    timeout: "3m"

  # 빠른 도구는 짧게
  gofumpt:
    timeout: "30s"
```

---

## 언어별 설정

언어 단위로 도구를 관리합니다.

### 기본 구조

```yaml
languages:
  <언어명>:
    enabled: true|false        # 언어 처리 여부
    preferred_tools: [<도구목록>]  # 사용할 도구 (순서대로)
    extensions: [<확장자목록>]  # 파일 확장자
```

### Go 설정

```yaml
languages:
  Go:
    enabled: true
    preferred_tools:
      - gofumpt      # 포매터 1순위
      - goimports    # 포매터 2순위
      - golangci-lint # 린터
    extensions:
      - .go
      - .mod
      - .sum
```

### Python 설정

```yaml
languages:
  Python:
    enabled: true
    preferred_tools:
      - black   # 포매터
      - ruff    # 포매터+린터
      # pylint는 제외 (느림)
    extensions:
      - .py
      - .pyi
```

### JavaScript/TypeScript 설정

```yaml
languages:
  JavaScript:
    enabled: true
    preferred_tools:
      - prettier
      - eslint
    extensions:
      - .js
      - .jsx
      - .mjs
      - .cjs

  TypeScript:
    enabled: true
    preferred_tools:
      - prettier
      - eslint
      - tsc       # TypeScript만
    extensions:
      - .ts
      - .tsx
      - .mts
      - .cts
```

### Rust 설정

```yaml
languages:
  Rust:
    enabled: true
    preferred_tools:
      - rustfmt
      - clippy
    extensions:
      - .rs
```

---

## 파일 필터링

처리할 파일과 제외할 파일을 제어합니다.

### exclude (제외 패턴)

**기본 구조**:

```yaml
exclude:
  - "<glob 패턴>"
  - "<glob 패턴>"
```

**일반적인 제외 패턴**:

```yaml
exclude:
  # 의존성
  - "node_modules/**"
  - "vendor/**"
  - ".venv/**"
  - "venv/**"

  # 빌드 산출물
  - "dist/**"
  - "build/**"
  - "target/**"
  - "out/**"

  # Git
  - ".git/**"
  - ".github/**"

  # 생성 파일
  - "**/*.pb.go"          # Protobuf
  - "**/*_gen.go"         # Generated Go
  - "**/*.generated.ts"   # Generated TS
  - "**/__pycache__/**"   # Python cache

  # 테스트 데이터
  - "testdata/**"
  - "fixtures/**"
  - "mocks/**"

  # 설정
  - ".idea/**"
  - ".vscode/**"
  - "*.lock"

  # 미니파이
  - "**/*.min.js"
  - "**/*.min.css"
```

**패턴 문법**:
- `*`: 임의의 문자열 (슬래시 제외)
- `**`: 임의의 디렉토리 (슬래시 포함)
- `?`: 임의의 단일 문자
- `[abc]`: a, b, c 중 하나
- `{js,ts}`: js 또는 ts

**예제**:

```yaml
exclude:
  # 특정 파일
  - "main.go"
  - "config.json"

  # 특정 패턴
  - "test_*.py"      # test_로 시작하는 Python 파일
  - "*_test.go"      # _test.go로 끝나는 Go 파일

  # 특정 디렉토리
  - "legacy/**"      # legacy 디렉토리 전체
  - "**/temp/**"     # 모든 temp 디렉토리

  # 복수 확장자
  - "**/*.{min.js,min.css,map}"
```

### include (포함 패턴)

**exclude보다 우선합니다**. 특정 파일을 명시적으로 포함합니다.

```yaml
exclude:
  - "generated/**"

include:
  - "generated/important.go"  # exclude되어도 포함
```

**사용 예**:

```yaml
# 기본적으로 모든 생성 파일 제외
exclude:
  - "**/*_gen.go"
  - "**/*.generated.ts"

# 특정 생성 파일만 검사
include:
  - "src/api_gen.go"         # 이 파일은 검사
  - "models/*.generated.ts"  # 이 디렉토리는 검사
```

---

## 실전 예제

### 예제 1: Go 마이크로서비스

```yaml
# .gzquality.yml
default_workers: 4
timeout: "10m"

tools:
  gofumpt:
    enabled: true
    priority: 10

  goimports:
    enabled: true
    priority: 9

  golangci-lint:
    enabled: true
    priority: 5
    config_file: ".golangci.yml"
    args: ["--fast"]  # 로컬 빠른 검사
    timeout: "5m"

languages:
  Go:
    enabled: true
    preferred_tools: [gofumpt, goimports, golangci-lint]

exclude:
  - "vendor/**"
  - "**/*.pb.go"
  - "**/*_gen.go"
  - "testdata/**"
```

### 예제 2: Python 데이터 사이언스

```yaml
default_workers: 2  # Jupyter 노트북 많으면 메모리 고려
timeout: "15m"

tools:
  black:
    enabled: true
    priority: 10
    args: ["--line-length=88"]

  ruff:
    enabled: true
    priority: 7
    args: ["--fix", "--select=E,F,W"]

  pylint:
    enabled: false  # 느려서 비활성화

languages:
  Python:
    enabled: true
    preferred_tools: [black, ruff]
    extensions: [.py, .pyi]

exclude:
  - ".venv/**"
  - "venv/**"
  - "__pycache__/**"
  - "*.egg-info/**"
  - ".ipynb_checkpoints/**"
  - "data/**"          # 데이터 파일
  - "models/**"        # 학습된 모델
  - "notebooks/old/**" # 오래된 노트북
```

### 예제 3: React + TypeScript 프론트엔드

```yaml
default_workers: 6
timeout: "8m"

tools:
  prettier:
    enabled: true
    priority: 10
    config_file: ".prettierrc"
    args: ["--write"]

  eslint:
    enabled: true
    priority: 5
    config_file: ".eslintrc.json"
    args: ["--fix"]

  tsc:
    enabled: true
    priority: 3
    args: ["--noEmit"]

languages:
  TypeScript:
    enabled: true
    preferred_tools: [prettier, eslint, tsc]
  JavaScript:
    enabled: true
    preferred_tools: [prettier, eslint]

exclude:
  - "node_modules/**"
  - "build/**"
  - "dist/**"
  - "coverage/**"
  - "public/static/**"
  - "**/*.min.js"
  - "**/*.bundle.js"
```

### 예제 4: 멀티 언어 모노레포

```yaml
default_workers: 8
timeout: "20m"

tools:
  # Go
  gofumpt:
    enabled: true
    priority: 10
  golangci-lint:
    enabled: true
    priority: 5
    config_file: "config/.golangci.yml"

  # Python
  black:
    enabled: true
    priority: 10
  ruff:
    enabled: true
    priority: 7

  # TypeScript
  prettier:
    enabled: true
    priority: 10
  eslint:
    enabled: true
    priority: 5

languages:
  Go:
    enabled: true
    preferred_tools: [gofumpt, golangci-lint]
  Python:
    enabled: true
    preferred_tools: [black, ruff]
  TypeScript:
    enabled: true
    preferred_tools: [prettier, eslint]

exclude:
  # 공통
  - "**/node_modules/**"
  - "**/vendor/**"
  - "**/.venv/**"
  - "**/dist/**"
  - "**/build/**"

  # 서비스별
  - "services/legacy/**"      # 레거시 서비스 제외
  - "services/*/testdata/**"  # 모든 서비스의 testdata
```

---

## 고급 설정

### CI/CD 전용 설정

```yaml
# .gzquality.ci.yml (CI 전용)
default_workers: 16  # CI 서버 성능 활용
timeout: "30m"

tools:
  golangci-lint:
    enabled: true
    args: ["--enable-all"]  # CI에서만 전체 검사

  pylint:
    enabled: true  # CI에서만 활성화
    timeout: "10m"

  # 로컬과 동일한 포매터 설정
```

**사용법**:
```bash
# CI에서
gz-quality check --config .gzquality.ci.yml --since main
```

### 환경별 설정

```bash
# 개발 환경
.gzquality.yml           # 기본 (빠른 검사)

# CI 환경
.gzquality.ci.yml        # 전체 검사

# Pre-commit
.gzquality.precommit.yml # 초고속 검사
```

**Pre-commit 예**:
```yaml
# .gzquality.precommit.yml
default_workers: 4
timeout: "2m"  # 빠른 피드백

tools:
  gofumpt:
    enabled: true
  black:
    enabled: true
  prettier:
    enabled: true

  # 린터는 비활성화 (빠른 검사)
  golangci-lint:
    enabled: false
  eslint:
    enabled: false
```

### 팀별 설정

```yaml
# .gzquality.team.yml (팀 표준)
# 팀원 모두 이 설정 사용

default_workers: 4
timeout: "10m"

tools:
  golangci-lint:
    enabled: true
    config_file: ".golangci.yml"
    # 팀 표준 린트 규칙

  black:
    args: ["--line-length=100"]  # 팀 표준 라인 길이

exclude:
  # 팀 표준 제외 패턴
  - "vendor/**"
  - "node_modules/**"
```

### 도구 버전 고정

설정 파일에서는 버전을 지정할 수 없지만, CI에서 고정 가능:

```yaml
# .github/workflows/quality.yml
- name: Install tools
  run: |
    go install mvdan.cc/gofumpt@v0.6.0
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
    pip install black==24.1.0 ruff==0.1.14
```

---

## 설정 검증

### 문법 검증

```bash
# YAML 문법 확인
yamllint .gzquality.yml

# 또는
python3 -c "import yaml; yaml.safe_load(open('.gzquality.yml'))"
```

### 설정 테스트

```bash
# 실행 계획 확인 (실제 실행 안 함)
gz-quality run --dry-run --verbose

# 특정 파일로 테스트
gz-quality run --files="main.go" --dry-run
```

### 설정 디버깅

```bash
# 상세 로그
gz-quality run --verbose

# 특정 도구만 테스트
gz-quality tool gofumpt --dry-run --verbose
```

---

## 마이그레이션

### 기존 도구에서 마이그레이션

**pre-commit에서**:
```yaml
# .pre-commit-config.yaml → .gzquality.yml

# 이전
repos:
  - repo: https://github.com/psf/black
    hooks:
      - id: black
        args: [--line-length=100]

# 이후
tools:
  black:
    enabled: true
    args: ["--line-length=100"]
```

**Make에서**:
```makefile
# Makefile → .gzquality.yml

# 이전
fmt:
    gofumpt -w .
    black .

lint:
    golangci-lint run
    ruff check .

# 이후 (gz-quality run 하나로 통합)
```

---

## 설정 예제 저장소

완전한 예제를 GitHub에서 확인:

- **Go 서비스**: `examples/go-service/.gzquality.yml`
- **Python 앱**: `examples/python-app/.gzquality.yml`
- **React 앱**: `examples/react-app/.gzquality.yml`
- **모노레포**: `examples/monorepo/.gzquality.yml`

---

## 다음 단계

- **[사용 예제](./02-examples.md)** - 실전 워크플로우 패턴
- **[CI/CD 통합](../integration/CI_INTEGRATION.md)** - 자동화 설정
- **[Pre-commit Hooks](../integration/PRE_COMMIT_HOOKS.md)** - 커밋 전 자동 검사
- **[문제 해결](./05-troubleshooting.md)** - 설정 관련 문제 해결

---

**설정 예제가 더 필요하면**: [GitHub Issues](https://github.com/Gizzahub/gzh-cli-quality/issues)에 프로젝트 유형을 알려주세요!
