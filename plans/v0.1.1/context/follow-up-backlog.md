# v0.1.1 follow-up backlog

These are candidate tasks after 0.1.1. None is part of the release. They
come from the merged PR bodies, the code comments, the lifetime tests, and
the release scan.

How to use this list with parallel agents:

- Pick items that do not share files. One agent owns one file set.
- Give the agent the anchors and the close condition, not just the title.
- Do not treat a designed-only file as shipped behavior.
- A close condition needs a command that exits 0 or a test that fails
  before the change and passes after it.
- Run `make test`, `make lint`, `make golden`, and `make claim-scan` once
  at integration, not after every micro-fix.

## Patch engine

### PATCH-01 Wire or delete the retained-style engine

- Why: `resolveStylesRetainedWithContext` has only test callers. It is not
  in the layout or patch call path.
- Anchors: `internal/layout/style_cascade.go:2118`,
  `internal/layout/style_retain.go`, `internal/layout/style_retain_test.go`,
  `internal/layout/style_sheets_retain_test.go`.
- Trap: retained entries point into style chunk backings that
  `Result.Release` can recycle. Wiring it without a lifetime fix can serve
  reused memory.
- Close when: a differential test proves retained resolve equals full
  resolve on a patched tree and a production caller uses it, or the code
  and tests are deleted.

### PATCH-02 Wire or delete the block intrinsic memo

- Why: `internal/layout/memo_block.go:8` says "designed, not wired". Only
  `memo_block_test.go` calls it.
- Close when: a production caller has parity coverage and a release
  interaction test, or the file and test are deleted.

### PATCH-03 Expose or remove batch damage windows

- Why: `SubtreeBatchResult.Windows` is produced and asserted by tests, but
  `layout.RelayoutElements` discards it (`layout/relayout_elements.go:82`).
- Anchors: `internal/layout/relayout_subtree.go:2459`,
  `internal/layout/relayout_subtree.go:2461`,
  `internal/layout/batch_splice_test.go`.
- Close when: a public damage API or an internal consumer uses the
  windows with op-range translation and has paint-only plus escalation
  tests, or the field and its tests are removed.

### PATCH-04 Wire or delete unused pools and dead helpers

- Why: `internal/layout/box_alloc.go:34` says the call-site change is not
  wired. `rentBox`, `rentClipRect`, and `rentBoxRows` have no production
  caller. `styleStore.recycle` and `putStyleStoreChunkSlice` are dead.
- Anchors: `internal/layout/box_alloc.go:295`,
  `internal/layout/box_alloc.go:325`,
  `internal/layout/box_alloc.go:390`,
  `internal/layout/style.go:1255`.
- Close when: production callers have allocation tests, or the helpers
  are deleted and `make lint` plus targeted tests pass.

### PATCH-05 Cover paint-only metadata changes

- Why: `sameBoxPaintInputs` does not compare animation, transition,
  transform, timeline, or clip metadata
  (`internal/layout/relayout_subtree.go:1397-1420`). A metadata-only edit
  can take the paint-only path and keep stale metadata.
- Close when: a test changes only those fields, proves geometry is
  unchanged, and asserts the patched display carries the new metadata, or
  the fast path declines and falls back to a full splice.

### PATCH-06 Validate the sibling memo beyond child count

- Why: the sibling memo checks `childLen` only. A same-length child
  replacement can serve stale sibling data.
- Anchors: `internal/css/match.go:84-88`, `internal/css/match.go:183-211`.
- Close when: a generation or child-identity check is in place and a test
  replaces a same-length child and observes a miss plus a rebuild.

### PATCH-07 Fix stale comments and test names

- Why: `internal/layout/relayout_row_span_test.go:9` still says "EXPECTED
  TO FAIL". `TestBatchSpliceRefusalMatchesSingleError`
  (`internal/layout/batch_splice_test.go:197-203`) now asserts success.
  `documentation/library-api.md` cites old line numbers.
- Close when: `rg 'EXPECTED TO FAIL'` is clean, the test name matches its
  assertion, and the docs line references resolve.

### PATCH-08 Define table damage coverage

- Why: a table-internal patch retargets to the owning table, but
  `ElementPatch.Old` and `New` stay the named element's box. Content that
  widened inside the table can sit outside the damage rectangle.
- Anchors: `layout/relayout_element.go:493-540`,
  `internal/layout/relayout_subtree.go:1295`,
  `plans/PR/pr-batched-relayout-and-caches.md:114`.
- Close when: the rule is documented for hosts and a widening-cell test
  proves the host can cover the change, or the public patch carries a
  wider damage window.

### PATCH-09 Honor or document `RestampTransforms` options

- Why: `RestampTransforms` still ignores its `Options` argument.
- Anchors: `layout/reproject.go`, `plans/PR/pr-batched-relayout-and-caches.md:102`.
- Close when: the argument is honored with a test, or the call is
  documented as option-free and the signature is corrected.

### PATCH-10 Wire or remove subtree reuse

- Why: `trySubtreeReuse` is wired only into
  `RelayoutSubtreeWithOverrides`. The public empty-override path delegates
  to `RelayoutElement`, and non-empty overrides always miss, so the path
  is effectively dormant.
- Anchors: `internal/layout/subtree_reuse.go:25-29`,
  `internal/layout/subtree_reuse.go:81-85`,
  `layout/relayout_element.go:228-230`.
- Close when: the public path reaches it with a parity and allocation
  test, or the code is removed.

### PATCH-11 Decide the dirty-set contract

- Why: `html.Document.DirtyIDs`, `DirtyStructural`, and `ClearDirty`
  record edits, but no cascade or layout call reads them.
- Anchors: `html/attribute.go:289`, `internal/html/dirty.go:9`.
- Close when: the retained-style path consumes the set with parity tests,
  or the docs state that the accessors are host-facing metadata only.

## Correctness and render cost

### FIX-01 Repair the outline early exit

- Why: `hasPotentialChrome` requires raw `OutlineWidth > 0`, but CSS gives
  `outline-style: solid` the `medium` width. An element with only a solid
  outline style can skip chrome paint entirely.
- Anchors: `internal/layout/layout_chrome.go:497`,
  `internal/layout/outline.go:38-45`,
  `internal/layout/outline.go:63`.
- Close when: the gate calls `outlinePaints(&sty)` or an equivalent check,
  and a full-layout test with `outline-style: solid` and no width finds the
  outline operation. Add the regression test before the fix.

### PERF-04 Finish measurement rollback

- Why: flex, table, and multicol measurement mark and reset chunks. Grid
  measurement still builds boxes and styles without rollback.
- Anchors: `internal/layout/grid_memo.go:154-163`.
- Close when: the grid measurement path rolls back both chunks and an
  allocation test pins the ceiling.

### PERF-05 Bound the gradient cache and prove 1D filter parity

- Why: the gradient cache is capped by 1024 entries, not bytes. A 1D
  gradient becomes a 1-pixel PNG texture, and an element filter sees that
  texture instead of a full-size one. No parity test exists.
- Anchors: `internal/layout/gradient.go:54`, `:490`, `:527-531`,
  `:569-605`; `internal/layout/background_image.go:158-166`.
- Close when: a byte budget caps the cache, and a filter parity test
  compares the collapsed texture with the full-size path.

### PERF-06 Benchmark deferred chrome containment

- Why: `shiftDeferredChrome` replaced a subtree map with recursive
  `boxContains`. This removes allocations but can become
  entries-times-subtree on large centered flex layouts.
- Anchors: `internal/layout/layout_flow.go:1617`, `:1636`.
- Close when: a large centered flex fixture compares both strategies, and
  the chosen path has a documented crossover.

### PERF-07 Benchmark the capacity heuristics

- Why: `estimateOpCapacity` uses `nodes + nodes/4 + 16`, and the flattened
  box list is sized from chunk count times 64. Partly filled chunks
  over-reserve.
- Anchors: `internal/layout/layout.go:1630-1648`,
  `internal/layout/layout_context.go:68`.
- Close when: invoice, table, and flex benchmarks report growth counts, and
  the heuristic is either kept with evidence or replaced.

### PERF-08 Pool multicol anonymous styles

- Why: `multicol.go` builds an anonymous style by value, so it bypasses the
  style arena used by the flex path.
- Anchors: `internal/layout/multicol.go:220-222`.
- Close when: the multicol path uses `allocStyle` and an allocation test
  shows no regression.

### API-01 Document the `OperationIDs` aliasing rule

- Why: `WalkPlaced` returns `OperationIDs` as sub-slices of one backing
  array. The public `Box.OperationIDs` field is mutable.
- Anchors: `internal/layout/placed.go:281`,
  `layout/displaylist.go:570-609`.
- Close when: the field doc says read-only, or each box gets its own copy
  with a test that a host append cannot affect a later box.

## Cache and memory

### MEM-01 Fix the html document registries

- Why: `elementRegistry` and `dirtyRegistry` use a strong `*Document`
  key. The finalizer in `html/attribute.go:398` cannot run while the map
  holds the document.
- Anchors: `html/attribute.go:246-256`, `html/attribute.go:260-284`,
  `html/attribute.go:375-413`.
- Close when: weak keys or an explicit `Close`/drop path removes the
  entry, and a test calls `Find`, `SetAttribute`, or `SetText`, drops the
  document, runs `runtime.GC()`, and probes a weak pointer.

### MEM-02 Publish the aggregate style pool bound

- Why: `TestStyleChunkPoolMemoryBound` covers only the shared 16-cap pool.
  The 8/32/64 lists and the two pooled subtree stores sit outside it.
- Anchors: `internal/layout/layout_chunks.go:88-101`,
  `internal/layout/style.go:1131-1148`,
  `internal/layout/relayout_subtree.go:607-651`.
- Close when: one test measures the aggregate retained backing and the
  docs state the real bound, or the extra pools are removed.

### MEM-03 Measure 8 versus 32 style chunks

- Why: `cb96a04` raised the cap from 8 to 32. No benchmark or RSS result
  justifies it.
- Anchors: `internal/layout/layout_chunks.go:81`,
  `internal/layout/layout_chunks_test.go:88`.
- Close when: a repeated-redraw benchmark reports retained heap for both
  caps, and the chosen cap is recorded in the code comment and release
  docs.

### MEM-04 Sweep dead weak memo entries

- Why: dead weak keys and values stay in the selector memos until a cap
  evicts them. The attribute memo also keeps original and lowered strings.
- Anchors: `internal/css/match.go:175-239`,
  `internal/css/match.go:684-729`.
- Close when: insert or soft-cap eviction drops nil keys and values, and
  a test proves a dead entry does not survive the next sweep.

### MEM-05 Count the full hover memo key

- Why: `memoBytes` excludes focus, hover, and active strings. The cache
  also retains the last four sheet sets.
- Anchors: `css/hover_scopes.go:14-28`,
  `css/hover_scopes.go:255-303`.
- Close when: every string in the key is counted, the cache evicts by
  sheet set, and the byte-budget test covers the full key.

### MEM-06 Cap or document `gridMeasureCache`

- Why: `trackKidMemo` has a 4096 cap. `gridMeasureCache` has no entry
  cap. It is pass-scoped, but its size follows the document.
- Anchors: `internal/layout/grid_memo.go:127-142`,
  `internal/layout/grid_tracks.go:19-21`.
- Close when: a count or byte cap exists with a test, or the pass-scoped
  bound is documented as the only limit.

### MEM-07 Document `Display.Release` and release order

- Why: the public call has a doc comment but no library API entry. Hosts
  need to know that released slices are not zeroed and that a patch
  display keeps its own result.
- Anchors: `layout/displaylist.go:225-236`,
  `layout/displaylist_release_test.go`.
- Close when: `documentation/library-api.md` and the architecture page
  describe the lifecycle, and a double-release plus release-before-patch
  test is linked.

### MEM-08 Include subtree stores in lifetime coverage

- Why: `subtreeStyleStores` keeps two document payloads on purpose. No
  test drops a document and checks the store age or the intern map.
- Anchors: `internal/layout/relayout_subtree.go:607-651`.
- Close when: a test bounds the store count and intern size across many
  documents, and a dropped document does not keep a live tree.

## Docs and catalog

### DOC-01 Document `HoverChangedScopes`

- Why: the public CSS method has no entry in `documentation/library-api.md`
  or the CSS architecture page.
- Anchors: `css/hover_scopes.go:75-107`,
  `documentation/architecture/06-css.md:515-535`.
- Close when: the docs state the input, the empty and `false` cases, the
  nearest-id rule, and a host fallback example.

### DOC-02 Document the batch and dirty API

- Why: `RelayoutElements` has one sentence. `Display.Release` and the
  dirty accessors are absent.
- Anchors: `documentation/library-api.md:63-68`,
  `documentation/library-api.md:133-149`,
  `documentation/architecture/02-library-api.md:102-114`.
- Close when: the docs give signatures, the all-or-nothing rule, the
  `prev` stale rule, `ErrNoElement` mapping, and the dirty lifecycle.

### DOC-03 Rewrite the patch refusal text

- Why: the docs still say a move or resize is refused. PR #35 escalates
  size changes to an ancestor.
- Anchors: `documentation/library-api.md:97-103`,
  `documentation/library-api.md:114-125`,
  `documentation/architecture/02-library-api.md:98-100`.
- Close when: the docs describe escalation, table retargeting, and the
  remaining structural refusals, with the damage caveat.

### DOC-04 Repair the `wrap-inside` catalog handler

- Why: `properties.json` lists `wrapBreakInsideAvoid`, which was removed
  in `8bce5a1`. `make catalog-check` reports `unknown handler function:
  wrap-inside -> wrapBreakInsideAvoid`.
- Anchors: `testdata/css/catalog/properties.json:20476`,
  `internal/layout/wrap_break.go`.
- Close when: the handler entry names a live function or is removed, and
  `make catalog-check` only reports the seven sibling OwnFrame strings.

### DOC-05 Update the AGENTS file-size bullet

- Why: `AGENTS.md:246-250` says one file is over the limit. The allowlist
  has eight.
- Anchors: `AGENTS.md:246-250`,
  `scripts/file-size-allowlist.txt:9-16`.
- Close when: the bullet lists the eight files and their counts, and
  `make size-check` stays clean.

### DOC-06 Record the catalog-check decision

- Why: `make catalog-check` is red on `master`, but `RELEASE.md` does not
  list it. The two causes are a sibling OwnFrame mismatch and the
  `wrap-inside` drift.
- Anchors: `RELEASE.md:9-15`, `Makefile:119`.
- Close when: `wrap-inside` is fixed, the OwnFrame cases are added in the
  sibling checkout or marked external, and `RELEASE.md` states whether
  catalog-check blocks a release.

## WASM

### WASM-01 Carry `backdropFilter` in the WASM group JSON

- Why: `layout.DisplayGroup.BackdropFilter` exists, but
  `drawGroupJSON` has no field. Browser consumers cannot see the value.
- Anchors: `bindings/wasm/drawing_list.go:242-250`,
  `bindings/wasm/drawing_list.go:716-760`,
  `documentation/library-api.md:360`,
  `documentation/compatibility-matrix.md:622`.
- Close when: the field is added with an `omitempty` tag, a contract test
  asserts it, the schema text is updated, and the release note or catalog
  no longer implies WASM carries it.

## File size

Split each file by responsibility, not by length. Do not reorder or rename
code during a split. After each split, run `go build ./...`, the focused
package tests, and `make size-check`, then update
`scripts/file-size-allowlist.txt`.

### SIZE-01 `internal/layout/relayout_subtree.go` (2,835)

Split target resolution, splice commit, table retarget, batch bookkeeping,
and damage windows into separate same-package files.

### SIZE-02 `internal/layout/layout.go` (2,497)

Split engine setup, group construction, page assembly, and the public
entry points. Keep the `engine` type and its helpers together.

### SIZE-03 `internal/layout/style_cascade.go` (2,289)

Split property application, inherited resolution, retained-style work, and
the cascade entry point.

### SIZE-04 `internal/layout/flex.go` (2,184)

Split flex child collection, row and column placement, and the simple
height fast paths.

### SIZE-05 `internal/layout/inline_paint.go` (2,123)

Split inline item collection, line emission, and paint emission. Keep
`inline_image.go` as the image seam.

### SIZE-06 `internal/layout/transform.go` (2,098)

Split matrix math, stamping, and the flat 2D fast path.

### SIZE-07 `internal/html/tree.go` (2,049)

Split node construction, tree edits, and traversal helpers.

### SIZE-08 `internal/layout/style_value_accept.go` (2,018)

Split value parsing by property family. Keep the acceptance gate in one
place.

## Performance validation

### PERF-01 Fill the invoice benchmark figures

- Why: `CHANGELOG.md` and `documentation/library-api.md` say figures are
  TBD.
- Anchors: `layout/invoice_scale_bench_test.go`,
  `CHANGELOG.md:14-15`.
- Close when: `BenchmarkInvoiceScale` runs on a recorded environment and
  the command, date, and numbers are committed, or the TBD language is
  removed.

### PERF-02 Reproduce or label the OwnFrame example numbers

- Why: PR #31's body records 3,745 ms to 537 ms and 2,134 MB to 160 MB on
  an OwnFrame example. No Blinkless harness reproduces them.
- Anchors: `plans/PR/pr-batched-relayout-and-caches.md`,
  `plans/v0.1.1/release-note/v0.1.1.md`.
- Close when: a committed Blinkless benchmark reproduces the shape of the
  result, or the numbers are labeled downstream-only everywhere they
  appear.

### PERF-03 Add allocation budgets for the patch path

- Why: the paint-only and batch paths have allocation tests, but the
  budgets are not stated next to the claims.
- Anchors: `layout/relayout_elements_paint_only_alloc_test.go`,
  `layout/displaylist_patch_alloc_test.go`,
  `internal/layout/relayout_splice_alloc_test.go`.
- Close when: each budget has headroom, a named ceiling, and a failure
  message that names the path.

## Release process

### REL-01 Add the promotion check that v0.1.0 missed

- Why: `v0.1.0` was tagged without a `CHANGELOG.md` section or a
  `RELEASE.md` version bump. The release PR body said those updates
  "ride the release PR", but they did not land.
- Anchors: `CHANGELOG.md`, `RELEASE.md`,
  `plans/v0.1.0/PR/pr-release-0.1.0.md:27-28`.
- Close when: the release skill or `RELEASE.md` has a check that fails
  when the dated section or version line is missing before the tag.
