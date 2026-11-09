# Render-cost context

This note covers PR #31 (`e3df1db`, `76ee989`, `22ffbf0`, `9ba4eb5`,
`d16e629`, merge `f50f76a`, branch `perf/render-cost-suite`) and the
later allocation work that connects to it. Anchors are at `3e3a8b7`.

## What each wave changed

**Wave 1, `e3df1db`.** Added a resolved-style arena. `Result` and `engine`
hold `styleChunks` and `spareChunks`; `allocStyle` copies a base style into
a chunk instead of letting a 5.8 KB stack copy escape. `markStyles` and
`resetStyles` let a speculative measure pass reclaim records.
`Result.Release` returns chunks. Gradients gained a process-global cache
keyed by raw source, dimensions, current color, and interpolation, cleared
wholesale above 1024 entries. PNG encoding moved to `png.BestSpeed` with a
pooled `png.EncoderBuffer` and a pooled `bytes.Buffer`, copying the bytes
out before the buffer is reused. Flex child partitioning gained a pre-scan
that returns the input slice when no child is out of flow; `flexChildren`
returns `node.Children` when every child is a visible element; row and grid
overrides moved from `override := *style` to `allocStyle`.
`OperationOwnersPlaced` stopped rebuilding `PlacedElements`, and
non-retained display construction releases the result.

**Wave 2, `76ee989`.** Added the box arena: `boxChunks`, `spareBoxChunks`,
`markBoxes`/`resetBoxes`, and `allocBox`. Block, flex, grid, table, cell,
image, HR, SVG, and multicol boxes now allocate from 64-box chunks. Flex
measure passes roll back both boxes and styles. Gradient backgrounds with a
non-repeating linear gradient at exactly 0, 90, 180, or 270 degrees
request a 1-pixel-wide or 1-pixel-tall texture, then draw it at the full
destination size. `PlacedElements` collects all operation IDs into one
backing slice and returns sub-slices. Inline font metrics skip
`splitTextByFace` when every rune is covered by the primary face.
`estimateOpCapacity` was raised to `2*nodes+16`.

**Wave 3, `22ffbf0`.** Added `flexKidPool` and `flexChildrenPooled`, so
filtered flex child slices are rented and returned. Anonymous flex items use
`allocStyle` plus a field reset instead of a fresh style copy stored on the
heap. `TextContent` gained pooled `strings.Builder` use and a `hasAnyText`
early exit. `WalkPlaced` became a callback walk that reuses one
`PlacedElement`; `PlacedElements` wraps it for callers that need a slice.
`OperationOwnersFromParents` derives owners from a parent-index slice
instead of the full placed records. `newDisplay` builds public boxes and
parent indices in one walk, then derives owners. Background blend and size
lookups gained empty and single-token fast paths.

**Wave 4, `9ba4eb5`.** Extended speculative rollback to
`flexItemBaseHeight`, `flexMinCrossMainSizeUncached`,
`measureVerticalColumnItemCross`, `measureCellHeight`, and
`measureMulticolChildHeight`. Added the chrome early exit in `prependChrome`
after overflow clipping. Replaced the deferred-chrome subtree map with
recursive `boxContains` plus zero-delta and leaf fast paths. Paint order
moved to `slices.SortStableFunc` with a total-order `comparePaintOrder` and
an already-sorted scan. Added the single-class attribute-key memo. Reduced
`estimateOpCapacity` to `nodes + nodes/4 + 16`. Bumped the placed
operation-ID arena to `4*len(ops)` and added range clamping plus a pre-scan
for ID-less boxes.

**Lint, `d16e629`.** Split the chunk code into `layout_chunks.go` and the
flex child helpers into `flex_children.go`; removed the dead `operationIDs`
helper, tightened pool type assertions, and updated the file-size ledger. No
intended behavior change.

**Current HEAD differs in details.** Style arena chunks are now 16 records
with a bounded 32-chunk free list, not 64-record `sync.Pool` entries. Flex
overrides now go through a per-run cache. Parser tag and attribute scratch,
simple flex heights, and `Display.Release` are later PR #35 and #36 work,
not PR #31.

## Key files and line anchors

| Mechanism | Current anchors |
|---|---|
| Box and style chunks, release, rollback | `internal/layout/layout_chunks.go:33`, `:68`, `:116`, `:126`, `:148`, `:241`, `:251`, `:273`; constants `:68-82` |
| Engine and result chunk fields | `internal/layout/layout.go:219-220`, `:671-674`; flatten capacity `internal/layout/layout_context.go:68` |
| Op capacity heuristic | `internal/layout/layout.go:1630-1648` |
| Gradient cache, 1D test, PNG pool | `internal/layout/gradient.go:54`, `:490`, `:509`, `:514`, `:527-531`, `:569-605`, `:621`; request `internal/layout/background_image.go:158-166` |
| Flex child pool and anonymous style | `internal/layout/flex_children.go:21`, `:58`, `:108`, `:177`; partition `internal/layout/flex.go:337` |
| Measurement rollback sites | `internal/layout/flex.go:1459`, `:1915`, `:2024`; `internal/layout/flex_logical_axes.go:164`, `:278`; `internal/layout/layout_tables.go:1234`; `internal/layout/multicol.go:935` |
| Chrome bypass | `internal/layout/layout_chrome.go:488`, gate `:526`; deferred containment `internal/layout/layout_flow.go:1617`, `:1636` |
| Paint order | `internal/layout/paint_order.go:53`, `:65`, `:253`, `:295` |
| Placed walk and owners | `internal/layout/placed.go:60`, `:162`, `:281`; `internal/layout/op_owners.go:32`, `:39`; `layout/displaylist.go:404-409`, `:570-609` |
| Class-key memo | `internal/layout/style_match_cache.go:14`, `:28-29`, `:291-329` |
| TextContent pool and font fast path | `internal/html/html.go:83`, `:121`, `:135`; `internal/layout/inline_paint.go:1982-1996` |

## Test coverage that exists

PR #31 changed no test files. Coverage is indirect through the existing
suite:

- `internal/layout/paint_order_test.go:5-164` exercises `PaintOrder` through
  the new comparator. It does not test the already-sorted branch directly.
- `internal/layout/background_image_test.go:152` checks that a gradient
  paints chrome. `internal/layout/paint_modern_test.go:30,65` checks conic
  pixels. `internal/layout/css_behavior_color_interpolation_test.go:90,122`
  calls `renderGradientPNGWithInterp` with cacheable keys, but does not
  assert cache hits, bounds, or 1D collapse.
- `internal/layout/layout_chunks_test.go:10,62,88` tests the style free list
  bound, clearing, and the 4 MiB cap. `style_chunk_reuse_test.go:70,94`
  tests style-store reset identity. `style_store_chunk_test.go:8,68,122`
  tests store chunks and release. None tests `allocBox`/`allocStyle`
  rollback after a measure pass.
- `layout/displaylist_test.go:85,134,177,196` checks boxes, parent indices,
  and owners. `layout/displaylist_emit_test.go:181` checks fused owners and
  order against the separate functions and a 2-alloc bound.
  `layout/displaylist_alloc_test.go:187` pins `DisplayOrder` at one alloc.
- `internal/layout/style_match_cross_test.go:102-109` checks that the
  single-class key survives across layouts. That test was added later, not
  by PR #31.
- `internal/layout/outline_test.go:11` covers the outline shorthand.
  `border_medium_default_test.go:196` checks `effectiveOutline` directly,
  not the layout path.
- TextContent behavior is covered by `internal/html/template_select_test.go`
  and `html/attribute_patch_test.go:182,210`.

## Performance claims and provenance

| Claim | Source | Status |
|---|---|---|
| Cold latency 3,745 ms to 537 ms (6.97x), heap 2,134 MB to 160 MB (>13x) | PR #31 body, `ownframe-perf/examples/performance` | Downstream example app, not committed here. No fixture, command, commit, or raw results. |
| About 40.8 MB of heap churn reclaimed | PR #31 body | No committed profile or benchmark. Only flex, table, and multicol rollback is implemented; grid measurement still allocates. |
| Engine figures TBD; downstream numbers now recorded with provenance | `CHANGELOG.md` 0.1.1 Performance section | The release PR records the OwnFrame example numbers as downstream-only and keeps the invoice benchmark TBD. |
| Scale benchmark landed, figures TBD | `documentation/library-api.md` batch relayout section; harness `layout/invoice_scale_bench_test.go:45` | Harness only. No recorded result. |
| Display order path costs one alloc | `layout/displaylist_alloc_test.go:187` | Committed allocation assertion, not speed. |
| Flex allocation churn benchmark | `internal/layout/flex_alloc_bench_test.go:11` | Harness added later, no committed output. |

There is no committed benchmark for the gradient cache, 1D collapse, PNG
pool, chrome bypass, box and style arenas, class memo, paint-order sort, or
display walks. `testdata/golden/benchmarks/README.md:1-4` says its harness
was removed with the PDF writer, so those historical numbers do not apply.

## Risks, gaps, and follow-ups

1. **Outline bypass can drop paint.** `hasPotentialChrome` tests raw
   `OutlineWidth > 0`, but `outlinePaints` uses `effectiveOutline`, which
   supplies the CSS medium width when `outline-style: solid` has no width.
   A layout with only `outline-style: solid` can now paint no outline. Use
   `outlinePaints(&sty)` in the gate and add a full-layout test.
   `internal/layout/layout_chrome.go:497`,
   `internal/layout/outline.go:38-45`.
2. **Rollback is incomplete.** `measureGridItemPrefH`
   (`internal/layout/grid_memo.go:154-163`) is wired and still builds boxes
   and styles without mark/reset. The block intrinsic memo
   (`internal/layout/layout.go:1608-1622`) is unwired but has the same
   shape. Add rollback and an allocation assertion.
3. **Gradient cache has no byte budget.** It is process-global and capped at
   1024 entries, not bytes. Returned `[]byte` values are shared across ops.
   A 1D gradient is also filtered after collapse, so an element filter sees
   a 1-pixel texture instead of a full-size one. Add a filter parity test
   and a byte cap.
4. **`shiftDeferredChrome` changed complexity.** Replacing the subtree map
   with `boxContains` removes allocations but makes each entry walk the
   subtree. Large centered flex subtrees with many chrome entries can
   regress from linear to entries-times-subtree. Benchmark it and keep a
   map fallback for large sets.
5. **Operation ID slices alias.** `WalkPlaced` returns `OperationIDs`
   sub-slices of one backing array. The exported `Box.OperationIDs` field
   is mutable, so a host append can write into a later box's IDs. Document
   it as read-only or copy per box.
6. **Capacity estimates are unbenchmarked.** `nodes + nodes/4 + 16` is a
   heuristic. The flattened box list is sized from chunk count times 64, so
   partly filled chunks over-reserve. Add invoice and table benchmarks and
   compare growth counts.
7. **Multicol anonymous styles still copy.** `internal/layout/multicol.go:220-222`
   builds an anonymous style by value, bypassing the arena. Pool it like
   the flex path.
8. **Release-note corrections.** Boxes and parents come from one walk;
   owners come from a second walk over parents. The 1D gradient path is a
   1-pixel PNG texture stretched by the op, not a vector fill. The style
   arena is 16-entry chunks; the 8/16/32/64 chunks belong to the style
   store.

## Do not claim

- Any exact speedup, cold-start time, or heap reduction from PR #31.
- "40.8 MB reclaimed" as a Blinkless measurement.
- That every `noEmit` pass rolls back boxes and styles. Grid measurement
  does not.
- "Zero allocations" or "no PNG encoding" for gradients. Cache misses still
  rasterize and encode, and the output is still PNG bytes.
- Pixel-identical 1D gradients under element filters until a parity test
  exists.
- That chrome bypass preserves every outline. The medium-width outline case
  is a live gap.
- That owners are built in the same walk as boxes.
- That parser tag and attribute scratch is part of PR #31. It landed later in
  `40a436f` and `82be8d6`.
- That PR #31 added a public API. `Result.Release` is internal; public
  `Display.Release` came later.
- That style chunks are 64 entries or unbounded. Current HEAD uses 16-entry
  chunks and a bounded 32-chunk free list.
