# fix(layout): stop caches pinning old trees and borrow flex styles

## Summary

The layout and selector caches kept released DOM trees, CSS documents, and layout results reachable, and every flex container copied a large resolved style to the heap. This PR makes cache ownership weak or pass-scoped, borrows the cascade style in flex layout, and raises the bounded style free list from 8 to 32 chunks so successive full layouts reuse backing. Layout output is unchanged.

---

## Motivation / context

- Base: `master` at `d805508`. The PR adds four commits on `fix/layout-allocation-and-cache-lifetimes`.
- Plans: no file under `plans/` is dedicated to this change. PR #35's follow-up list in `plans/PR/pr-batched-relayout-and-caches.md` called out cache ownership and chunk lifetime. This branch handles the layout-side caches and the style free list; the `html/attribute.go` id and dirty registries remain a separate item.
- Issues: none. Open issue #18 covers mask-border frame regions and does not overlap this work. Prior work: #35 introduced the bounded memos and style chunk pool.
- Downstream: ownframe's Loglens dashboard repeats redraws. That path is where the per-container style copy and the retained old trees showed up.
- This branch builds standalone. It has no `go.work`, no `replace` directive, and no sibling-checkout requirement.

---

## Changes

### Flex layout allocation

- `internal/layout/flex.go`: `buildFlex` takes `*ResolvedStyle` instead of a struct value. `internal/layout/layout.go` passes the cascade pointer from `buildInFlowDisplay`, so a flex container no longer copies the full `ResolvedStyle` at the build entry.
- `internal/layout/aspect_ratio.go`: `flexAspectRatioStyle` returns the borrowed record unless `aspect-ratio` must supply an auto height. Only that branch calls `allocStyle`, sets `Height` on the private copy, and uses the copy for the build.
- The selected flow function and the later height, deferred-child, and chrome calls still take the style by value. This change removes the heap-escaped build-entry copy and the aspect-ratio copy that happened even when no auto height was needed, not those call copies.
- `TestFlexBuildKeepsSharedStyleImmutable` lays out five flex shapes (row with aspect-ratio, `max-content`, `-webkit-box`, column, vertical writing) and compares the resolved style before and after layout. The shared cascade record must not change.

### Cache ownership and release

- Grid memo: `internal/layout/grid_tracks.go` removes the package-global `trackKidMemo` and `trackKidMemoMu`. `internal/layout/grid_memo.go` adds a `gridCaches` struct holding `gridMeasureCache` and `trackKidMemo`; `internal/layout/layout.go` embeds it on the engine and clears both in `releaseScratch`. Nested measurements in one pass still hit, and entries cannot survive the pass.
- Selector sibling memo: `internal/css/match.go` keys `sibParents` by `weak.Pointer[html.Node]` and stores `weak.Pointer[sibParentData]` values. Both sides are weak because `sibParentData` holds child nodes and element metrics; weakening only the key would still retain the tree. The 2048 soft cap and 16384 hard cap stay, and a collected entry rebuilds from the live tree.
- Selector attribute memo: `attrCache` keys use `weak.Pointer[html.Node]`. Dead entries can hold their original and lowered strings until eviction; the existing soft and hard caps bound that. A miss recomputes from the live attribute.
- Element-patch fingerprint memo: `layout/relayout_element.go` changes `relayoutFingerprintKey` to weak `*ilayout.Result` and weak `*css.Document` fields. The 1024-entry cap and clear-on-overflow behavior stay. A miss after garbage collection falls through to the normal patch path, so the cache only changes the no-op fast path's hit rate.
- `TestSelectorMemosDoNotKeepDroppedTreesAlive`, `TestGridIntrinsicsDoNotKeepReleasedTreesAlive`, `TestFingerprintCacheDoesNotRetainReleasedDocument`, and `TestReleasedLayoutResultIsCollectible` populate the relevant cache or layout path, drop the tree or result, run `runtime.GC()` three times, and assert the weak probe is nil. The fingerprint test now calls `RelayoutElement` and probes both the document and the layout result, so it fails against the old strong result key. The grid test also requires the fixture to exercise the track-intrinsic memo.
- `TestSiblingEntryEvictedParentRebuilds` was updated for the weak key.

### Style chunk retention

- `internal/layout/layout_chunks.go`: `maxStyleChunksRetained` moves from 8 to 32. A chunk is 16 `ResolvedStyle` records, about 93 KiB at the documented 5,872-byte record size, so at full retention the shared free list moves from about 750 KiB to about 2.9 MiB. That stays under the existing 4 MiB test bound in `TestStyleChunkPoolMemoryBound` with about 1.1 MiB of headroom.
- The larger working set lets successive full layouts reuse chunks instead of allocating new backing after each result is released. The relayout path does not feed this free list.
- `dc1d518` deliberately lowered this cap from 32 to 8 to cut idle retention. This PR restores 32 and records the memory tradeoff in the code comment and architecture guide. No benchmark or RSS measurement justifies the increase; it is a bounded reuse policy, not a measured win.

### Documentation

- `documentation/architecture/07-layout.md` adds a "Cache lifetime during interactive layout" section. It records the pass-scoped grid caches, the flex borrow rule, the weak memo semantics and caps, and the 32-chunk style free list.
- `documentation/architecture/02-library-api.md` corrects the stale `relayout_element.go` line reference.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | `buildFlex` no longer copies the 5,872-byte style record at entry, and the aspect-ratio path copies only when it must fill an auto height. The style free list retains up to 32 chunks instead of 8, so successive full layouts reuse more backing. No benchmark is recorded in this PR, so no end-to-end speedup is claimed. |
| **Memory** | Weak keys let garbage collection reclaim released DOM trees, CSS documents, and layout results after the caches are populated. The grid memo is cleared when the layout pass ends. The style free list grows from about 750 KiB to about 2.9 MiB at full retention, still under the 4 MiB test bound. No RSS measurement is included. No runtime memory cap or idle forced release is added. |
| **Behavior / correctness** | No drawing-list or fixture change. A weak memo miss recomputes from live state, so correctness does not depend on a hit. `TestFlexBuildKeepsSharedStyleImmutable` pins that borrowed cascade records are not mutated. Existing grid memo differential tests still cover cached and uncached output. |
| **API / CLI** | None. No exported function, type, or command changes. |
| **Dependencies** | None. `weak` is in the standard library and `go.mod` is unchanged. |
| **Binary size / build time** | Not measured in this PR. The change adds no dependency and removes a global map and mutex. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | No migration. Cache hits are best-effort; a miss runs the existing path. |

No exported API, CLI, or drawing-list format changes.

---

## Downstream note (ownframe)

The ownframe branch `fix/loglens-performance-and-profiling` depends on this work. Its new `TestNavigationRetainedHeapPlateaus` fails against blinkless `d805508` and passes against this branch's head. After this merges, ownframe should point both module requirements at the merged `master` revision and keep its workspace limited to its own two modules. Until then, the ownframe branch pins the branch head and must not merge before this engine work lands.

---

## Test plan

- [x] `make test GO_TEST_FLAGS='-count=1'` (full, uncached)
- [x] `make lint` (golangci-lint v1.64.8 and `size-check`)
- [x] `make golden`
- [x] `make claim-scan`
- [x] `make build`
- [x] `make matrix-check`
- [x] `go test -race -count=1 -p 2 -parallel 2 ./internal/css/... ./internal/layout/... ./layout/...`
- [x] `go vet -p 2 ./internal/css ./internal/layout ./layout`
- [x] `gofmt -l` on changed Go files and `git diff --check`
- [x] New lifetime tests for selector memos, grid tracks, the fingerprint document key, the fingerprint result key, and released layout results
- [x] `TestFlexBuildKeepsSharedStyleImmutable`
- [x] `TestStyleChunkPoolMemoryBound` for the 32-chunk cap
- [x] `GOOS=js GOARCH=wasm go build ./bindings/wasm`
- [ ] `make catalog-check` is red on `master` with `unknown handler function: wrap-inside -> wrapBreakInsideAvoid`. The failure is pre-existing, outside this diff, and not part of PR CI.

### Commands

```sh
make test GO_TEST_FLAGS='-count=1'
make lint
make golden
make claim-scan
make build
make matrix-check

go test -race -count=1 -p 2 -parallel 2 ./internal/css/... ./internal/layout/... ./layout/...
go test ./layout -run 'TestFingerprintCacheDoesNotRetainReleasedDocument|TestReleasedLayoutResultIsCollectible'
go test ./internal/layout -run 'TestFlexBuildKeepsSharedStyleImmutable|TestGridIntrinsicsDoNotKeepReleasedTreesAlive|TestStyleChunkPoolMemoryBound'
```

---

## Screenshots / sample output

```text
No visual surface changed. This branch has no page raster, screenshot, or golden fixture change.
```

The new collection tests populate the relevant cache or layout path, drop the tree or layout result, call `runtime.GC()` three times, and fail if a weak probe still resolves. The fingerprint test now reaches `relayoutStoreFingerprint` through `RelayoutElement`; the old test did not, so it could not catch a strong result key. The flex test compares a resolved style before and after layout. `make golden` passes, so the drawing lists are unchanged.

---

## Related issues

- None.
- Prior work: #35 (merged) introduced the bounded memos and style chunk pool. This branch makes those memos non-owning and restores the larger chunk cap.
- Open issue #18 (mask-border frame regions) does not overlap.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`bug`, `documentation`)
- [x] Related issues checked: none matches this branch
- [x] Filled body committed under `plans/PR/pr-layout-allocation-and-cache-lifetimes.md`

---

## Follow-ups (out of scope)

- The `html/attribute.go` `elementRegistry` and `dirtyRegistry` maps still key on `*Document`. The registry key keeps the document reachable, so the finalizer that would drop the entries does not run. Move them to weak keys or add an explicit drop.
- Add a benchmark or retained-heap measurement for the Loglens redraw path, and decide whether 32 style chunks is the right working set or should stay at 8.
- Consider a sweep that deletes selector memo entries whose weak value is nil, so dead keys do not sit in the map until a cap evicts them.
- Fix the pre-existing `make catalog-check` `wrap-inside` handler drift separately.
- No runtime memory limit, idle GC trigger, or forced release call is added. Collection stays GC-driven.

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented (none)
- [ ] New rules have fixture coverage when applicable (not applicable: these are lifetime tests)
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords (none)
- [ ] No secrets or generated artifacts committed
- [ ] Diff-stat-by-extension table pasted at the bottom
- [ ] Weak memo misses recompute from live state and never return stale data
- [ ] Flex layout does not mutate the borrowed cascade style
- [ ] The style free list tradeoff is acceptable at 32 chunks

---

## Diff stat by extension

Branch `fix/layout-allocation-and-cache-lifetimes`, base `master` (`d805508`), 5 commits.

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 14 | 234 | 65 |
| `.md` | 3 | 209 | 1 |
| **Total** | **17** | **443** | **66** |
