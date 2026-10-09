# Prepare and the render lifecycle

`internal/convert` has no PDF job and no Go files in its own directory. Two subpackages remain.

## prepare

`internal/convert/prepare` is the shared load, parse, stylesheet, and `@font-face` phase. `prepare.Document` (`prepare.go:210`) calls `loader.Load`, `html.ParseDocument`, `CollectSheets`, and `MergeFontFaces`, and returns `Prepared{Resource, Root, Resources, Sheets, Registry}` (`prepare.go:198`). The registry is `*fonts.Registry` from `internal/fonts`, imported as `pdf` in this package (`prepare.go:15`).

`css.Apply` is the production caller. It builds a `ResourceContext` (`css/css.go:176`), then calls `prepare.CollectTreeSheets` (`css/css.go:186`) and `MergeFontFaces` (`css/css.go:200`). `ResourceContext` (`prepare.go:43`) wraps `load.ResourceContext`; the exported `Loader`, `Base`, and `Load` fields are deprecated snapshots (`prepare.go:47-52`).

## render

`internal/convert/render` owns the stage order. `Pipeline` (`pipeline.go:14`) has three methods: `RenderObjects`, `Assemble`, `Finalize`. `Run` (`pipeline.go:28`) calls them in that order and checks the context between stages. No production pipeline implements the interface in this tree; `pipeline_test.go` exercises the ordering, the stop-on-error behavior, and the nil guards.

`render.ErrNilContext` re-exports `errs.ErrNilContext` (`pipeline.go:22`).

There is no TOC pass, no outline, no link annotation pass, and no `doc.Write`. The stage names survive from the PDF pipeline; nothing here builds pages or a PDF.
