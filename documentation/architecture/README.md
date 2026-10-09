# Architecture notes

These notes describe the engine as it is now. There is no PDF writer and no page bitmap encoder in the tree.

| Note | What it covers |
|------|----------------|
| [01-entrypoints-cli.md](01-entrypoints-cli.md) | Public packages, bindings, and the internal pipeline |
| [02-library-api.md](02-library-api.md) | Library API |
| [03-settings.md](03-settings.md) | Settings the load and layout stages read |
| [04-load.md](04-load.md) | Fetch and ACL |
| [05-html-parser.md](05-html-parser.md) | HTML tree |
| [06-css.md](06-css.md) | Stylesheets |
| [07-layout.md](07-layout.md) | Boxes and drawing operations |
| [08-convert-pipeline.md](08-convert-pipeline.md) | Prepare and the render lifecycle |
| [10-imageout-svg.md](10-imageout-svg.md) | SVG images and the image-op fallback |

The one-page map is [../architecture.md](../architecture.md).
