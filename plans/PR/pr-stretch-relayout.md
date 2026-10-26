## Summary

Lets `RelayoutSubtree` patch a flex-stretched target by building it with the block size its containing block assigned. Hover patches on the orders blotter previously returned `errSubtreeResized` and fell back to a full page layout; they now apply, cutting a hover from about 100ms to about 11ms.

## Motivation / context

- Plans: engine follow-up from the ownframe trade-terminal performance work.
- Issues: no linked issue

## Changes

### `internal/layout/relayout_subtree.go`

- `relayoutBuild` builds through a style override (`relayoutBlockSizeOverride`) instead of the bare fresh style when the containing block assigned the target's block size.
- `relayoutAssignedBlockSize` reads the retained assignment fingerprint (`Height == MaxHeight` with percentages cleared) plus the containing block.
- `relayoutFlexAssignsBlockSize` qualifies row flex (`flexItemCrossStretchNode`) and column flex (`relayoutFlexColumnFilled`), excluding vertical writing modes.
- `relayoutFlexColumnFilled` requires a definite column that consumes its content height with no leftover free space and no wrap, so a content-sized column still rejects a size change.
- The override routes through the engine's own `forceFlexItemCrossSize`, so a patched box matches what the full flex pass writes.

### Tests

- `relayout_subtree_flex_test.go` (new): patches a flex-grown column row and a cross-stretched row, byte-compares patched ops against a full relayout, and rejects a content height change for block, indefinite column, and definite column with free space.
- `relayout_subtree_test.go`, `relayout_subtree_grid_test.go`: share the `relayoutOptionsForCSS` helper.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Orders blotter hover 99.9ms to 11.2ms average (8.9x), median 99.7 to 11.2, p90 105.5 to 13.1. All measured hovers changed 21 operations, so the fast path is a scoped patch, not a skipped event. |
| **Memory** | No change. |
| **Behavior / correctness** | Patched output is byte-equal to a full relayout for the stretched cases. Content-sized targets keep the resize guard. |
| **API / CLI** | No public API change; the new helpers are unexported. |
| **Dependencies** | None. |
| **Binary size / build time** | No change. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

## Test plan

- [x] `go vet -p 1 ./internal/layout/ ./layout/`
- [x] `go test -p 1 -count=1 ./internal/layout/ ./layout/`
- [x] `go test -p 1 -count=1 ./internal/css/ ./internal/svg/`
- [ ] Full `make lint` / `make test` gate (CI)

### Commands

```sh
go vet -p 1 ./internal/layout/ ./layout/
go test -p 1 -count=1 ./internal/layout/ ./layout/ ./internal/css/ ./internal/svg/
```

## Screenshots / sample output

```
orders hover, 1600x900, 30 measured hovers alternating ord-row-0/ord-row-1
before: avg 99.88ms median 99.66 p90 105.46
after:  avg 11.18ms median 11.24 p90 13.05
```

## Related issues

- Relates to ownframe PR #46

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/PR/pr-<slug>.md` when process-gated

## Follow-ups (out of scope)

- Splice-aware display assembly after a patch (reuse `OpOwners`, `Order`, `Boxes` outside the changed window) would cut the remaining watchlist hover cost.

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] No new root aliases, synonym methods, or example `internal/` imports
- [ ] New exported methods return named `ownframe` types (not `host.Stats` / `*state.Store` unless re-exported here)
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
- [ ] Diff-stat-by-extension table pasted at the bottom

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 4 | 617 | 16 |
| **Total** | **4** | **617** | **16** |
