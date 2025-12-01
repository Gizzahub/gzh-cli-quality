# Migration Guide

기존 품질 도구 설정에서 gzh-cli-quality로 마이그레이션하는 방법을 안내합니다.

## 목차

- [왜 마이그레이션해야 하나요?](#왜-마이그레이션해야-하나요)
- [시나리오별 마이그레이션](#시나리오별-마이그레이션)
  - [1. golangci-lint에서 마이그레이션](#1-golangci-lint에서-마이그레이션)
  - [2. pre-commit에서 마이그레이션](#2-pre-commit에서-마이그레이션)
  - [3. npm scripts에서 마이그레이션](#3-npm-scripts에서-마이그레이션)
  - [4. Makefile에서 마이그레이션](#4-makefile에서-마이그레이션)
  - [5. CI/CD 파이프라인 마이그레이션](#5-cicd-파이프라인-마이그레이션)
- [점진적 마이그레이션 전략](#점진적-마이그레이션-전략)
- [롤백 계획](#롤백-계획)

---

## 왜 마이그레이션해야 하나요?

### 현재 방식의 문제점

```bash
# 기존 방식: 각 도구를 개별 실행
gofumpt -w .
goimports -w .
golangci-lint run
black .
ruff check --fix .
prettier --write .
eslint --fix .
```

**문제:**
- ❌ 순차 실행으로 느림 (10개 도구 = 10배 시간)
- ❌ 각 도구마다 다른 명령어/옵션
- ❌ Git staged 파일만 처리하기 어려움
- ❌ 통합 리포트 불가능
- ❌ 팀 설정 공유 어려움

### gzh-cli-quality 방식

```bash
# 통합 방식: 하나의 명령어
gz-quality run --staged
```

**장점:**
- ✅ 병렬 실행으로 빠름 (50%+ 시간 단축)
- ✅ 단일 명령어, 일관된 인터페이스
- ✅ Git 네이티브 지원 (staged/changed/since)
- ✅ 통합 JSON/HTML 리포트
- ✅ YAML 설정 파일로 팀 표준화

---

## 시나리오별 마이그레이션

### 1. golangci-lint에서 마이그레이션

#### Before: golangci-lint만 사용

```bash
# .github/workflows/ci.yml
- name: Run golangci-lint
  uses: golangci/golangci-lint-action@v6
  with:
    version: latest
```

**문제:** Go 도구만 체크, 다른 언어는 별도 설정 필요

#### After: gz-quality 사용

```bash
# .github/workflows/ci.yml
- name: Install gz-quality
  run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest

- name: Run quality checks
  run: gz-quality check --since ${{ github.event.pull_request.base.sha }}
```

**장점:** 모든 언어 자동 감지 및 체크

#### 마이그레이션 단계

**Step 1: gz-quality 설치**

```bash
go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest
```

**Step 2: 기존 .golangci.yml 유지**

```yaml
# .gzquality.yml
tools:
  golangci-lint:
    enabled: true
    config_file: ".golangci.yml"  # 기존 설정 재사용
    priority: 5
```

**Step 3: 로컬에서 테스트**

```bash
# 기존 방식
golangci-lint run

# 새 방식 (같은 결과)
gz-quality tool golangci-lint

# 전체 품질 검사 (Go + 다른 언어)
gz-quality run
```

**Step 4: CI/CD 업데이트**

```diff
# .github/workflows/ci.yml
- - name: Run golangci-lint
-   uses: golangci/golangci-lint-action@v6
-   with:
-     version: latest

+ - name: Run quality checks
+   run: |
+     go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest
+     gz-quality check --since main
```

---

### 2. pre-commit에서 마이그레이션

#### Before: pre-commit framework

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/psf/black
    rev: 23.12.0
    hooks:
      - id: black

  - repo: https://github.com/charliermarsh/ruff-pre-commit
    rev: v0.1.8
    hooks:
      - id: ruff

  - repo: https://github.com/golangci/golangci-lint
    rev: v1.55.2
    hooks:
      - id: golangci-lint
```

**문제:**
- 각 도구마다 별도 repo 설정
- 버전 관리 복잡 (각 도구 rev 업데이트)
- Python 환경 필요

#### After: gz-quality Git Hooks

```bash
# hooks/pre-commit
#!/bin/bash
gz-quality run --staged --fix
```

**장점:**
- 단일 바이너리 (Python 불필요)
- 통합 버전 관리
- 더 빠른 실행 (병렬)

#### 마이그레이션 단계

**Step 1: pre-commit 제거 (선택사항)**

```bash
# pre-commit 비활성화 (롤백 가능하도록 보관)
mv .pre-commit-config.yaml .pre-commit-config.yaml.backup
pre-commit uninstall
```

**Step 2: gz-quality hook 설치**

```bash
# 프로젝트의 hooks 디렉토리 사용
mkdir -p hooks
cat > hooks/pre-commit << 'EOF'
#!/bin/bash
set -e

echo "🔍 Running quality checks on staged files..."
gz-quality run --staged --fix

# 수정된 파일 재 스테이징
git diff --name-only --cached | xargs -r git add

echo "✅ Quality checks passed!"
EOF

chmod +x hooks/pre-commit
```

**Step 3: Git hook 연결**

```bash
# 방법 1: 심볼릭 링크
ln -sf ../../hooks/pre-commit .git/hooks/pre-commit

# 방법 2: core.hooksPath 설정
git config core.hooksPath hooks
```

**Step 4: 테스트**

```bash
# 파일 수정
echo "# test" >> README.md
git add README.md

# 커밋 시도 (hook이 자동 실행됨)
git commit -m "test: verify pre-commit hook"
```

**병행 사용 (점진적 마이그레이션)**

두 시스템을 동시에 사용할 수도 있습니다:

```yaml
# .pre-commit-config.yaml (병행 사용)
repos:
  - repo: local
    hooks:
      - id: gz-quality
        name: gz-quality
        entry: gz-quality run --staged --fix
        language: system
        pass_filenames: false
```

---

### 3. npm scripts에서 마이그레이션

#### Before: package.json scripts

```json
{
  "scripts": {
    "format": "prettier --write .",
    "lint": "eslint --fix .",
    "typecheck": "tsc --noEmit",
    "quality": "npm run format && npm run lint && npm run typecheck"
  }
}
```

**문제:**
- 순차 실행으로 느림
- npm 의존성 필요
- 다른 언어 도구와 통합 어려움

#### After: gz-quality

```json
{
  "scripts": {
    "quality": "gz-quality run",
    "quality:check": "gz-quality check",
    "quality:staged": "gz-quality run --staged"
  }
}
```

**장점:**
- 병렬 실행으로 빠름
- 모든 언어 통합 지원
- npm 없이도 동작 (Go 바이너리)

#### 마이그레이션 단계

**Step 1: 기존 도구 설정 보존**

```yaml
# .gzquality.yml
tools:
  prettier:
    enabled: true
    config_file: ".prettierrc"  # 기존 설정 재사용
  eslint:
    enabled: true
    config_file: ".eslintrc.json"
  tsc:
    enabled: true
    config_file: "tsconfig.json"
```

**Step 2: package.json 업데이트**

```diff
{
  "scripts": {
-   "format": "prettier --write .",
-   "lint": "eslint --fix .",
-   "typecheck": "tsc --noEmit",
-   "quality": "npm run format && npm run lint && npm run typecheck"
+   "quality": "gz-quality run",
+   "quality:check": "gz-quality check",
+   "quality:fix": "gz-quality run --fix"
  }
}
```

**Step 3: 병행 사용 패턴**

점진적 마이그레이션을 위해:

```json
{
  "scripts": {
    "format": "prettier --write .",
    "lint": "eslint --fix .",
    "quality:old": "npm run format && npm run lint",
    "quality:new": "gz-quality run",
    "quality": "gz-quality run"  // 새 방식 우선
  }
}
```

---

### 4. Makefile에서 마이그레이션

#### Before: Makefile targets

```makefile
.PHONY: fmt lint quality

fmt:
	gofumpt -w .
	goimports -w .
	black .
	prettier --write .

lint:
	golangci-lint run
	ruff check .
	eslint .

quality: fmt lint
	@echo "Quality checks complete"
```

**문제:**
- 각 도구를 순차 실행
- 도구 설치 여부 확인 어려움
- Git staged 파일만 처리하기 복잡

#### After: Makefile with gz-quality

```makefile
.PHONY: quality quality-check quality-fix

quality: ## Run all quality checks
	gz-quality run

quality-check: ## Check only (no modifications)
	gz-quality check

quality-fix: ## Run with auto-fix
	gz-quality run --fix

quality-staged: ## Check staged files only
	gz-quality run --staged
```

**장점:**
- 단순화된 targets
- 일관된 인터페이스
- 병렬 실행

#### 마이그레이션 단계

**Step 1: 기존 targets 백업**

```makefile
# 기존 targets를 legacy- prefix로 보존
.PHONY: legacy-fmt legacy-lint

legacy-fmt:
	gofumpt -w .
	black .
	prettier --write .

legacy-lint:
	golangci-lint run
	ruff check .
	eslint .
```

**Step 2: 새 targets 추가**

```makefile
# gz-quality targets
.PHONY: quality quality-check quality-staged

quality: ## Run all quality checks (formatters + linters)
	@echo "Running quality checks..."
	gz-quality run

quality-check: ## Check only (no file modifications)
	@echo "Running quality checks (check only)..."
	gz-quality check

quality-staged: ## Check staged files only (for pre-commit)
	@echo "Checking staged files..."
	gz-quality run --staged --fix
```

**Step 3: CI/CD 통합**

```makefile
ci-quality: ## Quality checks for CI/CD
	gz-quality check --since $(BASE_BRANCH) --report json --output quality-report.json
```

**전체 예시:**

```makefile
# Variables
BASE_BRANCH ?= main
QUALITY_REPORT ?= quality-report.json

# Quality targets
.PHONY: quality quality-check quality-fix quality-staged ci-quality

quality: ## Run all quality checks
	gz-quality run

quality-check: ## Check only (no modifications)
	gz-quality check

quality-fix: ## Run with auto-fix
	gz-quality run --fix

quality-staged: ## Check staged files only
	gz-quality run --staged --fix

ci-quality: ## Quality checks for CI/CD
	gz-quality check \
		--since $(BASE_BRANCH) \
		--report json \
		--output $(QUALITY_REPORT)

# Legacy targets (for rollback)
.PHONY: legacy-quality

legacy-quality:
	gofumpt -w .
	golangci-lint run
	black .
	ruff check --fix .

# Help target
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
```

---

### 5. CI/CD 파이프라인 마이그레이션

#### Before: GitHub Actions (개별 도구)

```yaml
# .github/workflows/quality.yml
name: Quality Checks

on: [push, pull_request]

jobs:
  golang:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: golangci/golangci-lint-action@v6

  python:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: psf/black@stable
      - uses: chartboost/ruff-action@v1

  javascript:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: npm install
      - run: npm run lint
```

**문제:**
- 3개의 job, 3배 느림
- 각 job마다 checkout 필요
- 설정 중복

#### After: GitHub Actions (통합)

```yaml
# .github/workflows/quality.yml
name: Quality Checks

on: [push, pull_request]

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install gz-quality
        run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest

      - name: Run quality checks
        run: |
          BASE_SHA=${{ github.event.pull_request.base.sha || 'main' }}
          gz-quality check --since $BASE_SHA --report json --output quality-report.json

      - name: Upload report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: quality-report
          path: quality-report.json
```

**장점:**
- 단일 job, 빠른 실행
- 한 번의 checkout
- 통합 리포트

#### 마이그레이션 단계

**Step 1: 새 workflow 추가 (병행 운영)**

```bash
# 기존 workflow 보존
mv .github/workflows/quality.yml .github/workflows/quality-old.yml

# 새 workflow 생성
cat > .github/workflows/quality.yml << 'EOF'
name: Quality Checks

on:
  push:
    branches: [main, master, develop]
  pull_request:
    branches: [main, master, develop]

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history for --since

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install gz-quality
        run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest

      - name: Run quality checks
        run: |
          if [ "${{ github.event_name }}" == "pull_request" ]; then
            BASE_SHA=${{ github.event.pull_request.base.sha }}
          else
            BASE_SHA="${{ github.event.before }}"
          fi
          gz-quality check --since $BASE_SHA

      - name: Generate report
        if: always()
        run: gz-quality check --report json --output quality-report.json

      - name: Upload report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: quality-report
          path: quality-report.json
EOF
```

**Step 2: 검증 기간**

```yaml
# 두 workflow 동시 실행 (비교 검증)
# quality-old.yml: 기존 방식
# quality.yml: 새 방식

# 1-2주 동안 결과 비교
# 문제 없으면 quality-old.yml 제거
```

**Step 3: GitLab CI/CD**

```yaml
# .gitlab-ci.yml
quality:
  stage: test
  image: golang:1.24
  before_script:
    - go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest
  script:
    - gz-quality check --since $CI_MERGE_REQUEST_TARGET_BRANCH_SHA
  artifacts:
    reports:
      junit: quality-report.json
    when: always
```

---

## 점진적 마이그레이션 전략

### 4단계 접근법

#### Phase 1: 병행 운영 (1-2주)

```bash
# 기존 방식 유지
make lint          # 기존 Makefile target

# 새 방식 추가
make quality-new   # gz-quality 사용
```

**목표:** 결과 비교, 문제 파악

#### Phase 2: 로컬 전환 (1주)

```bash
# 개발자 로컬 환경에서 gz-quality 사용
gz-quality run --staged  # 커밋 전
```

**목표:** 팀원들의 피드백 수집

#### Phase 3: CI/CD 전환 (1주)

```yaml
# CI/CD에서 gz-quality 사용
- name: Quality Checks
  run: gz-quality check --since main
```

**목표:** CI/CD 안정성 확인

#### Phase 4: 완전 전환

```bash
# 기존 설정 제거
rm .pre-commit-config.yaml
# Makefile에서 legacy targets 제거
```

**목표:** 완전한 마이그레이션

### 롤백 포인트

각 단계마다 롤백 가능:

```bash
# Phase 1 → 롤백: gz-quality 제거만
# Phase 2 → 롤백: 로컬 설정만 복원
# Phase 3 → 롤백: CI/CD workflow 복원
# Phase 4 → 롤백: 백업 파일 복원
```

---

## 롤백 계획

### 빠른 롤백 (긴급)

```bash
# 1. gz-quality 비활성화
git config core.hooksPath .git/hooks  # 기본 hooks로 복원

# 2. 기존 설정 복원
mv .pre-commit-config.yaml.backup .pre-commit-config.yaml
pre-commit install

# 3. CI/CD 복원
git checkout HEAD~1 -- .github/workflows/quality.yml
git commit -m "Rollback to old quality checks"
```

### 부분 롤백 (특정 도구만)

```yaml
# .gzquality.yml
tools:
  golangci-lint:
    enabled: false  # gz-quality에서 비활성화

# Makefile에서 개별 실행
golangci-lint-direct:
	golangci-lint run
```

### 점진적 복원

```bash
# Step 1: 로컬 환경만 복원
make legacy-quality

# Step 2: 문제 파악 후 결정
# - gz-quality 설정 조정
# - 또는 완전 롤백
```

---

## 체크리스트

### 마이그레이션 전

- [ ] 현재 품질 도구 목록 작성
- [ ] 기존 설정 파일 백업
- [ ] gz-quality 설치 및 테스트
- [ ] 팀원들에게 계획 공유

### 마이그레이션 중

- [ ] `.gzquality.yml` 작성
- [ ] 로컬에서 결과 비교 (기존 vs 새 방식)
- [ ] Git hooks 설정
- [ ] CI/CD workflow 업데이트
- [ ] 1-2주 병행 운영

### 마이그레이션 후

- [ ] 기존 설정 파일 제거
- [ ] 문서 업데이트 (README, CONTRIBUTING)
- [ ] 팀원 교육 및 피드백 수집
- [ ] 성능 개선 확인 (실행 시간 비교)

---

## FAQ

### Q1: 기존 설정을 그대로 사용할 수 있나요?

**A:** 네! gz-quality는 각 도구의 기존 설정 파일을 그대로 사용합니다.

```yaml
# .gzquality.yml
tools:
  golangci-lint:
    config_file: ".golangci.yml"  # 기존 설정 재사용
  prettier:
    config_file: ".prettierrc"
  eslint:
    config_file: ".eslintrc.json"
```

### Q2: 일부 도구만 마이그레이션할 수 있나요?

**A:** 네! 선택적으로 활성화/비활성화 가능합니다.

```yaml
# .gzquality.yml
tools:
  golangci-lint:
    enabled: true   # gz-quality로 실행
  ruff:
    enabled: false  # 직접 실행
```

### Q3: 성능이 정말 빠른가요?

**A:** 네! 병렬 실행으로 평균 50% 이상 빠릅니다.

```bash
# 기존 방식 (순차 실행)
$ time (gofumpt -w . && golangci-lint run && black . && ruff check .)
# 실행 시간: 45초

# gz-quality (병렬 실행)
$ time gz-quality run
# 실행 시간: 20초 (56% 단축)
```

### Q4: 롤백이 어렵지 않나요?

**A:** 매우 쉽습니다. 기존 설정 파일을 삭제하지 않고 보존하면 즉시 롤백 가능합니다.

```bash
# 기존 설정 보존
mv .pre-commit-config.yaml .pre-commit-config.yaml.backup

# 문제 발생 시 즉시 복원
mv .pre-commit-config.yaml.backup .pre-commit-config.yaml
```

---

## 추가 리소스

- [빠른 시작 가이드](./00-quick-start.md)
- [설정 가이드](./03-configuration.md)
- [Cookbook (실전 예제)](./08-cookbook.md)
- [CI/CD 통합 가이드](../integration/CI_INTEGRATION.md)
- [Pre-commit Hooks 가이드](../integration/PRE_COMMIT_HOOKS.md)

---

**마이그레이션 지원이 필요하신가요?**

- GitHub Issues: https://github.com/Gizzahub/gzh-cli-quality/issues
- GitHub Discussions: https://github.com/Gizzahub/gzh-cli-quality/discussions
