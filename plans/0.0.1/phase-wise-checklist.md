# 0.0.1 - Remove the PDF writer

> **Parent:** none. `plans/` was empty on 2026-10-08. This file is the only 0.0.1 ledger.
> **Status:** in progress. The PDF writer is out of the build. `go build ./...` exited 0 after the cut. README, RELEASE.md, and this ledger are updated. A full doc rewrite and the module rename are still open.
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

- [ ] Delete PDF byte tests under `internal/pdf/`: `pdf_test.go`, `pdf20_test.go`, `policy_test.go`, `compliance_test.go`, `structure_test.go`, `struct_test.go`, `semantic_oracle_test.go`, `semantic_converted_test.go`, `page_ops_test.go`, `content_blend_test.go`, `content_lifetime_test.go`, `flate_parallel_test.go`, `flate_serial_test.go`, `flate_release_test.go`, `image_test.go`, `image_pixels_test.go`, `text_autospace_test.go`, `subset_align_test.go`, `review_regression_test.go`. Expected: nothing in these files survives, they assert headers, xref, profiles, XObjects, or structure trees.
- [ ] Split `internal/pdf/font_test.go`, `fonttype0_test.go`, `faces_test.go`, `shape_test.go`, `shape_language_test.go`, `bench_test.go`. Delete the cases that embed a font or call `Content` / `Write`. Keep face parse, advance, registry, and `BenchmarkShapeRun` until phase 3 moves them.
- [ ] Keep, and move in phase 3, the face tests: `woff_test.go`, `font_file_cache_test.go`, `font_lazy_test.go`, `font_resource_regression_test.go`, `font_instance_test.go`, `cpal_test.go`, `registry_test.go`, `registry_lookup_test.go`, `registry_bench_test.go`.
- [ ] Delete `internal/pdfprofile/profile_test.go` with the package in phase 8. It parses `PDF/A` and `PDF/UA` strings only.

### 1.2 Convert and golden tests

- [ ] Delete `internal/convert/golden_test.go` and `compliance_golden_test.go`. `fixturePageBounds` is at `golden_test.go` around line 243. `TestGoldenCorpusAllFixtures` checks `%PDF-`, xref, `/FontFile2`, and `pdf.ParseSemantic`. That contract measures a PDF file, not the drawing list.
- [ ] Delete the other convert tests that only check PDF bytes: `convert_test.go`, `headers_toc_outline_links_test.go`, `hf_links_test.go`, `quality_test.go`, `web_fixtures_test.go`, `fontface_test.go`, `wk_compare_test.go`, `fuzz_test.go`, `cancellation_test.go`, `perf_test.go`, `perf3_assemble_test.go`, `perf3_page_blocks_test.go`, `repeated_resources_bench_test.go`, `header_footer_bench_test.go`.
- [ ] Delete `internal/convert/fixturetests/` PDF oracles: `output_ops_test.go`, `output_fixture_shape_test.go`, `output_fixture_01_test.go` through `output_fixture_64_next_72_props_test.go`, `pixel_regression_test.go`, `pixel_regression_cases_test.go`, `pixel_regression_corpus_test.go`, `fixture_helpers_test.go`. They parse `output/*.pdf` or rasterize it with Ghostscript.
- [ ] Trim mixed convert tests. Drop the `runPDF` / `%PDF-` cases from `page_first_test.go`, `page_side_test.go`, `page_named_test.go`, `perf3_nav_test.go`, `perf3_smartshrink_test.go`, `simplify_test.go`, `import_stylesheet_test.go`, `seams_test.go`, `benchmarks_test.go`. Keep prepare checks and `layoutBody` checks until phase 5 deletes that function.
- [ ] Keep `internal/convert/prepare/*_test.go`, `render/pipeline_test.go`, `orchestration_contract_test.go`. `render/plan_test.go` goes in phase 5 with `render/plan.go` (copy and collate indexes).

### 1.3 Root, CLI, chrome, binding tests

- [ ] Delete PDF cases in `document_render_test.go`, `document_bench_test.go`, `document_bench_validate_test.go`, `document_perf3_pin_test.go`, `document_bench_html_test.go`. Keep the PNG cases (`BenchmarkLibraryImage`, image validation).
- [ ] Delete `document_test.go` cases that map `Document`, `PDFVersion`, `PDFProfile`, margins, and copies. Keep `TestContentValidate`, `TestHTMLCopiesBytes`, and the `ImageDocument` cases.
- [ ] Delete `cmd/blinkless/main_test.go` and `internal/app/pdf_test.go` with those programs in phase 2.
- [ ] Delete `test/chrome/pdf_output_test.go` and `profile_bench_test.go`. Keep `test/chrome/manifest_test.go` and the HTML cases under `test/chrome/cases/`.
- [ ] Delete PDF assertions in `bindings/c/cshared_test.go` (`requirePDFShape`), `bindings/wasm/contract_test.go` PDF mode, `bindings/wasm/fixture_test.go` (`assertFixturePDF`), `bindings/python/tests/test_binding.py` PDF cases, `bindings/python/tests/bench_library.py` `bench_pdf`, and `bindings/python/examples/invoice.py`. Keep the PNG cases.

### 1.4 Sample files

- [ ] Delete committed PDF samples under `output/` (`output/*.pdf`, `output/pdf-1.7/`, `output/pdf-2.0/`, `output/pdf-1.7-compliance/`, `output/pdf-2.0-compliance/`, `output/python/`, `output/wkhtmltopdf/`, `output/validated/`). `output/README.md` calls these viewer-smoke files. `fixturetests` also treats `output/validated/` as a Ghostscript oracle. The HTML inputs under `testdata/golden/` stay.
- [ ] Delete `testdata/golden/benchmarks/output/*.pdf`. Keep the `.html.tmpl` templates and any PNG bench images.
- [ ] Delete `compliance/fixtures/*.pdf`. Delete generated trees `test/chrome/pdf/` and `test/chrome/cases/pdf/` if present. They are gitignored outputs.
- [ ] Leave `testdata/golden/*.html`, `testdata/golden/assets/`, `testdata/web/*.html`, and `test/chrome/cases/*.html`. Those are renderer inputs.

## Phase 2: PDF programs

Proof: `go build ./...` succeeds. `cmd/blinkless` still builds. Nothing in `cmd/` imports `internal/pdf`.

### 2.1 Commands and examples

- [ ] Delete `cmd/blinkless/` (`main.go` calls `cli.ModePDF` and `app.RunPDF`). Delete `examples/pdf/`.
- [ ] Delete `internal/app/pdf.go` symbols `RunPDF`, `BuildPDFRequest`, `DefaultTOCXSL`. Keep the shared sentinels in that file if `internal/app/image.go` still uses them (`ErrNilCommand`, `ErrNilContext`). Move those sentinels next to `image.go` if the file would otherwise be empty.
- [ ] Keep `cmd/blinkless/` and `internal/app/image.go`. They import `internal/cli` and `internal/imageout`, not `internal/pdf`. The rename of the binary is phase 10.

### 2.2 CLI flags that only exist for the PDF file

- [ ] Delete `ModePDF` flags in `internal/cli/flags.go`: `--pdf-version`, `--pdf-profile`, `--copies`, `--collate`, `--grayscale`, `--title`, `--no-pdf-compression`, `--page-offset`, `--outline`, `--outline-depth`, `--exclude-from-outline`, `--dump-outline`, `--dump-default-toc-xsl`, `--cover`, `--toc`, `--external-links`, `--internal-links`, header and footer flags, TOC flags. Drop the matching parser tests in `internal/cli/cli_test.go`.
- [ ] Keep `ModeImage` flags and the shared flags the image binary already accepts (`--html`, `--url`, `--allow`, `--font-path`, `--zoom`, cookies, media type). Keep `TestImageGrammarAndOptions`.
- [ ] Page size, orientation, and margins are layout inputs today and PDF CLI flags. After phase 5 nothing in the image path reads them from `settings.PdfGlobal`. Delete the flags with `ModePDF`. Keep `settings` page-size parsers only if a layout caller remains. Check `internal/imageout` before deleting `ParsePageSize`.

## Phase 3: Move the font code out of the PDF package

Proof: `go test ./internal/fonts ./internal/pdf -count=1` (single packages, not `./...`). Face tests pass from `internal/fonts`. `internal/pdf` still builds because of temporary aliases.

The PNG path calls `pdf.DefaultFont`, `pdf.Registry`, `pdf.RegistryFromGlobal`, `pdf.ShapeRunWithFeaturesLanguage`, and glyph contour helpers (`internal/imageout/imageout.go`, `frame.go`, `ttfraster.go`). Layout measures with `pdf.Font` (`internal/layout/inline_paint.go`, `layout_flow.go`). Public `css.Document.registry` is `*pdf.Registry` (`css/css.go`).

### 3.1 Move

- [ ] Move these files from `internal/pdf/` into `internal/fonts/`: face and metrics (`fonts.go` minus `PDFAscent`, `PDFDescent`, `PDFCapHeight`, `PDFBBox`), `faces.go`, `assets/` plus `assets.go`, `registry.go`, `font_file_cache.go`, `woff.go`, `shape.go`, `shape_gotext.go`, `glyph.go`, `font_instance.go`, `cpal.go`, `numbers.go` (sfnt constants). Move the face tests listed in phase 1.1 with them.
- [ ] Leave a temporary alias in `internal/pdf` (`type Font = fonts.Font`, same for `Registry`, `FaceSet`) so current imports still compile. Delete the aliases in phase 8.
- [ ] `RegistryFromGlobal` takes `settings.PdfGlobal` (`internal/pdf/registry.go`). Change it to take the font paths and the system-font flag, or a small struct in `internal/settings` that is not named for PDF. `imageout` is the caller that must keep working.
- [ ] Update `internal/fonts/fonts.go` package comment. It currently says decoded bytes are handed to `internal/pdf`.

## Phase 4: Point the renderer at internal/fonts

Proof: `go build ./...`. `rg 'internal/pdf' internal/imageout internal/layout css internal/pubstate internal/convert/prepare` shows font aliases only, no `pdf.Document` outside paint and convert.

- [ ] Retype font fields in the measurement files: `internal/layout/layout.go` (`Op.Font`), `layout_context_styles.go`, `layout_section.go`, `inline_paint.go`, `inline_text_box.go`, `font_palette.go`, `font_synthesis_geom.go`, `font_variation.go`, `style_font_size_adjust_props.go`.
- [ ] Retype `internal/imageout/imageout.go`, `frame.go`, `ttfraster.go`.
- [ ] Retype `css/css.go` and `internal/pubstate/state.go`.
- [ ] Retype `internal/convert/prepare/prepare.go` and `prepare/styles.go` (`pdf.ParseFontBytes` at `styles.go` around line 531). `Prepared.Registry` becomes the fonts package registry.
- [ ] Leave `Paint` importing `*pdf.Document` until phase 6. This phase does not delete paint.

## Phase 5: Delete the PDF job in convert

Proof: `rg 'pdfPipeline|PaintContext|NewPDFRequest|convert.Run' --glob '*.go'` finds no production caller. `go test ./internal/imageout ./internal/convert/prepare ./internal/convert/render -count=1` passes. Image mode never called `convert.Run`. Its entry is `imageout.RunRequest` (`internal/imageout/imageout.go`).

Body paint ends in `renderObject` (`internal/convert/convert.go`), which calls `layout.PaintContext`. PDF assembly starts at `pdfPipeline.Assemble` (`internal/convert/pdf_pipeline.go`). `Finalize` writes the file at the `run.doc.Write` call.

### 5.1 Delete these files

- [ ] `internal/convert/pdf_pipeline.go` (`Assemble`, `Finalize`, TOC, outline, links, copies, header/footer, `Write`).
- [ ] `internal/convert/toc.go`, `outline.go`, `links.go`, `hf.go`, `hf_geometry.go`, `page_plan.go`, `page_margin_boxes.go`, `page_blocks.go`.
- [ ] `internal/convert/render/plan.go` and `plan_test.go`. Image mode does not call them. Keep `internal/convert/render/pipeline.go`. `imagePipeline.Assemble` is a no-op and `RunRequest` calls `render.Run`.
- [ ] The PDF half of `internal/convert/convert.go`: `NewPDFRequest`, `Run`, `policyForProfile`, `PolicyForGlobal`, `renderObject`, `layoutBody`. Keep nothing in this file unless a symbol is still referenced by `prepare` or tests. If the file becomes empty, delete it.
- [ ] `internal/outline/` including `outline_test.go`. The package builds a heading tree and dump XML. The only callers are convert PDF assembly. It does not import `internal/pdf`, and `imageout` does not import it.
- [ ] Keep `internal/line/`. `imageout` calls `line.Emit`.

### 5.2 What convert must still export

- [ ] Keep `internal/convert/prepare/` (`Document`, `BuildOptions`, sheet collection, `@font-face`). `prepareImageDocument` calls it.
- [ ] Keep `internal/convert/render/pipeline.go` and its test. Delete the PDF adapter, not the interface.

## Phase 6: Delete PDF drawing inside layout

Proof: `rg 'func Paint|PaintContext|pdf.Document|StructElem' internal/layout` is empty. `go test ./layout ./screen ./internal/imageout -count=1` passes. Public `layout.DisplayList` still returns ops.

`Paint` (`internal/layout/paint.go`) paginates the list and paints it into `doc`. The page-break, repeated table header, and orphan passes run only from `PaintContext`. `screen.Render` calls `layout.Lay`. `Lay` calls `imageout`, which rasterizes `Ops` and does not call `Paint`.

### 6.1 Delete the writer entry

- [ ] Delete `Paint`, `PaintContext`, `PaintBand`, `PaintBandContext`, and the operator emitters in `paint.go` (`drawFill`, `drawStroke`, `drawLine`, `drawText`, `drawImage`, `drawLinkXform`, `canvasToPDF`). Keep `StyleOf`, `FakeBoldFor`, and `PaintLineGeometry`. `imageout` calls those.
- [ ] Delete `internal/layout/tagging.go` (`buildStructureTree` writes `*pdf.StructElem`).
- [ ] Delete `internal/layout/paint_groups.go` (PDF form XObjects). Keep `BlendGroup` in `blend_group.go`.
- [ ] Delete `drawGridRun` in `grid_run.go` and `pdfCTMFromCSS` in `transform.go`. Keep the rest of both files. They build ops.
- [ ] Remove `StructElem *pdf.StructElem` from `opExtra` (`internal/layout/op_extra.go`). Keep URI, image bytes, transform, and blend on the op. `OpLinkURI` stays as a hit target. It is not a PDF annotation.

### 6.2 Delete PDF page splitting

- [ ] Delete the files that only rewrite ops onto a PDF page height: `paint_pagination_fixpoint.go`, `paint_pagination_split.go`, `paint_pagination_chrome.go`, `paint_pagination_seal.go`, `paint_flow_breaks.go`, `paint_flow_orphans.go`, `paint_flow_tables.go`, `paint_flow_index.go`, `page_named.go`. Their only production caller is `PaintContext`.
- [ ] In `sticky.go`, keep `tagSticky` (it stamps `Op.StickyID` during layout). Delete `applyStickyPrint`.
- [ ] Keep multicol's use of `Options.Height` as a column height (`internal/layout/multicol.go`). `imageout` passes viewport height there, so the image changes if that snap goes away.
- [ ] Stop filling `Result.Pages` and `Result.Locations` from paint. `PlacedElements` (`internal/layout/placed.go`) is the box list that does not need `Paint`. Keep it.
- [ ] Delete `HasFragmentLinks` on `Result`. It exists so the PDF link pass can find `#` URIs. The URI on the op stays.

### 6.3 Layout tests

- [ ] Delete tests that read a PDF content stream or a structure tree: `tagging_test.go`, the `%PDF-` cases in `layout_test.go` (`TestPaginateAndPaint`, `TestPaintSinglePage`), `rounded_border_test.go`, the stream searches in `color_adjust_paint_test.go` and `blend_test.go`, and the structure-tree cases in `architecture_followup_test.go`.
- [ ] Rewrite the geometry tests that currently call `pdf.NewDocument` and `Paint` to assert `Op` fields or `PlacedElements`. Files the survey marked as mixed include `transform_test.go`, `flex_test.go`, `grid_test.go`, `multicol_test.go`, `orphans_widows_test.go`, `pagination_ctx_test.go`, `pagination_thead_test.go`, `sticky_test.go`, and the `fixture56_*` / `flex_chrome_*` paint tests. A test that only checked which PDF page an op landed on is deleted with phase 6.2. A test that checks box x/y stays.
- [ ] `layout/*_test.go` (the public package) does not import `internal/pdf`. Those tests stay.

## Phase 7: Public PDF API and binding PDF symbols

Proof: `rg 'WritePDF|PDFProfile|html_to_pdf|convert_html_to_pdf' --glob '*.go' --glob '*.py' --glob '*.h'` is empty outside docs. `go build ./...` succeeds. Image symbols still build.

### 7.1 Root library

- [ ] Delete `Document`, `NewDocument`, `WritePDF`, `WritePDFOutline`, `PDF`, and `Validate` on that type (`document.go`, `document_validate.go`). Delete `Page`, `TOC`, `HeaderFooter`, `Margin` if nothing image-side uses them.
- [ ] Delete PDF errors and `executePDFTo` / `toPDFRequest` / `pdfGlobal` (`api.go`, `document.go`). Delete `LibraryVersion` (`0.12.7-dev`) and `Version()`. That string is the wkhtmltopdf settings id, not this product's version (`api.go`).
- [ ] Keep `ImageDocument`, `WriteImage`, `Image`, `Content`, `HTML`, `File`, `URL`, `Crop`, `NetworkPolicy`.
- [ ] `ImageDocument.toImageRequest` still builds `settings.PdfGlobal`. Point it at the image settings type so the image API does not mention PDF.

### 7.2 Bindings

Do not delete `bindings/c/`, `bindings/python/`, or `bindings/wasm/`. Each binding returns either PDF bytes or PNG/JPEG bytes. Strip the PDF half.

- [ ] C: delete `bindings/c/options_pdf.go`, `blinkless_html_to_pdf`, and `GwkPdfOptions` in `bindings/c/include/blinkless.h`. Keep `blinkless_html_to_image`, `options_image.go`, and the free/last-error helpers. The symbol rename is phase 10.
- [ ] Python: delete `Document`, `PDFOptions`, `convert_html_to_pdf`, `convert_file_to_pdf`, `convert_url_to_pdf`. Keep `ImageDocument`, `convert_html_to_image`.
- [~] WASM request modes and output: moved to [the compatibility checklist, Phase 5](html-css-json-compatibility-checklist.md#phase-5-usable-drawing-list-json-from-wasm). The current adapter returns JSON, and frontend/ is absent. That phase owns drawing-list mode, serialization, fixtures, and consumer proof; this historical PNG/JPEG instruction is superseded.

## Phase 8: Delete the writer package, profiles, and PDF settings

Proof: `rg 'internal/pdf"|internal/pdfprofile' --glob '*.go'` is empty. `go build ./...` succeeds. `make test-quick` is the gate for this phase (capped concurrency, see the Makefile). Record the exit code here when it passes.

- [ ] Delete the writer files in `internal/pdf/`: `pdf.go`, `content.go`, `flate_parallel.go`, `text_autospace.go`, `subset.go`, `fonttype0.go`, `fontpdf.go`, `images.go`, `structure.go`, `policy.go`, `metadata.go`, `icc.go`, `outputintent.go`, `semantic.go`, `page_ops.go`, `doc.go`, and the temporary font aliases from phase 3. Delete the directory if it is empty.
- [ ] Delete `internal/pdfprofile/`.
- [ ] Delete PDF settings keys and their setters: `pdfversion`, `pdfprofile`, `usecompression`, `copies`, `collate`, `title`, `grayscale` / `colormode`, `outline`, `outlinedepth`, `excludefromoutline`, `dumpoutline`, `dumpoutlinewithdefaulttocxsl`, `externallinks`, `locallinks`, `resolverelativelinks`, `includeinoutline`, `useoutline`, `pageoffset`, `header.*`, `footer.*`, `toc.*`, `istableofcontent`, `iscover`. Homes: `internal/settings/settings.go` and `reflect.go`. Delete `object_roles.go` cover/TOC stamps.
- [ ] Keep load and image settings: allow lists, proxy, cookies, zoom, timeout, font paths, background, media type, image width/height/crop/format/quality. Keep `internal/load`, `internal/html`, `internal/css`, `internal/svg`, `markup`, `screen`.

## Phase 9: Make targets, scripts, compliance

Proof: `make help` or a read of the Makefile no longer lists `golden`, `samples` PDF lines, or `c-shared` under the old PDF library name. `make test-quick` still passes.

- [ ] Delete Makefile targets `golden`, `golden-update`, `chrome-cases-pdf`, `run`, `screenshots`, `weasyprint`, `bench`, `bench-engine`, `bench-inprocess`, `bench-cli-compare`, `python-api`, `samples-python`. Drop `BenchmarkLibraryPDF` from `bench-lib`. Drop `./internal/pdf` from `RACE_PKGS`.
- [ ] Split `samples`: delete the PDF recipe, keep the PNG recipe that calls the image binary.
- [ ] Split `build`: stop building `bin/blinkless`. Keep the image binary until phase 10 renames it.
- [ ] Keep `wasm`, `c-shared`, and `python-binding-test` only for the image symbols. Rename artifacts in phase 10. Delete them if phase 7 removed the last exported image symbol by mistake. It should not.
- [ ] Delete `compliance/` (veraPDF, `verify_pdfs.sh`, `structure_tree_check.py`, fixtures). Delete repo-root `verapdf/` if it is the installed validator tree.
- [ ] Delete `scripts/inspect_pdf_fonts.py`, `scripts/inspect_pdf_ops.py`, `scripts/puppeteer_print.js` if it only prints PDFs, and the PDF half of `scripts/bench-external.sh`. Keep `scripts/check-file-size.sh` and `scripts/pr-diff-stat.sh`.
- [ ] `scripts/check_versions.sh` reads a `VERSION` file that is gone. `release.yml` requires `VERSION` to match the tag. Do not recreate the old PDF version file in this phase. Phase 10 decides the blinkless version source. Until then, the workflow must not fail the build on a missing `VERSION`.

## Phase 10: Docs, READMEs, release file, and the name blinkless

Proof: `rg -n 'blinkless|blinkless' -g '!frontend/public/data/**' -g '!reports/**'` returns only the lines this phase explicitly keeps (upstream URLs, if any remain). `make claim-scan` exits 0 after the docs match the code. Do not hand-edit `docs/`. It is the built site. Rebuild it from `frontend/` after the source copy changes.

`VERSION` and `RELEASE.md` are already absent. Do not restore the old release checklist.

### 10.1 Docs to delete

- [ ] Delete `documentation/architecture/09-pdf-writer.md`.
- [ ] Delete `documentation/cli.md`, `documentation/python.md`, `documentation/wasm.md`, `documentation/MIGRATION-0.2.4.md`, `documentation/benchmarks.md`, `documentation/performance.md`, `documentation/samples.md`, `documentation/ghostscript-licensing.md`.
- [ ] Delete `documentation/comparison-with-others/` (`sebastiaanklippert-go-wkhtmltopdf.md`, `landscape-2026.md`, `landscape-2026.html`).
- [ ] Delete `reports/wkhtmltopdf-open-issues/` and `reports/wkhtmltopdf-open-issues.html`. Delete `frontend/public/data/issues.json` with the dossier page. Those are the upstream issue mirror, not this product. A rename would corrupt them.
- [ ] Delete `skills/chrome-debug/`, `skills/chrome-debug-v2/`, `skills/chrome-flex-pdf-closure/`, `skills/diagnose-golden-fixture/` if they only exist to debug PDF output. Read each skill header before deleting. Keep a skill that diagnoses the drawing list.

### 10.2 Docs to rewrite

Keep the description of load, parse, CSS, layout, and the drawing list. Cut the PDF sink.

- [ ] `documentation/overview.md`, `documentation/architecture.md`, `documentation/architecture/README.md`.
- [ ] `documentation/architecture/02-library-api.md` through `08-convert-pipeline.md` and `10-imageout-svg.md`. `07-layout.md` keeps the drawing list. Cut "canvas to PDF" and `Paint` into `pdf.Document`. `08` keeps `prepare`. Cut assemble and `Write`.
- [ ] `documentation/architecture/01-entrypoints-cli.md` becomes the image command, or is deleted if the image command's README is enough.
- [ ] `documentation/getting-started.md`, `library-api.md`, `fidelity.md`, `compatibility-matrix.md`, `fonts.md`, `deferred.md`, `THREAT-MODEL.md`, `integration-security.md`, `documentation/README.md`.
- [ ] `CHANGELOG.md`: add an unreleased note that the PDF writer is gone and the drawing list remains. Do not rewrite old release sections into blinkless history.
- [ ] `CONTRIBUTING.md`: the pipeline stops at the drawing list and the PNG encode. Remove stable-PDF-bytes rules.
- [ ] `AGENTS.md`: product name blinkless, pipeline stops at the drawing list, drop `make golden` as a release gate, drop the PDF writer package from the map. Leave the no-git and capped-test rules.

### 10.3 README files

- [ ] `README.md` lines 2-14 name `blinkless`, the wkhtmltopdf CLI, and a pipeline that ends in write. Rewrite the title, the first paragraph, and the binary list for blinkless. Describe the drawing list and the PNG path. Remove the wkhtmltopdf work-alike claim.
- [ ] `documentation/README.md` and `documentation/architecture/README.md`: drop links to the deleted PDF chapters. Use the name blinkless.
- [ ] `frontend/README.md`: product site for the renderer. Rebuild `docs/` after `frontend/src` copy changes. Do not edit `docs/` by hand.
- [ ] `bindings/python/README.md`: image binding only, package name handled with the rename below.
- [ ] `output/README.md`: delete with the sample PDFs, or replace with one paragraph if a PNG sample directory remains.
- [ ] `testdata/golden/README.md`: HTML fixtures for the renderer. Remove "HTML in, PDF out".
- [ ] `testdata/golden/benchmarks/README.md` and `testdata/wasm/README.md`: cut PDF timings and the PDF request mode.
- [ ] `testdata/fonts/README.md`, `testdata/golden/assets/README.md`, `test/chrome/README.md`, `skills/improve-codebase/README.md`: name pass only, where the old product name appears. `compliance/README.md` and `compliance/verapdf/README.md` die with phase 9.
- [ ] `LICENSE` has no product name and no PDF claim. Leave it.

### 10.4 Release file

- [ ] Write a new `RELEASE.md` for blinkless after phases 1-9. It lists the gates that still match the tree: `make test`, `make lint`, `make claim-scan`, `make build` of the image binary. It does not list `make golden`, veraPDF, or a `VERSION` stamp until a version file exists again.
- [ ] Point `.github/workflows/release.yml` and `scripts/check_versions.sh` at that decision. Today both require `VERSION`. The file is gone on purpose. Do not invent `0.0.1` inside `VERSION` from this plan alone.

### 10.5 Name conversion

Do this after phases 1-9 so the replace does not touch files that were deleted. One commit for the module path. A second commit for user-facing strings if that split is easier to review.

Defining lines to change:

- [ ] `go.mod` line 1, then every import of `github.com/chinmay-sawant/blinkless`. About 800 import lines. Also `Makefile` ldflags, `.github/workflows/ci.yml` (including the short `-X blinkless/internal/cli.Version` form), and `.github/workflows/release.yml`.
- [ ] Root package name `package blinkless` in `doc.go`, `api.go`, `document.go`, `document_validate.go`, and the root tests. Package comment in `doc.go` line 1.
- [ ] User-facing strings: `README.md`, `internal/cli/help.go` (`command := "blinkless"`, image command, `Name:` line), `cmd/blinkless` error prefix, `frontend/package.json` name, `frontend/index.html` title, `frontend/src/components/PageTitle.jsx`, `Footer.jsx`, `GitHubStars.jsx` repo slug, `LandingPage.jsx`.
- [ ] Binary and artifact names: `bin/blinkless`, `frontend/public/wasm/blinkless.wasm`, `dist/libblinkless.so`, `bindings/c/include/blinkless.h` (`BLINKLESS_H`), `bindings/python/pyproject.toml` name `blinkless`, `scripts/build_cshared_for_wheel.sh`. Rename the image command directory if the binary is renamed. C ABI symbol names are a break. Bump the ABI version in the same commit. CI currently treats ABI 1 as frozen (`.github/workflows/ci.yml`).
- [ ] `skills/improve-codebase/references/blinkless.md` renamed with the docs.

Do not replace these with blinkless:

- [ ] `https://wkhtmltopdf.org/`, `github.com/wkhtmltopdf/wkhtmltopdf`, and the outline XML namespace `http://wkhtmltopdf.org/outline`. Delete the feature that emits that XML (phase 5) instead of renaming the namespace.
- [ ] `internal/cli/help.go` license attribution for the original project. Keep the copyright line. Change the product sentence around it.
- [ ] `LibraryVersion` value `0.12.7-dev`. Delete the constant in phase 7. It is not a name.
- [ ] The GitHub Pages URL and module path move together. `go get` of the new path fails until the repository path matches. Land the module edit when the GitHub repository is `blinkless`, or record here that local builds use a `replace` until then.

### 10.6 Frontend copy

- [ ] Rewrite `frontend/src/pages/LandingPage.jsx`, `LiveDemoPage.jsx`, `ShowcasePage.jsx`, `BenchmarksPage.jsx`, `DocumentationPage.jsx` so they describe the renderer.
- [ ] Remove the PDF viewer and the upstream dossier: `components/PdfViewer.jsx`, `pages/DossierPage.jsx`, `components/IssueCard.jsx`, and the PDF-only content JSON under `frontend/src/data/content/`.
- [ ] Rebuild the site and confirm `docs/` updates from that build. CI fails if `docs/` is dirty relative to the frontend build.

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
