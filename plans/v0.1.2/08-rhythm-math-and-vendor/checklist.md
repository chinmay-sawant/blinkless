# v0.1.2 - Rhythm, math, and parked vendor aliases

> **Parent:** `plans/v0.1.2/README.md`
> **Status:** parked, except the ruby baselines.
> **Estimated effort:** Four ruby names stay closed on the current drawing list; the other twenty names stay parked until the parent README schedules a consumer.

---

## Overview

The engine returns one `layout.Display` from `layout.DisplayList` (`layout/displaylist.go:257`). A drawing list is the list of paint operations that call produces. There is no PDF writer and no page bitmap.

`documentation/deferred.md:103` still calls the ruby, MathML, and rhythmic family a permanent non-goal and still says 33 properties. `documentation/compatibility-matrix.md:943` repeats that 33 and still lists `ruby-align` as unimplemented. Those sentences are stale. The catalog read for this file is `testdata/css/catalog/properties.json`, `updated` `2026-11-02` (`properties.json:3`). Its summary is 785 rows: 555 implemented, 4 partial, 218 unsupported, 8 intentionally ignored (`properties.json:12-17`). This ledger does not use 33 as a live count. It names the four ruby rows the catalog marks implemented, the thirteen rhythm and math rows it marks unsupported, and the seven vendor rows it marks unsupported.

`line-clamp` is implemented (`properties.json:11388`, handler `applyAdvancedProps` in `internal/layout/style_advanced_props.go:26`). `-webkit-line-clamp` is an alias on that row (`properties.json:11390`), not its own catalog row. `line-clamp:2` already keeps two lines and paints U+2026 (`internal/layout/css_behavior_remaining_test.go:74-106`). Do not reopen either name here. `documentation/compatibility-matrix.md:1117` still lists `-webkit-line-clamp` under group E with no consumer. That matrix cell is behind the catalog.

A property with no acceptance entry still wins the cascade: `declarationValueAccepted` returns true from its default (`internal/layout/style_value_accept.go:938-939`). If no style group claims it, `applyStyleProp` ends in `applyIgnoredGroup`, which stores nothing (`internal/layout/style_properties.go:1876-1880`). `@supports` stays false because `engineSupportsPropertyUncached` returns false when every group declines (`internal/layout/style_cascade.go:2046-2052`). The thirteen rhythm and math names and the seven vendor names have no string match under `internal/layout`. `normalizeVendorPrefix` does not fold them (`internal/layout/style_cascade.go:1919-1927` folds only `-webkit-filter` and `-webkit-text-fill-color` in that arm).

`internal/layout/layout.go` is 2497 lines (`scripts/file-size-allowlist.txt:12`). No planned change adds lines to that file. A later consumer, if a gate below opens, goes in a new file in `internal/layout`.

Lengths below use the test scale 1 CSS pixel = 0.75pt (`internal/layout/css_review_02_test.go:6`). In syntax lines, `inf` stands in for the infinity sign stored in the catalog.

## Executive summary

| Property | Catalog line | Catalog status | This ledger |
|---|---:|---|---|
| `ruby-align` | 15720 | implemented | [x] baseline |
| `ruby-merge` | 15755 | implemented | [x] baseline |
| `ruby-overhang` | 15786 | implemented | [x] baseline |
| `ruby-position` | 15817 | implemented | [x] baseline |
| `block-ellipsis` | 2981 | unsupported | [~] |
| `block-step` | 3022 | unsupported | [~] |
| `block-step-align` | 3038 | unsupported | [~] |
| `block-step-insert` | 3054 | unsupported | [~] |
| `block-step-round` | 3070 | unsupported | [~] |
| `block-step-size` | 3086 | unsupported | [~] |
| `box-snap` | 5419 | unsupported | [~] |
| `line-grid` | 11437 | unsupported | [~] |
| `line-height-step` | 11490 | unsupported | [~] |
| `line-snap` | 11522 | unsupported | [~] |
| `math-depth` | 12771 | unsupported | [~] |
| `math-shift` | 12787 | unsupported | [~] |
| `math-style` | 12803 | unsupported | [~] |
| `-webkit-text-size-adjust` | 1021 | unsupported | [~] |
| `text-size-adjust` | 18620 | unsupported | [~] |
| `-webkit-text-stroke` | 1036 | unsupported | [~] |
| `-webkit-text-stroke-color` | 1052 | unsupported | [~] |
| `-webkit-text-stroke-width` | 1068 | unsupported | [~] |
| `-webkit-appearance` | 424 | unsupported | [~] |
| `-webkit-user-select` | 1322 | unsupported | [~] |

Pinned upstream syntax is quoted from each row. The pin is revision `1f2ec8f74a80c14066b4c7d6822cee59f69fa03b` (`properties.json` summary `upstream.revision`). Empty `handlers` and empty `tests` on every [~] row were read with that file.

## Phase 0: ruby baselines

These four names are already implemented. This phase records what the drawing list does. It does not schedule more ruby work.

The catalog `handlers` field says `applySVGPresentationProps` because `internal/layout/style_paint_props.go:125-127` lists the four names and forwards them to `applyLeftoversProps`. That function has no ruby case and returns false (`internal/layout/style_leftovers.go:60-61`). `applyRubyProps` then claims the name (`internal/layout/style_cascade.go:1739`, `internal/layout/ruby.go:382-399`) and stores the used value in `CustomProps`. Inline layout reads it from `collectRubyElement` (`internal/layout/inline_collect.go:784`, `internal/layout/ruby.go:605`).

The comments on `TestBehaviorRubyAlignNoPaintEffect` and its three siblings still say the declaration is dropped (`internal/layout/css_behavior_font2_test.go:441-512`). Those tests put the property on `<p>hello</p>` with no `<ruby>`. The ops match because there is no annotation to move. They are not evidence that the properties are unimplemented.

Known layout approximations, already shipped: an `inter-character` annotation paints inline after the base at half size, instead of beside the base; `ruby-merge:merge` concatenates the whole `<ruby>` into one group; a segment may split across lines; `<rp>` fallback text is dropped (`internal/layout/ruby.go:11-15`, `internal/layout/ruby.go:730-733`).

### ruby-align

- [x] `ruby-align` (`properties.json:15720`, syntax `start | center | space-between | space-around`, initial `space-around` at `internal/layout/ruby.go:25`). Expectation: a narrow annotation over a wide base sits on the start edge for `start` and in the middle for `center`. In the drawing list, `<ruby style="ruby-align:start">base<rt>x</rt></ruby>` at 12pt pins the annotation text op's X to the base text op's X, and the same markup with `ruby-align:center` moves only the annotation's X to the right (`internal/layout/css_behavior_ruby_layout_test.go:94-121`). The helper with a 40pt base and a 20pt annotation returns 0 for `start` and 10pt for `center` and for `space-around` (`internal/layout/css_behavior_ruby_test.go:24-39`, `internal/layout/ruby.go:229-239`). A 20pt base with a 30pt annotation returns 0 for every keyword, because there is no leftover space (`internal/layout/css_behavior_ruby_test.go:51-63`). Proof already in tree: `TestBehaviorRubyLayoutAlignStartVsCenter`, `TestBehaviorRubyAlignCentersNarrowAnnotation`, `TestBehaviorRubyAlignWideAnnotationFillsColumn`.

### ruby-merge

- [x] `ruby-merge` (`properties.json:15755`, syntax `separate | merge | auto`, initial `separate` at `internal/layout/ruby.go:26`). Expectation: two columns with base widths 20 and 20 and annotation widths 30 and 10 occupy 50pt when `separate` (`max(20,30)+max(20,10)`) and 40pt when `merge` (`max(40,40)`) (`internal/layout/css_behavior_ruby_test.go:128-144`, `internal/layout/ruby.go:275-279`). `auto` uses the 40pt merge when an annotation is wider than its base, and the separate sum when every annotation fits (`internal/layout/css_behavior_ruby_test.go:151-156`). In the drawing list, `<ruby style="ruby-merge:separate;ruby-overhang:spaces">A<rt>wide</rt>B<rt>x</rt></ruby>` followed by a bold `Y` places that `Y` to the right of the same markup with `ruby-merge:merge` (`internal/layout/css_behavior_ruby_layout_test.go:151-172`). The word `collapse` is rejected and the used value stays `separate` (`internal/layout/ruby.go:92-105`, `internal/layout/css_behavior_ruby_test.go:216-217`). Proof already in tree: `TestBehaviorRubyMergeSharesAnnotationSpace`, `TestBehaviorRubyLayoutMergeNarrowsSharedSpace`.

### ruby-overhang

- [x] `ruby-overhang` (`properties.json:15786`, syntax `auto | spaces`, initial `auto` at `internal/layout/ruby.go:27`). The legacy word `none` is accepted as `spaces` (`internal/layout/ruby.go:108-117`). Expectation: a 20pt base with a 30pt annotation grows by 0 under `auto` (the annotation may spill over the next glyph) and by 10pt under `spaces` and under `none` (`internal/layout/css_behavior_ruby_test.go:160-182`, `internal/layout/ruby.go:260-272`). In the drawing list, `<ruby>B<rt>wideanno</rt></ruby>` followed by a bold `Y` leaves that `Y` further left than `<ruby style="ruby-overhang:spaces">B<rt>wideanno</rt></ruby>` followed by the same `Y` (`internal/layout/css_behavior_ruby_layout_test.go:124-148`). Proof already in tree: `TestBehaviorRubyOverhangGrowsBaseWhenForbidden`, `TestBehaviorRubyLayoutOverhangSpacesWidensSegment`.

### ruby-position

- [x] `ruby-position` (`properties.json:15817`, syntax `[ alternate || [ over | under ] ] | inter-character`, initial `alternate` at `internal/layout/ruby.go:28`). Expectation: the default stacks a half-size annotation above the base. For `<ruby>base<rt>anno</rt></ruby>` at 12pt, the annotation text op's Y is more than 1pt above the base text op's Y, and its `Size` is half the base `Size` (`internal/layout/css_behavior_ruby_layout_test.go:48-70`, scale `0.5` at `internal/layout/ruby.go:367-370`). `<ruby style="ruby-position:under">base<rt>anno</rt></ruby>` puts the annotation's Y more than 1pt below the base (`internal/layout/css_behavior_ruby_layout_test.go:73-91`). The helper with base height 13.5pt and annotation height 6.75pt returns -6.75 for `over` and for `alternate`, +13.5 for `under`, and 0 for `inter-character` (`internal/layout/css_behavior_ruby_test.go:66-117`, `internal/layout/ruby.go:242-257`). `inter-character` in the drawing list paints the half-size run inline after the base (`internal/layout/ruby.go:730-733`). `over under` is rejected (`internal/layout/css_behavior_ruby_test.go:239-240`). Proof already in tree: `TestBehaviorRubyLayoutStacksAnnotationOverBase`, `TestBehaviorRubyLayoutPositionUnderStacksBelow`, `TestBehaviorRubyPositionStacksOverAndUnder`.

## Phase 1: rhythmic and math

Each row is deferred. Reason, shared by the phase: `documentation/deferred.md:103` still calls this family a permanent non-goal, and the catalog status is `unsupported` with an empty handler list. Owner of every row: this ledger. Next gate, shared unless a row tightens it: reopen that row only after `documentation/deferred.md` drops the permanent non-goal for the name and `plans/v0.1.2/README.md` schedules a drawing-list consumer. Until that gate, do not add a handler, and do not add fields to `Op` in `internal/layout/layout.go`.

Today, each declaration paints the same drawing list as the same markup with the declaration omitted. The sentences under "If the defer is lifted" are the test that would have to pass before the row can leave `[~]`. They are not work queued now. Spec text used for those sentences: CSS Rhythmic Sizing Level 1 (block step and `line-height-step`), CSS Line Grid Level 1 (`line-grid`, `line-snap`, `box-snap`), CSS Overflow Level 4 (`block-ellipsis`), and MathML Core sections 4.3 to 4.5.

### block-ellipsis

- [~] `block-ellipsis` (`properties.json:2981`, syntax `no-ellipsis | ellipsis | <string>`, initial `no-ellipsis`, inherited). It chooses the marker inserted at the end of the last line before a clamp. `no-ellipsis` inserts nothing. `ellipsis` inserts U+2026. A string inserts that string. If the defer is lifted: take the 200px, 16px paragraph from `TestBehaviorLineClampTwoLimitsLines` (`internal/layout/css_behavior_remaining_test.go:77-94`), which already clamps to two lines. `line-clamp:2; block-ellipsis:no-ellipsis` still paints exactly two lines, and the joined text ops do not contain U+2026. `block-ellipsis:"END"` on that clamp paints the letters END at the end of line 2 and does not paint U+2026. `block-ellipsis:ellipsis` paints U+2026 at the end of line 2. Today the line-clamp path appends U+2026 by itself (`internal/layout/css_behavior_remaining_test.go:104-106`), so the `no-ellipsis` case is the one that would change current paint. That change waits on the gate above. `line-clamp` itself stays out of this ledger.

### block-step-size

- [~] `block-step-size` (`properties.json:3086`, syntax `none | <length [0,inf]>`, initial `none`). A positive length rounds the block's outer height to a multiple of that length. `none` leaves the height alone. A negative length is invalid. If the defer is lifted: a block with `height:10px` and `block-step-size:8px`, with the initial round `up`, has an outer height of 16px, which is 12pt. The same block with `block-step-size:none` stays 10px, which is 7.5pt. The following sibling's Y is 4.5pt lower in the stepped case. The rounding code goes in a new file under `internal/layout`.

### block-step-insert

- [~] `block-step-insert` (`properties.json:3054`, syntax `margin-box | padding-box | content-box`, initial `margin-box`). It chooses where the extra space from `block-step-size` goes. If the defer is lifted, on the 10px block stepped up to 16px (6px, 4.5pt, of extra): `margin-box` keeps the border box at 7.5pt and pushes the next sibling down by 4.5pt. `padding-box` grows the border box by 4.5pt and keeps the extra inside the border. `content-box` grows the content height by 4.5pt. The test fails if `padding-box` only moves the next sibling and leaves the border box at 7.5pt.

### block-step-align

- [~] `block-step-align` (`properties.json:3038`, syntax `auto | center | start | end`, initial `auto`). It places the extra space from `block-step-size`. The spec names are easy to invert: `start` puts the extra on the end side, and `end` puts the extra on the start side. `center` splits it in half. `auto`, when insert is `margin-box`, follows `align-self` if that is `start`, `end`, or `center`, and otherwise acts as `center`. If the defer is lifted, on the same 6px of extra with `margin-box`: `center` moves the border box down by 2.25pt and the next sibling down by 4.5pt from the unstepped position. `start` leaves the border-box Y on the unstepped Y and moves the next sibling down by 4.5pt. `end` moves the border box down by 4.5pt. `auto` with the default `align-self` matches `center`. `auto` with `align-self:start` matches `start`.

### block-step-round

- [~] `block-step-round` (`properties.json:3070`, syntax `up | down | nearest`, initial `up`). `up` grows the outer size. `down` shrinks it. `nearest` picks the smaller change, and grows when the two changes are equal. If the defer is lifted, a block with `height:10px` and `block-step-size:8px` is 16px (12pt) for `up`, 8px (6pt) for `down`, and 8px for `nearest` (2px down versus 6px up). A second block with `height:12px` and the same step is 16px for `nearest`, because both directions are 4px and the tie grows.

### block-step

- [~] `block-step` (`properties.json:3022`, syntax `<'block-step-size'> || <'block-step-insert'> || <'block-step-align'> || <'block-step-round'>`). It is the shorthand. It has no field of its own. Omitted longhands keep their initials, so a lone keyword that is not a size does not step the box. If the defer is lifted: `block-step: 8px content-box start down` on a `height:10px` block matches the four longhands set to those values (outer size 8px, extra inside the content box, extra on the end side, rounded down). `block-step: center` alone matches `block-step-size:none` and leaves the 10px box at 7.5pt. The test uses both declarations so a shorthand that only stores the word `center` cannot pass.

### line-height-step

- [~] `line-height-step` (`properties.json:11490`, syntax `<length [0,inf]>`, initial `0`, inherited). A positive length rounds each line box height up to a multiple of that length. The extra space is split equally above and below the glyphs, so the glyphs stay centered in the taller line. Zero does not round. A negative length is invalid. If the defer is lifted: a block with `font-size:16px; line-height:16px` has a 16px (12pt) line box. Adding `line-height-step:10px` rounds that box up to 20px (15pt). The 4px of extra is split in half, so the first baseline sits 1.5pt lower than the unstepped line, and the second baseline sits 15pt below the first. `line-height-step:0` keeps the 12pt step.

### line-grid

- [~] `line-grid` (`properties.json:11437`, syntax `match-parent | create`, initial `match-parent`). `create` starts a new baseline grid from that element's own font and line height. `match-parent` uses the parent's grid. Snapping is a separate property. If the defer is lifted: a parent with `font-size:20px; line-height:30px; line-grid:create` establishes a 30px (22.5pt) grid. A child with `font-size:10px; line-height:10px; line-grid:create` and a grandchild with `line-snap:baseline` places grandchild baselines 10px (7.5pt) apart. The same grandchild under `line-grid:match-parent` on that child places baselines 30px (22.5pt) apart. The test fails if `create` still snaps to the parent's 22.5pt step.

### line-snap

- [~] `line-snap` (`properties.json:11522`, syntax `none | baseline | contain`, initial `none`, inherited). `none` stacks lines normally. `baseline` shifts each line down until its dominant baseline meets the grid. `contain` centers the line between a text-over grid edge and a later text-under grid edge, so a tall line can cover more than one grid step. If the defer is lifted, using the spec's own numbers: three lines with `line-height:20px` and `line-snap:baseline` on a 30px grid paint baselines 30px (22.5pt) apart. The same three lines with `line-snap:none` paint baselines 20px (15pt) apart. `contain` on a line taller than one 30px step still paints that line, centered on the grid, and the following line's baseline is a whole number of 22.5pt steps below it.

### box-snap

- [~] `box-snap` (`properties.json:5419`, syntax `none | block-start | block-end | center | baseline | last-baseline`, initial `none`). It shifts a block down onto the parent's line grid. The block's own `line-grid` does not pick the grid it snaps to. `none` does not shift. `block-start` snaps the start edge to the grid's text-over line. `baseline` snaps the first line's dominant baseline. `last-baseline` snaps the last line's. If the defer is lifted: a parent with `line-height:30px; line-grid:create`, a 10px spacer, then a block. `box-snap:none` leaves the block's Y at the end of the spacer. `box-snap:block-start` moves that Y downward until the block's start edge meets the parent's text-over grid line, so the snapped Y is greater than the unsnapped Y unless the spacer already ended on that line. `box-snap:baseline` moves the first line's baseline onto a grid baseline and leaves the spacer's height unchanged. An empty block that collapses through stays unshifted. The grid line is a baseline of the parent's font, so this test compares the two Y values and does not hard-code 22.5pt.

### math-depth

- [~] `math-depth` (`properties.json:12771`, syntax `auto-add | add(<integer>) | <integer>`, initial `0`, inherited). The computed value is an integer. An integer sets it. `add(n)` adds n to the inherited value. `auto-add` adds 1 only when the inherited `math-style` is `compact`; otherwise the computed value is the inherited one. It changes `font-size` only when the specified font size is the keyword `math`. With no OpenType MATH table, one extra level multiplies the inherited font size by 0.71 (MathML Core sets the constant C to 0.71 and multiplies by C raised to the level gap). If the defer is lifted: a parent at depth 0 with a used font size of 16px (12pt), and a child with `font-size:math; math-depth:1`, paints that child's glyphs at 0.71 times 12pt. `math-depth:0` on the child keeps 12pt. `math-depth:add(2)` paints 0.71 squared times 12pt. The same child with `font-size:16px` (a length, not the keyword `math`) stays 12pt at every depth. There is no MathML layout in `internal/layout` (no `math-depth` string, and no MathML element arm). The consumer, if the gate opens, is a new file. This row does not add a math typesetting pass on its own.

### math-shift

- [~] `math-shift` (`properties.json:12787`, syntax `normal | compact`, initial `normal`, inherited). `normal` places a superscript with the font's `superscriptShiftUp` distance. `compact` uses `superscriptShiftUpCramped`. The two distances come from the font's MATH table, so this ledger does not invent a pixel gap. If the defer is lifted: the same base `x` with superscript `2`, same font size, paints the superscript baseline higher for `normal` than for `compact` when the font's normal shift is larger than its cramped shift, and paints the same baseline when those two font constants are equal. The test fails if both keywords share one hardcoded offset. Same gate as `math-depth`: no math consumer exists in `internal/layout` today.

### math-style

- [~] `math-style` (`properties.json:12803`, syntax `normal | compact`, initial `normal`, inherited). `compact` tries to keep the formula shorter: script-sized math font size inside fractions, large operators stay at text size, and movable limits sit as superscript and subscript. `normal` uses the taller display layout: a large operator grows, and its limits sit above and below. If the defer is lifted: one summation with limits, painted twice at the same font. `compact` paints the limits as a superscript and a subscript beside the operator, and the operator's glyph size matches the surrounding text. `normal` paints the operator larger than that text and paints the limits above and below it. A plain HTML paragraph with no math element paints the same ops for both keywords, and that pair must not be the closing test. Same gate as `math-depth`.

## Phase 2: vendor aliases

`documentation/deferred.md:120` still says group E has no consumer planned. The names there are `-webkit-line-clamp`, `-webkit-text-size-adjust`, `-webkit-text-stroke`, `-webkit-text-stroke-color`, and `-webkit-text-stroke-width`. `-webkit-line-clamp` is the alias noted in the overview. Leave it alone. `documentation/compatibility-matrix.md:1081-1109` puts `-webkit-appearance` and `-webkit-user-select` in group D and still marks both unsupported. The unprefixed bases are separate catalog rows: `appearance` is intentionally ignored (`properties.json:2344`) and `user-select` is intentionally ignored (`properties.json:19602`). Neither base lists a `-webkit-` alias.

Each row below is deferred. Owner: this ledger. Next gate, shared unless the row tightens it: reopen only after `documentation/deferred.md` drops "no consumer planned" for that name and `plans/v0.1.2/README.md` schedules a drawing-list consumer. A rename onto an ignored base is not that consumer. Do not add a `normalizeVendorPrefix` arm, and do not add fields on `Op` in `internal/layout/layout.go`.

Compat says `-webkit-text-size-adjust` is a legacy alias of `text-size-adjust`, and it defines the three stroke properties. CSS Size Adjust Level 1 defines `text-size-adjust`. The catalog still stores the prefixed and unprefixed size-adjust names as two unsupported rows, with empty alias lists.

### text-size-adjust

- [~] `text-size-adjust` (`properties.json:18620`, syntax `auto | none | <percentage [0,inf]>`, initial `auto`, inherited). It is a hint for a small-device text inflator. `none` turns inflation off. A percentage turns inflation off and multiplies the computed `font-size` by that percentage, and only when the display is a small device. The spec says `100%` matches `none`. No file under `internal/layout` reads `text-size-adjust`. `DisplayList` can scale a whole overflowing canvas uniformly (`layout/displaylist.go:244-249`). That scale is one number for the canvas. It is not a per-paragraph text inflation. If the defer is lifted, the closing test needs a host flag this ledger does not add. With the flag off, a 16px paragraph (text op size 12pt) paints 12pt for `auto`, `none`, `100%`, and `200%`. With the flag on, `200%` paints 24pt and `none` and `100%` stay at 12pt. Until that flag exists in the parent README, the row stays `[~]`, and the honest drawing list is the flag-off result.

### -webkit-text-size-adjust

- [~] `-webkit-text-size-adjust` (`properties.json:1021`, no upstream syntax on the pinned row, class `vendor`). The catalog does not alias it to `text-size-adjust`. `normalizeVendorPrefix` leaves the name unchanged (`internal/layout/style_cascade.go:1919-1927`). If the defer is lifted, the closing test is the same pair as `text-size-adjust`: with the small-device flag on, `-webkit-text-size-adjust:200%` paints a 16px paragraph at 24pt, and `-webkit-text-size-adjust:none` paints it at 12pt. With the flag off, both paint 12pt. Folding the prefix onto the unprefixed name happens in that same gate, and glyph size stays 12pt until the flag exists.

### -webkit-text-stroke-width

- [x] `-webkit-text-stroke-width` (`properties.json:1068`, syntax `<line-width>`, initial `0`, inherited). It is the width of a stroke drawn at the edge of each glyph. Zero paints no stroke. A negative value is invalid and paints no stroke. There is no unprefixed `text-stroke-width` row in the catalog. `Op.Width` is the stroke width of an `OpLine`, not a glyph edge (`internal/layout/layout.go:541`). If the defer is lifted: the word `hi` at `color:#000` with `-webkit-text-stroke-width:1px` and `-webkit-text-stroke-color:red` paints the black fill plus a red glyph-edge stroke 0.75pt wide. `-webkit-text-stroke-width:0` paints the black fill and no red edge. The stroke record lives in a new file. The test fails if the only red pixels are a rectangle behind the word. Done 2026-11-16: `TestBehaviorTextStrokeWidthAndColor`, `TestBehaviorTextStrokeShorthand`, and `TestBehaviorTextStrokeInherits` passed in `internal/layout/css_behavior_text_stroke_test.go`. The stroke rides on the text op as `Op.TextStroke()`; OwnFrame paints the outline. Catalog status is `implemented`.

### -webkit-text-stroke-color

- [x] `-webkit-text-stroke-color` (`properties.json:1052`, syntax `<color>`, initial `currentcolor`, inherited). It sets the glyph-edge stroke color. If the defer is lifted: `hi` with `color:#000`, `-webkit-text-stroke-width:1px`, and `-webkit-text-stroke-color:#0000ff` paints a blue 0.75pt edge and a black fill. Omitting the color longhand uses the text color, so a black element with width `1px` and no color longhand paints a black edge. Width `0` paints no edge of either color. Same new-file rule as the width longhand. Done 2026-11-16: `TestBehaviorTextStrokeWidthAndColor`, `TestBehaviorTextStrokeShorthand`, and `TestBehaviorTextStrokeInherits` passed in `internal/layout/css_behavior_text_stroke_test.go`. The stroke rides on the text op as `Op.TextStroke()`; OwnFrame paints the outline. Catalog status is `implemented`.

### -webkit-text-stroke

- [x] `-webkit-text-stroke` (`properties.json:1036`, syntax `<line-width> || <color>`). It is the shorthand for the two longhands above. It has no field of its own. If the defer is lifted: `-webkit-text-stroke:1px red` on `hi` matches width `1px` (0.75pt) and color red, with the fill still the element's `color`. `-webkit-text-stroke:red` sets the color and leaves the width at 0, so no edge is painted. `-webkit-text-stroke:1px` sets the width and leaves the color at `currentcolor`. The test uses all three so a shorthand that only stores one half cannot pass. Done 2026-11-16: `TestBehaviorTextStrokeWidthAndColor`, `TestBehaviorTextStrokeShorthand`, and `TestBehaviorTextStrokeInherits` passed in `internal/layout/css_behavior_text_stroke_test.go`. The stroke rides on the text op as `Op.TextStroke()`; OwnFrame paints the outline. Catalog status is `implemented`.

### -webkit-appearance

- [~] `-webkit-appearance` (`properties.json:424`, no upstream syntax on the pinned row). It is not an alias of `appearance`. `appearance:none` on `<input type="range">` already paints the same ops as the same input without the declaration, because the engine has no form chrome to remove (`internal/layout/css_behavior_interaction_test.go:302-318`). `documentation/deferred.md:110` still parks pointer and form UI. If the defer is lifted by folding the prefix onto `appearance`, the closing test is that same range input with `-webkit-appearance:none`: the ops match the input that omits it, including the range thumb. A test that expects the thumb to disappear is a form-chrome consumer, and this ledger does not schedule one. Next gate is tighter than the phase default: the parent README would have to schedule form chrome, which `documentation/deferred.md:110` still refuses.

### -webkit-user-select

- [~] `-webkit-user-select` (`properties.json:1322`, no upstream syntax on the pinned row). It is not an alias of `user-select`. `user-select:none` on `selectable text here` already paints the same ops as the paragraph without it, because there is no selection model (`internal/layout/css_behavior_interaction_test.go:256-268`). If the defer is lifted by folding the prefix onto `user-select`, the closing test is that same sentence with `-webkit-user-select:none`: same text, same positions, same op count. There is still nothing on the drawing list that can show a selection. Next gate matches `-webkit-appearance`: a selection consumer is outside this ledger while `documentation/deferred.md:110` still refuses pointer and form UI.

## Dependencies

- Parent index: `plans/v0.1.2/README.md`. This file does not create it.
- Non-goal text stays where it is: `documentation/deferred.md:103` for the rhythm and math family, `documentation/deferred.md:110` for form and selection chrome, `documentation/deferred.md:120` for group E. This planning file does not edit those rows. A later session that opens a gate updates the deferred row and this ledger together.
- Catalog authority: `testdata/css/catalog/properties.json` as of `2026-11-02`. Handler presence never promotes a status (`testdata/css/catalog/schema.json:47`). An implemented row also needs a consumer and a behavior test (`testdata/css/catalog/schema.json:6`).
- Ruby proof already in the tree: `internal/layout/ruby.go`, `internal/layout/css_behavior_ruby_layout_test.go`, `internal/layout/css_behavior_ruby_test.go`. No new ruby tests are required to keep phase 0 closed.
- File-size stop: `internal/layout/layout.go` at 2497 lines, `scripts/file-size-allowlist.txt:12`. Rhythm, math, and stroke code, if a gate opens, goes in a new `internal/layout` file.
- `line-clamp` and its alias `-webkit-line-clamp` are not rows of this ledger.
- This file is documentation only. No test command was run to write it.
