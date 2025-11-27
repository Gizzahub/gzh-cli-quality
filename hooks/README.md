# Git Hooks for gzh-cli-quality

이 디렉토리에는 `gzh-cli-quality`를 Git 워크플로우에 통합하기 위한 hooks와 설정 파일이 포함되어 있습니다.

## 설치 방법

### 방법 1: 자동 설치 (권장)

```bash
bash hooks/install.sh
```

이 스크립트는:
- 기존 pre-commit hook을 백업
- gz-quality pre-commit hook을 설치
- 실행 권한을 자동으로 설정

### 방법 2: 수동 설치

```bash
cp hooks/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

### 방법 3: pre-commit 프레임워크 사용

```bash
# pre-commit 설치
pip install pre-commit

# 설정 파일 복사
cp .pre-commit-config.example.yaml .pre-commit-config.yaml

# 프레임워크 설치
pre-commit install

# 테스트 실행
pre-commit run --all-files
```

## 사용 방법

### 기본 사용

hook이 설치되면 `git commit` 시 자동으로 실행됩니다:

```bash
git add .
git commit -m "feat: add new feature"
# → gz-quality check --staged 자동 실행
```

### Hook 건너뛰기

필요한 경우 hook을 건너뛸 수 있습니다:

```bash
git commit --no-verify -m "WIP: work in progress"
```

### 환경 변수로 동작 변경

```bash
# 포매팅만 실행
export GZ_QUALITY_MODE=format
git commit -m "style: format code"

# 포매팅 + 린팅 실행
export GZ_QUALITY_MODE=run
git commit -m "refactor: improve code quality"

# 체크만 실행 (기본값)
export GZ_QUALITY_MODE=check
git commit -m "fix: resolve bug"
```

## 파일 설명

### `pre-commit`

Git pre-commit hook 스크립트입니다.

**기능**:
- Staged 파일에서 품질 검사 실행
- 관련 파일 타입만 검사 (.go, .py, .js, .ts, .rs)
- 실패 시 커밋 중단

**설정 옵션**:
```bash
export GZ_QUALITY_CMD="gz-quality"           # gz-quality 명령어 경로
export GZ_QUALITY_MODE="check"        # 실행 모드 (check/format/run)
export GZ_QUALITY_FLAGS="--staged"    # 추가 플래그
```

### `install.sh`

Hook 설치 자동화 스크립트입니다.

**기능**:
- Git 저장소 확인
- gz-quality 설치 여부 확인
- 기존 hook 백업
- 새 hook 설치 및 권한 설정

**사용**:
```bash
bash hooks/install.sh
```

### `.pre-commit-hooks.yaml` (프로젝트 루트)

pre-commit 프레임워크용 hook 정의 파일입니다.

**제공 hooks**:
- `gz-quality-check`: 전체 품질 검사
- `gz-quality-format`: 포매팅만
- `gz-quality-check-go`: Go 파일만
- `gz-quality-check-python`: Python 파일만
- `gz-quality-check-javascript`: JS/TS 파일만
- `gz-quality-check-rust`: Rust 파일만

### `.pre-commit-config.example.yaml` (프로젝트 루트)

사용자용 pre-commit 설정 예제 파일입니다.

**사용**:
```bash
cp .pre-commit-config.example.yaml .pre-commit-config.yaml
# 필요에 따라 편집
pre-commit install
```

## 예제 워크플로우

### 개발 워크플로우

```bash
# 1. 코드 작성
vim main.go

# 2. 변경 사항 확인
gz-quality check --changed

# 3. 자동 수정 적용
gz-quality run --changed --fix

# 4. Stage 및 커밋 (hook 자동 실행)
git add main.go
git commit -m "feat: implement new feature"
# → pre-commit hook이 gz-quality check --staged 실행
```

### CI와 함께 사용

```bash
# 로컬: pre-commit hook으로 빠른 피드백
git commit

# CI: 전체 검사
gz-quality check --since main --report json
```

## 커스터마이징

### 특정 파일 제외

`.gz-qualityuality.yml`에서 제외 패턴 설정:

```yaml
exclude:
  - "vendor/**"
  - "node_modules/**"
  - "**/*_gen.go"
  - "**/*.pb.go"
```

### 언어별 Hook

`pre-commit` hook을 언어별로 분리:

```bash
# Go 전용 hook
if git diff --cached --name-only | grep -q '\.go$'; then
    gz-quality tool golangci-lint --staged
fi

# Python 전용 hook
if git diff --cached --name-only | grep -q '\.py$'; then
    gz-quality tool ruff --staged --fix
fi
```

### 여러 Hook 조합

`.git/hooks/pre-commit`:

```bash
#!/bin/bash
set -e

# 1. gz-quality 품질 검사
gz-quality check --staged

# 2. 추가 검사 (예: 테스트)
if git diff --cached --name-only | grep -q '_test\.go$'; then
    go test ./...
fi

# 3. 커밋 메시지 검증 (pre-commit이 아닌 commit-msg hook)
# commitlint 등 사용
```

## 문제 해결

### gz-quality를 찾을 수 없음

```bash
# PATH 확인
echo $PATH

# gz-quality 위치 확인
which gz-quality

# PATH에 추가
export PATH=$PATH:$(go env GOPATH)/bin

# 또는 절대 경로 사용
export GZ_QUALITY_CMD="$HOME/go/bin/gz-quality"
```

### Hook이 실행되지 않음

```bash
# 실행 권한 확인
ls -la .git/hooks/pre-commit

# 권한 설정
chmod +x .git/hooks/pre-commit

# Hook 테스트
.git/hooks/pre-commit
```

### 너무 느림

```bash
# staged 파일만 검사 (기본값)
export GZ_QUALITY_FLAGS="--staged"

# 또는 포매팅만
export GZ_QUALITY_MODE=format

# 또는 특정 도구만
export GZ_QUALITY_CMD="gz-quality tool gofumpt"
```

## 제거

### Git hook 제거

```bash
rm .git/hooks/pre-commit
```

### pre-commit 프레임워크 제거

```bash
pre-commit uninstall
rm .pre-commit-config.yaml
```

## 관련 문서

- [CI 통합 가이드](../docs/CI_INTEGRATION.md)
- [사용 예제](../docs/EXAMPLES.md)
- [기여 가이드](../CONTRIBUTING.md)

---

**참고**: Hook은 로컬 개발 환경에서만 실행됩니다. CI/CD 파이프라인에서는 별도로 gz-quality를 실행해야 합니다.
