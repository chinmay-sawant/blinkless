# Layout engine

`internal/layout` turns a parsed HTML tree and its CSS into boxes and a drawing list. The public `layout` package exposes that list as `DisplayList`. Nothing here writes a PDF page or paints a page bitmap; the drawing list is the output.

Public boxes are CSS pixels, y down, origin at the top left. `DisplayList` operations are canvas points (1 point is 1/72 inch, and 1 CSS pixel is 0.75 points), y down; for text, Y is the baseline (`layout/displaylist.go:13-15`, `:27-29`).

## The display list

`layout.DisplayList` (`layout/displaylist.go:134`) and `DisplayListOptions` (`:139`) return a `Display` (`:100-124`): `Ops`, an `Order` index in paint order, `Boxes`, the canvas size in CSS pixels, and the point/pixel conversion factors. `DisplayOp` is the internal `Op` type under an exported name (`:55`); the record itself is `internal/layout/layout.go:402`.

- Operation kinds (`layout/displaylist.go:67-97`): fill and stroke rectangles, line, text run, image, link URI, list bullet, and table grid run. Two kinds carry no paint: `DisplayOpNoop`, which the overflow clip pass writes when it removes an op, and `DisplayOpUnknown`, the boundary marker for a blend or isolation group.
- Paint order comes from `ilayout.PaintOrder` (`internal/layout/paint_order.go:14`). That is the sequence a painter should follow so z-index, the outline layer, and backgrounds below content come out right; `DisplayOrder` exposes it to callers (`layout/displaylist.go:210`).
- `Boxes` comes from `PlacedElements` (`internal/layout/placed.go:20`) converted to CSS pixels (`layout/layout.go:35-52`).
- An image op keeps its encoded bytes. When orientation or a clip cannot stay in those bytes, that one op is re-encoded as a PNG (`internal/layout/image_exif.go:497`; callers at `internal/layout/layout_images.go:384` and `internal/layout/clip_path.go:535`). That is the bitmap fallback, not a picture of the page (`layout/displaylist.go:126-133`).

The engine entry is `ilayout.LayoutContext` (`internal/layout/layout.go:1118`). `Result` (`:155`) carries the ops, canvas bounds, the box tree, and fields left from the removed page splitter (`:163-179`). An unset viewport width falls back to 1024 CSS pixels (`layout/displaylist.go:20`, `:153-163`); an unset height uses that width as the percentage-height containing block. Canvas height is the taller of the content and the requested minimum (`:189-194`).

No page splitter and no PDF writer remain in the tree. `page-break-*` values are still parsed onto the style (`internal/layout/style_properties.go:1594-1595`), but nothing splits the list. One path still treats `Options.Height` as a page boundary. Multicol snaps its column lines there (`internal/layout/multicol.go:360-367`). `IndependentBlocks` (`internal/layout/independent_blocks.go:21`) reports body children that can be laid out one at a time; only tests consume it today.

## Screen pixel alignment

Layout is exact in CSS points. Border widths, table rules, text decorations, and outlines keep the value the stylesheet asked for (`internal/layout/style_values.go:493-510`, `:512-518`); nothing floors them onto a device grid in computed styles, layout geometry, boxes, or hit testing. A widthless border shorthand takes the CSS initial medium, 3 CSS px = 2.25pt (`internal/layout/style_values.go:375-432`, default at `:425-427`), and an outline with a visible style and no width takes the same medium (`internal/layout/outline.go:38-48`).

A consumer that paints a screen raster opts in at the display-list boundary: `layout.SnapDisplayToDevicePixels` (`layout/displaylist_snap.go:36`) returns a copy whose positive `OpStrokeRect` and `OpLine` widths are quantized to `max(1, floor(cssPx*dsf+1e-6))/dsf` CSS pixels (`layout/displaylist_snap.go:68-81`), converting through `cssPxToPt = 0.75` (`layout/displaylist.go:15`). Coordinates, `Boxes`, `Order`, the canvas size, and payloads are untouched. `OpGridRun` segments keep their exact widths (`layout/displaylist_snap.go:50-63`), because a collapsed table row replays as one batched run. A print consumer replays the unquantized list, so print geometry stays exact. A nil display is `ErrNilDocument` and a non-finite or non-positive device scale factor is `ErrBadDeviceScale` (`layout/errors.go:9-15`).

## How a document gets there

1. `html.Parse` builds the tree (`internal/html`).
2. `css.Apply` attaches stylesheets.
3. `layout.DisplayList` calls `ilayout.LayoutContext`, which resolves styles, loads the default face from `internal/fonts`, and emits operations.

`ResolveStyles` (`internal/layout/layout_section.go:15`) exposes the same cascade for callers that need style pointers, and `NodeWithWorkspace` (`:40`) builds one already-styled node. The parse and match layer is described in `06-css.md`; the cascade and its acceptance gate live in `internal/layout/style_cascade.go` and `internal/layout/style_value_accept.go`.

## What layout owns

- Block, inline, table, flex, grid, multicol, and float placement
- The drawing list (`Op` / `DisplayOp`)
- Font choice and glyph positions through `internal/fonts`
- Paint appearance resolution for replay consumers: `StyleOf` (`internal/layout/paint_style.go:18`) resolves fill color, alpha, stroke width, and fake bold for an op, and `FakeBoldFor` (`:42`) answers the fake-bold question, which `layout.DisplayFakeBold` exposes (`layout/displaylist.go:218`)

## What it does not own

- Fetching URLs (`internal/load`)
- Writing image files or page rasters. The page image pipeline was removed; the only encoder left is the single-op PNG fallback (`internal/layout/image_exif.go:497`).
- Producing PDF files, page numbers, headers, footers, outlines, or a table of contents. The PDF writer is gone from this tree.
- JavaScript. `<script>` is parsed as raw text and never executed.

## Geometry fixes from the review waves

Waves B and E fixed five defects found by the wave A CSS review and one found by the wave D geometry recheck. The rows and their evidence live at `plans/0.0.1/html-css-json-compatibility-checklist.md:141-146`. All six are in the current tree.

**C1, distributed justify-content kept the gap.** `justifyDistributed` (`internal/layout/flex.go:1338-1367`) subtracts the fixed gaps before it shares the leftover space, then adds the shared amount on top of the gap. Row lines call it at `flex.go:1331-1332`, columns at `internal/layout/flex_columns.go:410-411`.

**C2, `fr` tracks share free space.** `distributeFrGridTracks` (`internal/layout/grid_tracks.go:274-296`) sizes fractional tracks from the space left after the non-flexible bases, divides by the total of the `fr` values, and floors each track at its own base size. The state and share passes are `frGridTrackState` (`:300-321`) and `frTrackShare` (`:325-353`); the entry is `resolveGridTrackSizes` (`:404-437`). Column intrinsics keep min-content and max-content separate (`:144-151`): the smallest width the content can take, and the width it wants without wrapping. That separation is what stopped `1fr` tracks from sizing from their content.

**C3, normal line height comes from face metrics.** `lineHeightOf` (`internal/layout/line_height.go:13-23`) returns a declared line height or `normalLineHeight` (`:30-50`), which reads the face's horizontal-header table (`hhea`) ascent, descent, and line gap (`:52-94`) and rounds each to whole CSS pixels. The 1.2 ratio remains the fallback when the face is unusable (`internal/layout/inline_collect.go:13`; `line_height.go:33`, `:46`). Residual: the rounding at `line_height.go:40-44` differs from the raw ascent and descent paint uses (`internal/layout/inline_paint.go:1824-1838`), a drift smaller than the 1.0 px geometry tolerance recorded by the wave D recheck.

**C4, root margin reporting.** `applyRootBoxMargins` (`internal/layout/root_margin.go:35-46`) shifts the body box by the collapsed first-child top margin (`applyRootBodyMargins` `:169-188`) and lets the escaped bottom margin extend the html height (`extendRootHTMLHeight` `:192-210`). In-flow children do not move (`:9-14`). Limits: the body shift applies only when the body's own top margin is zero (`:169-172`), and nonzero body margins keep the older stacking with a height correction (`rootBodyTopCorrection` `:216-234`).

**C5, a flex item's auto height encloses floats.** `isFlexItemNode` (`internal/layout/flex.go:1302-1317`) marks direct children of flex containers. `buildBlock` forces a fresh float state for them (`internal/layout/layout.go:1651`; `pushBFCFloatsForce` `internal/layout/layout_flow.go:849-871`), so the item height includes floating descendants.

**C6, absolute boxes anchor to the initial containing block.** `flowAbsCB` (`internal/layout/layout_flow.go:309-349`) returns the padding box, which is the content box plus the parent's padding, for a positioned, transformed, or containment parent; the initial containing block (the viewport box) for a root body with no containing-block ancestor (`absAnchorsToICB` `:365-381`); and the content box otherwise. The ICB height is the viewport height (`:284-287`). `TestAbsPositionedRootBodyAnchorsToICB` (`internal/layout/css_review_02_test.go:261-297`) pins both the plain body and the `position:relative` body against the Chrome reference in its comment. Remaining scope: absolute children of non-positioned intermediate blocks and `html{position:relative}` still use the immediate parent box (checklist row CSS-REVIEW-02-C6).

## text-wrap-style: balance

- The acceptance gate takes `auto`, `balance`, and `stable` (`internal/layout/style_value_accept.go:185-186`); the `text-wrap` shorthand takes at most one mode and one style (`:329-363`).
- `balanceCanApply` (`internal/layout/inline_balance.go:92-102`) requires `balance`, no cap on the number of rendered lines, no first-line indent, and no active float.
- `balanceLineWidth` (`:44-72`) packs the segment at full width and gives up outside two to six lines or when an item cannot fit whole (`:79-90`, `:129-162`). Inside that window it bisects for the widest width that keeps the line count, starting at 80% of the average normal line width and stopping when the remaining range is under one CSS pixel (constants `:12-19`).
- `inline.go` computes the width lazily at the start of each forced-break segment, meaning the items between two `<br>` tags (`internal/layout/inline.go:213-250`; `balanceSegmentWidth` `inline_balance.go:106-108`, `inlineSegmentEnd` `:114-122`), and breaks at `min(lineW, balanceW)` while alignment keeps the full line width (`inline.go:276-281`).
- Limits: `stable` wraps greedily like `auto` (`inline_balance.go:3-10`). `pretty` and `avoid-short-last-line` are rejected by the acceptance gate and fall back to greedy wrapping (`style_value_accept.go:357-358`). Blink tries a dynamic-programming breaker (ScoreLineBreaker) before its bisection; the two agree on the pinned cases and can pick different break sets when uneven word widths leave several valid sets (`inline_balance.go:31-35`).

## The declaration acceptance gate

`supportedDeclaration` (`internal/layout/style_value_accept.go:83-113`) is the single check that decides whether a declaration's value is usable. The cascade and the `@supports` probe share it:

- Custom properties, CSS-wide keywords, and values containing `var()` pass with no property check (`:89-105`).
- Everything else goes through `declarationValueAccepted` (`:137-207`) and its extended tables (`:427-473`, `:481-588`). A property with no entry is accepted (`:585-586`, rule stated at `:126-134`).

The cascade calls the gate before a declaration can win: author sheets (`internal/layout/style_cascade.go:647`), inline style (`:658`), and pseudo-element rules (`:712`). An invalid later declaration cannot replace an earlier valid one. `engineSupportsProperty` (`:1614-1648`) applies the same predicate and then checks that a property handler exists, so `@supports (display: bogus)` is false.

This is the gate that keeps `pretty` and `avoid-short-last-line` out of line breaking and stops an invalid `display` value from removing a box. The table models all 90 advertised implemented properties after the grid and grid-template grammar landed (`style_value_accept.go:519-522`; checklist row CSS-01b). An unlisted property still relies on its applier to drop a bad value, so an invalid higher-priority value for an unmodeled property can win the cascade.

## Tables and the parser-inserted tbody

The HTML parser inserts an implicit `tbody` when it sees `tr`, `td`, or `th` directly under a table (`internal/html/tables.go:78-85`). Layout does not need the source to write `<tbody>`:

- `collectTableRows` (`internal/layout/layout_tables.go:348-384`) flattens row groups. A `display: table-row` child becomes a row; a `display: table-header-group` child recurses as header rows; any other `*-row-group` recurses as body rows. The user-agent style sheet gives `tbody` `display: table-row-group` (`internal/layout/style_values.go:1729-1731`), so the implicit wrapper is collected like a written one.
- `resolveHeaderRows` (`layout_tables.go:399-422`) validates the header count after empty rows are stripped and falls back to a leading band of `th` rows.
- `useBlockForTableDisplay` (`internal/layout/layout.go:1606-1629`) treats `display: table` as a real table when a child is `tr`, `tbody`, `thead`, `tfoot`, `colgroup`, `col`, or `caption` (`:1622-1625`), and as a block sized to its content otherwise. `isTableDisplay` (`:1596-1604`) lists the table display values, and `buildTable` (`layout_tables.go:10-14`) is the entry.
- The prepare tolerance test pins the parser side of the same shape (`internal/convert/prepare/tree_tolerance_test.go:5`, `:96-127`).

## Multicol balance

`buildMulticol` (`internal/layout/multicol.go:26-34`) implements `column-count`, `column-width`, `column-gap`, `column-span`, and `column-fill: balance|auto`. Balance is the initial value (`balance := style.ColumnFill != overflowAuto` at `:374`).

- A single anonymous text item goes through `placeMulticolAnonColumns` (`:442-567`), which lays the strip out at one column width and shifts line bands into the following columns. Item batches go through `placeMulticolLine` (`:694-746`) with `target = totalH/nCols` (`:707-713`).
- `advanceMulticolColumn` (`:761-779`) starts a new column when the next item would overflow the usable height, or when balance has passed its target.
- Balance does not clamp the usable column height to a definite container height; only `column-fill: auto` does (`:611-625`). The comment records why. Flex stretch rebuilds children with a Height. The height resolver cannot tell that Height apart from an author height, so capping balance to it made a short fixture snap to a new page repeatedly.
- A multicol line never straddles a page boundary. `multicolColumnHeight` (`:585-642`) snaps to the next page when too little space remains, and `TestMulticolLinesDoNotStraddlePages` (`internal/layout/multicol_test.go:188-257`) checks that every text op's ascent and descent, the space its glyphs occupy above and below the baseline, stays inside one page of `Options.Height`.
- The wave B normal-line-height fix changed the line metrics. The same wave adjusted `multicol.go` to keep the no-straddle invariant (`temps/waveB-followups.md:85-87`), and the test now derives the line box from face metrics instead of the removed 1.2em estimate (`multicol_test.go:226-228`).
- Limit: floats and absolute boxes inside columns use the ordinary block formatting context path. Chrome balances floats across columns; this engine does not (`multicol.go:32-33`).

## Fidelity evidence, current numbers

- Drawing-list tests: `make golden` runs `TestDisplay` in `./layout`.
- Geometry runs (2026-10-09): 11 flex, grid, logical-property, and text fixtures compared against Chrome `143.0.7499.40` at 1024x768 with a 1.0 px tolerance. The wave A baseline had 101 of 215 like-for-like units within tolerance, where a unit is one element box measured in both engines; the wave D recheck after the C1-C5 fixes had 151 of 215 (`documentation/fidelity.md:57`). These are per-case comparisons, not a parity claim.
- Chrome cases: `test/chrome` holds 40 selected Flexbox targets. The manifest records 26 completed and 14 blocked (`test/chrome/README.md:26`). The blocked root causes are clustered in `temps/chrome-cases/blocked-dossier.md`; the CAT-07 row points there (`plans/0.0.1/html-css-json-compatibility-checklist.md:116`). The dossier notes that the C6 fix closed the panel deltas in cases 17, 18, and 19, and that everything else was still open at its date (`temps/chrome-cases/blocked-dossier.md:15-20`).
- Defect rows and evidence: `plans/0.0.1/html-css-json-compatibility-checklist.md:141-146` for C1 through C6.
