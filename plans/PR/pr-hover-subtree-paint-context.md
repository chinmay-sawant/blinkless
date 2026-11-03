## Summary

Subtree relayouts now retain inherited positioning and stacking state, which keeps fixed sidebar content visible during hover updates. CSS-aware hover scopes also let callers limit layout work to elements whose hover selector match changed.

---

## Motivation / context

- Plans: None.
- Issues: No related issue found.

---

## Changes

### Paint context in subtree relayout

- Seed inherited positioning and stacking state before rebuilding a subtree.
- Preserve viewport-fixed paint flags when rebuilding descendants.
- Permit viewport-fixed elements in subtree patches when no ancestor changes their containing block.

### CSS hover scopes

- Report subtrees containing elements whose hover selector match changes.
- Return the nearest ID-bearing scope, and fall back when no safe scope exists.
- Cover selector changes, inherited subtree paint, generated content, and missing IDs in tests.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Avoids full layouts when a broad container has no hover-dependent selector changes. |
| **Memory** | No direct memory change. |
| **Behavior / correctness** | Keeps fixed sidebar text and stacking order correct after subtree patches. |
| **API / CLI** | Adds `css.Document.HoverChangedScopes`. |
| **Dependencies** | None. |
| **Binary size / build time** | No material change expected. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] Focused CSS hover-scope tests.
- [x] Focused subtree paint-context and viewport-fixed tests.
- [x] `make test TEST_P=1 TEST_PARALLEL=1`.
- [x] `make lint`.
- [x] `make golden`.
- [x] `make claim-scan`.

### Commands

```sh
go test -p 1 ./css -run '^TestHoverChangedScopes' -count=1
go test -p 1 ./internal/layout -run '^TestRelayoutSubtree(InheritsPaintContext|ViewportFixed)$' -count=1
make test TEST_P=1 TEST_PARALLEL=1
make lint
make golden
make claim-scan
```

---

## Screenshots / sample output

No visual surface changed. Native framebuffer coverage for the Loglens sidebar is in the OwnFrame change that consumes this API.

---

## Related issues

- None.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`).
- [x] Labels applied (`bug`).
- [x] Related issues checked; none found.
- [x] Filled body committed under `plans/PR/pr-hover-subtree-paint-context.md`.

---

## Follow-ups (out of scope)

- None.

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan.
- [ ] No unrelated changes in diff.
- [ ] Public CSS method has a Go doc comment.
- [ ] Viewport-fixed subtree cases have fixture coverage.

---

## Diff stat by extension

<!-- Generated with bash scripts/pr-diff-stat.sh master. -->

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 5 | 321 | 7 |
| `.md` | 1 | 117 | 0 |
| **Total** | **6** | **438** | **7** |
