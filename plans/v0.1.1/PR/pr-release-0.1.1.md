# chore(release): promote v0.1.1 and record the post-0.1.0 context

## Summary

- Adds the v0.1.1 release note, the 39-commit ledger, and an agent-ready
  context pack for everything that landed after `v0.1.0`.
- Promotes `CHANGELOG.md` and `RELEASE.md`, backfills the missing `0.1.0`
  changelog section, and updates the version-facing docs.
- Changes no Go source and no public API. This is a documentation and plans
  PR.

---

## Motivation / context

- Plans: `plans/v0.1.1/README.md`,
  `plans/v0.1.1/release-note/v0.1.1.md`,
  `plans/v0.1.1/context/follow-up-backlog.md`.
- Range: `v0.1.0` (`2e2abfc`) through `3e3a8b7`. That is 39 commits, six
  merged pull requests (#31 through #36), 251 files, 30,511 insertions, and
  2,283 deletions.
- The `v0.1.0` tag was cut without the `CHANGELOG.md` section or the
  `RELEASE.md` version bump that its release PR body said would ride the
  release PR. This PR backfills `0.1.0` and promotes `0.1.1`.
- The context pack is written for parallel follow-up agents: each note names
  the files, the proof, and the traps. The backlog holds 45 candidate tasks
  across 8 areas.

---

## Changes

### Release record

- `plans/v0.1.1/release-note/v0.1.1.md`: the release note.
- `plans/v0.1.1/release-note/commits.md`: 39 commits in order, merges
  included.
- `plans/v0.1.1/PR/release-v0.1.1.md`: the GitHub release body.
- `plans/v0.1.1/PR/pr-release-0.1.1.md`: this body.
- `plans/v0.1.1/README.md`: the version index.

### Agent context pack

- `plans/v0.1.1/context/render-cost.md`: PR #31 waves, anchors, claims, and
  the outline regression.
- `plans/v0.1.1/context/patch-engine.md`: PR #34 and #35 API, behavior,
  tests, and dormant code.
- `plans/v0.1.1/context/cache-lifetime.md`: cache ownership, bounds, and
  the html registry gap.
- `plans/v0.1.1/context/docs-and-gates.md`: catalog counts, doc gaps, CI
  state, and version stamps.
- `plans/v0.1.1/context/follow-up-backlog.md`: 45 candidate tasks with
  anchors and close conditions.

### Promotion and docs corrections

- `CHANGELOG.md`: adds `0.1.1 (2026-11-10)` and the missing
  `0.1.0 (2026-10-30)` section. The empty `Unreleased` heading stays.
- `RELEASE.md`: 0.0.3 to 0.1.1; adds `make matrix-check` and records the
  known red `make catalog-check`.
- `plans/README.md`: adds the `v0.1.0` and `v0.1.1` entries.
- `plans/v0.1.0/README.md`: corrects the tag from `b28b7d6` to the peeled
  commit `2e2abfc`.
- `README.md` and `documentation/overview.md`: current release sentence.
- `documentation/fidelity.md`: 2026-11-02 catalog counts, 4 partial, 218
  unsupported, and `backdrop-filter` in the Partial list.
- `documentation/library-api.md`: documents `RelayoutElements`,
  `Display.Release`, the dirty accessors, and `HoverChangedScopes`.
  Corrects the patch refusal and damage text.
- `documentation/architecture/02-library-api.md`: batch API, display
  release, dirty accessors, and the patch display wording.
- `documentation/architecture/06-css.md`: hover scopes.
- `documentation/compatibility-matrix.md`: says the Go display group
  carries `backdrop-filter`, and the WASM JSON does not add it in 0.1.1.

---

## What shipped since v0.1.0

| PR | Merge | Area | Main evidence |
|----|-------|------|---------------|
| #31 | `f50f76a` | Render cost: box and style chunks, gradient cache, flex pools, chrome bypass, paint order, display walk | `internal/layout/layout_chunks.go`, `gradient.go`, `flex_children.go`, `paint_order.go`, `layout/displaylist.go` |
| #32 | `5dea7a1` | `backdrop-filter` metadata on display groups | `internal/layout/blend_group.go`, `layout/displaylist.go`, `layout/backdrop_filter_test.go` |
| #33 | `295776b` | Lint and size repair for #32 | `internal/layout/blend_group.go`, `scripts/file-size-allowlist.txt` |
| #34 | `1b88147` | Paint context in subtree relayout, hover scopes | `internal/layout/relayout_subtree.go`, `css/hover_scopes.go` |
| #35 | `d805508` | Batch relayout, size escalation, table retarget, caches, HTML edit index | `layout/relayout_elements.go`, `internal/layout/relayout_subtree.go`, `html/attribute.go` |
| #36 | `3e3a8b7` | Cache ownership and lifetime | `internal/css/match.go`, `layout/relayout_element.go`, `internal/layout/grid_memo.go` |

### Public API added

```go
func layout.RelayoutElements(ctx context.Context, doc *css.Document, prev *layout.Display,
    elementIDs []string, options layout.Options) (*layout.Display, error)
func (d *layout.Display) Release()
func (d *css.Document) HoverChangedScopes(previous string) ([]string, bool)
func (d *html.Document) DirtyIDs() []string
func (d *html.Document) DirtyStructural() bool
func (d *html.Document) ClearDirty()
```

`layout.DisplayGroup.BackdropFilter` is a new field on the existing Go
type. `internal/html.DirtySet` is new in an internal package.

### Behavior changes hosts must know

| Change | What to do |
|--------|------------|
| A size-changing patch may widen to an ancestor. | Do not assume only the named element moved. Repaint the escalated range, or keep a full-layout fallback. |
| A table cell, row, or inline edit retargets to the owning table. | `ElementPatch.Old` and `New` stay the named element's box, so they may not cover content that widened inside the table. |
| A no-op or inert override returns `prev` with empty payload fields. | Read `patch.Display`; do not read `Ops`, `Boxes`, `Old`, or `New` unconditionally. |
| `RelayoutElements` is all-or-nothing. | Validate ids before the batch, or call per element when partial success is needed. |
| `Display.Release` returns chunks to pools. | Release only when no reader will use the display. Released slices are not zeroed. |
| `HoverChangedScopes` can return `false`. | Lay out the page when it does. |
| The dirty accessors record edits but no layout path consumes them. | Clear the set after adoption. Do not treat it as incremental layout. |
| `backdrop-filter` is metadata only, and the WASM JSON omits it. | Composite in the host. Do not build browser features on a WASM field that does not exist. |

### Known issues

- `outline-style: solid` with no explicit width can paint no outline. The
  chrome early exit tests raw `OutlineWidth`
  (`internal/layout/layout_chrome.go:497`), but CSS gives a visible style the
  `medium` width (`internal/layout/outline.go:38-45`). This is `FIX-01` in
  the backlog.
- `Box.OperationIDs` points into a shared backing array. Treat it as
  read-only.
- The gradient texture cache is capped by entry count, not bytes, and the 1D
  gradient path has no filter parity test.

### Not wired in 0.1.1

- Retained style resolve (`internal/layout/style_cascade.go:2118`,
  `style_retain.go`) has only test callers.
- The block intrinsic memo (`internal/layout/memo_block.go:8`) is designed
  and tested, not wired.
- `rentBox`, `rentClipRect`, `rentBoxRows`, `rentOpIDs`, and `rentPlaced` in
  `internal/layout/box_alloc.go` have no production caller.
- `SubtreeBatchResult.Windows` is produced and tested, but
  `RelayoutElements` discards it.
- The dirty accessors do not drive a cascade or layout path yet.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | No engine change in this PR. The note records the 0.1.1 render-cost and patch-path work. |
| **Memory** | No engine change. The note records the weak caches, the 32-chunk style pool, and the html registry gap. |
| **Behavior / correctness** | No engine change. The note records size escalation, table retargeting, and the outline regression. |
| **API / CLI** | No API change in this PR. The note records the 0.1.1 additions. No CLI exists. |
| **Dependencies** | None. `go.mod` is unchanged since `v0.1.0`. |
| **Binary size / build time** | None from this PR. No binary is produced. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None in this PR | This PR changes docs and plans only. |
| 0.1.1 host changes are listed in the release note | Read the behavior table in `plans/v0.1.1/release-note/v0.1.1.md` before pinning OwnFrame to the tag. |

---

## Test plan

- [x] Read all 39 commit messages and the changed Markdown in the range.
- [x] `bash scripts/pr-diff-stat.sh v0.1.0` for the range table.
- [ ] `make test` (not run; this PR changes no Go source)
- [ ] `make lint` (not run; this PR changes no Go source; CI runs it)
- [ ] `make golden` (not run; no layout, paint, or fixture change)
- [ ] `make claim-scan` (not run; docs only)
- [ ] `make matrix-check` (not run; docs only)

### Commands

```sh
bash scripts/pr-diff-stat.sh v0.1.0
make test
make lint
make golden
make claim-scan
make matrix-check
```

CI runs `make test` and `make lint` on this PR. The local engine gates were
not rerun because the diff changes no Go source. The merged PRs recorded
their own gates in the release note.

---

## Screenshots / sample output

No visual surface changed. No drawing list, fixture, or page raster changed.
The catalog counts and the release text are the only data changes.

---

## Related issues

- None. Open issue #18 (mask-border frame regions) does not overlap this
  release.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`documentation`, `enhancement`)
- [x] Related issues checked; none applies
- [x] Filled body committed under
  `plans/v0.1.1/PR/pr-release-0.1.1.md`

---

## Follow-ups (out of scope)

- Tag `v0.1.1` after merge and publish the GitHub release from
  `plans/v0.1.1/PR/release-v0.1.1.md`.
- Fix the outline early exit (`FIX-01`) before the next release.
- Work the 45 candidate tasks in
  `plans/v0.1.1/context/follow-up-backlog.md`. One agent owns one file set.
- Decide the `make catalog-check` gate after the `wrap-inside` handler and
  the sibling OwnFrame cases are fixed.

---

## Reviewer checklist

- [ ] The release note matches the merged commits and does not claim
      designed-only code as shipped.
- [ ] The `0.1.0` backfill is accurate and does not rewrite history.
- [ ] The catalog counts match `testdata/css/catalog/properties.json`.
- [ ] The new API docs match the current signatures.
- [ ] No engine source or generated artifact is in the diff.
- [ ] The PR has an assignee and labels.
- [ ] The diff-stat-by-extension table is pasted below.

---

## Diff stat by extension

### Range `v0.1.0..3e3a8b7`

| Extension | Files | Insertions | Deletions |
|-----------|------:|-----------:|----------:|
| `.go` | 238 | 29,653 | 2,139 |
| `.json` | 1 | 147 | 133 |
| `.md` | 10 | 688 | 10 |
| `.txt` | 1 | 8 | 1 |
| `.yml` | 1 | 15 | 0 |
| **Total** | **251** | **30,511** | **2,283** |

### This PR

| Extension | Files | Insertions | Deletions |
|-----------|------:|-----------:|----------:|
| `.md` | 21 | 2,441 | 71 |
| **Total** | **21** | **2,441** | **71** |
