# v0.1.2 - Draft borders, shapes, and parked leftovers

> **Parent:** `plans/v0.1.2/README.md`
> **Status:** parked. 13 of the 18 property rows are [~]; border-shape, border-boundary, text-fit, shape-image-threshold and float-defer are [x]. No opened test proves a clip, a shape path, a fitted text size, a threshold contour, a deferred float, or a column rule on the content edges.
> **Estimated effort:** eighteen parked names, each waiting on one behavior test before the catalog can leave `unsupported`.

---

## Overview

The engine returns a drawing list (`layout.DisplayList`). A solid square border is one straight line (`OpLine`) per side. `border-radius` already rounds that rectangle: the fill and the stroke keep radius fields. The fourteen draft names in phases 1 through 4 have no handler and do not change those ops.

`documentation/deferred.md:117` leaves those fourteen names, plus `text-fit`, `shape-image-threshold`, and `float-defer`, unsupported on purpose. Matrix section 5.5 records the fourteen as draft-not-ready (`documentation/compatibility-matrix.md:1148-1172`). `shape-inside` and `shape-padding` already paint a lite inside contour (`internal/layout/style_shape_inside.go`, `internal/layout/shape_inside.go`). This file does not replan them. `corner-shape` already has a path op for a non-round corner (`layout/displaylist.go:153-156`) and is outside this file.

`column-rule-visibility-items` is the column-rule name the row-rules ledger left open. The ten column-rule names that ledger closed (break and the nine insets) stay closed.

A search of `*.go` found no handler for the fourteen border names or for `column-rule-visibility-items`. `applyStyleProp` walks `styleGroups` and, when every group returns false, calls `applyIgnoredGroup`, which discards the name and the value (`internal/layout/style_cascade.go:1974-1980`, `internal/layout/style_properties.go:1876-1881`).

New code for any row below goes in a same-package file. Do not grow `internal/layout/layout.go`.

## Executive summary

| Family | Names | Catalog status | This file |
|---|---|---|---|
| Border clips | 11 | `unsupported`, handlers and tests empty | [~] |
| `border-limit` | 1 | `unsupported` | [~] |
| `border-shape` | 1 | `implemented` | [x] |
| `border-boundary` | 1 | `implemented` | [x] |
| `text-fit` | 1 | `implemented` | [x] |
| `shape-image-threshold` | 1 | `implemented` | [x] |
| `float-defer` | 1 | `implemented` | [x] |
| `column-rule-visibility-items` | 1 | `unsupported` | [~] |
| `shape-inside`, `shape-padding` | 2 | `implemented` | Not replanned |

## Phase 1: border clips

Eleven names. Catalog syntax for each physical and logical longhand is `none | [ <length-percentage [0,∞]> | <flex> ]+`. The three shorthands (`border-clip`, `border-inline-clip`, `border-block-clip`) are `<'border-top-clip'>`, one value. Each catalog row is `unsupported`, with an empty handler list and an empty test list. The limitation says there is no apply handler and the declaration is ignored. Matrix section 5.5 says the same (`documentation/compatibility-matrix.md:1159-1172`).

CSS Borders 4 section 5 (partial borders) is marked not ready for implementation. `none` leaves the edge whole. A list splits the edge into parts: the first part is painted, the second is skipped, the third is painted, and so on. Percentages use the length of that border edge. When the listed parts are shorter than the edge and the list has no flexible length (`fr`, a share of whatever edge is left), the draft treats the list as if `1fr` were added at the end, so the remainder is painted.

This engine draws a horizontal edge from the left and a vertical edge from the top (`internal/layout/layout_chrome.go:314-318`, `:334-335`, `:351-352`, `:368-369`). A solid style is one segment (`internal/layout/layout_chrome.go:45-46`). The first clip part starts at that same end.

Logical sides use the map already used for logical radii: block-start is the top, block-end is the bottom, inline-start is the left, inline-end is the right (`internal/layout/style_logical_border.go:6-12`; the side-radius test states the same map at `internal/layout/style_logical_border_test.go:36`).

### 1.1 border-clip

Catalog: `unsupported` at `testdata/css/catalog/properties.json:3803` (status `:3814`). Syntax `<'border-top-clip'>`. The draft shorthand sets all four sides to that one value.

Current behavior: no `*.go` match. `applyIgnoredGroup` drops it (`internal/layout/style_cascade.go:1974-1980`, `internal/layout/style_properties.go:1876-1881`). A solid border stays one `OpLine` per side (`internal/layout/layout_chrome.go:45-46`).

If the defer is lifted: `box-sizing: border-box; width: 100px; height: 40px; border: 4px solid #c00; border-clip: 20px 10px`. Each border edge is the box edge (100px across, 40px down). Top and bottom: a 20px line, a 10px gap, then a 70px line. Left and right: a 20px line, a 10px gap, then a 10px line. `border-clip: none` keeps one segment per side.

- [~] Reason: Borders 4 section 5 is marked not ready for implementation, and no paint test shows the gap. Owner: this ledger. Next gate: a behavior test on that 100px by 40px box with two segments and a gap on every side. Leave the catalog `unsupported` until that test exists. Do not grow `internal/layout/layout.go`.

### 1.2 border-top-clip

Catalog: `unsupported` at `testdata/css/catalog/properties.json:4974` (status `:4985`). Syntax `none | [ <length-percentage [0,∞]> | <flex> ]+`.

Current behavior: no `*.go` match. `applyIgnoredGroup` drops it. The top edge stays one solid `OpLine` from the left (`internal/layout/layout_chrome.go:314-318`, `:45-46`).

If the defer is lifted: the same 100px by 40px box with `border-top-clip: 20px 10px` and a 4px solid border. The top edge is a 20px line, a 10px gap, and a 70px line. The right, bottom, and left edges stay one segment each.

- [~] Reason: same draft defer as `border-clip`, and no top-edge gap test. Owner: this ledger. Next gate: the top-edge half of the 1.1 test. Do not grow `internal/layout/layout.go`.

### 1.3 border-right-clip

Catalog: `unsupported` at `testdata/css/catalog/properties.json:4709` (status `:4720`). Same longhand syntax as `border-top-clip`.

Current behavior: no `*.go` match. `applyIgnoredGroup` drops it. The right edge stays one solid `OpLine` from the top (`internal/layout/layout_chrome.go:334-335`).

If the defer is lifted: `border-right-clip: 20px 10px` on the 40px-tall box. The right edge is a 20px line from the top, a 10px gap, and a 10px line. The other three edges stay one segment each.

- [~] Reason: same draft defer, and no right-edge gap test. Owner: this ledger. Next gate: the right-edge half of the 1.1 test. Do not grow `internal/layout/layout.go`.

### 1.4 border-bottom-clip

Catalog: `unsupported` at `testdata/css/catalog/properties.json:3622` (status `:3633`). Same longhand syntax as `border-top-clip`.

Current behavior: no `*.go` match. `applyIgnoredGroup` drops it. The bottom edge stays one solid `OpLine` from the left (`internal/layout/layout_chrome.go:351-352`).

If the defer is lifted: `border-bottom-clip: 20px 10px` on the 100px-wide box. The bottom edge is a 20px line, a 10px gap, and a 70px line. The other three edges stay one segment each.

- [~] Reason: same draft defer, and no bottom-edge gap test. Owner: this ledger. Next gate: the bottom-edge half of the 1.1 test. Do not grow `internal/layout/layout.go`.

### 1.5 border-left-clip

Catalog: `unsupported` at `testdata/css/catalog/properties.json:4517` (status `:4528`). Same longhand syntax as `border-top-clip`.

Current behavior: no `*.go` match. `applyIgnoredGroup` drops it. The left edge stays one solid `OpLine` from the top (`internal/layout/layout_chrome.go:368-369`).

If the defer is lifted: `border-left-clip: 20px 10px` on the 40px-tall box. The left edge is a 20px line from the top, a 10px gap, and a 10px line. The other three edges stay one segment each.

- [~] Reason: same draft defer, and no left-edge gap test. Owner: this ledger. Next gate: the left-edge half of the 1.1 test. Do not grow `internal/layout/layout.go`.

### 1.6 border-inline-clip

Catalog: `unsupported` at `testdata/css/catalog/properties.json:4138` (status `:4149`). Syntax `<'border-top-clip'>`. The draft shorthand sets inline-start and inline-end to that one value.

Current behavior: no `*.go` match. `applyIgnoredGroup` drops it. Left and right stay one segment each.

If the defer is lifted: `border-inline-clip: 20px 10px` on the 40px-tall box, horizontal-tb and left-to-right, so both vertical edges split: 20px, a 10px gap, 10px. Top and bottom stay one segment each.

- [~] Reason: same draft defer, and no inline-edge gap test. Owner: this ledger. Next gate: a test that both vertical edges split and the horizontal edges do not. Do not grow `internal/layout/layout.go`.

### 1.7 border-inline-start-clip

Catalog: `unsupported` at `testdata/css/catalog/properties.json:4337` (status `:4348`). Same longhand syntax as `border-top-clip`.

Current behavior: no `*.go` match. `applyIgnoredGroup` drops it. Under the logical map in the phase intro, the left edge stays one segment.

If the defer is lifted: `border-inline-start-clip: 20px 10px` splits only the left edge (20px, 10px gap, 10px). The right edge stays one segment.

- [~] Reason: same draft defer, and no inline-start gap test. Owner: this ledger. Next gate: the left-edge test from 1.5, declared with this logical name. Do not grow `internal/layout/layout.go`.

### 1.8 border-inline-end-clip

Catalog: `unsupported` at `testdata/css/catalog/properties.json:4200` (status `:4211`). Same longhand syntax as `border-top-clip`.

Current behavior: no `*.go` match. `applyIgnoredGroup` drops it. Under the logical map, the right edge stays one segment.

If the defer is lifted: `border-inline-end-clip: 20px 10px` splits only the right edge (20px, 10px gap, 10px). The left edge stays one segment.

- [~] Reason: same draft defer, and no inline-end gap test. Owner: this ledger. Next gate: the right-edge test from 1.3, declared with this logical name. Do not grow `internal/layout/layout.go`.

### 1.9 border-block-clip

Catalog: `unsupported` at `testdata/css/catalog/properties.json:3243` (status `:3254`). Syntax `<'border-top-clip'>`. The draft shorthand sets block-start and block-end to that one value.

Current behavior: no `*.go` match. `applyIgnoredGroup` drops it. Top and bottom stay one segment each.

If the defer is lifted: `border-block-clip: 20px 10px` on the 100px-wide box splits both horizontal edges (20px, 10px gap, 70px). Left and right stay one segment each.

- [~] Reason: same draft defer, and no block-edge gap test. Owner: this ledger. Next gate: a test that both horizontal edges split and the vertical edges do not. Do not grow `internal/layout/layout.go`.

### 1.10 border-block-start-clip

Catalog: `unsupported` at `testdata/css/catalog/properties.json:3438` (status `:3449`). Same longhand syntax as `border-top-clip`.

Current behavior: no `*.go` match. `applyIgnoredGroup` drops it. Under the logical map, the top edge stays one segment.

If the defer is lifted: `border-block-start-clip: 20px 10px` splits only the top edge (20px, 10px gap, 70px). The bottom edge stays one segment.

- [~] Reason: same draft defer, and no block-start gap test. Owner: this ledger. Next gate: the top-edge test from 1.2, declared with this logical name. Do not grow `internal/layout/layout.go`.

### 1.11 border-block-end-clip

Catalog: `unsupported` at `testdata/css/catalog/properties.json:3305` (status `:3316`). Same longhand syntax as `border-top-clip`.

Current behavior: no `*.go` match. `applyIgnoredGroup` drops it. Under the logical map, the bottom edge stays one segment.

If the defer is lifted: `border-block-end-clip: 20px 10px` splits only the bottom edge (20px, 10px gap, 70px). The top edge stays one segment.

- [~] Reason: same draft defer, and no block-end gap test. Owner: this ledger. Next gate: the bottom-edge test from 1.4, declared with this logical name. Do not grow `internal/layout/layout.go`.

## Phase 2: border limit

### 2.1 border-limit

Catalog: `unsupported` at `testdata/css/catalog/properties.json:4632` (status `:4643`). Syntax `all | [ sides | corners ] <length-percentage [0,∞]>? | [ top | right | bottom | left ] <length-percentage [0,∞]>`. Empty handlers and tests. Matrix section 5.5 marks it draft-not-ready (`documentation/compatibility-matrix.md:1169`). The name lives in the same Borders 4 section 5 that is marked not ready for implementation.

Current behavior: no `*.go` match. `applyIgnoredGroup` drops it (`internal/layout/style_cascade.go:1974-1980`, `internal/layout/style_properties.go:1876-1881`). A solid top border is one full-width `OpLine` (`internal/layout/layout_chrome.go:45-46`, `:314-318`).

If the defer is lifted, using the draft's own examples: `border-top: solid; border-limit: left 4em` paints a top `OpLine` whose width is 4em, starting at the left end this engine already uses. The rest of that top edge is absent. `border-limit: sides 50%` paints the middle half of each side, measured from the center of the side, and stops before the corner region the draft takes from the border radii. `border-limit: all` is the initial and matches today's full-edge line.

- [~] Reason: section 5 is marked not ready for implementation, and no partial-border test exists. Owner: this ledger. Next gate: a behavior test where `border-limit: left 4em` shortens the top line to 4em and `border-limit: all` keeps the full edge. Do not grow `internal/layout/layout.go`.

## Phase 3: border shape

### 3.1 border-shape

Catalog: `unsupported` at `testdata/css/catalog/properties.json:4824` (status `:4835`). Syntax `none | [ <basic-shape> <geometry-box>?]{1,2}`. Empty handlers and tests. The limitation says there is no apply handler. Matrix section 5.5 says Chrome has painted `border-shape` since version 147, and this tree still has no reader (`documentation/compatibility-matrix.md:1171`).

Current behavior: no `*.go` match. `applyIgnoredGroup` drops it. `border-radius` already paints. On a wide short box, `border-radius: 999px` clamps to half the height and becomes a pill (`internal/layout/border_radius.go:229`). The ops are a fill rect and a stroke rect with per-corner radii (`internal/layout/layout_chrome.go:834-838`). A round box keeps those rect ops. `OpPath` is the corner-shape outline, and a round box does not use it (`layout/displaylist.go:153-156`).

`border-radius` rounds the four corners of the rectangle. The ops stay `OpFillRect` and `OpStrokeRect` with `Radius` fields. `border-shape` sets the outline to a basic-shape path. One shape is stroke mode: the border is a stroke along that path, and the width is the computed border width of the relevant side (CSS Borders 4 section 7.2). Two shapes paint the band between the outer path and the inner path.

If the defer is lifted: `width: 100px; height: 60px; border: 4px solid #00f; border-shape: ellipse(50% 50%)`. The blue stroke follows the ellipse. The straight top and bottom of the pill that `border-radius: 50%` paints on this same box are absent. The ops are path commands for the ellipse. `border-shape: ellipse(50% 50%) ellipse(40% 40%)` paints the band between the two ellipses. `border-shape: none` leaves the radius pill on the rect ops.

- [x] Done. One circle, ellipse or inset shape paints one fill path and one border path (`internal/layout/border_shape_paint.go`, hooked in `prependChrome`). Proof: `TestBorderShapeEllipsePath` in `layout/declared_shapes_test.go`, which also checks `none` and the two-shape band keep the ordinary border. Not done: the two-shape band, polygon, path, xywh, rect and shape(), and non-border-box geometry boxes.

## Phase 4: round display boundary

### 4.1 border-boundary

Catalog: `unsupported` at `testdata/css/catalog/properties.json:3787` (status `:3798`). Syntax `none | parent | display`. Empty handlers and tests. Matrix section 5.5 marks it draft-not-ready under css-round-display-1 (`documentation/compatibility-matrix.md:1163`).

Current behavior: no `*.go` match. `applyIgnoredGroup` drops it. The border stays the rectangle from `appendBorderOps` (`internal/layout/layout_chrome.go:214-218`).

The draft draws extra border where the element meets a boundary, inward from that boundary, and does not move the box. `parent` uses the parent's border edge. `display` uses the screen edge. `none` is the initial.

If the defer is lifted: a child with `border: 5px solid red; border-boundary: parent` inside a parent with `border-radius: 50%`, sized so the child meets that curve. The child's border ops gain segments along the parent's rounded edge where the two areas meet. The child's box x and y stay the position layout already computed. `border-boundary: none` keeps the four rectangular lines. `display` needs a screen outline. This tree stores no screen shape, so `display` keeps the rectangular lines until a viewport outline exists.

- [x] Done for `parent`. `addBorderBoundary` in `internal/layout/border_boundary.go` strokes the parent's rounded corner inside the child's box. Proof: `TestBorderBoundaryParentTracesCurve` in `layout/declared_shapes_test.go`. Not done: `display` (no screen outline is stored) and non-uniform child borders.

## Phase 5: parked leftovers

`shape-inside` and `shape-padding` are implemented. Catalog status `implemented` (`testdata/css/catalog/properties.json:16905` and `:16986`). Handlers are `applyShapeInsideProps` and `parseShapeInside`, plus `parseShapePadding` for the padding. Behavior tests include `TestBehaviorShapeInsideCircleTapersLines` and `TestBehaviorShapeInsidePaddingShrinksMiddleLine`. The shapes ledger phases 1 through 3 closed them (`plans/v0.0.3/shapes/shapes-checklist.md`). This phase does not reopen those rows.

### 5.1 text-fit

Catalog: `unsupported` at `testdata/css/catalog/properties.json:18400` (status `:18420`), with a handler listed. The values field is `[ none | grow | shrink ] [consistent | per-line | per-line-all]? <percentage>?`. The limitation (`:18405-18407`) says the value is parsed and stored (`TextFit`, `style_text_spacing_props.go:36`) and no scale search reads it. A scale search tries larger or smaller sizes until the text fits the line. The missing behavior is grow/shrink scale-to-fit. `TestTextSpacingTrimApply` asserts the stored string only.

Current behavior: `applyTextAutospaceProps` stores the value when `isTextFitValue` accepts it (`internal/layout/style_text_spacing_props.go:33-37`). That check accepts `none`, `auto`, and `scale` (`:96-101`). `grow` and `shrink` fail the check, so they are not stored. The initial is `none` (`internal/layout/style.go:823`). The field comment lists the same three strings (`internal/layout/style.go:527`). `TestTextSpacingTrimApply` writes `scale` and checks the string (`internal/layout/text_box_spacing_hyphen_test.go:79-84`). A search of `*.go` for `TextFit` finds the field, the intern hash, the applier, and that test. No line breaker reads it.

The scale-search the catalog limitation names: `grow` tries larger scales until the text fills the inline size of the box. `shrink` tries smaller scales until the text fits. `consistent` uses one scale for every line. `per-line` gives each line its own scale and leaves the last line, and any line that ends in a forced break, at the declared size. `per-line-all` scales those lines too. The optional percentage caps the scale.

If the defer is lifted: a heading with `width: 200px; font-size: 12px; text-fit: grow consistent`, whose unscaled text is narrower than 200px. The `OpText` run has a larger `Size` than the same markup with `text-fit: none`, and the run's width meets the content box, or the percentage cap when one is set. The used width of the box stays 200px. Today `text-fit: scale` stores the string and the `OpText` size stays 12px. `text-fit: grow` does not store, so the run stays 12px too.

- [x] Done for one line of plain text. `applyTextFit` in `internal/layout/text_fit.go` scales the font to the content width: grow enlarges, shrink reduces, a percentage caps the scale. Proof: `TestTextFitScalesOneLine` in `layout/declared_shapes_test.go`. Not done: blocks with elements or several runs, and the difference between the consistent and per-line modes.

### 5.2 shape-image-threshold

Catalog: `unsupported` at `testdata/css/catalog/properties.json:16846` (status `:16857`). Syntax `<opacity-value>`. The limitation says no image shape is parsed, `shape-outside` rejects `url()`, and the shapes ledger decided not to add `url()` shapes here.

Current behavior: `applyShapeProps` owns `shape-outside` and `shape-margin` and returns false for every other name (`internal/layout/style_shape_props.go:27-44`). The file comment says `shape-image-threshold` has no handler (`:15-17`). `parseShapeOutside` accepts `circle`, `ellipse`, and `inset`; any other function name, including `url()`, returns false (`:73-81`). A search of `internal/layout/shape_exclusion.go` found no alpha read. A left float's contour, when one exists, moves the following line through `rightEdgeAt` (`internal/layout/float.go:176-179`). With `url()` rejected, that contour is absent and the following text starts at the rectangular float edge.

The shapes checklist phase 4 records the same choice: keep rejecting image shapes, and leave this property planned (`plans/v0.0.3/shapes/shapes-checklist.md:87-97`).

If the defer is lifted: a left float whose image is transparent on the left half and opaque on the right, with `shape-outside: url(...)`. `shape-image-threshold: 0` counts faint pixels as solid, so `rightEdgeAt` moves left and the following `OpText` starts further left. `shape-image-threshold: 1` counts only fully opaque pixels, so the following text stays at the opaque edge. The float's own Y comes from `placeFloat` and does not move. The scan reads the image bytes. It does not rasterize a page.

- [x] Done for url() float shapes. `buildShapeImageMask` in `internal/layout/shape_image.go` reads the image alpha per scanline and keeps pixels above the threshold. Proof: `TestShapeImageThreshold` in `layout/shape_image_threshold_test.go` (text starts at 60pt for threshold 0 and 30pt for 0.5). Not done: shape-margin growth up and down, and gradient or other image sources.

### 5.3 float-defer

Catalog: `unsupported` at `testdata/css/catalog/properties.json:8986` (status `:8997`). Syntax `<integer> | last | none`. The limitation says no page-float model exists, and the shapes ledger chose to support only `none`. Integer and `last` stay unsupported.

Current behavior: `applyFloatPageProps` handles `float-offset`, `float-reference`, and the footnote longhands, and returns false otherwise (`internal/layout/style_float_page_props.go:43-64`). The file comment says `float-defer` has no handler and no defer model (`:19`). `placeFloat` packs a left or right float beside floats already on the line, or stacks it below when that pack overflows (`internal/layout/layout_flow.go:1354-1384`). Nothing skips a line for a defer count.

The shapes checklist phase 5 records the same choice and the float result a later wave would test (`plans/v0.0.3/shapes/shapes-checklist.md:100-112`).

If the defer is lifted: `float-defer: none` keeps today's float Y. `float-defer: 1` places the float lower than `none` by about one line, so the float's ops shift down and the line at the old Y is full width. `last` places the float at or below an earlier float in the same block. The drawing list stays one canvas.

- [x] Done for multicol columns. `deferFloatsToColumns` in `internal/layout/float_defer.go` moves a float N columns right, or to the last column. Proof: `TestFloatDeferMovesFloatToLaterColumn` in `layout/declared_shapes_test.go`. Not done: page deferral, since `Result.Pages` is never filled.

### 5.4 column-rule-visibility-items

Catalog: `unsupported` at `testdata/css/catalog/properties.json:6422` (status `:6433`). Syntax `all | around | between | normal`. Empty handlers and tests. The limitation says there is no apply handler.

Current behavior: no `*.go` match for this name. `applyColumnRuleProps` stores `column-rule`, `column-rule-break`, the width, style, and color longhands, and the inset longhands (`internal/layout/style_multicol_props.go:38-48`, `:57-70`, `:80-90`). It has no visibility case. `rule-visibility-items` expands only to `row-rule-visibility-items` (`internal/layout/style_gap_rules.go:147-148`). The row field is `RowRuleVisibility`. `around` and `all` also paint the content edges; `between` and `normal` do not (`internal/layout/gap_rules.go:186-193`). `TestBehaviorRowRuleVisibilityAround` expects one horizontal line for `between` and three for `around` (`internal/layout/css_behavior_row_rule_test.go:234-242`). Column paint only walks the gaps between columns: `emitColumnRules` loops `i < nCols-1` (`internal/layout/multicol.go:1010-1012`) and `paintGridColumnRules` loops `len(cols)-1` (`internal/layout/gap_rules.go:515`). A search found no `ColumnRuleVisibility` field.

The row-rules ledger closed the other ten `column-rule-*` names (break and the nine insets) in phase 6. Those stay closed (`plans/v0.0.3/row-rules/row-rules-checklist.md:30`, `:138`). Its open row for this name says the next gate is a column visibility wave with edge paint tests (`:47`). This file repeats that gate.

If the defer is lifted: two columns and `column-rule: 3px solid #c00; column-rule-visibility-items: between` paint one vertical `OpLine` in the gap, which is what `emitColumnRules` does today. `around` also paints a vertical line at the start of the content box and one at the end, inside the border, so the list has three vertical lines. `all` matches `around` for a single rule. `normal` follows the row reading in `rowRulePaintsAround`: the edge lines stay off. `rule-visibility-items` still writes the row half only, so the column wave has to store the column half before that shorthand can paint column edges. The shorthand is not one of the eighteen names here.

- [~] Reason: no column visibility field and no edge paint. Owner: this ledger. Next gate: a column visibility wave with edge paint tests, the gate already written on the row-rules ledger. `between` stays one gap line. `around` adds the two content edges. Do not reopen the ten closed `column-rule-*` names. Do not grow `internal/layout/layout.go`.

## Dependencies

- Drop path for a name with no group: `applyStyleProp` and `applyIgnoredGroup` (`internal/layout/style_cascade.go:1974-1980`, `internal/layout/style_properties.go:1876-1881`).
- Solid border lines: `appendBorderLineSegments` and `appendBorderOpsSides` (`internal/layout/layout_chrome.go:36-46`, `:245-373`).
- Radius paint already shipped: `internal/layout/border_radius.go`, `appendRoundedBorderOps` (`internal/layout/layout_chrome.go:818-838`).
- Corner-shape path op, out of scope: `layout/displaylist.go:153-156`.
- Text-fit storage: `internal/layout/style_text_spacing_props.go:33-37` and `:96-101`. The scale search is the catalog gap at `testdata/css/catalog/properties.json:18405-18407`.
- Outside shapes and the `url()` rejection: `internal/layout/style_shape_props.go:11-17` and `:73-81`. Float line shortening: `internal/layout/float.go:171-179`.
- Inside shapes, already shipped, not replanned: `internal/layout/style_shape_inside.go`, `internal/layout/shape_inside.go`, and `plans/v0.0.3/shapes/shapes-checklist.md` phases 1 through 3. Phase 4 keeps `shape-image-threshold` off. Phase 5 keeps `float-defer` at `none`.
- Float placement: `placeFloat` (`internal/layout/layout_flow.go:1340-1408`). Page-float handlers, with no `float-defer` case: `internal/layout/style_float_page_props.go:11-19` and `:43-64`.
- Column rules that already paint: `applyColumnRuleProps` (`internal/layout/style_multicol_props.go:35-94`), `emitColumnRules` (`internal/layout/multicol.go:978-1013`), `paintGridColumnRules` (`internal/layout/gap_rules.go:489-530`). Row visibility, the pattern to copy: `internal/layout/gap_rules.go:186-193`. The closed column-rule names stay in `plans/v0.0.3/row-rules/row-rules-checklist.md`.
- Catalog and matrix: `testdata/css/catalog/properties.json`, `documentation/compatibility-matrix.md` section 5.5, `documentation/deferred.md:117`.
- Output stays `layout.DisplayList`. Do not add a page rasterizer. Do not grow `internal/layout/layout.go`.
