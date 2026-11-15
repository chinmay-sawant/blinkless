# v0.1.2 - Inline text leftovers

> **Parent:** `plans/v0.1.2/README.md`
> **Status:** not started. Baseline rows record behavior already asserted by current tests.
> **Estimated effort:** Five properties, each one stored value and one drawing-list test, in the order below.

---

## Overview

The engine output is `layout.DisplayList` (`doc.go:3`, `layout/displaylist.go:257`). One canvas. No PDF file and no page PNG (`doc.go:4`).

A text op is one painted run: kind, X, Y, width, and the string it carries. A fill rect is one painted rectangle. The tests below read those fields the same way the existing text tests do (`internal/layout/css_behavior_text_test.go:8-12`).

These five catalog rows are `unsupported`. A search of `internal/layout` for each name found no apply handler and no test. No opened test asserts them, so every row in this file stays open.

Work order is the quickest real text change first: `baseline-source`, `line-fit-edge`, `line-padding`, `inline-sizing`, `word-space-transform`.

`alignment-baseline`, `dominant-baseline`, `fill`, `fill-rule`, and `shape-rendering` are already implemented. This file does not replan them. They are the detail standard: a concrete input, the box or paint result, and the test name.

## Executive summary

| Property | Catalog | What changes in the drawing list | Where the new code lives |
|---|---|---|---|
| `baseline-source` | `unsupported`, syntax `auto \| first \| last` | First line of a two-line inline-block shares Y with the neighboring text | `internal/layout/inline_baseline_source.go` |
| `line-fit-edge` | `unsupported`, syntax `leading \| <text-edge>` | A padded span with `text` drops the text op; `leading` leaves the padding out of the line | `internal/layout/inline_line_fit.go` |
| `line-padding` | `unsupported`, syntax `<length>` | Each line's text op starts further right by that length | `internal/layout/inline_line_padding.go` |
| `inline-sizing` | `unsupported`, syntax `normal \| stretch` | The span's red fill grows to the line height; the text op Y stays | `internal/layout/inline_sizing.go` |
| `word-space-transform` | `unsupported`, syntax `none \| [ space \| ideographic-space ] && auto-phrase?` | A zero-width space becomes a real space and the text op gets wider | `internal/layout/inline_word_space.go` |

Shared rules for every property:

- Add the field on `ResolvedStyle` and its initial in `initialStyle` (`internal/layout/style.go:649`). Regenerate `internal/layout/style_intern_gen.go` with the `go:generate` line on `internal/layout/style.go:2`. The generated file compares fields one by one (`internal/layout/style_intern_gen.go:291` is the `WordSpacing` pair).
- Register the new apply function in `styleGroups` (`internal/layout/style_cascade.go:1702`). `make catalog-check` only sees a property whose case label sits in a function reached from that list (`scripts/css-catalog-map.py:346-371`).
- The inherit table already holds 64 entries and the mask is full (`internal/layout/style_cascade.go:581-586`). Do not append to `inheritableProps`. Copy an inherited value in a side function called from `inheritProps`, the same shape as `inheritGapBProps` (`internal/layout/style_cascade.go:588-611`).
- A token outside the grammar leaves the previous value. That is the rule in `applyAlignmentBaselineProperty` (`internal/layout/style_leftovers.go:316-321`).
- Do not add lines to `internal/layout/layout.go` or `internal/layout/inline_paint.go`. Both are on the size ledger (`scripts/file-size-allowlist.txt:11-12`). A call from an existing function is allowed. The body lives in the new file.

## Phase 1: Five text properties

### 1.1 baseline-source

Catalog row `testdata/css/catalog/properties.json:2965-2978`. Status `unsupported`. `values` is empty. `upstream_syntax` is `auto | first | last`. `upstream_class` is `draft`. Handlers and tests are empty. The limitation says there is no apply handler and the declaration is ignored. Pinned upstream revision `1f2ec8f74a80c14066b4c7d6822cee59f69fa03b`. Webref href `https://drafts.csswg.org/css-inline-3/#propdef-baseline-source` (`testdata/css/catalog/upstream/webref-ed-css-1f2ec8f.json:1074-1075`).

CSS Inline 3: the property picks the first baseline set or the last baseline set when an inline-level box has more than one. Initial `auto`. It does not inherit. `auto` means last for `inline-block`, and first for every other box. `first` means the first baseline. `last` means the last baseline. It applies to inline-level boxes that establish an independent formatting context.

Current behavior: a search of `internal/layout` found no `baseline-source` apply handler. `collectInlineBlockItem` lays out the inline-block and, when it has children, treats the bottom border edge as the baseline (`internal/layout/inline_collect.go:824-848`). `atomicInlineAlign` then puts that border edge on the line: ascent is the top margin plus the border-box height, and descent is the bottom margin (`internal/layout/inline.go:1690-1710`). `emitLine` stores the first line's baseline on the block the first time it is zero (`internal/layout/inline.go:1310-1312`). The inline-block placer does not read that stored value. `alignedInlineTop` moves the box to the line (`internal/layout/inline_vertical_align.go:22-43`).

What to expect, nearest pattern `alignment-baseline`. `TestSvgTextAlignmentBaselineCentralIdeographic` lays out `<p style="margin:0;font-size:16px">Ag</p>` and checks that `alignment-baseline:central` raises the text op and `ideographic` lowers it (`internal/layout/css_behavior_svg_text_test.go:122-153`). The shift is `alignmentBaselineShift` (`internal/layout/inline_vertical_align.go:165-186`).

Input:

```html
<p style="margin:0;font-size:16px">go
  <span style="display:inline-block;baseline-source:first">Ag<br>Ag</span>
</p>
```

`baseline-source:first` puts the first `Ag` text op on the same Y as the `go` text op. `baseline-source:last` puts the second `Ag` on that Y, and the first `Ag` sits one line higher. `auto` on this inline-block matches `last`. A bad token leaves the used value at `auto` and matches the `last` placement.

- [ ] Acceptance: store `auto`, `first`, and `last` on the inline-block. `auto` selects the last baseline for `inline-block`. `first` aligns the first line's text baseline with the parent line. `last` aligns the last line's text baseline with the parent line. The two-line sample above is the proof.
- [ ] Storage: field on `ResolvedStyle`, initial `auto` in `initialStyle` (`internal/layout/style.go:830` is the `AlignmentBaseline` initial beside it). The property does not inherit, same as `alignment-baseline` (`internal/layout/style_leftovers.go:314`). Do not copy it in `inheritGapBProps`. `dominant-baseline` does get that copy (`internal/layout/style_cascade.go:601-603`); this property must not.
- [ ] Consumer: new file `internal/layout/inline_baseline_source.go`. `collectInlineBlockItem` calls it. The helper returns the baseline Y inside the inline-block (first line, last line, or the current bottom border edge when the box has no line). `alignedInlineTop` keeps doing the move. Do not add this logic to `layout.go`.
- [ ] Behavior test: package `layout`, file `internal/layout/css_behavior_inline_text_test.go`, test `TestBehaviorBaselineSourceFirstAlignsFirstLine`. Assert the first `Ag` text op Y equals the `go` text op Y for `first`, and the second `Ag` text op Y equals the `go` text op Y for `last` and for `auto`.
- [ ] Catalog flip: in `testdata/css/catalog/properties.json`, set `values` to `auto | first | last`, `status` to `implemented`, `handlers` to the new apply function, `tests` to the test above, and `source` to the new file. Remove the no-handler limitation.
- [~] Flex baseline export still uses `firstBaseline` only (`internal/layout/flex.go:1754-1758`). Reason: this phase proves the keyword on inline-block text ops. Owner: this ledger. Next gate: a flex test where `baseline-source:last` on a multi-line flex item moves `baselineOffset` off `firstBaseline`. Until that test exists, the catalog limitation can say flex items still export the first baseline.

Gate: `go test ./internal/layout -run 'TestBehaviorBaselineSourceFirstAlignsFirstLine'`

### 1.2 line-fit-edge

Catalog row `testdata/css/catalog/properties.json:11421-11434`. Status `unsupported`. `values` is empty. `upstream_syntax` is `leading | <text-edge>`. `upstream_class` is `draft`. Handlers and tests are empty. The limitation says there is no apply handler and the declaration is ignored. Same pinned upstream revision. Webref href `https://drafts.csswg.org/css-inline-3/#propdef-line-fit-edge` (`testdata/css/catalog/upstream/webref-ed-css-1f2ec8f.json:4162-4163`).

CSS Inline 3: this picks which font edges size an inline box's contribution to the line. Initial `leading`. It inherits. `leading` uses ascent and descent plus any positive half-leading, and the span's margin, border, and padding do not change the line height. Any other value uses that text edge, drops positive half-leading on that span, and does add the span's margin, border, and padding to the line. A single keyword sets both edges when it can; `cap` and `ex` are over edges, `alphabetic` is an under edge, and the missing side is `text`.

Current behavior: a search of `internal/layout` found no `line-fit-edge` apply handler. For a text item with inline chrome, `lineMetrics` takes font ascent and descent, adds half of (`line-height` minus that ascent and descent), then adds padding and border on the top and bottom (`internal/layout/inline.go:1612-1637`). That half is the half-leading. Chrome is turned on for a static inline box (`internal/layout/inline_collect.go:247-271` and `internal/layout/inline_collect.go:1083-1090`). So a padded span already makes the line taller. `text-box-edge` can retarget cap, ex, and alphabetic, and only while trimming (`internal/layout/inline_text_box.go:45-78`).

What to expect, nearest pattern `text-box-edge` for the metric and `line-height` for the line height. `TestBehaviorLineHeightExplicitEnlargesBox` shows a 12pt paragraph with `line-height:24pt` lays out 24pt tall, taller than the normal sibling (`internal/layout/css_behavior_text_test.go:35-54`). `TestBehaviorTextBoxEdgeRetargetsTrimmedMetrics` is the catalog behavior test for `text-box-edge` (`testdata/css/catalog/properties.json:17879-17881`). Reuse `fontCapHeightFace` and `fontExHeightFace` (`internal/layout/inline_text_box.go:66-72`). Do not invent a second cap ratio.

Input:

```html
<p style="margin:0;font-size:16px">
  <span style="padding-top:20px;line-fit-edge:leading">Ag</span> Z
</p>
```

The same markup with `line-fit-edge:text` instead of `leading`. Both words sit on one line. The `Z` text op Y is lower for `text` than for `leading`. The drop is at least 10pt: `text` keeps the 20px (15pt) padding in the line, and `leading` leaves it out. The drop is not exactly 15pt, because `text` also drops positive half-leading on that span. `text` matches what the current chrome path already does with the padding.

- [ ] Acceptance: store `leading` and a text edge of one or two keywords from `text`, `ideographic`, `ideographic-ink`, `cap`, `ex`, and `alphabetic`, with `cap` and `ex` only on the over side and `alphabetic` only on the under side. `leading` stops the span's padding and border from growing the line. Any other stored value keeps that padding in the line and drops positive half-leading on that span. The block's own zero-width line contribution, from its font and `line-height`, still uses `line-height` (`internal/layout/inline.go:1578-1583`).
- [ ] Storage: field on `ResolvedStyle`, initial `leading` in `initialStyle`. Inherited, so the side inherit function copies it when the element does not declare it. Do not append to `inheritableProps`.
- [ ] Consumer: new file `internal/layout/inline_line_fit.go`. `lineMetrics` calls it in place of the unconditional chrome add at `internal/layout/inline.go:1634-1637`. Cap and ex heights come from the helpers in `internal/layout/inline_text_box.go`. Do not add this logic to `layout.go`.
- [ ] Behavior test: package `layout`, file `internal/layout/css_behavior_inline_text_test.go`, test `TestBehaviorLineFitEdgeTextKeepsPadding`. Assert the `Z` text op Y for `line-fit-edge:text` with `padding-top:20px` is at least 10pt below the same op for `line-fit-edge:leading`.
- [ ] Catalog flip: set `values` to `leading | <text-edge>`, `status` to `implemented`, `handlers` to the new apply function, `tests` to the test above, and `source` to the new file. Remove the no-handler limitation.

Gate: `go test ./internal/layout -run 'TestBehaviorLineFitEdgeTextKeepsPadding'`

### 1.3 line-padding

Catalog row `testdata/css/catalog/properties.json:11506-11519`. Status `unsupported`. `values` is empty. `upstream_syntax` is `<length>`. `upstream_class` is `draft`. Handlers and tests are empty. The limitation says there is no apply handler and the declaration is ignored. Same pinned upstream revision. Webref href `https://drafts.csswg.org/css-text-4/#propdef-line-padding` (`testdata/css/catalog/upstream/webref-ed-css-1f2ec8f.json:4199-4200`).

CSS Text 4: a length added at the start and at the end of each line, between the innermost inline box's content edge and the neighboring text or atomic inline. Initial `0`. It inherits. It is not a justification opportunity. The spec's own example is `line-padding: 0.5em` on a paragraph, which shows extra background on both sides of every line.

Current behavior: a search of `internal/layout` found no `line-padding` apply handler. The first line can already move right: when `TextIndent` is non-zero, `lineX` increases and `lineW` decreases (`internal/layout/inline.go:390-397`). That runs only for `idx == 0`. Later lines stay at the content edge. `text-indent` is stored in `applyTextSpacingProps` (`internal/layout/style_properties.go:1603-1604`).

What to expect, nearest pattern `text-indent`. `TestBehaviorTextIndentShiftsFirstLine` lays out `hello` at 12pt with `text-indent:24pt` and requires the text op X to move right by 24pt (`internal/layout/css_behavior_text_test.go:115-128`).

Input:

```html
<p style="margin:0;font-size:16px;line-padding:24px">Hello</p>
```

The `Hello` text op X is 18pt to the right of the same paragraph with no `line-padding` (24px times 0.75). The text op width stays the width of `Hello`. A second line in a narrow box starts at that same inset. `text-indent` would have moved only the first line (`internal/layout/inline.go:390-397`).

- [ ] Acceptance: store a length in points. Initial `0`. The innermost inline at each end of a line inserts that length between its content edge and the text. Both ends lose that much from the line's available width. The extra gap is not a justify gap.
- [ ] Storage: `float64` field on `ResolvedStyle`, initial `0`, next to `TextIndent` (`internal/layout/style.go:322`). Inherited, so the side inherit function copies it when the element does not declare it. Do not append to `inheritableProps`.
- [ ] Consumer: new file `internal/layout/inline_line_padding.go`. The line loop calls it for every line, start and end. The first-line indent at `internal/layout/inline.go:390-397` stays as it is. Do not add this logic to `layout.go`.
- [ ] Behavior test: package `layout`, file `internal/layout/css_behavior_inline_text_test.go`, test `TestBehaviorLinePaddingInsetsText`. Assert the `Hello` text op X moves right by 18pt, and a wrapped second line starts at that same X.
- [ ] Catalog flip: set `values` to `<length>`, `status` to `implemented`, `handlers` to the new apply function, `tests` to the test above, and `source` to the new file. Remove the no-handler limitation.

Gate: `go test ./internal/layout -run 'TestBehaviorLinePaddingInsetsText'`

### 1.4 inline-sizing

Catalog row `testdata/css/catalog/properties.json:10899-10912`. Status `unsupported`. `values` is empty. `upstream_syntax` is `normal | stretch`. `upstream_class` is `draft`. Handlers and tests are empty. The limitation says there is no apply handler and the declaration is ignored. Same pinned upstream revision. Webref href `https://drafts.csswg.org/css-inline-3/#propdef-inline-sizing` (`testdata/css/catalog/upstream/webref-ed-css-1f2ec8f.json:3941-3942`).

CSS Inline 3: this sets how tall the inline box's own content area is. Initial `normal`. It inherits. It applies to inline boxes, and not to ruby container boxes or internal ruby boxes. `normal` sizes that area to the first available font. `stretch`, after the line is sized and the glyphs are placed, moves the margin edges so they meet the line's top and bottom. The glyphs do not move. `line-height` does not set the inline box's own height.

Current behavior: a search of `internal/layout` found no `inline-sizing` apply handler. `paintInlineChrome` paints the background from font ascent, descent, half the `line-height` gap, and padding (`internal/layout/inline_paint.go:244-266`). The fill is an `OpFillRect` at that rectangle. Ruby annotations are reserved outside the baseline row (`internal/layout/inline.go:1605-1609`).

What to expect, nearest pattern `background-color` for the fill and `line-height` for the line the fill grows to. `TestBehaviorBackgroundColorPaintsFill` lays out a 100px by 50px red div and requires a red fill at that box (`internal/layout/css_behavior_borders_test.go:498-515`). `TestBehaviorLineHeightExplicitEnlargesBox` is the line-height proof cited above.

Input:

```html
<p style="margin:0;line-height:normal">
  <span style="font-size:32px">Big</span>
  <span style="font-size:12px;line-height:normal;background:#ff0000;inline-sizing:stretch">sm</span>
</p>
```

The red fill rect is as tall as the line made by `Big`. The same span with `inline-sizing:normal` (or with the property left off) paints a shorter red fill, about the 12px font. The `sm` text op Y is the same in both. The letters stay on the baseline.

- [ ] Acceptance: store `normal` and `stretch`. `normal` keeps today's font-sized background. `stretch` makes the inline background's top and bottom meet the line box after the glyphs are placed. The text op Y does not change. Do not stretch a ruby annotation (`internal/layout/inline.go:1605-1609`).
- [ ] Storage: field on `ResolvedStyle`, initial `normal` in `initialStyle`. Inherited, so the side inherit function copies it when the element does not declare it. Do not append to `inheritableProps`.
- [ ] Consumer: new file `internal/layout/inline_sizing.go`. `paintInlineChrome` calls it for the fill height. The stretch math does not live in `inline_paint.go` (size ledger, `scripts/file-size-allowlist.txt:11`). Do not add this logic to `layout.go`.
- [ ] Behavior test: package `layout`, file `internal/layout/css_behavior_inline_text_test.go`, test `TestBehaviorInlineSizingStretchGrowsFill`. Assert the red `OpFillRect` height equals the line height for `stretch`, is shorter for `normal`, and the `sm` text op Y matches in both.
- [ ] Catalog flip: set `values` to `normal | stretch`, `status` to `implemented`, `handlers` to the new apply function, `tests` to the test above, and `source` to the new file. Remove the no-handler limitation.

Gate: `go test ./internal/layout -run 'TestBehaviorInlineSizingStretchGrowsFill'`

### 1.5 word-space-transform

Catalog row `testdata/css/catalog/properties.json:20294-20307`. Status `unsupported`. `values` is empty. `upstream_syntax` is `none | [ space | ideographic-space ] && auto-phrase?`. `upstream_class` is `draft`. Handlers and tests are empty. The limitation says there is no apply handler and the declaration is ignored. Same pinned upstream revision. Webref href `https://drafts.csswg.org/css-text-4/#propdef-word-space-transform` (`testdata/css/catalog/upstream/webref-ed-css-1f2ec8f.json:7503-7504`).

CSS Text 4: this replaces word separators in the element's text. Initial `none`. It inherits. `none` changes nothing. `space` replaces each expandable separator with U+0020. `ideographic-space` replaces each with U+3000. Expandable separators are U+200B, a `wbr` element, and a virtual separator from phrase detection. `auto-phrase` asks for phrase detection. If the engine has no detector for that language, the spec says there are no virtual separators. `auto-phrase` alone is outside the grammar, because `space` or `ideographic-space` is required. Do not replace a separator that sits against a forced line break.

Current behavior: a search of `internal/layout` found no `word-space-transform` apply handler and no `wbr` reader. The HTML parser does insert a `wbr` element (`internal/html/tree.go:921-923`) and treats it as void (`internal/html/tree.go:2042-2044`). A search of `internal/layout` found no special case for U+200B. `Op.Text` copies the run text after soft hyphens are stripped (`internal/layout/inline_paint.go:377-379`). `word-spacing` adds advance after each ordinary space (`internal/layout/style_properties.go:1597-1601`).

What to expect, nearest pattern `word-spacing`. `TestBehaviorWordSpacingWidensText` lays out `hello world` at 12pt with `white-space:nowrap` and `word-spacing:4pt`, and requires the text ops' total width to grow by at least 3.5pt (`internal/layout/css_behavior_text_test.go:74-88`).

Input:

```html
<p style="margin:0;font-size:16px;white-space:nowrap;word-space-transform:space">A&#x200b;B</p>
```

The text op is wider than the same paragraph with `word-space-transform:none`, by about one space. Its text contains U+0020 and does not contain U+200B, because the replacement happens before measurement and `Op.Text` copies that string (`internal/layout/inline_paint.go:377-379`). `ideographic-space` puts U+3000 in the text op instead. `space auto-phrase` on `HelloWorld` adds no extra character, because this engine has no phrase detector.

`<wbr>` between `A` and `B` with `word-space-transform:space` produces the same wider text op. With `none`, the `wbr` adds no space.

- [ ] Acceptance: store `none`, `space`, `ideographic-space`, and either of those two plus `auto-phrase`. `space` replaces U+200B and `wbr` with U+0020. `ideographic-space` replaces them with U+3000. `auto-phrase` inserts nothing, because there is no phrase detector. A separator next to a forced break stays as it is. A bare `auto-phrase` token leaves the previous value.
- [ ] Storage: field on `ResolvedStyle`, initial `none` in `initialStyle`. Inherited, so the side inherit function copies it when the element does not declare it. Do not append to `inheritableProps`.
- [ ] Consumer: new file `internal/layout/inline_word_space.go`. The text collector calls it before `measureTextFace`, and the `wbr` element is visited there. Do not add this logic to `layout.go`.
- [ ] Behavior test: package `layout`, file `internal/layout/css_behavior_inline_text_test.go`, test `TestBehaviorWordSpaceTransformSpaceWidensRun`. Assert `space` makes the `A&#x200b;B` text op wider than `none` and stores U+0020, `ideographic-space` stores U+3000, `space auto-phrase` leaves `HelloWorld` one word wide, and `<wbr>` under `space` widens the run.
- [ ] Catalog flip: set `values` to `none | [ space | ideographic-space ] && auto-phrase?`, `status` to `implemented`, `handlers` to the new apply function, `tests` to the test above, and `source` to the new file. Remove the no-handler limitation. A limitation may say `auto-phrase` inserts no virtual separators because this engine has no phrase detector.

Gate: `go test ./internal/layout -run 'TestBehaviorWordSpaceTransformSpaceWidensRun'`

## Dependencies

This ledger does not implement the properties. It does not create `plans/v0.1.2/README.md`.

Order is the table in the executive summary. `baseline-source` is first because the inline-block already records a first-line baseline and already places the bottom border edge. `line-fit-edge` is next because `lineMetrics` already adds padding and `text-box-edge` already knows cap and ex. `line-padding` is an inset on every line, beside the first-line indent. `inline-sizing` changes a fill height after the line exists. `word-space-transform` is last because it rewrites characters and has to read `wbr`, which layout does not read today.

Shared storage, inherit-mask, `styleGroups` registration, and the size ledger are in the executive summary. Four properties inherit (`line-fit-edge`, `line-padding`, `inline-sizing`, `word-space-transform`). `baseline-source` does not.

Per-property proof is the `go test ./internal/layout -run '...'` command on that section. Do not run those tests while this file is only a plan.

Full closure, after the rows are implemented and the tests pass:

`make test`, `make golden`, `make lint`, `make claim-scan`, `make catalog-check`, `make matrix-check`.

`make catalog-check` reads `testdata/css/catalog/properties.json` and does not rewrite statuses (`Makefile:114-120`). `make matrix-check` fails if `documentation/compatibility-matrix.md` drifts from `python3 scripts/css-catalog-map.py --matrix` (`Makefile:127-132`). A catalog flip updates the matrix section in the same change.
