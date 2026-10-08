# Release

blinkless 0.0.1 is the renderer cut. The PDF writer is not part of this tree.

`VERSION` is not used. The image binary stamps `internal/cli.Version` from the Makefile as `0.0.1`.

## Gates

Run these before calling the tree releasable:

1. `go build ./...`
2. `make test-quick`
3. `make lint`
4. `make claim-scan` after documentation matches the renderer

Do not run `make golden`, veraPDF, or a PDF byte compare. Those checks measured the writer.

## What ships

- Drawing list from HTML and CSS (`layout.DisplayList`)
- PNG and JPEG through `ImageDocument`, `screen.Render`, and `bin/blinkless`

## What does not ship

- PDF files, PDF profiles, the `blinkless` command, veraPDF, and the golden PDF corpus
