# v0.1.2 - Reading order, overlay, and cascade leftovers

> **Parent:** `plans/v0.1.2/README.md`
> **Status:** not started.
> **Estimated effort:** Two properties can change pixels on this canvas (`all`, and `link-parameters` on an SVG image). `background-tbd` would change background images once its draft name is stable. The other four do not change the drawing list until a reading list, a scrollport, or a top layer exists.

---

## Overview

The public result is one `layout.Display` from `layout.DisplayList` (`layout/displaylist.go:257-258`). `Ops` are the paint records, `Order` is the order to replay them, and `Boxes` are the element border boxes in document order (`layout/displaylist.go:159-175`). There is no PDF, no page bitmap (`layout/displaylist.go:238-242`), and no scrolling viewport. Production paint keeps a zero scroll offset (`internal/layout/scroll.go:11-13`).

A shorthand is one declaration that stands in for several longhands at that spot in the source. The cascade picks a winner per longhand using source order (`internal/layout/style_cascade.go:858-862`, `1406`). A property with no apply arm is dropped by `applyIgnoredGroup` (`internal/layout/style_properties.go:1876-1880`) after `applyStyleProp` finishes the dispatch table (`internal/layout/style_cascade.go:1974-1980`).

These seven rows are `unsupported` in `testdata/css/catalog/properties.json` (header totals at lines 11-17: 785 properties, 555 implemented, 4 partial, 218 unsupported, 8 intentionally ignored, `updated` `2026-11-02`). The pinned inventory is webref `ed/css` revision `1f2ec8f74a80c14066b4c7d6822cee59f69fa03b`. Rows below were read from that file and from the `href` on each matching object in `testdata/css/catalog/upstream/webref-ed-css-1f2ec8f.json`. A stored row with no consumer outside the style layer stays `unsupported` (`properties.json:9`).

`internal/layout/layout.go` is on the size allowlist at 2497 lines (`scripts/file-size-allowlist.txt:12`). No phase adds lines to that file.

Phases run from the property that can change paint or box order on this canvas to the property that cannot.

## Executive summary

| Order | Property | Catalog line | Pinned syntax | What this canvas can show |
|---|---|---:|---|---|
| 1 | `all` | 1490 | `initial \| inherit \| unset \| revert \| revert-layer \| revert-rule` | Resets longhands, so fills and text colors change. |
| 2 | `background-tbd` | 2933 | `<bg-layer>#` | Would add background images and leave the color fill. The draft says not to implement it yet. |
| 3 | `link-parameters` | 11538 | `none \| <param()>#` | Can change the PNG bytes of an SVG image. An `<a href>` stays as written. |
| 4 | `reading-flow` | 15100 | `normal \| source-order \| flex-visual \| flex-flow \| grid-rows \| grid-columns \| grid-order` | No change to box positions or paint order. |
| 5 | `reading-order` | 15116 | `<integer>` | No change to box positions or paint order. |
| 6 | `background-attachment` | 2492 | `scroll \| fixed \| inherit` | `fixed` is stored and painted as `scroll`. |
| 7 | `overlay` | 14063 | `none \| auto` | An author value cannot move a box. |

## Phase 1: all

Catalog row `all` (`testdata/css/catalog/properties.json:1490-1503`). Status `unsupported`. `values` is empty. `handlers` and `tests` are empty. `upstream_class` is `draft`. The limitation says there is no apply handler and the declaration is ignored. Upstream object at `testdata/css/catalog/upstream/webref-ed-css-1f2ec8f.json:646-652`, href `https://drafts.csswg.org/css-cascade-5/#propdef-all`, syntax `initial | inherit | unset | revert | revert-layer | revert-rule`.

CSS Cascading 5 defines `all` as the shorthand that resets every property except `direction` and `unicode-bidi`. It does not reset custom properties (`--*`). It accepts only the CSS-wide keywords.

Current behavior: a search of `internal/layout` style dispatch found no `case "all"`. The string `"all"` is a value of other properties (`internal/layout/style_multicol_props.go:182`). `cssWideKeyword` accepts `inherit`, `initial`, `unset`, `revert`, and `revert-layer` (`internal/layout/style_value_accept.go:220-228`). It does not accept `revert-rule`. A property with no acceptance entry is still accepted (`internal/layout/style_value_accept.go:236-238`), then `applyIgnoredGroup` drops it. So `all: initial` and `all: revert-rule` both disappear. The color applier copies the parent for `inherit` and `currentcolor` only (`internal/layout/style_properties.go:1250-1255`). A keyword it does not parse leaves the color already copied from the parent (`internal/layout/style_cascade.go:348`). Stuffing the word `initial` into the `color` longhand is not enough.

`direction` is copied from the parent (`internal/layout/style_cascade.go:439`). `unicode-bidi` is stored by `applyTextSupportProps` (`internal/layout/style_text_support_props.go:31-34`). Initial color is black and the initial background is transparent (`internal/layout/style.go:705-706`, `initialStyle` at `649`). A positive background alpha becomes an `OpFillRect` (`internal/layout/layout_chrome.go:543-547`).

What to expect. Put this in a style element so source order is the file order:

```css
.parent { color: red; background-color: blue; direction: rtl; }
.child { all: initial; }
```

```html
<div class="parent"><p class="child">Hi</p></div>
```

The child's text op has `R`, `G`, and `B` all 0. The child has no `OpFillRect` (initial background alpha is 0). The parent still has a blue fill. The child's `direction` stays `rtl`, and its `unicode-bidi` stays whatever the parent stored. The same block with `.child { all: initial; color: red; }` keeps red text, because the later longhand wins (`internal/layout/style_cascade.go:1406`). `.child { all: inherit; }` inside the blue parent gives the child a blue `OpFillRect`, because `all: inherit` assigns `background-color` even though that longhand is absent from the inherit table.

`unset` follows that same table: `color` copies the parent, `background-color` uses the transparent initial. `revert` and `revert-layer` are accepted keywords, and the appliers that mention them reset to an initial or to `none` (`internal/layout/style_paint_props.go:465-466`, `internal/layout/style_float_page_props.go:98`). There is no walk back to a user-agent origin. `revert-rule` is outside `cssWideKeyword`.

The expansion function lives in `internal/layout/style_all.go`. `expandShorthandDeclaration` (`internal/layout/style_shorthand.go:9`) calls it, so the new longhands share the declaration's source order. The body copies fields from `initialStyle` or from the parent. It skips `direction`, `unicode-bidi`, and names that start with `--`.

- [x] **Acceptance:** `all: initial` on `.child` yields a text op at `0,0,0` and no child fill, while `direction` stays `rtl`. `all: inherit` yields a child `OpFillRect` whose `R,G,B` match the parent's blue fill. A later `color` longhand beats `all`.
- [x] **Storage:** no new field. The winning longhands are the existing `ResolvedStyle` fields. `internal/layout/style_all.go` owns the expansion.
- [x] **Consumer:** the existing fill path (`internal/layout/layout_chrome.go:543-547`) and the existing text color. Do not add a second paint path, and do not add lines to `internal/layout/layout.go`.
- [x] **Behavior test:** `TestBehaviorAllInitialResetsColorAndKeepsDirection` in `internal/layout/css_behavior_all_test.go`. Assert the child text RGB is `0,0,0`, the child has no `OpFillRect`, and the used direction is still `rtl`. `TestBehaviorAllInheritPaintsParentBackground` asserts the child fill RGB equals the parent fill RGB. `TestBehaviorAllLonghandAfterWins` uses `all: initial; color: red` and asserts the text `R` is the red channel.
- [x] **Catalog flip:** set `all` to `implemented` with source `internal/layout/style_all.go`, handler `expandAllDeclaration`, and the three test names. Leave `revert`, `revert-layer`, and `revert-rule` as a limitation until a cascade keeps the user-agent winner apart from the author winner.
- [~] **Revert keywords:** `revert`, `revert-layer`, and `revert-rule` do not walk cascade origins in this tree. Reason: appliers that recognize `revert` reset to initial or `none`, and `revert-rule` is not a wide keyword (`internal/layout/style_value_accept.go:222-228`). Owner: this ledger. Next gate: the cascade stores the user-agent declaration and the previous rule's declaration as separate winners, and a test shows `all: revert` restoring a user-agent color after an author color.

## Phase 2: background-tbd

Catalog row `background-tbd` (`testdata/css/catalog/properties.json:2933-2946`). Status `unsupported`. `values`, `handlers`, and `tests` are empty. `upstream_class` is `draft`. Upstream object at `testdata/css/catalog/upstream/webref-ed-css-1f2ec8f.json:1051-1057`, href `https://drafts.csswg.org/css-backgrounds-4/#propdef-background-tbd`, syntax `<bg-layer>#`.

The backgrounds-4 section calls this the shorthand that sets the same longhands as `background` except `background-color`, so a later image layer leaves an earlier color in place. The section is marked "Not Ready For Implementation", and the name is still discussed in CSSWG issue 9083. A color token in the value makes the declaration invalid. The example in the draft is a green `background-color` followed by `background-tbd: url(a.png) top left, url(b.png) top left no-repeat`.

Current behavior: a search of `*.go` found no `background-tbd`. The declaration reaches `applyIgnoredGroup`. The real `background` shorthand expands only to `background-color` and `background-image` when those tokens parse (`internal/layout/style_cascade.go:1070-1089`). `applyBackgroundShorthand` writes the image and the first color (`internal/layout/style_paint_props.go:370-376`). It does not write attachment, position, repeat, clip, or origin.

What to expect once the name is stable. A block with `background-color: green` and a later `background-tbd: url(a.png) top left no-repeat` keeps a green `OpFillRect` and gains an `OpImage` for `a.png`. `background-tbd: url(fail.png) red` is dropped, so an earlier valid `background` still paints. Until the draft drops the not-ready banner, shipping the placeholder name would freeze a word the working group has not kept.

The file for that later change is `internal/layout/style_background_layer.go`, called from `expandShorthandDeclaration`. It must not grow `expandBackgroundDeclaration` or `internal/layout/layout.go`.

- [~] **Acceptance:** green fill stays and an `OpImage` appears for the layer list. Reason: backgrounds-4 marks the section not ready, and the property name is open (CSSWG issue 9083). Owner: this ledger waits on the CSS backgrounds draft. Next gate: the pinned webref row has a finished name and the not-ready banner is gone. The test below becomes valid on that gate.
- [~] **Storage:** reuse the existing background image, position, repeat, size, clip, origin, and attachment fields. No new field. Same reason, owner, and gate as the acceptance row.
- [~] **Consumer:** the existing background image painter (`internal/layout/background_image.go`). Same reason, owner, and gate. The future parser lives in `internal/layout/style_background_layer.go`.
- [~] **Behavior test:** `TestBehaviorBackgroundTbdKeepsColor` in `internal/layout/css_behavior_backgrounds_test.go`. Assert a green `OpFillRect` and one `OpImage` whose source is `a.png`. `TestBehaviorBackgroundTbdRejectsColor` asserts `background-tbd: url(fail.png) red` leaves the previous image and color in place. Same reason, owner, and gate.
- [~] **Catalog flip:** do not mark `implemented` under the name `background-tbd`. On the gate, rename the row to the published name, then point source, handler, and tests at the new file. Same reason, owner, and gate.

## Phase 3: link-parameters

Catalog row `link-parameters` (`testdata/css/catalog/properties.json:11538-11551`). Status `unsupported`. `values`, `handlers`, and `tests` are empty. `upstream_class` is `draft`. Upstream object at `testdata/css/catalog/upstream/webref-ed-css-1f2ec8f.json:4212-4218`, href `https://drafts.csswg.org/css-link-params-1/#propdef-link-parameters`, syntax `none | <param()>#`.

The current draft passes parameters into a linked resource. `param(--color: #ff0000)` sets an environment value the resource reads with `env()`. `none` passes nothing. The property applies to an element that is an external resource, and to CSS images on that element. It does not rewrite an `<a href>`.

Current behavior: a search of `*.go` found no `link-parameters` and no `env(` reader. External image bytes are rasterized as stored. `resolveImageData` calls `svg.RasterizeContext` with the file bytes (`internal/layout/layout_flow.go:90-104`). Inline SVG does the same (`internal/layout/layout_svg.go:36`). `RasterizeContext` takes the bytes and a max side (`internal/svg/raster.go:85`). An anchor becomes `OpLinkURI` from the authored href (`internal/layout/inline_paint.go:2031-2048`). `DisplayOpLinkURI` paints nothing (`layout/displaylist.go:146-147`).

What to expect. An SVG file whose root uses `fill="env(--color, black)"`, referenced by `<img>`, with `link-parameters: param(--color: #ff0000)` on that `img`. Before `RasterizeContext`, a rewrite in `internal/svg/link_params.go` replaces that `env()` with `#ff0000`. The `OpImage` PNG bytes differ from the same `img` with `link-parameters: none`. The same property on an `<a href="https://example.com/a">` leaves `Op.URI` equal to `https://example.com/a`. tdewolff/canvas is not assumed to evaluate `env()`; the rewrite happens in our SVG package before the raster call.

Storage and the apply arm live in `internal/layout/style_link_params.go`. One registration line joins `styleGroups` (`internal/layout/style_cascade.go:1702`). The raster call sites pass the rewritten bytes. No new lines in `internal/layout/layout.go`.

- [ ] **Acceptance:** `param(--color: #ff0000)` changes the `OpImage` PNG of an SVG that uses `env(--color, black)`. `none` matches the file rasterized with no rewrite. An `<a>` URI is unchanged.
- [ ] **Storage:** the parameter list on `ResolvedStyle`, parsed in `internal/layout/style_link_params.go`. `none` clears it. A `param()` that fails the grammar is ignored and the rest of the list is kept.
- [ ] **Consumer:** `internal/svg/link_params.go` rewrites the SVG XML, then the existing `svg.RasterizeContext` calls run. Background images that share `resolveImageData` use the same rewrite.
- [ ] **Behavior test:** `TestBehaviorLinkParametersRecolorsSvg` in `internal/layout/css_behavior_link_params_test.go`. Assert the `OpImage` bytes differ between `param(--color: #ff0000)` and `none`, and that a non-SVG image's bytes stay equal. `TestBehaviorLinkParametersLeavesAnchorURI` asserts `Op.URI` is the authored href.
- [ ] **Catalog flip:** set `link-parameters` to `implemented` with source `internal/layout/style_link_params.go`, handler `applyLinkParams`, and the two test names, after those tests fail if the rewrite is removed.
- [~] **Iframe and other nested documents:** this tree has no nested document to style. Reason: the only linked resources we rasterize are SVG bytes. Owner: this ledger. Next gate: a host that styles a nested document. The SVG test above does not wait on that gate.

## Phase 4: reading-flow

Catalog row `reading-flow` (`testdata/css/catalog/properties.json:15100-15113`). Status `unsupported`. `values`, `handlers`, and `tests` are empty. `upstream_class` is `draft`. Upstream object at `testdata/css/catalog/upstream/webref-ed-css-1f2ec8f.json:5519-5525`, href `https://drafts.csswg.org/css-display-4/#propdef-reading-flow`, syntax `normal | source-order | flex-visual | flex-flow | grid-rows | grid-columns | grid-order`.

Display 4 uses this property for speech and for linear sequential navigation (keyboard order and screen-reader order). `normal` follows DOM order. `source-order` makes a reading-flow container and still follows DOM order aside from `reading-order` on the children. `flex-visual` and `flex-flow` apply to flex containers. `grid-rows`, `grid-columns`, and `grid-order` apply to grid containers. The draft says the property changes neither layout nor painting order, so it has no effect on the visual canvas. Flex visual order in this engine is the `order` property: `setFlexOrderValue` (`internal/layout/style_properties.go:378-379`) and the sort in `internal/layout/flex_columns.go:300-301`.

Current behavior: a search of `internal/layout` for `reading-flow` found no matches. The declaration is dropped by `applyIgnoredGroup`.

What to expect on this canvas: nothing moves. Three flex items, `flex-direction: row-reverse`, `reading-flow: flex-visual`. Item 3 is the leftmost box, item 1 is the rightmost, the same as the identical flex without `reading-flow`. `Display.Order` and `Display.Boxes` stay on the same positions. In a browser the reading sequence would be item 3, then 2, then 1, left to right. This display list has no reading sequence. `Order` is paint order (`layout/displaylist.go:170-171`). `Boxes` are document order (`layout/displaylist.go:173-174`).

- [~] **Acceptance:** box X positions and paint order match the same flex without the property. A reading sequence of item 3, item 2, item 1 is the result a reading list would show. Reason: the only effect is speech and sequential navigation, and this engine has neither. Owner: this ledger. Next gate: a reading-order slice on `layout.Display`, separate from `Order` and `Boxes`.
- [~] **Storage:** a keyword on the flex or grid container in `internal/layout/reading_order.go`. Same reason, owner, and gate. Do not store it by reusing `FlexOrder`.
- [~] **Consumer:** the future reading-order slice. Paint and flex placement do not read it. Same reason, owner, and gate. The file is `internal/layout/reading_order.go`, not `internal/layout/layout.go`.
- [~] **Behavior test:** `TestBehaviorReadingFlowFlexVisualKeepsBoxPositions` would assert equal box X values with and without `reading-flow: flex-visual` on a `row-reverse` flex. `TestBehaviorReadingFlowFlexVisualReadingList` would assert the reading slice is item 3, item 2, item 1 while the leftmost box is still item 3. The second test becomes valid when the reading slice exists. Same reason, owner, and gate.
- [~] **Catalog flip:** leave `unsupported`. A test that only proves the boxes did not move does not satisfy the catalog rule that `implemented` needs a consumer outside the style layer (`properties.json:9`). Same reason, owner, and gate.

## Phase 5: reading-order

Catalog row `reading-order` (`testdata/css/catalog/properties.json:15116-15129`). Status `unsupported`. `values`, `handlers`, and `tests` are empty. `upstream_class` is `draft`. Upstream object at `testdata/css/catalog/upstream/webref-ed-css-1f2ec8f.json:5528-5534`, href `https://drafts.csswg.org/css-display-4/#propdef-reading-order`, syntax `<integer>`. Initial value in the draft is `0`.

It applies to a direct block-level, grid, or flex child of a reading-flow container (the parent `reading-flow` is something other than `normal`). The integer is an ordinal group. Siblings are visited from the lowest integer upward. Ties use the parent's `reading-flow`. The draft's block example is `reading-flow: source-order` on the parent and `reading-order: -1` on the third child, so the reading sequence visits that child first and the others in source order. Layout and paint order stay on source order.

Current behavior: a search of `internal/layout` for `reading-order` found no matches. The declaration is dropped by `applyIgnoredGroup`.

What to expect on this canvas: the third child's box stays in source position. A reading list, once it exists, would list that child first when the parent is `source-order` and this child is `-1`. With the parent at `normal`, the integer has no reading container to join.

- [~] **Acceptance:** five block children keep source box Y order when the third has `reading-order: -1`. The reading slice, if the phase 4 gate exists, lists the third child first and the others in source order. Reason: same missing reading list as phase 4. Owner: this ledger. Next gate: the phase 4 reading-order slice on `layout.Display`.
- [~] **Storage:** an integer on the child, default 0, in `internal/layout/reading_order.go`. A non-integer is ignored. Same reason, owner, and gate. Do not write `FlexOrder`.
- [~] **Consumer:** the phase 4 reading slice, and only when the parent keyword is not `normal`. Same reason, owner, and gate.
- [~] **Behavior test:** `TestBehaviorReadingOrderPullsThirdChildFirst` asserts source Y order for the five boxes, and asserts the reading slice starts with the third child once that slice exists. `TestBehaviorReadingOrderIgnoredWhenFlowNormal` asserts a parent `reading-flow: normal` leaves the reading slice in source order even if a child sets `-1`. Same reason, owner, and gate.
- [~] **Catalog flip:** leave `unsupported` until the reading slice and `TestBehaviorReadingOrderPullsThirdChildFirst` both exist. Same reason, owner, and gate.

## Phase 6: background-attachment

Catalog row `background-attachment` (`testdata/css/catalog/properties.json:2492-2514`). Status `unsupported`. `values` is `scroll | fixed | inherit`. Handler `applyColorBackgroundProps`. Test `TestBackgroundLonghands`. `upstream_class` is `draft`. Upstream object at `testdata/css/catalog/upstream/webref-ed-css-1f2ec8f.json:885-893`, href `https://drafts.csswg.org/css2/#propdef-background-attachment`, syntax `scroll | fixed | inherit`. The pinned syntax has no `local`. The catalog limitation cites `style_properties.go:1297` and `background_image.go:343`. The case is now at `1322`, and the paint comment is at `361`.

Current behavior: the longhand stores the lowercased string on `ResolvedStyle.BackgroundAttachment` (`internal/layout/style.go:373`, `internal/layout/style_properties.go:1322-1323`). Interning compares and hashes that string (`internal/layout/style_intern_gen.go:337`, `771`). `TestBackgroundLonghands` sets `background-attachment: fixed` and checks the stored string (`internal/layout/style_backgrounds_borders_test.go:56-68`). No paint function reads the field. `tileBackgroundRepeat` positions the tile from the background origin (`internal/layout/background_image.go:369`). The comment above it says `fixed` is a no-op because there is no viewport scroll, and it paints as `scroll` (`internal/layout/background_image.go:361-363`). Production scroll offset stays zero (`internal/layout/scroll.go:11-13`).

The `background` shorthand does not write `BackgroundAttachment` (`internal/layout/style_cascade.go:1070-1089`). This phase covers the longhand.

What to expect: a box with one background image and `background-attachment: fixed` emits the image op at the same `X,Y` as the same box with the property omitted (`scroll` is the initial). The image moves with the element's padding box. It does not stay pinned while content moves, because nothing on this canvas scrolls.

- [ ] **Acceptance:** `fixed` and omitted attachment emit the same `OpImage` `X,Y` for the same url, position, and element box.
- [x] **Storage:** `TestBackgroundLonghands` asserts `BackgroundAttachment == "fixed"` after `applyRestProps` (`internal/layout/style_backgrounds_borders_test.go:56-68`). The field is `internal/layout/style.go:373`, written at `internal/layout/style_properties.go:1322-1323`.
- [~] **Consumer:** a distinct `fixed` position needs a scrollport. Reason: paint does not read the field, and the comment at `internal/layout/background_image.go:361-363` keeps `fixed` on the `scroll` path. Owner: this ledger. Next gate: a production scroll offset other than the zero offset in `internal/layout/scroll.go:11-13`. The test-only offset map stays out of this phase. The future read, when that gate exists, belongs in `internal/layout/background_image.go`.
- [ ] **Behavior test:** `TestBehaviorBackgroundAttachmentFixedPaintsAsScroll` in `internal/layout/css_behavior_backgrounds_test.go`. Build two displays that differ only by `background-attachment: fixed`. Assert the `OpImage` `X` and `Y` are equal.
- [~] **Catalog flip:** leave `unsupported`. Reason: the storage test does not read the drawing list, and `properties.json:9` keeps a stored row with no outside consumer at `unsupported`. Owner: this ledger. Next gate: the scrollport consumer above, plus `TestBehaviorBackgroundAttachmentFixedPaintsAsScroll` if the status stays a documented no-op. When the row is edited, replace the stale `style_properties.go:1297` citation with `1322`. Do not add `local`; it is absent from the pinned syntax.

## Phase 7: overlay

Catalog row `overlay` (`testdata/css/catalog/properties.json:14063-14076`). Status `unsupported`. `values`, `handlers`, and `tests` are empty. `upstream_class` is `draft`. Upstream object at `testdata/css/catalog/upstream/webref-ed-css-1f2ec8f.json:5105-5111`, href `https://drafts.csswg.org/css-position-4/#propdef-overlay`, syntax `none | auto`.

Position 4 uses `overlay` with the top layer (popover, modal dialog, fullscreen). `auto` means an element that is in the top layer is painted as a sibling of the root, after the rest of the document. `none` means it is not painted there. The draft's user-agent rule is `* { overlay: none !important; }`, so an author declaration cannot turn the property on. The user agent adds `overlay: auto` when it puts an element in the top layer. A transition can delay the moment `auto` becomes `none`.

Current behavior: a search of `internal/layout` found no CSS `overlay` property. Matches are other uses of the word: the blend mode `blendOverlay` (`internal/layout/blend.go:13`), the SVG paint helper `svgPaintValues.overlay` (`internal/layout/layout_svg.go:1560-1561`), and comments about stacked boxes. `dialog` in `internal/html/tree.go:818` is an element name in the parser, not a top layer. Author `overlay: auto` is dropped by `applyIgnoredGroup`. The drawing list is unchanged.

What to expect on this canvas: a box with `overlay: auto` keeps its normal place in `Display.Order` and inside any ancestor clip. If a top layer is added later, a promoted element's ops are replayed after the root stacking context, and `overlay: none` at computed-value time removes that promotion. An author stylesheet still cannot promote the element by itself.

- [~] **Acceptance:** author `overlay: auto` leaves box positions and `Display.Order` unchanged. A promoted element, once a top layer exists, paints after the root, and `overlay: none` removes that promotion. Reason: the engine has no top layer, and the draft forbids author styles from setting `auto`. Owner: this ledger. Next gate: a document top-layer list plus a promotion path (popover, modal dialog, or fullscreen) that sets `overlay: auto` from the user agent.
- [~] **Storage:** a bool on the element, written only by that user-agent promotion, in `internal/layout/top_layer.go`. Same reason, owner, and gate. An author value does not set it.
- [~] **Consumer:** paint order appends promoted elements after the root. The function lives in `internal/layout/top_layer.go` and is called from the existing paint-order build. Same reason, owner, and gate. Do not add it to `internal/layout/layout.go`.
- [~] **Behavior test:** `TestBehaviorOverlayAuthorAutoDoesNotReorder` asserts equal `Display.Order` with and without author `overlay: auto`. `TestBehaviorOverlayTopLayerPaintsLast` becomes valid with the promotion gate: the promoted box's fill is the last painted op, and a parent `overflow: hidden` does not clip it. Same reason, owner, and gate.
- [~] **Catalog flip:** leave `unsupported`. The negative order test has no top-layer consumer. Same reason, owner, and gate.

## Dependencies

Future checks, not run for this planning file:

- `go test ./internal/layout -run 'TestBehaviorAllInitialResetsColorAndKeepsDirection|TestBehaviorAllInheritPaintsParentBackground|TestBehaviorAllLonghandAfterWins|TestBehaviorLinkParametersRecolorsSvg|TestBehaviorLinkParametersLeavesAnchorURI|TestBehaviorBackgroundAttachmentFixedPaintsAsScroll'`
- The reading-flow, reading-order, background-tbd, and overlay tests join that `-run` list only after their gates exist.
- Later, in order: `make test`, `make golden`, `make lint`, `make claim-scan`, `make catalog-check`, `make matrix-check`.

`all` lands before any catalog flip that depends on reset behavior. `reading-order` waits on the reading slice from `reading-flow`. `background-attachment` stays a stored longhand until a scrollport exists. `overlay` waits on a top layer. `background-tbd` waits on a stable draft name. `link-parameters` can land beside `all` because its SVG rewrite does not need those gates. No phase edits `internal/layout/layout.go`.
