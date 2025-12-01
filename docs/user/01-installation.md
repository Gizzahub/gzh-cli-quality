# 설치 가이드

gzh-cli-quality를 시스템에 설치하는 모든 방법을 안내합니다.

## 목차

- [시스템 요구사항](#시스템-요구사항)
- [gz-quality 설치](#gz-quality-설치)
- [품질 도구 설치](#품질-도구-설치)
- [설치 확인](#설치-확인)
- [문제 해결](#문제-해결)

---

## 시스템 요구사항

### 필수 요구사항

| 항목 | 버전 | 확인 방법 |
|------|------|----------|
| **Go** | 1.24.0 이상 | `go version` |
| **Git** | 2.0 이상 | `git --version` |

### 선택 요구사항 (언어별)

프로젝트에서 사용하는 언어에 따라 해당 도구가 필요합니다:

| 언어 | 도구 | 설치 방법 |
|------|------|----------|
| **Go** | gofumpt, goimports, golangci-lint | [Go 도구 설치](#go-도구) |
| **Python** | black, ruff, pylint | [Python 도구 설치](#python-도구) |
| **JavaScript/TypeScript** | prettier, eslint, tsc | [Node 도구 설치](#javascripttypescript-도구) |
| **Rust** | rustfmt, clippy | [Rust 도구 설치](#rust-도구) |

---

## gz-quality 설치

### 방법 1: Go Install (권장)

가장 간단하고 빠른 설치 방법입니다.

```bash
# 최신 안정 버전
go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest

# 특정 버전
go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@v0.1.1

# 설치 확인
gz-quality version
```

**PATH 설정**:

Go로 설치한 바이너리는 `$GOPATH/bin`에 저장됩니다. PATH에 추가하세요:

```bash
# Bash 사용자
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc

# Zsh 사용자 (macOS 기본)
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc

# Fish 사용자
set -Ua fish_user_paths (go env GOPATH)/bin

# 확인
which gz-quality
gz-quality version
```

---

### 방법 2: 소스에서 빌드

개발 버전이나 커스텀 빌드가 필요한 경우:

```bash
# 1. 리포지토리 클론
git clone https://github.com/Gizzahub/gzh-cli-quality.git
cd gzh-cli-quality

# 2. 빌드
make build

# 3. 바이너리 확인
./build/gz-quality version

# 4. (선택) 시스템에 설치
make install  # $GOPATH/bin에 설치
```

**빌드 옵션**:

```bash
# 릴리스 빌드 (최적화)
make build

# 개발 빌드 (디버그 정보 포함)
go build -o build/gz-quality ./cmd/gz-quality

# 특정 OS/아키텍처용 빌드
GOOS=linux GOARCH=amd64 go build -o build/gz-quality-linux-amd64 ./cmd/gz-quality
GOOS=darwin GOARCH=arm64 go build -o build/gz-quality-darwin-arm64 ./cmd/gz-quality
GOOS=windows GOARCH=amd64 go build -o build/gz-quality-windows-amd64.exe ./cmd/gz-quality
```

---

### 방법 3: 바이너리 다운로드

GitHub Releases에서 미리 빌드된 바이너리를 다운로드할 수 있습니다.

**Linux (x86_64)**:
```bash
# 다운로드
curl -LO https://github.com/Gizzahub/gzh-cli-quality/releases/latest/download/gz-quality-linux-amd64

# 실행 권한 부여
chmod +x gz-quality-linux-amd64

# 설치
sudo mv gz-quality-linux-amd64 /usr/local/bin/gz-quality

# 확인
gz-quality version
```

**macOS (Intel)**:
```bash
# 다운로드
curl -LO https://github.com/Gizzahub/gzh-cli-quality/releases/latest/download/gz-quality-darwin-amd64

# 실행 권한 부여
chmod +x gz-quality-darwin-amd64

# 설치
sudo mv gz-quality-darwin-amd64 /usr/local/bin/gz-quality

# 확인
gz-quality version
```

**macOS (Apple Silicon)**:
```bash
# 다운로드
curl -LO https://github.com/Gizzahub/gzh-cli-quality/releases/latest/download/gz-quality-darwin-arm64

# 실행 권한 부여
chmod +x gz-quality-darwin-arm64

# 설치
sudo mv gz-quality-darwin-arm64 /usr/local/bin/gz-quality

# 확인
gz-quality version
```

**Windows (PowerShell)**:
```powershell
# 다운로드
Invoke-WebRequest -Uri "https://github.com/Gizzahub/gzh-cli-quality/releases/latest/download/gz-quality-windows-amd64.exe" -OutFile "gz-quality.exe"

# PATH에 추가 (관리자 권한 필요)
Move-Item gz-quality.exe C:\Windows\System32\

# 확인
gz-quality version
```

---

### 방법 4: Docker

Docker 컨테이너로 실행:

```bash
# 이미지 빌드
docker build -t gz-quality:latest https://github.com/Gizzahub/gzh-cli-quality.git

# 실행
docker run --rm -v $(pwd):/workspace gz-quality:latest run

# Alias 설정
alias gz-quality='docker run --rm -v $(pwd):/workspace gz-quality:latest'
```

**Docker Compose**:

```yaml
# docker-compose.yml
version: '3.8'
services:
  quality:
    build: https://github.com/Gizzahub/gzh-cli-quality.git
    volumes:
      - .:/workspace
    working_dir: /workspace
```

```bash
# 실행
docker-compose run --rm quality run
```

---

## 품질 도구 설치

gz-quality가 설치되면 필요한 품질 도구를 설치합니다.

### 자동 설치 (권장)

```bash
# 프로젝트 분석 및 권장 도구 확인
gz-quality analyze

# 권장 도구 자동 설치
gz-quality install

# 특정 언어 도구만 설치
gz-quality install --language Go
gz-quality install --language Python

# 모든 지원 도구 설치
gz-quality install --all
```

---

### Go 도구

#### gofumpt (필수)

```bash
# 설치
go install mvdan.cc/gofumpt@latest

# 확인
gofumpt -version
```

#### goimports (필수)

```bash
# 설치
go install golang.org/x/tools/cmd/goimports@latest

# 확인
goimports -h
```

#### golangci-lint (필수)

```bash
# macOS/Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# 또는 Go install
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Windows (PowerShell, 관리자 권한)
Invoke-WebRequest -Uri https://github.com/golangci/golangci-lint/releases/latest/download/golangci-lint-windows-amd64.exe -OutFile golangci-lint.exe
Move-Item golangci-lint.exe $env:GOPATH\bin\

# 확인
golangci-lint --version
```

---

### Python 도구

#### black (필수)

```bash
# pip 사용
pip install black

# pipx 사용 (권장)
pipx install black

# uv 사용 (최신)
uv tool install black

# 확인
black --version
```

#### ruff (필수)

```bash
# pip 사용
pip install ruff

# pipx 사용 (권장)
pipx install ruff

# uv 사용 (최신)
uv tool install ruff

# 확인
ruff --version
```

#### pylint (선택)

```bash
# pip 사용
pip install pylint

# pipx 사용
pipx install pylint

# 확인
pylint --version
```

**Python 도구 PATH 설정**:

```bash
# pip로 설치한 경우
export PATH="$HOME/.local/bin:$PATH"

# pipx로 설치한 경우 (자동으로 PATH 추가됨)
pipx ensurepath
```

---

### JavaScript/TypeScript 도구

#### Node.js 설치 확인

```bash
# Node.js 버전 확인
node --version

# npm 버전 확인
npm --version
```

**Node.js 설치 (필요시)**:

```bash
# macOS (Homebrew)
brew install node

# Ubuntu/Debian
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

# Windows (Chocolatey)
choco install nodejs
```

#### prettier (필수)

```bash
# 전역 설치
npm install -g prettier

# 또는 프로젝트별 설치
npm install --save-dev prettier

# 확인
prettier --version
```

#### eslint (필수)

```bash
# 전역 설치
npm install -g eslint

# 또는 프로젝트별 설치
npm install --save-dev eslint

# 확인
eslint --version
```

#### TypeScript (TypeScript 프로젝트 전용)

```bash
# 전역 설치
npm install -g typescript

# 또는 프로젝트별 설치
npm install --save-dev typescript

# 확인
tsc --version
```

---

### Rust 도구

#### Rust 설치 확인

```bash
# Rust 버전 확인
rustc --version
cargo --version
```

**Rust 설치 (필요시)**:

```bash
# Linux/macOS
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Windows
# https://rustup.rs/ 에서 rustup-init.exe 다운로드 후 실행
```

#### rustfmt (필수)

```bash
# 설치
rustup component add rustfmt

# 확인
rustfmt --version
cargo fmt --version
```

#### clippy (필수)

```bash
# 설치
rustup component add clippy

# 확인
cargo clippy --version
```

---

## 설치 확인

### 전체 확인

```bash
# gz-quality 버전 및 도구 상태
gz-quality version

# 프로젝트 분석
cd your-project
gz-quality analyze

# 사용 가능한 도구 목록
gz-quality list

# 설치된 도구만 표시
gz-quality list --installed
```

### 출력 예시

```
gzh-cli-quality v0.1.1

설치된 도구:
  gofumpt       v0.6.0      /Users/user/go/bin/gofumpt
  goimports     v0.16.1     /Users/user/go/bin/goimports
  golangci-lint v1.55.2     /Users/user/go/bin/golangci-lint
  black         24.1.0      /Users/user/.local/bin/black
  ruff          0.1.14      /Users/user/.local/bin/ruff
  prettier      3.1.0       /usr/local/bin/prettier
  eslint        8.56.0      /usr/local/bin/eslint
  tsc           5.3.3       /usr/local/bin/tsc

✅ 8개 도구 설치됨
```

---

## 문제 해결

### "gz-quality: command not found"

**원인**: PATH 설정 누락

**해결**:
```bash
# GOPATH 확인
go env GOPATH

# PATH에 추가
export PATH="$PATH:$(go env GOPATH)/bin"

# 영구 적용 (.bashrc, .zshrc 등에 추가)
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc
```

---

### "go: command not found"

**원인**: Go 미설치

**해결**:

**macOS**:
```bash
brew install go
```

**Linux (Ubuntu/Debian)**:
```bash
sudo apt update
sudo apt install golang-go
```

**Windows**:
1. https://go.dev/dl/ 에서 설치 프로그램 다운로드
2. 설치 후 시스템 재시작

---

### 도구가 감지되지 않음

**원인**: 도구 미설치 또는 PATH 문제

**해결**:

```bash
# 1. 도구 설치 확인
which gofumpt
which black
which prettier

# 2. 없으면 설치
gz-quality install

# 3. PATH 확인
echo $PATH

# 4. 수동 설치 (예: gofumpt)
go install mvdan.cc/gofumpt@latest
```

---

### 권한 오류 (Permission denied)

**원인**: 설치 디렉토리 쓰기 권한 없음

**해결**:

**Linux/macOS**:
```bash
# 사용자 디렉토리에 설치 (sudo 불필요)
# Go 도구는 $GOPATH/bin에 자동 설치됨

# Python 도구는 --user 플래그 사용
pip install --user black ruff

# npm은 prefix 설정
npm config set prefix ~/.npm-global
export PATH=~/.npm-global/bin:$PATH
```

**Windows**:
- PowerShell을 관리자 권한으로 실행
- 또는 사용자 디렉토리에 설치

---

### macOS "개발자를 확인할 수 없음" 오류

**원인**: macOS Gatekeeper 보안

**해결**:
```bash
# 1. 바이너리 실행 권한 부여
xattr -d com.apple.quarantine /usr/local/bin/gz-quality

# 2. 또는 시스템 환경설정에서 "보안 및 개인 정보 보호" → "일반" → "확인 없이 열기"
```

---

### Docker 관련 문제

**문제**: "Cannot connect to Docker daemon"

**해결**:
```bash
# Docker 실행 확인
docker ps

# Docker 시작
# macOS/Windows: Docker Desktop 실행
# Linux:
sudo systemctl start docker

# 현재 사용자를 docker 그룹에 추가
sudo usermod -aG docker $USER
```

---

## 업그레이드

### gz-quality 업그레이드

```bash
# Go install로 설치한 경우
go install github.com/Gizzahub/gzh-cli-quality/cmd/gz-quality@latest

# 소스에서 빌드한 경우
cd gzh-cli-quality
git pull
make build
make install
```

### 품질 도구 업그레이드

```bash
# 모든 도구 업그레이드
gz-quality upgrade

# 특정 도구만 업그레이드
gz-quality upgrade golangci-lint
gz-quality upgrade ruff

# 수동 업그레이드
go install mvdan.cc/gofumpt@latest           # Go 도구
pip install --upgrade black ruff             # Python 도구
npm update -g prettier eslint typescript     # Node 도구
rustup update                                # Rust 도구
```

---

## 제거

### gz-quality 제거

```bash
# Go install로 설치한 경우
rm $(go env GOPATH)/bin/gz-quality

# 소스에서 빌드한 경우
rm /usr/local/bin/gz-quality

# Docker 이미지 제거
docker rmi gz-quality:latest
```

### 품질 도구 제거

```bash
# Go 도구
rm $(go env GOPATH)/bin/gofumpt
rm $(go env GOPATH)/bin/goimports
rm $(go env GOPATH)/bin/golangci-lint

# Python 도구
pip uninstall black ruff pylint
# 또는
pipx uninstall black ruff pylint

# Node 도구
npm uninstall -g prettier eslint typescript

# Rust 도구
rustup component remove rustfmt clippy
```

---

## 다음 단계

설치가 완료되었습니다! 이제 다음 단계로 진행하세요:

1. **[빠른 시작 가이드](./00-quick-start.md)** - 첫 실행 및 기본 사용법
2. **[설정 가이드](./03-configuration.md)** - 프로젝트 맞춤 설정
3. **[사용 예제](./02-examples.md)** - 실전 워크플로우 패턴
4. **[CI/CD 통합](../integration/CI_INTEGRATION.md)** - 자동화 설정

---

**문제가 계속되면**: [문제 해결 가이드](./05-troubleshooting.md) 또는 [GitHub Issues](https://github.com/Gizzahub/gzh-cli-quality/issues) 참조
