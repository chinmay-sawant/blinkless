# Go library API

Module path: `github.com/chinmay-sawant/blinkless`.

The root package is the module anchor. The calls live in `html`, `css`, and `layout`.

## Drawing list

| Package | Call | Result |
|---------|------|--------|
| `html` | `Parse` | An HTML tree |
| `css` | `Parse`, `Apply` | Stylesheets applied to that tree |
| `layout` | `DisplayList` | Boxes and drawing operations |

`css.Apply` needs `WidthPx` and `HeightPx` greater than zero. Pass linked CSS through the HTML `<style>` block or `css.Options.Extra`.

An image on the list keeps its encoded bytes. Orientation or a clip that cannot stay in those bytes re-encodes that one operation as a PNG. That is the bitmap fallback.

## Errors

`html.Parse`, `css.Apply`, and `layout.DisplayList` return errors for a bad document, a bad size, or a nil context. `errors.Is` matches `css.ErrBadSize`, `css.ErrNilContext`, `layout.ErrNilContext`, and `layout.ErrNilDocument`.
