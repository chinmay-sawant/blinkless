## Summary

Adds first-wave support for 10 standard CSS mask property names and eight selected `-webkit-` aliases. Blinkless attaches each mask to its element's display-list group and exports the mask data through the Go and WASM contracts. The catalog records the properties as Partial because Blinkless does not replay mask pixels itself.

## Motivation / context

- Plans: `plans/v0.0.1/css-masking/css-masking-checklist.md`
- Issues: no matching GitHub issue exists; this work follows the plan.

## Changes

### CSS cascade and values

- Add the `mask` shorthand and selected longhands, including CSS-wide values, custom properties, declaration ordering, list cycling, and shared declaration acceptance for `@supports` (`internal/layout/style_cascade.go:857`, `internal/layout/style_mask_accept.go:47`).
- Map eight selected WebKit aliases to the standard mask properties. Map the selected legacy composite values to standard operators (`internal/layout/style_cascade.go:1679`, `internal/layout/style_cascade.go:1843`).
- Cover the accepted values, aliases, source ordering, and rejected declarations in behavior and cascade tests (`internal/layout/css_behavior_mask_test.go:17`, `internal/layout/style_cascade_test.go:76`).

### Display-list groups and SVG masks

- Add optional mask metadata to `layout.BlendGroup`. The descriptor carries each layer's CSS values and resolved image bytes (`internal/layout/blend_group.go:33`, `internal/layout/mask_paint.go:25`, `internal/layout/layout.go:1050`).
- Convert selected external SVG `<mask>` fragments to alpha PNG data. The supported SVG subset and missing features are documented at `internal/svg/mask.go:62`.
- Export mask layers and image-table references in the WASM drawing-list contract (`bindings/wasm/drawing_list.go:205`, `bindings/wasm/drawing_list.go:578`).

### Catalog and documentation

- Record all 18 selected property names as Partial and update the compatibility matrix, deferred list, and library API notes (`plans/v0.0.1/css-masking/css-masking-checklist.md:13`, `documentation/compatibility-matrix.md:44`).
- Update the catalog totals to 389 implemented, 19 partial, 369 unsupported, and 8 intentionally ignored.

## Impact

| Area | Impact |
|------|--------|
| Performance | Masked elements get group and layer metadata. External SVG masks are rasterized. No performance benchmark was run. |
| Memory | Mask layers may retain resolved image bytes. WASM output refers to those bytes through the drawing list's image table. |
| Behavior / correctness | Adds mask values and source data to display-list groups. A downstream painter must apply the mask during replay. |
| API / CLI | Adds `MaskDescriptor` and `MaskLayer`, an optional `BlendGroup.Mask`, and an optional `mask` field in WASM group JSON. No CLI change. |
| Dependencies | None added. |
| Binary size / build time | Not measured. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| Positional `layout.BlendGroup` literals | Add the new `Mask` field or use keyed fields. WASM consumers should accept the optional `mask` field. |

## Test plan

- [x] `make test`
- [x] `make lint`
- [x] `go build ./...`
- [x] `make golden`
- [x] `make wasm-test`
- [x] `make claim-scan`, `make catalog-check`, `make matrix-check`, and `git diff --check`

The completed-tree checklist records these gates as passing on 2026-10-13.

### Commands

```sh
make test
go build ./...
make golden
make wasm-test
make lint
make claim-scan
make catalog-check
make matrix-check
git diff --check
```

## Screenshots / sample output

Blinkless has no page-pixel renderer. `internal/layout/css_behavior_mask_test.go:17` checks the mask group and its paint ownership. `internal/svg/mask_test.go:13` checks mask alpha at known pixels. `testdata/css/masking/element-group-reference.html` records the browser reference scene.

## Related issues

- No linked issue. `gh issue list` returned no issues; this is plan-driven work.

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Label applied: `enhancement`
- [x] Related issues checked; no matching ticket exists
- [x] Filled body saved under `plans/PR/pr-css-masking-first-wave.md`

## Follow-ups (out of scope)

- CSS gradients, `image-set()`, mask-border painting, SVG mask-region bounds, and pixel replay inside Blinkless remain outside this first wave.

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API changes documented
- [ ] New rules have fixture coverage where applicable
- [ ] PR has assignee and label
- [ ] Related issues use correct keywords
- [ ] No secrets or generated artifacts committed
- [ ] Diff-stat-by-extension table pasted at the bottom

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 23 | 3488 | 168 |
| `.html` | 1 | 55 | 0 |
| `.json` | 1 | 394 | 113 |
| `.md` | 6 | 312 | 56 |
| `.png` | 2 | Binary | Binary |
| `.py` | 1 | 29 | 9 |
| `.txt` | 1 | 1 | 1 |
| **Total** | **35** | **4279** | **347** |
