# feat(css): implement the v0.1.2 pending CSS properties and clear lint on the branch

## Summary

- Implements 133 more CSS properties in the engine and records them in the catalog: implemented rows go from 555 to 688, partial rows from 4 to 0, and unsupported rows from 218 to 89 (of 785).
- Adds the v0.1.2 checklists and validation plan these changes were worked from, a demo page for the first ten styles, and a lint pass so the branch passes `make lint`.
- Changes no public Go API signature. New styles are stored on `layout.Box` (`Box.Declared`) and, where the engine can act on them, change the drawing list.

---

## Motivation / context

- Plans: `plans/v0.1.2/README.md`, `plans/v0.1.2/validation.md`, and the nine phase checklists under `plans/v0.1.2/`.
- The catalog is `testdata/css/catalog/properties.json`. At the 2026-11-02 start of v0.1.2 it listed 555 implemented, 4 partial and 218 unsupported rows. This branch moves 133 rows to implemented and closes the 4 partial rows.
- The OwnFrame window `examples/blinkless-all` (in the `ownframe` repository) is where a person checks these styles by hand. That checking found no engine mismatch; two host-side demo bugs were fixed in `ownframe`.

---

## Changes

### Styles, in the order they landed

- The first, second and third batches of ten styles (`331447b`, `e9075d2`, `880862f`). The third batch is `marker`, `marker-start`, `marker-mid`, `marker-end`, `stop-color`, `stop-opacity`, `path-length`, `paint-order`, `link-parameters` and `continue`. `paint-order` and `path-length` change pixels (`3aab091`). Each style has a fixture under `testdata/css/v012` and a behavior test.
- `margin-break`, `image-rendering`, `vector-effect`, `flood-color`, `flood-opacity` (`1eacdb7`). `-webkit-text-stroke` and its two longhands (`b9e677b`).
- 27 timeline, trigger, interest and interactivity styles stored on `layout.Box` (`1e62672`), with comma lists and named scrollers (`d4d1928`).
- 61 navigation, border, motion path, fragmentation, reading order and SVG leftover styles (`bb688b4`, `8357404`): the border clips, `border-shape`, `border-boundary`, `text-fit`, `float-defer`, `shape-image-threshold` and the `offset` family; SVG text drawn as outlines, `baseline-shift` and `lighting-color` (`e09eedd`); `background-attachment` fixed, `background-tbd`, `copy-into` and the footnote longhands (`8467e53`); `margin-break` flipped to implemented (`3663860`).

### Docs and plans

- v0.1.2 checklists and the validation plan; ledger rows closed as tests pass (`485988e`, `0f2a6f6`, `9a27517`, `91531d8`).
- One demo page for the ten styles and a readable decoration-break page (`1863696`, `2ce217e`, `9eb7f60`).

### Lint

- `6f4d13b` clears 99 `golangci-lint` findings in the SVG presentation and layout style code: named consts for repeated strings and numbers, long functions split by responsibility, five `//nolint:gochecknoglobals` with a reason. No behavior change. `internal/layout/style_cascade.go` shrinks from 2300 to 2279 lines and the file-size ledger records it.
- One value changes shape: `backdropGroupBegins` returns an empty slice instead of nil. Its only caller checks `len`.

### Fixes found while validating

- `refract()` in `backdrop-filter`: `e9075d2` limited validation to `blur()` and `saturate()`, which dropped the `refract(<length>)` extension OwnFrame paints. OwnFrame's GPU parity test (`make graphics-check`) failed against this branch because of it. `refract` is accepted again with a non-negative length, and a test checks that a negative value does not replace an earlier valid list.
- `make catalog-check` failed on eight rows. Four animation rows now name quoted cases that exist in the OwnFrame tests, and `wrap-inside` names a real handler. The catalog row for `backdrop-filter` and the compatibility matrix list `refract`.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Not measured in this PR. No benchmark was run. |
| **Memory** | Not measured in this PR. |
| **Behavior / correctness** | New styles change the drawing list where the engine acts on them; the rest are stored on `Box.Declared` for the host. |
| **API / CLI** | New exported style values on `layout.Box`; no signature removed. No CLI exists in this tree. |
| **Dependencies** | None added. |
| **Binary size / build time** | Not measured. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] `make test` exits 0
- [x] `make lint` exits 0 (including `size-check`)
- [x] `make golden` exits 0
- [x] `make claim-scan` exits 0 ("clean")
- [x] `make matrix-check` exits 0
- [x] `make catalog-check` exits 0 (see "Fixes found while validating")

### Commands

```sh
make test
make lint
make golden
make claim-scan
make matrix-check
make catalog-check
```

---

## Screenshots / sample output

No visual surface was changed by the lint commit. The styles were checked by eye in the OwnFrame `blinkless-all` window (all 45 examples, six tabs); results are in the `ownframe` pull request.

---

## Related issues

- None.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`enhancement`)
- [x] Related issues filled (none)
- [x] Filled body committed under `plans/v0.1.2/PR/pr-v012-pending-css.md`

---

## Follow-ups (out of scope)

- Cut a release and bump the `ownframe` module pin; `ownframe` needs these features (for example `DisplayOp.TextStroke`) and does not build against the pinned `v0.1.1` without them.

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 153 | 20077 | 660 |
| `.html` | 25 | 711 | 0 |
| `.json` | 1 | 3850 | 988 |
| `.md` | 23 | 3947 | 163 |
| `.svg` | 1 | 1 | 0 |
| `.txt` | 1 | 6 | 6 |
| **Total** | **204** | **28592** | **1817** |

---

🤖 Generated with [Claude Code](https://claude.com/claude-code)
