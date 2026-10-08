# Layout engine

`internal/layout` turns a parsed HTML tree and its CSS into boxes and a drawing list. That list is the renderer's output. Nothing in this package writes a PDF page.

The public package `layout` (`layout/doc.go`) exposes two calls on the same placement:

- `Lay` paints the placement to an image.
- `DisplayList` returns the drawing operations and no image.

Coordinates on the public boxes are CSS pixels, y down, origin at the top left. `DisplayList` operations are canvas points, y down. For text, Y is the baseline (`layout/displaylist.go`).

## What an operation is

Each operation is a rectangle, a line of text, an image, or a link box. The internal type is `Op` in `internal/layout/layout.go`. The public name is `layout.DisplayOp`.

A caller can read the list and draw it on its own canvas. `internal/imageout` is the built-in drawer: it walks the same operations and encodes one PNG or JPEG (`imageout.RunRequest` in `internal/imageout/imageout.go`).

## How a document gets there

1. `html.Parse` builds the tree.
2. `css.Apply` attaches stylesheets.
3. `layout.LayoutContext` resolves styles, loads faces from `internal/fonts` when the caller did not pass a registry, and emits operations.
4. `imageout` rasterizes those operations. `screen.Render` does the inline-HTML path and returns a PNG plus element boxes. It does not fetch linked files.

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
