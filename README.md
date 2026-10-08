# blinkless

blinkless turns HTML into a drawing list and, when you ask for a picture, a PNG or JPEG.

The list is the product. Each entry is a rectangle, a line of text, an image, or a link box, in canvas points with y pointing down. `layout.DisplayList` returns that list. `layout.Lay`, `screen.Render`, and `ImageDocument` draw the same list into an image.

```text
load -> parse -> style -> layout -> drawing list -> PNG or JPEG
```

There is no PDF writer. Page boxes, xref, embedded font programs, and PDF profiles are gone. Font loading and glyph positioning stay, because the picture path needs them.

## Library

```go
import "github.com/chinmay-sawant/blinkless/screen"

frame, err := screen.Render(ctx, html, 800, 600)
```

`ImageDocument` is the root image API. `html.Parse`, `css.Apply`, and `layout.DisplayList` are the pieces underneath.

The module path is `github.com/chinmay-sawant/blinkless`. The root package clause is `package blinkless`.

## Command

```bash
make build
./bin/blinkless --html '<h1>Hello</h1>' -o hello.png
```

## Tests

```bash
make test-quick
```

`make golden` renders every golden HTML fixture to a PNG and checks the file header. `make samples` writes those PNGs under `output/`.
