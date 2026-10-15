# blinkless 0.0.2

Security release after `v0.0.1`. The engine still returns `layout.DisplayList` from HTML and CSS. It does not write a PDF, encode a page bitmap, or ship a `blinkless` command.

## Security

Hostile documents stop at fixed budgets. A stylesheet over 32 MiB is rejected. HTML nesting and the formatting list stop at 1024. Dashed edges, underlines, and border-image tiles stop at 1024. Decoded images stop at 16 million pixels. SVG text is removed before raster so that path does not load a system font.

`html.Parse` still does not cap its own byte length. Use `ParseWithLimit` or bound the bytes before parse. The default limit there is 64 MiB.

Local files open through `os.Root`. Restricted mode refuses private addresses and a proxied hostname. The C binding caps the allow list and the last-error buffer, rejects an HTML length at or above 2^63, and recovers panics. WASM applies the JSON and HTML byte caps.

The closed records are `plans/v0.0.1/security/` ledgers 01 through 09. The note in the repo is `plans/v0.0.2/release-note/v0.0.2.md`.

## Also in this tag

The Python package, the PyPI workflow, and the tag-versus-file version check are removed. GitHub CI runs `make test` and `make lint` only.

`Makefile` `BINDINGS_VERSION` and `WASM_VERSION` stay `0.0.1`. There is no `VERSION` file.

## Still true

A canvas draw that has already started still finishes. A C caller must pass a valid pointer. No script runs. That does not prove every bug is gone.
