# Library API

Module path: `github.com/chinmay-sawant/blinkless`. The root package is the
module anchor: it declares the package clause and a one-line pipeline summary,
and nothing else (`doc.go:1-9`). The calls live in `html`, `css`, and
`layout`.

The pipeline is load, HTML parse, CSS cascade, then layout as a drawing list.
`layout.DisplayList` returns that list (`layout/doc.go:1-3`). The list is the
output. There is no PDF writer and no page bitmap encoder: the root package
says the engine does not encode a page bitmap (`doc.go:4`), and the CSS and
layout packages say they do not write a PDF (`css/doc.go:8`,
`layout/doc.go:2-3`).

## Parser: `html`

`html.Parse(source []byte) (*html.Document, error)` parses UTF-8 HTML into a
tree the CSS package can style (`html/parse.go:34-49`). A nil source parses as
an empty document (`html/parse.go:34-35`). The public wrapper calls the
internal parser `internal/html.ParseDocument`, which strips a leading UTF-8
BOM (`html/parse.go:39`, `internal/html/html.go:174-181`).

`html.Document` is opaque. It holds the engine's own tree and exposes no
fields (`html/parse.go:21-25`). `css.Apply` reads that tree through the
internal pubstate handoff (`css/css.go:122`, `internal/pubstate/state.go:44`),
and `layout.DisplayList` reads the styled form (`layout/displaylist.go:148`,
`internal/pubstate/state.go:59`). `Document.Find(id)` returns the element with
that id and its descendant text (`html/parse.go:51-70`).

Two parser details are deliberate:

- `markup.Parse` returns a detached copy for inspection; it is not the tree
  the CSS and layout packages accept (`markup/markup.go:114-123`,
  `html/doc.go:3-5`).
- `html.Document` has no doctype or document-mode accessor; `markup.Node.Mode`
  covers detached inspection (`html/doc.go:7-11`).

## CSS: `css`

`css.Parse(source string) (*css.Sheet, error)` parses one stylesheet
(`css/css.go:81-93`). `css.Apply(ctx, doc, opts) (*css.Document, error)`
collects the document's style elements and the extra sheets in `Options.Extra`
(`css/css.go:95-97`, `css/css.go:206-222`).

`Options` carries the viewport in CSS pixels (`WidthPx`, `HeightPx`), the
media type (`Media`: `"screen"` or `"print"`, empty means screen), extra
sheets, and the focused, hovered, and pressed element ids (`css/css.go:65-79`,
`css/css.go:147-156`). `Apply` rejects a nil context, a nil document, a
non-positive size, and an unknown media type (`css/css.go:101-120`); the
errors are `css.ErrNilContext`, `css.ErrNilDocument`, `css.ErrBadSize`, and
`css.ErrBadMedia` (`css/errors.go:5-16`).

`Apply` does not stop at inline `<style>` blocks. It routes linked and
`@import` sheets through the shared loader (`css/css.go:158-204`,
`internal/convert/prepare/styles.go:175-285`). Images are not fetched here;
layout resolves them through its own option (see below). The returned
`*css.Document` is what `layout.DisplayList` places (`css/css.go:95-97`).

`css.Relayout` restyles the same tree for a new viewport and pointer state
without reparsing the HTML or the stylesheet text. It shares the tree and
reuses the parsed sheets and font registry, so the tree must not change
between `Apply` and `Relayout`; a source or theme change needs a fresh
`Apply` (`css/relayout.go:10-24`).

## Rendering: `layout`

`layout.DisplayList(ctx, doc) (*layout.Display, error)` lays the styled
document out and returns the display list (`layout/displaylist.go:126-136`).
`layout.DisplayListOptions(ctx, doc, options)` is the same call with an image
resolver (`layout/displaylist.go:138-139`).

`layout.Options.Images` is `func(src string) ([]byte, error)`. It returns
encoded image bytes (PNG, JPEG, or SVG) for one source, such as an `<img src>`
value or a CSS `background-image` target. Nil means no source resolves, so
image paint is skipped (`layout/layout.go:23-33`).

The `Display` shape (`layout/displaylist.go:99-124`):

| Field | Meaning |
|-------|---------|
| `Ops []DisplayOp` | Operations in source order |
| `Order []int` | Indexes into `Ops` in paint order, ready to iterate |
| `Boxes []Box` | Element border boxes in document order, CSS pixels |
| `Width`, `Height int` | Canvas size in CSS pixels |
| `PointsPerPixel`, `PixelPerPoint float64` | Reciprocals converting op coordinates (points) to canvas pixels and back (72/96 and 96/72) |

`Box` is `ID`, `Tag`, `Action`, `Text`, `X`, `Y`, `W`, `H`, with geometry in
CSS pixels (`layout/layout.go:10-21`). Use boxes for hit testing and
operations for painting.

`DisplayOp` is the internal operation type under an exported alias
(`layout/displaylist.go:22-55`). Coordinates are canvas points, y down. For
text and bullet operations, Y is the baseline (`layout/displaylist.go:22-35`).
The kinds cover rectangles, lines, text, images, link targets, bullets, and
table grid runs (`layout/displaylist.go:65-97`). `DisplayOpNoop` (deactivated
by clipping) and `DisplayOpUnknown` (blend or isolation boundary marker)
paint nothing and must be skipped (`layout/displaylist.go:31-35`,
`layout/displaylist.go:67-76`). Rare payloads sit behind accessors such as
`LinkURI`, `ImageBytes`, `ImageAlt`, `Transform`, and `Opacity`; the plain
fields are safe to read (`layout/displaylist.go:37-49`).

An image operation keeps its encoded bytes. When orientation or a clip cannot
stay in those bytes, that one operation is re-encoded as a PNG. That is the
bitmap fallback, and it is not a picture of the page (`layout/doc.go:6-9`,
`layout/layout.go:26-32`).

`DisplayList` and `DisplayListOptions` reject a nil context and a document
that `css.Apply` did not produce: `layout.ErrNilContext` and
`layout.ErrNilDocument` (`layout/errors.go:5-11`,
`layout/displaylist.go:140-151`).

## Browser binding: `bindings/wasm`

`bindings/wasm` wraps the same three calls and serializes the `Display` to one
versioned JSON document. `Convert` parses the inline HTML, applies screen
media, lays the document out, and marshals the drawing list; it does not
encode a PNG or JPEG (`bindings/wasm/contract.go:123-188`).

The browser build exports `blinklessWASM(requestJSON, progressFn)`
(`bindings/wasm/main_js_wasm.go:13-17`). The native build of the same package
runs `Convert` on a fixture and prints the same JSON, which is how the
contract tests compare the two (`bindings/wasm/main_native.go:12-17`,
`bindings/wasm/main_native.go:31-53`).

The request is `{ "html": ..., "mode": ..., "width": ..., "height": ... }`
(`bindings/wasm/contract.go:41-51`):

- `html` is required, inline, and at most 4 MiB
  (`bindings/wasm/contract.go:16-26`, `bindings/wasm/contract.go:99-105`).
- `mode` defaults to `display`, the only accepted value
  (`bindings/wasm/contract.go:16-26`, `bindings/wasm/contract.go:107-114`).
- `width` and `height` are viewport CSS pixels from 0 to 4096. A zero width
  falls back to 1024; a zero height follows the width
  (`bindings/wasm/contract.go:116-121`, `bindings/wasm/contract.go:141-149`).

Unknown JSON fields are rejected, as are the removed writer fields
`pageSize`, `orientation`, `padding`, and `quality`
(`bindings/wasm/contract.go:41-45`, `bindings/wasm/contract.go:68-92`).

The browser result envelope is `{ ok, mode, mime, schema, version, width,
height, bytes }`; `bytes` holds the UTF-8 JSON document
(`bindings/wasm/main_js_wasm.go:62-77`). A failure returns
`{ ok: false, error: { code, message } }` with a stable code:
`invalid_request`, `resource_limit`, `timeout`, `render_error`, or
`internal_error` (`bindings/wasm/contract.go:190-208`,
`bindings/wasm/main_js_wasm.go:79-88`).

The JSON document (`bindings/wasm/drawing_list.go:46-77`):

| Field | Meaning |
|-------|---------|
| `schema`, `version` | `blinkless.drawinglist/1` and `1` (`bindings/wasm/drawing_list.go:19-26`) |
| `units` | `points` for every operation coordinate (`bindings/wasm/drawing_list.go:24-25`) |
| `width`, `height` | Canvas size in CSS pixels (`bindings/wasm/drawing_list.go:53-55`) |
| `pxPerPt`, `ptPerPx` | `Display.PixelPerPoint` (96/72) and `Display.PointsPerPixel` (72/96) (`bindings/wasm/drawing_list.go:56-60`) |
| `ops` | One entry per `layout.DisplayOp`, source order (`bindings/wasm/drawing_list.go:61-63`) |
| `order` | Indexes into `ops` in paint order (`bindings/wasm/drawing_list.go:64-66`) |
| `boxes` | Element border boxes, CSS pixels, document order (`bindings/wasm/drawing_list.go:67-68`) |
| `groups` | Blend and isolation groups referenced by ops (`bindings/wasm/drawing_list.go:69-70`) |
| `fonts` | Font faces referenced by text and bullet entries (`bindings/wasm/drawing_list.go:71-73`) |
| `images` | Encoded image payloads referenced by image entries (`bindings/wasm/drawing_list.go:74-76`) |

Font and image ids are content-derived: `f-` or `i-` plus the first 8 bytes of
SHA-256 over the payload, so identical payloads share one table entry
(`bindings/wasm/drawing_list.go:214-238`,
`bindings/wasm/drawing_list.go:595-602`). Image format is sniffed as `png`,
`jpeg`, or `binary` (`bindings/wasm/drawing_list.go:604-614`). A serialized
result is capped at 32 MiB and fails with `resource_limit` rather than
truncating (`bindings/wasm/contract.go:28-31`,
`bindings/wasm/drawing_list.go:243-259`).

The field-by-field schema with examples lives in
[../library-api.md](../library-api.md). `bindings/wasm/drawing_list.go` is its
executable form.

## What is gone

These names no longer exist in the tree. They belonged to the removed PDF and
page-image pipelines, and the current API has no one-for-one replacements:

- The PDF writer and its page model. Layout does not write a PDF and does not
  split the list onto pages (`layout/doc.go:2-3`); the root package says the
  engine does not encode a page bitmap (`doc.go:4`).
- `internal/imageout` and the page rasterizer. An image operation keeps its
  encoded bytes, and only orientation or a clip re-encodes that one operation
  as a PNG (`layout/doc.go:6-9`).
- The root `ImageDocument`, `WriteImage`, and `Image`. The root package now
  declares the pipeline summary only (`doc.go:1-9`).
- `layout.Lay` and `screen.Render`. The layout entry points are `DisplayList`
  and `DisplayListOptions` (`layout/displaylist.go:126-139`); there is no
  `screen` package.

A caller that needs bytes still has the drawing-list JSON from
`bindings/wasm`, or can replay `Display.Ops` onto its own canvas and keep text
as glyphs (`layout/displaylist.go:126-134`).
