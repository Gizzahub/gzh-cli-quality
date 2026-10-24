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

Release workflow는 `contents: read`로 실행하는 비게시 빌드 경로다.

- 브랜치의 `workflow_dispatch`: 같은 출시 품질 검사와 snapshot 빌드를 수행하고
  `release-candidate` artifact를 업로드한다.
- `v*` 태그 push: 같은 출시 품질 검사 후 고정 GoReleaser의
  `release --skip=publish --clean`으로 태그 버전의 archive·checksum·cask를 만들고
  `release-tag-build` artifact를 업로드한다. snapshot이 아니므로 태그 검증은 유지한다.
- 어느 이벤트도 GitHub Release·tap을 게시하거나 쓰기 권한을 증명하지 않는다.
  workflow의 `GITHUB_TOKEN`은 읽기 전용이며 정적 tap PAT secret은 사용하지 않는다.
  `--skip=publish`는 v2.19.0에서 지원하는 비게시 옵션이다.

## 태그 우선 로컬 게시 계약

실제 게시 주체는 호스트의 infra 소유 GitHub App credential broker와 로컬
GoReleaser 실행 경로다. broker의 그룹·명령·권한 정책·credential 보관은 infra가
소유하며 이 저장소에서 복제하지 않는다. 정적 PAT나 저장소 Actions secret의
존재를 게시 권한의 증거로 삼지 않는다. 토큰 값은 소스·로그·문서에 기록하지 않는다.

1. 출시 품질 검사·독립 리뷰·정확한 후보 SHA·새 patch 태그를 확정하고 승인받는다.
1. 후보를 integration branch에 반영한 뒤 승인된 Git 경로로 태그를 먼저 push한다.
   원격 태그를 commit까지 peel한 SHA와 로컬 clean checkout의 SHA가 정확히
   같은지 재확인한다. 태그가 없거나 다르면 게시를 중단한다.
1. build-only 태그 workflow와 산출물을 확인한다. Actions가 게시와 경쟁하지 않으므로
   수동 게시와 tag push가 같은 공개 asset을 덮어쓰는 경로를 만들지 않는다.
1. infra broker에서 source release와 tap에 필요한 Contents 쓰기 범위·수명을
   제한한 App installation credential을 증명한다. 증명·갱신·회수는 broker 계약을
   따른다. credential이 발급됐다는 사실만으로 쓰기 성공을 주장하지 않는다.
1. broker가 credential을 주입한 로컬 process에서 고정 GoReleaser의 정상
   `release --clean`을 실행한다. source credential은 `GITHUB_TOKEN`, tap credential은
   기존 config의 `HOMEBREW_TAP_TOKEN` 환경 슬롯에 전달한다. 이름이 같아도 정적 PAT를
   뜻하지 않는다. shell 인자·파일·원시 로그에 credential을 넣지 않는다.
1. 공개 뒤 실제 URL·checksum, cask strict online audit, 격리 fresh install·
   실제 이전 공개 버전에서의 upgrade·버전 출력을 검증한다.

v2.19.0에는 별도의 artifact-only `publish` 명령이 없다. 정상 `release`는 다시
빌드·검증·게시하므로 앞선 CI archive와 같다고 가정하지 않고 최종 공개 bytes를
확인한다. 정상 게시에 `--snapshot`이나 validation 생략을 사용하지 않는다.
기존 공개 immutable release를 재실행해 덮어쓰지 않는다. release 공개 뒤 tap만
실패하면 전체 게시를 재실행하지 않고, 검증한 공개 manifest에 맞는 tap 단계만 복구한다.

### target_commitish와 권한 경계

`.goreleaser.yml`은 `release.target_commitish`를 지정하지 않는다. 고정 v2.19.0의
GitHub client는 이 값이 비어 있으면 요청에 `target_commitish`를 넣지 않는다.
[GitHub 공식 API](https://docs.github.com/en/rest/releases/releases#create-a-release)는
기존 Git 태그가 있으면 이 값을 사용하지 않으며, 생략 시 default branch를
기본값으로 설명한다. workflow 파일을 default branch와 다르게 만드는 target은
별도 Workflows 쓰기 권한을 요구할 수 있으므로, tag-first/default-branch 통합을
검증하고 임의의 historical `target_commitish`나 Workflows 쓰기 권한을 추가하지 않는다.
이 소스 분석은 실제 App credential 권한 증명이나 게시 성공을 대신하지 않는다.

다음 patch 준비 대상은 `v0.1.4`이며 아직 이 문서만으로 승인·태그·공개를 뜻하지 않는다.
Go 설치 경로는 `github.com/Gizzahub/gzh-cli-quality/cmd/quality`이며,
Go가 기본으로 만드는 파일 이름은 `quality`다. 배포 압축파일과 cask는
`gz-quality`라는 파일 이름을 사용한다.

## 2026-10-23 확인 결과

당시 `make release-quality`와 snapshot 산출물 생성은 통과했으며 전체
커버리지는 87.5686%다. 분모·제외 대상·80% 하한을 변경하지 않았다.
도구 전용 인자·파서 실행, 비파괴 검사와 명시적 fix, 취소·실패 처리와
캐시 일부 적중 시 실패 집계에 회귀 테스트를 추가했다.

Registry 조회는 6.749–6.895 ns/op로 20ns 상한을 충족했다. 기존 no-op
도구 벤치마크에서는 캐시 적중 168–170µs, 비캐시 3.3µs로 캐시가 느리다.
이는 외부 도구 비용이 없는 workload이며 실제 linter 실행을 대표하지 않는다.
PRODUCT.md의 캐시 60배 향상 조건은 아직 입증하지 못했다.
당시 품질 검사 통과를 제품 출시 승인으로 해석하지 않아 게시를 보류했다.
이 절은 초기 조사 기록이며 이후 공개된 v0.1.3의 상태를 미게시로 되돌리는 설명이 아니다.
릴리스 준비 작업의 세부 증거와 미해결 항목은 devbox의 TASK-350 및
ISSUE-362에서 추적한다.
