## Summary

Add `Op.XformValue`, a pointer-receiver accessor that returns the stored 2D transform without copying the 256-byte operation. Ownframe's replay scan reads a text op's matrix for every operation; with `Transform`'s value receiver the compiler copied the whole op first, which dominated the scan profile.

## Motivation / context

- Ownframe PR [#42](https://github.com/chinmay-sawant/ownframe/pull/42) CI failed the performance gate: `BenchmarkDrawRectScan/dirty` and `/full` ran about 1.86x the `master` baseline.
- Profiling showed `textProjection` spent most of its time in the value-receiver copy inside `Transform`, not in the matrix comparison.
- Reading the promoted `Xform` field directly is not an option for the caller: the embedded payload is nil on an op that never attached one, and binding the shared empty extra just to guard the read changes `Transform` from identity to zero. Ownframe's dirty diff compares `Transform`, so that bind would mark every text run changed on every redraw.
- A pointer-receiver accessor avoids both the copy and the state change.

## Changes

### `internal/layout/op_read.go`

- Add `func (op *Op) XformValue() Matrix2D`. A nil op or nil payload returns the zero matrix; otherwise it returns `op.opExtra.Xform` without allocating or binding the shared extra.
- Leave `Transform` unchanged for existing callers. `XformValue` is additive and does not change the drawing list or the WASM schema.

### `internal/layout/op_read_xform_test.go`

- Cover a nil payload (zero matrix from `XformValue`, identity from `Transform`), a set transform, and an op bound to `emptyExtra`. The test also checks that `XformValue` leaves the shared pointer in place.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Removes one 256-byte value copy per text op from the caller's scan. No other path changes. |
| **Memory** | No new state. The accessor allocates nothing. |
| **Behavior / correctness** | Returns the same matrix `Transform` returns for an op with a payload. For a nil payload it returns zero, which callers already treat as no transform. |
| **API / CLI** | Additive exported method. No CLI in this tree. |
| **Dependencies** | None. |
| **Binary size / build time** | One small method and one test file. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | Existing `Transform` callers are unchanged. New callers that need the raw matrix can use `XformValue`. |

## Test plan

- [x] `go test ./internal/layout -run TestXformValueReadsWithoutBinding -count=1`
- [x] `gofmt -l internal/layout/op_read.go internal/layout/op_read_xform_test.go`
- [x] `make lint`
- [ ] `make test` (CI)

### Commands

```sh
go test ./internal/layout -run TestXformValueReadsWithoutBinding -count=1
make test
make lint
```

## Screenshots / sample output

No visual surface. The focused test compiles and passes; CI runs the full package and lint.

## Related issues

None. This branch does not close an open issue.

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`).
- [x] `enhancement` label applied.
- [x] Related issues reviewed; no open issue matches this branch.
- [x] Filled body committed under `plans/PR/pr-copyless-xform-value.md`.

## Follow-ups (out of scope)

- The accessor returns the zero matrix for a nil payload, while `Transform` documents identity. The difference is intentional and documented. Callers that need identity can keep using `Transform`.
- Ownframe must switch `internal/replay/projected_text.go` to `XformValue` after the pin bump. That is a separate commit in the ownframe PR.

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
| `.go` | 2 | 58 | 0 |
| **Total** | **2** | **58** | **0** |
