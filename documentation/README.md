# blinkless documentation

Guides for the HTML renderer. The module path is `github.com/chinmay-sawant/blinkless`.

The engine loads HTML, parses it, applies CSS, and builds a drawing list. `layout.DisplayList` returns that list. The list is the output: there is no PDF writer, no page PNG or JPEG encoder, and no CLI.

Start with [getting-started.md](getting-started.md) and [overview.md](overview.md).

## Guides

| Document | Purpose |
|----------|---------|
| [overview.md](overview.md) | What the renderer is |
| [getting-started.md](getting-started.md) | Build, first drawing list, library call |
| [library-api.md](library-api.md) | `html`, `css`, `layout`, WASM drawing-list JSON |
| [fidelity.md](fidelity.md) | What layout does and does not promise |
| [compatibility-matrix.md](compatibility-matrix.md) | HTML tags and CSS properties the layout engine accepts |
| [fonts.md](fonts.md) | Bundled faces, `@font-face`, shaping |
| [deferred.md](deferred.md) | CSS that is still out of scope |
| [architecture.md](architecture.md) | Package map |

## Architecture notes

| Document | Purpose |
|----------|---------|
| [architecture/01-entrypoints-cli.md](architecture/01-entrypoints-cli.md) | Public packages, bindings, build targets |
| [architecture/02-library-api.md](architecture/02-library-api.md) | Public Go API |
| [architecture/03-settings.md](architecture/03-settings.md) | Settings the load and layout stages read |
| [architecture/04-load.md](architecture/04-load.md) | Fetch and local-file ACL |
| [architecture/05-html-parser.md](architecture/05-html-parser.md) | HTML tree |
| [architecture/06-css.md](architecture/06-css.md) | Stylesheets and cascade |
| [architecture/07-layout.md](architecture/07-layout.md) | Boxes and the drawing list |
| [architecture/08-convert-pipeline.md](architecture/08-convert-pipeline.md) | Prepare and the render lifecycle |
| [architecture/10-imageout-svg.md](architecture/10-imageout-svg.md) | SVG images and the image-op fallback |

## Security

| Document | Purpose |
|----------|---------|
| [THREAT-MODEL.md](THREAT-MODEL.md) | Local files, network, limits |
| [integration-security.md](integration-security.md) | Embedding the renderer behind HTTP |
