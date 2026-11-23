# v0.1.6 - Phase 03: P1 CI Verification, Race, Fuzz & Makefile Repair

> **Parent:** [README.md](README.md)
> **Branch:** `chore/v0.1.6-go-review-hardening`
> **Priority:** P1
> **Status:** Open (6 / 6 rows implemented; 3 closure gates pending primary-session runs)

---

## Overview

`blinkless` already has strong unit tests (2,277 `Test*`) and 8 fuzz targets, but its `Makefile` and `.github/workflows/ci.yml` leave major verification gaps: `test-race` covers only 3 of 18 packages, none of the 8 fuzzers or `wasm-test` run in CI, `govulncheck` is absent, and several Makefile targets and benchmark docs still reference the deleted `./internal/convert` PDF pipeline.

---

## 3.1 Race, Fuzz, Wasm & Security CI Wiring

- [x] `BLK-CI-01` Expand `RACE_PKGS` to `./...` (at minimum adding `./css` and `./internal/pubstate`) and update stale comments (`Makefile:28-29, 48-50`, `.github/workflows/ci.yml:27`).
  - **Root cause:** `RACE_PKGS ?= ./internal/layout ./internal/load ./internal/fonts` omits `./css` (which owns `TestHoverCacheConcurrentRace`) and `./internal/pubstate` (whose `state_race_test.go:22` instructs `"Run it with -race"`), and `Makefile:28` wrongly claims `test-race` is `"Not part of GitHub CI"`.
  - **Fix:** Set `RACE_PKGS ?= ./...` (keeping `-p $(TEST_P) -parallel $(TEST_PARALLEL)` for memory control) and update the stale comment.
  - **Proof:** `make test-race` exits `0` across all 18 packages locally and in GitHub Actions CI.
  - **Evidence (2426019):** `Makefile` now has `RACE_PKGS ?= ./...` and the stale "Not part of GitHub CI" comment is replaced; `.github/workflows/ci.yml` has a `race` job running `make test-race`. `make -n test-race` prints `go test -race -count=1 -p 2 -parallel 2  ./...`. `go test -race -count=1 -p 2 -parallel 2 ./css ./internal/pubstate` exit `0` (`ok css 1.161s`, `ok internal/pubstate 1.017s`, the two packages the old list skipped). `go test -race -count=1 -run 'TestConfig|TestTestScrollOffset|TestScrollRuntime|TestSimpleFlex|TestFlexStyleOverride|TestSharedText|Container|TestMulticol' ./internal/layout` exit `0`; `go test -race -count=1 -run 'FlexInvoice|PaintRestyle' ./layout` exit `0`. The full `make test-race` over every package is `BLK-P1B-GATE-01` (pending primary-session run).

- [x] `BLK-CI-02` Wire all 8 existing fuzz targets (plus new fuzzers for selector matching and SVG rasterization) into `.github/workflows/ci.yml`.
  - **Root cause:** `blinkless` has 8 fuzz targets (`FuzzParseHTML`, `FuzzUnescapeEntities`, `FuzzSplitTTC`, `FuzzTableDirectory`, `FuzzContentRange`, `FuzzExpandRepeat`, `FuzzEvalMath`, `FuzzParseCSS`), and **0** run in CI; selector matching and SVG rasterization have no fuzz target yet.
  - **Fix:** Add a `fuzz` CI job running 10–15s smoke passes of every fuzz target, and add `FuzzMatchSelector` (`internal/css`) and `FuzzRasterizeSVG` (`internal/svg`).
  - **Proof:** Local `go test -fuzz=... -fuzztime=10s` passes for every target + CI workflow runs green.
  - **Evidence (2426019):** added `FuzzMatchSelector` (`internal/css/fuzz_match_test.go`, parses a selector list with `css.ParseSelectors` and matches it against every element of a fixed tree, asserting `Match` is deterministic) and `FuzzRasterizeSVG` (`internal/svg/fuzz_raster_test.go`, asserts error-xor-PNG with positive size and a 10s wall bound). `scripts/fuzz-targets.txt` lists all 10 targets; `make fuzz-smoke FUZZ_TIME=10s` loops over it and the CI `fuzz` job runs `make fuzz-smoke FUZZ_TIME=15s`. Local `go test -run '^$' -fuzz '^<name>$' -fuzztime=10s <pkg>` exit `0` for every target: FuzzParseHTML, FuzzUnescapeEntities (`./internal/html`), FuzzSplitTTC, FuzzTableDirectory (`./internal/fonts`), FuzzContentRange (`./internal/load`), FuzzExpandRepeat (`./internal/layout`), FuzzEvalMath, FuzzParseCSS, FuzzMatchSelector (`./internal/css`), FuzzRasterizeSVG (`./internal/svg`). No new `testdata/fuzz` corpus files were produced (no failures found). The green CI run is `BLK-P1B-GATE-03`.

- [x] `BLK-CI-03` Wire `make wasm-test`, `make build` (`CGO_ENABLED=0`), `make claim-scan`, `make catalog-check`, `make matrix-check`, `go mod verify`, and `govulncheck` into `.github/workflows/ci.yml`.
  - **Root cause:** `.github/workflows/ci.yml` is currently a 39-line single-job workflow that skips Wasm contract verification, `govulncheck`, `go mod verify`, and catalog/matrix gates.
  - **Fix:** Expand `.github/workflows/ci.yml` with explicit jobs (pinning GitHub Actions to commit SHAs) covering tests, race, lint, wasm-test, catalog/matrix/claim gates, and vulnerability scanning.
  - **Proof:** Green CI run across all added jobs on a PR.
  - **Evidence (2426019):** `.github/workflows/ci.yml` rewritten into 7 jobs (`test` with `CGO_ENABLED=0 go build ./...`, `make test`, `make golden`; `race`; `lint`; `checks` = `make claim-scan catalog-check matrix-check`; `wasm` = `make wasm-test`; `fuzz`; `security` = `make mod-verify` + `make vuln`). Every action keeps its existing commit-SHA pin (`actions/checkout@11d5960a...`, `actions/setup-go@40f1582b...`); YAML parses (`python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"` exit `0`). `.github/workflows/release.yml` no longer installs golangci-lint v1 (`make lint` installs the pin). New Makefile targets `fuzz-smoke`, `vuln` (`govulncheck@v1.1.4`), `mod-verify`. Local runs: `make wasm-test` exit `0`; `make claim-scan catalog-check matrix-check` exit `0` (`claim-scan: clean`, `check ok (785 properties...)`, `matrix-sync: ok`); `go mod verify` -> `all modules verified`; `make build` exit `0`. `govulncheck ./...` first reported 23 findings, 3 of them in required modules; fixed by `go get golang.org/x/net@v0.60.0 golang.org/x/image@v0.43.0 github.com/andybalholm/brotli@v1.2.7` (indirect; `go.mod` `go 1.26` became `go 1.26.0` because the bumped modules require it; `x/text` moved to v0.42.0 transitively). It now reports 20 findings, all in the Go standard library of the local go1.26.4 toolchain (fixed by go1.26.4 -> go1.26.11); CI's `go-version: '1.26'` resolves to the latest patch. `go test ./internal/svg ./internal/fonts ./internal/load` exit `0` after the bump. Primary session: consider bumping the `toolchain` line in `go.mod` to the newest 1.26 patch so local `make vuln` is clean too, and mirror the three module bumps in `ownframe`/`examples/go.mod`. The green workflow run is `BLK-P1B-GATE-03`.

## 3.2 Broken Makefile Targets & Obsolete Docs

- [x] `BLK-MK-01` Remove or repair broken `./internal/convert` targets and stale `lint-frontend` no-op in `Makefile` (`Makefile:38-42, 71-72, 184-198`).
  - **Root cause:** `bench-engine` and `bench-cli-compare` invoke `go test ./internal/convert` (package does not exist); `test-unit` filters out `/internal/convert$`; `lint-frontend` prints `"frontend/ is not in this tree"` even though `frontend/` exists.
  - **Fix:** Point benchmark targets at existing `./layout` and `./internal/layout` display-list/relayout benchmarks, clean up `test-unit`, and align `lint-frontend` with the actual `frontend/` tree.
  - **Proof:** `make test-unit`, `make bench`, and `make wasm-test` all exit `0`.
  - **Evidence (2426019 + follow-up):** `Makefile`: `bench-cli-compare` and `bench-inprocess` removed; `bench-engine` (and `bench`, which now just depends on it) runs `go test ./layout ./internal/layout ./internal/style -run '^$' -bench 'Benchmark(RelayoutMedium|FlexInvoiceHeight|InvoiceScale|RelayoutStages|HoverCycle|NewDisplayForPatch|RelayoutPatchRepeat|RelayoutPatchVarying|StyleResolutionContext|NewRuleIndex|GridMemoNested)$' -benchmem -benchtime=$(BENCH_TIME) -count=$(BENCH_COUNT)`; every benchmark name verified to exist with `go test ... -list 'Benchmark.*'` (exit `0`). `test-unit` now filters `/temps/` (the scratch trees) instead of the nonexistent `/internal/convert$`. `lint-frontend` runs `cd frontend && npm exec -- tsc --noEmit`: `make lint-frontend` exit `0`. `make wasm-test` exit `0`. `grep -n "internal/convert" Makefile .github/workflows/*.yml` -> no matches. The benchmark run itself (`make bench`) and `make test-unit` (full suite) are left to the primary session: benchmarks must not run in a subagent and `test-unit` is a whole-suite run (`make -n test-unit` prints `go list ./... | grep -v '/temps/'`).

- [x] `BLK-MK-02` Make `make test-quick` actually honor `testing.Short()` or remove the misleading target (`Makefile:34-36`).
  - **Root cause:** `Makefile:34` documents `test-quick` as `"Skip long perf-budget tests (testing.Short)"`, yet there are `0` `testing.Short()` checks in the entire repository.
  - **Fix:** Add `if testing.Short() { t.Skip(...) }` to genuinely slow soak/perf-budget tests (or remove the no-op claim if none exist).
  - **Proof:** `make test-quick` measurably skips slow perf tests and exits faster than `make test`.
  - **Evidence (2426019):** `testing.Short()` skips added to the four slow or wall-clock-budget tests found by timing `go test -json` over the non-layout packages: `TestSlowServerTimeout` (`internal/load`, 3.0s), `TestLookupExactBeforeGeneric` (`internal/fonts`, 2.2s system-font scan), `TestOversizeShapeRunFallsBack` (`internal/fonts`, 5s fallback budget), `TestPatchIDLookupStaysCheap` (`html`, wall-clock budget). `go test -short -count=1 -v -run 'TestSlowServerTimeout|TestLookupExactBeforeGeneric|TestPatchIDLookupStaysCheap|TestOversizeShapeRunFallsBack' ./internal/load ./internal/fonts ./html` exit `0` and prints `--- SKIP` for all four (0.00s each). Makefile and CONTRIBUTING.md comments now describe the real behavior. Comparing wall time of `make test-quick` against `make test` is a whole-suite run: pending primary-session run.

- [x] `BLK-MK-03` Archive or replace obsolete pre-refactor PDF/PNG benchmark and architecture reports (`testdata/golden/benchmarks/benchmark-results.txt`, `reports/critical-golang-architecture-review.md`).
  - **Root cause:** Both artifacts still measure deleted `internal/convert` / `Document.WritePDF` / `ImageDocument.WriteImage` APIs from Aug–Sep 2026.
  - **Fix:** Either mark them explicitly archived or regenerate current display-list/relayout benchmark baselines.
  - **Proof:** `make claim-scan` clean and no active doc/Makefile references to deleted PDF writer benchmarks.
  - **Evidence (2426019):** archived rather than regenerated, because the benchmarked APIs no longer exist: `testdata/golden/benchmarks/benchmark-results.txt` and `reports/critical-golang-architecture-review.md` now open with an `ARCHIVED (2026-11-23)` banner; `testdata/golden/benchmarks/README.md` has the same banner, its reproduction table says "retired", and it no longer tells readers to run `make bench-cli-compare`/`bench-lib`. The live baseline is `make bench` (display-list and relayout benchmarks). `make claim-scan` exit `0` (`claim-scan: clean`); `grep -n "bench-cli-compare\|bench-inprocess\|internal/convert" Makefile .github` -> no matches. Left as is, because they are retired scripts rather than active targets: `scripts/bench-external.sh` and `scripts/bench-performance-recovery.sh` still mention `make bench-cli-compare` in their headers and the recovery script's `external` mode; the primary session may retire them with the other removed-pipeline scripts.

---

## Closure Gates

- [ ] `BLK-P1B-GATE-01` `make test-race` covers `./...` and exits `0`. (Target now covers `./...`; the full run is pending primary-session run.)
- [ ] `BLK-P1B-GATE-02` Every target listed in `Makefile` `.PHONY` runs cleanly without referencing non-existent `./internal/convert`. (`grep internal/convert Makefile` is empty; running `test-unit`, `bench`, `test-quick`, `final-evidence` end to end is pending primary-session run.)
- [ ] `BLK-P1B-GATE-03` Updated `.github/workflows/ci.yml` passes all jobs (test, race, lint, wasm-test, fuzz smoke, govulncheck, catalog/matrix/claim checks). (Workflow written and locally dry-run per job; a green run on a pushed PR is pending primary-session run.)
