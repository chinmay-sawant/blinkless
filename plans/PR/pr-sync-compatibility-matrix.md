## Summary

Refresh the CSS compatibility matrix and the website catalog from the Blinkless property catalog. Both now report the support recorded for the current engine.

## Motivation / context

The matrix header and older feature summaries disagreed with the generated property tables. The website export also lagged the source catalog. OwnFrame uses the matrix as the source for its engine compatibility snapshot.

## Changes

- Remove the duplicate matrix totals and replace stale family status summaries with references to the generated tables.
- Regenerate the website catalog. It contains 688 implemented properties. The full catalog has 785 entries, including 89 unsupported and 8 intentionally ignored properties.

## Impact

| Area | Impact |
| --- | --- |
| Behavior and correctness | The matrix and website data reflect the catalog. |
| Engine runtime, API, and dependencies | Unchanged. |

## Breaking changes / migration

None.

## Test plan

- [x] `make test`
- [x] `make catalog-check matrix-check claim-scan`
- [x] Compare every matrix property name and status with the source and verify the exported website fields for each supported property.
- [x] Run the website exporter twice and compare its output.
- [ ] GitHub Actions test and lint checks

## Screenshots / sample output

The website catalog contains 688 implemented rows. The PR updates the data used by the site.

## Related issues

No linked issue.

## PR metadata checklist

- Self-assignment and the documentation label are set on the PR.
- This body is tracked under `plans/PR/`.

## Follow-ups

Update OwnFrame's Blinkless pin and compatibility matrix after this PR merges.

## Reviewer checklist

- [ ] Matrix claims match the source catalog.
- [ ] Website rows match the exporter's supported-property subset.

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.json` | 1 | 766 | 12 |
| `.md` | 2 | 97 | 237 |
| **Total** | **3** | **863** | **249** |
