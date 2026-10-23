# Homebrew 배포

## 배포 계약

독립 실행 파일은 `gz-quality`이며 진입점은 `cmd/quality`다.
GoReleaser가 macOS·Linux·Windows의 amd64·arm64 압축파일과 SHA-256
체크섬을 만들고, `Gizzahub/homebrew-tap`의 `Casks/gz-quality.rb`를 갱신한다.
압축파일에는 실행 파일·LICENSE·README.md가 포함된다.

배포 도구 버전의 정본은 `.make/vars.mk`의 `GORELEASER_VERSION`이다.
로컬 `make install-goreleaser`는 그 버전을 저장소의 `bin/tools`에 설치하며,
GitHub Actions도 `make print-goreleaser-version`으로 같은 값을 읽는다.
전역 GoReleaser 설치를 변경하지 않는다.

## 출시 전 검증

`make release-quality`는 기존 `make quality`의 포맷·린트·race 테스트를
실행한 뒤 PRODUCT.md의 전체 커버리지 하한 80%를 검사한다.
`make release-coverage-check`는 이미 만들어진 `coverage.out`만 검사한다.
보고서가 없거나 하한 미달이면 실패한다. 원시 statement 수로 비교하므로
반올림해서 80.0%로 표시되는 79.99%도 거부한다. 일반 CI의 커버리지 하한과
출시 자격은 별개이며, 일반 CI 통과만으로 출시 가능하다고 판단하지 않는다.

`make release-dry`는 고정된 GoReleaser로 설정·6개 플랫폼 빌드·압축파일·
체크섬·cask 생성을 검사한다. snapshot 모드이므로 게시하지 않는다.
생성된 cask는 로컬 검사용이며, snapshot URL이 실제 릴리스에 존재한다는
뜻이 아니다. 공개 태그를 만든 뒤 생성되는 cask의 URL·체크섬도 검증해야 한다.

Release workflow를 브랜치에서 수동 실행하면 같은 출시 품질 검사와 snapshot
빌드를 수행하고 `release-candidate` artifact만 업로드한다.
수동 실행으로 GitHub Release나 tap을 갱신하지 않는다.

## 게시 조건

- 출시 품질 검사와 산출물 검증을 통과해야 한다.
- 공개할 태그와 대상 커밋을 확정하고 승인받아야 한다.
- 저장소 Actions secret `HOMEBREW_TAP_TOKEN`은
  `Gizzahub/homebrew-tap`에 Contents 쓰기 권한이 있어야 한다.
  토큰 값은 소스·로그·문서에 기록하지 않는다.
- `v*` 태그 push만 게시 경로를 실행한다. tap 토큰이 없으면
  GitHub Release 생성 전에 실패한다.
- 게시 후 실제 cask의 strict online audit, 다운로드 체크섬과
  새 설치·업그레이드 동작을 확인해야 한다.

다음 패치 버전 후보는 `v0.1.3`이다. 이는 예약·승인·태그 생성이 아니다.
Go 설치 경로는 `github.com/Gizzahub/gzh-cli-quality/cmd/quality`이며,
Go가 기본으로 만드는 파일 이름은 `quality`다. 배포 압축파일과 cask는
`gz-quality`라는 파일 이름을 사용한다.

## 2026-10-23 확인 결과

현재 `make release-quality`와 snapshot 산출물 생성은 통과했으며 전체
커버리지는 87.5686%다. 분모·제외 대상·80% 하한을 변경하지 않았다.
도구 전용 인자·파서 실행, 비파괴 검사와 명시적 fix, 취소·실패 처리와
캐시 일부 적중 시 실패 집계에 회귀 테스트를 추가했다.

Registry 조회는 6.749–6.895 ns/op로 20ns 상한을 충족했다. 기존 no-op
도구 벤치마크에서는 캐시 적중 168–170µs, 비캐시 3.3µs로 캐시가 느리다.
이는 외부 도구 비용이 없는 workload이며 실제 linter 실행을 대표하지 않는다.
PRODUCT.md의 캐시 60배 향상 조건은 아직 입증하지 못했다.
따라서 품질 검사 통과를 제품 출시 승인으로 해석하지 않으며 게시하지 않았다.
릴리스 준비 작업의 세부 증거와 미해결 항목은 devbox의 TASK-350 및
ISSUE-362에서 추적한다.
