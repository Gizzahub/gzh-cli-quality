# gzh-cli-quality

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.24-blue.svg)](https://golang.org/doc/go1.24)
[![Coverage](https://img.shields.io/badge/coverage-76.2%25-brightgreen.svg)](./coverage.html)
[![Benchmarks](https://github.com/Gizzahub/gzh-cli-quality/workflows/Benchmarks/badge.svg)](https://github.com/Gizzahub/gzh-cli-quality/actions/workflows/benchmarks.yml)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey.svg)](#설치)
[![Languages](https://img.shields.io/badge/languages-Go%20%7C%20Python%20%7C%20JS%2FTS%20%7C%20Rust-orange.svg)](#지원-도구)
[![Tools](https://img.shields.io/badge/tools-11+-blueviolet.svg)](#지원-도구)
[![Documentation](https://img.shields.io/badge/docs-comprehensive-success.svg)](./docs/)

</div>

---

**멀티 언어 코드 품질 도구 오케스트레이터** - Go, Python, JavaScript/TypeScript, Rust 프로젝트의 포매팅과 린팅을 하나의 명령어로 통합 실행합니다.

## 주요 기능

- **통합 실행**: 11+ 품질 도구를 단일 CLI로 제공
- **병렬 처리**: Worker Pool 패턴으로 빠른 실행
- **Git 통합**: staged/changed 파일만 선택적 처리
- **무설정 시작**: 언어/도구 자동 감지로 즉시 사용
- **멀티 리포트**: JSON, HTML, Markdown 출력 지원
- **고성능**: 나노초 수준 최적화 (Registry 조회 14ns, 필터링 8ns)
- **결과 캐싱**: 파일 기반 캐시로 반복 실행 최적화 (캐시 히트 시 ~60x 빠름)

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

#### Go Install

```bash
# 최신 안정 버전
go install github.com/Gizzahub/gzh-cli-quality/cmd/quality@v0.1.3

# 또는 최신 개발 버전
go install github.com/Gizzahub/gzh-cli-quality/cmd/quality@latest
```

Go Install로 설치되는 실행 파일 이름은 `quality`입니다. 릴리스 archive와
`make build`/`make install`은 `gz-quality`라는 이름을 사용합니다.

#### Homebrew (macOS)

```bash
brew install --cask gizzahub/tap/gz-quality
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
| `gz-quality cache status` | 캐시 상태 확인 |
| `gz-quality cache clear` | 캐시 삭제 |

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
--no-cache            # 캐시 비활성화

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

> 📄 **전체 설정 예시**: [`.gzquality.yml.example`](./.gzquality.yml.example) 파일 참조 (372줄, 5가지 시나리오 포함)

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

cache:
  enabled: true           # 캐시 활성화 (기본: true)
  directory: ".gzquality-cache"
  max_size: "1GB"
  ttl: "168h"             # 7일
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

## 📚 문서

> **처음 사용하시나요?** → [5분 빠른 시작 가이드](./docs/user/00-quick-start.md)로 시작하세요!

### 👤 사용자 가이드 (한국어)

#### 시작하기
- **[5분 빠른 시작](./docs/user/00-quick-start.md)** ⭐ - 설치부터 첫 실행까지
- **[설치 가이드](./docs/user/01-installation.md)** 📦 - 상세 설치 방법 (모든 플랫폼)
- **[설정 가이드](./docs/user/03-configuration.md)** ⚙️ - `.gzquality.yml` 완벽 가이드
- **[FAQ](./docs/user/06-faq.md)** ❓ - 자주 묻는 질문 (30개)
- **[문제 해결](./docs/user/05-troubleshooting.md)** 🔧 - 상세 트러블슈팅

#### 실전 가이드
- [사용 예제](./docs/user/02-examples.md) - 실전 워크플로우 패턴
- **[Migration Guide](./docs/user/07-migration.md)** 🔄 - 기존 도구에서 전환하기 (837줄)
- **[Cookbook](./docs/user/08-cookbook.md)** 📚 - 10가지 실전 레시피 (1143줄)
- [CI/CD 통합](./docs/integration/CI_INTEGRATION.md) - GitHub Actions, GitLab CI, CircleCI
- [Pre-commit Hooks](./docs/integration/PRE_COMMIT_HOOKS.md) - 커밋 전 자동 검사
- [멀티 리포지토리](./docs/integration/MULTI_REPO_WORKFLOWS.md) - 대규모 프로젝트

### 🤖 LLM/AI 도구용 문서 (English)

- **[Context](./docs/llm/CONTEXT.md)** - Project overview, architecture, and integration
- **[Capabilities](./docs/llm/CAPABILITIES.md)** - Complete feature reference with 23+ execution modes

### 🔧 개발자 문서

#### 설계 및 아키텍처
- [제품 요구사항 (PRD)](./docs/developer/PRD.md) - 제품 비전 및 요구사항
- [상세 요구사항](./docs/developer/REQUIREMENTS.md) - 기능/비기능 요구사항 명세
- [시스템 아키텍처](./docs/developer/ARCHITECTURE.md) - 컴포넌트 설계 및 데이터 플로우

#### API 및 확장
- [API 레퍼런스](./docs/developer/API.md) - CLI 명령어 및 Go 패키지 API
- [도구 추가 가이드](./docs/developer/ADDING_TOOLS.md) - 새로운 품질 도구 통합 방법

#### 품질 및 성능
- [테스트 가이드](./docs/developer/TESTING.md) - 테스트 전략 및 커버리지 (76.2%)
- [성능 벤치마크](./docs/developer/BENCHMARKS.md) - 27개 벤치마크 및 성능 메트릭
- [커버리지 리포트](./docs/developer/COVERAGE.md) - 패키지별 테스트 커버리지 상세
- [캐싱 시스템](./docs/developer/CACHING.md) - 캐시 아키텍처 및 구현 상세

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

# 성능 벤치마크 실행
go test -bench=. -benchmem ./tools ./detector ./executor

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
