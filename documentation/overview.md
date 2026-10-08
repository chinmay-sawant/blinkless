# Overview

blinkless is a pure-Go HTML renderer. It does not write PDF files and it does not start a browser.

The module path is `github.com/chinmay-sawant/blinkless`. Direct dependencies are `go-text/typesetting` (glyph positioning) and `tdewolff/canvas` (SVG images). Default builds use `CGO_ENABLED=0`.

## Pipeline

```text
file or URL or inline HTML
        -> internal/load
        -> internal/html
        -> internal/css
        -> internal/layout   (boxes and a drawing list)
        -> internal/imageout (one PNG or JPEG)
```

`layout.DisplayList` stops at the drawing list. Each operation is a rectangle, a line of text, an image, or a link box. Coordinates are canvas points, y down.

`screen.Render` parses inline HTML, applies style sheets passed by the caller, and returns a PNG. It does not fetch linked style sheets or images. `ImageDocument` and `bin/blinkless` do, under the local-file and network rules.

## What it is for

- HTML templates that should become a picture: invoices, receipts, posters, simple pages
- A drawing list a host program can inspect
- A static image binary with no browser process

## What it is not

- A PDF writer. Page files, xref, embedded font programs, PDF profiles, outlines, and tables of contents are not produced.
- A browser. JavaScript does not run. CSS coverage is the subset in the compatibility matrix.
- A pixel match for Chrome.

## Fonts

Faces live in `internal/fonts/assets`. `LoadDefaultFaces` supplies Liberation and DejaVu. Extra faces come from `--font-path` and `@font-face`.
