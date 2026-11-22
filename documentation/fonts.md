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
codepoint (`FindWithGlyph`, prefers DejaVu/Noto names) → the automatic system
CJK fallback below.

## Automatic system CJK fallback

The bundled faces carry no Han, kana, or Hangul. So that Japanese, Chinese, and
Korean text does not render as missing-glyph boxes, layout borrows an installed
operating-system face for those characters. It is **on by default** and needs
no configuration; the host OS is detected at run time.

| Host | Files tried, in order |
|------|-----------------------|
| **Windows** | `%WINDIR%\Fonts`: Yu Gothic, Meiryo, MS Gothic, BIZ UDGothic, Microsoft YaHei and JhengHei, SimSun, Malgun Gothic (bold variants for bold text) |
| **macOS** | Hiragino Sans, Hiragino Sans GB, PingFang, Apple SD Gothic Neo, STHeiti, Arial Unicode |
| **Linux, Android** | TrueType families found by name under `/usr/share/fonts`, `/usr/local/share/fonts`, `/system/fonts`, `~/.fonts`, `~/.local/share/fonts`: Droid Sans Fallback, Takao, VL Gothic, IPA, WenQuanYi, Noto Sans CJK, Unifont |
| **WSL** | The Linux list, then `/mnt/c/Windows/Fonts` |

Boundaries:

- It serves only CJK and fullwidth code points (CJK symbols, kana, Han,
  Hangul, compatibility and fullwidth forms, Han extensions). Latin documents
  never read a font file, and emoji and general symbols are not served.
- Each candidate file is read and parsed once, on the first lookup that needs
  it. A miss is cached per code point.
- `.ttc` collections are split into standalone faces (`fonts.SplitTTC`), so a
  shaper that loads one face per byte slice keeps working.
- CFF/`OTTO` faces are skipped like everywhere else. Noto Sans CJK ships as
  CFF on most Linux distributions, so a Linux box needs a TrueType CJK family
  (Droid Sans Fallback, Takao, WenQuanYi, and similar) for coverage.
- Output depends on the fonts installed on the machine. A host with no CJK
  font still shows boxes. Pin or bundle fonts with `@font-face` when a
  document must render the same everywhere.
- Opt out with `layout.SetSystemFontFallback(false)` or the environment
  variable `BLINKLESS_SYSTEM_FONTS=off`.
- macOS and Linux candidate lists are written from platform conventions and
  are not exercised on those hosts by CI. Windows and WSL are validated
  against real fonts.

## Opt-in discovery

Directory discovery is **opt-in** (privacy + startup). The public engine
scans no font directory by default; document faces arrive through
`@font-face`. The only default read is the narrow CJK fallback above. The settings model
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
- **Explicit kerning:** `font-kerning: normal` and an enabled OpenType
  `kern` setting include kerning in horizontal text widths and line fit.
  A `kern` tag in `font-feature-settings` takes precedence. The default
  `font-kerning: auto` keeps the original layout widths. Runs with
  nonzero letter spacing use individual glyph advances.

## Image payloads

An image operation carries its encoded source bytes (PNG, JPEG, or
SVG-as-rasterized-PNG). One payload is re-encoded as a PNG when orientation
or a clip requires it; that is the only bitmap fallback. See
[architecture/10-imageout-svg.md](architecture/10-imageout-svg.md).
