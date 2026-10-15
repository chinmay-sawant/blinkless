## Summary

Adds the v0.0.1 release note for the 54 commits on `master`, from the gowkhtmltopdf import through the CSS mask merge. The note records the drawing-list cut, the catalog counts, and the version strings that still disagree. It does not promote `CHANGELOG.md`, retag the bindings, or create a git tag.

## Motivation / context

- Plans: `plans/v0.0.1/release-note/v0.0.1.md`
- Issues: no matching GitHub issue exists. The note was requested for the renderer cut named in `RELEASE.md`.

## Changes

### Release note

- Add `plans/v0.0.1/release-note/v0.0.1.md` for the product a caller gets at `c085b34`.
- Add `plans/v0.0.1/release-note/commits.md` with one row per commit. The row count matches `git log master`.
- Add `plans/v0.0.1/release-note/README.md` and link the note from `plans/README.md`.

### Left unchanged

- `CHANGELOG.md` still has `## Unreleased` and the old 0.2.x sections.
- `bindings/python/pyproject.toml` and `BLINKLESS_VERSION` still say `0.2.6`.
- The Makefile build stamps stay `0.0.1`. No `v*` tag is created.

## Impact

| Area | Impact |
|------|--------|
| Performance | None. Markdown only. |
| Memory | None. |
| Behavior / correctness | None. No engine, catalog, or user doc file changes. |
| API / CLI | None. |
| Dependencies | None. |
| Binary size / build time | None. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | No runtime change. |

## Test plan

- [x] Diff is markdown under `plans/` only. Confirmed with `git diff --stat` before commit.
- [ ] `make test` not run. No Go files change.
- [ ] `make lint` not run. No Go files change.
- [ ] `make golden` not run. Layout and paint are untouched.
- [ ] `make claim-scan` not run. `doc.go`, `README.md`, and `documentation/` are untouched.

### Commands

```sh
git diff --stat master...HEAD
bash scripts/pr-diff-stat.sh master
```

## Screenshots / sample output

```
no visual surface changed
```

## Related issues

- None. No GitHub issue is linked.

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled. There is no ticket ID.
- [x] Filled body committed under `plans/PR/pr-release-v001.md`

## Follow-ups (out of scope)

- Promote `CHANGELOG.md` and decide whether the binding stamps move from `0.2.6` to `0.0.1`.
- Fix the stale catalog totals in the header of `documentation/compatibility-matrix.md`.
- Tag `v0.0.1` only after those stamps agree.

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
| `.md` | 0 | 0 | 0 |
| **Total** | **0** | **0** | **0** |
