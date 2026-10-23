# Markdown formatter 적용 기록

## 현재 계약

선택기는 저장소 전용 Python `scripts/mdformat-files.py`이며, installer는 healthy
runtime도 Python >=3.11인지 확인한다. `UV_PYTHON`이 unset일 때만 >=3.11 runtime을
기본 선택하고 명시값은 덮어쓰지 않는다. changed 검사는 `origin/master` merge-base,
committed·staged·unstaged·untracked Markdown과 type-change를 포함하며 index/worktree
불일치는 실패한다. Git index 120000 symlink(Windows에서 일반 파일로 materialize된 경우
포함)는 제외한다.

CI는 PR base SHA, dev branch의 `origin/master`, trunk push의 before SHA를 사용하고
수동 실행은 전체 검사다. Ubuntu와 Windows native smoke는 각각 10분 초기 예산을 쓴다.
constraints와 top-level version/plugin 검사는 installer 계약으로 계속 유지한다.

## 계약과 범위

ISSUE-352의 고정 묶음은 mdformat 1.0.0, mdformat-gfm 1.0.0,
mdformat-frontmatter 2.1.2다. `make install-mdformat`은 uv로 저장소의
`bin/tools`에 설치하고 정확한 버전·플러그인 집합과 constraints의 다섯 하위
의존성을 확인한다. 정상 버전 문자열을 출력한 뒤 실패한 명령도 거부한다.

기존 `fmt-md`를 저장소 전용 실행 파일로 전환했다. Git이 추적하거나 ignore하지
않은 일반 `.md` 파일만 선택하며 symlink·디렉터리와 ignored venv를 제외한다.
공백·개행은 NUL 구분으로 보존하고, `./` 접두사로 `-` 시작 파일명을 처리한다.
Bash와 NUL 입력을 지원하는 xargs를 사용하며 Git·formatter 실패를 Make에 전파한다.

원래 target은 formatter가 없어도 경고 후 성공했고, formatter 실패 뒤에도 성공
메시지를 출력해 실패를 가릴 수 있었다. 이제 설치·검증·정렬이 성공한 뒤 완료를 표시한다.
Go 도구 pin·기존 품질 프로파일과 전역 formatter는 변경하지 않았다.
`quality`는 기존 Go fmt·lint·test를 유지하면서 `origin/master` 이후 Markdown을
읽기 전용으로 검사한다. 전체 검사는 `make format-md-all-check`로 명시적으로 실행한다.

## 검증 증거

base는 `a7f8ccab204f272d9de3735f78e58423fd819d54`이며,
구현·독립 리뷰 요구 등급은 strong이다.
기존 Git Markdown 35개 중 33개는 일반 파일, 두 개는 symlink다.
일반 파일 33개의 구·신 formatter 출력은 모두 byte 동일했고 모두 기존부터
canonical 서식과 달랐다. 이 문서들을 일괄 변경하지 않았다.

`tmp/issue352/regression.json`은 문서별 출력 비교와 다음 진단 결과를 보존한다.

- YAML frontmatter·표 처리·재실행 멱등성과 미정렬 입력 거부.
- 플러그인 순서 허용, 잘못된 버전·추가·누락 플러그인 및 정상 출력 뒤 exit 42 거부.
- 다섯 하위 의존성 검사와 불일치 거부.
- 공백·개행·선행 `-` 경로, Markdown 확장자 제한, symlink·디렉터리·ignored venv 제외.
- 빈 선택과 마지막 제외 파일의 정상 처리, Git·formatter 실패 전파.

설치는 exit 0(1.713초), 좁은 회귀는 exit 0(26.894초)이었다.
진단 코드는 ignored `tmp/issue352`에 두고 소유자·base·보존 조건을 기록했다.
실행 결과와 원본 로그를 통합·보드 기록까지 보존한다.

## 외부 워커와 검증 경계

2026-10-22 KST 01시대에 GLM Flash를 호출했으며 평일 15~19시 금지 시간대 밖이었다.
`claudz --model sonnet --effort medium`은 startup 로그에서 glm-5.3-flash 매핑을
보였고, Make 설정 세 파일을 수정한 뒤 12턴 제한으로 exit 1이었다.
부모가 남은 diff를 검사하고 설치·회귀를 실행했으므로 워커 종료를 성공으로 계산하지 않는다.
`grok --effort high --always-approve`의 제한된 분석은 exit 0이었다.
Grok의 확장자 필터 제안은 실제 Git pathspec과 회귀 assertion으로 확인했다.
분석 결과만으로 구현·통합 PASS를 주장하지 않는다.

커밋 검사는 기존 `make quality`(fmt·lint·전체 race test)를 사용한다.
현재 입력에서 exit 0(13.352초)이었고 실행 후 Go diff는 없었다.
`tmp/issue352/checks.json`과 `quality.log`에 결과를 보존하며 성능 개선을 주장하지 않는다.
테스트 timeout은 기존 5분이고 외부 진단 제한은 임시 1800초다.
프로젝트 timeout·병렬도와 상세 메트릭 수집 설계는 이번 범위에서 변경하지 않았다.
CE lifecycle 선언이 없어 run-doctor는 BLOCKED였으며 기존 Git worktree 절차로 격리했다.
저장소 전용 lifecycle 스크립트는 추가하지 않았다.

native 통합·master push·제품 설치·release·Windows 실행·Python 선택 강제는 미검증이다.
실제 전달 결과는 별도 영수증으로 기록해야 한다.
