## Summary

Promote the tree to 0.0.3. CHANGELOG gains a dated 0.0.3 section with corrected catalog counts, RELEASE.md names the feature release, and the release note plus commit ledger land under `plans/v0.0.3/release-note/`. No engine code changes in this PR.

---

## Motivation / context

- Plans: `plans/v0.0.3/README.md`, `plans/v0.0.3/release-note/v0.0.3.md`
- Release skill: `skills/release-note/SKILL.md`
- Issues: see **Related issues**

---

## Changes

### Promote files

- `CHANGELOG.md`: `## Unreleased (after 0.0.2)` becomes an empty `## Unreleased` plus a dated `## 0.0.3 (2026-10-23)` section. Catalog counts corrected to the live catalog (785 rows, 555 implemented, 3 partial, 219 unsupported, 8 intentionally ignored, dated 2026-10-20). The C-binding removal now reads as shipped in `v0.0.3`. The two stale `see Unreleased` pointers now point at 0.0.3.
- `RELEASE.md`: version line names 0.0.3 as the feature release following the 0.0.2 security cut.
- `plans/v0.0.3/release-note/v0.0.3.md`: release note mirroring the v0.0.2 shape (What changed, Removed, Version strings, What still does not ship), covering the 85 commits from `23cc975` through `a0f695b`.
- `plans/v0.0.3/release-note/commits.md`: full commit ledger after `v0.0.2`, in order, merges included.
- `plans/v0.0.3/PR/release-v0.0.3.md`: GitHub release body, identical to the release note.
- `plans/v0.0.3/README.md` and `plans/README.md`: index lines for the release note draft.

### Docs check

- Generic docs (`README.md`, `documentation/*`, `output/README.md`) carry no `unreleased` language. Verified with the skill's `rg` check. No rewrites needed there.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None. No code changes. |
| **Memory** | None. |
| **Behavior / correctness** | None. Docs and ledger only. |
| **API / CLI** | None. |
| **Dependencies** | None. |
| **Binary size / build time** | None. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [ ] `go build ./...`
- [ ] `make test-quick`
- [ ] `make golden`
- [ ] `make lint`
- [ ] `make claim-scan`

### Commands

```sh
go build ./...
make test-quick
make golden
make lint
make claim-scan
```

---

## Screenshots / sample output

No visual surface changed. Docs and ledger only.

---

## Related issues

- None. No open PRs or tracking issues cover this promote step.

---

## PR metadata checklist (author)

- [ ] Self-assigned (`--assignee @me`)
- [ ] Labels applied
- [ ] Related issues filled with real ticket IDs
- [ ] Filled body committed under `plans/v0.0.3/PR/pr-release-0.0.3.md`

---

## Follow-ups (out of scope)

- Tag `v0.0.3` after merge, then `gh release create v0.0.3` with `plans/v0.0.3/PR/release-v0.0.3.md` as notes.
- `documentation/css-showcase.html` still says 389 implemented properties. Stale since v0.0.2. Left alone here.
- The untold pending tasks stay out of this release.

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
| `.md` | 8 | 345 | 11 |
| **Total** | **8** | **345** | **11** |
