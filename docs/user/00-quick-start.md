# 5분 빠른 시작 가이드

gzh-cli-quality를 처음 사용하시나요? 이 가이드를 따라 5분 안에 첫 품질 검사를 시작할 수 있습니다.

## 사전 요구사항

- Go 1.24.0 이상 설치
- Git 2.0 이상 설치
- 체크할 프로젝트 (Go/Python/JavaScript/TypeScript/Rust)

---

## 1단계: 설치 (1분)

### 방법 1: Go Install (권장)

```bash
# 최신 안정 버전 설치
go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest

# 설치 확인
gz-quality version
```

### 방법 2: 소스에서 빌드

```bash
# 리포지토리 클론
git clone https://github.com/Gizzahub/gzh-cli-quality.git
cd gzh-cli-quality

# 빌드
make build

# 바이너리는 build/gz-quality에 생성됨
./build/gz-quality version
```

### 설치 확인

```bash
# 명령어가 실행되면 설치 성공
gz-quality version

# 출력 예시:
# gzh-cli-quality v0.1.1
```

**문제 발생 시**: [문제 해결 가이드](./05-troubleshooting.md#설치-문제) 참조

---

## 2단계: 첫 실행 (2분)

### 프로젝트 분석

```bash
# 프로젝트 디렉토리로 이동
cd /path/to/your/project

# 프로젝트 분석 (어떤 도구가 필요한지 확인)
gz-quality analyze
```

**출력 예시**:
```
📊 프로젝트 분석 결과

감지된 언어:
  ✓ Go (15 files)
  ✓ Python (8 files)

권장 도구:
  Go:
    ✓ gofumpt (설치됨)
    ✓ goimports (설치됨)
    ✗ golangci-lint (미설치)
  Python:
    ✓ black (설치됨)
    ✗ ruff (미설치)
```

### 필요한 도구 설치

```bash
# 프로젝트에 필요한 도구만 설치
gz-quality install

# 또는 특정 도구만 설치
gz-quality install golangci-lint
gz-quality install ruff
```

### 첫 품질 검사

```bash
# 전체 프로젝트 검사
gz-quality run

# 상세 출력 보기
gz-quality run --verbose
```

**출력 예시**:
```
🔍 프로젝트 분석 중...
📋 실행 계획: 4개 도구, 23개 파일
⚡ 실행 중 (4 workers)...
  ✅ gofumpt (0.5s) - 15 files
  ✅ goimports (0.3s) - 15 files
  ✅ black (0.4s) - 8 files
  ⚠️ ruff (1.2s) - 3 issues
    utils.py:42:15 warning: unused import 'os' (F401)
    main.py:18:1 error: missing docstring (D100)
    config.py:5:80 warning: line too long (E501)
✨ 완료: 2.4s, 23 files, 3 issues
```

---

## 3단계: 커밋 전 검사 (2분)

### 변경된 파일만 검사

```bash
# 파일 수정
echo "# test" >> README.md

# staged 파일 추가
git add README.md

# staged 파일만 검사
gz-quality run --staged
```

### 자동 수정 적용

```bash
# 포매팅 문제를 자동으로 수정
gz-quality run --staged --fix

# 수정된 파일 다시 stage
git add .

# 린팅만 검사 (수정 없이)
gz-quality check --staged
```

### 커밋

```bash
# 이슈가 없으면 커밋
git commit -m "docs: update README"
```

---

## 일반적인 사용 패턴

### 패턴 1: 빠른 개발 루프

```bash
# 1. 코드 작성
vim main.go

# 2. 변경 파일만 포매팅
gz-quality run --changed --format-only --fix

# 3. 전체 검사
gz-quality check --changed

# 4. 커밋
git add .
git commit -m "feat: add new feature"
```

### 패턴 2: PR 전 전체 검사

```bash
# main 브랜치 이후 변경된 모든 파일 검사
gz-quality check --since main

# 리포트 생성 (CI/CD 용)
gz-quality check --since main --report json --output quality-report.json
```

### 패턴 3: 특정 도구만 실행

```bash
# Go 코드만 포매팅
gz-quality tool gofumpt --fix

# Python 린팅만
gz-quality tool ruff

# golangci-lint에 추가 옵션 전달
gz-quality tool golangci-lint -- --enable-all
```

---

## 설정 파일 생성 (선택사항)

### 프로젝트 맞춤 설정

```bash
# 설정 파일 생성
gz-quality init

# .gzquality.yml 파일이 생성됨
```

### 기본 설정 예시

```yaml
# .gzquality.yml
default_workers: 4
timeout: "10m"

tools:
  golangci-lint:
    enabled: true
    config_file: ".golangci.yml"
  ruff:
    enabled: true
    args: ["--fix"]

exclude:
  - "vendor/**"
  - "node_modules/**"
  - ".git/**"
```

**설정 상세**: [설정 가이드](./03-configuration.md) 참조

---

## Pre-commit Hook 설정 (선택사항)

커밋할 때마다 자동으로 품질 검사:

```bash
# hooks 디렉토리로 이동
cd .git/hooks

# pre-commit hook 생성
cat > pre-commit << 'EOF'
#!/bin/bash
gz-quality run --staged --fix
EOF

# 실행 권한 부여
chmod +x pre-commit

# 테스트
git add .
git commit -m "test"  # 자동으로 품질 검사 실행
```

**상세 가이드**: [Pre-commit Hooks](../integration/PRE_COMMIT_HOOKS.md) 참조

---

## 다음 단계

축하합니다! 이제 gzh-cli-quality의 기본 사용법을 익혔습니다.

### 더 알아보기

- 📖 [기본 사용법](./02-basic-usage.md) - 모든 명령어 상세 설명
- ⚙️ [설정 가이드](./03-configuration.md) - 프로젝트 맞춤 설정
- 🔄 [워크플로우](./04-workflows.md) - 실전 워크플로우 패턴
- 🔧 [문제 해결](./05-troubleshooting.md) - 흔한 문제와 해결 방법
- 🤖 [CI/CD 통합](../integration/CI_INTEGRATION.md) - GitHub Actions, GitLab CI 등

### 도움이 필요하신가요?

- 💬 [GitHub Issues](https://github.com/Gizzahub/gzh-cli-quality/issues)
- 📚 [전체 문서](../../README.md#문서)
- ❓ [FAQ](./06-faq.md)

---

**팁**: `gz-quality --help` 명령어로 언제든지 도움말을 볼 수 있습니다.
