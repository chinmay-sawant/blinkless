# Commits after v0.0.2

`v0.0.2` is `23cc975` (2026-10-15). These commits are on `master` after that
tag, in order, through `a0f695b` (2026-10-21). Merge commits are included.

| Commit | Subject |
|--------|---------|
| `04c167a` | chore: add minimal flat product and documentation website in frontend and docs |
| `b1bda5a` | chore: update hero tagline to Go-native alternative to Chromium's Blink |
| `eba3139` | feat(frontend): redesign the site around the drawing list |
| `1198cc4` | docs: add the frontend site PR body |
| `af49f68` | docs: fill the frontend site PR diff table |
| `964ad3f` | Merge pull request #11 from chinmay-sawant/chore/frontend |
| `7f12a5b` | fix(scripts): clear export-frontend-data lint |
| `eb58b50` | docs: add the export lint PR body |
| `d1c5224` | Merge pull request #12 from chinmay-sawant/fix/export-frontend-data-lint |
| `bef88fa` | docs: add the v0.0.3 feature ledgers |
| `58cf6c6` | docs: index the v0.0.3 ledgers |
| `7791edd` | chore: remove knowledge-base references from AGENTS.md |
| `c13a036` | feat: ship 3D aliases, gradient masks, and row rules |
| `2b4f38f` | docs: add the 3D, mask, and row-rules PR body |
| `9882877` | fix: finalize 3d display geometry and mask raster |
| `7cc73fc` | fix: preserve hidden backfaces in print media |
| `2d1a7c8` | Fix Blinkless 3D transforms and backface printing |
| `75f7937` | fix(layout): clear golangci-lint findings in the 3D and mask code |
| `74c4fa5` | docs: refresh the 3D, mask, and row-rules PR body |
| `52e4b67` | Merge pull request #13 from chinmay-sawant/feat/3d-perspective-mask-row-rules |
| `91dc0a0` | feat(layout): paint junction-aware gap rules and complete mask parity |
| `1367e3d` | docs: add the gap-rule mask parity PR body |
| `0f35c64` | Merge pull request #14 from chinmay-sawant/feature/gap-rule-junctions-and-mask-parity |
| `dbc98fa` | Anchor positioning one-canvas subset (v0.0.3 anchors) |
| `bb06e35` | feat(layout): fit-based anchor option selection and break export |
| `1fdfe79` | docs: add the anchors PR body |
| `2729e31` | docs: refresh the anchors PR diff table |
| `36594da` | docs: promote position-visibility to implemented in the catalog |
| `a57eec0` | docs: refresh the anchors PR diff table |
| `9086a6c` | Merge pull request #15 from chinmay-sawant/feature/v0.0.3-anchors |
| `4896ac9` | feat(layout): corner shapes for rounded boxes |
| `bc917a2` | fix(layout): trace corner outlines in polygon order |
| `bc8e3ca` | docs: promote corner shapes to implemented in the catalog |
| `20606e1` | fix(layout): expand corner-shape values in border-radius order |
| `320dc96` | fix(layout): paint the root canvas background across the full canvas |
| `fbde5d6` | fix(layout): keep table rows, headings, and one-line blocks whole |
| `7a948be` | fix(layout): fit print layouts to the canvas width |
| `16d26a3` | fix(layout): resolve grid item percentage widths as auto in track sizing |
| `b3d9123` | fix(layout): crop object-fit images to their content box |
| `2840481` | docs: drop corner shapes from the deferred property census |
| `f0e0395` | docs: add the corner shapes PR body |
| `12a14ca` | Merge pull request #16 from chinmay-sawant/feature/v0.0.3-corner-shapes |
| `b66ed39` | feat(layout): CSS shape-inside and shape-padding storage and line inset |
| `ac4b438` | feat(layout): mask-border descriptor, shorthand, and WebKit aliases |
| `d4d7209` | merge: shapes ledger wave 1 (shape-inside, shape-padding) |
| `3ada97e` | merge: mask-border ledger wave 1 (descriptor, shorthand, aliases) |
| `e6abe07` | feat(layout): named flows, masonry tolerance, exclusions, and wrap breaks |
| `77a6158` | merge: regions ledger wave 1 (named flows, tolerance, exclusions, wrap breaks) |
| `06de259` | feat(wasm): serialize the mask-border group descriptor |
| `09d1ade` | merge: mask-border WASM group serialization (wave 2) |
| `03bc499` | docs: promote mask-border, regions, and shapes ledgers in the catalog |
| `34a2eda` | merge: catalog, matrix, and deferred docs closure (wave 2) |
| `5e952ea` | docs: refresh matrix prose for the mask-border and regions ledgers |
| `b322fb1` | chore: ignore local .worktrees directories |
| `f3778a5` | refactor(layout): clear enable-all lint findings in the wave-1 ledgers |
| `6de0557` | docs: add the mask-border, regions, and shapes PR body |
| `a7facbc` | test(layout): close the audit weak-proof gaps and refresh the ledgers |
| `591d337` | docs: refresh the ledger citations the catalog promotion moved |
| `26f54ec` | fix(layout): move remainder-short avoid groups whole and pin the tolerance default |
| `c9514d1` | docs: correct the PR body counts, masonry default, and breaking-change note |
| `5a7630c` | docs: refresh the PR diff table after the review fixes |
| `361e5e1` | feat(layout): compose the mask-border frame raster and render it |
| `2ee6ec7` | Merge pull request #17 from chinmay-sawant/feature/v0.0.3-mask-border-regions-shapes |
| `b26de13` | feat(css): export animation and transition metadata |
| `36cc384` | chore: remove C bindings and align docs with the 0.0.2 tree |
| `c2395cb` | docs: correct the C binding removal status in the changelog |
| `fe8dc83` | docs(pr): add the animation and transition metadata PR body |
| `55fcf3b` | Merge pull request #19 from chinmay-sawant/feature/animations-transitions |
| `6d0dbce` | feat: record CSS motion runtime evidence in catalog |
| `ca35deb` | docs: sync Blinkless CSS motion catalog frontend |
| `35dd49f` | docs(pr): add CSS motion runtime catalog PR body |
| `caaff1d` | Merge pull request #20 from chinmay-sawant/feat/css-motion-catalog-runtime |
| `81910e0` | perf(layout): index selectors by their rightmost compound |
| `3fa0a49` | perf(layout): lay out single-column grids in one pass |
| `e0c3c9c` | perf(layout): build multi-column auto-row grids in one pass |
| `6b29cc9` | feat(layout): re-lay out one subtree and keep the rest of the page |
| `56bc604` | fix: address review findings on selector index and layout work |
| `fae1977` | docs(pr): add selector index and element patch PR body |
| `c1054ff` | Merge pull request #21 from chinmay-sawant/chore/selector-index |
| `1c68b28` | fix(scripts): keep frontend catalog export in sync |
| `7be2b49` | docs(pr): add export-frontend-data catalog PR body |
| `cd83e3a` | Merge pull request #22 from chinmay-sawant/fix/export-frontend-data-catalog |
| `7ec69a5` | docs: sync engine docs with recent layout and catalog work |
| `71da232` | docs(pr): add docs sync PR body |
| `a0f695b` | Merge pull request #23 from chinmay-sawant/docs/sync-recent-commits |
