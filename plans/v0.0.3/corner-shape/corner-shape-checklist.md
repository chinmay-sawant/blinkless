# v0.0.3 - Corner shape

> **Parent:** `plans/v0.0.3/README.md`
> **Status:** not started. Baseline rows record verified current behavior. Implementation rows are open.
> **Estimated effort:** 34 catalog properties, all unsupported, across 5 phases. `border-radius` is not in this count.

---

## Overview

`corner-shape` replaces the quarter-ellipse at a rounded corner with another curve. The corner box is still the used border radius. The curve inside that box is the new part.

Blinkless already paints elliptical corners. A background fill is `OpFillRect` with per-corner radii (`internal/layout/layout_chrome.go:473-486`). A solid rounded border is `OpStrokeRect` with the same radii (`internal/layout/layout_chrome.go:656-675`). The public comment says a fill rect carries per-corner radii for a rounded or elliptical box (`layout/displaylist.go:80-82`). There is no path kind (`internal/layout/layout.go:369-385`). A scoop, notch, bevel, or squircle does not fit those radii fields.

This ledger is the 34 `corner*` names from the catalog. All 34 are `unsupported`. Ordinary `border-radius` and its longhands stay as they are. Do not replan them. When a `corner*` shorthand also sets a radius, it calls the existing radius setters.

`documentation/deferred.md` calls this family a permanent non-goal. This file replaces that sentence as the planning home. Do not edit `deferred.md`.

The pinned webref stores the keyword type as `<corner-shape-value>` and does not expand it. The CSS Borders 4 draft (editor's draft, fetched 2026-10-15) defines:

`round | scoop | bevel | notch | square | squircle | superellipse(<number> | infinity | -infinity)`

Keyword to superellipse K: `round` is 1, `squircle` is 2, `square` is infinity, `bevel` is 0, `scoop` is -1, `notch` is -infinity. `round` is the initial value and must match today's quarter-ellipse. Source: `https://drafts.csswg.org/css-borders-4/#typedef-corner-shape-value`. This is not a Chrome parity claim. The catalog does not record a browser result for these names.

The output stays a display list. Do not add a page rasterizer. Do not grow `internal/layout/layout.go`.

## Executive summary

| Piece | Count | Home |
|---|---|---|
| Shape longhands, one corner | 8 | physical and logical single corners |
| Shape longhands, one or two values | 8 | top, right, bottom, left, and the block and inline pairs |
| `corner-shape` | 1 | one to four values, top then right then bottom then left |
| Shorthands that also set a radius | 17 | `corner` plus the physical and logical pairs |
| Total | 34 | all unsupported today |

Logical names use the horizontal left-to-right map already hardcoded for border radii (`internal/layout/style_logical_border.go:3-16`). `start-start` is top-left. This wave does not add a writing-mode table.

## Phase 0: contract and baseline

- [x] Elliptical corners already paint. `border-top-left-radius: 16px` sets only `RadiusTopLeft` on the red fill (`internal/layout/css_behavior_borders_test.go:207`). Radii are resolved, then overlap-reduced, in `usedBorderRadiiXY` (`internal/layout/border_radius.go:189`). Both axes are stored (`internal/layout/layout.go:441-446`).
- [x] Rounded solid borders reuse those radii on `OpStrokeRect` (`internal/layout/layout_chrome.go:656-675`). Dashed sides stay line segments that stop at the radii (`internal/layout/layout_chrome.go:713-716`).
- [x] Logical border radii use a fixed map: `start-start` writes top-left (`internal/layout/style_logical_border.go:6-8`). The opened test checks that map in horizontal left-to-right flow (`internal/layout/css_behavior_borders_rest_test.go:405`).
- [x] The display list has no path operation (`internal/layout/layout.go:369-385`, `layout/displaylist.go:80-99`). A consumer that does not know a kind skips it (`bindings/wasm/drawing_list.go:485-486`).
- [ ] Add one path kind before any scoop paint. Constant `OpPath` with explicit value 9 in new file `internal/layout/corner_path.go`. Do not insert it into the iota block in `layout.go`. `OpGridRun` is the last iota value (`internal/layout/layout.go:384`). Noop stays 255 (`internal/layout/layout_measure.go:1218`).
- [ ] Put the path commands on `opExtra` (`internal/layout/op_extra.go:11`), not on the hot `Op` struct. The hot record is documented at 256 bytes (`internal/layout/layout.go:404-410`). One new pointer field on `opExtra` is allowed. No new fields in `layout.go`.
- [ ] Restate the kind in `layout/displaylist.go` next to the other kind constants, and teach `drawKind` in `bindings/wasm/drawing_list.go` so the default arm does not report it as unknown.
- [ ] `round` (K = 1) does not emit `OpPath`. It keeps today's radii. A box with `border-radius` and no `corner-shape`, or with `corner-shape: round`, paints the same fill and stroke ops as today.
- [ ] Any other K emits one outer path for the background fill and one for the border stroke. Both use the used border-radius as the corner box. The border image path (`internal/layout/layout_chrome.go:513-514`) stays on the image painter and ignores `corner-shape` in this wave.
- [ ] The CSS Borders 4 draft also scales `box-shadow` and the overflow clip from the corner path, instead of reusing the border curve. This wave does not do that scale. `box-shadow` keeps today's radii (`internal/layout/layout_chrome.go:479`). Overflow clip keeps the padding box (`internal/layout/layout_chrome.go:468-469`). The behavior test says so.
- [~] Vertical writing modes. Reason: `logicalRadiusCorner` always returns top-left for `start-start` and does not read `writing-mode` (`internal/layout/style_logical_border.go:3-16`). Owner: this ledger. Next gate: a writing-mode map shared with border logical radii, not a corner-only table.

**Gate:** no paint work starts until the path kind, the extra payload, the Go alias, and the WASM kind all exist. `round` must not change `layout.TestDisplay`.

## Phase 1: store the 34 names

New file `internal/layout/style_corner_shape.go`. Invalid values leave the previous declaration. The initial shape is `round` on all four corners. Radius tokens go through `setCornerRadius` / `assignCornerRadiiX` (`internal/layout/border_radius.go:159`). Do not add a second radius store.

Shape value acceptance, shared by every `*-shape` row:

- [ ] Accept `round`, `scoop`, `bevel`, `notch`, `square`, `squircle`, and `superellipse()` with a number, `infinity`, or `-infinity`. Store the K number, not the keyword text, plus the keyword so the declared form can be tested. Reject a bad token.

### Single-corner shape longhands

Each row stores one K on that corner. Physical names write that corner. Logical names use `logicalRadiusCorner`'s map.

- [ ] `corner-top-left-shape`. Catalog `<corner-shape-value>`. Writes the top-left K.
- [ ] `corner-top-right-shape`. Catalog `<corner-shape-value>`. Writes the top-right K.
- [ ] `corner-bottom-right-shape`. Catalog `<corner-shape-value>`. Writes the bottom-right K.
- [ ] `corner-bottom-left-shape`. Catalog `<corner-shape-value>`. Writes the bottom-left K.
- [ ] `corner-start-start-shape`. Catalog `<corner-shape-value>`. Writes the top-left K.
- [ ] `corner-start-end-shape`. Catalog `<corner-shape-value>`. Writes the top-right K.
- [ ] `corner-end-start-shape`. Catalog `<corner-shape-value>`. Writes the bottom-left K.
- [ ] `corner-end-end-shape`. Catalog `<corner-shape-value>`. Writes the bottom-right K.

### One-or-two-value shape longhands

One value sets both corners. Two values follow CSS Borders 4 section 3.8.1: physical pairs go left-to-right or top-to-bottom, and logical pairs go start then end on the other axis. `border-top-radius` in this tree does not do that. It writes one radius onto both top corners (`internal/layout/style_paint_props.go:282-284`). Do not copy its two-token split. Those two tokens are the x and y radii of one corner (`internal/layout/style_paint_props.go:306-322`).

- [ ] `corner-top-shape`. Catalog `<'corner-top-left-shape'>{1,2}`. Left then right: top-left, then top-right.
- [ ] `corner-bottom-shape`. Catalog `<'corner-top-left-shape'>{1,2}`. Left then right: bottom-left, then bottom-right.
- [ ] `corner-left-shape`. Catalog `<'corner-top-left-shape'>{1,2}`. Top then bottom: top-left, then bottom-left.
- [ ] `corner-right-shape`. Catalog `<'corner-top-left-shape'>{1,2}`. Top then bottom: top-right, then bottom-right.
- [ ] `corner-block-start-shape`. Catalog `<'corner-top-left-shape'>{1,2}`. Start then end on the inline axis: `start-start` then `start-end`. On the fixed map that is top-left then top-right.
- [ ] `corner-block-end-shape`. Catalog `<'corner-top-left-shape'>{1,2}`. Start then end: `end-start` then `end-end`. On the fixed map that is bottom-left then bottom-right.
- [ ] `corner-inline-start-shape`. Catalog `<'corner-top-left-shape'>{1,2}`. Start then end on the block axis: `start-start` then `end-start`. On the fixed map that is top-left then bottom-left.
- [ ] `corner-inline-end-shape`. Catalog `<'corner-top-left-shape'>{1,2}`. Start then end: `start-end` then `end-end`. On the fixed map that is top-right then bottom-right.

### Four-corner shape shorthand

- [ ] `corner-shape`. Catalog `<'corner-top-left-shape'>{1,4}`. The fetched draft assigns the four values to the top, right, bottom, and left sides. A missing left copies right, a missing bottom copies top, and a missing right copies top (CSS Borders 4 section 3.8.2). That is not the `border-radius` corner order. One value sets all four corners. A side value sets both corners on that side. When two sides name the same corner, the side later in top, right, bottom, left order wins. The test records that tie-break, because the fetched paragraph does not.

### Shorthands that also set a radius

Catalog grammar is a radius piece, a shape piece, or both. A missing piece resets that piece to the initial value, the same way a border shorthand resets omitted parts. The radius piece calls the existing setter. The shape piece uses the rows above.

- [ ] `corner`. Catalog `<'border-radius'> || <'corner-shape'>`. Sets all four radii and all four shapes.
- [ ] `corner-top`. Catalog `<'border-top-radius'> || <'corner-top-shape'>`.
- [ ] `corner-right`. Catalog `<'border-top-radius'> || <'corner-top-shape'>`.
- [ ] `corner-bottom`. Catalog `<'border-top-radius'> || <'corner-top-shape'>`.
- [ ] `corner-left`. Catalog `<'border-top-radius'> || <'corner-top-shape'>`.
- [ ] `corner-block-start`. Catalog `<'border-top-radius'> || <'corner-top-shape'>`.
- [ ] `corner-block-end`. Catalog `<'border-top-radius'> || <'corner-top-shape'>`.
- [ ] `corner-inline-start`. Catalog `<'border-top-radius'> || <'corner-top-shape'>`.
- [ ] `corner-inline-end`. Catalog `<'border-top-radius'> || <'corner-top-shape'>`.
- [ ] `corner-top-left`. Catalog `<'border-top-left-radius'> || <'corner-top-left-shape'>`.
- [ ] `corner-top-right`. Catalog `<'border-top-left-radius'> || <'corner-top-left-shape'>`.
- [ ] `corner-bottom-right`. Catalog `<'border-top-left-radius'> || <'corner-top-left-shape'>`.
- [ ] `corner-bottom-left`. Catalog `<'border-top-left-radius'> || <'corner-top-left-shape'>`.
- [ ] `corner-start-start`. Catalog `<'border-top-left-radius'> || <'corner-top-left-shape'>`. Maps to top-left.
- [ ] `corner-start-end`. Catalog `<'border-top-left-radius'> || <'corner-top-left-shape'>`. Maps to top-right.
- [ ] `corner-end-start`. Catalog `<'border-top-left-radius'> || <'corner-top-left-shape'>`. Maps to bottom-left.
- [ ] `corner-end-end`. Catalog `<'border-top-left-radius'> || <'corner-top-left-shape'>`. Maps to bottom-right.

**Gate:** a store test for each of the 34 names. No paint yet. `border-radius: 16px` without a corner property still sets only radii.

## Phase 2: paint

New file `internal/layout/corner_shape_paint.go`, called from `prependChrome` (`internal/layout/layout_chrome.go:460`) where the radii are chosen. Do not add the branch inside `layout.go`.

- [ ] If every corner is `round`, keep the existing fill rect and stroke rect. Zero `OpPath`.
- [ ] If any corner is not `round`, emit `OpPath` for the background fill instead of a radiused `OpFillRect`. The path's outer edge follows the superellipse in each corner box. Straight edges between corners stay straight.
- [ ] Emit `OpPath` for the border stroke of that box instead of `OpStrokeRect`. Stroke width and color still come from the existing border sides. A first cut may use one width and one color when the four sides match. Differing side colors stay on the existing stroke-rect path until a follow-up row says otherwise.
- [ ] Dashed and dotted borders keep today's line segments when every corner is `round`. When a corner is not `round`, do not run the dash segments through the new curve in this wave. Paint a solid path, and add a test that says dashed-plus-scoop is solid for now.
- [ ] `border-image` stays the existing image painter and does not read corner shape.
- [ ] Overflow clip and background-clip stay on the padding or content box they already use (`internal/layout/layout_chrome.go:468-496`). They do not follow a scooped outer path in this wave. Say that in the behavior test so it is not an accident.

**Gate:** a scooped box emits `OpPath` and does not emit a radiused fill for that background. A `round` box emits neither a new kind nor a new radius.

## Phase 3: behavior tests

New file `internal/layout/css_behavior_corner_shape_test.go`. Each row checks the drawing list, not a page PNG.

- [ ] `corner-shape: round` on a 16px radius matches the op geometry of `border-radius: 16px` alone.
- [ ] `corner-shape: bevel` emits `OpPath` and does not set elliptical radii on a fill rect for that box.
- [ ] `corner-shape: scoop` emits a path whose corner is concave relative to `round`. Compare the path commands, not a screenshot.
- [ ] `corner-shape: notch` and `corner-shape: square` and `corner-shape: squircle` each emit a path and store a different K.
- [ ] `superellipse(1)` matches `round`. `superellipse(0)` matches `bevel`. `superellipse(-1)` matches `scoop`.
- [ ] `corner-top-left-shape: bevel` changes only the top-left corner. The other three stay `round` and may stay on the radius fields if the painter can mix one path corner with elliptical corners. If the painter cannot mix, all four corners go to `OpPath` and the test says the other three K values are 1.
- [ ] `corner-top-right-shape`, `corner-bottom-right-shape`, and `corner-bottom-left-shape` each change only their corner, under the same mix rule.
- [ ] `corner-top-shape: bevel scoop` sets top-left bevel and top-right scoop.
- [ ] `corner-bottom-shape`, `corner-left-shape`, and `corner-right-shape` set the two corners named in phase 1.
- [ ] `corner-block-start-shape: notch` sets both top corners on the fixed map.
- [ ] `corner-block-end-shape`, `corner-inline-start-shape`, and `corner-inline-end-shape` set the pairs from phase 1.
- [ ] `corner-start-start-shape: bevel` matches `corner-top-left-shape: bevel` in horizontal left-to-right flow (`internal/layout/css_behavior_borders_rest_test.go:405` is the radius precedent).
- [ ] `corner-start-end-shape`, `corner-end-start-shape`, and `corner-end-end-shape` match top-right, bottom-left, and bottom-right.
- [ ] `corner: 16px bevel` sets a 16px radius and K = 0 on all four corners.
- [ ] `corner-top: 16px scoop` sets the top radii and the top shapes, and leaves the bottom corners at the initial radius and `round`.
- [ ] `corner-right`, `corner-bottom`, and `corner-left` do the same for their sides.
- [ ] `corner-block-start`, `corner-block-end`, `corner-inline-start`, and `corner-inline-end` follow the fixed logical map.
- [ ] `corner-top-left: 16px notch` sets only the top-left radius and the top-left K.
- [ ] `corner-top-right`, `corner-bottom-right`, and `corner-bottom-left` do the same for their corner.
- [ ] `corner-start-start`, `corner-start-end`, `corner-end-start`, and `corner-end-end` match those physical corners in horizontal left-to-right flow.
- [ ] A bad shape token does not replace an earlier `bevel`.
- [ ] WASM JSON includes the path kind and the path commands for one scooped box (`bindings/wasm/drawing_list.go`).

**Gate:** the 34 names each appear in a test above. `make golden` stays green because `round` does not move existing radius fixtures.

## Phase 4: catalog and closure

- [ ] Set all 34 catalog rows to `implemented` only after the phase 3 test for that name exists. Handler `style_corner_shape.go`. Partial is the right status if the dashed-border and background-clip limits in phase 2 are still open. Say that limit in the catalog `limitations` field.
- [ ] Do not change `border-radius` catalog rows.
- [ ] Regenerate the matrix property section.
- [ ] Do not edit `documentation/deferred.md` unless the user asks.
- [ ] Do not grow `internal/layout/layout.go`.
- [ ] Run `make test`. Do not run bare `go test ./...`.
- [ ] Run `make golden`.
- [ ] Run `make lint`.
- [ ] Run `make claim-scan`. Do not claim the scooped corner is a page PNG or a Chrome screenshot.
- [ ] Run `make catalog-check` and `make matrix-check`.
- [ ] GitHub CI runs only `make test` and `make lint` (`.github/workflows/ci.yml:1`, `.github/workflows/ci.yml:20`, `.github/workflows/ci.yml:26`).

Focused run during the work:

```sh
go test ./internal/layout -run 'TestBehaviorBorderTopLeftRadius|TestCornerShape|TestBehaviorBorderStartStartRadius' -count=1
go test ./bindings/wasm -run 'TestDrawing' -count=1
```

**Gate:** open until those commands exit 0 on the change that flips the catalog.

## Dependencies

- Radius math stays in `internal/layout/border_radius.go`. Corner shape calls it and does not copy it.
- Paint entry stays `prependChrome` in `internal/layout/layout_chrome.go`. The new branch lives in `internal/layout/corner_shape_paint.go`.
- Logical map stays `logicalRadiusCorner` in `internal/layout/style_logical_border.go`. Vertical writing modes are the non-goal in phase 0.
- The path payload lives on `opExtra` (`internal/layout/op_extra.go`). The kind constant lives in `internal/layout/corner_path.go`.
- Public kinds and WASM kinds must learn `OpPath` in the same change as the constant, or a caller will skip the scoop.
- `border-image`, overflow clip, and dashed borders are named limits, not silent drops.
