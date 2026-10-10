# Chrome harness

Comparison harness for CSS-REVIEW-01. It lays each fixture out with the
public `layout.DisplayList` API and captures the same fixture in headless
Chrome, then joins the two box sets by element path. All paths below are
repo relative. Run from the repo root.

Reference browser: Chrome 143.0.7499.40 (system binary
`/usr/bin/google-chrome`, headless, viewport 1024 x 768 CSS px,
`deviceScaleFactor` 1, scrollbars hidden).
Tolerance: 1.0 CSS px on every compared coordinate (`TOLERANCE_PX` in
`join.py`).

## Files

- `cases.json`: the 11 cases. Each entry has `slug`, `category`,
  `fixture`, and `note`. Slugs are the file stems used under `raw/`.
- `main.go` (`go run ./test/chrome/harness`): lays one HTML fixture out
  through `css.Apply` plus `layout.DisplayList` and writes
  `raw/<slug>.blinkless.json`. One entry per display box joined to its DOM
  element by tag, id, data-action, and normalized text. Flags: `-fixture`
  (required), `-width`, `-height`, `-out`. Match quality is one of
  `exact`, `duplicate`, `text-fallback`, `order-fallback`, or `unmatched`.
- `capture.js` (`node test/chrome/harness/capture.js <fixture> <out>
  [width] [height]`): drives system Chrome with `puppeteer-core` from
  `scripts/puppeteer/node_modules`, sets the viewport, pins the generic
  font families to Liberation Sans over CDP, loads the fixture over
  `file://`, waits for fonts and two animation frames, then writes
  `raw/<slug>.browser.json` with `getBoundingClientRect` rects, computed
  `display`/`visibility`/`position`/`transform`, the transform chain, and
  a font probe. Logs element count and viewport sizes to stdout.
- `run_cases.sh` (`bash test/chrome/harness/run_cases.sh`): reads
  `cases.json`, then per case runs the Go dumper to
  `raw/<slug>.blinkless.json` and `capture.js` to
  `raw/<slug>.browser.json`. Honors `WIDTH`, `HEIGHT`, `CHROME_BIN`, and
  exports `FONTCONFIG_FILE=test/chrome/harness/fonts.conf`. Exit code is
  0 only when every dump and every capture exited. Does not write the
  joined report.
- `join.py` (`python3 test/chrome/harness/join.py`): reads both
  `raw/<slug>.*.json` files per case, joins by element path
  (`tag:nth-of-type(n)` chain, with unique-suffix fallback for fragment
  documents that lack html/body), applies the transform policy, and
  writes `raw/<slug>.joined.json` plus `geometry-report.md`. Prints a
  per-slug verdict map. Exit code 0 when the report was written.
  Duplicate-key groups (flex/grid reorder) compare order-blind by the
  minimum-bottleneck 1:1 match. Pure-translate subtrees compare relative
  to the transformed ancestor, the transformed element itself compares
  size only, and non-translate transforms are skipped, as are
  `display:none`, `visibility:hidden`, and `position:fixed`.
- `fonts.conf`: fontconfig snippet used only when `FONTCONFIG_FILE` is
  exported. Maps `sans-serif`, `Arial`, and `Helvetica` to the installed
  Liberation Sans so browser text metrics compare against the embedded
  face. Rejects `*.woff`/`*.woff2` selects.
- `raw/`: gitignored output dir for `.blinkless.json`, `.browser.json`,
  `.joined.json`, and `geometry-report.md`. Created by `run_cases.sh`.

## Cases (11)

1. `flex-align-items-stretch` (flex)
2. `flex-flow-row-wrap` (flex)
3. `flex-gap-002-ltr` (flex)
4. `grid-flex-full` (grid)
5. `grid-areas-dense` (grid)
6. `grid-minmax-intrinsic` (grid)
7. `logical-flex-auto-margins` (logical)
8. `logical-writing-mode-006` (logical)
9. `text-long-text-wrap` (text)
10. `text-typography` (text)
11. `print-nested-float` (print)

See `cases.json` for the fixture path and note behind each slug.

## Commands

```sh
bash test/chrome/harness/run_cases.sh   # dumps + browser captures, exit 0
python3 test/chrome/harness/join.py     # writes geometry-report.md
```
