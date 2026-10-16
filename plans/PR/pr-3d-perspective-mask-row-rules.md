## Summary

This pull request implements four WebKit 3D prefix aliases and multi-axis 3D transform functions, adds support for linear gradients and image sets in CSS masks, and introduces row-rule styling and painting for grid and flex layouts. It also updates the CSS catalog, compatibility tables, deferred notes, and v0.0.3 phase ledgers to record passed test gates. Follow-up commits on the branch finalize projected operations into display geometry, preserve hidden backfaces in print media, and compose preserve-3d chains in world space.

---

## Motivation / context

- Plans: `plans/v0.0.3/3d-perspective/3d-perspective-checklist.md`, `plans/v0.0.3/mask/mask-checklist.md`, `plans/v0.0.3/row-rules/row-rules-checklist.md`
- Issues: see **Related issues**

Earlier versions deferred several CSS transform, masking, and rule features or left them unsupported in the catalog. This work completes the remaining items for 3D perspective aliases and multi-value transforms, brings gradient masks to parity with existing gradient background support, and adds gap rule painting between grid and flex container rows.

---

## Changes

### 3D transform aliases and multi-axis functions

- Mapped `-webkit-backface-visibility`, `-webkit-perspective`, `-webkit-perspective-origin`, and `-webkit-transform-style` to unprefixed properties in `normalizeWebkitTransformProperty` (`internal/layout/style_cascade.go:1769-1786`).
- Added parsing and projection handling for `translate3d(x, y, z)`, `scale3d(x, y, z)`, 3-value `translate`, `scale` with Z factor, and `rotate` with an axis vector in `internal/layout/transform_3d.go` and `internal/layout/transform.go`.
- Added tests for WebKit aliases in `internal/layout/css_behavior_webkit_3d_alias_test.go` and transform gap behavior in `internal/layout/css_behavior_transform_3d_gap_test.go`.

### CSS masking (gradients and image sets)

- Accepted `linear-gradient` and `image-set` values for `mask-image` in `internal/layout/style_mask_accept.go` and `internal/layout/style_value_accept.go`.
- Implemented gradient mask rasterization via `maskGradientPNG` and candidate resolution in `internal/layout/mask_source.go:34-40`.
- Integrated mask raster alpha application in `internal/layout/mask_paint.go:45-115`.
- Added regression tests in `internal/layout/css_behavior_mask_test.go`.

### Row rules (gap rules)

- Added parsing for `row-rule`, `row-rule-color`, `row-rule-style`, and `row-rule-width` in `internal/layout/style_gap_rules.go:28-40`.
- Added rule segment generation and drawing list emission between rows in `internal/layout/gap_rules.go:9-40`.
- Connected container painting for flex wrapped rows and grid rows in `internal/layout/flex.go` and `internal/layout/grid.go`.
- Added tests in `internal/layout/css_behavior_row_rule_test.go` and `internal/layout/style_gap_rules_test.go`.

### Follow-up fixes (display geometry, mask raster, print backfaces)

- Finalized projected operations into display geometry (`9882877`). `FinalizeDisplayOps` runs at the end of `DisplayListOptions`: rectangles and links become 4-point paths, lines and grid lines become projected segments, and text gets a projected anchor plus a local `GlyphBasis`. Each op's own transform and projective matrix are cleared so painters do not apply the projection twice. New accessors `PathValue`, `GlyphBasisValue`, and `ProjectiveTransformValue` (`internal/layout/op_read.go`) expose the result. An image op that still carries a projection is rejected with a static error.
- Fixed mask gradient stop positions (`internal/layout/mask_gradient.go`, `9882877`). Lengths such as `black 6px, transparent 12px` in linear and repeating-linear gradients are rewritten as percentages of the gradient-line length before rasterization, so stops are no longer read as raw `t` values. The rasterized PNG bytes travel on the mask group (`MaskDescriptor.Raster`, `internal/layout/mask_paint.go`).
- Corrected perspective math (`internal/layout/transform.go`, `internal/layout/transform_3d.go`, `9882877`). The vanishing point is taken relative to `transform-origin`, `baked3DForBox` returns the raw stored matrix with the origin baked by the caller, and a culled backface is deactivated (`Kind = OpKindNoop`) instead of left zero-sized.
- Preserved hidden backfaces in print media (`7cc73fc`). The transform stamping path now carries the active media; `backface-visibility: hidden` still culls on screen but keeps hidden faces when the media is print, matching Chrome's print-to-PDF output (`internal/layout/transform.go`).
- Composed preserve-3d chains in world space (`2d1a7c8`). New `internal/layout/transform_preserve.go` stamps an op range with the composed world transform: projective when a perspective distance applies, plain affine otherwise, with `OpImage` under a preserve chain baking a center-projected Jacobian. Chrome splices also call `shiftDescendantOpRanges` (`internal/layout/layout_chrome.go`) so descendant op ranges stay aligned after a shift.
- Regression coverage: backface cull/visible/mirror tests, backface break tests, preserve-build divergence and flat-boundary tests, `TestTransformBoxViewBox*`, and gradient mask raster tests (`internal/layout/css_behavior_*_test.go`).

### Catalog, docs, and maintenance

- Updated catalog entries in `testdata/css/catalog/properties.json`.
- Updated `documentation/compatibility-matrix.md` and `documentation/deferred.md` to reflect supported features and remove outdated claims.
- Updated checklists under `plans/v0.0.3/` to close completed items.
- Removed outdated knowledge-base references in `AGENTS.md`.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Gap rule painting and gradient mask rasterization run only when declared. Display finalization adds one linear pass over the op list per run; other default paths are unchanged. |
| **Memory** | Small additions to style structures for row rule properties and mask sources. |
| **Behavior / correctness** | WebKit 3D aliases now map to unprefixed properties. Gradients and image sets now work in `mask-image`. Row rules paint between flex/grid rows. 3D perspective is baked into display geometry, and hidden backfaces are preserved in print media. |
| **API / CLI** | Additive: the public `layout` package gains display-list payload accessors (`PathValue`, `GlyphBasisValue`, `ProjectiveTransformValue`) and result types (`DisplayPoint`, `DisplayPath`, `GlyphBasis`). No breaking changes. |
| **Dependencies** | No new dependencies. |
| **Binary size / build time** | Modest addition of layout code and tests. File size limit enforced. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] `make test-quick` exited 0 (unit and integration suites pass; the CI test step is green at `2d1a7c8`)
- [x] `make golden` exited 0 (`TestDisplay` in `layout/` passes)
- [x] `make lint` exited 0 (golangci-lint v1.64.8 and file size checks pass; the `2d1a7c8` findings are cleared in `75f7937`)
- [x] `make claim-scan` exited 0 (no forbidden claims in documentation)

### Commands

```sh
make test-quick
make golden
make lint
make claim-scan
```

---

## Screenshots / sample output

```
ok  	github.com/chinmay-sawant/blinkless/layout	0.018s
```

---

## Related issues

- None. No GitHub issue is linked.
- Relates to `plans/v0.0.3/` feature roadmaps.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`documentation`, `enhancement`)
- [x] Related issues filled. There is no ticket ID.
- [x] Filled body committed under `plans/PR/pr-3d-perspective-mask-row-rules.md`

---

## Follow-ups (out of scope)

- Corner-shape, speech, anchors, and scroll timeline checklists in `plans/v0.0.3/` remain open for future work.
- Reconcile stale `documentation/compatibility-matrix.md` rows that still contradict the implemented 3D path: the four `-webkit-` aliases are listed as "print-noop (3D)" (lines 1097-1100), and two notes still call 3D/perspective a permanent non-goal (lines 923 and 1127), while `documentation/deferred.md` records them as implemented.
- Refresh `plans/v0.0.3/3d-perspective/` and `plans/v0.0.3/mask/` checklist rows that describe the pre-fix backface and mask paths touched by `9882877`, `7cc73fc`, and `2d1a7c8`.
- Fix the stale PDF-era comment in `internal/layout/blend_group.go:46` ("The PDF adapter uses these bytes...") since this tree ships no PDF writer.

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
| `.go` | 37 | 3702 | 187 |
| `.json` | 1 | 685 | 339 |
| `.md` | 19 | 3235 | 74 |
| **Total** | **57** | **7622** | **600** |
