# Pinned upstream inventory: webref ed/css

This directory holds the frozen upstream CSS property inventory that the
engine catalog in `../properties.json` is measured against. It is evidence
for checks and reviews, not a runtime input.

## Pin

- Repository: https://github.com/w3c/webref
- Path: `ed/css`
- Revision: `1f2ec8f74a80c14066b4c7d6822cee59f69fa03b` (2026-10-08)
- Retrieved: 2026-10-09 with `curl` against `raw.githubusercontent.com` at
  that revision
- Consolidated: 125 per-spec JSON files merged by property name into
  `webref-ed-css-1f2ec8f.json`
- Totals: 821 unique properties

The consolidated file keeps property names, declaring spec file names,
spec links, and value syntax. Prose descriptions are dropped. `specs` lists
every declaring file; `href` and `value` keep the first non-empty entry in
file-name order.

## Provenance classes

Every property carries exactly one class, so draft, vendor, SVG, and
browser-UI properties never collapse into one unexplained denominator:

| Class | Rule | Count |
|---|---|---|
| `vendor` | Name carries a vendor prefix (`-webkit-`, `-moz-`, `-ms-`, `-o-`) | 70 |
| `svg` | Declared only by `SVG.json`, `fill-stroke.json`, or `svg-strokes.json` | 52 |
| `browser-ui` | Declared by `css-ui.json`, `css-forms.json`, or `compat.json` (with SVG/CSS2 cross-references), and no other editor draft owns it | 28 |
| `draft` | Every other W3C CSS-family editor draft property | 671 |

Class precedence when rules overlap: `vendor`, then `svg`, then
`browser-ui`, then `draft`.

## License

webref is MIT licensed, Copyright (c) 2020 World Wide Web Consortium. The
license text is copied verbatim in `LICENSE-webref.txt` and the copyright
notice travels with the consolidated file.

## Refresh

To move the pin to a new webref revision:

1. Fetch every `ed/css/*.json` file at the new revision and rebuild the
   consolidated file (same fields and class rules).
2. Replace `webref-ed-css-<short-revision>.json` and update the pin fields.
3. Update `../properties.json` in the same change: `upstream_class` values
   and the `summary.upstream` block must match the new pin, and every new
   upstream property needs a row or alias. `make catalog-check` fails until
   the two agree, so drift cannot land silently. The check discovers the pin
   by the `webref-ed-css-*.json` glob, so keep exactly one such file.
