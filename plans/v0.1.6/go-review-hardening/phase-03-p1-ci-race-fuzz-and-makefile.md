# v0.1.6 - Phase 03: P1 CI Verification, Race, Fuzz & Makefile Repair

> **Parent:** [README.md](README.md)
> **Branch:** `chore/v0.1.6-go-review-hardening`
> **Priority:** P1
> **Status:** Open (0 / 6 complete)

---

## Overview

`blinkless` already has strong unit tests (2,277 `Test*`) and 8 fuzz targets, but its `Makefile` and `.github/workflows/ci.yml` leave major verification gaps: `test-race` covers only 3 of 18 packages, none of the 8 fuzzers or `wasm-test` run in CI, `govulncheck` is absent, and several Makefile targets and benchmark docs still reference the deleted `./internal/convert` PDF pipeline.

---

## 3.1 Race, Fuzz, Wasm & Security CI Wiring

- [ ] `BLK-CI-01` Expand `RACE_PKGS` to `./...` (at minimum adding `./css` and `./internal/pubstate`) and update stale comments (`Makefile:28-29, 48-50`, `.github/workflows/ci.yml:27`).
  - **Root cause:** `RACE_PKGS ?= ./internal/layout ./internal/load ./internal/fonts` omits `./css` (which owns `TestHoverCacheConcurrentRace`) and `./internal/pubstate` (whose `state_race_test.go:22` instructs `"Run it with -race"`), and `Makefile:28` wrongly claims `test-race` is `"Not part of GitHub CI"`.
  - **Fix:** Set `RACE_PKGS ?= ./...` (keeping `-p $(TEST_P) -parallel $(TEST_PARALLEL)` for memory control) and update the stale comment.
  - **Proof:** `make test-race` exits `0` across all 18 packages locally and in GitHub Actions CI.

- [ ] `BLK-CI-02` Wire all 8 existing fuzz targets (plus new fuzzers for selector matching and SVG rasterization) into `.github/workflows/ci.yml`.
  - **Root cause:** `blinkless` has 8 fuzz targets (`FuzzParseHTML`, `FuzzUnescapeEntities`, `FuzzSplitTTC`, `FuzzTableDirectory`, `FuzzContentRange`, `FuzzExpandRepeat`, `FuzzEvalMath`, `FuzzParseCSS`), and **0** run in CI; selector matching and SVG rasterization have no fuzz target yet.
  - **Fix:** Add a `fuzz` CI job running 10–15s smoke passes of every fuzz target, and add `FuzzMatchSelector` (`internal/css`) and `FuzzRasterizeSVG` (`internal/svg`).
  - **Proof:** Local `go test -fuzz=... -fuzztime=10s` passes for every target + CI workflow runs green.

- [ ] `BLK-CI-03` Wire `make wasm-test`, `make build` (`CGO_ENABLED=0`), `make claim-scan`, `make catalog-check`, `make matrix-check`, `go mod verify`, and `govulncheck` into `.github/workflows/ci.yml`.
  - **Root cause:** `.github/workflows/ci.yml` is currently a 39-line single-job workflow that skips Wasm contract verification, `govulncheck`, `go mod verify`, and catalog/matrix gates.
  - **Fix:** Expand `.github/workflows/ci.yml` with explicit jobs (pinning GitHub Actions to commit SHAs) covering tests, race, lint, wasm-test, catalog/matrix/claim gates, and vulnerability scanning.
  - **Proof:** Green CI run across all added jobs on a PR.

## 3.2 Broken Makefile Targets & Obsolete Docs

- [ ] `BLK-MK-01` Remove or repair broken `./internal/convert` targets and stale `lint-frontend` no-op in `Makefile` (`Makefile:38-42, 71-72, 184-198`).
  - **Root cause:** `bench-engine` and `bench-cli-compare` invoke `go test ./internal/convert` (package does not exist); `test-unit` filters out `/internal/convert$`; `lint-frontend` prints `"frontend/ is not in this tree"` even though `frontend/` exists.
  - **Fix:** Point benchmark targets at existing `./layout` and `./internal/layout` display-list/relayout benchmarks, clean up `test-unit`, and align `lint-frontend` with the actual `frontend/` tree.
  - **Proof:** `make test-unit`, `make bench`, and `make wasm-test` all exit `0`.

- [ ] `BLK-MK-02` Make `make test-quick` actually honor `testing.Short()` or remove the misleading target (`Makefile:34-36`).
  - **Root cause:** `Makefile:34` documents `test-quick` as `"Skip long perf-budget tests (testing.Short)"`, yet there are `0` `testing.Short()` checks in the entire repository.
  - **Fix:** Add `if testing.Short() { t.Skip(...) }` to genuinely slow soak/perf-budget tests (or remove the no-op claim if none exist).
  - **Proof:** `make test-quick` measurably skips slow perf tests and exits faster than `make test`.

- [ ] `BLK-MK-03` Archive or replace obsolete pre-refactor PDF/PNG benchmark and architecture reports (`testdata/golden/benchmarks/benchmark-results.txt`, `reports/critical-golang-architecture-review.md`).
  - **Root cause:** Both artifacts still measure deleted `internal/convert` / `Document.WritePDF` / `ImageDocument.WriteImage` APIs from Aug–Sep 2026.
  - **Fix:** Either mark them explicitly archived or regenerate current display-list/relayout benchmark baselines.
  - **Proof:** `make claim-scan` clean and no active doc/Makefile references to deleted PDF writer benchmarks.

---

## Closure Gates

- [ ] `BLK-P1B-GATE-01` `make test-race` covers `./...` and exits `0`.
- [ ] `BLK-P1B-GATE-02` Every target listed in `Makefile` `.PHONY` runs cleanly without referencing non-existent `./internal/convert`.
- [ ] `BLK-P1B-GATE-03` Updated `.github/workflows/ci.yml` passes all jobs (test, race, lint, wasm-test, fuzz smoke, govulncheck, catalog/matrix/claim checks).
