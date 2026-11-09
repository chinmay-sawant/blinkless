# Release index

0.1.1 is a performance and patch-path release on the 0.1 line. It follows
0.1.0. The engine range covers `master` from tag `v0.1.0` (`2e2abfc`)
through `3e3a8b7`: 39 commits, six merged pull requests (#31 through #36),
251 files, 30,511 insertions, and 2,283 deletions. The release PR adds
documentation and plans only.

- Release note: [release-note/v0.1.1.md](release-note/v0.1.1.md)
- Commit ledger: [release-note/commits.md](release-note/commits.md)
- GitHub release body: [PR/release-v0.1.1.md](PR/release-v0.1.1.md)
- Release PR body: [PR/pr-release-0.1.1.md](PR/pr-release-0.1.1.md)

## Context pack

These notes are written for parallel follow-up agents. Each one names the
files, the proof, and the traps. Anchors were taken at `3e3a8b7` before the
release edits. Go source line numbers are stable; documentation line numbers
may have moved in the release PR.

- [Render cost](context/render-cost.md)
- [Patch engine](context/patch-engine.md)
- [Cache lifetime](context/cache-lifetime.md)
- [Docs and gates](context/docs-and-gates.md)
- [Follow-up backlog](context/follow-up-backlog.md)

## Workstreams

| PR | Area | Main entry points |
|----|------|-------------------|
| #31 | Render cost, box and style chunks, gradients, flex children, paint order, display build | `internal/layout/layout_chunks.go`, `internal/layout/gradient.go`, `internal/layout/flex_children.go`, `internal/layout/paint_order.go`, `layout/displaylist.go` |
| #32 | `backdrop-filter` metadata on display groups | `internal/layout/style_properties.go`, `internal/layout/blend_group.go`, `layout/displaylist.go` |
| #33 | Lint and size repair for #32 | `internal/layout/blend_group.go`, `scripts/file-size-allowlist.txt` |
| #34 | Paint context in subtree relayout, hover scopes | `internal/layout/relayout_subtree.go`, `css/hover_scopes.go` |
| #35 | Batch relayout, size escalation, table retarget, caches, HTML edit index | `layout/relayout_elements.go`, `internal/layout/relayout_subtree.go`, `internal/layout/style.go`, `html/attribute.go` |
| #36 | Cache ownership and lifetime | `internal/css/match.go`, `layout/relayout_element.go`, `internal/layout/grid_memo.go`, `internal/layout/layout_chunks.go` |

## Verification state

Each merged PR ran the GitHub `test + lint` job. PR #32 merged with that
job red. PR #33 repaired it. PR #34, #35, and #36 merged green. The PR
bodies for #34, #35, and #36 record `make golden` and `make claim-scan` as
well.

`make catalog-check` was already red before this release. Seven animation
and transition sample strings are absent from the sibling OwnFrame
checkout, and the catalog still lists the removed `wrapBreakInsideAvoid`
handler for `wrap-inside`. `make matrix-check` passed at PR #36.

The release PR changes documentation, plans, and catalog-facing prose
only. It changes no Go source. Local engine gates were not rerun for that
docs-only change. CI still runs `make test` and `make lint` on the PR.
