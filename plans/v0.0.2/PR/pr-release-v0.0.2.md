## Summary

Promotes 0.0.2, the security release after the published `v0.0.1` tag. The note records the resource caps, the loader and binding checks, and the removal of the Python package. `Makefile` binding stamps stay `0.0.1`.

## Motivation / context

- Plans: `plans/v0.0.2/release-note/v0.0.2.md`
- Issues: no GitHub issue. The controls are `plans/v0.0.1/security/` ledgers 01 through 09.

`v0.0.1` is already published at `108c2c9`. This cut is the commits after that tag. The tag is `v0.0.2`, not `v0.0.0`, because a lower number would sort behind the release that already exists. The folder is `plans/v0.0.2`.

## Changes

### Release record

- Add `plans/v0.0.2/` with the note, the commit list, the GitHub Release body, and this PR body.
- Move the old `## Unreleased` renderer text in `CHANGELOG.md` into `## 0.0.1 (2026-10-15)`.
- Add `## 0.0.2 (2026-10-15)` for the security caps and the Python removal. Leave an empty `## Unreleased`.
- Point `RELEASE.md` and `plans/README.md` at 0.0.2.

### User docs

- `README.md`, `documentation/overview.md`, and `documentation/library-api.md` name the 32 MiB stylesheet cap, the 1024 nesting cap, the uncapped `html.Parse`, and the SVG text strip.

### Left unchanged

- `BINDINGS_VERSION` and `WASM_VERSION` stay `0.0.1`.
- `BLINKLESS_VERSION` in the C header stays `0.2.6`.
- No engine code changes in this branch. The security code is already on `master` from pull request #6.

## Impact

| Area | Impact |
|------|--------|
| Performance | None in this branch. The caps already landed in #6. |
| Memory | None in this branch. |
| Behavior / correctness | None in this branch. Documentation now matches the caps. |
| API / CLI | No new call. |
| Dependencies | None. |
| Binary size / build time | None. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None in this branch | The breaking caps and the Python removal already merged. |

## Test plan

- [x] `go build ./...` exit 0
- [x] `make test-quick` exit 0
- [x] `make golden` exit 0 (`TestDisplay`)
- [x] `make lint` exit 0. `layout.go` is 2142 lines and still the one allowlisted file.
- [x] `make claim-scan` exit 0

### Commands

```sh
go build ./...
make test-quick
make golden
make lint
make claim-scan
bash scripts/pr-diff-stat.sh master
```

## Screenshots / sample output

```
no visual surface changed
```

## Related issues

- None. No GitHub issue is linked.
- Pull request #6 is the security implementation. Pull request #9 removed the Python package.

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled. There is no ticket ID.
- [x] Filled body committed under `plans/v0.0.2/PR/pr-release-v0.0.2.md`

## Follow-ups (out of scope)

- `html.Parse` still has no byte cap.
- Binding stamps stay `0.0.1` until a later cut moves them.

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
| placeholder | 0 | 0 | 0 |
