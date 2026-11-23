# v0.1.6 - Phase 04: P2 Lint Hygiene, Dead Code & `internal/layout` Split

> **Parent:** [README.md](README.md)
> **Branch:** `chore/v0.1.6-go-review-hardening`
> **Priority:** P2
> **Status:** Open (0 / 7 complete)

---

## Overview

`blinkless` currently has severe lint and package-cohesion debt: **8 production files (~9,700+ LOC)** carry file-top `//nolint:all` directives that silence `errcheck`, `gosec`, `govet`, and `staticcheck`; **~1,190 of 1,315 `//nolint` directives (~90%)** reference linters not enabled in `.golangci.yml`; `golangci-lint` is still on `v1.64.8` with no `nolintlint` enforcement; dead `wkhtmltopdf` packages remain; and `internal/layout` has grown to **246 source files / 103,207 LOC (71.6% of the codebase)**.

---

## 4.1 Remove `//nolint:all` Blankets & Upgrade Linter Discipline

- [ ] `BLK-LINT-01` Remove file-top `//nolint:all` from all 8 production files and fix every surfaced real diagnostic (`internal/layout/multicol.go:1`, `helpers.go:1`, `layout_chrome.go:1`, `layout_svg.go:1`, `layout_measure.go:1`, `style_values.go:1`, `internal/html/tree.go:3`, `internal/html/fragment.go:15`).
  - **Known real bug already hidden:** `internal/layout/multicol.go:20-21, 48, 68` passes raw slices to `sync.Pool.Put` (`staticcheck` `SA6002` slice-header boxing allocation).
  - **Fix:** Remove `//nolint:all` one file at a time (starting with `multicol.go` and `helpers.go`), fix real `errcheck`/`gosec`/`govet`/`staticcheck` findings, and only keep narrowly scoped, reasoned per-line directives where genuinely unavoidable.
  - **Proof:** `git grep '//nolint:all' -- '*.go' ':!*_test.go'` returns **0 matches** and `make lint` exits `0`.

- [ ] `BLK-LINT-02` Strip ~1,190 orphaned `//nolint` directives targeting disabled linters (`exhaustruct`, `gochecknoglobals`, `cyclop`, `paralleltest`, `wsl`, `mnd`, `funlen`, `varnamelen`, `nlreturn`, `lll`) and remove test-only file-top `//nolint:all` blankets.
  - **Root cause:** `1,315` `//nolint` directives pollute the codebase because `.golangci.yml` leaves `nolintlint` unconfigured (`allow-unused: true`, `require-specific: false`).
  - **Fix:** Clean up unused directives across source and test files (including the 8 test-file `//nolint:all` blankets).
  - **Proof:** `//nolint` count drops from `1,315` to only active, specific, explained suppressions.

- [ ] `BLK-LINT-03` Upgrade `golangci-lint` from `v1.64.8` to `v2.15.0` (matching `ownframe`), enable strict `nolintlint` (`allow-unused: false`, `require-specific: true`, `require-explanation: true`), and remove the `bindings/wasm` lint exclusion (`.golangci.yml:49`, `Makefile:13`).
  - **Fix:** Migrate `.golangci.yml` to v2 schema, enforce `nolintlint`, and lint `bindings/wasm` under `GOOS=js GOARCH=wasm`.
  - **Proof:** `make lint` exits `0` on `golangci-lint v2.15.0` with zero unreasoned or unused `//nolint` directives.

## 4.2 Dead Code, Process Globals & `internal/layout` Extraction

- [ ] `BLK-DEAD-01` Delete dead packages `internal/convert/render` and `internal/profiling` and trim legacy `wkhtmltopdf` residue in `internal/settings` and `internal/load`.
  - **Root cause:** `internal/convert/render` (58 LOC) and `internal/profiling` (7 LOC) have zero callers; `internal/settings` (2,115 LOC, incl. 1,044-LOC `reflect.go` of wkhtmltopdf dotted setting dispatchers) and `internal/load` (1,982 LOC of cookie/POST/MultiPageLoader) remain wired into `css.Apply`/`css.Relayout` even though `css.Apply` only uses `MediaType`.
  - **Fix:** Delete the two dead packages first, then decouple `css.Apply`/`css.Relayout` from `settings.DefaultPdfGlobal()` and unused `load` options where no active caller depends on them.
  - **Proof:** `make build`, `make test`, `make golden`, `make lint`, and `ownframe` integration build all exit `0`.

- [ ] `BLK-TEST-01` Move process-wide test bypass switches (`flexRowMemoBypass`, `simpleFlexHeightBypass`, `multicolMemoBypass`, `paintRestyleDisabled`, `maxSharedTextWidthEntries`, `testScrollOffsets`) onto `layout.Options` or per-engine test config.
  - **Root cause:** Global mutable test flags force serial test execution (`//nolint:paralleltest` × 96) and risk cross-test interference under `-race`.
  - **Fix:** Follow the existing pattern in `layout.go:136-146` (`gridMemoBypass`, `flexMinBypass`, `blockMemoBypass`) so tests can run safely with `t.Parallel()`.
  - **Proof:** `go test -race ./...` passes with parallelized memo-bypass tests.

- [ ] `BLK-ARCH-01` Plan and execute extraction of `internal/style` (CSS cascade, rule indexing, specificity/inheritance, `@supports` probing, UA rules, style interning — currently ~60 files / ~24,400 LOC) out of the `103,207`-LOC `internal/layout` god package.
  - **Root cause:** `internal/layout` holds 246 source files / 103k non-test LOC + 69k test LOC in a single package, coupling style computation tightly to layout geometry, SVG serialization, and display-list generation and slowing compile/test cycles.
  - **Fix:** Define explicit input/output boundary types (`*html.Node`, `[]*css.Stylesheet`, `css.MatchState` → `*style.ResolvedStyle` + rule index), move `style*.go` into `internal/style`, and keep `internal/layout` focused on box tree construction, formatting contexts, and display emission.
  - **Proof:** `make test`, `make golden`, `make lint`, `make catalog-check`, and `ownframe` `make test` all pass; `internal/layout` non-test LOC drops by ~24k.

- [ ] `BLK-DEP-01` Evaluate replacing or isolating `github.com/tdewolff/canvas` for SVG-as-image rasterization (`go.mod:9`, `internal/svg/raster.go`).
  - **Root cause:** `tdewolff/canvas` is used solely for SVG-as-image rasterization, yet drags in 23 indirect modules (including 8 unmaintained 2016–2022 repos like `freetype-go`, `xgb`, `poly2tri-go`, `rasterx`, and untrusted parsers like `fpdf`, `goldmark`, `star-tex`) and forces a process-wide mutex due to global mutable state in `bentleyOttmann`.
  - **Fix:** Prototype or evaluate a minimal pure-Go SVG rasterizer (or slim fork/subset) and record a go/no-go decision with Chrome/golden parity and binary-size measurements (do not swap unless visual parity holds).
  - **Proof:** Recorded benchmark + golden parity decision in this file (or `[~]` with rationale if retained).

---

## Closure Gates

- [ ] `BLK-P2-GATE-01` Zero production `//nolint:all` files remain; `make lint` runs `golangci-lint v2.15.0` with strict `nolintlint` and exits `0`.
- [ ] `BLK-P2-GATE-02` `make test`, `make test-race`, `make golden`, `make wasm-test`, `make claim-scan`, `make catalog-check`, and `make matrix-check` all exit `0`.
- [ ] `BLK-P2-GATE-03` Downstream `ownframe` `make test` and `make graphics-check` pass against the updated `blinkless` commit.
