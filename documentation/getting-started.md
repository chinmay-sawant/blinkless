# Getting started

## Requirements

Go 1.26 or newer. The module pins the toolchain in `go.mod`. The first build downloads `go-text/typesetting` and `tdewolff/canvas`.

## Build the image command

```sh
make build
./bin/blinkless --html '<h1>Hello</h1>' -o hello.png
./bin/blinkless --allow-local-files -o invoice.png testdata/golden/fixture-01-simple-invoice.html
```

`make samples` writes one PNG per golden body fixture into `output/`. `make golden` renders those fixtures in a test and checks that each file starts with a PNG header.

## Library

```go
package main

import (
    "context"
    "os"

    blinkless "github.com/chinmay-sawant/blinkless"
    "github.com/chinmay-sawant/blinkless/screen"
)

func main() {
    frame, err := screen.Render(context.Background(), []byte("<h1>Hello</h1>"), 800, 600)
    if err != nil {
        panic(err)
    }
    if err := os.WriteFile("hello.png", frame.PNG, 0o644); err != nil {
        panic(err)
    }

    doc := &blinkless.ImageDocument{
        Source: blinkless.HTML([]byte("<h1>Hello</h1>")),
        Width:  800,
    }
    f, err := os.Create("image.png")
    if err != nil {
        panic(err)
    }
    defer f.Close()
    if err := doc.WriteImage(context.Background(), f); err != nil {
        panic(err)
    }
}
```

The root package clause is `package blinkless`. A default import of `github.com/chinmay-sawant/blinkless` is named `blinkless`.

`screen.Render` does not fetch linked files. For a local HTML file with style sheets and images, use `ImageDocument` with `AllowLocalFiles: true` or the command flag `--allow-local-files`.

## Tests

```sh
make test-quick
```

Targeted package tests during a change:

```sh
go test ./internal/imageout ./internal/layout ./screen
```
