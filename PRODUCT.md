# Product Goals (No-PRD)

**Project**: gzh-cli-quality (`gz-quality` binary)
**Doc Type**: Goals + Constraints + Quality Gates
**Status**: Active
**Last Updated**: 2026-07-16

______________________________________________________________________

## Product Intent

gzh-cli-quality is a **multi-language quality tool orchestrator**. Through one CLI
(and importable library) it:

- runs 11+ existing formatters and linters across Go / Python / JS·TS / Rust,
- wraps those tools without reimplementing their rules (SOUL 신념 1),
- and speeds repeated runs with a file-based result cache and parallel workers.

This is a feature-library project — a single PRODUCT.md is sufficient. It
replaces a PRD.

| 제공하는 것 (Is)                       | 되지 않을 것 (Is Not)           |
| -------------------------------------- | ------------------------------- |
| 11+ 기존 도구의 통합 실행 (fmt + lint) | linter/formatter 자체 구현      |
| 병렬 실행·결과 캐싱·git 필터           | 도구 규칙(rule)을 정의하는 엔진 |
| 다중 언어 자동 감지·무설정 시작        | 단일 언어 전용 도구             |
| JSON/HTML/Markdown 리포트              | IDE 플러그인·GUI                |

______________________________________________________________________

## Goals (Measurable Targets)

G1. **Integrated coverage**

- Target: Go·Python·JS·TS·Rust 4개 언어군, 11+ 도구를 단일 CLI로 제공

G2. **Cache efficiency**

- Target: 파일 기반 결과 캐시가 모두 적중하면 결과를 만드는 도구 실행 0회.
  quality 자체 처리 시간은 지정된 Darwin gofumpt workload에서 파일당 100µs 이하
  (`BenchmarkCache_InstalledGofumpt` `-count=6` WarmHit 중앙값 ÷ 파일 수,
  workload와 측정 조건은 BENCHMARKS.md).
- 버전 조회 subprocess 0회는 위 측정에서 실행 파일의 강한 identity로 버전 캐시를
  재사용할 수 있을 때의 조건이다. Windows와 Rust 등의 안전한 버전 재조회는 유지하며,
  모든 플랫폼·도구에 버전 subprocess 0회나 같은 시간 상한을 보증하지 않는다.
- 비캐시 대비 배율은 참고값이다. 배율은 도구 실행 비용 ÷ quality 고정 비용이라 도구마다
  달라지고 quality가 소유하지 않는 도구 비용에 좌우된다. 2026-10-25 이전 목표 60배는
  gofumpt workload에서 버전 캐시 추가 후에도 약 6.8~7.9배로 미충족이었다
  ([BENCHMARKS.md](docs/developer/BENCHMARKS.md)).

G3. **Parallel throughput**

- Target: Worker Pool로 CPU 코어 수만큼 병렬; Registry 조회 < 20ns, 필터링 < 10ns

G4. **Git-scoped runs**

- Target: `--staged` / `--changed` / `--since <ref>`로 변경 파일만 선택 검사

G5. **Test reliability**

- Target: 커버리지 >= 80% (현재 76.2% → 목표 상향)

______________________________________________________________________

## Non-Goals (Explicitly Out of Scope)

- No linter/formatter 자체 개발 (기존 도구를 감쌀 뿐)
- No 도구 규칙(rule)을 정의하는 엔진
- No 단일 언어 전용화 — 다중 언어 오케스트레이션이 정체성
- No IDE 플러그인·GUI
- No CI/CD 시스템 자체 (파이프라인용 출력·종료 코드만 제공)

______________________________________________________________________

## Guardrails and Technical Constraints

**Architecture**

- 플러그인형 tool 추상화; `executor`/`detector`/`report`/`cache` 관심사 분리
- 외부 도구 실행 시 입력을 sanitize한다 (command injection 방지)

**Dependency Boundaries**

- `gzh-cli-core`만 의존 가능; 다른 feature 라이브러리 의존 금지 (GUIDELINES §2)
- 언어별 품질 도구는 외부 설치 대상 (`gz-quality install` 지원)

**Compatibility**

- Go 1.25+ (`go.mod` go 1.25.7; devbox 툴체인 1.26); Git 2.0+

**Safety**

- `--fix`는 명시적 opt-in — 기본 동작은 비파괴적 검사(`check`)

______________________________________________________________________

## Quality Gates (Release Readiness)

**Build and Lint**

- `make quality` (fmt + lint + test) pass with no warnings

**Testing**

- `make test-coverage` pass; 커버리지 >= 80%

**Performance**

- Registry 조회 < 20ns; 결과 캐시 전부 적중 시 결과 도구 실행 0회.
  G2에 지정한 Darwin gofumpt 측정에서 버전 subprocess 0회·파일당 100µs 이하.

**Docs**

- 지원 도구·CLI 명령 레퍼런스가 실제 동작·플래그와 일치

______________________________________________________________________

## Decision Rules

- 새 도구/언어 지원은 "기존 도구 감싸기"여야 한다 — 자체 linter 구현은
  SOUL 게이트 1(재발명 금지)에서 거절된다
- 새 기능은 SOUL.md 4-게이트(틈 · 라이브러리 · 대량/전환 · 날카로움)를 통과해야 한다
- 최소 하나의 goal에 매핑되거나 명시적으로 승인되어야 한다
- Guardrails 위반은 문서화된 예외를 요구한다
- Quality Gates 미충족 시 릴리스는 차단된다

______________________________________________________________________

**End of Document**
