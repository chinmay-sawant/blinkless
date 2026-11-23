# v0.1.6 - Phase 02: P1 Contention, Cache Bounds & API Edge Cases

> **Parent:** [README.md](README.md)
> **Branch:** `chore/v0.1.6-go-review-hardening`
> **Priority:** P1
> **Status:** Open (0 / 10 complete)

---

## Overview

This phase removes global write-lock contention on hot cache hits, replaces unbounded or permanent-freeze global caches (including `*pdf.Font` SFNT retention), fixes nested `@media`, `NaN` gradient poisoning, `internal/svg` semaphore/error bugs, and closes public API edge cases (`html.Parse`, `css.Apply`, `css.Relayout`, doc comments).

---

## 2.1 Lock Contention & Production Hot-Path Fixes

- [ ] `BLK-PERF-01` Eliminate exclusive `Lock()` on cache hits in `lookupAcceptVerdict` and `supportsMemos` (`internal/layout/style_value_accept.go:109-121`, `internal/layout/style_cascade.go:1710-1719`).
  - **Root cause:** Both acquire `RLock()`, unlock, then acquire **exclusive `Lock()`** on every cache hit solely to increment a test hit counter (`acceptVerdictHits++`, `supportsProbeHits++`); `acceptVerdictKey` also allocates `prop + "\x00" + value` per lookup.
  - **Fix:** Replace hit counters with `atomic.Uint64` (matching `text_measure_cache.go`) and use a `struct{ prop, value string }` map key for zero-alloc lookup.
  - **Proof:** Parallel layout benchmark showing zero write-lock contention and reduced allocations under `go test -bench`.

- [ ] `BLK-PERF-02` Gate `testScrollOffsetOf` behind a fast atomic check so production overflow boxes skip global `RLock()` (`internal/layout/scroll.go:191`, `internal/layout/scroll_runtime.go:79-88`).
  - **Root cause:** Every production overflow box acquires global `testScrollOffsetsMu.RLock()` and queries a map that is always empty outside tests.
  - **Fix:** Maintain an `atomic.Int32` active-entry counter (fast-returning zero when `== 0`), or move test scroll offsets onto `layout.Options`.
  - **Proof:** Unit test preserving existing test-scroll behavior + benchmark confirming zero mutex traffic when no test offsets are installed.

- [ ] `BLK-PERF-03` Short-circuit `css.Relayout` DOM walk and `Loader` creation on state-only (`hover`/`focus`/`active`) changes (`css/relayout.go:60-75`, `css/css.go:157-203`, `css/state_pseudos.go:13-29`).
  - **Root cause:** `css.Relayout` instantiates `settings.DefaultPdfGlobal()`, a fresh `load.Loader`, and runs `CollectTreeSheets` over the entire DOM even when width, height, and media type are unchanged; `UsesStatePseudo` also rescans every rule in `d.sheets` on every call without caching.
  - **Fix:** When `width`, `height`, and `media` match `doc`'s previous values (and DOM mutation version is unchanged), reuse `doc.sheets` and `doc.registry` directly; memoize `UsesStatePseudo` on `*css.Document`.
  - **Proof:** Regression tests verifying identical sheets on hover/focus/active relayout + benchmark measuring hover relayout speedup.

- [ ] `BLK-PERF-04` Avoid exclusive lock + full DOM walk on repeated negative ID lookups in `html.Document.Find` and avoid holding `sharedCompiledMu` across `newRuleIndex` (`html/attribute.go:408-433`, `internal/layout/style_match_cache.go:86-126`).
  - **Root cause:** `Find` takes `idx.mu.Lock()` and walks the entire DOM on every miss for a non-existent ID; `sharedCompiledMu.Lock()` is held across `newRuleIndex(sheets)` and `sheetsCacheKey` uses `fmt.Sprintf` on every layout.
  - **Fix:** Populate the full ID index once on first `Find` (invalidated by DOM mutation version) so misses are O(1) under `RLock`; build `newRuleIndex` outside `sharedCompiledMu.Lock()` and use a non-allocating key hash/struct.
  - **Proof:** Benchmark + `-race` tests for `html.Document.Find` misses and concurrent layout.

## 2.2 Global Cache Eviction, Memory Retention & `NaN` Safety

- [ ] `BLK-CACHE-01` Add eviction and byte/pointer-lifetime bounds to the 5 permanent-freeze global caches (`internal/layout/style_cascade.go:415, 1725`, `style_value_accept.go:136`, `style_match_cache.go:62, 265, 400`, `internal/fonts/advance_cache.go:213`, `system_fallback.go:318`).
  - **Root cause:** `inlineDecls` (cap 1024, no byte cap), `acceptVerdicts` (4096), `supportsMemos` (4096), `singleAttrCache` (1024), and `sharedScaledFaces` (256) stop admitting new entries forever once full; `sharedScaledFaces` also strongly pins `*pdf.Font` pointers (with full SFNT bytes) in global memory; `sharedCompiled.tags` and `systemFallback.results` are unbounded maps.
  - **Fix:** Use bounded eviction (clear-on-full or ring/LRU) plus byte budgets; key `sharedScaledFaces` with `weak.Pointer[Font]` (or per-font cache) so unloaded fonts can be GC'd; bound `sharedCompiled.tags` and `systemFallback.results` (and avoid holding `systemFallback` mutex across filesystem I/O).
  - **Proof:** Stress test inserting >2× capacity entries and verifying steady hit rates and bounded memory / font GC reclamation.

- [ ] `BLK-CACHE-02` Add `math.IsNaN` / `math.IsInf` guards and byte cap to gradient rasterization (`internal/layout/gradient.go:550-634`).
  - **Root cause:** `width <= 0 || height <= 0` is `false` for `NaN`, and `gradCacheKey` stores raw `float64` RGB channels (`NaN != NaN`), inserting unreachable duplicate entries; `gradCache` (1024 × up to 512×512 PNGs) and `gradBufPool` lack byte caps.
  - **Fix:** Reject non-finite `width`, `height`, and RGB channels (`math.IsNaN`/`math.IsInf`), enforce a total byte cap on `gradCache`, and drop oversized buffers in `gradBufPool.Put`.
  - **Proof:** Unit tests with `NaN`/`Inf` dimensions and color channels confirming clean rejection and bounded cache bytes.

## 2.3 Engine & Public API Edge-Case Fixes

- [ ] `BLK-CSS-06` Support nested `@media` inside `parseNestedAtRule` (`internal/css/css.go:729-754`).
  - **Root cause:** `parseNestedAtRule` handles `@position-try`, `@container`, `@supports`, and `@layer`, but falls through to `skipAtRule` for nested `@media` inside `@layer`, `@supports`, or `@media`.
  - **Fix:** Add a `hasFoldPrefix(block, "@media")` branch combining the outer and inner media queries (e.g., `outer + " and " + inner` when outer is non-empty).
  - **Proof:** Unit + layout tests covering `@layer base { @media (min-width: 600px) { ... } }` and `@media screen { @media (max-width: 800px) { ... } }`.

- [ ] `BLK-SVG-01` Fix `ErrNilContext` sentinel and replace 1 ms polling ticker in `internal/svg/raster.go` (`internal/svg/raster.go:86-88, 124-142`).
  - **Root cause:** `ctx == nil` returns `errNotSVG` instead of `ErrNilContext`; `lockCanvas` polls `canvasMu.TryLock()` via `time.NewTicker(time.Millisecond)` (allocating a ticker and adding up to 1 ms wake jitter).
  - **Fix:** Return a proper nil-context error on `ctx == nil`, and replace `canvasMu` + `time.NewTicker` with a 1-buffered channel semaphore (`select { case canvasSem <- struct{}{}: ... case <-ctx.Done(): return ctx.Err() }`).
  - **Proof:** Unit tests for nil context, canceled context during contention, and zero-ticker wakeup latency.

- [ ] `BLK-API-01` Enforce a default input cap on public `html.Parse`, clone `opts.Extra` in `css.Apply`, fix `layout.go:82-83` doc comment, and fix `relayout_subtree.go` ABA tag (`html/parse.go:41`, `html/attribute.go:448-456`, `css/css.go:141`, `layout/layout.go:82-83`, `internal/layout/relayout_subtree.go:552-571`).
  - **Root cause:** `html.Parse` calls `ParseDocument` without a default byte/node cap; `css.Apply` stores `opts.Extra` without cloning; `layout/layout.go:82-83` tells callers to multiply points by `PointsPerPixel` instead of `PixelPerPoint`; `relayout_subtree.go:552-571` hashes `reflect.ValueOf.Pointer()` on `root` and only `Sheets[0]`; `SetAttribute` unconditionally lowercases new attributes (blocking dynamic SVG `viewBox`).
  - **Fix:** Route `html.Parse` through `ParseDocumentWithLimit` with a generous documented default cap (or expose `ParseWithLimit`), `slices.Clone(opts.Extra)`, fix the doc comment to `Display.PixelPerPoint`, hash all `opts.Sheets` with safe object identity (`weak.Pointer` / version), and preserve foreign-element attribute case in `SetAttribute`.
  - **Proof:** Unit tests covering each behavior (`html`, `css`, `layout`, `internal/layout`).

- [ ] `BLK-API-02` Audit `Display.Release()` pooled `Boxes` contract and `Box`/`DisplayOp` coordinate unit docs (`layout/displaylist.go:56-65, 242-246`).
  - **Root cause:** `Display.Release()` clears and returns exported `Display.Boxes` to `boxSlicePool` (silent zeroing if any caller retains a sub-slice) and `DisplayOp` doc warns against writing plain fields while `ownframe` patches them directly.
  - **Fix:** Clearly document `Display.Release()` ownership semantics (and check whether an explicit `DisplayOp` patch/setter helper should be exported for `ownframe` so `ownframe` stops violating the read-only field comment).
  - **Proof:** Updated doc comments + contract test in `layout/`.

---

## Closure Gates

- [ ] `BLK-P1A-GATE-01` `make test`, `make golden`, `go test -race ./...`, and `make lint` exit `0`.
- [ ] `BLK-P1A-GATE-02` `make claim-scan`, `make catalog-check`, and `make matrix-check` exit `0`.
- [ ] `BLK-P1A-GATE-03` Record before/after benchmark and allocation results for `BLK-PERF-01`–`BLK-PERF-04`.
