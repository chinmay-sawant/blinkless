# v0.0.3 - Row rules

> **Parent:** `plans/v0.0.3/README.md`
> **Status:** phases 1 through 6 landed. Phase 6 (junction-aware segments, breaks, cap and junction insets, grid column rules, multicol cap insets) landed 2026-10-16 and closed on the package run `go test ./internal/layout -count=1` -> `ok github.com/chinmay-sawant/blinkless/internal/layout 0.266s`. Phase 6's full end-of-session gate exited 0 on 2026-10-16 (see Phase 6).
> **Estimated effort:** 27 catalog properties (all unsupported when this ledger was written), across 6 phases. `column-rule` is the pattern and is not in this count.

---

## Overview

A column rule is a line painted in the gap between columns. Blinkless already does that for multicol. A row rule is the same idea for the gap between rows. The `rule*` names are shorthands that set both axes. This file plans the row axis and those shorthands. It does not replan `column-rule`.

The paint we copy is a stroked line. `emitColumnRules` centers a vertical rule in the multicol gap and calls `emitBorderLine` (`internal/layout/multicol.go:893-924`, `internal/layout/layout_chrome.go:198-207`). The op is a line, not a new kind. Row rules use that same line painter, drawn horizontally in a row gap.

Grid row gaps already changed box positions before this work; a gap by itself paints no rule. `TestBehaviorRowGapGridUsedSpacing` places the second row 20px below the first (`internal/layout/css_behavior_flex_grid_test.go:272`). The gap size comes from `styleGaps` (`internal/layout/grid.go:859-867`, `internal/layout/style.go:131`), which flex shares. Phases 1 and 2 added the row-rule painter in `internal/layout/gap_rules.go`; phase 6 made it junction-aware.

`column-rule` width, style, and color were implemented when this ledger was written. Phase 6 added the column half of break and insets: grid column rules paint with junction-aware segments, and multicol column rules honor cap insets. Where a `rule*` shorthand sets both axes, the same storage functions write both halves. `column-rule-visibility-items` has no storage or consumer and stays unsupported (see Phase 6).

`documentation/deferred.md` called `row-rule*` and `rule*` a permanent non-goal at planning time; it now records the shipped subset (`documentation/deferred.md:106`). This file stays the planning home for the family.

The output is a display list. Rules are `OpLine` segments. Do not add a page rasterizer. Do not grow `internal/layout/layout.go`.

Grammar source for the names is the catalog, which points at CSS Gaps (`https://drafts.csswg.org/css-gaps-1/#propdef-row-rule` and `https://drafts.csswg.org/css-gaps-1/#propdef-rule`). This is not a Chrome parity claim.

## Executive summary

| Family | Count | Catalog status |
|---|---|---|
| `column-rule`, `column-rule-color`, `column-rule-style`, `column-rule-width` | 4 | implemented, cited as the pattern, not replanned |
| Other `column-rule-*` | 11 | 10 implemented (break and the nine insets, phase 6); `column-rule-visibility-items` unsupported, out of scope |
| `row-rule*` | 15 | implemented |
| `rule*` | 12 | implemented |
| In-scope total | 27 | |

Catalog status was updated by the integrating session on 2026-10-16 (`testdata/css/catalog/properties.json`); the integrating session's end-of-session run is the authority for `make catalog-check` and `make matrix-check`. The row closures below cite tests and code, not the catalog file.

## Phase 0: contract and baseline

- [x] `column-rule: 3px solid #c00` paints one vertical line centered in the gap. On a 300px box with two columns and a 20px gap the rule sits 150px from the container's left. Proof: `TestBehaviorColumnRulePaintedBetweenColumns` (`internal/layout/css_behavior_columns_test.go:183`).
- [x] `column-rule-width: 6px` sets the stroke width. With no color, the line uses the text color. Proof: `TestBehaviorColumnRuleWidthUsedThickness` (`internal/layout/css_behavior_columns_test.go:218`).
- [x] `column-rule-color: #00f` paints one blue line of the declared width. Proof: `TestBehaviorColumnRuleColorBluePaintsRule` (`internal/layout/css_behavior_text_remaining_test.go:351`).
- [x] `column-rule-style: dashed` paints more than one segment at the same gap-middle X. Solid paints one segment. Proof: `TestBehaviorColumnRuleStyleDashedSegmentsRule` (`internal/layout/css_behavior_text_remaining_test.go:316`).
- [x] The shorthand and the three longhands are stored in `applyColumnRuleProps` (`internal/layout/style_multicol_props.go:35-52`). The shorthand parser is `parseRuleShorthand` (`internal/layout/style_paint_props.go:221`). Paint is `emitColumnRules` (`internal/layout/multicol.go:893`).
- [x] No `row-rule` or `rule` handler existed when this baseline was written. Phases 1 through 4 added `applyRowRuleGroup`. The catalog rows now match the paint tests.
- [x] New files: `internal/layout/style_gap_rules.go` for storage, `internal/layout/gap_rules.go` for paint. Call `emitBorderLine`. Do not add a kind. The only later edit to `emitColumnRules` is the phase 6 cap-inset span (`internal/layout/multicol.go:907-908`); the width, style, and color path is unchanged.
- [x] `column-rule-break` and the nine `column-rule-inset*` names: stored in `applyColumnRuleProps` and painted since phase 6. Proof: Phase 6.
- [~] `column-rule-visibility-items`. Reason: no storage or consumer exists (`internal/layout/style_gap_rules.go:147-148` maps `rule-visibility-items` to the row half only, and `applyColumnRuleProps` has no case for it at `internal/layout/style_multicol_props.go:35-94`), so `around`/`all` cannot paint at the first and last column gaps. Owner: this ledger. Next gate: a column visibility wave with edge paint tests. `rule-inset` and `rule-break` store and paint both axes; `rule-visibility-items` still stores the row half only.

**Gate:** column-rule tests stay green. No row line existed at this gate.

## Phase 1: store `row-rule` width, style, and color

Copy the column-rule subset first. One list item is enough for the first wave. Extra list items are stored and the painter uses the first item until a later row says otherwise. Say that in the test so a two-item list is not a silent success.

- [x] `row-rule-width`. Catalog `<line-width-list> | <auto-line-width-list>`. Accept one width with the same parser as `column-rule-width` (`parseOutlineWidth` in `internal/layout/style_multicol_props.go:58-61`). `auto` uses the medium border width, the same initial `column-rule` uses (`internal/layout/style.go:140`).
- [x] `row-rule-style`. Catalog `<line-style-list> | <auto-line-style-list>`. Accept one line style with `parseOutlineStyle`, the same parser as `column-rule-style` (`internal/layout/style_multicol_props.go:62-65`). `none` paints nothing.
- [x] `row-rule-color`. Catalog `<line-color-list> | <auto-line-color-list>`. Accept one color. Missing color means the element's used color, matching `emitColumnRules` (`internal/layout/multicol.go:913-918`).
- [x] `row-rule`. Catalog `<gap-rule-list> | <gap-auto-rule-list>`. One shorthand sets width, style, and color through `parseRuleShorthand` (`internal/layout/style_paint_props.go:221`). Omitted pieces reset to the initial values, as the column shorthand does (`internal/layout/style_multicol_props.go:96-106`).
- [x] A bad token does not replace an earlier valid width, style, or color.
- [x] Storage lives on the resolved style next to the column-rule fields, interned with them. Do not overload `ColumnRuleWidth` for rows.

**Gate:** store tests only. No line ops yet.

## Phase 2: paint the row gap

New file `internal/layout/gap_rules.go`.

- [x] Grid: after the row positions are known, paint one horizontal rule in each gap that `RowGap` created (paint call at `internal/layout/grid.go:121-123`). The line is centered in the gap and spans the grid content width. Style `none` or width 0 paints nothing, matching `emitColumnRules` (`internal/layout/multicol.go:894-905`).
- [x] Flex: paint one horizontal rule in each row gap when `flex-wrap` produces more than one line and `row-gap` is positive. A single flex line paints no row rule.
- [x] Multicol: do not invent row rules between column lines. Multicol rows exist only when `column-wrap` makes extra rows. If that wrap path has no row boxes today, skip multicol in this wave and say so in the test. Do not change `emitColumnRules`.
- [x] The op is the same line segment `emitBorderLine` already appends (`internal/layout/layout_chrome.go:198-207`). Dashed style uses the same segment splitter as `column-rule-style: dashed`.
- [x] A zero gap paints no rule even if `row-rule` is set. The rule does not add gap size. `TestBehaviorRowGapGridUsedSpacing` stays at 20px of space.
- [x] Behavior test: a two-row grid with `row-gap: 20px; row-rule: 3px solid #c00` emits one horizontal line, 3px wide, red, centered in the 20px gap. Dashed emits more than one segment at that Y. No rule when style is `none`.

**Gate:** the grid test paints a line. Column-rule tests still pass. Golden `layout.TestDisplay` stays green.

## Phase 3: inset, break, visibility, and the `rule*` shorthands

These names are in the catalog and have no column-rule consumer to copy, except where `rule` sets width, style, or color. Paint stays in `internal/layout/gap_rules.go`.

### Row longhands beyond the first subset

- [x] `row-rule-break`. Catalog `none | normal | intersection`. Store the keyword. `normal` keeps the phase 2 line. `none` does not break the line at a crossing. `intersection` stops the row line at unflanked column-gap junctions. Phase 6 made the crossing real (`internal/layout/gap_rules.go:216-321`, `:555-584`); proof: `TestBehaviorRowRuleBreakIntersectionSplitsAtColumnGaps` (`internal/layout/css_behavior_gap_break_test.go:107`) and `TestBehaviorRowRuleIntersectionFlankedMerge` (`:182`).
- [x] `row-rule-visibility-items`. Catalog `all | around | between | normal`. `between` and `normal` paint only between items, which is phase 2. `around` also paints a rule before the first row and after the last row, inside the content box, not outside the border. `all` paints between and around. Known divergence: `normal` is treated as `between` (see the divergences section below).
- [x] `row-rule-inset`. Catalog `<'column-rule-inset-cap'> [ / <'column-rule-inset-junction'> ]?`. This shortens the line. The cap pair is the two ends. The junction pair is the gap left at a crossing. Store both. Phase 6 paints both: cap insets at content edges, junction insets at grid junctions, percentages against the crossing gap size (`internal/layout/gap_rules.go:339-377`). Proof: `TestBehaviorRowRuleJunctionAndCapInsets` (`internal/layout/css_behavior_gap_break_test.go:151`), `TestRuleInsetCopySemantics` (`internal/layout/style_gap_rules_test.go:242`).
- [x] `row-rule-inset-cap`. Catalog `<inset-value> <inset-value>?`. One value sets both ends. Two values set start then end.
- [x] `row-rule-inset-cap-start`. Catalog `<inset-value>`. Shortens the start of the row line.
- [x] `row-rule-inset-cap-end`. Catalog `<inset-value>`. Shortens the end of the row line.
- [x] `row-rule-inset-start`. Catalog `<inset-value>`. Same stored end as cap-start when the shorthand is not used. If both are set, the longhand that appears later in the cascade wins, which is ordinary CSS. Do not invent a second meaning.
- [x] `row-rule-inset-end`. Catalog `<inset-value>`. Same relationship to cap-end.
- [x] `row-rule-inset-junction`. Catalog `<inset-value> <inset-value>?`. Painted at grid junction endpoints since Phase 6; lengths, percentages (crossing gap size), and overlap-join all apply (`internal/layout/gap_rules.go:361-376`). In flex there are no junctions, so the value is stored and inert.
- [x] `row-rule-inset-junction-start`. Catalog `<inset-value>`. Painted at grid junction start endpoints since Phase 6 (`TestBehaviorRowRuleJunctionAndCapInsets`, `internal/layout/css_behavior_gap_break_test.go:151`).
- [x] `row-rule-inset-junction-end`. Catalog `<inset-value>`. Stored pair: `TestRuleInsetCopySemantics` (`internal/layout/style_gap_rules_test.go:242`). Paint reads it for end endpoints at grid junctions through the shared segment kernel (`internal/layout/gap_rules.go:353-359`, field bound at `:202`). No row fixture ends at a junction; that endpoint's paint is proven on the column axis by `TestBehaviorColumnRuleJunctionAndCapInsets` (`internal/layout/css_behavior_gap_break_test.go:206`), so this row closes on shared-kernel plus storage proof.

An `<inset-value>` is a length or a percentage of the gap segment, or `auto`. `auto` means 0 until a test cites a different initial from the gaps draft. Write that choice into the store test.

### `rule*` shorthands

`rule` maps to both axes. The column half writes width, style, and color through `applyColumnRuleProps`, and break and insets through `expandGapRuleDeclaration` (`internal/layout/style_gap_rules.go:124-176`, `internal/layout/style_multicol_props.go:41-42`, `:79-94`).

- [x] `rule`. Catalog `<'column-rule'>`. Sets `column-rule` and `row-rule` width, style, and color. The column line paints in multicol and, since Phase 6, in grid column gaps (`internal/layout/gap_rules.go:489-531`). The row line paints in grid and flex.
- [x] `rule-width`. Catalog `<'column-rule-width'>`. Sets both widths.
- [x] `rule-style`. Catalog `<'column-rule-style'>`. Sets both styles.
- [x] `rule-color`. Catalog `<'column-rule-color'>`. Sets both colors.
- [x] `rule-break`. Catalog `<'column-rule-break'>`. Sets both axes (`duplicateGapLonghand` to `column-rule-break` and `row-rule-break`, `internal/layout/style_gap_rules.go:145-146`).
- [~] `rule-visibility-items`. Catalog `<'column-rule-visibility-items'>`. Sets `row-rule-visibility-items` only; `column-rule-visibility-items` has no storage or consumer (`internal/layout/style_gap_rules.go:147-148`). Owner: this ledger. Next gate: a column visibility wave.
- [x] `rule-inset`. Catalog `<'column-rule-inset'>`. Sets both axes (`expandBothAxisRuleInset`, `internal/layout/style_gap_rules.go:250-263`).
- [x] `rule-inset-cap`. Catalog `<'column-rule-inset-cap'>`. Sets both axes (`expandBothAxisInsetPair`, `internal/layout/style_gap_rules.go:161-162`).
- [x] `rule-inset-start`. Catalog `<'column-rule-inset-start'>`. Sets both axes (`insetSideDecls` with both, `internal/layout/style_gap_rules.go:191-192`).
- [x] `rule-inset-end`. Catalog `<'column-rule-inset-end'>`. Sets both axes (`insetSideDecls`, `internal/layout/style_gap_rules.go:193-194`).
- [x] `rule-inset-junction`. Catalog `<'column-rule-inset-junction'>`. Sets both axes (`expandBothAxisInsetPair`, `internal/layout/style_gap_rules.go:163-166`).
- [x] `rule-overlap`. Catalog `row-over-column | column-over-row`. `row-over-column` appends the row line after the column line; `column-over-row` appends the column line after the row line (`internal/layout/gap_rules.go:410-429`). Proof: `TestBehaviorRuleOverlapOrdersLines` (`internal/layout/css_behavior_row_rule_test.go:332`). Multicol paints no row rule, so its order is unchanged.

**Gate:** each of the 27 names has a store test. Width, style, color, the shorthand `rule`, cap and junction insets, visibility, and overlap have paint tests. The junction paint and `intersection` breaks this gate originally left open were closed in phase 6 (see Phase 6).

## Phase 4: behavior tests

New file `internal/layout/css_behavior_row_rule_test.go`.

- [x] `row-rule: 3px solid #c00` on a two-row grid paints the centered red line from phase 2.
- [x] `row-rule-width: 6px` with a solid style paints a 6px stroke. Color defaults to black text, the same check as `TestBehaviorColumnRuleWidthUsedThickness`.
- [x] `row-rule-color: #00f` paints blue.
- [x] `row-rule-style: dashed` paints at least two segments. `none` paints zero.
- [x] `row-rule-visibility-items: around` paints a rule in the gap plus a rule along the top and bottom of the content box. `between` paints only the gap rule.
- [x] `row-rule-inset-cap: 10px` makes the gap line 20px shorter than the full content width, 10px at each end.
- [x] `row-rule-break: none` stores `none` and still paints the phase 2 line.
- [x] `rule: 3px solid #c00` on a multicol box still paints the vertical column rule from `TestBehaviorColumnRulePaintedBetweenColumns`, and on a grid also paints the horizontal row rule.
- [x] `rule-width`, `rule-style`, and `rule-color` change both axes. A grid test checks the row line. A multicol test checks the column line.
- [x] `rule-overlap: row-over-column` orders the row line after the column line in one grid that has both gaps. If the grid painter has no column-rule line yet, this row stays open rather than painting a fake column line. Do not build column-rule inset to get there. A column line in a grid is new and belongs in this row only as paint order of two `OpLine`s, using the existing column-rule width, style, and color fields.
- [x] Bad tokens keep the previous value for each of the 27 names. One table-driven store test is enough if each name is an entry.

**Gate:** the paint rows that phase 3 marked as painted are green. Column-rule tests are unchanged.

## Phase 5: catalog and closure

- [x] Mark a `row-rule*` or `rule*` name `implemented` only when its paint test exists. Phase 5 left the junction-inset names and `row-rule-break: intersection` open on purpose. Phase 6 painted them (see Phase 6) and the integrating session flipped those catalog rows.
- [x] Do not change the four implemented `column-rule` rows except if `rule` writes their fields and a test still covers them.
- [x] The 11 out-of-scope `column-rule-*` names: phase 6 brought 10 of them in scope (break and the nine insets) after `rule-break` and `rule-inset` made the shared grammar real, and those 10 are implemented. `column-rule-visibility-items` stays unsupported on purpose (row above).
- [x] Regenerate the matrix property section. Phase 6 flips more rows, so the integrating session regenerates it again and owns `make matrix-check` (see Phase 6).
- [x] The deferred row-rule sentence was updated when this phase closed, because leaving "not implemented" would have been false. The host still draws the line ops.
- [x] Do not grow `internal/layout/layout.go`.
- [x] Run `make test`. Do not run bare `go test ./...`. `make test` exited 0 on 2026-10-16.
- [x] Run `make golden`.
- [x] Run `make lint`. `make lint` exited 0 on 2026-10-16.
- [x] Run `make claim-scan`. Do not claim a page PNG or Chrome parity for gap decorations.
- [x] Run `make catalog-check` and `make matrix-check`.
- [x] GitHub CI runs only `make test` and `make lint` (`.github/workflows/ci.yml:1`, `.github/workflows/ci.yml:20`, `.github/workflows/ci.yml:26`).

Focused run during the work:

```sh
go test ./internal/layout -run 'TestBehaviorColumnRule|TestBehaviorRowRule|TestBehaviorRowGap' -count=1
```

The dated `make test` and `make lint` runs above cover the phase 5 change. They do not cover the phase 6 paint.

**Gate:** phase 5's gate closed on the phase 5 change. Phase 6's full gate closed at the end of this file (see Phase 6).

## Phase 6: junction-aware segments, both-axis insets, and multicol cap insets

Landed 2026-10-16. The fixtures mirror css-gaps-1 geometry probes run against Chrome 155 (chrome-headless-shell): a 3x2 grid, columns at [0,50], [70,120], [140,190], rows at [0,50], [70,120], in CSS px (`internal/layout/css_behavior_gap_break_test.go:5-9`).

- [x] Junction-aware segments on both axes: `gapRuleSegments` builds endpoints at the gap bounds and at every visible junction, drops pairs covered by spanning items, merges what stays clear, and breaks unflanked junctions when the rule break is `intersection` (`internal/layout/gap_rules.go:224-265`, endpoints `:273-297`, discontiguous `:301-309`, flanked `:313-321`). Proof: `TestBehaviorRowRuleBreakIntersectionSplitsAtColumnGaps` (`internal/layout/css_behavior_gap_break_test.go:107`), `TestBehaviorRowRuleSpanDropsCoveredPair` (`:123`), `TestBehaviorRowRuleBreakNonePaintsThroughJunctions` (`:137`), `TestBehaviorRowRuleIntersectionFlankedMerge` (`:182`).
- [x] Grid column rules paint from the `column-rule` fields with the same kernel: `paintGridColumnRules` (`internal/layout/gap_rules.go:489-531`), junctions `gridColumnGapJunctions` (`:650-675`), spans `gridColumnGapSpans` (`:713-731`), flanking `gridRowGapFlanked` (`:694-709`). Proof: `TestBehaviorColumnRuleBreakIntersectionSplitsAtRowGaps` (`internal/layout/css_behavior_gap_break_test.go:193`), `TestBehaviorColumnRuleJunctionAndCapInsets` (`:206`), `TestBehaviorRulePaintsBothAxes` (`internal/layout/css_behavior_row_rule_test.go:275`).
- [x] Cap and junction insets on both axes: `gapRuleInsetOffset` reads cap insets at content edges and junction insets at junction endpoints (`internal/layout/gap_rules.go:345-377`); `rowRuleInsets` and `columnRuleInsets` bind both field sets (`:197-214`); the storage grammar is `expandGapRuleDeclaration` (`internal/layout/style_gap_rules.go:124-176`). Proof: `TestBehaviorRowRuleJunctionAndCapInsets` (`internal/layout/css_behavior_gap_break_test.go:151`), `TestBehaviorColumnRuleJunctionAndCapInsets` (`:206`), `TestRuleInsetCopySemantics` (`internal/layout/style_gap_rules_test.go:242`).
- [x] Percentages resolve against the crossing gap size at junctions and are no-ops at caps, where there is no crossing gap (`internal/layout/gap_rules.go:368-373`). Stored percentage proof: `TestRuleInsetCopySemantics` (`internal/layout/style_gap_rules_test.go:357-366`).
- [x] `overlap-join` extends an endpoint into its junction by half the crossing gap plus half the used crossing rule width (`internal/layout/gap_rules.go:362-367`), reading the crossing axis rule width (`:483`, `:511`). Proof: `TestBehaviorRowRuleJunctionOverlapJoin` (`internal/layout/css_behavior_gap_break_test.go:167`): 12px with a 4px column rule, 11.5px with the initial medium 3px.
- [x] Multicol column rules honor cap insets; percentage, junction, and overlap-join insets are no-ops on a multicol line: `emitColumnRules` (`internal/layout/multicol.go:893-924`) and `columnRuleCapInset` (`:931-937`). Proof: `TestColumnRuleCapInsetsSpanTheLine` (`internal/layout/multicol_test.go:400`), whose comment records the Chrome 155.0.8059.84 probe on a 240x120 content box (`:396-399`).
- [x] `rule-overlap` orders the two grid lines: `paintGridGapRules` (`internal/layout/gap_rules.go:410-429`). Proof: `TestBehaviorRuleOverlapOrdersLines` (`internal/layout/css_behavior_row_rule_test.go:332`).
- [x] Package gate: `go test ./internal/layout -count=1` -> `ok github.com/chinmay-sawant/blinkless/internal/layout 0.266s` on 2026-10-16. A targeted run of this phase's tests also exits 0 (`ok github.com/chinmay-sawant/blinkless/internal/layout 0.074s`).
- [x] Companion fixture: `../gowkhtmltopdf/testdata/golden/fixture-65-props-390-to-420-row-rules-masks-3d.html` carries cap and junction demos and has no `(partial)` row-rule labels left; only the unused `.chrome-partial` class definition remains (line 151). The regenerated PDFs live in that repo and are not evidence for this tree.
- [x] End-of-session full gate on the phase 6 change: `make test`, `make golden`, `make lint`, `make claim-scan`, `make catalog-check`, `make matrix-check` all exited 0 on 2026-10-16 (logs under `/tmp/opencode/gates3/`); `make lint` chains `size-check`, also clean. `go test ./internal/layout -count=1` -> `ok github.com/chinmay-sawant/blinkless/internal/layout 0.234s`.

**Gate:** closed. The integrating session's end-of-session run exited 0 on the change that flips the catalog rows; the package run above is green.

## Known divergences vs Chrome 155 (gap rules)

Checked against the code on 2026-10-16:

1. Flex intersection junctions are not painted. `emitFlexRowGapRules` calls `emitRowGapRule`, which builds a bounds-only geometry (`internal/layout/gap_rules.go:68-103`, `:139-178`); no junction list or item spans exist on that path, so `row-rule-break: intersection` has no crossing to break at in flex.
2. `row-rule-visibility-items: normal` is treated as `between`. Only `around` and `all` add the content-edge rules (`internal/layout/gap_rules.go:187-194`).
3. Flex column rules are not painted. `paintGridGapRules` is called only from grid layout (`internal/layout/grid.go:123`), and the flex row-gap painter emits no vertical line, so `column-rule` on a flex container paints nothing.
4. Gap decorations paint after items. Grid paints items in `emitGridBoxes` and then the rules (`internal/layout/grid.go:121-123`); flex places items in `placeFlexLineMeasured` and then paints the rules (`internal/layout/flex.go:275-302`). The line ops sit on top of item paint in the display list.
5. Column-rule extents use the content box. `paintGridColumnRules` spans the first row track's top to the last row track's bottom (`internal/layout/gap_rules.go:503-508`), and row rules span `contentX..contentX+contentW`. Chrome 155 bounds column-rule extents to the grid border box in the recorded probes; the css-gaps spec text speaks of the container's content edges.

## Dependencies

- Pattern to copy: `applyColumnRuleProps` and `applyColumnRuleShorthand` in `internal/layout/style_multicol_props.go`, and `parseRuleShorthand` in `internal/layout/style_paint_props.go`.
- Line painter to call: `emitBorderLine` in `internal/layout/layout_chrome.go`. Do not copy the segment math.
- Multicol paint: `emitColumnRules` in `internal/layout/multicol.go`; phase 6 extended it with cap insets (`:907-908`) and left its width, style, and color path alone.
- Gap sizes already computed for grid: `internal/layout/grid.go`. Flex gap is the same `RowGap` field (`internal/layout/style.go:131`).
- New code: `internal/layout/style_gap_rules.go` and `internal/layout/gap_rules.go`.
- Phase 5 kept the 11 `column-rule-*` inset, break, and visibility names out of scope. Phase 6 brought the 10 whose grammar `rule-inset` and `rule-break` share in scope and painted them; `column-rule-visibility-items` remains out of scope.
