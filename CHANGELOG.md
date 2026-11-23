# Changelog

All notable changes to blinkless are recorded here. The project follows
semantic versioning. There is no `VERSION` file, and GitHub CI does not
compare tags to source version stamps.

The imported 0.1.x through 0.2.x changelog entries predate this repository
and are not listed here.

## Unreleased

## 0.1.6 (2026-11-23)

Work after tag `v0.1.5` (`d486e95`), merged in PRs #45 to #49. The full account is in `plans/v0.1.6/release/v0.1.6.md`.

### Added
- **Automatic system CJK font fallback.** Han, kana, Hangul, and fullwidth characters that no CSS family, bundled face, or opt-in registry face covers now resolve to an installed operating-system face instead of a missing-glyph box. It is on by default, reads a short per-OS candidate list lazily (Yu Gothic, Meiryo, and MS Gothic on Windows, Hiragino and Arial Unicode on macOS, common TrueType CJK families on Linux and Android, and the Windows font folder under WSL), and never runs for Latin text. Turn it off with `layout.SetSystemFontFallback(false)` or `BLINKLESS_SYSTEM_FONTS=off`.
- **TrueType collection (`.ttc`) support** for system fallback faces. `fonts.SplitTTC` rebuilds each member as a standalone SFNT, so shapers that load one face per byte slice keep working.

### Fixed
- **`html.Document` memory leak.** Moved the per-document element index and dirty set onto unexported fields of `html.Document` and removed the global `elementRegistry`/`dirtyRegistry` maps and dead finalizer so discarded documents are garbage-collected.
- **`CloneResult` style map and chunk ownership isolation.** `CloneResult` now clones `Result.styles` and leaves `styleChunks`/`storeChunks` unowned until a batch commits via `TransferChunksTo`, so aborted `RelayoutElements` batches leave `prev` intact and releasing a clone alongside its source cannot double-free pooled chunks.
- **Blend-group splice leaks on layout overrides and group removal.** Wired `relayoutGroupWindow` into `RelayoutSubtreeWithOverrides` and `spliceOverridePatch`, and updated `widenGroupWindow` so removing a blend group (such as hover-leave dropping `backdrop-filter` or `isolation: isolate`) strips its stale begin/end markers cleanly.
- **`cloneOpExtraInto` display path deep copy.** Deep-copied `Path.Points` during slab op cloning so mutating a cloned op's path points cannot mutate the source.
- **Stacking-context classification on ancestor paint replay.** `relayoutEnterAncestorPaint` and `pushZ` now use `initiatesBlendGroup(style)`, fixing empty `MixBlendMode` falsely forcing `zIndexSet` and including `backdrop-filter` stacking contexts.
- **`clip-path: ellipse(...)` mixed radii and `outline-style: solid`.** `resolveEllipseRadii` now resolves `rx` and `ry` independently per axis, and `hasPotentialChrome`/`prependChrome` guard outlines with `outlinePaints(sty)` so `outline-style: solid` paints with default `medium` width.
- **Bounded LRU image effect caches and filter gating.** `image_effect_cache.go` now trims cached PNG slices with `bytes.Clone` (exact `cap == len`), bounds total bytes (32 MiB) with LRU eviction instead of full-map clears, and aligns `applyImageFilterToImageWithInterp` with implemented filter kinds.
- **Allocation-free `paintOnlyBGDelta`.** Replaced struct copies and `reflect.DeepEqual` in `paintOnlyBGDelta` with generated `styleInternEqualExceptBG`.

- **Go review hardening (PR #48, PR #49).** Fixed the `Node.EnsureAttrs` race, the `prepare.SheetCache` map race, and the shared `transitionsFrom` slice, so callers no longer need an engine-wide lock. Fixed combinator backtracking, UTF-8 offset bugs, case-sensitive custom-property resolution (depth cap 64), sibling cache invalidation, and nested `@media` ordering with a 64-entry cross-product cap. `html.Parse` now caps source at 64 MiB, and `html.ParseWithLimit` takes an explicit limit. `css.Apply` clones `opts.Extra`.
- **`ResolvedStyle` shrinks from 5,872 to 512 bytes (PR #46)** through lazily shared groups. Benchmarks were mixed: invoice layout 3.2% slower, medium relayout 6.7% faster.

### Changed
- **Pointer `*ResolvedStyle` in chrome helpers.** Updated internal chrome paint helpers called from `layout_chrome.go` to take `*ResolvedStyle` instead of copying the struct by value.
- **Dead code and CI hygiene.** Removed unwired `resolveStylesRetainedWithContext` and the dead `bench-lib` Makefile target, pinned GitHub Actions SHAs with least-privilege permissions, and added race/claim/catalog/matrix gates to CI.

## 0.1.5 (2026-11-22)

Work after tag `v0.1.4` (`a208693`), merged in PR #44.

### Added
- **Paint restyle for background-color changes.** `layout.RelayoutElements`
  no longer lays out an element whose only change is the background color of
  boxes it already placed. The batch resolves the new subtree styles, proves
  each changed element differs only in `BGColor`, that it is a plain
  block-level box with one solid unclipped fill the kept window holds exactly
  once, and that the DOM and text are unchanged, then rewrites that fill's
  color in its private copy. The code is `internal/layout/paint_restyle.go`,
  and `layout/relayout_paint_restyle_test.go` compares every case with a full
  layout and with the forced slow path.
- **Cloned operation payloads come from one slab.** `cloneOps` carves op
  payloads from a single slice per clone instead of one allocation per op
  (`internal/layout/clone_ops_slab_test.go`).

### Changed
- **Layout animations reuse base styles between frames.** Layout overrides
  such as animated `margin-left` reuse the containing block's resolved
  styles while the inputs and retained styles still match.
- **Image filter and clip-path mask raster caching.** `applyImageFilterToImageWithInterp`
  and `maskImageWithClipPath` cache encoded PNG outputs keyed by SHA-256 of
  the source bytes, relative box geometry, and canonical filter/shape
  signatures, plus a direct-`Pix` fast path for `clip-path: inset(...)`.
- **Zero-copy ASCII word breaking and pointer `ResolvedStyle`.** `(*engine).breakToken`
  takes `*ResolvedStyle` instead of copying the struct by value, and
  `splitTextToWidthASCII` slices ASCII tokens directly without `[]rune`
  allocations.

### Fixed
- **Explicit font kerning changes text width.** `font-kerning: normal` and
  an explicit OpenType `kern` setting tighten or widen layout widths to match
  the kerning used by a shaping host. `font-kerning: auto` keeps its
  existing layout widths.
- **Grouped element patches replace their group markers.** Patching an
  element with a backdrop filter, blend mode, or isolation replaces its group
  markers on hover in single-element and batch relayout.
- **Element patch checks the root's own margin change.** `relayoutBuild`
  compares the element's margins against `prev.styles` and reports a change
  as `errSubtreeResized` so the escalation re-lays out the holder
  (`layout/relayout_root_margin_test.go`).

## 0.1.4 (2026-11-21)

Work after tag `v0.1.3` (`4c8b22b`), merged in PR #43.

### Changed
- **Sub-millisecond hover relayout.** Added `SameBoxes` incremental box
  patching, persisted immutable flex style overrides across subtree relayouts,
  pooled clone box arenas and `Display.Boxes` slices, and switched hot `Op`
  helpers to pointer receivers to cut hover relayout allocations and latency.

## 0.1.3 (2026-11-21)

Work after tag `v0.1.2` (`7abfce3`), merged in PR #41 and PR #42.

### Changed
- **Engine allocation and throughput pass.** Eliminated `O(N^2)` deferred
  chrome scans, moved `ResolvedStyle` `Declared` and `Triggers` behind lazy
  pointers, added single-pass flex column/row fast paths, cached
  `measureStyleID`, pooled PNG encoders and per-layout text/rune width caches,
  slab-allocated HTML nodes and small attribute maps, and bounded flex
  override caches.
- **Website catalog refresh.** Rebuilt the site catalog from the updated CSS
  property inventory.

## 0.1.2 (2026-11-19)

Work after tag `v0.1.1` (`bddc6e5`), merged in PR #38, PR #39, and PR #40.
Tracked in [`plans/v0.1.2/`](plans/v0.1.2/README.md).

### Added
- **Viewport sticky placement and height media queries.** Added viewport
  sticky placement and height media query support (PR #39).
- **Font feature tests.** `internal/layout/font_feature_settings_test.go` pins how `font-feature-settings`, `font-kerning`, `font-variant-ligatures` and the `font-variant` shorthand reach `Op.FontFeatures`.
- **61 additional CSS properties exported on `Box.Declared`.** Added storage, export, and layout/host behavior for caret, `nav-*`, `spatial-navigation-*`, reading order, `border-shape`, `border-boundary`, `border-*-clip`, `border-limit`, `column-rule-visibility-items`, `text-fit`, `float-defer`, `shape-image-threshold`, `offset-*` motion path, `bookmark-*`, `block-ellipsis`, `margin-break`, `background-attachment: fixed`, `-webkit-text-stroke`, and SVG geometry/text/paint-order properties.
- **Timeline, trigger, interest, and interactivity export.** Stored and exported 27 timeline and trigger styles on `layout.Box` (`scroll-timeline*`, `view-timeline*`, `timeline-scope`, `pointer-timeline*`, `timeline-trigger*`, `event-trigger*`, `interest-delay*`, `interactivity`) with comma-list and scroller resolution (`layout/timeline_list_test.go`).
- **Compatibility matrix sync.** Synced `documentation/compatibility-matrix.md` and the site catalog with the updated property set (PR #40).

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
