# v0.0.1 - CSS masking first wave checklist

> **Parent:** [HTML, CSS, and JSON compatibility](../html-css-json-compatibility-checklist.md) - shared catalog and behavior-test rules.
> **Status:** complete. All 18 selected property names are recorded as Partial, and every final gate passed on 2026-10-13.
> **Estimated effort:** 18 CSS property names across four implementation phases.

---

## Overview

CSS masks control which parts of an element remain visible. Blinkless returns an ordered display list, a set of paint instructions for another program to replay. This first wave adds mask data to element groups so a display-list consumer can apply the mask after painting the element and its children. It does not add a page renderer to Blinkless.

The selected names are 10 standard properties and 8 WebKit-prefixed forms. The catalog records all 18 as Partial because Blinkless exports their group data but does not replay mask pixels itself. Catalog values name the accepted subset. The catalog, not the number of selected names, is the support record.

A masked group includes the element's background, border, text, images, and child paint. That follows the [W3C CSS Masking group model](https://www.w3.org/TR/css-masking-1/#module-interactions). The public Go API carries `MaskDescriptor` on `BlendGroup` (`internal/layout/blend_group.go:13-67`, `layout/displaylist.go:55-62`). WASM exposes the same group and layer data (`bindings/wasm/drawing_list.go:205-231,578-603`).

## Executive summary

| Standard property | WebKit-prefixed form |
|---|---|
| `mask` | `-webkit-mask` |
| `mask-clip` | `-webkit-mask-clip` |
| `mask-composite` | `-webkit-mask-composite` |
| `mask-image` | `-webkit-mask-image` |
| `mask-mode` | no alias in this wave |
| `mask-origin` | `-webkit-mask-origin` |
| `mask-position` | `-webkit-mask-position` |
| `mask-repeat` | `-webkit-mask-repeat` |
| `mask-size` | `-webkit-mask-size` |
| `mask-type` | no alias in this wave |

The accepted source subset is `url(...)` and `none`. Resolved raster images use their original PNG or JPEG bytes. An external SVG `<mask>` fragment is rasterized through its SVG source. A direct image's `match-source` mode uses alpha. An SVG mask's `match-source` mode uses the source mask's `mask-type`, whose initial value is luminance ([W3C `mask-mode`](https://www.w3.org/TR/css-masking-1/#the-mask-mode), [W3C `mask-type`](https://www.w3.org/TR/css-masking-1/#mask-type)).

CSS gradients and `image-set()` sources are outside this subset. SVG masks support the graphics subset already handled by `internal/svg`; mask-region clipping, external stylesheet cascade, filters, animation, and nested mask references are not handled. The `mask` shorthand resets `mask-border` values, but mask-border painting is outside this wave.

## Phase 0: lock the contract and baseline

- [x] Verify the 18 selected catalog rows were `unsupported` before implementation. Evidence from 2026-10-13: all 18 had no handler or test; `make catalog-check` and `make matrix-check` passed at the starting baseline of 389 implemented, 1 partial, 387 unsupported, and 8 intentionally ignored.
- [x] Record the browser reference in `testdata/css/masking/element-group-reference.html`. Chrome 143.0.7499.40, viewport 380 by 140 CSS pixels, device scale factor 1. Puppeteer captured `/tmp/blinkless-css-mask-reference.png`. The alpha fixture uses an opaque white right half and transparent left half; the luminance fixture uses opaque white and black halves.
- [x] Record browser pixel expectations. At y=50, x=50 and x=230 show body green `(0,160,0)`, x=120 and x=300 show element red `(230,30,30)`, and x=179 and x=359 show the blue border `(0,0,255)`. At y=95, x=90 and x=270 show green through the transparent/black halves; x=150 and x=330 show the yellow child box `(255,220,0)`. White child-text pixels are present on both visible halves.
- [x] Select raster URLs and external SVG mask fragments as sources. `mask_paint.go` uses the image resolver, and `internal/svg.RasterizeMask` consumes the raw SVG source when the URL has a fragment. Local same-document fragments without resolver-provided SVG source stay unresolved.
- [x] Keep `mask-type` in scope. `TestBehaviorSVGMaskTypeFlowsIntoMaskImagePayload` and the SVG alpha, luminance, and linearRGB pixel tests prove that `mask-type` changes the processed SVG mask alpha for `match-source`.
- [x] Select nested element groups as the representation. Mask groups use the existing group markers, carry a `MaskDescriptor`, preserve their parent links, and serialize to the Go and WASM contracts. The consumer applies the group mask after replaying its operations.

**Gate:** passed by the baseline checks, Chrome reference capture, SVG mask tests, and Go/WASM group contract tests listed above.

## Phase 1: represent and apply an element mask

- [x] Add mask fields to resolved style and style interning. Proof: `TestStyleInternGenerated` and the focused layout test run.
- [x] Attach mask layers to an isolated element group. `TestBehaviorMaskLayerPropertiesAttachToAllElementPaint` checks background, border, text, child image, sibling separation, and balanced group markers.
- [x] Preserve nested mask groups and group paint order. `TestBehaviorMaskGroupsNestAndRepeatLayerValues` and the `TestPaintOrder` tests pass.
- [x] Update the Go group aliases, WASM group JSON, image-table references, and library API notes. `TestDrawingGroupSerializesMaskAndImageReference` and `TestConvertSerializesSVGMaskInGroupContract` pass.

**Gate:** focused layout, SVG, paint-order, and WASM contract tests pass. Final repository gates are recorded in Phase 4.

## Phase 2: implement standard properties

### 2.1 Image source and mode

- [x] Accept `url(...)` and `none`, multiple comma-separated mask images, and the selected alpha, luminance, and `match-source` values. `TestBehaviorMaskGroupsNestAndRepeatLayerValues` and `TestBehaviorMaskMatchSourceUsesImageAlpha` pass.
- [x] Apply `mask-type` for the selected SVG `<mask>` fragment source. Alpha and luminance output pixels, explicit mode override, and inline SVG `mask-type` are covered in `internal/svg/mask_test.go` and `TestBehaviorSVGMaskTypeFlowsIntoMaskImagePayload`.

### 2.2 Layer values

- [x] Carry accepted `mask-position`, `mask-size`, `mask-repeat`, `mask-origin`, and `mask-clip` tokens into each mask layer. The element-group test asserts each value and the nested-layer test checks CSS list cycling.
- [x] Carry standard `mask-composite` values in layer order. The group test checks `exclude` and the nested-layer test checks two-layer ordering. Pixel compositing is performed by the display-list consumer.

### 2.3 Shorthand and acceptance

- [x] Expand the `mask` shorthand, apply defaults, reset `mask-border` longhands, and resolve CSS-wide values. `TestMaskShorthandExpansionAndBorderReset` and `TestMaskShorthandCSSWideKeywords` pass.
- [x] Resolve custom properties in the shorthand while keeping shorthand and longhand source order. `TestMaskShorthandResolvesCustomPropertiesAndKeepsLonghandOrder` passes.
- [x] Make declaration acceptance and `@supports` use the same value rules. `TestBehaviorMaskSupportsMatchesAcceptedValues` and `TestBehaviorMaskInvalidValuesDoNotReplaceEarlierDeclarations` pass.
- [x] Add behavior coverage for each standard property at the drawing-list contract. The mask group tests assert accepted layer values, resolved image data, SVG preprocessing, shorthand output, and paint ownership.

**Gate:** passed for the documented value subset in focused tests. CSS gradients, `image-set()`, SVG features listed above, and pixel replay are outside the first-wave contract.

## Phase 3: add WebKit-prefixed forms

- [x] Map all eight selected aliases to the standard mask behavior. `TestBehaviorMaskVendorAliasesReachMaskGroup` checks each alias in the resulting mask group.
- [x] Map `source-over`, `source-out`, `source-in`, and `xor` to the standard composite tokens. The alias behavior test checks `source-over` and `xor`; cascade acceptance tests cover the accepted legacy values.
- [x] Test prefixed/unprefixed source order. `TestBehaviorMaskVendorAliasUsesCSSSourceOrder` passes with both declaration orders.
- [x] Test valid and invalid `@supports` values for mask declarations and a prefixed composite value. `TestBehaviorMaskSupportsMatchesAcceptedValues` and `TestCascadeEngineSupportsPropertyValues` pass.

**Gate:** passed for the eight selected aliases and their documented value subset. The six `-webkit-mask-box-image*` names remain outside this 18-name target.

## Phase 4: reconcile catalog, docs, and final evidence

- [x] Update the catalog rows and summary from code and behavior tests. `make catalog-check` passes with 389 implemented, 19 partial, 369 unsupported, and 8 intentionally ignored.
- [x] Regenerate the compatibility matrix property section and update the older mask and `clip-path` summaries. `make matrix-check` passes.
- [x] Update `documentation/deferred.md` and `documentation/library-api.md` with current behavior and limits.
- [x] Update the local CSS masking knowledge-base page, index, and operation log after the final gates.
- [x] Run `make test` on the completed tree. Exit 0; all test packages passed.
- [x] Run `go build ./...` on the completed tree. Exit 0.
- [x] Run `make golden` on the completed tree. Exit 0; `layout.TestDisplay` passed.
- [x] Run `make wasm-test` on the completed tree. Exit 0; contract tests, JSON consumer check, and WASM build passed.
- [x] Run `make lint` on the completed tree. Exit 0; golangci-lint and the file-size check passed.
- [x] Run `make claim-scan`, `make catalog-check`, `make matrix-check`, and `git diff --check`. Each exited 0. Catalog totals are 389 implemented, 19 partial, 369 unsupported, and 8 intentionally ignored.

**Gate:** passed on 2026-10-13. Every checklist item is complete.

## Test method

Blinkless has no page-pixel renderer. The Go tests therefore check the returned drawing list: each mask group owns the element's paint, has the right parent, and carries the expected source bytes and CSS values. The SVG tests inspect output PNG alpha at known pixels. WASM tests parse the serialized group and verify its image references. Chrome's screenshot provides the browser reference for group masking and the listed sample colors; it does not claim that Blinkless rendered those pixels. `make golden` runs the public drawing-list tests in `layout`, not a page raster comparison.

Run focused tests during edits:

```sh
GOCACHE=/tmp/blinkless-mask-gocache go test ./internal/layout -run 'TestMaskShorthand|TestBehaviorMask|TestBehaviorSVGMask|TestPaintOrder|TestSupportedDeclarationMaskGrammar|TestCascadeEngineSupportsPropertyValues|TestStyleInternGenerated' -count=1
GOCACHE=/tmp/blinkless-mask-gocache go test ./internal/svg
GOCACHE=/tmp/blinkless-mask-gocache go test ./bindings/wasm -run 'TestDrawingGroupSerializesMaskAndImageReference|TestConvertSerializesSVGMaskInGroupContract' -count=1
```

## Dependencies

- CSS declaration acceptance and `@supports` share `supportedDeclaration` in `internal/layout/style_value_accept.go` and vendor normalization in `internal/layout/style_cascade.go`.
- The layout engine stops at `layout.DisplayList`; a caller replays its operations (`layout/displaylist.go:129-144`). Mask pixel replay belongs to that caller.
- SVG mask alpha conversion uses the sRGB or linearRGB color path in `internal/svg/mask.go:285-369`.
- The property catalog is authoritative for per-property status. The generated matrix must match it (`scripts/css-catalog-map.py`, `scripts/check-matrix-sync.sh`).
