# perf(layout): batch relayout patches, add fast height paths, and bound caches

Filled copy of the PR template. Base: `master`. Head: `fix/hover-subtree-paint-context`. Diff before this plan file: 136 files, +26,658 / -1,286.

---

## Summary

- Turns the single-element patch path into a placement patch engine. `RelayoutSubtree` now escalates size-changing edits to the nearest size-stable ancestor, retargets table-internal edits to the owning table, and commits through a fused splice instead of refusing or rebuilding the page. A new `RelayoutElements` batch entry threads several ids over one retained placement.
- Adds a public batch API and a display-list lifetime: `RelayoutElements`, `Display.Release`, paint-only batch commits that reuse the previous display's boxes, breaks, owners, and order, and patch/display builders that reuse buffers.
- Cuts the layout hot path with differential-tested shortcuts: simple flex row and stack heights without a measure build, a bounded flex override cache, descriptor-keyed text measurement caches, a single-text-run inline path, a grid occupancy bitset, and pooled scratch.
- Bounds the style and selector caches, reuses style chunks, and adds a cross-layout compiled bundle. The retained-style engine and the block-intrinsic memo are present and tested but **not wired** into `Layout`; the PR description must not present them as shipped behavior.
- Replaces repeated HTML tree walks with a per-document id index, in-place `SetAttribute`/`SetText`, a dirty set, and parser allocation cuts. Selector matching gains bounded sibling, attribute, hover-list, and hover-outcome memos.
- Adds a mask-free flat 2D transform stamp, window-scoped owner and paint-order splices, conservative page-break equality, and a single-visit overflow clip path.

---

## Motivation / context

- This is the engine half of the Phase 07 patch-path work recorded in the ownframe `plans/v0.0.3/performance/` plan. Patch-heavy UIs call `RelayoutElement`/`RelayoutElements` on every keystroke, hover, and row update; the old path either refused common size changes or paid a page-sized rebuild.
- It follows the merged paint-context fix in PR #34. The branch is one merge commit behind `master` because that PR was merged; the three-dot diff contains only the 16 follow-on commits.
- `CHANGELOG.md` records the work as a landed/designed ledger. It marks figures TBD and calls out the block memo and part of box pooling as designed-only.
- Issues: none. Open issue #18 (mask-border frame regions) is unrelated.

---

## Changes

### Subtree relayout and batch splices

- `internal/layout/relayout_subtree.go` now resolves a patch target, checks the target and ancestors, and widens the rebuild while the result changes size (`relayoutBuildEscalating`). Growth that reaches the root, a moved table grid, a float, a sticky box, an anchor, a named flow, or an ancestor transform/opacity still refuses.
- Table-internal edits retarget: a `td` goes through the owning table (`tableOwnerTarget`), a box-less `tr` or inline content through the nearest table ancestor (`tableAncestorTarget`), and the rebuilt table splices only when the grid keeps its geometry (`relayoutSameTable`).
- `RelayoutSubtrees` batches ids over one retained placement (`internal/layout/relayout_subtree.go:2443`). Each id sees the placement the previous id left, matching sequential calls. A failure aborts the batch and leaves the input untouched because the public wrapper works on `CloneResult`.
- Two commit shapes: a geometry-neutral window overwrites its operations in place (`commitPaintOnlySplice`), while anything else runs the fused splice (`commitSubtreeSplice`) that slides the operation tail, rebases ranges, swaps the box, shifts later ranges, and rebuilds the flat box list at exact capacity.
- Owners and paint order move incrementally per splice (`spliceOwnerWindow`, `splicePaintOrder`) instead of being derived per id. The census advances from the spliced window and rescans only when the removed window held the page maximum or the only fragment link.
- Hidden atomic inline boxes are kept in the box tree (`internal/layout/inline.go:emitLineItems`), so a row visibility toggle patches instead of failing on a child-count mismatch.
- `SubtreeBatchResult.Windows` carries `SpliceWindow` damage descriptors and the tests assert them, but no code in `layout/` consumes them yet. The public wrapper only reads `PaintOnly`, `Owners`, `Order`, `Texts`, and `Textless`. Host-facing damage reporting is still test-only.

### Public patch API and display list

```go
func RelayoutElement(ctx, doc *css.Document, prev *Display, elementID string, options Options) (*ElementPatch, error)
func RelayoutElementWithOverrides(ctx, doc *css.Document, prev *Display, elementID string, overrides []Override, options Options) (*ElementPatch, error)
func RelayoutElements(ctx, doc *css.Document, prev *Display, elementIDs []string, options Options) (*Display, error)
func (d *Display) Release()
```

- `ElementPatch` carries `Ops`, `OpOwners`, `Boxes`, `Old`/`New image.Rectangle`, and `Display`. The host edits the styled tree first with `SetText`/`SetAttribute`; `prev` must carry `RetainState`.
- Fast paths skip engine work when the input fingerprint is unchanged, when an override is provably inert (`object-position` on a non-replaced element; spacing or font-size on a text-free subtree), or when a text-free subtree would otherwise load the default font. The fingerprint map is capped at 1024 entries and cleared wholesale on overflow.
- Empty batch ids return `prev`; an empty override list delegates to `RelayoutElement`. `ErrBatchUnknownID` maps to `ErrNoElement`; other engine errors pass through.
- Display construction reuses `prev.Boxes`, owners, order, and keyframes buffers when counts/capacities fit (`newDisplayForPatch`), and the batch build keeps `prev` page breaks when `PageBreaksEqual` proves they cannot move.
- `Display.Release` is nil-safe and idempotent. It delegates to a refcounted `Result.Release` that returns box, style, and store chunks to engine pools at zero. Patch displays retain the result, so releasing `prev` first leaves the patch valid (`displaylist_release_test.go`).

### Layout measurement and allocation

- `simpleFlexRowHeight` and `simpleFlexStackHeight` accept only fully guarded shapes (row/stack flex, nowrap, horizontal LTR static flow, no intrinsic/percent height, no aspect-ratio, no containment, single-run text children) and compute `padding + border + max(child outer height)` arithmetically. Any unproven shape declines to the existing measure build; `flex_simple_height_test.go` compares ops, height, max-content X, and every box rect with the shortcut on and off.
- `flex_override_cache.go` caches finished `ResolvedStyle` overrides per run, keyed by the base style pointer plus forced-length and flag bits, capped at 256 entries. Only interned cascade styles are cacheable; chunk-allocated copies past the cap are not stored.
- Text measurement gains exact descriptors (face, family, weight, italic, optical sizing, variations, emoji variant, scaled size, letter/word spacing, tab size, autospace), a process-wide descriptor interner capped at 4096, per-engine and shared string/rune caches (8192/4096), a shared coverage cache, and a line-height cache (1024). Long strings bypass the caches.
- `layoutSingleTextRun` handles the common one-text-run inline block, then falls back to the generic collector for wrapping, floats, soft hyphens, vertical writing, or multi-word content. Collection helpers now thread `*ResolvedStyle` to avoid 5.8 KB copies.
- Grid placement uses one `[]uint64` occupancy bitset per row instead of nested maps, and grid track intrinsics are memoized per `(node, style, axis)` with a 4096-entry cap.
- Box and style chunks are pooled and released through `Result.Release`; flex, grid, multicol, and table scratch slices are rented. `box_alloc.go` also contains `rentBox`, `rentClipRect`, `rentBoxRows`, `rentOpIDs`, and `rentPlaced`, which have **no production caller** yet.

### Style engine and caches

- `styleStore` copies each `ResolvedStyle` into 8/16/32/64-entry chunks, deduplicates by fingerprint plus full equality, and keeps pointers stable when a new chunk starts. `Result.Release` returns the chunk backings to bounded per-size free lists.
- Selector matching gains a per-parent sibling memo (soft cap 2048, hard cap 16384), an `i`-flag lowered-attribute memo (same caps), a per-pass match/gate cache, and pure-value memos for `@supports`, inline declarations, and accepted values (1024/4096 stop-at-cap).
- A cross-layout compiled bundle caches the rule index and interned tags keyed by sheet pointer/count, capped at 64 entries. It deliberately excludes per-document match outcomes.
- Subtree style resolution rents a pooled store keyed by document root and first sheet pointer, bounded at 2 stores / 4 chunks / 1024 intern records. A rejected store transfers its chunks to the previous result.
- `subtree_reuse.go` adds a paint-only fast path for class-only edits: it matches the kept subtree against the DOM, compares content and layout fingerprints, rebuilds without emit, splices the window, and refreshes the census. It is called from `RelayoutSubtreeWithOverrides`, not from `RelayoutSubtree`.
- **Not wired:** `style_retain.go` (`resolveStylesRetainedWithContext`) has only test callers, and `memo_block.go` says "designed, not wired". The retainer's ancestor-chain check also omits the element's own tag/attributes and sibling-position pseudos, and its entries would point into chunk backings that `Release` can recycle. Do not claim retained styles or the block memo as shipped.

### HTML/CSS patch path

- `html/attribute.go` keeps a per-document id index, so `SetAttribute`, `SetText`, and `Find` no longer walk the tree on every edit. `SetAttribute` writes the canonical store and the ordered list, preserves stored spelling, and updates id-index entries on rename. `SetText` reuses the single-text-node shape; a subtree drop deletes only ids that pointed into the dropped set.
- `internal/html/dirty.go` adds a monotonic dirty-id set plus a structural flag. Public accessors are `Document.DirtyIDs`, `Document.DirtyStructural`, and `Document.ClearDirty`.
- Parser allocation work reuses one attribute scratch per tokenizer run, slices attribute values zero-copy until an entity/rebuild, pools tree and attribute builders, and canonicalizes common tag/attribute names through switches rather than a growing intern table. `parse_alloc_test.go` records 9037 → 7035 → 3531 allocations per parse with a 4500 ceiling.
- Selector matching adds ASCII case folding, direct lower-case attribute reads, a `:has` budget only when the selector contains `:has`, and the memos above. `css/hover_scopes.go` caches at most four hover selector lists, precomputes element-local selectors, and bounds the hover outcome memo at 1024 entries / 512 KiB of key bytes.
- Transform stamping drops the separate `TransformInfo` bake; `placedTransformInfo` computes `World = parentAccum * Own`, and flat 2D subtrees use a mask-free stamp path. 3D/preserve-3d keeps the coverage-mask path.
- `spliceOwnerWindow` recomputes only the committed op window; `splicePaintOrder` skips the full sort for a neutral, already-sorted window. Overflow clipping visits each fragment once and applies inherited clips to exclusive ops including chrome.
- `PageBreaksEqual` compares every field `PageBreaks` reads today. True is a proof for any page height; false is conservative.

### Docs

- `CHANGELOG.md` adds the landed/designed ledger for Phase 07. `documentation/library-api.md` documents `RelayoutElements`, the scale benchmark, and the block-or-table patch rule. Its line references for `SetAttribute`/`SetText` are stale and it does not mention the new dirty accessors.
- `plans/PR/pr-backdrop-filter-ci.md` is an accidental inclusion: it is the PR body for already-merged PR #33 and was added by a performance commit. Drop it from this branch.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | No-op and inert patches skip the engine; text-free patches skip font loading; batches build one display instead of one per id; paint-only batches skip the placed walk, box rebuild, and break comparison; simple flex rows/stacks skip the noEmit measure build; repeated text measures become map lookups; HTML edits avoid tree walks; owners/order/page breaks are spliced or reused. |
| **Memory** | New process-wide caches are bounded: 64 compiled bundles, 4096 descriptors, 8192 text widths, 4096 rune widths, 1024 line heights, 1024 hover entries / 512 KiB, 4096 grid contributions, 2 pooled subtree stores, 256 flex overrides. HTML id/dirty registries are global and currently unbounded per document (see risks). Display chunk pools are capped (16K boxes, 4K clip rects, 64 buckets, 2K slices). |
| **Behavior / correctness** | Size-changing patches now escalate and can move siblings instead of erroring; table cell/row/inline edits patch through the owning table when the grid does not move; hidden atomic inline boxes keep their box shape; flat 2D transforms stamp without a mask. Every accepting path is compared against a full `Layout` in tests. |
| **API / CLI** | Adds `layout.RelayoutElements`, `layout.Display.Release`, and `html.Document.DirtyIDs`/`DirtyStructural`/`ClearDirty`. `RestampTransforms` still ignores its `Options` argument. No removals. |
| **Dependencies** | None. |
| **Binary size / build time** | More engine code and tests; no new module dependency. No material binary-size claim is made. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| A size-changing `RelayoutElement` that previously returned `errSubtreeResized` may now succeed by widening to a size-stable ancestor. | Hosts that treated the error as "do a full relayout" can keep that fallback, but should not assume a successful patch only moved the named element. |
| A no-op fast-path hit returns `prev` with only `Display` set; `ElementPatch.Ops`, `Boxes`, and `Old`/`New` may be empty. | Do not read patch payload fields unconditionally. Use the display, or fall back to a full redraw when bounds are needed. |
| Table-internal edits that previously errored now patch the owning table. | Re-test damage rectangles: `ElementPatch.Old`/`New` stay the named element's box, which may under-invalidate content that widened inside the table. |
| `RelayoutElements` is all-or-nothing. One bad id aborts every id and leaves `prev` untouched. | Validate ids before batching, or call per element when partial success is required. |
| `Display.Release` returns chunks to pools; released `Ops`/`Boxes` slices are ordinary Go memory and are not zeroed. | Release only when no reader will use the display. A patch display retains its result, so releasing `prev` before the patch is safe. |
| Retained styles, the block-intrinsic memo, batch `SpliceWindow` damage, and op-range translation are not wired. | Do not build host features on them yet. |

---

## Test plan

- [x] `go test -count=1 ./...` (all packages pass on 2026-11-09).
- [x] Focused subtree, batch splice, paint-only, size-escalation, row-span, area-chart damage, and splice-allocation tests.
- [x] Focused element fast-path, table-retarget, batch-parity, display-list emit/patch/release, and scale-benchmark tests.
- [x] Focused style store/chunk/retain/reuse/cross-layout tests, including concurrency cases.
- [x] Focused flex/text/inline/grid/pooling differential tests.
- [x] Focused HTML attribute/dirty/find-index/parse-alloc tests and CSS match/hover-scope tests.
- [ ] `make test TEST_P=1 TEST_PARALLEL=1`
- [ ] `make lint`
- [ ] `make golden`
- [ ] `make claim-scan`

### Commands

```sh
go test ./...
make test TEST_P=1 TEST_PARALLEL=1
make lint
make golden
make claim-scan
```

---

## Screenshots / sample output

No visual change is expected from the engine work. The patch tests assert that a patched display is operation-for-operation equal to a full layout, including a 300-row table and a 1000-row invoice page.

---

## Related issues

- None. Open issue #18 (mask-border) does not overlap this branch.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`enhancement`)
- [x] Related issues checked (none)
- [x] Filled body saved under `plans/PR/`

---

## Follow-ups (out of scope)

- Drop `plans/PR/pr-backdrop-filter-ci.md`; it belongs to merged PR #33.
- Wire or delete the retained-style retainer and `memo_block.go`, and resolve chunk lifetime before wiring retention.
- Consume `SubtreeBatchResult.Windows` in the host repaint path, or remove the dormant damage descriptors.
- Fix the global `elementRegistry`/`dirtyRegistry` maps, which key on `*Document` and prevent collection; add an explicit drop or weak-key scheme.
- Validate the sibling memo with a generation or child identity, not only the parent child count.
- Fix stale comments/tests: `relayout_row_span_test.go` says "EXPECTED TO FAIL"; `TestBatchSpliceRefusalMatchesSingleError` now asserts success; `library-api.md` line references are stale.
- Wire or delete unused pools (`rentBox`, `rentClipRect`, `rentBoxRows`, `rentOpIDs`, `rentPlaced`) and dead helpers (`styleStore.recycle`, `putStyleStoreChunkSlice`).
- Add a paint-only test that changes animation/transition/transform/timeline/clip metadata without moving geometry; `sameBoxPaintInputs` does not compare those fields today.

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] No public removals or signature breaks
- [ ] New rules have fixture coverage
- [ ] PR has assignee and labels
- [ ] Related issues use correct keywords
- [ ] No secrets or generated artifacts committed
- [ ] Diff-stat-by-extension table pasted at the bottom

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
|-----------|------:|-----------:|----------:|
| `.go` | 133 | 26,507 | 1,284 |
| `.md` | 3 | 151 | 2 |
| **Total** | **136** | **26,658** | **1,286** |
