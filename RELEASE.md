# Release

blinkless 0.1.1 is the performance and patch-path release on the 0.1 line.
It follows 0.1.0. The PDF writer is not part of this tree.

`VERSION` is not used. There is no image binary and no `internal/cli` package.
The release record is `CHANGELOG.md` plus the ledger under `plans/v0.1.1/`.

## Gates

Run these before calling the tree releasable:

1. `go build ./...`
2. `make test-quick`
3. `make golden`
4. `make lint`
5. `make claim-scan` after documentation matches the renderer
6. `make matrix-check` after documentation matches the catalog

Do not run veraPDF or a PDF byte compare. Those checks measured the writer.

`make catalog-check` is red on `master` before 0.1.1. Seven animation and
transition sample strings are absent from the sibling OwnFrame checkout, and
the catalog still lists the removed `wrapBreakInsideAvoid` handler for
`wrap-inside`. Do not add it to this gate list until both are fixed.

## What ships

- Drawing list from HTML and CSS (`layout.DisplayList`)
- Bitmap fallback on a single image operation when orientation or a clip must re-encode that payload as PNG

## What does not ship

- PDF files, PDF profiles, the `blinkless` command, veraPDF, and the golden PDF corpus
