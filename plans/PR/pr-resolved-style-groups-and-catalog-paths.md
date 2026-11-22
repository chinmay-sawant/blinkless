## Summary

Reduce the per-element `ResolvedStyle` record from 5,872 bytes to 512 bytes by storing less common CSS values in optional groups. Also update the downstream catalog references and runner so the four moved OwnFrame view-transition tests are actually executed.

## Motivation / context

- Plans: no plan file applies to these two completed changes.
- Issues: none identified.

## Changes

### Resolved style storage

- Move uncommon CSS properties into lazy groups and share immutable initial-value groups.
- Update style access, inheritance, equality, hashing, and interning to use the grouped values.
- Add size and copy-semantics tests for the 512-byte core.

### Downstream catalog checks

- Point the four view-transition catalog entries at `internal/viewtransition/catalog_test.go`.
- Run integration tests by the Go package named in each catalog reference.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Mixed benchmark results. Across eight interleaved runs, invoice layout was 3.2% slower, medium full apply was 1.0% slower, and medium relayout was 6.7% faster. |
| **Memory** | `ResolvedStyle` shrank from 5,872 to 512 bytes. Shared initial groups avoid storing repeated defaults per element. |
| **Behavior / correctness** | CSS initial values and style interning remain covered by tests. The moved catalog tests now run in their actual package. |
| **API / CLI** | No public API or CLI changes. |
| **Dependencies** | No dependency changes. |
| **Binary size / build time** | Not measured. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | No migration required. |

## Test plan

- [x] `make test`
- [x] `make lint`
- [x] `make golden`
- [x] `make claim-scan`
- [x] `make catalog-check`
- [x] `make matrix-check`
- [x] `GOFLAGS=-v xvfb-run -a make catalog-ownframe-check` ran all four downstream view-transition cases.

### Commands

```sh
make test
make golden
make lint
make claim-scan
make catalog-check
make matrix-check
GOFLAGS=-v xvfb-run -a make catalog-ownframe-check
```

## Screenshots / sample output

No visual surface changed. The generated tests and layout goldens passed.

## Related issues

- None identified.

## PR metadata checklist (author)

- [x] Self-assigned (`@me`)
- [x] Labels applied
- [x] Related issues checked. No related issue IDs apply.
- [x] Filled body committed under `plans/PR/pr-resolved-style-groups-and-catalog-paths.md`.

## Follow-ups (out of scope)

- None.

## Reviewer checklist

- [x] Behavior matches summary and test plan
- [x] No unrelated changes in diff
- [x] Public API / CLI changes documented
- [x] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [x] No issue references need Closes/Relates keywords.
- [x] No secrets or generated artifacts committed
- [x] Diff-stat-by-extension table pasted at the bottom

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 239 | 7712 | 6374 |
| `.json` | 1 | 8 | 8 |
| `.md` | 1 | 100 | 0 |
| `.py` | 1 | 15 | 8 |
| `.txt` | 1 | 0 | 1 |
| **Total** | **243** | **7835** | **6391** |
