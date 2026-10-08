# Go library API

Module path: `github.com/chinmay-sawant/blinkless`.

The root package exports `ImageDocument`, `Content`, and the source helpers `HTML`, `File`, and `URL`. It does not export a PDF document type.

## Image

```go
blinkless "github.com/chinmay-sawant/blinkless"

doc := &blinkless.ImageDocument{
    Source:          blinkless.File("report.html"),
    Width:           1024,
    AllowLocalFiles: true,
}
err := doc.WriteImage(ctx, writer)
png, err := doc.Image(ctx)
```

`Format` is `png`, `jpg`, or `jpeg`. `Crop` is a pixel rectangle. `Network` is the same policy type the loader uses.

## Drawing list and inline PNG

| Package | Call | Result |
|---------|------|--------|
| `html` | `Parse` | An HTML tree |
| `css` | `Parse`, `Apply` | Stylesheets applied to that tree |
| `layout` | `DisplayList` | Boxes and drawing operations, no image |
| `layout` | `Lay` | The same placement painted to an image |
| `screen` | `Render` | PNG bytes plus element boxes for inline HTML |

`screen.Render` does not load linked style sheets or images. Pass the CSS in the HTML `<style>` block, or use `ImageDocument` when the file has links.

## Errors

`ImageDocument.Validate` and `WriteImage` return errors for a bad source, a bad size, or a bad crop before the renderer runs. `errors.Is` matches the sentinels in the root package (`ErrEmptyHTML`, `ErrInvalidContent`, `ErrMissingImageOutput`, `ErrNilContext`).
