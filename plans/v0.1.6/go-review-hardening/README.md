# v0.1.6 - Go Review Hardening Plan

> **Parent:** [plans/v0.1.6/README.md](../README.md) (and [plans index](../../README.md))
> **Branch:** `chore/v0.1.6-go-review-hardening`
> **Baseline:** `master` at `b2fe5ba5` (tag `v0.1.5` + PR #47, 2026-11-23)
> **Status:** Open (code complete at `2426019`; `BLK-ARCH-01` parked with measurements; remaining gates need `make test`, `make lint`, CI, and benchmarks)

---

## Overview

This directory holds the phase-wise execution checklist produced from the cross-checked 10-part Go code review of `blinkless` (`b2fe5ba5`). Every task row corresponds to a source-verified concurrency hazard, CSS/layout correctness bug, global lock-contention or memory-retention defect, CI/Makefile wiring gap, or lint/architecture hygiene issue.

## Phase Summary

| Order | Phase | Scope | Tasks | Closure Gates | File |
| :---: | :--- | :--- | :---: | :---: | :--- |
| 1 | **P0: Concurrency & Engine Correctness** | `Node.EnsureAttrs()` race, `prepare.SheetCache` map panic, CSS combinator backtracking, UTF-8 `strings.ToLower` slicing, `ResolveCustomProps` (`ResolveVars` + depth cap + case-sensitivity), `siblingEntryFor` invalidation, `transitionsFrom` mutable global slice | **8** | **3** | [phase-01](phase-01-p0-concurrency-and-correctness.md) |
| 2 | **P1: Contention, Cache Bounds & API Edge Cases** | Hit-path `Lock()` → `atomic.Uint64`, `testScrollOffsetOf` production guard, 5 freezing caches & `sharedScaledFaces` `*Font` retention, `gradCache` byte cap & `NaN` guard, nested `@media`, `svg/raster.go` `ErrNilContext` & semaphore, `html.Parse` default caps, `opts.Extra` clone, `css.Relayout` hover fast path | **10** | **3** | [phase-02](phase-02-p1-contention-caches-and-edge-cases.md) |
| 3 | **P1: CI Verification, Race, Fuzz & Makefile Repair** | Expand `RACE_PKGS` to `./...` (adding `./css` & `./internal/pubstate`), wire 8 fuzz targets + `wasm-test` + `govulncheck` into CI, remove broken `./internal/convert` Makefile targets & obsolete PDF benchmark reports, fix `make test-quick` | **6** | **3** | [phase-03](phase-03-p1-ci-race-fuzz-and-makefile.md) |
| 4 | **P2: Lint Hygiene, Dead Code & `internal/layout` Split** | Remove 8 production file-top `//nolint:all` blankets (incl. `multicol.go` `SA6002`), strip ~1,190 orphaned `//nolint`s, upgrade `golangci-lint` v1.64.8 → v2 + `nolintlint`, delete dead `internal/convert/render` & `internal/profiling`, trim `wkhtmltopdf` residue, split `internal/style` out of `internal/layout` | **7** | **3** | [phase-04](phase-04-p2-lint-hygiene-and-architecture.md) |
| | **Total** | | **31** | **12** | |

---

## Execution & Evidence Rules

- Use consistent row states across all phase files:
  - `[ ]` open / not yet implemented or not proven;
  - `[x]` implemented and proven with commit hash, exact command, exit status, and observed result beside the row;
  - `[~]` intentionally deferred or parked with reason, owner boundary, and next gate.
- Write a failing regression test (or `-race` reproducer) before fixing each bug in Phases 01–02.
- Preserve `blinkless`'s pure-Go zero-CGO runtime (`CGO_ENABLED=0 go build ./...`) and 2-direct-dependency allowlist (`TestDirectModuleAllowlist`).
- Every implementation phase requires `make test`, `make golden`, `make lint`, `make claim-scan`, `make catalog-check`, `make matrix-check`, and `go test -race ./...` (once Phase 01 fixes the concurrency bugs).

## Cross-Repo Coordination with `ownframe`

- **Paired OwnFrame plan:** `ownframe` branch `chore/v0.0.4-go-review-hardening` (`plans/v0.0.4/go-review-hardening/`).
- **Unblocks `ownframe` `HYG-ENG-01`:** Closing `BLK-CONC-01` (`Node.EnsureAttrs()`) and `BLK-CONC-02` (`prepare.SheetCache`) in Phase 01 and publishing a `blinkless` commit/tag is the mandatory prerequisite before `ownframe` may remove `internal/render/engineMu`.
