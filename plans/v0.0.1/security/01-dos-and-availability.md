# 0.0.1 - DoS and availability

> **Parent:** `plans/v0.0.1/security/security-checklist.md` (DoS and availability slice)
> **Status:** drafted 2026-10-14 from read-only security sweep; all implementation rows open.
> **Estimated effort:** HTML, CSS, layout, SVG, fonts, and WASM hardening in six phases, gated by package tests, `make golden`, and `make lint`.

## Overview

This ledger is the DoS and availability slice of the 0.0.1 security work. It tracks resource
exhaustion, recursive input structures, unbounded allocation, numeric edge cases, raster
blowup, and host-memory abuse. Each row names one source location or review item and closes
only with a proof command. No row closes from intent.

## Executive summary

| Surface | Risk | Primary code | Existing control |
|---|---|---|---|
| HTML resources and recursion | High | `internal/html` | Partial caps |
| CSS matching and parsing | High | `internal/css` | Depth tests |
| Layout amplification | High | `internal/layout` | Targeted guards |
| SVG and raster work | Medium | `internal/svg` | Partial limits |
| Font resources | Medium | `internal/fonts` | Bounds checks |
| Browser/WASM loading | Medium | `bindings/wasm`, `internal/load` | Caps under review |

## Phase 1: HTML resource and recursion guards

Proof for the phase: package-level `go test ./internal/html` plus any targeted regression that
the changed code demands.

- [ ] Adoption agency algorithm can bypass maxElementDepth (`internal/html/formatting.go:233`). Proof: `go test ./internal/html`.
- [ ] Recursive Walk, appendText, and cloneSubtree paths need one shared guard (`internal/html/html.go:93-157`; `internal/html/select.go:76`). Proof: `go test ./internal/html`.
- [ ] Attribute count and name lengths need caps (`internal/html/html.go:601`; `internal/html/tree.go:1387`). Proof: `go test ./internal/html`.
- [ ] Selected-content cloning needs a cap (`internal/html/select.go:14,68-76`). Proof: `go test ./internal/html`.
- [ ] Numeric entity scanning needs a cap (`internal/html/entities.go:162-172`). Proof: `go test ./internal/html`.
- [ ] Parse-time size cap needs an explicit decision (`internal/html/html.go:164`). Proof: `go test ./internal/html`.
- [ ] The reprocess cap of 8 is documented only; convert it to a hard limit and test (`internal/html/tree.go:118-123`). Proof: `go test ./internal/html`.

## Phase 2: CSS matching and parse safety

Proof for the phase: `go test ./internal/css`.

- [ ] Nested `:has` matching needs a budget (`internal/css/has.go:210-314`). Proof: `go test ./internal/css`.
- [ ] Legal deep nesting chains need a documented bound (`internal/css/parse_depth_test.go:12-42`). Proof: `go test ./internal/css`.
- [ ] Oversized CSS input can drive unbounded allocation in the CSS parser (`internal/css`). Proof: `go test ./internal/css`.
- [ ] `calc` can propagate Inf and NaN through styles (`internal/css/math_expr.go:29-166`). Proof: `go test ./internal/css`.
- [ ] CSS math parser needs a depth limit (`internal/css/math_expr.go:29-166`). Proof: `go test ./internal/css`.

## Phase 3: Layout amplification guards

Proof for the phase: targeted `go test ./internal/layout` runs for the touched helper.

- [ ] Nil root entry needs a guard (`internal/layout/layout.go:1119` and related layout entry points). Proof: `go test ./internal/layout`.
- [ ] Table colspan and rowspan need caps (`internal/layout/layout_measure.go:862`; `internal/layout/layout_tables.go:472,499-527`). Proof: `go test ./internal/layout`.
- [ ] Multicol column count needs a cap (`internal/layout/multicol.go:286,327`). Proof: `go test ./internal/layout`.
- [ ] Grid placement iteration needs a cap (`internal/layout/grid_placement.go:313,335`). Proof: `go test ./internal/layout`.
- [ ] Layout recursion needs a depth limit (`internal/layout/layout.go:1509`; `internal/layout/layout_flow.go:207`). Proof: `go test ./internal/layout`.
- [ ] Background tile repetition needs review for an upper bound (`internal/layout/background_image.go:15`). Proof: `go test ./internal/layout`.
- [ ] PNG dimensions are trusted at layout time; validate them (`internal/layout/layout_images.go:613`). Proof: `go test ./internal/layout`.
- [ ] Image decode can allocate memory without bounds (`internal/layout/layout_images.go:385`; `internal/layout/border_image.go:614`). Proof: `go test ./internal/layout`.
- [ ] `alignContent` NaN handling needs review (`internal/layout/flex.go:647`). Proof: `go test ./internal/layout`.

## Phase 4: SVG and raster availability

Proof for the phase: `go test ./internal/svg`.

- [ ] Mask XML recursion needs a depth guard (`internal/svg/mask.go:184,202`). Proof: `go test ./internal/svg`.
- [ ] `emulateTags` scanning needs a bound (`internal/svg/raster.go:377`). Proof: `go test ./internal/svg`.
- [ ] Canvas mutex acquisition needs a deadline (`internal/svg/raster.go:45,100`). Proof: `go test ./internal/svg`.
- [ ] Extreme raster dimensions need finite range checks (`internal/svg/raster.go:121-140`). Proof: `go test ./internal/svg`.
- [ ] Supersampling needs review for memory and time bounds (`internal/svg/raster.go:84`). Proof: `go test ./internal/svg`.

## Phase 5: Font resource blowup

Proof for the phase: `go test ./internal/fonts`.

- [ ] CMap out-of-bounds reads need bounds checks (`fonts.go:364-377`). Proof: `go test ./internal/fonts`.
- [ ] CMap12 expansion can allocate without a limit (`fonts.go:517-541`). Proof: `go test ./internal/fonts`.
- [ ] CMap4 processing can blow up on crafted fonts (`fonts.go:457-497`). Proof: `go test ./internal/fonts`.
- [ ] `glyf` and `loca` tables need bounds checks (`fonts.go:807-837`). Proof: `go test ./internal/fonts`.
- [ ] `maxTables` parsing needs a cap and a failure test (`fonts.go:197-219`). Proof: `go test ./internal/fonts`.
- [ ] Shaping on huge text can exhaust CPU and memory (`shape_gotext.go`). Proof: `go test ./internal/fonts`.

## Phase 6: Browser/WASM DoS

Proof for the phase: `go test ./bindings/wasm ./internal/load`.

- [ ] HTML size cap must be enforced before string copy (`bindings/wasm/main_js_wasm.go:29,43,contract.go:21`). Proof: `go test ./bindings/wasm`.
- [ ] Base64 conversion makes double copies; release one of them (`bindings/wasm/drawing_list.go:536,557`). Proof: `go test ./bindings/wasm`.
- [ ] Metadata sent from load needs caps (`internal/load`). Proof: `go test ./internal/load`.

## Closing gates

- [ ] Closing gates: `make test`, `make lint`, `make golden`.
```<|end|><|start|>tool to=write<|channel|>analysis <|constrain|>xml<|message|><rfl:parameter name="content">
Markdown file plans/v0.0.1/security/01-dos-and-availability.md created.