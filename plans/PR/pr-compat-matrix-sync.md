## Summary

Sync the compatibility matrix prose with `testdata/css/catalog/properties.json` for `background-image`, `display`, and `overflow`, and record the implemented mask and mask-border statuses. No engine code changed.

## Motivation / context

- Plans: `plans/PR/pr-compat-matrix-sync.md`
- Issues: none. This syncs the two dirty files found on `master` (`9d2ba1c`) so the matrix and catalog agree.

## Changes

### Compatibility matrix

- Added `input`, `textarea`, `select`, `button` form-control row and inline `svg` row in `documentation/compatibility-matrix.md`.
- Updated value lists: `background-image` adds gradient forms, `display` adds `flex`, `inline-flex`, `grid`, `inline-grid`, `overflow` adds `clip`.
- Moved `mask-border*` and `-webkit-mask-box-image*` rows from Partial to Implemented, and narrowed the unsupported mask list from seven names to four.
- Rewrote section 5.3.2 alias counts from 34 Unsupported to 10 Unsupported plus 24 mapped onto implemented bases.

### Catalog

- `testdata/css/catalog/properties.json`: same three value-list updates for `background-image`, `display`, and `overflow`. No status field changed in this diff.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None. Documentation and catalog values only. |
| **Memory** | None. |
| **Behavior / correctness** | No engine behavior changed. Docs now match the catalog. |
| **API / CLI** | No API or CLI change. |
| **Dependencies** | None. |
| **Binary size / build time** | None. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

## Test plan

- [x] `make claim-scan`
- [x] `make matrix-check`
- [x] `go test ./internal/layout -run TestCascade|TestDisplay|TestBackground|TestOverflow -count=1`
- [ ] `make test` (not run: no Go code changed; targeted layout tests pass)
- [ ] `make lint` (not run: docs and JSON only)
- [ ] `make build` (not run: no build surface changed)
- [ ] `make golden` (not run: no layout or fixture change)
- [ ] `make catalog-check` (fails both before and after this change on unresolvable OwnFrame cases for animation rows, unrelated to this diff)

### Commands

```sh
make claim-scan
make matrix-check
go test ./internal/layout -run 'TestCascade|TestDisplay|TestBackground|TestOverflow' -count=1
bash scripts/pr-diff-stat.sh master
```

## Screenshots / sample output

```text
claim-scan: clean
matrix-sync: ok (section 2 matches python3 scripts/css-catalog-map.py --matrix)
ok github.com/chinmay-sawant/blinkless/internal/layout 0.041s
catalog-check: pre-existing fail on OwnFrame animation cases, same on clean tree
```

No visual surface changed.

## Related issues

- None. Sync of local dirty files on `master`.

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`documentation`)
- [x] Related issues filled with real ticket IDs (none apply)
- [x] Filled body committed under `plans/PR/pr-compat-matrix-sync.md`

## Follow-ups (out of scope)

- The pre-existing `make catalog-check` OwnFrame failures for animation rows stay open.

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
| `.json` | 1 | 3 | 3 |
| `.md` | 2 | 131 | 29 |
| **Total** | **3** | **134** | **32** |
