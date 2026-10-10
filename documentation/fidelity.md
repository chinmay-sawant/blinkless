# Fidelity guide

blinkless lays out authored HTML and CSS into a drawing list. The list is the output. The engine does not write PDF files and it does not encode a page PNG or JPEG. It is not a browser.

This guide is the product-facing fidelity story. The normative per-property contract is the [compatibility matrix](compatibility-matrix.md). Capability work and its evidence are tracked in [`plans/v0.0.1/html-css-json-compatibility-checklist.md`](../plans/v0.0.1/html-css-json-compatibility-checklist.md).

---

## Product positioning

| You need | Expectation |
|----------|-------------|
| HTML and CSS templates laid out into a drawing list | **In scope** |
| Repeatable layout, pure Go, `CGO_ENABLED=0` default build, no browser process | **In scope** |
| Pixel-perfect clone of an arbitrary website | **Out of scope** |
| Full CSS (flex, grid, absolute, fixed, sticky) | **Partial, per property.** The catalog in the matrix is the contract. Flex, grid, and positioning place onto one canvas; sticky is tagged during layout and there is no page scrollport |
| JavaScript-driven pages | **Out of scope.** `<script>` is parsed as raw text and never executed |
| PDF, PDF/A, PDF/UA, encryption, forms | **Not produced.** The writer was removed |
| Page PNG/JPEG rendering | **Not produced.** The drawing list is the output. One image operation is re-encoded as a PNG only when orientation or a clip requires it |
| Full Unicode / CJK typesetting | **Partial.** Bundled Liberation and DejaVu faces plus fallback; extra faces from configured font directories and `@font-face`; Arabic shaping via `go-text/typesetting`; Indic partial; no CGO HarfBuzz |

### Report templates

Controlled templates (invoices, receipts, statements, reports) can rely on what the matrix records: richer selectors (`:nth-child`, attribute, sibling combinators), float lite, `inline-block`, `box-sizing: border-box`, `text-align: justify` on non-final lines, and table-cell `vertical-align` top/middle/bottom. Lists carry `decimal`, alpha, and roman markers. Tables support `rowspan`, `<caption>`, and `border-collapse` lite. This is not a full CSS2 float engine: prefer `clear` after chrome, and treat complex float wrap as best-effort.

---

## How to read the matrix

Status labels in [compatibility-matrix.md](compatibility-matrix.md) come from `testdata/css/catalog/properties.json` (schema v1):

| Label | Meaning |
|-------|---------|
| **Implemented** | A handler parses the value, a consumer outside the style layer reads it, and a behavior test resolves |
| **Partial** | A handler and a consumer exist, but a named behavior is missing or unverified |
| **Unsupported** | No consumer and no observable behavior. The declaration may be parsed and stored, or dropped entirely |
| **Intentionally ignored** | Deliberately ignored print-noop UI chrome |

Section 1 of the matrix is a rendering allowlist for HTML tags, not a parser conformance claim. If a row says Implemented, the row cites a code path and a behavior test. Prefer the matrix over marketing prose.

---

## Claims, kept separate

Parser compliance, CSS coverage, and rendering comparisons are three claims with three evidence sets. Do not merge them into one compatibility percentage.

### CSS coverage

`testdata/css/catalog/properties.json` is the normative record. Measured 2026-10-09: 785 rows, 90 implemented, 301 partial, 387 unsupported, 7 intentionally ignored. The upstream inventory is webref `ed/css` revision `1f2ec8f74a80c14066b4c7d6822cee59f69fa03b` (821 properties). `make catalog-check` fails when the catalog disagrees with the code, and `make matrix-check` fails when the matrix section drifts from the catalog. A test-strength pass over some implemented rows is still open in the checklist. A catalog status is not a browser-parity claim.

### HTML parser compliance

The engine measures against a pinned copy of `html5lib/html5lib-tests` revision `9329e64694e7835d0dcff9811e22856ef6ad16f9` (2026-06-22) under `testdata/html-conformance/`. Latest recorded run (2026-10-09, wave D): tokenizer 6686 of 7036 passed with 0 failed and 350 unsupported; tree construction 1225 of 1792 passed with 331 failed, 228 unsupported, and 8 script-on-only cases excluded. The claim is UTF-8 parsing with scripting disabled. Fragment parsing, select and template states, and integration remain open. This is not a browser byte-encoding claim or a full parser compliance claim.

### Rendering comparisons

Browser comparisons are per-case, not parity. The recorded 2026-10-09 geometry runs compared 11 flex, grid, logical-property, and text fixtures against Chrome `143.0.7499.40` at 1024x768 with a 1.0 px tolerance: the wave A baseline had 101 of 215 elements within tolerance, and the wave D recheck, after the C1-C5 fixes, had 151 of 215. `test/chrome` holds the case manifest, currently 26 completed and 14 blocked with measured evidence or reasons. A pass here does not promote a catalog row and does not cover arbitrary pages.

---

## How we prove fidelity

| Mechanism | What it proves |
|-----------|----------------|
| Catalog and matrix checks (`make catalog-check`, `make matrix-check`) | Property statuses match the code, and the matrix section matches the catalog |
| Public drawing-list tests (`make golden`, `go test ./layout -run TestDisplay`) | Op kinds, paint order, unit factors, text faces, and real style coverage on the public API |
| Pinned parser corpus (`TestHTMLConformance`, `HTML_CONFORMANCE_STRICT=1`) | UTF-8 tokenizer and tree-construction cases against the pinned html5lib revision |
| Recorded browser comparisons (`test/chrome`, geometry runs) | Specific fixture geometry against a named Chrome version, per case |
| Unit and integration tests under `internal/*` | Layout, CSS match, load, fonts, SVG rasterization, and bindings |

Details: [testdata/html-conformance/README.md](../testdata/html-conformance/README.md), [test/chrome/README.md](../test/chrome/README.md).

---

## Failure modes (graceful degrade)

The engine should not crash on unsupported input:

| Input | Behavior |
|-------|----------|
| Unknown CSS property or value | Declaration ignored |
| Unknown `display` value | Ignored. The matrix records what flex, grid, and positioning cover |
| `<script>` | Parsed as raw text and never executed; the parser runs with scripting disabled |
| Missing font family name | Falls back through the author's stack, then Liberation Sans (DejaVu for uncovered glyphs) |
| Missing bold face | Fake stroke bold only if the face is missing |
| Missing or corrupt image | Paint skipped; no process crash |
| Local file without ACL opt-in | Load denied (secure default) |
| Oversized HTTP body or timeout | Loader limits; error returned |

Security defaults: [THREAT-MODEL.md](THREAT-MODEL.md), [integration-security.md](integration-security.md).

---

## Claims language (allowed vs banned)

Allowed: controlled HTML and CSS templates, pure Go, repeatable layout, drawing-list output, `CGO_ENABLED=0` default builds, and catalog statuses reported with their named limitations.

Banned or over-claim: pixel perfect, full CSS, browser replacement, WebKit parity, Wikipedia visual parity, marketing pixel match, "paste any URL and get Chrome-quality print", any claim that this tree writes PDF, PDF/A, PDF/UA, or page PNG/JPEG files, presenting the CSS catalog as browser coverage, and presenting parser corpus counts or per-case geometry runs as overall rendering parity.

---

## Related docs

| Doc | Role |
|-----|------|
| [compatibility-matrix.md](compatibility-matrix.md) | Normative support contract |
| [fonts.md](fonts.md) | Bundled faces, discovery, `@font-face`, shaping limits |
| [overview.md](overview.md) | Product overview |
| [architecture.md](architecture.md) | Pipeline packages |
| [library-api.md](library-api.md) | Library calls and the WASM drawing-list JSON schema |
| [deferred.md](deferred.md) | Deferred features |
| [getting-started.md](getting-started.md) | Build and first call |
| [plans/v0.0.1/html-css-json-compatibility-checklist.md](../plans/v0.0.1/html-css-json-compatibility-checklist.md) | Active capability ledger and evidence |
