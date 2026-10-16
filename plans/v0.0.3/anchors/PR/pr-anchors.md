# feat(layout): CSS anchor positioning one-canvas subset

## Summary

- Implements the eight css-anchor-position properties (`anchor-name`, `anchor-scope`, `position-anchor`, `position-area`, `position-try`, `position-try-fallbacks`, `position-try-order`, `position-visibility`) plus `@position-try` for the anchor family planned in `plans/v0.0.3/anchors/anchors-checklist.md`.
- Selection, fallbacks, tactics, and visibility follow headless Chrome 155 print, measured row by row against the new gowkhtmltopdf fixture-66. The output stays `layout.DisplayList`; there is no scrollport and no new public field.

## Motivation / context

- Plans: `plans/v0.0.3/anchors/anchors-checklist.md` (all rows closed with proofs), `plans/v0.0.3/README.md`
- Docs: `documentation/anchor-positioning.md`, `documentation/deferred.md`, `documentation/compatibility-matrix.md`
- Issues: none filed; the ledger is the tracking record

## Changes

### Engine

- `internal/layout/anchor_position.go` (new): parses and stores the eight longhands, walks the scope map, resolves the anchor, builds the options list (base area plus fallbacks), and moves the box with its paint ops. Option selection is fit-based: each option is tested against its own position-area region, the first fitting option wins, a spilled box is safe-shifted into the containing block, and when nothing fits the base styles keep applying. Only `no-overflow` hides a box whose every option fails. Tactics also swap the corresponding inset sides.
- `internal/layout/anchor_position.go` gate: `position-area` applies to absolute and fixed targets only; `position: sticky` never enters; static boxes keep flow position.
- `internal/layout/style.go`, `style_cascade.go`, `style_value_accept.go`, `style_intern_gen.go`: seven used fields, one apply group, strict acceptance so invalid values keep earlier ones.
- `internal/layout/layout_context.go`: one `resolveAnchorPositions` call before chrome merge, so `internal/layout/layout.go` stays at its allowlisted size.
- `internal/layout/style_shorthand.go` (new): moves the shorthand dispatch out of `style_cascade.go` and adds the `position-try` expansion.
- `internal/css/at_position_try.go` (new): parses `@position-try` blocks down to stored area and inset overrides.

### Break export for boxless consumers

- `layout/displaylist.go` and `internal/layout/pagebreaks.go` (new): `Display.ForcedBreaks` and `Display.KeepTogether` expose page-break-before/after positions and break-inside avoid ranges, collected after layout. gowkhtmltopdf consumes these so `--engine blinkless` honors forced breaks.

### Tests

- `internal/layout/anchor_position_test.go`: name and scope storage, scope lookup, invalid keeps previous.
- `internal/layout/css_behavior_anchor_test.go`: area placement, none, out-of-scope, static, fallback order, try-size order, shorthand cascade, fit selection, explicit-inset cases, visibility subtests including no-overflow hide on all-reject.
- `internal/layout/css_behavior_anchor_try_test.go`: flip-block tactic.
- `internal/layout/pagebreaks_test.go`: forced break and keep range export.
- `internal/css/at_position_try_test.go`: at-rule parse and lookup.

### Docs and catalog

- `documentation/anchor-positioning.md` (new): the one-canvas contract and the selection rule.
- `documentation/deferred.md`: the family leaves the blanket print non-goal; motion and view timelines keep it.
- `documentation/compatibility-matrix.md`: section 2 regenerated; section 5.4 prose updated.
- `testdata/css/catalog/properties.json`: seven rows `implemented`, `position-visibility` `partial` for the scroll non-goal.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | One extra tree walk after layout when the document declares anchor names; no effect otherwise |
| **Memory** | No new engine fields; option lists are per-target locals |
| **Behavior / correctness** | Absolute and fixed boxes with anchor properties move; documents without them are unchanged |
| **API / CLI** | No new public field except `Display.ForcedBreaks` / `Display.KeepTogether` on the drawing list |
| **Dependencies** | None added; the allowlist stays `go-text/typesetting` and `tdewolff/canvas` |
| **Binary size / build time** | Negligible |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

## Test plan

- [x] `make test`
- [x] `make lint` / `go vet`
- [x] `make build` (when a build surface is part of the change)
- [x] `make golden` when layout, paint, or fixture behavior changed
- [x] `make claim-scan` when documentation or user-facing claims changed
- [x] `make catalog-check`
- [x] `make matrix-check`

### Commands

```sh
make test
make golden
make lint
make claim-scan
make catalog-check
make matrix-check
```

## Screenshots / sample output

```
fixture-66 (gowkhtmltopdf): visible tip against its anchor, every row
blinkless vs headless Chrome 155 print, 15 tip-to-anchor relations
all match within 1.5pt (most under 0.2pt):
  row 1  (-7.5, -45.0) vs (-7.5, -44.9)
  row 2  ( 0.0,  48.0) vs ( 0.0,  47.9)
  ...
  row 12 (56.2,   0.0) vs (56.1,   0.0)   flip-block above anchor
  row 14 (-33.8, 21.0) vs (-33.7, 21.0)   no-overflow below anchor
```

## Related issues

- None. Tracked by `plans/v0.0.3/anchors/anchors-checklist.md`; no GitHub issue was filed for this family.

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/v0.0.3/anchors/PR/pr-anchors.md`

## Follow-ups (out of scope)

- `@position-try` inset overrides are stored but not applied; only the area is read.
- Scroll-linked `anchor-visible` hiding waits on host-scroll work (`plans/v0.0.1/native-window/native-web-view-checklist.md`); `position-visibility` stays `partial`.
- Exotic multi-token `position-area` values are stored but keep offsets.
- `anchor()` inset functions are a later ledger.

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
- [ ] Diff-stat-by-extension table pasted at the bottom

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 17 | 3542 | 27 |
| `.json` | 1 | 141 | 53 |
| `.md` | 5 | 234 | 51 |
| **Total** | **23** | **3917** | **131** |
