# v0.1.6 - Phase 01: P0 Concurrency & Engine Correctness

> **Parent:** [README.md](README.md)
> **Branch:** `chore/v0.1.6-go-review-hardening`
> **Priority:** P0 (must land before `ownframe` removes `internal/render/engineMu`)
> **Status:** Open (0 / 8 complete)

---

## Overview

This phase fixes confirmed data races, concurrent map panics, CSS selector false negatives, Unicode byte-offset panics, CSS custom-property resolution bugs, sibling-cache staleness, and global slice mutation hazards on `blinkless` `master` (`b2fe5ba5`).

---

## 1.1 Concurrency & Shared Mutable State Fixes

- [ ] `BLK-CONC-01` Fix data race on lazy `Node.EnsureAttrs()` during concurrent read-only passes (`internal/html/html.go:71-75, 111-128`).
  - **Root cause:** Small attribute lists stay map-free during parsing (`n.Attrs == nil`, stored in `n.AttrList`), and read-only passes (`css/hover_scopes.go:195`, `internal/layout/subtree_reuse.go:293, 324`, `layout_svg.go:308`, `layout/relayout_element.go:717`) lazily materialize `n.Attrs = m` with no synchronization, racing if two goroutines read the same `*html.Document`.
  - **Fix:** Either avoid mutating `n.Attrs` on read-only paths (scan small `n.AttrList` linearly — typically <= 8 attrs — or expose a read-only `AttrValue`/`ForEachAttr` API) OR guard materialization safely (e.g., `atomic.Pointer[map[string]string]` or eager materialization for large lists at parse time). Prefer read-only `AttrList` iteration to eliminate both allocation and mutation.
  - **Proof:** Concurrent `DisplayList` + `HoverChangedScopes` / `RelayoutElement` race test on a shared `*html.Document` passing `go test -race ./internal/html ./css ./layout ./internal/layout`.

- [ ] `BLK-CONC-02` Synchronize `prepare.SheetCache` shared across `css.Relayout` documents (`internal/convert/prepare/sheet_cache.go:13-40`, `internal/convert/prepare/styles.go:139-290`, `css/relayout.go:60-75`).
  - **Root cause:** `SheetCache` holds three unsynchronized Go maps (`inline`, `links`, `imports`) and `css.Relayout` returns a new `*css.Document` sharing the same `doc.cache` pointer. Concurrent `css.Relayout` calls trigger `fatal error: concurrent map read and map write`.
  - **Fix:** Guard `SheetCache` map reads/writes with an internal `sync.RWMutex` via accessor methods (`getInline`/`setInline`, `getLink`/`setLink`, `getImport`/`setImport`).
  - **Proof:** Concurrent `css.Relayout` race test (including `@media`-gated `<link>`/`@import` activation) passing `go test -race ./css ./internal/convert/prepare`.

- [ ] `BLK-CONC-03` Stop returning a globally cached mutable slice from `transitionsFrom` (`layout/layout.go:400-448`).
  - **Root cause:** `transitionsFrom` caches `[]Transition` in global `transitionSliceCache` keyed by `&records[0]` pointer + length and returns the cached slice directly into exported `Box.Transitions`. Caller mutation corrupts the global cache for all documents, and strong `*ilayout.TransitionRecord` keys pin backing arrays globally (plus risk stale hits if slices are reused).
  - **Fix:** Return a cloned slice (`slices.Clone`) or cache per-engine/per-display instead of globally (or drop global caching if conversion cost is negligible on small transition lists).
  - **Proof:** Unit test mutating `box.Transitions[0]` and verifying subsequent `DisplayList` calls are unaffected + benchmark check.

## 1.2 CSS Selector, Unicode & Custom-Property Correctness

- [ ] `BLK-CSS-01` Add combinator backtracking for descendant (` `) and general sibling (`~`) in `leftmostMatch` and `matchPseudoWalk` (`internal/css/match.go:414-469, 511-547`).
  - **Root cause:** `leftmostMatch` and `matchPseudoWalk` walk selector parts right-to-left in a flat greedy loop without backtracking. For `.a > .b .c` against `<div class="a"><div class="b"><div class="x"><div class="b"><span class="c">`, the nearest `.b` inside `.x` fails `.a >`, and matching immediately fails without trying the outer `.b`. Same false negative for `.a + .b ~ .c`.
  - **Fix:** Convert the combinator walk into a recursive (or explicit-stack) matcher that backtracks across all candidate ancestors for ` ` and all candidate preceding siblings for `~` (preserving the existing `:has()` visit budget and zero-alloc fast path for single-part / child-only selectors).
  - **Proof:** Regression tests in `internal/css/` covering `.a > .b .c`, `.a > .b .c::before`, `.a + .b ~ .c`, and deep-ancestor non-match performance bounds.

- [ ] `BLK-CSS-02` Fix UTF-8 `strings.ToLower` byte-offset mismatch in `FontFaceURLs` and `expandOneRepeat` (`internal/css/css.go:616-652`, `internal/layout/grid_parse.go:130-159`).
  - **Root cause:** Both compute `strings.ToLower(src)`, find a byte index in the lowercased string, and slice original `src` using that index. Non-ASCII runes whose UTF-8 byte length changes under `strings.ToLower` (e.g., `İ` U+0130: 2 B → 3 B; `Ⱥ`/`Ⱦ`: 2 B → 3 B) shift offsets, corrupting extracted URLs/grid templates or panicking out-of-bounds.
  - **Fix:** Replace `strings.ToLower` indexing with an ASCII-only case-insensitive byte search helper (`indexASCIIFold`) that preserves exact byte offsets in `src`.
  - **Proof:** Unit + fuzz tests (`FuzzParseCSS`, `FuzzExpandRepeat`) with inputs like `local("İstanbul"), url(Font.woff2)` and `[İstanbul] repeat(2, 1fr)`.

- [ ] `BLK-CSS-03` Fix `ResolveCustomProps` embedded `var()` expansion and add a recursion depth cap (`internal/css/values.go:718-765`).
  - **Root cause:** `ResolveCustomProps` calls `ResolveVar` (`values.go:749`), which only expands values *starting* with `var(`, so embedded references like `--border: 1px solid var(--accent)` or `--w: calc(var(--u) * 2)` fail to expand. In addition, `eval` detects cycles via `stack[name]` but has no depth cap for long linear chains (`--v1: var(--v2); ...`).
  - **Fix:** Call `ResolveVars` inside `ResolveCustomProps` and enforce a bounded recursion depth (e.g., `maxCustomPropDepth = 64`), returning invalid/empty beyond the limit.
  - **Proof:** Unit tests in `internal/css/css_test.go` covering embedded `var()` in custom properties (`--border: 1px solid var(--accent)`, `--w: calc(var(--u) * 2)`) and a 1,000-step linear chain without stack overflow.

- [ ] `BLK-CSS-04` Preserve CSS Custom Properties Level 1 case sensitivity across declaration parsing and `parseVarFn` (`internal/css/values.go:584-595`, CSS declaration parser).
  - **Root cause:** `parseVarFn` lowercases custom property names (`strings.ToLower`), collapsing `--colorPrimary` and `--colorprimary` contrary to CSS Custom Properties Level 1.
  - **Coordination note:** Update declaration parsing and `parseVarFn` together so declared custom properties and `var(--...)` lookups both preserve exact case without breaking existing camelCase custom property lookups.
  - **Proof:** Unit + layout tests verifying `--colorPrimary` and `--colorprimary` resolve independently while standard CSS properties remain case-insensitive.

- [ ] `BLK-CSS-05` Fix stale `siblingEntryFor` invalidation on same-length child replacement (`internal/css/match.go:175-205`, `html/attribute.go:129-148`).
  - **Root cause:** `siblingEntryFor` caches `sibParentData` keyed by `weak.Make(parent)` and validates solely with `entry.childLen == len(parent.Children)`. Replacing one element child with one `TextNode` via `SetText` keeps `len == 1`, returning stale `elems` containing the detached element.
  - **Fix:** Track a per-node or per-document mutation counter (e.g., `parent.ChildVersion` incremented on child splice/replace) and compare `entry.version == parent.ChildVersion` instead of `childLen` alone.
  - **Proof:** Regression test calling `html.Document.SetText` on an element with one child element and verifying `:first-child`/`:nth-child`/`:only-of-type` reflect the updated tree.

---

## Closure Gates

- [ ] `BLK-P0-GATE-01` `go test -race ./internal/html ./css ./internal/css ./internal/convert/prepare ./layout ./internal/layout` exits `0`.
- [ ] `BLK-P0-GATE-02` `make test`, `make golden`, `make lint`, `make claim-scan`, `make catalog-check`, and `make matrix-check` exit `0`.
- [ ] `BLK-P0-GATE-03` Record passing commit hash to unblock `ownframe` Phase 04 `HYG-ENG-01` (`engineMu` removal).
