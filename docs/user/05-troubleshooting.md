# 문제 해결 가이드

gzh-cli-quality 사용 중 발생하는 흔한 문제와 해결 방법을 안내합니다.

## 목차

- [설치 문제](#%EC%84%A4%EC%B9%98-%EB%AC%B8%EC%A0%9C)
- [실행 문제](#%EC%8B%A4%ED%96%89-%EB%AC%B8%EC%A0%9C)
- [성능 문제](#%EC%84%B1%EB%8A%A5-%EB%AC%B8%EC%A0%9C)
- [도구 관련 문제](#%EB%8F%84%EA%B5%AC-%EA%B4%80%EB%A0%A8-%EB%AC%B8%EC%A0%9C)
- [Git 통합 문제](#git-%ED%86%B5%ED%95%A9-%EB%AC%B8%EC%A0%9C)
- [설정 문제](#%EC%84%A4%EC%A0%95-%EB%AC%B8%EC%A0%9C)
- [CI/CD 문제](#cicd-%EB%AC%B8%EC%A0%9C)

______________________________________________________________________

## 설치 문제

### Q1: "gz-quality: command not found" 에러

**증상**:

````bash
$ gz-quality version
bash: gz-quality: command not found
```bash

**원인**: `$PATH`에 Go bin 디렉토리가 없음

**해결 방법**:

```bash
# 1. Go bin 경로 확인
go env GOPATH
# 출력: /Users/username/go (또는 /home/username/go)

# 2. PATH에 추가 (bash 사용자)
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc

# 2. PATH에 추가 (zsh 사용자)
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc

# 3. 확인
gz-quality version
```bash

**대안**: 절대 경로로 실행
```bash
$(go env GOPATH)/bin/gz-quality version
```bash

---

### Q2: "go: command not found" 에러

**증상**:
```bash
$ go version
bash: go: command not found
```bash

**원인**: Go가 설치되지 않음

**해결 방법**:

**macOS (Homebrew)**:
```bash
brew install go
go version
```bash

**Linux (Ubuntu/Debian)**:
```bash
sudo apt update
sudo apt install golang-go
go version
```bash

**공식 바이너리 설치**:
1. https://go.dev/dl/ 에서 다운로드
2. 설치 후 PATH 설정:
```bash
export PATH=$PATH:/usr/local/go/bin
```bash

---

### Q3: 특정 버전 설치 실패

**증상**:
```bash
$ go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.0
go: github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.0: invalid version
```bash

**원인**: 존재하지 않는 버전 태그

**해결 방법**:

```bash
# 1. 사용 가능한 버전 확인
git ls-remote --tags https://github.com/Gizzahub/gzh-cli-quality.git

# 2. 최신 버전 설치
go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest

# 3. 특정 커밋 설치
go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@commit-hash
```bash

---

## 실행 문제

### Q4: "No quality tools found" 에러

**증상**:
```bash
$ gz-quality run
Error: No quality tools found for this project
```bash

**원인**: 도구가 설치되지 않았거나 감지되지 않음

**해결 방법**:

```bash
# 1. 프로젝트 분석
gz-quality analyze

# 2. 필요한 도구 설치
gz-quality install

# 3. 특정 도구 수동 설치
# Go 도구
go install mvdan.cc/gofumpt@latest
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Python 도구
pip install black ruff pylint

# JavaScript 도구
npm install -g prettier eslint typescript
```bash

---

### Q5: "Permission denied" 에러

**증상**:
```bash
$ gz-quality run
Error: fork/exec /usr/local/bin/gofumpt: permission denied
```bash

**원인**: 도구 실행 권한 없음

**해결 방법**:

```bash
# 1. 도구 위치 확인
which gofumpt

# 2. 실행 권한 부여
chmod +x /usr/local/bin/gofumpt

# 3. 소유자 확인
ls -la /usr/local/bin/gofumpt

# 4. 필요시 재설치
go install mvdan.cc/gofumpt@latest
```bash

---

### Q6: 특정 파일이 처리되지 않음

**증상**: 일부 파일이 품질 검사에서 누락됨

**원인**:
- 파일이 `.gzquality.yml`의 `exclude` 패턴에 포함됨
- 지원하지 않는 파일 확장자

**해결 방법**:

```bash
# 1. 제외 패턴 확인
cat .gzquality.yml | grep -A 10 "exclude:"

# 2. 실행 계획 확인
gz-quality run --dry-run --verbose

# 3. 설정 파일 수정
# .gzquality.yml
exclude:
  - "vendor/**"      # 유지
  - "node_modules/**" # 유지
  # - "tests/**"     # 주석 처리 또는 제거

# 4. 특정 파일 강제 포함
gz-quality run --files="tests/test_main.py"
```yaml

---

## 성능 문제

### Q7: 실행이 너무 느림 (5분 이상)

**증상**: 전체 검사가 5분 이상 소요됨

**원인**:
- 대규모 프로젝트
- 느린 도구 (golangci-lint, pylint)
- 워커 수 부족

**해결 방법**:

```bash
# 1. 변경된 파일만 검사
gz-quality check --changed

# 2. 워커 수 증가 (CPU 코어 수에 맞춤)
gz-quality run --workers 8

# 3. 느린 도구 비활성화 (.gzquality.yml)
tools:
  pylint:
    enabled: false  # 로컬에서 비활성화
  golangci-lint:
    enabled: true
    timeout: "5m"   # 타임아웃 설정

# 4. 포매팅만 빠르게 실행
gz-quality run --format-only --fix

# 5. 캐시 활용 (도구별 설정)
# golangci-lint는 자동으로 캐시 사용
```bash

**추가 팁**:
```bash
# 시간 측정
time gz-quality run

# 병목 지점 찾기
gz-quality run --verbose 2>&1 | grep "duration"
```bash

---

### Q8: 메모리 부족 에러

**증상**:
```bash
$ gz-quality run
fatal error: out of memory
```bash

**원인**: 대용량 파일 또는 너무 많은 병렬 워커

**해결 방법**:

```bash
# 1. 워커 수 감소
gz-quality run --workers 2

# 2. 파일 분할 처리
gz-quality run --files="**/*.go"
gz-quality run --files="**/*.py"

# 3. 대용량 파일 제외
# .gzquality.yml
exclude:
  - "**/*.min.js"
  - "**/*.bundle.js"
  - "**/*_gen.go"

# 4. 시스템 메모리 확인
free -h  # Linux
vm_stat  # macOS
```bash

---

## 도구 관련 문제

### Q9: golangci-lint가 너무 느림

**증상**: golangci-lint 실행이 1분 이상 소요

**해결 방법**:

```bash
# 1. 빠른 린터만 활성화 (.golangci.yml)
linters:
  disable-all: true
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused

# 2. 변경된 파일만 검사
gz-quality tool golangci-lint --changed

# 3. 캐시 활용 확인
golangci-lint cache status

# 4. 캐시 정리 후 재실행
golangci-lint cache clean
```yaml

---

### Q10: ruff와 black이 충돌함

**증상**: ruff로 포매팅한 후 black이 다시 수정함

**원인**: 포매팅 규칙 충돌

**해결 방법**:

```bash
# 방법 1: ruff만 사용 (권장)
# .gzquality.yml
tools:
  ruff:
    enabled: true
    args: ["--fix"]
  black:
    enabled: false  # 비활성화

# 방법 2: black 설정을 ruff와 맞춤
# pyproject.toml
[tool.black]
line-length = 88

[tool.ruff]
line-length = 88

# 방법 3: ruff의 formatter 사용
# pyproject.toml
[tool.ruff]
format = true
```bash

---

### Q11: prettier와 eslint가 충돌함

**증상**: prettier로 포매팅 후 eslint가 오류 표시

**해결 방법**:

```bash
# 1. eslint-config-prettier 설치
npm install --save-dev eslint-config-prettier

# 2. .eslintrc.json 수정
{
  "extends": [
    "eslint:recommended",
    "prettier"  // 마지막에 추가
  ]
}

# 3. 확인
gz-quality run --format-only --fix
gz-quality check
```bash

---

### Q12: TypeScript 타입 체크 실패

**증상**:
```bash
$ gz-quality tool tsc
Error: Cannot find module 'typescript'
```bash

**해결 방법**:

```bash
# 1. TypeScript 설치 확인
npm list typescript

# 2. 설치
npm install --save-dev typescript

# 3. tsconfig.json 생성
npx tsc --init

# 4. 실행
gz-quality tool tsc
```bash

---

## Git 통합 문제

### Q13: "--staged" 옵션이 파일을 찾지 못함

**증상**:
```bash
$ gz-quality run --staged
No files to process
```bash

**원인**: Staged 파일이 없음

**해결 방법**:

```bash
# 1. Staged 파일 확인
git status

# 2. 파일 stage
git add .

# 3. 다시 실행
gz-quality run --staged

# 4. 변경된 파일로 테스트
gz-quality run --changed
```bash

---

### Q14: "--since" 옵션 에러

**증상**:
```bash
$ gz-quality run --since main
Error: invalid commit reference: main
```bash

**원인**: 브랜치 이름이 다름 (master vs main)

**해결 방법**:

```bash
# 1. 브랜치 목록 확인
git branch -a

# 2. 올바른 브랜치명 사용
gz-quality run --since master
# 또는
gz-quality run --since origin/main

# 3. 커밋 해시 사용
gz-quality run --since abc1234

# 4. 상대 참조 사용
gz-quality run --since HEAD~5
```bash

---

### Q15: Git이 설치되지 않았다는 에러

**증상**:
```bash
$ gz-quality run --staged
Error: git command not found
```bash

**해결 방법**:

```bash
# 1. Git 설치
# macOS
brew install git

# Ubuntu/Debian
sudo apt install git

# 2. 확인
git --version

# 3. PATH 설정 (필요시)
export PATH="/usr/bin:$PATH"
```yaml

---

## 설정 문제

### Q16: 설정 파일이 인식되지 않음

**증상**: `.gzquality.yml` 수정이 반영되지 않음

**해결 방법**:

```bash
# 1. 파일 위치 확인 (프로젝트 루트에 있어야 함)
ls -la .gzquality.yml

# 2. YAML 문법 확인
yamllint .gzquality.yml

# 또는 온라인 검증
cat .gzquality.yml | python -m yaml

# 3. 들여쓰기 확인 (스페이스 2칸)
# ❌ 잘못된 예
tools:
    gofumpt:  # 4칸 들여쓰기 (잘못됨)

# ✅ 올바른 예
tools:
  gofumpt:    # 2칸 들여쓰기
    enabled: true

# 4. 실행 계획 확인
gz-quality run --dry-run --verbose
```yaml

---

### Q17: 특정 도구 설정이 적용되지 않음

**증상**: `args` 또는 `config_file` 설정이 무시됨

**해결 방법**:

```bash
# 1. 설정 확인
# .gzquality.yml
tools:
  golangci-lint:
    enabled: true
    config_file: ".golangci.yml"  # 경로 확인
    args: ["--fast"]              # 인자 확인

# 2. 설정 파일 경로 확인 (절대 경로 사용)
tools:
  golangci-lint:
    config_file: "/absolute/path/to/.golangci.yml"

# 3. 직접 도구 실행으로 테스트
golangci-lint run --config .golangci.yml --fast

# 4. Verbose 모드로 확인
gz-quality run --verbose
```bash

---

## CI/CD 문제

### Q18: GitHub Actions에서 도구를 찾지 못함

**증상**:
```yaml
# GitHub Actions 로그
Error: gofumpt: command not found
```bash

**해결 방법**:

```yaml
# .github/workflows/quality.yml
name: Quality Check

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

      # 중요: Go bin을 PATH에 추가
      - name: Add Go bin to PATH
        run: echo "$(go env GOPATH)/bin" >> $GITHUB_PATH

      - name: Install gz-quality
        run: go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest

      # 도구 설치
      - name: Install tools
        run: gz-quality install

      - name: Run quality check
        run: gz-quality check --since origin/${{ github.base_ref }}
```bash

---

### Q19: CI에서 타임아웃 발생

**증상**: CI 작업이 10분 후 타임아웃

**해결 방법**:

```yaml
# 1. 타임아웃 증가
jobs:
  quality:
    runs-on: ubuntu-latest
    timeout-minutes: 30  # 기본 10분에서 증가

    steps:
      - name: Run quality check
        run: |
          gz-quality check \
            --since origin/main \
            --timeout 20m \
            --workers 4
```bash

```yaml
# 2. 변경된 파일만 검사
- name: Run quality check
  run: |
    # PR의 변경 파일만
    gz-quality check --since origin/${{ github.base_ref }}
```yaml

```yaml
# 3. 느린 도구 비활성화
- name: Create CI config
  run: |
    cat > .gzquality.yml << EOF
    tools:
      pylint:
        enabled: false
      golangci-lint:
        enabled: true
        timeout: "5m"
    EOF

- name: Run quality check
  run: gz-quality check
```bash

---

### Q20: Docker에서 권한 문제

**증상**:
```bash
$ docker run myimage gz-quality run
Error: permission denied
```bash

**해결 방법**:

```dockerfile
# Dockerfile
FROM golang:1.24-alpine

# 비root 사용자 생성
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# gz-quality 설치
RUN go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest

# 사용자 전환
USER appuser

# 작업 디렉토리
WORKDIR /app

# 실행
CMD ["gz-quality", "run"]
```bash

```bash
# Docker Compose
services:
  quality:
    image: myimage
    user: "1000:1000"  # UID:GID
    volumes:
      - ./:/app:rw      # 읽기/쓰기 권한
```bash

---

## 일반 문제 해결 단계

문제가 계속되면 다음 순서로 진행하세요:

### 1단계: 버전 확인
```bash
gz-quality version
go version
git --version
```bash

### 2단계: Verbose 모드 실행
```bash
gz-quality run --verbose --dry-run
```bash

### 3단계: 설정 초기화
```bash
# 기존 설정 백업
mv .gzquality.yml .gzquality.yml.backup

# 새로 생성
gz-quality init

# 테스트
gz-quality run
```bash

### 4단계: 캐시 정리
```bash
# Go 캐시
go clean -cache -modcache

# golangci-lint 캐시
golangci-lint cache clean

# 재설치
go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest
```bash

### 5단계: 이슈 리포트

문제가 해결되지 않으면 GitHub 이슈 제출:

```bash
# 디버그 정보 수집
gz-quality version > debug.txt
go version >> debug.txt
git --version >> debug.txt
gz-quality run --verbose --dry-run >> debug.txt 2>&1
```bash

https://github.com/Gizzahub/gzh-cli-quality/issues/new 에 `debug.txt` 내용 첨부

---

## 추가 도움

- 📚 [전체 문서](../../README.md)
- 💬 [GitHub Issues](https://github.com/Gizzahub/gzh-cli-quality/issues)
- 📖 [FAQ](./06-faq.md)
- 🔗 [CI/CD 통합 가이드](../integration/CI_INTEGRATION.md)

---

**팁**: 대부분의 문제는 PATH 설정, 도구 설치, 설정 파일 문법 오류 중 하나입니다. 위 내용으로 90% 이상 해결 가능합니다.
````
