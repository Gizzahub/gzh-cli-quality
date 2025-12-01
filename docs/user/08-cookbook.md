# Cookbook - 실전 레시피

gzh-cli-quality를 실제 프로젝트에서 사용하는 구체적인 방법과 패턴을 제공합니다. 각 레시피는 복사-붙여넣기로 바로 사용 가능합니다.

## 목차

- [프로젝트 타입별 레시피](#프로젝트-타입별-레시피)
  - [1. 모노레포 (Monorepo)](#1-모노레포-monorepo)
  - [2. 마이크로서비스](#2-마이크로서비스)
  - [3. 풀스택 프로젝트](#3-풀스택-프로젝트)
  - [4. 레거시 코드베이스](#4-레거시-코드베이스)
- [워크플로우별 레시피](#워크플로우별-레시피)
  - [5. 빠른 커밋 전 검사](#5-빠른-커밋-전-검사)
  - [6. PR 리뷰 자동화](#6-pr-리뷰-자동화)
  - [7. Nightly 전체 검사](#7-nightly-전체-검사)
- [팀 협업 레시피](#팀-협업-레시피)
  - [8. 점진적 팀 도입](#8-점진적-팀-도입)
  - [9. 코드 리뷰 체크리스트](#9-코드-리뷰-체크리스트)
  - [10. 품질 메트릭 추적](#10-품질-메트릭-추적)

---

## 프로젝트 타입별 레시피

### 1. 모노레포 (Monorepo)

**시나리오:** 하나의 리포지토리에 여러 프로젝트/서비스가 있는 경우

#### 1.1 기본 구조

```
monorepo/
├── services/
│   ├── api/          # Go backend
│   ├── web/          # TypeScript frontend
│   └── worker/       # Python worker
├── libs/
│   ├── shared/       # TypeScript shared
│   └── utils/        # Go utils
└── .gzquality.yml    # 전체 설정
```

#### 1.2 전체 모노레포 설정

```yaml
# .gzquality.yml (root level)
default_workers: 8  # 모노레포는 더 많은 워커 사용

tools:
  # Go tools
  gofumpt:
    enabled: true
    priority: 10
  golangci-lint:
    enabled: true
    priority: 5
    config_file: ".golangci.yml"

  # Python tools
  black:
    enabled: true
    priority: 10
  ruff:
    enabled: true
    priority: 7

  # JavaScript/TypeScript tools
  prettier:
    enabled: true
    priority: 10
  eslint:
    enabled: true
    priority: 5
  tsc:
    enabled: true
    priority: 3

languages:
  Go:
    enabled: true
    extensions: [.go]
  Python:
    enabled: true
    extensions: [.py, .pyi]
  TypeScript:
    enabled: true
    extensions: [.ts, .tsx]
  JavaScript:
    enabled: true
    extensions: [.js, .jsx]

exclude:
  # 각 서비스의 빌드 아웃풋
  - "services/*/dist/**"
  - "services/*/build/**"
  - "services/*/.next/**"

  # 공통 제외
  - "node_modules/**"
  - "vendor/**"
  - ".venv/**"
  - "**/__pycache__/**"
```

#### 1.3 서비스별 검사 (선택적)

```bash
# 특정 서비스만 검사
cd services/api
gz-quality run

# 또는 루트에서 특정 경로
gz-quality run services/api

# 여러 서비스 동시 검사 (병렬)
gz-quality run services/api services/web
```

#### 1.4 변경된 서비스만 검사 (최적화)

```bash
# PR에서 변경된 파일만 검사
gz-quality run --since origin/main

# 스크립트로 변경된 서비스 감지
#!/bin/bash
# scripts/check-changed-services.sh

CHANGED_SERVICES=$(git diff --name-only origin/main | \
  grep '^services/' | \
  cut -d'/' -f2 | \
  sort -u)

for service in $CHANGED_SERVICES; do
  echo "Checking service: $service"
  gz-quality run "services/$service"
done
```

#### 1.5 Makefile 통합

```makefile
# Makefile (root)

# 전체 모노레포 검사
quality:
	gz-quality run

# 서비스별 검사
quality-api:
	gz-quality run services/api

quality-web:
	gz-quality run services/web

quality-worker:
	gz-quality run services/worker

# 변경된 것만 검사 (PR용)
quality-changed:
	gz-quality run --since origin/main

# 병렬 검사 (개별 프로세스)
quality-parallel:
	@echo "Running parallel checks..."
	@(gz-quality run services/api &)
	@(gz-quality run services/web &)
	@(gz-quality run services/worker &)
	@wait
	@echo "All checks complete"
```

---

### 2. 마이크로서비스

**시나리오:** 독립적인 리포지토리를 가진 여러 마이크로서비스

#### 2.1 공통 설정 공유

```bash
# 중앙 설정 리포지토리 생성
git clone https://github.com/yourorg/config-shared.git

# 각 서비스에서 심볼릭 링크
cd user-service
ln -s ../config-shared/.gzquality.yml .gzquality.yml
ln -s ../config-shared/.golangci.yml .golangci.yml
```

#### 2.2 템플릿 설정 (git submodule)

```bash
# 메인 리포지토리에 submodule 추가
git submodule add https://github.com/yourorg/quality-config .quality-config

# .gzquality.yml에서 참조
cat > .gzquality.yml << 'EOF'
# Base configuration from shared config
tools:
  golangci-lint:
    config_file: ".quality-config/.golangci.yml"
  prettier:
    config_file: ".quality-config/.prettierrc"
  eslint:
    config_file: ".quality-config/.eslintrc.json"

# Service-specific overrides
tools:
  golangci-lint:
    args:
      - "--timeout=5m"  # This service needs more time

exclude:
  - "generated/**"  # This service has generated code
EOF
```

#### 2.3 CI/CD 통합 (GitHub Actions)

```yaml
# .github/workflows/quality.yml
# 모든 마이크로서비스에서 동일하게 사용

name: Quality Checks

on:
  push:
    branches: [main, develop]
  pull_request:

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
          submodules: true  # 공통 설정 submodule

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install gz-quality
        run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest

      - name: Run quality checks
        run: |
          if [ "${{ github.event_name }}" == "pull_request" ]; then
            gz-quality check --since ${{ github.event.pull_request.base.sha }}
          else
            gz-quality check
          fi

      - name: Generate report
        if: always()
        run: gz-quality check --report json --output quality-report.json

      - name: Upload report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: quality-report
          path: quality-report.json
```

---

### 3. 풀스택 프로젝트

**시나리오:** Frontend + Backend가 하나의 리포지토리에 있는 경우

#### 3.1 프로젝트 구조

```
fullstack-app/
├── backend/          # Go/Python backend
│   ├── cmd/
│   ├── internal/
│   └── go.mod
├── frontend/         # React/Vue frontend
│   ├── src/
│   ├── package.json
│   └── tsconfig.json
├── .gzquality.yml    # 공통 설정
└── Makefile
```

#### 3.2 설정 파일

```yaml
# .gzquality.yml
default_workers: 6

tools:
  # Backend (Go)
  gofumpt:
    enabled: true
    priority: 10
  golangci-lint:
    enabled: true
    priority: 5
    config_file: "backend/.golangci.yml"

  # Frontend (TypeScript/React)
  prettier:
    enabled: true
    priority: 10
    config_file: "frontend/.prettierrc"
  eslint:
    enabled: true
    priority: 5
    config_file: "frontend/.eslintrc.json"
  tsc:
    enabled: true
    priority: 3
    config_file: "frontend/tsconfig.json"

# 명시적 include (frontend/backend만)
include:
  - "backend/**/*.go"
  - "frontend/src/**/*.{ts,tsx,js,jsx}"

exclude:
  - "frontend/dist/**"
  - "frontend/build/**"
  - "frontend/node_modules/**"
  - "backend/vendor/**"
```

#### 3.3 Makefile

```makefile
# Makefile
.PHONY: quality quality-backend quality-frontend

# 전체 검사
quality:
	gz-quality run

# Backend만
quality-backend:
	cd backend && gz-quality run

# Frontend만
quality-frontend:
	cd frontend && gz-quality run

# 커밋 전 검사 (staged files)
quality-staged:
	gz-quality run --staged --fix

# PR 검사
quality-pr:
	gz-quality check --since origin/main
```

#### 3.4 Pre-commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

set -e

echo "🔍 Checking code quality..."

# Backend 변경 확인
if git diff --cached --name-only | grep -q '^backend/'; then
  echo "  📦 Backend changes detected"
fi

# Frontend 변경 확인
if git diff --cached --name-only | grep -q '^frontend/'; then
  echo "  🎨 Frontend changes detected"
fi

# 통합 검사 (병렬)
gz-quality run --staged --fix

# 수정된 파일 재 스테이징
git diff --name-only --cached | xargs -r git add

echo "✅ Quality checks passed!"
```

---

### 4. 레거시 코드베이스

**시나리오:** 큰 레거시 프로젝트에 품질 도구 점진적 도입

#### 4.1 점진적 활성화 전략

**Step 1: 포매터만 활성화 (1주차)**

```yaml
# .gzquality.yml
tools:
  # 포매터만 활성화 (파일 수정, 논란 적음)
  gofumpt:
    enabled: true
  black:
    enabled: true
  prettier:
    enabled: true
  rustfmt:
    enabled: true

  # 린터는 비활성화 (너무 많은 이슈)
  golangci-lint:
    enabled: false
  ruff:
    enabled: false
  eslint:
    enabled: false
  clippy:
    enabled: false
```

**Step 2: 새 파일만 린팅 (2주차)**

```bash
# 최근 2주간 변경된 파일만 검사
gz-quality check --since $(git log --since="2 weeks ago" --format=%H | tail -1)
```

**Step 3: 점진적 린터 활성화 (3-4주차)**

```yaml
# .gzquality.yml
tools:
  golangci-lint:
    enabled: true
    args:
      # 기본 린터만 활성화 (에러 최소화)
      - "--disable-all"
      - "--enable=errcheck,ineffassign,unused,govet"
      - "--max-issues-per-linter=10"  # 이슈 제한
```

#### 4.2 디렉토리별 제외 (점진적 적용)

```yaml
# .gzquality.yml
exclude:
  # 레거시 모듈 (당장 손대지 않음)
  - "legacy/**"
  - "deprecated/**"
  - "old-*/**"

  # 외부 코드
  - "third_party/**"
  - "vendor/**"
  - "node_modules/**"

  # 생성된 코드
  - "**/*.generated.*"
  - "**/*_gen.go"
  - "**/*_pb.go"

# 새 모듈만 검사
include:
  - "src/new-features/**"
  - "services/v2/**"
```

#### 4.3 경고만 출력 (실패하지 않음)

```bash
# CI/CD에서 실패하지 않도록
gz-quality check || echo "Quality issues found, but not failing build"

# 또는 continue-on-error 사용 (GitHub Actions)
```

```yaml
# .github/workflows/quality.yml
- name: Quality checks (non-blocking)
  continue-on-error: true
  run: gz-quality check
```

#### 4.4 주간 리포트 (점진적 개선 추적)

```bash
#!/bin/bash
# scripts/weekly-quality-report.sh

echo "📊 Weekly Quality Report ($(date))"
echo "=================================="

# 전체 이슈 수
TOTAL_ISSUES=$(gz-quality check --report json 2>/dev/null | \
  jq '[.results[].issues | length] | add')

echo "Total issues: $TOTAL_ISSUES"

# 언어별 이슈
gz-quality check --report json 2>/dev/null | \
  jq -r '.results[] | "\(.language): \(.issues | length) issues"'

# 이전 주와 비교
if [ -f "quality-report-last-week.json" ]; then
  LAST_WEEK=$(jq '[.results[].issues | length] | add' quality-report-last-week.json)
  DIFF=$((TOTAL_ISSUES - LAST_WEEK))

  if [ $DIFF -lt 0 ]; then
    echo "✅ Improvement: $((DIFF * -1)) issues fixed this week!"
  else
    echo "⚠️  Regression: $DIFF new issues this week"
  fi
fi

# 현재 리포트 저장
gz-quality check --report json --output quality-report-last-week.json
```

---

## 워크플로우별 레시피

### 5. 빠른 커밋 전 검사

**목표:** 커밋 전 1-2초 내 빠른 피드백

#### 5.1 최적화된 Pre-commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

set -e

# staged 파일 개수 확인
STAGED_COUNT=$(git diff --cached --name-only | wc -l)

if [ $STAGED_COUNT -eq 0 ]; then
  echo "No staged files"
  exit 0
fi

echo "🔍 Checking $STAGED_COUNT staged files..."

# 포매터만 실행 (빠름)
gz-quality run --staged --format-only --fix

# 수정된 파일 재 스테이징
git diff --name-only --cached | xargs -r git add

echo "✅ Format checks passed ($STAGED_COUNT files)"

# 린트는 선택적 (환경 변수로 제어)
if [ "$GZ_QUALITY_SKIP_LINT" != "1" ]; then
  echo "🔍 Running linters..."
  gz-quality check --staged --lint-only || {
    echo "⚠️  Lint issues found. Fix them or set GZ_QUALITY_SKIP_LINT=1 to skip"
    exit 1
  }
fi

echo "✅ All checks passed!"
```

#### 5.2 빠른 검사 설정

```yaml
# .gzquality.yml
default_workers: 8  # 최대 병렬화

tools:
  # 빠른 포매터 우선
  gofumpt:
    priority: 10
  prettier:
    priority: 10

  # 느린 린터는 낮은 우선순위
  golangci-lint:
    priority: 1
    args:
      - "--fast"  # 빠른 모드
```

#### 5.3 커밋 템플릿

```bash
# .git/hooks/prepare-commit-msg

# 품질 검사 결과를 커밋 메시지에 추가
if [ -f ".quality-check-result" ]; then
  echo "" >> "$1"
  echo "# Quality Check Results:" >> "$1"
  cat .quality-check-result >> "$1"
fi
```

---

### 6. PR 리뷰 자동화

**목표:** PR에서 자동으로 코드 품질 리포트 생성

#### 6.1 GitHub Actions Workflow

```yaml
# .github/workflows/pr-quality.yml
name: PR Quality Check

on:
  pull_request:
    types: [opened, synchronize]

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install gz-quality
        run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest

      - name: Run quality checks
        id: quality
        continue-on-error: true
        run: |
          gz-quality check \
            --since ${{ github.event.pull_request.base.sha }} \
            --report json \
            --output quality-report.json

          # Exit code 저장
          echo "exit_code=$?" >> $GITHUB_OUTPUT

      - name: Parse quality report
        id: parse
        run: |
          TOTAL_ISSUES=$(jq '[.results[].issues | length] | add // 0' quality-report.json)
          echo "total_issues=$TOTAL_ISSUES" >> $GITHUB_OUTPUT

          # 언어별 이슈 수
          jq -r '.results[] | "\(.language): \(.issues | length)"' quality-report.json > issues-by-lang.txt
          echo "issues_by_lang<<EOF" >> $GITHUB_OUTPUT
          cat issues-by-lang.txt >> $GITHUB_OUTPUT
          echo "EOF" >> $GITHUB_OUTPUT

      - name: Comment PR
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const report = JSON.parse(fs.readFileSync('quality-report.json', 'utf8'));
            const totalIssues = ${{ steps.parse.outputs.total_issues }};

            let body = '## 📊 Code Quality Report\n\n';

            if (totalIssues === 0) {
              body += '✅ **No quality issues found!** Great work!\n\n';
            } else {
              body += `⚠️ **Found ${totalIssues} issue(s)**\n\n`;
              body += '### Issues by Language\n\n';
              body += '${{ steps.parse.outputs.issues_by_lang }}'.split('\n').map(l => `- ${l}`).join('\n');
              body += '\n\n';

              // 상위 5개 이슈 표시
              body += '### Top Issues\n\n';
              let issueCount = 0;
              for (const result of report.results) {
                for (const issue of result.issues.slice(0, 5)) {
                  body += `- **${issue.file}:${issue.line}** (${issue.tool}): ${issue.message}\n`;
                  issueCount++;
                  if (issueCount >= 5) break;
                }
                if (issueCount >= 5) break;
              }
            }

            body += '\n---\n';
            body += '💡 *Run `gz-quality run --fix` locally to auto-fix some issues*\n';

            // 기존 코멘트 찾기
            const comments = await github.rest.issues.listComments({
              owner: context.repo.owner,
              repo: context.repo.repo,
              issue_number: context.issue.number,
            });

            const existingComment = comments.data.find(
              c => c.user.login === 'github-actions[bot]' && c.body.includes('Code Quality Report')
            );

            if (existingComment) {
              // 업데이트
              await github.rest.issues.updateComment({
                owner: context.repo.owner,
                repo: context.repo.repo,
                comment_id: existingComment.id,
                body
              });
            } else {
              // 새로 생성
              await github.rest.issues.createComment({
                owner: context.repo.owner,
                repo: context.repo.repo,
                issue_number: context.issue.number,
                body
              });
            }

      - name: Fail if issues found
        if: steps.parse.outputs.total_issues > 0
        run: |
          echo "❌ Quality checks failed with ${{ steps.parse.outputs.total_issues }} issues"
          exit 1
```

---

### 7. Nightly 전체 검사

**목표:** 매일 밤 전체 코드베이스 검사 및 트렌드 추적

#### 7.1 GitHub Actions Schedule

```yaml
# .github/workflows/nightly-quality.yml
name: Nightly Quality Check

on:
  schedule:
    - cron: '0 2 * * *'  # 매일 오전 2시 (UTC)
  workflow_dispatch:  # 수동 실행 가능

jobs:
  full-quality-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install gz-quality
        run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest

      - name: Run full quality check
        run: |
          gz-quality check \
            --report json \
            --output quality-report-$(date +%Y%m%d).json

      - name: Generate HTML report
        run: |
          gz-quality check \
            --report html \
            --output quality-report-$(date +%Y%m%d).html

      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: quality-reports-$(date +%Y%m%d)
          path: quality-report-*
          retention-days: 90

      - name: Track metrics
        run: |
          TOTAL_ISSUES=$(jq '[.results[].issues | length] | add // 0' quality-report-*.json)
          echo "total_issues=$TOTAL_ISSUES" >> metrics.txt

          # 메트릭 파일에 추가
          echo "$(date +%Y-%m-%d),$TOTAL_ISSUES" >> quality-metrics.csv

      - name: Commit metrics
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add quality-metrics.csv
          git commit -m "chore: update nightly quality metrics [skip ci]"
          git push

      - name: Send notification
        if: failure()
        uses: actions/github-script@v7
        with:
          script: |
            await github.rest.issues.create({
              owner: context.repo.owner,
              repo: context.repo.repo,
              title: 'Nightly Quality Check Failed',
              body: 'The nightly quality check has failed. Please review the artifacts.',
              labels: ['quality', 'automated']
            });
```

---

## 팀 협업 레시피

### 8. 점진적 팀 도입

**목표:** 팀 전체가 거부감 없이 도구를 도입

#### 8.1 4주 도입 계획

**Week 1: 오리엔테이션**

```bash
# 팀 미팅: 도구 소개 (30분)
# - gz-quality 데모
# - 기존 방식 vs 새 방식 비교
# - 성능 벤치마크 공유

# 자발적 참여자 모집
# - 2-3명의 early adopters
# - 로컬에서만 사용
```

**Week 2: 파일럿 프로그램**

```yaml
# .gzquality.yml (관대한 설정)
tools:
  # 포매터만 활성화 (자동 수정)
  gofumpt: {enabled: true}
  black: {enabled: true}
  prettier: {enabled: true}

  # 린터는 비활성화
  golangci-lint: {enabled: false}
  ruff: {enabled: false}
  eslint: {enabled: false}

# 파일럿 참여자만 pre-commit hook 설정
```

**Week 3: 전체 배포 (선택적)**

```bash
# 팀 전체 알림
# - 로컬에서 선택적 사용 가능
# - CI/CD는 아직 비활성화

# 설치 스크립트 제공
cat > scripts/setup-quality.sh << 'EOF'
#!/bin/bash
echo "Installing gz-quality..."
go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest

echo "Setting up pre-commit hook (optional)..."
read -p "Install pre-commit hook? [y/N] " -n 1 -r
if [[ $REPLY =~ ^[Yy]$ ]]; then
  cp hooks/pre-commit .git/hooks/pre-commit
  chmod +x .git/hooks/pre-commit
  echo "✅ Pre-commit hook installed"
fi

echo "✅ Setup complete!"
echo "Run: gz-quality run --staged"
EOF
```

**Week 4: CI/CD 통합**

```yaml
# .github/workflows/quality.yml (non-blocking)
- name: Quality checks
  continue-on-error: true  # 실패해도 빌드 성공
  run: gz-quality check --since main
```

#### 8.2 팀 피드백 수집

```bash
# Google Form 또는 GitHub Discussion
# 질문 예시:
# 1. gz-quality 사용이 개발 속도에 영향을 주었나요?
# 2. 가장 유용한 기능은?
# 3. 개선이 필요한 부분은?
# 4. CI/CD 통합을 언제 활성화하면 좋을까요?
```

---

### 9. 코드 리뷰 체크리스트

**목표:** PR 리뷰 시 품질 검사 통합

#### 9.1 PR 템플릿

```markdown
# Pull Request Template

## Changes
<!-- 변경 사항 설명 -->

## Quality Checks

### Automated Checks
- [ ] gz-quality checks passed
- [ ] Test coverage maintained/improved
- [ ] No new linting warnings

### Manual Review
- [ ] Code follows team conventions
- [ ] Documentation updated
- [ ] Breaking changes documented

## Quality Report
<!-- GitHub Actions가 자동으로 코멘트 추가 -->

## Reviewer Notes
<!-- 리뷰어를 위한 특별 노트 -->
```

#### 9.2 리뷰어 체크스크립트

```bash
#!/bin/bash
# scripts/review-pr.sh <PR_NUMBER>

PR_NUMBER=$1

echo "📋 PR #$PR_NUMBER Review Checklist"
echo "================================="

# PR 정보 가져오기
gh pr view $PR_NUMBER --json title,author,additions,deletions

# 품질 검사 실행
echo ""
echo "🔍 Running quality checks..."
gh pr checkout $PR_NUMBER
gz-quality check --since main

# 테스트 커버리지 확인
echo ""
echo "📊 Test coverage..."
make test-coverage

# 변경된 파일 크기 확인
echo ""
echo "📏 Changed files size..."
git diff main --stat | awk '{if ($3 > 500) print "⚠️  Large change:", $1, $3}'

# 리뷰 제안
echo ""
echo "💡 Review suggestions:"
echo "- Check for over-engineering"
echo "- Verify error handling"
echo "- Look for security issues"
```

---

### 10. 품질 메트릭 추적

**목표:** 코드 품질을 수치화하고 개선 추적

#### 10.1 메트릭 수집 스크립트

```bash
#!/bin/bash
# scripts/collect-metrics.sh

OUTPUT_DIR="metrics"
DATE=$(date +%Y-%m-%d)

mkdir -p $OUTPUT_DIR

echo "📊 Collecting quality metrics..."

# 1. 품질 이슈 수
gz-quality check --report json --output $OUTPUT_DIR/quality-$DATE.json
TOTAL_ISSUES=$(jq '[.results[].issues | length] | add // 0' $OUTPUT_DIR/quality-$DATE.json)

# 2. 코드 줄 수
TOTAL_LINES=$(find . -name "*.go" -o -name "*.py" -o -name "*.ts" | xargs wc -l | tail -1 | awk '{print $1}')

# 3. 테스트 커버리지
COVERAGE=$(go test -coverprofile=coverage.out ./... 2>&1 | grep "coverage:" | awk '{print $2}')

# 4. 메트릭 CSV에 추가
echo "$DATE,$TOTAL_ISSUES,$TOTAL_LINES,$COVERAGE" >> $OUTPUT_DIR/metrics.csv

# 5. 리포트 생성
cat > $OUTPUT_DIR/report-$DATE.md << EOF
# Quality Metrics Report - $DATE

## Summary
- **Total Issues**: $TOTAL_ISSUES
- **Total Lines**: $TOTAL_LINES
- **Test Coverage**: $COVERAGE

## Trend
\`\`\`
$(tail -5 $OUTPUT_DIR/metrics.csv)
\`\`\`

## Issues by Language
\`\`\`
$(jq -r '.results[] | "\(.language): \(.issues | length)"' $OUTPUT_DIR/quality-$DATE.json)
\`\`\`

## Top Issues
\`\`\`
$(jq -r '.results[].issues | .[:5][] | "- \(.file):\(.line) [\(.tool)] \(.message)"' $OUTPUT_DIR/quality-$DATE.json)
\`\`\`
EOF

echo "✅ Metrics saved to $OUTPUT_DIR/"
echo "   - quality-$DATE.json"
echo "   - report-$DATE.md"
echo "   - metrics.csv"
```

#### 10.2 대시보드 (Grafana/Prometheus 스타일)

```python
#!/usr/bin/env python3
# scripts/generate-dashboard.py

import json
import matplotlib.pyplot as plt
import pandas as pd
from datetime import datetime, timedelta

# CSV 읽기
df = pd.read_csv('metrics/metrics.csv',
                 names=['date', 'issues', 'lines', 'coverage'],
                 parse_dates=['date'])

# 최근 30일 데이터
df = df[df['date'] > datetime.now() - timedelta(days=30)]

# 플롯 생성
fig, axes = plt.subplots(2, 2, figsize=(15, 10))

# 1. 이슈 트렌드
axes[0, 0].plot(df['date'], df['issues'], marker='o')
axes[0, 0].set_title('Quality Issues Trend')
axes[0, 0].set_ylabel('Number of Issues')
axes[0, 0].grid(True)

# 2. 코드 증가율
axes[0, 1].plot(df['date'], df['lines'], marker='s', color='green')
axes[0, 1].set_title('Code Lines Trend')
axes[0, 1].set_ylabel('Lines of Code')
axes[0, 1].grid(True)

# 3. 커버리지
coverage_pct = df['coverage'].str.rstrip('%').astype(float)
axes[1, 0].plot(df['date'], coverage_pct, marker='^', color='orange')
axes[1, 0].set_title('Test Coverage Trend')
axes[1, 0].set_ylabel('Coverage (%)')
axes[1, 0].grid(True)

# 4. 이슈 밀도 (이슈/1000줄)
issue_density = (df['issues'] / df['lines'] * 1000)
axes[1, 1].plot(df['date'], issue_density, marker='D', color='red')
axes[1, 1].set_title('Issue Density (per 1000 lines)')
axes[1, 1].set_ylabel('Issues per 1000 lines')
axes[1, 1].grid(True)

plt.tight_layout()
plt.savefig('metrics/dashboard.png', dpi=150)
print("✅ Dashboard saved to metrics/dashboard.png")
```

---

## 고급 팁

### 커스텀 리포터

```go
// scripts/custom-reporter.go
// gz-quality의 JSON 출력을 Slack/Discord로 전송

package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type QualityReport struct {
	Results []Result `json:"results"`
}

type Result struct {
	Language string  `json:"language"`
	Issues   []Issue `json:"issues"`
}

type Issue struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Tool    string `json:"tool"`
	Message string `json:"message"`
}

func main() {
	data, _ := os.ReadFile("quality-report.json")

	var report QualityReport
	json.Unmarshal(data, &report)

	totalIssues := 0
	for _, result := range report.Results {
		totalIssues += len(result.Issues)
	}

	// Slack webhook 전송
	message := fmt.Sprintf("🔍 Quality Report: %d issues found", totalIssues)
	// sendToSlack(message)

	fmt.Println(message)
}
```

---

## 추가 리소스

- [빠른 시작](./00-quick-start.md)
- [Migration Guide](./07-migration.md)
- [CI/CD Integration](../integration/CI_INTEGRATION.md)
- [Configuration Guide](./03-configuration.md)

---

**더 많은 레시피가 필요하신가요?**

GitHub Discussions에서 공유해주세요: https://github.com/Gizzahub/gzh-cli-quality/discussions
