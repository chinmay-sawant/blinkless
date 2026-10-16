# v0.0.3 - Anchor positioning

> **Parent:** `plans/v0.0.3/README.md`
> **Status:** complete 2026-10-16. All rows validated. `make test`, `make golden`, `make lint`, `make claim-scan`, `make catalog-check` and `make matrix-check` exit 0.
> **Estimated effort:** 8 properties, 6 phases.

---

## Overview

Anchor positioning places one absolutely or fixed positioned box against another box on the same canvas. The output stays `layout.DisplayList`. Paint Y grows downward. 1 CSS pixel is 0.75 points (`layout/displaylist.go:12-14`, `layout/displaylist.go:27-29`). There is no page bitmap and no JavaScript.

This ledger is only `anchor-name`, `anchor-scope`, `position-anchor`, `position-area`, `position-try`, `position-try-fallbacks`, `position-try-order`, and `position-visibility`. It does not include `scroll-timeline`, `view-timeline`, `animation-timeline`, or `pointer-timeline`.

`documentation/deferred.md` now lists anchor positioning as an implemented one-canvas subset. The catalog, read on 2026-10-16 from `testdata/css/catalog/properties.json` (`updated` field `2026-10-16`), marks all eight rows `implemented` for the one-canvas contract, each with handler `applyAnchorProps` and resolvable behavior tests. `position-visibility` carries the scroll non-goal as a limitation. Phase 5 landed those edits after the tests passed.

Absolute and fixed boxes already move. `buildOutOfFlow` sets X from `left` or `right`, then Y from `resolveAbsY`, then shifts the box and its paint ops. New placement math lives in `internal/layout/anchor_position.go`. That math was not added to `internal/layout/layout.go`, which stays at its allowlisted size. Anchor placement runs once from `finalizeResult` in `internal/layout/layout_context.go` before `finalizeChrome` and may shift again.

The drawing list does not need a new public field. A passing test shows the positioned box's X and Y changed. The host draws those ops where they already are.

## Executive summary

| Property | Catalog status | Pinned syntax |
|---|---|---|
| `anchor-name` | implemented | `none \| <anchor-name>#` |
| `anchor-scope` | implemented | `none \| all \| <anchor-name>#` |
| `position-anchor` | implemented | `auto \| <anchor-name>` |
| `position-area` | implemented | `none \| <position-area>` |
| `position-try` | implemented | `<'position-try-order'>? <'position-try-fallbacks'>` |
| `position-try-fallbacks` | implemented | `none \| [ [<dashed-ident> \|\| <try-tactic>] \| <position-area> ]#` |
| `position-try-order` | implemented | `normal \| <try-size>` |
| `position-visibility` | implemented | `always \| [ anchor-valid \|\| anchor-visible \|\| no-overflow ]` |

Syntax strings are the pinned upstream inventory `webref-ed-css` revision `1f2ec8f74a80c14066b4c7d6822cee59f69fa03b`, spec `css-anchor-position`. `position-visibility` uses the singular tokens `anchor-valid` and `anchor-visible` from that inventory (`testdata/css/catalog/upstream/webref-ed-css-1f2ec8f.json:5488`). Do not rename them in this ledger.

`anchor()` inside `top`, `left`, `right`, `bottom`, or `inset` is not one of these eight properties. It stays out. `position-area` is enough to move a box.

## Phase 0: current contract

- [x] Confirm all eight catalog rows are `unsupported`, with empty `handlers` and empty `tests`. Read from `testdata/css/catalog/properties.json` on 2026-10-15. Name lines from search: `anchor-name` at 936, `anchor-scope` at 952, `position-anchor` at 12821, `position-area` at 12837, `position-try` at 12853, `position-try-fallbacks` at 12869, `position-try-order` at 12885, `position-visibility` at 12901. Each limitation says the declaration is ignored.
- [x] Confirm no layout apply handler mentions these names. A search of `internal/layout/*.go` finds no `anchor-name` or `position-anchor` property handler. Absolute placement is still `left` / `right` / `top` / `bottom` in `buildOutOfFlow` (`internal/layout/layout.go:2032-2044`).
- [x] Record the placement rule in `internal/layout/anchor_position.go` before any keyword list is hardcoded. `position: absolute` and `position: fixed` both enter anchor placement (`placeAnchorTarget` returns otherwise); `position: sticky` does not. The first slice covers absolute and fixed only. Proven by the absolute/fixed gate in code plus the static no-move test.
- [x] Read `<anchor-name>`, `<position-area>`, `<try-tactic>`, and `<try-size>` from the pinned `css-anchor-position` spec at the hrefs in `testdata/css/catalog/upstream/webref-ed-css-1f2ec8f.json` before accepting a keyword. The accept tables in `anchor_position.go` match that text. Proven by storage tests rejecting unknown tokens.
- [x] `position-anchor: auto` uses the anchor only when exactly one in-scope `anchor-name` exists. If several exist, `auto` does not guess, and the box keeps its laid-out offsets. A later explicit `position-anchor` names one. Proven by `TestAnchorScopeLookup` single versus ambiguous cases.
- [~] Scrollport visibility for `anchor-visible`. Reason: this tree has no scroll offset. The native view ledger owns host scroll (`plans/v0.0.1/native-window/native-web-view-checklist.md`), and Blinkless returns one drawing list. Owner: the host scroll work, not this file. Next gate: a test with a non-zero scroll offset hides the positioned box when the anchor lies outside that port. Until then, `anchor-visible` is stored and treated like `anchor-valid` on this canvas.
- [~] The `anchor()` inset function. Reason: it is not one of the eight properties in this ledger. Owner: a later inset-function ledger. Next gate: that ledger exists and names the properties it changes. `position-area` still moves the box without it.

Gate. A short note in this file states the one-anchor `auto` rule, the absolute and fixed limit, and the two non-goals above. No code yet.

## Phase 1: names and scope

New code lives in `internal/layout/anchor_position.go`. Style fields land beside the other used values, not in `internal/layout/layout.go`.

- [x] Store `anchor-name: none` and one or more dashed-ident names on the element. A missing or invalid name leaves the previous value. Path: `internal/layout/anchor_position.go`. Expected: two boxes can carry different names, and `none` clears the list. Proof: `TestAnchorNameScope` reads the used names.
- [x] Store `anchor-scope: none | all | <anchor-name>#`. `none` is the initial and does not hide names. `all` hides every name declared on that element from descendants outside the element. A name list hides only those names. Path: `internal/layout/anchor_position.go`. Expected: a descendant inside the scope sees the name, and a box outside it does not. Proof: `TestAnchorScopeLookup`.
- [x] Build the in-scope name map while walking the box tree, before out-of-flow placement. A name on a box is visible to later boxes that are not excluded by an ancestor scope. Path: `internal/layout/anchor_position.go`, called once from `finalizeResult` in `layout_context.go` rather than grown inside `layout.go`. Expected: the map is empty when no `anchor-name` is set. Proof: `TestAnchorScopeLookup` empty-map case plus the two-box test in Phase 2.

Gate. Names and scope are stored and looked up. Nothing moves yet.

## Phase 2: move one box with position-area

- [x] Store `position-anchor` as `auto` or one dashed-ident. An unknown name does not replace an earlier valid name. Path: `internal/layout/anchor_position.go`. Expected: the used anchor is the named box when it is in scope, and unset when it is not. Proof: `TestAnchorInvalidLeavesPrevious` plus the placement test.
- [x] Apply `position-area` only when the target is absolute or fixed and an in-scope anchor was found. Place the target's border box in the chosen cell of the grid around the anchor's border box. The keyword list comes from the pinned `<position-area>` production opened in Phase 0. Path: `internal/layout/anchor_position.go`, invoked once from `finalizeResult` in `layout_context.go` before chrome merge (not inside `layout.go`, which stays at its allowlisted size). Expected: a target with the keyword that means "above the anchor" has a smaller Y than the anchor's Y. Options are tested in order against their own position-area region and the first fitting option wins; the box is safe-shifted to stay inside the containing block. The anchor's own X and Y do not change. Proof: `TestBehaviorPositionAreaPlacesAboveAnchor` plus `TestBehaviorPositionAreaTopExplicitKeepsFallback` and `TestBehaviorPositionAreaBottomExplicitAnchorsY` in `internal/layout/css_behavior_anchor_test.go`.
- [x] `position-area: none` leaves the absolute offsets from `left` / `top` alone. Expected: the box matches a document that omits `position-area`. Proof: `TestBehaviorPositionAreaNone`.
- [x] An anchor that is not in scope does not move the target. Expected: scoped-out name behaves like a missing name. Proof: `TestBehaviorPositionAreaOutOfScope` with `anchor-scope` wrapping the anchor and the target outside that subtree.
- [x] Do not change in-flow boxes. A `position: static` target with `position-area` set keeps its flow Y. Proof: `TestBehaviorPositionAreaStatic`.

Gate. Two boxes on one canvas. The positioned box's paint ops move with the box. `make golden` is still a closure gate, not this phase's only check. Run the new test with `go test ./internal/layout -run TestBehaviorPositionArea`.

## Phase 3: try order and fallbacks

- [x] Store `position-try-fallbacks`. Accept `none`, a list of `<position-area>` values, and `<try-tactic>` keywords taken from the pinned spec. Path: `internal/layout/anchor_position.go`. Expected: options are tried in order, the base first, and the first option whose box fits its position-area region wins regardless of the visibility token. When the base fits, the fallback is unused; when the base spills, the fallback wins even under visibility `always`. Proof: `TestBehaviorPositionTryAlwaysKeepsBase` and `TestBehaviorPositionTryBaseFitsWins`.
- [x] `position-try-order: normal` keeps fallback document order. Size keywords from `<try-size>` pick the candidate whose border box is largest on that axis among candidates that fit their regions. Path: same file. Expected: a wide candidate wins for the inline-size keyword when both fit. Proof: `TestBehaviorPositionTryOrderMostInlineSizePicksWide`.
- [x] Expand `position-try` into order plus fallbacks, including the case with only fallbacks. CSS-wide keywords reset both longhands. Path: the shorthand expander in `internal/layout/style_shorthand.go`, not `layout.go`. Expected: a following longhand still overrides the shorthand, matching source order. Proof: `TestBehaviorPositionTryShorthandExpands` and `TestBehaviorPositionTryShorthandCascadeOrder`.
- [x] Dashed-ident fallbacks need `@position-try`. Parse that at-rule only far enough to store the fallback's `position-area` or inset overrides. Path: `internal/css/at_position_try.go` next to the existing at-rule parsers, plus lookup in `anchor_position.go`. Do not parse timeline at-rules. Expected: `position-try-fallbacks: --above` uses the area stored on `@position-try --above`. An unknown ident is skipped. Proof: `TestPositionTryAtRule`.
- [x] A tactic flips the chosen area across the axis the pinned spec names and swaps the corresponding inset sides (`top`/`bottom` for a vertical flip, `left`/`right` for a horizontal one, both for `flip-start`). Tactic entries are fallback candidates tried in order. Expected: the flipped candidate's Y is on the opposite side of the anchor from the unflipped candidate when the base spills. Proof: `TestBehaviorPositionTryTactic` plus fixture-66 row 12, where both engines flip only after the base spills and swap the tip's insets.

Gate. A target with two fallbacks ends on the candidate the order rule selects. The other candidate's geometry is absent.

## Phase 4: visibility on one canvas

- [x] Store `position-visibility` using the pinned tokens `always`, `anchor-valid`, `anchor-visible`, and `no-overflow`. Invalid tokens do not replace an earlier valid value. Path: `internal/layout/anchor_position.go`. Proof: the store subtest in `TestBehaviorPositionVisibility`.
- [x] `always` never hides: the base styles keep applying. When the base spills its region, the box is safe-shifted to stay inside the containing block when it can; when a later option fits its region, that option wins instead. Expected: the fill op stays in `Result.Ops` in every case. Proof: the `base-fits-keeps-laid-out`, `always-overflow-takes-fallback`, and `always-shifts-base-to-fit` subtests.
- [x] `anchor-valid` rejects a candidate when the named anchor is missing or out of scope, then tries the next fallback. If none are valid, the box keeps ordinary absolute placement and still paints. Expected: no anchor means the `left` / `top` position, not a removed box. Proof: `anchor-valid-dangling` and `anchor-valid-out-of-scope` subtests.
- [x] `no-overflow` hides the box only when every option fails its fit test, base and fallbacks alike. Each option's box must fit its position-area region. Expected: when one option fits it wins; when none fit, no fill op for the tip is in `Result.Ops` while the anchor still paints. Proof: `TestBehaviorPositionVisibilityNoOverflow` and `TestBehaviorPositionVisibilityNoOverflowHidesWhenAllReject`, plus fixture-66 row 9 where both engines hide the tip, matching headless Chrome 155 print measured on 2026-10-16.
- [x] `anchor-visible` follows the Phase 0 non-goal. It is accepted and stored, and on this canvas it does not hide a box whose anchor is on the canvas. Expected: identical geometry to `anchor-valid` until a scroll offset exists. Proof: `AnchorVisibleMatchesAnchorValid` comparison test pointing at the `[~]` row.

Gate. Visibility changes which candidate wins. It does not add a scrollport and it does not drop ops for `always`.

## Phase 5: catalog, docs, and gates

- [x] Point the eight catalog rows at `internal/layout/anchor_position.go` and the new behavior tests. All eight rows are `implemented` for behavior a test observes in the drawing list. The scrollport case stays the Phase 0 host-scroll non-goal, named in the `position-visibility` limitation. `make catalog-check` exits 0.
- [x] Update `documentation/deferred.md` so this family is no longer a blanket print non-goal. States the one-canvas contract and the scroll non-goal.
- [x] Update `documentation/compatibility-matrix.md` only through the matrix generator. Generated property rows untouched by hand; `make matrix-check` exits 0.
- [x] Update the local knowledge-base page for anchor positioning in the same session as the code, per the repo ledger rule. Landed as `documentation/anchor-positioning.md`. This checklist does not do that edit.
- [x] Run `make test`. Exit 0.
- [x] Run `make golden`. Exit 0. `layout.TestDisplay` passes.
- [x] Run `make lint`. Exit 0. `internal/layout/layout.go` did not grow.
- [x] Run `make claim-scan`. Exit 0.
- [x] Run `make catalog-check`. Exit 0.
- [x] Run `make matrix-check`. Exit 0.

GitHub CI runs only `make test` and `make lint` (`.github/workflows/ci.yml`). The other gates are local. Do not run bare `go test ./...`.

Gate. All five commands above exit 0, and the catalog rows match the tests.

## Dependencies

- Absolute and fixed placement already shifts ops in `buildOutOfFlow` (`internal/layout/layout.go:2042-2044`). Anchor placement runs after that shift and may shift again.
- `anchor-scope` changes lookup only. It does not create a containing block.
- `position-try` depends on Phase 2 placement and Phase 4 overflow checks.
- `@position-try` depends on the CSS parser's at-rule path. It does not depend on JavaScript or on scroll timelines.
- Scroll-linked visibility waits on the host scroll work named in Phase 0. This ledger does not start that work.
- No new direct module. The allowlist stays `github.com/go-text/typesetting` and `github.com/tdewolff/canvas`.
