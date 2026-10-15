## Summary

The product site is a dark, paper-and-ink page for the drawing list. The home page shows a real `layout.DisplayList`, the docs have shareable links, and each Go API call opens its own example.

---

## Motivation / context

- Plans: none
- Issues: see **Related issues**

---

## Changes

### Site

- Dark theme is the default, with a light switch that is remembered.
- The headline breaks before "A drawing list out."
- The specimen is engine output. The white page covers the canvas, and the black and red bars sit inside a 20px margin.
- GitHub shows a star and the live star count.
- "Read the docs" and the `go get` line are the same height.
- "Where it stops" explains JavaScript, PDF, and a page picture in terms of the drawing list.

### Docs

- Sections are addresses such as `#docs/api` and `#docs/quickstart`.
- The CSS page lists the 389 implemented and 19 partial catalog names.
- Clicking a Go API signature opens that call's example.

### Data

- `scripts/export-frontend-data` writes `specimen.json` and `catalog.json` from the engine and the CSS catalog.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | No engine change. |
| **Memory** | No engine change. |
| **Behavior / correctness** | The published site and `frontend/` change. Layout behavior does not. |
| **API / CLI** | No public Go API change. |
| **Dependencies** | No module changes. |
| **Binary size / build time** | No. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] `cd frontend && npx tsc --noEmit`
- [x] `cd frontend && npm run build`
- [x] Browser check of the dark specimen, the API examples, and the headline break
- [ ] `make test` (not run; no layout behavior change)
- [ ] `make lint`
- [ ] `make golden`
- [ ] `make claim-scan`

### Commands

```sh
cd frontend && npx tsc --noEmit
cd frontend && npm run build
```

---

## Screenshots / sample output

The home page specimen lists one outer fill at 0, 0 covering the 420 by 132 canvas, then the black bar at 20, 20 and the red bar at 20, 74.

---

## Related issues

- None

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/PR/pr-frontend-site.md`

---

## Follow-ups (out of scope)

- None

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
| `.css` | 2 | 100 | 0 |
| `.go` | 1 | 291 | 0 |
| `.html` | 2 | 53 | 0 |
| `.jpg` | 4 | Binary | Binary |
| `.js` | 3 | 251 | 0 |
| `.json` | 7 | 4907 | 0 |
| `.md` | 1 | 131 | 0 |
| `.ts` | 4 | 155 | 0 |
| `.tsx` | 11 | 1349 | 0 |
| No extension | 2 | 27 | 0 |
| **Total** | **37** | **7264** | **0** |
