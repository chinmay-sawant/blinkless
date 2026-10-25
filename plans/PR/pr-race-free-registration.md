## Summary

Remove two data races that show up when one goroutine parses and styles a document while another lays out or restamps a different one. The readers are now registered at package initialization instead of on every `Parse` and `Apply`, and a restamp no longer writes the shared zero-payload singleton.

## Motivation / context

- Ownframe PR [#42](https://github.com/chinmay-sawant/ownframe/pull/42) runs `make test-race` over every package. Its CI failed with `DATA RACE` reports in `internal/pubstate` and in `internal/layout.resetWindowTransforms`.
- Both races are in this module. Ownframe calls `html.Parse` and `css.Apply` from render setup while layout reads `pubstate.StyledOf`, and it restamps displays that share `emptyExtra`.
- There is no separate issue. This is the dependency fix for the Ownframe CI failure.

## Changes

### Registration at package initialization

- Replace `css.registerStyled` and `html.registerRoot` with package-level `var _ = pubstate.RegisterStyled(...)` / `RegisterRoot(...)` initializers. `Apply` and `Parse` no longer write package state.
- Make the two register functions return a `struct{}` token so the call can initialize a blank variable, and document the init-time, lock-free invariant.
- Keep the readers as plain package variables. There is still no map, so documents remain garbage-collectable, and `Root`/`StyledOf` do not pay an atomic load.

### Immutable `emptyExtra`

- In `resetWindowTransforms`, clear the stored matrix only when the operation owns a detached extra. Ops that still point at `emptyExtra` never carried a transform, and the singleton is the no-transform value by definition.
- Keep the hot path allocation-free: skip the shared pointer instead of calling `setXform`, which would detach and allocate for every untouched op on every restamp.

### Tests

- Add `TestConcurrentApplyAndReads` in `internal/pubstate`. It runs `html.Parse` and `css.Apply` in writer goroutines against `pubstate.Root` and `pubstate.StyledOf` readers. On the pre-fix code it produced 12 race reports.
- Add `TestRestampResetLeavesSharedExtraReadOnly` in `internal/layout`. It resets a window bound to `emptyExtra` while another goroutine reads `Transform()`. On the pre-fix code it produced a race and observed the value change to identity.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | No change to the hot layout path. `Root` and `StyledOf` still do one function-pointer call. `resetWindowTransforms` compares two pointers and skips the shared singleton; it adds no allocation. |
| **Memory** | No new retained state. The tests allocate only inside test code. |
| **Behavior / correctness** | Public behavior is unchanged. Registration happens at the same point relative to callers, before any `Parse`, `Apply`, `Root`, or `StyledOf` can run. The reset leaves the shared zero payload untouched; paint paths already gate on `XformSet`. |
| **API / CLI** | `RegisterRoot` and `RegisterStyled` now return `struct{}`. They are internal, so no public API changes. There is no CLI in this tree. |
| **Dependencies** | None. |
| **Binary size / build time** | No new packages or build targets. Two test files. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | The register functions are internal. Callers that already import the public packages need no change. |

## Test plan

- [x] `make test`
- [x] `make lint`
- [x] `make test-race RACE_PKGS='./...'`
- [x] `go test -race ./internal/pubstate ./css ./html`
- [x] `go test -race ./internal/layout`
- [x] Pre-fix controls fail with the expected race reports

### Commands

```sh
make test
make lint
make test-race RACE_PKGS='./...'
go test -race ./internal/pubstate ./css ./html
go test -race ./internal/layout
```

Focused coverage:

- `TestConcurrentApplyAndReads`
- `TestRestampResetLeavesSharedExtraReadOnly`

## Screenshots / sample output

No visual surface. The pre-fix control produced 12 `DATA RACE` reports from `pubstate` and one from `resetWindowTransforms`; both tests pass after the change.

## Related issues

None. This branch does not close an open issue.

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`).
- [x] `enhancement` label applied.
- [x] Related issues reviewed; no open issue matches this branch.
- [x] Filled body committed under `plans/PR/pr-race-free-registration.md`.

## Follow-ups (out of scope)

- The init-time rule is enforced by comments, not by the type system. A later `RegisterRoot` or `RegisterStyled` call outside package initialization would reintroduce the race. Moving the readers to `atomic.Pointer` would make late registration safe at the cost of an atomic load on every read.
- Architecture doc line citations in `documentation/architecture/02-library-api.md` for `internal/pubstate/state.go` and `css/css.go` are a few lines off after this change. No gate checks them.
- `emptyExtra` is the zero extra, so `Transform()` on an op that never set `XformSet` returns the zero matrix while the method comment says identity. Every paint path gates on `XformSet`, so painting is unchanged.

## Reviewer checklist

- [ ] Behavior matches summary and test plan.
- [ ] No unrelated changes in diff.
- [ ] Public API / CLI changes documented.
- [ ] New rules have fixture coverage when applicable.
- [ ] PR has assignee and labels.
- [ ] Related issues use correct Closes/Relates keywords.
- [ ] No secrets or generated artifacts committed.
- [ ] Diff-stat-by-extension table pasted at the bottom.

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 6 | 257 | 44 |
| **Total** | **6** | **257** | **44** |
