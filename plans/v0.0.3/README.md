# v0.0.3 CSS coverage ledgers

Planning only. No property behavior changed with these files. Each directory holds one phase checklist. Implementation rows are open. A `[x]` row is a baseline a reader checked in the current code on 2026-10-15. A `[~]` row is a deliberate non-goal with a reason and a next gate.

The engine result is `layout.DisplayList`. It does not paint a page bitmap (`layout/displaylist.go:129-133`). These ledgers keep that boundary. Animation, transitions, scroll, speech, and masks store values for a host. They do not add a clock, a scrollport, a voice, or an in-repo pixel composite.

`documentation/deferred.md` still calls several of these groups permanent non-goals. That prose describes the engine today. These files are the planning home. Each ledger updates the matching deferred sentence only after its own tests pass.

SVG presentation is one ledger. Each unsupported attribute has its own subsection. The accept path, the SVG rasterizer, and the catalog update are shared, so a separate file per attribute would split one contract.

| Ledger | Names | What is already true | What the open rows add |
|---|---:|---|---|
| [Animation](animation/animation-checklist.md) | 32 | No handler. `@keyframes` is not read. | Keyframes and timing on the drawing list. No in-engine clock. |
| [Transitions](transitions/transitions-checklist.md) | 11 | No handler. | Timing for property changes on the drawing list. No in-engine clock. |
| [Scroll](scroll/scroll-checklist.md) | 48 | Five physical `scroll-margin` names are stored (`internal/layout/scroll.go:49`). | Snap, padding, overscroll, scrollbar, and timeline values on the box. No scrollport. |
| [Speech](speech/speech-checklist.md) | 19 | No handler. | Aural values on the element. No synthesizer. |
| [SVG presentation](svg-presentation/svg-presentation-checklist.md) | 61 | 18 names already change SVG raster or text. | 43 unsupported attributes, one subsection each. |
| [Corner shape](corner-shape/corner-shape-checklist.md) | 34 | Elliptical `border-radius` already paints. | The `corner*` draft, after a path op exists on the drawing list. |
| [Shapes](shapes/shapes-checklist.md) | 4 open | `shape-outside` and `shape-margin` already wrap floats. | `shape-inside`, `shape-padding`, `shape-image-threshold`, `float-defer`. |
| [Row rules](row-rules/row-rules-checklist.md) | 27 | `column-rule` width, style, color, and shorthand are the pattern. | `row-rule*` and `rule*` gap lines. |
| [Anchors](anchors/anchors-checklist.md) | 8 | Absolute and fixed boxes already move from `left` and `top`. | Place one box against one named box. |
| [Regions](regions/regions-checklist.md) | 9 | The engine does not paginate. | Named flow between boxes on one canvas. Paged fragmentation stays out. |
| [Mask sources](mask/mask-checklist.md) | 2 holes | The first wave already carries mask groups (`plans/v0.0.1/css-masking/css-masking-checklist.md`). | Gradient and `image-set()` bytes on that group. Pixel replay stays with the host. |
| [Mask border](mask-border/mask-border-checklist.md) | 13 | The `mask` shorthand already resets mask-border (`internal/layout/style_cascade.go:908-914`). | Border-mask data on the group. Slices are not painted here. |
| [3D perspective](3d-perspective/3d-perspective-checklist.md) | 14 | The 10 unprefixed names already project. `rotateY` plus ancestor `perspective` changes the matrix. | Four `-webkit-` aliases, `translate3d`, `scale3d`, and the layout proofs still missing. |

Start at phase 0 of a ledger before writing code for it. Do not treat two ledgers as one change. The closure gate on every ledger is `make test`, `make golden`, `make lint`, `make claim-scan`, `make catalog-check`, and `make matrix-check`. GitHub CI runs only `make test` and `make lint`.
