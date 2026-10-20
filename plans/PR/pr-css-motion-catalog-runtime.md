# feat(css-catalog): record runtime evidence for CSS motion

## Summary

The CSS catalog now records where motion behavior runs and links 43 motion cases to tests in OwnFrame. Blinkless adds each element's parent index and paint operation IDs, plus an owner index for each operation, so a host can find an element's paint without guessing from overlapping rectangles (`layout/layout.go:15-21`, `layout/displaylist.go:163-166`).

## Motivation / context

- The catalog previously tracked Blinkless handlers and tests, but did not require test evidence from the OwnFrame runtime.
- This branch pairs the two repositories in the catalog and makes the scope of each motion row explicit (`testdata/css/catalog/README.md:17-26`).

## Changes

### Display-list ownership

- Add `ParentIndex` and `OperationIDs` to `layout.Box`. `ParentIndex` points to the closest parent element. `OperationIDs` lists paint operations emitted by that element and its children (`layout/layout.go:8-21`).
- Add `OpOwners` to `layout.Display`. Each entry points to the box that owns the operation; operations shared across sibling boxes point to their closest shared parent (`layout/displaylist.go:157-173`).
- Assign IDs to chrome paint operations inserted outside the usual layout path (`internal/layout/layout_chrome.go:539-587`).

### Catalog evidence

- Move the catalog schema to version 2. Motion rows record whether Blinkless exports metadata only or the Blinkless and OwnFrame pair has runtime tests (`testdata/css/catalog/schema.json:5-23`).
- Validate each downstream test function and named case. `make catalog-ownframe-check` checks those references, creates a temporary Go workspace for both repositories, and runs the listed OwnFrame tests against this Blinkless checkout (`scripts/check-css-ownframe-integration.py:16-58`).
- Record all 43 OwnFrame motion tests in the catalog and update the compatibility matrix, feature notes, checklists, and generated site (`testdata/css/catalog/README.md:26-44`).

## Impact

| Area | Impact |
|------|--------|
| Performance | Adds walks over element boxes and paint operations when building a display list. No performance benchmark was run for this change. |
| Memory | Adds an owner index per operation and operation ID lists for element subtrees. |
| Behavior / correctness | Adds ownership metadata to the returned display list. Paint order is unchanged. |
| API / CLI | Adds exported fields to `layout.Box` and `layout.Display`. No CLI changes. |
| Dependencies | No dependency changes. |
| Binary size / build time | No size or build-time measurement was made. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| New exported fields on `layout.Box` and `layout.Display` | Keyed literals keep working. Code that creates these structs with positional fields must be updated for the added fields. |

## Test plan

The branch commits record these gates as passed before this PR. The pull request CI will run `make test` and `make lint` again on the final PR revision.

- [x] `make test`
- [x] `make lint`
- [x] `make build`
- [x] `make golden`
- [x] `make claim-scan`

### Commands

```sh
make test
make lint
make build
make golden
make wasm-test
make claim-scan
make catalog-check
make catalog-ownframe-check
make matrix-check
cd frontend && npm run build
```

Recorded results also include `make wasm-test`, `make catalog-check`, `make catalog-ownframe-check` for 43 motion cases, and `make matrix-check`. The frontend commit records a passing `npm run build`.

## Screenshots / sample output

The documentation site catalog changed. No screenshot is attached. The frontend build passed as recorded in the branch.

## Related issues

- None. The open mask-border issue does not cover this work.

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied: `enhancement`, `documentation`
- [x] Related issues reviewed; no matching issue applies
- [x] Filled body committed under `plans/PR/pr-css-motion-catalog-runtime.md`

## Follow-ups (out of scope)

- The host still supplies time, timeline progress, trigger events, and old and new view-transition snapshots (`testdata/css/catalog/README.md:24`).
- CSS motion that lacks an OwnFrame behavior test remains outside the `blinkless+ownframe` scope.

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
- [ ] Diff-stat-by-extension table pasted at the bottom

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 6 | 300 | 7 |
| `.html` | 1 | 1 | 1 |
| `.js` | 2 | 205 | 205 |
| `.json` | 3 | 673 | 330 |
| `.md` | 7 | 226 | 95 |
| `.py` | 2 | 134 | 6 |
| `.tsx` | 1 | 19 | 9 |
| No extension | 1 | 6 | 1 |
| **Total** | **23** | **1564** | **654** |
