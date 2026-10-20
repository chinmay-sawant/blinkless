# v0.0.3 - Transition metadata

> **Parent:** `plans/v0.0.3/README.md`
> **Status:** validated on 2026-10-20.
> **Scope:** 6 standard properties and 5 `-webkit-transition*` aliases.

## Result

Blinkless accepts transition declarations and exports one record per comma-list item. The record contains the property name, duration, delay, easing text, and behavior. OwnFrame can bind those records to paint from before and after display lists and sample them over time (`../ownframe/internal/csstransition/bind.go:12-48`, `../ownframe/internal/transition/step.go:9-43`). The host supplies both lists and the clock; Blinkless does not compare styles or start transitions itself.

The parser, shorthand expansion, and cascade handler live in `internal/layout/transition_style.go`. Shared easing parsing lives in `internal/layout/easing_value.go`. Records are built in `internal/layout/animation_transition.go`, carried by `internal/layout/placed.go`, copied to public boxes in `layout/layout.go`, and encoded by `bindings/wasm/drawing_list.go`.

## Delivered contract

- [x] Accept and cascade the 6 standard properties: `transition`, `transition-property`, `transition-duration`, `transition-delay`, `transition-timing-function`, and `transition-behavior`. Reject invalid later declarations without replacing an earlier valid value. Proof: `TestTransitionDeclarationValueAccepted`, `TestApplyTransitionPropsAndWideKeywords`, and `TestInvalidLaterAnimationTransitionDeclarationsDoNotWin`.
- [x] Expand the shorthand into its five longhands with defaults, preserve comma-list alignment, and apply CSS-wide keywords. A negative duration is rejected; a negative delay is retained. Proof: `TestExpandTransitionShorthand` and `TestApplyTransitionPropsAndWideKeywords`.
- [x] Parse the easing forms covered by `easingValueAccepted`, including keywords, `cubic-bezier()`, `steps()`, and `linear()`. Proof: `TestEasingValueAccepted`.
- [x] Normalize the 5 aliases `-webkit-transition`, `-webkit-transition-delay`, `-webkit-transition-duration`, `-webkit-transition-property`, and `-webkit-transition-timing-function`. Keep declaration source order when prefixed and standard names are mixed. Proof: `TestWebkitAnimationTransitionAliasesMatchStandardProperties` and `TestWebkitAnimationTransitionAliasesRespectCascadeOrder`.
- [x] Keep each comma-list item aligned in one `layout.Transition` record and expose it on `layout.Box`. Carry the same record through the WASM JSON document while keeping schema `blinkless.drawinglist/1`. Proof: `TestDisplayAnimationAndTransitionMetadata`, `TestDrawingListAnimationAndTransition`, and `make wasm-test`.
- [x] Record parser limits without claiming full CSS parser conformance. Transition property names use `css.IsIdentToken`, which accepts ASCII identifier characters only (`internal/css/css.go:413`). CSS escape decoding and non-ASCII identifiers are outside this metadata result. Blinkless does not compare before and after styles or generate transition frames; the host owns that runtime.
- [x] Sample `transition-behavior` for a discrete `position` change in OwnFrame and record the downstream test in the catalog. `allow-discrete` holds the old paint before halfway, then switches to the new paint. Visibility transitions remain unsupported because Blinkless emits no visibility paint operation. Proof: `make catalog-ownframe-check` passed all 43 motion cases; OwnFrame `make test` and `make build` passed with the local Blinkless workspace; `make catalog-check` reported 555 implemented and 3 partial.
- [x] Pass the final code and documentation gates on 2026-10-20. Blinkless passed `make test`, `make claim-scan`, `make lint`, `make golden`, `make wasm-test`, `make catalog-check`, `make catalog-ownframe-check`, and `make matrix-check`. OwnFrame passed `make test` and `make build`. Updated motion documentation in both repositories; reran Blinkless `make claim-scan` and `make matrix-check` successfully. The catalog reports 785 properties: 555 implemented, 3 partial, 219 unsupported, and 8 intentionally ignored.

## Scope decision

Blinkless v0.0.3 still exports transition metadata and does not compare before and after styles. OwnFrame samples six standard transition properties and five WebKit aliases. It also samples `transition-behavior: allow-discrete` for the supported `position` paint value. It does not automatically diff Page styles or dispatch transition events.
