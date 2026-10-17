# feat(layout): mask-border, regions, and shapes ledgers (v0.0.3 wave)

## Summary

- Implements three v0.0.3 ledgers on one branch: mask-border (13 properties plus six WebKit aliases, carried as a `MaskBorder` descriptor on the blend group), regions/exclusions (9 properties as four one-canvas contracts), and shapes inside (shape-inside and shape-padding line insets).
- Every new property stores data on the existing drawing list; nothing adds a page rasterizer, a `Result.Pages` writer, or a new paint op kind. Pixel mask application stays with the display-list host.
- Catalog and matrix move with the code: the 13 mask-border rows are Partial, the 9 regions rows and shape-inside/shape-padding are Implemented, and `documentation/deferred.md` no longer calls regions a permanent non-goal.

## Motivation / context

- Plans: `plans/v0.0.3/mask-border/mask-border-checklist.md`, `plans/v0.0.3/regions/regions-checklist.md`, `plans/v0.0.3/shapes/shapes-checklist.md`
- Issues: see **Related issues**

## Changes

### Mask border (`internal/layout/mask_border.go`, new)

- `MaskBorder` descriptor on `BlendGroup` next to `Mask`: source token, image bytes, pixel size, slice insets and fill flag, widths, outsets, repeat pair, and mode. Lengths are in points.
- Six longhands accepted and stored; invalid values do not replace an earlier valid one. The descriptor attaches only when the used source is not `none`, and emits no `OpImage` (the mask shorthand reset already zeroes the source).
- The `mask-border` shorthand expands into its longhands in any order; CSS-wide keywords reset all seven names.
- Six `-webkit-mask-box-image*` aliases normalize onto the standard names in `normalizeWebkitMaskProperty`; source order follows the cascade.

### Regions, masonry tolerance, exclusions, wrap breaks (new: `named_flow.go`, `exclusion.go`, `wrap_break.go`, `masonry_tolerance.go`, `flex_lines.go`)

- `flow-into` / `flow-from` move a named flow's children into region boxes on the one canvas. A second `flow-from` region adopts the remainder; `region-fragment` chooses slice vs whole-child movement. No region means the content is not painted, not a page.
- `flow-tolerance` is a masonry packing knob: `normal` is one em of slack, a length is points, `infinite` keeps the shortest-column choice, and non-masonry grids ignore it.
- `wrap-flow` registers a box as an exclusion through the existing `floatState`; `wrap-through: none` builds against a scratch float state. `wrap-before` / `wrap-after` / `wrap-inside` drive line and flex breaks.
- `flexWrapLines` and its line-formation helpers moved verbatim from `flex.go` into `flex_lines.go` (file-size gate).

### Shapes inside (`internal/layout/style_shape_inside.go`, `internal/layout/shape_inside.go`, new)

- `shape-inside` accepts `auto`, `outside-shape`, `circle()`, `ellipse()`, `inset()`, and the four shape-box keywords; `polygon()`, `path()`, `url()`, and `display` stay rejected. `shape-padding` shrinks the inside contour (length or percentage).
- The inside interval narrows the element's own line boxes before float exclusion runs. `auto` and rejected tokens paint byte-identical to before. Lite rule: the contour needs a definite content height, so auto-height boxes are a no-op (documented; a measure pre-pass is a follow-up).

### WASM serialization (`bindings/wasm/drawing_list.go`)

- Group JSON gains `maskBorder` with source, image resource reference, slice, fill, width, outset, repeat, and mode. Image bytes ride the existing resource table. `MaskBorder` is re-exported from the public `layout` package next to the mask aliases.

### Catalog, matrix, docs

- 26 catalog rows updated (24 status changes plus limitation-only edits to `float-defer` and `shape-image-threshold`) with values, limitations, handlers, and tests: mask-border Partial; regions Implemented for the one-canvas contracts with the paged-fragmentation non-goal named; shape-inside/shape-padding Implemented with the Chrome no-BCD note kept.
- Compatibility matrix section 2 regenerated; header counts, section 5 rows, the phase-83 tally notes, and the vendor group C alias table refreshed to agree with section 2.
- `documentation/deferred.md`: mask row updated for the descriptor, regions row rewritten as four one-canvas contracts, v0.2.7 leftovers list corrected, vendor group C updated.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | No change on documents that do not use the new properties, with one deliberate exception: a masonry container with no `flow-tolerance` declaration now uses the CSS initial `normal` (one em of slack) instead of the old shortest-column packing; an explicit `flow-tolerance: infinite` restores the old choice. Every other hook is gated on the new style fields. |
| **Memory** | `ResolvedStyle` gains shape/flow/wrap fields; `BlendGroup` gains the `MaskBorder` pointer. Descriptors hold the source image bytes once, shared with the existing image resolver. |
| **Behavior / correctness** | New behavior only when the new properties are set, except the masonry default above. The mask shorthand reset, the first-wave mask aliases, border-image painting, float wrapping, flex, and grid suites are unchanged and green. |
| **API / CLI** | Additive: `layout.MaskBorder` and `layout.DisplayGroup.MaskBorder` exported; WASM group JSON gains an optional `maskBorder` object. No CLI in this tree. |
| **Dependencies** | None. Direct module allowlist untouched (`go-text/typesetting`, `tdewolff/canvas`). |
| **Binary size / build time** | Negligible; no new packages. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| Masonry `flow-tolerance` default | A `grid-template-rows: masonry` container with no `flow-tolerance` declaration now uses the CSS initial `normal` (one em of slack) instead of the old shortest-column packing. Set `flow-tolerance: infinite` to keep the old choice. |

## Test plan

- [x] `make test`
- [x] `make lint` (includes `size-check`)
- [x] `make golden`
- [x] `make claim-scan`
- [x] `make catalog-check`
- [x] `make matrix-check`

### Commands

```sh
make test
make golden
make lint
make claim-scan
make catalog-check
make matrix-check
```

All six exit 0 on the branch head.

## Screenshots / sample output

```
no visual surface changed; the output is the drawing list.
New behavior suites: TestBehaviorMaskBorder*, TestMaskBorderShorthand*,
TestBehaviorNamedFlow*, TestBehaviorFlowTolerance*, TestBehaviorWrap*,
TestBehaviorShapeInside*, TestShapeInsideStore*, TestShapePaddingStore*.
WASM: TestDrawingGroupSerializesMaskBorderAndImageReference (check-wasm-contract.sh exits 0).
```

## Related issues

- Relates to the v0.0.3 CSS coverage ledgers (`plans/v0.0.3/README.md`)

## PR metadata checklist (author)

- [ ] Self-assigned (`--assignee @me`)
- [ ] Labels applied
- [ ] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/PR/pr-mask-border-regions-shapes.md`

## Follow-ups (out of scope)

- Pixel replay of `MaskBorder` and the first-wave mask stays with the display-list host.
- Paged fragmentation remains a non-goal; `Result.Pages` is still not written.
- `shape-image-threshold` and `float-defer` keep their recorded decisions (no `url()` image shapes; only `float-defer: none`).
- Gradient `mask-border-source` waits on the mask-source ledger.
- shape-inside on auto-height boxes needs a measure pre-pass.

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
| `.go` | 33 | 5118 | 173 |
| `.html` | 12 | 288 | 0 |
| `.json` | 1 | 661 | 150 |
| `.md` | 6 | 137 | 126 |
| `.txt` | 1 | 1 | 1 |
| No extension | 1 | 3 | 0 |
| **Total** | **54** | **6208** | **450** |
