# Docs, catalog, CI, and release-state context

Range: annotated tag `v0.1.0` object `5dd4571`, peeled commit `2e2abfc`
(2026-10-30), through `3e3a8b7` (2026-11-10). 39 commits.

## Catalog and stale counts

Current catalog `testdata/css/catalog/properties.json:13-17`: 785 rows,
555 implemented, 4 partial, 218 unsupported, 8 intentionally ignored,
updated 2026-11-02.

`documentation/compatibility-matrix.md:6` and `:46` match. Headings at
`:617` (`Partial (4)`) and `:626` (`Unsupported (218)`) match.

Stale living docs:

- `documentation/fidelity.md:49` says "Measured 2026-10-20: 785 rows, 555
  implemented, 3 partial, 219 unsupported". Correct: 2026-11-02, 4 partial,
  218 unsupported.
- `documentation/fidelity.md:53` says "The three Partial rows left in the
  catalog" and lists three. Correct: four, adding `backdrop-filter`.

Historical counts at `CHANGELOG.md:99-102`, `:125-128`,
`plans/v0.1.0/release-note/v0.1.0.md:70-72`, `:84-85` describe 0.0.3 and
0.1.0. Leave them as history.

## Documentation gaps since v0.1.0

- `css.Document.HoverChangedScopes` (`css/hover_scopes.go:75-83`) has no
  documentation. `documentation/library-api.md` never names it;
  `documentation/architecture/06-css.md:522-528` documents only
  `UsesStatePseudo`. Hosts cannot discover scoped hover relayout.
- `layout.Display.Release` (`layout/displaylist.go:225-236`) is absent from
  `documentation/library-api.md` and from the Display API table at
  `documentation/architecture/02-library-api.md:102-114`. The lifecycle
  steps at `library-api.md:133-149` tell hosts to replace `prev` but never
  to release it, so pooled chunks stay retained.
- `html.Document.DirtyIDs`, `DirtyStructural`, and `ClearDirty`
  (`html/attribute.go:286-313`) are undocumented. `library-api.md:70-77`
  covers only `SetAttribute` and `SetText`.
- `layout.RelayoutElements` (`layout/relayout_elements.go:14-47`) gets one
  sentence at `library-api.md:63-68`. Missing: signature, all-or-nothing
  rule, error mapping, and the rule that `prev` becomes stale.
- Backdrop filter contract is split. Go `DisplayGroup` carries
  `BackdropFilter` (`layout/displaylist.go:106-108`; field at
  `internal/layout/blend_group.go:29-32`), but WASM `drawGroupJSON` has no
  field (`bindings/wasm/drawing_list.go:242-250`) and the documented schema
  is `{id, mode, isolate, parent, mask, maskBorder}` (`library-api.md:360`).
  `compatibility-matrix.md:622` says the display list carries the filter
  list on its group. That is false for the WASM JSON consumer.
- Patch refusal text is stale. `library-api.md:97-103` and `:123-125` say a
  move or resize is refused. PR #35 changed this to escalate size-changing
  edits (`internal/layout/relayout_subtree.go:127`,
  `plans/PR/pr-batched-relayout-and-caches.md:9`).
  `architecture/02-library-api.md:99-100` still says a host "copies the
  patch" into its display.
- `SubtreeBatchResult.Windows` is test-only
  (`plans/PR/pr-batched-relayout-and-caches.md:37`). Do not document it as
  shipped.

## CI and gate state, PRs 31-36

One CI job, `test + lint` (`.github/workflows/ci.yml:10-26`), runs
`make test` and `make lint` only. Release job: `test + lint + GitHub
Release` (`.github/workflows/release.yml:20`).

| PR | Merge | Check | Result | Notes |
|---|---|---|---|---|
| 31 | `f50f76a` | `test + lint` | SUCCESS | No labels or body file. Body claims clean lint and test. |
| 32 | `5dea7a1` | `test + lint` | FAILURE | Merged red. Lint job `41401763398`: 4 golangci findings and stale size-check. Its `make test` and `make lint` boxes were unchecked. |
| 33 | `295776b` | `test + lint` | SUCCESS | Fixed #32: moved `hasBackdropFilter` to `blend_group.go`, renamed `op` to `paintOp`, added blank lines, allowlist 2138 to 2139 (`plans/PR/pr-backdrop-filter-ci.md:7-20`). |
| 34 | `1b88147` | `test + lint` | SUCCESS | Claims make test, lint, golden, claim-scan. |
| 35 | `d805508` | `test + lint` | SUCCESS | Claims go test, race, make test, lint, golden, claim-scan. Catalog not run. |
| 36 | `3e3a8b7` | `test + lint` | SUCCESS | Claims test, lint, golden, claim-scan, build, matrix-check, race, vet, gofmt, wasm. Catalog unchecked and red (`plans/PR/pr-layout-allocation-and-cache-lifetimes.md:94`). |

Committed body files exist only for #33 through #36. #31 and #32 have none
in `v0.1.0..HEAD`.

## Version and stamps

| Item | State |
|---|---|
| `v0.1.0` tag | Annotated object `5dd4571` peels to `2e2abfc`. `plans/v0.1.0/README.md:6` says `b28b7d6`; that is the last content commit, not the tag. |
| CHANGELOG | No `## 0.1.0`. `## Unreleased` is line 10, `## 0.0.3` is line 56. |
| `RELEASE.md:3` | Says "blinkless 0.0.3 is the feature release." |
| `plans/README.md` | Ends at v0.0.3, line 21. No v0.1.0 or v0.1.1. |
| WASM stamp | `Makefile:77` `WASM_VERSION := 0.0.2`. Leave it (`skills/release-note/SKILL.md:54`). |
| Go | `go 1.26`, `toolchain go1.26.4` (`go.mod:3,5`). |
| Direct deps | `go-text/typesetting v0.3.4`, `tdewolff/canvas` pseudo-version (`go.mod:8-9`). Two only. |

The v0.1.0 release PR body says CHANGELOG and RELEASE updates "ride the
release PR" (`plans/v0.1.0/PR/pr-release-0.1.0.md:27-28`), but
`test-quick`, `golden`, and `lint` stayed unchecked (`:54-56`) and the tag
was cut anyway. `catalog-check` was already noted red (`:57`).

## File-size allowlist

`scripts/file-size-allowlist.txt:9-16` records eight files. All match
`wc -l`:

| File | Lines |
|---|---:|
| `internal/html/tree.go` | 2049 |
| `internal/layout/flex.go` | 2184 |
| `internal/layout/inline_paint.go` | 2123 |
| `internal/layout/layout.go` | 2497 |
| `internal/layout/relayout_subtree.go` | 2835 |
| `internal/layout/style_cascade.go` | 2289 |
| `internal/layout/style_value_accept.go` | 2018 |
| `internal/layout/transform.go` | 2098 |

`AGENTS.md:246-250` is wrong: it says one over-limit file (`layout.go` at
2,267) and that `inline_paint.go` dropped under the limit. There are eight,
`layout.go` is 2,497, and `inline_paint.go` is 2,123 and allowlisted. The
allowlist header (`:1-2`) says to keep it in sync with that AGENTS bullet.

## Known pre-existing gate failures

`make catalog-check` (`Makefile:119-120`) exits 1. Recorded output:

```text
css-catalog-map: check failed
  unresolvable OwnFrame case: animation-composition -> examples/css-preview/catalog_animation_composition_test.go lacks 'add'
  unresolvable OwnFrame case: animation-range -> ... lacks 'entry 10px exit 90px'
  unresolvable OwnFrame case: animation-range-center -> ... lacks 'source 50%'
  unresolvable OwnFrame case: animation-range-end -> ... lacks '75%'
  unresolvable OwnFrame case: animation-range-start -> ... lacks '25%'
  unresolvable OwnFrame case: animation-trigger -> ... lacks 'play reset'
  unresolvable OwnFrame case: trigger-scope -> ... lacks '--fire'
  unknown handler function: wrap-inside -> wrapBreakInsideAvoid
```

Two causes:

1. Seven OwnFrame case strings are absent from the sibling checkout. This
   predates v0.1.0 and is recorded at
   `plans/v0.1.0/PR/pr-release-0.1.0.md:57`. It is a sibling-repo mismatch,
   not a Blinkless source bug.
2. `wrapBreakInsideAvoid` existed at v0.1.0 (`wrap_break.go:120-121`) and
   was removed in `8bce5a1` on the PR #35 branch, but the catalog still
   lists it at `properties.json:20476`. This is source drift and is
   fixable.

`make catalog-ownframe-check` inherits the failure (`Makefile:124`).
`make matrix-check` passes today. `RELEASE.md:9-15` omits catalog-check and
matrix-check, so this red gate is not on the written release checklist.

## Files a complete docs sweep must update

1. `CHANGELOG.md`: add `## 0.1.1 (2026-11-10)` and the missing
   `## 0.1.0 (2026-10-30)` section. Move the current Unreleased bullets
   into the 0.1.1 section.
2. `RELEASE.md:3`: 0.0.3 to 0.1.1.
3. `plans/README.md`: add v0.1.0 and v0.1.1 entries after line 21.
4. `plans/v0.1.0/README.md:6`: change `b28b7d6` to `2e2abfc`; note the tag
   object `5dd4571`.
5. New `plans/v0.1.1/README.md`, `release-note/v0.1.1.md`,
   `release-note/commits.md`, `PR/release-v0.1.1.md`, and
   `PR/pr-release-0.1.1.md`. Use `v0.1.0..HEAD` and
   `bash scripts/pr-diff-stat.sh v0.1.0`.
6. `documentation/fidelity.md:49,53`: 2026-11-02, 4 partial, 218
   unsupported, four Partial rows including `backdrop-filter`.
7. `documentation/library-api.md`: add `RelayoutElements`,
   `Display.Release`, the dirty accessors, and `HoverChangedScopes`;
   correct the refusal text at `:97-103` and `:123-125`.
8. `documentation/architecture/02-library-api.md`: add `Release` to the
   Display table, add the batch API and dirty accessors, fix `:99-100`.
9. `documentation/architecture/06-css.md`: add `HoverChangedScopes` after
   `:528`.
10. `testdata/css/catalog/properties.json:20476`: remove
    `wrapBreakInsideAvoid` from the `wrap-inside` handler list, or point it
    at a live function.
11. `AGENTS.md:246-250`: record eight allowlisted files, `layout.go` at
    2,497, and `inline_paint.go` at 2,123.
12. `README.md:13` and `documentation/overview.md:18`: add a current 0.1.1
    sentence. `output/README.md` has no version claim, so no edit.
13. `documentation/compatibility-matrix.md:622`: either add
    `BackdropFilter` to the WASM JSON and schema, or state that the field
    reaches Go hosts only.

The 0.1.1 release PR applied items 1 through 9 and 12 through 13. Item 10
(the catalog handler) and item 11 (the AGENTS wording) are in the follow-up
backlog. The generated section 2 of the matrix was left byte-identical to
the catalog output so `make matrix-check` stays green.

## Candidate parallel docs tasks

- Agent A: `testdata/css/catalog/properties.json` only. Remove the stale
  handler, rerun `make catalog-check`, report the remaining OwnFrame gap.
- Agent B: `documentation/library-api.md` only. Cover `RelayoutElements`,
  `Release`, the dirty accessors, `HoverChangedScopes`, patch refusal, and
  leave the group schema line to Agent F.
- Agent C: `documentation/architecture/02-library-api.md` and
  `06-css.md`. No file overlap with B.
- Agent D: release promotion. `CHANGELOG.md`, `RELEASE.md`,
  `plans/README.md`, `plans/v0.1.0/README.md`, new `plans/v0.1.1/**`.
- Agent E: `AGENTS.md` and `scripts/file-size-allowlist.txt` wording only.
  Do not change counts.
- Agent F: WASM group field. `bindings/wasm/drawing_list.go`, its tests,
  and the group schema line in `library-api.md` after Agent B finishes.
- Agent G: sibling OwnFrame checkout. Add the seven missing case strings or
  correct the catalog references. This is outside the Blinkless tree.
- Agent H: final gate run after all edits: `make test`, `make lint`,
  `make golden`, `make claim-scan`, `make matrix-check`, `make
  catalog-check`.
