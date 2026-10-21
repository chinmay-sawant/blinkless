## Summary

Add retained layout state and `layout.RelayoutElement`, so a host can edit one element and re-place only that subtree, with `html.Document.SetAttribute` and `SetText` for the tree edit. Index selectors by their rightmost compound and build single-column and multi-column auto-row grids in one pass, cutting the CSS preview catalog build and the click rebuild times.

## Motivation / context

- Branch: `chore/selector-index`.
- Plans: no dedicated ledger entry; the work is four focused commits plus a review-fix commit.
- Issues: see **Related issues**.

## Changes

### Element patches

- `layout.Options.RetainState` keeps the resolved styles and the box tree a patch needs. It is off by default, so a host that never patches keeps the cheaper display.
- `layout.RelayoutElement` finds the element by id, re-places its subtree inside the retained placement, and returns an `ElementPatch` with `Ops`, `OpOwners`, `Boxes`, `Old`, `New`, and `Display`.
- `ElementPatch.Display` is the whole patched page. A host adopts it and treats the previous display as stale, because a count change moves every later operation index, the paint order, and the owners.
- `internal/layout.RelayoutSubtree` re-resolves the subtree's styles from the parent, lays the element out in the width its kept box carries, splices the new operations over the old range, shifts the tail, updates the box tree and ancestor ranges, merges the deferred background and border chrome, and re-applies the ancestor clip.
- The element keeps its place and size. A change that moves or resizes it is refused, so the boxes after it stay valid.
- `html.Document.SetAttribute` and `html.Document.SetText` edit one attribute or one text node in place. An attribute keeps the parser's spelling, and the ordered attribute list stays in step with the map.
- Refusals: no retained state, missing id, element without a box, document root, non-block box, ancestor transform or opacity, float, sticky, anchor, named flow region, anchor outside the subtree, `@container`, or a size or position change. An out-of-flow box is accepted when its containing block is inside the subtree, including the fixed-position case.
- The patched display recomputes page breaks, because a style patch can move them.

### Selector index

- The cascade buckets rules by the rightmost id, class, tag, or universal compound, for both the element cascade and the `::before` and `::after` scans.
- Candidate order keeps sheet, rule, and selector order, so equal-order cascade ties do not change.
- Generated-content matching is skipped when no rule declares `content`.
- `css.Document.UsesStatePseudo` lets a host skip a hover, active, or focus redraw when no rule can match that state.
- Class tokens now split on every Unicode whitespace character, not just U+0020.
- `focus-visible` is a shared constant in the CSS package.

### One-pass grids

- Single-column grids build each item once at its final row position. The path uses the resolved track width (`cols[0]`), keeps row gaps and gap rules, applies justify and align-content offsets, and resolves the used height.
- Multi-column auto-row grids build each row's items at the cell widths, take the tallest item as the row height, and stretch the shorter items by extending the border box and regenerating their background and border chrome.
- The single-column path falls back to the two-pass layout for explicit rows or areas, placement, spans, percentage or min/max heights, and non-stretch justify.
- The auto-row path also falls back for flex, multicol, table, subgrid, vertical writing, clipping, out-of-flow boxes, transforms, blend or isolation groups, and nested grids whose rows or align-content need a measure pass.
- Inline splitting grows the item slice geometrically instead of recopying on every split.
- The engine caches the last face key, and soft-hyphen splitting returns early when the text has no soft hyphen.

### Docs, tests, and ledgers

- `documentation/library-api.md` describes the patch API and the host contract.
- `documentation/architecture/02-library-api.md` and `documentation/architecture/07-layout.md` match the code paths and citations.
- `CHANGELOG.md` records the element patches, selector index, and one-pass grids.
- New tests cover the single-column track width, the extra-column fallback, row-rule painting, the relayout containing-block guard, selector order, class lookup, and pseudo separation.
- `scripts/file-size-allowlist.txt` records `internal/layout/layout.go` at 2190 lines.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | CSS property catalog build: about 208 ms to 139 ms. Single-column CSS preview click rebuild: 87.6-90.9 ms to 33.3-36.8 ms. Multi-column rebuild: 33.3-36.8 ms to 26.7-27.8 ms. On a 43-card page at 1908x999, an element patch costs 1.65 ms and 2.66 MB against 25.0 ms and 14.45 MB for a full redraw. |
| **Memory** | Retained state costs 167 KB on a 31-element page, 277 KB on 283 elements, and 454 KB on 703 elements. The style store now starts its backing arrays at eight records and grows to 64, which cuts the retained styles. A patch still copies the page operation list, which is 2.66 MB in the benchmark. |
| **Behavior / correctness** | The patch keeps the element's place and size. A change that would move or resize it is refused. A patched display replaces the previous one. The engine does not lock the retained state, so one display must not be patched from two goroutines at once. |
| **API / CLI** | Adds `layout.Options.RetainState`, `layout.RelayoutElement`, `layout.ElementPatch`, `layout.ErrNoRetainedState`, `html.Document.SetAttribute`, `html.Document.SetText`, and `css.Document.UsesStatePseudo`. No CLI change. |
| **Dependencies** | None. |
| **Binary size / build time** | No CGO and no new module. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | The new API is additive and `RetainState` is off by default. A host that patches must adopt `ElementPatch.Display` and discard the previous display. |

## Test plan

- [x] `make test` (fresh run with `GO_TEST_FLAGS='-count=1'`, 17 packages, exit 0)
- [x] `make lint` (0 issues, size-check clean)
- [x] `make build`
- [x] `make golden`
- [x] `make claim-scan`

### Commands

```sh
make test GO_TEST_FLAGS='-count=1'
make lint
make build
make golden
make claim-scan
```

## Screenshots / sample output

No screenshot surface changed. The layout tests check the drawing list and `make golden` passes.

## Related issues

None. Issue #18 (mask-border) is unrelated to this branch.

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs (none for this branch)
- [x] Filled body committed under `plans/v0.0.3/PR/pr-selector-index.md`

## Follow-ups (out of scope)

- Add tests for the refusal branches: missing context, cancelled context, missing retained state, unknown id, element without a box, document root, non-block box, transform or opacity ancestor, float, sticky, anchor, named flow region, anchor outside the subtree, `@container`, and a size or position change.
- Cover duplicate id handling.
- Cover patching the same display twice and patching a zoomed display.
- Add differential tests that compare the selector index with the old linear walk on generated stylesheets.
- Assert the grid fast-path gates directly instead of only through output.
- Patch support is block-only. Inline and table boxes still take a full relayout.

## Reviewer checklist

- [x] Behavior matches summary and test plan
- [x] No unrelated changes in diff
- [x] Public API / CLI changes documented
- [x] New rules have fixture coverage when applicable
- [x] PR has assignee and labels
- [x] Related issues use correct Closes/Relates keywords (none)
- [x] No secrets or generated artifacts committed
- [x] Diff-stat-by-extension table pasted at the bottom

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 36 | 4779 | 93 |
| `.md` | 5 | 336 | 7 |
| `.txt` | 1 | 1 | 1 |
| **Total** | **42** | **5116** | **101** |
