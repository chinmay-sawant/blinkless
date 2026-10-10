# Committed browser evidence

The browser runs for the Chrome cases used to live only under the
gitignored `temps/` tree, so CI skipped the browser evidence check. These
files are the committed distillation. Raw JSON and screenshots stay in
`temps/`; each file lists its sources.

## Files

### `browser-evidence.md`

- Method for the 2026-10-09 Chrome run: Chrome/143.0.7499.40, puppeteer-core,
  viewport 1024 x 768 CSS px, deviceScaleFactor 1, Liberation Sans, 1.0 CSS px
  tolerance, join and transform policy.
- Completed-case table: the 28 passing cases with elements, boxes, compared
  units, max delta and verdict.
- Blocked-case table: the 8 failing cases with before/after max deltas,
  failing counts and changed-unit counts.
- Paint-width validation for the blocked cases at dsf 1, 1.25, 1.5, 2 plus
  the fractional-dsf deviation rows.
- Raw sources: `temps/chrome-cases/geometry-report.md`,
  `temps/chrome-cases/dsf-rerun/geometry-report.md`,
  `temps/chrome-cases/dsf-rerun/box-deltas-before-after.md`,
  `temps/chrome-cases/dsf-rerun/paint-width-validation.md`,
  `temps/chrome-cases/raw/*.joined.json`,
  `temps/chrome-cases/dsf-rerun/raw/*.joined.json`.

The manifest browser entries for the 36 `temps/chrome-cases/` cases should
point `evidence.file` here (`test/chrome/evidence/browser-evidence.md`).
`wpt-align-items-stretch` and `wpt-gap-002-ltr` cite the older
`temps/css-review/` reports and are not covered by this file.
`legacy-flex-algorithm` and `legacy-flex-algorithm-minmax` carry go-test
evidence and have no browser entry.

### `dsf-border-widths.md`

- Method for the 2026-10-10 device-pixel run: viewport 400 x 1000 CSS px,
  ten border probes, full-viewport screenshots, coverage-based painted-width
  measurement.
- Computed border-top width table at dsf 1, 1.25, 1.5, 2.
- Painted border-width table at the same dsfs with the exact
  `computed * dsf` products.
- Formula check: Chrome computes `max(1, floor(specified_css_px))` CSS px
  at every dsf and paints `computed * dsf` device px with fractional
  coverage; the `floor(cssPx * dsf)` snap policy deviates at fractional dsf.
- Fractional-dsf deviation rows: engine snap device px vs Chrome device px.
- Raw sources: `temps/chrome-cases/dsf/report.md`,
  `temps/chrome-cases/dsf/raw/measurements.json` (93,051 bytes, cited only),
  `temps/chrome-cases/dsf/raw/screens/`,
  `temps/chrome-cases/dsf/raw/method-validation.log`,
  `temps/chrome-cases/dsf/raw/run.log`, `temps/chrome-cases/dsf/probe.html`,
  `measure.js`, `inspect.js`, and
  `temps/chrome-cases/dsf-rerun/paint/snap-*.json` plus `borders-*.json`.

## Related files

- `test/chrome/manifest.json` records the per-case status and evidence.
- `test/chrome/manifest_test.go` resolves browser evidence: the file must
  exist, contain the case id and contain the browser version string
  (`Chrome/143.0.7499.40`).
- `test/chrome/README.md` describes the case set and the evidence rules.
