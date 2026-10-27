# Contributing to gzh-cli-quality

gzh-cli-quality에 기여해주셔서 감사합니다! 이 문서는 프로젝트에 기여하는 방법을 안내합니다.

## 개발 환경 설정

### 요구사항

- Go 1.26 이상 (개발 toolchain: Go 1.27.3, `go.mod` 기준)
- Make
- Git
- golangci-lint (optional, `make lint` 실행시 필요)

### 프로젝트 클론

```bash
git clone https://github.com/Gizzahub/gzh-cli-quality.git
cd gzh-cli-quality
```

### 의존성 설치

```bash
go mod download
```

### 빌드 및 테스트

```bash
# 빌드
make build

# 테스트 실행
make test

# 린트 실행
make lint

# 전체 품질 검사
make quality
```

## 개발 워크플로우

### 1. 브랜치 생성

```bash
git checkout -b feature/your-feature-name
# 또는
git checkout -b fix/your-bug-fix
```

### 2. 코드 작성

- 기존 코드 스타일을 따라주세요
- 새로운 기능에는 테스트를 추가해주세요
- 공개 API에는 문서 주석을 작성해주세요

### 3. 품질 검사

```bash
# 포매팅
go fmt ./...

# 린트
make lint

# 테스트
make test

# 전체 품질 검사
make quality
```

### 4. 커밋

커밋 메시지는 [Conventional Commits](https://www.conventionalcommits.org/) 규칙을 따릅니다:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Type:**

- `feat`: 새로운 기능
- `fix`: 버그 수정
- `docs`: 문서 변경
- `style`: 코드 포매팅, 세미콜론 누락 등
- `refactor`: 리팩토링
- `test`: 테스트 추가
- `chore`: 빌드 프로세스, 도구 설정 등

**예시:**

```
feat(detector): Add PHP language detection

- Implement PHP file type detector
- Add PHPStan and PHP-CS-Fixer tool support
- Update language registry

Closes #123
```

### 5. Pull Request

1. Fork 저장소
1. 브랜치에서 작업
1. 품질 검사 통과 확인
1. Pull Request 생성
1. CI 통과 대기
1. 리뷰 반영

## 코드 스타일 가이드

### Go 코드

- `go fmt`로 포매팅
- `golangci-lint`로 린트
- 공개 함수/타입에 주석 작성
- 에러 처리 누락 금지
- 테스트 커버리지 유지

### 네이밍

- 파일: `snake_case.go`
- 타입: `PascalCase`
- 함수: `PascalCase` (exported), `camelCase` (unexported)
- 상수: `PascalCase` 또는 `SCREAMING_SNAKE_CASE`
- 변수: `camelCase`

### 테스트

- 테스트 파일: `*_test.go`
- 테스트 함수: `TestFunctionName`
- Table-driven tests 권장
- 모킹은 최소화

```go
func TestNewTool(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"valid input", "test", "test", false},
		{"empty input", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTool(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NewTool() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

## 새로운 도구 추가

새로운 품질 도구를 추가하는 방법은 [docs/developer/ADDING_TOOLS.md](docs/developer/ADDING_TOOLS.md)를 참조하세요.

### 체크리스트

- [ ] `QualityTool` 인터페이스 구현
- [ ] 도구 설치 지침 추가
- [ ] 설정 파일 감지 구현
- [ ] 출력 파싱 구현
- [ ] 단위 테스트 작성
- [ ] 통합 테스트 작성
- [ ] 문서 업데이트

## 릴리스 프로세스

릴리스는 메인테이너가 수행합니다:

1. 버전 태그 생성: `git tag -a v1.2.3 -m "Release v1.2.3"`
1. 태그 푸시: `git push origin v1.2.3`
1. GitHub Actions가 자동으로 릴리스 생성
1. GoReleaser가 멀티 플랫폼 바이너리 빌드

## 이슈 리포팅

버그 리포트나 기능 요청은 [GitHub Issues](https://github.com/Gizzahub/gzh-cli-quality/issues)에 등록해주세요.

### 버그 리포트

다음 정보를 포함해주세요:

- gz-quality 버전: `gz-quality version`
- Go 버전: `go version`
- OS 및 아키텍처
- 재현 단계
- 예상 동작
- 실제 동작
- 로그 또는 오류 메시지

### 기능 요청

다음 정보를 포함해주세요:

- 기능 설명
- 사용 사례
- 예상 동작
- 대안 검토 여부

## 라이선스

기여하신 코드는 [MIT License](LICENSE)로 배포됩니다.

## 행동 강령

- 서로 존중하고 배려해주세요
- 건설적인 피드백을 제공해주세요
- 다양성과 포용성을 존중해주세요

## 질문?

질문이 있으시면 [GitHub Discussions](https://github.com/Gizzahub/gzh-cli-quality/discussions)에 올려주세요.

______________________________________________________________________

다시 한번 기여해주셔서 감사합니다! 🙏
