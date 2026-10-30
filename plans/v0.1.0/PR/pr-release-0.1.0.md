## Summary

Promote 0.1.0: a stabilization release for OwnFrame. It hardens the
drawing-list surface OwnFrame consumes (restamp pass, timelines, SVG
paint records, override relayout, race-free pipelines) plus a layout
perf window, with the catalog and matrix sync. No new output format. Full promotion (CHANGELOG, RELEASE.md, generic docs) rides the
release PR.

## Motivation / context

- Plans: `plans/v0.1.0/release-note/v0.1.0.md`, `plans/v0.1.0/release-note/commits.md`
- Issues: none. Range is `v0.0.3` (`8767e82`) through `b28b7d6`.

## Changes

### Release note

- `plans/v0.1.0/release-note/v0.1.0.md`: What changed, Supported surface, Removed (none),
  Version strings, What still does not ship.
- `plans/v0.1.0/release-note/commits.md`: 26 commits in order, merges
  included.
- `plans/v0.1.0/PR/release-v0.1.0.md`: byte-identical GitHub release body.
- `plans/v0.1.0/README.md`: version index.

### Pending in the release PR (not in this note)

- `CHANGELOG.md`: dated `## 0.1.0` section above 0.0.3.
- `RELEASE.md`: version line 0.0.3 to 0.1.0.
- Generic docs unreleased-language sweep per `skills/release-note/SKILL.md`.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Faster first-launch and repeated layouts (memos, metadata reuse). |
| **Memory** | Shared raster, grid, flex, and advance caches. |
| **Behavior / correctness** | Additive: restamp pass, timelines, SVG paint restyle, override relayout. Race-free concurrency. Flex and abspos geometry fixes. |
| **API / CLI** | New: `TimelineDef`, `TransformOverride`, `RestampTransforms`, `SVGPaintOverride`, `RestyleInlineSVG`, `RelayoutElementWithOverrides`, `Op.XformValue`. No CLI (none exists). |
| **Dependencies** | None. |
| **Binary size / build time** | None claimed. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

## Test plan

- [x] `go build ./...` (exit 0 at `b28b7d6`)
- [x] `make claim-scan` (clean, same HEAD)
- [x] `make matrix-check` (section 2 matches, same HEAD)
- [x] Targeted `go test ./internal/layout` (pass, same HEAD)
- [ ] `make test-quick` (release gate, runs in the release PR)
- [ ] `make golden` (release gate, runs in the release PR)
- [ ] `make lint` (release gate, runs in the release PR)
- [ ] `make catalog-check` (known fail at HEAD on 7 OwnFrame animation cases; fails on the clean tree too, unrelated to this range)

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

No visual surface changed. Drawing-list schema stays `blinkless.drawinglist/1`.

## Related issues

- None.

## PR metadata checklist (author)

- [ ] Self-assigned (`--assignee @me`)
- [ ] Labels applied (`documentation`, `enhancement`)
- [ ] Related issues filled with real ticket IDs (none apply)
- [x] Filled body committed under `plans/v0.1.0/PR/pr-release-0.1.0.md`

## Follow-ups (out of scope)

- Push tag `v0.1.0` to publish the GitHub Release.
- `make catalog-check` OwnFrame failures for animation rows stay open.

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

<!-- Generated at release-PR time with bash scripts/pr-diff-stat.sh master. -->
