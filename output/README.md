# Sample artifacts

This directory holds leftovers from the period when the engine wrote PDFs and
page PNGs. `make samples` no longer regenerates them: the target now runs
`make golden`, which executes the public drawing-list tests (`TestDisplay*` in
`./layout`) and writes nothing.

There is no `cmd/` CLI and no `bin/blinkless`, so no command in the current
tree rebuilds these page rasters. Treat them as leftover rasters, not golden
byte baselines.

## Files

| File | Note |
|------|------|
| `fixture-01-simple-invoice.png` | 1024x323 raster of fixture-01 from the removed image pipeline |
| `fixture-21-detailed-report.png` | 1024x2723 raster of fixture-21 |
| `fixture-57-vanguard-telemetry-audit.png` | 4096x2776 raster of fixture-57 |
| `fixture-57.png` | Same bytes as the file above |

## Directories

- `pdf-1.7/`, `pdf-1.7-compliance/`, `pdf-2.0/`, `pdf-2.0-compliance/`,
  `python/` (with its empty `pdf-*` subdirs), and `validated/` hold no
  files. `validated/` held the Ghostscript reference PDFs for the deleted
  pixel-regression tests.
- `profiles/` is local profiling output (gitignored). Derived reports live
  under `plans/`.
- `wkhtmltopdf/` holds two historical benchmark snapshots
  (`benchmark-results.csv`, `benchmark-summary.md`) from the external
  comparison run. The PDFs named in them are gone.

The Python binding's `convert_*_to_pdf` helpers raise `RuntimeError`
(`bindings/python/src/blinkless/_lib.py:221`), so nothing in the current tree
produces PDFs.

The directory is its own Go module (`output/go.mod`) so these bytes stay out
of the parent module zip.
