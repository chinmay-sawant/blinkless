## Summary

`scripts/export-frontend-data` dropped `implementation_scope` when it regenerated `frontend/src/data/catalog.json`, so the 43 Blinkless+OwnFrame rows lost their badge and count. The exporter now carries the field through and escapes non-ASCII runes so regeneration matches the committed ASCII-only JSON style. The catalog and the built docs site are regenerated.

---

## Motivation / context

- Plans: `plans/PR/pr-export-frontend-data-catalog.md`
- Issues: see **Related issues**

Found while cross-checking `frontend/src/data/catalog.json` against `testdata/css/catalog/properties.json`. The committed catalog was in sync, but running the documented regeneration command would have removed `implementation_scope` from 43 rows and rewritten 43 radius values as raw UTF-8.

---

## Changes

### Exporter

- `scripts/export-frontend-data/main.go` reads `implementation_scope` from the catalog row and writes it on the site entry. The 43 `blinkless+ownframe` rows now survive regeneration.
- `writeJSON` escapes non-ASCII runes as `\uXXXX`. Go's JSON encoder writes UTF-8, while the committed site data is ASCII-only (`\u221e`), so this keeps regeneration diff-free.
- `escapeNonASCII` handles surrogate pairs for code points above U+FFFF.
- Two `//nolint:tagliatelle` directives record that the catalog schema requires snake_case keys.

### Generated data

- `frontend/src/data/catalog.json` regenerated: 558 rows, 555 implemented, 3 partial, 43 scoped.
- The `view-transition-group` note now follows the exporter's 180-byte cap and ends at `::view-transition-* selectors ...`. The previous 192-byte note was a manual exception.
- `docs/` rebuilt with `npm run build`; the hashed asset changes from `index--2mC-OoU.js` to `index-B9_stwEr.js`.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None. |
| **Memory** | None. |
| **Behavior / correctness** | Regeneration preserves the 43 `blinkless+ownframe` rows instead of dropping the field. |
| **API / CLI** | `go run ./scripts/export-frontend-data` is unchanged. |
| **Dependencies** | None. |
| **Binary size / build time** | No meaningful change. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] `go build ./...`
- [x] `golangci-lint run ./scripts/export-frontend-data/...`
- [x] `make size-check`
- [x] `npm run build` in `frontend/`
- [x] `go run ./scripts/export-frontend-data` twice; the second run produced no diff
- [ ] `make test` was not run. No engine code changed.
- [ ] `make golden` was not run. No layout or paint change.
- [ ] `make claim-scan` was not run. No product claims changed.

### Commands

```sh
go build ./...
golangci-lint run ./scripts/export-frontend-data/...
make size-check
cd frontend && npm run build
go run ./scripts/export-frontend-data
```

---

## Screenshots / sample output

```
The only visible change is the shortened view-transition-group note.
The rebuilt site embeds the regenerated catalog.
```

---

## Related issues

- None. No GitHub issue is linked.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled. There is no ticket ID.
- [x] Filled body committed under `plans/PR/pr-export-frontend-data-catalog.md`

---

## Follow-ups (out of scope)

- Decide whether catalog notes should stay capped at 180 bytes or show the full limitation text. Raising the cap would expand other rows.

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
| `.go` | 1 | 44 | 13 |
| `.html` | 1 | 1 | 1 |
| `.js` | 1 | 1 | 1 |
| `.json` | 1 | 1 | 1 |
| `.md` | 1 | 129 | 0 |
| **Total** | **5** | **176** | **16** |
