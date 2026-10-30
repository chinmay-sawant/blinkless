# blinkless 0.1.0

0.1.0 is a stabilization release. It hardens the 0.0.3 drawing-list
engine for its downstream consumer: OwnFrame reads the drawing list from
this tree and samples the animation, transition, and timeline records
with host-supplied time and timeline inputs. HTML and CSS still go in. A
drawing list still comes out. This tree does not write a PDF, does not
encode a page as PNG or JPEG, and does not ship a `blinkless` command.

`v0.0.3` is published at `8767e82`. This note is the work after that tag,
through `b28b7d6`. The commit list is [commits.md](commits.md): 26 commits
across 6 merged pull requests (#25 through #30). Nine of the commits are
docs-only PR bodies. The window holds no new output format and no catalog
growth; it is perf, concurrency, and consumer-surface work on top of 0.0.3.

## What changed

This window hardens the surface OwnFrame consumes. Everything below lands
as drawing-list content and public API, not as a new output format. The
public shape is still `layout.DisplayList`, and the drawing-list JSON
schema is still `blinkless.drawinglist/1`.

- Animated transforms, timelines, and SVG paint (`72e595d`,
  `layout/reproject.go`, `layout/timeline.go`, `layout/svg_paint.go`).
  `TransformOverride` and `RestampTransforms` run an additive restamp pass
  for 3D and perspective geometry. `TimelineDef` carries named scroll and
  view timelines. `SVGPaintOverride` and `RestyleInlineSVG` restyle inline
  SVG paint without touching the rasterizer. The display list is unchanged
  when none of this is used. Blinkless records the values; it does not
  advance a clock or interpolate between frames.
- Relayout with overrides (`8d4693e`,
  `layout/relayout_element.go`). `RelayoutElementWithOverrides` takes an
  `ElementPatch` with post-cascade property overrides, re-places the
  containing block, and returns the whole patched display. This builds on
  the plain `RelayoutElement` that shipped in 0.0.3.
- Display metadata reuse across restamps (`2b608a5`). Repeated stamps of
  the same page reuse resolved metadata instead of recomputing it.
- Race-free concurrent pipelines (`53ba184`, `internal/pubstate/state.go`).
  CSS and HTML readers register once at package init, so concurrent
  `Parse`, `Apply`, and layout reads are safe. `resetWindowTransforms`
  leaves the shared empty extra alone. Covered by
  `TestConcurrentApplyAndReads` under `-race`. Same output, now safe to
  run side by side.
- Copyless transform reads (`31adc22`,
  `internal/layout/op_read.go:52`). `(*Op).XformValue` returns the stored
  matrix without copying the operation. No behavior change.
- Faster layout across four perf commits (`a599ae5`, `c1c0ad5`,
  `2c22297`, `f6edf62`). First-launch work on icon and grid heavy pages
  drops through memoization: one raster per repeated SVG icon
  (`internal/svg/raster.go`), grid and flex measure memos, a per-pass style
  match cache, a per-font advance cache (`internal/fonts/advance_cache.go`),
  and an element-local selector check (`SelectorIsElementLocal` in
  `internal/css/match.go`). News and orders pages skip repeated measure
  passes. Pseudo content and counters memoize per run. Transformed paint
  intersects ancestor overflow clips. Generic alloc cuts reuse backing
  arrays for border, outline, and chrome ops. Flex and abspos geometry
  fixes ride along: max-content flex items may overflow the lane, and
  stretched abspos boxes solve against the containing height.
- Flex-stretch relayout patch (`992d281`,
  `internal/layout/relayout_subtree.go`). Flex-assigned block sizes route
  through style overrides, so hover patches on flex-distributed rows no
  longer fall back to a full page layout.
- Lint and file-size refactor (`1a7cbb9`). Hot functions split into
  focused files with no behavior change. The allowlist now holds a single
  entry: `internal/layout/layout.go` at 2133 lines.
- Catalog and matrix sync (`75bd6a8`, PR #30). The `background-image`,
  `display`, and `overflow` value lists now match between
  `testdata/css/catalog/properties.json` and
  `documentation/compatibility-matrix.md`. The matrix prose records the
  implemented mask and mask-border statuses. Catalog counts are unchanged:
  785 rows, 555 implemented, 3 partial, 219 unsupported, 8 intentionally
  ignored, dated 2026-10-20. The upstream pin is unchanged.

GitHub Actions on a branch and on a `v*` tag runs `make test` and `make lint` only. The tag is not compared to a version file.

## Supported in this tree

What 0.0.1 through 0.0.3 built, and what 0.1.0 keeps working. Detail lives
in the earlier notes; this is the short map.

- Parse and cascade. `html.Parse` handles doctypes, namespaces, foster
  parenting, the adoption agency, templates, select, fragments, and ruby
  recovery. `css.Apply` cascades author styles onto the tree. The catalog
  holds 785 rows: 555 implemented, 3 partial, 219 unsupported, 8
  intentionally ignored, dated 2026-10-20. Implemented means a handler, a
  consumer outside the style layer, and a resolving behavior test, not
  Chrome parity.
- Layout. Block, flex, and grid (one-pass single-column and multi-column
  auto-row grids), multicol breaks with orphans and widows, floats, abspos
  and fixed placement against named anchors with position-area and try
  fallbacks, corner shapes through path ops, interior shape wrapping,
  named flows with masonry tolerance and exclusions, row and column gap
  rules with junction geometry, and object-fit clipping. 3D transforms
  flatten to 2D at stamp time. Wide-gamut colors ride outside 0..1 until
  dynamic-range-limit folds them.
- Masks. Ten standard mask properties plus eight `-webkit-` aliases attach
  layer data to element groups. Gradient and `image-set()` sources resolve
  to bytes. Thirteen mask-border names compose a nine-slice frame raster
  on the group. The host composites; there is no in-engine pixel replay.
  `mask-type` and `-webkit-mask-composite` stay Partial.
- Motion records. Animation and transition values, keyframes, per-box
  ownership (`ParentIndex`, `OperationIDs`, `Display.OpOwners`), and
  declared scroll and view timelines ride the display list. OwnFrame
  samples them with host-supplied time and inputs; 43 motion cases carry
  `blinkless+ownframe` runtime tests. This engine has no clock.
- Patches and restamps. `RelayoutElement` re-places one retained subtree;
  0.1.0 adds property overrides and a flex-stretch fix so hover patches
  stay scoped. `RestampTransforms` re-projects transforms without a fresh
  layout. `RestyleInlineSVG` restyles inline SVG paint through a bounded
  cache. Concurrent parse, apply, layout, and restamp are race-free
  under `-race`.
- Output. `layout.DisplayList` returns ops, order, boxes, groups, fonts,
  and images. The WASM binding serializes schema
  `blinkless.drawinglist/1`. Screen callers can snap strokes with
  `SnapDisplayToDevicePixels`. The host draws; nothing here writes a page
  image.
- Budgets. 0.0.2 caps hostile input: 32 MiB stylesheets, 1024 element depth
  and formatting entries, 1024 dash and tile steps, 16M decoded image
  pixels, restricted local reads, and binding byte caps. Public
  `html.Parse` still takes no byte cap; `ParseWithLimit` defaults to
  64 MiB.
- Docs and site. The compatibility matrix mirrors the catalog. The
  frontend source and the built docs site ship generated catalog and
  specimen data.

## Removed

Nothing in this window. The removals that define this tree (PDF writer
and profiles, page rasterizer, `blinkless` command, Python package, C
binding) landed in 0.0.1 through 0.0.3 and stay gone here.

## Version strings

These stamps are not one number.

| Stamp | Value | Where |
|-------|-------|-------|
| This release | 0.1.0 | GitHub tag `v0.1.0` |
| Previous release | 0.0.3 | tag `v0.0.3` at `8767e82` |
| Catalog date | 2026-10-20 | `testdata/css/catalog/properties.json` |
| WASM build stamp | 0.0.2 | `Makefile` `WASM_VERSION` |
| Downstream | OwnFrame | sibling checkout reads `layout.DisplayList` |

The WASM stamp was not moved in this release. There is no `VERSION` file.

## What still does not ship

PDF files, PDF profiles, the `blinkless` command, veraPDF, and a golden PDF corpus. One image operation is re-encoded as PNG when orientation or a clip requires it. That is not a picture of the page.

Animation, transition, and timeline records carry no clock. Masks and mask borders carry values for the host to composite; there is no in-engine pixel replay. Scroll, speech, and SVG presentation rows stay unsupported except where their ledgers say otherwise. No document script runs, and that is not a proof that every bug is gone.
