# gzh-cli-quality 사용 예제

실제 프로젝트에서 `gzq`를 활용하는 다양한 사용 사례와 예제입니다.

## 목차

- [기본 사용법](#기본-사용법)
- [Git 워크플로우 통합](#git-워크플로우-통합)
- [프로젝트별 사용 사례](#프로젝트별-사용-사례)
- [고급 설정](#고급-설정)
- [문제 해결](#문제-해결)

---

## 기본 사용법

### 1. 전체 프로젝트 품질 검사

```bash
# 모든 파일에 대해 포매팅 + 린팅 실행
gzq run

# 상세 출력
gzq run --verbose

# 실행 계획만 확인 (실제 실행 안 함)
gzq run --dry-run
```

### 2. 자동 수정 적용

```bash
# 포매팅 도구가 자동으로 코드 수정
gzq run --fix

# 포매팅만 수정 (린팅은 검사만)
gzq run --format-only --fix
```

### 3. 린팅만 실행 (코드 수정 없이 검사만)

```bash
# 모든 린터 실행
gzq check

# 특정 파일만 검사
gzq check --files="*.go,*.py"
```

---

## Git 워크플로우 통합

### 커밋 전 검사

```bash
# 1. staged 파일만 검사
gzq check --staged

# 2. staged 파일 포매팅 + 검사
gzq run --staged --fix

# 3. 이슈가 없으면 커밋
git commit -m "feat: implement new feature"
```

**팁**: Pre-commit hook으로 자동화 ([CI 통합](#ci-통합) 참조)

### Pull Request 검사

```bash
# main 브랜치 이후 변경된 파일만 검사
gzq check --since main

# 리포트 생성
gzq check --since main --report json --output pr-quality-report.json
```

### 작업 중인 파일만 검사

```bash
# 변경된 모든 파일 (staged + modified + untracked)
gzq check --changed

# 특정 커밋 이후 변경 파일
gzq check --since HEAD~5
gzq check --since v1.0.0
```

---

## 프로젝트별 사용 사례

### Go 프로젝트

```bash
# Go 파일만 검사
gzq tool gofumpt
gzq tool goimports
gzq tool golangci-lint

# 병렬로 모든 Go 도구 실행
gzq run --format-only  # gofumpt + goimports
gzq check              # golangci-lint

# golangci-lint에 추가 옵션 전달
gzq tool golangci-lint -- --enable-all
```

**프로젝트 설정 (`.gzquality.yml`)**:

```yaml
default_workers: 4
timeout: "5m"

languages:
  Go:
    enabled: true
    preferred_tools: [gofumpt, goimports, golangci-lint]

tools:
  golangci-lint:
    enabled: true
    priority: 5
    config_file: ".golangci.yml"
  gofumpt:
    enabled: true
    priority: 10
```

### Python 프로젝트

```bash
# Python 파일만 검사
gzq tool black --fix
gzq tool ruff --fix
gzq tool pylint

# isort 대신 ruff 사용
gzq run --format-only  # black + ruff format
gzq check              # ruff lint + pylint
```

**프로젝트 설정**:

```yaml
languages:
  Python:
    enabled: true
    preferred_tools: [black, ruff, pylint]

tools:
  ruff:
    enabled: true
    priority: 10
    args: ["--fix", "--exit-zero"]
  black:
    enabled: true
    priority: 9
    args: ["--line-length=100"]
  pylint:
    enabled: true
    priority: 5

exclude:
  - "venv/**"
  - ".venv/**"
  - "__pycache__/**"
```

### JavaScript/TypeScript 프로젝트

```bash
# JS/TS 파일 검사
gzq tool prettier --fix
gzq tool eslint --fix
gzq tool tsc

# 특정 디렉토리만
gzq run --files="src/**/*.ts"
```

**프로젝트 설정**:

```yaml
languages:
  JavaScript:
    enabled: true
    preferred_tools: [prettier, eslint]
  TypeScript:
    enabled: true
    preferred_tools: [prettier, eslint, tsc]

tools:
  prettier:
    enabled: true
    priority: 10
    config_file: ".prettierrc"
  eslint:
    enabled: true
    priority: 8
    config_file: ".eslintrc.json"
  tsc:
    enabled: true
    priority: 5

exclude:
  - "node_modules/**"
  - "dist/**"
  - "build/**"
```

### Rust 프로젝트

```bash
# Rust 파일 검사
gzq tool rustfmt
gzq tool clippy

# cargo-fmt 사용
gzq tool cargo-fmt
```

**프로젝트 설정**:

```yaml
languages:
  Rust:
    enabled: true
    preferred_tools: [rustfmt, clippy]

tools:
  rustfmt:
    enabled: true
    priority: 10
  clippy:
    enabled: true
    priority: 5
    args: ["--", "-D", "warnings"]

exclude:
  - "target/**"
```

### 멀티 언어 모노레포

```bash
# 전체 레포지토리 검사
gzq run --workers 8

# 특정 서비스만
cd services/backend && gzq run
cd services/frontend && gzq run

# 특정 언어만
gzq run --files="**/*.go"
gzq run --files="**/*.ts,**/*.tsx"
```

**루트 설정 (`.gzquality.yml`)**:

```yaml
default_workers: 8
timeout: "10m"

languages:
  Go:
    enabled: true
    preferred_tools: [gofumpt, golangci-lint]
  TypeScript:
    enabled: true
    preferred_tools: [prettier, eslint, tsc]
  Python:
    enabled: true
    preferred_tools: [black, ruff]

exclude:
  - "node_modules/**"
  - "vendor/**"
  - ".venv/**"
  - "dist/**"
  - "build/**"
```

---

## 고급 설정

### 성능 최적화

```bash
# 워커 수 증가 (CPU 코어 수에 맞춤)
gzq run --workers 16

# 타임아웃 설정
export GZQ_TIMEOUT=10m
gzq run
```

**설정 파일**:

```yaml
default_workers: 16
timeout: "10m"
parallel_execution: true
```

### 리포트 생성

```bash
# JSON 리포트
gzq check --report json --output quality-report.json

# HTML 리포트
gzq check --report html --output quality-report.html

# Markdown 리포트
gzq check --report markdown --output quality-report.md
```

### 특정 도구 비활성화

```yaml
tools:
  pylint:
    enabled: false  # pylint 비활성화

  golangci-lint:
    enabled: true
    config_file: ".golangci.yml"
```

### 파일 제외 패턴

```yaml
exclude:
  # 의존성
  - "node_modules/**"
  - "vendor/**"
  - ".venv/**"

  # 빌드 산출물
  - "dist/**"
  - "build/**"
  - "target/**"

  # 생성 파일
  - "**/*.pb.go"
  - "**/*_gen.go"
  - "**/generated/**"

  # 테스트 데이터
  - "testdata/**"
  - "fixtures/**"
```

---

## 문제 해결

### 도구가 설치되지 않음

```bash
# 1. 도구 목록 확인
gzq list

# 2. 누락된 도구 설치
gzq install golangci-lint
gzq install ruff
gzq install prettier

# 3. 모든 도구 설치
gzq install
```

### 특정 도구만 실행

```bash
# tool 명령어 사용
gzq tool gofumpt --staged
gzq tool eslint --changed

# 추가 인자 전달
gzq tool golangci-lint -- --enable-all --max-issues-per-linter 0
```

### 성능 문제

```bash
# 1. 변경된 파일만 검사
gzq check --changed

# 2. 워커 수 조정
gzq run --workers 2  # 낮은 리소스 환경

# 3. 타임아웃 증가
gzq run --timeout 15m

# 4. 병렬 실행 비활성화
gzq run --workers 1
```

### 설정 파일 디버깅

```bash
# 1. 프로젝트 분석
gzq analyze

# 2. 설정 파일 재생성
rm .gzquality.yml
gzq init

# 3. 실행 계획 확인
gzq run --dry-run --verbose
```

### 도구 버전 확인

```bash
# 모든 도구 버전 확인
gzq version

# 특정 도구 업그레이드
gzq upgrade golangci-lint
gzq upgrade ruff

# 모든 도구 업그레이드
gzq upgrade
```

---

## 팁과 요령

### 1. 빠른 피드백 루프

```bash
# 작업 중: 변경된 파일만 포매팅
gzq run --changed --format-only --fix

# 커밋 전: staged 파일 전체 검사
gzq check --staged

# PR 전: since main 전체 검사
gzq check --since main --report json
```

### 2. 점진적 도입

```bash
# Phase 1: 포매팅만 적용
gzq run --format-only --fix

# Phase 2: 경고 수준 린팅
gzq check  # 설정 파일에서 strict 모드 끄기

# Phase 3: 엄격한 린팅
gzq check  # 설정 파일에서 strict 모드 켜기
```

### 3. 대규모 코드베이스

```bash
# 1단계: 변경 파일만
gzq run --changed --fix

# 2단계: 커밋 후 전체 검사 (CI)
gzq check --report json

# 3단계: 점진적으로 전체 수정
gzq run --fix
```

---

**관련 문서**:
- [CI 통합 가이드](./CI_INTEGRATION.md)
- [도구 추가하기](./ADDING_TOOLS.md)
- [API 레퍼런스](./API.md)
