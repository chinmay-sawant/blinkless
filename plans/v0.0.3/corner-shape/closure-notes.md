# Corner shape closure notes - 2026-10-17

Branch: `feature/v0.0.3-corner-shapes`.

## What shipped

All 34 `corner*` rows in `testdata/css/catalog/properties.json` moved
`unsupported` -> `implemented`:

- 8 single-corner shape longhands: `corner-top-left-shape`,
  `corner-top-right-shape`, `corner-bottom-right-shape`,
  `corner-bottom-left-shape`, `corner-start-start-shape`,
  `corner-start-end-shape`, `corner-end-start-shape`, `corner-end-end-shape`
- 8 pair shape longhands: `corner-top-shape`, `corner-bottom-shape`,
  `corner-left-shape`, `corner-right-shape`, `corner-block-start-shape`,
  `corner-block-end-shape`, `corner-inline-start-shape`,
  `corner-inline-end-shape`
- 1 four-side shorthand: `corner-shape`
- 17 radius-plus-shape shorthands: `corner`, the four physical sides, the
  four logical sides, the four physical single corners, the four logical
  single corners

The intermediate pass recorded these as `partial`; the owner direction was
`implemented`, so they were promoted in the same day. The wave limits stay
recorded in each row's `limitations` field (they are real, just not
status-bearing):

- a non-round corner with a dashed or dotted border paints a solid path
  (dash segments do not follow the curve);
- border sides that differ in width or color keep the rounded stroke-rect
  path;
- `background-clip` and overflow clip keep their existing boxes instead of
  following the scooped outer path;
- `box-shadow` keeps today's radii;
- logical names use the fixed horizontal left-to-right map (no
  vertical writing-mode table).

Each row cites its handler (`applyCornerShapeLonghand`,
`ApplyCornerShapePairProp`, `applyCornerShapeShorthand`,
`applyCornerSideShorthand`, or `applyCornerSingleShorthand`), its engine
source files, and at least one behavior test that actually drives the name
(`css_behavior_corner_shape_a_test.go`, `css_behavior_corner_shape_b_test.go`,
and `bindings/wasm/corner_shape_test.go` for the WASM half).

## What did not move

- `border-radius` rows: untouched.
- `documentation/deferred.md`: untouched (the ledger stays the planning
  home, per the checklist).
- `internal/layout/layout.go`: net one edited line (the `censusOps` case
  list gained `OpPath` on an existing line), no growth.

## Chrome 155 verification and the corrected corner-shape expansion

The `corner-shape` 1-4 value expansion was re-derived against Chrome
155.0.8059.84 on 2026-10-17 (`getComputedStyle` on live documents):

- four values shape top-left, top-right, bottom-right, bottom-left in that
  order (`corner-shape: scoop bevel notch squircle` -> scoop/bevel/notch/
  squircle), matching the border-radius path order;
- two values set top-left and bottom-right, then top-right and bottom-left;
- three values set top-left, top-right and bottom-left, then bottom-right;
- every keyword and both `superellipse(infinity)` forms are accepted, bare
  `infinity` is not;
- the 16 shape longhands and `corner-shape` are shipped; the 17 `corner`/
  `corner-*` shorthands are in the draft but not implemented in Chrome 155.

The first pass had implemented a side-order tie-break, which resolved the
top-left corner to the left side's value and diverged from Chrome on row 17
of the sibling fixture 67. The engine now expands in border-radius order;
`TestCornerShapeBCornerShapeExpansionMatchesChrome` pins the mapping and the
longhand equivalence.

## Downstream refreshes

- `documentation/compatibility-matrix.md` section 2 regenerated from the
  catalog after the flip.
- `frontend/src/data/catalog.json` regenerated with
  `go run ./scripts/export-frontend-data` (34 corner rows plus the anchor
  rows the site data was missing).
- Sibling `gowkhtmltopdf` gained the blinkless `OpPath` bridge (corner paths
  map to its fill/stroke kinds with path geometry stored local to the op box,
  so pagination shifts carry it), golden fixture 67
  (`testdata/golden/fixture-67-coordinate-shapes.html`) with all 34 examples
  and a per-row Chrome 155 support badge, plus its page envelope. The first
  visual pass of that fixture caught a real trace bug here: the top-left and
  bottom-right corners were emitted in the opposite traversal direction and
  drew a self-crossing chord. The corner trace now emits each corner from its
  start edge point to its end edge point in clockwise order
  (`appendCornerShapePaintPoints` takes a direction flag).
- `fixture-67-blinkless.pdf` (this engine) and `fixture-67-chrome.pdf`
  (Chrome for Testing 155.0.8059.84) are committed beside fixture 66's pair
  in the `gowkhtmltopdf` root; both render 6 pages and row 17 now overlays
  its Chrome reference.

## Gate output on the promoted catalog

- `make catalog-check` -> exit 0 (785 properties, implemented=488,
  partial=3, unsupported=286, intentionally ignored=8)
- `make matrix-check` -> exit 0 (section 2 regenerated from the catalog)
- `make claim-scan` -> exit 0
- `make lint` -> exit 0 (golangci-lint v1.64.8 + size-check)
- `make golden` -> exit 0 (`TestDisplay` in `./layout`)
- `make test` -> exit 0 (full suite, capped concurrency)
- Focused: `go test ./internal/layout -run
  'TestBehaviorBorderTopLeftRadius|TestCornerShape|TestBehaviorBorderStartStartRadius' -count=1`
  -> exit 0; `go test ./bindings/wasm -run 'TestDrawing' -count=1` -> exit 0

## Follow-ups left for a later wave

- dashed/dotted borders through a shaped path (segment clipping);
- per-side colors and widths on a shaped border;
- background-clip and overflow clip following the scooped outer path;
- a writing-mode table shared with the border logical radii (the open `[~]`
  row in the checklist).
