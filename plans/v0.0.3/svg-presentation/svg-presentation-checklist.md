# v0.0.3 - SVG presentation

> **Parent:** `plans/v0.0.3/README.md`
> **Status:** not started. Baseline rows record verified current behavior. Implementation rows are open.
> **Estimated effort:** 61 catalog properties (18 implemented, 43 open) across 7 phases.

---

## Overview

Blinkless returns an ordered display list. An inline `<svg>` is turned into one image operation whose bytes are a PNG. `internal/svg` rasterizes that one payload. The page is not encoded as a PNG, and this plan does not add a page rasterizer.

The catalog lists 61 names for this ledger. Eighteen are `implemented`. Forty-three are `unsupported`. `documentation/deferred.md` still says only five of these are parsed and the other 53 are a hard defer. That sentence is stale. This file is the planning home. Do not edit `deferred.md`.

Two paths exist today, and they are not the same:

- Cascade storage. `applySVGPresentationProps` (`internal/layout/style_paint_props.go:91`) writes fields on the resolved style. The serializer then copies those fields onto the SVG element as attributes (`internal/layout/layout_svg.go:182`). A stylesheet rule works for this path.
- Style-attribute bake. Some names have no resolved-style field. The serializer reads the raw `style` attribute and copies a valid token onto the element (`internal/layout/layout_svg.go:265-344` and `internal/layout/layout_svg.go:619`). A stylesheet rule does not reach this path. The catalog therefore still says `unsupported` even when an inline style already changes pixels.

SVG text is removed before the rasterizer runs (`internal/svg/raster.go:189-191`). A text property can be stored or written into the XML and still not move pixels. That limit is called out on the text rows. It is not a reason to load a system font under the canvas lock.

Ruby is out of scope.

## Executive summary

| Group | Count | What this file does |
|---|---|---|
| Catalog implemented | 18 | Phase 0 baseline only, plus a short hole list where the opened tests do not cover a stylesheet or a `url()` paint |
| Inline style already changes pixels, catalog still unsupported | 13 | Own subsection. Promote storage so a stylesheet reaches the bake that already exists |
| Inline style is serialized, raster does not change | 3 | `paint-order`, `vector-effect`, `text-rendering` |
| Draft longhands pinned as pixel-inert | 18 | Own subsection. Do not alias them onto `fill` or `stroke` |
| No test and no bake | 9 | Own subsection, from acceptance through catalog |

The 13 pixel bakes are `cx`, `cy`, `r`, `rx`, `ry`, `x`, `y`, `d`, `marker-end`, `marker-mid`, `marker-start`, `stop-color`, and `stop-opacity`. The 9 with no bake are `baseline-shift`, `fill-break`, `flood-color`, `flood-opacity`, `glyph-orientation-vertical`, `image-rendering`, `lighting-color`, `marker-side`, and `stroke-alignment`.

## Phase 0: contract and baseline

### 0.1 Shared contract

- [x] An inline SVG becomes one `OpImage`. `buildInlineSVG` rasterizes the serialized tree and emits the PNG (`internal/layout/layout_svg.go:15-54`). `Rasterize` is the entry (`internal/svg/raster.go:62`). The public kinds are fill rect, stroke rect, line, text, image, link, bullet, and grid run (`layout/displaylist.go:80-99`). There is no page image.
- [x] There is no vector path operation. The kind list stops at `OpGridRun` (`internal/layout/layout.go:369-385`). SVG geometry stays inside the raster. This ledger does not add a path kind. Corner shapes are a different file.
- [x] Fill and stroke colors, opacities, width, dash, cap, join, and miter limit are copied from resolved style onto attributes (`internal/layout/layout_svg.go:182-237`).
- [x] `d` is the exception that already lets CSS replace the attribute. The serializer drops the `d` attribute when the style attribute carries `d`, then writes one `d` (`internal/layout/layout_svg.go:140-148`, `internal/layout/layout_svg.go:640-650`). `TestBehaviorSvgCSSDOverridesDAttribute` requires the CSS shape, not the attribute shape (`internal/layout/css_behavior_svg_markers_test.go:132`).
- [x] Other baked names keep the presentation attribute and refuse a second copy. `TestBehaviorSvgGeoAttrWinsNoDuplicate` requires `cx="10"` to survive `style="cx:40px"` (`internal/layout/css_behavior_svg_geo_test.go:222`). Chrome would let the CSS win. The test pins the opposite on purpose.
- [ ] Make CSS win over the presentation attribute for every name this ledger bakes, using the `d` rule as the pattern. Update `TestBehaviorSvgGeoAttrWinsNoDuplicate` in the same change. Do not do this one name at a time inside `layout.go`.
- [ ] Store stylesheet values for the names that today read only `node.Attribute("style")`. New file `internal/layout/svg_presentation.go`. The bake reads the stored value. Do not grow `internal/layout/layout.go`.
- [ ] Do not add a page rasterizer, a general SVG filter graph, or a system-font load under the canvas lock.

**Gate:** the two `[x]` contract rows above are the baseline. The two open rows block every subsection that says "phase 0 CSS-wins rule" or "stylesheet store".

### 0.2 Implemented baselines

Each row is the behavior the opened test actually checks. Older comments that say `applyLeftoversProps` has no case are stale: the cases are in `internal/layout/style_leftovers.go:11-64`.

- [x] `fill`. A hex color is stored and baked as `fill="#ff0000"`, and the SVG emits an image op. `none` stores an empty color and zero opacity (`internal/layout/style_paint_props.go:93-101`, `internal/layout/layout_svg.go:188-194`). Proof: `TestBehaviorFillBakesPaint` (`internal/layout/css_behavior_svg_paint_test.go:208`).
- [x] `fill-opacity`. `0.4` is stored and baked (`internal/layout/style_paint_props.go:102-105`, `internal/layout/layout_svg.go:195-197`). Proof: `TestBehaviorFillOpacityBakesPaint` (`internal/layout/css_behavior_svg_paint_test.go:235`). The bake skips the attribute when opacity is 1.
- [x] `fill-rule`. `nonzero` and `evenodd` are stored (`internal/layout/style_leftovers.go:273-277`). Inline style is baked from the style attribute, not from the stored field (`internal/layout/layout_svg.go:243-249`). The canvas parser drops the attribute, so `internal/svg` reverses a two-subpath hole before parse (`internal/svg/raster.go:184-188`, `internal/svg/raster.go:413-427`). Proof: `TestPaintOutputFillRuleEvenOddHolesRaster` (`internal/layout/css_behavior_paint_output_test.go:29`) requires a transparent hole. `TestBehaviorFillRuleNoPaintEffect` (`internal/layout/css_behavior_svg_paint_test.go:261`) only compares op geometry, not PNG bytes, so it stays green.
- [x] `stroke`. A hex color is stored and baked (`internal/layout/style_paint_props.go:106-110`, `internal/layout/layout_svg.go:198-200`). Proof: `TestBehaviorStrokeBakesPaint` (`internal/layout/css_behavior_svg_paint_test.go:133`). There is no `none` arm.
- [x] `stroke-opacity`. `0.3` is stored and baked (`internal/layout/style_paint_props.go:116-119`, `internal/layout/layout_svg.go:204-206`). Proof: `TestBehaviorStrokeOpacityBakesPaint` (`internal/layout/css_behavior_svg_paint_test.go:160`).
- [x] `stroke-width`. `3pt` is stored and baked (`internal/layout/style_paint_props.go:111-115`, `internal/layout/layout_svg.go:201-203`). Proof: `TestBehaviorStrokeWidthBakesPaint` (`internal/layout/css_behavior_svg_paint_test.go:182`). Width 0 is not written.
- [x] `stroke-dasharray`. Lengths are stored (`internal/layout/style_leftovers.go:13-14`, `internal/layout/style_leftovers.go:67-80`) and baked (`internal/layout/layout_svg.go:207-215`). Proof: `TestBehaviorSvgBakeDasharrayDashesStroke` (`internal/layout/css_behavior_svg_bake_test.go:137`) requires a pixel diff. `TestLeftoversPropsWave5` (`internal/layout/style_leftovers_test.go:9`) checks storage only.
- [x] `stroke-dashoffset`. A length is stored and baked (`internal/layout/style_leftovers.go:15-18`, `internal/layout/layout_svg.go:216-218`). Proof: `TestBehaviorSvgBakeDashoffsetShiftsPhase` (`internal/layout/css_behavior_svg_bake_test.go:158`).
- [x] `stroke-linecap`. The token is stored lowercased. Only `butt`, `round`, and `square` are baked (`internal/layout/style_leftovers.go:19-20`, `internal/layout/layout_svg.go:219-224`). Proof: `TestBehaviorSvgBakeLinecapRoundsEnds` (`internal/layout/css_behavior_svg_bake_test.go:178`).
- [x] `stroke-linejoin`. Stored lowercased. Baked values are `miter`, `miter-clip`, `round`, `bevel`, and `arcs` (`internal/layout/style_leftovers.go:21-22`, `internal/layout/layout_svg.go:232-237`). Proof: `TestBehaviorSvgBakeLinejoinRoundsCorners` (`internal/layout/css_behavior_svg_bake_test.go:199`).
- [x] `stroke-miterlimit`. A number `>= 1` is stored and baked before linejoin (`internal/layout/style_leftovers.go:23-26`, `internal/layout/layout_svg.go:225-231`). Proof: `TestBehaviorSvgBakeMiterlimitClipsSpike` (`internal/layout/css_behavior_svg_bake_test.go:221`).
- [x] `shape-rendering`. The four hints are stored (`internal/layout/style_leftovers.go:285-289`). Inline style is baked from the style attribute (`internal/layout/layout_svg.go:251-263`). `crispEdges` rounds rect, circle, ellipse, and line geometry to whole pixels (`internal/svg/raster.go:413-427`). Proof: `TestPaintOutputShapeRenderingCrispSnapsRaster` (`internal/layout/css_behavior_paint_output_test.go:81`).
- [x] `clip-rule`. `nonzero` and `evenodd` are stored (`internal/layout/style_leftovers.go:83-90`). The consumer is the CSS clip-path mask, not an SVG `clipPath` attribute. A star polygon keeps a twice-wound center under `nonzero` and cuts it out under `evenodd` (`internal/layout/css_behavior_clip_rule_wire_test.go:33`). Inline `polygon(evenodd, ...)` beats the property (`internal/layout/css_behavior_clip_rule_wire_test.go:55`). The value inherits (`internal/layout/css_behavior_clip_rule_wire_test.go:66`). A bad token keeps the default (`internal/layout/css_behavior_clip_rule_wire_test.go:84`). The same cut applies to a background image (`internal/layout/css_behavior_clip_rule_wire_test.go:132`). A plain div with no clip-path does not move (`internal/layout/css_behavior_text_remaining_test.go:384`). Do not replan this consumer.
- [x] `color-interpolation`. `auto` and `srgb` store `srgb`. `linearrgb` stores `linearrgb` (`internal/layout/style_leftovers.go:93-108`). A black-to-white `background-image` gradient is brighter at the midpoint in linear light (`internal/layout/css_behavior_color_interpolation_test.go:123`). A solid SVG rect does not change op geometry (`internal/layout/css_behavior_svg_paint_test.go:276`). The property is not baked onto an SVG gradient element.
- [x] `color-interpolation-filters`. Same three tokens (`internal/layout/style_leftovers.go:111-125`). `filter: grayscale(1)` on a red gradient paints a different midpoint in linear light (`internal/layout/css_behavior_color_interpolation_test.go:189`). This is the CSS filter path, not an SVG `feFlood` or lighting primitive.
- [x] `text-anchor`. HTML text is unchanged (`internal/layout/css_behavior_text2_test.go:103`). `applyLeftoversProps` has no `text-anchor` case, so the cascade drops it. Inline style on `<text>`, `<tspan>`, and `<textPath>` is baked (`internal/layout/layout_svg.go:653-675`). `middle` is written and `bogus` is dropped (`internal/layout/css_behavior_svg_text_test.go:64`). The two rasters match because text is stripped (`internal/svg/raster.go:189-191`, assertion at `internal/layout/css_behavior_svg_text_test.go:84-86`).
- [x] `alignment-baseline`. Selectors are stored (`internal/layout/style_leftovers.go:315-320`). On HTML text, `central` and `ideographic` move the text op baseline (`internal/layout/css_behavior_svg_text_test.go:126`). `middle` and `hanging` move it too, and `hanging` at 16px is a 5px rise (`internal/layout/css_behavior_svg_text_test.go:160`). The shift is `alignmentBaselineShift` (`internal/layout/inline_vertical_align.go:166-186`). A solid SVG rect does not change op geometry (`internal/layout/css_behavior_svg_paint_test.go:520`). The property is not inherited (`internal/layout/style_leftovers.go:309-314`).
- [x] `dominant-baseline`. Selectors are stored (`internal/layout/style_leftovers.go:301-306`). `hanging` raises a 16px run by about 5px (`internal/layout/css_behavior_property_stores_test.go:237`). `middle`, `central`, `mathematical`, and `text-before-edge` raise the run. `ideographic` and `text-after-edge` lower it (`internal/layout/css_behavior_perspective_test.go:240`). `auto` and `alphabetic` share the alphabetic baseline (`internal/layout/css_behavior_perspective_test.go:298`). A bad token keeps `auto` (`internal/layout/css_behavior_perspective_test.go:318`). The value inherits (`internal/layout/css_behavior_perspective_test.go:328`). The metric table is `baselineShiftFromMetrics` (`internal/layout/inline_vertical_align.go:133-156`).

### 0.3 Holes in the implemented set

These are open. They are not a second implementation of the property.

- [ ] `fill` and `stroke` accept `<paint>`, but the applier only parses a color (`internal/layout/style_paint_props.go:93-110`). `url(#g)` is dropped. The stop-color test says so (`internal/layout/css_behavior_svg_geo_test.go:170-171`). Accept a `url(#id)` paint and bake it. Do not add a new paint server.
- [ ] `stroke: none` is not stored. Add the same `none` arm `fill` already has (`internal/layout/style_paint_props.go:94-97`). Prove a rect with a red stroke loses the stroke when a later `stroke: none` wins.
- [ ] `fill-rule` and `shape-rendering` stylesheet rules stay unbaked (`internal/layout/layout_svg.go:241-242` and `internal/layout/layout_svg.go:272-273`). Bake `st.FillRule` and `st.ShapeRendering`. Prove a stylesheet `fill-rule: evenodd` opens the hole that the inline test already opens.
- [ ] `color-interpolation` on an inline SVG gradient is not baked. After the CSS gradient path stays green, bake the stored keyword onto the gradient element and require a pixel diff. Do not retune `renderGradientPNGWithInterp`.
- [~] `text-anchor` pixel movement on SVG text. Reason: text is stripped before parse so the canvas lock cannot load a system font (`internal/svg/raster.go:189-191`). The opened test requires equal rasters. Owner: this ledger. Next gate: a separate SVG text shaper that does not call the system font loader under `canvasMu`. Until that plan exists, do not remove the strip to make this row green.

**Gate:** phase 0 holes are open. Implemented catalog rows stay implemented. Do not flip them back to unsupported.

## Phase 1: stylesheet store for bakes that already change pixels

Shared rules: new code in `internal/layout/svg_presentation.go`. CSS wins over the attribute (phase 0). The rasterizer keeps reading attributes. Behavior tests go in `internal/layout/css_behavior_svg_present_test.go`. A catalog row flips to `implemented` only after a consumer outside the style layer and a behavior test exist. That is the catalog seed rule (`testdata/css/catalog/properties.json` seed `conservative_rule`).

### cx

Catalog syntax `<length-percentage>`. No apply handler.

- [x] Baseline: inline `cx:40px` moves a circle versus `cx:10px`. Proof: `TestBehaviorSvgGeoCxMovesCircle` (`internal/layout/css_behavior_svg_geo_test.go:21`). Bake: `internal/layout/layout_svg.go:275-278`.
- [ ] Acceptance: allow the units `svgBakeLengthValue` already allows: plain number, px, pt, pc, cm, mm, q, in, and `%` (`internal/layout/layout_svg.go:384-405`). Reject font and viewport units.
- [ ] Storage: store the canonical length on the resolved style from a stylesheet rule.
- [ ] Consumer: bake the stored value with the phase 0 CSS-wins rule. No new painter.
- [ ] Behavior test: a stylesheet `cx: 40px` differs in pixels from `cx: 10px`, and CSS beats `cx="10"`.
- [ ] Catalog: mark `cx` implemented only after that test.

### cy

Catalog syntax `<length-percentage>`.

- [x] Baseline: inline `cy:20px` moves a circle versus `cy:8px`. Proof: `TestBehaviorSvgGeoCyMovesCircle` (`internal/layout/css_behavior_svg_geo_test.go:42`). Bake: `internal/layout/layout_svg.go:280-283`.
- [ ] Acceptance: same length units as `cx`.
- [ ] Storage: stylesheet value on the resolved style.
- [ ] Consumer: bake with CSS-wins. No new painter.
- [ ] Behavior test: stylesheet `cy` pixel diff, and CSS beats a `cy` attribute.
- [ ] Catalog: mark `cy` implemented only after that test.

### r

Catalog syntax `<length-percentage>`. Negative radii are invalid.

- [x] Baseline: inline `r:12px` paints a larger disc than `r:4px`. Proof: `TestBehaviorSvgGeoRResizesCircle` (`internal/layout/css_behavior_svg_geo_test.go:63`). Bake drops negatives (`internal/layout/layout_svg.go:285-288`, `internal/layout/layout_svg.go:408-424`).
- [ ] Acceptance: non-negative length or percentage in the same units as `cx`. A negative value does not replace an earlier valid radius.
- [ ] Storage: stylesheet value on the resolved style.
- [ ] Consumer: bake with CSS-wins.
- [ ] Behavior test: stylesheet `r` pixel diff, negative value ignored, CSS beats an `r` attribute.
- [ ] Catalog: mark `r` implemented only after that test.

### rx

Catalog syntax `<length-percentage> | auto`.

- [x] Baseline: inline `rx:10px` changes corner pixels versus `rx:0px`. Proof: `TestBehaviorSvgGeoRxRoundsRect` (`internal/layout/css_behavior_svg_geo_test.go:85`). Bake: `internal/layout/layout_svg.go:290-293`.
- [ ] Acceptance: non-negative lengths as today, plus `auto`. `auto` is not a number, so `svgBakeLengthValue` currently rejects it (`internal/layout/layout_svg.go:388-405`).
- [ ] Storage: stylesheet length or `auto` on the resolved style.
- [ ] Consumer: bake `auto` as the attribute `auto`. CSS-wins.
- [ ] Behavior test: stylesheet `rx` pixel diff, `auto` baked, CSS beats an `rx` attribute.
- [ ] Catalog: mark `rx` implemented only after that test.

### ry

Catalog syntax `<length-percentage> | auto`.

- [x] Baseline: inline `ry:8px` changes corner pixels versus `ry:0px`. Proof: `TestBehaviorSvgGeoRyRoundsRect` (`internal/layout/css_behavior_svg_geo_test.go:107`). Bake: `internal/layout/layout_svg.go:295-298`.
- [ ] Acceptance: non-negative lengths, plus `auto`, same hole as `rx`.
- [ ] Storage: stylesheet length or `auto`.
- [ ] Consumer: bake `auto`. CSS-wins.
- [ ] Behavior test: stylesheet `ry` pixel diff, `auto` baked, CSS beats an `ry` attribute.
- [ ] Catalog: mark `ry` implemented only after that test.

### x

Catalog syntax `<length-percentage>`.

- [x] Baseline: inline `x:24px` moves a rect versus `x:2px`. Proof: `TestBehaviorSvgGeoXMovesRect` (`internal/layout/css_behavior_svg_geo_test.go:128`). Bake: `internal/layout/layout_svg.go:300-303`.
- [ ] Acceptance: same length units as `cx`.
- [ ] Storage: stylesheet value on the resolved style.
- [ ] Consumer: bake with CSS-wins.
- [ ] Behavior test: stylesheet `x` pixel diff, and CSS beats an `x` attribute.
- [ ] Catalog: mark `x` implemented only after that test.

### y

Catalog syntax `<length-percentage>`.

- [x] Baseline: inline `y:10px` moves a rect versus `y:2px`. Proof: `TestBehaviorSvgGeoYMovesRect` (`internal/layout/css_behavior_svg_geo_test.go:149`). Bake: `internal/layout/layout_svg.go:305-308`.
- [ ] Acceptance: same length units as `cx`.
- [ ] Storage: stylesheet value on the resolved style.
- [ ] Consumer: bake with CSS-wins.
- [ ] Behavior test: stylesheet `y` pixel diff, and CSS beats a `y` attribute.
- [ ] Catalog: mark `y` implemented only after that test.

### d

Catalog syntax `none | <string>`. The live bake only accepts `path("...")`.

- [x] Baseline: a path with only `d: path('M8 8 H56 V32 H8 Z')` paints, and an empty path paints nothing. Proof: `TestBehaviorSvgCSSDProvidesPathData` (`internal/layout/css_behavior_svg_markers_test.go:111`). CSS replaces a `d` attribute (`internal/layout/css_behavior_svg_markers_test.go:132`). Parser: `internal/layout/layout_svg.go:733-751`.
- [ ] Acceptance: keep `path("...")` and `path('...')`. `none` clears path data. A bare string that is not `path(...)` stays rejected until a test shows the pinned canvas can use it. Do not pass an unparsed string through as a `d` attribute.
- [ ] Storage: store the path data from a stylesheet rule. Today only the style attribute is read (`internal/layout/layout_svg.go:644-648`).
- [ ] Consumer: bake one `d` attribute. CSS already wins over the attribute. Keep that.
- [ ] Behavior test: a stylesheet `d: path(...)` paints, and `d: none` paints nothing.
- [ ] Catalog: mark `d` implemented only after the stylesheet test. The inline pixel test is not enough while the catalog has no handler.

### marker-end

Catalog syntax `none | <marker-ref>`.

- [x] Baseline: inline `marker-end: url(#ah)` paints more pixels than the same line without it. Proof: `TestBehaviorSvgMarkerEndPaintsArrowhead` (`internal/layout/css_behavior_svg_markers_test.go:74`). Bake: `internal/layout/layout_svg.go:688-718`. Only `url(...)` is emitted (`internal/layout/layout_svg.go:721-730`).
- [ ] Acceptance: `none` and `url(#id)`. Anything else does not replace an earlier valid ref.
- [ ] Storage: stylesheet value on the resolved style. The shorthand `marker` is not a catalog row in this ledger. Do not add it.
- [ ] Consumer: bake `marker-end`. CSS-wins over a `marker-end` attribute. The canvas marker painter already draws `url(#id)`.
- [ ] Behavior test: a stylesheet `marker-end` pixel increase, and CSS beats the attribute.
- [ ] Catalog: mark `marker-end` implemented only after that test.

### marker-mid

Catalog syntax `none | <marker-ref>`.

- [x] Baseline: the same bake writes `marker-mid` (`internal/layout/layout_svg.go:696-718`). `TestBehaviorSvgMarkerShorthandPaintsVertices` paints a two-segment path with `marker: url(#ah)` and requires more pixels (`internal/layout/css_behavior_svg_markers_test.go:92`). That shorthand expands onto start, mid, and end. It is not a `marker-mid` longhand test.
- [ ] Acceptance: `none` and `url(#id)`.
- [ ] Storage: stylesheet longhand, separate from `marker-end`.
- [ ] Consumer: bake `marker-mid` only. CSS-wins.
- [ ] Behavior test: a path with a mid vertex and stylesheet `marker-mid: url(#ah)` paints more than the same path with `marker-mid: none`. Do not use the shorthand as the proof.
- [ ] Catalog: mark `marker-mid` implemented only after that test.

### marker-start

Catalog syntax `none | <marker-ref>`.

- [x] Baseline: the bake writes `marker-start` in the same loop as `marker-end` (`internal/layout/layout_svg.go:696-718`). The opened shorthand test does not isolate the longhand (`internal/layout/css_behavior_svg_markers_test.go:92`).
- [ ] Acceptance: `none` and `url(#id)`.
- [ ] Storage: stylesheet longhand.
- [ ] Consumer: bake `marker-start` only. CSS-wins.
- [ ] Behavior test: stylesheet `marker-start: url(#ah)` paints more than `marker-start: none` on a path whose first vertex is otherwise bare.
- [ ] Catalog: mark `marker-start` implemented only after that test.

### stop-color

Catalog syntax is empty in the row. The bake accepts `#rgb`, `#rrggbb`, `rgb()`, and `rgba()`, and only on a `<stop>` (`internal/layout/layout_svg.go:310-318`, `internal/layout/layout_svg.go:427-455`). Named colors stay unbaked because the canvas parser would turn them black.

- [x] Baseline: inline `stop-color` on the first stop recolours a gradient. Proof: `TestBehaviorSvgGeoStopColorRecolorsGradient` (`internal/layout/css_behavior_svg_geo_test.go:173`). The rect's `fill="url(#g)"` stays an attribute because the fill cascade drops `url()`.
- [ ] Acceptance: hex and `rgb()` / `rgba()` as today. Add named colors only if they are converted to hex before bake, so the canvas parser cannot turn them black.
- [ ] Storage: stylesheet value on the stop's resolved style.
- [ ] Consumer: bake onto `<stop>` only. CSS-wins over a `stop-color` attribute.
- [ ] Behavior test: a stylesheet `stop-color` pixel diff. A `stop-color` on a `<rect>` does not bake.
- [ ] Catalog: mark `stop-color` implemented only after that test.

### stop-opacity

Catalog syntax is empty in the row. The bake accepts a number from 0 to 1 or a percentage from 0% to 100%, on `<stop>` only (`internal/layout/layout_svg.go:319-323`, `internal/layout/layout_svg.go:458-481`).

- [x] Baseline: inline `stop-opacity: 0.2` fades the gradient start. Proof: `TestBehaviorSvgGeoStopOpacityFadesGradient` (`internal/layout/css_behavior_svg_geo_test.go:197`).
- [ ] Acceptance: number 0 to 1, and percentage 0% to 100%. A value outside that range does not replace an earlier valid opacity.
- [ ] Storage: stylesheet value on the stop.
- [ ] Consumer: bake onto `<stop>` only. CSS-wins.
- [ ] Behavior test: stylesheet `stop-opacity: 0.2` pixel diff versus an opaque stop.
- [ ] Catalog: mark `stop-opacity` implemented only after that test.

**Gate:** each name above stays unsupported in the catalog until its own behavior test and catalog row close. Pixel tests that rasterize call `svgBakeRequireMem` and skip under 1200 MB available (`internal/layout/css_behavior_svg_bake_test.go:49-64`). Do not weaken that skip.

## Phase 2: serialized attributes the rasterizer ignores

The pinned canvas `setAttribute` switch has no case for these three. The bake comments say not to assert pixels (`internal/layout/layout_svg.go:325-329`). New emulation, if any, goes in `internal/svg/present.go`, called from `rasterizeCanvas` next to `emulateDroppedPresentation`. No general filter graph.

### paint-order

Catalog syntax `normal | [ fill || stroke || markers ]`.

- [x] Baseline: inline `paint-order: stroke` is written into the XML and is not a pixel test. Proof: `TestBehaviorSvgGeoPaintOrderGap` (`internal/layout/css_behavior_svg_geo_test.go:245`). Validator: `internal/layout/layout_svg.go:484-517`.
- [ ] Acceptance: `normal`, or one or more of `fill`, `stroke`, and `markers` without repeats. That is what the validator already allows.
- [ ] Storage: stylesheet value on the resolved style.
- [ ] Consumer: bake the attribute. Then either emulate paint order for a rect with both fill and stroke in `internal/svg/present.go`, or stop at the serialize gap. Do not mark the catalog implemented on a serialize-only test. The opened gap test must stay green until a pixel test replaces it.
- [ ] Behavior test: if emulation lands, a blue stroke under a red fill differs in pixels from the default fill-then-stroke order. If emulation does not land, the catalog row stays unsupported and this behavior row stays open.
- [ ] Catalog: implemented only after a pixel diff.

### vector-effect

Catalog syntax `none | non-scaling-stroke | non-scaling-size | non-rotation | fixed-position`. The bake accepts only `none` and `non-scaling-stroke` (`internal/layout/layout_svg.go:520-528`).

- [x] Baseline: inline `non-scaling-stroke` is serialized and is not a pixel test. Proof: `TestBehaviorSvgGeoVectorEffectGap` (`internal/layout/css_behavior_svg_geo_test.go:261`).
- [ ] Acceptance: all five catalog tokens. Unknown tokens do not bake.
- [ ] Storage: stylesheet value.
- [ ] Consumer: bake the attribute. `non-scaling-stroke` needs a pixel test under a viewBox scale. The other three tokens stay serialized until a test shows a visible difference. Do not invent a scale model in the page painter.
- [ ] Behavior test: a scaled viewport with `non-scaling-stroke` keeps a thicker stroke in pixels than the same stroke without it. If the canvas cannot show that, the catalog row stays open.
- [ ] Catalog: implemented only after that pixel diff.

### text-rendering

Catalog syntax `auto | optimizeSpeed | optimizeLegibility | geometricPrecision`.

- [x] Baseline: inline `optimizeLegibility` is serialized in camelCase and is not a pixel test. Proof: `TestBehaviorSvgGeoTextRenderingGap` (`internal/layout/css_behavior_svg_geo_test.go:278`). Mapper: `internal/layout/layout_svg.go:532-546`.
- [ ] Acceptance: the four catalog keywords, baked in the camelCase form the mapper already uses.
- [ ] Storage: stylesheet value.
- [~] Pixel consumer. Reason: SVG text is stripped before parse (`internal/svg/raster.go:189-191`), so the raster cannot show a hint. Owner: this ledger. Next gate: the same future SVG text shaper as `text-anchor`. Until then the catalog stays unsupported. Storage and the serialize test can still land.
- [ ] Behavior test: the attribute is present in the serialized XML and the PNG bytes match the same text without the property.
- [ ] Catalog: leave `unsupported` while the pixel row is a non-goal. Do not flip it on a serialize-only test.

**Gate:** no catalog flips in this phase unless a pixel test exists.

## Phase 3: draft fill and stroke longhands

`TestBehaviorSvgFillStrokeColorInert` and `TestBehaviorSvgDraftPaintLonghandsInert` require these declarations not to change the raster (`internal/layout/css_behavior_svg_markers_test.go:157` and `internal/layout/css_behavior_svg_markers_test.go:193`). The fill-color test says Chrome 143 ignores `fill-color` and `stroke-color`, and aliasing them to `fill` and `stroke` would diverge from Chrome. Several inert tokens are not the catalog grammar (`stroke-align: inside`, `stroke-dashadjust: round`). Those rows prove "this string does not change pixels", not "the catalog value is accepted".

Shared decision, one gate for the whole phase:

- [ ] Decision: do not alias any name in this phase onto `fill`, `stroke`, or `stroke-dasharray`. A paint-server and dash-corner model is new code in `internal/layout/svg_presentation.go` plus `internal/svg/present.go`. If the decision is "keep ignoring", leave the catalog unsupported and do not close the consumer rows. The inert tests stay.

The subsections below are the work if that decision chooses to build them. `stroke-boundary`, `stroke-limit`, and `path-length` appear in the inert test and are not catalog names in this ledger. Leave them alone.

### fill-break

Catalog syntax `bounding-box | slice | clone`. No test.

- [ ] Acceptance: those three keywords only.
- [ ] Storage: stylesheet keyword on the resolved style.
- [ ] Consumer: this is fragment-boundary fill behavior. Blinkless does not split an SVG fill across pages inside the raster. Record the keyword on the image op extra in `internal/layout/op_extra.go` for a caller. Do not add a page rasterizer.
- [ ] Behavior test: the three keywords store, a bad token does not, and the PNG matches the same SVG without the property.
- [ ] Catalog: leave `unsupported` until a consumer other than storage exists. An op-extra field is that consumer only if a drawing-list test reads it back.

### fill-color

Catalog syntax `<color>`.

- [x] Baseline: `fill-color: #0000ff` next to `fill: #ff0000` does not change the pixel count. Proof: `TestBehaviorSvgFillStrokeColorInert` (`internal/layout/css_behavior_svg_markers_test.go:157`).
- [ ] Acceptance: a color the existing color parser accepts. Not an alias of `fill`.
- [ ] Storage: its own field, not `Fill`.
- [ ] Consumer: no raster change until the phase decision builds a paint-server layer. Until then the inert test stays the contract.
- [ ] Behavior test: keep the inert pixel pin. Add a store test only after the decision says to store it.
- [ ] Catalog: stay `unsupported` while the consumer is "do not paint".

### fill-image

Catalog syntax `<paint>#`.

- [x] Baseline: `fill-image: url(x.png)` does not change the pixel count (`internal/layout/css_behavior_svg_markers_test.go:202` and the loop at `internal/layout/css_behavior_svg_markers_test.go:225`).
- [ ] Acceptance: a list of paints. `none` clears the list.
- [ ] Storage: the list on the resolved style, not in `Fill`.
- [ ] Consumer: a paint-server layer in `internal/svg/present.go` that fills from the image. Do not reuse the HTML `background-image` painter as if it were an SVG fill.
- [ ] Behavior test: a solid `fill-image` that the server can raster differs in pixels from `fill-image: none`. Until the server exists, the inert pin stays.
- [ ] Catalog: implemented only after the pixel diff.

### fill-origin

Catalog syntax `match-parent | fill-box | stroke-box | content-box | padding-box | border-box`.

- [x] Baseline: `fill-origin: fill-box` does not change pixels (`internal/layout/css_behavior_svg_markers_test.go:203`).
- [ ] Acceptance: those six keywords.
- [ ] Storage: keyword on the resolved style.
- [ ] Consumer: the paint-server layer uses the keyword as the origin box for `fill-image`. Without `fill-image`, the keyword does not change pixels.
- [ ] Behavior test: two origins on the same `fill-image` differ in pixels, or the row stays open.
- [ ] Catalog: implemented only together with a `fill-image` pixel test.

### fill-position

Catalog syntax `<position>#`.

- [x] Baseline: `fill-position: center` does not change pixels (`internal/layout/css_behavior_svg_markers_test.go:204`).
- [ ] Acceptance: the same position syntax the background position parser already accepts. Reject a token that parser rejects.
- [ ] Storage: one position per paint layer.
- [ ] Consumer: offsets `fill-image` inside the origin box.
- [ ] Behavior test: `0 0` versus `center` differs once `fill-image` paints.
- [ ] Catalog: implemented only with that pixel test.

### fill-repeat

Catalog syntax `<repeat-style>#`.

- [x] Baseline: `fill-repeat: repeat` does not change pixels (`internal/layout/css_behavior_svg_markers_test.go:205`).
- [ ] Acceptance: the background repeat keywords.
- [ ] Storage: one repeat per paint layer.
- [ ] Consumer: repeats `fill-image`. `no-repeat` paints one tile.
- [ ] Behavior test: `repeat` versus `no-repeat` differs once `fill-image` paints.
- [ ] Catalog: implemented only with that pixel test.

### fill-size

Catalog syntax `<bg-size>#`.

- [x] Baseline: `fill-size: cover` does not change pixels (`internal/layout/css_behavior_svg_markers_test.go:206`).
- [ ] Acceptance: the background size syntax, including `cover` and `contain`.
- [ ] Storage: one size per paint layer.
- [ ] Consumer: sizes `fill-image` before repeat.
- [ ] Behavior test: `cover` versus a fixed length differs once `fill-image` paints.
- [ ] Catalog: implemented only with that pixel test.

### stroke-align

Catalog syntax `center | inset | outset`. The inert pin uses `stroke-align: inside`, which is not in that grammar (`internal/layout/css_behavior_svg_markers_test.go:212`).

- [x] Baseline: the non-catalog token `inside` does not change pixels. That does not prove `center`, `inset`, or `outset`.
- [ ] Acceptance: `center`, `inset`, and `outset` only. `inside` stays invalid.
- [ ] Storage: keyword on the resolved style.
- [ ] Consumer: inset and outset shift the stroke to one side of the path in `internal/svg/present.go`. `center` stays the canvas default. Do not alias this to `stroke-alignment`.
- [ ] Behavior test: `inset` versus `center` on a wide stroke differs in pixels.
- [ ] Catalog: implemented only after that pixel diff.

### stroke-alignment

Catalog syntax `center | inner | outer`. No test. This is a different property from `stroke-align`.

- [ ] Acceptance: `center`, `inner`, and `outer` only.
- [ ] Storage: its own field, not the `stroke-align` field.
- [ ] Consumer: `inner` and `outer` use the same one-sided stroke as inset and outset. Map `inner` to the inset side and `outer` to the outset side in one helper, and keep two stored keywords so the declared name round-trips.
- [ ] Behavior test: `inner` versus `center` differs in pixels on the same shape as `stroke-align`.
- [ ] Catalog: implemented only after that test. Do not close it by pointing at `stroke-align`.

### stroke-break

Catalog syntax `bounding-box | slice | clone`. The inert pin uses `stroke-break: bounding-box` (`internal/layout/css_behavior_svg_markers_test.go:213`).

- [x] Baseline: `bounding-box` does not change pixels.
- [ ] Acceptance: the three catalog keywords.
- [ ] Storage: keyword on the resolved style.
- [ ] Consumer: same as `fill-break`. The keyword rides on the image op for a caller. No page splitter.
- [ ] Behavior test: the keyword is stored and the PNG is unchanged.
- [ ] Catalog: stay `unsupported` until a drawing-list read of the extra field exists.

### stroke-color

Catalog syntax `<color>#`.

- [x] Baseline: `stroke-color: #0000ff` next to `stroke: #ff0000` does not change the pixel count (`internal/layout/css_behavior_svg_markers_test.go:169-176`).
- [ ] Acceptance: a list of colors. Not an alias of `stroke`.
- [ ] Storage: its own list.
- [ ] Consumer: no raster change until the phase decision says the paint-server layer paints stroke colors separately from `stroke`.
- [ ] Behavior test: keep the inert pin until that layer exists.
- [ ] Catalog: stay `unsupported` while it must not change pixels.

### stroke-dash-corner

Catalog syntax `none | <length>`. The inert pin uses `stroke-dash-corner: round`, which is not that grammar (`internal/layout/css_behavior_svg_markers_test.go:219`).

- [x] Baseline: the token `round` does not change dash pixels.
- [ ] Acceptance: `none` or a length. `round` stays invalid.
- [ ] Storage: `none` or a length on the resolved style.
- [ ] Consumer: the length shapes the dash corner in `internal/svg/present.go`. `none` keeps today's dashes.
- [ ] Behavior test: a positive length differs in pixels from `none` on a dashed rect. The old `round` pin can stay as an invalid-value test.
- [ ] Catalog: implemented only after the pixel diff.

### stroke-dash-justify

Catalog syntax `none | [ stretch | compress ] || [ dashes || gaps ]`. The inert pin uses `stretch` (`internal/layout/css_behavior_svg_markers_test.go:220`).

- [x] Baseline: `stretch` alone does not change dash pixels.
- [ ] Acceptance: the catalog combination. A lone `dashes` or `gaps` without stretch or compress is invalid.
- [ ] Storage: the chosen pair on the resolved style.
- [ ] Consumer: stretch or compress the dash pattern so the pattern fits the path in `internal/svg/present.go`.
- [ ] Behavior test: `stretch dashes` differs in pixels from `none` on a dashed line.
- [ ] Catalog: implemented only after that pixel diff.

### stroke-dashadjust

Catalog syntax `none | [stretch | compress] [dashes | gaps]?`. The inert pin uses `stroke-dashadjust: round`, which is not that grammar (`internal/layout/css_behavior_svg_markers_test.go:221`).

- [x] Baseline: the token `round` does not change dash pixels.
- [ ] Acceptance: the catalog grammar. `round` stays invalid. This name is not an alias of `stroke-dash-justify`. Store both if both are declared.
- [ ] Storage: its own field.
- [ ] Consumer: same dash-fit helper as `stroke-dash-justify`, with this property's stored value as the input when `stroke-dash-justify` is absent.
- [ ] Behavior test: `compress gaps` differs in pixels from `none`.
- [ ] Catalog: implemented only after that test, separate from `stroke-dash-justify`.

### stroke-dashcorner

Catalog syntax `none | <length>`. The inert pin uses `stroke-dashcorner: round` (`internal/layout/css_behavior_svg_markers_test.go:222`).

- [x] Baseline: the token `round` does not change dash pixels.
- [ ] Acceptance: `none` or a length. Not an alias spelling of `stroke-dash-corner` in the catalog sense: both names exist as separate rows. Store both. The consumer reads `stroke-dashcorner` only when `stroke-dash-corner` was not set.
- [ ] Storage: its own field.
- [ ] Consumer: same corner-length helper as `stroke-dash-corner`.
- [ ] Behavior test: a length differs in pixels from `none`.
- [ ] Catalog: implemented only after that test, separate from `stroke-dash-corner`.

### stroke-image

Catalog syntax `<paint>#`.

- [x] Baseline: `stroke-image: url(x.png)` does not change pixels (`internal/layout/css_behavior_svg_markers_test.go:207`).
- [ ] Acceptance: a list of paints. `none` clears it.
- [ ] Storage: its own list, not `Stroke`.
- [ ] Consumer: the paint-server layer strokes from the image. Depends on the phase decision.
- [ ] Behavior test: a rasterizable `stroke-image` differs from `none`.
- [ ] Catalog: implemented only after that pixel diff.

### stroke-origin

Catalog syntax `match-parent | fill-box | stroke-box | content-box | padding-box | border-box`.

- [x] Baseline: `stroke-origin: stroke-box` does not change pixels (`internal/layout/css_behavior_svg_markers_test.go:208`).
- [ ] Acceptance: those six keywords.
- [ ] Storage: keyword on the resolved style.
- [ ] Consumer: origin box for `stroke-image`. No effect without a stroke image.
- [ ] Behavior test: two origins differ once `stroke-image` paints.
- [ ] Catalog: implemented only with that pixel test.

### stroke-position

Catalog syntax `<position>#`.

- [x] Baseline: `stroke-position: center` does not change pixels (`internal/layout/css_behavior_svg_markers_test.go:209`).
- [ ] Acceptance: the same position syntax as `fill-position`.
- [ ] Storage: one position per stroke paint.
- [ ] Consumer: offsets `stroke-image`.
- [ ] Behavior test: `0 0` versus `center` differs once `stroke-image` paints.
- [ ] Catalog: implemented only with that pixel test.

### stroke-repeat

Catalog syntax `<repeat-style>#`.

- [x] Baseline: `stroke-repeat: repeat` does not change pixels (`internal/layout/css_behavior_svg_markers_test.go:210`).
- [ ] Acceptance: the background repeat keywords.
- [ ] Storage: one repeat per stroke paint.
- [ ] Consumer: repeats `stroke-image`.
- [ ] Behavior test: `repeat` versus `no-repeat` differs once `stroke-image` paints.
- [ ] Catalog: implemented only with that pixel test.

### stroke-size

Catalog syntax `<bg-size>#`.

- [x] Baseline: `stroke-size: cover` does not change pixels (`internal/layout/css_behavior_svg_markers_test.go:211`).
- [ ] Acceptance: the background size syntax.
- [ ] Storage: one size per stroke paint.
- [ ] Consumer: sizes `stroke-image`.
- [ ] Behavior test: `cover` versus a fixed length differs once `stroke-image` paints.
- [ ] Catalog: implemented only with that pixel test.

**Gate:** the phase decision is still open. Inert tests must stay green until a replacement pixel test lands in the same change.

## Phase 4: names with no bake

### baseline-shift

Catalog syntax `<length-percentage> | sub | super | top | center | bottom`. No test.

- [ ] Acceptance: a length or percentage, or one of `sub`, `super`, `top`, `center`, `bottom`.
- [ ] Storage: its own field. Do not write it into `VerticalAlign`. `sub` and `super` already shift HTML text through `vertical-align` (`internal/layout/inline_vertical_align.go:54-58`). Sharing the shift helper is fine. Sharing the property is not.
- [ ] Consumer: for HTML text, add the stored shift in `effectiveVerticalAlignShift` (`internal/layout/inline_vertical_align.go:49`). For SVG text, bake the attribute and expect equal PNG bytes while text is stripped (`internal/svg/raster.go:189-191`).
- [ ] Behavior test: HTML `baseline-shift: 4px` moves the text op Y. HTML `super` moves it. SVG text rasters match with and without the property, and the attribute is present.
- [ ] Catalog: implemented after the HTML text-op test. The SVG pixel non-goal does not block that, because the HTML shift is a real consumer.

### flood-color

Catalog syntax `<color>`. No test. This is an SVG filter primitive, not a CSS flood.

- [ ] Acceptance: a color the existing color parser accepts.
- [ ] Storage: stylesheet color on the resolved style.
- [ ] Consumer: bake onto `feFlood` only. There is no filter graph in this ledger. If a pixel probe shows the pinned canvas ignores `flood-color`, add a one-element solid fill in `internal/svg/present.go` for `feFlood` only. Do not implement `feGaussianBlur`, `feMerge`, or lighting.
- [ ] Behavior test: two `feFlood` colors differ in pixels, or the row stays open. A `flood-color` on a `<rect>` does not bake.
- [ ] Catalog: implemented only after the pixel diff.

### flood-opacity

Catalog syntax `<'opacity'>`.

- [ ] Acceptance: the same opacity syntax as `fill-opacity` (`internal/layout/style_paint_props.go:102-105`).
- [ ] Storage: a 0 to 1 alpha on the `feFlood` style.
- [ ] Consumer: the same one-element `feFlood` fill multiplies by this alpha. No filter graph.
- [ ] Behavior test: `flood-opacity: 0.2` differs in pixels from an opaque flood.
- [ ] Catalog: implemented only after that pixel diff.

### glyph-orientation-vertical

Catalog syntax `auto | 0deg | 90deg | 0 | 90`.

- [ ] Acceptance: those five tokens. `90` and `90deg` store one canonical value.
- [ ] Storage: stylesheet keyword.
- [ ] Consumer: bake the attribute on vertical SVG text. Pixels stay equal while text is stripped.
- [~] Pixel rotation of SVG glyphs. Reason: `internal/svg/raster.go:189-191` removes text before parse. Owner: this ledger. Next gate: the SVG text shaper named on `text-anchor`. HTML text does not rotate for this property.
- [ ] Behavior test: the attribute is in the XML and the PNG matches the same SVG without the property.
- [ ] Catalog: stay `unsupported` while the only consumer is a stripped serialize.

### image-rendering

Catalog syntax `auto | optimizeQuality | optimizeSpeed`. The catalog row does not list `pixelated` or `crisp-edges`. Do not add them.

- [ ] Acceptance: those three keywords only.
- [ ] Storage: keyword on the resolved style of an `img` or `svg`.
- [ ] Consumer: do not add a resampler. When an image is already re-encoded because of orientation or a clip, `optimizeSpeed` may pick the cheaper filter in the existing re-encode path. Otherwise attach the keyword to the image op extra so a caller can read it (`internal/layout/op_extra.go:11`). New file `internal/layout/image_rendering.go`.
- [ ] Behavior test: the three keywords store, a bad token does not, and a drawing-list test reads the keyword back from the image op. No page PNG.
- [ ] Catalog: implemented only after that drawing-list read. Storage alone is not enough under the catalog seed rule.

### lighting-color

Catalog syntax `<color>`.

- [ ] Acceptance: a color the existing parser accepts.
- [ ] Storage: stylesheet color.
- [ ] Consumer: bake onto `feDiffuseLighting` and `feSpecularLighting` only. Do not build a lighting equation. If the pinned canvas ignores the attribute, stop at the serialize gap the way `paint-order` does.
- [ ] Behavior test: the attribute is serialized. A pixel assertion is allowed only if a probe shows the canvas already changes pixels. Do not fake a light.
- [ ] Catalog: stay `unsupported` on a serialize-only result.

### marker-side

Catalog syntax `match-self | match-parent`. This is not `marker-start`. The bake loop does not mention it (`internal/layout/layout_svg.go:696-718`).

- [ ] Acceptance: those two keywords only.
- [ ] Storage: keyword on the resolved style.
- [ ] Consumer: bake the attribute. Then probe whether the pinned canvas changes pixels. If it does not, do not invent a marker coordinate system. Leave the pixel row open and the catalog unsupported.
- [ ] Behavior test: both keywords serialize, a bad token does not, and a pixel diff is recorded only when the probe shows one.
- [ ] Catalog: implemented only after a pixel diff.

**Gate:** filter and marker-side pixel rows that the canvas cannot show stay open. Do not close them with a comment.

## Phase 5: catalog, docs, and closure

- [ ] Update `testdata/css/catalog/properties.json` only for names whose behavior test and consumer landed in this change. Handlers point at `svg_presentation.go` or the existing applier. Tests point at the new behavior test.
- [ ] Regenerate the compatibility matrix property section so it matches the catalog.
- [ ] Do not edit `documentation/deferred.md` in the implementation change unless the user asks. This checklist is the planning home.
- [ ] Do not grow `internal/layout/layout.go`. `make lint` runs the file-size check.
- [ ] Run `make test`. Record the exit code. This is the full suite, capped by the Makefile. Do not run bare `go test ./...`.
- [ ] Run `make golden`. It runs `layout.TestDisplay`. It does not rasterize a page.
- [ ] Run `make lint`.
- [ ] Run `make claim-scan`. Do not claim a page PNG, a PDF, or pixel parity with Chrome for names whose tests only check XML.
- [ ] Run `make catalog-check` and `make matrix-check`.
- [ ] GitHub CI runs only `make test` and `make lint` (`.github/workflows/ci.yml:1`, `.github/workflows/ci.yml:20`, `.github/workflows/ci.yml:26`). The other gates are local.

Focused runs during the work, one package at a time:

```sh
go test ./internal/layout -run 'TestBehaviorSvg|TestPaintOutputFillRule|TestPaintOutputShapeRendering|TestSvgText|TestBehaviorFill|TestBehaviorStroke|TestBehaviorClipRule|TestColorInterpolation' -count=1
go test ./internal/svg -count=1
```

**Gate:** open until every command above exits 0 on the change that flips catalog rows.

## Dependencies

- Cascade entry for the implemented names is `applySVGPresentationProps` (`internal/layout/style_paint_props.go:91`) and `applyLeftoversProps` (`internal/layout/style_leftovers.go:11`).
- Inline SVG serialization is `internal/layout/layout_svg.go`. New bake arms go in `internal/layout/svg_presentation.go` and are called from `writeSVGNode`, not pasted into `layout.go`.
- Raster entry is `internal/svg/raster.go`. `emulateDroppedPresentation` already covers `fill-rule` and `shape-rendering`. Further emulation goes in `internal/svg/present.go`.
- Text baseline shift for `alignment-baseline` and `dominant-baseline` is `internal/layout/inline_vertical_align.go`. Leave that path alone except for the `baseline-shift` HTML hook.
- `clip-rule` is consumed by the clip-path mask tests under `internal/layout/css_behavior_clip_rule_wire_test.go`. Do not move it into the SVG bake.
- The property catalog is the support record. The matrix must match it.
- `documentation/deferred.md` is not the live list for these 61 names.
