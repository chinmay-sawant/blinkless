# Commits after v0.1.0

`v0.1.0` is the annotated tag object `5dd4571`, peeled to commit `2e2abfc`.
These commits are on `master` after that tag, in order, through `3e3a8b7`,
the merge of PR #36. Merge commits are included. The release PR adds the
release note and the context files after this range.

| Commit | Subject |
|--------|---------|
| `e3df1db` | perf(layout): pool resolved style arenas, cache gradients, and eliminate flex allocations |
| `76ee989` | perf: wave 2 optimizations - chunked box arena, 1D gradient collapse, png buffer pool, and placed element id buffer |
| `22ffbf0` | perf(layout): Wave 3 allocations slashing and fused box placement |
| `9ba4eb5` | perf(layout): Wave 4 speculative measurement rollback, chrome bypass, and sorting optimizations |
| `d16e629` | fix(lint): resolve all linter warnings, file length limits, and style checks |
| `f50f76a` | Merge pull request #31 from chinmay-sawant/perf/render-cost-suite |
| `75a1274` | feat(layout): carry backdrop-filter on display groups |
| `5dea7a1` | Merge pull request #32 from chinmay-sawant/feat/backdrop-filter |
| `2c7a05d` | fix(layout): resolve backdrop-filter CI lint and size-check failures |
| `295776b` | Merge pull request #33 from chinmay-sawant/fix/backdrop-filter-ci |
| `24a7ab7` | fix(layout): preserve paint context in subtree relayouts |
| `36edc21` | docs(pr): update hover context diff statistics |
| `1b88147` | Merge pull request #34 from chinmay-sawant/fix/hover-subtree-paint-context |
| `40a436f` | perf: share style resolve, index tree edits, and pool layout allocs |
| `ed98110` | perf: retained styles, batched relayout, indexed tree edits, pooled layout allocs |
| `2656c80` | perf(layout): reuse flex overrides and text measures |
| `82be8d6` | perf(html): reuse the tag attribute scratch |
| `e6f18ca` | perf(layout): derive simple flex row heights without a measure build |
| `3f58fbc` | perf(layout): memoize simple flex row cross sizes |
| `5824ee0` | perf(layout): patch paint-only batches in place |
| `6a2f0b2` | perf(layout): pool patch style stores, intern hover memo strings |
| `bd419c6` | layout: escalate size-changing splices, retarget table-internal patches |
| `30f64f7` | test(layout): cover area chart scale damage escalation |
| `e45a24d` | perf(css,layout): bound global caches that pin old trees, reuse style chunks |
| `08a18a7` | perf(layout): keep hidden atomic inline boxes for patchable subtrees |
| `ee07764` | perf(css,layout): fast-path unchanged hover scopes, bound memo cache, and pool style chunks |
| `9ab93bc` | perf(css,layout): skip non-endpoint hover selectors, pool flex buffers, and bound style stores |
| `7ef441e` | perf(layout,html): add Display.Release, pool patch buffers, preallocate rule indexes, and cut HTML node allocs |
| `dc1d518` | layout, html: intern common tag and attribute names, reduce idle style chunk pooling bounds |
| `cc367e7` | docs(pr): add PR body for batched relayout and caches |
| `8bce5a1` | fix(ci): clear the lint and size gates |
| `2487261` | docs(pr): record the lint and size-gate repair |
| `d805508` | Merge pull request #35 from chinmay-sawant/fix/hover-subtree-paint-context |
| `780905b` | fix(layout): stop caches pinning old trees and borrow flex styles |
| `cb96a04` | perf(layout): restore 32 style chunks within the 4 MiB test bound |
| `8e88f3c` | test(layout): prove the fingerprint cache releases results |
| `7772b2f` | docs(layout): describe cache ownership and style pool bounds |
| `7bbdaa4` | docs(pr): add PR body for layout allocation and cache lifetimes |
| `3e3a8b7` | Merge pull request #36 from chinmay-sawant/fix/layout-allocation-and-cache-lifetimes |
