## Summary

Fixes the CI failure on the merged backdrop-filter PR (#32). Renames short test locals, adds required blank lines, moves the helper to its cohesive file, and syncs the file-size ledger.

## Motivation / context

- PR #32 merged with `test + lint` red on `feat/backdrop-filter`.
- Failing job: `41401763398` (Lint step). Tests passed; `golangci-lint run ./...` failed with 4 findings, and `size-check` was stale behind it.

## Changes

### Area 1: lint findings

- `internal/layout/layout.go`: blank line before `return` in `hasBackdropFilter` (nlreturn).
- `layout/backdrop_filter_test.go`: rename `op` to `paintOp` (varnamelen, 2 sites) and separate cuddled `if` blocks with blank lines (wsl).

### Area 2: over-limit file growth

- Moved `hasBackdropFilter` from `internal/layout/layout.go` to `internal/layout/blend_group.go`, next to the group type it serves. Net effect: `layout.go` 2138 to 2139 instead of 2145.
- Updated `scripts/file-size-allowlist.txt` to 2139.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None. Same predicate, new home. |
| **Memory** | None. |
| **Behavior / correctness** | None. No logic change; move plus style fixes only. |
| **API / CLI** | None. |
| **Dependencies** | None. |
| **Binary size / build time** | None. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

## Test plan

- [x] `go test -count=1 -p 1 ./layout/ -run 'Backdrop|DisplayBackdrop'`
- [x] `go test -count=1 -p 2 -parallel 2 ./internal/layout/`
- [x] `make lint` (golangci-lint plus size-check)
- [x] `make golden`

### Commands

```sh
go test -count=1 -p 1 ./layout/ -run 'Backdrop|DisplayBackdrop'
go test -count=1 -p 2 -parallel 2 ./internal/layout/
make lint
make golden
```

## Screenshots / sample output

```text
ok  github.com/chinmay-sawant/blinkless/layout  0.005s
ok  github.com/chinmay-sawant/blinkless/internal/layout  1.585s
size-check: clean (1 allowlisted over-limit file).
ok  github.com/chinmay-sawant/blinkless/layout  0.016s (golden TestDisplay)
```

## Related issues

- Relates to #32

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [ ] Filled body committed under `plans/PR/pr-backdrop-filter-ci.md` (kept alongside for `gh --body-file`; not part of this diff)

## Follow-ups (out of scope)

- None.

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
| `.go` | 3 | 21 | 14 |
| `.txt` | 1 | 1 | 1 |
| **Total** | **4** | **22** | **15** |
