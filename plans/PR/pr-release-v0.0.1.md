## Summary

Names the GitHub Release tag `v0.0.1` in the release note. The note previously said `master` had no tag. Creating the release makes that sentence false, so the published notes should not say it.

## Motivation / context

- Plans: `plans/v0.0.1/release-note/v0.0.1.md`
- Issues: no matching GitHub issue exists.

## Changes

- In `plans/v0.0.1/release-note/v0.0.1.md` and `README.md`, say the release tag is `v0.0.1`.
- Leave the binding stamps at `0.2.6` and leave `CHANGELOG.md` unpromoted.

## Impact

| Area | Impact |
|------|--------|
| Performance | None. |
| Memory | None. |
| Behavior / correctness | None. Wording in the release note only. |
| API / CLI | None. |
| Dependencies | None. |
| Binary size / build time | None. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | No runtime change. |

## Test plan

- [x] Diff is two markdown sentences under `plans/v0.0.1/release-note/`.
- [ ] `make test` not run. No Go files change.
- [ ] `make golden` not run. Layout is untouched.
- [ ] `make claim-scan` not run. `documentation/` is untouched.

### Commands

```sh
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
- [x] Filled body committed under `plans/PR/pr-release-v0.0.1.md`

## Follow-ups (out of scope)

- The release workflow compares a `v*` tag with `BLINKLESS_VERSION` and the Python package version. Both still say `0.2.6`, so that workflow fails on `v0.0.1` until those stamps move.

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
| `.md` | 3 | 84 | 2 |
| **Total** | **3** | **84** | **2** |
