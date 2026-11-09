# Overview

blinkless is a pure-Go HTML and CSS layout engine. It returns a drawing list. It does not write PDF files, it does not encode a page bitmap, and it does not start a browser.

The module path is `github.com/chinmay-sawant/blinkless`. Direct dependencies are `go-text/typesetting` (glyph positioning) and `tdewolff/canvas` (SVG images). Default builds use `CGO_ENABLED=0`.

## Pipeline

```text
HTML + CSS
        -> html.Parse
        -> css.Apply
        -> layout.DisplayList
```

`layout.DisplayList` returns the drawing list. Each operation is a filled or stroked rectangle, a line, a text run, an image, a link target, a list marker, or collapsed table border segments. Coordinates are canvas points, y down. The engine does not encode that list as a page PNG or JPEG.

0.1.1 is the current release. It adds batch subtree relayout, carries `backdrop-filter` metadata on display groups, and cuts layout allocations on the patch path. The output is still the drawing list. The 0.0.2 resource caps stay: a stylesheet over 32 MiB is rejected, element nesting stops at 1024, `html.Parse` does not cap the source length, and SVG text is removed before raster. Details are in `plans/v0.1.1/release-note/v0.1.1.md`.

An image operation keeps the source bytes. When orientation or a clip cannot stay in those bytes, that one operation is re-encoded as a PNG. That bitmap fallback is part of the list. It is not a picture of the page.

## What it is for

- A drawing list a host program can replay
- HTML and CSS placement for invoices, receipts, and simple pages

## What it is not

- A PDF writer. Page files, xref, embedded font programs, PDF profiles, outlines, and tables of contents are not produced.
- A browser. JavaScript does not run. CSS coverage is the subset in the compatibility matrix.
- A pixel match for Chrome.

## Fonts

Faces live in `internal/fonts/assets`. `LoadDefaultFaces` supplies Liberation and DejaVu. Extra faces come from configured font directories and `@font-face`.
