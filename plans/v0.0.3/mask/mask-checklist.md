# v0.0.3 - Mask source holes

> **Parent:** `plans/v0.0.3/README.md`
> **Status:** not started, except baseline rows verified in this session.
> **Estimated effort:** 2 source kinds plus 1 non-goal, 4 phases.

---

## Overview

This ledger is the leftover mask work. It is not a second copy of the first wave.

The finished ledger is `plans/v0.0.1/css-masking/css-masking-checklist.md`. That wave shipped 10 standard properties and 8 `-webkit-mask*` aliases as Partial. The drawing list carries mask group data. This repo does not replay mask pixels. Do not replan `mask`, `mask-image`, `mask-mode`, `mask-clip`, `mask-composite`, `mask-origin`, `mask-position`, `mask-repeat`, `mask-size`, `mask-type`, or those shipped aliases. Their group contract stays.

The holes that ledger names, and that the catalog still names, are CSS gradient sources, `image-set()` sources, and an in-repo pixel replay backend. Catalog text, read on 2026-10-15 (`properties.json` `updated` is `2026-10-13`):

- `mask` limitation: gradients, `image-set`, mask-border painting, and an in-repo pixel replay backend are unsupported.
- `mask-image` limitation: only `url(...)` and `none` are accepted. Gradients and `image-set` are unsupported. Pixels are not replayed in this repository.

`mask-border` painting is the mask-border ledger, not this one.

The accept gate is `maskImageLayerValueAccepted`. It returns true only for `none` or a single `url()` (`internal/layout/style_mask_accept.go:480-492`). The shorthand classifier calls that same function (`internal/layout/style_mask_accept.go:233`). A parenthesized function already stays one token: `nextSpaceToken` does not split inside parentheses (`internal/layout/style_values.go:459-486`), and layer commas split only at depth zero (`internal/layout/helpers.go:6-9`).

Gradient rasterization for backgrounds already exists. `isGradientFunc` recognizes linear, radial, and conic gradients, including the repeating forms (`internal/layout/gradient.go:43-50`). `renderGradientPNG` returns PNG bytes (`internal/layout/gradient.go:471-473`) and caps a side at 512 (`internal/layout/gradient.go:486`). Mask resolution today only fetches a URL (`internal/layout/mask_paint.go:68-84`) and stores bytes on `MaskLayer` (`internal/layout/blend_group.go:45-67`).

The host draws the group. `BlendGroup` says a painter applies the mask after the group's ops (`internal/layout/blend_group.go:13-16`). `DisplayList` does not paint a page bitmap (`layout/displaylist.go:129-133`). Keep it that way.

New source helpers go in `internal/layout/mask_source.go`. `resolveMaskLayer` may call them. Do not add this code to `internal/layout/layout.go`.

Paint Y grows downward. 1 CSS pixel is 0.75 points (`layout/displaylist.go:12-14`).

## Executive summary

| Hole | Today | This ledger |
|---|---|---|
| CSS gradients as `mask-image` | rejected | PNG bytes on the existing `MaskLayer` |
| `image-set()` as `mask-image` | no parser in `*.go` | pick one candidate, then the URL or gradient path |
| In-repo pixel replay | absent on purpose | non-goal, host composites |

The first-wave properties stay Partial. Accepting a gradient does not mean Blinkless composites the mask onto a page.

## Phase 0: the hole, and the replay decision

- [x] The first wave is complete and is not reopened. Evidence: `plans/v0.0.1/css-masking/css-masking-checklist.md` marks the 18 names done on 2026-10-13, and the catalog still says Partial for `mask` and `mask-image` with the gradient and `image-set` limits quoted above.
- [x] Gradients are rejected at acceptance. `TestBehaviorMaskSupportsMatchesAcceptedValues` uses `@supports (mask-image: linear-gradient(red, blue))` as the branch that must not apply (`internal/layout/css_behavior_mask_test.go:339-346`). `TestBehaviorMaskInvalidValuesDoNotReplaceEarlierDeclarations` keeps `url(earlier.png)` when a later `linear-gradient` is declared (`internal/layout/css_behavior_mask_test.go:349-361`). `style_value_accept_test.go:87` expects `mask-image` plus `linear-gradient(red, blue)` to be rejected.
- [x] `image-set` has no Go parser. A search of `*.go` finds the word only in docs and plans, not in a function.
- [x] URL masks already put bytes on the group. `maskDescriptor` builds `MaskLayer` values (`internal/layout/mask_paint.go:25-57`) and `resolveRasterMaskLayer` fills image bytes (`internal/layout/mask_paint.go:111`).
- [~] In-repo pixel replay. Reason: the host composites the group. This repository's output is the drawing list, and the first wave already recorded that split. Adding a page rasterizer would undo it. Owner: the display-list host, including the native view renderer in `plans/v0.0.1/native-window/native-web-view-checklist.md`, not Blinkless layout. Next gate: a host fixture shows a masked group whose pixels match the Chrome reference captured for the first wave. Until that gate, do not rasterize a page here, and do not apply mask pixels inside `layout.DisplayList`.

Gate. Replay stays out. The next phases only add source bytes to the group the host already receives.

## Phase 1: gradient sources

- [ ] Accept a gradient function as a `mask-image` layer when `isGradientFunc` is true (`internal/layout/gradient.go:43-50`). `none` and `url()` stay accepted. A broken gradient still fails and does not replace an earlier valid image. Path: `maskImageLayerValueAccepted` in `internal/layout/style_mask_accept.go`, with the gradient check helper in `internal/layout/mask_source.go`.
- [ ] When the shorthand image token is a gradient, `parseMaskShorthand` keeps it as `layer.image` because the classifier already calls `maskImageLayerValueAccepted` (`internal/layout/style_mask_accept.go:229-233`). Do not rework position, size, repeat, origin, clip, or composite. Expected: `mask: linear-gradient(red, blue)` expands to `mask-image` of that function and still resets mask-border the way `TestMaskShorthandExpansionAndBorderReset` pins (`internal/layout/style_cascade.go:908-914`, `internal/layout/style_cascade_test.go:76-110`). Proof: extend that cascade test with a gradient image token. The reset expectations stay.
- [ ] Rasterize the gradient to PNG with `renderGradientPNG` (`internal/layout/gradient.go:471-473`) inside `resolveMaskLayer` (`internal/layout/mask_paint.go:68-72`). Store the bytes, pixel width, and pixel height on the existing `MaskLayer` fields (`internal/layout/blend_group.go:49-52`). Leave `Source` as the gradient function text. Do not set `Preprocessed` unless the SVG mask path already would. Do not emit a background paint op.
- [ ] Choose the raster size from the mask layer's resolved length, and from the element's border box when the size is `auto`. Convert CSS pixels to the pixel dimensions the gradient function already takes. Honor the existing 512 cap (`internal/layout/gradient.go:486`). Expected: a 20px by 10px box does not ask for a larger PNG than the cap. Proof: the behavior test reads `PixelWidth` and `PixelHeight`.
- [ ] Update the old rejection tests in the same change. `@supports (mask-image: linear-gradient(red, blue))` becomes a branch that does apply. The "invalid does not replace" fixture needs a value that is still invalid, such as `mask-image: not-an-image()`. `style_value_accept_test.go:87` flips to true for a gradient. Proof: `TestBehaviorMaskGradientBytesOnGroup` in `internal/layout/css_behavior_mask_test.go` finds a non-empty PNG signature in `ImageBytes` and does not find a new filled rect for the gradient itself.
- [ ] Repeating gradients use the same `isGradientFunc` prefixes. One repeating fixture is enough if `renderGradientPNG` already accepts that string for backgrounds. If it returns false, leave that form rejected and name it in the catalog limitation. Do not write a second gradient rasterizer.

Gate. A gradient mask is group data plus PNG bytes. The element's own fill is still the unmasked paint op. Run `go test ./internal/layout -run 'TestBehaviorMask|TestMaskShorthand'`.

## Phase 2: image-set sources

- [ ] Parse `image-set()` in `internal/layout/mask_source.go`. Accept a list of candidates. Each candidate is a `url()`, or a gradient once Phase 1 has landed, followed by an optional resolution (`1x`, `2dppx`) or type. This session found no existing parser, so the grammar is the CSS images `image-set()` form. Read that form before coding tokens this file does not already name. Reject an empty `image-set()` and do not replace an earlier valid image.
- [ ] Pick the `1x` candidate when one exists. Otherwise pick the first candidate. Do not fetch a network URL beyond the image resolver the URL path already uses (`internal/layout/mask_paint.go:111`). Expected: `image-set(url(a.png) 2x, url(b.png) 1x)` resolves `b.png`. Proof: `TestBehaviorMaskImageSetPicks1x` with the test image map.
- [ ] A chosen URL goes through `resolveRasterMaskLayer`. A chosen gradient goes through the Phase 1 PNG path. `Source` on the layer becomes the chosen candidate text, not the whole `image-set()` string, so the host does not parse `image-set` again. Proof: the group layer source and bytes.
- [ ] The shorthand accepts `image-set()` as one image token for the same reason a gradient stays one token (`internal/layout/style_values.go:474-478`). Expected: `mask: image-set(url(b.png) 1x)` expands `mask-image` to that function and still resets mask-border. Proof: one cascade assertion beside the gradient shorthand test.
- [ ] `@supports (mask-image: image-set(url(b.png) 1x))` matches the accept function. An `image-set` with no candidates does not match. Proof: extend `TestBehaviorMaskSupportsMatchesAcceptedValues`.

Gate. `image-set` reduces to one layer the URL or gradient path already stores. No new group field.

## Phase 3: catalog, docs, and gates

- [ ] Update the `mask` and `mask-image` limitations so they no longer say gradients and `image-set` are unsupported, once Phases 1 and 2 pass. Keep the pixel-replay sentence, and point it at the Phase 0 non-goal. Status stays Partial. Do not flip the other eight standard mask properties' text except where it repeats the same source sentence. The shipped `-webkit-mask*` aliases share the standard accept path (`normalizeVendorPrefix` already maps them). Their limitations get the same source sentence and no other edit.
- [ ] Update `documentation/deferred.md` only in the mask sentence that still says gradients and pixel replay are unsupported (`documentation/deferred.md:108`). Pixel replay remains unsupported in this repository. Gradients and `image-set` do not.
- [ ] Regenerate the compatibility matrix. The first-wave matrix rows currently repeat the gradient limit (`documentation/compatibility-matrix.md` mask rows).
- [ ] WASM already copies `group.Mask` layers to JSON (`bindings/wasm/drawing_list.go:578-581`). Gradient and `image-set` bytes use that copy if `ImageBytes` is set. Add a WASM test only if the existing mask group JSON test ignores non-URL sources. Do not add a replay canvas to the WASM build.
- [ ] Update the local knowledge-base mask page in the implementation session.
- [ ] Run `make test`. Exit 0.
- [ ] Run `make golden`. Exit 0.
- [ ] Run `make lint`. Exit 0. `internal/layout/layout.go` did not grow.
- [ ] Run `make claim-scan`. Exit 0. Do not claim the engine composites mask pixels.
- [ ] Run `make catalog-check`. Exit 0.
- [ ] Run `make matrix-check`. Exit 0.

GitHub CI runs only `make test` and `make lint`. Do not run bare `go test ./...`.

Gate. Limitations match the new sources and still say this repo does not replay pixels.

## Dependencies

- First-wave group contract: `MaskDescriptor` and `MaskLayer` (`internal/layout/blend_group.go:38-67`). This ledger fills `ImageBytes`. It does not add a second mask object.
- Gradient math stays in `internal/layout/gradient.go`. `mask_source.go` calls it.
- Shorthand reset of mask-border stays `internal/layout/style_cascade.go:908-914`. Mask-border painting is `plans/v0.0.3/mask-border/mask-border-checklist.md`.
- Pixel replay stays with the host. The Phase 0 non-goal is the boundary.
- No new direct module.
