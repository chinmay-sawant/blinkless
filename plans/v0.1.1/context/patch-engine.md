# Patch engine context

This note covers PR #34 (`24a7ab7`, `36edc21`, merge `1b88147`) and PR #35
(`40a436f` through `8bce5a1`, merge `d805508`). Anchors are at `3e3a8b7`.

## Public API added or changed

```go
func RelayoutElement(ctx context.Context, doc *css.Document, prev *Display,
    elementID string, options Options) (*ElementPatch, error)
func RelayoutElementWithOverrides(ctx context.Context, doc *css.Document, prev *Display,
    elementID string, overrides []Override, options Options) (*ElementPatch, error)
func RelayoutElements(ctx context.Context, doc *css.Document, prev *Display,
    elementIDs []string, options Options) (*Display, error)
func (d *Display) Release()
func (d *css.Document) HoverChangedScopes(previous string) ([]string, bool)
func (d *html.Document) DirtyIDs() []string
func (d *html.Document) DirtyStructural() bool
func (d *html.Document) ClearDirty()
```

`RelayoutElement` (`layout/relayout_element.go:135`) edits the tree first,
then re-places one subtree inside the retained placement from `prev`.
`prev` must come from `DisplayListOptions` with `Options.RetainState`
(`layout/layout.go:157`, `layout/displaylist.go:269`). On success
`ElementPatch.Display` is the whole patched page and `prev` is stale. The
engine does not lock retained state, so one display must not be patched
concurrently (`layout/relayout_element.go:133`).

`RelayoutElements` (`layout/relayout_elements.go:41`) threads ids
sequentially over one private clone of the placement and builds one
display. Empty ids returns `prev, nil`. A missing id becomes `ErrNoElement`;
a refused id returns the matching per-element error and leaves `prev`
untouched (`layout/relayout_elements.go:53-69`). Ids cannot run in parallel
because each splice feeds the next (`layout/relayout_elements.go:35-40`).

`Display.Release` (`layout/displaylist.go:227`) is nil-safe and idempotent.
It drops the retained result and returns box, style, and store chunks to
pools at refcount zero (`internal/layout/layout_chunks.go:31-66`). A patch
display retains its result, so releasing `prev` first is safe.

`HoverChangedScopes` (`css/hover_scopes.go:83`) returns the outermost
nearest-id-bearing scopes whose hover-dependent selector matches changed
between `previous` and the document's current hover id. Empty result with
`true` means nothing changed. `false` means no safe scope exists, or the
document is nil, so the caller must lay out the page
(`css/hover_scopes.go:75-107`).

Dirty accessors live on `html.Document` (`html/attribute.go:289,299,309`).
`DirtyIDs` is sorted; the set is monotonic until `ClearDirty`;
`DirtyStructural` records a `SetText` child-shape change
(`internal/html/dirty.go:9-18`).

## Behavior changes hosts must know

**Size escalation.** A size-changing single or batch patch no longer
refuses immediately. `relayoutBuildEscalating` rebuilds the target, and on
`errSubtreeResized` widens to the holder until an ancestor keeps its size
(`internal/layout/relayout_subtree.go:127-180`). A moved table grid,
root-reaching growth, float, sticky box, anchor, named flow, ancestor
transform or opacity, and `@container` documents still refuse
(`internal/layout/relayout_subtree.go:221-251,405-459,480-575`).

**Table retargeting.** A cell patches through its owning table
(`tableOwnerTarget`, `internal/layout/relayout_subtree.go:349`); a box-less
row or inline content patches through the nearest table ancestor
(`tableAncestorTarget`, `:378`). The rebuilt table commits only when the
whole grid keeps its geometry (`relayoutSameTable`, `:1295`).

**Paint-only splices.** Batch only. When op count, box geometry,
pagination inputs, and paint-order keys are unchanged,
`commitPaintOnlySplice` copies the new ops over the old window and keeps
the previous box list, ranges, owners, order, breaks, keyframes, size, and
zoom (`internal/layout/relayout_subtree.go:1930-1957`;
`layout/relayout_elements.go:360-379`). `Box.Text` for the patched elements
and ancestors is refreshed in place (`layout/relayout_elements.go:126-152`).
The previous display shares those boxes, so it is stale after adoption.

**All-or-nothing batches.** The batch works on `CloneResult(prev.result)`
and copies owners and order. Any failure returns `nil, error`; no partial
patch is adopted (`layout/relayout_elements.go:53-69`).

**No-op fast path payload.** A fingerprint hit or provably inert override
returns `&ElementPatch{Display: prev}` with empty `Ops`, `OpOwners`,
`Boxes`, `Old`, and `New` (`layout/relayout_element.go:156-172,821-838`).
Image subtrees skip the fingerprint (`:162-172`).

**Damage rectangles.** `ElementPatch.Old` and `New` are the named
element's border box, rounded to whole CSS pixels
(`layout/relayout_element.go:493-540`). `Old` falls back to `New` when the
id has no previous box; `New` falls back to the first box of the patched
range. Batch has no public damage payload. `SpliceWindow` carries
operation-window damage internally
(`internal/layout/relayout_subtree.go:2430-2444`), but `RelayoutElements`
reads only `PaintOnly`, `Owners`, `Order`, `Texts`, and `Textless`
(`layout/relayout_elements.go:82-88`).

**Display release.** Release only when no reader will use the display.
Released slices are not a memory-safety guarantee. `Display.Release` does
not clear `Ops` or `Boxes`.

**Hover scopes.** A changed element without an id uses its nearest
id-bearing ancestor. The helper is conservative and bounded by a four-list
cache, a 1024-entry memo, and a 512 KiB key budget
(`css/hover_scopes.go:14-28,136-173`).

**Dirty accessors.** Only `SetAttribute` and `SetText` mark. `SetText`
marks the edited element and ancestors up to the nearest block; only a
child-shape change sets the structural flag (`html/attribute.go:346-370`).
The dirty set does not replace a full layout check for edits made by other
means.

## Key files and line anchors

| Concern | Anchor |
|---|---|
| `ElementPatch` | `layout/relayout_element.go:83` |
| `RelayoutElement` | `layout/relayout_element.go:135` |
| No-op fingerprint | `layout/relayout_element.go:156-182,548-775` |
| Overrides | `layout/relayout_element.go:224`; `layout/layout.go:195` |
| Batch entry and commit | `layout/relayout_elements.go:41,53,82` |
| Display and `Release` | `layout/displaylist.go:159,225` |
| Escalation | `internal/layout/relayout_subtree.go:127` |
| Table retarget | `internal/layout/relayout_subtree.go:349,378,1295` |
| Splices | `internal/layout/relayout_subtree.go:1896,1937` |
| Batch structures | `internal/layout/relayout_subtree.go:2434,2459,2493` |
| Hover scopes | `css/hover_scopes.go:75-107` |
| Tree edits and dirty | `html/attribute.go:20,86,289`; `internal/html/dirty.go:9` |

## Test coverage

- `layout/relayout_element_fast_test.go`: identical no-op returns `prev`,
  ignored attributes stay no-op, text-free font skip, inert overrides, and
  negative cases that must still relayout.
- `layout/relayout_element_parity_test.go`: patched result matches a full
  relayout; splice guards hold.
- `layout/relayout_element_table_test.go`: cell text, row class, and
  span-in-cell splices; moved grid refuses.
- `layout/relayout_elements_test.go`: empty batch returns `prev`, batch
  matches sequential calls, refusal leaves `prev` untouched, large table
  commits all fills.
- `layout/relayout_elements_paint_only_test.go`: paint-only class change
  matches full layout, box text refreshes, padding and mixed batches fall
  back as a whole.
- `layout/relayout_elements_breaks_test.go`: page breaks are reused only
  when equality proves they cannot move.
- `layout/displaylist_release_test.go`: nil, idempotent, retained, and
  patched release lifecycle.
- `internal/layout/relayout_size_escalation_test.go`: width, height, and
  batch escalation; exhaustion and float refusal.
- `internal/layout/batch_splice_test.go`: batch equals sequential splices,
  incremental owners and order equal a full rebuild, refusal parity,
  op-window-only damage, census parity, paint order parity.
- `internal/layout/batch_paint_only_test.go`: paint-only parity, text
  refresh, padding fallback, mixed batch fallback.
- `internal/layout/relayout_areachart_damage_test.go`: narrow spacer change
  versus escalated scale damage.
- `internal/layout/relayout_subtree_test.go`: inherited paint context and
  viewport-fixed patches.
- `css/hover_scopes_test.go`, `css/hover_scopes_cache_test.go`: selector
  changes, id requirement, memo parity, concurrency.
- `html/dirty_test.go`, `html/attribute_patch_test.go`,
  `html/find_index_test.go`: dirty marking, rename, concurrent edits,
  id-index parity.

## Present but not wired into the layout call path

**Retained styles.** `resolveStylesRetainedWithContext`
(`internal/layout/style_cascade.go:2118-2128`) has only test callers.
`style_retain.go:11-16` says the interface keeps the style pass compiling
"until it lands". It is not called from `Layout` or the patch paths. Its
entries would also point into chunk backings that `Release` can recycle.

**Block intrinsic memo.** `internal/layout/memo_block.go:8` says "designed,
not wired"; only `memo_block_test.go` calls it.

**Per-box struct and slice pools.** `internal/layout/box_alloc.go:34` says
"Adopt contract for the future call-site change (not wired yet)".
`rentBox`, `rentClipRect`, and `rentBoxRows` have no production caller
(`:295,325,390`). `returnBoxPtrs` is used by `Result.Release`.

**Batch damage windows.** `SpliceWindow` is produced
(`internal/layout/relayout_subtree.go:2542,2651`) and asserted by tests,
but no host-facing consumer exists. `layout/relayout_elements.go:82-88`
discards `batch.Windows`.

**Subtree reuse.** `trySubtreeReuse` is wired only into
`RelayoutSubtreeWithOverrides` (`internal/layout/relayout_element.go:80`).
Its header says the single-element entry is that function with no overrides
(`subtree_reuse.go:25-29`), but the public
`RelayoutElementWithOverrides` delegates empty overrides to
`RelayoutElement` (`layout/relayout_element.go:228-230`), and non-empty
overrides always miss (`subtree_reuse.go:81-85`). It is effectively dormant
for public callers.

**Dirty set consumption.** The set is recorded by `html`, but no layout
call reads `DirtyIDs`. The intended consumer is the unwired retained-style
path.

**Hover scopes.** Public helper only. No internal caller.

## Migration notes and under-invalidation risks

- A successful size-changing patch may now move siblings. Do not assume
  only the named element changed. Repaint the escalated ancestor, or keep a
  full-layout fallback.
- `Old` and `New` may not cover the escalated range or table content that
  widened inside a retargeted table. The PR body names this exact table
  risk (`plans/PR/pr-batched-relayout-and-caches.md:114`).
- Do not read `ElementPatch.Ops`, `Boxes`, `Old`, or `New` unconditionally
  after a no-op or inert-override call.
- Releasing `prev` first is safe for a patch display, but release order
  matters for any other retained reader.
- Batch is all-or-nothing. Validate ids up front, or call per element when
  partial success is required.
- `ClearDirty` after adoption. Marking stops at the nearest block, so wider
  damage relies on the structural flag and a full layout.
- `HoverChangedScopes` returning `false` must fall back to a full layout.
- `sameBoxPaintInputs` does not compare animation, transition, transform,
  timeline, or clip metadata
  (`internal/layout/relayout_subtree.go:1397-1420`). A metadata-only change
  can take the paint-only path and keep stale metadata.

## Candidate follow-ups

1. Wire or delete retained styles (`internal/layout/style_retain.go`,
   `style_cascade.go:2118`). Resolve chunk lifetime first.
2. Wire or delete the block memo (`internal/layout/memo_block.go:8`).
3. Expose or consume `SubtreeBatchResult.Windows`, or remove it
   (`internal/layout/relayout_subtree.go:2434,2493`;
   `layout/relayout_elements.go:82`).
4. Extend `sameBoxPaintInputs` for animation, transition, transform,
   timeline, and clip metadata, and add a metadata-only test
   (`internal/layout/relayout_subtree.go:1397`).
5. Validate the sibling memo with a generation or child identity, not only
   child count (`internal/css/match.go:183-211`).
6. Wire or delete unused pools and dead helpers
   (`internal/layout/box_alloc.go:295-397`;
   `internal/layout/style.go:1256-1262`).
7. Fix stale docs and test comments: `documentation/library-api.md:70-80`
   cites old line numbers; `internal/layout/relayout_row_span_test.go:9`
   still says "EXPECTED TO FAIL";
   `TestBatchSpliceRefusalMatchesSingleError` now asserts success
   (`internal/layout/batch_splice_test.go:197-203`).
8. Add explicit table-window damage guidance, or a public damage API, for
   retargeted table patches.

## Do not claim

- Do not claim retained styles, the block intrinsic memo, or per-box struct
  pooling shipped. They are designed and tested only.
- Do not claim host-facing damage rectangles. `SpliceWindow` is internal
  and test-only; `RelayoutElements` drops it.
- Do not claim batch ids run in parallel. They run sequentially by design.
- Do not claim every patch succeeds. Floats, sticky boxes, anchors, named
  flows, transforms, opacity, container queries, moved grids, and
  root-reaching growth still refuse.
- Do not claim the dirty accessors drive incremental cascade or layout. No
  layout path consumes them yet.
- Do not claim paint-only handles animation, transition, transform,
  timeline, or clip metadata changes.
- Do not claim engine performance figures. The 0.1.1 CHANGELOG records only
  downstream numbers with provenance and keeps the invoice benchmark TBD.
- Do not claim `Display.Release` frees memory immediately or zeroes
  released slices.
- Do not claim table patches damage the whole table. `Old` and `New` stay
  the named element's box.
