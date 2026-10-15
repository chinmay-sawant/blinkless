# v0.0.3 - Scroll

> **Parent:** `plans/v0.0.3/README.md` - index of these ledgers.
> **Status:** not started. Baseline rows record what the code already does. Implementation rows are open.
> **Estimated effort:** 48 CSS property names across 8 phases. Five physical scroll-margin names are already stored. The other 43 are unsupported. No WebKit aliases in this set.

---

## Overview

Scroll properties describe snap positions, padding, margins, overscroll, and scrollbar chrome. Blinkless does not scroll. Production paint clips overflow to the padding box at offset zero. This ledger stores the values on the box in the drawing list. A host applies the offset.

The result is `layout.Display` from `layout.DisplayList` (`layout/displaylist.go:129-138`). There is no page PNG, no PDF, and no JavaScript. There is no scrollport in this repo.

`documentation/deferred.md:111` calls a 41-name scroll set a permanent print non-goal, and it says `scroll-margin-*` is not implemented. That is behind the code for five physical names, and it does not list `view-timeline*` or `pointer-timeline*`. `view-timeline*` is on the anchor row at `documentation/deferred.md:113`. `pointer-timeline*` is inside the 45-name animation row at `documentation/deferred.md:110`. This file owns all 48 names below. It does not own `anchor-name` or `position-anchor`. Those are `plans/v0.0.3/anchors/anchors-checklist.md`.

The host scrollport is already sketched in `plans/v0.0.1/native-window/native-web-view-checklist.md` phase 2 (lines 47-55). Phase 7 of this ledger points there. Do not edit `deferred.md` in the planning session.

## Executive summary

| Property | Catalog line | Status | Pinned syntax |
|---|---:|---|---|
| `overscroll-behavior-x` | 12095 | unsupported | `contain | none | auto | chain` |
| `overscroll-behavior-y` | 12111 | unsupported | `contain | none | auto | chain` |
| `overscroll-behavior-block` | 12063 | unsupported | `contain | none | auto | chain` |
| `overscroll-behavior-inline` | 12079 | unsupported | `contain | none | auto | chain` |
| `overscroll-behavior` | 12047 | unsupported | `[ contain | none | auto | chain ]{1,2}` |
| `scroll-behavior` | 13863 | unsupported | `auto | smooth` |
| `scroll-axis-lock` | 13847 | unsupported | `auto | none` |
| `scroll-initial-target` | 13879 | unsupported | `none | nearest` |
| `scroll-marker-group` | 14106 | unsupported | `none | [ [ before | after ] || [ links | tabs ] ]` |
| `scroll-target-group` | 14346 | unsupported | `none | auto` |
| `scroll-margin-top` | 14083 | implemented | `<length>` |
| `scroll-margin-right` | 14060 | implemented | `<length>` |
| `scroll-margin-bottom` | 13966 | implemented | `<length>` |
| `scroll-margin-left` | 14037 | implemented | `<length>` |
| `scroll-margin` | 13895 | implemented | `<length>{1,4}` |
| `scroll-margin-block-start` | 13950 | unsupported | `<length>` |
| `scroll-margin-block-end` | 13934 | unsupported | `<length>` |
| `scroll-margin-inline-start` | 14021 | unsupported | `<length>` |
| `scroll-margin-inline-end` | 14005 | unsupported | `<length>` |
| `scroll-margin-block` | 13918 | unsupported | `<length>{1,2}` |
| `scroll-margin-inline` | 13989 | unsupported | `<length>{1,2}` |
| `scroll-padding-top` | 14282 | unsupported | `auto | <length-percentage [0,inf]>` |
| `scroll-padding-right` | 14266 | unsupported | `auto | <length-percentage [0,inf]>` |
| `scroll-padding-bottom` | 14186 | unsupported | `auto | <length-percentage [0,inf]>` |
| `scroll-padding-left` | 14250 | unsupported | `auto | <length-percentage [0,inf]>` |
| `scroll-padding-block-start` | 14170 | unsupported | `auto | <length-percentage [0,inf]>` |
| `scroll-padding-block-end` | 14154 | unsupported | `auto | <length-percentage [0,inf]>` |
| `scroll-padding-inline-start` | 14234 | unsupported | `auto | <length-percentage [0,inf]>` |
| `scroll-padding-inline-end` | 14218 | unsupported | `auto | <length-percentage [0,inf]>` |
| `scroll-padding-block` | 14138 | unsupported | `[ auto | <length-percentage [0,inf]> ]{1,2}` |
| `scroll-padding-inline` | 14202 | unsupported | `[ auto | <length-percentage [0,inf]> ]{1,2}` |
| `scroll-padding` | 14122 | unsupported | `[ auto | <length-percentage [0,inf]> ]{1,4}` |
| `scroll-snap-align` | 14298 | unsupported | `[ none | start | end | center ]{1,2}` |
| `scroll-snap-stop` | 14314 | unsupported | `normal | always` |
| `scroll-snap-type` | 14330 | unsupported | `none | [ x | y | block | inline | both | pair ] [ mandatory | proximity ]?` |
| `scroll-timeline-name` | 14394 | unsupported | `[ none | <dashed-ident> ]#` |
| `scroll-timeline-axis` | 14378 | unsupported | `[ block | inline | x | y ]#` |
| `scroll-timeline` | 14362 | unsupported | `[ <'scroll-timeline-name'> <'scroll-timeline-axis'>? ]#` |
| `view-timeline-name` | 17082 | unsupported | `[ none | <dashed-ident> ]#` |
| `view-timeline-axis` | 17050 | unsupported | `[ block | inline | x | y ]#` |
| `view-timeline-inset` | 17066 | unsupported | `[ [ auto | <length-percentage> ]{1,2} ]#` |
| `view-timeline` | 17034 | unsupported | `[ <'view-timeline-name'> [ <'view-timeline-axis'> || <'view-timeline-inset'> ]? ]#` |
| `pointer-timeline-name` | 12774 | unsupported | `[ none | <dashed-ident> ]#` |
| `pointer-timeline-axis` | 12758 | unsupported | `[ block | inline | x | y ]#` |
| `pointer-timeline` | 12742 | unsupported | `[ <'pointer-timeline-name'> <'pointer-timeline-axis'>? ]#` |
| `scrollbar-color` | 14410 | unsupported | `auto | <color>{2}` |
| `scrollbar-gutter` | 14426 | unsupported | `auto | stable && both-edges?` |
| `scrollbar-width` | 14442 | unsupported | `auto | thin | none` |

`inf` stands in for the infinity sign in the catalog. Rows were read on 2026-10-15. Catalog header totals: 785 properties, 389 implemented, 19 partial, 369 unsupported, 8 intentionally ignored (`testdata/css/catalog/properties.json:11-18`, `updated` `2026-10-13` at line 4). The 43 unsupported rows have empty `handlers` and empty `tests`. A search of `*.go` found `scroll-snap-align` and `scroll-snap-type` only in comments that say snapping is not implemented (`internal/layout/scroll.go:20-22`).

## Phase 0: lock the contract and the baseline

### 0.1 Pipeline and the missing scrollport

- [x] The public pipeline is `html.Parse` (`html/parse.go:36`), `css.Apply` (`css/css.go:98`), and `layout.DisplayList` (`layout/displaylist.go:137`). `css.Options` is `WidthPx`, `HeightPx`, `Media`, `Extra`, `Focus`, `Hover`, and `Active` (`css/css.go:69-78`).
- [x] `layout.Box` is id, tag, action, text, and the border box in CSS pixels (`layout/layout.go:10-21`). `PlacedElement` is the same fields in points (`internal/layout/placed.go:7-16`). `boxesFrom` copies those fields only (`layout/layout.go:35-48`). WASM `drawBoxJSON` copies the same six (`bindings/wasm/drawing_list.go:194-203`, loop at `296-309`). No scroll field is on the public box.
- [x] Overflow clipping reads `scrollOffsetAwarePort` (`internal/layout/overflow_clip.go:116-125`). That helper adds the carried offset and does not inset by scroll-margin (`internal/layout/scroll_runtime.go:90-102`). The comment at `overflow_clip.go:120-123` says per-target margins are not part of the scroller clip.
- [x] The carried offset is a test map. `setTestScrollOffset` is documented as test-only (`internal/layout/scroll_runtime.go:42-44`). Production misses the map and reads zero (`scroll_runtime.go:77-88`). `scroll.go:11-13` says production keeps the zero offset.
- [x] `internal/layout/layout.go` is 2142 lines (`scripts/file-size-allowlist.txt:9`). Do not add scroll code there.
- [x] GitHub CI runs only `make test` and `make lint` (`.github/workflows/ci.yml:19-26`).

### 0.2 Physical scroll-margin is stored, and the catalog claim is ahead of the public box

- [x] `applyScrollProps` (`internal/layout/scroll.go:49-76`) stores non-negative lengths on `ResolvedStyle.ScrollMarginTop`, `ScrollMarginRight`, `ScrollMarginBottom`, and `ScrollMarginLeft` (`internal/layout/style.go:395-398`). It is the last `styleGroups` entry (`internal/layout/style_cascade.go:1676`).
- [x] The shorthand expands one to four lengths in top, right, bottom, left order (`internal/layout/scroll.go:87-120`). A negative token or a bad token writes nothing (`scroll.go:99-103`, `scroll.go:123-133`).
- [x] Interning compares and hashes the four floats (`internal/layout/style_intern_gen.go:358-361`, `729-732`) and lists the field names (`style_intern_gen.go:1084-1087`).
- [x] `scrollMarginsOf` reads those fields (`internal/layout/scroll.go:156-168`). `TestScrollMarginShorthandInsetsVisibleRect` fails if `scroll-margin: 40px` does not store 40px on all four sides (`internal/layout/css_behavior_scroll_test.go:74-80`). The four longhand tests are `TestScrollMarginTopShiftsVisibleRect` (`css_behavior_scroll_test.go:111`), `TestScrollMarginBottomShiftsVisibleRect` (`134`), `TestScrollMarginLeftShiftsVisibleRect` (`158`), and `TestScrollMarginRightShiftsVisibleRect` (`180`). `TestScrollMarginPairShorthandExpands` checks two tokens (`203`). `TestScrollMarginRejectsNegative` checks `-10px` and `40px bogus` (`275`). `TestScrollRuntimeMarginsShiftVisibleClip` checks the same five declarations (`internal/layout/css_behavior_scroll_runtime_test.go:121-134`).
- [x] Those tests read `item.style` through `scrollMarginsOf`. They do not read `layout.Box`. `TestScrollMarginCoveringPortReadsEmpty` builds a `ScrollMargins` value in the test (`css_behavior_scroll_test.go:250-261`). It does not parse CSS. The runtime "trimmed fill" is a probe `Op` clipped inside the test (`css_behavior_scroll_runtime_test.go:206-218`), not an op from `Result.Ops`.
- [x] `TestBehaviorScrollMarginNoPaintEffect` and the four longhand copies (`internal/layout/css_behavior_overflow2_test.go:406-468`) only compare paint ops. Their comments still say there is no stored field (`css_behavior_overflow2_test.go:399-405`). That comment is stale. The tests still pass because paint does not use the margin. They do not fail if the stored length is dropped.
- [x] The catalog status for the five names is `implemented`. `scroll-margin` names handler `applySVGPresentationProps` and test `TestBehaviorScrollMarginNoPaintEffect` (`testdata/css/catalog/properties.json:13895-13915`). The other four physical rows follow that pattern (`13966`, `14037`, `14060`, `14083`). `applySVGPresentationProps` only forwards those names to `applyLeftoversProps` (`internal/layout/style_paint_props.go:120-127`). `applyLeftoversProps` has no scroll-margin case and returns false (`internal/layout/style_leftovers.go:60-61`). The writer is `applyScrollProps`.
- [x] Matrix section 2 repeats the paint-props file and the NoPaintEffect tests (`documentation/compatibility-matrix.md:373-377`). Section 5.4 still lists the names inside the unsupported 41 and says the catalog records them as Partial (`compatibility-matrix.md:1123` and `1131-1135`). `deferred.md:111` still says they are not implemented. Those sentences disagree with `properties.json` and with `scroll.go`.
- [x] No three-token or four-token shorthand test was found. The code has those arms (`scroll.go:113-116`). They stay open in phase 4.
- [x] There is no `@supports` test for scroll-margin. `supportedDeclaration` has no scroll-margin case, so an unlisted name is accepted (`style_value_accept.go:135-138`). `applyScrollProps` returns true even when the length is rejected (`scroll.go:54-57`). Reading those two functions, `@supports (scroll-margin-top: -10px)` is true today while the used value stays 0. That is a code reading, not a test result.

### 0.3 Contract to implement

- [ ] Copy snap, padding, margin, overscroll, scrollbar, and timeline values onto `layout.Box.Scroll`. Lengths are CSS pixels, matching `layout.Box` (`layout/layout.go:10-11`). The internal store may keep points. The host-facing number is CSS pixels. `40px` must not be reported as `30` because 40px is 30pt (`internal/layout/css_behavior_scroll_test.go:10-11` states 1px = 0.75pt).
- [ ] Logical longhands write the physical fields using the writing-mode and direction map already used for margins (`internal/layout/style_logical_box.go:8-15`, block pair at `105-114`). Put the scroll function in `internal/layout/scroll_style.go`. Do not grow `style_logical_box.go` with scroll cases.
- [ ] Do not turn `testScrollOffsets` into a product scrollport. Applying an offset stays with the host. Phase 7 points at the native view plan.
- [ ] Storing `scroll-snap-align` and `scroll-snap-type` does not choose a snap offset. `scroll.go:20-26` already leaves that pass out.
- [ ] Timeline names are stored. Joining them to `animation-timeline` is the host's work. The animation ledger says the same thing.
- [ ] Do not paint scrollbar chrome and do not generate scroll-marker boxes.
- [ ] New parsers live in `internal/layout/scroll_style.go`. Copy to the public box in `internal/layout/scroll_payload.go`. `applyScrollProps` stays the writer for the five physical margin names. Register one new group, `applyScrollStyleProps`, next to it in `styleGroups` (`internal/layout/style_cascade.go:1676`) for every other name in this file. Do not add lines to `internal/layout/layout.go`.
- [ ] A name is done only when the public box shows the value and a test through `layout.DisplayList` fails if it is removed. The existing scroll-margin tests do not meet that bar. Catalog status `implemented` is a claim (`properties.json:9`).

**Gate:** physical margin storage stays. Public payload, logical longhands, and the other 43 names stay open. No scrollport lands in this ledger.

## Phase 1: style fields and interning

Physical margin fields already exist. Do not add a second copy. Other names get fields on `ResolvedStyle` (`internal/layout/style.go`), parsed in `internal/layout/scroll_style.go`. Regenerate interning from `internal/layout/style.go:2`.

### Overscroll

- [ ] `overscroll-behavior-x` (`properties.json:12095`, syntax `contain | none | auto | chain`). Store the used value on `ResolvedStyle`. Store the keyword. The public-box test uses `contain`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `overscroll-behavior-y` (`properties.json:12111`, syntax `contain | none | auto | chain`). Store the used value on `ResolvedStyle`. The test uses `none` so it cannot pass by copying the x value. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `overscroll-behavior-block` (`properties.json:12063`, syntax `contain | none | auto | chain`). Store the used value on `ResolvedStyle`. Map onto x or y with the writing-mode rules in `applyLogicalMarginBlockPair` (`internal/layout/style_logical_box.go:105-114`). Horizontal-tb writes y. `vertical-rl` writes x. The test checks both modes. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `overscroll-behavior-inline` (`properties.json:12079`, syntax `contain | none | auto | chain`). Store the used value on `ResolvedStyle`. Map onto the inline axis. `direction` changes inline start and end for margins (`internal/layout/style_logical_box.go:21-29`). Use that rule for the inline axis too. The test uses `rtl`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `overscroll-behavior` (`properties.json:12047`, syntax `[ contain | none | auto | chain ]{1,2}`). No field of its own. The expander writes the longhands. One keyword sets x and y. Two keywords set x then y. The test uses `contain none`.
### Scroll behavior and grouping

- [ ] `scroll-behavior` (`properties.json:13863`, syntax `auto | smooth`). Store the used value on `ResolvedStyle`. Store the keyword. The test uses `smooth`. Do not animate the scroll. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-axis-lock` (`properties.json:13847`, syntax `auto | none`). Store the used value on `ResolvedStyle`. The test uses `none`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-initial-target` (`properties.json:13879`, syntax `none | nearest`). Store the used value on `ResolvedStyle`. The test uses `nearest`. Do not scroll to the element. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-marker-group` (`properties.json:14106`, syntax `none | [ [ before | after ] || [ links | tabs ] ]`). Store the used value on `ResolvedStyle`. The test uses `before links` and fails if either keyword is dropped. Do not generate marker boxes. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-target-group` (`properties.json:14346`, syntax `none | auto`). Store the used value on `ResolvedStyle`. The test uses `auto`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
### Scroll margin

- [x] `scroll-margin-top` (`properties.json:14083`, syntax `<length>`). `ResolvedStyle.ScrollMarginTop` is written by `applyScrollMarginSide` (`internal/layout/scroll.go:79-85`) and interned (`internal/layout/style_intern_gen.go:1084-1087`). `TestScrollMarginTopShiftsVisibleRect` reads it (`internal/layout/css_behavior_scroll_test.go:111`). Physical top. Already stored. See the phase 0 baseline.
- [x] `scroll-margin-right` (`properties.json:14060`, syntax `<length>`). `ResolvedStyle.ScrollMarginRight` is written by `applyScrollMarginSide` (`internal/layout/scroll.go:79-85`) and interned (`internal/layout/style_intern_gen.go:1084-1087`). `TestScrollMarginRightShiftsVisibleRect` reads it (`internal/layout/css_behavior_scroll_test.go:180`). Physical right. Already stored.
- [x] `scroll-margin-bottom` (`properties.json:13966`, syntax `<length>`). `ResolvedStyle.ScrollMarginBottom` is written by `applyScrollMarginSide` (`internal/layout/scroll.go:79-85`) and interned (`internal/layout/style_intern_gen.go:1084-1087`). `TestScrollMarginBottomShiftsVisibleRect` reads it (`internal/layout/css_behavior_scroll_test.go:134`). Physical bottom. Already stored.
- [x] `scroll-margin-left` (`properties.json:14037`, syntax `<length>`). `ResolvedStyle.ScrollMarginLeft` is written by `applyScrollMarginSide` (`internal/layout/scroll.go:79-85`) and interned (`internal/layout/style_intern_gen.go:1084-1087`). `TestScrollMarginLeftShiftsVisibleRect` reads it (`internal/layout/css_behavior_scroll_test.go:158`). Physical left. Already stored.
- [x] `scroll-margin` (`properties.json:13895`, syntax `<length>{1,4}`). No field of its own. `applyScrollMarginShorthand` writes the four physical fields (`internal/layout/scroll.go:87-120`). One-token and two-token forms have tests (`css_behavior_scroll_test.go:74` and `203`). One to four non-negative lengths, margin order. Already expanded in `applyScrollMarginShorthand`.
- [ ] `scroll-margin-block-start` (`properties.json:13950`, syntax `<length>`). Store the used value on `ResolvedStyle`. Map with `applyLogicalMarginBlockPair`: horizontal-tb writes top (`style_logical_box.go:112-114`), `vertical-rl` writes right (`106-108`). The public test uses `40px` and checks that physical field. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-margin-block-end` (`properties.json:13934`, syntax `<length>`). Store the used value on `ResolvedStyle`. Horizontal-tb writes bottom. The test uses `20px`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-margin-inline-start` (`properties.json:14021`, syntax `<length>`). Store the used value on `ResolvedStyle`. Follow `applyLogicalMargin` including `direction` (`style_logical_box.go:8-15` and `21-29`). The test covers `ltr` and `rtl`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-margin-inline-end` (`properties.json:14005`, syntax `<length>`). Store the used value on `ResolvedStyle`. The opposite inline side from inline-start. The test uses a different length. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-margin-block` (`properties.json:13918`, syntax `<length>{1,2}`). No field of its own. The expander writes the longhands. One length sets block-start and block-end. Two lengths set start then end. The test uses `40px 10px` in horizontal-tb and expects top 40px and bottom 10px on the public box.
- [ ] `scroll-margin-inline` (`properties.json:13989`, syntax `<length>{1,2}`). No field of its own. The expander writes the longhands. One or two lengths on the inline sides, after the direction map. The test uses `30px 5px`.
### Scroll padding

- [ ] `scroll-padding-top` (`properties.json:14282`, syntax `auto | <length-percentage [0,inf]>`). Store the used value on `ResolvedStyle`. Store `auto` or a non-negative CSS pixel length or a percentage. Do not reuse `parseScrollMarginLength` (`internal/layout/scroll.go:123`): that helper rejects percentages and has no `auto`. The test uses `20px`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-padding-right` (`properties.json:14266`, syntax `auto | <length-percentage [0,inf]>`). Store the used value on `ResolvedStyle`. The test uses `10%` and fails if the percentage is dropped or treated as 10px. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-padding-bottom` (`properties.json:14186`, syntax `auto | <length-percentage [0,inf]>`). Store the used value on `ResolvedStyle`. The test uses `auto` and fails if it is stored as 0px. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-padding-left` (`properties.json:14250`, syntax `auto | <length-percentage [0,inf]>`). Store the used value on `ResolvedStyle`. The test uses `8px`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-padding-block-start` (`properties.json:14170`, syntax `auto | <length-percentage [0,inf]>`). Store the used value on `ResolvedStyle`. Same writing-mode map as scroll-margin-block-start. The test uses `12px` in horizontal-tb and expects the top padding field. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-padding-block-end` (`properties.json:14154`, syntax `auto | <length-percentage [0,inf]>`). Store the used value on `ResolvedStyle`. The test uses `6px` and expects the bottom field in horizontal-tb. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-padding-inline-start` (`properties.json:14234`, syntax `auto | <length-percentage [0,inf]>`). Store the used value on `ResolvedStyle`. Include `direction`. The test covers `rtl`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-padding-inline-end` (`properties.json:14218`, syntax `auto | <length-percentage [0,inf]>`). Store the used value on `ResolvedStyle`. The test uses a different length from inline-start. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-padding-block` (`properties.json:14138`, syntax `[ auto | <length-percentage [0,inf]> ]{1,2}`). No field of its own. The expander writes the longhands. One value sets both block sides. Two values set start then end.
- [ ] `scroll-padding-inline` (`properties.json:14202`, syntax `[ auto | <length-percentage [0,inf]> ]{1,2}`). No field of its own. The expander writes the longhands. One or two values on the inline sides.
- [ ] `scroll-padding` (`properties.json:14122`, syntax `[ auto | <length-percentage [0,inf]> ]{1,4}`). No field of its own. The expander writes the longhands. One to four values in top, right, bottom, left order, like `scroll-margin`. `auto` is legal on any side. A negative length rejects the whole declaration.
### Snap

- [ ] `scroll-snap-align` (`properties.json:14298`, syntax `[ none | start | end | center ]{1,2}`). Store the used value on `ResolvedStyle`. Store block and inline keywords. One keyword sets both. The test uses `start center`. Do not choose a snap offset. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-snap-stop` (`properties.json:14314`, syntax `normal | always`). Store the used value on `ResolvedStyle`. The test uses `always`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-snap-type` (`properties.json:14330`, syntax `none | [ x | y | block | inline | both | pair ] [ mandatory | proximity ]?`). Store the used value on `ResolvedStyle`. Store the axis and the strictness. The test uses `y mandatory`. `scroll.go:20-22` already says choosing a snap offset is not implemented. This row stores the value. It does not add that pass. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
### Scroll timeline

- [ ] `scroll-timeline-name` (`properties.json:14394`, syntax `[ none | <dashed-ident> ]#`). Store the used value on `ResolvedStyle`. Store `none` or the dashed-ident list. The test uses `--page`. The animation ledger stores `animation-timeline` names. The host joins them. This row does not. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-timeline-axis` (`properties.json:14378`, syntax `[ block | inline | x | y ]#`). Store the used value on `ResolvedStyle`. Store one axis keyword per list index, aligned with the name list. The test uses `x`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scroll-timeline` (`properties.json:14362`, syntax `[ <'scroll-timeline-name'> <'scroll-timeline-axis'>? ]#`). No field of its own. The expander writes the longhands. Each item writes a name and an optional axis. A missing axis leaves the initial axis. The test uses `--page y`.
### View timeline

- [ ] `view-timeline-name` (`properties.json:17082`, syntax `[ none | <dashed-ident> ]#`). Store the used value on `ResolvedStyle`. The test uses `--card`. These four names are in the anchor deferred row today (`documentation/deferred.md:113`), not the scroll row. This ledger owns them. `anchor-name` and `position-anchor` stay in `plans/v0.0.3/anchors/anchors-checklist.md`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `view-timeline-axis` (`properties.json:17050`, syntax `[ block | inline | x | y ]#`). Store the used value on `ResolvedStyle`. The test uses `block`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `view-timeline-inset` (`properties.json:17066`, syntax `[ [ auto | <length-percentage> ]{1,2} ]#`). Store the used value on `ResolvedStyle`. Store one or two insets per list index. The test uses `20px 10px` and fails if either number is dropped. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `view-timeline` (`properties.json:17034`, syntax `[ <'view-timeline-name'> [ <'view-timeline-axis'> || <'view-timeline-inset'> ]? ]#`). No field of its own. The expander writes the longhands. Expand into name, optional axis, and optional inset. The test uses `--card block 10px`.
### Pointer timeline

- [ ] `pointer-timeline-name` (`properties.json:12774`, syntax `[ none | <dashed-ident> ]#`). Store the used value on `ResolvedStyle`. The test uses `--ptr`. These three names sit in the 45-name animation sentence (`documentation/deferred.md:110`). This ledger owns them. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `pointer-timeline-axis` (`properties.json:12758`, syntax `[ block | inline | x | y ]#`). Store the used value on `ResolvedStyle`. The test uses `inline`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `pointer-timeline` (`properties.json:12742`, syntax `[ <'pointer-timeline-name'> <'pointer-timeline-axis'>? ]#`). No field of its own. The expander writes the longhands. The test uses `--ptr x`.
### Scrollbar

- [ ] `scrollbar-color` (`properties.json:14410`, syntax `auto | <color>{2}`). Store the used value on `ResolvedStyle`. Store `auto` or two colors. The test uses `red blue` and fails if either color is dropped. Do not paint a scrollbar. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scrollbar-gutter` (`properties.json:14426`, syntax `auto | stable && both-edges?`). Store the used value on `ResolvedStyle`. The test uses `stable both-edges`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `scrollbar-width` (`properties.json:14442`, syntax `auto | thin | none`). Store the used value on `ResolvedStyle`. The test uses `thin`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.

**Gate:** `go test ./internal/layout -run 'TestStyleInternGenerated|TestScrollMargin' -count=1` passes. The four existing margin fields still intern. New fields are in the generated list.

## Phase 2: value acceptance and @supports

Call a new `scrollValueAccepted` from `declarationValueAccepted` (`internal/layout/style_value_accept.go:139`). `@supports` uses `engineSupportsProperty` (`internal/layout/style_cascade.go:1895`). A bad token does not replace an earlier good one (`style_cascade.go:693-695`).

- [ ] **Physical scroll-margin acceptance.** Today a negative length is dropped by the applier (`internal/layout/scroll.go:123-133`, test `TestScrollMarginRejectsNegative` at `css_behavior_scroll_test.go:275`) but `@supports` still follows the default-accept path (`style_value_accept.go:135-138`) and `applyScrollProps` returns true anyway (`scroll.go:54-56`). Make `@supports (scroll-margin-top: -10px)` false, and make the same negative declaration lose the cascade so it cannot sit in the winning map. Proof: `go test ./internal/layout -run 'TestScrollMarginAccept|TestCascadeEngineSupportsPropertyValues' -count=1`.
### Overscroll

- [ ] `overscroll-behavior-x` (`properties.json:12095`). Accept `contain | none | auto | chain`. Store the keyword. The public-box test uses `contain`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `overscroll-behavior-y` (`properties.json:12111`). Accept `contain | none | auto | chain`. The test uses `none` so it cannot pass by copying the x value. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `overscroll-behavior-block` (`properties.json:12063`). Accept `contain | none | auto | chain`. Map onto x or y with the writing-mode rules in `applyLogicalMarginBlockPair` (`internal/layout/style_logical_box.go:105-114`). Horizontal-tb writes y. `vertical-rl` writes x. The test checks both modes. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `overscroll-behavior-inline` (`properties.json:12079`). Accept `contain | none | auto | chain`. Map onto the inline axis. `direction` changes inline start and end for margins (`internal/layout/style_logical_box.go:21-29`). Use that rule for the inline axis too. The test uses `rtl`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `overscroll-behavior` (`properties.json:12047`). Accept `[ contain | none | auto | chain ]{1,2}`. One keyword sets x and y. Two keywords set x then y. The test uses `contain none`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] **Expansion.** `overscroll-behavior` writes the longhands in its note. A CSS-wide keyword resets them. A later longhand wins. Proof: `go test ./internal/layout -run TestScrollExpand -count=1`.
### Scroll behavior and grouping

- [ ] `scroll-behavior` (`properties.json:13863`). Accept `auto | smooth`. Store the keyword. The test uses `smooth`. Do not animate the scroll. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-axis-lock` (`properties.json:13847`). Accept `auto | none`. The test uses `none`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-initial-target` (`properties.json:13879`). Accept `none | nearest`. The test uses `nearest`. Do not scroll to the element. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-marker-group` (`properties.json:14106`). Accept `none | [ [ before | after ] || [ links | tabs ] ]`. The test uses `before links` and fails if either keyword is dropped. Do not generate marker boxes. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-target-group` (`properties.json:14346`). Accept `none | auto`. The test uses `auto`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
### Scroll margin

- [ ] `scroll-margin-top` (`properties.json:14083`). Covered by the physical acceptance row above for negative lengths. Also accept the catalog syntax `<length>` and reject a bad token. Physical top. Already stored. See the phase 0 baseline.
- [ ] `scroll-margin-right` (`properties.json:14060`). Covered by the physical acceptance row above for negative lengths. Also accept the catalog syntax `<length>` and reject a bad token. Physical right. Already stored.
- [ ] `scroll-margin-bottom` (`properties.json:13966`). Covered by the physical acceptance row above for negative lengths. Also accept the catalog syntax `<length>` and reject a bad token. Physical bottom. Already stored.
- [ ] `scroll-margin-left` (`properties.json:14037`). Covered by the physical acceptance row above for negative lengths. Also accept the catalog syntax `<length>` and reject a bad token. Physical left. Already stored.
- [ ] `scroll-margin` (`properties.json:13895`). Covered by the physical acceptance row above for negative lengths. Also accept the catalog syntax `<length>{1,4}` and reject a bad token. One to four non-negative lengths, margin order. Already expanded in `applyScrollMarginShorthand`.
- [x] **Expansion already present.** `scroll-margin` writes one to four physical sides (`internal/layout/scroll.go:110-120`).
- [ ] **Missing expansion tests.** Add the three-token and four-token cases. The arms exist (`scroll.go:113-116`) and no test for them was found. Proof: `go test ./internal/layout -run TestScrollMarginExpand -count=1`.
- [ ] `scroll-margin-block-start` (`properties.json:13950`). Accept `<length>`. Map with `applyLogicalMarginBlockPair`: horizontal-tb writes top (`style_logical_box.go:112-114`), `vertical-rl` writes right (`106-108`). The public test uses `40px` and checks that physical field. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-margin-block-end` (`properties.json:13934`). Accept `<length>`. Horizontal-tb writes bottom. The test uses `20px`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-margin-inline-start` (`properties.json:14021`). Accept `<length>`. Follow `applyLogicalMargin` including `direction` (`style_logical_box.go:8-15` and `21-29`). The test covers `ltr` and `rtl`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-margin-inline-end` (`properties.json:14005`). Accept `<length>`. The opposite inline side from inline-start. The test uses a different length. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-margin-block` (`properties.json:13918`). Accept `<length>{1,2}`. One length sets block-start and block-end. Two lengths set start then end. The test uses `40px 10px` in horizontal-tb and expects top 40px and bottom 10px on the public box. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] **Expansion.** `scroll-margin-block` writes the longhands in its note. A CSS-wide keyword resets them. A later longhand wins. Proof: `go test ./internal/layout -run TestScrollExpand -count=1`.
- [ ] `scroll-margin-inline` (`properties.json:13989`). Accept `<length>{1,2}`. One or two lengths on the inline sides, after the direction map. The test uses `30px 5px`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] **Expansion.** `scroll-margin-inline` writes the longhands in its note. A CSS-wide keyword resets them. A later longhand wins. Proof: `go test ./internal/layout -run TestScrollExpand -count=1`.
### Scroll padding

- [ ] `scroll-padding-top` (`properties.json:14282`). Accept `auto | <length-percentage [0,inf]>`. Store `auto` or a non-negative CSS pixel length or a percentage. Do not reuse `parseScrollMarginLength` (`internal/layout/scroll.go:123`): that helper rejects percentages and has no `auto`. The test uses `20px`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-padding-right` (`properties.json:14266`). Accept `auto | <length-percentage [0,inf]>`. The test uses `10%` and fails if the percentage is dropped or treated as 10px. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-padding-bottom` (`properties.json:14186`). Accept `auto | <length-percentage [0,inf]>`. The test uses `auto` and fails if it is stored as 0px. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-padding-left` (`properties.json:14250`). Accept `auto | <length-percentage [0,inf]>`. The test uses `8px`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-padding-block-start` (`properties.json:14170`). Accept `auto | <length-percentage [0,inf]>`. Same writing-mode map as scroll-margin-block-start. The test uses `12px` in horizontal-tb and expects the top padding field. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-padding-block-end` (`properties.json:14154`). Accept `auto | <length-percentage [0,inf]>`. The test uses `6px` and expects the bottom field in horizontal-tb. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-padding-inline-start` (`properties.json:14234`). Accept `auto | <length-percentage [0,inf]>`. Include `direction`. The test covers `rtl`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-padding-inline-end` (`properties.json:14218`). Accept `auto | <length-percentage [0,inf]>`. The test uses a different length from inline-start. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-padding-block` (`properties.json:14138`). Accept `[ auto | <length-percentage [0,inf]> ]{1,2}`. One value sets both block sides. Two values set start then end. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] **Expansion.** `scroll-padding-block` writes the longhands in its note. A CSS-wide keyword resets them. A later longhand wins. Proof: `go test ./internal/layout -run TestScrollExpand -count=1`.
- [ ] `scroll-padding-inline` (`properties.json:14202`). Accept `[ auto | <length-percentage [0,inf]> ]{1,2}`. One or two values on the inline sides. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] **Expansion.** `scroll-padding-inline` writes the longhands in its note. A CSS-wide keyword resets them. A later longhand wins. Proof: `go test ./internal/layout -run TestScrollExpand -count=1`.
- [ ] `scroll-padding` (`properties.json:14122`). Accept `[ auto | <length-percentage [0,inf]> ]{1,4}`. One to four values in top, right, bottom, left order, like `scroll-margin`. `auto` is legal on any side. A negative length rejects the whole declaration. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] **Expansion.** `scroll-padding` writes the longhands in its note. A CSS-wide keyword resets them. A later longhand wins. Proof: `go test ./internal/layout -run TestScrollExpand -count=1`.
### Snap

- [ ] `scroll-snap-align` (`properties.json:14298`). Accept `[ none | start | end | center ]{1,2}`. Store block and inline keywords. One keyword sets both. The test uses `start center`. Do not choose a snap offset. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-snap-stop` (`properties.json:14314`). Accept `normal | always`. The test uses `always`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-snap-type` (`properties.json:14330`). Accept `none | [ x | y | block | inline | both | pair ] [ mandatory | proximity ]?`. Store the axis and the strictness. The test uses `y mandatory`. `scroll.go:20-22` already says choosing a snap offset is not implemented. This row stores the value. It does not add that pass. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
### Scroll timeline

- [ ] `scroll-timeline-name` (`properties.json:14394`). Accept `[ none | <dashed-ident> ]#`. Store `none` or the dashed-ident list. The test uses `--page`. The animation ledger stores `animation-timeline` names. The host joins them. This row does not. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-timeline-axis` (`properties.json:14378`). Accept `[ block | inline | x | y ]#`. Store one axis keyword per list index, aligned with the name list. The test uses `x`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scroll-timeline` (`properties.json:14362`). Accept `[ <'scroll-timeline-name'> <'scroll-timeline-axis'>? ]#`. Each item writes a name and an optional axis. A missing axis leaves the initial axis. The test uses `--page y`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] **Expansion.** `scroll-timeline` writes the longhands in its note. A CSS-wide keyword resets them. A later longhand wins. Proof: `go test ./internal/layout -run TestScrollExpand -count=1`.
### View timeline

- [ ] `view-timeline-name` (`properties.json:17082`). Accept `[ none | <dashed-ident> ]#`. The test uses `--card`. These four names are in the anchor deferred row today (`documentation/deferred.md:113`), not the scroll row. This ledger owns them. `anchor-name` and `position-anchor` stay in `plans/v0.0.3/anchors/anchors-checklist.md`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `view-timeline-axis` (`properties.json:17050`). Accept `[ block | inline | x | y ]#`. The test uses `block`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `view-timeline-inset` (`properties.json:17066`). Accept `[ [ auto | <length-percentage> ]{1,2} ]#`. Store one or two insets per list index. The test uses `20px 10px` and fails if either number is dropped. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `view-timeline` (`properties.json:17034`). Accept `[ <'view-timeline-name'> [ <'view-timeline-axis'> || <'view-timeline-inset'> ]? ]#`. Expand into name, optional axis, and optional inset. The test uses `--card block 10px`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] **Expansion.** `view-timeline` writes the longhands in its note. A CSS-wide keyword resets them. A later longhand wins. Proof: `go test ./internal/layout -run TestScrollExpand -count=1`.
### Pointer timeline

- [ ] `pointer-timeline-name` (`properties.json:12774`). Accept `[ none | <dashed-ident> ]#`. The test uses `--ptr`. These three names sit in the 45-name animation sentence (`documentation/deferred.md:110`). This ledger owns them. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `pointer-timeline-axis` (`properties.json:12758`). Accept `[ block | inline | x | y ]#`. The test uses `inline`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `pointer-timeline` (`properties.json:12742`). Accept `[ <'pointer-timeline-name'> <'pointer-timeline-axis'>? ]#`. The test uses `--ptr x`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] **Expansion.** `pointer-timeline` writes the longhands in its note. A CSS-wide keyword resets them. A later longhand wins. Proof: `go test ./internal/layout -run TestScrollExpand -count=1`.
### Scrollbar

- [ ] `scrollbar-color` (`properties.json:14410`). Accept `auto | <color>{2}`. Store `auto` or two colors. The test uses `red blue` and fails if either color is dropped. Do not paint a scrollbar. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scrollbar-gutter` (`properties.json:14426`). Accept `auto | stable && both-edges?`. The test uses `stable both-edges`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.
- [ ] `scrollbar-width` (`properties.json:14442`). Accept `auto | thin | none`. The test uses `thin`. Proof: `go test ./internal/layout -run TestScrollAccept -count=1`.

## Phase 3: drawing-list payload and WASM JSON

`internal/layout/scroll_payload.go` copies `Box.Scroll` through `PlacedElement` (`internal/layout/placed.go:7`) and `boxesFrom` (`layout/layout.go:35`). WASM adds omitempty `scroll` on `drawBoxJSON` (`bindings/wasm/drawing_list.go:194`). Schema stays `blinkless.drawinglist/1` (`drawing_list.go:20-23`). Lengths on the public box are CSS pixels. Do not edit `internal/layout/layout.go`.

The five stored margins are included. Their style fields do not count as the drawing-list field.

- [ ] `overscroll-behavior-x` (`properties.json:12095`) is a field on `layout.Box.Scroll` the host can read. Store the keyword. The public-box test uses `contain`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `overscroll-behavior-y` (`properties.json:12111`) is a field on `layout.Box.Scroll` the host can read. The test uses `none` so it cannot pass by copying the x value. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `overscroll-behavior-block` (`properties.json:12063`) is a field on `layout.Box.Scroll` the host can read. Map onto x or y with the writing-mode rules in `applyLogicalMarginBlockPair` (`internal/layout/style_logical_box.go:105-114`). Horizontal-tb writes y. `vertical-rl` writes x. The test checks both modes. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `overscroll-behavior-inline` (`properties.json:12079`) is a field on `layout.Box.Scroll` the host can read. Map onto the inline axis. `direction` changes inline start and end for margins (`internal/layout/style_logical_box.go:21-29`). Use that rule for the inline axis too. The test uses `rtl`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `overscroll-behavior` (`properties.json:12047`) has no JSON key. The host reads the longhand fields. One keyword sets x and y. Two keywords set x then y. The test uses `contain none`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-behavior` (`properties.json:13863`) is a field on `layout.Box.Scroll` the host can read. Store the keyword. The test uses `smooth`. Do not animate the scroll. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-axis-lock` (`properties.json:13847`) is a field on `layout.Box.Scroll` the host can read. The test uses `none`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-initial-target` (`properties.json:13879`) is a field on `layout.Box.Scroll` the host can read. The test uses `nearest`. Do not scroll to the element. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-marker-group` (`properties.json:14106`) is a field on `layout.Box.Scroll` the host can read. The test uses `before links` and fails if either keyword is dropped. Do not generate marker boxes. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-target-group` (`properties.json:14346`) is a field on `layout.Box.Scroll` the host can read. The test uses `auto`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-margin-top` (`properties.json:14083`) is a field on `layout.Box.Scroll` the host can read. Physical top. Already stored. See the phase 0 baseline. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-margin-right` (`properties.json:14060`) is a field on `layout.Box.Scroll` the host can read. Physical right. Already stored. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-margin-bottom` (`properties.json:13966`) is a field on `layout.Box.Scroll` the host can read. Physical bottom. Already stored. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-margin-left` (`properties.json:14037`) is a field on `layout.Box.Scroll` the host can read. Physical left. Already stored. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-margin` (`properties.json:13895`) has no JSON key. The host reads the longhand fields. One to four non-negative lengths, margin order. Already expanded in `applyScrollMarginShorthand`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-margin-block-start` (`properties.json:13950`) is a field on `layout.Box.Scroll` the host can read. Map with `applyLogicalMarginBlockPair`: horizontal-tb writes top (`style_logical_box.go:112-114`), `vertical-rl` writes right (`106-108`). The public test uses `40px` and checks that physical field. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-margin-block-end` (`properties.json:13934`) is a field on `layout.Box.Scroll` the host can read. Horizontal-tb writes bottom. The test uses `20px`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-margin-inline-start` (`properties.json:14021`) is a field on `layout.Box.Scroll` the host can read. Follow `applyLogicalMargin` including `direction` (`style_logical_box.go:8-15` and `21-29`). The test covers `ltr` and `rtl`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-margin-inline-end` (`properties.json:14005`) is a field on `layout.Box.Scroll` the host can read. The opposite inline side from inline-start. The test uses a different length. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-margin-block` (`properties.json:13918`) has no JSON key. The host reads the longhand fields. One length sets block-start and block-end. Two lengths set start then end. The test uses `40px 10px` in horizontal-tb and expects top 40px and bottom 10px on the public box. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-margin-inline` (`properties.json:13989`) has no JSON key. The host reads the longhand fields. One or two lengths on the inline sides, after the direction map. The test uses `30px 5px`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-padding-top` (`properties.json:14282`) is a field on `layout.Box.Scroll` the host can read. Store `auto` or a non-negative CSS pixel length or a percentage. Do not reuse `parseScrollMarginLength` (`internal/layout/scroll.go:123`): that helper rejects percentages and has no `auto`. The test uses `20px`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-padding-right` (`properties.json:14266`) is a field on `layout.Box.Scroll` the host can read. The test uses `10%` and fails if the percentage is dropped or treated as 10px. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-padding-bottom` (`properties.json:14186`) is a field on `layout.Box.Scroll` the host can read. The test uses `auto` and fails if it is stored as 0px. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-padding-left` (`properties.json:14250`) is a field on `layout.Box.Scroll` the host can read. The test uses `8px`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-padding-block-start` (`properties.json:14170`) is a field on `layout.Box.Scroll` the host can read. Same writing-mode map as scroll-margin-block-start. The test uses `12px` in horizontal-tb and expects the top padding field. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-padding-block-end` (`properties.json:14154`) is a field on `layout.Box.Scroll` the host can read. The test uses `6px` and expects the bottom field in horizontal-tb. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-padding-inline-start` (`properties.json:14234`) is a field on `layout.Box.Scroll` the host can read. Include `direction`. The test covers `rtl`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-padding-inline-end` (`properties.json:14218`) is a field on `layout.Box.Scroll` the host can read. The test uses a different length from inline-start. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-padding-block` (`properties.json:14138`) has no JSON key. The host reads the longhand fields. One value sets both block sides. Two values set start then end. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-padding-inline` (`properties.json:14202`) has no JSON key. The host reads the longhand fields. One or two values on the inline sides. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-padding` (`properties.json:14122`) has no JSON key. The host reads the longhand fields. One to four values in top, right, bottom, left order, like `scroll-margin`. `auto` is legal on any side. A negative length rejects the whole declaration. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-snap-align` (`properties.json:14298`) is a field on `layout.Box.Scroll` the host can read. Store block and inline keywords. One keyword sets both. The test uses `start center`. Do not choose a snap offset. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-snap-stop` (`properties.json:14314`) is a field on `layout.Box.Scroll` the host can read. The test uses `always`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-snap-type` (`properties.json:14330`) is a field on `layout.Box.Scroll` the host can read. Store the axis and the strictness. The test uses `y mandatory`. `scroll.go:20-22` already says choosing a snap offset is not implemented. This row stores the value. It does not add that pass. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-timeline-name` (`properties.json:14394`) is a field on `layout.Box.Scroll` the host can read. Store `none` or the dashed-ident list. The test uses `--page`. The animation ledger stores `animation-timeline` names. The host joins them. This row does not. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-timeline-axis` (`properties.json:14378`) is a field on `layout.Box.Scroll` the host can read. Store one axis keyword per list index, aligned with the name list. The test uses `x`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scroll-timeline` (`properties.json:14362`) has no JSON key. The host reads the longhand fields. Each item writes a name and an optional axis. A missing axis leaves the initial axis. The test uses `--page y`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `view-timeline-name` (`properties.json:17082`) is a field on `layout.Box.Scroll` the host can read. The test uses `--card`. These four names are in the anchor deferred row today (`documentation/deferred.md:113`), not the scroll row. This ledger owns them. `anchor-name` and `position-anchor` stay in `plans/v0.0.3/anchors/anchors-checklist.md`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `view-timeline-axis` (`properties.json:17050`) is a field on `layout.Box.Scroll` the host can read. The test uses `block`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `view-timeline-inset` (`properties.json:17066`) is a field on `layout.Box.Scroll` the host can read. Store one or two insets per list index. The test uses `20px 10px` and fails if either number is dropped. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `view-timeline` (`properties.json:17034`) has no JSON key. The host reads the longhand fields. Expand into name, optional axis, and optional inset. The test uses `--card block 10px`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `pointer-timeline-name` (`properties.json:12774`) is a field on `layout.Box.Scroll` the host can read. The test uses `--ptr`. These three names sit in the 45-name animation sentence (`documentation/deferred.md:110`). This ledger owns them. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `pointer-timeline-axis` (`properties.json:12758`) is a field on `layout.Box.Scroll` the host can read. The test uses `inline`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `pointer-timeline` (`properties.json:12742`) has no JSON key. The host reads the longhand fields. The test uses `--ptr x`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scrollbar-color` (`properties.json:14410`) is a field on `layout.Box.Scroll` the host can read. Store `auto` or two colors. The test uses `red blue` and fails if either color is dropped. Do not paint a scrollbar. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scrollbar-gutter` (`properties.json:14426`) is a field on `layout.Box.Scroll` the host can read. The test uses `stable both-edges`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.
- [ ] `scrollbar-width` (`properties.json:14442`) is a field on `layout.Box.Scroll` the host can read. The test uses `thin`. Proof: `go test ./bindings/wasm -run TestDrawingListScroll -count=1`.

## Phase 4: behavior tests

The test that closes a row calls `layout.DisplayList` (`layout/displaylist.go:137`) and reads `Display.Boxes`. The existing `TestScrollMargin*` tests stay as style coverage. They do not close these rows.

- [ ] `overscroll-behavior-x` (`properties.json:12095`). Fail if this value is dropped before `layout.Box.Scroll`. Store the keyword. The public-box test uses `contain`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `overscroll-behavior-y` (`properties.json:12111`). Fail if this value is dropped before `layout.Box.Scroll`. The test uses `none` so it cannot pass by copying the x value. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `overscroll-behavior-block` (`properties.json:12063`). Fail if this value is dropped before `layout.Box.Scroll`. Map onto x or y with the writing-mode rules in `applyLogicalMarginBlockPair` (`internal/layout/style_logical_box.go:105-114`). Horizontal-tb writes y. `vertical-rl` writes x. The test checks both modes. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `overscroll-behavior-inline` (`properties.json:12079`). Fail if this value is dropped before `layout.Box.Scroll`. Map onto the inline axis. `direction` changes inline start and end for margins (`internal/layout/style_logical_box.go:21-29`). Use that rule for the inline axis too. The test uses `rtl`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `overscroll-behavior` (`properties.json:12047`). Fail if this value is dropped before `layout.Box.Scroll`. One keyword sets x and y. Two keywords set x then y. The test uses `contain none`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-behavior` (`properties.json:13863`). Fail if this value is dropped before `layout.Box.Scroll`. Store the keyword. The test uses `smooth`. Do not animate the scroll. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-axis-lock` (`properties.json:13847`). Fail if this value is dropped before `layout.Box.Scroll`. The test uses `none`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-initial-target` (`properties.json:13879`). Fail if this value is dropped before `layout.Box.Scroll`. The test uses `nearest`. Do not scroll to the element. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-marker-group` (`properties.json:14106`). Fail if this value is dropped before `layout.Box.Scroll`. The test uses `before links` and fails if either keyword is dropped. Do not generate marker boxes. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-target-group` (`properties.json:14346`). Fail if this value is dropped before `layout.Box.Scroll`. The test uses `auto`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [x] `scroll-margin-top` style coverage exists: `TestScrollMarginTopShiftsVisibleRect` (`internal/layout/css_behavior_scroll_test.go:111`) fails if `ScrollMarginTop` stays 0 after a 40px declaration. It reads `scrollMarginsOf`, not `layout.Box`.
- [ ] `scroll-margin-top` public box. `TestDisplayScroll` fails if `layout.Box.Scroll` does not show 40 CSS pixels. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [x] `scroll-margin-right` style coverage exists: `TestScrollMarginRightShiftsVisibleRect` (`internal/layout/css_behavior_scroll_test.go:180`) fails if `ScrollMarginRight` stays 0 after a 40px declaration. It reads `scrollMarginsOf`, not `layout.Box`.
- [ ] `scroll-margin-right` public box. `TestDisplayScroll` fails if `layout.Box.Scroll` does not show 40 CSS pixels. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [x] `scroll-margin-bottom` style coverage exists: `TestScrollMarginBottomShiftsVisibleRect` (`internal/layout/css_behavior_scroll_test.go:134`) fails if `ScrollMarginBottom` stays 0 after a 40px declaration. It reads `scrollMarginsOf`, not `layout.Box`.
- [ ] `scroll-margin-bottom` public box. `TestDisplayScroll` fails if `layout.Box.Scroll` does not show 40 CSS pixels. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [x] `scroll-margin-left` style coverage exists: `TestScrollMarginLeftShiftsVisibleRect` (`internal/layout/css_behavior_scroll_test.go:158`) fails if `ScrollMarginLeft` stays 0 after a 40px declaration. It reads `scrollMarginsOf`, not `layout.Box`.
- [ ] `scroll-margin-left` public box. `TestDisplayScroll` fails if `layout.Box.Scroll` does not show 40 CSS pixels. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [x] `scroll-margin` style coverage exists for one token and two tokens (`css_behavior_scroll_test.go:74` and `203`).
- [ ] `scroll-margin` public box, including the three-token and four-token forms that have no test today. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-margin-block-start` (`properties.json:13950`). Fail if this value is dropped before `layout.Box.Scroll`. Map with `applyLogicalMarginBlockPair`: horizontal-tb writes top (`style_logical_box.go:112-114`), `vertical-rl` writes right (`106-108`). The public test uses `40px` and checks that physical field. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-margin-block-end` (`properties.json:13934`). Fail if this value is dropped before `layout.Box.Scroll`. Horizontal-tb writes bottom. The test uses `20px`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-margin-inline-start` (`properties.json:14021`). Fail if this value is dropped before `layout.Box.Scroll`. Follow `applyLogicalMargin` including `direction` (`style_logical_box.go:8-15` and `21-29`). The test covers `ltr` and `rtl`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-margin-inline-end` (`properties.json:14005`). Fail if this value is dropped before `layout.Box.Scroll`. The opposite inline side from inline-start. The test uses a different length. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-margin-block` (`properties.json:13918`). Fail if this value is dropped before `layout.Box.Scroll`. One length sets block-start and block-end. Two lengths set start then end. The test uses `40px 10px` in horizontal-tb and expects top 40px and bottom 10px on the public box. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-margin-inline` (`properties.json:13989`). Fail if this value is dropped before `layout.Box.Scroll`. One or two lengths on the inline sides, after the direction map. The test uses `30px 5px`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-padding-top` (`properties.json:14282`). Fail if this value is dropped before `layout.Box.Scroll`. Store `auto` or a non-negative CSS pixel length or a percentage. Do not reuse `parseScrollMarginLength` (`internal/layout/scroll.go:123`): that helper rejects percentages and has no `auto`. The test uses `20px`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-padding-right` (`properties.json:14266`). Fail if this value is dropped before `layout.Box.Scroll`. The test uses `10%` and fails if the percentage is dropped or treated as 10px. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-padding-bottom` (`properties.json:14186`). Fail if this value is dropped before `layout.Box.Scroll`. The test uses `auto` and fails if it is stored as 0px. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-padding-left` (`properties.json:14250`). Fail if this value is dropped before `layout.Box.Scroll`. The test uses `8px`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-padding-block-start` (`properties.json:14170`). Fail if this value is dropped before `layout.Box.Scroll`. Same writing-mode map as scroll-margin-block-start. The test uses `12px` in horizontal-tb and expects the top padding field. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-padding-block-end` (`properties.json:14154`). Fail if this value is dropped before `layout.Box.Scroll`. The test uses `6px` and expects the bottom field in horizontal-tb. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-padding-inline-start` (`properties.json:14234`). Fail if this value is dropped before `layout.Box.Scroll`. Include `direction`. The test covers `rtl`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-padding-inline-end` (`properties.json:14218`). Fail if this value is dropped before `layout.Box.Scroll`. The test uses a different length from inline-start. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-padding-block` (`properties.json:14138`). Fail if this value is dropped before `layout.Box.Scroll`. One value sets both block sides. Two values set start then end. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-padding-inline` (`properties.json:14202`). Fail if this value is dropped before `layout.Box.Scroll`. One or two values on the inline sides. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-padding` (`properties.json:14122`). Fail if this value is dropped before `layout.Box.Scroll`. One to four values in top, right, bottom, left order, like `scroll-margin`. `auto` is legal on any side. A negative length rejects the whole declaration. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-snap-align` (`properties.json:14298`). Fail if this value is dropped before `layout.Box.Scroll`. Store block and inline keywords. One keyword sets both. The test uses `start center`. Do not choose a snap offset. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-snap-stop` (`properties.json:14314`). Fail if this value is dropped before `layout.Box.Scroll`. The test uses `always`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-snap-type` (`properties.json:14330`). Fail if this value is dropped before `layout.Box.Scroll`. Store the axis and the strictness. The test uses `y mandatory`. `scroll.go:20-22` already says choosing a snap offset is not implemented. This row stores the value. It does not add that pass. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-timeline-name` (`properties.json:14394`). Fail if this value is dropped before `layout.Box.Scroll`. Store `none` or the dashed-ident list. The test uses `--page`. The animation ledger stores `animation-timeline` names. The host joins them. This row does not. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-timeline-axis` (`properties.json:14378`). Fail if this value is dropped before `layout.Box.Scroll`. Store one axis keyword per list index, aligned with the name list. The test uses `x`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scroll-timeline` (`properties.json:14362`). Fail if this value is dropped before `layout.Box.Scroll`. Each item writes a name and an optional axis. A missing axis leaves the initial axis. The test uses `--page y`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `view-timeline-name` (`properties.json:17082`). Fail if this value is dropped before `layout.Box.Scroll`. The test uses `--card`. These four names are in the anchor deferred row today (`documentation/deferred.md:113`), not the scroll row. This ledger owns them. `anchor-name` and `position-anchor` stay in `plans/v0.0.3/anchors/anchors-checklist.md`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `view-timeline-axis` (`properties.json:17050`). Fail if this value is dropped before `layout.Box.Scroll`. The test uses `block`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `view-timeline-inset` (`properties.json:17066`). Fail if this value is dropped before `layout.Box.Scroll`. Store one or two insets per list index. The test uses `20px 10px` and fails if either number is dropped. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `view-timeline` (`properties.json:17034`). Fail if this value is dropped before `layout.Box.Scroll`. Expand into name, optional axis, and optional inset. The test uses `--card block 10px`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `pointer-timeline-name` (`properties.json:12774`). Fail if this value is dropped before `layout.Box.Scroll`. The test uses `--ptr`. These three names sit in the 45-name animation sentence (`documentation/deferred.md:110`). This ledger owns them. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `pointer-timeline-axis` (`properties.json:12758`). Fail if this value is dropped before `layout.Box.Scroll`. The test uses `inline`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `pointer-timeline` (`properties.json:12742`). Fail if this value is dropped before `layout.Box.Scroll`. The test uses `--ptr x`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scrollbar-color` (`properties.json:14410`). Fail if this value is dropped before `layout.Box.Scroll`. Store `auto` or two colors. The test uses `red blue` and fails if either color is dropped. Do not paint a scrollbar. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scrollbar-gutter` (`properties.json:14426`). Fail if this value is dropped before `layout.Box.Scroll`. The test uses `stable both-edges`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.
- [ ] `scrollbar-width` (`properties.json:14442`). Fail if this value is dropped before `layout.Box.Scroll`. The test uses `thin`. Proof: `go test ./layout -run TestDisplayScroll -count=1`.

## Phase 5: catalog, matrix, and the deferred pointer

`make catalog-check` discovers handlers from `styleGroups` (`scripts/css-catalog-map.py:336-370`). For the five physical names, point the handler at `applyScrollProps` and the test at the new public-box test. Keep status `implemented` only if that public test passes. Until it does, the current `implemented` claim rests on `TestBehaviorScrollMarginNoPaintEffect`, which does not fail when the length is dropped (`css_behavior_overflow2_test.go:406-413`).

- [ ] `overscroll-behavior-x` (`properties.json:12095`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `overscroll-behavior-y` (`properties.json:12111`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `overscroll-behavior-block` (`properties.json:12063`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `overscroll-behavior-inline` (`properties.json:12079`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `overscroll-behavior` (`properties.json:12047`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-behavior` (`properties.json:13863`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-axis-lock` (`properties.json:13847`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-initial-target` (`properties.json:13879`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-marker-group` (`properties.json:14106`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-target-group` (`properties.json:14346`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-margin-top` (`properties.json:14083`). Replace handler `applySVGPresentationProps` with `applyScrollProps`. Replace the NoPaintEffect test with the public-box test. Status stays `implemented` only after that public test passes. Proof: `make catalog-check`.
- [ ] `scroll-margin-right` (`properties.json:14060`). Replace handler `applySVGPresentationProps` with `applyScrollProps`. Replace the NoPaintEffect test with the public-box test. Status stays `implemented` only after that public test passes. Proof: `make catalog-check`.
- [ ] `scroll-margin-bottom` (`properties.json:13966`). Replace handler `applySVGPresentationProps` with `applyScrollProps`. Replace the NoPaintEffect test with the public-box test. Status stays `implemented` only after that public test passes. Proof: `make catalog-check`.
- [ ] `scroll-margin-left` (`properties.json:14037`). Replace handler `applySVGPresentationProps` with `applyScrollProps`. Replace the NoPaintEffect test with the public-box test. Status stays `implemented` only after that public test passes. Proof: `make catalog-check`.
- [ ] `scroll-margin` (`properties.json:13895`). Replace handler `applySVGPresentationProps` with `applyScrollProps`. Replace the NoPaintEffect test with the public-box test. Status stays `implemented` only after that public test passes. Proof: `make catalog-check`.
- [ ] `scroll-margin-block-start` (`properties.json:13950`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-margin-block-end` (`properties.json:13934`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-margin-inline-start` (`properties.json:14021`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-margin-inline-end` (`properties.json:14005`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-margin-block` (`properties.json:13918`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-margin-inline` (`properties.json:13989`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-padding-top` (`properties.json:14282`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-padding-right` (`properties.json:14266`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-padding-bottom` (`properties.json:14186`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-padding-left` (`properties.json:14250`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-padding-block-start` (`properties.json:14170`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-padding-block-end` (`properties.json:14154`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-padding-inline-start` (`properties.json:14234`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-padding-inline-end` (`properties.json:14218`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-padding-block` (`properties.json:14138`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-padding-inline` (`properties.json:14202`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-padding` (`properties.json:14122`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-snap-align` (`properties.json:14298`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-snap-stop` (`properties.json:14314`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-snap-type` (`properties.json:14330`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-timeline-name` (`properties.json:14394`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-timeline-axis` (`properties.json:14378`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scroll-timeline` (`properties.json:14362`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `view-timeline-name` (`properties.json:17082`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `view-timeline-axis` (`properties.json:17050`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `view-timeline-inset` (`properties.json:17066`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `view-timeline` (`properties.json:17034`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `pointer-timeline-name` (`properties.json:12774`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `pointer-timeline-axis` (`properties.json:12758`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `pointer-timeline` (`properties.json:12742`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scrollbar-color` (`properties.json:14410`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scrollbar-gutter` (`properties.json:14426`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.
- [ ] `scrollbar-width` (`properties.json:14442`). Set the handler and the public-box test. Status follows that test (`properties.json:9`). Proof: `make catalog-check`.

- [ ] Point `documentation/deferred.md:111` at `plans/v0.0.3/scroll/scroll-checklist.md`. Say the five physical margins are stored and the public box work is this ledger. Remove the blanket "not implemented" sentence for the names this file owns.
- [ ] Take `view-timeline*` out of the blanket sentence at `documentation/deferred.md:113` and `pointer-timeline*` out of `documentation/deferred.md:110`, with a pointer here. Leave `anchor-name` and `position-anchor` on the anchor ledger.
- [ ] Fix the matrix. Section 2 (`documentation/compatibility-matrix.md:373-377`) should name `scroll.go` and the public test. Section 5.4 (`1123`, `1131-1135`) should stop calling the five names Partial and stop listing finished names as unsupported. Regenerate the property section with `python3 scripts/css-catalog-map.py --matrix`. The phase-84 prose is hand-written. Update it in the same change so it matches the catalog.
- [ ] Update the local knowledge-base page in the same session as the code. This planning session does not write that page.

**Gate:** `make catalog-check` and `make matrix-check` exit 0. Record the new totals. The planning baseline was 389 implemented, 19 partial, 369 unsupported, 8 intentionally ignored (`properties.json:12-17`).


## Phase 6: closure gates

These stay open until the commands exit 0. During the work, run one package: `go test ./internal/layout -run '<Name>' -count=1`, `go test ./layout -run '<Name>' -count=1`, or `go test ./bindings/wasm -run '<Name>' -count=1`. Full gates go through the Makefile. Do not run bare `go test ./...`.

- [ ] `make test` exits 0.
- [ ] `make golden` exits 0. That target runs `TestDisplay` in `./layout`. It does not rasterize a page.
- [ ] `make lint` exits 0. It runs the file-size check. `internal/layout/layout.go` stays 2142 lines unless a reviewed allowlist edit says otherwise (`scripts/file-size-allowlist.txt:9`).
- [ ] `make claim-scan` exits 0. `claim-scan` reads `doc.go`, `README.md`, and `documentation/*.md` (`Makefile:104-114`). Do not claim a page PNG, a PDF, a scrollport, or a speech synthesizer.
- [ ] `make catalog-check` exits 0.
- [ ] `make matrix-check` exits 0.
- [ ] GitHub CI (`.github/workflows/ci.yml:1`, steps at `19-26`) runs only `make test` and `make lint`. The other gates are local.

**Gate:** every command above exited 0, and the output was recorded on the row.

## Phase 7: in-engine scrollport

- [~] Do not build a scrollport, do not move boxes by a scroll offset, and do not promote `testScrollOffsets` (`internal/layout/scroll_runtime.go:27-44`) to production. Reason: production clipping uses offset zero (`internal/layout/overflow_clip.go:116-125`). Choosing a snap offset and scrolling to it is the pass `scroll.go:20-26` already leaves out. Owner: the host plan `plans/v0.0.1/native-window/native-web-view-checklist.md` phase 2 (lines 47-55), which already owns scroll positions, overscroll rules, and scrollbars for a live view. Next gate: that host plan names `layout.Box.Scroll` as the fields it will read, and a host test applies an offset without Blinkless moving the box. Until then this repo only stores the values.

## Dependencies

- Physical scroll-margin already depends on `applyScrollProps` (`internal/layout/scroll.go:49`) and the intern generator (`internal/layout/style.go:2`).
- Logical longhands depend on the writing-mode map in `internal/layout/style_logical_box.go`. They do not depend on a scrollport.
- `scroll-padding` must not call `parseScrollMarginLength` (`internal/layout/scroll.go:123`). Percentages and `auto` are legal for padding and illegal for margin.
- Timeline names are joined to `animation-timeline` by the host. See `plans/v0.0.3/animation/animation-checklist.md`.
- The scrollport non-goal depends on the native view plan. This ledger does not start that work.
- `anchor-name` and `position-anchor` are out of scope. See `plans/v0.0.3/anchors/anchors-checklist.md`.
- No new direct module. The allowlist stays `github.com/go-text/typesetting` and `github.com/tdewolff/canvas`.

