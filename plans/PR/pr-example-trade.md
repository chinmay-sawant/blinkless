## Summary

Four rounds of engine-only layout work driven by an example trade-terminal profile: per-run memos, fast paths, and allocation cuts lower first-launch and navigation layout cost (news redraw down about 32-42%, orders fixture about 31% faster, about 14% fewer allocs on a flex-heavy bench), while flex, abspos, and overflow geometry fixes correct the marquee loop width, sheen stretch height, pill intrinsic width, and transformed overflow clipping. Everything is generic: caches key on node, style, and size with no page-specific branches, and geometry changes cite the spec section they implement.

## Motivation / context

- An example trade terminal profiled first Redraw at about 1.2s of layout time for roughly 2000 boxes with about 1.4GB of allocs at 1600x900, and 100-340ms of pure layout per screen navigation at 1920x1080, with the news and markets targets slowest.
- The same profile showed the marquee and sheen animations visually broken (wrong loop point, zero-height sweeps) and several small badges and pills mis-measured.
- Each round keeps rendering unchanged by construction: cache keys cover full inputs, every skip implies the full walk would find nothing, and geometry fixes are covered by new layout regression tests.
- Issues: see **Related issues**.

## Changes

### Area 1: Per-run layout memos and fast paths

- `internal/layout/grid_memo.go`, `grid_tracks.go`: memoize preferred-height measurement per node plus containing width within one Layout run; `layoutStandardGrid` places cells directly through `placeGridItemsLocked` when rows are locked, skipping the discarded noEmit measure pass.
- `internal/layout/flex.go`, `flex_base_cache_test.go`: memoize column flex base heights on the hypothetical-measurement path only, plus a per-run memo for `flexMinCrossMainSize` floor queries; grow, shrink, and clamp logic untouched.
- `internal/layout/layout_svg.go`, `svg_raster_cache.go`, `internal/svg/raster.go`: engine-local inline-SVG raster cache keyed by serialized bytes plus maxSide so duplicate icons rasterize once per run; cap icon supersample at 2x, use BestSpeed encoding at or above 128x128px, and share the encoder helper on the mask path.
- `internal/layout/pseudo_content.go`, `counter.go`: memoize the winning `content` declaration and the counter/quote environment per (element, pseudo-element) for one run (environments served as clones, since evaluation mutates quote depth); skip pseudo probes when the rule index shows no matching content rule and reuse one pseudo style context.
- `internal/layout/style_cascade.go`, `style_rule_index.go`, `internal/css/match.go`: per-pass match cache replaying signature-pure selector hits in order, memoized node-independent media/supports gates, and interned namespaced tag per cascade pass (`matchTag`).
- `internal/fonts/advance_cache.go`, `fonts.go`, `internal/layout/text_measure_cache.go`: per-face glyph advance cache plus per-run string and rune width caches; computation unchanged so widths are identical.
- Measure/paint fast paths: skip `mask-image` parsing for `none`, skip OpenType feature/language string building with no shaping fields set, and skip the color-adjust sweep outside standard schemes.

### Area 2: Allocation cuts

- `internal/layout/layout_chrome.go`, `outline.go`: border and outline assembly append into one reused backing array with a single pre-growth step per box, instead of leaving chained discarded growth backings.
- `internal/layout/layout_context_styles.go`, `layout_section.go`: op slice setup shares the `workspaceOps` helper, and `assembleNodeResult` calls `eng.releaseScratch` so a finished run drops unadopted per-run caches instead of pooling them across runs.
- `internal/layout/layout_flow.go`: `markBoxSubtree` pre-sizes its set with one counting walk (`countSubtreeBoxes`).
- `internal/layout/layout_svg.go`: the shared serialize pool drops oversized buffers for the GC (`maxPooledSerializeBytes`, 64 KiB); pooled SVG serialize buffers cut serialize bytes about 91% and allocs about 94%.
- `internal/layout/style_cascade.go`, `layout_measure.go`, `flex_lines.go`, `inline_collect.go`: pass `*ResolvedStyle` through hot per-item paths (the struct is about 6KB), single-copy override pattern, and scratch reuse in flex line breaking.

### Area 3: Flex, abspos, and overflow geometry fixes

- `internal/layout/flex.go` `flexIntrinsicWidth`: count anonymous text runs as items mirroring `flexChildren`, with gaps across all items, and skip out-of-flow children, so a shrink-wrapped pill measures its true max-content width.
- `internal/layout/flex.go` `flexItemBaseWidth`: no longer add the item's own margins into a value the layout treats as border-box, since placement adds margins again.
- `internal/layout/flex.go` lane clamp: max-content items may overflow the lane instead of clamping to it, so the status tape measures its true doubled width and the `translateX(-50%)` marquee loop point is right.
- `internal/layout/layout.go` `absoluteBuildWidth`: clamp shrink-to-fit abspos boxes by min/max-width per CSS 2.1 section 10.4.
- `internal/layout/layout.go` `buildFlex` partition plus `flowAbsCB`: abspos children of flex containers flush through the same out-of-flow path as blocks instead of consuming line space; a `top:0` plus `bottom:0` box with auto height synthesizes its used height so sheen sweeps paint.
- `internal/layout/border_radius.go` `snapRadiiXY`: collapse sub-1e-9 rx/ry splits so a dusty flex width cannot flip a 3px circle into an ellipse.
- `internal/layout/overflow_clip.go`, `placed.go`, `op_extra.go`: record the accumulated ancestor overflow box on each element, add `SetOverflowClip`/`OverflowClip` on the display op, and stamp the clip where overflow clipping already runs; hosts painting through a transform intersect the paint with it while static paint keeps the baked geometry.
- `documentation/architecture/10-imageout-svg.md`: document the encoder helper sharing.

### Area 4: Tests

- New: `flex_base_cache_test.go`, `grid_memo_test.go`, `pseudo_content_cache_test.go`, `counter_env_cache_test.go`, `style_match_cache_test.go`, `svg_inline_cache_test.go`, `svg_raster_cache_test.go`, `overflow_clip_placed_test.go`, `internal/css/match_element_local_test.go`, `internal/svg/raster_perf_test.go`, `layout/flex_absolute_test.go`, `layout/flex_intrinsic_test.go`, `layout/absolute_stretch_test.go` — covering the marquee width, abspos stretch, flex overlay, pill intrinsic width, cache isolation, and placed overflow records.
- Updated: `css_partial_remaining_test.go`, `style_advanced_props_test.go` expectations for the new memo/fast paths.

### Area 5: CI-driven refactors and file splits

- No behavior change: every refactor below is a pure move or a behavior-preserving
  extraction, verified by the unchanged layout, css, fonts, svg, and golden suites.
- Complexity budget: `buildFlex`, `flexIntrinsicWidth`, `flexClampMainWidthsInto`,
  `scanMatches`, `layoutBlockChild`, `buildOutOfFlow`, `shiftOutOfFlowBox`,
  `layoutStandardGrid`, `mayMatchPseudo`, and `styleHasShapingFeatures` each shed a
  decision branch into a named helper (`partitionFlexOutOfFlow`,
  `resolveFlexContentHeight`, `flushFlexDeferredOutOfFlow`,
  `accumulateIntrinsicElementWidth`, `applyIntrinsicGapsAndMax`,
  `redistributeClampedWidths`, `storeMatchEntry`, `marginTrimForChild`,
  `insetImageMargins`, `finishBlockChild`, `absStretchUsedHeight`,
  `shiftDeferredFrom`, `gridUsedHeightFromRows`, `pseudoBucketMatchesClass`,
  `fontVariantShapingSet`); dead wrapper `flexClampMainWidths` deleted.
- Lint hygiene: pseudo-class link/visited/checked literals folded into the shared
  `pseudoClass*` constants; `cell` test comparisons use the existing
  `tableCellKind`; the pseudo-element replay test uses black/white so no color
  literal repeats across the package; short test locals renamed; `t.Context()`
  replaces `context.Background()`; `//nolint` directives that the refactors made
  stale removed.
- File splits per the ~2,000-line soft limit (no behavior change, new files are
  same-package seams): intrinsic flex measurement to `flex_intrinsic.go`,
  match-cache machinery to `style_match_cache.go`, engine face resolution to
  `layout_faces.go`, and rune measurement beside its cache in
  `text_measure_cache.go`. `scripts/file-size-allowlist.txt` records the new
  `layout.go` count (2133, down from 2197).

## Impact

| Area | Impact |
|------|--------|
| **Performance** | News redraw down about 32% at 1080p / 42% at 900p (pixel-identical); orders fixture layout about 31% faster with identical ops; about 14% fewer allocs on a flex-heavy bench; about 4% less per-layout alloc from measure/paint fast paths. Refactors in Area 5 add no new work to the hot path. |
| **Memory** | Per-run memos live for one Layout run and are released after; serialize pool caps pooled buffers at 64 KiB so megabyte-scale documents fall through to the GC. |
| **Behavior / correctness** | Marquee loop point, sheen stretch height, pill/badge intrinsic widths, flex abspos flushing, radii snapping, and transformed overflow clipping corrected; all covered by layout regression tests. |
| **API / CLI** | Additive: `PlacedElement`/public `Box` overflow record and `SetOverflowClip`/`OverflowClip` on the display op. No CLI in this tree. |
| **Dependencies** | None. |
| **Binary size / build time** | No new dependencies; new cache files are small engine-local tables. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

## Test plan

- [x] `gofmt -l` on touched trees (clean)
- [x] `go vet` on `internal/layout`, `internal/css`, `internal/fonts`, `internal/svg` (clean)
- [x] `go test ./internal/css/ ./internal/fonts/ ./internal/svg/ ./internal/layout/ ./layout/` (pass)
- [x] `go test ./...` full tree after the refactors and file splits (pass, no failures)
- [x] `golangci-lint run ./...` full tree (clean) plus `make size-check` (clean)
- [x] `make build` (pass)
- [x] `make golden` equivalent: `go test ./layout/ -run TestDisplay` (pass)
- [ ] `make test` (CI)
- [ ] `make lint` (CI)

### Commands

```sh
make test
make lint
make golden
```

## Screenshots / sample output

```
ok  github.com/chinmay-sawant/blinkless/internal/css
ok  github.com/chinmay-sawant/blinkless/internal/fonts
ok  github.com/chinmay-sawant/blinkless/internal/svg
ok  github.com/chinmay-sawant/blinkless/internal/layout
ok  github.com/chinmay-sawant/blinkless/layout
go test ./layout/ -run TestDisplay  # pass
```

No visual surface changed by construction (pixel-identical redraws); geometry fixes are covered by layout regression tests.

## Related issues

None. This branch does not close an open issue (the only open issue, #18 mask-border regions, is unrelated).

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/PR/pr-example-trade.md` when process-gated

## Follow-ups (out of scope)

- Downstream example-terminal re-profile after the pin bump to confirm end-to-end navigation wins.
- Further alloc work on remaining per-box temporaries called out in the trade-terminal profiles.

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
- [ ] Diff-stat-by-extension table pasted at the bottom

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 62 | 4773 | 986 |
| `.md` | 2 | 167 | 3 |
| `.txt` | 1 | 1 | 1 |
| **Total** | **65** | **4941** | **990** |
