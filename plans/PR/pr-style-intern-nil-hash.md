## Summary

Master lint failed because `styleInternHashStringMapCached` mixed a nil custom-property map with `h ^ 0`. staticcheck SA4016 rejects that xor. The nil map now multiplies the running hash by `styleInternPrime`, which is the same value.

---

## Motivation / context

- Plans: none. This is the lint failure on the four commits already on `master` after `v0.1.9` (`c023b8d` through `d3047f9`).
- Issues: see **Related issues**

---

## Changes

### Style intern hash

- `internal/layout/style.go`: nil custom-property maps no longer xor the hash with zero.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None. Same arithmetic. |
| **Memory** | None. |
| **Behavior / correctness** | Unchanged. `h ^ 0` was `h`. |
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

- [x] `gofmt` on `internal/layout/style.go`
- [ ] `make lint` on CI (this is the check that failed on `master`)
- [ ] `make test` on CI

### Commands

```sh
make lint
make test
```

---

## Screenshots / sample output

```
no visual surface changed
```

---

## Related issues

- No tracked issue. Master push CI run 45685111621 failed in the lint job on `internal/layout/style.go:698` (SA4016).

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/PR/pr-<slug>.md` when process-gated

---

## Follow-ups (out of scope)

- The four performance commits after `v0.1.9` are already on `master`. This PR does not replay them.

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
|-----------|-------|------------|-----------|
| `.go` | 1 | 1 | 1 |
| `.md` | 1 | 108 | 0 |
| **Total** | **2** | **109** | **1** |

---
