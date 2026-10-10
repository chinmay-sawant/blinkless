# Browser evidence: Chrome case geometry

Committed distillation of the gitignored `temps/chrome-cases/` run. The
manifest cites this file for browser evidence so the check works on a
checkout without `temps/`. Raw data stays under `temps/` and is listed at
the end.

## Method

- Browser: Chrome/143.0.7499.40 (`/usr/bin/google-chrome --version`, headless).
- Driver: puppeteer-core (24.43.1 recorded in
  `temps/chrome-cases/dsf/raw/measurements.json`), `temps/chrome-cases/capture.js`.
  Chrome is launched with `--hide-scrollbars` and
  `--force-device-scale-factor=1`.
- Viewport: 1024 x 768 CSS px, deviceScaleFactor 1, scrollbars hidden.
- Engine side: public `layout.DisplayList` element border boxes, CSS px,
  media=screen, same viewport.
- Fonts: `FONTCONFIG_FILE=temps/css-review/fonts.conf` maps
  sans-serif/Arial/Helvetica to Liberation Sans. 16 px probe widths:
  DejaVu Sans=160.297, Liberation Sans=140.469, monospace=153.625,
  sans-serif=140.469 px.
- Tolerance: 1.0 CSS px on every compared coordinate.
- Join: element path `tag:nth-of-type(n)` chain; identity key (tag, id,
  data-action, normalized text), order-independent. A box path that is a
  unique suffix of a browser path (fragment documents without html/body)
  joins by suffix. Duplicate-key groups are matched 1:1 by the
  minimum-bottleneck assignment over all elements in the group.
- Transform policy: Chromium rects include CSS transforms, Blinkless boxes
  do not. Pure-translate subtrees compare relative to the transformed
  ancestor; the transformed element compares size only; non-translate
  transforms are skipped.
- Delta = engine minus Chrome in CSS px. A case passes when every compared
  coordinate is within tolerance.

## Completed cases (2026-10-09 run)

28 of the 36 measured cases pass. `Elements` is the Blinkless DOM element
count and `Boxes` is the Blinkless display-list box count (including the
structural `#document` box, which is skipped by design); `Compared` counts
joined units.

| Case | Category | Elements | Boxes | Compared | Max delta | Verdict |
|---|---|---|---|---|---|---|
| legacy-flex-algorithm-margins | flex-sizing | 20 | 16 | 15 | 0.68 px | pass |
| legacy-columns-auto-size | column-sizing | 23 | 19 | 18 | 0.68 px | pass |
| legacy-definite-main-size | column-sizing | 13 | 9 | 8 | 0.68 px | pass |
| legacy-justify-content | alignment | 24 | 20 | 19 | 0.68 px | pass |
| legacy-flex-align | alignment | 11 | 7 | 6 | 0.68 px | pass |
| legacy-flex-align-vertical-writing | alignment | 12 | 8 | 7 | 0.68 px | pass |
| legacy-flex-flow-orientations | direction | 11 | 7 | 6 | 0.68 px | pass |
| legacy-flex-flow | direction | 12 | 8 | 7 | 0.68 px | pass |
| legacy-multiline | wrapping | 15 | 11 | 10 | 0.68 px | pass |
| legacy-multiline-align-content-column | wrapping | 12 | 8 | 7 | 0.68 px | pass |
| legacy-flex-flow-auto-margins | direction | 28 | 24 | 23 | 0.68 px | pass |
| legacy-flex-align-baseline | alignment | 44 | 40 | 39 | 0.68 px | pass |
| legacy-multiline-align-self | alignment | 24 | 20 | 19 | 0.68 px | pass |
| wpt-flex-basis-011 | flex-sizing | 14 | 10 | 9 | 0.68 px | pass |
| wpt-rtl-flow-reverse | direction | 13 | 9 | 8 | 0.68 px | pass |
| wpt-flexbox-margin-auto | flex-sizing | 13 | 7 | 6 | 0.68 px | pass |
| wpt-min-size-auto-overflow-clip | min-size | 11 | 7 | 6 | 0.68 px | pass |
| wpt-flow-row-wrap | wrapping | 13 | 9 | 8 | 0.67 px | pass |
| wpt-aspect-ratio-cross-size-002 | intrinsic-sizing | 13 | 9 | 8 | 0.67 px | pass |
| wpt-flex-item-percentage-abspos | positioning | 13 | 9 | 8 | 0.67 px | pass |
| wpt-definite-sizes-002 | column-sizing | 12 | 8 | 7 | 0.67 px | pass |
| wpt-percentage-heights-005 | column-sizing | 12 | 8 | 7 | 0.67 px | pass |
| wpt-flex-minimum-width-aspect | min-size | 12 | 8 | 7 | 0.67 px | pass |
| wpt-break-nested-float-print | pagination | 13 | 9 | 8 | 0.67 px | pass |
| wpt-column-reverse-multiline | wrapping | 13 | 9 | 8 | 0.67 px | pass |
| wpt-flex-base-size-max-width | flex-sizing | 13 | 9 | 8 | 0.67 px | pass |
| blink-replaced-aspect-ratio-precision | replaced | 11 | 6 | 5 | 0.67 px | pass |
| blink-scrollbars-row-reverse-vrl | overflow | 10 | 6 | 5 | 0.67 px | pass |

The other four manifest cases are covered elsewhere:
`legacy-flex-algorithm` and `legacy-flex-algorithm-minmax` carry go-test
evidence (`internal/layout/flex_chrome_cases_01_02_test.go`);
`wpt-align-items-stretch` and `wpt-gap-002-ltr` cite browser reports under
`temps/css-review/`, which this file does not distill.

## Blocked cases: box deltas before vs after

The 8 cases below fail the box comparison. The after column is the
current-tree rerun (waves A-F at `6f1a31a` plus the uncommitted
display-list snap work). Every compared unit kept the same deltas before
and after, so the snap work does not move element boxes: `SnapDisplayToDevicePixels`
clones `Ops`, `Order` and `Boxes` and rewrites only positive stroke widths
(`layout/displaylist_snap.go:45-63`).

| Case | Category | Elements | Boxes | Compared | Before max | After max | Before failing | After failing | Changed units |
|---|---|---|---|---|---|---|---|---|---|
| wpt-writing-mode-006 | direction | 48 | 44 | 43 | 5.42 px | 5.42 px | 2 | 2 | 0 |
| wpt-auto-margins-column | alignment | 15 | 11 | 10 | 1.33 px | 1.33 px | 2 | 2 | 0 |
| wpt-flex-factor-less-than-one | flex-sizing | 26 | 22 | 21 | 4.00 px | 4.00 px | 2 | 2 | 0 |
| wpt-flex-item-compressible | min-size | 23 | 19 | 18 | 3.39 px | 3.39 px | 2 | 2 | 0 |
| wpt-flex-container-max-content | intrinsic-sizing | 18 | 14 | 13 | 6.75 px | 6.75 px | 12 | 12 | 0 |
| wpt-flex-cross-size-border-box | alignment | 12 | 8 | 7 | 1.34 px | 1.34 px | 3 | 3 | 0 |
| blink-gap-decorations-basic | wrapping | 15 | 11 | 10 | 1.36 px | 1.36 px | 3 | 3 | 0 |
| wpt-flex-container-min-content | intrinsic-sizing | 12 | 8 | 7 | 2.71 px | 2.71 px | 3 | 3 | 0 |

The failing `html` and `body` rows are usually consequences of a child row.
The full per-unit rows live in
`temps/chrome-cases/dsf-rerun/box-deltas-before-after.md` and
`temps/chrome-cases/dsf-rerun/geometry-report.md`; they are identical to
the 2026-10-09 run (`temps/chrome-cases/geometry-report.md`).

## Blocked cases: paint-width validation

Engine strokes were joined to their Chrome elements by border-box edge and
path at dsf 1, 1.25, 1.5, 2. At dsf 1 every joined element's snapped
stroke width equals Chrome's computed border width.

| Case | Bordered elements | dsf 1 matches | Fractional-dsf deviations |
|---|---|---|---|
| wpt-writing-mode-006 | 9 | 9/9 | 18 |
| wpt-auto-margins-column | 3 | 3/3 | 6 |
| wpt-flex-factor-less-than-one | 7 | 7/7 | 14 |
| wpt-flex-item-compressible | 6 | 6/6 | 12 |
| wpt-flex-container-max-content | 7 | 7/7 | 20 |
| wpt-flex-cross-size-border-box | 1 | 1/1 | 2 |
| blink-gap-decorations-basic | 2 | 2/2 | 5 |
| wpt-flex-container-min-content | 3 | 3/3 | 8 |

Totals: 38 bordered elements joined, 0 element mismatches at dsf 1.
Exception: `wpt-flex-cross-size-border-box` has two 8 pt `solid transparent`
borders. The engine paints no stroke for them and Chrome computes 10 px with
no ink either, so there is no painted width to compare. That case's box
delta is layout-only.

At fractional dsf the snap policy floors `cssPx * dsf` to whole device
pixels while Chrome paints fractional coverage. The measured deviation rows:

| Specified | dsf | Engine device px | Chrome device px | Engine CSS px | Chrome CSS px |
|---|---|---|---|---|---|
| 1pt | 1.25 | 1 | 1.25 | 0.8 | 1 |
| 1pt | 1.5 | 2 | 1.5 | 1.3333 | 1 |
| 2pt | 1.25 | 3 | 2.5 | 2.4 | 2 |
| 2pt | 1.5 | 4 | 3 | 2.6667 | 2 |
| 2pt | 2 | 5 | 4 | 2.5 | 2 |

At dsf 1 and wherever `cssPx * dsf` is an integer (for example 1pt at dsf 2,
2pt at dsf 1) the policy matches Chrome. The Chrome side of this table comes
from `test/chrome/evidence/dsf-border-widths.md`.

## Raw sources

- `temps/chrome-cases/geometry-report.md`: 2026-10-09 run, the 36-case
  summary and per-case detail (generated 17:37 UTC).
- `temps/chrome-cases/dsf-rerun/geometry-report.md`: 8 blocked-case rerun
  (generated 2026-10-09 18:47 UTC).
- `temps/chrome-cases/dsf-rerun/box-deltas-before-after.md`: before/after
  comparison and the after-side failing units.
- `temps/chrome-cases/dsf-rerun/paint-width-validation.md`: per-case paint
  tables and the fractional-dsf deviation rows.
- `temps/chrome-cases/raw/*.joined.json` and
  `temps/chrome-cases/dsf-rerun/raw/*.joined.json`: per-unit joined data.
- Harness: `temps/chrome-cases/run_cases.sh`, `capture.js`, `join.py`.

## Reproduction

```sh
bash temps/chrome-cases/run_cases.sh   # dumps + browser captures, exit 0
python3 temps/chrome-cases/join.py     # writes the geometry report
```
