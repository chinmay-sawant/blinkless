<p align="center">
  <img src="assets/01-flat-cartoon.png" alt="Blue Blinkless gopher arranging page content" width="160">
</p>

<h1 align="center">blinkless</h1>

blinkless is a layout engine for HTML and CSS. Both are inputs. HTML alone is not enough: the tree has no placement until the stylesheets are applied.

`html.Parse` reads the document. `css.Apply` applies its `<style>` blocks and any extra sheets you pass. `layout.DisplayList` places that styled document and returns a drawing list. The list is the output.

A browser engine such as Blink does this too, and then it keeps going. It runs script and it paints pixels. This engine does not run script. The drawing list is where the layout stops.

Each entry is one drawing operation: a filled rectangle, a stroked rectangle, a line, a text run, an image, or a link box. Operation coordinates are canvas points, y down. Text Y is the baseline. Element boxes on the same result are CSS pixels, y down, origin at the top left.

```text
HTML + CSS -> parse -> style -> layout -> drawing list
```

```go
tree, err := html.Parse([]byte("<h1>Hello</h1>"))
styled, err := css.Apply(ctx, tree, css.Options{WidthPx: 800, HeightPx: 600})
display, err := layout.DisplayList(ctx, styled)

for _, index := range display.Order {
    op := display.Ops[index]
    // draw op on your own canvas
}
```

`display.Order` is paint order. `display.Ops` is source order.

The module path is `github.com/chinmay-sawant/blinkless`. The root package clause is `package blinkless`. The layout call is `github.com/chinmay-sawant/blinkless/layout`.
