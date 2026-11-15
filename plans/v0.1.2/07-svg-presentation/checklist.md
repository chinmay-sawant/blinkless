# v0.1.2 - SVG presentation leftovers

> **Parent:** `plans/v0.1.2/README.md`
> **Status:** in progress. Phase 1 is done (geometry, markers, stops). `paint-order` and `path-length` are done and emulated in `internal/svg/present.go`. Phases 2 to 5 are otherwise open.
> **Estimated effort:** Forty-five unsupported names: fourteen inline styles already move pixels and still need a stylesheet store, three are copied onto the SVG and then ignored when the PNG is drawn, five are not copied at all, twenty draft longhands must stay off `fill` and `stroke`, and three filter leftovers stay outside a filter graph.

---

## Overview

Blinkless returns a drawing list: the ordered paint operations for the page (filled rectangles, text runs, images, and the other kinds below). An inline `svg` becomes one image operation in that list. `buildInlineSVG` turns the subtree into XML and calls `svg.RasterizeContext` (`internal/layout/layout_svg.go:27-36`). When the call is painting and the PNG bytes are non-empty, it emits `OpImage` with those bytes (`internal/layout/layout_svg.go:54-60`). `rasterizeCanvas` encodes that one SVG through `encodeAndMemoize` (`internal/svg/raster.go:420`) and `encodeRasterPNG` (`internal/svg/raster.go:215-221`). The drawing-list kinds run from `OpUnknown` through `OpGridRun` (`internal/layout/layout.go:485-500`), plus `OpPath` for a corner-shape box (`internal/layout/corner_path.go:15-19`). Orientation or a crop re-encodes that one image as PNG (`internal/layout/layout_images.go:389-413`). This ledger does not add a page encoder.

A stylesheet rule is a rule in a `style` element or a linked sheet. The style attribute is the `style="..."` text on the element itself. Bake means copying a CSS value onto an SVG attribute before the PNG step. The PNG step reads attributes. It does not read the style attribute. The catalog is `testdata/css/catalog/properties.json`. A consumer is code that uses the value after it has been saved, such as the PNG step or a text position. Saving the value alone is not a consumer.

Before `canvas.ParseSVG`, `rasterizeCanvas` removes `text`, `tspan`, and `textPath` (`internal/svg/raster.go:384-386`, remover at `internal/svg/strip_text.go:15-20`). That removal is the strip. The comment on that call says `getFontFace` loads a system font while the lock `canvasMu` is held (`internal/svg/raster.go:60-62` and `internal/svg/raster.go:384-385`). Stripping the elements keeps that load from running. SVG glyphs disappear. `TestRasterizeMissingFontFamilyDoesNotPanic` covers a missing family (`internal/svg/strip_text_test.go:9-28`). `TestSvgTextTextAnchorMiddleAnchorsSvgText` requires the two image payloads to match (`internal/layout/css_behavior_svg_text_test.go:84-86`).

This file is the execution ledger for catalog names that are still `unsupported`. It supersedes the open rows in `plans/v0.0.3/svg-presentation/svg-presentation-checklist.md`. That file is left unchanged. Phase 0 records five implemented properties whose opened tests already assert the behavior. Do not schedule those five again.

The parent index is `plans/v0.1.2/README.md`.

Two paths exist, and they differ:

- Resolved style. After CSS is applied, each element has one computed style. `applySVGPresentationProps` (`internal/layout/style_paint_props.go:91`) stores `fill` and `stroke` there, and forwards `fill-rule`, `shape-rendering`, `dominant-baseline`, and `alignment-baseline` to `applyLeftoversProps` (`internal/layout/style_leftovers.go:43-50`). The serializer copies the stored fill, stroke, opacity, width, dash, cap, join, and miter limit onto attributes (`internal/layout/layout_svg.go:365-419`). A stylesheet rule reaches this path.
- Style-attribute bake. Geometry, stops, `paint-order`, `vector-effect`, `text-rendering`, `d`, and markers are read from the element's `style` attribute (`internal/layout/layout_svg.go:447-524`, `internal/layout/layout_svg.go:1760-1772`, `internal/layout/layout_svg.go:1819-1841`). A stylesheet rule does not reach this path. The catalog stays `unsupported` while the only consumer is that attribute read. The seed rule says `implemented` needs a consumer outside the style layer and a behavior test (`testdata/css/catalog/properties.json:9`).

The v0.0.3 ledger called `marker` a non-catalog shorthand and left `path-length` out. Both are catalog rows now (`testdata/css/catalog/properties.json:12032` and `testdata/css/catalog/properties.json:14503`).

Grouping checked against `internal/layout/layout_svg.go`:

- Phase 1 (done 2026-11-16), inline style already moves pixels: `cx`, `cy`, `r`, `rx`, `ry`, `x`, `y`, `d`, `marker`, `marker-end`, `marker-mid`, `marker-start`, `stop-color`, `stop-opacity`. `marker-mid` and `marker-start` are written by the same loop as `marker-end`. No opened test isolates those two longhands, so they have no `[x]` baseline.
- Phase 2, written into the XML and ignored by the rasterizer: `paint-order` (done: emulated in `internal/svg/present.go`), `vector-effect`, `text-rendering` (`internal/layout/layout_svg.go:506-524`).
- Phase 3, no bake arm: `path-length` (done: dashes rescaled in `internal/svg/present.go`), `image-rendering`, `glyph-orientation-vertical`, `baseline-shift`, `marker-side`. `path-length` was listed with the pixel bakes. The serializer never reads the name, and `TestBehaviorSvgDraftPaintLonghandsInert` requires the same painted-pixel count, so it sits here.
- Phase 4, draft fill and stroke longhands. They must not be copied onto `fill` or `stroke`.
- Phase 5, filter leftovers: `flood-color`, `flood-opacity`, `lighting-color`.

New storage and bake code goes in `internal/layout/svg_presentation.go` (package `layout`). `internal/layout/layout.go` is already on the size ledger at 2497 lines (`scripts/file-size-allowlist.txt:12`). Do not add arms there. Raster fixes for attributes the canvas parser drops go in a new `internal/svg/present.go`, called from `rasterizeCanvas` beside `emulateDroppedPresentation` (`internal/svg/raster.go:383`). That file does not exist yet.

Handler gap, used below, means the catalog limitation string "No apply handler in internal/layout; declaration is ignored (graceful degrade)" with pinned upstream revision `1f2ec8f74a80c14066b4c7d6822cee59f69fa03b`. For a phase 1 name the limitation names the missing handler. The style-attribute bake still changes pixels.

## Executive Summary

| Group | Names | What to do |
|---|---|---|
| Phase 0, catalog `implemented` | `fill`, `fill-rule`, `shape-rendering`, `alignment-baseline`, `dominant-baseline` | Baseline only. Tests already assert the behavior. |
| Phase 1, inline pixels, all `implemented` | `cx`, `cy`, `r`, `rx`, `ry`, `x`, `y`, `d`, `marker`, `marker-end`, `marker-mid`, `marker-start`, `stop-color`, `stop-opacity` | Done. A stylesheet value is stored and baked, and `marker-mid` and `marker-start` have their own tests. A shape with two or more markers is written once per marker, because the canvas misplaces all but the first. |
| Phase 2, copied into the XML, PNG ignores them | `paint-order` (done), `vector-effect`, `text-rendering` | Store and bake. Catalog flips only after a pixel diff. `paint-order` has one. `text-rendering` pixels stay deferred while text is removed. |
| Phase 3, no bake | `path-length` (done), `image-rendering`, `glyph-orientation-vertical`, `baseline-shift`, `marker-side` | Acceptance through a real consumer. SVG text pixels stay deferred while text is removed. |
| Phase 4, draft longhands | `fill-break`, `fill-color`, `fill-image`, `fill-origin`, `fill-position`, `fill-repeat`, `fill-size`, `stroke-align`, `stroke-alignment`, `stroke-break`, `stroke-color`, `stroke-dash-corner`, `stroke-dash-justify`, `stroke-dashadjust`, `stroke-dashcorner`, `stroke-image`, `stroke-origin`, `stroke-position`, `stroke-repeat`, `stroke-size` | Do not alias onto `fill` or `stroke`. |
| Phase 5, filter leftovers | `flood-color`, `flood-opacity`, `lighting-color` | One element each, or a serialize gap. No filter graph. |

## Phase 0: what to expect from the implemented paint properties

These five are catalog `implemented`. The rows are `[x]` because the opened tests assert the behavior. Do not reimplement them. Do not flip the catalog rows.

The behavior tests name Chrome 143.0.7499.40 as their reference browser. The expectations below are what those tests require in this tree.

### fill

Catalog row `fill` (`testdata/css/catalog/properties.json:8501`) is `implemented`. Syntax `<paint>`. Handler `applySVGPresentationProps`. Listed test `TestBehaviorFillBakesPaint`. Limitations empty.

`applySVGPresentationProps` stores a parsed color on `Fill` and sets `FillSet`. The token `none` stores a zero color, `FillSet`, and `FillOpacity` 0 (`internal/layout/style_paint_props.go:93-101`). The serializer writes `fill="#rrggbb"` when `FillSet` is set, and `fill="none"` when the color is zero and the opacity is 0 (`internal/layout/layout_svg.go:370-375`). An authored `fill` attribute wins because the bake checks `written.has("fill")` (`internal/layout/layout_svg.go:370`).

What to expect. A 64 by 28 svg, viewBox `0 0 64 28`, holds `<rect x="2" y="2" width="60" height="24" style="fill:#ff0000"/>` (`internal/layout/css_behavior_svg_paint_test.go:43-46`). The rect's used `Fill` is `(1, 0, 0)` and `FillSet` is true. The serialized document contains `fill="#ff0000"`. Layout emits an `OpImage`. That is `TestBehaviorFillBakesPaint` (`internal/layout/css_behavior_svg_paint_test.go:208-228`).

A second opened test covers `none` through the restyle API. `TestRestyleSVGPaintFillNoneKeepsOwnChildPaint` sets the root fill to `none`. The child rect that inherits that fill has alpha 0 at pixel `(20, 15)`. The child that carries its own `fill:#0000ff` stays blue (`internal/layout/svg_paint_test.go:342-379`).

- [x] `TestBehaviorFillBakesPaint` stores red, bakes `fill="#ff0000"`, and emits an image op (`internal/layout/css_behavior_svg_paint_test.go:211-228`).
- [x] `TestRestyleSVGPaintFillNoneKeepsOwnChildPaint` paints the inherited `none` rect with alpha 0 and leaves the child's own blue fill (`internal/layout/svg_paint_test.go:373-379`).

### fill-rule

Catalog row `fill-rule` (`testdata/css/catalog/properties.json:8643`) is `implemented`. Syntax `nonzero | evenodd`. Handler `applySVGPresentationProps`. Listed tests: `TestBehaviorFillRuleNoPaintEffect`, `TestBehaviorSvgBakeFillRuleGap`, `TestPaintOutputFillRuleEvenOddHolesRaster`. Limitations empty.

The initial value is `nonzero` (`internal/layout/style.go:827`). `applyFillRuleProperty` stores `nonzero` or `evenodd` and leaves any other token unchanged (`internal/layout/style_leftovers.go:274-278`). A missing declaration copies the parent (`internal/layout/style_cascade.go:593-595`). The serializer does not read `st.FillRule`. It copies `nonzero` or `evenodd` from the style attribute when no `fill-rule` attribute is already written (`internal/layout/layout_svg.go:420-431`). The comment at `internal/layout/layout_svg.go:420` says the property has no resolved-style field. That comment is stale. The field is `FillRule` (`internal/layout/style.go:573`).

`emulateDroppedPresentation` runs before parse (`internal/svg/raster.go:379-383`). For `fill-rule="evenodd"` on a path of two nested straight subpaths with the same winding, it reverses the inner subpath so the parser's nonzero fill leaves a hole (`internal/svg/raster.go:623-627`, `internal/svg/raster.go:1084-1088`). Curves, relative commands, and any other subpath count pass through.

What to expect. Inside the 64 by 28 svg, the path `d="M10 10 H54 V34 H10 Z M22 16 H42 V28 H22 Z"` with `style="fill:#ff0000;fill-rule:evenodd"` serializes `fill-rule="evenodd"`. The nonzero raster fills the inner square. The evenodd raster leaves the hole center transparent. The center sample is 32/64 of the width and 22/28 of the height. Nonzero red there has red channel at least `0x8000` and a non-zero alpha. Evenodd alpha at that pixel is 0. That is `TestPaintOutputFillRuleEvenOddHolesRaster` (`internal/layout/css_behavior_paint_output_test.go:29-71`). `TestBehaviorSvgRasterFillRuleEvenoddCarvesHole` uses `d="M4 4 L60 4 L60 24 L4 24 Z M20 8 L44 8 L44 20 L20 20 Z"` and requires a non-zero pixel diff (`internal/layout/css_behavior_raster_cases_test.go:26-46`). It is not in the catalog test list.

`TestBehaviorSvgBakeFillRuleGap` only requires the attribute `fill-rule="evenodd"` on a solid rect (`internal/layout/css_behavior_svg_bake_test.go:244-250`). `TestBehaviorFillRuleNoPaintEffect` compares drawing-list op geometry of two solid red rects and requires a match (`internal/layout/css_behavior_svg_paint_test.go:261-267`). Both documents are one image of the same box, so that comparison stays green while the PNG bytes differ. `TestPropertyStoreFillRuleStoresAndInherits` stores `evenodd` on a div, inherits it, and keeps `nonzero` for `bogus` (`internal/layout/css_behavior_property_stores_test.go:138-162`).

- [x] `TestPaintOutputFillRuleEvenOddHolesRaster` requires a transparent hole under `evenodd` and a filled red hole under `nonzero` (`internal/layout/css_behavior_paint_output_test.go:48-71`).
- [x] `TestBehaviorSvgRasterFillRuleEvenoddCarvesHole` requires a non-zero pixel diff on the second holed path (`internal/layout/css_behavior_raster_cases_test.go:41-46`).
- [x] `TestPropertyStoreFillRuleStoresAndInherits` stores `evenodd`, inherits it, and rejects `bogus` (`internal/layout/css_behavior_property_stores_test.go:143-162`).
- [x] `TestBehaviorFillRuleNoPaintEffect` requires identical op geometry for a solid rect (`internal/layout/css_behavior_svg_paint_test.go:264-267`). It does not read PNG bytes.

### shape-rendering

Catalog row `shape-rendering` (`testdata/css/catalog/properties.json:16992`) is `implemented`. Syntax `auto | optimizeSpeed | crispEdges | geometricPrecision`. Handler `applySVGPresentationProps`. Listed tests: `TestBehaviorShapeRenderingNoPaintEffect`, `TestBehaviorSvgBakeShapeRenderingGap`, `TestPaintOutputShapeRenderingCrispSnapsRaster`. Limitations empty.

The initial value is `auto` (`internal/layout/style.go:828`). `applyShapeRenderingProperty` stores `auto`, `optimizespeed`, `crispedges`, or `geometricprecision` (`internal/layout/style_leftovers.go:286-290`). A missing declaration copies the parent (`internal/layout/style_cascade.go:597-599`). The serializer does not read `st.ShapeRendering`. It writes the camelCase attribute from the style attribute: `auto`, `optimizeSpeed`, `crispEdges`, `geometricPrecision` (`internal/layout/layout_svg.go:433-445`).

`emulateCrispAttr` rounds `x`, `y`, `width`, `height`, `rx`, and `ry` on a `rect`, the circle and ellipse centers and radii, and the line endpoints, when the element asks for `crispEdges` (`internal/svg/raster.go:999-1011`). Path `d` is left alone (`internal/svg/raster.go:1000-1001`). The comment on `applyShapeRenderingProperty` says the hint cannot change raster pixels (`internal/layout/style_leftovers.go:283-284`). That comment is stale. The pixel test below requires a diff.

What to expect. The same 64 by 28 svg holds `<rect x="2.5" y="2.5" width="60" height="24" style="fill:#ff0000;shape-rendering:crispEdges"/>`. The serialized document contains `shape-rendering="crispEdges"`. The crisp raster and the auto raster differ in at least one pixel. That is `TestPaintOutputShapeRenderingCrispSnapsRaster` (`internal/layout/css_behavior_paint_output_test.go:81-102`). `TestBehaviorSvgRasterShapeRenderingCrispEdgesSnapsRect` uses width `59.5` and height `23.5` and also requires a non-zero diff (`internal/layout/css_behavior_raster_cases_test.go:54-74`).

`TestPropertyStoreShapeRenderingStoresHint` stores `crispedges` on a div, inherits it, and keeps `auto` for `bogus` (`internal/layout/css_behavior_property_stores_test.go:171-196`). `TestBehaviorShapeRenderingNoPaintEffect` compares op geometry of two solid rects and requires a match (`internal/layout/css_behavior_svg_paint_test.go:412-418`). `TestBehaviorSvgBakeShapeRenderingGap` only checks the attribute (`internal/layout/css_behavior_svg_bake_test.go:258-265`).

- [x] `TestPaintOutputShapeRenderingCrispSnapsRaster` requires `shape-rendering="crispEdges"` and a non-zero pixel diff against auto (`internal/layout/css_behavior_paint_output_test.go:89-102`).
- [x] `TestBehaviorSvgRasterShapeRenderingCrispEdgesSnapsRect` requires a non-zero pixel diff on the fractional 59.5 by 23.5 rect (`internal/layout/css_behavior_raster_cases_test.go:64-74`).
- [x] `TestPropertyStoreShapeRenderingStoresHint` stores lowercased `crispedges`, inherits it, and rejects `bogus` (`internal/layout/css_behavior_property_stores_test.go:176-196`).
- [x] `TestBehaviorShapeRenderingNoPaintEffect` requires identical op geometry (`internal/layout/css_behavior_svg_paint_test.go:415-418`). It does not read PNG bytes.

### alignment-baseline

Catalog row `alignment-baseline` (`testdata/css/catalog/properties.json:1459`) is `implemented`. Syntax `baseline | <baseline-metric>`. Handler `applySVGPresentationProps`. Listed tests: `TestBehaviorAlignmentBaselineNoPaintEffect`, `TestSvgTextAlignmentBaselineCentralIdeographic`, `TestSvgTextAlignmentBaselineMiddleHanging`. Limitations empty.

The initial value is `auto` (`internal/layout/style.go:830`). `applyAlignmentBaselineProperty` stores `auto`, `baseline`, `alphabetic`, `hanging`, `ideographic`, `middle`, `central`, `mathematical`, `text-before-edge`, `text-after-edge`, `before-edge`, and `after-edge` (`internal/layout/style_leftovers.go:316-321`). `inheritGapBProps` does not copy `AlignmentBaseline` (`internal/layout/style_cascade.go:588-611`). The applier comment says the property is not inherited (`internal/layout/style_leftovers.go:314-315`).

On HTML text, `alignmentBaselineShift` runs when `vertical-align` is `baseline` or empty (`internal/layout/inline_vertical_align.go:169-186`). `auto` and `baseline` defer to `DominantBaseline`. Any other stored selector uses `baselineShiftFromMetrics` (`internal/layout/inline_vertical_align.go:133-156`). Hanging uses the fixed ratio `0.3125` of the font size (`internal/layout/inline_vertical_align.go:158-163`), which is 5px at 16px. The shift is added in `effectiveVerticalAlignShift` (`internal/layout/inline_vertical_align.go:70`).

What to expect on HTML. A paragraph `Ag` at `font-size:16px` and margin 0 (`internal/layout/css_behavior_svg_text_test.go:97-99`). The initial used value is `auto`. `central` raises the first text op by more than 0.5pt and at most 8pt. `ideographic` lowers it by the same bounds. `alphabetic` matches the default ops. That is `TestSvgTextAlignmentBaselineCentralIdeographic` (`internal/layout/css_behavior_svg_text_test.go:126-153`). `middle` raises the run inside those same bounds. `hanging` raises it by `pxToPt(5)`, checked with `near`. `bogus` matches the default ops. That is `TestSvgTextAlignmentBaselineMiddleHanging` (`internal/layout/css_behavior_svg_text_test.go:160-178`).

What to expect on an svg rect. `<rect ... style="fill:#ff0000;alignment-baseline:middle"/>` inside the 64 by 28 svg produces the same drawing-list ops as the rect with only `fill:#ff0000`. That is `TestBehaviorAlignmentBaselineNoPaintEffect` (`internal/layout/css_behavior_svg_paint_test.go:520-526`) and the same comparison in `TestSvgPaintPinAlignmentBaselineMiddle` (`internal/layout/css_behavior_svg_paint_pins_test.go:45-51`). The property is not baked onto the rect. The text shift above is the consumer.

- [x] `TestSvgTextAlignmentBaselineCentralIdeographic` stores `central`, raises that run, lowers `ideographic`, and keeps `alphabetic` on the default baseline (`internal/layout/css_behavior_svg_text_test.go:133-153`).
- [x] `TestSvgTextAlignmentBaselineMiddleHanging` raises `middle`, raises `hanging` by 5px at 16px, and ignores `bogus` (`internal/layout/css_behavior_svg_text_test.go:167-178`).
- [x] `TestBehaviorAlignmentBaselineNoPaintEffect` requires identical svg-rect ops for `alignment-baseline:middle` (`internal/layout/css_behavior_svg_paint_test.go:523-526`).

### dominant-baseline

Catalog row `dominant-baseline` (`testdata/css/catalog/properties.json:8279`) is `implemented`. Syntax `auto | <baseline-metric>`. Handler `applySVGPresentationProps`. Listed tests: `TestBehaviorDominantBaselineNoPaintEffect`, `TestPerspDominantBaselineBeyondHanging`, `TestPerspDominantBaselineInitialAndBogus`, `TestPerspDominantBaselineInherits`. The limitations text calls each of those tests a no-op that still needs engine work (`testdata/css/catalog/properties.json:8284-8288`). The opened tests assert a Y shift for every selector except `auto` and `alphabetic`. Leave the row `implemented`. Do not treat the limitation sentences as a reason to rebuild the property.

The initial value is `auto` (`internal/layout/style.go:829`). `applyDominantBaselineProperty` stores `auto`, `alphabetic`, `hanging`, `ideographic`, `middle`, `central`, `mathematical`, `text-before-edge`, and `text-after-edge` (`internal/layout/style_leftovers.go:302-307`). A missing declaration copies the parent (`internal/layout/style_cascade.go:601-603`). The HTML shift is the same `baselineShiftFromMetrics` table, reached when `alignment-baseline` is `auto` (`internal/layout/inline_vertical_align.go:181-183`).

What to expect. A paragraph `Ag` at `font-size:16px`. `hanging` stores `hanging` and raises the first text op by about 5px, and `near` requires that 5px distance. That is `TestPropertyStoreDominantBaselineHangingRaisesText` (`internal/layout/css_behavior_property_stores_test.go:237-280`). The catalog list omits this test. `middle`, `central`, `mathematical`, and `text-before-edge` each store the selector and raise the run by more than 0.5pt. `ideographic` and `text-after-edge` lower it by more than 0.5pt. That is `TestPerspDominantBaselineBeyondHanging` (`internal/layout/css_behavior_perspective_test.go:240-289`). `auto` is the initial. `alphabetic` stores `alphabetic` and paints the same ops as the default. `bogus` stays `auto`. That is `TestPerspDominantBaselineInitialAndBogus` (`internal/layout/css_behavior_perspective_test.go:298-321`). A parent `hanging` with no declaration on the child stores `hanging` on the child and raises the child run. That is `TestPerspDominantBaselineInherits` (`internal/layout/css_behavior_perspective_test.go:328-349`).

`TestBehaviorDominantBaselineNoPaintEffect` compares a bare paragraph with `dominant-baseline:alphabetic` and requires the same ops (`internal/layout/css_behavior_remaining_test.go:263-271`). The comment above it says other baselines stay on alphabetic until font tables exist (`internal/layout/css_behavior_remaining_test.go:259-260`). That comment is stale. The perspective test asserts the other selectors move.

- [x] `TestPropertyStoreDominantBaselineHangingRaisesText` stores `hanging` and raises a 16px run by about 5px (`internal/layout/css_behavior_property_stores_test.go:261-280`).
- [x] `TestPerspDominantBaselineBeyondHanging` raises `middle`, `central`, `mathematical`, and `text-before-edge`, and lowers `ideographic` and `text-after-edge` (`internal/layout/css_behavior_perspective_test.go:256-288`).
- [x] `TestPerspDominantBaselineInitialAndBogus` keeps `auto` and `alphabetic` on the same ops and rejects `bogus` (`internal/layout/css_behavior_perspective_test.go:306-321`).
- [x] `TestPerspDominantBaselineInherits` copies `hanging` onto the child and raises that run (`internal/layout/css_behavior_perspective_test.go:336-349`).
- [x] `TestBehaviorDominantBaselineNoPaintEffect` requires identical ops for `auto` and `alphabetic` (`internal/layout/css_behavior_remaining_test.go:266-271`).

## Phase 1: stylesheet store for bakes that already move pixels

Shared rules for every name in this phase:

- New code lives in `internal/layout/svg_presentation.go`. Call it from `writeSVGNode`. Do not grow `internal/layout/layout.go`.
- The rasterizer keeps reading attributes. There is no new page painter.
- CSS replaces the presentation attribute, following `d`. Today `writeSVGNode` drops an authored `d` when the style attribute carries `path(...)`, then writes one `d` (`internal/layout/layout_svg.go:288-298`, `internal/layout/layout_svg.go:326-328`, `internal/layout/layout_svg.go:1767-1771`). Other names keep the attribute. `TestBehaviorSvgGeoAttrWinsNoDuplicate` requires one `cx` and the value `10` when the element has `cx="10"` and `style="cx:40px"` (`internal/layout/css_behavior_svg_geo_test.go:222-236`).
- Behavior tests for the stylesheet half go in `internal/layout/css_behavior_svg_present_test.go`. That file does not exist yet.
- A catalog row flips to `implemented` only after its behavior test and a consumer outside the style layer exist (`testdata/css/catalog/properties.json:9`).
- Pixel tests that rasterize call `svgBakeRequireMem` and skip under 1200 MB available (`internal/layout/css_behavior_svg_bake_test.go:49-64`). Keep that skip.

- [x] Make CSS win over the presentation attribute for every name this phase bakes, using the `d` suppression as the pattern. Update `TestBehaviorSvgGeoAttrWinsNoDuplicate` in the same change. `d` already lets the style attribute win. Leave that precedence in place. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape`.
- [x] Read the stored stylesheet value in the bake. Until that store exists, the bake reads `node.Attribute("style")` only. Proof: `prepareSVGStylesheetGeometry` in `internal/layout/svg_presentation.go`.

### cx

Catalog row `cx` (`testdata/css/catalog/properties.json:8162`) is `unsupported`. Syntax `<length-percentage>`. Handlers empty. Listed test `TestBehaviorSvgGeoCxMovesCircle`. Handler gap.

Current behavior. When no `cx` attribute is written, `svgBakeLengthValue` accepts a plain number or `px`, `pt`, `pc`, `cm`, `mm`, `q`, `in`, or `%` from the style attribute and writes `cx` (`internal/layout/layout_svg.go:456-459`, validator `internal/layout/layout_svg.go:565-586`). Font units and viewport units return false. An authored `cx` attribute wins (`internal/layout/css_behavior_svg_geo_test.go:225-235`).

What to expect, already asserted for the style attribute. Inside the 64 by 28 svg, `<circle cy="14" r="8" style="fill:#ff0000;cx:10px"/>` and the same circle with `cx:40px` differ in at least one raster pixel. That is `TestBehaviorSvgGeoCxMovesCircle` (`internal/layout/css_behavior_svg_geo_test.go:21-36`).

What to expect once the stylesheet store lands. The same two circles, with `circle { cx: 40px }` in a stylesheet versus `cx: 10px`, differ in pixels. A circle with attribute `cx="10"` plus the `40px` rule paints the `40px` circle.

- [x] `TestBehaviorSvgGeoCxMovesCircle` requires a pixel diff between inline `cx:10px` and `cx:40px` (`internal/layout/css_behavior_svg_geo_test.go:31-36`).
- [x] Acceptance: the units `svgBakeLengthValue` already allows (`internal/layout/layout_svg.go:581-584`). Reject font units and viewport units. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Storage: store the length from a stylesheet rule on the resolved style in `internal/layout/svg_presentation.go`. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Consumer: bake the stored value with the phase CSS-wins rule. No new painter. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Behavior test: `TestBehaviorSvgPresentCx` in `internal/layout/css_behavior_svg_present_test.go`. A stylesheet `cx: 40px` differs in pixels from `cx: 10px`, and CSS beats `cx="10"`. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Catalog: mark `cx` `implemented` only after that test. The inline pixel test is already listed and is not enough while the handler list is empty. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.

### cy

Catalog row `cy` (`testdata/css/catalog/properties.json:8183`) is `unsupported`. Syntax `<length-percentage>`. Handlers empty. Listed test `TestBehaviorSvgGeoCyMovesCircle`. Handler gap.

Current behavior. Same length validator as `cx`, writing `cy` (`internal/layout/layout_svg.go:461-464`).

What to expect, already asserted. `<circle cx="32" r="6" style="fill:#ff0000;cy:8px"/>` versus `cy:20px` differs in pixels. That is `TestBehaviorSvgGeoCyMovesCircle` (`internal/layout/css_behavior_svg_geo_test.go:42-57`).

What to expect once stored. A stylesheet `cy: 20px` differs from `cy: 8px`, and CSS beats a `cy` attribute.

- [x] `TestBehaviorSvgGeoCyMovesCircle` requires a pixel diff between inline `cy:8px` and `cy:20px` (`internal/layout/css_behavior_svg_geo_test.go:51-56`).
- [x] Acceptance: the same length units as `cx`. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Storage: stylesheet length on the resolved style. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Consumer: bake with the phase CSS-wins rule. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Behavior test: `TestBehaviorSvgPresentCy`. Stylesheet pixel diff, and CSS beats a `cy` attribute. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Catalog: mark `cy` `implemented` only after that test. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.

### r

Catalog row `r` (`testdata/css/catalog/properties.json:15075`) is `unsupported`. Syntax `<length-percentage>`. Handlers empty. Listed tests `TestBehaviorSvgGeoRResizesCircle` and `TestBehaviorRubyParsersAcceptSpecValues`. Handler gap. The ruby test is outside this ledger. Do not treat it as the SVG consumer.

Current behavior. `svgBakeNonNegativeLength` drops a negative value (`internal/layout/layout_svg.go:466-469`, `internal/layout/layout_svg.go:589-605`).

What to expect, already asserted. `<circle cx="32" cy="14" style="fill:#ff0000;r:4px"/>` versus `r:12px` differs in pixels. The larger radius paints the larger disc. That is `TestBehaviorSvgGeoRResizesCircle` (`internal/layout/css_behavior_svg_geo_test.go:63-78`).

What to expect once stored. A stylesheet `r: 12px` differs from `r: 4px`. A negative stylesheet value does not replace an earlier valid radius. CSS beats an `r` attribute.

- [x] `TestBehaviorSvgGeoRResizesCircle` requires a pixel diff between inline `r:4px` and `r:12px` (`internal/layout/css_behavior_svg_geo_test.go:72-77`).
- [x] Acceptance: a non-negative length or percentage in the `cx` units. A negative value does not replace an earlier valid radius. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Storage: stylesheet length on the resolved style. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Consumer: bake with the phase CSS-wins rule. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Behavior test: `TestBehaviorSvgPresentR`. Stylesheet pixel diff, negative value ignored, CSS beats an `r` attribute. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Catalog: mark `r` `implemented` only after that test. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.

### rx

Catalog row `rx` (`testdata/css/catalog/properties.json:16164`) is `unsupported`. Syntax `<length-percentage> | auto`. Handlers empty. Listed test `TestBehaviorSvgGeoRxRoundsRect`. Handler gap.

Current behavior. Non-negative lengths bake (`internal/layout/layout_svg.go:471-474`). `auto` fails `svgBakeLengthValue` because it is not a number (`internal/layout/layout_svg.go:575-578`).

What to expect, already asserted. The shared rect `<rect x="2" y="2" width="60" height="24" style="fill:#ff0000;rx:0px"/>` (`internal/layout/css_behavior_svg_bake_test.go:29`) versus `rx:10px` differs in pixels. That is `TestBehaviorSvgGeoRxRoundsRect` (`internal/layout/css_behavior_svg_geo_test.go:85-101`).

What to expect once stored. A stylesheet `rx: 10px` differs from `rx: 0px`. `rx: auto` is written as the attribute `auto`. CSS beats an `rx` attribute.

- [x] `TestBehaviorSvgGeoRxRoundsRect` requires a pixel diff between inline `rx:0px` and `rx:10px` (`internal/layout/css_behavior_svg_geo_test.go:95-100`).
- [x] Acceptance: non-negative lengths as today, plus `auto`. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Storage: stylesheet length or `auto` on the resolved style. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Consumer: bake `auto` as the attribute `auto`. CSS-wins. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Behavior test: `TestBehaviorSvgPresentRx`. Stylesheet pixel diff, `auto` baked, CSS beats an `rx` attribute. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Catalog: mark `rx` `implemented` only after that test. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.

### ry

Catalog row `ry` (`testdata/css/catalog/properties.json:16185`) is `unsupported`. Syntax `<length-percentage> | auto`. Handlers empty. Listed test `TestBehaviorSvgGeoRyRoundsRect`. Handler gap.

Current behavior. Same non-negative bake as `rx` (`internal/layout/layout_svg.go:476-479`). `auto` is rejected by the number parser.

What to expect, already asserted. The shared rect with `ry:0px` versus `ry:8px` differs in pixels. That is `TestBehaviorSvgGeoRyRoundsRect` (`internal/layout/css_behavior_svg_geo_test.go:107-123`).

What to expect once stored. A stylesheet `ry: 8px` differs from `ry: 0px`. `auto` is baked. CSS beats an `ry` attribute.

- [x] `TestBehaviorSvgGeoRyRoundsRect` requires a pixel diff between inline `ry:0px` and `ry:8px` (`internal/layout/css_behavior_svg_geo_test.go:117-122`).
- [x] Acceptance: non-negative lengths, plus `auto`. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Storage: stylesheet length or `auto`. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Consumer: bake `auto`. CSS-wins. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Behavior test: `TestBehaviorSvgPresentRy`. Stylesheet pixel diff, `auto` baked, CSS beats an `ry` attribute. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Catalog: mark `ry` `implemented` only after that test. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.

### x

Catalog row `x` (`testdata/css/catalog/properties.json:20556`) is `unsupported`. Syntax `<length-percentage>`. Handlers empty. Listed test `TestBehaviorSvgGeoXMovesRect`. Handler gap.

Current behavior. `svgBakeLengthValue` writes `x` (`internal/layout/layout_svg.go:481-484`).

What to expect, already asserted. `<rect y="4" width="30" height="20" style="fill:#ff0000;x:2px"/>` versus `x:24px` differs in pixels. That is `TestBehaviorSvgGeoXMovesRect` (`internal/layout/css_behavior_svg_geo_test.go:128-143`).

What to expect once stored. A stylesheet `x: 24px` differs from `x: 2px`, and CSS beats an `x` attribute.

- [x] `TestBehaviorSvgGeoXMovesRect` requires a pixel diff between inline `x:2px` and `x:24px` (`internal/layout/css_behavior_svg_geo_test.go:138-143`).
- [x] Acceptance: the same length units as `cx`. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Storage: stylesheet length on the resolved style. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Consumer: bake with the phase CSS-wins rule. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Behavior test: `TestBehaviorSvgPresentX`. Stylesheet pixel diff, and CSS beats an `x` attribute. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Catalog: mark `x` `implemented` only after that test. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.

### y

Catalog row `y` (`testdata/css/catalog/properties.json:20577`) is `unsupported`. Syntax `<length-percentage>`. Handlers empty. Listed test `TestBehaviorSvgGeoYMovesRect`. Handler gap.

Current behavior. `svgBakeLengthValue` writes `y` (`internal/layout/layout_svg.go:486-489`).

What to expect, already asserted. `<rect x="4" width="30" height="16" style="fill:#ff0000;y:2px"/>` versus `y:10px` differs in pixels. That is `TestBehaviorSvgGeoYMovesRect` (`internal/layout/css_behavior_svg_geo_test.go:149-164`).

What to expect once stored. A stylesheet `y: 10px` differs from `y: 2px`, and CSS beats a `y` attribute.

- [x] `TestBehaviorSvgGeoYMovesRect` requires a pixel diff between inline `y:2px` and `y:10px` (`internal/layout/css_behavior_svg_geo_test.go:159-164`).
- [x] Acceptance: the same length units as `cx`. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Storage: stylesheet length on the resolved style. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Consumer: bake with the phase CSS-wins rule. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Behavior test: `TestBehaviorSvgPresentY`. Stylesheet pixel diff, and CSS beats a `y` attribute. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Catalog: mark `y` `implemented` only after that test. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.

### d

Catalog row `d` (`testdata/css/catalog/properties.json:8204`) is `unsupported`. Syntax `none | <string>`. Handlers empty. Listed test `TestBehaviorSvgCSSDProvidesPathData`. Handler gap.

Current behavior. `svgCSSPathData` accepts `path("...")` or `path('...')` and returns the inner path data. `none` and any other string return false (`internal/layout/layout_svg.go:1856-1875`). On a `path` element the serializer drops the `d` attribute when that function succeeds, then writes one `d` (`internal/layout/layout_svg.go:296-298`, `internal/layout/layout_svg.go:1767-1771`). The read is the style attribute. A stylesheet rule is not consulted.

What to expect, already asserted. A 64 by 40 svg (`internal/layout/css_behavior_svg_markers_test.go:32-34`). `<path fill="#000000"/>` paints 0 pixels with alpha at least 128. `<path fill="#000000" style="d:path('M8 8 H56 V32 H8 Z')"/>` paints more than 0. That is `TestBehaviorSvgCSSDProvidesPathData` (`internal/layout/css_behavior_svg_markers_test.go:111-124`). The same CSS path replaces `d="M8 8 h2 v2 h-2 Z"` and the painted count matches the CSS-only path, which differs from the attribute-only path. That is `TestBehaviorSvgCSSDOverridesDAttribute` (`internal/layout/css_behavior_svg_markers_test.go:132-148`). The catalog list omits the override test.

What to expect once stored. A stylesheet `d: path('M8 8 H56 V32 H8 Z')` paints more than 0 pixels on a path with no `d` attribute. `d: none` clears path data so the path paints 0. A bare string that is not `path(...)` stays rejected until a test shows the canvas parser can use it as a `d` attribute. Do not pass an unparsed string through.

- [x] `TestBehaviorSvgCSSDProvidesPathData` requires the inline `path(...)` to paint and the empty path to paint nothing (`internal/layout/css_behavior_svg_markers_test.go:114-124`).
- [x] `TestBehaviorSvgCSSDOverridesDAttribute` requires the style attribute to replace the `d` attribute (`internal/layout/css_behavior_svg_markers_test.go:135-148`).
- [x] Acceptance: keep `path("...")` and `path('...')`. `none` clears path data. A bare string stays rejected. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Storage: store the path data from a stylesheet rule. Today only the style attribute is read (`internal/layout/layout_svg.go:1860`). Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Consumer: bake one `d` attribute. The style attribute already wins over the attribute. Keep that, and extend the same win to a stylesheet rule. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Behavior test: `TestBehaviorSvgPresentD`. A stylesheet `d: path(...)` paints, and `d: none` paints nothing. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Catalog: mark `d` `implemented` only after the stylesheet test. Proof: `TestBehaviorSvgStylesheetGeometryMovesShape` passed. The stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.

### marker

Catalog row `marker` (`testdata/css/catalog/properties.json:12032`) is `unsupported`. Syntax `none | <marker-ref>`. Handlers empty. Listed test `TestBehaviorSvgMarkerShorthandPaintsVertices`. Handler gap. The v0.0.3 ledger said this name was not a catalog row. That sentence is stale.

Current behavior. `writeSVGMarkerAttrs` reads `marker` from the style attribute. For each of `marker-start`, `marker-mid`, and `marker-end`, an empty longhand takes the shorthand. Only a value that starts with `url(` and ends with `)` is written. `none` returns false and is not written (`internal/layout/layout_svg.go:1811-1853`). The comment says a lone `none` matches the no-marker initial (`internal/layout/layout_svg.go:1817-1818`).

What to expect, already asserted. A 64 by 40 svg holds the arrow marker `id="ah"` (`internal/layout/css_behavior_svg_markers_test.go:22-24`) and `<path d="M8 34 L32 8 L56 34" fill="none" style="stroke:#000000;stroke-width:3;marker:url(#ah)"/>`. Painted pixels with alpha at least 128 exceed the same path without `marker`. That is `TestBehaviorSvgMarkerShorthandPaintsVertices` (`internal/layout/css_behavior_svg_markers_test.go:92-103`). The test counts the shorthand expansion. It does not isolate one longhand.

What to expect once stored. A stylesheet `marker: url(#ah)` on that two-segment path paints more than `marker: none`. CSS beats a `marker-end` attribute on the end vertex because the shorthand fills each unset longhand. `marker: none` clears markers that came from a stylesheet. An unknown token does not replace a valid ref.

- [x] `TestBehaviorSvgMarkerShorthandPaintsVertices` requires inline `marker: url(#ah)` to paint more pixels than the bare path (`internal/layout/css_behavior_svg_markers_test.go:96-103`).
- [x] Acceptance: `none` and `url(#id)`. Anything else does not replace an earlier valid ref. Proof: `TestBehaviorSvgPresentMarker` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Storage: stylesheet shorthand on the resolved style, separate from the three longhand fields. Proof: `TestBehaviorSvgPresentMarker` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Consumer: expand onto each longhand that is unset, then bake those attributes. CSS-wins. `none` clears. Proof: `TestBehaviorSvgPresentMarker` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Behavior test: `TestBehaviorSvgPresentMarker`. Stylesheet `marker: url(#ah)` paints more than `marker: none` on the two-segment path, and a longhand set beside the shorthand keeps its own value. Proof: `TestBehaviorSvgPresentMarker` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Catalog: mark `marker` `implemented` only after that test. Proof: `TestBehaviorSvgPresentMarker` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.

### marker-end

Catalog row `marker-end` (`testdata/css/catalog/properties.json:12053`) is `unsupported`. Syntax `none | <marker-ref>`. Handlers empty. Listed test `TestBehaviorSvgMarkerEndPaintsArrowhead`. Handler gap.

Current behavior. The marker loop writes `marker-end` from the style attribute when `isSVGMarkerRef` is true (`internal/layout/layout_svg.go:1822-1840`). `none` is skipped.

What to expect, already asserted. The arrow marker plus `<path d="M8 30 H56" fill="none" style="stroke:#000000;stroke-width:3;marker-end:url(#ah)"/>` paints more pixels than the same line without `marker-end`. That is `TestBehaviorSvgMarkerEndPaintsArrowhead` (`internal/layout/css_behavior_svg_markers_test.go:74-85`).

What to expect once stored. A stylesheet `marker-end: url(#ah)` on that line paints more than `marker-end: none`. CSS beats a `marker-end` attribute.

- [x] `TestBehaviorSvgMarkerEndPaintsArrowhead` requires inline `marker-end: url(#ah)` to paint more pixels than the bare line (`internal/layout/css_behavior_svg_markers_test.go:78-85`).
- [x] Acceptance: `none` and `url(#id)`. Proof: `TestBehaviorSvgPresentMarkerEnd` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Storage: stylesheet longhand, separate from `marker` and from `marker-mid`. Proof: `TestBehaviorSvgPresentMarkerEnd` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Consumer: bake `marker-end`. CSS-wins. The canvas marker painter already draws `url(#id)`. Proof: `TestBehaviorSvgPresentMarkerEnd` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Behavior test: `TestBehaviorSvgPresentMarkerEnd`. Stylesheet pixel increase, and CSS beats the attribute. Proof: `TestBehaviorSvgPresentMarkerEnd` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Catalog: mark `marker-end` `implemented` only after that test. Proof: `TestBehaviorSvgPresentMarkerEnd` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.

### marker-mid

Catalog row `marker-mid` (`testdata/css/catalog/properties.json:12074`) is `unsupported`. Syntax `none | <marker-ref>`. Handlers empty. The listed test is `TestBehaviorSvgMarkerShorthandPaintsVertices`. That test declares `marker`, not `marker-mid` (`internal/layout/css_behavior_svg_markers_test.go:95-99`). Handler gap.

Current behavior. The same loop writes `marker-mid` when the style attribute has a `url(...)` longhand, or when the longhand is empty and the shorthand is a `url(...)` (`internal/layout/layout_svg.go:1822-1830`). No opened test declares `marker-mid` by itself, so there is no `[x]` baseline.

What to expect. The arrow marker plus `<path d="M8 34 L32 8 L56 34" fill="none" stroke="#000000" stroke-width="3" style="marker-mid:url(#ah)"/>`. The middle vertex is `(32, 8)`. Painted pixels exceed the same path with `marker-mid: none`. The shorthand test is not the proof.

- [x] Acceptance: `none` and `url(#id)`. Proof: `TestBehaviorSvgPresentMarkerMid` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Storage: stylesheet longhand, separate from `marker-end`. Proof: `TestBehaviorSvgPresentMarkerMid` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Consumer: bake `marker-mid` only. CSS-wins. Proof: `TestBehaviorSvgPresentMarkerMid` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Behavior test: `TestBehaviorSvgPresentMarkerMid`. A path with a mid vertex and stylesheet `marker-mid: url(#ah)` paints more than `marker-mid: none`. Do not use the shorthand as the proof. Proof: `TestBehaviorSvgPresentMarkerMid` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Catalog: mark `marker-mid` `implemented` only after that test. Point the catalog test at `TestBehaviorSvgPresentMarkerMid`, not at the shorthand test. Proof: `TestBehaviorSvgPresentMarkerMid` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.

### marker-start

Catalog row `marker-start` (`testdata/css/catalog/properties.json:12111`) is `unsupported`. Syntax `none | <marker-ref>`. Handlers empty. The listed test is `TestBehaviorSvgMarkerEndPaintsArrowhead` (`testdata/css/catalog/properties.json:12124`). That test declares `marker-end` on a horizontal line (`internal/layout/css_behavior_svg_markers_test.go:77-81`). It does not assert `marker-start`. Handler gap.

Current behavior. The marker loop writes `marker-start` under the same `url(...)` rule (`internal/layout/layout_svg.go:1822-1840`). No opened test declares `marker-start` by itself, so there is no `[x]` baseline.

What to expect. The arrow marker plus `<path d="M8 30 H56" fill="none" stroke="#000000" stroke-width="3" style="marker-start:url(#ah)"/>`. Painted pixels exceed the same line with `marker-start: none`. The first vertex is otherwise bare.

- [x] Acceptance: `none` and `url(#id)`. Proof: `TestBehaviorSvgPresentMarkerStart` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Storage: stylesheet longhand, separate from `marker-end`. Proof: `TestBehaviorSvgPresentMarkerStart` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Consumer: bake `marker-start` only. CSS-wins. Proof: `TestBehaviorSvgPresentMarkerStart` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Behavior test: `TestBehaviorSvgPresentMarkerStart`. Stylesheet `marker-start: url(#ah)` paints more than `marker-start: none` on that line. Proof: `TestBehaviorSvgPresentMarkerStart` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Catalog: mark `marker-start` `implemented` only after that test. Replace the listed test name, which asserts `marker-end`. Proof: `TestBehaviorSvgPresentMarkerStart` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.

### stop-color

Catalog row `stop-color` (`testdata/css/catalog/properties.json:17135`) is `unsupported`. The object has no `upstream_syntax` field (`testdata/css/catalog/properties.json:17151-17152`). Handlers empty. Listed test `TestBehaviorSvgGeoStopColorRecolorsGradient`. Handler gap.

Current behavior. On a `stop` element only, a `#rgb`, `#rrggbb`, `rgb()`, or `rgba()` value from the style attribute is written (`internal/layout/layout_svg.go:491-498`, validator `internal/layout/layout_svg.go:607-636`). Named colors stay unbaked. The comment says the canvas parser would turn them black (`internal/layout/layout_svg.go:607-610`).

What to expect, already asserted. A horizontal `linearGradient` `id="g"` has `<stop offset="0" style="stop-color:#ff0000"/>` versus `stop-color:#00ff00`, and a second stop `stop-color:#0000ff`. The rect is `<rect x="2" y="2" width="60" height="24" fill="url(#g)"/>`. The two rasters differ. That is `TestBehaviorSvgGeoStopColorRecolorsGradient` (`internal/layout/css_behavior_svg_geo_test.go:173-190`). The rect's `fill="url(#g)"` stays an attribute. The test comment says the fill cascade drops `url()` (`internal/layout/css_behavior_svg_geo_test.go:169-171`).

What to expect once stored. A stylesheet `stop-color: #00ff00` on the first stop differs in pixels from `#ff0000`. A `stop-color` on a `rect` does not bake. CSS beats a `stop-color` attribute. Named colors bake only after they are converted to hex, so the canvas parser cannot turn them black.

- [x] `TestBehaviorSvgGeoStopColorRecolorsGradient` requires a pixel diff between inline red and green stop colors (`internal/layout/css_behavior_svg_geo_test.go:182-190`).
- [x] Acceptance: hex and `rgb()` / `rgba()` as today. Named colors only after a hex conversion. Proof: `TestBehaviorSvgPresentStopColor` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Storage: stylesheet color on the stop's resolved style. Proof: `TestBehaviorSvgPresentStopColor` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Consumer: bake onto `stop` only. CSS-wins over a `stop-color` attribute. Proof: `TestBehaviorSvgPresentStopColor` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Behavior test: `TestBehaviorSvgPresentStopColor`. Stylesheet pixel diff. A `stop-color` on a `rect` does not bake. Proof: `TestBehaviorSvgPresentStopColor` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Catalog: mark `stop-color` `implemented` only after that test. Proof: `TestBehaviorSvgPresentStopColor` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.

### stop-opacity

Catalog row `stop-opacity` (`testdata/css/catalog/properties.json:17155`) is `unsupported`. The object has no `upstream_syntax` field (`testdata/css/catalog/properties.json:17171-17172`). Handlers empty. Listed test `TestBehaviorSvgGeoStopOpacityFadesGradient`. Handler gap.

Current behavior. On a `stop` only, a number from 0 to 1 or a percentage from 0% to 100% is written (`internal/layout/layout_svg.go:500-504`, validator `internal/layout/layout_svg.go:639-662`).

What to expect, already asserted. The same gradient with the first stop `stop-color:#ff0000` versus `stop-color:#ff0000;stop-opacity:0.2` differs in pixels. That is `TestBehaviorSvgGeoStopOpacityFadesGradient` (`internal/layout/css_behavior_svg_geo_test.go:197-214`).

What to expect once stored. A stylesheet `stop-opacity: 0.2` on the first stop differs from an opaque stop. A value outside 0..1 or 0%..100% does not replace an earlier valid opacity. A `stop-opacity` on a `rect` does not bake.

- [x] `TestBehaviorSvgGeoStopOpacityFadesGradient` requires a pixel diff for inline `stop-opacity: 0.2` (`internal/layout/css_behavior_svg_geo_test.go:206-214`).
- [x] Acceptance: number 0 to 1, and percentage 0% to 100%. A value outside that range does not replace an earlier valid opacity. Proof: `TestBehaviorSvgPresentStopOpacity` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Storage: stylesheet value on the stop. Proof: `TestBehaviorSvgPresentStopOpacity` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Consumer: bake onto `stop` only. CSS-wins. Proof: `TestBehaviorSvgPresentStopOpacity` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Behavior test: `TestBehaviorSvgPresentStopOpacity`. Stylesheet `stop-opacity: 0.2` pixel diff versus an opaque stop. Proof: `TestBehaviorSvgPresentStopOpacity` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.
- [x] Catalog: mark `stop-opacity` `implemented` only after that test. Proof: `TestBehaviorSvgPresentStopOpacity` passed in `internal/layout/css_behavior_svg_present_paint_test.go`. A stylesheet value is stored and beats the presentation attribute. Catalog status is `implemented`.

**Gate:** each name above stays `unsupported` until its own behavior test and catalog row close.

## Phase 2: serialized attributes the rasterizer ignores

The bake comment says the canvas `setAttribute` switch has no case for these three, so the gap tests assert XML and never pixels (`internal/layout/layout_svg.go:506-510`). `emulateDroppedPresentation` rewrites `fill-rule` and `crispEdges` only (`internal/svg/raster.go:615-621`). It does not mention these three. New pixel behavior goes in `internal/svg/present.go`, and `paint-order` is there now: a filled and stroked shape is written as one copy per part in paint order. No filter graph.

### paint-order

Catalog row `paint-order` (`testdata/css/catalog/properties.json:14482`) is `unsupported`. Syntax `normal | [ fill || stroke || markers ]`. Handlers empty. Listed test `TestBehaviorSvgGeoPaintOrderGap`. Handler gap.

Current behavior. `svgBakePaintOrder` accepts `normal`, or one or more of `fill`, `stroke`, and `markers` without repeats (`internal/layout/layout_svg.go:665-698`). The serializer writes the attribute from the style attribute (`internal/layout/layout_svg.go:511-514`).

What to expect, already asserted. `<rect x="2" y="2" width="60" height="24" style="fill:#ff0000;stroke:#0000ff;stroke-width:4pt;paint-order:stroke"/>` serializes `paint-order="stroke"`. The test does not compare pixels. That is `TestBehaviorSvgGeoPaintOrderGap` (`internal/layout/css_behavior_svg_geo_test.go:245-252`).

What to expect if emulation lands. The same rect with stylesheet `paint-order: stroke` differs in pixels from the default fill-then-stroke order: the blue stroke is painted under the red fill, so the stroke pixels inside the rect change. If emulation does not land, the catalog row stays `unsupported`.

- [x] `TestBehaviorSvgGeoPaintOrderGap` requires `paint-order="stroke"` in the XML and makes no pixel assertion (`internal/layout/css_behavior_svg_geo_test.go:248-252`).
- [x] Acceptance: `normal`, or one or more of `fill`, `stroke`, and `markers` without repeats. That is what the validator already allows.
- [x] Storage: stylesheet value on the resolved style.
- [x] Consumer: bake the attribute. Then either emulate paint order for a rect with both fill and stroke in `internal/svg/present.go`, or stop at the serialize gap. The opened gap test stays green until a pixel test replaces it.
- [x] Behavior test: `TestBehaviorSvgPresentPaintOrder`. If emulation lands, the blue-stroke / red-fill rect differs in pixels from the default order. If emulation does not land, this row stays open. Proof: `TestBehaviorSvgPresentPaintOrder` passed: stroke-first puts the red fill over the inner half of the blue stroke, and the default order stays blue. The emulation is `internal/svg/present.go`. Catalog status is `implemented`.
- [x] Catalog: `implemented` only after a pixel diff. Proof: `TestBehaviorSvgPresentPaintOrder` passed: stroke-first puts the red fill over the inner half of the blue stroke, and the default order stays blue. The emulation is `internal/svg/present.go`. Catalog status is `implemented`.

### vector-effect

Catalog row `vector-effect` (`testdata/css/catalog/properties.json:19623`) is `unsupported`. Syntax `none | non-scaling-stroke | non-scaling-size | non-rotation | fixed-position`. Handlers empty. Listed test `TestBehaviorSvgGeoVectorEffectGap`. Handler gap.

Current behavior. The validator accepts only `none` and `non-scaling-stroke` (`internal/layout/layout_svg.go:701-710`). The serializer writes that token from the style attribute (`internal/layout/layout_svg.go:516-519`).

What to expect, already asserted. `<line x1="10" y1="14" x2="54" y2="14" style="fill:none;stroke:#ff0000;stroke-width:6pt;vector-effect:non-scaling-stroke"/>` serializes `vector-effect="non-scaling-stroke"`. No pixel assertion. That is `TestBehaviorSvgGeoVectorEffectGap` (`internal/layout/css_behavior_svg_geo_test.go:261-269`).

What to expect if a scale probe shows a difference. A viewport that scales the same line down keeps a thicker stroke with `non-scaling-stroke` than the same stroke without it. `non-scaling-size`, `non-rotation`, and `fixed-position` are serialized once acceptance allows them. They stay off the catalog until a test shows a visible difference. Do not invent a scale model in the page painter.

- [x] `TestBehaviorSvgGeoVectorEffectGap` requires `vector-effect="non-scaling-stroke"` and makes no pixel assertion (`internal/layout/css_behavior_svg_geo_test.go:264-269`).
- [ ] Acceptance: all five catalog tokens. Unknown tokens do not bake.
- [ ] Storage: stylesheet value.
- [ ] Consumer: bake the attribute. `non-scaling-stroke` needs a pixel test under a viewBox scale.
- [ ] Behavior test: `TestBehaviorSvgPresentVectorEffect`. A scaled viewport with `non-scaling-stroke` keeps a thicker stroke in pixels than the same stroke without it. If the canvas cannot show that, the row stays open.
- [ ] Catalog: `implemented` only after that pixel diff.

### text-rendering

Catalog row `text-rendering` (`testdata/css/catalog/properties.json:18570`) is `unsupported`. Syntax `auto | optimizeSpeed | optimizeLegibility | geometricPrecision`. Handlers empty. Listed test `TestBehaviorSvgGeoTextRenderingGap`. Handler gap.

Current behavior. `svgBakeTextRendering` returns `auto`, `optimizeSpeed`, `optimizeLegibility`, or `geometricPrecision` (`internal/layout/layout_svg.go:713-727`). The serializer writes that attribute from the style attribute (`internal/layout/layout_svg.go:521-524`).

What to expect, already asserted. `<text x="4" y="18" style="font-size:12px;fill:#000000;text-rendering:optimizeLegibility">hi</text>` serializes `text-rendering="optimizeLegibility"`. No pixel assertion. That is `TestBehaviorSvgGeoTextRenderingGap` (`internal/layout/css_behavior_svg_geo_test.go:278-285`). After that attribute is written, `stripSVGTextElements` removes the `text` element before parse (`internal/svg/raster.go:386`). The PNG cannot show the hint.

- [x] `TestBehaviorSvgGeoTextRenderingGap` requires `text-rendering="optimizeLegibility"` and makes no pixel assertion (`internal/layout/css_behavior_svg_geo_test.go:281-285`).
- [ ] Acceptance: the four catalog keywords, baked in the camelCase form the mapper already uses.
- [ ] Storage: stylesheet value.
- [~] Pixel consumer. Reason: SVG text is removed before parse (`internal/svg/raster.go:384-386`), so the raster cannot show a hint. Owner: this ledger. Next gate: an SVG text drawer that shapes glyphs without calling the system font loader under `canvasMu`. Until that drawer exists, do not remove the strip to make a pixel diff. Storage and the serialize test can still land.
- [ ] Behavior test: `TestBehaviorSvgPresentTextRendering`. The attribute is present in the serialized XML, and the PNG bytes match the same text without the property.
- [ ] Catalog: leave `unsupported` while the pixel row stays deferred. Do not flip it on a serialize-only test.

**Gate:** no catalog flips in this phase unless a pixel test exists. `text-rendering` has no pixel test while text is stripped.

## Phase 3: names with no bake

None of these names appear in `writeSVGPresentationAttrs` or `writeSVGExtraPresentationAttrs`. A search of `internal/layout/layout_svg.go` finds no bake arm for them. `path-length` moved out of the pixel-bake list and is now done: `internal/svg/present.go` scales `stroke-dasharray` and `stroke-dashoffset` by real length over `pathLength`.

### path-length

Catalog row `path-length` (`testdata/css/catalog/properties.json:14503`) is `unsupported`. Syntax `none | <length> [0,∞]` (`testdata/css/catalog/properties.json:14516`). Handlers empty. Tests empty. Handler gap.

Current behavior. The serializer never reads `path-length`. `TestBehaviorSvgDraftPaintLonghandsInert` puts `path-length:100` on a dashed line and requires the painted-pixel count to match the line without it (`internal/layout/css_behavior_svg_markers_test.go:218`, comparison `internal/layout/css_behavior_svg_markers_test.go:233-239`). The test comment says the canvas renderer has no `pathLength` support, so baking the attribute would not change pixels (`internal/layout/css_behavior_svg_markers_test.go:187-188`).

What to expect while that limit holds. `<path d="M4 20 H60" fill="none" style="stroke:#000000;stroke-width:3;stroke-dasharray:6 3;path-length:100"/>` paints the same pixel count as the path without `path-length`. That pin is already asserted.

What to expect if a later probe shows the canvas honors `pathLength`. A dashed path with `path-length: 100` spaces its dashes differently from the same path with `path-length: none`, and the pixel counts differ. Until that probe shows a difference, do not bake the attribute, and do not flip the catalog.

- [x] `TestBehaviorSvgDraftPaintLonghandsInert` requires `path-length:100` to keep the dashed line's painted-pixel count (`internal/layout/css_behavior_svg_markers_test.go:218` and `internal/layout/css_behavior_svg_markers_test.go:233-239`).
- [x] Acceptance: `none` or a length that is at least 0, matching the catalog syntax. A negative length does not replace an earlier valid value.
- [x] Storage: stylesheet `none` or length on the resolved style.
- [x] Consumer: bake `pathLength` only after a probe shows the canvas changes dash pixels. If the probe shows no difference, leave the attribute unbaked and leave this row open.
- [x] Behavior test: `TestBehaviorSvgPresentPathLength`. Either a pixel diff between `100` and `none` on the dashed line, or the row stays open and the inert pin stays. Proof: `TestBehaviorSvgPresentPathLength` passed: `path-length:112` on a 56 unit line equals a hand-written half-size dash pattern, and `TestEmulateShapeLength` covers each shape. The emulation is `internal/svg/present.go`. Catalog status is `implemented`.
- [x] Catalog: `implemented` only after that pixel diff. The inert pin is not the consumer. Proof: `TestBehaviorSvgPresentPathLength` passed: `path-length:112` on a 56 unit line equals a hand-written half-size dash pattern, and `TestEmulateShapeLength` covers each shape. The emulation is `internal/svg/present.go`. Catalog status is `implemented`.

### image-rendering

Catalog row `image-rendering` (`testdata/css/catalog/properties.json:10742`) is `unsupported`. Syntax `auto | optimizeQuality | optimizeSpeed`. Handlers empty. Tests empty. Handler gap. The syntax does not list `pixelated` or `crisp-edges`. Do not add them.

Current behavior. No apply handler and no bake. A search of `internal/layout` finds no `image-rendering` reader. `opExtra` has image bytes and pixel size, and no rendering keyword (`internal/layout/op_extra.go:11-14`).

What to expect. An `img` or an inline svg with `image-rendering: optimizeSpeed` stores that keyword. A drawing-list test reads it back from the image op. The page is still the drawing list. A bad token such as `pixelated` leaves the previous value. Do not add a resampler. When an image is already re-encoded because of orientation or a crop (`internal/layout/layout_images.go:389-413`), `optimizeSpeed` may pick the cheaper filter on that existing path. Otherwise the keyword rides on the image op for a caller.

- [ ] Acceptance: `auto`, `optimizeQuality`, and `optimizeSpeed` only.
- [ ] Storage: keyword on the resolved style of an `img` or `svg`, in new file `internal/layout/image_rendering.go`.
- [ ] Consumer: no new resampler. Optional cheaper filter on the existing re-encode path. Otherwise the keyword is on the image op extra so a caller can read it.
- [ ] Behavior test: `TestBehaviorSvgPresentImageRendering`. The three keywords store, a bad token does not, and a drawing-list test reads the keyword back from the image op.
- [ ] Catalog: `implemented` only after that drawing-list read. Storage alone fails the seed rule (`testdata/css/catalog/properties.json:9`).

### glyph-orientation-vertical

Catalog row `glyph-orientation-vertical` (`testdata/css/catalog/properties.json:10097`) is `unsupported`. Syntax `auto | 0deg | 90deg | 0 | 90`. Handlers empty. Tests empty. Handler gap.

Current behavior. No apply handler and no bake. SVG text is removed before parse (`internal/svg/raster.go:384-386`), so a rotation of those glyphs cannot show up in the PNG.

What to expect. On vertical SVG text, `glyph-orientation-vertical: 90deg` is stored and, once a bake exists, written as an attribute. `90` and `90deg` store one canonical value. The PNG matches the same SVG without the property, because the text element is gone before parse. HTML text does not rotate for this property.

- [ ] Acceptance: those five tokens. `90` and `90deg` store one canonical value.
- [ ] Storage: stylesheet keyword.
- [ ] Consumer: bake the attribute on vertical SVG text.
- [~] Pixel rotation of SVG glyphs. Reason: `stripSVGTextElements` removes the text before parse (`internal/svg/raster.go:384-386`, `internal/svg/strip_text.go:15-20`). Owner: this ledger. Next gate: the SVG text drawer named on `text-rendering`. Until that drawer exists, do not remove the strip.
- [ ] Behavior test: `TestBehaviorSvgPresentGlyphOrientationVertical`. The attribute is in the XML and the PNG matches the same SVG without the property.
- [ ] Catalog: stay `unsupported` while the only consumer is a serialize of stripped text.

### baseline-shift

Catalog row `baseline-shift` (`testdata/css/catalog/properties.json:2949`) is `unsupported`. Syntax `<length-percentage> | sub | super | top | center | bottom`. Handlers empty. Tests empty. Handler gap.

Current behavior. No apply handler and no bake. HTML `vertical-align: sub` and `super` already shift a run by a fraction of the font size (`internal/layout/inline_vertical_align.go:54-58`). That is a different property. Do not write `baseline-shift` into `VerticalAlign`.

What to expect on HTML. A paragraph at 16px with `baseline-shift: 4px` moves the text op Y above the same paragraph without it. `baseline-shift: super` moves it too. The shift is added in `effectiveVerticalAlignShift` (`internal/layout/inline_vertical_align.go:49-70`) from its own field. Sharing the shift helper with `vertical-align` is fine. Sharing the property is not.

What to expect on SVG text. The attribute is present on the serialized `text` element, and the PNG matches the same text without the property, because the element is removed before parse.

- [ ] Acceptance: a length or percentage, or one of `sub`, `super`, `top`, `center`, `bottom`.
- [ ] Storage: its own field. Do not write it into `VerticalAlign`.
- [ ] Consumer: for HTML text, add the stored shift in `effectiveVerticalAlignShift` (`internal/layout/inline_vertical_align.go:49`). For SVG text, bake the attribute.
- [~] SVG text pixel movement. Reason: text is stripped before parse (`internal/svg/raster.go:384-386`). Owner: this ledger. Next gate: the SVG text drawer named on `text-rendering`. The HTML text-op move can land before that drawer exists.
- [ ] Behavior test: `TestBehaviorSvgPresentBaselineShift`. HTML `baseline-shift: 4px` moves the text op Y. HTML `super` moves it. SVG text rasters match with and without the property, and the attribute is present.
- [ ] Catalog: `implemented` after the HTML text-op test. The deferred SVG pixels do not block that. The HTML shift is a real consumer.

### marker-side

Catalog row `marker-side` (`testdata/css/catalog/properties.json:12095`) is `unsupported`. Syntax `match-self | match-parent`. Handlers empty. Tests empty. Handler gap. This is a different property from `marker-start`. The marker bake loop names `marker-start`, `marker-mid`, and `marker-end` only (`internal/layout/layout_svg.go:1822`).

Current behavior. No apply handler and no bake.

What to expect. Both keywords serialize onto the element that declares them. A bad token does not. A pixel diff is recorded only when a probe shows the canvas changes the marker. If the probe shows no difference, do not invent a marker coordinate system. Leave the pixel row open and the catalog `unsupported`.

- [ ] Acceptance: `match-self` and `match-parent` only.
- [ ] Storage: keyword on the resolved style.
- [ ] Consumer: bake the attribute. Then probe whether the canvas changes pixels.
- [ ] Behavior test: `TestBehaviorSvgPresentMarkerSide`. Both keywords serialize, a bad token does not, and a pixel diff is recorded only when the probe shows one.
- [ ] Catalog: `implemented` only after a pixel diff.

**Gate:** a pixel row the canvas cannot show stays open. Do not close it with a comment.

## Phase 4: draft longhands

`TestBehaviorSvgFillStrokeColorInert` requires `fill-color` and `stroke-color` to leave the painted-pixel count unchanged (`internal/layout/css_behavior_svg_markers_test.go:157-177`). `TestBehaviorSvgDraftPaintLonghandsInert` requires the same for the other declarations in its two slices (`internal/layout/css_behavior_svg_markers_test.go:193-240`). The fill-color test comment says Chrome 143 ignores `fill-color` and `stroke-color`, and aliasing them to `fill` and `stroke` would diverge from that (`internal/layout/css_behavior_svg_markers_test.go:151-156`). Several inert tokens are outside the catalog grammar (`stroke-align: inside`, `stroke-dashadjust: round`). Those rows prove the string does not change pixels. They do not prove the catalog value is accepted.

`stroke-boundary`, `stroke-limit`, and the old "leave path-length alone" note are not the work of this phase. `path-length` has its own heading in phase 3. `stroke-boundary` and `stroke-limit` are not in the unsupported list for this ledger. Leave them in the inert test. Do not add catalog rows for them here.

Shared rule: new code in `internal/layout/svg_presentation.go` plus `internal/svg/present.go`. Do not grow `internal/layout/layout.go`.

- [~] Do not alias any name in this phase onto `fill`, `stroke`, or `stroke-dasharray`. Reason: the inert tests require the same painted-pixel count, and copying the declaration into those fields would change pixels. Owner: this ledger. Next gate: a paint-server or dash model that reads each name from its own field, with a pixel test that replaces the matching inert pin in the same change. Until that gate, the catalog stays `unsupported` and the inert tests stay.

The subsections are the work if that gate is built. Each name repeats the alias deferral with its own reason.

### fill-break

Catalog row `fill-break` (`testdata/css/catalog/properties.json:8524`) is `unsupported`. Syntax `bounding-box | slice | clone`. Handlers empty. Tests empty. Handler gap. The inert test does not mention `fill-break`.

What to expect. The three keywords store. `bogus` does not. The PNG of a filled rect matches the same rect without the property. Blinkless does not split an SVG fill across pages inside the raster. The keyword can ride on the image op extra (`internal/layout/op_extra.go:11`) for a caller. That extra field is a consumer only when a drawing-list test reads it back. Do not add a page encoder.

- [~] Do not store `fill-break` in `Fill`. Reason: the catalog value is a fragment keyword, and `applySVGPresentationProps` has no `fill-break` case (`internal/layout/style_paint_props.go:92-128`), so the declaration is ignored today. Writing it into `Fill` would paint a color from a keyword. Owner: this ledger. Next gate: the op-extra read described in the behavior row.
- [ ] Acceptance: `bounding-box`, `slice`, and `clone` only.
- [ ] Storage: stylesheet keyword on its own field.
- [ ] Consumer: record the keyword on the image op extra. No page splitter.
- [ ] Behavior test: `TestBehaviorSvgPresentFillBreak`. The three keywords store, a bad token does not, and the PNG matches the same SVG without the property. A drawing-list test reads the keyword back.
- [ ] Catalog: leave `unsupported` until that drawing-list read exists.

### fill-color

Catalog row `fill-color` (`testdata/css/catalog/properties.json:8540`) is `unsupported`. Syntax `<color>`. Handlers empty. Tests empty. Handler gap. The inert test is not listed on the row.

What to expect, already asserted. `<rect x="8" y="8" width="48" height="24" style="fill:#ff0000"/>` and the same rect with `fill-color:#0000ff` added paint the same pixel count. That is `TestBehaviorSvgFillStrokeColorInert` (`internal/layout/css_behavior_svg_markers_test.go:160-166`).

What to expect if a paint layer is built. The layer reads `fill-color` from its own field. A rect with `fill-color: #0000ff` and no `fill` paints blue. Until that layer exists, the inert pin stays the contract.

- [x] `TestBehaviorSvgFillStrokeColorInert` requires `fill-color:#0000ff` beside `fill:#ff0000` to keep the painted-pixel count (`internal/layout/css_behavior_svg_markers_test.go:160-166`).
- [~] Do not store `fill-color` in `Fill` and do not bake it as `fill`. Reason: the test above would fail, and the test comment says Chrome ignores the longhand (`internal/layout/css_behavior_svg_markers_test.go:151-156`). Owner: this ledger. Next gate: a paint layer in `internal/svg/present.go` that reads a fill-color field, with a pixel test in the same change that retires this pin.
- [ ] Acceptance: a color the existing color parser accepts.
- [ ] Storage: its own field, after the next gate says to store it.
- [ ] Consumer: no raster change until that paint layer exists.
- [ ] Behavior test: keep `TestBehaviorSvgFillStrokeColorInert` green. Add `TestBehaviorSvgPresentFillColor` only after the gate.
- [ ] Catalog: stay `unsupported` while the consumer is "do not paint".

### fill-image

Catalog row `fill-image` (`testdata/css/catalog/properties.json:8556`) is `unsupported`. Syntax `<paint>#`. Handlers empty. Tests empty. Handler gap.

What to expect, already asserted. The rect `fill:#0645ad` plus `fill-image: url(x.png)` paints the same pixel count (`internal/layout/css_behavior_svg_markers_test.go:202`, comparison `internal/layout/css_behavior_svg_markers_test.go:225-230`).

What to expect if the paint layer is built. A solid image the server can raster differs in pixels from `fill-image: none`. The HTML `background-image` painter is not the SVG fill.

- [x] `TestBehaviorSvgDraftPaintLonghandsInert` requires `fill-image: url(x.png)` to keep the rect's painted-pixel count (`internal/layout/css_behavior_svg_markers_test.go:202` and `internal/layout/css_behavior_svg_markers_test.go:225-230`).
- [~] Do not store `fill-image` in `Fill`. Reason: the inert pin would fail if the url were painted as a fill color, and the property is a paint list of its own. Owner: this ledger. Next gate: the paint layer in `internal/svg/present.go` with `TestBehaviorSvgPresentFillImage`.
- [ ] Acceptance: a list of paints. `none` clears the list.
- [ ] Storage: the list on the resolved style, not in `Fill`.
- [ ] Consumer: the paint layer fills from the image.
- [ ] Behavior test: keep the inert pin until `TestBehaviorSvgPresentFillImage` shows a pixel diff between a rasterizable image and `none`.
- [ ] Catalog: `implemented` only after that pixel diff.

### fill-origin

Catalog row `fill-origin` (`testdata/css/catalog/properties.json:8595`) is `unsupported`. Syntax `match-parent | fill-box | stroke-box | content-box | padding-box | border-box`. Handlers empty. Tests empty. Handler gap.

What to expect, already asserted. `fill-origin: fill-box` on the `#0645ad` rect keeps the pixel count (`internal/layout/css_behavior_svg_markers_test.go:203`).

What to expect with a fill image. Two origins on the same `fill-image` differ in pixels. Without `fill-image`, the keyword does not change pixels.

- [x] `TestBehaviorSvgDraftPaintLonghandsInert` requires `fill-origin: fill-box` to keep the pixel count (`internal/layout/css_behavior_svg_markers_test.go:203` and `internal/layout/css_behavior_svg_markers_test.go:225-230`).
- [~] Do not alias `fill-origin` onto `fill`. Reason: it is an origin box for `fill-image`, and the inert pin shows no pixel change on a solid fill. Owner: this ledger. Next gate: a pixel diff between two origins on the same `fill-image`.
- [ ] Acceptance: those six keywords.
- [ ] Storage: keyword on the resolved style.
- [ ] Consumer: the paint layer uses the keyword as the origin box for `fill-image`.
- [ ] Behavior test: `TestBehaviorSvgPresentFillOrigin`. Two origins differ in pixels, or the row stays open.
- [ ] Catalog: `implemented` only together with a `fill-image` pixel test.

### fill-position

Catalog row `fill-position` (`testdata/css/catalog/properties.json:8611`) is `unsupported`. Syntax `<position>#`. Handlers empty. Tests empty. Handler gap.

What to expect, already asserted. `fill-position: center` keeps the pixel count (`internal/layout/css_behavior_svg_markers_test.go:204`).

What to expect with a fill image. `0 0` versus `center` differs once `fill-image` paints.

- [x] `TestBehaviorSvgDraftPaintLonghandsInert` requires `fill-position: center` to keep the pixel count (`internal/layout/css_behavior_svg_markers_test.go:204` and `internal/layout/css_behavior_svg_markers_test.go:225-230`).
- [~] Do not alias `fill-position` onto `fill`. Reason: it offsets `fill-image` inside the origin box. Owner: this ledger. Next gate: a pixel diff between `0 0` and `center` on a painted `fill-image`.
- [ ] Acceptance: the position syntax the background position parser already accepts. Reject a token that parser rejects.
- [ ] Storage: one position per paint layer.
- [ ] Consumer: offsets `fill-image` inside the origin box.
- [ ] Behavior test: `TestBehaviorSvgPresentFillPosition`. `0 0` versus `center` differs once `fill-image` paints.
- [ ] Catalog: `implemented` only with that pixel test.

### fill-repeat

Catalog row `fill-repeat` (`testdata/css/catalog/properties.json:8627`) is `unsupported`. Syntax `<repeat-style>#`. Handlers empty. Tests empty. Handler gap.

What to expect, already asserted. `fill-repeat: repeat` keeps the pixel count (`internal/layout/css_behavior_svg_markers_test.go:205`).

What to expect with a fill image. `repeat` versus `no-repeat` differs once `fill-image` paints. `no-repeat` paints one tile.

- [x] `TestBehaviorSvgDraftPaintLonghandsInert` requires `fill-repeat: repeat` to keep the pixel count (`internal/layout/css_behavior_svg_markers_test.go:205` and `internal/layout/css_behavior_svg_markers_test.go:225-230`).
- [~] Do not alias `fill-repeat` onto `fill`. Reason: it repeats `fill-image`. Owner: this ledger. Next gate: a pixel diff between `repeat` and `no-repeat` on a painted `fill-image`.
- [ ] Acceptance: the background repeat keywords.
- [ ] Storage: one repeat per paint layer.
- [ ] Consumer: repeats `fill-image`.
- [ ] Behavior test: `TestBehaviorSvgPresentFillRepeat`. `repeat` versus `no-repeat` differs once `fill-image` paints.
- [ ] Catalog: `implemented` only with that pixel test.

### fill-size

Catalog row `fill-size` (`testdata/css/catalog/properties.json:8674`) is `unsupported`. Syntax `<bg-size>#`. Handlers empty. Tests empty. Handler gap.

What to expect, already asserted. `fill-size: cover` keeps the pixel count (`internal/layout/css_behavior_svg_markers_test.go:206`).

What to expect with a fill image. `cover` versus a fixed length differs once `fill-image` paints.

- [x] `TestBehaviorSvgDraftPaintLonghandsInert` requires `fill-size: cover` to keep the pixel count (`internal/layout/css_behavior_svg_markers_test.go:206` and `internal/layout/css_behavior_svg_markers_test.go:225-230`).
- [~] Do not alias `fill-size` onto `fill`. Reason: it sizes `fill-image` before repeat. Owner: this ledger. Next gate: a pixel diff between `cover` and a fixed length on a painted `fill-image`.
- [ ] Acceptance: the background size syntax, including `cover` and `contain`.
- [ ] Storage: one size per paint layer.
- [ ] Consumer: sizes `fill-image` before repeat.
- [ ] Behavior test: `TestBehaviorSvgPresentFillSize`. `cover` versus a fixed length differs once `fill-image` paints.
- [ ] Catalog: `implemented` only with that pixel test.

### stroke-align

Catalog row `stroke-align` (`testdata/css/catalog/properties.json:17239`) is `unsupported`. Syntax `center | inset | outset`. Handlers empty. Tests empty. Handler gap.

What to expect, already asserted. The rect carries `stroke-align: inside` (`internal/layout/css_behavior_svg_markers_test.go:212`). `inside` is not in the catalog grammar. The painted-pixel count matches the rect without it. That does not prove `center`, `inset`, or `outset`.

What to expect if the one-sided stroke is built. On a wide stroke, `inset` versus `center` differs in pixels. `outset` shifts the stroke to the other side. `center` stays the canvas default. `inside` stays invalid.

- [x] `TestBehaviorSvgDraftPaintLonghandsInert` requires the non-catalog token `inside` to keep the pixel count (`internal/layout/css_behavior_svg_markers_test.go:212` and `internal/layout/css_behavior_svg_markers_test.go:225-230`).
- [~] Do not alias `stroke-align` onto `stroke`, and do not store it in the `stroke-alignment` field. Reason: the catalog keywords shift the stroke to one side, and `stroke-alignment` is a separate row. Owner: this ledger. Next gate: a pixel diff between `inset` and `center` on a wide stroke.
- [ ] Acceptance: `center`, `inset`, and `outset` only. `inside` stays invalid.
- [ ] Storage: keyword on its own field.
- [ ] Consumer: `inset` and `outset` shift the stroke in `internal/svg/present.go`. `center` stays the canvas default.
- [ ] Behavior test: `TestBehaviorSvgPresentStrokeAlign`. `inset` versus `center` on a wide stroke differs in pixels. The `inside` pin can stay as an invalid-value test.
- [ ] Catalog: `implemented` only after that pixel diff.

### stroke-alignment

Catalog row `stroke-alignment` (`testdata/css/catalog/properties.json:17255`) is `unsupported`. Syntax `center | inner | outer`. Handlers empty. Tests empty. Handler gap. No opened test declares this name. It is a different property from `stroke-align`.

What to expect. `inner` versus `center` on a wide stroke differs in pixels, on the same shape used for `stroke-align`. `inner` uses the inset side and `outer` uses the outset side, in one helper. The stored keywords stay distinct so the declared name round-trips.

- [~] Do not alias `stroke-alignment` onto `stroke` or onto `stroke-align`. Reason: the catalog tokens are `center | inner | outer`, and closing this row by pointing at `stroke-align` would hide a second property. Owner: this ledger. Next gate: `TestBehaviorSvgPresentStrokeAlignment` shows `inner` versus `center` differing in pixels.
- [ ] Acceptance: `center`, `inner`, and `outer` only.
- [ ] Storage: its own field, not the `stroke-align` field.
- [ ] Consumer: `inner` and `outer` use the one-sided stroke helper. Map `inner` to the inset side and `outer` to the outset side. Keep two stored keywords.
- [ ] Behavior test: `TestBehaviorSvgPresentStrokeAlignment`. `inner` versus `center` differs in pixels.
- [ ] Catalog: `implemented` only after that test. Do not close it by pointing at `stroke-align`.

### stroke-break

Catalog row `stroke-break` (`testdata/css/catalog/properties.json:17271`) is `unsupported`. Syntax `bounding-box | slice | clone`. Handlers empty. Tests empty. Handler gap.

What to expect, already asserted. `stroke-break: bounding-box` on the filled rect keeps the pixel count (`internal/layout/css_behavior_svg_markers_test.go:213`).

What to expect once stored. The keyword is on the image op extra, and the PNG is unchanged. Same consumer shape as `fill-break`. No page splitter.

- [x] `TestBehaviorSvgDraftPaintLonghandsInert` requires `stroke-break: bounding-box` to keep the pixel count (`internal/layout/css_behavior_svg_markers_test.go:213` and `internal/layout/css_behavior_svg_markers_test.go:225-230`).
- [~] Do not alias `stroke-break` onto `stroke`. Reason: it is a fragment keyword, and the inert pin shows no pixel change. Owner: this ledger. Next gate: a drawing-list read of the extra field.
- [ ] Acceptance: the three catalog keywords.
- [ ] Storage: keyword on its own field.
- [ ] Consumer: the keyword rides on the image op for a caller. No page splitter.
- [ ] Behavior test: `TestBehaviorSvgPresentStrokeBreak`. The keyword is stored, the PNG is unchanged, and a drawing-list test reads the extra field.
- [ ] Catalog: stay `unsupported` until that drawing-list read exists.

### stroke-color

Catalog row `stroke-color` (`testdata/css/catalog/properties.json:17287`) is `unsupported`. Syntax `<color>#`. Handlers empty. Tests empty. Handler gap.

What to expect, already asserted. A path `d="M8 20 H56"` with `stroke:#ff0000;stroke-width:3` paints the same pixel count when `stroke-color:#0000ff` is added (`internal/layout/css_behavior_svg_markers_test.go:169-176`).

What to expect if the paint layer strokes from its own color list. Until that layer exists, the inert pin stays.

- [x] `TestBehaviorSvgFillStrokeColorInert` requires `stroke-color:#0000ff` beside `stroke:#ff0000` to keep the painted-pixel count (`internal/layout/css_behavior_svg_markers_test.go:169-176`).
- [~] Do not store `stroke-color` in `Stroke` and do not bake it as `stroke`. Reason: the test above would fail, and the test comment says Chrome ignores the longhand (`internal/layout/css_behavior_svg_markers_test.go:151-156`). Owner: this ledger. Next gate: the paint layer paints stroke colors from their own list, with a pixel test in the same change that retires this pin.
- [ ] Acceptance: a list of colors.
- [ ] Storage: its own list.
- [ ] Consumer: no raster change until that layer exists.
- [ ] Behavior test: keep the inert pin until `TestBehaviorSvgPresentStrokeColor` exists.
- [ ] Catalog: stay `unsupported` while it must not change pixels.

### stroke-dash-corner

Catalog row `stroke-dash-corner` (`testdata/css/catalog/properties.json:17303`) is `unsupported`. Syntax `none | <length>`. Handlers empty. Tests empty. Handler gap.

What to expect, already asserted. The dashed line carries `stroke-dash-corner: round` (`internal/layout/css_behavior_svg_markers_test.go:219`). `round` is not in the catalog grammar. The painted-pixel count matches the dashed line without it.

What to expect if the corner length is built. A positive length differs in pixels from `none` on a dashed rect. `round` stays invalid.

- [x] `TestBehaviorSvgDraftPaintLonghandsInert` requires the token `round` to keep the dash pixel count (`internal/layout/css_behavior_svg_markers_test.go:219` and `internal/layout/css_behavior_svg_markers_test.go:233-239`).
- [~] Do not alias `stroke-dash-corner` onto `stroke-dasharray`. Reason: the catalog value is `none` or a length, and the inert token `round` must keep today's dashes. Owner: this ledger. Next gate: a pixel diff between a positive length and `none` on a dashed rect.
- [ ] Acceptance: `none` or a length. `round` stays invalid.
- [ ] Storage: `none` or a length on its own field.
- [ ] Consumer: the length shapes the dash corner in `internal/svg/present.go`. `none` keeps today's dashes.
- [ ] Behavior test: `TestBehaviorSvgPresentStrokeDashCorner`. A positive length differs in pixels from `none`. The `round` pin can stay as an invalid-value test.
- [ ] Catalog: `implemented` only after the pixel diff.

### stroke-dash-justify

Catalog row `stroke-dash-justify` (`testdata/css/catalog/properties.json:17319`) is `unsupported`. Syntax `none | [ stretch | compress ] || [ dashes || gaps ]`. Handlers empty. Tests empty. Handler gap.

What to expect, already asserted. `stroke-dash-justify: stretch` keeps the dash pixel count (`internal/layout/css_behavior_svg_markers_test.go:220`). `stretch` alone is one of the catalog combinations. The pin shows it does not change pixels today.

What to expect if dash fitting is built. `stretch dashes` differs in pixels from `none` on a dashed line. A lone `dashes` or `gaps` without `stretch` or `compress` is invalid.

- [x] `TestBehaviorSvgDraftPaintLonghandsInert` requires `stretch` to keep the dash pixel count (`internal/layout/css_behavior_svg_markers_test.go:220` and `internal/layout/css_behavior_svg_markers_test.go:233-239`).
- [~] Do not alias `stroke-dash-justify` onto `stroke-dasharray` or onto `stroke-dashadjust`. Reason: both dash-fit names exist as separate rows, and the inert pin shows `stretch` leaves the dashes alone. Owner: this ledger. Next gate: a pixel diff for `stretch dashes` versus `none`.
- [ ] Acceptance: the catalog combination. A lone `dashes` or `gaps` without stretch or compress is invalid.
- [ ] Storage: the chosen pair on its own field.
- [ ] Consumer: stretch or compress the dash pattern so the pattern fits the path, in `internal/svg/present.go`.
- [ ] Behavior test: `TestBehaviorSvgPresentStrokeDashJustify`. `stretch dashes` differs in pixels from `none` on a dashed line.
- [ ] Catalog: `implemented` only after that pixel diff.

### stroke-dashadjust

Catalog row `stroke-dashadjust` (`testdata/css/catalog/properties.json:17335`) is `unsupported`. Syntax `none | [stretch | compress] [dashes | gaps]?`. Handlers empty. Tests empty. Handler gap.

What to expect, already asserted. `stroke-dashadjust: round` keeps the dash pixel count (`internal/layout/css_behavior_svg_markers_test.go:221`). `round` is not in the catalog grammar.

What to expect if dash fitting is built. `compress gaps` differs in pixels from `none`. This name is not an alias of `stroke-dash-justify`. Store both if both are declared. The helper reads this value when `stroke-dash-justify` is absent.

- [x] `TestBehaviorSvgDraftPaintLonghandsInert` requires the token `round` to keep the dash pixel count (`internal/layout/css_behavior_svg_markers_test.go:221` and `internal/layout/css_behavior_svg_markers_test.go:233-239`).
- [~] Do not alias `stroke-dashadjust` onto `stroke-dash-justify` or onto `stroke-dasharray`. Reason: the catalog grammar is its own, and `round` must stay invalid. Owner: this ledger. Next gate: a pixel diff for `compress gaps` versus `none`.
- [ ] Acceptance: the catalog grammar. `round` stays invalid.
- [ ] Storage: its own field.
- [ ] Consumer: the same dash-fit helper as `stroke-dash-justify`, using this stored value when `stroke-dash-justify` is absent.
- [ ] Behavior test: `TestBehaviorSvgPresentStrokeDashadjust`. `compress gaps` differs in pixels from `none`.
- [ ] Catalog: `implemented` only after that test, separate from `stroke-dash-justify`.

### stroke-dashcorner

Catalog row `stroke-dashcorner` (`testdata/css/catalog/properties.json:17380`) is `unsupported`. Syntax `none | <length>`. Handlers empty. Tests empty. Handler gap.

What to expect, already asserted. `stroke-dashcorner: round` keeps the dash pixel count (`internal/layout/css_behavior_svg_markers_test.go:222`). `round` is not in the catalog grammar.

What to expect if the corner length is built. A length differs in pixels from `none`. Both this name and `stroke-dash-corner` exist as separate rows. Store both. The consumer reads `stroke-dashcorner` only when `stroke-dash-corner` was not set.

- [x] `TestBehaviorSvgDraftPaintLonghandsInert` requires the token `round` to keep the dash pixel count (`internal/layout/css_behavior_svg_markers_test.go:222` and `internal/layout/css_behavior_svg_markers_test.go:233-239`).
- [~] Do not treat `stroke-dashcorner` as a second spelling of `stroke-dash-corner` in one field, and do not alias it onto `stroke-dasharray`. Reason: the catalog has two rows, and the inert token must not change dashes. Owner: this ledger. Next gate: a pixel diff between a length and `none` from this property's own field.
- [ ] Acceptance: `none` or a length.
- [ ] Storage: its own field.
- [ ] Consumer: the same corner-length helper as `stroke-dash-corner`, used when `stroke-dash-corner` was not set.
- [ ] Behavior test: `TestBehaviorSvgPresentStrokeDashcorner`. A length differs in pixels from `none`.
- [ ] Catalog: `implemented` only after that test, separate from `stroke-dash-corner`.

### stroke-image

Catalog row `stroke-image` (`testdata/css/catalog/properties.json:17425`) is `unsupported`. Syntax `<paint>#`. Handlers empty. Tests empty. Handler gap.

What to expect, already asserted. `stroke-image: url(x.png)` on the filled rect keeps the pixel count (`internal/layout/css_behavior_svg_markers_test.go:207`).

What to expect if the paint layer strokes from an image. A rasterizable `stroke-image` differs from `none`.

- [x] `TestBehaviorSvgDraftPaintLonghandsInert` requires `stroke-image: url(x.png)` to keep the pixel count (`internal/layout/css_behavior_svg_markers_test.go:207` and `internal/layout/css_behavior_svg_markers_test.go:225-230`).
- [~] Do not store `stroke-image` in `Stroke`. Reason: the inert pin shows the url does not change pixels, and the property is a paint list. Owner: this ledger. Next gate: the paint layer with `TestBehaviorSvgPresentStrokeImage`.
- [ ] Acceptance: a list of paints. `none` clears it.
- [ ] Storage: its own list, not `Stroke`.
- [ ] Consumer: the paint layer strokes from the image. Depends on the phase gate.
- [ ] Behavior test: a rasterizable `stroke-image` differs from `none`, or the inert pin stays.
- [ ] Catalog: `implemented` only after that pixel diff.

### stroke-origin

Catalog row `stroke-origin` (`testdata/css/catalog/properties.json:17551`) is `unsupported`. Syntax `match-parent | fill-box | stroke-box | content-box | padding-box | border-box`. Handlers empty. Tests empty. Handler gap.

What to expect, already asserted. `stroke-origin: stroke-box` keeps the pixel count (`internal/layout/css_behavior_svg_markers_test.go:208`).

What to expect with a stroke image. Two origins differ once `stroke-image` paints. Without a stroke image, the keyword does not change pixels.

- [x] `TestBehaviorSvgDraftPaintLonghandsInert` requires `stroke-origin: stroke-box` to keep the pixel count (`internal/layout/css_behavior_svg_markers_test.go:208` and `internal/layout/css_behavior_svg_markers_test.go:225-230`).
- [~] Do not alias `stroke-origin` onto `stroke`. Reason: it is the origin box for `stroke-image`, and the inert pin shows no pixel change. Owner: this ledger. Next gate: two origins differ on a painted `stroke-image`.
- [ ] Acceptance: those six keywords.
- [ ] Storage: keyword on the resolved style.
- [ ] Consumer: origin box for `stroke-image`.
- [ ] Behavior test: `TestBehaviorSvgPresentStrokeOrigin`. Two origins differ once `stroke-image` paints.
- [ ] Catalog: `implemented` only with that pixel test.

### stroke-position

Catalog row `stroke-position` (`testdata/css/catalog/properties.json:17567`) is `unsupported`. Syntax `<position>#`. Handlers empty. Tests empty. Handler gap.

What to expect, already asserted. `stroke-position: center` keeps the pixel count (`internal/layout/css_behavior_svg_markers_test.go:209`).

What to expect with a stroke image. `0 0` versus `center` differs once `stroke-image` paints.

- [x] `TestBehaviorSvgDraftPaintLonghandsInert` requires `stroke-position: center` to keep the pixel count (`internal/layout/css_behavior_svg_markers_test.go:209` and `internal/layout/css_behavior_svg_markers_test.go:225-230`).
- [~] Do not alias `stroke-position` onto `stroke`. Reason: it offsets `stroke-image`. Owner: this ledger. Next gate: `0 0` versus `center` differs on a painted `stroke-image`.
- [ ] Acceptance: the same position syntax as `fill-position`.
- [ ] Storage: one position per stroke paint.
- [ ] Consumer: offsets `stroke-image`.
- [ ] Behavior test: `TestBehaviorSvgPresentStrokePosition`. `0 0` versus `center` differs once `stroke-image` paints.
- [ ] Catalog: `implemented` only with that pixel test.

### stroke-repeat

Catalog row `stroke-repeat` (`testdata/css/catalog/properties.json:17583`) is `unsupported`. Syntax `<repeat-style>#`. Handlers empty. Tests empty. Handler gap.

What to expect, already asserted. `stroke-repeat: repeat` keeps the pixel count (`internal/layout/css_behavior_svg_markers_test.go:210`).

What to expect with a stroke image. `repeat` versus `no-repeat` differs once `stroke-image` paints.

- [x] `TestBehaviorSvgDraftPaintLonghandsInert` requires `stroke-repeat: repeat` to keep the pixel count (`internal/layout/css_behavior_svg_markers_test.go:210` and `internal/layout/css_behavior_svg_markers_test.go:225-230`).
- [~] Do not alias `stroke-repeat` onto `stroke`. Reason: it repeats `stroke-image`. Owner: this ledger. Next gate: `repeat` versus `no-repeat` differs on a painted `stroke-image`.
- [ ] Acceptance: the background repeat keywords.
- [ ] Storage: one repeat per stroke paint.
- [ ] Consumer: repeats `stroke-image`.
- [ ] Behavior test: `TestBehaviorSvgPresentStrokeRepeat`. `repeat` versus `no-repeat` differs once `stroke-image` paints.
- [ ] Catalog: `implemented` only with that pixel test.

### stroke-size

Catalog row `stroke-size` (`testdata/css/catalog/properties.json:17599`) is `unsupported`. Syntax `<bg-size>#`. Handlers empty. Tests empty. Handler gap.

What to expect, already asserted. `stroke-size: cover` keeps the pixel count (`internal/layout/css_behavior_svg_markers_test.go:211`).

What to expect with a stroke image. `cover` versus a fixed length differs once `stroke-image` paints.

- [x] `TestBehaviorSvgDraftPaintLonghandsInert` requires `stroke-size: cover` to keep the pixel count (`internal/layout/css_behavior_svg_markers_test.go:211` and `internal/layout/css_behavior_svg_markers_test.go:225-230`).
- [~] Do not alias `stroke-size` onto `stroke`. Reason: it sizes `stroke-image`. Owner: this ledger. Next gate: `cover` versus a fixed length differs on a painted `stroke-image`.
- [ ] Acceptance: the background size syntax.
- [ ] Storage: one size per stroke paint.
- [ ] Consumer: sizes `stroke-image`.
- [ ] Behavior test: `TestBehaviorSvgPresentStrokeSize`. `cover` versus a fixed length differs once `stroke-image` paints.
- [ ] Catalog: `implemented` only with that pixel test.

**Gate:** the alias deferral stays until a replacement pixel test lands in the same change as the code that paints the longhand. Inert tests stay green until that change.

## Phase 5: filter leftovers

These three are filter primitive properties. This ledger does not build a filter graph. No `feGaussianBlur`, no `feMerge`, and no lighting equation. New bake code still goes in `internal/layout/svg_presentation.go`. A one-element paint, if a probe shows the canvas ignores the attribute, goes in `internal/svg/present.go`.

### flood-color

Catalog row `flood-color` (`testdata/css/catalog/properties.json:9048`) is `unsupported`. Syntax `<color>`. Handlers empty. Tests empty. Handler gap.

Current behavior. No apply handler and no bake. This is an SVG filter primitive, not a CSS flood on a normal box.

What to expect. An `feFlood` with `flood-color: #ff0000` and another with `flood-color: #0000ff` differ in pixels, or the row stays open. A `flood-color` on a `rect` does not bake. If a probe shows the canvas ignores `flood-color`, add a one-element solid fill for `feFlood` only.

- [ ] Acceptance: a color the existing color parser accepts.
- [ ] Storage: stylesheet color on the resolved style.
- [ ] Consumer: bake onto `feFlood` only. One-element solid fill in `internal/svg/present.go` if the canvas ignores the attribute. Do not implement the rest of the filter graph.
- [ ] Behavior test: `TestBehaviorSvgPresentFloodColor`. Two `feFlood` colors differ in pixels, or the row stays open. A `flood-color` on a `rect` does not bake.
- [ ] Catalog: `implemented` only after the pixel diff.

### flood-opacity

Catalog row `flood-opacity` (`testdata/css/catalog/properties.json:9064`) is `unsupported`. Syntax `<'opacity'>`. Handlers empty. Tests empty. Handler gap.

Current behavior. No apply handler and no bake.

What to expect. `flood-opacity: 0.2` on an `feFlood` differs in pixels from an opaque flood. The opacity parser is the one `fill-opacity` already uses (`internal/layout/style_paint_props.go:102-105`). The one-element flood fill multiplies by this alpha.

- [ ] Acceptance: the same opacity syntax as `fill-opacity` (`internal/layout/style_paint_props.go:102-105`).
- [ ] Storage: a 0 to 1 alpha on the `feFlood` style.
- [ ] Consumer: the same one-element `feFlood` fill multiplies by this alpha. No filter graph.
- [ ] Behavior test: `TestBehaviorSvgPresentFloodOpacity`. `flood-opacity: 0.2` differs in pixels from an opaque flood.
- [ ] Catalog: `implemented` only after that pixel diff.

### lighting-color

Catalog row `lighting-color` (`testdata/css/catalog/properties.json:11343`) is `unsupported`. Syntax `<color>`. Handlers empty. Tests empty. Handler gap.

Current behavior. No apply handler and no bake.

What to expect. The attribute is serialized on `feDiffuseLighting` and `feSpecularLighting`. A pixel assertion is allowed only if a probe shows the canvas already changes pixels. Do not fake a light. If the canvas ignores the attribute, stop at the serialize gap the way `paint-order` does, and leave the catalog `unsupported`.

- [ ] Acceptance: a color the existing parser accepts.
- [ ] Storage: stylesheet color.
- [ ] Consumer: bake onto `feDiffuseLighting` and `feSpecularLighting` only. Do not build a lighting equation.
- [ ] Behavior test: `TestBehaviorSvgPresentLightingColor`. The attribute is serialized. A pixel assertion is allowed only if a probe shows the canvas already changes pixels.
- [ ] Catalog: stay `unsupported` on a serialize-only result.

**Gate:** filter pixel rows the canvas cannot show stay open.

## Phase 6: catalog and closure

- [ ] Update `testdata/css/catalog/properties.json` only for names whose behavior test and consumer landed in that change. Handlers point at `svg_presentation.go` or the existing applier. Tests point at the new behavior test.
- [ ] Regenerate the compatibility matrix property section so it matches the catalog. `make matrix-check` runs `scripts/check-matrix-sync.sh` (`Makefile:131-132`).
- [ ] Do not edit `documentation/deferred.md` in the implementation change unless the user asks. This checklist is the planning home.
- [ ] Do not grow `internal/layout/layout.go`. `make lint` runs the file-size check (`Makefile:52-57`).
- [ ] Run `make test` (`Makefile:31-32`). Record the exit code. Do not run bare `go test ./...`.
- [ ] Run `make golden`. It runs `TestDisplay` in `./layout` (`Makefile:149-150`). It does not rasterize a page.
- [ ] Run `make lint`.
- [ ] Run `make claim-scan` (`Makefile:102-112`). Do not add a page-PNG claim, a PDF claim, or a Chrome pixel-parity claim for a name whose test only checks XML.
- [ ] Run `make catalog-check` (`Makefile:119-120`) and `make matrix-check`.
- [ ] GitHub CI runs `make test` and `make lint` (`.github/workflows/ci.yml:19-26`). The other gates are local.

These commands were not run for this documentation file.

Focused run during the implementation, one package at a time:

```sh
go test ./internal/layout -run 'TestBehaviorSvg...'
```

**Gate:** open until every command above exits 0 on the change that flips catalog rows.

## Dependencies

- Future focused run: `go test ./internal/layout -run 'TestBehaviorSvg...'`. Later, on the change that flips catalog rows: `make test`, `make golden`, `make lint`, `make claim-scan`, `make catalog-check`, `make matrix-check`. None of these were run while writing this file.
- Cascade entry for the implemented names is `applySVGPresentationProps` (`internal/layout/style_paint_props.go:91`) and `applyLeftoversProps` (`internal/layout/style_leftovers.go:11`).
- Inline SVG serialization is `internal/layout/layout_svg.go`. New bake arms go in `internal/layout/svg_presentation.go` and are called from `writeSVGNode`.
- Raster entry is `RasterizeContext` (`internal/svg/raster.go:85`). `emulateDroppedPresentation` covers `fill-rule` and `shape-rendering` (`internal/svg/raster.go:615-640`). Further emulation goes in `internal/svg/present.go`, which does not exist yet.
- Text baseline shift for `alignment-baseline` and `dominant-baseline` is `internal/layout/inline_vertical_align.go`. Leave that path alone except for the `baseline-shift` HTML hook.
- The property catalog is the support record (`testdata/css/catalog/properties.json`). The matrix must match it (`Makefile:127-132`).
- This file supersedes the open rows in `plans/v0.0.3/svg-presentation/svg-presentation-checklist.md`. That file was not edited.
