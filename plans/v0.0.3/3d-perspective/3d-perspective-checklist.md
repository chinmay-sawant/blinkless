# v0.0.3 - 3D perspective leftovers

> **Parent:** `plans/v0.0.3/README.md`
> **Status:** not started, except baseline rows verified in this session.
> **Estimated effort:** 14 catalog names, 4 of them still unsupported. 6 phases. The 10 unprefixed names are already implemented. This ledger does not rebuild them.

---

## Overview

The catalog, read on 2026-10-15 from `testdata/css/catalog/properties.json` (`updated` is `2026-10-13`), says these are implemented: `backface-visibility`, `perspective`, `perspective-origin`, `rotate`, `scale`, `transform`, `transform-box`, `transform-origin`, `transform-style`, `translate`.

These are unsupported, with no handler: `-webkit-backface-visibility`, `-webkit-perspective`, `-webkit-perspective-origin`, `-webkit-transform-style`.

`documentation/deferred.md:115` still says 3D matrices, perspective, and backface culling are not implemented. That sentence is behind the code. Phase 5 corrects it. Several test comments say the same stale thing. The bodies of the later tests disagree with those comments. Phase 0 records what the code does. Do not "fix" a working projection because a comment says the engine is 2D.

The drawing list is the ordered paint instructions. Each paint op may carry one 2D matrix, six numbers that scale, skew, and shift that op. A perspective camera makes the near side of a rotated box larger than the far side. With no camera, the projection is flat: a 60 degree turn around the vertical axis halves the width and does not change the height. This file calls that flat case orthographic. `XformSet` marks that the matrix was written (`internal/layout/layout.go:484`). Identity matrices stay unset. The stamp writes the matrix with `setXform` (`internal/layout/transform.go:1444-1445`). Paint Y grows downward. 1 CSS pixel is 0.75 points (`layout/displaylist.go:12-14`, `layout/displaylist.go:27-29`). Layout box X, Y, width, and height do not move when a 3D function runs. The matrix on the op does. `perspAssertSameOps` compares kind, text, and box geometry, and it does not compare the matrix (`internal/layout/css_behavior_perspective_test.go:23-29`). A test that uses it can pass while the matrix still changes.

2D `translate`, `rotate`, and `scale` already move the matrix. This ledger does not replan those.

New parsers go in `internal/layout/transform_3d.go`, which already owns `parse3DFunc`. Alias cases go in the existing WebKit normalizer in `internal/layout/style_cascade.go`. New tests go in `internal/layout/css_behavior_transform_3d_gap_test.go`. Do not add this work to `internal/layout/layout.go`.

## Executive summary

| Piece | What the code does now | This ledger |
|---|---|---|
| 2D `transform`, `translate`, `rotate`, `scale`, `transform-origin` | matrix on the op changes | leave it |
| `transform-box` | content box and SVG view box change the origin | leave it |
| `perspective`, `perspective-origin` | 3D children foreshorten | leave it |
| `transform-style: preserve-3d` | nested `rotateY` composes once | leave the 3D path |
| `backface-visibility` | the element's own `rotateY(180deg)` can hide its paint | one composed-chain hole |
| `rotateX`, `rotateY`, `rotateZ`, `rotate3d`, `translateZ`, `scaleZ`, `perspective()`, `matrix3d` | parsed onto the 3D side matrix and projected by the shared stamp | layout proofs where a test does not pin the numbers |
| `translate3d()`, `scale3d()`, Z on the `translate` and `scale` properties, axis form of `rotate` | rejected or ignored | implement |
| four `-webkit-` aliases | ignored | map them |

## Phase 0: the matrix contract

### How a 3D list becomes a 2D matrix

- [x] A transform list parses to a 2D matrix plus a 3D side matrix. 3D functions add identity to the 2D matrix and multiply the side matrix (`internal/layout/transform.go:285-287`, `internal/layout/transform.go:417-423`). The style stores both (`internal/layout/style_properties.go:1809-1818`, fields at `internal/layout/style.go:506-509`).
- [x] `parse3DFunc` accepts `rotateX`, `rotateY`, `rotateZ`, `rotate3d`, `translateZ`, `scaleZ`, `perspective()`, and `matrix3d` (`internal/layout/transform_3d.go:190-208`). `TestTransform3DParseAccepts3DFuncs` lists those strings (`internal/layout/transform_3d_test.go:71-87`). It rejects an empty `rotateY()`, a zero `rotate3d` axis, a percent `translateZ`, a non-positive `perspective()`, a short `matrix3d`, and an unknown name (`internal/layout/transform_3d_test.go:90-106`). Those rejections stay.
- [x] At stamp time, `boxTransformAccum` bakes the 2D matrix about the origin, then, when `HasTransform3D` is set and `preserveComposed` is clear, flattens the side matrix and multiplies it on (`internal/layout/transform.go:1292-1310`). The camera is the ancestor `perspective` distance when the pre-pass filled it, else the `perspective()` function's own distance, else orthographic (`internal/layout/perspective_pass.go:48-62`). The pre-pass copies the nearest ancestor perspective onto a 3D descendant (`internal/layout/perspective_pass.go:16-36`).
- [x] `preserve-3d` composes in the chain and flattens once. `markPreserveComposed` sets the flag so `boxTransformAccum` skips its own 3D branch (`internal/layout/transform.go:1339-1345`, `internal/layout/transform.go:1303`). `preserveContinuer` multiplies the chain and flattens the product (`internal/layout/transform.go:1205-1231`).
- [x] Backface culling reads the element's own side matrix, flattens it with a zero camera, and, when the facing is negative and `backface-visibility` is `hidden`, zeroes that box's exclusive ops (`internal/layout/transform.go:1449-1472`, `internal/layout/transform.go:1418-1428`). A 2D matrix never qualifies, because `HasTransform3D` is required.

### Properties the tests already show

- [x] `transform: translate(10pt, 5pt)` bakes E=10 and F=5 (`internal/layout/css_behavior_paint_effects_test.go:146-161`). The `translate` property does the same (`internal/layout/css_behavior_paint_effects_test.go:195-210`). Do not replan 2D translate.
- [x] `rotate: 90deg` bakes the quarter turn (`internal/layout/css_behavior_paint_effects_test.go:216-236`). `scale: 2 3` bakes A=2 and D=3 (`internal/layout/css_behavior_paint_effects_test.go:242-257`). Do not replan those 2D forms.
- [x] `transform-origin` changes the baked translation of a 2D rotation (`internal/layout/css_behavior_paint_effects_test.go:168-189`). `-webkit-transform` and `-webkit-transform-origin` already map (`internal/layout/style_cascade.go:1764-1772`). `TestWebkitPrefixAliases` covers `-webkit-transform` (`internal/layout/style_cascade_test.go:454`). Do not replan those two aliases.
- [x] `transform-box: content-box` shifts the baked matrix when padding insets the content box (`internal/layout/css_behavior_property_stores_test.go:16-57`). On a plain HTML div, `fill-box` and `view-box` bake the same matrix, which is the border box (`internal/layout/css_behavior_final_test.go:171-181`, `internal/layout/css_behavior_transform_viewport_test.go:145-157`). On SVG, `view-box` uses the nearest SVG viewport and the baked translation differs (`internal/layout/css_behavior_transform_viewport_test.go:116-138`). `resolveTransformOriginWithBox` is that switch (`internal/layout/transform.go:120-144`). Do not replan `transform-box`.
- [x] `rotateY(60deg)` with no perspective bakes A=0.5 and D=1 (`internal/layout/css_behavior_perspective_build_test.go:83-113`). `rotateX(45deg)` and `rotateY(45deg)` set `HasTransform3D` and stamp a matrix, while the layout box stays put (`internal/layout/css_behavior_perspective_test.go:113-145`). The `rotateX` test does not pin the numeric bake. That pin is Phase 2, not a rewrite of the parser.
- [x] `perspective: 500px` on an ancestor changes the child's baked matrix for `rotateY(60deg)` (`internal/layout/css_behavior_perspective_build_test.go:121-153`). A 2D `rotate(10deg)` child is not foreshortened (`internal/layout/css_behavior_perspective_test.go:49-79`). `perspective(500px) rotateY(60deg)` differs from `rotateY` alone (`internal/layout/css_behavior_perspective_build_test.go:161-188`). `perspective(500px) rotate(10deg)` matches plain `rotate(10deg)` because there is no depth (`internal/layout/css_behavior_perspective_test.go:89-103`).
- [x] `perspective-origin` moves the vanishing point. `25% 75%` is a quarter of the width and three quarters of the height (`internal/layout/css_behavior_origin_build_test.go:54-70`). `perspectiveCenter` reads `PerspectiveOriginSet` (`internal/layout/transform_3d.go:360-376`). Left versus right origins change the stamped E of a `rotateY(45deg)` child (`internal/layout/css_behavior_origin_build_test.go:152-189`).
- [x] Nested `rotateY(60deg)` over `rotateY(-60deg)` bakes A about 0.25 under `flat` and about 1 under `preserve-3d` (`internal/layout/css_behavior_preserve_build_test.go:44-88`). A `flat` middle clears the chain. A `preserve-3d` middle does not (`internal/layout/css_behavior_preserve_build_test.go:126-156`). Two nested 2D rotations paint the same for `flat` and `preserve-3d` (`internal/layout/css_behavior_perspective_test.go:189-211`). That 2D result is expected. It is not evidence that `preserve-3d` is ignored.
- [x] `backface-visibility: hidden` with `rotateY(180deg)` zeroes the red fill. `visible` paints it (`internal/layout/css_behavior_backface_build_test.go:32-74`). `scaleX(-1)` and `rotate(10deg)` stay painted (`internal/layout/css_behavior_backface_build_test.go:80-100`, `internal/layout/css_behavior_svg_paint_test.go:499-511`).

### Comments and names that contradict those tests

These comments are stale. The tests can still pass. Do not treat the comments as the contract.

- [x] `TestPerspPerspectiveFunctionRejected` is named "rejected." The body requires `HasTransform3D` and identical paint only for a depthless `rotate` (`internal/layout/css_behavior_perspective_test.go:82-103`). The build test shows the function changes a `rotateY` bake. Both assertions can be true. The name cannot.
- [x] `TestPerspRotate3DRejected` is named "rejected." The body requires a stamped matrix for `rotateX` and `rotateY` (`internal/layout/css_behavior_perspective_test.go:106-145`). `translateZ(100px)` and an identity `matrix3d` parse and do not stamp a matrix, because an orthographic depth-only transform is identity. That is applied identity, not a rejected function.
- [x] `TestBehaviorPerspectiveNoLayoutEffect` says the declaration is dropped (`internal/layout/css_behavior_remaining_test.go:219-225`). The fixture has no 3D child, so identical ops do not show a drop. `applyPerspectiveProperty` stores a positive distance (`internal/layout/style_leftovers.go:165-180`). The comment above that function still says "stored only" (`internal/layout/style_leftovers.go:161-163`). The ancestor build test is the one that shows foreshortening.
- [x] `TestPerspPerspectiveOriginNoEffect` says `perspectiveCenter` returns the default center and the property is dropped (`internal/layout/css_behavior_perspective_test.go:155-160`). The fixture has no 3D child. The origin build tests show the center moves and the stamped E moves. The comment on `perspectiveCenter` still calls the body a default-center stub (`internal/layout/transform_3d.go:345-349`). The origin build file header says there is no production caller and that layout divergence is pending (`internal/layout/css_behavior_origin_build_test.go:13-16`). `propagatePerspective` calls `perspectiveCenter` (`internal/layout/perspective_pass.go:31`), and `TestOriginBuildShiftsProjectedOutput` in that same file asserts the layout difference.
- [x] `TestBehaviorTransformStylePreserve3DNoPaintEffect` says the declaration is dropped and 3D is a print non-goal (`internal/layout/css_behavior_final_test.go:184-191`). Its fixture uses 2D `rotate` only. `applyTransformStyleProperty` still says both values flatten with no paint difference (`internal/layout/style_leftovers.go:139-142`). The preserve build test shows a paint difference for nested `rotateY`.
- [x] `TestBehaviorBackfaceVisibilityNoPaintEffect` says the declaration is dropped (`internal/layout/css_behavior_svg_paint_test.go:492-498`). Its fixture is a 2D rotation, which must not cull. `applyBackfaceVisibilityProperty` still says hidden paints like visible (`internal/layout/style_leftovers.go:150-153`). The backface build test culls `rotateY(180deg)`.
- [x] The header of `css_behavior_perspective_test.go` says the parser rejects 3D functions at `transform.go:351` (`internal/layout/css_behavior_perspective_test.go:7-10`). Line 351 is inside `scanTransformParens`. Acceptance is at `internal/layout/transform.go:417-423`.
- [x] `TestPerspBuildPreserveComposedSkipsProjection` sets `preserveComposed` by hand and expects `boxTransformAccum` to return identity for a lone `rotateY` (`internal/layout/css_behavior_perspective_build_test.go:191-216`). That matches the skip at `internal/layout/transform.go:1303`. It does not mean a real `preserve-3d` tree skips projection. The chain flattens in `preserveContinuer`. A reader who calls `boxTransformAccum` on a flagged box and stops there will think `preserve-3d` does nothing.
- [x] `internal/layout/style.go:499-501` says paint is unchanged until a stamp-time consumer lands. `boxTransformAccum` is that consumer and it has landed.

Gate. The contract above is the baseline. Later phases do not re-derive 2D translate, rotate, or scale, and they do not rebuild perspective for `rotateY`.

## Phase 1: the four prefixed aliases

`normalizeWebkitTransformProperty` maps `-webkit-transform` and `-webkit-transform-origin` only (`internal/layout/style_cascade.go:1764-1772`). The four names below fall through and are ignored.

- [ ] Map `-webkit-backface-visibility` to `backface-visibility`.
- [ ] Map `-webkit-perspective` to `perspective`.
- [ ] Map `-webkit-perspective-origin` to `perspective-origin`.
- [ ] Map `-webkit-transform-style` to `transform-style`.

Path: `normalizeWebkitTransformProperty` in `internal/layout/style_cascade.go`. No new file for four case lines.

- [ ] Each alias reaches the same used value and the same baked result as the standard property. Proof: `TestBehaviorWebkitPerspectiveAliasForeshortens` uses `-webkit-perspective: 500px` on the ancestor and `rotateY(60deg)` on the child, and the baked A differs from the orthographic case the way `TestPerspBuildAncestorPerspectiveForeshortens` already requires. One more case each for origin, `preserve-3d` nested `rotateY`, and hidden `rotateY(180deg)`.
- [ ] Declaration order between the alias and the standard property follows the cascade, both orders. Proof: a cascade test.
- [ ] Do not map any other `-webkit-` name in this change.

Gate. The four catalog rows can leave `unsupported` only after this phase's tests pass. Status becomes `implemented` because they alias behavior the unprefixed tests already observe. Run `go test ./internal/layout -run 'TestBehaviorWebkitPerspective|TestWebkitPrefix|TestPerspBuild|TestPreserveBuild|TestBackface'`.

## Phase 2: functions the parser still drops

- [ ] `translate3d()` and `scale3d()` are not in the `parse3DFunc` switch (`internal/layout/transform_3d.go:193-208`). A list that contains one of them fails the whole list, so a leading 2D translate is dropped too. Parse `translate3d(tx, ty, tz)` as a 2D translate of tx and ty plus `translateZ` of tz. Percentages on tx and ty stay on the existing percent path. Percentages on tz stay rejected, matching `translateZ`. Parse `scale3d(sx, sy, sz)` as a 2D scale of sx and sy plus `scaleZ` of sz. One number does not apply. Path: `internal/layout/transform_3d.go` and the 2D arm in `parseOneTransformFunc` (`internal/layout/transform.go:393-427`). Expected: `translate3d(10px, 0, 0)` stamps a non-zero E, and `scale3d(2, 3, 1)` stamps A=2 and D=3. Proof: `TestBehaviorTranslate3dAndScale3d` in the new gap test file.
- [ ] The `translate` property ignores the third value (`internal/layout/transform.go:1102-1103`). The `scale` property ignores the third value (`internal/layout/transform.go:1046-1047`). Feed that third value into the same Z side matrix as `translateZ` and `scaleZ`, and set `HasTransform3D`. A one-value or two-value form stays the 2D path that already passes. Expected: `translate: 0 0 100px` under an ancestor `perspective: 500px` bakes a different A than the same declaration without perspective. `scale: 1 1 2` under that ancestor also differs from `scale: 1 1`. Proof: two fixtures. Do not change the assertions in `TestBehaviorTranslatePropertyMovesPaint` or `TestBehaviorScalePropertyScalesPaint`.
- [ ] `ApplyRotate` accepts only one angle (`internal/layout/transform.go:983-1003`). The catalog syntax for `rotate` includes an axis form. Accept `x`, `y`, or `z` plus an angle, and the three-number axis plus an angle, by building the same side matrix as `rotateX`, `rotateY`, `rotateZ`, or `rotate3d`. A bare angle stays the 2D path `TestBehaviorRotatePropertyBakesMatrix` already pins. Expected: `rotate: y 60deg` bakes A=0.5 with no perspective, matching `rotateY(60deg)`. Proof: one fixture. The keyword list must match the pinned syntax on the `rotate` catalog row before the accept function is written.
- [ ] Pin `rotateX(60deg)` the way `rotateY(60deg)` is pinned. Orthographic bake D is 0.5 and A is 1. The current test only checks that some matrix was stamped (`internal/layout/css_behavior_perspective_test.go:137-139`). Proof: `TestBehaviorRotateXNarrowsHeight` in the gap file. If the stamp does not match, fix the flatten path. Do not replace the `rotateY` test.

Gate. The 2D tests still pass. The new forms set `HasTransform3D` and change a baked component. Run `go test ./internal/layout -run 'TestBehaviorTranslate|TestBehaviorScale|TestBehaviorRotate|TestBehaviorTranslate3d|TestTransform3D'`.

## Phase 3: layout proofs for functions the parser already accepts

These share the stamp path in Phase 0. No new parser unless a proof fails.

- [ ] `rotateZ(90deg)` bakes the same quarter turn as `rotate(90deg)`. Proof: compare A, B, C, D with `TestBehaviorRotatePropertyBakesMatrix`. If the side-matrix flatten does not match the 2D rotate, fix `rotateZ3D` or the flatten multiply. Do not leave it as parse-only.
- [ ] `rotate3d(0, 1, 0, 60deg)` bakes A=0.5, the same orthographic narrow as `rotateY(60deg)`. Proof: one fixture. A zero axis stays rejected (`internal/layout/transform_3d_test.go:94`).
- [ ] `scaleZ(2)` with no perspective does not stamp a matrix, same rule as identity depth. `scaleZ(2)` under `perspective: 500px` on an ancestor does change the bake. Proof: the two fixtures. The unit flatten for `translateZ` already shows orthographic A=1 and a perspective scale (`internal/layout/transform_3d_test.go:54-68`). This row is the layout stamp, which that unit test does not cover.
- [ ] `translateZ(250px)` under an ancestor `perspective: 500px` foreshortens. The unit test expects A=2 when the distance is twice the translation (`internal/layout/transform_3d_test.go:54-68`). The layout row uses a child box and reads the stamped A. `TestPerspRotate3DRejected` only covers `translateZ` without perspective, where no matrix is stamped. Keep that assertion.
- [ ] A non-identity `matrix3d` that is a Y rotation of 60 degrees bakes A=0.5. The identity `matrix3d` assertion in `TestPerspRotate3DRejected` stays: no stamped matrix. Proof: one non-identity fixture. Column-major parsing is `parseMatrix3DFunc` (`internal/layout/transform_3d.go:319-342`).

Gate. Each function either changes the stamped matrix in the way the row names, or the stamp bug it exposes is fixed in `transform_3d.go` or `boxTransformAccum`. Run `go test ./internal/layout -run 'TestBehaviorRotateZ|TestBehaviorScaleZ|TestBehaviorTranslateZ|TestBehaviorMatrix3d|TestPersp'`.

## Phase 4: backface of a composed child

- [ ] Write the failing probe first. A `preserve-3d` parent with `transform: rotateY(180deg)` and `backface-visibility: visible`, and a child with no transform of its own and `backface-visibility: hidden`. Expected if the composed chain is honored: the child's fill is zeroed, because the chain turns it around. Today `backfaceCulled` flattens only `boxNode.style.Transform3D` (`internal/layout/transform.go:1470`), so a child with an identity side matrix stays painted. Proof: `TestBehaviorBackfaceHiddenCullsComposedChild` records the current result before the fix, then expects the zeroed fill.
- [ ] Cull from the flattened chain when the child is inside an active `preserve-3d` chain. Use the facing `flattenChainForBox` already returns through `flatten3D` (`internal/layout/transform.go:1178-1185`). The chain's facing is the second return value of `flatten3D` (`internal/layout/transform_3d.go:181-183`). `backfaceCulled` has to see that facing, not a fresh flatten of the local matrix. A child with `backface-visibility: visible` stays painted. A 2D `rotate` parent still does not cull, matching `TestBehaviorBackfaceVisibilityNoPaintEffect`.
- [ ] Do not change `TestBackfaceHiddenCullRotateY180`. The element's own half turn still zeroes the fill.

Gate. The new test and the old backface tests pass. Run `go test ./internal/layout -run 'TestBackface|TestBehaviorBackface|TestPreserveBuild'`.

## Phase 5: comments, catalog, and gates

- [ ] Rewrite the stale comments listed in Phase 0 so they describe the 3D stamp. Do not change the assertions of those tests. Rename `TestPerspPerspectiveFunctionRejected` and `TestPerspRotate3DRejected` to names that match the bodies, and update the catalog `tests` entries that cite the old names (`perspective` cites both).
- [ ] Mark the four aliases `implemented` after Phase 1. Leave the 10 unprefixed rows `implemented`. Add the new gap tests to the `transform` row's test list if the catalog test pointer is how status is justified. Do not downgrade `perspective` because an old comment said it was dropped.
- [ ] Rewrite `documentation/deferred.md:115` and the group D sentence that still calls the four aliases print no-ops (`documentation/deferred.md:120`). State the baked-matrix contract, the host's job as drawing the matrix, and the functions Phase 2 added.
- [ ] `make claim-scan` must still reject any sentence that says the engine paints a page bitmap or matches Chrome pixel for pixel. The tests compare matrices and fill sizes, not a screenshot.
- [ ] Update the local knowledge-base page for transforms in the implementation session.
- [ ] Run `make test`. Exit 0.
- [ ] Run `make golden`. Exit 0.
- [ ] Run `make lint`. Exit 0. `internal/layout/layout.go` did not grow.
- [ ] Run `make claim-scan`. Exit 0.
- [ ] Run `make catalog-check`. Exit 0.
- [ ] Run `make matrix-check`. Exit 0.

GitHub CI runs only `make test` and `make lint`. Do not run bare `go test ./...`.

Gate. The six commands exit 0. The deferred page no longer says perspective is unimplemented.

## Dependencies

- Stamp path: `boxTransformAccum`, `propagatePerspective`, `parse3DFunc`. Phases 2 and 3 extend that path. They do not add a second matrix on the op.
- `transform-box` and `transform-origin` stay the origin inputs (`internal/layout/transform.go:1300`).
- The four aliases depend on the unprefixed properties, which already work.
- `translate3d` depends on the 2D percent-translate path (`internal/layout/transform.go:1266-1278`) for tx and ty.
- Composed backface depends on the preserve chain (`internal/layout/transform.go:1205-1231`). It does not depend on a page rasterizer.
- No new direct module.
