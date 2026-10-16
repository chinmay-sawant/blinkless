## Summary

Gap rules now paint junction-aware segments on both axes, with `none`/`normal`/`intersection` breaks, cap and junction insets, overlap-join geometry, and column-rule cap insets in multicol. Mask acceptance catches up with Chrome 155 for repeat pairs, SVG boxes, and 1/2/4-value positions, while the `-webkit-` family keeps Chrome's narrower legacy grammar. The catalog flips 34 rows to implemented, and the compatibility matrix, phase ledgers, frontend data, and built site are regenerated to match.

---

## Motivation / context

- Plans: `plans/v0.0.3/row-rules/row-rules-checklist.md`, `plans/v0.0.3/mask/mask-checklist.md`
- Issues: see **Related issues**

The v0.0.3 row-rules track deferred junction handling. Painted rules either stopped short of crossings or overlapped them, and only plain lengths were supported for insets. The mask track kept a narrower grammar than Chrome for repeats, boxes, and positions. This change closes both ledgers except for the divergences listed under Follow-ups.

---

## Changes

### Gap rules (junction-aware painting and insets)

- Reworked segment painting in `internal/layout/gap_rules.go`: row and column rules split at crossings on both axes and join with overlap geometry.
- `row-rule-break` and `column-rule-break` accept `none`, `normal`, and `intersection`.
- Added cap and junction insets. Shorthands: `row-rule-inset`, `row-rule-inset-cap`, `row-rule-inset-junction`, `row-rule-inset-start`, `row-rule-inset-end`, plus the cap/junction start/end longhands. Same family for `column-rule-inset-*` in `internal/layout/style_multicol_props.go`.
- Percentage insets resolve against the crossing gap at junctions and are no-ops at caps. Negative lengths are stored and applied (`ruleInset` carries value, percent flag, and overlap-join flag).
- `ResolvedStyle` gains `ColumnRuleBreak` and the four `ColumnRuleInset*` fields; style interning handles the new fields and the `ruleInset` payload (`internal/layout/style_intern_gen.go`).
- Grid painting passes placed items into gap rule painting (`internal/layout/grid.go:123`). Multicol column rules honor cap insets (`internal/layout/multicol.go`).
- Tests: `internal/layout/css_behavior_gap_break_test.go` (new), `internal/layout/style_gap_rules_test.go`, `internal/layout/multicol_test.go`.

### Mask acceptance parity with Chrome 155

- `internal/layout/style_mask_accept.go` accepts repeat pairs such as `space repeat` and `round`, SVG boxes (`fill-box`, `stroke-box`, `view-box`) on unprefixed `mask-origin` and `mask-clip`, and 1/2/4-value positions. Generic 3-value position forms (the `background-position` style grammar) are rejected for masks.
- The `mask` shorthand consumes position tokens greedily, with repeats and boxes in Chrome's order.
- The `-webkit-mask` family keeps Chrome's narrower legacy grammar through `webkitMaskLegacyValueAccepted` (`internal/layout/style_value_accept.go`), so legacy shorthand behavior does not widen.
- Tests: `internal/layout/css_behavior_mask_test.go`, `internal/layout/style_value_accept_test.go`.

### Catalog, documentation, plans, and built site

- `testdata/css/catalog/properties.json`: 34 rows flip to implemented (8 row-rule, 10 column-rule, 16 mask). Counts move from 412/27/338/8 to 446/3/328/8 of 785 (implemented/partial/unsupported/intentionally ignored).
- `documentation/compatibility-matrix.md` and `documentation/deferred.md` regenerated.
- `plans/v0.0.3/row-rules/row-rules-checklist.md` and `plans/v0.0.3/mask/mask-checklist.md` rows close on passing gates.
- `frontend/src/data/catalog.json` regenerated; `frontend/src/data/specimen.json` stayed byte-identical.
- Built site regenerated: `docs/index.html`, and the hashed bundle is now `docs/assets/index-DWrCnspY.js` (renamed from `index-DE0Lmo0Y.js`).

### Sibling repo fixture (gowkhtmltopdf)

- Fixture 65 is updated and its PDFs regenerated in `gowkhtmltopdf`, committed there as `9dc713c` on `feat/blinkless-implementation`. This PR is blinkless-only.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Junction splitting runs only for containers that declare gap rules. Mask changes are parser-time. No default-path cost. |
| **Memory** | Small. New `ruleInset` payload per rule endpoint and the four new column-rule inset fields on `ResolvedStyle`. |
| **Behavior / correctness** | Row and column rules join at crossings and honor breaks and insets. Multicol column rules honor cap insets. Mask acceptance matches Chrome 155, and the catalog, matrix, and site match the code. |
| **API / CLI** | Additive at most. No new exported API in the public `layout` package; `ResolvedStyle` in `internal/layout` gains fields. No CLI exists. |
| **Dependencies** | No new dependencies. The allowlist is unchanged. |
| **Binary size / build time** | Modest layout code and test growth. `size-check` passes with the one allowlisted over-limit file unchanged. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] `make test` exited 0 (log: `/tmp/opencode/gates3/test.log`)
- [x] `make golden` exited 0, `TestDisplay` in `./layout` (log: `/tmp/opencode/gates3/golden.log`)
- [x] `make lint` exited 0, chains `size-check` (log: `/tmp/opencode/gates3/lint.log`)
- [x] `make claim-scan` exited 0 (log: `/tmp/opencode/gates3/claim-scan.log`)
- [x] `make catalog-check` exited 0, 446/3/328/8 of 785 (log: `/tmp/opencode/gates3/catalog-check.log`)
- [x] `make matrix-check` exited 0, matrix section 2 matches the catalog map (log: `/tmp/opencode/gates3/matrix-check.log`)
- [x] Uncached `go test ./internal/layout -count=1` and `go test ./layout -count=1` pass on the same tree

### Commands

```sh
make test
make golden
make lint
make claim-scan
make catalog-check
make matrix-check
```

---

## Screenshots / sample output

```
$ make test            # /tmp/opencode/gates3/test.log
ok  	github.com/chinmay-sawant/blinkless/internal/layout	0.599s
ok  	github.com/chinmay-sawant/blinkless/layout	0.034s

$ make golden          # /tmp/opencode/gates3/golden.log
ok  	github.com/chinmay-sawant/blinkless/layout	0.018s
```

---

## Related issues

- None. No open GitHub issue matches this work; `gh issue list --state open --limit 40` returns nothing.
- Relates to the `plans/v0.0.3/` roadmaps for row rules and masking.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`enhancement`, `documentation`)
- [x] Related issues filled. There is no matching ticket.
- [x] Filled body committed under `plans/PR/pr-gap-rule-junctions-mask-parity.md`

---

## Follow-ups (out of scope)

- Pin the four extra mask strings in tests so accepted grammar cannot drift silently.
- Remove the dangling doc comment at the end of `internal/layout/multicol_test.go`.
- Known divergences from Chrome stay documented and unfixed here: flex intersection junctions paint nothing; `row-rule-visibility-items: normal` is treated as `between`; flex column rules paint nothing; decorations paint after items; Chrome bounds column-rule extents to the grid border box while blinkless uses the content box.

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
- [ ] Diff-stat-by-extension table pasted at the bottom

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 14 | 1885 | 166 |
| `.html` | 1 | 1 | 1 |
| `.js` | 1 | 17 | 17 |
| `.json` | 2 | 456 | 152 |
| `.md` | 5 | 313 | 113 |
| **Total** | **23** | **2672** | **449** |
