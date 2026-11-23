# Prepare phase

`internal/convert` has no PDF job and no Go files in its own directory. One subpackage remains: `prepare`.

## prepare

`internal/convert/prepare` is the shared load, parse, stylesheet, and `@font-face` phase. `prepare.Document` (`prepare.go:210`) calls `loader.Load`, `html.ParseDocument`, `CollectSheets`, and `MergeFontFaces`, and returns `Prepared{Resource, Root, Resources, Sheets, Registry}` (`prepare.go:198`). The registry is `*fonts.Registry` from `internal/fonts`, imported as `pdf` in this package (`prepare.go:15`).

`css.Apply` is the production caller. It builds a `ResourceContext` (`css/css.go:176`), then calls `prepare.CollectTreeSheets` (`css/css.go:186`) and `MergeFontFaces` (`css/css.go:200`). `ResourceContext` (`prepare.go:43`) wraps `load.ResourceContext`; the exported `Loader`, `Base`, and `Load` fields are deprecated snapshots (`prepare.go:47-52`).
