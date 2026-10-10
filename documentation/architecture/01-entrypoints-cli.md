# Entry points

There is no CLI in this tree. `cmd/` is empty and no Makefile target writes a CLI binary. The ways into the engine are the public Go packages and the bindings.

## Public Go packages

| Package | Entry point | Result |
|---------|-------------|--------|
| `html` | `html.Parse` (`html/parse.go:36`) | the document tree |
| `css` | `css.Parse` (`css/css.go:81`), `css.Apply` (`css/css.go:97`) | a `css.Document` with its stylesheets attached |
| `layout` | `layout.DisplayList` (`layout/displaylist.go:134`), `layout.DisplayListOptions` (`layout/displaylist.go:139`) | a `layout.Display` drawing list |
| `markup` | `markup.Parse` (`markup/markup.go:116`) | a detached copy of the tree for inspection |

`css.Apply` collects the document's `<style>` blocks plus any `Options.Extra` sheets. `layout.DisplayList` places that styled document and returns the drawing operations. The list is the output (`layout/doc.go:1-9`).

## Internal pipeline

`internal/convert/prepare` is the shared load, parse, stylesheet, and `@font-face` phase. `prepare.Document` (`internal/convert/prepare/prepare.go:210`) returns `Prepared{Resource, Root, Resources, Sheets, Registry}` (`prepare.go:198`). `css.Apply` drives it: it builds a `ResourceContext` (`css/css.go:176`) and calls `CollectTreeSheets` (`css/css.go:186`) and `MergeFontFaces` (`css/css.go:200`).

`internal/convert/render` owns the stage order `RenderObjects` -> `Assemble` -> `Finalize` (`internal/convert/render/pipeline.go:14-18`, `Run` at `pipeline.go:28`). No production pipeline implements it today; the test pipeline in `pipeline_test.go` exercises the ordering.

## Browser adapter

`bindings/wasm` is the browser entry. It exposes `blinklessWASM` (`bindings/wasm/main_js_wasm.go:15`). The request is JSON: `html`, `mode`, `width`, `height` (`bindings/wasm/contract.go:46-51`). `mode` accepts only `"display"`, and the old writer fields (pageSize, orientation, padding, quality) are rejected as unknown (`contract.go:43-45`). The response is the versioned drawing list, schema `blinkless.drawinglist/1` (`bindings/wasm/drawing_list.go:21`). Images resolve from `data:` URLs only (`bindings/wasm/resources.go:24-51`).

`make wasm` builds `dist/wasm/blinkless.wasm` and copies `wasm_exec.js`, `testdata/wasm/sample.html`, and `testdata/wasm/manifest.json` (Makefile `wasm` target). `make wasm-test` runs `scripts/check-wasm-contract.sh` first.

## C and Python bindings

`bindings/c` builds the c-shared library with `make c-shared` (needs `CGO_ENABLED=1`). It exposes the frozen ABI in `bindings/c/include/blinkless.h`: version and ABI queries, status codes, and a process-wide last-error slot (`bindings/c/main.go`). A drawing-list entry point is not wired yet; `exports_stub.go` keeps the package compiling under `CGO_ENABLED=0`.

`bindings/python` loads that library through ctypes. Version and ABI queries work. The conversion entry points (`convert_html_to_pdf`, `convert_file_to_pdf`, `convert_url_to_pdf`, `convert_html_to_image`, `Document.pdf`, `ImageDocument.image`) raise `RuntimeError` with the removal reason (`bindings/python/README.md`).

## Build targets

- `make build`: `CGO_ENABLED=0 go build ./...` (Makefile `build` target). It compiles every package and writes no binary.
- `make golden`: runs the public drawing-list tests in `./layout` (`TestDisplay`).
- `make samples`: aliases `make golden` and writes nothing.
