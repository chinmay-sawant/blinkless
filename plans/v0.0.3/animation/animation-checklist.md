# v0.0.3 - Animation metadata and view-transition data

> **Parent:** `plans/v0.0.3/README.md`
> **Status:** validated on 2026-10-19.
> **Scope:** 23 standard properties and 9 `-webkit-animation*` aliases.

## Result

Blinkless keeps animation values for a host that may sample them later. It also keeps parsed `@keyframes` rules on the display list. Blinkless does not advance time, apply keyframe values, activate triggers, or build view-transition snapshots.

The parser and cascade code lives in `internal/css/keyframes.go` and `internal/layout/animation_style.go`. The per-item records are built in `internal/layout/animation_transition.go` and carried by `internal/layout/placed.go`. The public boxes and keyframe table are in `layout/layout.go` and `layout/displaylist.go`. WASM encodes the same values in `bindings/wasm/drawing_list.go`.

## Delivered contract

- [x] Parse `@keyframes` and `@-webkit-keyframes` into source-ordered rules with offsets and the declarations written at each offset. Duplicate names stay in source order; a host selects the last matching rule. Proof: `TestKeyframesPreserveOffsetsAndDeclarations`, `TestKeyframesSelectorListsAndDocumentOrder`, and `TestDisplayAnimationAndTransitionMetadata`.
- [x] Accept and cascade the 23 standard properties: `animation`, `animation-composition`, `animation-delay`, `animation-delay-start`, `animation-delay-end`, `animation-direction`, `animation-duration`, `animation-fill-mode`, `animation-iteration-count`, `animation-name`, `animation-play-state`, `animation-range`, `animation-range-center`, `animation-range-end`, `animation-range-start`, `animation-timeline`, `animation-timing-function`, `animation-trigger`, `trigger-scope`, `view-transition-class`, `view-transition-group`, `view-transition-name`, and `view-transition-scope`. Proof: `TestAnimationDeclarationValueAccepted`, `TestApplyAnimationPropsStoresCommaLists`, `TestExpandAnimationDeclaration`, and `TestAnimationTransitionSupportsAcceptanceAndCascade`.
- [x] Normalize the 9 aliases `-webkit-animation`, `-webkit-animation-delay`, `-webkit-animation-direction`, `-webkit-animation-duration`, `-webkit-animation-fill-mode`, `-webkit-animation-iteration-count`, `-webkit-animation-name`, `-webkit-animation-play-state`, and `-webkit-animation-timing-function`. Keep declaration source order when prefixed and standard names are mixed. Proof: `TestWebkitAnimationTransitionAliasesMatchStandardProperties` and `TestWebkitAnimationTransitionAliasesRespectCascadeOrder`.
- [x] Keep each comma-list item aligned in one `layout.Animation` record. Preserve time values in milliseconds, the `auto` duration flag, infinite iteration, timeline and range text, and trigger text. Keep trigger-scope and view-transition identity values on the matching `layout.Box`. Proof: `TestDisplayAnimationAndTransitionMetadata` and `TestDisplayAnimationTriggerMetadata`.
- [x] Carry animation records and keyframes through the public Go display list and the WASM JSON document. Keep the drawing-list schema at `blinkless.drawinglist/1`. Proof: `TestDisplayAnimationAndTransitionMetadata`, `TestDrawingListAnimationAndTransition`, and `make wasm-test`.
- [x] Record parser limits without claiming full CSS parser conformance. `css.IsIdentToken` accepts ASCII identifier characters only (`internal/css/css.go:413`). Length-percentage math uses `css.EvalMath`, which evaluates numeric values without checking CSS dimensions (`internal/layout/animation_style.go`, `internal/css/math_expr.go:29`). CSS escape decoding, non-ASCII identifiers, dimensional math validation, runtime sampling, event activation, and view-transition snapshots are outside this metadata result.
- [x] Keep the catalog and reference docs aligned with the shipped Partial status and the runtime limits. Proof: `make catalog-check`, `make matrix-check`, and `make claim-scan` passed.
- [x] Pass the final code gates: `GOCACHE=/tmp/blinkless-v003-gocache make test`, `GOCACHE=/tmp/blinkless-v003-gocache make golden`, `GOCACHE=/tmp/blinkless-v003-gocache make lint`, and `GOCACHE=/tmp/blinkless-v003-gocache make wasm-test`.

## Scope decision

The in-engine clock, frame sampling, interpolation, animation and transition events, trigger activation, and view-transition snapshots are not part of this version's metadata contract. This is a completed scope decision, not pending work.
