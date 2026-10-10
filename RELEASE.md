# Release

blinkless 0.0.1 is the renderer cut. The PDF writer is not part of this tree.

`VERSION` is not used. There is no image binary and no `internal/cli` package.

## Gates

Run these before calling the tree releasable:

1. `go build ./...`
2. `make test-quick`
3. `make golden`
4. `make lint`
5. `make claim-scan` after documentation matches the renderer

Do not run veraPDF or a PDF byte compare. Those checks measured the writer.

## What ships

- Drawing list from HTML and CSS (`layout.DisplayList`)
- Bitmap fallback on a single image operation when orientation or a clip must re-encode that payload as PNG

## What does not ship

- PDF files, PDF profiles, the `blinkless` command, veraPDF, and the golden PDF corpus
