# feat(layout): ship corner shapes and the downstream layout fixes

## Summary

- Implements the 34 `corner*` properties from the CSS Borders 4 draft: 8 single-corner shape longhands, 8 pair longhands, `corner-shape`, and 17 radius-plus-shape shorthands. A non-round corner paints as a solid `OpPath` outline; `round` keeps today's radii and emits no path op. The output stays `layout.DisplayList`; no page rasterizer is added.
- Carries five layout fixes that landed while the shape was verified downstream: keep-together ranges for table rows, headings, and single-line blocks; the print width fit; canvas background geometry; object-fit cropping; grid percentage track sizing.
- Catalog: all 34 rows move unsupported to implemented (implemented=488, partial=3, unsupported=286, intentionally ignored=8). The dashed-border, mixed-side, clip, and writing-mode wave limits stay named in each row's `limitations`.

## Motivation / context

- Plans: `plans/v0.0.3/corner-shape/corner-shape-checklist.md` (all rows closed with proofs), `plans/v0.0.3/corner-shape/closure-notes.md`.
- Issues: none filed; `gh issue list --state all` is empty and the checklist is the tracking record.
- Sibling verification: gowkhtmltopdf fixture 67 renders all 34 examples and compares blinkless against Chrome for Testing 155.0.8059.84. That fixture lives in the sibling repo, not this tree.

## Changes

### Corner shapes

- Store the 34 names in `internal/layout/style_corner_shape.go` and its `_corners`, `_pairs`, `_shorthand`, and `_side` siblings: the keyword map (round 1, squircle 2, square infinity, bevel 0, scoop -1, notch -infinity) plus `superellipse(<number> | infinity | -infinity)`. An invalid token keeps the previous declaration.
- Expansion follows border-radius corner order: `corner-shape: scoop bevel notch squircle` sets top-left scoop, top-right bevel, bottom-right notch, bottom-left squircle, and the shorter forms copy like border-radius. `TestCornerShapeBCornerShapeExpansionMatchesChrome` pins the mapping; the values come from `getComputedStyle` probes on Chrome 155.0.8059.84 after an earlier side-order tie-break diverged on the sibling fixture's row 17.
- Add `OpPath` (explicit value 9) in `internal/layout/corner_path.go`, with the command list on `opExtra.CornerPath`, so the hot `Op` record stays at 256 bytes. Paint lives in `internal/layout/corner_shape_paint.go` and is called from `prependChrome`: `round` keeps the radiused fill and stroke rects with zero paths; any other shape emits a solid `OpPath` for the background fill and one for the border stroke.
- Corners trace clockwise from each corner's start edge point to its end edge point, with a direction flag for the top-left and bottom-right corners. The first cut drew a self-crossing chord on those two corners, which the sibling fixture 67 visual pass caught.
- Public contract: `DisplayOpPath`, `CornerPath`, `PathCmd`, `PathVerb`, the verb constants, and `CornerPathValue` in `layout/displaylist.go`. The WASM drawing list learns the `path` kind and the `cornerPath` payload (`bindings/wasm/drawing_list.go`), so its default arm no longer reports the kind as unknown.
- Integration: projective transforms project every path command point (`internal/layout/display_geometry.go`), overflow clipping drops paths whose bounds miss the clip (`internal/layout/overflow_clip.go`), paint order places paths in the chrome layer (`internal/layout/paint_order.go`), and op ownership treats a path as box paint (`internal/layout/op_ownership.go`).
- Tests: `internal/layout/css_behavior_corner_shape_a_test.go` and `_b_test.go` drive every one of the 34 names through the drawing list; `bindings/wasm/corner_shape_test.go` covers the JSON path commands.

### Keep-together page-break ranges

- `internal/layout/pagebreaks.go` exports three more keep-together families: every table row, a heading with the content that follows it, and a single-line block whose line cannot fit the page it starts on. The single-line range covers the border box, so a moved line carries its padding, border, and background.
- Table boxes record row tops and heights after rowspan growth (`internal/layout/layout.go`, `internal/layout/layout_tables.go`), so row ranges are exact. `PageBreaks` now takes the page content height, and a zero height skips the single-line check. A forced break wins over any range it cuts.
- Tests: `pagebreaks_heading_keep_test.go`, `pagebreaks_table_rows_test.go`, `fixture31_row_split_test.go`, and updated `pagebreaks_test.go` pin fixtures 21, 31, 60, 61, 62, and 64.

### Print width fit

- `DisplayListOptions` re-lays a print document out at `width/contentWidth` when the content draws past the canvas, the smart shrink the print renderers apply. The fit runs before `FinalizeDisplayOps`, so transformed paths and text basis are stamped at the fitted scale. Screen media stays at its authored scale.
- `fixture_47_certificate_test.go` pins fixture 47, whose scaled absolute background drew 86pt past the one-page canvas before the fit.

### Canvas background geometry

- Mark the color-scheme canvas fill with `setCanvasBackground` and snap its rect to the final canvas in `applyCanvasBackgroundGeometry`, so a root margin no longer shrinks it and a viewport-raised canvas is fully covered. Consumers can tell the canvas fill from an element background with `IsCanvasBackground()`.
- A corner-shape canvas fill keeps its baked command list and emission rect. Tests: `css_behavior_canvas_fill_test.go`, plus corner path counting in `css_behavior_mask_test.go`.

### Object-fit cropping

- A fitted image whose paint rect overflows the content box (cover, or none pushed outside by object-position) is cropped to the visible sub-rect and re-encoded as PNG, so the op is exactly the painted region instead of spilling over sibling content and page boundaries. An undecodable payload keeps the fitted rect.
- The shared `objectFitPaintForImage` path replaces the duplicate filter and clip handling in block and inline image paint (`internal/layout/object_fit.go`, `internal/layout/layout_images.go`, `internal/layout/inline_image.go`). Test: `object_fit_clip_test.go`.

### Grid percentage sizing

- Intrinsic track sizing marks the containing block indefinite and resolves percentage widths as auto, so a `width:100%` image contributes its intrinsic width instead of echoing the viewport back into a track's min-content base (`internal/layout/grid_tracks.go`, `imgCBIndefinite` in `internal/layout/layout.go`).

### Catalog and docs

- `testdata/css/catalog/properties.json`: all 34 `corner*` rows move to implemented, with the wave limits kept in `limitations`. `documentation/compatibility-matrix.md` section 2 is regenerated from the catalog.
- `frontend/src/data/catalog.json` is regenerated with the 34 corner rows plus the anchor rows the committed site data was missing.
- `documentation/deferred.md` drops the corner-shape family from the deferred census (94 properties to 60). `documentation/library-api.md`, `documentation/architecture/02-library-api.md`, and `07-layout.md` document the `path`/`cornerPath` JSON and the print fit.
- `plans/v0.0.3/corner-shape/` records the shipped rows, the Chrome probes, the sibling audit, and the open follow-ups.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | A round box keeps today's fill and stroke ops with zero paths. A non-round box adds one path op per painted fill or stroke. The print fit re-runs layout only for print documents that overflow the canvas. No benchmark was run. |
| **Memory** | Path commands live on `opExtra`, so the hot `Op` record stays at 256 bytes. Table boxes keep row tops and heights. |
| **Behavior / correctness** | Corner shapes paint. `KeepTogether` covers table rows, headings, and single lines. The canvas fill op covers the full canvas. Fitted images are cropped to their content box. Overflowing print layouts are fitted. Grid percentage widths resolve as auto in intrinsic sizing. |
| **API / CLI** | New: `DisplayOpPath`, `CornerPath`, `PathCmd`, `PathVerb`, `CornerPathValue`, `IsCanvasBackground`. WASM JSON gains the `path` kind and `cornerPath`. No CLI change. |
| **Dependencies** | None added. The direct-module allowlist stays `go-text/typesetting` and `tdewolff/canvas`. |
| **Binary size / build time** | Not measured. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| New `path` operation kind and `cornerPath` JSON payload | A consumer that does not replay paths skips the kind like any unknown kind; shaped corners stay unpainted until it does. |
| `Display.KeepTogether` carries more ranges | Paginators that honored break-inside ranges treat the new ranges the same way. |
| Print layouts that overflow are re-laid at a fitted scale | Affected print documents return smaller point geometry, matching the print renderers' smart shrink. Screen media is unaffected. |
| The canvas fill op now spans the canvas and is marked | A paginating consumer can paint it on every page using `IsCanvasBackground()`; a single-canvas consumer paints it once. |

## Test plan

- [x] `make test`
- [x] `make lint` / `go vet` (lint includes `make size-check`)
- [x] `make build`
- [x] `make golden` when layout, paint, or fixture behavior changed
- [x] `make claim-scan` when documentation or user-facing claims changed
- [x] `make catalog-check`
- [x] `make matrix-check`
- [x] `git diff --check`

The full gate set ran green on 2026-10-17 at `2840481`, plus uncached focused runs of `go test ./internal/layout -run 'TestCornerShape|TestBehaviorBorderTopLeftRadius|TestBehaviorBorderStartStartRadius' -count=1` and `go test ./bindings/wasm -run 'TestDrawing' -count=1`. The PR-body commit after it touches only `plans/`. `internal/layout/layout.go` stays the single allowlisted over-limit file (2,153 lines).

### Commands

```sh
make test
make golden
make lint
make claim-scan
make catalog-check
make matrix-check
make build
```

## Screenshots / sample output

```
No page raster exists in this tree; the checks are drawing-list tests.

Chrome 155.0.8059.84 getComputedStyle probe, pinned by
TestCornerShapeBCornerShapeExpansionMatchesChrome:
  corner-shape: scoop bevel notch squircle
    -> top-left scoop, top-right bevel, bottom-right notch, bottom-left squircle

Gate output, 2026-10-17:
  css-catalog-map: check ok (785 properties, 500 handler names, implemented=488, partial=3, unsupported=286, intentionally ignored=8)
  matrix-sync: ok (section 2 matches python3 scripts/css-catalog-map.py --matrix)
  claim-scan: clean
  golangci-lint v1.64.8 + size-check: clean
  go test ./layout/ -run TestDisplay: ok
  CGO_ENABLED=0 go build ./...: ok
  make test: full suite green
```

## Related issues

- None. `gh issue list --state all` returned no issues; this is plan-driven work tracked by `plans/v0.0.3/corner-shape/corner-shape-checklist.md`.

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied: `enhancement`, `documentation`
- [x] Related issues checked; none exist
- [x] Filled body committed under `plans/v0.0.3/corner-shape/PR/pr-corner-shapes.md`

## Follow-ups (out of scope)

- Dashed and dotted borders through a shaped path (segment clipping); today a non-round corner with a dashed or dotted border paints a solid path.
- Per-side colors and widths on a shaped border; today differing sides keep the rounded stroke-rect path.
- `background-clip` and overflow clip following the scooped outer path; today they keep their existing boxes, and a clipped path drops only when its bounds miss entirely.
- `box-shadow` scaled from the corner path (the CSS Borders 4 rule); today it keeps today's radii.
- A writing-mode table shared with the logical border radii; the checklist keeps one open `[~]` row for vertical writing modes.
- `border-image` keeps using the image painter and ignores corner shape, by design in this wave.

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
- [ ] Diff-stat-by-extension table pasted at the bottom

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 42 | 5839 | 204 |
| `.json` | 2 | 1014 | 210 |
| `.md` | 8 | 420 | 124 |
| `.txt` | 1 | 1 | 1 |
| **Total** | **53** | **7274** | **539** |
