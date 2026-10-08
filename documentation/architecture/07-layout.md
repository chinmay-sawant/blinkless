# Layout engine

`internal/layout` turns a parsed HTML tree and its CSS into boxes and a drawing list. That list is the renderer's output. Nothing in this package writes a PDF page.

The public package `layout` (`layout/doc.go`) exposes `DisplayList`. It returns the drawing operations. It does not paint a page bitmap.

Coordinates on the public boxes are CSS pixels, y down, origin at the top left. `DisplayList` operations are canvas points, y down. For text, Y is the baseline (`layout/displaylist.go`).

## What an operation is

Each operation is a rectangle, a line of text, an image, or a link box. The internal type is `Op` in `internal/layout/layout.go`. The public name is `layout.DisplayOp`.

A caller reads the list and draws it on its own canvas. An image operation keeps its encoded bytes. When orientation or a clip changes those pixels, that one operation is re-encoded as a PNG (`encodePNGImage` in `internal/layout/image_exif.go`). That is the bitmap fallback. It is not a picture of the page.

## How a document gets there

1. `html.Parse` builds the tree.
2. `css.Apply` attaches stylesheets.
3. `layout.DisplayList` calls `layout.LayoutContext`, which resolves styles, loads the default face from `internal/fonts`, and emits operations.

An unset viewport width falls back to 1024 CSS pixels. An unset height uses that width as the percentage-height containing block. The canvas is the taller of the content and the requested minimum.

There is no second pass that splits the list onto PDF pages. `page-break-*` values are still parsed onto the style. They do not produce extra pages.

## What layout owns

- Block, inline, table, flex, grid, and float placement
- The drawing list (`Op` / `DisplayOp`)
- Font choice and glyph positions through `internal/fonts`
- Paint appearance shared with the rasterizer: `StyleOf` and `FakeBoldFor` in `internal/layout/paint_style.go`

## What it does not own

- Fetching URLs (`internal/load`)
- Encoding PNG or JPEG (`internal/imageout`)
- PDF objects, page numbers, headers, footers, outlines, or a table of contents

JavaScript does not run. CSS outside the compatibility matrix is ignored or stored without a visual effect.
