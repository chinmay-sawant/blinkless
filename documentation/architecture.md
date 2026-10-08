# Architecture

blinkless loads HTML, parses it, applies CSS, builds a drawing list, and can encode that list as one PNG or JPEG.

The module path is `github.com/chinmay-sawant/blinkless`. Notes with more detail live under [architecture/](architecture/).

## Package map

| Package | Responsibility |
|---------|----------------|
| root `document.go` | `ImageDocument`, `Content`, `WriteImage` |
| `html` | Public parser |
| `css` | Public stylesheets |
| `layout` | `Lay` paints an image. `DisplayList` returns drawing operations and no image |
| `screen` | `Render` parses inline HTML, applies CSS, lays out, returns a PNG |
| `cmd/blinkless` | Image command, built as `bin/blinkless` |
| `internal/app` | `RunImage` |
| `internal/cli` | Argument parse for the image command |
| `internal/settings` | Page, load, font, and image settings |
| `internal/load` | HTTP, file, and `data:` fetches under an ACL |
| `internal/html` | Tokenizer and tree. No JavaScript |
| `internal/css` | Selector match and cascade |
| `internal/layout` | Boxes and the drawing list (`Op`) |
| `internal/fonts` | Face parse, registry, glyph positions, bundled font files |
| `internal/convert/prepare` | Load, parse, collect style sheets, merge `@font-face` |
| `internal/convert/render` | Stage order: render, assemble, finalize |
| `internal/imageout` | Rasterize the drawing list to PNG or JPEG |
| `internal/svg` | SVG used as an `<img>` |

There is no `internal/pdf` package and no PDF command.

## Pipeline

```text
input
  -> internal/load
  -> prepare.Document
  -> internal/layout   drawing list
  -> internal/imageout PNG or JPEG
```

`imageout.RunRequest` (`internal/imageout/imageout.go`) builds that pipeline. It does not call a PDF writer.
