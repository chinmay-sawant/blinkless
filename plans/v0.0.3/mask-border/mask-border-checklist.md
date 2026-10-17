# v0.0.3 - Mask border

> **Parent:** `plans/v0.0.3/README.md`
> **Status:** Phases 0-3 implemented and verified in the mask-border worktree (`TestBehaviorMaskBorder*`, `TestMaskBorder*`, `TestMaskShorthand*` green). Phase 4 catalog, WASM, matrix, knowledge-base, and gate rows remain for the merge session.
> **Estimated effort:** 13 property names, 5 phases.

---

## Overview

`mask-border` cuts a source image into nine pieces, four corners, four edges, and a center, and uses that frame as a mask around the border box. The host applies it. Blinkless records the slice data on the element's group, the same way the first mask wave records `MaskDescriptor` (`internal/layout/blend_group.go:38-43`).

This ledger covers seven standard properties and six aliases:

| Standard | Alias |
|---|---|
| `mask-border` | `-webkit-mask-box-image` |
| `mask-border-source` | `-webkit-mask-box-image-source` |
| `mask-border-slice` | `-webkit-mask-box-image-slice` |
| `mask-border-width` | `-webkit-mask-box-image-width` |
| `mask-border-outset` | `-webkit-mask-box-image-outset` |
| `mask-border-repeat` | `-webkit-mask-box-image-repeat` |
| `mask-border-mode` | no alias in this set |

The catalog, read on 2026-10-15 (`properties.json` `updated` is `2026-10-13`), marks all 13 `unsupported` with no handler. The first wave already said the six `-webkit-mask-box-image*` names were outside its 18-name target (`plans/v0.0.1/css-masking/css-masking-checklist.md` Phase 3 gate).

The `mask` shorthand already resets the seven standard longhands. `expandMaskShorthandDeclaration` appends `mask-border: initial`, `mask-border-source: none`, `mask-border-mode: alpha`, `mask-border-slice: 0`, `mask-border-width: auto`, `mask-border-outset: 0`, and `mask-border-repeat: stretch` (`internal/layout/style_cascade.go:855-914`). CSS-wide keywords reset them to that keyword (`internal/layout/style_cascade.go:918-935`). `TestMaskShorthandExpansionAndBorderReset` pins the reset (`internal/layout/style_cascade_test.go:76-110`). Nothing reads those wins afterward, because there is no apply handler. This ledger adds the reader. It does not change the reset list.

`border-image` already paints. `applyBorderImageProps` stores the longhands (`internal/layout/border_image.go:29-48`). `appendBorderImageByRepeat` builds the nine slices (`internal/layout/border_image.go:335-350`). `newBorderImageOp` emits `OpImage` ops flagged `IsBackground` (`internal/layout/border_image.go:361-370`). `TestBehaviorBorderImageShorthandPaintsFrame` expects those painted cells (`internal/layout/css_behavior_borders_rest_test.go:497-513`).

Do not reuse that painter as the mask-border result. Those ops are a visible border image. A mask-border is a mask. Reuse the slice math as the pattern for numbers: outset expansion in `borderImageOuterBounds` (`internal/layout/border_image.go:294-303`), thickness clamping in `scaleBorderImageThickness` (`internal/layout/border_image.go:306-332`), and the stretch versus repeat split in `appendBorderImageByRepeat`. Copy the geometry into a `MaskBorder` descriptor on the group. Do not call `appendBorderImageStretched` for a mask-border, and do not add a page rasterizer.

New code lives in `internal/layout/mask_border.go`. Do not add it to `internal/layout/layout.go`.

Paint Y grows downward. 1 CSS pixel is 0.75 points (`layout/displaylist.go:12-14`). The host draws the list (`layout/displaylist.go:129-133`).

## Executive summary

| Question | Answer |
|---|---|
| Does a mask-border paint slices into the border? | No. `border-image` does that. Mask-border must not. |
| What does the drawing list carry? | One `MaskBorder` on the group: source bytes, slice, width, outset, repeat, mode. |
| Who applies the alpha? | The host. Same split as `MaskLayer`. |
| What already shipped? | The `mask` shorthand reset. Not the properties. |

`documentation/deferred.md:108` and the vendor group C note at `documentation/deferred.md:119` still say mask-border has no consumer. Phase 4 updates those sentences after the tests pass.

## Phase 0: contract and the reset that already exists

- [x] All 13 catalog rows are `unsupported` with empty handlers. Standard names start at `testdata/css/catalog/properties.json:10336` (`mask-border`) through `mask-border-width`. Aliases start at line 276 (`-webkit-mask-box-image`) through `-webkit-mask-box-image-width`.
- [x] The `mask` shorthand resets the seven longhands to the initial values listed above. Proof: `internal/layout/style_cascade.go:908-914` and `TestMaskShorthandExpansionAndBorderReset` (`internal/layout/style_cascade_test.go:76-110`).
- [x] `border-image` paints `OpImage` cells. Proof: `newBorderImageOp` (`internal/layout/border_image.go:361-370`) and `TestBehaviorBorderImageShorthandPaintsFrame`.
- [x] The group gains `MaskBorder` next to `Mask` on `BlendGroup` (`internal/layout/blend_group.go:20-36`). Nil means no mask-border. The descriptor holds source token, image bytes, pixel size, slice offsets, widths, outsets, repeat pair, fill flag, and mode `alpha` or `luminance`. Values that are lengths are in points, matching the rest of the list. Path: `internal/layout/mask_border.go` for the type, field added on `BlendGroup` in `internal/layout/blend_group.go`.
- [x] A `mask-border-source` of `none`, including the shorthand reset, stores no descriptor. Expected: `mask: url(mask.png)` still creates the first-wave `Mask` and does not create `MaskBorder`. Proof: extend `TestMaskShorthandExpansionAndBorderReset` with a layout fixture that had a mask-border source earlier in the cascade and asserts the group has a nil `MaskBorder` after the shorthand.
- [~] In-repo pixel replay of the mask-border. Reason: the host composites, as with the first-wave mask. Painting the slices as `OpImage` would show a border image, which is the wrong effect. Owner: the display-list host. Next gate: a host fixture applies the group's `MaskBorder` and checks the alpha around the border box. Do not add that fixture's rasterizer to this repo.

Gate. The descriptor is specified. The reset test's expectations are unchanged. Replay stays out.

## Phase 1: longhands on the group

Syntax from the catalog, which points at the border-image types:

- `mask-border-source` matches `border-image-source`: `none | <image>`.
- `mask-border-slice` matches `border-image-slice`, including the `fill` flag.
- `mask-border-width` matches `border-image-width`.
- `mask-border-outset` matches `border-image-outset`.
- `mask-border-repeat` matches `border-image-repeat`.
- `mask-border-mode` is `luminance | alpha`. Initial `alpha`, which is what the mask shorthand reset already writes.

- [x] Accept and store the six longhands. Invalid values do not replace an earlier valid value. Path: `internal/layout/mask_border.go`. A URL source uses the image resolver the mask path already uses (`internal/layout/mask_paint.go:111`). A gradient source uses `renderGradientPNG` only after the mask-source ledger accepts gradients. Until that lands, a gradient source fails the accept check and does not replace a URL. Do not block this phase on gradients.
- [x] Attach a `MaskBorder` when the used source is not `none`. Resolve slice, width, outset, and repeat with the same number rules as `border_image.go`, by calling shared helpers if they can return numbers without appending ops. If a helper always appends `OpImage`, duplicate the arithmetic in `mask_border.go` rather than emitting ops. Expected: no new `OpImage` for the mask-border source. The element's background image count stays the same as a document without `mask-border`. Proof: `TestBehaviorMaskBorderDescriptorNotPaint` in `internal/layout/css_behavior_mask_border_test.go`.
- [x] `mask-border-slice: 30 fill` sets the fill flag and the four slice numbers. `mask-border-width: 10px` and `mask-border-outset: 10px` show up as points on the descriptor (`10px` is 7.5pt). `mask-border-repeat: repeat` stores the repeat token rather than tiling pixels. Proof: the descriptor fields on the group.
- [x] `mask-border-mode: luminance` is stored and not baked into the bytes, unless the source is an SVG mask fragment that the first-wave path already preprocesses. A normal PNG keeps its bytes and the mode token, matching `MaskLayer.Mode` (`internal/layout/blend_group.go:57-60`). Proof: mode field equals `luminance` and bytes equal the resolver payload.
- [x] The first-wave `Mask` and the new `MaskBorder` can both be set. They do not clobber each other. Proof: one element with `mask-image` and `mask-border-source` has both pointers non-nil.

Gate. The group carries numbers and source bytes. The op list gains no border-image cells. Run `go test ./internal/layout -run 'TestBehaviorMaskBorder|TestMaskShorthand'`.

## Phase 2: the mask-border shorthand

- [x] Expand `mask-border` with the catalog syntax: source, slice, optional width, optional outset, repeat, and mode, in the combinations the syntax allows. Omitted pieces use the initial values the mask shorthand already resets to (`internal/layout/style_cascade.go:908-914`). Path: the cascade expander, helper in `internal/layout/mask_border.go`.
- [x] A following longhand overrides the shorthand piece it names. An earlier longhand loses. Proof: a cascade test in `internal/layout/style_cascade_test.go` that mirrors `TestMaskShorthandExpansionAndBorderReset`.
- [x] CSS-wide keywords on `mask-border` reset all seven longhands, including `mask-border` itself. Proof: `initial` yields source `none` and therefore no descriptor at layout.
- [x] `@supports (mask-border: url(border.png) 30 fill / 10px / 10px stretch alpha)` matches the shorthand accept function. A negative slice does not match. Proof: one supports test.
- [x] The existing `mask` shorthand reset still wins over an earlier `mask-border`. Do not edit the reset values. Proof: `TestMaskShorthandExpansionAndBorderReset` still passes, plus the layout fixture from Phase 0.

Gate. The shorthand is sugar for the longhands. It does not paint.

## Phase 3: WebKit aliases

Map these in the existing WebKit normalizer. `normalizeWebkitTransformProperty` is the wrong function. Mask aliases already have `normalizeWebkitMaskProperty`, used from `normalizeVendorPrefix` (`internal/layout/style_cascade.go:1679-1686`). Add the six names there, or in a small helper that function calls. Do not put them in `layout.go`.

| Alias | Maps to |
|---|---|
| `-webkit-mask-box-image` | `mask-border` |
| `-webkit-mask-box-image-source` | `mask-border-source` |
| `-webkit-mask-box-image-slice` | `mask-border-slice` |
| `-webkit-mask-box-image-width` | `mask-border-width` |
| `-webkit-mask-box-image-outset` | `mask-border-outset` |
| `-webkit-mask-box-image-repeat` | `mask-border-repeat` |

There is no mode alias in this set. Do not invent `-webkit-mask-box-image-mode`.

- [x] Each alias reaches the same descriptor fields as the standard property. Proof: `TestBehaviorMaskBorderAliasReachesGroup`, one declaration per alias.
- [x] Source order between alias and standard property follows the cascade, both declaration orders. Proof: a pair of stylesheets, same pattern as the first wave's alias order test.
- [x] `@supports` accepts a prefixed source and rejects a bogus slice. Proof: one supports test.
- [x] The eight first-wave `-webkit-mask*` aliases stay mapped as they are. This phase does not edit their tests except to confirm they still pass.

Gate. Six aliases, one code path. Run `go test ./internal/layout -run 'TestBehaviorMaskBorder|TestBehaviorMask'`.

## Phase 4: catalog, WASM, docs, and gates

- [ ] Mark the 13 names Partial, not implemented. The group carries the data and this repo does not apply the mask pixels. Limitations say that, and say gradient sources wait on the mask-source ledger if they are still rejected.
- [ ] Serialize `MaskBorder` beside `Mask` in the WASM group JSON. The mask copy is `bindings/wasm/drawing_list.go:578-581`. Add the border object next to `drawMaskJSON` (`bindings/wasm/drawing_list.go:212-221`). Proof: a WASM contract test reads slice, width, outset, repeat, mode, and image bytes.
- [x] Update `documentation/deferred.md:108` and the group C sentence at `documentation/deferred.md:119`. The aliases are no longer "no consumer" once Phase 3 passes. Pixel replay stays unsupported here.
- [x] Update `documentation/library-api.md` only to name the new group field. Do not claim the engine masks pixels.
- [ ] Regenerate the compatibility matrix.
- [ ] Update the local knowledge-base mask page in the implementation session.
- [ ] Run `make test`. Exit 0.
- [ ] Run `make golden`. Exit 0.
- [ ] Run `make lint`. Exit 0. `internal/layout/layout.go` did not grow.
- [ ] Run `make claim-scan`. Exit 0.
- [ ] Run `make catalog-check`. Exit 0.
- [ ] Run `make matrix-check`. Exit 0.

GitHub CI runs only `make test` and `make lint`. Do not run bare `go test ./...`.

Gate. Catalog status is Partial for all 13. The reset test from the first wave still passes.

## Dependencies

- First-wave mask group and the shorthand reset (`internal/layout/style_cascade.go:855-914`). This ledger must not break that reset.
- Border-image arithmetic is the pattern (`internal/layout/border_image.go:294-350`). The `OpImage` painter is not the output.
- Image bytes come from the existing resolver (`internal/layout/mask_paint.go:111`). Gradients come from the mask-source ledger if that has landed. This ledger does not start a new rasterizer.
- Pixel application stays with the host. The Phase 0 non-goal is the boundary.
- No new direct module.
