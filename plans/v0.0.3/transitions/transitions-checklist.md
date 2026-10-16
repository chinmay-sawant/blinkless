# v0.0.3 - Transitions

> **Parent:** `plans/v0.0.3/README.md` - index of these ledgers.
> **Status:** not started. Baseline rows record what the code already does. Implementation rows are open.
> **Estimated effort:** 11 CSS property names (6 unprefixed, 5 `-webkit-transition*` aliases) across 8 phases.

---

## Overview

A transition describes how a property changes from an old value to a new one. Blinkless does not keep the old frame and does not move values as time passes. This ledger stores the timing and the property names on the drawing list. A host that has two lists, before and after, samples the change itself.

The result is still `layout.Display` from `layout.DisplayList` (`layout/displaylist.go:129-138`). There is no page PNG, no PDF, no JavaScript, and no clock.

`documentation/deferred.md:110` calls animation, transition, and view-transition properties a permanent print non-goal (45 names). The five `-webkit-transition*` aliases are in vendor group D on `documentation/deferred.md:120`. This file is the planning home for the 11 transition names. The animation ledger owns animation and view-transition names. The scroll ledger owns `pointer-timeline*`. Do not edit `deferred.md` in the planning session. Phase 5 points the transition part of those rows here.

## Executive summary

| Property | Catalog line | Status | Pinned syntax |
|---|---:|---|---|
| `transition-property` | 16869 | unsupported | `none | <single-transition-property>#` |
| `transition-duration` | 16853 | unsupported | `<time [0s,inf]>#` |
| `transition-delay` | 16837 | unsupported | `<time>#` |
| `transition-timing-function` | 16885 | unsupported | `<easing-function>#` |
| `transition-behavior` | 16821 | unsupported | `<transition-behavior-value>#` |
| `transition` | 16805 | unsupported | `<single-transition>#` |
| `-webkit-transition` | 677 | unsupported | `(none on the vendor row)` |
| `-webkit-transition-delay` | 692 | unsupported | `(none on the vendor row)` |
| `-webkit-transition-duration` | 707 | unsupported | `(none on the vendor row)` |
| `-webkit-transition-property` | 722 | unsupported | `(none on the vendor row)` |
| `-webkit-transition-timing-function` | 737 | unsupported | `(none on the vendor row)` |

All 11 rows were read from `testdata/css/catalog/properties.json` on 2026-10-15 (`updated` is `2026-10-13`, `properties.json:4`). Each is `unsupported`, with empty `handlers` and empty `tests`, and the limitation "No apply handler in internal/layout; declaration is ignored". `inf` in the table stands in for the infinity sign in the catalog. A search of `*.go` found no `transition-property` string. Header totals: 785 properties, 389 implemented, 19 partial, 369 unsupported, 8 intentionally ignored (`properties.json:11-18`).

## Phase 0: lock the contract and the baseline

### 0.1 What the code does today


- [x] The public pipeline is `html.Parse` (`html/parse.go:36`), `css.Apply` (`css/css.go:98`), and `layout.DisplayList` (`layout/displaylist.go:137`). `css.Options` is `WidthPx`, `HeightPx`, `Media`, `Extra`, `Focus`, `Hover`, and `Active` (`css/css.go:69-78`). `Media` is screen or print. Anything else, including `speech`, returns `ErrBadMedia` (`css/css.go:147-155`, `css/css_test.go:58-60`).
- [x] `layout.Box` is id, tag, action, text, and the border box in CSS pixels (`layout/layout.go:10-21`). `PlacedElement` is the same fields in points (`internal/layout/placed.go:7-16`). `boxesFrom` copies them (`layout/layout.go:35-48`). WASM `drawBoxJSON` copies the same six (`bindings/wasm/drawing_list.go:194-203`, loop at `296-309`). Schema is `blinkless.drawinglist/1` (`bindings/wasm/drawing_list.go:20-23`).
- [x] A name with no apply arm still passes `supportedDeclaration` (`internal/layout/style_value_accept.go:135-138`, unlisted default described at `607-608`). The cascade keeps it (`internal/layout/style_cascade.go:693-695`). `applyStyleProp` then calls `applyIgnoredGroup`, which drops the name and the value (`internal/layout/style_cascade.go:1879-1885`, `internal/layout/style_properties.go:1854-1859`).
- [x] `@supports` is true only when a `styleGroups` function returns true (`internal/layout/style_cascade.go:1888-1920`). The last group is `applyScrollProps` (`internal/layout/style_cascade.go:1676`).
- [x] `internal/layout/layout.go` is 2142 lines (`scripts/file-size-allowlist.txt:9`). New code does not go in that file. `style_cascade.go` is 1985 lines and `style_value_accept.go` is 1753 lines. Add a call, not a new switch body, in each.
- [x] GitHub CI runs only `make test` and `make lint` (`.github/workflows/ci.yml:19-26`). `make catalog-check` is `python3 scripts/css-catalog-map.py --check` (`Makefile:121-122`). `make matrix-check` is `bash scripts/check-matrix-sync.sh` (`Makefile:128-129`).

- [x] `-webkit-transition*` is not rewritten. `normalizeOtherWebkitProperty` (`internal/layout/style_cascade.go:1717-1742`) has no transition case.
- [x] There is no transition field on `layout.Box` (`layout/layout.go:12-21`).

### 0.2 Contract to implement

- [ ] Store one transition record per comma-separated index, after list alignment, so property, duration, delay, timing function, and behavior for index i stay together.
- [ ] `transition-property` stores names. It does not copy every used value on the element. The destination look is already on the paint ops when this engine paints that property. The host diffs two display lists for the start value. This ledger does not invent a from-value.
- [ ] Do not add a ticker. Phase 7 is the non-goal. The animation ledger's phase 7 is the same non-goal. One host clock, not two.
- [ ] New code lives in `internal/layout/transition_style.go` and `internal/layout/transition_payload.go`. Easing lives in `internal/layout/easing_value.go` so the animation ledger does not grow a second parser. Register `applyTransitionProps` as one entry in `styleGroups` (`internal/layout/style_cascade.go:1642`). Do not add lines to `internal/layout/layout.go`.
- [ ] Public data is `layout.Box.Transitions`. Copy it in `boxesFrom` (`layout/layout.go:35`). WASM adds an omitempty `transitions` field on `drawBoxJSON` (`bindings/wasm/drawing_list.go:194`). Schema stays `blinkless.drawinglist/1` (`drawing_list.go:20-23`).
- [ ] A name is done only when the public box shows the value and a test fails if it is removed. Catalog status is not that proof (`properties.json:9`).

**Gate:** no code lands before the record shape and the no-ticker rule are the ones above.

## Phase 1: style fields and interning

Add fields on `ResolvedStyle` (`internal/layout/style.go`). Parse in `internal/layout/transition_style.go`. `TestStyleInternGeneratedFieldsComplete` guards the generated intern list (`internal/layout/style_intern_gen_test.go:14-27`). Shorthands and aliases do not get their own fields.

### Longhands

- [ ] `transition-property` (`properties.json:16869`, syntax `none | <single-transition-property>#`). Store the used value on `ResolvedStyle` in `internal/layout/style.go`. Parse it in `internal/layout/transition_style.go`. Store `none` or the property names. If the pinned production includes `all`, store the keyword `all`. Do not expand `all` into every property this engine knows. The public-box test uses `opacity` and fails if that name is missing. Do not require opacity paint to change. Regenerate interning from `internal/layout/style.go:2`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `transition-duration` (`properties.json:16853`, syntax `<time [0s,inf]>#`). Store the used value on `ResolvedStyle` in `internal/layout/style.go`. Parse it in `internal/layout/transition_style.go`. Reject a negative time. Store milliseconds. The public-box test uses `250ms` and fails if the field is not 250. Regenerate interning from `internal/layout/style.go:2`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `transition-delay` (`properties.json:16837`, syntax `<time>#`). Store the used value on `ResolvedStyle` in `internal/layout/style.go`. Parse it in `internal/layout/transition_style.go`. `<time>` may be negative. The public-box test uses `-100ms` and fails if the field is missing or forced to 0. Regenerate interning from `internal/layout/style.go:2`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `transition-timing-function` (`properties.json:16885`, syntax `<easing-function>#`). Store the used value on `ResolvedStyle` in `internal/layout/style.go`. Parse it in `internal/layout/transition_style.go`. Parse easing in `internal/layout/easing_value.go`, shared with the animation ledger. Read the pinned production before accepting a keyword. The test uses one non-initial keyword from that production. Regenerate interning from `internal/layout/style.go:2`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `transition-behavior` (`properties.json:16821`, syntax `<transition-behavior-value>#`). Store the used value on `ResolvedStyle` in `internal/layout/style.go`. Parse it in `internal/layout/transition_style.go`. Read the pinned production before accepting a keyword. Store it. Do not start interpolating discrete properties. The test uses one non-initial keyword from that production. Regenerate interning from `internal/layout/style.go:2`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
### Shorthand

- [ ] `transition` (`properties.json:16805`, syntax `<single-transition>#`). No field of its own. The expander writes the longhands. Read `<single-transition>` from the upstream row's spec before coding. Reset only the longhands that production resets. Check whether `transition-behavior` is one of them. Do not guess. A later longhand overrides one component. The test uses a shorthand that sets property and duration together.
### Prefixed aliases

- [ ] `-webkit-transition` (`properties.json:677`) has no `ResolvedStyle` field. It rewrites to `transition` before apply.
- [ ] `-webkit-transition-delay` (`properties.json:692`) has no `ResolvedStyle` field. It rewrites to `transition-delay` before apply.
- [ ] `-webkit-transition-duration` (`properties.json:707`) has no `ResolvedStyle` field. It rewrites to `transition-duration` before apply.
- [ ] `-webkit-transition-property` (`properties.json:722`) has no `ResolvedStyle` field. It rewrites to `transition-property` before apply.
- [ ] `-webkit-transition-timing-function` (`properties.json:737`) has no `ResolvedStyle` field. It rewrites to `transition-timing-function` before apply.


## Phase 2: value acceptance and @supports

One acceptance function in `internal/layout/transition_style.go`, called from `declarationValueAccepted` (`internal/layout/style_value_accept.go:139`). CSS-wide keywords stay accepted (`style_value_accept.go:102-104`). A list is accepted only when every item is accepted.

Alias rewrites go in `normalizeWebkitTransitionProperty`, called from `normalizeOtherWebkitProperty` (`internal/layout/style_cascade.go:1717`).

### Longhands

- [ ] `transition-property` (`properties.json:16869`). Accept `none | <single-transition-property>#` in the new acceptance function. `engineSupportsProperty` (`internal/layout/style_cascade.go:1895`) returns the same answer. A bad token does not replace an earlier good one (`style_cascade.go:693-695`). Store `none` or the property names. If the pinned production includes `all`, store the keyword `all`. Do not expand `all` into every property this engine knows. The public-box test uses `opacity` and fails if that name is missing. Do not require opacity paint to change. Proof: `go test ./internal/layout -run TestTransitionAccept -count=1`.
- [ ] `transition-duration` (`properties.json:16853`). Accept `<time [0s,inf]>#` in the new acceptance function. `engineSupportsProperty` (`internal/layout/style_cascade.go:1895`) returns the same answer. A bad token does not replace an earlier good one (`style_cascade.go:693-695`). Reject a negative time. Store milliseconds. The public-box test uses `250ms` and fails if the field is not 250. Proof: `go test ./internal/layout -run TestTransitionAccept -count=1`.
- [ ] `transition-delay` (`properties.json:16837`). Accept `<time>#` in the new acceptance function. `engineSupportsProperty` (`internal/layout/style_cascade.go:1895`) returns the same answer. A bad token does not replace an earlier good one (`style_cascade.go:693-695`). `<time>` may be negative. The public-box test uses `-100ms` and fails if the field is missing or forced to 0. Proof: `go test ./internal/layout -run TestTransitionAccept -count=1`.
- [ ] `transition-timing-function` (`properties.json:16885`). Accept `<easing-function>#` in the new acceptance function. `engineSupportsProperty` (`internal/layout/style_cascade.go:1895`) returns the same answer. A bad token does not replace an earlier good one (`style_cascade.go:693-695`). Parse easing in `internal/layout/easing_value.go`, shared with the animation ledger. Read the pinned production before accepting a keyword. The test uses one non-initial keyword from that production. Proof: `go test ./internal/layout -run TestTransitionAccept -count=1`.
- [ ] `transition-behavior` (`properties.json:16821`). Accept `<transition-behavior-value>#` in the new acceptance function. `engineSupportsProperty` (`internal/layout/style_cascade.go:1895`) returns the same answer. A bad token does not replace an earlier good one (`style_cascade.go:693-695`). Read the pinned production before accepting a keyword. Store it. Do not start interpolating discrete properties. The test uses one non-initial keyword from that production. Proof: `go test ./internal/layout -run TestTransitionAccept -count=1`.
### Shorthand

- [ ] `transition` (`properties.json:16805`). Accept `<single-transition>#` in the new acceptance function. `engineSupportsProperty` (`internal/layout/style_cascade.go:1895`) returns the same answer. A bad token does not replace an earlier good one (`style_cascade.go:693-695`). Read `<single-transition>` from the upstream row's spec before coding. Reset only the longhands that production resets. Check whether `transition-behavior` is one of them. Do not guess. A later longhand overrides one component. The test uses a shorthand that sets property and duration together. Proof: `go test ./internal/layout -run TestTransitionAccept -count=1`.
- [ ] **Expansion.** `transition` writes the longhands named in its note. CSS-wide keywords reset those longhands. Proof: `go test ./internal/layout -run TestTransitionExpand -count=1`.
### Prefixed aliases

- [ ] `-webkit-transition` (`properties.json:677`) rewrites to `transition` in a helper called from `normalizeOtherWebkitProperty` (`internal/layout/style_cascade.go:1717`). The vendor row has no `upstream_syntax`. Accept the same values as `transition`. `@supports` is true for the alias only when it is true for the same value of `transition`. Both source orders are tested. The later declaration wins (`style_cascade.go:691-698`). Proof: `go test ./internal/layout -run TestWebkitTransitionAlias -count=1`.
- [ ] `-webkit-transition-delay` (`properties.json:692`) rewrites to `transition-delay` in a helper called from `normalizeOtherWebkitProperty` (`internal/layout/style_cascade.go:1717`). The vendor row has no `upstream_syntax`. Accept the same values as `transition-delay`. `@supports` is true for the alias only when it is true for the same value of `transition-delay`. Both source orders are tested. The later declaration wins (`style_cascade.go:691-698`). Proof: `go test ./internal/layout -run TestWebkitTransitionAlias -count=1`.
- [ ] `-webkit-transition-duration` (`properties.json:707`) rewrites to `transition-duration` in a helper called from `normalizeOtherWebkitProperty` (`internal/layout/style_cascade.go:1717`). The vendor row has no `upstream_syntax`. Accept the same values as `transition-duration`. `@supports` is true for the alias only when it is true for the same value of `transition-duration`. Both source orders are tested. The later declaration wins (`style_cascade.go:691-698`). Proof: `go test ./internal/layout -run TestWebkitTransitionAlias -count=1`.
- [ ] `-webkit-transition-property` (`properties.json:722`) rewrites to `transition-property` in a helper called from `normalizeOtherWebkitProperty` (`internal/layout/style_cascade.go:1717`). The vendor row has no `upstream_syntax`. Accept the same values as `transition-property`. `@supports` is true for the alias only when it is true for the same value of `transition-property`. Both source orders are tested. The later declaration wins (`style_cascade.go:691-698`). Proof: `go test ./internal/layout -run TestWebkitTransitionAlias -count=1`.
- [ ] `-webkit-transition-timing-function` (`properties.json:737`) rewrites to `transition-timing-function` in a helper called from `normalizeOtherWebkitProperty` (`internal/layout/style_cascade.go:1717`). The vendor row has no `upstream_syntax`. Accept the same values as `transition-timing-function`. `@supports` is true for the alias only when it is true for the same value of `transition-timing-function`. Both source orders are tested. The later declaration wins (`style_cascade.go:691-698`). Proof: `go test ./internal/layout -run TestWebkitTransitionAlias -count=1`.


## Phase 3: drawing-list payload and WASM JSON

`internal/layout/transition_payload.go` copies the record through `PlacedElement` (`internal/layout/placed.go:7`) and `boxesFrom` (`layout/layout.go:35`). Times are milliseconds. Do not edit `internal/layout/layout.go`.

- [ ] `transition-property` (`properties.json:16869`) is copied by `internal/layout/transition_payload.go` onto `layout.Box.Transitions` and then onto `drawBoxJSON` (`bindings/wasm/drawing_list.go:194`) with `omitempty`. Store `none` or the property names. If the pinned production includes `all`, store the keyword `all`. Do not expand `all` into every property this engine knows. The public-box test uses `opacity` and fails if that name is missing. Do not require opacity paint to change. Proof: `go test ./bindings/wasm -run TestDrawingListTransition -count=1`.
- [ ] `transition-duration` (`properties.json:16853`) is copied by `internal/layout/transition_payload.go` onto `layout.Box.Transitions` and then onto `drawBoxJSON` (`bindings/wasm/drawing_list.go:194`) with `omitempty`. Reject a negative time. Store milliseconds. The public-box test uses `250ms` and fails if the field is not 250. Proof: `go test ./bindings/wasm -run TestDrawingListTransition -count=1`.
- [ ] `transition-delay` (`properties.json:16837`) is copied by `internal/layout/transition_payload.go` onto `layout.Box.Transitions` and then onto `drawBoxJSON` (`bindings/wasm/drawing_list.go:194`) with `omitempty`. `<time>` may be negative. The public-box test uses `-100ms` and fails if the field is missing or forced to 0. Proof: `go test ./bindings/wasm -run TestDrawingListTransition -count=1`.
- [ ] `transition-timing-function` (`properties.json:16885`) is copied by `internal/layout/transition_payload.go` onto `layout.Box.Transitions` and then onto `drawBoxJSON` (`bindings/wasm/drawing_list.go:194`) with `omitempty`. Parse easing in `internal/layout/easing_value.go`, shared with the animation ledger. Read the pinned production before accepting a keyword. The test uses one non-initial keyword from that production. Proof: `go test ./bindings/wasm -run TestDrawingListTransition -count=1`.
- [ ] `transition-behavior` (`properties.json:16821`) is copied by `internal/layout/transition_payload.go` onto `layout.Box.Transitions` and then onto `drawBoxJSON` (`bindings/wasm/drawing_list.go:194`) with `omitempty`. Read the pinned production before accepting a keyword. Store it. Do not start interpolating discrete properties. The test uses one non-initial keyword from that production. Proof: `go test ./bindings/wasm -run TestDrawingListTransition -count=1`.
- [ ] `transition` (`properties.json:16805`) has no JSON key. The host reads the longhand fields. Read `<single-transition>` from the upstream row's spec before coding. Reset only the longhands that production resets. Check whether `transition-behavior` is one of them. Do not guess. A later longhand overrides one component. The test uses a shorthand that sets property and duration together. Proof: `go test ./layout -run TestDisplayTransition -count=1`.
- [ ] `-webkit-transition` (`properties.json:677`) has no JSON key. The host reads the `transition` value on `layout.Box.Transitions`. Proof: `go test ./bindings/wasm -run TestDrawingListTransition -count=1`.
- [ ] `-webkit-transition-delay` (`properties.json:692`) has no JSON key. The host reads the `transition-delay` value on `layout.Box.Transitions`. Proof: `go test ./bindings/wasm -run TestDrawingListTransition -count=1`.
- [ ] `-webkit-transition-duration` (`properties.json:707`) has no JSON key. The host reads the `transition-duration` value on `layout.Box.Transitions`. Proof: `go test ./bindings/wasm -run TestDrawingListTransition -count=1`.
- [ ] `-webkit-transition-property` (`properties.json:722`) has no JSON key. The host reads the `transition-property` value on `layout.Box.Transitions`. Proof: `go test ./bindings/wasm -run TestDrawingListTransition -count=1`.
- [ ] `-webkit-transition-timing-function` (`properties.json:737`) has no JSON key. The host reads the `transition-timing-function` value on `layout.Box.Transitions`. Proof: `go test ./bindings/wasm -run TestDrawingListTransition -count=1`.

## Phase 4: behavior tests

The behavior test that closes a row builds `layout.Display` with `layout.DisplayList` (`layout/displaylist.go:137`, called from tests at `layout/displaylist_test.go:34`) and reads `layout.Box.Transitions`. `layoutHTML` (`internal/layout/layout_test.go:41`) reads the internal result and does not close the row.

- [ ] `transition-property` (`properties.json:16869`). Fail if this value is dropped before `layout.Box.Transitions`. Store `none` or the property names. If the pinned production includes `all`, store the keyword `all`. Do not expand `all` into every property this engine knows. The public-box test uses `opacity` and fails if that name is missing. Do not require opacity paint to change. Proof: `go test ./layout -run TestDisplayTransition -count=1`.
- [ ] `transition-duration` (`properties.json:16853`). Fail if this value is dropped before `layout.Box.Transitions`. Reject a negative time. Store milliseconds. The public-box test uses `250ms` and fails if the field is not 250. Proof: `go test ./layout -run TestDisplayTransition -count=1`.
- [ ] `transition-delay` (`properties.json:16837`). Fail if this value is dropped before `layout.Box.Transitions`. `<time>` may be negative. The public-box test uses `-100ms` and fails if the field is missing or forced to 0. Proof: `go test ./layout -run TestDisplayTransition -count=1`.
- [ ] `transition-timing-function` (`properties.json:16885`). Fail if this value is dropped before `layout.Box.Transitions`. Parse easing in `internal/layout/easing_value.go`, shared with the animation ledger. Read the pinned production before accepting a keyword. The test uses one non-initial keyword from that production. Proof: `go test ./layout -run TestDisplayTransition -count=1`.
- [ ] `transition-behavior` (`properties.json:16821`). Fail if this value is dropped before `layout.Box.Transitions`. Read the pinned production before accepting a keyword. Store it. Do not start interpolating discrete properties. The test uses one non-initial keyword from that production. Proof: `go test ./layout -run TestDisplayTransition -count=1`.
- [ ] `transition` (`properties.json:16805`). Fail if this value is dropped before `layout.Box.Transitions`. Read `<single-transition>` from the upstream row's spec before coding. Reset only the longhands that production resets. Check whether `transition-behavior` is one of them. Do not guess. A later longhand overrides one component. The test uses a shorthand that sets property and duration together. Proof: `go test ./layout -run TestDisplayTransition -count=1`.
- [ ] `-webkit-transition` (`properties.json:677`). Declare the alias with a non-initial value of `transition` and fail if `layout.Box.Transitions` lacks it. Proof: `go test ./layout -run TestDisplayTransition -count=1`.
- [ ] `-webkit-transition-delay` (`properties.json:692`). Declare the alias with a non-initial value of `transition-delay` and fail if `layout.Box.Transitions` lacks it. Proof: `go test ./layout -run TestDisplayTransition -count=1`.
- [ ] `-webkit-transition-duration` (`properties.json:707`). Declare the alias with a non-initial value of `transition-duration` and fail if `layout.Box.Transitions` lacks it. Proof: `go test ./layout -run TestDisplayTransition -count=1`.
- [ ] `-webkit-transition-property` (`properties.json:722`). Declare the alias with a non-initial value of `transition-property` and fail if `layout.Box.Transitions` lacks it. Proof: `go test ./layout -run TestDisplayTransition -count=1`.
- [ ] `-webkit-transition-timing-function` (`properties.json:737`). Declare the alias with a non-initial value of `transition-timing-function` and fail if `layout.Box.Transitions` lacks it. Proof: `go test ./layout -run TestDisplayTransition -count=1`.

## Phase 5: catalog, matrix, and the deferred pointer

- [ ] `transition-property` (`properties.json:16869`). Point the catalog row at `applyTransitionProps` and the public-box test. Aliases may list the normalizer once `scripts/css-catalog-map.py` records the vendor pair (`scripts/css-catalog-map.py:379-400`). Status follows the test, using the rule at `properties.json:9`. Proof: `make catalog-check`.
- [ ] `transition-duration` (`properties.json:16853`). Point the catalog row at `applyTransitionProps` and the public-box test. Aliases may list the normalizer once `scripts/css-catalog-map.py` records the vendor pair (`scripts/css-catalog-map.py:379-400`). Status follows the test, using the rule at `properties.json:9`. Proof: `make catalog-check`.
- [ ] `transition-delay` (`properties.json:16837`). Point the catalog row at `applyTransitionProps` and the public-box test. Aliases may list the normalizer once `scripts/css-catalog-map.py` records the vendor pair (`scripts/css-catalog-map.py:379-400`). Status follows the test, using the rule at `properties.json:9`. Proof: `make catalog-check`.
- [ ] `transition-timing-function` (`properties.json:16885`). Point the catalog row at `applyTransitionProps` and the public-box test. Aliases may list the normalizer once `scripts/css-catalog-map.py` records the vendor pair (`scripts/css-catalog-map.py:379-400`). Status follows the test, using the rule at `properties.json:9`. Proof: `make catalog-check`.
- [ ] `transition-behavior` (`properties.json:16821`). Point the catalog row at `applyTransitionProps` and the public-box test. Aliases may list the normalizer once `scripts/css-catalog-map.py` records the vendor pair (`scripts/css-catalog-map.py:379-400`). Status follows the test, using the rule at `properties.json:9`. Proof: `make catalog-check`.
- [ ] `transition` (`properties.json:16805`). Point the catalog row at `applyTransitionProps` and the public-box test. Aliases may list the normalizer once `scripts/css-catalog-map.py` records the vendor pair (`scripts/css-catalog-map.py:379-400`). Status follows the test, using the rule at `properties.json:9`. Proof: `make catalog-check`.
- [ ] `-webkit-transition` (`properties.json:677`). Point the catalog row at `applyTransitionProps` and the public-box test. Aliases may list the normalizer once `scripts/css-catalog-map.py` records the vendor pair (`scripts/css-catalog-map.py:379-400`). Status follows the test, using the rule at `properties.json:9`. Proof: `make catalog-check`.
- [ ] `-webkit-transition-delay` (`properties.json:692`). Point the catalog row at `applyTransitionProps` and the public-box test. Aliases may list the normalizer once `scripts/css-catalog-map.py` records the vendor pair (`scripts/css-catalog-map.py:379-400`). Status follows the test, using the rule at `properties.json:9`. Proof: `make catalog-check`.
- [ ] `-webkit-transition-duration` (`properties.json:707`). Point the catalog row at `applyTransitionProps` and the public-box test. Aliases may list the normalizer once `scripts/css-catalog-map.py` records the vendor pair (`scripts/css-catalog-map.py:379-400`). Status follows the test, using the rule at `properties.json:9`. Proof: `make catalog-check`.
- [ ] `-webkit-transition-property` (`properties.json:722`). Point the catalog row at `applyTransitionProps` and the public-box test. Aliases may list the normalizer once `scripts/css-catalog-map.py` records the vendor pair (`scripts/css-catalog-map.py:379-400`). Status follows the test, using the rule at `properties.json:9`. Proof: `make catalog-check`.
- [ ] `-webkit-transition-timing-function` (`properties.json:737`). Point the catalog row at `applyTransitionProps` and the public-box test. Aliases may list the normalizer once `scripts/css-catalog-map.py` records the vendor pair (`scripts/css-catalog-map.py:379-400`). Status follows the test, using the rule at `properties.json:9`. Proof: `make catalog-check`.

### Deferred pointer

The per-name catalog rows above say `make catalog-check`. Do not mark them done from a style-only test.

- [ ] Point the transition names inside `documentation/deferred.md:110` and the five aliases inside `documentation/deferred.md:120` at `plans/v0.0.3/transitions/transitions-checklist.md`. Remove "permanent non-goal" for these 11 names only. Leave animation names on the animation ledger's row.
- [ ] Refresh `documentation/compatibility-matrix.md:1122` so the 45-name sentence no longer says transitions have no consumer. Regenerate the property section with `python3 scripts/css-catalog-map.py --matrix`.
- [ ] Update the local knowledge-base page in the same session as the code. This planning session does not write that page.

**Gate:** `make catalog-check` and `make matrix-check` exit 0. Record the new totals from the command. The planning baseline was 389 implemented, 19 partial, 369 unsupported, 8 intentionally ignored (`properties.json:12-17`).


## Phase 6: closure gates

These stay open until the commands exit 0 on the tree that contains the implementation. During the work, run one package: `go test ./internal/layout -run '<Name>' -count=1`, `go test ./internal/css -run '<Name>' -count=1`, `go test ./layout -run '<Name>' -count=1`, or `go test ./bindings/wasm -run '<Name>' -count=1`. Full gates go through the Makefile. Do not run bare `go test ./...`.

- [ ] `make test` exits 0.
- [ ] `make golden` exits 0. That target runs `TestDisplay` in `./layout`. It does not rasterize a page.
- [ ] `make lint` exits 0. `make lint` runs the file-size check. `internal/layout/layout.go` stays 2142 lines unless a reviewed allowlist edit says otherwise (`scripts/file-size-allowlist.txt:9`).
- [ ] `make claim-scan` exits 0. `claim-scan` reads `doc.go`, `README.md`, and `documentation/*.md` (`Makefile:104-114`). Do not claim a page PNG, a PDF, a running clock, a scrollport, or a speech synthesizer.
- [ ] `make catalog-check` exits 0.
- [ ] `make matrix-check` exits 0.
- [ ] GitHub CI (`.github/workflows/ci.yml:1`, steps at `19-26`) runs only `make test` and `make lint`. The other gates are local. A green CI run is not a green catalog.

**Gate:** every command above exited 0, and the output was recorded on the row.

## Phase 7: in-engine clock

- [~] Do not add a ticker that samples transition progress. Reason: `layout.DisplayList` returns one list (`layout/displaylist.go:129-138`). There is no second frame in this repo. Owner: a future host, the same owner as phase 7 of `plans/v0.0.3/animation/animation-checklist.md`. The native view plan mentions animation-frame callbacks only as part of a script engine (`plans/v0.0.1/native-window/native-web-view-checklist.md:93`). Next gate: that host plan names one Blinkless call that takes time `t` and returns a new display list, and a test shows two lists. Until then the host reads `Box.Transitions` and samples time itself.

## Dependencies

- Acceptance shares `supportedDeclaration` with the rest of the cascade (`internal/layout/style_value_accept.go:83-87`).
- Easing shares `internal/layout/easing_value.go` with the animation ledger. Whichever ledger lands first creates that file.
- The from-value is the host's previous display list. This ledger does not store it.
- The clock non-goal is shared with the animation ledger. Do not build a second clock here.
- No new direct module. The allowlist stays `github.com/go-text/typesetting` and `github.com/tdewolff/canvas`.

