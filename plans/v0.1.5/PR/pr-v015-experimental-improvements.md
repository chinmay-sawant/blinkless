# PR: `v0.1.5` — Paint-Only Restyle, Hover/Animation Stabilization, Explicit Kerning, and Raster Effect Caches

## Summary

Ships the `v0.1.5` performance and correctness wave across layout restyling, hover patching, layout animations, explicit font kerning, raster effect caching, and word-breaking allocations.

## Changes

### 1. Paint-Only Restyle Fast Path, Root Margin Splice Fix, and Slab Op Clone (`e6ce1f4`)
- **Paint-only `background-color` restyle (`internal/layout/paint_restyle.go`, `layout/relayout_elements.go`):** `RelayoutElements` rewrites the solid fill color in place when a restyle changes only `background-color` on plain unclipped block boxes with unchanged DOM and text, skipping the engine build.
- **Root margin escalation (`internal/layout/relayout_subtree.go`, `internal/layout/relayout_escalation.go`):** Compares all four margins against the placement's prior style and escalates to the holder (or falls back to full layout) when a hover patch changes margins.
- **Slab op clone (`internal/layout/clone_ops.go`):** `cloneOps` allocates copied op payloads from one contiguous slab instead of one allocation per op (~40% fewer allocations per patch).

### 2. Grouped Hover Stabilization, Animation Base Style Cache, and Explicit Kerning (`465af59`)
- **Grouped hover splice window (`internal/layout/group_window.go`, `internal/layout/blend_group.go`):** Widens the splice window over existing group start/end markers when a rebuilt element carries its own group pair, preventing op count from growing by 2 on every hover.
- **Layout animation base style cache (`internal/layout/override_base.go`):** Caches the containing block's resolved base styles across consecutive layout-animation ticks when inputs, target, parent style, and retained styles are unchanged, invalidating on DOM edits (`html/attribute.go`, `internal/html/html.go`). Cuts 64-box layout-animation tick allocations by 61% and 256-box allocations by 74%.
- **Explicit OpenType kerning (`internal/fonts/kerning.go`, `internal/layout/text_kerning.go`, `internal/layout/text_measure_cache_desc.go`):** Measures explicit `font-kerning: normal` and `font-feature-settings: "kern"` with OpenType positioning while preserving existing `font-kerning: auto` widths.

### 3. Raster Effect Caches and Zero-Copy ASCII Word Breaking
- **Image filter & clip-path mask caches (`internal/layout/image_effect_cache.go`, `internal/layout/filter.go`, `internal/layout/clip_path.go`):** Bounded process-wide caches keyed by source image SHA-256, relative box geometry `(drawnX - refX, drawnY - refY, drawnW, drawnH, refW, refH)`, and canonical filter/shape signatures, plus a direct-`Pix` fast path for `clip-path: inset(...)`.
  - 64-node `linear-gradient` + `clip-path`: **42.33 ms → 0.96 ms** (44x faster), **35.9 MB → 751 KB** (48x less memory), hover **90.71 ms → 2.04 ms**.
  - 64-node `linear-gradient` + `filter: blur(4px)`: **12.03 ms → 0.77 ms** (15.7x faster), **5.28 MB → 610 KB** (8.7x less memory), hover **26.31 ms → 1.94 ms**.
- **Pointer `ResolvedStyle` and ASCII word breaking (`internal/layout/inline.go`, `internal/layout/inline_break.go`):** `(*engine).breakToken` takes `*ResolvedStyle` (eliminating per-call heap escapes) and `splitTextToWidthASCII` slices ASCII tokens without `[]rune` allocations.
  - 64-node narrow `width` / `max-width`: **1.73 ms → 0.91 ms**, **3.23 MB → 731 KB** (-77% bytes, -44% allocs).

## Verification

- `make test-quick`: **PASS**
- `go test -count=1 ./internal/layout ./layout`: **PASS**
- `make golden`: **PASS**
- `make lint` (`golangci-lint run ./...` + `make size-check`): **PASS**
