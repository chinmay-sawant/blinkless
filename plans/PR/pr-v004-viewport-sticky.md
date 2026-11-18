# feat(layout): add viewport sticky placement and height media queries

## Summary

- Adds `position: sticky` for pages that scroll in the viewport. `layout.StickyView` returns a copy of the drawing list with sticky boxes moved to where they sit after a scroll, and `layout.HasViewportSticky` says whether a page needs that at all.
- Re-checks `height` media conditions when the viewport changes, and speeds up flex layout (skip sorting when order is already right) and line-height lookups (a cache).
- Merges `master` into this branch and fixes the lint, matrix and catalog drift the merge exposed, so the branch passes the same gates as `master`.

---

## Motivation / context

- The `ownframe` v0.0.4 branch (`feature/v0.0.4-interaction-and-rendering-plans`) pins this branch. It calls `layout.StickyView` and `layout.HasViewportSticky`, and does not compile against `master`, which has neither.
- This branch was cut before the v0.1.2 work landed. `master` is 31 commits ahead of it, and both sides changed the placed-element walk in `internal/layout/placed.go`.
- Plans: `documentation/scrolling.md` (new page on viewport scrolling and sticky boxes).
- Issues: see **Related issues**.

---

## Changes

### Sticky placement

- `layout/sticky.go` adds `StickyConstraint`, `HasViewportSticky` (line 40) and `StickyView` (line 70). Layout stays unchanged and immutable; the sticky view is computed from it for a given scroll offset and viewport size.
- `internal/layout/sticky_placed.go` finds each sticky box's insets and its containing block (`placedSticky`, line 54). A sticky box stops at the edge of its containing block, so it scrolls away with it. Nested scrolling, nested sticky boxes and transformed or fixed ancestors are marked as not viewport sticky.
- `PlacedElement` gains `Sticky` and `Fixed` fields so a host can see which boxes pin.

### Height media queries

- `internal/css/media.go` and `internal/css/container.go` re-check height conditions when the viewport size changes. Container queries stay width-only.

### Speed

- `internal/layout/flex_items.go` skips sorting flex items that already have the right order.
- `internal/layout/line_height.go` caches line-height results, with tests for allocation and concurrent use.

### Merge with master

- The placed-element walk in `internal/layout/placed.go` now carries two things down the tree: the sticky scroll context (this branch) and the timeline scroller (master). `placedElement` builds one element from both (line 194).
- `internal/layout/sticky_placed.go` uses the existing `htmlBodyName` constant instead of the `"body"` string, and `placedElement` sets `Declared` explicitly, to satisfy `goconst` and `exhaustruct`.
- `testdata/css/catalog/properties.json` lists `sticky` for `position`. This branch had edited only the generated `documentation/compatibility-matrix.md`, so the two disagreed. The matrix is regenerated from the catalog.
- `scripts/file-size-allowlist.txt` line counts are recomputed for the merged files.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Flex layout skips a sort when items are already ordered, and line-height lookups are cached. No benchmark was run for this PR. |
| **Memory** | The line-height cache holds entries for reuse. Not measured. |
| **Behavior / correctness** | Pages without `position: sticky` lay out and paint as before. Height media conditions now follow the viewport size. |
| **API / CLI** | Adds `layout.StickyView`, `layout.HasViewportSticky`, `layout.StickyConstraint` and the `Sticky` and `Fixed` fields on `PlacedElement`. Nothing removed. |
| **Dependencies** | None added. |
| **Binary size / build time** | Not measured. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] `make test`
- [x] `make lint` (including `size-check`)
- [x] `make catalog-check`
- [x] `make matrix-check`
- [x] `make claim-scan`
- [x] `make golden`

Also checked from the `ownframe` repository, built against this branch (commit `2c9d915`):

- `make build`, `make test`, `make lint`, `make docs-check plans-check examples-check` all pass.
- `make graphics-check` passes, including the sticky pixel test `TestStickyViewportPixelsAndHits`.
- The `examples/blinkless-all` window was driven by hand on all six tabs with no mismatch.

### Commands

```sh
make test
make lint
make catalog-check
make matrix-check
make claim-scan
make golden
```

---

## Screenshots / sample output

No visual surface changed in this repository. Sticky pixels are checked in the `ownframe` graphics gate named above.

---

## Related issues

- None.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`enhancement`)
- [x] Related issues filled (none)
- [x] Filled body committed under `plans/PR/pr-v004-viewport-sticky.md`

---

## Follow-ups (out of scope)

- After this merges, re-pin `ownframe` to the merge commit on `master` instead of this branch.
- Nested overflow scrolling is not supported by sticky placement (see `documentation/scrolling.md`).

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 18 | 1062 | 168 |
| `.json` | 1 | 1 | 1 |
| `.md` | 6 | 188 | 4 |
| `.txt` | 1 | 1 | 1 |
| **Total** | **26** | **1252** | **174** |
