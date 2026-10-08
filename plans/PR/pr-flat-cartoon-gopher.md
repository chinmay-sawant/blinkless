## Summary

Add the selected flat-cartoon gopher to the README as the centered Blinkless mascot. The small page card points to the engine's drawing-list output described in `README.md:9`.

## Motivation / context

- Plans: None
- Issues: None

## Changes

### Branding

- Add the transparent mascot image at `assets/01-flat-cartoon.png`.

### README

- Center the mascot and project name above the introduction.
- Add alt text describing the mascot.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | No change |
| **Memory** | No change |
| **Behavior / correctness** | No change |
| **API / CLI** | No change |
| **Dependencies** | No change |
| **Binary size / build time** | No change |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

## Test plan

- [x] Confirm the README image path resolves and the PNG has an alpha channel.
- [ ] `make test` (not run; no Go source changed)
- [ ] `make lint` (not run; no code changed)

### Commands

```sh
file assets/01-flat-cartoon.png
```

## Screenshots / sample output

The README shows the selected mascot centered at 160 pixels wide.

## Related issues

None.

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`enhancement`, `documentation`)
- [x] Related issues checked; none apply
- [x] Filled body committed under `plans/PR/pr-flat-cartoon-gopher.md`

## Follow-ups (out of scope)

None.

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes / Relates keywords
- [ ] No secrets or generated artifacts committed
- [ ] Diff-stat-by-extension table pasted at the bottom

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.md` | 2 | 91 | 1 |
| `.png` | 1 | Binary | Binary |
| **Total** | **3** | **91** | **1** |
