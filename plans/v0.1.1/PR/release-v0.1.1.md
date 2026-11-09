# blinkless 0.1.1

0.1.1 is a performance and patch-path release on the 0.1 line. It follows
0.1.0. HTML and CSS go in. `layout.DisplayList` comes out. This tree still
has no PDF writer, no page rasterizer, and no `blinkless` command. The
drawing-list JSON schema stays `blinkless.drawinglist/1`. The Go display
group gains a `BackdropFilter` string.

`v0.1.0` is the annotated tag object `5dd4571`, peeled to commit
`2e2abfc`. This note covers 39 commits on `master` after that tag, through
`3e3a8b7`, the merge of PR #36. The commit list is
[commits.md](commits.md). The range holds six merged pull requests (#31
through #36): 18 perf commits, 5 fix commits, 2 layout commits, 1 feature
commit, 2 test commits, 5 docs commits, and 6 merge commits. No module
dependency changed. The catalog moves one row: `backdrop-filter` goes from
Unsupported to Partial, and the catalog date moves to 2026-11-02.

## What changed

### Render cost

PR #31 is four waves over the layout hot path, followed by more allocation
work in PR #35.

- Box and style storage moved to pooled chunks.
  `internal/layout/layout_chunks.go` allocates boxes in 64-box chunks,
  returns them through `Result.Release`, and keeps spare chunks for the
  next pass. Resolved style copies use a 16-entry arena, and the style
  store uses separate 8, 16, 32, and 64-entry chunks. Speculative
  measurement passes mark and reset the chunks, so a `noEmit` flex or
  multicol measurement does not keep the boxes and styles it built.
- Gradient textures are cached by source string, size, color, and
  interpolation. A pure vertical or horizontal linear gradient requests a
  one-pixel texture that the paint op stretches to the element size; the
  bytes are still PNG. PNG encoding uses `BestSpeed` and a buffer pool.
  `internal/layout/gradient.go`.
- Flex child slices are pooled, and anonymous flex items get their own
  resolved style without copying the container's box properties.
  `internal/layout/flex_children.go`.
- The chrome path exits early when an element has no background, border,
  outline, or shadow to paint (`internal/layout/layout_chrome.go`). The
  outline part of that gate has a known gap; see Known issues.
- Paint order uses `slices.SortStableFunc` and skips the sort when the
  index list is already ordered (`internal/layout/paint_order.go`).
- Display construction builds boxes and parent indices in one placed
  walk, then derives owners from the parent slice. The old path walked the
  placed boxes twice (`layout/displaylist.go`, `internal/layout/placed.go`,
  `internal/layout/op_owners.go`).
- PR #31's body records a downstream OwnFrame example measurement: cold
  latency 3,745 ms to 537 ms and heap 2,134 MB to 160 MB. Those are
  example-app figures measured outside this tree, not a committed
  Blinkless benchmark. This note does not promote them as an engine
  benchmark.
- Later commits in PR #35 add simple flex row and stack heights without a
  measure build, descriptor-keyed text measurement caches, a
  single-text-run inline path, a grid occupancy bitset, selector memos, an
  HTML id index, and a reused tokenizer attribute scratch. The parse
  allocation test records 9,037 to 7,035 to 3,531 allocations per parse
  with a 4,500 ceiling. Commit `e6f18ca` records about 57 percent off the
  invoice cold build and 53 percent off typing on its harness. No engine
  benchmark output is committed. `CHANGELOG.md` records the downstream
  numbers as downstream-only and keeps the invoice figures TBD.

### Patch engine

PR #34 keeps inherited positioning and stacking state when a subtree is
rebuilt, keeps viewport-fixed boxes patchable, and adds
`css.Document.HoverChangedScopes` so a host can find the subtrees whose
hover selector match changed. `internal/layout/relayout_subtree.go`,
`css/hover_scopes.go`.

PR #35 adds the batch path.

- `layout.RelayoutElements` threads several element ids over one retained
  placement and builds one display. The ids run sequentially because each
  splice feeds the next. A missing or refused id aborts the whole batch
  and leaves `prev` untouched. `layout/relayout_elements.go`.
- A size-changing patch no longer refuses immediately. It widens to the
  nearest size-stable ancestor. Table cells, rows, and inline content
  retarget to the owning table and splice only when the grid keeps its
  geometry. `internal/layout/relayout_subtree.go`.
- A batch can take a paint-only splice when geometry, pagination inputs,
  and paint-order keys do not change. The previous box list, breaks,
  keyframes, owners, and order are reused, and box text is refreshed in
  place. `commitPaintOnlySplice`, `patchedBatchDisplay`.
- `layout.Display.Release` returns retained box, style, and store chunks to
  their pools. It is nil-safe and idempotent. `layout/displaylist.go:227`.
- Fast paths skip no-op fingerprints, provably inert overrides, and
  text-free font loads. `layout/relayout_element.go`.
- Hidden atomic inline boxes stay in the box tree, so a row visibility
  toggle patches instead of failing on a child-count mismatch.
  `internal/layout/inline.go`.
- New public calls: `layout.RelayoutElements`, `layout.Display.Release`,
  `css.Document.HoverChangedScopes`, `html.Document.DirtyIDs`,
  `DirtyStructural`, `ClearDirty`, and `internal/html.DirtySet`.
  `DisplayGroup.BackdropFilter` is a new field on the existing Go type.

### Cache lifetime

PR #36 changes who owns the caches.

- The grid track memo moved from a process global to `gridCaches` on the
  engine and is cleared by `releaseScratch`. `internal/layout/grid_memo.go`,
  `internal/layout/layout.go:890`.
- Flex layout borrows the resolved style and allocates a private copy only
  when `aspect-ratio` must supply an auto height. `internal/layout/flex.go`,
  `internal/layout/aspect_ratio.go`.
- The element-patch fingerprint cache uses weak result and document keys.
  `layout/relayout_element.go:77`.
- The selector sibling memo uses weak node keys and weak sibling-data
  values; the attribute memo uses weak node keys. A collected entry
  becomes a miss and rebuilds from the live tree.
  `internal/css/match.go:93`, `internal/css/match.go:664`.
- The shared 16-cap style chunk pool keeps up to 32 chunks of 16 records,
  about 2.9 MiB at the documented 5,872-byte record size. That is under
  the existing 4 MiB bound in `TestStyleChunkPoolMemoryBound`. The other
  per-size style lists and the two pooled subtree stores sit outside that
  test bound. `internal/layout/layout_chunks.go:81`,
  `internal/layout/style.go:1131`.
- Collection tests cover dropped grid trees, selector trees, CSS
  documents, and released layout results.

The `html` id and dirty registries are still keyed by `*Document`.
`html/attribute.go` installs a finalizer that deletes both entries, but the
registry key itself keeps the document reachable, so collection is not
proven. This is a follow-up, not a shipped guarantee.

### Backdrop filter

- `backdrop-filter` is parsed and carried on the element's blend group,
  which the public `layout.DisplayGroup` aliases.
  `internal/layout/style_properties.go`, `internal/layout/blend_group.go`,
  `layout/displaylist.go:107`.
- The property moves from Unsupported to Partial. The catalog now has 785
  rows: 555 implemented, 4 partial, 218 unsupported, 8 intentionally
  ignored, dated 2026-11-02.
- OwnFrame supports one pixel `blur()` capped at 64px and one `saturate()`
  capped at 3. Other length units, duplicate functions, SVG `url()`
  filters, and other functions remain unsupported. Blinkless carries the
  value; it does not composite the backdrop.
- PR #32 merged with a red lint job. PR #33 fixed it by moving
  `hasBackdropFilter` next to the group type and repairing the size
  ledger.
- The WASM `drawGroupJSON` has no backdrop field. Go hosts see
  `DisplayGroup.BackdropFilter`; browser consumers of the WASM JSON do not
  see it in 0.1.1. That gap is in the follow-up ledger.

### Docs, catalog, and CI

- `documentation/architecture/07-layout.md` gains a "Cache lifetime during
  interactive layout" section.
- `documentation/library-api.md` records the batch relayout API and the
  scale benchmark.
- `documentation/fidelity.md` and
  `documentation/compatibility-matrix.md` move to the 2026-11-02 catalog
  counts.
- `.golangci.yml` keeps correctness linters on tests but drops the
  production cuddling, identifier-length, and complexity budgets from test
  files, with a comment.
- `scripts/file-size-allowlist.txt` now records eight over-limit files
  instead of one: `internal/html/tree.go` 2,049,
  `internal/layout/flex.go` 2,184, `internal/layout/inline_paint.go`
  2,123, `internal/layout/layout.go` 2,497,
  `internal/layout/relayout_subtree.go` 2,835,
  `internal/layout/style_cascade.go` 2,289,
  `internal/layout/style_value_accept.go` 2,018, and
  `internal/layout/transform.go` 2,098.
- GitHub CI runs `make test` and `make lint` only. PR #32 merged red;
  #33 fixed it; #34, #35, and #36 merged green.

## Behavior changes hosts must know

| Change | What to do |
|--------|------------|
| A size-changing patch may now succeed by widening to an ancestor. | Do not assume only the named element moved. Repaint the escalated range, or keep a full-layout fallback. |
| A table cell, row, or inline edit retargets to the owning table. | `ElementPatch.Old` and `New` stay the named element's box. They may not cover content that widened inside the table. |
| A no-op or inert override returns `prev` with empty `Ops`, `OpsOwners`, `Boxes`, `Old`, and `New`. | Read `patch.Display`; do not read the payload fields unconditionally. |
| `RelayoutElements` is all-or-nothing. | Validate ids before the batch, or call per element when partial success is needed. |
| `Display.Release` returns chunks to pools and sets `result` to nil. | Release only when no reader will use the display. Released slices are not zeroed, and the call is not a memory-safety guarantee. |
| `HoverChangedScopes` can return `false`. | Lay out the page when it does. An empty result with `true` means no selector match changed. |
| The dirty accessors record edits but no layout path consumes them yet. | Clear the set after adoption. Do not treat it as incremental layout. |
| `backdrop-filter` is metadata only, and the WASM JSON omits it. | Composite the backdrop in the host, and do not build browser features on a WASM field that does not exist. |

## Known issues

- `outline-style: solid` with no explicit width can paint no outline. The
  chrome early exit tests the raw `OutlineWidth`
  (`internal/layout/layout_chrome.go:497`), but CSS gives a visible style
  the `medium` width (`internal/layout/outline.go:38-45`). A fix and a
  full-layout test are in the follow-up backlog.
- `Box.OperationIDs` points into a shared backing array. Treat it as
  read-only; a host append can overwrite a later box's IDs.
- The gradient texture cache is capped by entry count, not bytes. A 1D
  gradient under an element filter has no parity test yet.

## Not wired in 0.1.1

These files are present, tested, or staged, but no production layout path
uses them. Do not build host features on them.

- Retained style resolve: `resolveStylesRetainedWithContext` in
  `internal/layout/style_cascade.go:2118` has only test callers.
  `internal/layout/style_retain.go` describes the contract.
- Block intrinsic memo: `internal/layout/memo_block.go:8` says "designed,
  not wired".
- Per-box struct and slice pools: `internal/layout/box_alloc.go:34` says
  the call-site change is not wired. `rentBox`, `rentClipRect`,
  `rentBoxRows`, `rentOpIDs`, and `rentPlaced` have no production caller.
- Batch damage windows: `SubtreeBatchResult.Windows` is produced and
  tested, but `RelayoutElements` discards it. There is no public damage
  payload.
- Subtree reuse: `trySubtreeReuse` is wired only into
  `RelayoutSubtreeWithOverrides`, and the public empty-override path
  delegates to `RelayoutElement`, so it is effectively dormant.
- Op-range translation and the retained-style path are not wired.
- The dirty accessors record edits, but no cascade or layout call reads
  `DirtyIDs` yet.

## Surface at this tag

- Parse and cascade. `html.Parse` handles doctypes, namespaces, foster
  parenting, the adoption agency, templates, select, fragments, and ruby
  recovery. `css.Apply` cascades author styles onto the tree. The catalog
  holds 785 rows: 555 implemented, 4 partial, 218 unsupported, 8
  intentionally ignored, dated 2026-11-02. Implemented means a handler, a
  consumer outside the style layer, and a resolving behavior test, not
  Chrome parity.
- Layout. Block, flex, and grid, multicol breaks with orphans and widows,
  floats, abspos and fixed placement against named anchors, corner shapes,
  interior shape wrapping, named flows, row and column gap rules, and
  `object-fit` clipping. 3D transforms flatten to 2D at stamp time.
- Masks and effects. Ten standard mask properties plus eight `-webkit-`
  aliases attach layer data to element groups. Thirteen mask-border names
  compose a nine-slice frame raster. `backdrop-filter` carries a filter
  list on the group. The host composites.
- Motion records. Animation and transition values, keyframes, per-box
  ownership, and declared scroll and view timelines ride the display list.
  Blinkless records them; the host supplies time and inputs. There is no
  clock in this engine.
- Patches and restamps. `RelayoutElement` re-places one retained subtree.
  `RelayoutElements` batches several. `RestampTransforms` re-projects
  transforms without a fresh layout. `RestyleInlineSVG` restyles inline
  SVG paint. The new batch and release calls are additive.
- Output. `layout.DisplayList` returns ops, order, boxes, groups, fonts,
  and images. The WASM binding serializes schema
  `blinkless.drawinglist/1`. Screen callers can snap strokes with
  `SnapDisplayToDevicePixels`. Nothing here writes a page image.
- Budgets. The 0.0.2 caps stay: 32 MiB stylesheets, 1024 element depth and
  formatting entries, 1024 dash and tile steps, 16M decoded image pixels,
  restricted local reads, and binding byte caps. Public `html.Parse` still
  takes no byte cap; `ParseWithLimit` defaults to 64 MiB.

## Version strings

These stamps are not one number.

| Stamp | Value | Where |
|-------|-------|-------|
| This release | 0.1.1 | GitHub tag `v0.1.1` |
| Previous release | 0.1.0 | tag `v0.1.0`, commit `2e2abfc` |
| Catalog date | 2026-11-02 | `testdata/css/catalog/properties.json` |
| WASM build stamp | 0.0.2 | `Makefile` `WASM_VERSION` |
| Go | 1.26 | `go.mod`, toolchain `go1.26.4` |
| Downstream | OwnFrame | sibling checkout reads `layout.DisplayList` |

The WASM stamp was not moved in this release. There is no `VERSION` file.

## Verification

The source pull requests recorded these gates:

- PR #31: GitHub `test + lint` green. Its body claims `make test` and
  `make lint` clean, and records the downstream OwnFrame example numbers.
- PR #32: GitHub `test + lint` red. The lint job found four golangci
  findings and a stale size ledger.
- PR #33: GitHub `test + lint` green after the lint and size repair.
- PR #34: GitHub `test + lint` green. Its body records `make test`,
  `make lint`, `make golden`, and `make claim-scan`.
- PR #35: GitHub `test + lint` green. Its body records `go test ./...`,
  the race suite, `make test`, `make lint`, `make golden`, and
  `make claim-scan`.
- PR #36: GitHub `test + lint` green. Its body records `make test`,
  `make lint`, `make golden`, `make claim-scan`, `make build`,
  `make matrix-check`, the race suite, `go vet`, and a WASM build.

PR #36 and the v0.1.0 release PR both record `make catalog-check` as red
before this release. The failure has two parts: seven animation and
transition sample strings are absent from the sibling OwnFrame checkout,
and the catalog lists the removed `wrapBreakInsideAvoid` handler for
`wrap-inside`. The first part is a sibling-checkout mismatch. The second
part is source drift that is not fixed in 0.1.1. `make matrix-check`
passed at PR #36.

The release PR changes documentation, plans, and catalog-facing prose
only. It changes no Go source. Local engine gates were not rerun for that
docs-only change. CI still runs `make test` and `make lint` on the release
PR.

## What still does not ship

PDF files, PDF profiles, the `blinkless` command, veraPDF, and a golden
PDF corpus. One image operation is re-encoded as PNG when orientation or a
clip requires it. That is not a picture of the page.

Animation, transition, and timeline records carry no clock. Masks, mask
borders, and backdrop filters carry values for the host to composite;
there is no in-engine pixel replay. `backdrop-filter` stays Partial and
the WASM JSON does not carry it. The retained-style path, block memo, some
box pools, batch damage windows, and op-range translation are not wired.
The `html` id and dirty registries can pin a document. Scroll, speech, and
SVG presentation rows stay unsupported except where their ledgers say
otherwise. No document script runs, and none of this is a proof that every
bug is gone.
