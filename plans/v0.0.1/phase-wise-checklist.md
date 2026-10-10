# 0.0.1 - Remove the PDF writer

> **Parent:** none. `plans/` was empty on 2026-10-08. This file is the only 0.0.1 ledger.
> **Status:** audited 2026-10-10 against the wave G tree and rechecked after wave H. The PDF writer, page rasterizer, `cmd/blinkless`, `internal/pdf`, `internal/pdfprofile`, `internal/cli`, `internal/app`, `internal/imageout`, and the compliance tooling are gone; the module rename landed. Closed rows carry inline evidence from those audits; retired rows name the removed subsystem and the reason. Open rows: font-test leftovers, the convert orchestration test, binding PDF stubs, `settings.PdfGlobal` keys, `Result.HasFragmentLinks`, Makefile and script cleanup, and the Python exception name.
> **Estimated effort:** ten build-green commits, in the order below. Each phase leaves `go build ./...` working.

---

## Follow-up checklist

[HTML, CSS, and JSON compatibility](html-css-json-compatibility-checklist.md) owns the findings from the 2026-10-08 repository review. It stays under 0.0.1 and has its own implementation gates. Use its final behavior and coverage evidence when closing related documentation rows here.

## Overview

Blinkless keeps the code that turns HTML into a drawing list. That list is `layout.Op` in `internal/layout/layout.go`. One op is a filled rectangle, a line of text, an image, or a link box. Coordinates are canvas points, y down. `Op.Font` is `*pdf.Font` (`internal/layout/layout.go`, field on `Op`).

A later step turns that list into a PDF file. `PaintContext` (`internal/layout/paint.go`) takes `*pdf.Document` and the list, splits the list onto page boxes, and writes PDF drawing commands onto each page. `pdfPipeline.Finalize` (`internal/convert/pdf_pipeline.go`) then calls `run.doc.Write`. That write, and the code that exists only to feed it, comes out.

The public `layout` package already stops at the list. Its comment says pagination and PDF writing are not part of that package (`layout/doc.go`). `screen.Render` and `ImageDocument` draw the same ops into a PNG. They stay.

The font code cannot leave with the writer. The PNG path loads faces, looks up fonts, and turns characters into glyph positions through `internal/pdf`. `internal/fonts` today only decodes a downloaded font and hands the bytes to `internal/pdf` (`internal/fonts/fonts.go`). Phase 3 moves the face into `internal/fonts`. Phase 6 then deletes `Paint`.

The word blinkless does not appear in this tree yet. The module line is still `github.com/chinmay-sawant/blinkless` (`go.mod` line 1). `VERSION` and `RELEASE.md` are already absent. Renaming is the last phase, after the PDF files are gone, so a replace does not rewrite files that are about to be deleted.

## Executive summary

Ten read-only passes covered the writer, the convert job, layout paint, the public API, the CLIs, the bindings, the tests, the docs, the old names, and the import list. They agree on one cut.

| Keep | Remove |
|---|---|
| Parse, CSS, box placement, the drawing list | `Paint` / `PaintContext` / `PaintBand` writing into `*pdf.Document` |
| `screen.Render`, `ImageDocument`, the image CLI | PDF page splitting that only `PaintContext` runs |
| `internal/convert/prepare` and `internal/convert/render` (the image path calls both) | `pdfPipeline`, TOC pages, PDF outlines, link annotations, copies, header/footer chrome on PDF pages |
| Font face, font lookup, glyph positions, default font bytes, after they move to `internal/fonts` | PDF objects, xref, content streams, `/FontFile2`, tagged structure, PDF version and profile |
| HTML fixtures under `testdata/golden/*.html` | Golden PDF contract, `output/*.pdf`, veraPDF, the PDF CLI |

`internal/pdf` is both things at once. Deleting the directory in one step removes the font code the PNG path still calls. Move the font code first.

Close a row only when its proof command exits 0. A row stays `[ ]` until then. `[~]` means deferred, with the reason on the row.

## Phase 1: PDF tests and sample PDFs

Proof for the phase: `go build ./...` still succeeds, and `rg -n '%PDF-|FontFile2|ParseSemantic' --glob '*_test.go'` no longer hits the files this phase deletes. Do not delete a layout test here if it calls `Paint` only to check box positions. Those move in phase 6.

### 1.1 Writer tests

- [x] Delete PDF byte tests under `internal/pdf/`: `pdf_test.go`, `pdf20_test.go`, `policy_test.go`, `compliance_test.go`, `structure_test.go`, `struct_test.go`, `semantic_oracle_test.go`, `semantic_converted_test.go`, `page_ops_test.go`, `content_blend_test.go`, `content_lifetime_test.go`, `flate_parallel_test.go`, `flate_serial_test.go`, `flate_release_test.go`, `image_test.go`, `image_pixels_test.go`, `text_autospace_test.go`, `subset_align_test.go`, `review_regression_test.go`. Expected: nothing in these files survives, they assert headers, xref, profiles, XObjects, or structure trees. (closed 2026-10-10: `ls internal/pdf` -> "No such file or directory", exit 2; `rg -n '%PDF-|FontFile2|ParseSemantic' --glob '*_test.go'` -> no hits, exit 1.)
- [ ] Split `internal/pdf/font_test.go`, `fonttype0_test.go`, `faces_test.go`, `shape_test.go`, `shape_language_test.go`, `bench_test.go`. Delete the cases that embed a font or call `Content` / `Write`. Keep face parse, advance, registry, and `BenchmarkShapeRun` until phase 3 moves them. (remainder 2026-10-10: the writer cases are gone with `internal/pdf`; face, advance, and registry tests live in `internal/fonts` and pass (`go test -p 2 -parallel 2 ./internal/fonts -run 'TestShapeRunKeepsTextAndAdvancesAligned|TestInstanceWghtChangesAdvanceAndOutline' -count=1` exit 0). `BenchmarkShapeRun` is absent from the tree (`rg -n 'BenchmarkShape'` -> no hits, exit 1); restore it or record the drop.)
- [ ] Keep, and move in phase 3, the face tests: `woff_test.go`, `font_file_cache_test.go`, `font_lazy_test.go`, `font_resource_regression_test.go`, `font_instance_test.go`, `cpal_test.go`, `registry_test.go`, `registry_lookup_test.go`, `registry_bench_test.go`. (remainder 2026-10-10: eight of nine moved to `internal/fonts` (all but `font_lazy_test.go`); `font_lazy_test.go` has no counterpart (`rg -il 'lazy' internal/fonts/*_test.go` -> no hits, exit 1); confirm intended.)
- [x] Delete `internal/pdfprofile/profile_test.go` with the package in phase 8. It parses `PDF/A` and `PDF/UA` strings only. (closed 2026-10-10: `ls internal/pdfprofile` -> "No such file or directory", exit 2.)

### 1.2 Convert and golden tests

- [x] Delete `internal/convert/golden_test.go` and `compliance_golden_test.go`. `fixturePageBounds` is at `golden_test.go` around line 243. `TestGoldenCorpusAllFixtures` checks `%PDF-`, xref, `/FontFile2`, and `pdf.ParseSemantic`. That contract measures a PDF file, not the drawing list. (closed 2026-10-10: `internal/convert` holds only `prepare/` and `render/`; `rg -n 'fixturePageBounds'` hits only `AGENTS.md` and this ledger.)
- [x] Delete the other convert tests that only check PDF bytes: `convert_test.go`, `headers_toc_outline_links_test.go`, `hf_links_test.go`, `quality_test.go`, `web_fixtures_test.go`, `fontface_test.go`, `wk_compare_test.go`, `fuzz_test.go`, `cancellation_test.go`, `perf_test.go`, `perf3_assemble_test.go`, `perf3_page_blocks_test.go`, `repeated_resources_bench_test.go`, `header_footer_bench_test.go`. (closed 2026-10-10: no `.go` files at the `internal/convert` top level.)
- [x] Delete `internal/convert/fixturetests/` PDF oracles: `output_ops_test.go`, `output_fixture_shape_test.go`, `output_fixture_01_test.go` through `output_fixture_64_next_72_props_test.go`, `pixel_regression_test.go`, `pixel_regression_cases_test.go`, `pixel_regression_corpus_test.go`, `fixture_helpers_test.go`. They parse `output/*.pdf` or rasterize it with Ghostscript. (closed 2026-10-10: directory absent.)
- [x] Trim mixed convert tests. Drop the `runPDF` / `%PDF-` cases from `page_first_test.go`, `page_side_test.go`, `page_named_test.go`, `perf3_nav_test.go`, `perf3_smartshrink_test.go`, `simplify_test.go`, `import_stylesheet_test.go`, `seams_test.go`, `benchmarks_test.go`. Keep prepare checks and `layoutBody` checks until phase 5 deletes that function. (closed 2026-10-10: named files are gone with the top-level convert package.)
- [ ] Keep `internal/convert/prepare/*_test.go`, `render/pipeline_test.go`, `orchestration_contract_test.go`. `render/plan_test.go` goes in phase 5 with `render/plan.go` (copy and collate indexes). (remainder 2026-10-10: prepare tests and `render/pipeline_test.go` are present and pass (`go test -p 2 -parallel 2 ./internal/convert/prepare -count=1` exit 0; `go test -p 2 -parallel 2 ./internal/convert/render -count=1` exit 0); `orchestration_contract_test.go` is absent (`rg -ni 'orchestration' internal/convert` -> no hits, exit 1); confirm whether it was dropped or needs restoring.)

### 1.3 Root, CLI, chrome, binding tests

- [x] Delete PDF cases in `document_render_test.go`, `document_bench_test.go`, `document_bench_validate_test.go`, `document_perf3_pin_test.go`, `document_bench_html_test.go`. Keep the PNG cases (`BenchmarkLibraryImage`, image validation). (closed 2026-10-10: the root package holds only `doc.go`; named files absent. The PNG keep-list left with the root document API removal, see 7.1.)
- [x] retired: Delete `document_test.go` cases that map `Document`, `PDFVersion`, `PDFProfile`, margins, and copies. Keep `TestContentValidate`, `TestHTMLCopiesBytes`, and the `ImageDocument` cases. (retired 2026-10-10: the root `Document` / `ImageDocument` API was removed entirely; the keep-list has no home, so the row is moot.)
- [x] Delete `cmd/blinkless/main_test.go` and `internal/app/pdf_test.go` with those programs in phase 2. (closed 2026-10-10: `cmd/` is empty; `internal/app` absent.)
- [x] Delete `test/chrome/pdf_output_test.go` and `profile_bench_test.go`. Keep `test/chrome/manifest_test.go` and the HTML cases under `test/chrome/cases/`. (closed 2026-10-10: both absent; `test/chrome/manifest_test.go` present.)
- [ ] Delete PDF assertions in `bindings/c/cshared_test.go` (`requirePDFShape`), `bindings/wasm/contract_test.go` PDF mode, `bindings/wasm/fixture_test.go` (`assertFixturePDF`), `bindings/python/tests/test_binding.py` PDF cases, `bindings/python/tests/bench_library.py` `bench_pdf`, and `bindings/python/examples/invoice.py`. Keep the PNG cases. (remainder 2026-10-10: `cshared_test.go` and `fixture_test.go` are gone and `bindings/wasm/contract_test.go` has no PDF mode (`rg -ni 'pdf' bindings/wasm/contract_test.go` -> no hits, exit 1); `test_binding.py` now asserts the removal stubs (`test_pdf_entry_points_report_removed`, `bindings/python/tests/test_binding.py:65`). Still present: `bindings/python/tests/bench_library.py:190` (`bench_pdf` asserts `%PDF-`) and `bindings/python/examples/invoice.py` (`convert_html_to_pdf` at line 22).)

### 1.4 Sample files

- [x] Delete committed PDF samples under `output/` (`output/*.pdf`, `output/pdf-1.7/`, `output/pdf-2.0/`, `output/pdf-1.7-compliance/`, `output/pdf-2.0-compliance/`, `output/python/`, `output/wkhtmltopdf/`, `output/validated/`). `output/README.md` calls these viewer-smoke files. `fixturetests` also treats `output/validated/` as a Ghostscript oracle. The HTML inputs under `testdata/golden/` stay. (closed 2026-10-10: `find output -name '*.pdf'` -> no output, exit 0. The directories are empty or hold documented non-PDF snapshots; `output/README.md` describes the leftovers.)
- [ ] Delete `testdata/golden/benchmarks/output/*.pdf`. Keep the `.html.tmpl` templates and any PNG bench images. (remainder 2026-10-10: 19 PDFs remain on disk (`find testdata/golden/benchmarks/output -name '*.pdf' | wc -l` -> 19); no `.gitignore` rule covers them.)
- [x] retired: Delete `compliance/fixtures/*.pdf`. Delete generated trees `test/chrome/pdf/` and `test/chrome/cases/pdf/` if present. They are gitignored outputs. (retired 2026-10-10: `compliance/` is gone; the two chrome trees are gitignored local outputs (`.gitignore:67-68`) whose stale copies are documented in `test/chrome/README.md:126-128`; nothing committed remains.)
- [x] Leave `testdata/golden/*.html`, `testdata/golden/assets/`, `testdata/web/*.html`, and `test/chrome/cases/*.html`. Those are renderer inputs. (closed 2026-10-10: all present; `go test -p 2 -parallel 2 ./layout -run 'TestDisplay' -count=1` exit 0.)

## Phase 2: PDF programs

Proof: `go build ./...` succeeds. `cmd/blinkless` still builds. Nothing in `cmd/` imports `internal/pdf`.

### 2.1 Commands and examples

- [x] Delete `cmd/blinkless/` (`main.go` calls `cli.ModePDF` and `app.RunPDF`). Delete `examples/pdf/`. (closed 2026-10-10: `cmd/` and `examples/` are empty.)
- [x] Delete `internal/app/pdf.go` symbols `RunPDF`, `BuildPDFRequest`, `DefaultTOCXSL`. Keep the shared sentinels in that file if `internal/app/image.go` still uses them (`ErrNilCommand`, `ErrNilContext`). Move those sentinels next to `image.go` if the file would otherwise be empty. (closed 2026-10-10: `internal/app` is absent.)
- [x] retired: Keep `cmd/blinkless/` and `internal/app/image.go`. They import `internal/cli` and `internal/imageout`, not `internal/pdf`. The rename of the binary is phase 10. (retired 2026-10-10: both directories were removed with the CLI; no image binary ships.)

### 2.2 CLI flags that only exist for the PDF file

- [x] Delete `ModePDF` flags in `internal/cli/flags.go`: `--pdf-version`, `--pdf-profile`, `--copies`, `--collate`, `--grayscale`, `--title`, `--no-pdf-compression`, `--page-offset`, `--outline`, `--outline-depth`, `--exclude-from-outline`, `--dump-outline`, `--dump-default-toc-xsl`, `--cover`, `--toc`, `--external-links`, `--internal-links`, header and footer flags, TOC flags. Drop the matching parser tests in `internal/cli/cli_test.go`. (closed 2026-10-10: `internal/cli` is absent.)
- [x] retired: Keep `ModeImage` flags and the shared flags the image binary already accepts (`--html`, `--url`, `--allow`, `--font-path`, `--zoom`, cookies, media type). Keep `TestImageGrammarAndOptions`. (retired 2026-10-10: `internal/cli` is absent and `TestImageGrammarAndOptions` is gone (`rg -n 'TestImageGrammarAndOptions'` -> no hits, exit 1).)
- [x] Page size, orientation, and margins are layout inputs today and PDF CLI flags. After phase 5 nothing in the image path reads them from `settings.PdfGlobal`. Delete the flags with `ModePDF`. Keep `settings` page-size parsers only if a layout caller remains. Check `internal/imageout` before deleting `ParsePageSize`. (closed 2026-10-10: the flags died with `internal/cli`; `ParsePageSize` survives only as the live `size.pagesize` settings-key validator (`internal/settings/reflect.go:694-698`) and its unit test; no layout caller remains. The settings surface is tracked by the Phase 8 row.)

## Phase 3: Move the font code out of the PDF package

Proof: `go test ./internal/fonts ./internal/pdf -count=1` (single packages, not `./...`). Face tests pass from `internal/fonts`. `internal/pdf` still builds because of temporary aliases.

The PNG path calls `pdf.DefaultFont`, `pdf.Registry`, `pdf.RegistryFromGlobal`, `pdf.ShapeRunWithFeaturesLanguage`, and glyph contour helpers (`internal/imageout/imageout.go`, `frame.go`, `ttfraster.go`). Layout measures with `pdf.Font` (`internal/layout/inline_paint.go`, `layout_flow.go`). Public `css.Document.registry` is `*pdf.Registry` (`css/css.go`).

### 3.1 Move

- [x] Move these files from `internal/pdf/` into `internal/fonts/`: face and metrics (`fonts.go` minus `PDFAscent`, `PDFDescent`, `PDFCapHeight`, `PDFBBox`), `faces.go`, `assets/` plus `assets.go`, `registry.go`, `font_file_cache.go`, `woff.go`, `shape.go`, `shape_gotext.go`, `glyph.go`, `font_instance.go`, `cpal.go`, `numbers.go` (sfnt constants). Move the face tests listed in phase 1.1 with them. (closed 2026-10-10: `internal/fonts` holds the moved files (`faces.go`, `registry.go`, `font_file_cache.go`, `woff.go`, `shape.go`, `shape_gotext.go`, `glyph.go`, `font_instance.go`, `cpal.go`, `numbers.go`, `assets/`; `assets.go` content merged, `decode.go` / `sfnt.go` / `woff2.go` added); `internal/pdf` is absent.)
- [x] retired: Leave a temporary alias in `internal/pdf` (`type Font = fonts.Font`, same for `Registry`, `FaceSet`) so current imports still compile. Delete the aliases in phase 8. (retired 2026-10-10: `internal/pdf` was deleted entirely in phase 8; the aliases never outlived it.)
- [ ] `RegistryFromGlobal` takes `settings.PdfGlobal` (`internal/pdf/registry.go`). Change it to take the font paths and the system-font flag, or a small struct in `internal/settings` that is not named for PDF. `imageout` is the caller that must keep working. (remainder 2026-10-10: still takes `settings.PdfGlobal` (`internal/fonts/registry.go:440`), called from `css/css.go:167` (`DefaultPdfGlobal`) and `internal/fonts/registry_lookup_test.go:38`; retype it with the Phase 8 settings work.)
- [x] Update `internal/fonts/fonts.go` package comment. It currently says decoded bytes are handed to `internal/pdf`. (closed 2026-10-10: no package comment remains (`rg -n 'Package fonts' internal/fonts` -> no hits, exit 1); the stale claim is gone.)

## Phase 4: Point the renderer at internal/fonts

Proof: `go build ./...`. `rg 'internal/pdf' internal/imageout internal/layout css internal/pubstate internal/convert/prepare` shows font aliases only, no `pdf.Document` outside paint and convert.

- [x] Retype font fields in the measurement files: `internal/layout/layout.go` (`Op.Font`), `layout_context_styles.go`, `layout_section.go`, `inline_paint.go`, `inline_text_box.go`, `font_palette.go`, `font_synthesis_geom.go`, `font_variation.go`, `style_font_size_adjust_props.go`. (closed 2026-10-10: no `internal/pdf` import remains (`rg -n 'internal/pdf' internal/layout layout --glob '*.go'` hits two stale comments only, at `style_font_variant_props.go:378` and `style_font_variant_props_test.go:135`); the files import `internal/fonts`.)
- [x] retired: Retype `internal/imageout/imageout.go`, `frame.go`, `ttfraster.go`. (retired 2026-10-10: `internal/imageout` was removed with the page rasterizer; nothing to retype.)
- [x] Retype `css/css.go` and `internal/pubstate/state.go`. (closed 2026-10-10: both import `internal/fonts` as `pdf` (`css/css.go:12`, `internal/pubstate/state.go:9`).)
- [x] Retype `internal/convert/prepare/prepare.go` and `prepare/styles.go` (`pdf.ParseFontBytes` at `styles.go` around line 531). `Prepared.Registry` becomes the fonts package registry. (closed 2026-10-10: both import `internal/fonts` as `pdf` (`prepare.go:15`, `styles.go:12`); `ParseFontBytes` at `styles.go:530`; `go test -p 2 -parallel 2 ./internal/convert/prepare -count=1` exit 0.)
- [x] Leave `Paint` importing `*pdf.Document` until phase 6. This phase does not delete paint. (closed 2026-10-10: `Paint` was deleted in phase 6; no writer import remains.)

## Phase 5: Delete the PDF job in convert

Proof: `rg 'pdfPipeline|PaintContext|NewPDFRequest|convert.Run' --glob '*.go'` finds no production caller. `go test ./internal/imageout ./internal/convert/prepare ./internal/convert/render -count=1` passes. Image mode never called `convert.Run`. Its entry is `imageout.RunRequest` (`internal/imageout/imageout.go`).

Body paint ends in `renderObject` (`internal/convert/convert.go`), which calls `layout.PaintContext`. PDF assembly starts at `pdfPipeline.Assemble` (`internal/convert/pdf_pipeline.go`). `Finalize` writes the file at the `run.doc.Write` call.

### 5.1 Delete these files

- [x] `internal/convert/pdf_pipeline.go` (`Assemble`, `Finalize`, TOC, outline, links, copies, header/footer, `Write`). (closed 2026-10-10: absent.)
- [x] `internal/convert/toc.go`, `outline.go`, `links.go`, `hf.go`, `hf_geometry.go`, `page_plan.go`, `page_margin_boxes.go`, `page_blocks.go`. (closed 2026-10-10: absent; `internal/convert` holds only `prepare/` and `render/`.)
- [x] `internal/convert/render/plan.go` and `plan_test.go`. Image mode does not call them. Keep `internal/convert/render/pipeline.go`. `imagePipeline.Assemble` is a no-op and `RunRequest` calls `render.Run`. (closed 2026-10-10: absent; `render/pipeline.go` and `pipeline_test.go` remain and pass.)
- [x] The PDF half of `internal/convert/convert.go`: `NewPDFRequest`, `Run`, `policyForProfile`, `PolicyForGlobal`, `renderObject`, `layoutBody`. Keep nothing in this file unless a symbol is still referenced by `prepare` or tests. If the file becomes empty, delete it. (closed 2026-10-10: no `.go` files at the `internal/convert` top level.)
- [x] `internal/outline/` including `outline_test.go`. The package builds a heading tree and dump XML. The only callers are convert PDF assembly. It does not import `internal/pdf`, and `imageout` does not import it. (closed 2026-10-10: absent.)
- [x] Keep `internal/line/`. `imageout` calls `line.Emit`. (closed 2026-10-10: present.)

### 5.2 What convert must still export

- [x] Keep `internal/convert/prepare/` (`Document`, `BuildOptions`, sheet collection, `@font-face`). `prepareImageDocument` calls it. (closed 2026-10-10: present with six test files; `go test -p 2 -parallel 2 ./internal/convert/prepare -count=1` exit 0.)
- [x] Keep `internal/convert/render/pipeline.go` and its test. Delete the PDF adapter, not the interface. (closed 2026-10-10: present; `go test -p 2 -parallel 2 ./internal/convert/render -count=1` exit 0.)

## Phase 6: Delete PDF drawing inside layout

Proof: `rg 'func Paint|PaintContext|pdf.Document|StructElem' internal/layout` is empty. `go test ./layout ./screen ./internal/imageout -count=1` passes. Public `layout.DisplayList` still returns ops.

`Paint` (`internal/layout/paint.go`) paginates the list and paints it into `doc`. The page-break, repeated table header, and orphan passes run only from `PaintContext`. `screen.Render` calls `layout.Lay`. `Lay` calls `imageout`, which rasterizes `Ops` and does not call `Paint`.

### 6.1 Delete the writer entry

- [x] Delete `Paint`, `PaintContext`, `PaintBand`, `PaintBandContext`, and the operator emitters in `paint.go` (`drawFill`, `drawStroke`, `drawLine`, `drawText`, `drawImage`, `drawLinkXform`, `canvasToPDF`). Keep `StyleOf`, `FakeBoldFor`, and `PaintLineGeometry`. `imageout` calls those. (closed 2026-10-10: none of the named symbols exist (`rg -n 'func Paint|PaintContext|PaintBand|drawFill|drawStroke|drawLinkXform|canvasToPDF' internal/layout` hits only the surviving `PaintOrder` helper at `paint_order.go:14` and a stale comment at `layout_measure.go:1208`); `StyleOf` / `FakeBoldFor` / `PaintLineGeometry` live in `paint_style.go` / `paint_geom.go`.)
- [x] Delete `internal/layout/tagging.go` (`buildStructureTree` writes `*pdf.StructElem`). (closed 2026-10-10: absent.)
- [x] Delete `internal/layout/paint_groups.go` (PDF form XObjects). Keep `BlendGroup` in `blend_group.go`. (closed 2026-10-10: absent; `blend_group.go` present.)
- [x] Delete `drawGridRun` in `grid_run.go` and `pdfCTMFromCSS` in `transform.go`. Keep the rest of both files. They build ops. (closed 2026-10-10: no hits; both files remain.)
- [x] Remove `StructElem *pdf.StructElem` from `opExtra` (`internal/layout/op_extra.go`). Keep URI, image bytes, transform, and blend on the op. `OpLinkURI` stays as a hit target. It is not a PDF annotation. (closed 2026-10-10: no `StructElem` hits in `internal/layout`.)

### 6.2 Delete PDF page splitting

- [x] Delete the files that only rewrite ops onto a PDF page height: `paint_pagination_fixpoint.go`, `paint_pagination_split.go`, `paint_pagination_chrome.go`, `paint_pagination_seal.go`, `paint_flow_breaks.go`, `paint_flow_orphans.go`, `paint_flow_tables.go`, `paint_flow_index.go`, `page_named.go`. Their only production caller is `PaintContext`. (closed 2026-10-10: none of the named files exist.)
- [x] In `sticky.go`, keep `tagSticky` (it stamps `Op.StickyID` during layout). Delete `applyStickyPrint`. (closed 2026-10-10: `tagSticky` at `sticky.go:14`; `applyStickyPrint` absent.)
- [x] Keep multicol's use of `Options.Height` as a column height (`internal/layout/multicol.go`). `imageout` passes viewport height there, so the image changes if that snap goes away. (closed 2026-10-10: `internal/layout/multicol.go:360` reads `e.opts.Height`.)
- [x] Stop filling `Result.Pages` and `Result.Locations` from paint. `PlacedElements` (`internal/layout/placed.go`) is the box list that does not need `Paint`. Keep it. (closed 2026-10-10: nothing fills them; only the `CloneResult` copies (`layout.go:216-217`) and the nil resets (`layout.go:328-329`) touch them; `PlacedElements` at `placed.go:20`. The vestigial fields themselves are not part of this row.)
- [ ] Delete `HasFragmentLinks` on `Result`. It exists so the PDF link pass can find `#` URIs. The URI on the op stays. (remainder 2026-10-10: still declared (`layout.go:197-198`) and set (`layout.go:1199`, `layout_section.go:126`); no PDF link pass consumes it.)

### 6.3 Layout tests

- [x] Delete tests that read a PDF content stream or a structure tree: `tagging_test.go`, the `%PDF-` cases in `layout_test.go` (`TestPaginateAndPaint`, `TestPaintSinglePage`), `rounded_border_test.go`, the stream searches in `color_adjust_paint_test.go` and `blend_test.go`, and the structure-tree cases in `architecture_followup_test.go`. (closed 2026-10-10: all named files and tests are absent (`rg -n 'TestPaginateAndPaint|TestPaintSinglePage' internal/layout layout` -> no hits, exit 1).)
- [x] Rewrite the geometry tests that currently call `pdf.NewDocument` and `Paint` to assert `Op` fields or `PlacedElements`. Files the survey marked as mixed include `transform_test.go`, `flex_test.go`, `grid_test.go`, `multicol_test.go`, `orphans_widows_test.go`, `pagination_ctx_test.go`, `pagination_thead_test.go`, `sticky_test.go`, and the `fixture56_*` / `flex_chrome_*` paint tests. A test that only checked which PDF page an op landed on is deleted with phase 6.2. A test that checks box x/y stays. (closed 2026-10-10: no test calls `pdf.NewDocument` or `Paint`; the only `internal/pdf` hits in `internal/layout` are comments.)
- [x] `layout/*_test.go` (the public package) does not import `internal/pdf`. Those tests stay. (closed 2026-10-10: no hits; `go test -p 2 -parallel 2 ./layout -run 'TestDisplay' -count=1` exit 0.)

## Phase 7: Public PDF API and binding PDF symbols

Proof: `rg 'WritePDF|PDFProfile|html_to_pdf|convert_html_to_pdf' --glob '*.go' --glob '*.py' --glob '*.h'` is empty outside docs. `go build ./...` succeeds. Image symbols still build.

### 7.1 Root library

- [x] Delete `Document`, `NewDocument`, `WritePDF`, `WritePDFOutline`, `PDF`, and `Validate` on that type (`document.go`, `document_validate.go`). Delete `Page`, `TOC`, `HeaderFooter`, `Margin` if nothing image-side uses them. (closed 2026-10-10: the root package holds only `doc.go`; `rg -n 'WritePDF|func Version|LibraryVersion|executePDFTo|toPDFRequest|pdfGlobal' --glob '*.go'` -> no hits, exit 1.)
- [x] Delete PDF errors and `executePDFTo` / `toPDFRequest` / `pdfGlobal` (`api.go`, `document.go`). Delete `LibraryVersion` (`0.12.7-dev`) and `Version()`. That string is the wkhtmltopdf settings id, not this product's version (`api.go`). (closed 2026-10-10: no Go symbol remains; the `0.12.7-dev` string survives only outside Go, see the name-conversion rows.)
- [x] retired: Keep `ImageDocument`, `WriteImage`, `Image`, `Content`, `HTML`, `File`, `URL`, `Crop`, `NetworkPolicy`. (retired 2026-10-10: the public root document API was removed entirely; the keep-list has no home.)
- [x] retired: `ImageDocument.toImageRequest` still builds `settings.PdfGlobal`. Point it at the image settings type so the image API does not mention PDF. (retired 2026-10-10: `ImageDocument` and `toImageRequest` no longer exist; image bytes flow through the drawing list and the bindings.)

### 7.2 Bindings

Do not delete `bindings/c/`, `bindings/python/`, or `bindings/wasm/`. Each binding returns either PDF bytes or PNG/JPEG bytes. Strip the PDF half.

- [x] C: delete `bindings/c/options_pdf.go`, `blinkless_html_to_pdf`, and `GwkPdfOptions` in `bindings/c/include/blinkless.h`. Keep `blinkless_html_to_image`, `options_image.go`, and the free/last-error helpers. The symbol rename is phase 10. (closed 2026-10-10: `options_pdf.go` absent; the exported set is abi/version/html_to_image/free/free_string/last_error_length/last_error (`rg -n '^//export' bindings/c/exports_cgo.go`); the committed header has no PDF symbols (`bindings/c/include/blinkless.h:10`). Leftover noted: a dead `GwkPdfOptions` typedef in `exports_cgo.go:42` with no caller.)
- [ ] Python: delete `Document`, `PDFOptions`, `convert_html_to_pdf`, `convert_file_to_pdf`, `convert_url_to_pdf`. Keep `ImageDocument`, `convert_html_to_image`. (remainder 2026-10-10: all five names survive as raising removal stubs: `convert_html_to_pdf` / `convert_file_to_pdf` / `convert_url_to_pdf` (`api.py:11-31`), `PDFOptions` (`document.py:230`), `Document.pdf` (`document.py:372-382`), `_lib.py:221`; `test_binding.py:65` pins the removal messages. Delete them or record the stub policy as deliberate.)
- [~] WASM request modes and output: moved to [the compatibility checklist, Phase 5](html-css-json-compatibility-checklist.md#phase-5-usable-drawing-list-json-from-wasm). The current adapter returns JSON, and frontend/ is absent. That phase owns drawing-list mode, serialization, fixtures, and consumer proof; this historical PNG/JPEG instruction is superseded.

## Phase 8: Delete the writer package, profiles, and PDF settings

Proof: `rg 'internal/pdf"|internal/pdfprofile' --glob '*.go'` is empty. `go build ./...` succeeds. `make test-quick` is the gate for this phase (capped concurrency, see the Makefile). Record the exit code here when it passes.

- [x] Delete the writer files in `internal/pdf/`: `pdf.go`, `content.go`, `flate_parallel.go`, `text_autospace.go`, `subset.go`, `fonttype0.go`, `fontpdf.go`, `images.go`, `structure.go`, `policy.go`, `metadata.go`, `icc.go`, `outputintent.go`, `semantic.go`, `page_ops.go`, `doc.go`, and the temporary font aliases from phase 3. Delete the directory if it is empty. (closed 2026-10-10: `ls internal/pdf` -> "No such file or directory", exit 2.)
- [x] Delete `internal/pdfprofile/`. (closed 2026-10-10: absent.)
- [ ] Delete PDF settings keys and their setters: `pdfversion`, `pdfprofile`, `usecompression`, `copies`, `collate`, `title`, `grayscale` / `colormode`, `outline`, `outlinedepth`, `excludefromoutline`, `dumpoutline`, `dumpoutlinewithdefaulttocxsl`, `externallinks`, `locallinks`, `resolverelativelinks`, `includeinoutline`, `useoutline`, `pageoffset`, `header.*`, `footer.*`, `toc.*`, `istableofcontent`, `iscover`. Homes: `internal/settings/settings.go` and `reflect.go`. Delete `object_roles.go` cover/TOC stamps. (remainder 2026-10-10: still registered in `internal/settings/reflect.go`: `colormode` (:527), `grayscale` (:537), `pageoffset` (:541), `copies` (:545), `collate` (:549), `outline` (:553), `outlinedepth` (:557), `title` (:580), `dumpoutline` (:568), `dumpoutlinewithdefaulttocxsl` (:572), `usecompression` (:576), `pdfversion` (:601), `pdfprofile` (:620), `excludefromoutline` (:646), `header.*` / `footer.*` (:759-769), `toc.*` (:812-813), `includeinoutline` (:910), `useoutline` (:914), `istableofcontent` (:918), `iscover` (:922). Also `ErrInvalidPDFProfile` / `ParsePDFProfile` (`settings.go:41,64`), the `PdfGlobal` type, and `object_roles.go` `StampCover` / `StampTOC` (:5,:19).)
- [x] Keep load and image settings: allow lists, proxy, cookies, zoom, timeout, font paths, background, media type, image width/height/crop/format/quality. Keep `internal/load`, `internal/html`, `internal/css`, `internal/svg`, `markup`, `screen`. (closed 2026-10-10: `internal/load`, `internal/html`, `internal/css`, `internal/svg`, and `markup` are present; the settings keep-list is present (`Proxy`, `Allow`, `Cookies`, `ZoomFactor`, `Background`, `FontPaths` in `settings.go`; image `Width` / `Height` / `Quality` / `SmartWidth` / `Crop` / `Format` in `ImageGlobal`, `settings.go:662-675`). `screen` was later removed with the page rasterizer (`rg -n 'package screen'` -> no hits, exit 1).)

## Phase 9: Make targets, scripts, compliance

Proof: `make help` or a read of the Makefile no longer lists `golden`, `samples` PDF lines, or `c-shared` under the old PDF library name. `make test-quick` still passes.

- [ ] Delete Makefile targets `golden`, `golden-update`, `chrome-cases-pdf`, `run`, `screenshots`, `weasyprint`, `bench`, `bench-engine`, `bench-inprocess`, `bench-cli-compare`, `python-api`, `samples-python`. Drop `BenchmarkLibraryPDF` from `bench-lib`. Drop `./internal/pdf` from `RACE_PKGS`. (remainder 2026-10-10, wave H recheck: `RACE_PKGS` already dropped `./internal/pdf` (`Makefile:32`); `golden` was repurposed to the drawing-list tests (`Makefile:146-147`), `samples` aliases it (`:154`), and `golden-update` is a tombstone that exits 2 (`:149-151`). Wave H removed `weasyprint`, `python-api`, `samples-python`, `screenshots`, `chrome-cases-pdf`, and `run` (`rg -n '^(weasyprint|python-api|samples-python|screenshots|chrome-cases-pdf|run):' Makefile` -> no hits, exit 1). Still PDF-era: `bench` (`:169`), `bench-engine` (`:182`), `bench-inprocess` (`:188`), `bench-lib` (`:194`), `bench-cli-compare` (`:202`); `bench-engine` and `bench-cli-compare` target `./internal/convert` test files that no longer exist, and `bench-lib` still matches `BenchmarkLibrary(PDF|Image)`.)
- [x] Split `samples`: delete the PDF recipe, keep the PNG recipe that calls the image binary. (closed 2026-10-10: `samples` runs `golden` and writes nothing (`Makefile:156-157`).)
- [x] Split `build`: stop building `bin/blinkless`. Keep the image binary until phase 10 renames it. (closed 2026-10-10: `build` runs `CGO_ENABLED=0 go build ./...` and writes no binary (`Makefile:88-89`); stale gitignored binaries under `bin/` are not produced by any target.)
- [x] Keep `wasm`, `c-shared`, and `python-binding-test` only for the image symbols. Rename artifacts in phase 10. Delete them if phase 7 removed the last exported image symbol by mistake. It should not. (closed 2026-10-10: all three targets present (`Makefile:94`, `:286`, `:305`); the wasm target now builds the drawing-list adapter, and the C entry point reports the image removal.)
- [x] Delete `compliance/` (veraPDF, `verify_pdfs.sh`, `structure_tree_check.py`, fixtures). Delete repo-root `verapdf/` if it is the installed validator tree. (closed 2026-10-10: `compliance/` and `verapdf/` are absent.)
- [ ] Delete `scripts/inspect_pdf_fonts.py`, `scripts/inspect_pdf_ops.py`, `scripts/puppeteer_print.js` if it only prints PDFs, and the PDF half of `scripts/bench-external.sh`. Keep `scripts/check-file-size.sh` and `scripts/pr-diff-stat.sh`. (remainder 2026-10-10, wave H recheck: `inspect_pdf_fonts.py`, `inspect_pdf_ops.py`, `puppeteer_print.js`, `scripts/puppeteer/print.sh`, and `scripts/weasyprint/print.sh` are deleted (`ls` -> No such file or directory, exit 2). Still present: `scripts/bench-external.sh` requires the removed `bin/blinkless` (`:20`, `GOWK_BIN` at `:37`; 11 `pdf` matches) and cannot run; `scripts/bench-performance-recovery.sh` still offers PDF modes (`:10-17`); `scripts/run-walltime.sh:21` writes a PDF; `scripts/compare_chrome_ana.py:44-45` reads `output/*.pdf`; `scripts/screenshot_showcase.py:2` rasterizes `output/*.pdf` into the removed `frontend/` and is orphaned by the removed `screenshots` target.)
- [x] `scripts/check_versions.sh` reads a `VERSION` file that is gone. `release.yml` requires `VERSION` to match the tag. Do not recreate the old PDF version file in this phase. Phase 10 decides the blinkless version source. Until then, the workflow must not fail the build on a missing `VERSION`. (closed 2026-10-10: the script was rewritten to read `bindings/python/pyproject.toml` and `bindings/c/include/blinkless.h` and to accept the tag version as an argument (`scripts/check_versions.sh:20-28`); it no longer reads `VERSION`. `release.yml` passes the tag version (`release.yml:70-72`).)

## Phase 10: Docs, READMEs, release file, and the name blinkless

Proof: `rg -n 'blinkless|blinkless' -g '!frontend/public/data/**' -g '!reports/**'` returns only the lines this phase explicitly keeps (upstream URLs, if any remain). `make claim-scan` exits 0 after the docs match the code. Do not hand-edit `docs/`. It is the built site. Rebuild it from `frontend/` after the source copy changes.

`VERSION` and `RELEASE.md` are already absent. Do not restore the old release checklist.

### 10.1 Docs to delete

- [x] Delete `documentation/architecture/09-pdf-writer.md`. (closed 2026-10-10: absent.)
- [x] Delete `documentation/cli.md`, `documentation/python.md`, `documentation/wasm.md`, `documentation/MIGRATION-0.2.4.md`, `documentation/benchmarks.md`, `documentation/performance.md`, `documentation/samples.md`, `documentation/ghostscript-licensing.md`. (closed 2026-10-10: all eight absent.)
- [x] Delete `documentation/comparison-with-others/` (`sebastiaanklippert-go-wkhtmltopdf.md`, `landscape-2026.md`, `landscape-2026.html`). (closed 2026-10-10: absent.)
- [x] Delete `reports/wkhtmltopdf-open-issues/` and `reports/wkhtmltopdf-open-issues.html`. Delete `frontend/public/data/issues.json` with the dossier page. Those are the upstream issue mirror, not this product. A rename would corrupt them. (closed 2026-10-10: `reports/` holds only `critical-golang-architecture-review.md`; `frontend/` is absent.)
- [x] Delete `skills/chrome-debug/`, `skills/chrome-debug-v2/`, `skills/chrome-flex-pdf-closure/`, `skills/diagnose-golden-fixture/` if they only exist to debug PDF output. Read each skill header before deleting. Keep a skill that diagnoses the drawing list. (closed 2026-10-10: all four skills are present and target the drawing list (`chrome-flex-pdf-closure/SKILL.md:10` says the engine emits a drawing list, not a PDF; `chrome-debug-v2/SKILL.md:47-48`; `diagnose-golden-fixture/SKILL.md:1-3`); none is PDF-only, so the conditional delete resolves to keep.)

### 10.2 Docs to rewrite

Keep the description of load, parse, CSS, layout, and the drawing list. Cut the PDF sink.

- [x] `documentation/overview.md`, `documentation/architecture.md`, `documentation/architecture/README.md`. (closed 2026-10-10: all updated (`overview.md:3` says no PDF files and no page bitmap; `architecture.md:24` says there is no `internal/pdf` package and no PDF command; `architecture/README.md:3`).)
- [x] `documentation/architecture/02-library-api.md` through `08-convert-pipeline.md` and `10-imageout-svg.md`. `07-layout.md` keeps the drawing list. Cut "canvas to PDF" and `Paint` into `pdf.Document`. `08` keeps `prepare`. Cut assemble and `Write`. (closed 2026-10-10: `02`, `03`, `05`, `06`, `07`, `08`, and `10` carry no `pdf.Document` / `PaintContext` / `assemble` / `bin/blinkless` / `ImageDocument` sink text (`rg -ni 'pdf\.Document|PaintContext|func Paint|assemble|bin/blinkless|ImageDocument|canvas to PDF' documentation/architecture/{02-library-api,03-settings,05-html-parser,06-css,07-layout,08-convert-pipeline,10-imageout-svg}.md` -> no hits, exit 1; `08-convert-pipeline.md:3` says `internal/convert` has no PDF job).)
- [x] `documentation/architecture/01-entrypoints-cli.md` becomes the image command, or is deleted if the image command's README is enough. (closed 2026-10-10: the note was repurposed to entry points and opens with "There is no CLI in this tree" (`01-entrypoints-cli.md:3`); the image-command option died with the CLI.)
- [x] `documentation/getting-started.md`, `library-api.md`, `fidelity.md`, `compatibility-matrix.md`, `fonts.md`, `deferred.md`, `THREAT-MODEL.md`, `integration-security.md`, `documentation/README.md`. (closed 2026-10-10: all nine re-checked clean of the removed sink (`getting-started.md:13`, `fidelity.md:3`, `compatibility-matrix.md:5`, `fonts.md:18`, `deferred.md:12`, `THREAT-MODEL.md:3`, `integration-security.md:187`, `documentation/README.md:5`; `library-api.md:53` documents the drawing-list JSON).)
- [x] `CHANGELOG.md`: add an unreleased note that the PDF writer is gone and the drawing list remains. Do not rewrite old release sections into blinkless history. (closed 2026-10-10, wave H recheck: the header names `bindings/python/pyproject.toml` and `bindings/c/include/blinkless.h` as the committed version sources gated by `scripts/check_versions.sh` (`CHANGELOG.md:4-6`); the Unreleased `### Removed` block records the writer, rasterizer, CLI, and frontend removal plus `layout.Lay` / `screen.Render` / `ImageDocument` (`:28-40`). Old release sections are untouched.)
- [x] `CONTRIBUTING.md`: the pipeline stops at the drawing list and the PNG encode. Remove stable-PDF-bytes rules. (closed 2026-10-10, wave H recheck: the guide is refreshed (`CONTRIBUTING.md:3` names the HTML and CSS layout engine; `:15` cites `internal/fonts.TestDirectModuleAllowlist`; `:25-27` makes `make golden` the drawing-list gate and demotes the `output/` rasters; `:103` says the pipeline has no page raster and no PDF; `:212-213` documents `output/` as leftovers). `rg -ni 'pdf|writer' CONTRIBUTING.md` now hits only accurate statements.)
- [x] `AGENTS.md`: product name blinkless, pipeline stops at the drawing list, drop `make golden` as a release gate, drop the PDF writer package from the map. Leave the no-git and capped-test rules. (closed 2026-10-10: the file names blinkless, stops the pipeline at the drawing list, and carries the no-git and capped-test rules (`AGENTS.md:12-24`, `:140-146`, `:154-169`); `make golden` is a verification gate for the drawing-list tests, not a writer-era release check.)

### 10.3 README files

- [x] `README.md` lines 2-14 name `blinkless`, the wkhtmltopdf CLI, and a pipeline that ends in write. Rewrite the title, the first paragraph, and the binary list for blinkless. Describe the drawing list and the PNG path. Remove the wkhtmltopdf work-alike claim. (closed 2026-10-10: the README is rewritten around `html.Parse` / `css.Apply` / `layout.DisplayList` with no wkhtmltopdf work-alike claim.)
- [x] `documentation/README.md` and `documentation/architecture/README.md`: drop links to the deleted PDF chapters. Use the name blinkless. (closed 2026-10-10: both drop the deleted chapters and say there is no PDF writer, page encoder, or CLI (`documentation/README.md:5`).)
- [x] retired: `frontend/README.md`: product site for the renderer. Rebuild `docs/` after `frontend/src` copy changes. Do not edit `docs/` by hand. (retired 2026-10-10: `frontend/` and the built `docs/` are not in this tree.)
- [x] `bindings/python/README.md`: image binding only, package name handled with the rename below. (closed 2026-10-10: updated to the drawing-list removal contract (`bindings/python/README.md:7-10`).)
- [x] `output/README.md`: delete with the sample PDFs, or replace with one paragraph if a PNG sample directory remains. (closed 2026-10-10: replaced with the leftovers description (`output/README.md:1-33`); it states `make samples` writes nothing and the PDF-named directories are empty.)
- [x] `testdata/golden/README.md`: HTML fixtures for the renderer. Remove "HTML in, PDF out". (closed 2026-10-10: opens with "there is no PDF writer and no stored golden bytes" (`testdata/golden/README.md:1-5`).)
- [x] `testdata/golden/benchmarks/README.md` and `testdata/wasm/README.md`: cut PDF timings and the PDF request mode. (closed 2026-10-10, wave H recheck: the README opens with a retirement banner (`testdata/golden/benchmarks/README.md:3-6`: the templates are fixtures for the harness removed with the PDF writer and no Go benchmark reads them) and retires the external path (`:26-28`); the former `documentation/benchmarks.md` / `performance.md` keepers are recorded as removed (`:167-168`). The dated PDF timing tables remain as labeled history, not runnable instructions. `testdata/wasm/README.md:15` stays clean.)
- [x] `testdata/fonts/README.md`, `testdata/golden/assets/README.md`, `test/chrome/README.md`, `skills/improve-codebase/README.md`: name pass only, where the old product name appears. `compliance/README.md` and `compliance/verapdf/README.md` die with phase 9. (closed 2026-10-10: no old product name appears in the four files; `test/chrome/README.md:126-128` documents the stale PDF trees and that the `make chrome-cases-pdf` target is gone. Incidental: `testdata/golden/assets/README.md:5,13` still says text is "searchable in PDF output" and that the converter embeds faces; that is not this name-pass row's scope.)
- [x] `LICENSE` has no product name and no PDF claim. Leave it. (closed 2026-10-10: MIT text with no product name and no PDF claim (`LICENSE:1-3`).)

### 10.4 Release file

- [x] Write a new `RELEASE.md` for blinkless after phases 1-9. It lists the gates that still match the tree: `make test`, `make lint`, `make claim-scan`, `make build` of the image binary. It does not list `make golden`, veraPDF, or a `VERSION` stamp until a version file exists again. (closed 2026-10-10: `RELEASE.md` exists for blinkless 0.0.1 with the current gates and what ships/does not; it lists `make golden`, which was repurposed to the drawing-list tests (`Makefile:149-150`) and is now an agreed release gate in `AGENTS.md:146`, superseding the row's "does not list make golden" clause. `VERSION` is not used.)
- [x] Point `.github/workflows/release.yml` and `scripts/check_versions.sh` at that decision. Today both require `VERSION`. The file is gone on purpose. Do not invent `0.0.1` inside `VERSION` from this plan alone. (closed 2026-10-10: `release.yml` resolves the version from the tag and gates it against `bindings/python/pyproject.toml` and `bindings/c/include/blinkless.h` (`release.yml:52-72`); the script no longer reads `VERSION` (`scripts/check_versions.sh:20-28`).)

### 10.5 Name conversion

Do this after phases 1-9 so the replace does not touch files that were deleted. One commit for the module path. A second commit for user-facing strings if that split is easier to review.

Defining lines to change:

- [x] `go.mod` line 1, then every import of `github.com/chinmay-sawant/blinkless`. About 800 import lines. Also `Makefile` ldflags, `.github/workflows/ci.yml` (including the short `-X blinkless/internal/cli.Version` form), and `.github/workflows/release.yml`. (closed 2026-10-10: `go.mod:1` is `module github.com/chinmay-sawant/blinkless` with no `replace`; `rg -n 'gowkhtmltopdf' --glob '*.go'` -> no hits, exit 1; the CLI ldflags died with `internal/cli` and `release.yml` no longer stamps it.)
- [x] Root package name `package blinkless` in `doc.go`, `api.go`, `document.go`, `document_validate.go`, and the root tests. Package comment in `doc.go` line 1. (closed 2026-10-10: `doc.go` declares `package blinkless` and the module anchor; `api.go`, `document.go`, and `document_validate.go` are gone.)
- [ ] User-facing strings: `README.md`, `internal/cli/help.go` (`command := "blinkless"`, image command, `Name:` line), `cmd/blinkless` error prefix, `frontend/package.json` name, `frontend/index.html` title, `frontend/src/components/PageTitle.jsx`, `Footer.jsx`, `GitHubStars.jsx` repo slug, `LandingPage.jsx`. (remainder 2026-10-10: the named surfaces are updated or gone (README.md rewritten; `internal/cli`, `cmd/blinkless`, and `frontend/` absent). Still exported: the Python base exception `GowkhtmltopdfError` (`bindings/python/src/blinkless/exceptions.py:14`, exported at `__init__.py:28,86`), kept with the retained error taxonomy.)
- [x] Binary and artifact names: `bin/blinkless`, `frontend/public/wasm/blinkless.wasm`, `dist/libblinkless.so`, `bindings/c/include/blinkless.h` (`BLINKLESS_H`), `bindings/python/pyproject.toml` name `blinkless`, `scripts/build_cshared_for_wheel.sh`. Rename the image command directory if the binary is renamed. C ABI symbol names are a break. Bump the ABI version in the same commit. CI currently treats ABI 1 as frozen (`.github/workflows/ci.yml`). (closed 2026-10-10: `dist/wasm/blinkless.wasm`, `dist/libblinkless.so`, `bindings/c/include/blinkless.h` (`BLINKLESS_H`), `bindings/python/pyproject.toml:6` (`name = "blinkless"`), and `scripts/build_cshared_for_wheel.sh:27-29` all carry the blinkless name; no CLI binary ships, so the image-command rename is moot; the C ABI stays frozen at 1 as the header and CI pin it.)
- [x] `skills/improve-codebase/references/blinkless.md` renamed with the docs. (closed 2026-10-10: present.)

Do not replace these with blinkless:

- [x] `https://wkhtmltopdf.org/`, `github.com/wkhtmltopdf/wkhtmltopdf`, and the outline XML namespace `http://wkhtmltopdf.org/outline`. Delete the feature that emits that XML (phase 5) instead of renaming the namespace. (closed 2026-10-10: no upstream `wkhtmltopdf.org` / `github.com/wkhtmltopdf` URL remains (`rg -n 'wkhtmltopdf\.org|github\.com/wkhtmltopdf'` -> no hits, exit 1), and the outline XML emitter is gone with `internal/outline`; there is nothing left to rename or emit.)
- [x] retired: `internal/cli/help.go` license attribution for the original project. Keep the copyright line. Change the product sentence around it. (retired 2026-10-10: `internal/cli` is not in this tree.)
- [x] `LibraryVersion` value `0.12.7-dev`. Delete the constant in phase 7. It is not a name. (closed 2026-10-10: no Go symbol remains (`rg -n 'LibraryVersion|func Version' --glob '*.go'` -> no hits, exit 1). The string survives as the documented settings-surface id (`bindings/c/include/blinkless.h:109`, `bindings/python/src/blinkless/__init__.py:63`) and as frozen fixture content (`testdata/golden/fixture-56-architecture-diagram.html:230`).)
- [x] The GitHub Pages URL and module path move together. `go get` of the new path fails until the repository path matches. Land the module edit when the GitHub repository is `blinkless`, or record here that local builds use a `replace` until then. (closed 2026-10-10: the module path is `github.com/chinmay-sawant/blinkless` (`go.mod:1`) with no `replace`, and `AGENTS.md:19` records the matching repository URL.)

### 10.6 Frontend copy

- [x] retired: Rewrite `frontend/src/pages/LandingPage.jsx`, `LiveDemoPage.jsx`, `ShowcasePage.jsx`, `BenchmarksPage.jsx`, `DocumentationPage.jsx` so they describe the renderer. (retired 2026-10-10: no `frontend/` in this tree.)
- [x] retired: Remove the PDF viewer and the upstream dossier: `components/PdfViewer.jsx`, `pages/DossierPage.jsx`, `components/IssueCard.jsx`, and the PDF-only content JSON under `frontend/src/data/content/`. (retired 2026-10-10: no `frontend/` in this tree.)
- [x] retired: Rebuild the site and confirm `docs/` updates from that build. CI fails if `docs/` is dirty relative to the frontend build. (retired 2026-10-10: no `frontend/` source and no built `docs/` in this tree.)

## Dependencies

```text
phase 1 tests and sample PDFs
    -> phase 2 PDF programs
        -> phase 3 move fonts (aliases left behind)
            -> phase 4 retype renderer
                -> phase 5 delete convert PDF job
                    -> phase 6 delete Paint (nothing calls it)
                        -> phase 7 public API and binding PDF symbols
                            -> phase 8 delete internal/pdf and pdfprofile
                                -> phase 9 Makefile, compliance, scripts
                                    -> phase 10 docs, READMEs, RELEASE.md, name
```

Phase 6 before phase 5 does not compile. `convert.go` calls `PaintContext`. Phase 8 before phase 4 does not compile. `imageout` and `css` import `internal/pdf` for fonts.

## Out of scope

- No edit to a knowledge-base directory. This session was told not to use one.
- No git commit in this session.
- No new direct module. The allowlist stays `go-text/typesetting` and `tdewolff/canvas`.
- Pagination of a drawing list onto pages is not reimplemented here. It lived inside `Paint` for the PDF file. A later plan can add a paged drawing list if the renderer needs pages without a PDF writer.
