## Summary

Adds behavior coverage from Wave A through Wave M (551 tests across 69 files in `internal/layout` plus the Chrome harness), moves the catalog from 90 to 389 implemented rows, and adds two generated offline demo pages for the implemented set.

---

## Motivation / context

- Plans: `plans/v0.0.1/css-behavior-coverage-tests-checklist.md`
- Issues: see **Related issues** (no open issue; this is plan-driven work, 10 read-only review agents covered the diff)

---

## Changes

### 3D transforms and perspective

- New row-major mat4 core in `internal/layout/transform_3d.go` with orthographic and perspective flattening, 3D function parsing (`rotateX/Y/Z`, `rotate3d`, `translateZ`, `scaleZ`, `perspective()`, `matrix3d`) on a side channel so prior 2D geometry is unchanged.
- Perspective pre-pass (`perspective_pass.go`) inherits ancestor distance before the stamp pass; `perspective-origin` moves the vanishing point (percent, length, keyword).
- `transform-style: preserve-3d` composes chains before flattening; `backface-visibility: hidden` culls 3D-rotated faces at stamp time (2D mirrors still paint).
- `transform-box: view-box` resolves against the nearest SVG viewport; `fill-box` keeps the object bounding box.

### Fragmentation, multicol, grid, overflow

- Orphans snap band boundaries in `fragment_lines.go`; widows moves breaks earlier via `fragment_widows.go`; forced `break-before/after: column` support in `multicol.go`.
- Grid `align-content` distribution on definite-height containers; `break-inside` stays honestly partial with a pinned gap (no straddle test at the distribution call).
- Overflow clipping reads the scroll-aware port; deprecated `clip: rect()` clips abspos boxes; scroll-margin cascade with test-offset viewport plus margin-inset snap and visible rects (`scroll.go`, `scroll_runtime.go`).

### Inline, text, fonts

- Wires `text-justify` (none, inter-character via cloned styles), `white-space-trim`, `tab-size`, `text-wrap-style: pretty` and `avoid-short-last-line` (pack-loop re-break), ruby annotation stacking, `underline-position` shift, emphasis skip, and baseline shifts.
- Variable fonts covered end to end (wght instancing, optical sizing at large sizes, COLR palette).
- New `inline_decoration.go` split out of `inline_paint.go` to hold the file-size gate.

### Style cascade, color, forced colors

- Wide-gamut pipeline: `ParseColorWide` in `internal/css/color_modern.go` keeps headroom through the cascade into the `dynamic-range-limit` paint clamp (`color()`, `lab()`, `lch()`, `hwb()`, wide `color-mix`).
- `color-scheme` dark defaults and forced-colors mapping (`Options.ForcedColorsActive`; `auto` maps to system colors, `none` keeps authors) across block, table, and text paths.
- Gap-B stores (transform box/style, baselines, clip-rule, interpolation, perspective) plus `inheritGapBProps` bypass; `background-color` accepts wide spellings; perspective and perspective-origin cascade cases.

### Ruby, GCPM, SVG, paint

- Ruby stacks `rt`/`rtc` at half size via `CustomProps` cascade plus inline emission (align, merge, overhang, over/under with sign flip); `clip-rule` threads from cascade to all three mask sites (evenodd cuts the star center).
- `string-set`, bookmark outline, and footnote collection land as tested standalone systems without layout or paint integration.
- SVG bakes inline presentation attrs (dash, joins, fill-rule, geometry, stop color/opacity, paint-order, vector-effect, CSS `d` over `d` attr); raster bridge emulates dropped presentation (evenodd winding reversal, crispEdges snap) before the pinned canvas parse.
- Gradients and filters sample in linear light via `color-interpolation` and `color-interpolation-filters`; border-image repeat edges tile with partial clip instead of stretch fallback.

### Catalog, matrix, harness, demo pages

- `testdata/css/catalog/properties.json`: 785 rows, 389 implemented, 1 partial (`break-inside`), 387 unsupported, 8 ignored. Compatibility matrix section 2 regenerated with header counts; coverage plan ledger closed with dated wave evidence; `scripts/file-size-allowlist.txt` records `layout.go` at 2185.
- Chrome comparison harness in `test/chrome/harness` (11 flex/grid/logical/text/print cases, DisplayList vs headless Chrome join at 1px tolerance; raw output gitignored, test-only).
- `scripts/generate_css_showcase.py` builds `documentation/css-showcase.html` (one card per implemented property with syntax, live demo, declaration); `scripts/generate_css_engine_only.py` builds `documentation/css-engine-only.html` (63 rows Chrome 143 shows as initial, each card cites engine behavior tests). Four scene plugins supply self-contained markup (SVG data URIs, no network, no script). No engine code touched by the showcase commit.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Small added work per box (perspective pre-pass, scroll RLock in clip path, per-box deprecated-rect scan). No measured delta; test scale only. |
| **Memory** | New style fields (3D state, perspective distance/origin, gap-B stores) and test-only scroll offset map. No measured delta. |
| **Behavior / correctness** | 389 catalog rows now have behavior tests; 3D, fragmentation, ruby, SVG bake, and wide-gamut paths change rendering as intended. `break-inside` stays partial by design. |
| **API / CLI** | None. New test-only hooks (`forcedColorsTestActive`, scroll test offsets) are not production API. |
| **Dependencies** | None added. |
| **Binary size / build time** | Not measured. New files: `transform_3d.go`, `perspective_pass.go`, `fragment_lines.go`, `fragment_widows.go`, `ruby.go`, `footnote.go`, `string_set.go`, `scroll.go`, `scroll_runtime.go`, `inline_decoration.go`, `internal/svg/raster.go`. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [ ] `make test`
- [ ] `make lint` / `go vet`
- [ ] `make build` (when a build surface is part of the change)
- [ ] `make golden` when layout, paint, or fixture behavior changed
- [ ] `make claim-scan` when documentation or user-facing claims changed

### Commands

```sh
bash scripts/pr-diff-stat.sh master
```

Note: this session was read-only review across 10 agents plus PR authoring. Working tree was clean, so no new commit was made. The `make test`, `make lint`, `make golden`, `make claim-scan`, `make catalog-check` green claims come from the Wave commit messages, not from a re-run here. Run the full gate set before merge.

---

## Screenshots / sample output

No visual surface changed in the engine sense. Behavior is pinned by the drawing-list tests (`go test ./internal/layout -run TestBehavior`). The two demo pages are static offline HTML (`documentation/css-showcase.html`, `documentation/css-engine-only.html`); the showcase commit reports 389/389 cards present with balanced markup and zero script tags.

---

## Related issues

- No open issues found (`gh issue list` empty; no `Closes/Fixes/Relates` keywords in the 15 commit messages). Plan-driven work under `plans/v0.0.1/css-behavior-coverage-tests-checklist.md`.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [ ] Related issues filled with real ticket IDs (none exist; see above)
- [x] Filled body committed under `plans/PR/pr-<slug>.md` when process-gated

---

## Follow-ups (out of scope)

Findings from the 10-agent scan that do not block this PR but should be tracked:

- Catalog count text fixes: Wave F message says 5 promoted rows but totals move +12/-12; Wave H/L messages sum to 787 not 785 (unsupported 389 should be 387); Wave L end silently drops 2 unsupported rows.
- Checklist staleness: Phase 8 rows still cite 190 implemented and header 190/198/389/8 (actual 389/1/387/8); Overview still buckets on 428 cells behind 90 rows; Status says final tree uncommitted (it is committed); wave-notes section referenced but missing.
- Stale code comments: `inline_balance.go:118-119,154` says the pack loop only consults `balanceCanApply`; `origin_build_test.go:11` header says no production caller of `perspectiveCenter` (now called). Fix comments.
- `internal/layout/layout.go` is 2185 lines, over the 2000 soft cap, and grew in this branch. Next touch should extract a cohesive piece instead of growing it.
- Weak SameOps-only pins with no non-empty guard (vacuous pass if the parser drops the declaration): scroll-margin x5 in `overflow2`, perspective pair and dominant-baseline in `remaining`, pointer-events/cursor/user-select/caret/resize in `interaction`, transform-box/style and dynamic-range/forced-color in `final`, shape-inside in `list_shape`. Add op-count or stored-value guards following the `appearance` pattern.
- Three commit subjects break conventional shape (`Showcase: ...`, `Wave M: ...`, `Wave L: ...`). Rename on squash to `feat(...)` / `docs(...)`.
- Engine edge cases to verify separately: text-only `backface-visibility` boxes (zeroExclusiveOps covers W/H only), nested preserve-3d net facing, `border-image` round/space sharing the repeat core, `path("junk")` suppressing a good `d` attr, global scroll-offset map leak via address reuse, `background: color(display-p3 ...)` shorthand rejecting what longhand accepts, em-origin accept/apply divergence.

---

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
| `.conf` | 1 | 35 | 0 |
| `.go` | 120 | 28998 | 197 |
| `.html` | 2 | 86 | 0 |
| `.js` | 1 | 219 | 0 |
| `.json` | 2 | 3252 | 1543 |
| `.md` | 4 | 635 | 393 |
| `.py` | 7 | 2549 | 0 |
| `.sh` | 1 | 60 | 0 |
| `.txt` | 1 | 1 | 1 |
| **Total** | **139** | **35835** | **2134** |
