# testdata/golden - Fixture Corpus

> **Parent:** `plans/0.0.1/phase-wise-checklist.md` (removal of the PDF pipeline).
> **Purpose:** HTML and CSS inputs for the layout engine. The output is
> `layout.DisplayList`; there is no PDF writer and no stored golden bytes.

## Layout

```
testdata/golden/
  README.md
  logo.png                       # relative asset for fixture-07 (160x48 PNG)
  certificate-background.jpg    # print-safe ornamental background for fixture-47
  style-05.css                   # relative stylesheet for fixtures 05/06
  theme-print-stories.css        # print theme for the poster, letter, storybook
  assets/                        # generated artwork (no bundled third-party fonts)
  fixture-01-simple-invoice.html       # single page, minimal CSS
  fixture-02-table-heavy-invoice.html  # wide table, borders, many rows
  fixture-03-multi-page-invoice.html   # >1 page, page-break usage
  fixture-04-*.html .. fixture-64-*.html   # phase-9.1+ corpus (skip *-header/footer companions)
  fixture-36-header.html / fixture-36-footer.html  # nested HF companions for fixture-36
  fixture-49-night-train-poster.html       # one-page illustrated poster
  fixture-50-letter-template.html           # one-page stationery template
  fixture-51-asteria-storybook.html        # four-page original anime-inspired story
  fixture-52-airline-boarding-pass.html    # one-page e-ticket + four boarding stubs
  fixture-53-asteria-observatory-poster.html # second poster variant
  fixture-54-ember-harbor-storybook.html   # four-page Ember Harbor storybook
  fixture-55-lantern-cooperative-report.html # self-contained pure HTML/CSS operations brief
  fixture-56-architecture-diagram.html      # 21-page architecture diagram, linked CSS
  fixture-56-architecture-diagram.css       # linked stylesheet for fixture-56
  architecture-diagram.html                 # corpus fixture (5-page library-API diagram)
  api/                                      # leftover from the removed library-API PDF generator
    architecture-diagram.html               # source template for the removed generator (5 pages)
  python_api/                               # leftover from the removed Python PDF generators
    architecture-diagram.html               # Python-syntax variant of the api/ template (5 pages)
    generate*.py                            # target the removed convert_*_to_pdf helpers, which raise RuntimeError
    test_generate.py                        # path-resolution tests (no shared library required)
  font-examples.html                        # 13 Google Fonts showcase (inline style; fonts not bundled; --font-path driven)
  out/                                      # leftover PDFs from before the writer was removed (gitignored)
```

There is no stored golden comparison. `make golden` runs the public
drawing-list tests (`TestDisplay*` in `./layout`) and never reads this
directory. Focused regression tests under `internal/layout` load individual
fixtures; the external engine comparison (`make bench`) uses its own
generated report template, not these fixtures.

When `fixture-NN-header.html` and/or `fixture-NN-footer.html` exist beside a
body fixture, they are companions, not standalone body fixtures. They were
inputs for the removed HTML header/footer feature.

## Fixture inventory

Every fixture carries a comment header naming it and stating what it
proves. The `Expected:` line in a header records the author's intent; no
table pins it and no stored output enforces it. There is no committed
golden output.

| Fixture | What it exercises | Pages |
|---------|-------------------|-------|
| 01 | Simple invoice: `div`, `h1`, `p`, `table`, `span`, bold total | 1 |
| 02 | Table-heavy: 15-row table, thead/tbody/tfoot, colspan subtotal, borders | 2 |
| 03 | Multi-page statement: `page-break-before/after/inside`, outline headings | ≥2 |
| 04 | Two-column layout via fixed table cells (float lite also available; see fixture-22) | 1 |
| 05 | Linked relative stylesheet (`style-05.css`) driving all styling | 1 |
| 06 | External `<a href>` → link URI ops + linked stylesheet | 1 |
| 07 | Relative `logo.png` + data: URI PNG, intrinsic sizes | 1 |
| 08 | Forced page breaks: before/inside/after + margin-gap convergence | 5 |
| 09 | Long multi-section report, natural pagination, outline fodder | ≥2 |
| 10 | Nested tables, `colspan` (2 and 4), th, tfoot totals | 1 |
| 11 | Paragraph-heavy justified prose, line-level pagination | ≥3 |
| 12 | Nested ul/ol/li with indentation and list-item styles | 1 |
| 13 | `pre` white-space preservation, `code` runs, log excerpts | 1 |
| 14 | Background colors, rgba() alpha, colored borders, hr | 1 |
| 15 | Markdown-ish bulleted requirements, nested lists, anchors | 1–2 |
| 16 | Full invoice: letterhead, bill-to, 24 line items, tfoot totals | 1–2 |
| 17 | Cover-style first page + `page-break-before` content page | 2 |
| 18 | Typography: h1–h6, strong/em/u/s/small, blockquote, code | 1 |
| 19 | Box model: fixed/min/max widths, margins, padding, borders | 1 |
| 20 | Image grid: four data: URI PNGs at intrinsic sizes | 1 |
| 21 | Detailed multi-section ops report (KPIs, WPs, invoice extract, REQs) | ≥3 |
| 22 | Float lite invoice chrome: float left/right + clear, inline-block badge, border-box | 1 |
| 23 | Multi-page table with repeating `<thead>` on continuation pages (phase 18) | ≥2 |
| 24 | Same-document `<a href="#id">` → link URI ops (phase 20) | 2 |
| 25 | Partial flex row: justify-content, gap, flex-grow (phase 17) | 1 |
| 26 | Position lite: relative offsets + absolute overlay (phase 17) | 1 |
| 27 | CJK/Unicode sample (pair with `--font-path` for real glyphs; phase 19) | 1 |
| 28 | flex-wrap, CSS grid lite, position:fixed stamp | 2 |
| 29 | Float beside table: float:right infobox + wrapping prose + clear (phase 17 quality) | 1 |
| 30 | Orphans/keep-with-next heuristic sample (phase 18; geometric fallback) | ≥2 |
| 31 | Print-scoped `position: sticky` (page content box = scrollport; phase 17 sticky-print) | ≥2 |
| 32 | Flex Stage A + Grid Stage B lite (`flex-grid-full.md`): reverse, space-evenly, gaps, column flex, template-rows, row span | 1 |
| 33 | Flex `%` basis cyclic honesty: definite vs indefinite main size (cyclic → auto) | 1 |
| 34 | Grid `template-areas` / `grid-area` names + `grid-auto-flow: dense` | 1 |
| 35 | Grid `minmax` / intrinsic measure lite + subgrid copy-inherit + masonry pack | 1 |
| 36 | Nested HTML HF: flex header + image + `#target` GoTo; placeholder footer (companions `fixture-36-header.html` / `fixture-36-footer.html`) | 1 |
| 37 | CSS `orphans`/`widows` parse + Rule 3 keep-together (tier-2-pending-3) | ≥2 |
| 38 | Float inside `td`: icon float + wrap + clear; table after float clears below (tier-2-pending-3) | 1 |
| 39 | CSS multicol lite: `column-count`/`gap`/`span`/`fill`; column boxes stay on one page (tier-2-pending-3) | ≥2 |
| 40 | Static 2D CSS `transform` badge (rotate) + abspos CB under transform (tier-2-pending-3) | 1 |
| 41 | `:has()` relational selector: article footnote border, `tr:has(td.neg)` row highlight (tier-2-pending-3) | 1 |
| 42 | `@container` size lite: named `inline-size` + unnamed `width` queries (tier-2-pending-3) | 1 |
| 43 | Self-contained five-page dossier: embedded images, flex/grid cards, floats, multicol prose, anchors, `:has()`, repeating tables, code, positioning, forms, and print styling | 5 |
| 44 | Receipt: compact paid transaction with line items, tax, and total | 1 |
| 45 | Purchase order: supplier/delivery blocks, item table, terms, and approvals | 1 |
| 46 | Contract: numbered clauses, obligations, commercial terms, and signatures | 1 |
| 47 | Certificate: ornamental background image, centered award composition, and signatures | 1 |
| 48 | Shipping document: addresses, package table, tracking, and handling instructions | 1 |
| 49 | Night train poster: generated art, free generic fonts, layered composition | 1 |
| 50 | Letter template: local mark, stationery hierarchy, quote, and signature | 1 |
| 51 | Original Asteria storybook: page illustrations, live text, and page breaks | 4 |
| 52 | Airline boarding pass: e-ticket itinerary, multi-column stubs, mono barcodes | 1 |
| 53 | Asteria poster variant: shared theme with a different illustration and copy | 1 |
| 54 | Ember Harbor storybook: cover + three chapter pages, shared `theme-print-stories.css`, local illustrations (needle `Ember Harbor`) | 4 |
| 55 | Self-contained operations brief: inline CSS, status cards, route table, action plan, and page breaks | 3 |
| 56 | Architecture diagram: hero, pipeline strip, TOC, 10 domain sections (modern semantic tags: `dialog`, `details/summary`, `mark`, `meter`, `progress`, `output`, `time`, `data`, `kbd`, `samp`, `var`, `dfn`, `cite`, `ruby`, `rt`, `rp`, `bdi`, `bdo`, `wbr`, `ins`, `del`, `sub`, `sup`, `aside`, `address`, `fieldset`, `legend`, `picture`, `search`; modern CSS: `oklch()`/`color-mix()`/`clamp()`/logical properties with graceful-degrade fallbacks), linked `fixture-56-architecture-diagram.css`, dependency DAG, PDF-vs-image, security; derived from `documentation/architecture/` (commit ef526f9) | 21 |
| 57 | Vanguard telemetry audit narrative plus browser-print probe gallery for 356 CSS properties; each probe carries its current catalog status from `testdata/css/catalog/properties.json` and representative print styling (`text-wrap-style` is partial with balance only; `box-decoration-break` is partial and storage-only; the GCPM `bookmark-*`, `footnote-*`, and `string-set` probes are unsupported); needle `VANGUARD-CSS-356-IMPLEMENTED`. | 9 |
| 58 | Unsupported CSS worklist audit: safe parsing, cascade degrade, and crash resilience verification gallery for 462 CSS properties; each probe carries its current catalog status from `testdata/css/catalog/properties.json` (unsupported, partial, intentionally ignored, or implemented); needle `UNSUPPORTED-WORKLIST-AUDIT`. | 9 |
| 59 | Apex Digital landing template: hero grid, auto-fit features/pricing/gallery grids, CSS variables, local asset images, print pagination (Chrome-parity target) | multi-page |
| 60 | Implemented CSS props audit slice A (~1/3): property/description/effect gallery; Liberation via `--font-path testdata/fonts/implemented-audit`; needle `IMPLEMENTED-PROPS-A`. | 8 |
| 61 | Implemented CSS props audit slice B (~1/3): same pattern; needle `IMPLEMENTED-PROPS-B`. | 8 |
| 62 | Implemented CSS props audit slice C (~1/3): same pattern; needle `IMPLEMENTED-PROPS-C`. | 8 |
| 63 | Page-level demos: `break-before`/`break-after`, `break-inside: avoid`, inert footnote properties, root `color-scheme: dark` | multi-page |
| 64 | Next 72 CSS props target audit for v0.2.7 (`plans/0.2.7/next-72-properties.json`); property/description/effect gallery; needle `NEXT-72-PROPS`. | 5–8 |
| font-examples | Font showcase: 1,125 free Google Fonts (fonts.google.com Feeling/Calligraphy filters + top-trending modern/display/script/handwriting) - randomized sampler: every font appears exactly once, each line in a random text style (regular, bold, italic, bold-italic, underline, strikethrough, underline+strikethrough, bold+underline, bold-italic+underline+strikethrough, letter-spaced, uppercase), rows span 100% width in a single column; inline `<style>`; fonts intentionally NOT bundled - render with `--font-path <dir>` or `Global().Set("fontpath", dir)`; falls back to Liberation Sans without font flags | 25 (with fonts, single column, number+name inline, overflow-wrap) |

## How the fixtures are used now

The output is `layout.DisplayList`, so there is no generated PDF to parse and
no pass/fail contract on fixture bytes. The corpus serves:

1. **Focused regression tests:** `internal/layout` tests load individual
   fixtures (for example `TestFixture16HeaderBG` in `fixture_bugs_test.go`).
2. **Manual inspection:** each header states what the fixture is meant to
   prove, and the `Expected:` line records the author's intent.

`make golden` gates the public drawing-list tests in `./layout`; it does not
walk this directory. If a layout change moves a fixture, add or update the
focused test that covers the behavior.

## How to run

```sh
make golden    # public drawing-list tests (TestDisplay*) in ./layout
```

`make golden-update` is a tombstone: it exits 2 and points at the removed
page encoder. There is no golden output to update. The old writer filled
`testdata/golden/out/` with PDFs; that directory is gitignored and now holds
leftovers.

The fixtures (HTML/CSS/PNG) are committed as renderer inputs. Nothing under
this directory is a stored baseline.

## Tolerance policy

Focused layout tests document their own tolerances. There is no byte-level
comparison, no stored PDF baseline, and no page-envelope table: the drawing
list is the output.
