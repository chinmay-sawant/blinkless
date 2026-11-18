## Summary

Rebuild the checked-in Blinkless website so the CSS catalog page shows the 688 implemented properties shipped in the source catalog.

## Motivation / context

PR #40 refreshed the source catalog and compatibility matrix. The generated site still displayed 555 implemented and three partial entries.

## Changes

- Regenerate the JavaScript bundle in `docs/assets/` and update the module script in `docs/index.html`.
- Add the 130 missing entries to the published build and update the three entries whose status changed to implemented.

## Impact

| Area | Impact |
| --- | --- |
| Website | The CSS catalog lists 688 implemented properties and zero partial entries. |
| Engine API and dependencies | No changes. |

## Breaking changes / migration

None.

## Test plan

- [x] `npm --prefix frontend run build` (TypeScript and Vite)
- [x] `make lint`
- [x] `make test`
- [x] Check the generated JavaScript syntax with Node.
- [x] Compare every name and status in the bundle referenced by `docs/index.html` with `frontend/src/data/catalog.json`.

The repository's `make lint-frontend` target is a no-op. The frontend build runs TypeScript checking.

## Screenshots / sample output

The generated bundle contains 688 implemented rows and zero partial rows.

## Related issues

Refs #40.

## PR metadata checklist

- Self-assignment and the documentation label are set on the PR.
- This body is tracked under `plans/PR/`.

## Follow-ups

None.

## Reviewer checklist

- [ ] The website serves the new bundle referenced by `docs/index.html`.
- [ ] The embedded catalog matches the frontend source JSON.

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.html` | 1 | 1 | 1 |
| `.js` | 2 | 205 | 205 |
| `.md` | 1 | 64 | 0 |
| **Total** | **4** | **270** | **206** |
