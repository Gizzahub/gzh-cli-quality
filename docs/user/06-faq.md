# 자주 묻는 질문 (FAQ)

gzh-cli-quality 사용 시 자주 묻는 질문과 간단한 답변입니다.

> 💡 **팁**: 더 상세한 내용은 [문제 해결 가이드](./05-troubleshooting.md)를 참조하세요.

______________________________________________________________________

## 🚀 시작하기

### Q1: gzh-cli-quality는 무엇인가요?

**A**: 멀티 언어 코드 품질 도구 오케스트레이터입니다. Go, Python, JavaScript/TypeScript, Rust 프로젝트의 포매팅과 린팅을 하나의 명령어로 통합 실행합니다.

**주요 장점**:

- 11+ 도구를 단일 CLI로 통합
- 병렬 처리로 빠른 실행
- Git 통합으로 변경 파일만 선택적 처리
- 무설정으로 즉시 사용 가능

______________________________________________________________________

### Q2: 설치가 어렵나요?

**A**: 아니요, 매우 간단합니다!

````bash
# Go가 설치되어 있다면
go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest

# 확인
gz-quality version
```bash

**상세**: [설치 가이드](./01-installation.md) 참조

---

### Q3: 어떤 도구들을 지원하나요?

**A**: 4개 언어, 11개 도구를 지원합니다:

| 언어 | 포매터 | 린터 |
|------|--------|------|
| Go | gofumpt, goimports | golangci-lint |
| Python | black | ruff, pylint |
| JavaScript/TypeScript | prettier | eslint, tsc |
| Rust | rustfmt, cargo-fmt | clippy |

---

### Q4: 프로젝트에 맞게 설정할 수 있나요?

**A**: 네, `.gzquality.yml` 파일로 모든 것을 커스터마이징할 수 있습니다.

```bash
# 설정 파일 생성
gz-quality init
```bash

**상세**: [설정 가이드](./03-configuration.md) 참조

---

## 💻 사용법

### Q5: 가장 기본적인 사용법은?

**A**: 프로젝트 루트에서 실행하세요:

```bash
# 전체 품질 검사
gz-quality run

# 자동 수정 적용
gz-quality run --fix
```bash

---

### Q6: 커밋 전에 빠르게 체크하려면?

**A**: `--staged` 플래그를 사용하세요:

```bash
# Staged 파일만 검사
gz-quality run --staged --fix

# 커밋
git commit -m "feat: add new feature"
```bash

---

### Q7: PR 전에 변경된 파일만 검사하려면?

**A**: `--since` 플래그로 특정 브랜치 이후 변경 사항만 체크:

```bash
# main 브랜치 이후 변경된 파일만
gz-quality check --since main

# 리포트 생성
gz-quality check --since main --report json --output report.json
```bash

---

### Q8: 특정 도구만 실행할 수 있나요?

**A**: 네, `tool` 명령어를 사용하세요:

```bash
# Go 포매팅만
gz-quality tool gofumpt

# Python 린팅만
gz-quality tool ruff --fix

# TypeScript 타입 체크
gz-quality tool tsc
```bash

---

### Q9: 파일 수정 없이 검사만 하려면?

**A**: `check` 명령어를 사용하세요:

```bash
# 린팅만 실행 (파일 수정 없음)
gz-quality check

# Staged 파일 검사
gz-quality check --staged
```yaml

---

### Q10: 느린 도구를 건너뛰려면?

**A**: 설정 파일에서 비활성화하세요:

```yaml
# .gzquality.yml
tools:
  pylint:
    enabled: false  # 로컬에서 비활성화

  golangci-lint:
    enabled: true
    timeout: "5m"   # 또는 타임아웃 설정
```bash

---

## 🔧 문제 해결

### Q11: "command not found" 오류가 나요

**A**: PATH에 Go bin 디렉토리를 추가하세요:

```bash
# 확인
echo $PATH

# 추가 (bash)
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc

# 추가 (zsh)
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
```bash

**상세**: [문제 해결 가이드 Q1](./05-troubleshooting.md#q1-gz-quality-command-not-found-에러)

---

### Q12: 도구가 감지되지 않아요

**A**: 자동 설치를 실행하세요:

```bash
# 프로젝트 분석
gz-quality analyze

# 필요한 도구 설치
gz-quality install

# 확인
gz-quality list --installed
```yaml

---

### Q13: 실행이 너무 느려요

**A**: 다음 방법을 시도하세요:

```bash
# 1. 변경된 파일만
gz-quality check --changed

# 2. 워커 수 증가
gz-quality run --workers 8

# 3. 느린 도구 비활성화 (.gzquality.yml)
tools:
  pylint:
    enabled: false
```bash

**상세**: [성능 최적화](./05-troubleshooting.md#q7-실행이-너무-느림-5분-이상)

---

### Q14: 특정 파일/디렉토리를 제외하려면?

**A**: `.gzquality.yml`에서 exclude 패턴 설정:

```yaml
exclude:
  - "node_modules/**"
  - "vendor/**"
  - "**/*_gen.go"
  - "dist/**"
```bash

---

### Q15: CI/CD에서 어떻게 사용하나요?

**A**: GitHub Actions 예제:

```yaml
- name: Quality Check
  run: |
    go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest
    gz-quality install
    gz-quality check --since origin/${{ github.base_ref }}
```bash

**상세**: [CI/CD 통합 가이드](../integration/CI_INTEGRATION.md)

---

## 🎯 고급 사용

### Q16: Pre-commit Hook으로 자동화하려면?

**A**: Hook 스크립트를 설치하세요:

```bash
# .git/hooks/pre-commit 생성
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
gz-quality run --staged --fix
EOF

# 실행 권한
chmod +x .git/hooks/pre-commit
```bash

**상세**: [Pre-commit Hooks 가이드](../integration/PRE_COMMIT_HOOKS.md)

---

### Q17: 팀 전체에 설정을 공유하려면?

**A**: `.gzquality.yml`을 Git에 커밋하세요:

```bash
# 설정 생성
gz-quality init

# 커밋
git add .gzquality.yml
git commit -m "chore: add quality config"

# 팀원들은 자동으로 사용
gz-quality run
```yaml

---

### Q18: 멀티 언어 모노레포에서 사용하려면?

**A**: 루트에 통합 설정을 만드세요:

```yaml
# .gzquality.yml
default_workers: 8

languages:
  Go:
    enabled: true
  Python:
    enabled: true
  TypeScript:
    enabled: true

exclude:
  - "**/node_modules/**"
  - "**/vendor/**"
  - "**/dist/**"
```bash

**예제**: [설정 가이드 - 멀티 언어 모노레포](./03-configuration.md#예제-4-멀티-언어-모노레포)

---

### Q19: Docker에서 실행하려면?

**A**: 컨테이너 이미지를 빌드하거나 사용하세요:

```bash
# 빌드
docker build -t gz-quality https://github.com/Gizzahub/gzh-cli-quality.git

# 실행
docker run --rm -v $(pwd):/workspace gz-quality run
```bash

**상세**: [설치 가이드 - Docker](./01-installation.md#방법-4-docker)

---

### Q20: 여러 환경에서 다른 설정을 사용하려면?

**A**: 환경별 설정 파일을 만드세요:

```bash
# 로컬 (기본)
.gzquality.yml

# CI 전용
.gzquality.ci.yml

# 실행
gz-quality check --config .gzquality.ci.yml
```bash

---

## 🤔 비교

### Q21: pre-commit과 어떻게 다른가요?

**A**:

| 특징 | gzh-cli-quality | pre-commit |
|------|-----------------|------------|
| **초점** | 코드 품질 (포매팅+린팅) | Git hook 관리 |
| **통합** | 단일 바이너리 | Python + 플러그인 |
| **병렬 실행** | ✅ 내장 | ✅ 지원 |
| **Git 통합** | ✅ --staged, --since | ✅ Hook 기반 |
| **설정** | YAML 파일 | YAML 파일 |
| **CI/CD** | ✅ 최적화됨 | ⚠️ 추가 설정 필요 |

**함께 사용 가능**: pre-commit에서 gz-quality를 호출하는 것도 가능합니다.

---

### Q22: Makefile/npm scripts와 어떻게 다른가요?

**A**:

**이전 (Makefile)**:
```makefile
fmt:
    gofumpt -w .
    black .
    prettier --write .

lint:
    golangci-lint run
    ruff check .
    eslint .
```bash

**이후 (gz-quality)**:
```bash
# 하나의 명령어로 통합
gz-quality run
```bash

**장점**:
- 병렬 실행으로 더 빠름
- Git 통합 (--staged, --since)
- 도구 자동 감지
- 표준화된 설정

---

### Q23: 왜 golangci-lint가 아닌 gz-quality를 사용하나요?

**A**: golangci-lint는 **Go 전용**이지만, gz-quality는 **멀티 언어**를 지원합니다.

**사용 시나리오**:
- **Go 프로젝트만**: golangci-lint도 충분
- **멀티 언어 프로젝트**: gz-quality가 필수
- **팀 표준화**: gz-quality로 통일된 워크플로우

**함께 사용**: gz-quality는 내부적으로 golangci-lint를 호출합니다.

---

## 📊 성능

### Q24: 얼마나 빠른가요?

**A**: 벤치마크 결과:

| 작업 | 시간 | 비고 |
|------|------|------|
| Registry 조회 | 14ns | 도구 찾기 |
| 파일 필터링 | 8ns | Glob 매칭 |
| 전체 실행 (100 파일) | 2-3초 | 4 workers |

**최적화 팁**: [설정 가이드 - 성능 최적화](./03-configuration.md#고급-설정)

---

### Q25: 대규모 모노레포에서도 빠른가요?

**A**: 네, 다음 전략을 사용하세요:

```bash
# 1. 변경된 파일만
gz-quality check --since main

# 2. 워커 수 증가
gz-quality run --workers 16

# 3. 캐싱 활용 (golangci-lint 자동)
```bash

**예제**: [멀티 리포지토리 워크플로우](../integration/MULTI_REPO_WORKFLOWS.md)

---

## 🆘 도움받기

### Q26: 버그를 발견했어요

**A**: GitHub Issues에 리포트해주세요:

1. https://github.com/Gizzahub/gzh-cli-quality/issues
2. 버그 템플릿 작성
3. 다음 정보 포함:
   - `gz-quality version` 출력
   - 재현 단계
   - 예상 동작 vs 실제 동작

---

### Q27: 새로운 기능을 제안하고 싶어요

**A**: Feature Request를 생성하세요:

1. GitHub Issues → New Issue → Feature Request
2. 다음 내용 포함:
   - 기능 설명
   - 사용 사례
   - 기대 효과

---

### Q28: 기여하고 싶어요

**A**: 환영합니다! 기여 가이드를 참조하세요:

- [CONTRIBUTING.md](../../CONTRIBUTING.md)
- [도구 추가 가이드](../developer/ADDING_TOOLS.md)
- [개발자 문서](../developer/)

---

### Q29: 문서가 부족한 부분이 있어요

**A**: 문서 개선 제안:

1. GitHub Issues 생성
2. 또는 Pull Request로 직접 수정
3. `docs/` 디렉토리의 마크다운 파일 수정

---

### Q30: 상업적으로 사용할 수 있나요?

**A**: 네! MIT 라이선스로 배포됩니다.

- ✅ 상업적 사용 가능
- ✅ 수정 가능
- ✅ 배포 가능
- ✅ Private 사용 가능

**조건**: 라이선스 표기 유지

**상세**: [LICENSE](../../LICENSE) 참조

---

## 📚 추가 리소스

### 더 알아보기

- **[빠른 시작 가이드](./00-quick-start.md)** - 5분 만에 시작
- **[설치 가이드](./01-installation.md)** - 상세 설치 방법
- **[설정 가이드](./03-configuration.md)** - 완벽한 설정 레퍼런스
- **[사용 예제](./02-examples.md)** - 실전 워크플로우
- **[문제 해결](./05-troubleshooting.md)** - 상세 트러블슈팅

### 통합 가이드

- **[CI/CD 통합](../integration/CI_INTEGRATION.md)** - GitHub Actions, GitLab CI 등
- **[Pre-commit Hooks](../integration/PRE_COMMIT_HOOKS.md)** - 자동화 설정
- **[멀티 리포지토리](../integration/MULTI_REPO_WORKFLOWS.md)** - 대규모 프로젝트

### 개발자 문서

- **[아키텍처](../developer/ARCHITECTURE.md)** - 시스템 설계
- **[API 레퍼런스](../developer/API.md)** - Go 패키지 API
- **[도구 추가](../developer/ADDING_TOOLS.md)** - 새 도구 통합

---

**질문이 더 있으신가요?**

- 💬 [GitHub Discussions](https://github.com/Gizzahub/gzh-cli-quality/discussions)
- 🐛 [GitHub Issues](https://github.com/Gizzahub/gzh-cli-quality/issues)
- 📧 프로젝트 메인테이너에게 문의

---

**마지막 업데이트**: 2025-12-01
````
