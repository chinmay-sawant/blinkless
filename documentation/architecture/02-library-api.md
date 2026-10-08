# Library API

The root package (`document.go`, `api.go`) exposes `ImageDocument`.

`WriteImage` validates the document, builds an `imageout.Request`, and calls `imageout.RunRequest`. `Image` buffers that PNG or JPEG.

`Content` is one source: in-memory HTML (`HTML`), a file path (`File`), or a URL (`URL`).

Other public packages:

- `html.Parse` returns the tree.
- `css.Parse` and `css.Apply` attach stylesheets.
- `layout.DisplayList` returns drawing operations and no image (`layout/displaylist.go`).
- `layout.Lay` paints the same placement.
- `screen.Render` (`screen/screen.go`) is parse, apply, lay, and a PNG for inline HTML. It does not fetch linked resources.

The module path is `github.com/chinmay-sawant/blinkless`. The root package clause is `package blinkless`.
