# v0.0.3 - CSS shapes inside

> **Parent:** `plans/v0.0.3/README.md`
> **Status:** implemented and merged on `feature/v0.0.3-mask-border-regions-shapes`. Phases 1-3 verified (`TestShapeInsideStore*`, `TestShapePaddingStore*`, `TestBehaviorShapeInside*` green); the phase 4 and 5 decision gates chose "no consumer" (no `url()` image shapes; `float-defer` stays at `none`); phase 6 catalog (`shape-inside` and `shape-padding` implemented) and the matrix are merged. Gate rows record the branch-level gate results. Baseline rows record verified behavior, with superseded baselines noted inline.
> **Estimated effort:** 6 catalog properties (2 implemented, 4 open) across 6 phases.

---

## Overview

`shape-outside` changes the hole a float leaves in the lines beside it. `shape-inside` would change the lines inside the element itself. Those are different hooks. This file plans the inside hook and three neighbors. It does not replan `shape-outside` or `shape-margin`.

Chrome does not implement `shape-inside`. The catalog limitation says "CSS Shapes 2 interior fitting; Chrome has no BCD support." The opened test says the same for Chrome 143 (`internal/layout/css_behavior_list_shape_test.go:92-93`). The first row in phase 1 is a decision gate. The rest of the rows are still written. This is not a claim of browser parity.

`shape-padding` has the same catalog note: no Chrome BCD support. `shape-image-threshold` has no alpha contour today because `url()` shapes are rejected. `float-defer` is a page-float property (`https://drafts.csswg.org/css-page-floats-3/#propdef-float-defer`), not a shape. The catalog limitation says there is no page-float defer model.

`shape-rendering` is an SVG hint. It belongs to `plans/v0.0.3/svg-presentation/svg-presentation-checklist.md`, not this file.

The output is a display list. Inside wrapping changes text op positions. It does not add a page rasterizer.

`documentation/deferred.md` lists `shape-inside`, `shape-padding`, `shape-image-threshold`, and `float-defer` as leftovers left unsupported. This file is the planning home. Do not edit `deferred.md`.

## Executive summary

| Property | Catalog status | This file |
|---|---|---|
| `shape-outside` | implemented | Baseline only. Subset is `none`, `circle()`, `ellipse()`, `inset()`. `polygon()`, `path()`, and `url()` are rejected |
| `shape-margin` | implemented | Baseline only. Expands the outside contour |
| `shape-inside` | unsupported | Decision gate, then storage and a line-box consumer |
| `shape-padding` | unsupported | Inset on the inside contour. Depends on `shape-inside` |
| `shape-image-threshold` | unsupported | Blocked on an image shape this tree does not parse |
| `float-defer` | unsupported | Decision gate. No page-float model exists |

## Phase 0: contract and baseline

- [x] `shape-outside` is stored by `applyShapeProps` (`internal/layout/style_shape_props.go:26-34`). The accepted subset is `none`, `circle()`, `ellipse()`, and `inset()`, optional shape-box keyword (`internal/layout/style_shape_props.go:11-16`). `polygon()`, `path()`, and `url()` leave the previous value. Proof: `TestBehaviorShapeOutsideNarrowsWrap` (`internal/layout/css_behavior_svg_paint_test.go:373`) plus `TestShapeOutsideStoreRejectedTokensKeepEarlier`. `none` builds no contour. `circle()` builds one.
- [x] The contour is resolved against the float margin box (`internal/layout/shape_exclusion.go:94-136`). A shape-box keyword is stored and then ignored at place time. The comment says resolution always uses the margin box (`internal/layout/style_shape_props.go:47-48`). Do not "fix" that in this ledger.
- [x] Line boxes beside a float ask `lineBounds` (`internal/layout/inline.go:609-617`), which calls `floatState.exclusion` (`internal/layout/float.go:171-206`). A left contour replaces the rectangular left edge with `rightEdgeAt`. That is the outside hook. There is no hook that narrows a block's own lines.
- [x] `shape-margin` is stored as a length or a percentage (`internal/layout/style_shape_props.go:35-38`). The percentage uses the reference diagonal (`internal/layout/shape_exclusion.go:139-144`). Proof: `TestBehaviorShapeMarginExpandsExclusion` (`internal/layout/css_behavior_svg_paint_test.go:327`). A 10px margin makes a wider exclusion than 0.
- [x] Baseline (superseded by phases 1-3): `shape-inside` used to be dropped (`applyShapeProps` had no arm and there was no `ShapeInside` field), so a circle inside a wrapping div painted the same ops as no declaration. Proof of the pre-change baseline: `TestBehaviorShapeInsideNoLayoutEffect` (`internal/layout/css_behavior_list_shape_test.go:94`). That test now pins the no-op cases (auto, rejected tokens, circle() on an auto-height box); a definite content height does narrow.
- [x] Do not replan `shape-outside` or `shape-margin`. New code lives in `internal/layout/style_shape_inside.go` (storage) and `internal/layout/shape_inside.go` (contour); the only edits outside those are the `ResolvedStyle` fields and initials in `style.go`, the group registration in `style_cascade.go`, the regenerated `style_intern_gen.go`, the line hook in `inline.go`, and tests. `internal/layout/layout.go` did not grow. No path kind was added; the inside effect stays `OpText` lines. Proof: `go test ./internal/layout -count=1 -p 1 -parallel 1` (1151 pass).

**Gate:** the five baseline rows are closed by the tests named above. The file-boundary row closed with phases 1-3 (the named files hold the new code).

## Phase 1: decide, then store `shape-inside`

Catalog syntax: `auto | outside-shape | [ <basic-shape> || shape-box ] | <image> | display`.

- [x] Decision gate. Chrome has no BCD support (`testdata/css/catalog/properties.json` limitation on `shape-inside`, and `internal/layout/css_behavior_list_shape_test.go:92-93`). Choice (a): implement the subset below anyway. The catalog limitation text must still say Chrome ignores `shape-inside` (parent edits `testdata/css/catalog/properties.json`); this ledger makes no browser-parity claim.
- [x] How inside differs from outside. Outside shortens the lines next to a float (`internal/layout/float.go:164-168`). Inside shortens the lines of the element that declared it: `buildShapeInside` (`internal/layout/shape_inside.go`) builds its own contour and never calls `buildShapeExclusion` on the element's own float; a non-floated div has no float exclusion and still narrows. Proof: `TestBehaviorShapeInsideCircleTapersLines`, `TestBehaviorShapeInsideDoesNotInherit`.
- [x] Acceptance, first subset: `auto`, `circle()`, `ellipse()`, and `inset()`, plus an optional shape-box keyword from the set `shape-outside` already parses (`margin-box`, `border-box`, `padding-box`, `content-box` in `internal/layout/style_shape_props.go:18-24`). `auto` clears the inside shape back to `auto`/empty/false. Proof: `TestShapeInsideStoreAcceptsSubset`, `TestShapeInsideStoreOutsideShapeFlagAndAutoReset`.
- [x] `outside-shape` stores a flag that means "copy this element's `shape-outside` contour." It does not parse a second function. Proof: `TestShapeInsideStoreOutsideShapeFlagAndAutoReset`, `TestBehaviorShapeInsideOutsideShapeCopiesOutside`.
- [x] `polygon()` and `path()` parse as rejected, the same as `shape-outside` today (`internal/layout/style_shape_props.go:15`). A later row can add them. Proof: `TestShapeInsideStoreRejectedTokensKeepEarlierValue`, `TestBehaviorShapeInsideNoLayoutEffect`.
- [x] `<image>` parses as rejected in this phase (`url()` fails the function-name switch in `parseShapeInside`). `shape-image-threshold` owns the image follow-up. Proof: `TestShapeInsideStoreRejectedTokensKeepEarlierValue`, `TestBehaviorShapeInsideNoLayoutEffect`.
- [x] `display` parses as rejected in this phase (no function parens, so the shape split fails). It needs the element's border geometry as a shape; recorded here as the follow-up, not as a stored keyword that does nothing. Proof: `TestShapeInsideStoreRejectedTokensKeepEarlierValue`, `TestBehaviorShapeInsideNoLayoutEffect`.
- [x] Storage: `ShapeInside` string and a shape-box keyword on the resolved style, interned the same way `ShapeOutside` is (`internal/layout/style_intern_gen.go:772`). Regenerated with `go generate ./` from `internal/layout` (the `//go:generate` directive in `style.go:2`); the generated equality, fingerprint, and field list cover `ShapeInside`, `ShapeInsideBox`, `ShapeInsideOutside`, `ShapePadding`, and `ShapePaddingPercent`. The fields live on `ResolvedStyle` in `style.go` because that struct has a single declaration site. Proof: `TestStyleInternGeneratedFieldsComplete`, `TestShapeInsideStoreInitials`.
- [x] A bad token does not replace an earlier `circle()`: `circle(10px at 20% 30%) padding-box` survives `polygon()`, `path()`, `url()`, `display`, `none`, a bare shape-box keyword, and junk. Proof: `TestShapeInsideStoreRejectedTokensKeepEarlierValue`.

**Gate:** store tests only, and they pass (`TestShapeInsideStore*`). Phases 1-3 landed in one change, so the phase 2 layout behavior below is also closed on tests.

## Phase 2: narrow the element's own lines

New file `internal/layout/shape_inside.go`.

- [x] Build an inside contour with the circle, ellipse, and inset math already in `internal/layout/shape_exclusion.go`. Pass the element's content box as the reference, not the float margin box. Named `buildShapeInside` (`internal/layout/shape_inside.go`) so it cannot be confused with `buildShapeExclusion`. Proof: `TestBehaviorShapeInsideCircleTapersLines`.
- [x] Shape-box: the first consumer uses the content box when the keyword is missing. `content-box`, `padding-box`, `border-box`, and `margin-box` change that reference rectangle. This is stricter than `shape-outside`, which stores the keyword and ignores it (`internal/layout/style_shape_props.go:47-48`); the outside behavior is untouched. Proof: `TestBehaviorShapeInsideShapeBoxMovesReference` (four distinct first-line x values with margin, border, and padding set).
- [x] `outside-shape` on an element that also has a supported `shape-outside` uses that contour, flipped to an inside inset of the same element. On an element with no `shape-outside`, `outside-shape` is a no-op. Proof: `TestBehaviorShapeInsideOutsideShapeCopiesOutside`.
- [x] Line breaking for that element's own inline content uses the inside interval the way `lineBounds` uses the outside interval (`internal/layout/inline.go:609`). The hook is the element's content width at each line Y, before float exclusion runs: `lineBounds` narrows by the inside span first, then intersects an active float interval, and the float exclusion leaves the span untouched when no float intrudes at that Y. Floats beside the element still apply after the inside inset. Proof: `TestBehaviorShapeInsideStacksWithFloatExclusion`.
- [x] `shape-inside: auto` matches today's full content width. The opened no-effect test stays true for `auto` and for a rejected token (and for `circle()` on an auto-height box: no definite content height, so the contour is nil). Proof: `TestBehaviorShapeInsideNoLayoutEffect`.
- [x] A `circle()` on a wide paragraph moves at least one text op's X or width versus `auto`. Children do not inherit the contour. Proof: `TestBehaviorShapeInsideCircleTapersLines`, `TestBehaviorShapeInsideDoesNotInherit`.
- [x] `display` and image values still paint like `auto`. Proof: `TestBehaviorShapeInsideNoLayoutEffect`.

**Gate:** one behavior test where a circle makes a middle line shorter than the first line. `shape-outside` tests stay green.

## Phase 3: `shape-padding`

Catalog syntax: a non-negative length or percentage. Catalog limitation: no Chrome BCD support.

- [x] Decision gate. Same Chrome fact as `shape-inside`. Phase 1 chose to implement inside shapes, so this phase proceeded. `shape-padding` without `shape-inside` does nothing, including on a `shape-outside` float. It is not a second `shape-margin`. Proof: `TestBehaviorShapeInsidePaddingAloneNoEffect`.
- [x] Acceptance: a length, or a percentage. Negative values do not replace an earlier padding. Percentage reference is the same diagonal `resolveShapeMargin` uses (`internal/layout/shape_exclusion.go:139-144`); 5% of the 150x150 reference resolves to the same geometry as 10px. Proof: `TestShapePaddingStoreLengthPercentAndReject`, `TestBehaviorShapeInsidePaddingShrinksMiddleLine`.
- [x] Storage: `ShapePadding` length and percent on the resolved style; the parser and shrink helpers live in `internal/layout/style_shape_inside.go` and `internal/layout/shape_inside.go`, the fields on `ResolvedStyle` in `style.go` (single struct declaration site), interned with the other shape fields.
- [x] Consumer: the contour shrinks by the padding before `buildShapeInside` returns it. `0` matches phase 2. Padding larger than the shape yields an empty interval and the line stays at its Y with width 0 (no page break invented). Proof: `TestBehaviorShapeInsidePaddingShrinksMiddleLine` (0px paints like no declaration), `TestBehaviorShapeInsideOversizePaddingKeepsZeroWidthLines`.
- [x] Behavior test: `shape-inside: circle()` plus `shape-padding: 10px` makes a shorter middle line than `shape-padding: 0px` (the literal form and the `circle(90px at 50% 0px)` form are both tested). A percentage paints identically to the equal-pixel length. `shape-padding` alone does not move ops. Proof: `TestBehaviorShapeInsidePaddingShrinksMiddleLine`, `TestBehaviorShapeInsidePaddingAloneNoEffect`.

**Gate:** depends on phase 2. Does not change `shape-margin` numbers.

## Phase 4: `shape-image-threshold`

Catalog syntax `<'opacity'>`. Catalog limitation: no alpha-contour extraction from float images.

- [x] Decision gate. `shape-outside` rejects `url()` (`internal/layout/style_shape_props.go:15`). There is no image shape to threshold. Choice: keep rejecting image shapes here; `shape-image-threshold` stays planned, not done. The acceptance, storage, consumer, and behavior rows below stay open. This ledger does not sneak `url()` into `shape-outside` under a threshold row.
- [ ] Acceptance, only if the decision adds image shapes: a number from 0 to 1, same parser as opacity. The initial value used by the contour is 0, matching the usual CSS initial, unless the spec row we copy says otherwise. Write the chosen initial into the test.
- [ ] Storage: the threshold on the resolved style even if the consumer is still gated. A stored-but-unused value is not catalog `implemented`.
- [ ] Consumer: when a float's `shape-outside` is a raster image, the exclusion interval at each Y includes pixels whose alpha is at or above the threshold. The scan reads the image's own bytes. It does not rasterize the page.
- [ ] `shape-inside` image values stay rejected until the outside image path exists. Then the same scan can feed `buildShapeInside`.
- [ ] Behavior test: a small PNG with a transparent half and an opaque half produces a different exclusion interval at threshold 0 and at threshold 1. No page PNG.

**Gate:** open until the decision row is filled in. Default implementation choice in this file is "do not add `url()` shapes here." The property stays planned, not done.

## Phase 5: `float-defer`

Catalog syntax `<integer> | last | none`. Source spec: CSS Page Floats, `https://drafts.csswg.org/css-page-floats-3/#propdef-float-defer`. Catalog limitation: no page-float defer model.

Floats today pack beside each other or stack under the previous float (`internal/layout/layout_flow.go:1151-1174`). Nothing defers a float to a later page or a later column.

- [ ] Decision gate. `none` is the initial and matches `placeFloat` today. An integer or `last` needs a fragmentation point the float can move to. This tree's display list is not a page-float model. Choice: stop at `none`; integer and `last` stay unsupported, the acceptance, storage, and consumer rows below stay open, and no page boxes are invented. (Gate text keeps this row open even with the choice recorded.)
- [ ] Acceptance: `none`, a non-negative integer, and `last`. A negative integer does not replace an earlier value.
- [ ] Storage: the keyword or the integer on the resolved style, in `internal/layout/style_shape_inside.go` only if that does not mix two jobs. Prefer `internal/layout/style_float_defer.go` so shape storage stays shape storage.
- [ ] Consumer: if the decision is line defer, `placeFloat` (`internal/layout/layout_flow.go:1154`) skips N line heights before packing. `last` means "after the other floats in this block", which is the stack-below path that already exists when horizontal room runs out. Do not move the float onto a later display-list page.
- [ ] Behavior test: `float-defer: none` matches today's float Y. `float-defer: 1` places the float lower than `none` by about one line. `last` places it at or below a previous float in the same block. A bad token keeps `none`.

**Gate:** the decision row stays open. No page model is added under this property.

## Phase 6: catalog and closure

- [x] Catalog: leave `shape-outside` and `shape-margin` implemented. Flip `shape-inside` and `shape-padding` only after their behavior tests land. Leave `shape-image-threshold` and `float-defer` unsupported if their decision gates chose not to build the consumer. Proof: both rows are `implemented`; the two gate-decided rows stay `unsupported`; `make catalog-check` exits 0.
- [x] Limitations text for `shape-inside` must still say Chrome has no BCD support, even if the status becomes implemented. The limitation keeps the no-BCD sentence and adds the definite-content-height lite rule.
- [x] Regenerate the matrix property section. Proof: `make matrix-check` exits 0 on the merged branch.
- [x] Do not edit `documentation/deferred.md` unless the user asks. The wave-2 doc closure updated the v0.2.7 leftovers row as part of the requested catalog/matrix/docs work; this checklist did not touch it during the implementation phases.
- [x] Do not grow `internal/layout/layout.go`. It stays at 2157 lines, the single allowlisted file.
- [x] Run `make test`. Do not run bare `go test ./...`.
- [x] Run `make golden`.
- [x] Run `make lint`.
- [x] Run `make claim-scan`. Do not claim Chrome parity for `shape-inside`. The claim scan is clean and no parity claim exists.
- [x] Run `make catalog-check` and `make matrix-check`.
- [x] GitHub CI runs only `make test` and `make lint` (`.github/workflows/ci.yml:1`, `.github/workflows/ci.yml:20`, `.github/workflows/ci.yml:26`).

Focused run during the work:

```sh
go test ./internal/layout -run 'TestBehaviorShape|TestBehaviorShapeInside' -count=1
```

**Gate:** open until those commands exit 0 on the change that flips catalog rows.

## Dependencies

- Outside parse and storage: `internal/layout/style_shape_props.go`.
- Outside geometry: `internal/layout/shape_exclusion.go`.
- Outside line hook: `lineBounds` in `internal/layout/inline.go` and `floatState.exclusion` in `internal/layout/float.go`.
- Float placement: `placeFloat` in `internal/layout/layout_flow.go`.
- Inside code is new: `internal/layout/style_shape_inside.go` and `internal/layout/shape_inside.go`.
- `shape-rendering` is not a dependency. It is the SVG checklist.
- Style interning must include any new field or two equal styles will not compare equal (`internal/layout/style_intern_gen.go` hashes `ShapeOutside` today).
