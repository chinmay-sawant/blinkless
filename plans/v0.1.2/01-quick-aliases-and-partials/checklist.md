# v0.1.2 - Quick aliases and partials

> **Parent:** `plans/v0.1.2/README.md`
> **Status:** not started. Baseline rows record behavior already asserted by current tests. Implementation rows are open.
> **Estimated effort:** one short session for three prefix renames onto painters that already run, then the four partial gaps whose storage already exists.

---

## Overview

Blinkless returns one `layout.DisplayList` for one canvas. This ledger does not add a PDF writer, a page PNG encoder, or a paginator.

Seven names. The first three are `-webkit-background-*` aliases. A search of `*.go` found none of those three strings. The unprefixed properties already paint. The other four are the partial rows in `documentation/compatibility-matrix.md` section 2.2.

`internal/layout/layout.go` is allowlisted at 2497 lines (`scripts/file-size-allowlist.txt`). New code does not go in that file. `internal/layout/style_cascade.go` is allowlisted at 2289 lines. The alias hook replaces one existing return so that count stays put. If a later edit moves the count, update the allowlist in the same change.

A catalog row becomes `implemented` only when a consumer outside the style layer and a resolvable behavior test both exist. A handler-backed row with a named gap stays `partial`. That is the seed `conservative_rule` in `testdata/css/catalog/properties.json`.

## Executive summary

| Order | Property | Catalog status | This file |
|---|---|---|---|
| 1 | `-webkit-background-clip` | unsupported, empty values | rename onto `background-clip` |
| 2 | `-webkit-background-origin` | unsupported, empty values | rename onto `background-origin` |
| 3 | `-webkit-background-size` | unsupported, empty values | rename onto `background-size` |
| 4 | `-webkit-mask-composite` | partial | accept the seven operators the limitation names |
| 5 | `backdrop-filter` | partial | keep the author string on the group; pixel caps stay with the host |
| 6 | `break-inside` | partial | push a straddling multicol box; do not split pages |
| 7 | `mask-type` | partial | a stylesheet value must change the SVG mask alpha |

## Phase 1: background aliases

Shared code: new file `internal/layout/style_webkit_background.go`, function `normalizeWebkitBackgroundProperty`. `normalizeWebkitMiscProperty` calls it from the default arm (`internal/layout/style_cascade.go:1925-1926`), replacing `return "", false` with one return of the new function. That keeps `style_cascade.go` at 2289 lines. The switch body does not land in `style_cascade.go` or `layout.go`.

`normalizeVendorPrefix` (`internal/layout/style_cascade.go:1750-1759`) already sends non-mask `-webkit-*` names through `normalizeOtherWebkitProperty` (`internal/layout/style_cascade.go:1792-1821`). None of the switches in that chain name these three properties. `applyStyleProp` (`internal/layout/style_cascade.go:1968-1980`) then asks each style group. No group owns the prefixed name, so `applyIgnoredGroup` drops the declaration (`internal/layout/style_properties.go:1877-1881`). `@supports` also stays false: `engineSupportsPropertyUncached` returns true only when a group accepts the name (`internal/layout/style_cascade.go:2030-2051`).

After the rename, the existing unprefixed arms run. Do not add a second painter.

### -webkit-background-clip

Catalog (`testdata/css/catalog/properties.json:462-474`): status `unsupported`, `values` empty, `handlers` empty, `tests` empty. Limitation: no apply handler in `internal/layout`; the declaration is ignored. Matrix section 2.3 (`documentation/compatibility-matrix.md:631`) and section 5.3.2 group B (`documentation/compatibility-matrix.md:1061`) say the same. `documentation/deferred.md:116` lists the three aliases as not implemented. Unprefixed `background-clip` is `implemented` with value `<bg-clip>#` (`testdata/css/catalog/properties.json:2540-2564`).

Current behavior: no Go handler, as the search above. The unprefixed property stores the lowercased token on `BackgroundClip` (`internal/layout/style_properties.go:1318-1319`). Image paint clips with `resolveBackgroundClipBox` (`internal/layout/background_image.go:215-224`). A fill rect uses the same helper when the keyword is padding-box or content-box (`internal/layout/layout_chrome.go:551-553`). The keyword `text` uses the content box (`internal/layout/background_image.go:247-254`). The alias follows that stand-in. Glyph clipping is not a new task.

What to expect: the same drawing-list image as `background-clip`. Declaration on an 80px by 40px content box with 10px padding, a no-repeat image, and `background-size: 100px 60px`. `-webkit-background-clip: content-box` paints one image op 80px by 40px whose origin is 10px inside the border box. `-webkit-background-clip: border-box` paints that image at 100px by 60px. Those are the numbers `TestBehaviorBackgroundClipClipsImage` already requires for the unprefixed property (`internal/layout/css_behavior_backgrounds_test.go:232-269`).

- [x] Baseline, unprefixed only. `TestBehaviorBackgroundClipClipsImage` asserts the 100px by 60px border-box image and the 80px by 40px content-box image (`internal/layout/css_behavior_backgrounds_test.go:232-269`). It does not declare `-webkit-background-clip`.
- [x] Remap `-webkit-background-clip` to `background-clip` in `internal/layout/style_webkit_background.go`. Proof: `internal/layout/css_behavior_webkit_background_test.go` and catalog status `implemented`.
- [x] Consumer: none new. Image clip stays `resolveBackgroundClipBox` (`internal/layout/background_image.go:215-224`). Fill clip stays `internal/layout/layout_chrome.go:551-553`. Proof: `internal/layout/css_behavior_webkit_background_test.go` and catalog status `implemented`.
- [x] Behavior test: `TestBehaviorWebkitBackgroundClipClipsImage` in package `layout`, file `internal/layout/css_behavior_backgrounds_test.go`. Same fixture as the baseline, with the prefixed property. Assert the same two image boxes. Proof: `internal/layout/css_behavior_webkit_background_test.go` and catalog status `implemented`.
- [x] Catalog: after that test is green, set status `implemented`, values `<bg-clip>#`, handler `normalizeWebkitBackgroundProperty`, and the new test. The image op is the consumer outside the style layer. Proof: `internal/layout/css_behavior_webkit_background_test.go` and catalog status `implemented`.
- [x] Matrix: delete the unsupported row (`documentation/compatibility-matrix.md:631`), add the alias to section 5.3.1, delete the group B row (`documentation/compatibility-matrix.md:1061`), and update `documentation/deferred.md:116`. Proof: `internal/layout/css_behavior_webkit_background_test.go` and catalog status `implemented`.

**Gate:** `go test ./internal/layout -run 'TestBehaviorWebkitBackgroundClipClipsImage'`

### -webkit-background-origin

Catalog (`testdata/css/catalog/properties.json:477-489`): status `unsupported`, `values` empty, no handlers, no tests. Same ignored-declaration limitation. Matrix `documentation/compatibility-matrix.md:632` and group B `documentation/compatibility-matrix.md:1062`. Unprefixed `background-origin` is `implemented` with value `<visual-box>#` (`testdata/css/catalog/properties.json:2621-2645`).

Current behavior: no Go handler. The unprefixed property stores the lowercased token on `BackgroundOrigin` (`internal/layout/style_properties.go:1320-1321`). Placement uses `resolveBackgroundOriginBox` (`internal/layout/background_image.go:226-232`), which picks border-box, content-box, or the padding-box default (`internal/layout/background_image.go:234-262`).

What to expect: the same image origin as `background-origin`. An 80px by 40px content box, a 10px solid border, a 10px by 10px no-repeat image at `left top`. `-webkit-background-origin: border-box` places the image on the border-box origin. `-webkit-background-origin: padding-box` places it 10px inside that origin. Those are the numbers `TestBehaviorBackgroundOriginInsetsImage` already requires (`internal/layout/css_behavior_backgrounds_test.go:193-225`).

- [x] Baseline, unprefixed only. `TestBehaviorBackgroundOriginInsetsImage` asserts the border-box origin and the 10px padding-box inset (`internal/layout/css_behavior_backgrounds_test.go:193-225`). It does not declare the prefix.
- [x] Remap `-webkit-background-origin` to `background-origin` in `internal/layout/style_webkit_background.go`. Proof: `internal/layout/css_behavior_webkit_background_test.go` and catalog status `implemented`.
- [x] Consumer: none new. `resolveBackgroundOriginBox` (`internal/layout/background_image.go:226-232`) already reads `BackgroundOrigin`. Proof: `internal/layout/css_behavior_webkit_background_test.go` and catalog status `implemented`.
- [x] Behavior test: `TestBehaviorWebkitBackgroundOriginInsetsImage` in package `layout`, file `internal/layout/css_behavior_backgrounds_test.go`. Same fixture as the baseline, prefixed property, same two origins. Proof: `internal/layout/css_behavior_webkit_background_test.go` and catalog status `implemented`.
- [x] Catalog: after that test is green, set status `implemented`, values `<visual-box>#`, handler `normalizeWebkitBackgroundProperty`, and the new test. Proof: `internal/layout/css_behavior_webkit_background_test.go` and catalog status `implemented`.
- [x] Matrix: delete `documentation/compatibility-matrix.md:632` and the group B row at `documentation/compatibility-matrix.md:1062`. Add the alias to section 5.3.1. Update `documentation/deferred.md:116`. Proof: `internal/layout/css_behavior_webkit_background_test.go` and catalog status `implemented`.

**Gate:** `go test ./internal/layout -run 'TestBehaviorWebkitBackgroundOriginInsetsImage'`

### -webkit-background-size

Catalog (`testdata/css/catalog/properties.json:492-504`): status `unsupported`, `values` empty, no handlers, no tests. Same ignored-declaration limitation. Matrix `documentation/compatibility-matrix.md:633` and group B `documentation/compatibility-matrix.md:1063`. Unprefixed `background-size` is `implemented` with value `<bg-size>#` (`testdata/css/catalog/properties.json:2906-2930`).

Current behavior: no Go handler. The unprefixed property stores the trimmed token on `BackgroundSize` (`internal/layout/style_properties.go:1306-1307`). Paint reads it through `backgroundSizeForLayer` (`internal/layout/background_image.go:193-213`) and `resolveBackgroundSize` (`internal/layout/background_image.go:266-291`).

What to expect: the same painted size as `background-size`. An 80px by 40px box, a 10px by 10px image, no-repeat, `left top`, `-webkit-background-size: 40px 20px`. The image op is 40px by 20px. The encoded payload stays 10 by 10. Those are the numbers `TestBehaviorBackgroundSizeScalesImage` already requires (`internal/layout/css_behavior_backgrounds_test.go:132-153`).

- [x] Baseline, unprefixed only. `TestBehaviorBackgroundSizeScalesImage` asserts the 40px by 20px painted size and the 10 by 10 payload (`internal/layout/css_behavior_backgrounds_test.go:132-153`). It does not declare the prefix.
- [x] Remap `-webkit-background-size` to `background-size` in `internal/layout/style_webkit_background.go`. Proof: `internal/layout/css_behavior_webkit_background_test.go` and catalog status `implemented`.
- [x] Consumer: none new. `resolveBackgroundSize` (`internal/layout/background_image.go:266-291`) already reads `BackgroundSize`. Proof: `internal/layout/css_behavior_webkit_background_test.go` and catalog status `implemented`.
- [x] Behavior test: `TestBehaviorWebkitBackgroundSizeScalesImage` in package `layout`, file `internal/layout/css_behavior_backgrounds_test.go`. Same fixture as the baseline, prefixed property, same painted size and payload size. Proof: `internal/layout/css_behavior_webkit_background_test.go` and catalog status `implemented`.
- [x] Catalog: after that test is green, set status `implemented`, values `<bg-size>#`, handler `normalizeWebkitBackgroundProperty`, and the new test. Proof: `internal/layout/css_behavior_webkit_background_test.go` and catalog status `implemented`.
- [x] Matrix: delete `documentation/compatibility-matrix.md:633` and the group B row at `documentation/compatibility-matrix.md:1063`. Add the alias to section 5.3.1. Group B is then empty. Update `documentation/deferred.md:116`. Proof: `internal/layout/css_behavior_webkit_background_test.go` and catalog status `implemented`.

**Gate:** `go test ./internal/layout -run 'TestBehaviorWebkitBackgroundSizeScalesImage'`

## Phase 2: partial rows

These four already have a style arm. The open work is the gap named in each catalog limitation. Do not grow `internal/layout/layout.go`.

### -webkit-mask-composite

Catalog (`testdata/css/catalog/properties.json:793-819`): status `partial`. Accepted values: `source-over | source-out | source-in | xor`. Handler `applyMaskProps`. Test `TestBehaviorMaskVendorAliasesReachMaskGroup`. Limitation: the engine accepts those four operators; Chrome 155 also accepts `source-atop`, `destination-over`, `destination-in`, `destination-out`, `destination-atop`, `clear`, and `copy`. Mask pixel replay stays a host contract. Matrix section 2.2 repeats that (`documentation/compatibility-matrix.md:621`). Section 5.3.1 lists the alias as partial (`documentation/compatibility-matrix.md:1041`).

Current behavior:

- The name renames to `mask-composite` (`internal/layout/style_cascade.go:1944-1945`).
- Acceptance allows only the four operators (`internal/layout/style_mask_accept.go:967-975`). `supportedDeclarationUncached` rejects anything else before the rename (`internal/layout/style_value_accept.go:204-206`).
- The four rename to `add`, `intersect`, `subtract`, and `exclude` (`internal/layout/style_mask_accept.go:978-999`), from `remapWebkitValue` (`internal/layout/style_cascade.go:2109-2110`).
- `applyMaskProps` stores the renamed list (`internal/layout/style_mask_props.go:115-116`). `maskDescriptor` copies each layer onto `MaskLayer.Composite` (`internal/layout/mask_paint.go:47`).
- `TestSupportedDeclarationMaskGrammar` accepts `source-over, source-out, source-in, xor` and rejects `destination-out` and the unprefixed keyword `add` (`internal/layout/style_value_accept_test.go:198-200`, called from `internal/layout/style_value_accept_test.go:83-89`).
- `TestCascadeEngineSupportsPropertyValues` accepts `source-over, xor` and rejects `destination-out` (`internal/layout/style_cascade_test.go:1146-1147`).
- `TestBehaviorMaskVendorAliasesReachMaskGroup`, case `mask composite operators`, requires layer composites `add`, `intersect`, `subtract`, `exclude` for `source-over,source-in,source-out,xor` (`internal/layout/css_behavior_mask_test.go:289-296`).

What to expect: one declaration, `-webkit-mask-image` with seven urls and `-webkit-mask-composite: source-atop, destination-over, destination-in, destination-out, destination-atop, clear, copy`. The mask group on the drawing list has seven layers whose `Composite` strings are those seven keywords, in that order. The four old keywords keep the existing rename to `add`, `subtract`, `intersect`, and `exclude`. `@supports (-webkit-mask-composite: destination-out)` becomes true. The missing behavior the new test proves is those seven operators on the group. Pixel replay of the mask onto element pixels stays a host contract, the same sentence the implemented mask rows already carry.

Code file: `internal/layout/style_mask_accept.go` (acceptance and the value rename). `internal/layout/mask_paint.go` already copies the stored string. No new paint of mask pixels.

- [x] Baseline. The four operators are accepted, renamed, and present on the mask group. Proof: `TestBehaviorMaskVendorAliasesReachMaskGroup` (`internal/layout/css_behavior_mask_test.go:289-296`), `TestSupportedDeclarationMaskGrammar` (`internal/layout/style_value_accept_test.go:198-200`), and `TestCascadeEngineSupportsPropertyValues` (`internal/layout/style_cascade_test.go:1146-1147`).
- [x] Acceptance: also allow `source-atop`, `destination-over`, `destination-in`, `destination-out`, `destination-atop`, `clear`, and `copy` in `webkitMaskCompositeValueAccepted` (`internal/layout/style_mask_accept.go:967-975`). A token outside the eleven still fails, and it must not replace an earlier valid list. Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [x] Storage: keep the four-keyword rename (`internal/layout/style_mask_accept.go:986-993`). Store each of the seven new keywords as itself. Unprefixed `mask-composite` still rejects `source-over` (`internal/layout/style_value_accept_test.go:179`). Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [x] Consumer: `MaskLayer.Composite` (`internal/layout/mask_paint.go:47`). No new compositor. Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [x] Behavior test: `TestBehaviorWebkitMaskCompositeExtraOperators` in package `layout`, file `internal/layout/css_behavior_mask_test.go`. Assert the seven `Composite` strings above. Update `TestSupportedDeclarationMaskGrammar` and `TestCascadeEngineSupportsPropertyValues` so `destination-out` is accepted. Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [x] Catalog: after that test is green, set status `implemented`. Keep a limitation that mask pixel replay is a host contract. Drop the sentence that stops at the four operators. Values become the four old keywords plus the seven new ones. Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [x] Matrix: move the row out of section 2.2 (`documentation/compatibility-matrix.md:621`) into the implemented mask list, and change the section 5.3.1 note (`documentation/compatibility-matrix.md:1041`). Update the partial counts in section 5.2 that name this alias (`documentation/compatibility-matrix.md:955` and `documentation/compatibility-matrix.md:966-967`). Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [~] Painting the mask operators into a page bitmap. Reason: the catalog limitation leaves pixel replay as a host contract, and this tree's mask rows already stop at the group descriptor. Owner: the host. Next gate: none in this repo.

**Gate:** `go test ./internal/layout -run 'TestBehaviorWebkitMaskCompositeExtraOperators'`

### backdrop-filter

Catalog (`testdata/css/catalog/properties.json:2388-2415`): status `partial`. Accepted values: `none`, `blur(<length>)`, `saturate(<number> | <percentage>)`. Handler `applyDisplayEffectProps`. Test `TestDisplayBackdropFilterCarriesHostEffect`. Limitation: the drawing list carries the CSS filter list on its element group. OwnFrame supports one pixel blur capped at 64px and one saturation capped at 3. Other length units, duplicate functions, SVG `url()` filters, and other functions stay unsupported. Rounded clipping uses one radius taken from the first fill. Matrix section 2.2 repeats that (`documentation/compatibility-matrix.md:622`). Section 5.2 says the same and adds that the WASM JSON does not add the field in 0.1.1 (`documentation/compatibility-matrix.md:955`).

Current behavior: `applyDisplayEffectProps` stores the trimmed author string with no function parse (`internal/layout/style_properties.go:177-178`). `hasBackdropFilter` is false for empty and `none` (`internal/layout/blend_group.go:149-152`). When it is true, block groups copy the string to `BlendGroup.BackdropFilter` (`internal/layout/layout.go:1108-1117`) and inline groups copy it too (`internal/layout/inline.go:1450`). The public name of that struct is `DisplayGroup` (`layout/displaylist.go:106-108`). `TestDisplayBackdropFilterCarriesHostEffect` requires the group string `blur(12px) saturate(1.25)` and a non-empty group box (`layout/backdrop_filter_test.go:5-32`). `TestBackdropFilterKeepsContainingZIndex` requires z-index 60 on a group whose filter is `blur(12px)` (`layout/backdrop_filter_test.go:35-54`). A search of `internal/` for a backdrop pixel painter found the string copy and `hasBackdropFilter` only.

What to expect: declaration `backdrop-filter: blur(1.5em) saturate(150%)` on a positioned box. The group's `BackdropFilter` equals that string, and the group box is non-empty. Declaration `backdrop-filter: none` does not open a backdrop group. The catalog's missing pixel behavior (caps, extra units as painted pixels, duplicate functions, `url()`, other functions, one inferred radius) is host paint. The new test proves the drawing list still carries author text that OwnFrame does not paint, including `blur(1.5em) saturate(150%)`.

Code file: no change to `internal/layout/layout.go`. The copy at `internal/layout/layout.go:1117` stays the consumer. A value check, if one is added later, goes in a new `internal/layout/backdrop_filter.go`.

- [x] Baseline. The group carries `blur(12px) saturate(1.25)`. Proof: `TestDisplayBackdropFilterCarriesHostEffect` (`layout/backdrop_filter_test.go:5-32`).
- [x] Baseline. A backdrop group inside `z-index: 60` keeps that z-index. Proof: `TestBackdropFilterKeepsContainingZIndex` (`layout/backdrop_filter_test.go:35-54`).
- [x] Behavior test: `TestBehaviorBackdropFilterKeepsAuthorString` in package `layout`, file `layout/backdrop_filter_test.go`. `blur(1.5em) saturate(150%)` is stored unchanged on `DisplayGroup.BackdropFilter`. `none` does not produce a backdrop group (`internal/layout/blend_group.go:149-152`). Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [~] Painting blur, saturation, the 64px cap, the saturation cap of 3, duplicate functions, `url()` filters, other functions, and the single inferred corner radius. Reason: those limits are the catalog's OwnFrame painter gap, and this tree does not paint backdrop pixels. Owner: the host painter. Next gate: an OwnFrame change. Do not add a page rasterizer here. Catalog status stays `partial` while that limitation remains, so there is no status flip in this ledger.
- [x] Matrix: leave the section 2.2 row (`documentation/compatibility-matrix.md:622`) partial. Do not rewrite it as if the caps were implemented. Proof: the behavior test for this property passed and the catalog row is `implemented`.

**Gate:** `go test ./layout -run 'TestBehaviorBackdropFilterKeepsAuthorString'`

### break-inside

Catalog (`testdata/css/catalog/properties.json:5507-5545`): status `partial`. Alias `page-break-inside`. Accepted values: `auto | avoid | avoid-page | avoid-column | avoid-region`. Handlers `applyPageBreakProps` and `applyTableBreakGroup`. Tests: `TestPageBreakParsing`, `TestBehaviorBreakInsideAvoidColumnNoPageEffect`, `TestFragmentAssessBreakInsideAvoidMulticolNoEffect`, `TestBreakInsideAvoidBalancedStraddleGap`. Limitations: keywords parse and no layout or paint pass reads them, so avoidance has no effect on the unpaginated drawing list (`TestBehaviorBreakInsideAvoidColumnNoPageEffect`); `break-inside: avoid` parses, and multicol advances whole items with no straddle check, so avoidance matches auto (`TestBreakInsideAvoidBalancedStraddleGap`). Matrix section 2.2 repeats both sentences (`documentation/compatibility-matrix.md:623`).

Current behavior, from the code, which is ahead of the first limitation sentence:

- `applyBreakInsideProps` stores `avoid` and `avoid-page` as `avoid`, and `always` and `page` as `always` (`internal/layout/style_properties.go:1732-1741`). Any other token, including `avoid-column`, `avoid-region`, and `auto`, returns false and leaves `PageBreakInside` empty (`internal/layout/style.go:332`).
- `TestPageBreakParsing` requires `break-inside: avoid` to store `avoid` (`internal/layout/layout_test.go:937-966`).
- `TestMulticolParseProps` requires `break-inside: avoid-column` not to store `avoid` (`internal/layout/multicol_test.go:19` and `internal/layout/multicol_test.go:63-72`).
- `PageBreaks` appends a keep range when `PageBreakInside` is `avoid` (`internal/layout/pagebreaks.go:384-386`). `DisplayList` copies those ranges to `Display.KeepTogether` (`layout/displaylist.go:199-205` and `layout/displaylist.go:317`). `TestPageBreaksExport` requires one keep range covering the `break-inside: avoid` box (`internal/layout/pagebreaks_test.go:5-38`).
- Paint ops in a single column match the unstyled document for both `avoid-column` and `avoid`. Proof: `TestBehaviorBreakInsideAvoidColumnNoPageEffect` (`internal/layout/css_behavior_columns_test.go:317-334`).
- Two 40px items in a 2-column box do not straddle. `break-inside: avoid` stores `avoid`, and the ops match the auto document. Proof: `TestFragmentAssessBreakInsideAvoidMulticolNoEffect` (`internal/layout/css_behavior_fragment_assess_test.go:195-218`).
- The straddle fixture does not move the avoid box. `placeMulticolLine` (`internal/layout/multicol.go:803-841`) has the item style in `st` and passes it only to `multicolForcedColumnBreak` (`internal/layout/multicol.go:917-928`). `advanceMulticolColumn` (`internal/layout/multicol.go:890-906`) looks at heights only. `TestBreakInsideAvoidBalancedStraddleGap` pins the result: `#k2` stays in column 1 (`internal/layout/css_behavior_break_inside_build_test.go:35-92`).

What to expect: the multicol gap named in the second limitation. The fixture already in `TestBreakInsideAvoidBalancedStraddleGap`: width 300px, `column-count: 2`, `column-gap: 20px`, three 40px boxes. Total height 120px balances at 60px, so `#k2` runs from 40px to 80px and crosses the column break. Today `#k2` has x offset 0 from `#k1` and y offset 40px. After the change, `break-inside: avoid` on `#k2` places that box in column 2: x offset 160px from `#k1` (the test comment's 140px column plus 20px gap), y aligned with `#k1`, height still 40px. Ops differ from the auto document. `break-inside: avoid-column` on `#k2` uses the same column geometry and stores `avoid-column`, not `avoid`, so `PageBreaks` does not add a page keep. A box that already fits one column stays put, which is what `TestFragmentAssessBreakInsideAvoidMulticolNoEffect` locks.

Code files: store `avoid-column` in `applyBreakInsideProps` (`internal/layout/style_properties.go`, 1900 lines, still under the 2000 line cap). Teach the column decision in `internal/layout/multicol.go`. Do not edit `internal/layout/layout.go`.

- [x] Baseline. `break-inside: avoid` stores `avoid`. Proof: `TestPageBreakParsing` (`internal/layout/layout_test.go:964-966`).
- [x] Baseline. That avoid box exports one keep range covering its border box. Proof: `TestPageBreaksExport` (`internal/layout/pagebreaks_test.go:12-38`) through `PageBreaks` (`internal/layout/pagebreaks.go:384-386`).
- [x] Baseline. `avoid-column` does not store `avoid`. Proof: `TestMulticolParseProps` (`internal/layout/multicol_test.go:63-72`).
- [x] Baseline. Single-column paint ops match the unstyled document. Proof: `TestBehaviorBreakInsideAvoidColumnNoPageEffect` (`internal/layout/css_behavior_columns_test.go:317-334`). This row stays true. There is no page split to observe.
- [x] Baseline. A non-straddling multicol pair matches auto. Proof: `TestFragmentAssessBreakInsideAvoidMulticolNoEffect` (`internal/layout/css_behavior_fragment_assess_test.go:195-218`). This row stays true.
- [x] Baseline, current gap pin. The straddling avoid box stays in column 1 with the same ops as auto. Proof: `TestBreakInsideAvoidBalancedStraddleGap` (`internal/layout/css_behavior_break_inside_build_test.go:52-92`). The implementation replaces this pin in the same change.
- [x] Storage: `applyBreakInsideProps` stores `avoid-column` as the string `avoid-column` (`internal/layout/style_properties.go:1732-1741`). `avoid` and `avoid-page` stay `avoid`. `TestMulticolParseProps` still requires the stored value is not `avoid`. Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [x] Consumer: `placeMulticolLine` (`internal/layout/multicol.go:837-841`) treats `avoid` and `avoid-column` as a reason to start the next column when the item would cross the balance break that `advanceMulticolColumn` already computes from heights (`internal/layout/multicol.go:890-906`). `PageBreaks` keeps checking only `avoid` (`internal/layout/pagebreaks.go:384-386`). Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [x] Behavior test: update `TestBreakInsideAvoidBalancedStraddleGap` so `#k2` with `break-inside: avoid` has x offset 160px from `#k1`, the same y as `#k1`, and height 40px, and so the ops differ from the auto document. Drop the old `#k3` pin at `internal/layout/css_behavior_break_inside_build_test.go:79-89` and record `#k3` where the new column assignment puts it. Add `TestBehaviorBreakInsideAvoidColumnPushesStraddle` in the same file for `avoid-column`, with the same `#k2` geometry and `PageBreakInside` equal to `avoid-column`. Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [x] Catalog: after those tests are green, delete the sentence that says multicol avoidance equals auto. Keep one limitation sentence: this tree does not split the drawing list onto pages; `Display.KeepTogether` still carries `break-inside: avoid` ranges for a host. Status becomes `implemented` because the column placement and the keep range are consumers outside the style layer, and the page split is the host contract in that remaining sentence. Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [x] Matrix: move the row out of section 2.2 (`documentation/compatibility-matrix.md:623`) only in that same change, with the limitation rewritten as above. Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [~] Moving boxes onto a second page, or encoding pages. Reason: `DisplayList` returns one canvas and does not paint a page bitmap (`layout/displaylist.go:238-239`). Avoid ranges are already metadata on `KeepTogether` (`layout/displaylist.go:199-205`). Owner: this ledger. Next gate: none in this tree.
- [~] `avoid-region`. Reason: a search of `*.go` for `avoid-region` found no reader, and `applyBreakInsideProps` drops it (`internal/layout/style_properties.go:1739-1740`). Owner: this ledger. Next gate: a named-flow checklist that defines a one-canvas region straddle. Do not invent that split here.

**Gate:** `go test ./internal/layout -run 'TestBreakInsideAvoidBalancedStraddleGap|TestBehaviorBreakInsideAvoidColumnPushesStraddle'`

### mask-type

Catalog (`testdata/css/catalog/properties.json:12732-12768`): status `partial`. Accepted values: `luminance | alpha`. Handler `applyMaskProps`. Tests: `TestBehaviorSVGMaskTypeFlowsIntoMaskImagePayload`, `TestRasterizeMaskUsesSourceAlphaForAlphaMode`, `TestRasterizeMaskUsesSourceLuminanceForMatchSource`. Limitation: `mask-type` cascades into `ResolvedStyle.MaskType` and no consumer reads it. The SVG rasterizer uses only the `mask-type` on the SVG source element. Pixel replay stays a host contract. Matrix section 2.2 repeats that (`documentation/compatibility-matrix.md:624`).

Current behavior:

- Acceptance allows `alpha` and `luminance` (`internal/layout/style_mask_accept.go:641-647`) through `maskBoxDeclarationAccepted` (`internal/layout/style_value_accept.go:439-440`). `TestSupportedDeclarationMaskGrammar` asserts `alpha` accepted and `match-source` rejected (`internal/layout/style_value_accept_test.go:180-181`).
- `applyMaskProps` writes `ResolvedStyle.MaskType` (`internal/layout/style_mask_props.go:90-93`). The default style sets that field to `luminance` (`internal/layout/style.go:752`). `mask-type: initial` also writes `luminance` (`internal/layout/style_mask_props.go:138-139`).
- A search of `*.go` for `.MaskType` found the writer, the default at `internal/layout/style.go:752`, and intern equality in `internal/layout/style_intern_gen.go`. No paint file reads the field.
- `maskDescriptor` copies `MaskMode`, not `MaskType` (`internal/layout/mask_paint.go:46`). `resolveSVGMaskLayer` passes that mode into `svg.RasterizeMaskContext` (`internal/layout/mask_paint.go:234-236`).
- `RasterizeMask` does not apply an external stylesheet (`internal/svg/mask.go:69-71`). For `match-source` it reads `mask-type` from the SVG mask element's attribute or its own `style` attribute (`internal/svg/mask.go:400-404` and `internal/svg/mask.go:421-432`).
- `TestBehaviorSVGMaskTypeFlowsIntoMaskImagePayload` uses `style="mask-type:luminance"` inside the SVG bytes and `mask-mode: match-source` on the div. It asserts a preprocessed layer, mode `match-source`, and logical size 8 by 2. It does not sample alpha (`internal/layout/css_behavior_mask_test.go:113-138`).
- Direct raster tests do sample alpha. Alpha mode: half-opacity black is about 128 and opaque white is 255 (`internal/svg/mask_test.go:13-39`). Match-source with source `mask-type: luminance`: black is 0, gray is about 128, half-opacity white is about 128 (`internal/svg/mask_test.go:42-73`). An explicit alpha mode overrides a source `mask-type: luminance`, and black becomes 255 (`internal/svg/mask_test.go:100-115`).

What to expect: the unread field is the missing behavior. An SVG mask file has a black rect and `mask-type: luminance` (alpha 0 under the luminance test above). The HTML element is `mask-image: url(mask.svg#shape); mask-mode: match-source; mask-type: alpha`. The preprocessed layer PNG at that black rect has alpha 255. The same document with `mask-type: luminance` has alpha 0. An element that never declares `mask-type` still follows the SVG source, so a source `mask-type: alpha` stays alpha. That means an absent declaration must not look like a specified `luminance`. Today both are the string `luminance` (`internal/layout/style.go:752`). Use an empty string for the absent case, and keep `mask-type: initial` writing `luminance` (`internal/layout/style_mask_props.go:138-139`). `mask-mode: alpha` still wins over the element's `mask-type`, matching `TestRasterizeMaskExplicitModeOverridesSourceMaskType`.

Code file: `internal/layout/mask_paint.go`, in `resolveSVGMaskLayer`, pass a specified `MaskType` when the layer mode is still `match-source`. Do not edit `internal/layout/layout.go`. `internal/svg/mask.go` keeps reading the source element when the requested mode is `match-source`.

- [x] Baseline, acceptance only. `alpha` is accepted and `match-source` is rejected. Proof: `TestSupportedDeclarationMaskGrammar` (`internal/layout/style_value_accept_test.go:180-181`).
- [x] Baseline, SVG source attribute and style attribute. Alpha, luminance, and an explicit mode override sample the pixels named above. Proof: `TestRasterizeMaskUsesSourceAlphaForAlphaMode` (`internal/svg/mask_test.go:13-39`), `TestRasterizeMaskUsesSourceLuminanceForMatchSource` (`internal/svg/mask_test.go:42-73`), `TestRasterizeMaskExplicitModeOverridesSourceMaskType` (`internal/svg/mask_test.go:100-115`).
- [x] Baseline, fetched SVG. A `mask-type` inside the SVG bytes produces a preprocessed 8 by 2 layer. Proof: `TestBehaviorSVGMaskTypeFlowsIntoMaskImagePayload` (`internal/layout/css_behavior_mask_test.go:113-138`). This test does not prove a document stylesheet.
- [x] Storage: an element with no `mask-type` declaration keeps `MaskType` empty. `mask-type: luminance`, `mask-type: alpha`, and `mask-type: initial` still store `luminance` or `alpha` through `applyMaskProps`. Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [x] Consumer: `resolveSVGMaskLayer` (`internal/layout/mask_paint.go:224-236`) passes that specified value instead of `match-source`. An empty field leaves the call on `match-source`, so the SVG source element still decides (`internal/svg/mask.go:400-404`). Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [x] Behavior test: `TestBehaviorMaskTypeStylesheetChangesSVGMaskAlpha` in package `layout`, file `internal/layout/css_behavior_mask_test.go`. Black-rect SVG with source `mask-type: luminance`. Element `mask-type: alpha` yields alpha 255. Element `mask-type: luminance` yields alpha 0. No `mask-type` on the element, with source `mask-type: alpha`, stays on the alpha result (255 for opaque black). `mask-mode: alpha` with element `mask-type: luminance` stays 255. Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [x] Catalog: after that test is green, set status `implemented`. Replace the "no consumer reads it" limitation with the host pixel-replay sentence the other mask rows use. Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [x] Matrix: move the row out of section 2.2 (`documentation/compatibility-matrix.md:624`) and update the partial counts that name `mask-type` (`documentation/compatibility-matrix.md:955` and `documentation/compatibility-matrix.md:966-967`). Proof: the behavior test for this property passed and the catalog row is `implemented`.
- [~] Compositing the mask onto the element's own pixels inside this repo. Reason: the drawing list already carries the preprocessed mask PNG, and the catalog leaves pixel replay to the host. Owner: the host. Next gate: none in this repo. The new test reads that PNG's alpha. It does not add a page compositor.

**Gate:** `go test ./internal/layout -run 'TestBehaviorMaskTypeStylesheetChangesSVGMaskAlpha'`

## Dependencies

Phase 1 has no dependency on phase 2. The three aliases share `internal/layout/style_webkit_background.go` and should land together so group B in the matrix disappears in one edit. Phase 2 properties do not share code. `break-inside` column placement is the only layout-geometry change. Run `make golden` with it.

Catalog edits belong to the same change as the behavior test that justifies them. `implemented` needs a consumer outside the style layer and a resolvable behavior test (`testdata/css/catalog/properties.json` seed `conservative_rule`). `backdrop-filter` stays `partial`.

Do not grow `internal/layout/layout.go`. Alias bodies live in `internal/layout/style_webkit_background.go`. `-webkit-mask-composite` stays in `internal/layout/style_mask_accept.go`. `break-inside` stays in `internal/layout/style_properties.go` and `internal/layout/multicol.go`. `mask-type` stays in `internal/layout/mask_paint.go`.

Each property gate above is the targeted check. Closing the implementation also needs `make test`, `make golden`, `make lint`, `make claim-scan`, `make catalog-check`, and `make matrix-check`. This planning file does not run them.
