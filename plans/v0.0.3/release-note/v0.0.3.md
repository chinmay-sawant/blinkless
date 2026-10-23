# blinkless 0.0.3

0.0.3 is a feature release. It sits on the 0.0.2 drawing-list engine.
HTML and CSS still go in. A drawing list still comes out. This tree does
not write a PDF, does not encode a page as PNG or JPEG, and does not ship
a `blinkless` command.

`v0.0.2` is already published at `23cc975`. This note is the work after
that tag, through `a0f695b`. The commit list is
[commits.md](commits.md). The controls are the closed ledgers
`plans/v0.0.3/animation`, `transitions`, `corner-shape`, `row-rules`,
`anchors`, `regions`, `mask`, `mask-border`, and `3d-perspective`, plus the
substantially closed `shapes` ledger. `scroll`, `speech`, and
`svg-presentation` stay open and are not claimed here. Pending work you
have not described is not in this note.

## What changed

New layout behaviors land as drawing-list content, not as a new output
format. The public shape is still `layout.DisplayList`.

- Element patches (`layout/relayout_element.go`, `RelayoutElement`). `layout.Options.RetainState` keeps the resolved styles and box tree a patch needs. `RelayoutElement` re-places one block subtree inside the retained page and returns the whole patched display. `html.Document.SetAttribute` and `html.Document.SetText` edit the tree in place (`internal/html/attribute.go`).
- Faster cascade and grids, same operations. The cascade indexes selectors by their rightmost compound (`internal/layout/style_rule_index.go`). Single-column and multi-column auto-row grids build items in one pass (`internal/layout/grid_single.go`, `internal/layout/grid_auto_row.go`), and inline splitting grows geometrically (`6b29cc9`, `81910e0`, `3fa0a49`, `e0c3c9c`).
- Corner shapes (`layout/displaylist.go`, `DisplayOpPath`). `border-radius` corners accept the `corner-shape` family and paint through a path operation. Non-round shapes emit a solid path; round corners keep radii (`4896ac9`, with fixes `bc917a2` and `20606e1`).
- Anchor positioning (`internal/layout/anchor_position.go`). Absolute and fixed boxes place against a named anchor, with `position-area`, `position-try`, fallbacks, and `position-visibility`. Break state is exported as `Display.ForcedBreaks` and `KeepTogether` (`dbc98fa`, `bb06e35`).
- 3D aliases, gradient mask sources, and row rules (`internal/layout/transform_3d.go`, `internal/layout/mask_source.go`, `internal/layout/gap_rules.go`). The `-webkit-` 3D aliases, gradient and `image-set()` mask sources, and `row-rule*` / `rule*` gap lines are implemented. `FinalizeDisplayOps` bakes perspective into geometry: rects become paths with tight bounds (`c13a036`, with fixes `9882877`, `7cc73fc`, and `2d1a7c8`). Junction-aware gap rules and mask parity followed in `91dc0a0`.
- Mask borders (`internal/layout/mask_border.go`). The `mask-border` descriptor, shorthand, and WebKit aliases compose a nine-slice frame raster on the element group, and the WASM drawing list carries the frame data (`ac4b438`, `06de259`, `361e5e1`).
- Interior shapes (`internal/layout/shape_inside.go`). `shape-inside` and `shape-padding` wrap content inside a shape (`b66ed39`).
- Named flows and regions (`internal/layout/named_flow.go`). Named flows, masonry tolerance, exclusions, and wrap breaks run on the single drawing list (`e6abe07`, with fix `26f54ec`).
- Animation and transition metadata (`layout/displaylist.go`, `Display.Keyframes`). Accepted animation and transition values, keyframes, and aligned timing records are exposed on the public element boxes and the display list, with per-operation owners in `Display.OpOwners`. Blinkless does not advance time or interpolate values (`b26de13`, `6d0dbce`).
- Catalog growth. The catalog moved from 389 implemented rows at `v0.0.2` to 555 implemented, 3 partial, 219 unsupported, and 8 intentionally ignored across 785 rows. The catalog is dated 2026-10-20 and the upstream pin (`webref-ed-css 1f2ec8f`, 821 properties) is unchanged.
- Frontend site. The `frontend/` source and the built `docs/` site were re-added and expanded, with generated catalog and specimen data (`04c167a`, `eba3139`, `1c68b28`).
- Hardening fixes in the same window: root canvas background (`320dc96`), table rows and headings kept whole (`fbde5d6`), print width fit (`7a948be`), grid percentage widths (`16d26a3`), and `object-fit` clipping (`b3d9123`).

GitHub Actions on a branch and on a `v*` tag runs `make test` and `make lint` only. The tag is not compared to a version file.

## Removed

The C binding is gone: `bindings/c`, `make c-shared`, and `BINDINGS_VERSION`. The engine is consumed through the Go packages and the WASM binding. This removal is committed on the current branch (`36cc384`) and has not shipped in a tag before this one.

## Version strings

These stamps are not one number.

| Stamp | Value | Where |
|-------|-------|-------|
| This release | 0.0.3 | `RELEASE.md`, GitHub tag `v0.0.3` |
| Previous release | 0.0.2 | tag `v0.0.2` at `23cc975` |
| Catalog date | 2026-10-20 | `testdata/css/catalog/properties.json` |
| WASM build stamp | 0.0.1 | `Makefile` `WASM_VERSION` |

The WASM stamp was not moved in this release. There is no `VERSION` file and no C header anymore.

## What still does not ship

PDF files, PDF profiles, the `blinkless` command, veraPDF, and a golden PDF corpus. One image operation is re-encoded as PNG when orientation or a clip requires it. That is not a picture of the page.

Animation and transition records carry no clock. Masks and mask borders carry values for the host to composite; there is no in-engine pixel replay. Scroll, speech, and SVG presentation rows stay unsupported except where their ledgers say otherwise. No document script runs, and that is not a proof that every bug is gone.
