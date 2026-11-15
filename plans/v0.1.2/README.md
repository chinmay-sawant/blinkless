# v0.1.2 pending CSS

Each directory holds one phase checklist. A `[x]` row is a behavior a current test already asserts. A `[ ]` row is open work. A `[~]` row is parked, with a reason, an owner, and a next gate. Close a `[ ]` row only after the test named on that row exits 0.

The catalog is `testdata/css/catalog/properties.json`, measured 2026-11-02: 785 rows, 555 implemented, 4 partial, 218 unsupported, 8 intentionally ignored. This ledger covers the 4 partial rows, the unsupported rows, and a few implemented baselines the new work has to match. It does not replan properties that are already implemented.

The engine result is `layout.DisplayList` on one canvas (`layout/displaylist.go`). These files do not add a page encoder, a PDF writer, or a second copy of `internal/layout/layout.go`.

Work from the top of the table. The first rows can change paint or box size on the canvas this tree already builds. Later rows are parked until a scrollport, a page splitter, a speech engine, or a spec that browsers actually ship.

| Order | Ledger | Names | What to do first |
|---|---|---:|---|
| 1 | [Quick aliases and partials](01-quick-aliases-and-partials/checklist.md) | 7 | Rename `-webkit-background-clip`, `-webkit-background-origin`, and `-webkit-background-size` onto the unprefixed painters. Then close the named gaps on `-webkit-mask-composite`, `backdrop-filter`, `break-inside`, and `mask-type`. |
| 2 | [Inline text](02-inline-text/checklist.md) | 5 | `baseline-source`, `line-fit-edge`, `line-padding`, `inline-sizing`, `word-space-transform`. Each section says what the text op Y or the line box must do. |
| 3 | [Sizing](03-sizing/checklist.md) | 8 | `size`, `min-size`, and `max-size` copy width and height. `zoom` multiplies used lengths. `min-intrinsic-sizing` and `frame-sizing` stay parked. |
| 4 | [Reading order and cascade](04-reading-and-cascade/checklist.md) | 7 | `all` can reset color and fills on this canvas. `reading-flow`, `reading-order`, `overlay`, and `background-attachment` stay parked. |
| 5 | [Fragmentation](05-fragmentation/checklist.md) | 10 | `box-decoration-break`, `margin-break`, `string-set`, and `continue` have a one-canvas test. Footnotes, bookmarks, and `copy-into` stay parked. |
| 6 | [Draft borders and shapes](06-draft-borders-and-shapes/checklist.md) | 18 | All 18 are `[~]`. Border clips, `border-shape`, `text-fit`, `shape-image-threshold`, `float-defer`, and `column-rule-visibility-items`. |
| 7 | [SVG presentation](07-svg-presentation/checklist.md) | 50 | Phase 0 records what to expect from `fill`, `fill-rule`, `shape-rendering`, `alignment-baseline`, and `dominant-baseline`. Later phases store stylesheet values for geometry that an inline style already moves. |
| 8 | [Rhythm, math, and vendor aliases](08-rhythm-math-and-vendor/checklist.md) | 24 | Four ruby properties stay `[x]`. Thirteen rhythmic and math names, plus seven vendor aliases, stay `[~]`. |
| 9 | [Print-noop](09-print-noop/checklist.md) | 102 | Speech, scroll, pointer, motion, and event timelines. Each name has its own expected value and a `[~]` row. |

Start at phase 1 of ledger 1. Do not treat two ledgers as one change.

`fill`, `fill-rule`, `shape-rendering`, `alignment-baseline`, and `dominant-baseline` are the detail standard for "what to expect." Their opened tests and the drawing-list result live in ledger 7, phase 0. New text work in ledger 2 uses the same shape: a snippet, the text op Y, and the test name.

Older ledgers that still have open rows for these names are historical. Use this directory instead:

- `plans/v0.0.3/svg-presentation/svg-presentation-checklist.md` points here at ledger 7.
- `plans/v0.0.3/scroll/scroll-checklist.md` and `plans/v0.0.3/speech/speech-checklist.md` point here at ledger 9.
- `column-rule-visibility-items` in `plans/v0.0.3/row-rules/row-rules-checklist.md` points here at ledger 6.

The closure gate on every implementation change is `make test`, `make golden`, `make lint`, `make claim-scan`, `make catalog-check`, and `make matrix-check`. GitHub CI runs `make test` and `make lint`. A documentation-only edit of these files does not run those gates.
