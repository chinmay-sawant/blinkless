# v0.0.3 - Regions, masonry tolerance, exclusions, and wrap breaks

> **Parent:** `plans/v0.0.3/README.md`
> **Status:** not started, except baseline rows verified in this session.
> **Estimated effort:** 9 properties, 6 phases.

---

## Overview

These nine names are not one spec. The pinned upstream inventory `webref-ed-css` revision `1f2ec8f74a80c14066b4c7d6822cee59f69fa03b` splits them:

| Property | Spec in the inventory | Href line |
|---|---|---|
| `flow-into`, `flow-from` | css-regions-1 | `testdata/css/catalog/upstream/webref-ed-css-1f2ec8f.json:3295-3310` |
| `region-fragment` | css-regions-1 | same file at 5537-5543 |
| `flow-tolerance` | css-grid-3 | same file at 3313-3319 |
| `wrap-flow`, `wrap-through` | css-exclusions-1 | same file at 7545-7569 |
| `wrap-before`, `wrap-after`, `wrap-inside` | css-text-4 | same file at 7527-7560 |

`documentation/deferred.md:109` lumps all nine together as a permanent print non-goal. That sentence is wrong about the specs and behind this ledger. Phase 5 rewrites it.

The catalog, read on 2026-10-15 (`properties.json` `updated` is `2026-10-13`), marks every name `unsupported` with no handler and no test.

The product output is one drawing list. `layout.DisplayList` does not paint a page bitmap (`layout/displaylist.go:129-137`). `Result.Pages` is documented as filled by Paint (`internal/layout/layout.go:193-197`). This session found no write to `Pages` except clone and nil (`internal/layout/layout.go:228`, `internal/layout/layout.go:340`). `break-before: page` stores `always` and does not move the box (`internal/layout/css_behavior_backface_breaks_test.go:126-159`). Normal flow is one continuous list (`internal/layout/css_behavior_margin_break_test.go:13-18`). Multicol can place columns side by side and may snap using `Options.Height` (`internal/layout/multicol.go:367-385`). That is column geometry on the same list, not a page the public API returns.

Paint Y grows downward. 1 CSS pixel is 0.75 points (`layout/displaylist.go:12-14`).

Do not add new logic to `internal/layout/layout.go`. Named flows go in `internal/layout/named_flow.go`. Masonry tolerance goes in `internal/layout/masonry_tolerance.go`. Exclusions go in `internal/layout/exclusion.go`. Text and flex wrap breaks go in `internal/layout/wrap_break.go`.

## Executive summary

The smaller contract is four separate layout changes on one canvas. None of them invent a page.

1. Regions. `flow-into` names an element's content. `flow-from` on a second box lays that content out inside the second box. Two boxes, one list. A third box with the same `flow-from` receives only what did not fit in the second. That is the only split.
2. `flow-tolerance` is a masonry packing knob, not a region knob. `emitMasonryItems` already packs each item into the shortest column (`internal/layout/grid_masonry.go:50-89`, `internal/layout/grid_masonry.go:142-152`). Tolerance decides when "shortest" loses to "this item would stick out too far."
3. Exclusions. `wrap-flow` makes a box push later line boxes aside. `wrap-through` lets a later box ignore that push. Floats already track a rectangular side exclusion (`internal/layout/float.go:15-47`).
4. Wrap breaks. `wrap-before: line` forces the next line. `wrap-inside: avoid` keeps a box on one line when it fits. These are line and flex breaks, not pages.

Paged fragmentation is a non-goal in Phase 0.

## Phase 0: what kind of split this is

- [x] Confirm the nine catalog rows are `unsupported` with empty handlers. Name lines in `testdata/css/catalog/properties.json`: `flow-from` 7321, `flow-into` 7337, `flow-tolerance` 7353, `region-fragment` 13032, `wrap-after` 17592, `wrap-before` 17608, `wrap-flow` 17624, `wrap-inside` 17640, `wrap-through` 17656.
- [x] Confirm the engine does not return pages. `DisplayList` returns one `Display` of ops (`layout/displaylist.go:137-142`). `break-before: page` leaves the second box at the in-flow Y (`internal/layout/css_behavior_backface_breaks_test.go:146-159`).
- [ ] Regions are a named-flow move on one canvas, not a drawing-list comment and not a page split. `flow-into: ident` on box A takes A's children out of normal flow. `flow-from: ident` on box B lays those children inside B's content box. Their paint ops use B's coordinates. If B is shorter than the content and box C has the same `flow-from`, the remainder starts at C. If C is absent, the remainder stays unpainted past B rather than inventing a page. Record this in the package comment of `internal/layout/named_flow.go`.
- [ ] `region-fragment: auto` may slice one child across B and C. `region-fragment: break` moves that whole child to C when it does not fit in B. Both stay on one canvas.
- [ ] `flow-tolerance` is not part of that flow. Implement it only in the masonry packer. `normal` means one em of slack. A length is that slack in points. `infinite` keeps today's shortest-column choice even when the item sticks out. Zero means no slack.
- [~] Paged fragmentation. Reason: nothing in the public API assigns ops to a page index, and a forced page break does not move a box. Building region chains that synthesize pages would pretend Paint still exists. Owner: a future pagination ledger, not this file. Next gate: a public result exposes a second page index, and a test shows `break-before: page` moving a box onto that index. Until that gate, `region-fragment: break` means "next region box on this canvas," and the `wrap-*` values do not mean "next page."

Gate. This phase's decision is written down. Regions change which box owns the ops. They do not create `Result.Pages`.

## Phase 1: two boxes and one named flow

- [ ] Store `flow-into` as `none` or a custom ident, plus the optional `element` or `content` keyword. Initial is `none`. `content` is the first slice and moves the children. `element` moves the element itself, including its own box. Invalid idents do not replace an earlier value. Path: `internal/layout/named_flow.go`. Proof: a style test.
- [ ] Store `flow-from` as `none` or one custom ident. The element becomes a region box and does not also paint its own in-flow children. Path: same file. Proof: a style test.
- [ ] Pull `content` into the region. Given A with `flow-into: article content` and B with `flow-from: article`, A's text ops are inside B's content box and are absent from A's old flow position. Path: `internal/layout/named_flow.go`, called from the flow walk by a helper, not by new code in `layout.go`. Expected: the text op Y is within B. Proof: `TestBehaviorNamedFlowMovesTextIntoSecondBox` in `internal/layout/css_behavior_named_flow_test.go`.
- [ ] `flow-into: none` and `flow-from: none` leave both boxes in normal flow. Expected: Y matches a document that omits both properties. Proof: the same file.
- [ ] An ident with a source and no region drops that source's children from the normal flow and does not invent a box to hold them. Expected: the source's text is not in `Result.Ops`. Proof: one fixture. This matches "no region means the content is not on this canvas" without creating a page.
- [ ] `element` moves the whole source box into the region, so the source's border paints inside the region rather than in the old flow. Expected: the border op's Y is inside B. Proof: one fixture after the `content` test is green.

Gate. Two boxes. The text changes coordinate parent. No `Pages` slice is allocated.

## Phase 2: a second region, fragment, and masonry tolerance

- [ ] Chain two `flow-from` boxes in tree order. Fill the first region's content box, then continue in the second. Expected: a long paragraph's first line is in B and a later line is in C. Both Y values are on the one canvas. Proof: `TestBehaviorNamedFlowContinuesInNextRegion`.
- [ ] `region-fragment: auto` allows one block child to start in B and continue in C. `region-fragment: break` on that child moves the whole child to C when the remaining space in B is shorter than the child. Expected: under `break`, no paint op of that child sits inside B. Proof: a pair of fixtures.
- [ ] Store `flow-tolerance` on the grid container. Path: `internal/layout/masonry_tolerance.go`. `masonryPlacement` calls it (`internal/layout/grid_masonry.go:92-118`). Do not read this property from `named_flow.go`.
- [ ] `infinite` keeps `shortestMasonryColumn`. A finite tolerance skips a column when the item's block size would extend past the neighboring column bottom by more than the tolerance, and picks the next column that passes. `normal` is one em at the container's font size. Expected: two masonry items change column, so the second item's X differs between `infinite` and `0px`. Proof: `TestBehaviorFlowToleranceChangesMasonryColumn`. The packer runs only for `grid-template-rows: masonry` (`internal/layout/grid.go:25-27`, `internal/layout/grid.go:81-82`). A `grid-template-columns: masonry` list is cleared and does not enter this path (`internal/layout/grid_masonry.go:14-21`). The fixture sets rows to `masonry` and columns to two definite tracks.
- [ ] A non-masonry grid ignores `flow-tolerance`. Expected: item X and Y match the same grid without the property. Proof: one comparison fixture.

Gate. Region overflow stays on the canvas. Tolerance only changes masonry X.

## Phase 3: exclusions

Floats already shorten later lines from `leftEdge` and `rightEdge` (`internal/layout/float.go:32-46`). `wrap-flow` adds an exclusion that is not required to be `float`.

- [ ] Store `wrap-flow` as `auto | both | start | end | minimum | maximum | clear`. `auto` means no exclusion. Path: `internal/layout/exclusion.go`. The line breaker reads the exclusion through the existing `floatState` rather than a second wrap engine.
- [ ] `wrap-flow: both` on a sized box shortens the following paragraph on both sides of that box, so a line's X starts after the exclusion's start edge and its width stops before the exclusion's end edge. Expected: the line's X is greater than the content start, and the line's right edge is less than the content end, while the exclusion box itself stays put. Proof: `TestBehaviorWrapFlowBothShortensLine`.
- [ ] `start` and `end` shorten only one side. `minimum` and `maximum` pick the side with the smaller or larger free gap. `clear` places the following line below the exclusion, which is the same Y outcome as `clear: both` past a float. Proof: one fixture per keyword, checking line X or line Y.
- [ ] Store `wrap-through` as `wrap` or `none`. `wrap` is the initial and keeps the exclusion. `none` on the paragraph makes its lines use the full content width even when an earlier `wrap-flow` exclusion overlaps them. Expected: line X returns to the content start. Proof: `TestBehaviorWrapThroughNoneIgnoresExclusion`.
- [ ] `wrap-flow: auto` does not change a real `float: left`. The float tests stay green. Proof: the existing float test file still passes under `go test ./internal/layout -run Float`.

Gate. A exclusion box and a paragraph on one canvas. Line X or Y changes. No new paint op kind.

## Phase 4: wrap-before, wrap-after, and wrap-inside

These are css-text-4 wrap opportunities. `line` and `avoid-line` talk to the inline line breaker. `flex` and `avoid-flex` talk to `flexWrapLines` (`internal/layout/flex.go:388`). They do not talk to `Result.Pages`.

- [ ] Store the three properties with the syntax in the executive table. Invalid values do not replace an earlier valid value. Path: `internal/layout/wrap_break.go`.
- [ ] `wrap-before: line` forces a line break before the element, so its text starts on the next line Y. `wrap-after: line` forces the following text onto the next line. `auto` does not. Proof: `TestBehaviorWrapBeforeLineBreaks`.
- [ ] `wrap-inside: avoid` keeps the element's text on one line when the line has room. When the line does not have room, the text wraps anyway. Expected: a short phrase stays one line, a long phrase still wraps. Proof: two fixtures.
- [ ] `avoid` and `avoid-line` skip a soft wrap immediately before or after the element when the text still fits. They do not overflow the content box without bound. Expected: a measured line count of 1 for the fitting fixture and more than 1 for the overflow fixture.
- [ ] `flex` on a flex item starts a new flex line even when the item would fit on the current line. `avoid-flex` keeps the item on the current flex line when it fits. Path: a helper called from `flexWrapLines`. Expected: the item's Y matches the next flex line for `flex`, and the current line for `avoid-flex`, using a container with `flex-wrap: wrap`. Proof: `TestBehaviorWrapBeforeFlexBreaksLine`.
- [ ] Do not treat `line` or `flex` as `break-before: page`. A box with `wrap-before: line` stays above `Options.Height` when its content is short. Proof: Y is below the previous line and below 800pt is not required. The assertion is "Y increased by about one line," not "Y jumped by a page."

Gate. Line and flex geometry change. Page assignment does not.

## Phase 5: catalog, docs, and gates

- [ ] Update each catalog row only after its behavior test passes. `flow-into`, `flow-from`, and `region-fragment` can become `implemented` for the one-canvas contract, with a limitation that names the paged-fragmentation non-goal. `flow-tolerance` becomes `implemented` only for masonry, and `partial` if normal grid ignores it, with that limitation written down. `wrap-flow` and `wrap-through` follow the exclusion tests. `wrap-before`, `wrap-after`, and `wrap-inside` follow the line and flex tests.
- [ ] Rewrite `documentation/deferred.md:109` so it no longer calls these nine one permanent non-goal. Name the four contracts and the page non-goal.
- [ ] Regenerate the compatibility matrix. Do not hand-edit generated rows.
- [ ] Update the local knowledge-base page in the implementation session. Not in this checklist's writing session.
- [ ] Run `make test`. Exit 0.
- [ ] Run `make golden`. Exit 0.
- [ ] Run `make lint`. Exit 0. `internal/layout/layout.go` did not grow.
- [ ] Run `make claim-scan`. Exit 0.
- [ ] Run `make catalog-check`. Exit 0.
- [ ] Run `make matrix-check`. Exit 0.

GitHub CI runs only `make test` and `make lint`. Do not run bare `go test ./...`.

Gate. The six commands exit 0. Catalog limitations mention the page non-goal instead of hiding it.

## Dependencies

- Named flow runs after in-flow children are measured and before paint ops are treated as final. It reuses the existing block and inline layout of those children inside the region width.
- `region-fragment` depends on Phase 1. It does not depend on multicol's `Options.Height` snap (`internal/layout/multicol.go:378-385`).
- `flow-tolerance` depends on `emitMasonryItems`. It does not depend on named flows.
- `wrap-flow` depends on `floatState` (`internal/layout/float.go:32-47`). `shape-outside` contours stay as they are. This ledger does not replan them.
- `wrap-before: flex` depends on flex wrap (`internal/layout/flex.go:240-247`). A nowrap container ignores `flex`.
- Paged fragmentation stays blocked on the Phase 0 gate. Do not start it from this file.
- No new direct module.
