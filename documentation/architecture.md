# Architecture

blinkless loads HTML, parses it, applies CSS, and returns a drawing list.

The module path is `github.com/chinmay-sawant/blinkless`. Notes with more detail live under [architecture/](architecture/).

## Package map

| Package | Responsibility |
|---------|----------------|
| `html` | Public parser |
| `css` | Public stylesheets |
| `layout` | `DisplayList` returns drawing operations |
| `internal/settings` | Page, load, and font settings |
| `internal/load` | HTTP, file, and `data:` fetches under an ACL |
| `internal/html` | Tokenizer and tree. No JavaScript |
| `internal/css` | Selector match and cascade |
| `internal/layout` | Boxes and the drawing list (`Op`) |
| `internal/fonts` | Face parse, registry, glyph positions, bundled font files |
| `internal/convert/prepare` | Load, parse, collect style sheets, merge `@font-face` |
| `internal/convert/render` | Stage order: render, assemble, finalize |
| `internal/svg` | SVG used as an `<img>`, rasterized onto one image operation |

There is no `internal/pdf` package and no PDF command.

## Pipeline

```text
HTML + CSS
  -> html.Parse
  -> css.Apply
  -> internal/layout   drawing list
```

There is no page PNG or JPEG encoder. An image operation may still carry a PNG payload when orientation or a clip re-encodes that one image.
