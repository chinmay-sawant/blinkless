# v0.0.3 - Mask source holes

> **Parent:** `plans/v0.0.3/README.md`
> **Status:** phases 1 through 3 landed; their closure gates exited 0 on 2026-10-16. The acceptance grammar wave landed 2026-10-16: 16 mask-family catalog rows are implemented, while `mask-type` and `-webkit-mask-composite` stay partial (see the acceptance wave section). The wave's end-of-session gate exited 0 on 2026-10-16.
> **Estimated effort:** 2 source kinds plus 1 non-goal, 4 phases.

---

## Overview

This ledger is the leftover mask work. It is not a second copy of the first wave.

The finished ledger is `plans/v0.0.1/css-masking/css-masking-checklist.md`. That wave shipped 10 standard properties and 8 `-webkit-mask*` aliases. The acceptance wave later brought 16 of the 18 mask-family catalog rows to implemented; `mask-type` and `-webkit-mask-composite` stay partial. The drawing list carries mask group data. This repo does not replay mask pixels. Do not replan `mask`, `mask-image`, `mask-mode`, `mask-clip`, `mask-composite`, `mask-origin`, `mask-position`, `mask-repeat`, `mask-size`, `mask-type`, or those shipped aliases. Their group contract stays.

The holes this ledger was written against were CSS gradient sources, `image-set()` sources, and an in-repo pixel replay backend. Phases 1 and 2 closed the first two. Catalog text at planning time, read on 2026-10-15 (`properties.json` `updated` is `2026-10-13`):

- `mask` limitation: gradients, `image-set`, mask-border painting, and an in-repo pixel replay backend are unsupported.
- `mask-image` limitation: only `url(...)` and `none` are accepted. Gradients and `image-set` are unsupported. Pixels are not replayed in this repository.

The current catalog no longer carries the gradient or `image-set` limits on `mask` and `mask-image`. Mask-border painting is the mask-border ledger, not this one; pixel replay stays a host contract.

The layer accept predicate is `maskImageLayerValueAccepted` (`internal/layout/style_mask_accept.go:563-584`): `none`, CSS gradients, `image-set()`, and `url()`. The shorthand classifier calls the same function (`internal/layout/style_mask_accept.go:283`). A parenthesized function already stays one token: `nextSpaceToken` does not split inside parentheses (`internal/layout/style_values.go:459-487`), and layer commas split only at depth zero (`internal/layout/helpers.go:6-9`).

Gradient rasterization for backgrounds already exists. `isGradientFunc` recognizes linear, radial, and conic gradients, including the repeating forms (`internal/layout/gradient.go:43-50`). `renderGradientPNG` returns PNG bytes (`internal/layout/gradient.go:471-473`) and caps a side at 512 (`internal/layout/gradient.go:486`). Mask resolution now picks the `image-set()` candidate and rasterizes gradients (`internal/layout/mask_paint.go:49-58`, `:92-108`), or fetches the URL (`:196-237`, `:239-255`), and stores bytes on `MaskLayer` (`internal/layout/blend_group.go:53-75`).

The host draws the group. `BlendGroup` says a painter applies the mask after the group's ops (`internal/layout/blend_group.go:13-15`). `DisplayList` does not paint a page bitmap (`layout/displaylist.go:144-148`). Keep it that way.

New source helpers go in `internal/layout/mask_source.go`. `resolveMaskLayer` may call them. Do not add this code to `internal/layout/layout.go`.

Paint Y grows downward. 1 CSS pixel is 0.75 points (`layout/displaylist.go:13-14`).

## Executive summary

| Hole | At planning time | Outcome |
|---|---|---|
| CSS gradients as `mask-image` | rejected | PNG bytes on the existing `MaskLayer`; implemented |
| `image-set()` as `mask-image` | no parser in `*.go` | pick one candidate, then the URL or gradient path; implemented |
| In-repo pixel replay | absent on purpose | non-goal, host composites |

The 16 mask-family rows are implemented after the acceptance wave. Accepting a gradient does not mean Blinkless composites the mask onto a page; pixel replay stays a host contract, not a partial catalog status.

## Phase 0: the hole, and the replay decision

- [x] The first wave is complete and is not reopened. Evidence: `plans/v0.0.1/css-masking/css-masking-checklist.md` marks the 18 names done on 2026-10-13. The acceptance wave then moved 16 mask-family rows to implemented; `mask-type` and `-webkit-mask-composite` stay partial (see the acceptance wave section).
- [x] Gradients were rejected at acceptance before phase 1. The phase 1 change flipped those fixtures: `@supports (mask-image: linear-gradient(red, blue))` now applies (`TestBehaviorMaskSupportsMatchesAcceptedValues`, `internal/layout/css_behavior_mask_test.go:375-404`), the invalid-does-not-replace fixture uses `not-an-image()` (`TestBehaviorMaskInvalidValuesDoNotReplaceEarlierDeclarations`, `:406-419`), and the accept table expects `mask-image: linear-gradient(red, blue)` true (`internal/layout/style_value_accept_test.go:111`).
- [x] `image-set` had no Go parser when this ledger was written. Phase 2 added one in `internal/layout/mask_source.go`; proof: `TestBehaviorMaskImageSetPicks1x` (`internal/layout/css_behavior_mask_test.go:440`).
- [x] URL masks already put bytes on the group. `maskDescriptor` builds `MaskLayer` values (`internal/layout/mask_paint.go:25-81`) and `resolveRasterMaskLayer` fills image bytes (`internal/layout/mask_paint.go:239-255`).
- [~] In-repo pixel replay. Reason: the host composites the group. This repository's output is the drawing list, and the first wave already recorded that split. Adding a page rasterizer would undo it. Owner: the display-list host, including the native view renderer in `plans/v0.0.1/native-window/native-web-view-checklist.md`, not Blinkless layout. Next gate: a host fixture shows a masked group whose pixels match the Chrome reference captured for the first wave. Until that gate, do not rasterize a page here, and do not apply mask pixels inside `layout.DisplayList`.

Gate. Replay stays out. The next phases only add source bytes to the group the host already receives.

## Phase 1: gradient sources

- [x] Accept a gradient function as a `mask-image` layer when `isGradientFunc` is true (`internal/layout/gradient.go:43-50`). `none` and `url()` stay accepted. A broken gradient still fails and does not replace an earlier valid image. Path: `maskImageLayerValueAccepted` in `internal/layout/style_mask_accept.go`, with the gradient check helper in `internal/layout/mask_source.go`.
- [x] When the shorthand image token is a gradient, `parseMaskShorthand` keeps it as `layer.image` because the classifier already calls `maskImageLayerValueAccepted` (`internal/layout/style_mask_accept.go:283`). Do not rework position, size, repeat, origin, clip, or composite. Expected: `mask: linear-gradient(red, blue)` expands to `mask-image` of that function and still resets mask-border the way `TestMaskShorthandExpansionAndBorderReset` pins (`internal/layout/style_cascade.go:908-914`, `internal/layout/style_cascade_test.go:76-111`). Proof: extend that cascade test with a gradient image token. The reset expectations stay.
- [x] Rasterize the gradient to PNG with `renderGradientPNG` (`internal/layout/gradient.go:471-473`); the mask descriptor routes gradients through `fillMaskGradient` (`internal/layout/mask_paint.go:53-58`, `:92-108`). Store the bytes, pixel width, and pixel height on the existing `MaskLayer` fields (`internal/layout/blend_group.go:59-61`). Leave `Source` as the gradient function text. Do not set `Preprocessed` unless the SVG mask path already would. Do not emit a background paint op.
- [x] Choose the raster size from the mask layer's resolved length, and from the element's border box when the size is `auto`. Convert CSS pixels to the pixel dimensions the gradient function already takes. Honor the existing 512 cap (`internal/layout/gradient.go:486`). Expected: a 20px by 10px box does not ask for a larger PNG than the cap. Proof: the behavior test reads `PixelWidth` and `PixelHeight`.
- [x] Update the old rejection tests in the same change. `@supports (mask-image: linear-gradient(red, blue))` became a branch that does apply. The "invalid does not replace" fixture needs a value that is still invalid, such as `mask-image: not-an-image()`. `style_value_accept_test.go:111` flips to true for a gradient. Proof: `TestBehaviorMaskGradientBytesOnGroup` (`internal/layout/css_behavior_mask_test.go:433`) finds a non-empty PNG signature in `ImageBytes` and does not find a new filled rect for the gradient itself.
- [x] Repeating gradients use the same `isGradientFunc` prefixes. One repeating fixture is enough if `renderGradientPNG` already accepts that string for backgrounds. If it returns false, leave that form rejected and name it in the catalog limitation. Do not write a second gradient rasterizer.

Gate. A gradient mask is group data plus PNG bytes. The element's own fill is still the unmasked paint op. Run `go test ./internal/layout -run 'TestBehaviorMask|TestMaskShorthand'`.

## Phase 2: image-set sources

- [x] Parse `image-set()` in `internal/layout/mask_source.go`. Accept a list of candidates. Each candidate is a `url()`, or a gradient once Phase 1 has landed, followed by an optional resolution (`1x`, `2dppx`) or type. This session found no existing parser, so the grammar is the CSS images `image-set()` form. Read that form before coding tokens this file does not already name. Reject an empty `image-set()` and do not replace an earlier valid image.
- [x] Pick the `1x` candidate when one exists. Otherwise pick the first candidate. Do not fetch a network URL beyond the image resolver the URL path already uses (`internal/layout/mask_paint.go:239-255`). Expected: `image-set(url(a.png) 2x, url(b.png) 1x)` resolves `b.png`. Proof: `TestBehaviorMaskImageSetPicks1x` with the test image map.
- [x] A chosen URL goes through `resolveRasterMaskLayer`. A chosen gradient goes through the Phase 1 PNG path. `Source` on the layer becomes the chosen candidate text, not the whole `image-set()` string, so the host does not parse `image-set` again. Proof: the group layer source and bytes.
- [x] The shorthand accepts `image-set()` as one image token for the same reason a gradient stays one token (`internal/layout/style_values.go:474-481`). Expected: `mask: image-set(url(b.png) 1x)` expands `mask-image` to that function and still resets mask-border. Proof: one cascade assertion beside the gradient shorthand test.
- [x] `@supports (mask-image: image-set(url(b.png) 1x))` matches the accept function. An `image-set` with no candidates does not match. Proof: extend `TestBehaviorMaskSupportsMatchesAcceptedValues`.

Gate. `image-set` reduces to one layer the URL or gradient path already stores. No new group field.

## Phase 3: catalog, docs, and gates

- [x] Update the `mask` and `mask-image` limitations so they no longer say gradients and `image-set` are unsupported, once Phases 1 and 2 pass. Keep the pixel-replay sentence, and point it at the Phase 0 non-goal. The acceptance wave later moved these rows to implemented; pixel replay is a host contract, not a partial status. Do not flip the other eight standard mask properties' text except where it repeats the same source sentence. The shipped `-webkit-mask*` aliases share the standard accept path (`normalizeVendorPrefix` maps them at `internal/layout/style_cascade.go:1856-1877`). Their limitations get the same source sentence and no other edit.
- [x] Update `documentation/deferred.md` only in the mask sentence that still said gradients and pixel replay were unsupported (`documentation/deferred.md:108`; the acceptance wave updated that row again). Pixel replay remains unsupported in this repository. Gradients and `image-set` do not.
- [x] Regenerate the compatibility matrix. The first-wave matrix rows repeated the gradient limit at planning time (`documentation/compatibility-matrix.md` mask rows).
- [x] WASM already copies `group.Mask` layers to JSON (`bindings/wasm/drawing_list.go:578-581`). Gradient and `image-set` bytes use that copy if `ImageBytes` is set. Add a WASM test only if the existing mask group JSON test ignores non-URL sources. Do not add a replay canvas to the WASM build.
- [x] Update the local knowledge-base mask page in the implementation session.
- [x] Run `make test`. Exit 0. `make test` exited 0 on 2026-10-16.
- [x] Run `make golden`. Exit 0.
- [x] Run `make lint`. Exit 0. `internal/layout/layout.go` did not grow. It is still 2142 lines. `make lint` exited 0 on 2026-10-16.
- [x] Run `make claim-scan`. Exit 0. Do not claim the engine composites mask pixels.
- [x] Run `make catalog-check`. Exit 0.
- [x] Run `make matrix-check`. Exit 0.

GitHub CI runs only `make test` and `make lint`. Do not run bare `go test ./...`.

Gate. Limitations match the new sources and still say this repo does not replay pixels.

## Acceptance wave (2026-10-16): 16 mask-family rows implemented

The catalog moved 16 rows from Partial to implemented: the 10 standard mask properties except `mask-type`, plus the 8 `-webkit-mask*` aliases except `-webkit-mask-composite`. Acceptance, computed values, and descriptor pass-through are complete; pixel replay stays a host contract.

- [x] Acceptance grammar: `maskImageLayerValueAccepted`, `maskModeLayerValueAccepted`, `maskPositionLayerValueAccepted`, `maskSizeLayerValueAccepted`, `maskRepeatLayerValueAccepted`, `maskOriginLayerValueAccepted`, `maskClipLayerValueAccepted`, `maskCompositeLayerValueAccepted`, and the shorthand state machine in `internal/layout/style_mask_accept.go`. Evidence: `TestSupportedDeclarationMaskGrammar` pins the grammar in 86 cases (`internal/layout/style_value_accept_test.go:83-212`); the declaration gate calls it from `internal/layout/style_value_accept.go:107-113` and the mask tables at `:283-323`.
- [x] Computed values and descriptor pass-through: the three-token repeat pair and the four-value edge offset (`url(a.png) space round left 10px top 20px`) reach `MaskLayer.Repeat` and `MaskLayer.Position`. Proof: `TestBehaviorMaskShorthandRepeatPairAndOffsetPositionReachDescriptor` (`internal/layout/css_behavior_mask_test.go:211-243`).
- [x] `-webkit-*` aliases share the path: `normalizeWebkitMaskProperty` maps the 8 aliases (`internal/layout/style_cascade.go:1856-1877`), and the group fields match the unprefixed declarations. Proof: `TestBehaviorMaskVendorAliasesReachMaskGroup` (`internal/layout/css_behavior_mask_test.go:245-256`) with `maskAliasCases` (`:272-333`).
- [x] Package gate: `go test ./internal/layout -count=1` -> `ok github.com/chinmay-sawant/blinkless/internal/layout 0.266s` on 2026-10-16. A targeted run including `TestSupportedDeclarationMaskGrammar` and the descriptor test also exits 0 (`ok github.com/chinmay-sawant/blinkless/internal/layout 0.074s`).
- [x] End-of-session full gate on the acceptance-wave change: `make test`, `make golden`, `make lint`, `make claim-scan`, `make catalog-check`, `make matrix-check` all exited 0 on 2026-10-16 (logs under `/tmp/opencode/gates3/`); `make lint` chains `size-check`, also clean. `go test ./internal/layout -count=1` -> `ok github.com/chinmay-sawant/blinkless/internal/layout 0.234s`.

### Known divergences and partials

- `mask-type` stays partial. The declaration cascades into `ResolvedStyle.MaskType` (`internal/layout/style_mask_props.go:90-97`; accepted at `internal/layout/style_value_accept.go:318-319`), but no consumer reads that field. The SVG mask rasterizer takes `mask-type` from the mask element inside the SVG source (`internal/svg/mask.go:401`), so a `mask-type` set by the host document's stylesheet never reaches mask source resolution. The catalog limitation records the gap as "mask-type set through an external stylesheet is not cascaded". The covered path is the SVG file's own attribute or style attribute (`TestBehaviorSVGMaskTypeFlowsIntoMaskImagePayload`, `internal/layout/css_behavior_mask_test.go:113-138`).
- `-webkit-mask-composite` stays partial. The engine accepts `source-over`, `source-in`, `source-out`, and `xor` (`internal/layout/style_mask_accept.go:967-976`); Chrome 155 also accepts `source-atop`, `destination-over`, `destination-in`, `destination-out`, `destination-atop`, `clear`, and `copy`. The rejection of the extra operators is pinned at `internal/layout/style_value_accept_test.go:200`.
- WebKit legacy grammar asymmetry, deliberate and Chrome-matched: unprefixed `mask-origin` and `mask-clip` accept the SVG boxes (`fill-box`, `stroke-box`, `view-box`) and `mask-clip: no-clip` (`internal/layout/style_mask_accept.go:945-956`), while `-webkit-mask-origin` and `-webkit-mask-clip` follow Chrome's narrower legacy grammar and reject them (`internal/layout/style_mask_accept.go:1002-1026`). Pinned by `TestSupportedDeclarationMaskGrammar` cases at `internal/layout/style_value_accept_test.go:168-177`, `:194-197`, `:202-203`.

## Dependencies

- First-wave group contract: `MaskDescriptor` and `MaskLayer` (`internal/layout/blend_group.go:38-75`). This ledger fills `ImageBytes`. It does not add a second mask object.
- Gradient math stays in `internal/layout/gradient.go`. `mask_source.go` calls it.
- Shorthand reset of mask-border stays `internal/layout/style_cascade.go:908-914`. Mask-border painting is `plans/v0.0.3/mask-border/mask-border-checklist.md`.
- Pixel replay stays with the host. The Phase 0 non-goal is the boundary.
- No new direct module.
