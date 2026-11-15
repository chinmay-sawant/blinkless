# v0.1.2 - Sizing leftovers

> **Parent:** `plans/v0.1.2/README.md`
> **Status:** not started.
> **Estimated effort:** Three shorthands and per-element zoom can land on the existing border box; the other four stay deferred until flex balance, a progress sample, or an embedded document exists.

---

## Overview

`layout.DisplayList` (`layout/displaylist.go:257-258`) lays a document out onto one canvas and returns the paint operations. It does not write a PDF and it does not encode a page image.

These eight catalog rows are `unsupported`. Each `values` list is empty. Each limitation says there is no apply handler in `internal/layout`, and each `source` is the pinned inventory `testdata/css/catalog/upstream/webref-ed-css-1f2ec8f.json` at revision `1f2ec8f74a80c14066b4c7d6822cee59f69fa03b`. The syntax below is `upstream_syntax` from `testdata/css/catalog/properties.json`, which matches the `value` on the inventory href.

A search of `internal/layout` for the quoted names `zoom`, `flex-line-count`, `interpolate-size`, `min-size`, `max-size`, `min-intrinsic-sizing`, and `frame-sizing` found no handler. The quoted name `size` hits the container type (`internal/layout/container.go:14`), the `contain` keyword (`internal/layout/style_containment_props.go:16`), and tests of those two. None of those hits is an apply arm for the property `size`. `applyStyleProp` walks `styleGroups` and then drops the declaration in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1974-1980`, `internal/layout/style_properties.go:1876-1880`).

The border box is `box.w` and `box.height` (`internal/layout/layout.go:1663-1664`). A background fill copies that rectangle into an `OpFillRect` (`internal/layout/layout_chrome.go:546`). Tests read the border box with `boxByID` (`internal/layout/css_review_02_test.go:8-37`). The engine stores points. `1` CSS pixel is `0.75pt` (`internal/layout/css_review_02_test.go:5-6`, `pxToPt` in `internal/layout/style_values.go:970`). Expectations below are CSS pixels, the same unit the opened tests assert.

New code stays out of `internal/layout/layout.go`. That file is the allowlisted over-limit file at 2497 lines (`scripts/file-size-allowlist.txt:12`). Each property below names the file that receives the new code.

## Executive summary

| Property | Catalog status | Upstream syntax | This file |
|---|---|---|---|
| `zoom` | unsupported | `<number [0,∞]> \| <percentage [0,∞]>` | Phase 2. Multiplies used lengths on that element |
| `flex-line-count` | unsupported | `<integer [1,∞]>` | Phase 3. Deferred until `flex-wrap: balance` exists |
| `interpolate-size` | unsupported | `numeric-only \| allow-keywords` | Phase 4. Static box stays put. Halfway height waits on a progress sample |
| `min-size` | unsupported | `<'min-width'> <'min-height'>?` | Phase 1. Copy `min-width` and `min-height` |
| `max-size` | unsupported | `<'max-width'> <'max-height'>?` | Phase 1. Copy `max-width` and `max-height` |
| `size` | unsupported | `<'width'> <'height'>?` | Phase 1. Copy `width` and `height` |
| `min-intrinsic-sizing` | unsupported | `legacy \| zero-if-scroll \|\| zero-if-extrinsic` | Phase 5. Deferred. Draft name, no shipped browser behavior |
| `frame-sizing` | unsupported | `auto \| content-width \| content-height \| content-block-size \| content-inline-size` | Phase 5. Deferred until an embedded document has a layout size |

## Phase 0: cousins that already move the border box

These rows are the behavior the new work copies. Each one is an assertion in a test opened for this ledger.

- [x] `width:120px;height:40px` on an empty div inside a `400px` by `200px` wrapper lays out at a `120px` by `40px` border box, flush with the wrapper. Proof: `TestBehaviorWidthContentBoxUsedSize` (`internal/layout/css_behavior_box_model_test.go:52-66`).
- [x] `height:60px` with no width stretches the empty div to the `400px` wrapper and keeps the `60px` height. Proof: `TestBehaviorHeightContentBoxUsedSize` (`internal/layout/css_behavior_box_model_test.go:73-82`).
- [x] `width:50px;min-width:120px;height:30px` lays out `120px` wide. `width:200px` with the same floor stays `200px`. Proof: `TestBehaviorMinWidthClampsUsedWidth` (`internal/layout/css_behavior_box_model_test.go:136-158`).
- [x] `height:20px;min-height:90px` on a `200px` wide box lays out `90px` tall. `height:150px` stays `150px`. Proof: `TestBehaviorMinHeightClampsUsedHeight` (`internal/layout/css_behavior_box_model_test.go:197-218`).
- [x] `width:300px;max-width:180px;height:30px` lays out `180px` wide. `width:80px` stays `80px`. Proof: `TestBehaviorMaxWidthClampsUsedWidth` (`internal/layout/css_behavior_box_model_test.go:166-188`).
- [x] `height:200px;max-height:60px` on a `200px` wide box lays out `60px` tall. `height:40px` stays `40px`. Proof: `TestBehaviorMaxHeightClampsUsedHeight` (`internal/layout/css_behavior_box_model_test.go:227-248`).
- [x] `flex-wrap:wrap` on a `120px` row with three `50px` by `30px` items places them at `(0,0)`, `(50,0)`, and `(0,30)`. Proof: `TestBehaviorFlexWrapWrapUsedPosition` (`internal/layout/css_behavior_flex_grid_test.go:83-95`).
- [x] `aspect-ratio:2/1` on a `160px` wide empty box lays out `80px` tall. `aspect-ratio:1/2` on a `120px` wide box lays out `240px` tall. Proof: `TestBehaviorAspectRatioDerivesHeightFromWidth` (`internal/layout/css_behavior_box_model_test.go:511-534`). The ratio is stored by `applyAspectRatioProps` (`internal/layout/style_aspect_ratio_props.go:13-27`).
- [x] Operator `Options.Zoom` scales painted lengths for the whole document. `Zoom` `2/3` turns a `12pt` font into about `8pt` on an `OpText`. Proof: `TestPrintZoomDensifies12ptTo8` (`internal/layout/vector_body_size_test.go:50-84`). The factor is `zoomScale` (`internal/layout/layout_faces.go:265-272`) stored on the engine (`internal/layout/layout_context.go:33`) and applied by `scalePt` (`internal/layout/layout_faces.go:262-263`). That field is the document zoom. It is a different control from the `zoom` property in phase 2.

The longhands are applied in `applyBoxSizingProps` (`internal/layout/style_properties.go:539-548`). `width` and `height` go through `setWidthValue` and `setHeightValue` (`internal/layout/style_properties.go:581-666`). `min-width` and `min-height` go through `setMinWidthValue` and `setMinHeightValue` (`internal/layout/style_properties.go:680-709`). `max-width` and `max-height` go through `setMaxWidthValue` and `setMaxHeightValue` (`internal/layout/style_properties.go:723-750`). `flex-wrap` stores `nowrap`, `wrap`, and `wrap-reverse` (`internal/layout/style_properties.go:318-321`). The initial flex wrap is `nowrap` and the initial align-items is `stretch` (`internal/layout/style.go:664-667`).

**Gate:** no new test. The nine rows above are already green in the opened tests.

## Phase 1: shorthands that write the existing longhands

CSS Box Sizing Level 4 defines three shorthands. One value is copied onto both axes. Two values set width, then height. The initial value is `auto` for `size` and `min-size`, and `none` for `max-size`.

The parser lives in `internal/layout/style_size_shorthand.go`. `applyBoxSizingProps` gains one case that calls that file. `internal/layout/layout.go` does not gain lines. WebKit's tree maps these three names onto the width and height longhands and enables them by default.

A shorthand is all or nothing. Both tokens must be accepted before either longhand is written. Acceptance reuses the longhand checks: `widthValueAccepted`, `heightValueAccepted`, `minExtentValueAccepted`, and `boxLengthValueAccepted` (`internal/layout/style_value_accept.go:487-532`). `height` does not accept `min-content` or `max-content` (`internal/layout/style_value_accept.go:500-507`). `size: min-content` is therefore rejected, and the previous width and height stay. The behavior tests use pixel lengths, which those checks already allow.

`@page` already uses the descriptor name `size` (`internal/css/css.go:365`). A search of `internal/layout` for `Page.Size` found no reader. The element property must call `setWidthValue` and `setHeightValue`. It must not call `pageDescriptors`.

### Min size

Catalog status `unsupported`. The `values` list is empty. Upstream syntax `<'min-width'> <'min-height'>?`. Href `https://drafts.csswg.org/css-sizing-4/#propdef-min-size`. Initial `auto`. The shorthand sets `min-width` and `min-height`. One value is copied to both.

No handler, as the overview search recorded. The name `min-size` inside `flexMinMainSizeUncached` is the flex automatic minimum (`internal/layout/flex.go:908-912`), which reads `MinWidth` and `MinWidthSet`. It does not parse the property `min-size`.

Copy `min-width` and `min-height`. An empty `#item` inside a `400px` wrapper:

```html
<div id="item" style="width:50px;height:20px;min-size:120px"></div>
```

The border box is `120px` by `120px`. The same markup with `min-size:120px 80px` is `120px` wide and `80px` tall. Those are the floors already asserted for each axis in phase 0, applied together. `min-size:auto` clears both floors, so `width:50px;height:20px` stays `50px` by `20px`.

- [ ] Acceptance in `internal/layout/style_size_shorthand.go`. One length or percentage is copied to both axes. Two values set `min-width`, then `min-height`. `auto` is accepted. A bad token, including a missing unit, does not write either axis.
- [ ] Storage. No new fields. Call `setMinWidthValue` and `setMinHeightValue` (`internal/layout/style_properties.go:680-709`). A rejected shorthand leaves a previous `min-width:200px` in place.
- [ ] Consumer. The existing min-width and min-height clamps. No new branch in `internal/layout/layout.go`.
- [ ] Behavior test `TestBehaviorMinSizeClampsUsedBorderBox` in package `internal/layout`, file `internal/layout/css_behavior_sizing_leftover_test.go`. Assert with `boxByID`: `min-size:120px` on `width:50px;height:20px` is `120px` by `120px`; `min-size:120px 80px` is `120px` by `80px`; `min-size:auto` keeps `50px` by `20px`; `min-size:120px bogus` keeps a previous `min-width:10px`.
- [ ] Catalog flip. Set `min-size` to `implemented` in `testdata/css/catalog/properties.json` only after that test is green. Fill `values` from the upstream syntax. The consumer is the existing min clamp, which sits outside the style function.

### Max size

Catalog status `unsupported`. The `values` list is empty. Upstream syntax `<'max-width'> <'max-height'>?`. Href `https://drafts.csswg.org/css-sizing-4/#propdef-max-size`. Initial `none`. The shorthand sets `max-width` and `max-height`. One value is copied to both.

No handler. Copy `max-width` and `max-height`.

```html
<div id="item" style="width:300px;height:200px;max-size:180px 60px"></div>
```

The border box is `180px` by `60px`. `max-size:100px` on the same `300px` by `200px` box is `100px` by `100px`. `max-size:none` leaves `300px` by `200px`.

- [ ] Acceptance in `internal/layout/style_size_shorthand.go`. `none`, one length, or two lengths. A bad token writes neither axis.
- [ ] Storage. Call `setMaxWidthValue` and `setMaxHeightValue` (`internal/layout/style_properties.go:723-750`). `none` stores the same unset ceiling those setters already use (`MaxWidth` and `MaxHeight` at `-1`, `internal/layout/style.go:695-700`).
- [ ] Consumer. The existing max-width and max-height clamps. No new branch in `internal/layout/layout.go`.
- [ ] Behavior test `TestBehaviorMaxSizeClampsUsedBorderBox` in package `internal/layout`, same test file. Assert `max-size:180px 60px` on `width:300px;height:200px` is `180px` by `60px`; `max-size:100px` is `100px` by `100px`; `max-size:none` keeps `300px` by `200px`; a bad second token keeps a previous `max-width:40px`.
- [ ] Catalog flip for `max-size` after that test is green. Same rule as `min-size`.

### Size

Catalog status `unsupported`. The `values` list is empty. Upstream syntax `<'width'> <'height'>?`. Href `https://drafts.csswg.org/css-sizing-4/#propdef-size`. Initial `auto`. The shorthand sets `width` and `height`. One value is copied to both. The sizing spec says this shorthand does not apply inside `@page`, because `@page` already has a `size` descriptor.

No handler for the element property. Copy `width` and `height`.

```html
<div id="outer" style="width:400px">
  <div id="item" style="size:120px 40px"></div>
</div>
```

`#item` is `120px` by `40px`, the same border box as `TestBehaviorWidthContentBoxUsedSize`. `size:120px` copies `120px` onto the height as well, so the empty div is `120px` by `120px`. `width:120px` alone on an empty div stays `0px` tall, which is what the rejected `aspect-ratio:4 : 1` case already shows (`internal/layout/css_behavior_box_model_test.go:522`). `size:auto` on a `400px` wrapper with no height stretches the width to `400px` and keeps the empty height at `0px`.

- [ ] Acceptance in `internal/layout/style_size_shorthand.go`. A token must pass `widthValueAccepted` for the width axis and `heightValueAccepted` for the height axis. `size:min-content` fails the height check and writes nothing. `size:120px` and `size:120px 40px` pass.
- [ ] Storage. Call `setWidthValue` and `setHeightValue` (`internal/layout/style_properties.go:581-666`). Do not store a separate size string. Do not write `internal/css/css.go` page `Size`.
- [ ] Consumer. The existing width and height used-size path. No new branch in `internal/layout/layout.go`.
- [ ] Behavior test `TestBehaviorSizeShorthandSetsUsedBorderBox` in package `internal/layout`, same test file. Assert `size:120px 40px` is `120px` by `40px` inside a `400px` wrapper; `size:120px` is `120px` by `120px`; `size:auto` is `400px` by `0px`; `size:120px min-content` keeps a previous `width:80px;height:10px`.
- [ ] Catalog flip for `size` after that test is green. The limitation must still say the element property is separate from the `@page` descriptor.

**Gate:** `go test ./internal/layout -run 'TestBehaviorMinSizeClampsUsedBorderBox|TestBehaviorMaxSizeClampsUsedBorderBox|TestBehaviorSizeShorthandSetsUsedBorderBox'`. Phase 0 tests stay green. `internal/layout/layout.go` stays at 2497 lines.

## Phase 2: zoom

### Zoom

Catalog status `unsupported`. The `values` list is empty. Upstream syntax `<number [0,∞]> | <percentage [0,∞]>`. Href `https://drafts.csswg.org/css-viewport/#propdef-zoom`. The draft fetched from that href says the initial value is `1`. A positive number is the factor. A percentage is that factor times `100` (`150%` is `1.5`). `0` and `0%` are stored as `1`. Negative values are illegal. The used value of a length is multiplied by the element's effective zoom before the rest of used-value math. Effective zoom is the element's own factor times each ancestor's factor. `auto` and percentages are left alone. The property changes layout. Nested `zoom:2` inside `zoom:2` makes a `100px` length `400px`.

No handler. `Options.Zoom` is the document factor from phase 0. CSS `zoom` must multiply on top of `e.scale` when both are set. With the default `Options.Zoom` of `0`, `zoomScale` returns `1`, so a test that leaves `Options.Zoom` unset measures the property alone.

The multiply to copy is `scalePt` (`internal/layout/layout_faces.go:262-263`), with a per-element factor in front of the document factor. `aspect-ratio` only fills the missing side (`TestBehaviorAspectRatioDerivesHeightFromWidth`).

```html
<div id="item" style="width:100px;height:40px;zoom:2;background:#f00"></div>
```

The border box is `200px` by `80px`. The background `OpFillRect` uses that width and height (`internal/layout/layout_chrome.go:546`). `zoom:150%` on the same box is `150px` by `60px`. `zoom:0` and `zoom:0%` keep `100px` by `40px`.

Percentages stay percentages. A child with `width:50%;height:40px;zoom:2` inside a `400px` parent is `200px` wide and `80px` tall.

Padding is a length, so it scales too. `width:100px;height:40px;padding:10px;zoom:2` under the initial `content-box` is a border box of `240px` by `120px` (`200px` content plus `20px` of padding on each side, and the same for the height).

New code is `internal/layout/zoom_css.go`: parse the factor, store it on `ResolvedStyle` (initial `1`), and expose the effective factor. `build` (`internal/layout/layout.go:1739`) and `pushZ` (`internal/layout/layout.go:1054`) already run once per box. The zoom stack has to be pushed from a helper those functions already call, or the same number of lines have to leave `layout.go` in the same change. The file stays at 2497 lines. `scalePt` keeps multiplying by `e.scale`. The element factor is a second multiply around that call, implemented in `zoom_css.go`, so a percentage resolved without `scalePt` is not multiplied again.

- [ ] Acceptance in `internal/layout/zoom_css.go`. A number greater than `0`, or a percentage greater than `0%`. `0` and `0%` store `1`. A negative number or percentage does not replace an earlier `2`.
- [ ] Storage. One float on `ResolvedStyle`, initial `1`, plus the ancestor product computed while the box is built. Register the apply function on `styleGroups` in `internal/layout/style_cascade.go` the same way `applyAspectRatioProps` is registered (`internal/layout/style_cascade.go:1728`).
- [ ] Consumer. Absolute lengths on that element, including `width`, `height`, and `padding`, pass through the element factor and then `scalePt`. A percentage width stays a fraction of the containing block. `Options.Zoom` still feeds `e.scale`.
- [ ] Behavior test `TestBehaviorZoomScalesUsedBorderBox` in package `internal/layout`, file `internal/layout/css_behavior_sizing_leftover_test.go`. `Options.Zoom` stays at the default. Assert `zoom:2` on `100px` by `40px` is `200px` by `80px`; `zoom:150%` is `150px` by `60px`; `zoom:0` keeps `100px` by `40px`; `width:50%;height:40px;zoom:2` inside a `400px` parent is `200px` by `80px`; `padding:10px;zoom:2` on `100px` by `40px` is `240px` by `120px`; a child `zoom:2` inside a parent `zoom:2` makes `width:100px;height:40px` into `400px` by `160px`.
- [ ] Catalog flip for `zoom` after that test is green. The limitation must still say `Options.Zoom` is the document factor and the property is the per-element factor.

**Gate:** `go test ./internal/layout -run 'TestBehaviorZoomScalesUsedBorderBox'`. `TestPrintZoomDensifies12ptTo8` stays green. `internal/layout/layout.go` stays at 2497 lines.

## Phase 3: flex line count

### Flex line count

Catalog status `unsupported`. The `values` list is empty. Upstream syntax `<integer [1,∞]>`. Href `https://drafts.csswg.org/css-flexbox-2/#propdef-flex-line-count`. The draft fetched from that href says the initial value is `1`. It applies to a flex container that can wrap. The minimum line count is capped by the number of items. `nowrap` ignores the property. The same draft says a wrapping container that is not `balance` still divides a definite cross size by the line count, after subtracting the gaps. MDN's page for this property, dated 2026-10-07, says the property does nothing unless `flex-wrap` includes `balance`. `balance` itself is Chrome 150 and later, and Firefox and Safari do not ship it.

No handler. `setFlexWrapValue` stores `nowrap`, `wrap`, and `wrap-reverse` only (`internal/layout/style_properties.go:318-321`), so `flex-wrap: balance` is dropped even though the catalog value string for `flex-wrap` includes `balance`. `flexWrapLines` packs lines by whether the next item fits (`internal/layout/flex_lines.go:15-60`). `flowFlexRow` copies the container height into the line's cross size only when wrap is off (`internal/layout/flex.go:443-445`). A wrapped container therefore has no definite single-line cross size for the draft's division rule to use.

The nowrap case can be pinned now, because it must not move a box. The case that changes a box cannot be built until `balance` packs lines.

Nowrap, which must match a row with no `flex-line-count`:

```html
<div id="row" style="display:flex;flex-wrap:nowrap;width:300px;height:40px;flex-line-count:4">
  <div id="a" style="width:50px;height:30px"></div>
</div>
```

`#a` stays `50px` by `30px` at `(0,0)`.

After `flex-wrap: balance` exists, this is the box that should move. Four `40px` by `20px` items in a `100px` wide row. Plain `wrap` fits two items per line, so the row's border box is `100px` by `40px` and the third item's top is `20px`. `flex-wrap:balance;flex-line-count:4` puts one item on each line, so the row's border box is `100px` by `80px` and the third item's top is `40px`. That expected size is the gate, and it is not asserted by any opened test.

- [ ] Acceptance in `internal/layout/style_flex_line_count.go`. An integer of `1` or greater. `0`, a negative integer, and a bare word do not replace an earlier `3`.
- [ ] Storage. One int on `ResolvedStyle`, initial `1`. Dispatch from `applyFlexKeywordProps` (`internal/layout/style_properties.go:269`).
- [~] Consumer. Reason: `flex-wrap: balance` is not stored, and `flowFlexRow` does not give a wrapped line a definite cross size (`internal/layout/flex.go:443-445`), so neither the MDN rule nor the draft's cross-size division can change a box. Owner: this file. Next gate: a `balance` packer in `internal/layout/flex_lines.go` (not in `layout.go`), then the four-line test below. The nowrap ignore can land with the storage, and it is the next row.
- [ ] Behavior test `TestBehaviorFlexLineCountNowrapKeepsOneLine` in package `internal/layout`, file `internal/layout/css_behavior_sizing_leftover_test.go`. The nowrap markup above keeps `#a` at `50px` by `30px` and y `0`.
- [~] Behavior test `TestBehaviorFlexLineCountBalanceFourLines` in package `internal/layout`, same file. Reason and owner match the consumer row. Next gate: with `flex-wrap:balance` and `flex-line-count:4`, the `100px` row of four `40px` by `20px` items is `80px` tall and the third item's y is `40px`. Until that packer exists the row stays open.
- [~] Catalog flip. Reason: a nowrap no-op is not a consumer, and the catalog rule requires a consumer outside the style layer plus a behavior test that resolves a real size. Owner: this file. Next gate: flip `flex-line-count` to `implemented` only after `TestBehaviorFlexLineCountBalanceFourLines` is green. Leave `flex-wrap` alone in that change.

**Gate:** the nowrap test can go green on its own. The phase stays open until the `[~]` rows close.

## Phase 4: interpolate size

### Interpolate size

Catalog status `unsupported`. The `values` list is empty. Upstream syntax `numeric-only | allow-keywords`. The inventory href is `https://drafts.csswg.org/css-values-5/#propdef-interpolate-size`. The values-5 draft fetched from that href says the property moved to CSS Box Sizing Level 4. The definition read there (`css-sizing-4` section 7.3) has initial `numeric-only`, inherits, and is not itself animated. `allow-keywords` lets a sizing keyword (`auto`, `min-content`, `max-content`, `fit-content`) interpolate with a length by treating the keyword as `calc-size(keyword, size)`. Two different keywords still do not interpolate. Chrome 129 and later ship this. Firefox and Safari do not.

No handler. A search of `internal/layout` for `calc-size` and `CalcSize` found no parser. Animation timing is stored as `AnimationRecord` for a host to sample (`internal/layout/animation_transition.go:46-48`). Transition timing is stored the same way (`internal/layout/animation_transition.go:68-69`). One call to `layout.DisplayList` builds one snapshot. Nothing in that call reads a progress and writes an in-between width.

On that single snapshot the property does not move a box. Copy the static `width` and `height` result from phase 0:

```html
<div id="item" style="width:100px;height:40px;interpolate-size:allow-keywords"></div>
```

The border box stays `100px` by `40px`, the same as `numeric-only` and the same as no declaration.

The size change needs a progress sample. Once a sampler exists, this is the box:

```html
<div id="item" style="interpolate-size:allow-keywords;height:0;overflow:hidden">
  <div style="height:100px"></div>
</div>
```

At progress `0` the border box height is `0`. At progress `1`, `height:auto` is the `100px` child, so the border box is `100px` tall. At progress `0.5`, `allow-keywords` makes the border box `50px` tall. `numeric-only` does not produce that `50px` height. A search found no containing-block sampler that already does this, and `calc-size()` is the spec's interpolation step, so the halfway row stays deferred.

- [ ] Acceptance in `internal/layout/style_interpolate_size.go`. `numeric-only` and `allow-keywords` only. A bad token keeps the previous keyword. The value inherits, same as the draft.
- [ ] Storage. One keyword on `ResolvedStyle`, initial `numeric-only`.
- [ ] Behavior test `TestBehaviorInterpolateSizeStaticBoxUnchanged` in package `internal/layout`, file `internal/layout/css_behavior_sizing_leftover_test.go`. The `100px` by `40px` box stays `100px` by `40px` for both keywords.
- [~] Consumer and behavior test `TestBehaviorInterpolateSizeHalfwayHeight`. Reason: one display list has no progress input, and `calc-size()` has no parser, so `allow-keywords` cannot change `box.height` yet. Owner: this file. Next gate: a sampler outside `layout.go` that lays the markup out at progress `0`, `0.5`, and `1` and expects heights `0`, `50px`, and `100px` for `allow-keywords`. The `50px` expectation is the used height halfway from `0` to the `100px` child. `numeric-only` must not report `50px`.
- [~] Catalog flip. Reason and owner match the consumer row. Next gate: flip to `implemented` only after the halfway test is green. The static test is not enough.

**Gate:** the static test can go green. The phase stays open until the halfway test exists.

## Phase 5: drafts that cannot change a box yet

### Min intrinsic sizing

Catalog status `unsupported`. The `values` list is empty. Upstream syntax `legacy | zero-if-scroll || zero-if-extrinsic`. Href `https://drafts.csswg.org/css-sizing-4/#propdef-min-intrinsic-sizing`. The draft says the initial value is `legacy`. `legacy` keeps today's min-content contribution. `zero-if-scroll` compresses that contribution when the box scrolls in that axis, by pretending the box is empty except for explicit `min-content`, `max-content`, and `fit-content` constraints. `zero-if-extrinsic` compresses it when the preferred size or the maximum size is a fixed length rather than a content size. The same section says the property name is still unsettled. The module intro says Level 4 is exploratory and an implementer should use Level 3. A web search for shipped browser behavior did not find an engine that honors this name.

No handler. Token min-content width is `minContentWidth` (`internal/layout/layout_measure.go:592-615`), which returns the token's full advance for normal wrapping. Flex items with `overflow` `auto`, `scroll`, or `hidden` already get a zero automatic minimum (`internal/layout/flex.go:2075-2084`). That zero is unconditional. It does not read a property, and it does not run for an ordinary block.

`legacy` matches the opened flex and measure code, so a static box does not move. `zero-if-scroll` on a shrink-to-fit block would need a min-content rule for scroll containers that this draft has not settled in a browser, and the name may still change. The row stays deferred.

```html
<div id="item" style="width:200px;height:50px;overflow-y:scroll;min-intrinsic-sizing:zero-if-scroll">
  <div style="height:400px"></div>
</div>
```

The border box stays `200px` by `50px`, which is the specified size. `TestBehaviorOverflowYScrollClipsTallChild` already clips a `200px` by `50px` scroll container (`internal/layout/css_behavior_interaction_test.go:99-113`). The property must not change that rectangle until a spec and a containing-block min-content rule exist.

- [ ] Acceptance in `internal/layout/style_min_intrinsic.go`. `legacy`, `zero-if-scroll`, `zero-if-extrinsic`, and the two flags together. A bad token keeps the previous value.
- [ ] Storage. The keyword flags on `ResolvedStyle`, initial `legacy`.
- [~] Consumer and behavior test `TestBehaviorMinIntrinsicSizingCompressesMinContent`. Reason: Level 4 calls itself exploratory, the draft still says the name may change, and no shipped browser behavior was found. `legacy` cannot move a box, and `zero-if-scroll` needs a min-content contribution rule for a scroll container's containing block that this tree does not have outside the flex automatic minimum. Owner: this file. Next gate: a spec or browser rule that names the used min-content width of a shrink-to-fit parent, then a test that compares `legacy` and `zero-if-scroll` on that parent. Until then the `200px` by `50px` box above stays `200px` by `50px`.
- [~] Catalog flip. Reason, owner, and next gate match the consumer row. Status stays `unsupported`.

### Frame sizing

Catalog status `unsupported`. The `values` list is empty. Upstream syntax `auto | content-width | content-height | content-block-size | content-inline-size`. Href `https://drafts.csswg.org/css-sizing-4/#propdef-frame-sizing`. The draft says the initial value is `auto`, which ignores any internal layout size. The other values copy one axis from that internal size. In HTML the only element with an internal layout size is an `iframe`, and only after the embedded document opts in with `<meta name="responsive-embedded-sizing">`. The spec's own example is an iframe whose inner document has `<div style="height:500px">`: the iframe height becomes `500px` after that document loads. `auto` keeps the iframe's own height.

No handler. The only `iframe` mention under `internal/layout` is a replaced-element tag list (`internal/layout/flex_simple_height.go:695`). There is no second document and no embedded layout size to read. This output is one canvas.

```html
<iframe id="frame" style="width:300px;height:150px;frame-sizing:content-height"></iframe>
```

There is no inner document, so the border box stays `300px` by `150px` for `content-height` and for `auto`. A div with `frame-sizing:content-height` is unchanged for the same reason: the draft limits the property to replaced elements that have an internal layout size.

- [ ] Acceptance in `internal/layout/style_frame_sizing.go`. The five keywords. A bad token keeps the previous keyword.
- [ ] Storage. One keyword on `ResolvedStyle`, initial `auto`.
- [~] Consumer and behavior test `TestBehaviorFrameSizingUsesEmbeddedHeight`. Reason: the draft's height change needs the embedded document's layout size, and this tree has no embedded document. Owner: this file. Next gate: a nested layout for an opted-in iframe, then a test where `frame-sizing:content-height` makes the iframe border box `500px` tall from an inner `height:500px` div, while `auto` keeps the iframe's own height. Do not invent that nested document in this phase. `internal/layout/layout.go` stays untouched.
- [~] Catalog flip. Reason, owner, and next gate match the consumer row. Status stays `unsupported`.

**Gate:** phase 5 stays open. Parsing tests may land. No catalog row flips.

## Dependencies

Phase 1 uses the width, height, min, and max setters that phase 0 already tests. Phase 2 multiplies the lengths those setters store, and also multiplies a literal `width` and `height` when the shorthands are absent. Phase 3 depends on `flex-wrap: balance`, which this file does not implement. Phase 4 depends on a progress sample and on `calc-size()`, which is not in the tree. Phase 5 depends on a min-content rule for scroll containers and on an embedded iframe document. Neither exists here.

Future test command, not run for this ledger:

`go test ./internal/layout -run 'TestBehaviorMinSizeClampsUsedBorderBox|TestBehaviorMaxSizeClampsUsedBorderBox|TestBehaviorSizeShorthandSetsUsedBorderBox|TestBehaviorZoomScalesUsedBorderBox|TestBehaviorFlexLineCountNowrapKeepsOneLine|TestBehaviorFlexLineCountBalanceFourLines|TestBehaviorInterpolateSizeStaticBoxUnchanged|TestBehaviorInterpolateSizeHalfwayHeight|TestBehaviorMinIntrinsicSizingCompressesMinContent|TestBehaviorFrameSizingUsesEmbeddedHeight'`

Closure after the open rows that are allowed to close:

- `make test`
- `make golden`
- `make lint`
- `make claim-scan`
- `make catalog-check`
- `make matrix-check`

Do not run those commands while this file is only a plan. Do not flip a catalog row whose behavior row is still `[ ]` or `[~]`. Do not grow `internal/layout/layout.go`.
