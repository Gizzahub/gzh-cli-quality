# CI/CD 통합 가이드

`gzh-cli-quality`를 다양한 CI/CD 플랫폼에 통합하는 방법을 안내합니다.

## 목차

- [GitHub Actions](#github-actions)
- [GitLab CI](#gitlab-ci)
- [CircleCI](#circleci)
- [Jenkins](#jenkins)
- [Pre-commit Hooks](#pre-commit-hooks)
- [Docker 통합](#docker-통합)
- [테스트 커버리지 통합](#테스트-커버리지-통합)

---

## GitHub Actions

### 기본 워크플로우

`.github/workflows/quality.yml`:

```yaml
name: Code Quality

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  quality:
    name: Quality Check
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0  # git history for --since

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install gz-quality
        run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1

      - name: Run quality check
        run: gz-quality check --report json --output quality-report.json

      - name: Upload quality report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: quality-report
          path: quality-report.json
```

### Pull Request 변경 파일만 검사

```yaml
- name: Get changed files
  id: changed-files
  run: |
    echo "files=$(git diff --name-only origin/${{ github.base_ref }}...HEAD | tr '\n' ',')" >> $GITHUB_OUTPUT

- name: Run quality check on changed files
  run: gz-quality check --since origin/${{ github.base_ref }}
```

### 매트릭스 빌드 (멀티 플랫폼)

```yaml
strategy:
  matrix:
    os: [ubuntu-latest, macos-latest, windows-latest]
    go: ['1.24']

runs-on: ${{ matrix.os }}
steps:
  - uses: actions/checkout@v4
  - uses: actions/setup-go@v5
    with:
      go-version: ${{ matrix.go }}
  - run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1
  - run: gz-quality check
```

### 코멘트로 결과 표시

```yaml
- name: Run quality check
  id: quality
  continue-on-error: true
  run: |
    gz-quality check --report markdown --output quality-report.md
    cat quality-report.md >> $GITHUB_STEP_SUMMARY

- name: Comment PR
  if: github.event_name == 'pull_request'
  uses: actions/github-script@v7
  with:
    script: |
      const fs = require('fs');
      const report = fs.readFileSync('quality-report.md', 'utf8');
      github.rest.issues.createComment({
        issue_number: context.issue.number,
        owner: context.repo.owner,
        repo: context.repo.repo,
        body: `## 🔍 Quality Check Results\n\n${report}`
      });
```

---

## GitLab CI

`.gitlab-ci.yml`:

```yaml
stages:
  - quality

variables:
  GZQ_VERSION: "latest"

quality-check:
  stage: quality
  image: golang:1.24
  before_script:
    - go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@${GZQ_VERSION}
    - export PATH=$PATH:$(go env GOPATH)/bin
  script:
    - gz-quality check --report json --output quality-report.json
  artifacts:
    reports:
      codequality: quality-report.json
    paths:
      - quality-report.json
    expire_in: 1 week
  only:
    - merge_requests
    - main
    - develop
```

### Merge Request만 검사

```yaml
quality-check:mr:
  extends: quality-check
  script:
    - git fetch origin $CI_MERGE_REQUEST_TARGET_BRANCH_NAME
    - gz-quality check --since origin/$CI_MERGE_REQUEST_TARGET_BRANCH_NAME
  only:
    - merge_requests
```

---

## CircleCI

`.circleci/config.yml`:

```yaml
version: 2.1

executors:
  go-executor:
    docker:
      - image: cimg/go:1.24
    working_directory: ~/project

jobs:
  quality-check:
    executor: go-executor
    steps:
      - checkout
      - restore_cache:
          keys:
            - go-mod-{{ checksum "go.sum" }}
      - run:
          name: Install gz-quality
          command: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1
      - run:
          name: Run quality check
          command: |
            gz-quality check --report json --output /tmp/quality-report.json
      - store_artifacts:
          path: /tmp/quality-report.json
          destination: quality-report
      - store_test_results:
          path: /tmp/quality-report.json

workflows:
  version: 2
  quality:
    jobs:
      - quality-check:
          filters:
            branches:
              only:
                - main
                - develop
```

---

## Jenkins

`Jenkinsfile`:

```groovy
pipeline {
    agent {
        docker {
            image 'golang:1.24'
        }
    }

    environment {
        GOPATH = "${WORKSPACE}/go"
        PATH = "${GOPATH}/bin:${env.PATH}"
    }

    stages {
        stage('Setup') {
            steps {
                sh 'go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1'
            }
        }

        stage('Quality Check') {
            steps {
                sh 'gz-quality check --report json --output quality-report.json'
            }
        }

        stage('Archive Results') {
            steps {
                archiveArtifacts artifacts: 'quality-report.json', fingerprint: true
                publishHTML([
                    reportDir: '.',
                    reportFiles: 'quality-report.html',
                    reportName: 'Quality Report'
                ])
            }
        }
    }

    post {
        always {
            cleanWs()
        }
        failure {
            emailext(
                subject: "Quality Check Failed: ${env.JOB_NAME} - ${env.BUILD_NUMBER}",
                body: "Check console output at ${env.BUILD_URL}",
                to: "${env.CHANGE_AUTHOR_EMAIL}"
            )
        }
    }
}
```

---

## Pre-commit Hooks

### Git Hooks 사용

`.git/hooks/pre-commit`:

```bash
#!/bin/bash
# gz-quality pre-commit hook

set -e

echo "🔍 Running quality checks on staged files..."

# Check if gz-quality is installed
if ! command -v gz-quality &> /dev/null; then
    echo "❌ gz-quality not found. Install it with: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1"
    exit 1
fi

# Run quality check on staged files
if ! gz-quality check --staged; then
    echo "❌ Quality check failed. Please fix the issues before committing."
    exit 1
fi

echo "✅ Quality check passed!"
exit 0
```

**설치**:

```bash
chmod +x .git/hooks/pre-commit
```

### pre-commit 프레임워크

`.pre-commit-config.yaml`:

```yaml
repos:
  - repo: local
    hooks:
      - id: gz-quality-check
        name: gz-quality quality check
        entry: gz-quality check
        language: system
        pass_filenames: false
        always_run: true
        stages: [commit]
```

**설치**:

```bash
pip install pre-commit
pre-commit install
```

### Husky (Node.js 프로젝트)

`package.json`:

```json
{
  "husky": {
    "hooks": {
      "pre-commit": "gz-quality check --staged"
    }
  }
}
```

**설치**:

```bash
npm install --save-dev husky
npx husky install
npx husky add .husky/pre-commit "gz-quality check --staged"
```

---

## Docker 통합

### Dockerfile

```dockerfile
FROM golang:1.24-alpine AS builder

# Install gz-quality
RUN go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1

FROM alpine:latest

# Copy gz-quality from builder
COPY --from=builder /go/bin/gz-quality /usr/local/bin/gz-quality

# Install required tools (optional)
RUN apk add --no-cache \
    git \
    make

WORKDIR /workspace

ENTRYPOINT ["gz-quality"]
CMD ["check"]
```

**빌드 및 사용**:

```bash
# 빌드
docker build -t gz-quality:latest .

# 사용
docker run --rm -v $(pwd):/workspace gz-quality:latest check
docker run --rm -v $(pwd):/workspace gz-quality:latest run --dry-run
```

### Docker Compose

`docker-compose.yml`:

```yaml
version: '3.8'

services:
  quality-check:
    image: golang:1.24
    working_dir: /workspace
    volumes:
      - .:/workspace
      - go-cache:/go
    command: >
      sh -c "
        go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1 &&
        /go/bin/gz-quality check --report json --output quality-report.json
      "

volumes:
  go-cache:
```

**사용**:

```bash
docker-compose run --rm quality-check
```

---

## 고급 설정

### 조건부 실행

```yaml
# GitHub Actions
- name: Run quality check
  if: github.event_name == 'pull_request'
  run: gz-quality check --since origin/${{ github.base_ref }}

- name: Run full quality check
  if: github.event_name == 'push' && github.ref == 'refs/heads/main'
  run: gz-quality check
```

### 캐싱

```yaml
# GitHub Actions - Go 모듈 캐싱
- uses: actions/cache@v4
  with:
    path: |
      ~/.cache/go-build
      ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
    restore-keys: |
      ${{ runner.os }}-go-

# GitLab CI - Go 모듈 캐싱
cache:
  paths:
    - .cache/go-build
    - go/pkg/mod
```

### 병렬 실행

```yaml
# GitHub Actions - 언어별 병렬 실행
jobs:
  quality-go:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: gz-quality run --files="**/*.go"

  quality-python:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: gz-quality run --files="**/*.py"

  quality-javascript:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: gz-quality run --files="**/*.js,**/*.ts"
```

### 실패 허용 (경고만)

```yaml
# GitHub Actions
- name: Run quality check (warning only)
  continue-on-error: true
  run: gz-quality check

# GitLab CI
quality-check:
  allow_failure: true
  script:
    - gz-quality check
```

---

## 모범 사례

### 1. 변경 파일만 검사

```bash
# Pull Request에서
gz-quality check --since origin/main

# Staged 파일만
gz-quality check --staged
```

### 2. 리포트 저장

```bash
# CI에서 JSON 리포트 생성
gz-quality check --report json --output quality-report.json

# 아티팩트로 저장
# GitHub Actions: uses: actions/upload-artifact
# GitLab CI: artifacts: paths:
```

### 3. 타임아웃 설정

```yaml
# GitHub Actions
- name: Run quality check
  timeout-minutes: 10
  run: gz-quality check

# GitLab CI
quality-check:
  timeout: 10m
  script:
    - gz-quality check
```

### 4. 도구 버전 고정

```bash
# 특정 버전 설치
go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v1.0.0

# 최신 버전
go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1
```

---

## 문제 해결

### gzq를 찾을 수 없음

```bash
# PATH에 Go bin 디렉토리 추가
export PATH=$PATH:$(go env GOPATH)/bin

# 또는 절대 경로 사용
$(go env GOPATH)/bin/gz-quality check
```

### Git history가 없음

```yaml
# Shallow clone 비활성화
- uses: actions/checkout@v4
  with:
    fetch-depth: 0  # 전체 히스토리
```

### 권한 오류

```yaml
# Docker에서 권한 문제
docker run --rm -v $(pwd):/workspace -u $(id -u):$(id -g) gz-quality:latest check
```

---

## 실제 프로젝트 시나리오

### 시나리오 1: 멀티 언어 모노레포

**요구사항**: Go, Python, TypeScript가 혼재된 모노레포에서 각 언어별 품질 검사

```yaml
# .github/workflows/quality.yml
name: Quality Check

on:
  pull_request:
    branches: [ main ]

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - uses: actions/setup-python@v5
        with:
          python-version: '3.12'

      - uses: actions/setup-node@v4
        with:
          node-version: '20'

      # Install language-specific tools
      - name: Install quality tools
        run: |
          # Go tools
          go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
          go install mvdan.cc/gofumpt@latest

          # Python tools
          pip install black ruff pylint

          # TypeScript tools
          npm install -g prettier eslint

      # Install gz-quality
      - name: Install gz-quality
        run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1

      # Run quality check on changed files only
      - name: Run quality check
        run: |
          gz-quality check \
            --since origin/${{ github.base_ref }} \
            --report markdown \
            --output quality-report.md

      - name: Comment PR with results
        if: always()
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            if (fs.existsSync('quality-report.md')) {
              const report = fs.readFileSync('quality-report.md', 'utf8');
              github.rest.issues.createComment({
                issue_number: context.issue.number,
                owner: context.repo.owner,
                repo: context.repo.repo,
                body: `## 🔍 Quality Check Results\n\n${report}`
              });
            }
```

### 시나리오 2: Staged 파일 Pre-commit 검사

**요구사항**: 커밋 전 staged 파일만 빠르게 검사하고 자동 수정

```yaml
# .github/workflows/pre-commit.yml
name: Pre-commit Check

on:
  push:
    branches-ignore:
      - main
      - master

jobs:
  pre-commit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install gz-quality
        run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1

      # Auto-fix and commit
      - name: Run quality check with auto-fix
        run: |
          gz-quality run --staged --fix || true

      - name: Commit fixes
        run: |
          git config --local user.email "action@github.com"
          git config --local user.name "GitHub Action"
          git add -A
          git diff --staged --quiet || git commit -m "style: auto-fix quality issues"

      - name: Push changes
        uses: ad-m/github-push-action@master
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          branch: ${{ github.ref }}
```

### 시나리오 3: 대규모 프로젝트 병렬 검사

**요구사항**: 10,000+ 파일 프로젝트에서 성능 최적화

```yaml
# .github/workflows/quality-parallel.yml
name: Quality Check (Parallel)

on:
  pull_request:

jobs:
  # Step 1: Changed files만 추출
  detect-changes:
    runs-on: ubuntu-latest
    outputs:
      files: ${{ steps.changes.outputs.files }}
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Detect changed files
        id: changes
        run: |
          FILES=$(git diff --name-only origin/${{ github.base_ref }}...HEAD | jq -R -s -c 'split("\n")[:-1]')
          echo "files=$FILES" >> $GITHUB_OUTPUT

  # Step 2: 언어별로 병렬 검사
  quality-go:
    needs: detect-changes
    if: contains(needs.detect-changes.outputs.files, '.go')
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1
      - run: gz-quality tool golangci-lint --since origin/${{ github.base_ref }}

  quality-python:
    needs: detect-changes
    if: contains(needs.detect-changes.outputs.files, '.py')
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: '3.12'
      - run: pip install ruff black
      - run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1
      - run: gz-quality tool ruff --since origin/${{ github.base_ref }}

  quality-typescript:
    needs: detect-changes
    if: contains(needs.detect-changes.outputs.files, '.ts') || contains(needs.detect-changes.outputs.files, '.tsx')
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - run: npm install -g eslint prettier
      - run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1
      - run: gz-quality tool eslint --since origin/${{ github.base_ref }}
```

### 시나리오 4: Fail-fast vs Fail-safe

**요구사항**: 개발 브랜치는 경고만, main 브랜치는 엄격하게

```yaml
# .github/workflows/quality-flexible.yml
name: Quality Check (Flexible)

on:
  push:
    branches: [ '**' ]

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install gz-quality
        run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1

      # Development branches: warning only
      - name: Run quality check (warning mode)
        if: github.ref != 'refs/heads/main'
        continue-on-error: true
        run: |
          gz-quality check --report markdown --output quality-report.md
          echo "⚠️ Quality check completed with warnings" >> $GITHUB_STEP_SUMMARY
          cat quality-report.md >> $GITHUB_STEP_SUMMARY

      # Main branch: strict mode
      - name: Run quality check (strict mode)
        if: github.ref == 'refs/heads/main'
        run: |
          gz-quality check --report markdown --output quality-report.md

      - name: Fail on issues
        if: github.ref == 'refs/heads/main'
        run: |
          if grep -q "❌" quality-report.md; then
            echo "::error::Quality check failed on main branch"
            exit 1
          fi
```

### 시나리오 5: Caching으로 성능 개선

**요구사항**: 도구 설치 시간 단축 (30초 → 5초)

```yaml
# .github/workflows/quality-cached.yml
name: Quality Check (Cached)

on:
  push:

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache: true

      # Cache Go tools
      - name: Cache Go tools
        uses: actions/cache@v3
        with:
          path: |
            ~/go/bin
          key: ${{ runner.os }}-go-tools-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-tools-

      # Install if not cached
      - name: Install gz-quality
        run: |
          if [ ! -f ~/go/bin/gz-quality ]; then
            go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1
          fi

          # Verify installation
          gz-quality version

      - name: Run quality check
        run: gz-quality check
```

---

## 성능 최적화 팁

### 1. Changed Files만 검사
```bash
# PR에서 변경된 파일만
gz-quality check --since origin/main

# 최근 3 커밋만
gz-quality check --since HEAD~3
```

### 2. Worker 수 조정
```bash
# CPU 코어 수에 맞춰 자동 조정 (기본값)
gz-quality run

# 수동 지정 (대규모 프로젝트)
gz-quality run --workers 8
```

### 3. 도구별 타임아웃 설정
```yaml
# .gzquality.yml
timeout: "10m"

tools:
  golangci-lint:
    timeout: "5m"
  eslint:
    timeout: "3m"
```

### 4. 캐싱 전략
```yaml
# Go 모듈 캐싱
- uses: actions/cache@v3
  with:
    path: ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}

# Node 모듈 캐싱
- uses: actions/cache@v3
  with:
    path: node_modules
    key: ${{ runner.os }}-node-${{ hashFiles('**/package-lock.json') }}
```

---

## 문제 해결

### GitHub Actions에서 도구를 찾을 수 없음

**증상**: `golangci-lint: command not found`

**해결**:
```yaml
- name: Add Go bin to PATH
  run: echo "$HOME/go/bin" >> $GITHUB_PATH

- name: Install tools
  run: |
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    which golangci-lint  # Verify installation
```

### Permission Denied 에러

**증상**: `permission denied: .gzquality.yml`

**해결**:
```yaml
- name: Fix permissions
  run: chmod -R 755 .

- name: Run quality check
  run: gz-quality check
```

### Out of Memory (OOM)

**증상**: 대규모 프로젝트에서 메모리 부족

**해결**:
```yaml
# 1. Worker 수 감소
- run: gz-quality run --workers 2

# 2. Changed files만 검사
- run: gz-quality check --since origin/main

# 3. 언어별 분리 실행
- run: gz-quality tool golangci-lint
- run: gz-quality tool ruff
```

---

## 테스트 커버리지 통합

### 커버리지 목표

프로젝트는 다음 커버리지 목표를 유지합니다:

| 패키지 | 최소 커버리지 | 권장 커버리지 | 현재 상태 |
|--------|--------------|--------------|----------|
| config | 80% | 90% | ✅ 85.1% |
| detector | 50% | 70% | ✅ 53.3% |
| git | 85% | 95% | ✅ 92.0% |
| executor | 75% | 85% | ✅ 80.0% |
| report | 90% | 95% | ✅ 95.3% |
| tools | 15% | 30% | ✅ 16.0% |
| **전체** | **40%** | **50%** | ✅ **45.9%** |

### GitHub Actions - 커버리지 체크

`.github/workflows/coverage.yml`:

```yaml
name: Test Coverage

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]

jobs:
  coverage:
    name: Test Coverage Check
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Run tests with coverage
        run: |
          go test ./... -coverprofile=coverage.out -covermode=atomic
          go tool cover -func=coverage.out -o coverage.txt

      - name: Check coverage threshold
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Current coverage: $COVERAGE%"

          if (( $(echo "$COVERAGE < 40.0" | bc -l) )); then
            echo "❌ Coverage $COVERAGE% is below minimum threshold of 40%"
            exit 1
          fi

          echo "✅ Coverage $COVERAGE% meets minimum threshold"

      - name: Generate coverage report
        run: go tool cover -html=coverage.out -o coverage.html

      - name: Upload coverage report
        uses: actions/upload-artifact@v4
        with:
          name: coverage-report
          path: |
            coverage.out
            coverage.html
            coverage.txt

      - name: Comment coverage on PR
        if: github.event_name == 'pull_request'
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const coverage = fs.readFileSync('coverage.txt', 'utf8');
            const lines = coverage.split('\n');

            let body = '## 📊 Test Coverage Report\n\n';
            body += '```\n' + lines.slice(-20).join('\n') + '\n```\n';

            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: body
            });
```

### GitLab CI - 커버리지 추적

`.gitlab-ci.yml`:

```yaml
test:coverage:
  stage: test
  image: golang:1.24
  script:
    - go test ./... -coverprofile=coverage.out -covermode=atomic
    - go tool cover -func=coverage.out

  coverage: '/total:.*?(\d+\.\d+)%/'

  artifacts:
    reports:
      coverage_report:
        coverage_format: cobertura
        path: coverage.xml
    paths:
      - coverage.out
      - coverage.html
```

### CircleCI - 커버리지 배지

`.circleci/config.yml`:

```yaml
jobs:
  test-coverage:
    docker:
      - image: cimg/go:1.24
    steps:
      - checkout
      - run:
          name: Run tests with coverage
          command: |
            go test ./... -coverprofile=coverage.out
            go tool cover -html=coverage.out -o coverage.html

      - run:
          name: Upload to Codecov
          command: |
            curl -Os https://uploader.codecov.io/latest/linux/codecov
            chmod +x codecov
            ./codecov -f coverage.out

      - store_artifacts:
          path: coverage.html
          destination: coverage-report
```

### 로컬 개발 - 커버리지 확인

**전체 커버리지 확인:**

```bash
# 전체 테스트 실행 및 커버리지 생성
go test ./... -coverprofile=coverage.out

# 커버리지 요약 확인
go tool cover -func=coverage.out

# HTML 리포트 생성 및 브라우저 열기
go tool cover -html=coverage.out -o coverage.html
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

**패키지별 커버리지 확인:**

```bash
# 특정 패키지만 테스트
go test ./config/... -coverprofile=config_coverage.out
go tool cover -func=config_coverage.out

# 커버리지가 낮은 부분 찾기
go tool cover -func=coverage.out | grep -v "100.0%" | sort -k3 -n
```

**커버리지 임계값 검증 스크립트:**

`scripts/check-coverage.sh`:

```bash
#!/bin/bash

MIN_COVERAGE=40.0

echo "Running tests with coverage..."
go test ./... -coverprofile=coverage.out -covermode=atomic

if [ $? -ne 0 ]; then
    echo "❌ Tests failed"
    exit 1
fi

COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

echo "Current coverage: $COVERAGE%"
echo "Minimum required: $MIN_COVERAGE%"

if (( $(echo "$COVERAGE < $MIN_COVERAGE" | bc -l) )); then
    echo "❌ Coverage is below minimum threshold"
    exit 1
fi

echo "✅ Coverage meets minimum threshold"

# 패키지별 커버리지 출력
echo ""
echo "Coverage by package:"
go tool cover -func=coverage.out | grep -E "^github.com" | \
    awk '{print $1 "\t" $3}' | \
    sed 's/github.com\/Gizzahub\/gzh-cli-quality\///' | \
    column -t
```

**사용:**

```bash
chmod +x scripts/check-coverage.sh
./scripts/check-coverage.sh
```

### Pre-commit Hook - 커버리지 체크

`.git/hooks/pre-commit` 또는 `.pre-commit-config.yaml`:

```yaml
- repo: local
  hooks:
    - id: test-coverage
      name: Check test coverage
      entry: scripts/check-coverage.sh
      language: script
      pass_filenames: false
      always_run: true
```

### 커버리지 배지

**README.md에 추가:**

```markdown
[![Coverage](https://img.shields.io/badge/coverage-45.9%25-brightgreen.svg)](https://github.com/Gizzahub/gzh-cli-quality)
```

**동적 배지 (Codecov):**

```markdown
[![codecov](https://codecov.io/gh/Gizzahub/gzh-cli-quality/branch/main/graph/badge.svg)](https://codecov.io/gh/Gizzahub/gzh-cli-quality)
```

**동적 배지 (Coveralls):**

```markdown
[![Coverage Status](https://coveralls.io/repos/github/Gizzahub/gzh-cli-quality/badge.svg?branch=main)](https://coveralls.io/github/Gizzahub/gzh-cli-quality?branch=main)
```

### 커버리지 개선 가이드

**1. 테스트되지 않은 코드 찾기:**

```bash
# 커버리지가 0%인 파일 찾기
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep "0.0%" | awk '{print $1}'
```

**2. 중요도 기반 우선순위:**

| 우선순위 | 패키지 유형 | 목표 커버리지 |
|---------|-----------|-------------|
| 높음 | 핵심 비즈니스 로직 (executor, git) | 80%+ |
| 중간 | 유틸리티/도구 (config, detector) | 60%+ |
| 낮음 | 외부 통합 (tools 구현체) | 30%+ |

**3. 테스트 작성 가이드:**

```go
// 좋은 테스트: 명확한 의도, 독립적, 빠름
func TestConfigLoad_ValidYAML(t *testing.T) {
    tmpDir := t.TempDir()
    configPath := filepath.Join(tmpDir, "config.yml")

    yamlContent := `default_workers: 8
timeout: "5m"`

    err := os.WriteFile(configPath, []byte(yamlContent), 0o644)
    require.NoError(t, err)

    config, err := LoadConfig(configPath)
    require.NoError(t, err)
    assert.Equal(t, 8, config.DefaultWorkers)
    assert.Equal(t, "5m", config.Timeout)
}
```

**4. 테스트 커버리지 vs 품질:**

- ✅ **의미있는 테스트**: 엣지 케이스, 에러 핸들링
- ❌ **숫자 채우기**: getter/setter만 테스트

---

**관련 문서**:
- [사용 예제](./EXAMPLES.md)
- [도구 추가하기](./ADDING_TOOLS.md)
- [API 레퍼런스](./API.md)
- [Pre-commit Hooks 가이드](./PRE_COMMIT_HOOKS.md)
