# Architecture notes

These notes describe the renderer as it is now. The PDF writer is not in the tree.

| Note | What it covers |
|------|----------------|
| [01-entrypoints-cli.md](01-entrypoints-cli.md) | `bin/blinkless` |
| [02-library-api.md](02-library-api.md) | Root `ImageDocument` and the public packages |
| [03-settings.md](03-settings.md) | Settings the image path reads |
| [04-load.md](04-load.md) | Fetch and ACL |
| [05-html-parser.md](05-html-parser.md) | HTML tree |
| [06-css.md](06-css.md) | Stylesheets |
| [07-layout.md](07-layout.md) | Boxes and drawing operations |
| [08-convert-pipeline.md](08-convert-pipeline.md) | Prepare and the image pipeline |
| [10-imageout-svg.md](10-imageout-svg.md) | Raster output and SVG images |

The one-page map is [../architecture.md](../architecture.md).
