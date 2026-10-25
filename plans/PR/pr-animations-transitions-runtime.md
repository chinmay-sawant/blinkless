## Summary

Add the engine primitives a host needs to animate a laid-out page without re-parsing the document or re-running the full page layout: per-element transform metadata plus a 3D and perspective restamp pass, a single-element relayout that applies property overrides after the cascade, scroll and view timeline declarations, and an inline SVG paint restyle with a bounded raster cache. All public changes are additive; a document that does not set the new fields and does not call the new functions produces the same output as before.

## Motivation / context

- This branch supplies the engine side of the Ownframe `features/animations-and-transitions` work. Ownframe's automatic motion driver calls `layout.RelayoutElementWithOverrides` for layout keyframes, `layout.RestampTransforms` for 3D and perspective samples, and `layout.RestyleInlineSVG` for SVG paint.
- Before this branch the only way to animate a transformed element was to re-run the full layout. The new path keeps the cached document, rewrites only the affected operations, and reports the union of the old and new painted area as damage.
- There is no separate issue. The branch is the engine half of the animation plan recorded in the Ownframe repository.

## Changes

### Animated transforms and the restamp pass

- Export per-element transform metadata as `layout.TransformInfo` and carry it on `layout.Box.Transform`. Fields: `OriginX`/`OriginY` (resolved page-space layout points), `Box`, `Parent`, `Own`, `World`, `HasMatrix`, `HasTransform3D`, `Perspective`, `PerspectiveOriginX`/`PerspectiveOriginY` (resolved vanishing point), `HasPerspective`, `Style`, and `Backface`. `Parent`, the origin, and `World` stay populated even when the element declares no transform, so a keyframe can compose against them (`layout/layout.go:53-87`, `internal/layout/transform_meta.go:12-57`).
- Add `layout.RestampTransforms(ctx, doc, prev, overrides, options)`. It consumes an already-laid-out display, applies one transform list and optional perspective per element, and returns an `ElementPatch` carrying the patched display, the rewritten operation windows, the patched boxes, and the old and new painted area in CSS pixels (`layout/reproject.go:104-110`, `:165-190`).
- Keep re-projection cheap: reuse box geometry, the cached used styles, operation count/order/IDs, and each operation's pre-finalize form. Recompute only the perspective pre-pass, the stamp walk, the operation matrices, finalization, and damage (`internal/layout/reproject.go:98-101`, `:257-273`, `internal/layout/op_extra.go:53-58`).
- Project only the text baseline origin and store the local Jacobian as `GlyphBasis`, so glyph geometry moves without re-shaping, re-breaking, or changing the unprojected box (`internal/layout/display_geometry.go:184-188`).
- Build the restamped display as a copy of the previous one and share operation owners, paint order, keyframes, boxes, and dimensions instead of rebuilding them. A restamp never adds, removes, or reorders operations and never moves boxes, so the index-based metadata stays aligned (`layout/reproject.go:193-204`).
- Add `layout.Matrix2D` (with `Apply`, `Mul`, `IsIdentity`), `IdentityMatrix`, and `BakeOrigin` for hosts that compose transforms outside the engine (`layout/matrix.go:13-24`).

### Single-element relayout with overrides

- Add `layout.Override{ElementID, Property, Value}` and `layout.RelayoutElementWithOverrides(ctx, doc, prev, elementID, overrides, options)`. Overrides are written to the element's used style after the cascade and before placement, descendants re-resolve against the overridden parent, and the cached styles are restored afterwards. The containing block is rebuilt, so siblings reflow (`layout/relayout_element.go:166`, `internal/layout/relayout_element.go:277-381`).
- Apply `font-size` first so `em` values in the same override list see it, then shorthands, then longhands in byte order (`internal/layout/relayout_element.go:483-524`).
- Allow sizes and logical sizes, insets, margin, padding, gap, flex basis/grow/shrink, grid templates and auto tracks, aspect ratio, border and outline widths and radii, border spacing, column metrics, font size and metrics, text indent, tab size, vertical align, decoration thickness and offset, and object position. Refuse paint, transforms, SVG paint, display and flow properties, wide keywords, `var()`, `!important`, and empty values (`layout/relayout_element.go:634-669`).
- Add `ErrUnknownOverrideProperty`, `ErrInvalidOverrideValue`, and `ErrOverrideReflow`. A refused property leaves `prev` unchanged (`layout/relayout_element.go:36-47`).

### Scroll and view timelines

- Accept and cascade seven properties: `scroll-timeline`, `scroll-timeline-name`, `scroll-timeline-axis`, `view-timeline`, `view-timeline-name`, `view-timeline-axis`, and `view-timeline-inset`. Shorthands expand to longhands before the cascade, so origin, specificity, and source order apply per longhand; a `var()` shorthand expands in the applier (`internal/layout/timeline_style.go:9-16`, `internal/layout/style_shorthand.go:14-20`).
- Export `layout.TimelineDef{Name, Kind, Axis, SubjectID, ScrollerID, InsetStart, InsetEnd}` and `Box.Timelines`. Resolve the name, kind, used physical axis (through `writing-mode`), the view subject id, and insets at layout time; leave scroll offset, progress, and `animation-timeline` matching to the host (`layout/timeline.go:19-59`, `internal/layout/timeline_style.go:177-198`, `:484-535`).
- Keep the existing rule that a timeline name is not inherited, while an explicit `inherit` copies all five values (`internal/layout/timeline_style_test.go:282`).

### Inline SVG paint restyle

- Add `layout.SVGPaintOverride` with `ElementID`, optional `PartID`, and seven restylable values with matching `Has*` flags, plus `layout.RestyleInlineSVG(ctx, doc, prev, overrides, options)` (`layout/svg_paint.go:41-63`, `:100`).
- Support `fill`, `stroke`, `fill-opacity`, `stroke-opacity`, `stroke-width`, `stroke-dasharray`, and `stroke-dashoffset`. A root override enters the root's used style before serialization so inheriting shapes follow it, while shapes with their own paint keep it. `currentColor` resolves to the element's used colour (`internal/layout/layout_svg.go:1266-1395`).
- Add a bounded per-layout raster cache keyed on the exact serialized SVG XML: 8 entries and 2,097,152 decoded pixels, FIFO eviction, and a raster larger than the pixel budget is not stored (`internal/layout/svg_raster_cache.go:10-20`, `:86-112`).
- Add `ErrNotInlineSVG`, `ErrNoSVGPart`, and `ErrSVGNotLaidOut` for misaddressed overrides (`layout/svg_paint.go:19-28`).

### Catalog

- Change the summary counts from 569 to 576 engine-registered properties and 216 to 209 upstream-only properties. The seven timeline properties are the difference (`testdata/css/catalog/properties.json:20-21`).

## Impact

| Area | Impact |
|------|--------|
| **Performance** | A restamp reuses the cached document, box geometry, used styles, operation owners, paint order, and keyframes. It rewrites only the affected operation windows and removes the per-frame metadata rebuild that dominated the Ownframe 3D profile. No benchmark or ns/op number is committed. The inline SVG cache avoids re-rasterizing repeated paint states; the public restyle path still rebuilds page metadata on every call. |
| **Memory** | `Box` grows by two fields. The SVG raster cache pins at most 8 serialized SVG strings and 2 Mi decoded pixels per layout result, dropped with the layout. A transform restamp shares the previous display's metadata instead of copying it and allocates one full operation clone per rewritten window for rollback. |
| **Behavior / correctness** | Static paint is unchanged when the new fields and calls are unused. A restamp matches a fresh layout for rotate, translate, scale, matrix, perspective, nested ancestors, flat boundaries, and preserve-3d chains in the committed parity tests. Overrides are atomic: a refused property leaves the previous display untouched. |
| **API / CLI** | Additive Go API only: new types, fields, functions, and sentinel errors. There is no CLI in this tree. The WASM drawing-list schema is unchanged. |
| **Dependencies** | None. `go.mod` and `go.sum` are unchanged on this branch. |
| **Binary size / build time** | Nine new source files plus tests. No new packages and no new build targets. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| `layout.Box` gains `Timelines` and `Transform` | Keyed struct literals keep compiling. External unkeyed literals must add the two fields or switch to keyed fields. |
| `RestampTransforms` and `RestyleInlineSVG` mutate and invalidate `prev` | Treat the previous display as stale after the call and use `patch.Display`. Lay out with `Options.RetainState`; otherwise both return `ErrNoRetainedState`. One display must not be patched from two goroutines. |
| A restamp leaves `Display.Boxes[].Transform` at the pre-animation matrices | Read the painted result from the operations, not from box metadata, until a later pass refreshes boxes. |
| Overrides cannot change paint, transforms, display, or flow | Use `RestampTransforms` for transforms and `RestyleInlineSVG` for SVG paint. |

## Test plan

- [x] `make test`
- [x] `make lint`
- [x] `make golden`
- [x] `make claim-scan`
- [ ] `make catalog-check` (fails against the sibling Ownframe working tree, see Follow-ups)
- [x] `make matrix-check`
- [x] `make wasm-test`
- [x] `make build`

### Commands

```sh
make test
make lint
make golden
make claim-scan
make catalog-check   # fails when ../ownframe is on the feature branch; see Follow-ups
make matrix-check
make wasm-test
```

Focused coverage in the branch:

- Restamp parity: `TestRestampRotateYMatchesFreshLayout`, `TestRestampMovesBetweenFrames`, `TestRestampPerspectiveMatchesFreshLayout`, `TestRestampTransformFunctionsMatchFreshLayout`, `TestRestampAncestorProjectionIsReapplied`, `TestRestampFlatBoundaryCompositionIsReapplied`, `TestRestampPerspectiveNoneRestoresOuterCamera`, `TestRestampNestedPerspectiveAndTransformOverrides`, `TestRestampPreserve3DChainMatchesFreshLayout`
- Transform metadata: `TestPlacedTransformOriginZeroIsLayoutPoints`, `TestPlacedTransformOriginPercentStaysInPoints`, `TestPlacedTransformAbsentKeepsParentAndOrigin`
- Overrides: `TestRelayoutElementWithOverridesWidensAndMovesSibling`, `TestOverrideFontSizeInheritsIntoTextRun`, `TestOverrideDoesNotPersist`, `TestOverrideRefusedPropertyLeavesDisplayUntouched`, `TestOverrideReflowEscapesTarget`
- Timelines: `TestTimelineDeclarationValueAccepted`, `TestTimelineShorthandAndLonghandCascadeOrder`, `TestTimelineLogicalAxisExport`, `TestViewTimelineInsetPercent`, `TestPlacedTimelinesScrollBeforeView`
- SVG paint: `TestRestyleInlineSVGFillChangesRaster`, `TestRestyleInlineSVGRootFillInherits`, `TestRestyleInlineSVGPartOverride`, `TestRestyleInlineSVGRasterCache`, `TestSVGRasterCacheBounds`

## Screenshots / sample output

Nothing visible changes on its own. The restamp parity tests compare operation kind, geometry, text, projective payload, path points, and glyph basis against a fresh layout. The SVG tests compare the produced raster after a paint override and assert cache hit/miss counts and the 8-entry bound. The drawing-list JSON is unchanged.

## Related issues

None. This branch does not close an open issue.

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`).
- [x] `enhancement` label applied.
- [x] Related issues reviewed; no open issue matches this branch.
- [x] Filled body committed under `plans/PR/pr-animations-transitions-runtime.md`.

## Follow-ups (out of scope)

- No documentation, `CHANGELOG.md`, or release-note entry was added on this branch. The new public API is documented in Go doc comments only.
- `make catalog-check` fails in a workspace that has the Ownframe branch checked out at `../ownframe`: seven `downstream_tests` case literals moved when Ownframe refactored the `css-preview` tests. `animation-composition` now uses `"add,add"`, the four `animation-range*` rows use a combined `animation-range:25% 75%` string, `animation-trigger` uses `"--fire play reset"`, and `trigger-scope` uses `"--fire play"`. The gate skips these refs when the sibling Ownframe directory is absent, which is why the branch itself is green.
- The seven timeline properties are counted as engine-registered, but their catalog rows still say `unsupported` with no handler, and `documentation/compatibility-matrix.md` still says no handler exists. The catalog status and the matrix need a reconciliation pass.
- `animation-timeline` remains an opaque string and is never cross-checked against a declared timeline name; joining the two is host work.
- `ScrollerID` is always empty. There is no nested-scroller timeline, no `timeline-scope`, and comma-separated timeline lists are rejected whole.
- The restamp pass supports 3D and perspective only; backface culling is one-way, opacity is not re-folded, table cells and off-chain boxes are refused, and images cannot finalize under a projective camera.
- `Display.Boxes[].Transform` is not refreshed after a restamp, and box-derived stamp state is not restored when a pass fails.
- Paint servers (`url(#gradient)`), element `opacity`, `fill-rule`, line cap/join, markers, `vector-effect`, filters, masks, and clip paths are not restylable through `SVGPaintOverride`.
- Colour alpha is discarded when used paint is read, so `transparent` and `rgba(..., 0.5)` restyle as opaque RGB.
- The SVG cache is FIFO rather than LRU, and the public restyle path still rebuilds page metadata on each call.
- No benchmark pins the restamp reuse win or the cache hit cost.

## Reviewer checklist

- [ ] Behavior matches summary and test plan.
- [ ] No unrelated changes in diff.
- [ ] Public API / CLI changes documented.
- [ ] New rules have fixture coverage when applicable.
- [ ] PR has assignee and labels.
- [ ] Related issues use correct Closes/Relates keywords.
- [ ] No secrets or generated artifacts committed.
- [ ] Diff-stat-by-extension table pasted at the bottom.

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 38 | 8136 | 112 |
| `.json` | 1 | 2 | 2 |
| `.txt` | 1 | 1 | 1 |
| **Total** | **40** | **8139** | **115** |
