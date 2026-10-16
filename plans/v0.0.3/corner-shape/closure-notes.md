# Corner shape closure notes - 2026-10-17

Branch: `feature/v0.0.3-corner-shapes` (uncommitted at handoff; nothing pushed).

## What flipped

All 34 `corner*` rows in `testdata/css/catalog/properties.json` moved
`unsupported` -> `partial` in one pass:

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

`partial`, not `implemented`, because the phase 2 limits stay open and are
named in each row's `limitations`:

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

## Gate output on the flipping change

- `make catalog-check` -> exit 0 (785 properties, implemented=454,
  partial=37, unsupported=286, intentionally ignored=8)
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
  row in the checklist);
- only then can rows move `partial` -> `implemented`.
