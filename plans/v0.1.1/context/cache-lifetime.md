# Cache ownership and lifetime context

This note covers PR #36 (`780905b`, `cb96a04`, `8e88f3c`, `7772b2f`,
`7bbdaa4`, merge `3e3a8b7`) and the cache-related commits in PR #35
(`6a2f0b2`, `e45a24d`, `ee07764`, `9ab93bc`, `7ef441e`, `dc1d518`).
Anchors are at `3e3a8b7`.

## Inventory

| Cache or pool | Owner and scope | Bound | What it can retain |
|---|---|---|---|
| `sibParents` | Process-wide, `sibMu` (`internal/css/match.go:90-96`) | Soft 2048, drops about a quarter; hard 16384, clears (`match.go:101-109`, `:222-239`) | Weak parent key and weak `sibParentData` value (`:180`, `:211`). A live value holds child nodes and sibling metrics. An idle entry retains nothing. |
| `attrCache` | Process-wide (`internal/css/match.go:674-678`) | Same 2048 and 16384 (`:106-108`, `:706-723`) | Weak node key (`:663-666`, `:685`), attribute name, original and lowered strings. Dead keys and strings stay until eviction. |
| `relayoutFingerprints` | Process-wide (`layout/relayout_element.go:63-80`) | 1024 entries, clear-all on overflow (`:58-61`, `:770-774`) | Weak `*Result` and weak `*css.Document` keys (`:76-80`), the element id, and a fingerprint string that copies subtree text and relevant attribute values. Count is capped; bytes are not. |
| `gridCaches` | One engine, one Layout pass (`internal/layout/grid_memo.go:9-14`, `layout.go:722-725`) | `trackKidMemo` 4096 per pass (`grid_tracks.go:19-21`, `:284-286`); `gridMeasureCache` has no count cap (`grid_memo.go:127-142`) | Node and style pointers plus numbers. `releaseScratch` nils both (`layout.go:901`; called at `layout_context.go:140` and `layout_section.go:143`). |
| Flex style borrow | No cache. Borrows the cascade record (`internal/layout/flex.go:265`, `:281`; `layout.go:2470-2472`; `aspect_ratio.go:5-15`) | One private copy only when aspect-ratio supplies an auto height (`aspect_ratio.go:8-12`) | The copy is allocated in the Result's style chunks (`layout_chunks.go:273-285`). The borrow itself retains nothing extra. |
| `styleChunkPool` | Process-wide (`internal/layout/layout_chunks.go:225-234`) | 32 chunks of 16 records, channel-bounded (`:68-82`, `:184-217`) | Only zeroed backings: `put` clears the full capacity (`:211`) and drops chunks over cap 16 (`:206-209`). Fed by `Result.Release` (`:49-53`), finalize spares (`layout_context.go:83-85`), and override chunks (`layout.go:905-908`). |
| styleStore per-size lists | Process-wide (`internal/layout/style.go:1131-1148`) | 8 x cap-8, 4 x cap-32, 2 x cap-64; the 16-cap step uses `styleChunkPool` (`:1181-1193`, `:1238-1241`) | Zeroed backings only (`:1246-1247`). About 1.79 MiB at full occupancy, outside the 4 MiB test. |
| `subtreeStyleStores` | Process-wide (`internal/layout/relayout_subtree.go:642-651`) | 2 stores; 4 chunks = 8+16+32+64 = 120 records each; intern map bounded at 1024 (`:607-621`) | Chunks and intern map are kept on purpose. Holds up to two documents' last patch payloads. The tag is `uintptr`, so it pins neither document nor sheets (`:623-633`). `ResolvedStyle` has no node pointers. |
| `hoverLists` | Process-wide, last 4 sheet sets (`css/hover_scopes.go:14-37`, `:166-170`) | 4 entries; per entry 1024 keys and 512 KiB counted (`:14-28`, `:287-292`) | Sheet pointers, copied selectors, interned tag and attrs, and focus, hover, and active strings that `memoBytes` does not count (`:255-263`, `:299`). No node keys. |
| `elementRegistry`, `dirtyRegistry` | Process-wide (`html/attribute.go:246-256`) | Unbounded | Strong `*Document` key. The values hold the root, the id map, and the dirty ids. The finalizer is set at `:398` and the delete path is `:405-413`, but the strong key keeps the document reachable, so the finalizer cannot run. `dirtyFor` (`:260-284`) can create an entry with no finalizer. |

## What is proven collectible

These tests populate the path, drop the tree or result, run `runtime.GC()`
three times, and fail if a `weak.Pointer` probe still resolves.

- Selector sibling and attribute memos: `internal/css/match_lifetime_test.go:12`.
  Dead-entry accumulation is separately bounded at
  `internal/css/match_sib_evict_test.go:147` (map stays at or below 2048
  after 3200 dropped trees); eviction and rebuild at `:51` and `:79`.
- Grid pass memo: `internal/layout/grid_lifetime_test.go:12` first asserts
  the track-intrinsic counters moved, then probes the released tree.
- Fingerprint document and result: `layout/relayout_element_fast_test.go:20`.
  It reaches `RelayoutElement` at `:26`, so the cache is populated.
  `TestReleasedLayoutResultIsCollectible` at `:47` covers a plain release.
- Bounds and correctness, not collection: `layout_chunks_test.go:10`, `:62`,
  `:88`; `style_store_chunk_test.go:68`, `:122`, `:148`;
  `style_chunk_reuse_test.go:70`; `relayout_creep_test.go:31` (zero style
  growth from patch 10 to 50); hover caps at `css/hover_memo_budget_test.go:39`
  and `css/hover_scopes_intern_test.go:224`.
- Flex borrow: `internal/layout/flex_style_pointer_test.go:8` proves the
  shared cascade record is unchanged. It does not prove collection.

Not covered: no test drops a public `html.Document` after `Find`,
`SetAttribute`, or `SetText`. The lifetime tests use `internal/html` nodes
or a public document that never touches those methods, so they do not
exercise the registries.

## Style free list tradeoff

At the documented 5,872 bytes per `ResolvedStyle`, one 16-record chunk is
93,952 bytes (91.75 KiB). The old cap of 8 retained 751,616 bytes (733 KiB).
The cap of 32 retains 3,006,464 bytes (2.87 MiB), so the change adds
2,254,848 bytes (2.15 MiB) of idle capacity.

`TestStyleChunkPoolMemoryBound` (`internal/layout/layout_chunks_test.go:88-101`)
computes `unsafe.Sizeof(ResolvedStyle{}) * 16 * cap(styleChunkPool.free)`
against `4 << 20` (4,194,304 bytes), leaving 1,187,840 bytes (1.13 MiB).
This guards the shared 16-cap pool only. It does not measure the per-size
8, 32, and 64 lists, the two pooled subtree stores, or map overhead.

Commit `cb96a04` and the PR body say the relayout path does not feed this
free list. The code disagrees. A patch `styleStore` takes its 16-cap step
from `styleChunkPool` (`internal/layout/style.go:1181-1193`) and returns
16-cap store chunks there when a store is dropped or its Result is released
(`style.go:1238-1241`, `layout_chunks.go:61-65`). No benchmark or RSS
result justifies 32 over 8; `dc1d518` had lowered it to 8.

## Remaining retention risks

- The html registries are the main gap. Every document passed to `Find`,
  `SetAttribute`, or `SetText` gets a permanent `elementRegistry` entry and
  usually a `dirtyRegistry` entry. The strong key prevents the finalizer.
  `dirtyFor` alone creates a dirty entry with no finalizer. Neither map has
  a cap.
- Dead weak entries are not swept. `sibParents` and `attrCache` keep dead
  keys. `attrCache` also keeps the original and lowered strings.
  `relayoutFingerprints` keeps the element id and a fingerprint that can be
  large.
- The hover cache retains the last four sheet sets and their full hover
  selector lists process-wide. The 512 KiB budget excludes focus, hover,
  and active strings, which are part of the key.
- The 4 MiB test is not a total style memory bound. Per-size lists (about
  1.79 MiB), the two subtree stores (about 1.34 MiB), and map overhead sit
  outside it. If every pool fills, the style backing pools alone total about
  6 MiB (1,072 records at 5,872 bytes).
- `gridMeasureCache` has no entry cap. It is bounded only by the work in
  one pass.
- `weak.Pointer.Value` is not guaranteed to become nil, per the `weak`
  package docs. The tests use three GCs as an empirical guard, not a
  language guarantee.
- The sibling memo validates only `childLen`. A same-length child
  replacement can serve stale sibling data. PR #35's follow-up list already
  calls this out.

## Key anchors

- `internal/css/match.go:93`, `:180`, `:211`, `:222-239`, `:664`, `:685`,
  `:706-723`.
- `css/hover_scopes.go:14-28`, `:51-59`, `:166-170`, `:255-303`, `:313-329`.
- `internal/layout/grid_memo.go:9-14`, `:127-142`;
  `internal/layout/grid_tracks.go:21`, `:263-288`;
  `internal/layout/layout.go:722-725`, `:890-901`.
- `internal/layout/layout_chunks.go:68-82`, `:174-234`, `:273-285`;
  `internal/layout/style.go:1131-1148`, `:1175-1253`, `:1277-1350`.
- `internal/layout/relayout_subtree.go:607-621`, `:623-651`, `:679-779`.
- `layout/relayout_element.go:58-80`, `:752-775`;
  `internal/html/dirty.go:9-18`, `:58-68`;
  `html/attribute.go:246-256`, `:260-284`, `:375-413`.
- `documentation/architecture/07-layout.md:218-250` documents the cache
  lifetimes.

## Candidate follow-ups

1. Move `elementRegistry` and `dirtyRegistry` to weak keys, or add an
   explicit `Document.Close` or drop. Install a cleanup for dirty-only
   documents. Add a test that calls `SetAttribute` or `Find`, drops the
   document, and probes it (`html/attribute.go:246-256`, `:374-413`).
2. Sweep entries whose weak key or value is nil at insert or at the soft
   cap (`internal/css/match.go:175-239`, `:684-729`).
3. Add one aggregate retention test across `styleChunkPool`, the 8, 32,
   and 64 lists, and `subtreeStyleStores`, and publish the combined bound
   (`internal/layout/layout_chunks.go:88-101`, `style.go:1131-1148`,
   `relayout_subtree.go:607-621`).
4. Benchmark 8 versus 32 chunks and record RSS on the repeated-redraw path.
   The current change is a policy choice, not a measured win.
5. Evict hover entries by sheet set and count focus, hover, and active
   strings in `memoBytes` (`css/hover_scopes.go:14-28`, `:255-303`).
6. Validate `sibParentData` with child identity or a generation, not only
   `childLen` (`internal/css/match.go:84-88`, `:188`, `:198`).
7. Delete the unused `styleStore.recycle` and `putStyleStoreChunkSlice` if
   they remain dead (`internal/layout/style.go:1255-1275`).

## Do not claim

- "All caches are bounded." The html registries are unbounded and keyed by
  `*Document`.
- "Dropped documents are always collectible." Any document that used
  `Find`, `SetAttribute`, or `SetText` is pinned, and the finalizer cannot
  run.
- "Weak entries disappear promptly." Dead keys and strings stay until a cap
  evicts them.
- "4 MiB is the total style pool bound." It covers only `styleChunkPool`.
  The per-size lists and subtree stores are outside it.
- "The relayout path does not feed the style free list." `styleStore` uses
  the shared pool for its 16-cap step and returns there on release.
- "The grid memo is capped." `gridMeasureCache` has no entry cap; only
  `trackKidMemo` has one.
- "The hover memo holds no page data." It retains the last four sheet sets
  and their hover selectors.
- "The 32-chunk cap is a measured performance win." No benchmark or RSS
  measurement is recorded.
- "Collection is guaranteed." The tests use three GCs against
  `weak.Pointer`, whose `Value` may not become nil by contract.
