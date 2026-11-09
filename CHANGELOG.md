# Changelog

All notable changes to blinkless are recorded here. The project follows
semantic versioning. There is no `VERSION` file, and GitHub CI does not
compare tags to source version stamps.

The imported 0.1.x through 0.2.x changelog entries predate this repository
and are not listed here.

## Unreleased

## 0.1.1 (2026-11-10)

Work after tag `v0.1.0` (`2e2abfc`), tracked in
[`plans/v0.1.1/`](plans/v0.1.1/README.md). Release note:
[`plans/v0.1.1/release-note/v0.1.1.md`](plans/v0.1.1/release-note/v0.1.1.md).
39 commits across six merged pull requests (#31 through #36). No new
output format and no dependency change.

### Added

- **Batch subtree relayout.** `layout.RelayoutElements` re-places several
  element ids over one retained placement and builds one display. A
  size-changing patch widens to the nearest size-stable ancestor, and
  table cells, rows, and inline content retarget to the owning table. The
  batch is all-or-nothing (`layout/relayout_elements.go`).
- **Display release.** `layout.Display.Release` returns retained box,
  style, and store chunks to their pools. It is nil-safe and idempotent
  (`layout/displaylist.go:227`).
- **Hover scopes and dirty accessors.**
  `css.Document.HoverChangedScopes` returns the nearest id-bearing
  subtrees whose hover selector match changed. `html.Document.DirtyIDs`,
  `DirtyStructural`, and `ClearDirty` record edits from `SetAttribute` and
  `SetText` (`css/hover_scopes.go`, `html/attribute.go`).
- **Backdrop filter metadata.** `backdrop-filter` parses and rides the
  element's display group as `DisplayGroup.BackdropFilter`. The catalog
  moves the property from Unsupported to Partial, with 785 rows: 555
  implemented, 4 partial, 218 unsupported, 8 intentionally ignored, dated
  2026-11-02.

### Performance

- **Render cost.** Four waves pool box and style chunks, roll back
  speculative measurement chunks, cache gradient textures, request a
  one-pixel texture for pure 1D linear gradients, pool flex child slices
  and PNG buffers, bypass chrome-less nodes, skip already-sorted paint
  orders, build display boxes and parents in one placed walk, and derive
  owners from the parent slice.
- **Patch path.** No-op and inert patches skip the engine, text-free
  patches skip font loads, paint-only batches reuse the previous boxes and
  breaks, and simple flex rows and stacks avoid a measure build. Text
  measurement, selector matching, grid placement, and HTML edits gain
  bounded caches and an id index.
- **Cache lifetime.** Grid memos are pass-scoped, flex builds borrow the
  cascade style, selector and fingerprint caches use weak keys, and the
  shared 16-cap style pool keeps up to 32 chunks (about 2.9 MiB, under the
  existing 4 MiB test bound). Released documents and layout results are
  shown collectible by focused tests.
- PR #31 records a downstream OwnFrame example measurement of 3,745 ms to
  537 ms cold latency and 2,134 MB to 160 MB heap. Those are example-app
  figures outside this tree, not a committed Blinkless benchmark. The
  invoice benchmark figures stay TBD.

### Fixed

- Subtree relayout preserves inherited positioning and stacking state and
  keeps viewport-fixed boxes patchable (`24a7ab7`).
- Hidden atomic inline boxes stay in the box tree, so row visibility
  toggles patch instead of failing (`08a18a7`).
- Each weak cache miss rebuilds from live state, so a collected entry
  cannot serve stale data (`780905b`).
- The backdrop-filter PR's lint and size failures were repaired in
  `2c7a05d`.
- `.golangci.yml` keeps correctness linters on tests while dropping
  production cuddling, identifier-length, and complexity budgets from test
  files. `scripts/file-size-allowlist.txt` records eight over-limit files.

### Documentation and data

- `documentation/architecture/07-layout.md` records cache lifetimes during
  interactive layout.
- `documentation/library-api.md` records the batch relayout API and the
  scale benchmark.
- `documentation/fidelity.md` and
  `documentation/compatibility-matrix.md` match the 2026-11-02 catalog.
- The catalog JSON is ASCII-escaped again after regeneration.

### Designed but not wired

- The retained-style resolve (`internal/layout/style_retain.go`,
  `resolveStylesRetainedWithContext`), the block intrinsic memo
  (`internal/layout/memo_block.go`), some per-box pools in
  `internal/layout/box_alloc.go`, and the batch damage windows
  (`SubtreeBatchResult.Windows`) are present and tested but have no
  production caller. Do not treat them as shipped behavior.
- The `html` id and dirty registries still key on `*Document`, so the
  finalizer cannot run. The release does not claim dropped documents are
  always collectible.

### Known issue

- `outline-style: solid` with no explicit width can paint no outline. The
  chrome early exit tests raw `OutlineWidth`
  (`internal/layout/layout_chrome.go:497`), but CSS gives a visible style
  the `medium` width (`internal/layout/outline.go:38-45`). The fix and its
  test are tracked in
  [`plans/v0.1.1/context/follow-up-backlog.md`](plans/v0.1.1/context/follow-up-backlog.md).

## 0.1.0 (2026-10-30)

Stabilization release for OwnFrame. It hardens the drawing-list surface
OwnFrame consumes (restamp pass, timelines, SVG paint records, override
relayout, race-free pipelines) and cuts first-launch layout cost. The
window holds 26 commits across six merged pull requests (#25 through #30),
tracked in [`plans/v0.1.0/`](plans/v0.1.0/README.md). Tag `v0.1.0` is
`2e2abfc`. This section was added in the 0.1.1 release PR; the 0.1.0 tag
was cut before the CHANGELOG section landed.

## 0.0.3 (2026-10-23)

Work after tag `v0.0.2`, tracked in [`plans/v0.0.3/`](plans/v0.0.3/README.md).
Release note:
[`plans/v0.0.3/release-note/v0.0.3.md`](plans/v0.0.3/release-note/v0.0.3.md).

### Added

- **Element patches.** `layout.Options.RetainState` keeps the resolved styles
  and box tree a patch needs, and `layout.RelayoutElement` re-places one block
  subtree inside the retained page and returns the whole patched display.
  `html.Document.SetAttribute` and `html.Document.SetText` edit the tree in
  place (`6b29cc9`).
- **Selector index and one-pass grids.** The cascade indexes selectors by
  their rightmost compound, single-column and multi-column auto-row grids
  build items in one pass, and inline splitting grows geometrically
  (`81910e0`, `3fa0a49`, `e0c3c9c`).
- **Corner shapes.** `border-radius` corners accept the `corner-shape`
  family and paint through a path operation (`4896ac9`, with fixes
  `bc917a2` and `20606e1`).
- **Anchor positioning.** Absolute and fixed boxes can place against a named
  anchor, with `position-area`, `position-try`, fallbacks, and
  `position-visibility` (`dbc98fa`, `bb06e35`).
- **3D aliases, gradient mask sources, and row rules.** The `-webkit-` 3D
  aliases, gradient and `image-set()` mask sources, and `row-rule*` /
  `rule*` gap lines are implemented (`c13a036`, with fixes `9882877`,
  `7cc73fc`, and `2d1a7c8`). Junction-aware gap rules and mask parity
  followed in `91dc0a0`.
- **Mask borders.** The `mask-border` descriptor, shorthand, and WebKit
  aliases compose a nine-slice frame raster on the element group, and the
  WASM drawing list carries the frame data (`ac4b438`, `06de259`,
  `361e5e1`).
- **Interior shapes.** `shape-inside` and `shape-padding` now wrap content
  inside a shape (`b66ed39`).
- **Named flows and regions.** Named flows, masonry tolerance, exclusions,
  and wrap breaks are implemented on the single drawing list (`e6abe07`).
- **Animation and transition metadata.** Accepted animation and transition
  values, keyframes, and aligned timing records are exposed on the public
  element boxes and the display list. Blinkless does not advance time or
  interpolate values (`b26de13`).
- **Frontend site.** The `frontend/` source and the built `docs/` site were
  re-added and expanded, with generated catalog and specimen data
  (`04c167a`, `eba3139`).
- **CSS catalog expansion.** The catalog moved from 389 implemented rows at
  `v0.0.2` to 555 implemented, 3 partial, 219 unsupported, and 8
  intentionally ignored. The catalog is dated 2026-10-20 and the upstream
  pin is unchanged.

### Fixed

- Frontend catalog export keeps `implementation_scope` and the committed
  ASCII-only JSON style, so regenerating the catalog no longer drops the 43
  Blinkless+OwnFrame rows (`1c68b28`).
- Root canvas background painting (`320dc96`).
- Table rows, headings, and one-line blocks stay whole (`fbde5d6`).
- Print width fit on the canvas (`7a948be`).
- Grid percentage widths (`16d26a3`).
- `object-fit` clipping (`b3d9123`).
- Remainder-short avoid groups and tolerance defaults (`26f54ec`).

### Removed

- **C shared-library binding.** `bindings/c`, `make c-shared`,
  `BINDINGS_VERSION`, and the generated `libblinkless` artifacts are
  removed. The engine is consumed through the Go packages and the WASM
  binding. This removal shipped in `v0.0.3`.

### Documentation and data

- Catalog counts in `documentation/fidelity.md` and
  `documentation/compatibility-matrix.md` now match
  `testdata/css/catalog/properties.json`: 785 rows, 555 implemented,
  3 partial, 219 unsupported, 8 intentionally ignored.
- `documentation/deferred.md` now points at the live `plans/` index and the
  v0.0.3 ledgers instead of the removed 0.2.x plans.
- The frontend catalog and specimen data were regenerated, and the built
  site under `docs/` was rebuilt.

## 0.0.2 (2026-10-15)

Security release after `v0.0.1`, tag `23cc975`. The output is still
`layout.DisplayList`. This release adds no CSS or layout features, and the
catalog is unchanged from 0.0.1.

### Security

- Stylesheet source over 32 MiB is rejected. HTML element depth and the
  active formatting list stop at 1024. `html.Parse` still does not cap its
  own byte length. `ParseWithLimit` does, with a 64 MiB default.
- Dashed edges, text-decoration steps, and border-image tiles stop at 1024.
  Decoded images stop at 16,777,216 pixels (16 Mi pixels). SVG `text`,
  `tspan`, and `textPath` are removed before raster.
- Restricted local-file reads use `os.Root`; the unrestricted path opens the
  file directly. FIFOs are rejected. Restricted mode refuses private
  addresses unless explicitly allowlisted, and refuses a proxied hostname.
- The C binding caps the allow list and the last-error buffer, rejects an
  HTML length at or above 2^63, and recovers panics. WASM applies the JSON
  and HTML byte caps.
- Other caps cover `:has` inspection, grid repeat, font metrics and
  composite glyphs, nested `var()`, fetch and header budgets, SVG tag
  rewriting, multicast and broadcast addresses, redirect `Location`
  redaction, CPAL offsets, and regular-file checks.
- At this tag the C binding is a rendering stub: it validates arguments and
  returns the image-encoding removal error. It has no drawing-list export.
- Records: `plans/v0.0.1/security/` ledgers 01 through 09. Release note:
  [`plans/v0.0.2/release-note/v0.0.2.md`](plans/v0.0.2/release-note/v0.0.2.md).

### Removed

- **Python package, PyPI publish, and the tag-vs-source version gate.**
  `bindings/python`, `.github/workflows/publish-pypi.yml`,
  `scripts/check_versions.sh`, and `make check-versions` are removed.
  `bindings/c` and `bindings/wasm` stay at this tag. GitHub CI runs
  `make test` and `make lint` only, and there is no `VERSION` file.

### Version strings

At `v0.0.2`, the C header still reported `BLINKLESS_VERSION "0.2.6"` and the
Makefile binding stamps were `0.0.1`. The C binding was removed after this
tag; see 0.0.3 above.

## 0.0.1 (2026-10-15)

Renderer cut, tag `108c2c9`. The release note covers 54 engine commits from
`23061d8` through `c085b34`; six release-doc commits follow through the tag.
HTML and CSS go in, and a `layout.DisplayList` drawing list comes out. This
tree does not write a PDF, does not encode a page as PNG or JPEG, and does
not ship a `blinkless` command.

### Added

- **Drawing-list output and public API.** `layout.DisplayList` returns a
  retained list of vector operations instead of a flattened bitmap. The
  public surface includes `Display`, `DisplayOp`, `DisplayGroup`,
  `DisplayKind`, `DisplayOrder`, `DisplayFakeBold`,
  `DisplayTransformText`, the `DisplayOp*` constants, the nil-safe accessors
  on an operation, and `layout.SnapDisplayToDevicePixels` for optional
  screen stroke snapping. The public packages are `html`, `css`, `layout`,
  and `markup`.
- **HTML parser work.** The tokenizer and tree construction gained input
  preprocessing, NUL replacement, raw-text and script-data states, doctypes,
  foreign namespaces, foster parenting, the adoption agency, templates,
  `select`, fragment parsing, ruby recovery, and frameset recovery
  (`a9381bf`, `c6ff6dc`, `9bd5d38`, `6f1a31a`).
- **CSS catalog and layout coverage.** The catalog reached 785 rows, with
  389 implemented, 19 partial, 369 unsupported, and 8 intentionally
  ignored. Layout work covered flex gaps, `fr` grid tracks, `line-height:
  normal`, margin collapse, `text-wrap-style`, border-image tiling,
  `text-justify`, `white-space-trim`, inline SVG stroke, ruby, `clip-rule`,
  dark `color-scheme`, variable fonts, 2D-projected 3D transforms, multicol
  breaks, and wide-gamut color.
- **First-wave masks.** Ten standard mask properties and eight `-webkit-`
  aliases are carried as group metadata on the drawing list. The host
  composites the pixels; there is no mask operation kind and no in-engine
  pixel replay (`30971e1`, merged at `c085b34`).
- **Drawing-list JSON.** The WASM binding returns schema
  `blinkless.drawinglist/1` with operations, paint order, boxes, groups,
  fonts, and images.

### Fixed

- A border shorthand with no width defaults to `medium` (3 CSS pixels,
  2.25 points). Explicit `0` stays `0`; `thin` is 1 CSS pixel and `thick`
  is 5 (`6892dc5`).
- The swapped `PointsPerPixel` and `PixelPerPoint` field names were
  corrected (`a9381bf`).

### Removed

- **PDF writer, page rasterizer, and CLI.** `internal/pdf`,
  `internal/pdfprofile`, the PDF paint path, `internal/imageout`,
  `screen.Render`, `ImageDocument`, `layout.Lay`, `cmd/blinkless`, and
  `internal/cli` are gone. One image operation keeps its encoded bytes and
  re-encodes that one payload as a PNG when orientation or a clip requires
  it.
- **Old frontend and docs.** `frontend/` and the built `docs/` site were
  removed at 0.0.1. The site returned after 0.0.2; see 0.0.3 above.
- **Old plans and `VERSION`.** The imported plan trees and the `VERSION`
  file were removed.
- **PDF exports and Python helpers.** The C header dropped
  `blinkless_html_to_pdf` and `GwkPdfOptions`. `blinkless_html_to_image`
  reports that image encoding was removed. The Python PDF helpers raise
  `RuntimeError` with the same removal message.
