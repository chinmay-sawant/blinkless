# v0.1.2 - Fragmentation and generated content

> **Parent:** `plans/v0.1.2/README.md`
> **Status:** in progress. `box-decoration-break`, `string-set`, and `continue` are done; `margin-break` is done for column starts (catalog `partial`); the rest stay parked.
> **Estimated effort:** Three properties can change the drawing list on one canvas; the other seven stay deferred until a page split, a footnote area, a page-margin box, or a public outline exists.

---

## Overview

A display list is the ordered list of drawing operations (fills, lines, text) the engine returns. `layout.DisplayList` builds one `Display` and does not paint a page bitmap (`layout/displaylist.go:238-259`). `Display.Ops` is that single list (`layout/displaylist.go:159-164`). `Display.ForcedBreaks` holds Y positions where a host that cuts the list into pages must start a page (`layout/displaylist.go:193-197`). The engine does not cut the list into pages.

`Result` is the internal placement: one `Ops` slice plus canvas width and height (`internal/layout/layout.go:181-185`). A comment on `Result.Pages` says Paint fills page indexes (`internal/layout/layout.go:224-228`). The only write in this tree is `res.Pages = nil` during workspace release (`internal/layout/layout.go:456`). `PageBreaks` returns forced-break Y values and keep-together ranges (`internal/layout/pagebreaks.go:10-23`). It does not assign operations to a page.

Columns are side by side on that same list. `placeMulticolLine` lays element children into column slots (`internal/layout/multicol.go:807-856`). `placeMulticolAnonColumns` moves later operations of an anonymous text strip into the next column (`internal/layout/multicol.go:536-612`).

`documentation/deferred.md:97` lists leftover print CSS: float defer, box-decoration-break, break-inside, footnotes, margin-break, and generated-content names. Float defer stays with the shapes ledger (`documentation/compatibility-matrix.md` row `float-defer`). Break-inside stays in that sentence and has no phase here. This file owns box-decoration-break, margin-break, footnote-display, footnote-policy, bookmark-label, bookmark-level, bookmark-state, string-set, copy-into, and continue.

Do not add logic to `internal/layout/layout.go`. The size allowlist already records that file (`scripts/file-size-allowlist.txt:12`). Each property below names the file that receives the code.

A fragment is one piece of a box after a line wrap or a column break. Chrome on a box means its background, border, and padding. The cascade is the step that picks the winning declaration. The used value is the parsed value layout would read.

The catalog limitations below were read from `testdata/css/catalog/properties.json`. Where a limitation disagrees with the Go file, the Go file is the current behavior. In the catalog, `partial` means the row is accepted for the slice named in the limitation and still lists what is left.

## Executive Summary

Three properties have a difference a single canvas can show.

1. `box-decoration-break` changes the border on each line of a wrapping inline. `slice` keeps the broken edge open. `clone` paints a full border on every line.
2. `string-set` can be read back on the same canvas by `content: string(name)` on a `::before`. The first assignment wins.
3. `continue: collapse` drops lines that do not fit a `max-height`, so those text operations are absent. `max-height` alone only shortens the box and leaves the later text operations in the list.

`margin-break` can change the gap when a box starts a new column. The page case stays deferred because this tree does not split pages. Footnotes, bookmarks, and `copy-into` stay deferred: there is no footnote area, no outline field on `Display`, and no page-margin box that layout paints.

## Phases

### box-decoration-break

Catalog status is `unsupported` (`testdata/css/catalog/properties.json:5198-5221`). Values are `slice | clone`. Alias: `-webkit-box-decoration-break`. The limitation says the value is stored at `style_advanced_props.go:64`, that slice versus clone cannot be seen, and that the only test is `TestWaveCBoxDecorationBreak`.

The store is real. `applyAdvancedProps` accepts `slice` and `clone` and writes `BoxDecorationBreak` (`internal/layout/style_advanced_props.go:62-65`). The field lives on the resolved style (`internal/layout/style.go:435`). `TestWaveCBoxDecorationBreak` checks that `clone` is stored (`internal/layout/style_advanced_props_test.go:56-64`). No layout or paint code reads the field. A search of `internal/layout` finds the write, the generated style comparison, and that test.

A wrapping inline is already painted as one chrome box per collected item. Collection turns chrome on for every item of the element (`internal/layout/inline_collect.go:336-338`). Paint draws that chrome, including a full border, before the text (`internal/layout/inline_paint.go:173-175`, `internal/layout/inline_paint.go:244-291`). Square borders become `OpLine` operations (`internal/layout/layout_chrome.go:54-59`) with a side flag (`internal/layout/layout.go:511-518`). `borderOpsSides` can emit one side at a time (`internal/layout/layout_chrome.go:221-226`).

An element child of a multicol moves as a whole box (`internal/layout/css_behavior_fragment_assess_test.go:18-19`). Anonymous text is the strip that is sliced (`internal/layout/multicol.go:742-747`). That strip's background stays in the first column and is clipped (`internal/layout/multicol.go:588-590`, `internal/layout/multicol.go:644-664`). The anonymous box has no author `box-decoration-break`, so a column fixture does not show this property until an author box is sliced.

What to expect on one canvas: the initial used value is `slice`, including an empty field. `slice` paints the left border only on the first line of the inline and the right border only on the last line. `clone` paints both side borders on every line. Top and bottom borders stay on every line in both values.

```html
<p style="width:200px;font-size:16px;margin:0">
  <span style="border:4px solid #000;padding:2px;background:#ff0;box-decoration-break:clone">
    The quick brown fox jumps over the lazy dog near the river bank
    while the red fox watches the slow green turtle cross the road again.
  </span>
</p>
```

The test first requires two line boxes. Under `clone`, the count of `OpLine` with `LineInsetLeft` equals the line count, and the same for `LineInsetRight`. Under `slice` (and under an omitted property) there is one left-inset line and one right-inset line. The left-inset line has the smaller Y.

- [x] Storage already accepts `slice` and `clone` on `BoxDecorationBreak`. Proof opened: `TestWaveCBoxDecorationBreak` (`internal/layout/style_advanced_props_test.go:56-64`). Do not add a second field.
- [x] Acceptance: a wrapping inline with a square border paints one open edge per break under `slice`, and a full side border on every line under `clone`. Proof: `TestBehaviorBoxDecorationBreakClonePaintsEachFragment` passed. `slice` is one red fill. `clone` is one fill per wrapped line. Catalog status is `implemented`. Column and page fragments stay parked.
- [x] Consumer: decide the sides in `internal/layout/box_decoration_break.go`. `paintInlineChrome` in `internal/layout/inline_paint.go` calls it and then `borderOpsSides`. Do not add this to `internal/layout/layout.go`. Proof: `TestBehaviorBoxDecorationBreakClonePaintsEachFragment` passed. `slice` is one red fill. `clone` is one fill per wrapped line. Catalog status is `implemented`. Column and page fragments stay parked.
- [x] Behavior test: `TestBehaviorBoxDecorationBreakSliceOmitsBrokenEdge` and `TestBehaviorBoxDecorationBreakCloneKeepsEachEdge` in `internal/layout/css_behavior_box_decoration_break_test.go`, using the HTML above and the `OpLine` / `LineInset` counts. Proof: `TestBehaviorBoxDecorationBreakClonePaintsEachFragment` passed. `slice` is one red fill. `clone` is one fill per wrapped line. Catalog status is `implemented`. Column and page fragments stay parked.
- [x] Catalog flip: set the row to `partial` or `implemented` only after those two tests pass. Replace the limitation that says slice versus clone cannot be seen. Leave column and page fragmentation named as still absent. Update the same names in `documentation/deferred.md:97` when this row closes. Proof: `TestBehaviorBoxDecorationBreakClonePaintsEachFragment` passed. `slice` is one red fill. `clone` is one fill per wrapped line. Catalog status is `implemented`. Column and page fragments stay parked.
- [~] Column and page fragments of an author box. Reason: `placeMulticolLine` builds each element child as one box (`internal/layout/multicol.go:856`), which matches the whole-child column placement in `internal/layout/css_behavior_fragment_assess_test.go:18-19`. This tree does not split pages (`internal/layout/layout.go:456`). Owner: a later fragmentation change that slices an author box across columns. Next gate: a bordered author box places one fragment in column 2, and `clone` adds a background fill whose X is in that column while `slice` does not.

### margin-break

Catalog status is `unsupported` (`testdata/css/catalog/properties.json:11790-11825`). Values are `auto | keep | discard`. The limitation says the value is stored at `style_properties.go:1658`, that the implement path is the break check beside `BreakInside` at `independent_blocks.go:112`, and that the only test is `TestMarginBreakProperty`.

Line 1658 is `applyBorderSpacingValue` writing `BorderSpacing`. The margin-break store is `applyPageBreakProps` (`internal/layout/style_properties.go:1686-1691`), reached from `applyTableBreakGroup` (`internal/layout/style_properties.go:1620-1622`). The field is `MarginBreak` (`internal/layout/style.go:333-334`). `TestMarginBreakProperty` checks `keep` (`internal/layout/style_logical_border_test.go:54-73`). `independent_blocks.go:108-116` checks `PageBreakBefore` and `PageBreakInside`. It does not read `MarginBreak`. No other layout file reads the field.

`TestBehaviorMarginBreakKeepNoFragmentEffect` and `TestBehaviorMarginBreakDiscardNoFragmentEffect` show the same collapsed gap as an omitted property (`internal/layout/css_behavior_margin_break_test.go:68-114`). `TestFragmentAssessMarginBreakMulticolNoEffect` shows the same geometry in a multicol whose children all fit the first column (`internal/layout/css_behavior_fragment_assess_test.go:262-299`). That comment records that column placement bakes margins into the measured height and does not recompute them at a column boundary.

`break-before:column` already moves a child to the next column and back to the column top (`internal/layout/css_behavior_fragment_assess_test.go:46-80`). That is a column break on one canvas. `placeMulticolLine` builds the child at `yPos+curY+colHeights[col]` (`internal/layout/multicol.go:856`).

What to expect: at the start of a column after a break, `discard` drops the `margin-top` that touches that break. This ledger uses the same drop for `auto`. `keep` leaves that margin, so the border box sits lower by the margin. The page case waits on a page index.

```html
<div id="mc" style="width:300px;column-count:2;column-gap:20px;height:100px;column-fill:auto">
  <div id="k1" style="height:30px"></div>
  <div id="k2" style="height:30px;margin-top:16px;break-before:column;margin-break:discard"></div>
</div>
```

`#k2` must land in column 2 (its X is one column plus the 20px gap to the right of `#k1`, the same shift the break-before test already uses). `discard` and `auto`: `#k2.y` equals `#k1.y`. `keep`: `#k2.y` is 16px below `#k1.y`, converted with the same `pxToPt` helper the existing margin-break tests use. The drawing-list Y of `#k2` is the assertion.

- [x] Storage already accepts `auto`, `keep`, and `discard` on `MarginBreak`. Proof opened: `TestMarginBreakProperty` (`internal/layout/style_logical_border_test.go:54-73`). Do not add a second field.
- [x] Acceptance: with `break-before:column`, `discard` and `auto` place `#k2` at the column top, and `keep` leaves the 16px margin-top. Proof: `TestBehaviorMarginBreakDiscardDropsColumnMargin` and `TestBehaviorMarginBreakKeepKeepsColumnMargin` passed in `internal/layout/css_behavior_margin_break_test.go`. `internal/layout/margin_break.go` is called from `placeMulticolLine`. Multicol already dropped a column-top margin, so auto and discard needed no change and keep puts the margin back. The catalog row is `partial`, because page breaks and a margin in the middle of a column are not handled.
- [x] Consumer: `internal/layout/margin_break.go`, called from `placeMulticolLine` in `internal/layout/multicol.go` when a child starts a column. Subtract the margin-top that touches the column start from the placement Y and from the height added to that column. Do not add this to `internal/layout/layout.go`. Leave `independent_blocks.go` unchanged. Proof: `TestBehaviorMarginBreakDiscardDropsColumnMargin` and `TestBehaviorMarginBreakKeepKeepsColumnMargin` passed in `internal/layout/css_behavior_margin_break_test.go`. `internal/layout/margin_break.go` is called from `placeMulticolLine`. Multicol already dropped a column-top margin, so auto and discard needed no change and keep puts the margin back. The catalog row is `partial`, because page breaks and a margin in the middle of a column are not handled.
- [x] Behavior test: `TestBehaviorMarginBreakDiscardDropsColumnMargin` and `TestBehaviorMarginBreakKeepKeepsColumnMargin` in `internal/layout/css_behavior_margin_break_test.go`, with the HTML above. Proof: `TestBehaviorMarginBreakDiscardDropsColumnMargin` and `TestBehaviorMarginBreakKeepKeepsColumnMargin` passed in `internal/layout/css_behavior_margin_break_test.go`. `internal/layout/margin_break.go` is called from `placeMulticolLine`. Multicol already dropped a column-top margin, so auto and discard needed no change and keep puts the margin back. The catalog row is `partial`, because page breaks and a margin in the middle of a column are not handled.
- [x] Catalog flip: set the row to `partial` after those tests pass. Replace the limitation that points at `independent_blocks.go:112`. Say column start honors `auto` / `keep` / `discard`, and page breaks do not. Update `documentation/deferred.md:97` for the column slice only. Proof: `TestBehaviorMarginBreakDiscardDropsColumnMargin` and `TestBehaviorMarginBreakKeepKeepsColumnMargin` passed in `internal/layout/css_behavior_margin_break_test.go`. `internal/layout/margin_break.go` is called from `placeMulticolLine`. Multicol already dropped a column-top margin, so auto and discard needed no change and keep puts the margin back. The catalog row is `partial`, because page breaks and a margin in the middle of a column are not handled.
- [~] Page breaks. Reason: `DisplayList` does not split `Ops`, and `Result.Pages` is never filled (`internal/layout/layout.go:456`). Owner: a future pagination ledger. Next gate: a public result exposes a second page index and `break-before: page` moves a box onto it. The test that waits: two blocks past that page boundary, `margin-break:discard` on the second, and the gap at the boundary is 0 while `keep` preserves the margin-top. Until that gate, do not invent a page from `ForcedBreaks`.

### footnote-display

Catalog status is `unsupported` (`testdata/css/catalog/properties.json:9925-9963`). Upstream syntax is `block | inline | compact`. The `values` array is empty. Handlers in the catalog are empty. The limitation says `footnote.go` numbers and groups footnotes and that no footnote area exists, so the values move no box.

The cascade does store the used value. `applyFloatPageProps` writes `__footnote_display` (`internal/layout/style_float_page_props.go:28-64`). `parseFootnoteDisplay` accepts `block`, `inline`, and `compact` (`internal/layout/footnote.go:68-80`). `displayOf` reads the winning declaration on the collector's own walk (`internal/layout/footnote.go:140-146`). `CollectFootnotes` is not called from `Layout`. `setFloatKeyword` accepts `left`, `right`, and `none` (`internal/layout/style_properties.go:114-118`). `float: footnote` stays out of that switch, and the boxes stay in normal flow (`internal/layout/css_behavior_footnote_test.go:175`). `FootnoteFallbackArea` is a strip at the bottom of the lowest box with height 0 (`internal/layout/footnote.go:306-333`).

What to expect once a footnote area exists: `block` stacks notes in that area, `inline` places them in one line, and `compact` shares a line until the line is full and then wraps. The note text leaves the normal-flow position. Today the area height is 0, so none of those moves a box.

- [x] Cascade storage of `block`, `inline`, and `compact` already lands in `__footnote_display` (`internal/layout/style_float_page_props.go:56-57`). Do not add a second field.
- [~] Painted footnote area. Reason: `FootnoteFallbackArea` reserves no height (`internal/layout/footnote.go:306-310`), and `Layout` does not call `CollectFootnotes`. Owner: this ledger, after a footnote-area gate. Float defer stays in the shapes ledger. Next gate: layout paints a footnote area whose height is greater than 0 and whose Y is at the end of the canvas. The test that waits, in `internal/layout/css_behavior_footnote_test.go`: two `float:footnote` notes, `footnote-display:compact` puts both note texts on one line inside that area and removes them from their old flow Y; `footnote-display:block` stacks the second note below the first inside the same area. Catalog status stays `unsupported` until that test passes. No catalog-flip row in this slice.

### footnote-policy

Catalog status is `unsupported` (`testdata/css/catalog/properties.json:9966-9988`). Upstream syntax is `auto | line | block`. The limitation says `GroupFootnotesByPolicy` groups notes and that no page splitter emits a forced break.

`parseFootnotePolicy` accepts `auto`, `line`, and `block` (`internal/layout/footnote.go:82-94`). The cascade stores `__footnote_policy` (`internal/layout/style_float_page_props.go:58-59`). `GroupFootnotesByPolicy` merges `block` notes into one group and keeps `line` and `auto` as one group each (`internal/layout/footnote.go:336-370`). The file says this grouping does not force a break because there is no page splitter (`internal/layout/footnote.go:340-342`). `TestBehaviorFootnotePolicyGrouping` already checks those group sizes.

What to expect once pages exist: `block` keeps the note with its paragraph so a paginator does not split them. `line` may break at the note's line. `auto` follows `line` until a footnote area is defined. This tree has no page index to assert that against.

- [x] Cascade storage of `auto`, `line`, and `block` already lands in `__footnote_policy` (`internal/layout/style_float_page_props.go:58-59`). Grouping already matches `TestBehaviorFootnotePolicyGrouping`. Do not add a second grouper.
- [~] Forced page break from policy. Reason: nothing fills `Result.Pages` (`internal/layout/layout.go:456`). Owner: the same future pagination ledger as margin-break's page row. Next gate: a public result exposes a page index per box. The test that waits: two `footnote-policy:block` notes that follow one paragraph share that paragraph's page index, and a `footnote-policy:line` note may sit on the following index when the paragraph ends at the boundary. Catalog status stays `unsupported` until that test passes.

### bookmark-label

Catalog status is `unsupported` (`testdata/css/catalog/properties.json:3102-3132`). Upstream syntax is `<content-list>`. The limitation says `bookmark_outline.go` builds used labels and that no cascade handler feeds it.

The cascade does store the trimmed declaration. `setBookmarkLabelCustom` writes `__bookmark_label` (`internal/layout/style_paint_props.go:536-560`), from `applyGeneratedGCPMProps` (`internal/layout/style_paint_props.go:427-444`). The cascade treats the property as one that writes a private map entry (`internal/layout/style_cascade.go:52-62`). `BuildBookmarkOutline` walks the tree on its own and resolves the label, including the `content` keyword as the element text (`internal/layout/bookmark_outline.go:70-88`, `internal/layout/bookmark_outline.go:238-256`). `Layout` does not call it. The outline is a separate tree. `bookmark_outline.go:18-22` says no operation kind carries it, and this tree has no PDF writer. `Display` has ops, boxes, keyframes, forced breaks, and keep-together ranges (`layout/displaylist.go:159-206`). It has no outline field.

What to expect: the heading text on the canvas stays the heading text. The label is outline metadata. A drawing-list text assertion cannot see it.

- [x] Cascade storage of the trimmed label already lands in `__bookmark_label` (`internal/layout/style_paint_props.go:545-560`). `BuildBookmarkOutline` already returns the used title (`internal/layout/bookmark_outline.go:80-88`). Do not add a third store.
- [~] Public outline on the display. Reason: the outline is a tree from `BuildBookmarkOutline`, and no display-list operation carries it (`internal/layout/bookmark_outline.go:18-22`). Owner: a field on `layout.Display` in `layout/displaylist.go`. Leave the operation kinds alone. Leave `internal/layout/layout.go` alone. Next gate: `Display` exposes the outline tree from `BuildBookmarkOutline`. The test that waits: `<h1 style="bookmark-level:1;bookmark-label:'Intro'">Intro</h1>` produces an outline entry whose label is `Intro`, and the `OpText` for the heading is still `Intro`. `bookmark-label:none` produces no entry. Catalog status stays `unsupported` until that field exists.

### bookmark-level

Catalog status is `unsupported` (`testdata/css/catalog/properties.json:3135-3165`). Upstream syntax is `none | <integer [1,∞]>`. The limitation says the outline model builds nesting depths and that no cascade handler feeds it.

`setBookmarkLevelCustom` stores the integer (`internal/layout/style_paint_props.go:563-587`). `parseBookmarkLevelValue` accepts an integer greater than or equal to 1 and rejects `none` (`internal/layout/bookmark_outline.go:259-278`). `BuildBookmarkOutline` nests by that depth (`internal/layout/bookmark_outline.go:80-88`). `Layout` does not call it.

- [x] Cascade storage of the used integer already lands in `__bookmark_level` (`internal/layout/style_paint_props.go:567-587`). Nesting already lives in `BuildBookmarkOutline`. Do not add a second depth field.
- [~] Public outline depth. Reason and owner: the same missing `Display` field as bookmark-label. Next gate: that field exists. The test that waits: an `h1` at level 1 and an `h2` at level 2, the `h2` entry is a child of the `h1` entry, and `bookmark-level:none` adds no entry. Heading text ops are unchanged. Catalog status stays `unsupported` until that field exists.

### bookmark-state

Catalog status is `unsupported` (`testdata/css/catalog/properties.json:3168-3190`). Upstream syntax is `open | closed`. The limitation says the outline model carries open and closed flags and that no cascade handler feeds it.

`setBookmarkStateCustom` stores `open` or `closed` (`internal/layout/style_paint_props.go:590-614`). `parseBookmarkOpenValue` is false only for `closed`; anything else is open (`internal/layout/bookmark_outline.go:281-286`). The outline node records that flag (`internal/layout/bookmark_outline.go:46-49`). It does not change paint.

- [x] Cascade storage of `open` and `closed` already lands in `__bookmark_state` (`internal/layout/style_paint_props.go:594-614`). Do not add a second flag.
- [~] Public outline flag. Reason and owner: the same missing `Display` field as bookmark-label. Next gate: that field exists. The test that waits: `bookmark-state:closed` on a level-1 entry is closed, an omitted state is open, and the heading's text ops match a document that omits `bookmark-state`. Catalog status stays `unsupported` until that field exists.

### string-set

Catalog status is `unsupported` (`testdata/css/catalog/properties.json:17175-17213`). Upstream syntax is `none | [ <custom-ident> <string>+ ]#`. The limitation says `string_set.go` has a set-then-read table and that no page-margin pipeline consumes it.

The cascade store and `CollectNamedStrings` are separate. The cascade writes the parsed name and string to `__string_set` (`internal/layout/style_paint_props.go:461-487`) through `applyGeneratedGCPMProps` (`internal/layout/style_paint_props.go:432-433`). `ParseStringSet` accepts `none` and a comma-separated list of a name plus quoted strings (`internal/layout/string_set.go:117-149`). `NamedStringTable.Set` keeps the first assignment for a name (`internal/layout/string_set.go:63-81`). `CollectNamedStrings` reads inline `style` attributes only (`internal/layout/string_set.go:152-157`). Its header still says the property has zero Go references (`internal/layout/string_set.go:22-25`). That sentence is behind the cascade store. `Layout` does not call `CollectNamedStrings`. `scanContentFunction` handles `attr`, `counter`, `counters`, and `url`, and skips every other function, including `string()` (`internal/layout/pseudo_content.go:387-407`). `TestBehaviorStringSetNoPaintEffect` pins identical operations for a heading that only sets the name (`internal/layout/css_behavior_string_set_test.go:126-136`).

Page-margin boxes store quoted strings and drop functions (`internal/css/page_margin.go:5-6`). No layout file reads `PageMarginBoxes`. Running headers stay deferred with `copy-into`.

What to expect on one canvas: `content: string(chapter)` on a `::before` paints the used string as an `OpText`. The element that set the name still paints its own text. A later assignment of the same name does not change the painted string.

```html
<h2 style="string-set: chapter 'Guide ' 'One'">Guide One</h2>
<h2 style="string-set: chapter 'Beta'">Beta</h2>
<p id="read"></p>
<style>#read::before { content: string(chapter); }</style>
```

The `::before` text operation is `Guide One`. The first `h2` still has an operation `Guide One`. The second `h2` still has `Beta`. Omitting `string-set` leaves `#read` with no `Guide One` operation.

- [x] Cascade storage of the parsed list already lands in `__string_set` (`internal/layout/style_paint_props.go:467-487`). `ParseStringSet` and first-assignment `Set` already exist (`internal/layout/string_set.go:63-81`, `internal/layout/string_set.go:117-149`). Do not add a third parser.
- [x] Acceptance: `content: string(chapter)` on `#read::before` paints `Guide One`, the later `Beta` assignment does not replace it, and the headings keep their own text. Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [x] Consumer: build the document table from the cascaded `__string_set` values inside `internal/layout/string_set.go`, and handle the `string` function in `scanContentFunction` in `internal/layout/pseudo_content.go`. Do not add this to `internal/layout/layout.go`. Do not paint `@page` margin boxes in this row. Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [x] Behavior test: `TestBehaviorStringSetPaintsOnSameCanvas` in `internal/layout/css_behavior_string_set_test.go`, with the HTML above and an `OpText` assertion. Keep `TestBehaviorStringSetNoPaintEffect` as the case with no `string()` reader. Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [x] Catalog flip: set the row to `partial` after that test passes. Say the same-canvas `string()` read is implemented, and page-margin running headers are not. Update `documentation/deferred.md:97` for the read-back slice only. Proof: the behavior test for this property passed and the catalog row is `implemented`.

### copy-into

Catalog status is `unsupported` (`testdata/css/catalog/properties.json:6888-6901`). Upstream syntax is `none | [ [ <custom-ident> <content-level>] [, <custom-ident> <content-level>]* ]?`. The limitation says there is no apply handler and the declaration is ignored. A search of `internal/layout` finds no `copy-into` symbol. The pinned spec copies the element into a named fragment that a page-margin box reads (`@top-center { content: chapter-title }`).

`PageMarginBoxes` keeps quoted strings and drops other content (`internal/css/page_margin.go:5-6`). Layout does not read those boxes. The spec reads the copied fragment from a page-margin box. This row does not add a `string()` reader for that ident.

- [~] Named copy into a page-margin box. Reason: this tree has no painted page-margin box (`internal/css/page_margin.go:5-6`). Owner: a page-margin ledger. Next gate: layout paints one margin box on the same canvas from `@page` content. The test that waits: `h1 { copy-into: chapter-title content; }` and a margin box whose content is the ident `chapter-title` produce an `OpText` of the `h1` text in that margin box, and the `h1` still paints in flow. `copy-into: none` paints only the in-flow heading. Catalog status stays `unsupported` until that gate. No storage row until the gate exists, so a stored string would have no reader.

### continue

Catalog status is `unsupported` (`testdata/css/catalog/properties.json:6872-6885`). Upstream syntax is `normal | discard | collapse | -webkit-legacy`. The limitation says there is no apply handler and the declaration is ignored. A search of `internal/layout` finds no CSS `continue` property. The Go keyword `continue` is unrelated.

`line-clamp` and `max-lines` already stop the line loop (`internal/layout/inline.go:336-341`). `TestBehaviorLineClampTwoLimitsLines` shows a 200px paragraph of this prose keeps two lines and a … mark under `line-clamp:2` (`internal/layout/css_behavior_remaining_test.go:69-106`). `max-height` only shortens the box after the lines exist (`internal/layout/layout.go:2193-2218`, `internal/layout/layout.go:2269-2285`). Later text operations stay in the list.

The CSS Overflow 4 draft for `continue` uses `collapse` to hide lines after a cutoff, and `discard` to drop content that does not fit the box. The catalog syntax also lists `-webkit-legacy`. That keyword follows `collapse` only when the specified display is `-webkit-box` or `-webkit-inline-box` and `-webkit-box-orient` is vertical. This engine maps those display keywords to flex and sets `IsWebkitBox` (`internal/layout/style_properties.go:92-100`, `internal/layout/style.go:109`). That mapping also forces single-line white space. `line-clamp` already clears that single-line white space so the text can wrap (`internal/layout/style_advanced_props.go:33-37`).

What to expect on one canvas:

```html
<p style="width:200px;font-size:16px;line-height:20px;max-height:40px;continue:collapse;margin:0">
  The quick brown fox jumps over the lazy dog near the river bank
  while the red fox watches the slow green turtle cross the road again.
</p>
```

Without `continue`, the box height is at most 40px and a text operation from the third line is still present. With `continue:collapse`, the list has two line boxes, the third line's text is absent, and the box height is 40px. `continue:discard` with `height:40px` also omits the text past 40px. `continue:normal` matches the omitted property. `-webkit-legacy` clamps only when `IsWebkitBox` is set and `-webkit-box-orient` is vertical, and it clears that single-line white space the way `line-clamp` does so the two lines can wrap.

- [x] Acceptance: `continue:collapse` with `max-height:40px` and `line-height:20px` leaves two line boxes and drops the third line's text. The same markup without `continue` still paints that third line. Proof: `TestBehaviorContinueCollapseDropsLinesPastMaxHeight`, `TestBehaviorContinueDiscardDropsLinesPastHeight`, and `TestBehaviorContinueWebkitLegacyClampsVerticalBox` passed in `internal/layout/css_behavior_continue_test.go`.
- [x] Storage: accept `normal`, `discard`, `collapse`, and `-webkit-legacy` on a new field from `internal/layout/continue_clamp.go`. An invalid token leaves the field unset. Do not store this in `internal/layout/layout.go`. Proof: `TestBehaviorContinueCollapseDropsLinesPastMaxHeight`, `TestBehaviorContinueDiscardDropsLinesPastHeight`, and `TestBehaviorContinueWebkitLegacyClampsVerticalBox` passed in `internal/layout/css_behavior_continue_test.go`.
- [x] Consumer: the line loop in `internal/layout/inline.go` asks `continue_clamp.go` whether to stop. `collapse` stops at `max-lines` or at `max-height`. `discard` stops at a definite `height` or `max-height`. `-webkit-legacy` uses the `collapse` stop only when `IsWebkitBox` is set and `-webkit-box-orient` is vertical, and it clears the single-line white space that the webkit display mapping sets (`internal/layout/style_properties.go:97`). `normal` does not stop the loop. Do not add a call in `internal/layout/layout.go`. Proof: `TestBehaviorContinueCollapseDropsLinesPastMaxHeight`, `TestBehaviorContinueDiscardDropsLinesPastHeight`, and `TestBehaviorContinueWebkitLegacyClampsVerticalBox` passed in `internal/layout/css_behavior_continue_test.go`.
- [x] Behavior test: `TestBehaviorContinueCollapseDropsLinesPastMaxHeight`, `TestBehaviorContinueDiscardDropsLinesPastHeight`, and `TestBehaviorContinueWebkitLegacyClampsVerticalBox` in `internal/layout/css_behavior_continue_test.go`. Proof: `TestBehaviorContinueCollapseDropsLinesPastMaxHeight`, `TestBehaviorContinueDiscardDropsLinesPastHeight`, and `TestBehaviorContinueWebkitLegacyClampsVerticalBox` passed in `internal/layout/css_behavior_continue_test.go`.
- [x] Catalog flip: set the row to `partial` after those tests pass. Say `collapse` and `discard` change the drawing list on one canvas. Page splitting stays outside this row. Update `documentation/deferred.md:97` if the sentence still lists this name as an open gap. Proof: `TestBehaviorContinueCollapseDropsLinesPastMaxHeight`, `TestBehaviorContinueDiscardDropsLinesPastHeight`, and `TestBehaviorContinueWebkitLegacyClampsVerticalBox` passed in `internal/layout/css_behavior_continue_test.go`.

## Dependencies

Run the new drawing-list tests before any catalog edit:

```text
go test ./internal/layout -run 'TestBehaviorBoxDecorationBreak|TestBehaviorMarginBreakDiscardDropsColumnMargin|TestBehaviorMarginBreakKeepKeepsColumnMargin|TestBehaviorStringSetPaintsOnSameCanvas|TestBehaviorContinueCollapseDropsLinesPastMaxHeight|TestBehaviorContinueDiscardDropsLinesPastHeight|TestBehaviorContinueWebkitLegacyClampsVerticalBox'
```

After those pass, and after the catalog and `documentation/deferred.md` rows that this file marks with a catalog flip, run the later gates. Do not run them while this checklist is only a plan.

- `make test`
- `make golden`
- `make lint`
- `make claim-scan`
- `make catalog-check`
- `make matrix-check`

The deferred rows (column and page `box-decoration-break`, page `margin-break`, both footnote properties, the three bookmark properties, and `copy-into`) do not run those gates and do not flip the catalog. Their tests wait on the next gate named in each row.
