# v0.1.2 - Validation plan for the first ten styles

> **Parent:** `plans/v0.1.2/README.md`
> **Status:** active. A catalog row stays `unsupported` until the named test exits 0.
> **Estimated effort:** ten styles, one fixture and one behavior test each, then one OwnFrame window pass.

---

## Overview

Blinkless returns one drawing list (`layout.DisplayList`). OwnFrame paints that list in a window. `Redraw` parses the HTML, applies the CSS, and calls `layout.DisplayListOptions` (`../ownframe/api.go:8-9`, `../ownframe/internal/render/document.go:37`). `Run` opens the window (`../ownframe/run.go:15-18`).

The proof is a Go test in this repo. The window is the picture a person checks after the test is green. A catalog row in `testdata/css/catalog/properties.json` becomes `implemented` only after that test exits 0. A failed or skipped test leaves the row `unsupported`. Do not use `partial` for these ten.

OwnFrame's module file requires blinkless `v0.1.1` (`../ownframe/go.mod:6`). A window launched from that requirement shows the released engine. The viewing branch `validation/blinkless-next-phase` in the sibling OwnFrame checkout adds this repo to its `go.work` so the window uses this checkout. Fixtures stay in this repo. OwnFrame receives the HTML string. The fixture is not copied into OwnFrame.

`make catalog-ownframe-check` already builds a temporary workspace with both checkouts and runs the motion tests (`scripts/check-css-ownframe-integration.py:46-57`). Those tests do not cover these ten styles.

## What a pass is

Each style has three artifacts:

1. `testdata/css/v012/<slug>/page.html`. The same file is the test input and the window input.
2. One test in `internal/layout`, named below. It reads that file, lays it out with `layoutHTML` (`internal/layout/layout_test.go:41-58`), and checks a used value in CSS pixels. One CSS pixel is 0.75 points (`internal/layout/css_behavior_box_model_test.go:8-20`).
3. The catalog row. Status `implemented`, filled `values`, the handler name, and this test. Set that row only after the test command prints ok.

The test command during the work is one package:

```
go test ./internal/layout -run '<TestName>' -count=1 -p 1
```

Do not run bare `go test ./...`. After all ten tests are green, one integration pass runs `make test`, `make golden`, `make lint`, `make claim-scan`, `make catalog-check`, and `make matrix-check`.

New code does not go in `internal/layout/layout.go` or `internal/layout/inline_paint.go`. Both are on the size ledger (`scripts/file-size-allowlist.txt`).

## The ten styles

| Style | Fixture slug | Test | What the number means | What the window shows |
|---|---|---|---|---|
| `-webkit-background-clip` | `webkit-background-clip` | `TestBehaviorWebkitBackgroundClipClipsImage` | Image box matches `TestBehaviorBackgroundClipClipsImage` (`internal/layout/css_behavior_backgrounds_test.go:232-269`) | Background stops at the content box |
| `-webkit-background-origin` | `webkit-background-origin` | `TestBehaviorWebkitBackgroundOriginInsetsImage` | Image origin matches `TestBehaviorBackgroundOriginInsetsImage` (`internal/layout/css_behavior_backgrounds_test.go:193-225`) | Image starts at the padding edge |
| `-webkit-background-size` | `webkit-background-size` | `TestBehaviorWebkitBackgroundSizeScalesImage` | Image box matches `TestBehaviorBackgroundSizeScalesImage` (`internal/layout/css_behavior_backgrounds_test.go:132-153`) | `40px 20px` paints a 40 by 20 image |
| `zoom` | `zoom` | `TestBehaviorZoomDoublesBorderBox` | `zoom: 2` on a 100 by 40 box is 200 by 80 | The box is twice as wide and twice as tall |
| `baseline-source` | `baseline-source` | `TestBehaviorBaselineSourceFirstAlignsFirstLine` | `first` aligns the first line baseline with the neighbor. `last` aligns the second line | The two-line box sits higher or lower |
| `line-fit-edge` | `line-fit-edge` | `TestBehaviorLineFitEdgeTextDropsPadding` | `text` keeps the span padding out of the line height. `leading` includes it | The line is shorter or taller |
| `inline-sizing` | `inline-sizing` | `TestBehaviorInlineSizingStretchGrowsFill` | `stretch` grows the span fill to the line height. The text op Y stays | A red span fills the line |
| `line-padding` | `line-padding` | `TestBehaviorLinePaddingInsetsEveryLine` | Each line's text X moves right by the length | Every line starts further in |
| `box-decoration-break` | `box-decoration-break` | `TestBehaviorBoxDecorationBreakClonePaintsEachFragment` | `clone` paints one background per wrapped fragment. `slice` paints one | A wrapped highlight is one bar or two |
| SVG geometry | `svg-geometry` | `TestBehaviorSvgStylesheetGeometryMovesShape` | A stylesheet `cx`, `cy`, `r`, `rx`, `ry`, `x`, `y`, or `d` changes the image the way the inline style already does | The circle, ellipse, rect, or path moves |

`zoom` here is the CSS property on one element. `Options.Zoom` is a different control. It scales the whole document (`internal/layout/layout_faces.go:265-272`). The element property must not write that field.

## Result on 2026-11-15

All ten tests passed:

```
go test ./internal/layout -count=1 -p 1 -run 'TestBehaviorWebkitBackgroundClipClipsImage|TestBehaviorWebkitBackgroundOriginInsetsImage|TestBehaviorWebkitBackgroundSizeScalesImage|TestBehaviorZoomDoublesBorderBox|TestBehaviorBaselineSourceFirstAlignsFirstLine|TestBehaviorLineFitEdgeTextDropsPadding|TestBehaviorInlineSizingStretchGrowsFill|TestBehaviorLinePaddingInsetsEveryLine|TestBehaviorBoxDecorationBreakClonePaintsEachFragment|TestBehaviorSvgStylesheetGeometryMovesShape'
```

The catalog now counts 572 implemented and 201 unsupported. Each of these rows is `implemented`. None was left `partial`.

`make catalog-check` still fails on this machine for reasons outside these ten styles: seven OwnFrame motion case strings no longer match the sibling checkout, and `wrap-inside` names a handler `wrapBreakInsideAvoid` that the checker does not resolve. `make matrix-check` matches section 2 after the catalog rewrite.

## Result on 2026-11-16 (third ten)

The third batch is `marker`, `marker-start`, `marker-mid`, `marker-end`, `stop-color`, `stop-opacity`, `path-length`, `paint-order`, `link-parameters`, and `continue`. Each has a behavior test and a catalog row now `implemented`:

| Style | Test |
|---|---|
| `marker`, longhands | `TestBehaviorSvgPresentMarker`, `TestBehaviorSvgPresentMarkerStart`, `TestBehaviorSvgPresentMarkerMid`, `TestBehaviorSvgPresentMarkerEnd`, `TestBehaviorSvgPresentMarkerAllThreeVertices` |
| `stop-color`, `stop-opacity` | `TestBehaviorSvgPresentStopColor`, `TestBehaviorSvgPresentStopOpacity` |
| `paint-order` | `TestBehaviorSvgPresentPaintOrder` |
| `path-length` | `TestBehaviorSvgPresentPathLength`, `TestEmulateShapeLength` |
| `link-parameters` | `TestBehaviorLinkParametersRecolorsSvg`, `TestBehaviorLinkParametersLeavesAnchorURI` |
| `continue` | `TestBehaviorContinueCollapseDropsLinesPastMaxHeight`, `TestBehaviorContinueDiscardDropsLinesPastHeight`, `TestBehaviorContinueWebkitLegacyClampsVerticalBox` |

The catalog counts 592 implemented and 185 unsupported. The OwnFrame window shows them as items 21 to 30 of the review page. The window pass found one bug the pixel counts missed: the canvas misplaced every marker after the first on one shape, fixed by writing one copy per marker (`splitSVGMarkerShape`).

## Result on 2026-11-16 (fourth batch, five styles)

| Style | Test | Note |
|---|---|---|
| `margin-break` | `TestBehaviorMarginBreakDiscardDropsColumnMargin`, `TestBehaviorMarginBreakKeepKeepsColumnMargin` | Column starts only. Catalog `partial`. |
| `image-rendering` | `TestBehaviorSvgPresentImageRendering` | The keyword is read from the image op. No resampler. |
| `vector-effect` | `TestBehaviorSvgPresentVectorEffect` | `non-scaling-stroke` keeps the stroke width under a viewBox scale. |
| `flood-color`, `flood-opacity` | `TestBehaviorSvgPresentFloodColorAndOpacity` | A filter that is one `feFlood` becomes a flat rect. |

The catalog counts 596 implemented, 1 partial, and 180 unsupported. `lighting-color` stayed unsupported: the canvas has no lighting filter and the ledger says not to fake one.

## Result on 2026-11-16 (text stroke)

`-webkit-text-stroke`, `-webkit-text-stroke-color`, and `-webkit-text-stroke-width` are implemented. The tests read `Op.TextStroke()` from the drawing list. OwnFrame paints the outline in both its window and bitmap renderers by repeating the run around each glyph under the fill, so the outline is about half as heavy as a centered browser stroke. The catalog counts 599 implemented, 1 partial, and 177 unsupported.

## Window pass

After the ten tests exit 0:

1. On the OwnFrame branch `validation/blinkless-next-phase`, `go.work` includes this checkout.
2. A small preview reads `testdata/css/v012/<slug>/page.html` from this repo and calls `ownframe.New` then `ownframe.Run`.
3. Open one slug at a time. Compare the picture to the table above.
4. If the picture disagrees with the test number, fix the test or the paint before marking the catalog row.

If the window cannot open in this session, the report names the slug, the command, and the number the person should see.

## Catalog rule

`implemented` requires a consumer outside the style layer and a behavior test (`testdata/css/catalog/properties.json` seed `conservative_rule`). For these ten, that test is the one in the table. Update `documentation/compatibility-matrix.md` in the same change as the catalog row, by regenerating the property tables with `python3 scripts/css-catalog-map.py --matrix` when the implementation pass edits the matrix. Do not hand-edit a status that the test did not prove.
