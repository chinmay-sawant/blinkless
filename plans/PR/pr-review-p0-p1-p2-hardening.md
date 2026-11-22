# fix(html,layout,ci): resolve P0-P2 review findings across DOM, relayout, caches, and lint

- Base: `master`
- Head: `fix/review-p0-p1-p2-hardening`

---

## Summary

Fixes P0 correctness and memory bugs in `html.Document`, `CloneResult`, blend-group splice windows, and slab payload cloning, resolves P1 stacking-context, clip-path, outline, raster-cache, and reflection-free restyle issues, and upgrades `blinkless` lint, CI, and release documentation hygiene.

---

## Motivation / context

- Plans: `plans/PR/pr-review-p0-p1-p2-hardening.md`
- Code review of `blinkless v0.1.5` exposed four P0 correctness/memory issues (`html.Document` global-map leak, `CloneResult` `styles`/chunk aliasing on aborted batches, blend-group marker leak/orphan bugs, and uncopied `Path.Points` in slab cloning) alongside P1/P2 cache, parity, and CI/lint gaps.
- Issues: none.

---

## Changes

### P0 Fixes
- **Eliminated `html.Document` memory leak (`html/parse.go`, `html/attribute.go`, `html/find_index_test.go`):** Moved `index` and `dirty` state directly onto unexported fields of `html.Document`, removing the global `elementRegistry`/`dirtyRegistry` maps and unreachable finalizer. Added `TestDocumentCollectibleAfterIndexAndDirtyUse`.
- **Isolated `CloneResult` styles and chunk ownership (`internal/layout/layout.go`, `internal/layout/layout_chunks.go`, `layout/relayout_elements.go`, `layout/relayout_batch_isolation_test.go`):** Cloned `res.styles` via `maps.Clone`, decoupled chunk ownership with explicit `TransferChunksTo`, and released clones on aborted batches so failed `RelayoutElements` batches leave `prev` untouched.
- **Fixed blend-group splice bugs (`internal/layout/group_window.go`, `internal/layout/relayout_element.go`, `layout/relayout_group_hover_test.go`):** Wired `relayoutGroupWindow` into `RelayoutSubtreeWithOverrides` and extended `widenGroupWindow` to strip old `Begin`/`End` group markers when a patch removes a blend group.
- **Deep-copied `Path.Points` in `cloneOpExtraInto` (`internal/layout/op_extra.go`, `internal/layout/clone_ops_slab_test.go`):** Matched `detachedExtraCopy` so slab-cloned display paths do not alias `prev.Ops`.

### P1 Fixes
- **Stacking-context parity (`internal/layout/relayout_subtree.go`, `internal/layout/layout.go`, `internal/layout/relayout_ancestor_paint_test.go`):** Replaced `MixBlendMode != blendNormal` checks with `initiatesBlendGroup(style)` in `relayoutEnterAncestorPaint` and `pushZ`, and preserved explicit ancestor `z-index` bands in `enterStackingContext`.
- **Independent ellipse radii & default solid outline (`internal/layout/clip_path.go`, `internal/layout/layout_chrome.go`, `internal/layout/clip_path_ellipse_test.go`, `internal/layout/outline_test.go`):** Resolved `rx` and `ry` independently in `resolveEllipseRadii`, and switched chrome outline guards to `outlinePaints(sty)`.
- **Bounded LRU raster caches & filter alignment (`internal/layout/image_effect_cache.go`, `internal/layout/filter.go`, `internal/layout/image_effect_cache_test.go`):** Replaced cliff-clearing maps with a bounded `byteLRUCache` (256 entries, 512 KiB/entry, 32 MiB total) using trimmed `bytes.Clone` entries and aligned `hasEffect` with implemented filter kinds.
- **Reflection-free `paintOnlyBGDelta` (`scripts/gen-style-intern/main.go`, `internal/layout/style_intern_gen.go`, `internal/layout/paint_restyle.go`):** Generated `styleInternEqualExceptBG(a, b *ResolvedStyle)` to eliminate 11.7 KB struct copies and `reflect.DeepEqual` on hover.

### P2 Improvements
- Switched internal chrome helpers in `internal/layout/layout_chrome.go` and sibling paint files to take `*ResolvedStyle` pointers.
- Removed unwired `resolveStylesRetainedWithContext` (`internal/layout/style_cascade.go`, deleted `style_retain.go`/`style_retain_test.go`) and dead `bench-lib` target in `Makefile`.
- Migrated `.golangci.yml` from `enable-all: true` to a curated `v1.64.8` ruleset, removing blanket `//nolint:all` headers on `internal/layout/style.go`, `internal/html/html.go`, and 15 other files.
- Hardened `.github/workflows/ci.yml` with SHA-pinned actions, least-privilege permissions, and `test-race`, `claim-scan`, `catalog-check`, and `matrix-check` steps.
- Updated `README.md`, `CHANGELOG.md` (`v0.1.2`-`v0.1.5` sections + `Unreleased`), `AGENTS.md`, and `scripts/file-size-allowlist.txt`.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Eliminates `reflect.DeepEqual` + 11.7 KB style copies on paint-only hover restyles and avoids value-copying `ResolvedStyle` in chrome helpers. |
| **Memory** | Fixes unconditional `html.Document` DOM retention leak, caps raster effect caches at 32 MiB with trimmed slices, and prevents double-release chunk pool corruption. |
| **Behavior / correctness** | Fixes aborted-batch state corruption, layout-override blend-group leaks, hover-leave orphaned group markers, `Path.Points` aliasing, ancestor stacking contexts, `ellipse()` independent axes, and `outline-style: solid`. |
| **API / CLI** | No breaking public API changes. |
| **Dependencies** | No new dependencies added (`TestDirectModuleAllowlist` passes). |
| **Binary size / build time** | Net -247 lines of code. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] `make test`
- [x] `make lint`
- [x] `make build` (`CGO_ENABLED=0 go build ./...`)
- [x] `make golden`
- [x] `make claim-scan`
- [x] `make matrix-check`
- [x] `make size-check`
- [x] `go test -race -p 2 -parallel 2 ./html ./layout ./internal/layout ./internal/html ./internal/fonts ./internal/load ./css ./internal/css`

### Commands

```sh
CGO_ENABLED=0 go build ./...
make test
make golden
make lint
make claim-scan
make matrix-check
make size-check
```

---

## Screenshots / sample output

```text
make test: PASS (all packages)
make golden: PASS (./layout TestDisplay)
make lint: PASS (golangci-lint v1.64.8: 0 issues; size-check: clean)
make claim-scan: PASS (claim-scan: clean)
make matrix-check: PASS (matrix-sync: ok)
```

---

## Related issues

- None (internal review hardening wave after `v0.1.5`).

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`bug`, `enhancement`)
- [x] Related issues filled
- [x] Filled body committed under `plans/PR/pr-review-p0-p1-p2-hardening.md`

---

## Follow-ups (out of scope)

- Splitting `ResolvedStyle` into a hot core + lazy rare-property sub-structs.
- Cutting a future `v0.1.6` tag when ready to release.

---

## Reviewer checklist

- [x] Behavior matches summary and test plan
- [x] No unrelated changes in diff
- [x] Public API / CLI changes documented
- [x] New rules have fixture coverage when applicable
- [x] PR has assignee and labels
- [x] Related issues use correct Closes/Relates keywords
- [x] No secrets or generated artifacts committed
- [x] Diff-stat-by-extension table pasted at the bottom

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 57 | 915 | 1154 |
| `.md` | 3 | 85 | 79 |
| `.txt` | 1 | 4 | 4 |
| `.yml` | 2 | 47 | 52 |
| No extension | 1 | 3 | 12 |
| **Total** | **64** | **1054** | **1301** |
