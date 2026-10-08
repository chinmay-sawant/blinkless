# Prepare and the image pipeline

`internal/convert` no longer has a PDF job. The parent directory has no Go files. Two subpackages remain.

## prepare

`internal/convert/prepare` loads a document, parses HTML, collects style sheets, and merges `@font-face`. `imageout` calls `prepare.Document` before layout. The font registry it returns is `*fonts.Registry` from `internal/fonts`.

## render

`internal/convert/render.Run` calls three stages in order: `RenderObjects`, `Assemble`, `Finalize`. The image pipeline implements that interface. Its assemble step does not build a PDF. `imageout.RunRequest` is the caller (`internal/imageout/imageout.go`).

There is no TOC, no PDF outline, no link annotation pass, and no `doc.Write`.
