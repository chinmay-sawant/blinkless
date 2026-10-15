## Summary

`make lint` failed in `scripts/export-frontend-data/main.go`. The exporter still writes the same `frontend/src/data/specimen.json` and `catalog.json`. Regenerating those files after the fix produced no diff.

---

## Motivation / context

- Plans: `plans/PR/pr-export-frontend-data-lint.md`
- Issues: see **Related issues**

`golangci-lint run ./...` is the first step of `make lint`. With every linter enabled, that one file failed whitespace, error wrapping, struct completeness, name length, complexity, and magic-number checks.

---

## Changes

### Exporter

- `writeSpecimen`, `writeCatalog`, and `writeJSON` wrap errors from `html.Parse`, `css.Apply`, `layout.DisplayList`, `os.ReadFile`, `json.Unmarshal`, `json.MarshalIndent`, and `os.WriteFile` with `%w`. See `scripts/export-frontend-data/main.go:105`, `:116`, `:124`, `:302`, `:307`, `:358`, and `:364`.
- `css.Apply` now sets every `css.Options` field. `Media` is `"screen"`, which `css/css.go:149` treats the same as an empty string. `Extra`, `Focus`, `Hover`, and `Active` stay empty.
- The red, black, and paper colors are the specimen CSS values `#9c3b2e`, `#1c1915`, and `#f6f4ef`, named at `main.go:43`. `paintLabel` at `main.go:205` walks those three colors. The old switch scored 13 on cyclop, and the limit is 10.
- A catalog note still drops the " Pinned upstream" tail and still cuts at 180 bytes. Those limits are `pinnedUpstreamMark` and `maxNoteLen` at `main.go:40` and `:52`.
- `os.WriteFile` still uses mode `0644`, through `siteDataPerm` at `main.go:31`. The files are committed site data. `scripts/gen-style-intern/main.go` uses the same mode for the same reason.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Same specimen document and same catalog read. |
| **Memory** | Same. |
| **Behavior / correctness** | Success JSON is unchanged. A failed read, parse, or write now returns a prefixed error. |
| **API / CLI** | `go run ./scripts/export-frontend-data` is unchanged. |
| **Dependencies** | None. |
| **Binary size / build time** | None. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] `golangci-lint run ./scripts/export-frontend-data/...` exited 0
- [x] `make lint` exited 0 with golangci-lint v1.64.8, then `make size-check`
- [x] `go run ./scripts/export-frontend-data` left `frontend/src/data/specimen.json` and `catalog.json` unchanged
- [ ] `make test` was not run. The exporter has no test package, and layout behavior did not change.
- [ ] `make golden` was not run. No layout or paint change.
- [ ] `make claim-scan` was not run. No product claims changed.

### Commands

```sh
golangci-lint run ./scripts/export-frontend-data/...
make lint
go run ./scripts/export-frontend-data
```

---

## Screenshots / sample output

```
no visual surface changed
```

---

## Related issues

- None. No GitHub issue is linked.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled. There is no ticket ID.
- [x] Filled body committed under `plans/PR/pr-export-frontend-data-lint.md`

---

## Follow-ups (out of scope)

- None.

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
| `.go` | 1 | 131 | 54 |
| `.md` | 1 | 116 | 0 |
| **Total** | **2** | **247** | **54** |
