# Complete the HTML, CSS, and JSON compatibility program

## Summary

- Lands the full compatibility program for the drawing-list engine: the pinned html5lib parser corpus with zero failures, the browser-checked CSS catalog, the remaining layout parity fixes, and the versioned WASM drawing-list JSON contract.
- Adds display-list-only device-pixel quantization with explicit DSF support and the Chrome-compatible medium widthless-border default, with committed browser evidence under `test/chrome/evidence/`.
- Reconciles the planning ledgers and retires the PDF-era tooling, comments, and scripts left behind by the writer removal.

## Motivation / context

- Plans: `plans/v0.0.1/html-css-json-compatibility-checklist.md` (all rows closed, with a decision log), `plans/v0.0.1/phase-wise-checklist.md` (12 open rows remain), `plans/v0.0.1/css-behavior-coverage-tests-checklist.md` (new: the 301 partial rows, of which 298 are test-only gaps), `plans/README.md`.
- Issues: none; the program is tracked by the plan ledgers above.

## Changes

### Parser

- Pinned html5lib-tests corpus at `9329e64694e7835d0dcff9811e22856ef6ad16f9` (scripting=false): tokenizer 6686 of 7036 with 0 failures (350 unsupported non-Data initial states), tree construction 1784 of 1792 with 0 failures (8 script-on skips), plus 50 local cases.
- Template contents outside the rendered tree, context-aware fragments (all 192 upstream fragment cases), foreign content, ruby, script-data escaped states, noscript, and frameset recovery.

### CSS

- Catalog at 785 rows: 90 implemented / 301 partial / 387 unsupported / 7 intentionally ignored, with matrix drift enforced by `scripts/check-matrix-sync.sh` in CI.
- Value acceptance covers 90 of 90 advertised implemented properties; `text-wrap-style: balance` ships with Chrome line-break references.

### Layout and paint

- Border lengths convert to points; nonzero-body margin collapse, aspect clamps, definite heights, atomic-inline line metrics, orthogonal auto width, flex gap distribution, grid fr distribution, normal line height from face metrics, float containment in flex items, and the abs-positioned initial containing block are fixed.
- `layout.SnapDisplayToDevicePixels(display, dsf)` quantizes positive `OpStrokeRect` and `OpLine` widths to whole device pixels; style, layout geometry, `Display.Boxes`, and hit-testing stay pt-exact, and print consumers replay the unquantized list.
- Widthless border shorthands resolve to Chrome's `medium` (3 CSS px), explicit `0` stays zero, and thin/medium/thick parse in the shorthand.

### Browser cases

- `test/chrome/manifest.json` stands at 32 completed / 8 blocked with browser evidence committed under `test/chrome/evidence/`; the blocked cases carry box deltas and a paint-width validation (38 of 38 joined bordered elements match Chrome at dsf 1), and the fractional-dsf snap deviation is documented.

### WASM / JSON

- The versioned drawing-list JSON schema (`blinkless.drawinglist/1`) ships with a node consumer check that deep-compares browser JSON against a native run.

### Ledgers, docs, and tooling

- Compatibility checklist 60 of 60; parent ledger 12 open / 94 closed (13 retired) / 1 deferred; AGENTS.md, CONTRIBUTING.md, CHANGELOG.md, RELEASE.md, and the architecture docs synced to the current tree.
- Release tooling gates the committed version sources (`bindings/python/pyproject.toml`, `bindings/c/include/blinkless.h`); six dead Makefile targets and five dead PDF-era scripts removed; stale comments cleaned across five packages.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | No material change; the snap is opt-in at the paint boundary. |
| **Memory** | No material change. |
| **Behavior / correctness** | Parser and CSS parity improve substantially; layout fixes move several fixtures; screen consumers can opt into device-pixel snapping. |
| **API / CLI** | Adds `layout.SnapDisplayToDevicePixels` and `ErrBadDeviceScale`. No CLI exists in this tree. |
| **Dependencies** | None added; the allowlist stays `github.com/go-text/typesetting` and `github.com/tdewolff/canvas`. |
| **Binary size / build time** | No material change. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| Widthless border shorthands now resolve to `medium` (3 CSS px, 2.25 pt) | Intentional Chrome parity; set an explicit width to keep the old 1 pt |
| None else | - |

## Test plan

- [x] `make test` (via `make final-evidence`)
- [x] `make lint`
- [x] `make build`
- [x] `make golden`
- [x] `make claim-scan`
- [x] `make final-evidence FINAL_EVIDENCE_FLAGS=--full` (11 of 11 steps)

### Commands

```sh
make final-evidence FINAL_EVIDENCE_FLAGS=--full
```

## Screenshots / sample output

```
[final-evidence] all 11 steps passed (2026-10-10T05:32:25Z)
[final-evidence] evidence: temps/final-evidence/20261010T053225Z
```

## Related issues

- No tracking issue; tracked by `plans/v0.0.1/html-css-json-compatibility-checklist.md` and `plans/v0.0.1/phase-wise-checklist.md`.

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`enhancement`, `documentation`)
- [x] Related issues section states the tracking situation
- [x] Filled body committed under `plans/PR/pr-html-css-json-compatibility.md`

## Follow-ups (out of scope)

- The 8 blocked Chrome cases keep box-level status until a screen-paint acceptance policy is chosen; paint widths are validated at dsf 1.
- Bench targets and the fenced PDF-era scripts (`bench-external.sh`, `bench-performance-recovery.sh`, `run-walltime.sh`, `compare_chrome_ana.py`, `screenshot_showcase.py`) need a removal decision.
- Case 34 inline-block baseline approximation (diagnosis recorded in `temps/waveB-followups.md`).
- `place-content`'s grid half has no consumer while the catalog row reads implemented.
- Python binding removal stubs and PDF settings keys (open parent-ledger rows).

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
| `.dat` | 60 | Binary | Binary |
| `.go` | 123 | 18756 | 1732 |
| `.html` | 45 | 395 | 392 |
| `.js` | 1 | 0 | 58 |
| `.json` | 6 | 23876 | 29 |
| `.jsonl` | 1 | 28 | 0 |
| `.md` | 68 | 5897 | 4247 |
| `.mjs` | 1 | 225 | 0 |
| `.py` | 12 | 1533 | 667 |
| `.sh` | 8 | 597 | 41 |
| `.test` | 16 | 59286 | 0 |
| `.toml` | 1 | 7 | 2 |
| `.txt` | 2 | 22 | 1 |
| `.yml` | 3 | 94 | 71 |
| No extension | 3 | 60 | 109 |
| **Total** | **350** | **135653** | **7349** |
