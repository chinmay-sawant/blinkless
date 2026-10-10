# Deferred features and workload priority

This page is the product-facing inventory of work that is **partial**,
**not implemented**, or **not planned**. The live post-MVP ledger is
[`plans/0.2.0/10-canonical-post-mvp-roadmap.md`](../plans/0.2.0/10-canonical-post-mvp-roadmap.md).
Leftover print CSS is tracked in
[`plans/0.2.6/48-canonical-0.2.6-css-coverage.md`](../plans/0.2.6/48-canonical-0.2.6-css-coverage.md)
(phases 48-53 landed; 54-56 closing). Fidelity language and degrade rules:
[fidelity.md](fidelity.md). The normative per-property contract is
[compatibility-matrix.md](compatibility-matrix.md).

Status here is about layout and CSS. PDF writing, PDF/A, and PDF/UA are not deferred. They are not in the product.

---

## Workload priority

The dominant practical workload for this engine is backend-generated
business documents, not arbitrary public websites:

```text
application data
  -> server-side HTML template
  -> HTML/CSS
  -> drawing list
```

Typical documents are invoices, receipts, reports, statements, purchase
orders, contracts, certificates, and shipping documents. Integrations in
Python, PHP, Ruby, and Go commonly expose both HTML/string/file input and
URL input. The template path is usually the primary path because it avoids
an HTTP loopback, authentication and cookie problems, and makes assets and
testing more predictable.

URL input is still valuable when it points to a server-rendered internal
page, such as `/orders/123/print`: it reuses an existing web view, CSS,
data loading, and localisation. It should not be conflated with a
client-rendered SPA URL. A dossier URL whose initial HTML is an empty root
element, with React constructing the document in JavaScript, is a
lower-frequency browser-rendering workload and is not representative of the
core invoice or report path.

Highest-impact order for blinkless:

1. **HTML strings** (in-memory / library `InlineHTML` and similar)
2. **Local HTML template files**
3. **Server-rendered internal URLs**
4. **JavaScript-heavy SPA URLs**

The core product should prioritise tables, pagination, headers and footers,
images, fonts, CSS layout, page breaks, and repeated sections. SPA execution
should remain a separate capability rather than the definition of the main
document workload.

String input via the library is the first-class path for (1). There is no
CLI; stdin HTML is **not** implemented - see the table. Do not treat a public
SPA URL as the acceptance bar for report work.

The opt-in browser adapter is narrower than a browser renderer. It accepts
inline HTML, runs the existing engine in a Web Worker, and returns one
versioned drawing-list JSON document, schema `blinkless.drawinglist/1`, instead
of PDF, PNG, or JPEG bytes. It does not execute document JavaScript or provide
native file, arbitrary network, or Chrome-parity behavior. See
[WASM drawing-list JSON](library-api.md#wasm-drawing-list-json).

---

## Deferred inventory

| Item | Status / reason | Next gate |
|------|-----------------|-----------|
| JavaScript | No JavaScript engine and no CLI. JS-related wkhtmltopdf keys are unknown or land in the settings ignored sink (`web.javascript`); `<script>` is never executed. | Permanent boundary |
| Full CSS / Chrome print parity | Not a product goal for this layout engine based on HTML templates (without any wrappers). | Phase 23 deferred |
| Full flex / grid / subgrid / masonry | **Partial** print CSS subset (fixtures 25/28/32-35). Joint subgrid intrinsic sizing and CSS Grid L3 masonry are out. | - |
| Chrome sticky scroll parity | Print-scoped sticky: page content box is the scrollport; overflow boxes are scrollports at **offset 0**. No continuous scroll, no Chrome pixel match. | Non-goal |
| CJK / complex scripts | Bundled faces plus `@font-face`; Arabic OpenType (GSUB) with presentation-form fallback; Indic **Partial**. `writing-mode` vertical keywords inherit and rotate glyphs (`RotateDeg == -90`); block/line layout stays horizontal. | No CGO HarfBuzz |
| AcroForm / `--enable-forms` | No form model; the PDF writer is gone. The `produce-forms` key is inert. | Not planned |
| XSLT TOC (`--xsl-style-sheet`) | Not implemented; there is no TOC pass in this tree. | Not planned |
| SVG image **output** (`--format svg`) | No image-file output mode; the drawing list carries encoded image payloads. | Not planned |
| BMP output | No image-file output mode. | Not planned |
| SOCKS5 proxy | `parseProxy` accepts `http` / `https` only. | Not planned |
| PDF 1.7 + PDF/A-3a / PDF/UA-1 | **Removed with the PDF writer** in the 0.0.1 renderer cut. The version/profile keys still parse and round-trip; nothing consumes them. | Not in the product |
| PDF 2.0 (ISO 32000-2) | **Removed with the PDF writer** in the 0.0.1 renderer cut. Version keys still parse and round-trip; nothing consumes them. | Not in the product |
| PDF/A-4 / PDF/UA-2 (PDF 2.0 conformance profiles) | **Removed with the PDF writer** in the 0.0.1 renderer cut. Profile keys still parse and round-trip; nothing consumes them. | Not in the product |
| PDF encryption / AcroForm / signatures | Out of scope; there is no writer. | Not planned |
| C ABI (`blinkless_*` c-shared exports) | **Shipped in 0.2.5**: opt-in c-shared exports under `bindings/c` (`-buildmode=c-shared`) power the Python bindings; see [python.md](python.md). Default Go builds stay `CGO_ENABLED=0`; cgo lives only in the isolated shared-library target. | Python bindings track, `plans/0.2.5/` |
| Browser WASM preview | **Shipped in 0.2.6**: opt-in `make wasm` build for inline HTML, returning a versioned drawing-list JSON document (schema `blinkless.drawinglist/1`, documented under [WASM drawing-list JSON](library-api.md#wasm-drawing-list-json)). The adapter runs in a worker with bounded input, output, image dimensions, and request lifetime; obsolete PDF/PNG/JPEG output modes and writer fields are rejected. | Future resource bridge with explicit origin, CORS, size, timeout, and cancellation rules |
| `--read-args-from-stdin` | **Not implemented.** No CLI; the `readargsfromstdin` key is inert. | Not planned |
| Stdin HTML input (`-`) | **Not implemented.** No CLI. `load.GuessURL("-")` falls through to **`http://-`**. Library callers should pass inline HTML. | Document honestly; not a hidden feature |
| WOFF2 / `data:` `@font-face` | Skipped (WOFF2 needs Brotli, not allowlisted; `data:` src rejected). Local TTF/OTF/WOFF1 under ACL works. | No Brotli module |
| `[subject]` placeholder | No writer and no metadata pass; expands **empty**. | Not planned |
| HTML header / footer | No header/footer engine in this tree; the settings keys are inert. | Not planned |
| `:hover` / `:focus` / `:active` | Parsed onto the compound; `matchPseudo` **never matches** (print has no pointer/focus). | - |
| `table-layout: fixed` | **Partial** (fixed lite): consumed when `fixed` and table width is definite (`layout_tables.go:45`). Content max-content ignored. | Matrix §2.5 |
| `@page size` | Parsed into the sheet; the writer-era `applyCSSPageMargins` consumer is gone, so nothing applies it. | Not in the product |
| `background-image` / gradients | **Implemented** multi-layer background images with pure-Go linear and radial gradient rasterization (`background_image.go`, `gradient.go`). | Phase 52 [x] |
| Overflow clip | **Partial**: `hidden`/`clip`/`auto`/`scroll` clip descendant paint to the padding box (`overflow_clip.go`). Sticky still uses overflow as scrollport at offset 0. | Phase 52 |
| Leftover print CSS | Active ledger: [`plans/0.2.6/48-canonical-0.2.6-css-coverage.md`](../plans/0.2.6/48-canonical-0.2.6-css-coverage.md). Do not treat `plans/0.2.0/phases/pending-phase-items/` as the live CSS list. | 0.2.6 leftovers: float wrap, duplex size, GCPM |
| `box-shadow` | **Partial** un-inset offset fill plus lite stacked-rect blur (`box_shadow.go`). Inset and spread ignored. | 52.5 `[x]` |
| `list-style-image` | Paints via the img fetch path; type marker fallback. | 53.4 `[x]` |
| `@page :first` / `:left` / `:right` | Parsed into the sheet; no page-rule consumer in this tree. | 54.1.2 `[x]` (parser only) |
| `page: ident` | Parsed onto the box; no page-rule consumer in this tree. | 54.1.3 `[x]` lite (parser only) |
| `@page` margin boxes (`@top-center`) | Unnamed quoted `@top-*` / `@bottom-*` are parsed into the sheet; no layout consumer in this tree. `running()` / corners out. | 54.3 `[x]` lite |
| Draft corner-shape CSS (34 properties: corner, corner-block-*, corner-inline-*, etc.) | Not implemented - Draft corner-shape CSS; not in the print-output engine. Left unsupported. | Deferred indefinitely (permanent non-goal) |
| Ruby/MathML/rhythmic niche (33 properties: block-ellipsis, block-step-*, box-snap, ruby-*, math-*, etc.) | Not implemented - Ruby/MathML/rhythmic niche; not implemented for print output. Left unsupported. | Deferred indefinitely (permanent non-goal) |
| Draft gap/row-rule decorations (27 properties: row-rule*, rule*) | Not implemented - Draft gap/row-rule decorations; no print consumer. Left unsupported. | Deferred indefinitely (permanent non-goal) |
| SVG presentation properties (53 properties: alignment-baseline, baseline-shift, color-interpolation, cx, cy, d, dominant-baseline, fill-*, marker-*, stroke-*, text-anchor, vector-effect, etc.) | Not implemented - SVG-as-`<img>` is implemented via `internal/svg` rasterizer; 5 CSS properties (`fill`, `stroke`, `stroke-width`, `fill-opacity`, `stroke-opacity`) parsed in style; remaining 53 SVG presentation properties are unsupported. | Phase 83 hard defer |
| Mask, clip, and filter effects (25 properties: clip-path, mask, mask-*, backdrop-filter, SVG filter primitives) | Not implemented - `overflow-clip` is implemented for descendant box clipping (`overflow_clip.go`); 2D image filter (opacity, blur, grayscale, invert, adjustments) on raster images + CSS `opacity()` on elements; `clip-path`, CSS `mask-*` properties, `backdrop-filter`, and CSS shader/SVG filter composition are unsupported. | Phase 83 hard defer |
| CSS Regions and Exclusions (9 properties: flow-from, flow-into, flow-tolerance, region-fragment, wrap-after, wrap-before, wrap-flow, wrap-inside, wrap-through) | Not implemented - CSS Regions and Exclusions are permanent non-goals for print output. | Permanent non-goal |
| Animation, transition, and view-transition properties (45 properties: animation, animation-*, transition, transition-*, trigger-*, view-transition-*) | Not implemented - Static print output has no animation time loop, transition timeline, trigger activation, or view-transition engine. Left unsupported. | Phase 84 print-noop (permanent non-goal) |
| Scroll snap, overscroll, and scrollbar properties (41 properties: overscroll-behavior-*, scroll-behavior, scroll-margin-*, scroll-padding-*, scroll-snap-*, scroll-timeline-*, scrollbar-*) | Not implemented - Static print output has no interactive scroll viewport, scroll snapping, scroll margin/padding offsets, scroll timeline drivers, or scrollbar chrome. Left unsupported. | Phase 84 print-noop (permanent non-goal) |
| Pointer, caret, and form UI properties (25 properties: appearance, caret-*, cursor, field-sizing, nav-*, pointer-events, resize, touch-action, user-select, etc.) | Not implemented - Static print output has no mouse pointer, cursor styling, text caret, focus navigation, dynamic form resizing, touch gestures, or OS window dragging. Left unsupported. | Phase 84 print-noop (permanent non-goal) |
| Anchor positioning, motion path, and view timeline properties (21 properties: anchor-*, offset-*, position-anchor, position-try-*, view-timeline-*, will-change) | Not implemented - Anchor positioning, motion path offset animations, view timelines, scroll anchoring, and paint invalidation hints have no print consumer. Left unsupported. | Phase 84 print-noop (permanent non-goal) |
| Speech and aural CSS properties (19 properties: cue-*, pause-*, rest-*, speak, speak-as, voice-*) | Not implemented - Visual print output has no aural speech synthesis, sound cues, pauses, rests, or speech voice properties. Left unsupported. | Phase 84 print-noop (permanent non-goal) |
| 3D transform and perspective properties (4 properties: backface-visibility, perspective, perspective-origin, transform-style) | Not implemented - Static 2D affine transforms (`transform`, `transform-origin`) are Implemented for paint CTM; 3D transform matrices, perspective projection, and backface culling are permanent non-goals for print output. | Phase 84 print-noop (permanent non-goal) |
| Vendor aliases - Implemented slice A (6) | **Implemented** via `normalizeVendorPrefix` `internal/layout/style_cascade.go:913` plus `display` value remaps at `internal/layout/style_properties.go:81` and value remaps at `internal/layout/style_cascade.go:1000` (`box-align`/`pack`/`orient`/`ordinal-group`). 6 new + 22 from Phase 69 = **28 total Implemented** (`TestWebkitPrefixAliases` `internal/layout/style_cascade_test.go:188`). List: `-webkit-box-align` -> `align-items`, `-webkit-box-flex` -> `flex-grow`, `-webkit-box-ordinal-group` -> `order` (N-1), `-webkit-box-orient` -> `flex-direction` (horizontal/vertical), `-webkit-box-pack` -> `justify-content` (justify->space-between), `-webkit-text-fill-color` -> `color`; plus `display: -webkit-box` -> `flex` and `-webkit-inline-box` -> `inline-flex`. | Phase 82 slice A [x] |
| Vendor aliases - Unsupported group B (3) | **Not implemented** - unprefixed `background-clip` / `background-origin` / `background-size` are Implemented; the `-webkit-background-clip` / `-origin` / `-size` aliases still do not remap. | matrix 5.3 |
| v0.2.7 next-72 leftovers (19) | **Not implemented** by choice. Borders-4 / Round Display drafts (14: `border-clip` family, `border-limit`, `border-shape`, `border-boundary`). Plus `text-fit` (store-only, no scale-search), `shape-inside`, `shape-padding`, `shape-image-threshold`, `float-defer`. Catalog stays Unsupported. | 87.4 / 87.6 / 87.7; matrix §2.3, §5.5 |
| Vendor aliases - Unsupported group C (14) | **Not implemented** - base hard-deferred. `-webkit-mask`, `-webkit-mask-image`, `-webkit-mask-size`, `-webkit-mask-repeat`, `-webkit-mask-position`, `-webkit-mask-origin`, `-webkit-mask-clip`, `-webkit-mask-composite`, `-webkit-mask-box-image`, `-webkit-mask-box-image-source`, `-webkit-mask-box-image-slice`, `-webkit-mask-box-image-width`, `-webkit-mask-box-image-outset`, `-webkit-mask-box-image-repeat` wait on `mask`/`mask-border` bases (Phase 83 hard defer). Left unsupported. | Phase 83 hard defer |
| Vendor aliases - Unsupported group D (20) | **Not implemented** - print-noop bases with no print consumer. `-webkit-animation` (9: animation, delay, direction, duration, fill-mode, iteration-count, name, play-state, timing-function), `-webkit-transition` (5: transition, delay, duration, property, timing-function), `-webkit-backface-visibility`, `-webkit-perspective`, `-webkit-perspective-origin`, `-webkit-transform-style`, `-webkit-appearance`, `-webkit-user-select`. Bases stay skipped for print, so aliases stay unsupported. | Permanent non-goal (print-noop) |
| Vendor aliases - Unsupported group E (5) | **Not implemented** - WebKit-native with no print consumer. `-webkit-line-clamp`, `-webkit-text-size-adjust`, `-webkit-text-stroke`, `-webkit-text-stroke-color`, `-webkit-text-stroke-width`. Left unsupported unless a base plus consumer lands. | No consumer planned |

---

## How to use this list

- A row marked **Phase 22** or **Phase 23** is a ledger item, not a shipped
  date. Phase 23 (open-web / browser competition) stays deferred unless the
  roadmap is amended.
- **Not planned** means there is no active design; a future amendment would
  be required.
- **Partial** means a report-shaped subset exists; Chrome/layout-test
  parity is still out.

When behavior claims change, update this table together with
[fidelity.md](fidelity.md) and [compatibility-matrix.md](compatibility-matrix.md).
