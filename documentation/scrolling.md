# Viewport scrolling and sticky boxes

`layout.DisplayList` returns normal document placement. A host that scrolls it
can call `layout.StickyView` with its current logical scroll offset and viewport
size. The returned display moves sticky paint and hit boxes together. It does
not run layout again. Keep the original display for the next call.

The units are CSS pixels before host zoom. For a 1.5 scale, divide the host's
pixel scroll offset and window dimensions by 1.5 before calling. The host still
applies its ordinary scroll translation and zoom when painting the result.
`layout.HasViewportSticky` lets ordinary pages keep their existing cache path.

The geometry follows the sticky view rectangle and containing-block constraint
in [CSS Positioned Layout section 3.4](https://www.w3.org/TR/css-position-3/#sticky-pos).
The element starts in normal flow, moves inward when its declared inset reaches
the viewport edge, and stops at its containing block. Scrolling back derives the
original position again. Bottom and right insets also constrain placement.

The contract currently covers ordinary horizontal-writing viewport sticky
boxes, their painted descendants, and ancestor clipping. Fixed descendants keep
their own coordinates. Nested scroll containers, nested sticky elements, and
transformed or fixed ancestors do not receive viewport sticky placement. The
host must not treat those cases as supported nested scrolling. Vertical writing,
3D content, and percentage insets are outside the verified scope.

A view borrows immutable payloads from its source and owns copies of changed
operations and boxes. It does not own the retained layout result. Do not patch,
animate, or reuse a view as the next source. Apply changes to the original,
then derive another view. A host that caches by display pointer must release
its derived-view cache entries after painting them. Sticky movement also changes
which in-flow element a pointer overlaps, so hit priority must follow paint
order instead of document order alone.

The metadata comes from `internal/layout/sticky_placed.go`; viewport math and
copy ownership live in `layout/sticky.go`. `layout/sticky_test.go` checks
activation, release, return, insets, fixed descendants, containing-block padding,
clipping metadata, and rejected scroll spaces with independent expected values.
The engine produces no framebuffer. Host pixel checks remain downstream work.

## Height media queries

Media features accept `height` and `block-size` alongside the existing width
forms. Supported forms are exact `height: 500px`, `min-height`, `max-height`, and
single comparisons such as `height >= 500px` or `500px < height`. Lengths use
px, pt, em, or rem. Unsupported units, percentages, and chained range syntax
remain unmatched. `css.Relayout` reevaluates the gates against viewport height
without reparsing the HTML or stylesheet.

Container queries retain width and inline-size support only. Adding height to
media-query parsing does not make block-axis container queries valid, including
inside negation. The tests in `internal/css/media_test.go` and
`css/relayout_gate_test.go` check both boundaries and viewport changes.
