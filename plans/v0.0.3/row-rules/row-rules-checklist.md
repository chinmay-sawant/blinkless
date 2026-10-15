# v0.0.3 - Row rules

> **Parent:** `plans/v0.0.3/README.md`
> **Status:** not started. Baseline rows record verified current behavior. Implementation rows are open.
> **Estimated effort:** 27 catalog properties, all unsupported, across 5 phases. `column-rule` is the pattern and is not in this count.

---

## Overview

A column rule is a line painted in the gap between columns. Blinkless already does that for multicol. A row rule is the same idea for the gap between rows. The `rule*` names are shorthands that set both axes. This file plans the row axis and those shorthands. It does not replan `column-rule`.

The paint we copy is a stroked line. `emitColumnRules` centers a vertical rule in the multicol gap and calls `emitBorderLine` (`internal/layout/multicol.go:888-914`, `internal/layout/layout_chrome.go:198-206`). The op is a line, not a new kind. Row rules use that same line painter, drawn horizontally in a row gap.

Grid row gaps already change box positions and do not paint a rule. `TestBehaviorRowGapGridUsedSpacing` places the second row 20px below the first (`internal/layout/css_behavior_flex_grid_test.go:272`). The gap size is `RowGap` (`internal/layout/grid.go:857-863`, `internal/layout/style.go:131`). Flex has the same gap fields. There is no row-rule painter.

`column-rule` width, style, and color are implemented. The catalog `column-rule-inset*`, `column-rule-break`, and `column-rule-visibility-items` rows are unsupported. This ledger does not add them. Where a `rule*` shorthand is specified to set both axes, the column half that maps onto width, style, or color may write the existing column-rule fields. The column half that maps onto inset, break, or visibility is a non-goal here.

`documentation/deferred.md` calls `row-rule*` and `rule*` a permanent non-goal. This file is the planning home. Do not edit `deferred.md`.

The output is a display list. Rules are `OpLine` segments. Do not add a page rasterizer. Do not grow `internal/layout/layout.go`.

Grammar source for the names is the catalog, which points at CSS Gaps (`https://drafts.csswg.org/css-gaps-1/#propdef-row-rule` and `https://drafts.csswg.org/css-gaps-1/#propdef-rule`). This is not a Chrome parity claim.

## Executive summary

| Family | Count | Catalog status |
|---|---|---|
| `column-rule`, `column-rule-color`, `column-rule-style`, `column-rule-width` | 4 | implemented, cited as the pattern, not replanned |
| Other `column-rule-*` | 11 | unsupported, out of scope |
| `row-rule*` | 15 | unsupported, in scope |
| `rule*` | 12 | unsupported, in scope |
| In-scope total | 27 | |

## Phase 0: contract and baseline

- [x] `column-rule: 3px solid #c00` paints one vertical line centered in the gap. On a 300px box with two columns and a 20px gap the rule sits 150px from the container's left. Proof: `TestBehaviorColumnRulePaintedBetweenColumns` (`internal/layout/css_behavior_columns_test.go:183`).
- [x] `column-rule-width: 6px` sets the stroke width. With no color, the line uses the text color. Proof: `TestBehaviorColumnRuleWidthUsedThickness` (`internal/layout/css_behavior_columns_test.go:218`).
- [x] `column-rule-color: #00f` paints one blue line of the declared width. Proof: `TestBehaviorColumnRuleColorBluePaintsRule` (`internal/layout/css_behavior_text_remaining_test.go:351`).
- [x] `column-rule-style: dashed` paints more than one segment at the same gap-middle X. Solid paints one segment. Proof: `TestBehaviorColumnRuleStyleDashedSegmentsRule` (`internal/layout/css_behavior_text_remaining_test.go:316`).
- [x] The shorthand and the three longhands are stored in `applyColumnRuleProps` (`internal/layout/style_multicol_props.go:35-68`). The shorthand parser is `parseRuleShorthand` (`internal/layout/style_paint_props.go:221`). Paint is `emitColumnRules` (`internal/layout/multicol.go:888`).
- [x] No `row-rule` or `rule` handler exists. The catalog status for all 27 in-scope names is `unsupported`, with the shared limitation that there is no apply handler.
- [ ] New files: `internal/layout/style_gap_rules.go` for storage, `internal/layout/gap_rules.go` for paint. Call `emitBorderLine`. Do not add a kind. Do not edit `emitColumnRules` except to let `rule` write the existing width, style, and color fields through the current setter.
- [~] `column-rule-break`, `column-rule-inset`, `column-rule-inset-cap`, `column-rule-inset-cap-end`, `column-rule-inset-cap-start`, `column-rule-inset-end`, `column-rule-inset-junction`, `column-rule-inset-junction-end`, `column-rule-inset-junction-start`, `column-rule-inset-start`, and `column-rule-visibility-items`. Reason: they are unsupported and this ledger does not replan `column-rule`. Owner: this ledger. Next gate: a separate column-rule inset plan. `rule-inset` and `rule-break` and `rule-visibility-items` still store the row half below.

**Gate:** column-rule tests stay green. No row line exists yet.

## Phase 1: store `row-rule` width, style, and color

Copy the column-rule subset first. One list item is enough for the first wave. Extra list items are stored and the painter uses the first item until a later row says otherwise. Say that in the test so a two-item list is not a silent success.

- [ ] `row-rule-width`. Catalog `<line-width-list> | <auto-line-width-list>`. Accept one width with the same parser as `column-rule-width` (`parseOutlineWidth` in `internal/layout/style_multicol_props.go:39-41`). `auto` uses the medium border width, the same initial `column-rule` uses (`internal/layout/style.go:140`).
- [ ] `row-rule-style`. Catalog `<line-style-list> | <auto-line-style-list>`. Accept one line style with `parseOutlineStyle`, the same parser as `column-rule-style` (`internal/layout/style_multicol_props.go:43-45`). `none` paints nothing.
- [ ] `row-rule-color`. Catalog `<line-color-list> | <auto-line-color-list>`. Accept one color. Missing color means the element's used color, matching `emitColumnRules` (`internal/layout/multicol.go:904-909`).
- [ ] `row-rule`. Catalog `<gap-rule-list> | <gap-auto-rule-list>`. One shorthand sets width, style, and color through `parseRuleShorthand` (`internal/layout/style_paint_props.go:221`). Omitted pieces reset to the initial values, as the column shorthand does (`internal/layout/style_multicol_props.go:59-68`).
- [ ] A bad token does not replace an earlier valid width, style, or color.
- [ ] Storage lives on the resolved style next to the column-rule fields, interned with them. Do not overload `ColumnRuleWidth` for rows.

**Gate:** store tests only. No line ops yet.

## Phase 2: paint the row gap

New file `internal/layout/gap_rules.go`.

- [ ] Grid: after the row positions are known, paint one horizontal rule in each gap that `RowGap` created (`internal/layout/grid.go:863`). The line is centered in the gap and spans the grid content width. Style `none` or width 0 paints nothing, matching `emitColumnRules` (`internal/layout/multicol.go:895-902`).
- [ ] Flex: paint one horizontal rule in each row gap when `flex-wrap` produces more than one line and `row-gap` is positive. A single flex line paints no row rule.
- [ ] Multicol: do not invent row rules between column lines. Multicol rows exist only when `column-wrap` makes extra rows. If that wrap path has no row boxes today, skip multicol in this wave and say so in the test. Do not change `emitColumnRules`.
- [ ] The op is the same line segment `emitBorderLine` already appends (`internal/layout/layout_chrome.go:198-206`). Dashed style uses the same segment splitter as `column-rule-style: dashed`.
- [ ] A zero gap paints no rule even if `row-rule` is set. The rule does not add gap size. `TestBehaviorRowGapGridUsedSpacing` stays at 20px of space.
- [ ] Behavior test: a two-row grid with `row-gap: 20px; row-rule: 3px solid #c00` emits one horizontal line, 3px wide, red, centered in the 20px gap. Dashed emits more than one segment at that Y. No rule when style is `none`.

**Gate:** the grid test paints a line. Column-rule tests still pass. Golden `layout.TestDisplay` stays green.

## Phase 3: inset, break, visibility, and the `rule*` shorthands

These names are in the catalog and have no column-rule consumer to copy, except where `rule` sets width, style, or color. Paint stays in `internal/layout/gap_rules.go`.

### Row longhands beyond the first subset

- [ ] `row-rule-break`. Catalog `none | normal | intersection`. Store the keyword. `normal` keeps the phase 2 line. `none` does not break the line at a crossing. `intersection` stops the row line where a column line crosses it. Until column and row lines can cross in one grid, `intersection` matches `normal` and the test says so.
- [ ] `row-rule-visibility-items`. Catalog `all | around | between | normal`. `between` and `normal` paint only between items, which is phase 2. `around` also paints a rule before the first row and after the last row, inside the content box, not outside the border. `all` paints between and around.
- [ ] `row-rule-inset`. Catalog `<'column-rule-inset-cap'> [ / <'column-rule-inset-junction'> ]?`. This shortens the line. The cap pair is the two ends. The junction pair is the gap left at a crossing. Store both. The painter applies the cap inset in this wave. Junction inset waits until a crossing exists, and the test says the junction half is stored and not yet painted.
- [ ] `row-rule-inset-cap`. Catalog `<inset-value> <inset-value>?`. One value sets both ends. Two values set start then end.
- [ ] `row-rule-inset-cap-start`. Catalog `<inset-value>`. Shortens the start of the row line.
- [ ] `row-rule-inset-cap-end`. Catalog `<inset-value>`. Shortens the end of the row line.
- [ ] `row-rule-inset-start`. Catalog `<inset-value>`. Same stored end as cap-start when the shorthand is not used. If both are set, the longhand that appears later in the cascade wins, which is ordinary CSS. Do not invent a second meaning.
- [ ] `row-rule-inset-end`. Catalog `<inset-value>`. Same relationship to cap-end.
- [ ] `row-rule-inset-junction`. Catalog `<inset-value> <inset-value>?`. Stored. Not painted until a column rule crosses the row rule in the same grid.
- [ ] `row-rule-inset-junction-start`. Catalog `<inset-value>`. Stored. Not painted in this wave.
- [ ] `row-rule-inset-junction-end`. Catalog `<inset-value>`. Stored. Not painted in this wave.

An `<inset-value>` is a length or a percentage of the gap segment, or `auto`. `auto` means 0 until a test cites a different initial from the gaps draft. Write that choice into the store test.

### `rule*` shorthands

`rule` maps to both axes. The column half may call `applyColumnRuleProps` for width, style, and color only.

- [ ] `rule`. Catalog `<'column-rule'>`. Sets `column-rule` and `row-rule` width, style, and color. The column line stays the multicol painter. The row line stays phase 2.
- [ ] `rule-width`. Catalog `<'column-rule-width'>`. Sets both widths.
- [ ] `rule-style`. Catalog `<'column-rule-style'>`. Sets both styles.
- [ ] `rule-color`. Catalog `<'column-rule-color'>`. Sets both colors.
- [ ] `rule-break`. Catalog `<'column-rule-break'>`. Sets `row-rule-break` only. The column half is the phase 0 non-goal.
- [ ] `rule-visibility-items`. Catalog `<'column-rule-visibility-items'>`. Sets `row-rule-visibility-items` only. The column half is the phase 0 non-goal.
- [ ] `rule-inset`. Catalog `<'column-rule-inset'>`. Sets `row-rule-inset` only. The column half is the phase 0 non-goal.
- [ ] `rule-inset-cap`. Catalog `<'column-rule-inset-cap'>`. Sets the row cap insets only.
- [ ] `rule-inset-start`. Catalog `<'column-rule-inset-start'>`. Sets the row start inset only.
- [ ] `rule-inset-end`. Catalog `<'column-rule-inset-end'>`. Sets the row end inset only.
- [ ] `rule-inset-junction`. Catalog `<'column-rule-inset-junction'>`. Sets the row junction insets only.
- [ ] `rule-overlap`. Catalog `row-over-column | column-over-row`. This name has no column-rule twin in the "do not replan" list. Store it. When a grid paints both a row line and a column line, `row-over-column` appends the row line after the column line. `column-over-row` appends the column line after the row line. Multicol column rules keep their current order when no row rule is painted.

**Gate:** each of the 27 names has a store test. Width, style, color, the shorthand `rule`, cap insets, and visibility `around` have a paint test. Junction paint and `intersection` breaks stay explicitly unpainted if no crossing fixture exists, and those rows stay open.

## Phase 4: behavior tests

New file `internal/layout/css_behavior_row_rule_test.go`.

- [ ] `row-rule: 3px solid #c00` on a two-row grid paints the centered red line from phase 2.
- [ ] `row-rule-width: 6px` with a solid style paints a 6px stroke. Color defaults to black text, the same check as `TestBehaviorColumnRuleWidthUsedThickness`.
- [ ] `row-rule-color: #00f` paints blue.
- [ ] `row-rule-style: dashed` paints at least two segments. `none` paints zero.
- [ ] `row-rule-visibility-items: around` paints a rule in the gap plus a rule along the top and bottom of the content box. `between` paints only the gap rule.
- [ ] `row-rule-inset-cap: 10px` makes the gap line 20px shorter than the full content width, 10px at each end.
- [ ] `row-rule-break: none` stores `none` and still paints the phase 2 line.
- [ ] `rule: 3px solid #c00` on a multicol box still paints the vertical column rule from `TestBehaviorColumnRulePaintedBetweenColumns`, and on a grid also paints the horizontal row rule.
- [ ] `rule-width`, `rule-style`, and `rule-color` change both axes. A grid test checks the row line. A multicol test checks the column line.
- [ ] `rule-overlap: row-over-column` orders the row line after the column line in one grid that has both gaps. If the grid painter has no column-rule line yet, this row stays open rather than painting a fake column line. Do not build column-rule inset to get there. A column line in a grid is new and belongs in this row only as paint order of two `OpLine`s, using the existing column-rule width, style, and color fields.
- [ ] Bad tokens keep the previous value for each of the 27 names. One table-driven store test is enough if each name is an entry.

**Gate:** the paint rows that phase 3 marked as painted are green. Column-rule tests are unchanged.

## Phase 5: catalog and closure

- [ ] Mark a `row-rule*` or `rule*` name `implemented` only when its paint test exists. Names that are stored but not painted (`row-rule-inset-junction*`, `row-rule-break: intersection`) stay `unsupported` or move to `partial` with the limit written in the catalog. Do not mark them implemented on a store test alone.
- [ ] Do not change the four implemented `column-rule` rows except if `rule` writes their fields and a test still covers them.
- [ ] Do not mark the 11 out-of-scope `column-rule-*` names implemented.
- [ ] Regenerate the matrix property section.
- [ ] Do not edit `documentation/deferred.md` unless the user asks.
- [ ] Do not grow `internal/layout/layout.go`.
- [ ] Run `make test`. Do not run bare `go test ./...`.
- [ ] Run `make golden`.
- [ ] Run `make lint`.
- [ ] Run `make claim-scan`. Do not claim a page PNG or Chrome parity for gap decorations.
- [ ] Run `make catalog-check` and `make matrix-check`.
- [ ] GitHub CI runs only `make test` and `make lint` (`.github/workflows/ci.yml:1`, `.github/workflows/ci.yml:20`, `.github/workflows/ci.yml:26`).

Focused run during the work:

```sh
go test ./internal/layout -run 'TestBehaviorColumnRule|TestBehaviorRowRule|TestBehaviorRowGap' -count=1
```

**Gate:** open until those commands exit 0 on the change that flips catalog rows.

## Dependencies

- Pattern to copy: `applyColumnRuleProps` and `applyColumnRuleShorthand` in `internal/layout/style_multicol_props.go`, and `parseRuleShorthand` in `internal/layout/style_paint_props.go`.
- Line painter to call: `emitBorderLine` in `internal/layout/layout_chrome.go`. Do not copy the segment math.
- Multicol paint to leave alone: `emitColumnRules` in `internal/layout/multicol.go`.
- Gap sizes already computed for grid: `internal/layout/grid.go`. Flex gap is the same `RowGap` field (`internal/layout/style.go:131`).
- New code: `internal/layout/style_gap_rules.go` and `internal/layout/gap_rules.go`.
- The 11 `column-rule-*` inset, break, and visibility names are out of scope even though `rule-inset` and `rule-break` share their grammar.
