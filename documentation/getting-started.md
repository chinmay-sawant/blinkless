# Getting started

## Requirements

Go 1.26 or newer. The module pins the toolchain in `go.mod`. The first build downloads `go-text/typesetting` and `tdewolff/canvas`.

## Build

```sh
make build
```

`make build` compiles the packages. It does not write `bin/blinkless`. `make golden` runs the drawing-list tests in `layout`.

## Library

```go
package main

import (
    "context"
    "fmt"

    "github.com/chinmay-sawant/blinkless/css"
    "github.com/chinmay-sawant/blinkless/html"
    "github.com/chinmay-sawant/blinkless/layout"
)

func main() {
    tree, err := html.Parse([]byte("<h1>Hello</h1>"))
    if err != nil {
        panic(err)
    }

    styled, err := css.Apply(context.Background(), tree, css.Options{
        WidthPx:  800,
        HeightPx: 600,
    })
    if err != nil {
        panic(err)
    }

    display, err := layout.DisplayList(context.Background(), styled)
    if err != nil {
        panic(err)
    }

    fmt.Println(len(display.Ops), display.Width, display.Height)
}
```

The root package clause is `package blinkless`. The layout call is `github.com/chinmay-sawant/blinkless/layout`.

## Tests

```sh
make test-quick
```

Targeted package tests during a change:

```sh
go test -p 2 -parallel 2 ./layout ./internal/layout
```
