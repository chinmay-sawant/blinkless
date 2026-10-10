# Fonts, discovery, and Unicode shaping limits

Operator and integrator notes for the bundled faces, opt-in discovery,
`@font-face`, and shaping. Architecture: [architecture.md](architecture.md).
Product claims: [fidelity.md](fidelity.md).

## Bundled faces

Every default conversion can use:

| Family | Faces | Role |
|--------|-------|------|
| **Liberation Sans** | Regular / Bold / Italic / BoldItalic | Default sans; `sans-serif`, Arial, Helvetica, … |
| **Liberation Serif** | Regular / Bold / Italic / BoldItalic | `serif`, Times, Georgia, … |
| **Liberation Mono** | Regular / Bold / Italic / BoldItalic | `monospace`, Courier, Consolas, … |
| **DejaVu Sans** | Regular + Bold | Unicode fallback (`system-ui`, last-resort glyphs) |

Faces live in `internal/fonts/assets` and are loaded by `fonts.LoadDefaultFaces`. Layout reports glyph positions from the same faces and never embeds a font program in a PDF.

## CSS generic / common-name mapping

`fonts.FaceSet.ResolveFamily` (used by layout after the opt-in registry):

| CSS token | Bundled face |
|-----------|----------------|
| `serif`, `georgia`, `times`, `times new roman`, `liberation serif` | Liberation Serif |
| `monospace`, `courier`, `courier new`, `consolas`, `monaco`, `liberation mono` | Liberation Mono |
| `sans-serif`, `arial`, `helvetica`, `tahoma`, `verdana`, `calibri`, `liberation sans` | Liberation Sans |
| `system-ui` | DejaVu Sans |

Named families that are not in this table are **not** rewritten to Liberation.
They resolve as named against the opt-in registry first; if nothing matches,
layout falls through the author’s comma stack and then to Liberation Sans.

Missing glyphs walk: CSS family (registry, then bundled) → Liberation
weight/style → DejaVu Regular/Bold → any opt-in registry face that covers the
codepoint (`FindWithGlyph`, prefers DejaVu/Noto names).

## Opt-in discovery

Discovery is **opt-in** (privacy + startup). The public engine scans nothing
by default; document faces arrive through `@font-face`. The settings model
carries the wkhtmltopdf-compatible font keys for engine callers that build a
font registry directly:

| Key | Effect |
|-----|--------|
| `font-path` | Scan the directory and children to **depth 2** for `.ttf` / `.otf`; repeatable |
| `use-system-fonts` | Also scan common OS font directories (e.g. `/usr/share/fonts`). Skips proprietary Windows/corefont trees |

`fonts.ScanFontDirs` reads `.ttf` and `.otf` only. **CFF / `OTTO` OpenType is
rejected** (TrueType outlines only). A file that fails `ParseTTF` is skipped.

Example (CJK / Hangul) with a remote `@font-face`:

```html
<style>
@font-face {
  font-family: "Hangul";
  src: url("https://fonts.internal.example/NotoSansKR-Regular.otf");
}
</style>
```

`testdata/fonts/NotoSansKR-HangulSubset.ttf` is a **tiny CI subset** for the
fixture-27 smoke, not a full CJK face. Full Noto CJK is not shipped.

## CJK and non-Latin runs

Text operations carry the resolved face and the shaped run; the host
renderer draws from those. Glyphs must exist in a face on the fallback
chain; with no covering face, CJK in Liberation Sans renders as missing
glyphs. Mixed Latin + CJK runs split across faces, with Latin drawn from
bundled Liberation when the CJK face lacks it.

## `@font-face`

`prepare.ResourceContext.MergeFontFaces` registers document faces on the
layout path.

| `src` | Behavior |
|-------|----------|
| `.woff2`, `.eot` | **Skipped** (warning). WOFF2 needs Brotli; not allowlisted |
| `data:` | **Skipped** (warning) |
| `https://` / `http://` TTF, OTF, WOFF1 | **Fetched** via `Fetch` → `load.FetchSub`, under the **same ACL, network policy, timeout, and body cap** as CSS/images |
| Local `url(...ttf\|otf\|woff)` | Denied by the default ACL; needs an allow prefix in the internal settings model |
| WOFF1 | Decompress → `ParseTTF` (TrueType outlines only) |

`font-weight` / `font-style` on `@font-face` are parsed but **ignored at
register time**. The alias is the family name only.

## Honest shaping limits

OpenType shaping uses [`go-text/typesetting`](https://github.com/go-text/typesetting)
when the active face has a **GSUB** table. `ShapeRun` / `ShapeTextFont` run
that shaper; text operations carry the shaped run and the face reference.
**There is no CGO HarfBuzz.**

- **Arabic / Hebrew:** OT joining + ligation (e.g. Lam-Alef) when GSUB is
  present and reverse-cmap covers the glyphs. **Fallback** (no face / no GSUB
  / unmapped glyph): RTL run reverse plus best-effort **presentation-form**
  joining in `ShapeText`. Faces without Presentation Forms **and** without
  usable GSUB reverse-cmap will still look disconnected.
- **Indic and other complex scripts:** **Partial**: OT applies when the face
  and reverse-cmap succeed; production Indic quality is **not** claimed
  (fallback keeps combining marks after the base; no in-tree matra
  reordering).
- **CJK (Han / kana / Hangul)** works when a capable TTF is on the font
  path. `writing-mode: vertical-rl` and `vertical-lr` rotate glyphs by -90
  degrees while the block still flows horizontally
  (`internal/layout/inline_vertical_writing.go`). Full vertical line
  stacking is not implemented. Per-grapheme-cluster vertical metrics and a
  dedicated vertical CJK face are still out.
- **IPA / uncommon Unicode:** when the CSS `font-family` face and Liberation
  lack a glyph, layout falls back to DejaVu (bundled) and then to any
  covering face registered from `@font-face` or the internal font registry.
  Full coverage needs a face that carries it.
- **OpenType `halt` / `palt`:** requested via typesetting `FontFeatures` for
  CJK / East-Asian punctuation runs in `ShapeTextFont`, and via
  `ParseFontFeatureSettings` / `ShapeTextFontWithFeatures` when CSS
  `font-feature-settings` is supplied. `font-feature-settings`,
  `font-kerning`, and `font-variant-caps` are Implemented (subset): the
  tags and keywords reach the shaper
  (`internal/layout/style_font_feature_props.go`).

## Image payloads

An image operation carries its encoded source bytes (PNG, JPEG, or
SVG-as-rasterized-PNG). One payload is re-encoded as a PNG when orientation
or a clip requires it; that is the only bitmap fallback. See
[architecture/10-imageout-svg.md](architecture/10-imageout-svg.md).
