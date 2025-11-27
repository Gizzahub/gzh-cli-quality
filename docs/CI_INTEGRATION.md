# CI/CD 통합 가이드

`gzh-cli-quality`를 다양한 CI/CD 플랫폼에 통합하는 방법을 안내합니다.

## 목차

- [GitHub Actions](#github-actions)
- [GitLab CI](#gitlab-ci)
- [CircleCI](#circleci)
- [Jenkins](#jenkins)
- [Pre-commit Hooks](#pre-commit-hooks)
- [Docker 통합](#docker-통합)

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

      - name: Install gzq
        run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gzq@latest

      - name: Run quality check
        run: gzq check --report json --output quality-report.json

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
  run: gzq check --since origin/${{ github.base_ref }}
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
  - run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gzq@latest
  - run: gzq check
```

### 코멘트로 결과 표시

```yaml
- name: Run quality check
  id: quality
  continue-on-error: true
  run: |
    gzq check --report markdown --output quality-report.md
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
    - go install github.com/Gizzahub/gzh-cli-quality/cmd/gzq@${GZQ_VERSION}
    - export PATH=$PATH:$(go env GOPATH)/bin
  script:
    - gzq check --report json --output quality-report.json
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
    - gzq check --since origin/$CI_MERGE_REQUEST_TARGET_BRANCH_NAME
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
          name: Install gzq
          command: go install github.com/Gizzahub/gzh-cli-quality/cmd/gzq@latest
      - run:
          name: Run quality check
          command: |
            gzq check --report json --output /tmp/quality-report.json
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
                sh 'go install github.com/Gizzahub/gzh-cli-quality/cmd/gzq@latest'
            }
        }

        stage('Quality Check') {
            steps {
                sh 'gzq check --report json --output quality-report.json'
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
# gzq pre-commit hook

set -e

echo "🔍 Running quality checks on staged files..."

# Check if gzq is installed
if ! command -v gzq &> /dev/null; then
    echo "❌ gzq not found. Install it with: go install github.com/Gizzahub/gzh-cli-quality/cmd/gzq@latest"
    exit 1
fi

# Run quality check on staged files
if ! gzq check --staged; then
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
      - id: gzq-check
        name: gzq quality check
        entry: gzq check
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
      "pre-commit": "gzq check --staged"
    }
  }
}
```

**설치**:

```bash
npm install --save-dev husky
npx husky install
npx husky add .husky/pre-commit "gzq check --staged"
```

---

## Docker 통합

### Dockerfile

```dockerfile
FROM golang:1.24-alpine AS builder

# Install gzq
RUN go install github.com/Gizzahub/gzh-cli-quality/cmd/gzq@latest

FROM alpine:latest

# Copy gzq from builder
COPY --from=builder /go/bin/gzq /usr/local/bin/gzq

# Install required tools (optional)
RUN apk add --no-cache \
    git \
    make

WORKDIR /workspace

ENTRYPOINT ["gzq"]
CMD ["check"]
```

**빌드 및 사용**:

```bash
# 빌드
docker build -t gzq:latest .

# 사용
docker run --rm -v $(pwd):/workspace gzq:latest check
docker run --rm -v $(pwd):/workspace gzq:latest run --dry-run
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
        go install github.com/Gizzahub/gzh-cli-quality/cmd/gzq@latest &&
        /go/bin/gzq check --report json --output quality-report.json
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
  run: gzq check --since origin/${{ github.base_ref }}

- name: Run full quality check
  if: github.event_name == 'push' && github.ref == 'refs/heads/main'
  run: gzq check
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
      - run: gzq run --files="**/*.go"

  quality-python:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: gzq run --files="**/*.py"

  quality-javascript:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: gzq run --files="**/*.js,**/*.ts"
```

### 실패 허용 (경고만)

```yaml
# GitHub Actions
- name: Run quality check (warning only)
  continue-on-error: true
  run: gzq check

# GitLab CI
quality-check:
  allow_failure: true
  script:
    - gzq check
```

---

## 모범 사례

### 1. 변경 파일만 검사

```bash
# Pull Request에서
gzq check --since origin/main

# Staged 파일만
gzq check --staged
```

### 2. 리포트 저장

```bash
# CI에서 JSON 리포트 생성
gzq check --report json --output quality-report.json

# 아티팩트로 저장
# GitHub Actions: uses: actions/upload-artifact
# GitLab CI: artifacts: paths:
```

### 3. 타임아웃 설정

```yaml
# GitHub Actions
- name: Run quality check
  timeout-minutes: 10
  run: gzq check

# GitLab CI
quality-check:
  timeout: 10m
  script:
    - gzq check
```

### 4. 도구 버전 고정

```bash
# 특정 버전 설치
go install github.com/Gizzahub/gzh-cli-quality/cmd/gzq@v1.0.0

# 최신 버전
go install github.com/Gizzahub/gzh-cli-quality/cmd/gzq@latest
```

---

## 문제 해결

### gzq를 찾을 수 없음

```bash
# PATH에 Go bin 디렉토리 추가
export PATH=$PATH:$(go env GOPATH)/bin

# 또는 절대 경로 사용
$(go env GOPATH)/bin/gzq check
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
docker run --rm -v $(pwd):/workspace -u $(id -u):$(id -g) gzq:latest check
```

---

**관련 문서**:
- [사용 예제](./EXAMPLES.md)
- [도구 추가하기](./ADDING_TOOLS.md)
- [API 레퍼런스](./API.md)
