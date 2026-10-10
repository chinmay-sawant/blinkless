# blinkless - Pull Request Template

## Summary

Plans a native web view built around the Blinkless drawing-list engine. Blinkless stays the HTML parser, CSS engine, layout engine, and source of paint operations. The browser behavior goes around that engine, not into it. No implementation ships in this PR.

---

## Motivation / context

- Plans: `plans/v0.0.1/native-window/native-web-view-checklist.md` (new, all rows open), `plans/README.md` (index entry)
- Issues: none; tracked by the plan ledger above

---

## Changes

### New plan

- Adds `plans/v0.0.1/native-window/native-web-view-checklist.md`, a 134-line proposal for a view that owns a live HTML document and presents drawing lists on a native surface
- Runs Phase 0 through Phase 8 plus a release boundary
- States up front that it does not claim browser parity
- Notes that the CSS catalog totals are not a readiness score, since the catalog measures a broad standards inventory rather than compatibility with a particular application

### Index

- Adds one line to `plans/README.md` linking the new checklist

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None. No code changes. |
| **Memory** | None. |
| **Behavior / correctness** | None. No code changes. |
| **API / CLI** | None. The plan proposes an API but defines none yet; Phase 0 is the gate for that decision. |
| **Dependencies** | None. |
| **Binary size / build time** | None. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [ ] `make test` (documentation-only; not run per the phase-wise checklist rule)
- [ ] `make lint` / `go vet` (documentation-only; not run)
- [ ] `make build` (not applicable)
- [x] `make golden` (no fixture or layout change; not required)
- [x] `make claim-scan`

### Commands

```sh
make claim-scan
```

Result: `claim-scan: clean`, exit 0.

Additional checks run for the plan itself:

- All 13 relative markdown links resolve, zero dead
- Zero em dashes in the new file
- No test or lint command run, because the phase-wise checklist template skips both for documentation-only changes

---

## Screenshots / sample output

```
no visual surface changed
```

---

## Related issues

- Relates to `plans/0.0.1/html-css-json-compatibility-checklist.md` (the capability work this proposal builds on)
- No tracking issue; tracked by the plan ledger

---

## Follow-ups (out of scope)

- Phase 0 decides between bundled application pages and general websites before any public API is committed. Only the second target requires JavaScript.
- The checklist cites three files in the sibling `ownframe` project. Those paths resolve on a local checkout with that folder present, but not for a reader who clones only blinkless.
- No implementation branch is created by this PR.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled
- [x] Filled body committed under `plans/PR/pr-native-window-proposal.md`

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.md` | 3 | 249 | 0 |
| **Total** | **3** | **249** | **0** |
