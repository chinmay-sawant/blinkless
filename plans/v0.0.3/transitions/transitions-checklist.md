# v0.0.3 - Transition metadata

> **Parent:** `plans/v0.0.3/README.md`
> **Status:** validated on 2026-10-19.
> **Scope:** 6 standard properties and 5 `-webkit-transition*` aliases.

## Result

Blinkless accepts transition declarations and exports one record per comma-list item. The record contains the property name, duration, delay, easing text, and behavior. A host with before and after display lists can decide whether to sample a transition. Blinkless does not compare those lists or interpolate values.

The parser, shorthand expansion, and cascade handler live in `internal/layout/transition_style.go`. Shared easing parsing lives in `internal/layout/easing_value.go`. Records are built in `internal/layout/animation_transition.go`, carried by `internal/layout/placed.go`, copied to public boxes in `layout/layout.go`, and encoded by `bindings/wasm/drawing_list.go`.

## Delivered contract

- [x] Accept and cascade the 6 standard properties: `transition`, `transition-property`, `transition-duration`, `transition-delay`, `transition-timing-function`, and `transition-behavior`. Reject invalid later declarations without replacing an earlier valid value. Proof: `TestTransitionDeclarationValueAccepted`, `TestApplyTransitionPropsAndWideKeywords`, and `TestInvalidLaterAnimationTransitionDeclarationsDoNotWin`.
- [x] Expand the shorthand into its five longhands with defaults, preserve comma-list alignment, and apply CSS-wide keywords. A negative duration is rejected; a negative delay is retained. Proof: `TestExpandTransitionShorthand` and `TestApplyTransitionPropsAndWideKeywords`.
- [x] Parse the easing forms covered by `easingValueAccepted`, including keywords, `cubic-bezier()`, `steps()`, and `linear()`. Proof: `TestEasingValueAccepted`.
- [x] Normalize the 5 aliases `-webkit-transition`, `-webkit-transition-delay`, `-webkit-transition-duration`, `-webkit-transition-property`, and `-webkit-transition-timing-function`. Keep declaration source order when prefixed and standard names are mixed. Proof: `TestWebkitAnimationTransitionAliasesMatchStandardProperties` and `TestWebkitAnimationTransitionAliasesRespectCascadeOrder`.
- [x] Keep each comma-list item aligned in one `layout.Transition` record and expose it on `layout.Box`. Carry the same record through the WASM JSON document while keeping schema `blinkless.drawinglist/1`. Proof: `TestDisplayAnimationAndTransitionMetadata`, `TestDrawingListAnimationAndTransition`, and `make wasm-test`.
- [x] Record parser limits without claiming full CSS parser conformance. Transition property names use `css.IsIdentToken`, which accepts ASCII identifier characters only (`internal/css/css.go:413`). CSS escape decoding and non-ASCII identifiers are outside this metadata result. Blinkless also does not compare before and after styles, start or cancel transitions, interpolate values, dispatch transition events, or generate frames.
- [x] Keep the catalog and reference docs aligned with the shipped Partial status and the runtime limits. Proof: `make catalog-check`, `make matrix-check`, and `make claim-scan` passed.
- [x] Pass the final code gates: `GOCACHE=/tmp/blinkless-v003-gocache make test`, `GOCACHE=/tmp/blinkless-v003-gocache make golden`, `GOCACHE=/tmp/blinkless-v003-gocache make lint`, and `GOCACHE=/tmp/blinkless-v003-gocache make wasm-test`.

## Scope decision

Transition state comparison, frame sampling, interpolation, and transition events are not part of this version's metadata contract. This is a completed scope decision, not pending work.
