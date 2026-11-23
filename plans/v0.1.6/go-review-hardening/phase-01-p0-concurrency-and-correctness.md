# v0.1.6 - Phase 01: P0 Concurrency & Engine Correctness

> **Parent:** [README.md](README.md)
> **Branch:** `chore/v0.1.6-go-review-hardening`
> **Priority:** P0 (must land before `ownframe` removes `internal/render/engineMu`)
> **Status:** Rows complete (8 / 8); closure gates pending primary-session run

---

## Overview

This phase fixes confirmed data races, concurrent map panics, CSS selector false negatives, Unicode byte-offset panics, CSS custom-property resolution bugs, sibling-cache staleness, and global slice mutation hazards on `blinkless` `master` (`b2fe5ba5`).

---

## 1.1 Concurrency & Shared Mutable State Fixes

- [x] `BLK-CONC-01` Fix data race on lazy `Node.EnsureAttrs()` during concurrent read-only passes (`internal/html/html.go:71-75, 111-128`).
  - **Root cause:** Small attribute lists stay map-free during parsing (`n.Attrs == nil`, stored in `n.AttrList`), and read-only passes (`css/hover_scopes.go:195`, `internal/layout/subtree_reuse.go:293, 324`, `layout_svg.go:308`, `layout/relayout_element.go:717`) lazily materialize `n.Attrs = m` with no synchronization, racing if two goroutines read the same `*html.Document`.
  - **Fix:** Either avoid mutating `n.Attrs` on read-only paths (scan small `n.AttrList` linearly — typically <= 8 attrs — or expose a read-only `AttrValue`/`ForEachAttr` API) OR guard materialization safely (e.g., `atomic.Pointer[map[string]string]` or eager materialization for large lists at parse time). Prefer read-only `AttrList` iteration to eliminate both allocation and mutation.
  - **Proof:** Concurrent `DisplayList` + `HoverChangedScopes` / `RelayoutElement` race test on a shared `*html.Document` passing `go test -race ./internal/html ./css ./layout ./internal/layout`.
  - **Evidence (2426019 + follow-up):** `go test -race -count=1 -run 'Concurrent' ./css` failed before the fix (DATA RACE: `Node.AttrValue` read vs `Node.EnsureAttrs` write at `html.go:125`, via `css/hover_scopes.go:195`) and exits `0` after. Read-only callers (`css/hover_scopes.go`, `internal/layout/subtree_reuse.go` x2, `internal/layout/layout_svg.go`, `layout/relayout_element.go`, `markup/markup.go`) now iterate directly via `Node.ForEachAttr`/`AttrCount`/`HasAttrs` (zero per-call map allocation for small-attr nodes kept in `AttrList`); `Node.AttrMap()` remains non-mutating and `EnsureAttrs` stays only on mutation paths. Tests: `css/relayout_race_test.go` (`TestHoverChangedScopesConcurrentSharedDocument`), `layout/shared_doc_race_test.go`, `internal/html/attr_map_test.go` (`TestAttrMapDoesNotMutate`, `TestForEachAttrZeroAllocAndNoMutation`). Exit `0`: `go test -race -count=1 ./internal/html ./css ./layout ./internal/layout ./markup`.

- [x] `BLK-CONC-02` Synchronize `prepare.SheetCache` shared across `css.Relayout` documents (`internal/convert/prepare/sheet_cache.go:13-40`, `internal/convert/prepare/styles.go:139-290`, `css/relayout.go:60-75`).
  - **Root cause:** `SheetCache` holds three unsynchronized Go maps (`inline`, `links`, `imports`) and `css.Relayout` returns a new `*css.Document` sharing the same `doc.cache` pointer. Concurrent `css.Relayout` calls trigger `fatal error: concurrent map read and map write`.
  - **Fix:** Guard `SheetCache` map reads/writes with an internal `sync.RWMutex` via accessor methods (`getInline`/`setInline`, `getLink`/`setLink`, `getImport`/`setImport`).
  - **Proof:** Concurrent `css.Relayout` race test (including `@media`-gated `<link>`/`@import` activation) passing `go test -race ./css ./internal/convert/prepare`.
  - **Evidence (2426019):** `go test -race -count=1 -run TestRelayoutConcurrentSharesSheetCache ./css` failed before the fix (DATA RACE in `SheetCache.importSheet`) and exits `0` after. `SheetCache` now guards all three maps with a `sync.RWMutex` (`internal/convert/prepare/sheet_cache.go`, `sheet_cache_notes.go`). Test: `css/relayout_race_test.go` (16 goroutines, `@media`-gated `@import`). Exit `0`: `go test -race -count=1 ./css ./internal/convert/prepare`.

- [x] `BLK-CONC-03` Stop returning a globally cached mutable slice from `transitionsFrom` (`layout/layout.go:400-448`).
  - **Root cause:** `transitionsFrom` caches `[]Transition` in global `transitionSliceCache` keyed by `&records[0]` pointer + length and returns the cached slice directly into exported `Box.Transitions`. Caller mutation corrupts the global cache for all documents, and strong `*ilayout.TransitionRecord` keys pin backing arrays globally (plus risk stale hits if slices are reused).
  - **Fix:** Return a cloned slice (`slices.Clone`) or cache per-engine/per-display instead of globally (or drop global caching if conversion cost is negligible on small transition lists).
  - **Proof:** Unit test mutating `box.Transitions[0]` and verifying subsequent `DisplayList` calls are unaffected + benchmark check.
  - **Evidence (2426019):** `go test -count=1 -run TestTransitionsNotSharedAcrossDisplays ./layout` failed before the fix (`transition slice was shared through the global cache`), exits `0` after. `transitionsFrom` (`layout/layout.go`) now builds a fresh slice per call; the global `transitionSliceCache` and its mutex are deleted. Test: `layout/transitions_isolation_test.go`. Benchmark to run by primary session: `go test -run '^$' -bench 'DisplayList' -benchmem ./layout` (conversion of 1-3 transitions per box is small; confirm no regression).

## 1.2 CSS Selector, Unicode & Custom-Property Correctness

- [x] `BLK-CSS-01` Add combinator backtracking for descendant (` `) and general sibling (`~`) in `leftmostMatch` and `matchPseudoWalk` (`internal/css/match.go:414-469, 511-547`).
  - **Root cause:** `leftmostMatch` and `matchPseudoWalk` walk selector parts right-to-left in a flat greedy loop without backtracking. For `.a > .b .c` against `<div class="a"><div class="b"><div class="x"><div class="b"><span class="c">`, the nearest `.b` inside `.x` fails `.a >`, and matching immediately fails without trying the outer `.b`. Same false negative for `.a + .b ~ .c`.
  - **Fix:** Convert the combinator walk into a recursive (or explicit-stack) matcher that backtracks across all candidate ancestors for ` ` and all candidate preceding siblings for `~` (preserving the existing `:has()` visit budget and zero-alloc fast path for single-part / child-only selectors).
  - **Proof:** Regression tests in `internal/css/` covering `.a > .b .c`, `.a > .b .c::before`, `.a + .b ~ .c`, and deep-ancestor non-match performance bounds.
  - **Evidence (2426019 + follow-up):** `leftmostMatch` and `matchPseudoWalk` now share the recursive `leftmostChain` (`internal/css/match.go`): descendant and `~` try every candidate nearest-first with a lazily allocated failure memo (`*chainFail` shared across the first candidate spine and all recursion depths; unreachable `matchLeftSibling`/`matchLeftAncestor` removed from `leftmostStep`); `>` and `+` keep the single-candidate path. Tests: `internal/css/combinator_backtrack_test.go` (`.a > .b .c`, `.a > .b .c::before`, `.a + .b ~ .c`, 200-deep non-match under 2 s). Exit `0`: `go test -race -count=1 ./internal/css`. The failing-first run was not captured with the old code (no `git stash` allowed); the `.a > .b .c` case is the plan's own counterexample for the old greedy loop. Benchmarks for primary: `go test -run '^$' -bench 'Match|Has' -benchmem ./internal/css`.

- [x] `BLK-CSS-02` Fix UTF-8 `strings.ToLower` byte-offset mismatch in `FontFaceURLs` and `expandOneRepeat` (`internal/css/css.go:616-652`, `internal/layout/grid_parse.go:130-159`).
  - **Root cause:** Both compute `strings.ToLower(src)`, find a byte index in the lowercased string, and slice original `src` using that index. Non-ASCII runes whose UTF-8 byte length changes under `strings.ToLower` (e.g., `İ` U+0130: 2 B → 3 B; `Ⱥ`/`Ⱦ`: 2 B → 3 B) shift offsets, corrupting extracted URLs/grid templates or panicking out-of-bounds.
  - **Fix:** Replace `strings.ToLower` indexing with an ASCII-only case-insensitive byte search helper (`indexASCIIFold`) that preserves exact byte offsets in `src`.
  - **Proof:** Unit + fuzz tests (`FuzzParseCSS`, `FuzzExpandRepeat`) with inputs like `local("İstanbul"), url(Font.woff2)` and `[İstanbul] repeat(2, 1fr)`.
  - **Evidence (2426019):** Added `css.IndexASCIIFold` (`internal/css/ascii_fold.go`); `FontFaceURLs` (`internal/css/css.go`) and `expandOneRepeat` (`internal/layout/grid_parse.go`) no longer slice the original with an index from `strings.ToLower`. Tests: `internal/css/ascii_fold_test.go` (`local("İstanbul"), URL(Font.woff2)`), `internal/layout/grid_repeat_fold_test.go` (`[İstanbul] REPEAT(2, 1fr)`), plus seeds added to `FuzzParseCSS` and `FuzzExpandRepeat`. Exit `0`: `go test -count=1 ./internal/css ./internal/layout`.

- [x] `BLK-CSS-03` Fix `ResolveCustomProps` embedded `var()` expansion and add a recursion depth cap (`internal/css/values.go:718-765`).
  - **Root cause:** `ResolveCustomProps` calls `ResolveVar` (`values.go:749`), which only expands values *starting* with `var(`, so embedded references like `--border: 1px solid var(--accent)` or `--w: calc(var(--u) * 2)` fail to expand. In addition, `eval` detects cycles via `stack[name]` but has no depth cap for long linear chains (`--v1: var(--v2); ...`).
  - **Fix:** Call `ResolveVars` inside `ResolveCustomProps` and enforce a bounded recursion depth (e.g., `maxCustomPropDepth = 64`), returning invalid/empty beyond the limit.
  - **Proof:** Unit tests in `internal/css/css_test.go` covering embedded `var()` in custom properties (`--border: 1px solid var(--accent)`, `--w: calc(var(--u) * 2)`) and a 1,000-step linear chain without stack overflow.
  - **Evidence (2426019 + follow-up):** `ResolveCustomProps` (`internal/css/values.go`) now expands through `ResolveVars` (including embedded `var()` inside `var(--missing, fallback)` via `ResolveVar`), omits undeclared custom-property names from the returned `memo` map, caps chains at `maxCustomPropDepth = 64`, and keeps results independent of map iteration order (depth-capped values are not memoized; total work bounded by `maxCustomPropEvals`). Cycles still resolve through the existing fallback rule (`TestResolveCustomPropsSelfReferenceWithFallback` unchanged). Tests: `internal/css/custom_props_resolve_test.go` (`--border: 1px solid var(--accent)`, `--w: calc(var(--u) * 2)`, 1,000-step chain), `internal/css/css_test.go` (`TestResolveCustomPropsDoesNotPolluteWithAbsentVar`, `TestResolveCustomPropsEmbeddedVarInsideFallback`). Exit `0`: `go test -race -count=1 ./internal/css`.

- [x] `BLK-CSS-04` Preserve CSS Custom Properties Level 1 case sensitivity across declaration parsing and `parseVarFn` (`internal/css/values.go:584-595`, CSS declaration parser).
  - **Root cause:** `parseVarFn` lowercases custom property names (`strings.ToLower`), collapsing `--colorPrimary` and `--colorprimary` contrary to CSS Custom Properties Level 1.
  - **Coordination note:** Update declaration parsing and `parseVarFn` together so declared custom properties and `var(--...)` lookups both preserve exact case without breaking existing camelCase custom property lookups.
  - **Proof:** Unit + layout tests verifying `--colorPrimary` and `--colorprimary` resolve independently while standard CSS properties remain case-insensitive.
  - **Evidence (2426019 + follow-up):** `parseVarFn` no longer lowercases names and `parseDeclarations` + `parseSupportsDeclaration` (`internal/css/at_supports.go`) keep `--*` names verbatim (`validPropName` accepts uppercase only for custom properties); all other property names stay lowercased. Tests: `internal/css/custom_props_resolve_test.go` (`TestCustomPropertyNamesCaseSensitive`, `TestSupportsPreservesCustomPropertyCase`), `internal/layout/cssvar_case_test.go` (cascade: `--sizeBig` and `--sizebig` resolve to 20px and 10px). Exit `0`: `go test -count=1 ./internal/css ./internal/layout ./css ./layout`.

- [x] `BLK-CSS-05` Fix stale `siblingEntryFor` invalidation on same-length child replacement (`internal/css/match.go:175-205`, `html/attribute.go:129-148`).
  - **Root cause:** `siblingEntryFor` caches `sibParentData` keyed by `weak.Make(parent)` and validates solely with `entry.childLen == len(parent.Children)`. Replacing one element child with one `TextNode` via `SetText` keeps `len == 1`, returning stale `elems` containing the detached element.
  - **Fix:** Track a per-node or per-document mutation counter (e.g., `parent.ChildVersion` incremented on child splice/replace) and compare `entry.version == parent.ChildVersion` instead of `childLen` alone.
  - **Proof:** Regression test calling `html.Document.SetText` on an element with one child element and verifying `:first-child`/`:nth-child`/`:only-of-type` reflect the updated tree.
  - **Evidence (2426019):** Added `Node.ChildVersion` (`internal/html/html.go`), bumped by `html.Document.SetText` on every child replace or drop (`html/attribute.go`); `siblingEntryFor` and `sibParentData` compare it with `childLen` (`internal/css/match.go`). Tests: `internal/css/sibling_version_test.go` (same-length element-to-text swap rebuilds the entry), `html/find_index_test.go` `TestSetTextBumpsChildVersion`. Exit `0`: `go test -race -count=1 ./html ./internal/css`. Note: SetText cannot make an element observably stale through public matching, so the regression asserts the memo entry itself.

---

## Closure Gates

- [x] `BLK-P0-GATE-01` `go test -race ./internal/html ./css ./internal/css ./internal/convert/prepare ./layout ./internal/layout` exits `0`. Partial result from the implementing agent: `go test -race -count=1 ./internal/html ./css ./internal/css ./internal/convert/prepare ./layout ./internal/layout ./html ./markup ./internal/fonts ./internal/svg` exited `0` for every package. Primary session (commit 2426019): `go test -race -count=1 ./css ./layout ./html ./markup ./internal/...` exit `0`, no failing packages; `go vet ./...` and `GOOS=js GOARCH=wasm go vet ./bindings/wasm` exit `0`.
- [ ] `BLK-P0-GATE-02` `make test`, `make golden`, `make lint`, `make claim-scan`, `make catalog-check`, and `make matrix-check` exit `0`. Pending primary-session run.
- [ ] `BLK-P0-GATE-03` Pending primary-session run (commit hash `2426019`). Record passing commit hash to unblock `ownframe` Phase 04 `HYG-ENG-01` (`engineMu` removal).
