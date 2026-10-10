# WASM fixture data

This directory holds the self-contained input shared by the browser adapter
tests and the WASM consumer check. It stays separate from `testdata/golden`
because the golden corpus checks native layout output, while this fixture
feeds the JSON drawing-list contract.

- `sample.html` is inline HTML with print CSS, a table, a forced page break,
  and a data-URL image. It does not require a file or network request.
- `manifest.json` describes the drawing-list request and the structural
  expectations the native tests and `scripts/wasm-consumer-check.mjs` both
  enforce: schema name, fixture, viewport, minimum operation/box/resource
  counts, and the operation kinds that must appear.

The contract itself (request shape, result document, units, operation kinds,
resource tables) is documented in `documentation/library-api.md`. The manifest
carries no PDF, PNG, or JPEG output assertions; the adapter emits JSON only.
