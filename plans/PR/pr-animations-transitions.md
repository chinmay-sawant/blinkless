# feat(css): export animation and transition metadata

## Summary

- Adds host-facing animation and transition metadata to the drawing list. Blinkless still does not advance time, sample frames, or interpolate values.
- Parses `@keyframes` and `@-webkit-keyframes` into stylesheet data and copies the rules to `layout.Display.Keyframes` (`layout/displaylist.go:171-173`).
- Removes the unused C shared-library binding and brings the changelog, catalog, docs, and built site in line with the 0.0.2 tree.

## Motivation / context

- Plans: `plans/v0.0.3/animation/animation-checklist.md`, `plans/v0.0.3/transitions/transitions-checklist.md`
- The engine has no clock and no page rasterizer. Animation and transition values are data a host can sample; they are not runtime behavior (`documentation/deferred.md`, animation and transition row).
- The C binding shipped in 0.0.2 as a rendering stub with no drawing-list export. It was unused and carried a stale version string, so this branch removes it and corrects the docs around it.

## Changes

### Animation metadata (`internal/layout/animation_style.go`, new; `internal/layout/animation_transition.go`, new)

- Accepts and cascades 23 standard names: `animation`, `animation-composition`, `animation-delay`, `animation-delay-start`, `animation-delay-end`, `animation-direction`, `animation-duration`, `animation-fill-mode`, `animation-iteration-count`, `animation-name`, `animation-play-state`, `animation-range`, `animation-range-center`, `animation-range-end`, `animation-range-start`, `animation-timeline`, `animation-timing-function`, `animation-trigger`, `trigger-scope`, `view-transition-class`, `view-transition-group`, `view-transition-name`, and `view-transition-scope`.
- Normalizes nine `-webkit-animation*` aliases onto the standard names. Prefixed and standard declarations keep their source order in the cascade.
- Aligns each comma-list item into one `layout.Animation` record. Times become milliseconds; `duration: auto` and `infinite` iterations become flags; timelines, ranges, and triggers stay CSS strings.
- Keeps `trigger-scope` and the four `view-transition-*` identity values on the matching `layout.Box`.

### Transition metadata (`internal/layout/transition_style.go`, new; `internal/layout/easing_value.go`, new)

- Accepts and cascades six standard names: `transition`, `transition-property`, `transition-duration`, `transition-delay`, `transition-timing-function`, and `transition-behavior`.
- Normalizes five `-webkit-transition*` aliases onto the standard names with the same source-order rule.
- Expands the `transition` shorthand into its five longhands, applies CSS-wide keywords, and rejects a negative duration while retaining a negative delay.
- Parses the accepted easing forms: keywords, `cubic-bezier()`, `steps()`, and `linear()`. Each comma-list item becomes one `layout.Transition` record.

### Keyframes (`internal/css/keyframes.go`, new)

- Parses `@keyframes` and `@-webkit-keyframes` into source-ordered rules with numeric offsets and the declarations written at each offset.
- Handles selector lists (`from`, `to`, and percentage lists) and keeps repeated names in order. A consumer selects the last matching rule, as CSS does.
- The public copy drops `!important` declarations (`layout/displaylist.go:300-303`), matching the CSS rule that important declarations inside keyframes do not apply.

### Public API and WASM

- `layout.Box` gains `Animations`, `Transitions`, `TriggerScope`, and the four view-transition identity fields (`layout/layout.go:10-26`).
- New exported types: `layout.Animation`, `layout.Transition`, `layout.KeyframesRule`, `layout.Keyframe`, and `layout.KeyframeDeclaration` (`layout/layout.go:28-77`).
- `layout.Display` gains `Keyframes` (`layout/displaylist.go:167-173`), filled from every stylesheet in document order by `keyframesFrom` (`layout/displaylist.go:286-319`).
- The WASM drawing list encodes the box metadata and an optional top-level `keyframes` array. The schema stays `blinkless.drawinglist/1`.
- `documentation/library-api.md` documents the field shapes, units, and the runtime boundary.

### C binding removal and tree cleanup

- Deletes `bindings/c`, including the cgo exports, stub, header, and tests.
- Drops `make c-shared`, `print-bindings-version`, `bindings-clean`, and `BINDINGS_VERSION` from the Makefile.
- Removes C binding references from `.golangci.yml`, `.gitignore`, `CONTRIBUTING.md`, `AGENTS.md`, the release-note skill, the threat model, the architecture entry-points note, the compatibility matrix, and the deferred table.
- Keeps `bindings/wasm` and its contract tests. The WASM stamp moves from 0.0.1 to 0.0.2.
- Replaces the inherited 0.1.x through 0.2.x changelog history with entries derived from the actual `v0.0.1` and `v0.0.2` tags and release notes. The Unreleased section records the 64 commits after `v0.0.2`.
- Regenerates the frontend catalog and specimen data and rebuilds `docs/`.
- Updates the compatibility matrix and fidelity counts to 785 rows, 512 implemented, 46 partial, 219 unsupported, and 8 intentionally ignored.

## Impact

| Area | Impact |
|------|--------|
| **Performance** | No change on documents without animation, transition, or keyframe declarations. `@keyframes` blocks are now retained instead of skipped, so a sheet with them allocates rule and declaration slices. The animation and transition handlers run only when the style has those fields set. |
| **Memory** | `ResolvedStyle`, `PlacedElement`, and `layout.Box` gain the new fields. Each record holds short strings and numbers; keyframe declarations keep raw value strings. The C shared-library build and its header are gone. |
| **Behavior / correctness** | Static paint is unchanged. Documents that do not set the new properties produce the same operations. Metadata and keyframes are additive output. The C binding removal changes only the opt-in `c-shared` build surface. |
| **API / CLI** | Additive Go fields and types, plus the removed C binding. WASM JSON gains box fields and an optional `keyframes` array. There is no CLI in this tree. |
| **Dependencies** | None. The direct module allowlist stays `go-text/typesetting` and `tdewolff/canvas`. |
| **Binary size / build time** | The C shared-library target and its artifacts are gone. The WASM stamp moves to 0.0.2. No new packages. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| C shared-library binding removed | `bindings/c`, `make c-shared`, and the generated `libblinkless` artifacts are gone. Use the Go packages or `bindings/wasm`. The stub never shipped in a blinkless tag with a drawing-list export. |
| `layout.Box` gains fields | Keyed struct literals keep compiling. Unkeyed literals must add the new fields or switch to keyed fields. |
| WASM JSON gains optional fields | The schema stays `blinkless.drawinglist/1`. A consumer that rejects unknown fields must ignore them. |

## Test plan

- [x] `make test`
- [x] `make lint`
- [x] `make golden`
- [x] `make claim-scan`
- [x] `make catalog-check`
- [x] `make matrix-check`
- [x] `make wasm-test`

### Commands

```sh
make test
make lint
make golden
make claim-scan
make catalog-check
make matrix-check
make wasm-test
```

Focused coverage in the branch:

- `TestKeyframesPreserveOffsetsAndDeclarations`, `TestKeyframesSelectorListsAndDocumentOrder`
- `TestDisplayAnimationAndTransitionMetadata`, `TestDisplayAnimationTriggerMetadata`
- `TestAnimationDeclarationValueAccepted`, `TestApplyAnimationPropsStoresCommaLists`, `TestExpandAnimationDeclaration`
- `TestTransitionDeclarationValueAccepted`, `TestExpandTransitionShorthand`, `TestInvalidLaterAnimationTransitionDeclarationsDoNotWin`
- `TestWebkitAnimationTransitionAliasesMatchStandardProperties`, `TestWebkitAnimationTransitionAliasesRespectCascadeOrder`
- `TestDrawingListAnimationAndTransition`

## Screenshots / sample output

No visual surface changed. The drawing-list tests check the new metadata; static operations are unchanged.

## Related issues

- None. This branch does not close an open issue.

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues reviewed; no open issue matches this branch
- [x] Filled body committed under `plans/PR/pr-animations-transitions.md`

## Follow-ups (out of scope)

- Runtime sampling, transition comparison, event dispatch, trigger activation, and view-transition snapshots remain outside this metadata contract.
- CSS escape decoding and non-ASCII identifiers stay parser limits (`internal/css/css.go:413`).
- `shape-inside` on auto-height boxes and other v0.0.3 ledgers continue in their own branches.

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
- [ ] Diff-stat-by-extension table pasted at the bottom

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 36 | 5476 | 1287 |
| `.h` | 1 | 0 | 172 |
| `.html` | 1 | 1 | 1 |
| `.js` | 2 | 205 | 205 |
| `.json` | 2 | 1655 | 388 |
| `.md` | 16 | 525 | 1035 |
| `.ts` | 1 | 2 | 2 |
| `.yml` | 2 | 6 | 6 |
| No extension | 2 | 5 | 36 |
| **Total** | **63** | **7875** | **3132** |
