# Calibration — blinkless

Load this file only when the module path is `blinkless`. It is
context for the three lenses, not a second finding list. Do not re-file
closed ledger rows unless current source has regressed.

Authoritative prose lives in `documentation/architecture/` and
`CONTRIBUTING.md`. This file is the short map.

## Product ceiling

Layout engine for HTML and CSS. No JS, no CGO, no browser, no PDF writer,
and no page rasterizer. Direct modules ⊆ `go-text/typesetting` +
`tdewolff/canvas` (`internal/fonts.TestDirectModuleAllowlist`). Flex/grid/
position are report features, not CSS completeness.

## Engine seam

| Stage | Job | Entry |
|---|---|---|
| Parse | HTML tree | `html.Parse` |
| Style | Cascade and used values | `css.Apply` |
| Layout | Drawing list | `layout.DisplayList` / `DisplayListOptions` |
| JSON | Browser contract | `bindings/wasm` (`serializeDisplay`) |

`internal/convert/prepare` owns the load, parse, stylesheet, and
font-resource phase; `internal/convert/render` owns the stage lifecycle
(`Pipeline`). `internal/pubstate` carries the styled document between the
public packages.

## DAG (do not invert)

- `layout` (public) imports `css`, `internal/fonts`, `internal/layout`, and
  `internal/pubstate`.
- `internal/layout` takes a styled document; it does not import
  `internal/settings`.
- Untrusted bytes enter only via `internal/load` (`Loader.Load`,
  `ResourceContext.Fetch`).
- `internal/svg` is the only rasterization path (SVG-as-image via
  `tdewolff/canvas`); it is not a page rasterizer.

## Contracts to preserve

- Nil context is `errs.ErrNilContext`. Alias it; do not `errors.New` a
  second value with the same meaning.
- `render.Run` checks `ctx.Err()` between stages.
- Layout is single-goroutine, zero locks (`internal/layout`).
- Image operations keep their encoded bytes. One operation is re-encoded as
  PNG only when orientation or a clip cannot stay in those bytes; the page
  itself is still the drawing list.
- The dependency allowlist is enforced by
  `internal/fonts.TestDirectModuleAllowlist`.

## Extension tables (owner of the next row)

| Change class | Owner | Must also touch |
|---|---|---|
| CSS property | `internal/layout` used values (`style.go`, `style_properties.go`, `style_cascade.go`) | optional `css/values.go`; consumer in layout/paint; compatibility matrix row |
| Selector / pseudo | `internal/css` parse **and** match together | unknown pseudos stay on the compound and never match |
| HTML element | `html` allowlist + default styles | matrix §1 if it paints |
| Display-list op kind | `internal/layout` emit + `layout/displaylist.go` public kind | paint order, WASM JSON projection, and a fixture test |
| Formatting context / pagination | `internal/layout` (flow and pagination files) | the shared paint policy and `PaintOrder` |
| Load / ACL | `internal/load` | THREAT-MODEL; deny stays default |
| Settings | `internal/settings` | only fields the pipeline actually consumes |

## Proof that is law here

- Non-doc change: `make lint` + `make test` before any `[x]`.
- Layout/paint: also `go test ./internal/layout/ -count=1` and
  `make golden` (public drawing-list tests in `./layout`).
- New golden fixture needs a focused test in the owning package; `make golden`
  must stay green.
- Security is a load unit (allow **and** deny), never a fixture file.
- `make claim-scan` forbids "stdlib-only", "zero third-party", and
  "byte-identical PDF" claims.

## Do not nag (decided)

No plugin system. No `x/net/html` swap. No CGO HarfBuzz. No pixel-diff
merge gate. No `gofumpt`. No mutex on layout. No context-on-every-struct.
No live network in `make test`. No site-specific MediaWiki cascade hacks.
No document-global Y shifts. No PDF writer revival without a policy change.
Closed ledger rows stay closed unless source regressed.
