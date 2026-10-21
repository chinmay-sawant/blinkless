## Summary

Sync the committed docs with the selector index, one-pass grid paths, and
element patch work in `6b29cc9` and `56bc604`, and with the frontend catalog
exporter fix in `1c68b28`. Add `documentation/site-data.md` so the catalog and
specimen regeneration rules are written down.

## Motivation / context

- Plans: `plans/PR/pr-docs-sync-recent-commits.md`
- Issues: none. This is a follow-up documentation pass for the three recent
  commits.

## Changes

### CSS architecture

- `documentation/architecture/06-css.md` no longer says the cascade has no
  index. It documents the rightmost-compound buckets in
  `internal/layout/style_rule_index.go`, the candidate order, the
  `::before`/`::after` buckets, and the content gate.
- Added `internal/css/state_pseudos.go` to the file map and documented
  `css.Document.UsesStatePseudo`.
- Refreshed the cascade, class-token, and layer line references that moved
  with the index work.

### Layout architecture

- `documentation/architecture/07-layout.md` fixes stale
  `layout/displaylist.go` and `internal/layout/layout.go` references.
- New "Selector index" and "One-pass grid paths" sections cover the fast
  paths and their fallbacks.
- `RelayoutElement` now points at the function instead of a comment.

### Public API

- `documentation/architecture/02-library-api.md` documents
  `Display.OpOwners`, `Keyframes`, `ForcedBreaks`, `KeepTogether`,
  `Box.ParentIndex`, `OperationIDs`, and the motion metadata fields.
- `documentation/library-api.md` points the bounds rounding at
  `layout/relayout_element.go:208-212`.

### Site data

- New `documentation/site-data.md` documents `scripts/export-frontend-data`:
  inputs, both JSON outputs, the regeneration command, `implementation_scope`
  preservation, ASCII escaping, note truncation, and the clean second-run
  check.
- Linked it from `documentation/README.md` and
  `documentation/architecture/01-entrypoints-cli.md`.

### Changelog

- `CHANGELOG.md` records `1c68b28` under Fixed.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None. Documentation only. |
| **Memory** | None. |
| **Behavior / correctness** | No engine behavior changed. The docs now match HEAD. |
| **API / CLI** | No API or CLI change. |
| **Dependencies** | None. |
| **Binary size / build time** | None. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

## Test plan

- [x] `make claim-scan`
- [ ] `make test` (not run: documentation only, no Go code changed)
- [ ] `make lint` (not run: documentation only, no Go code changed)
- [ ] `make build` (not run: no build surface changed)
- [ ] `make golden` (not run: no layout or fixture change)

### Commands

```sh
make claim-scan
bash scripts/pr-diff-stat.sh master
```

## Screenshots / sample output

`make claim-scan` output:

```text
claim-scan: clean
```

No visual surface changed.

## Related issues

- None. Follow-up documentation sync for `1c68b28`, `6b29cc9`, and `56bc604`.

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`documentation`)
- [x] Related issues filled with real ticket IDs (none apply)
- [x] Filled body committed under `plans/PR/pr-docs-sync-recent-commits.md`

## Follow-ups (out of scope)

- The Makefile `lint-frontend` target still echoes that `frontend/` is not in
  the tree, although it is. That is a Makefile message, not a documentation
  change.

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
| `.md` | 8 | 240 | 65 |
| **Total** | **8** | **240** | **65** |
