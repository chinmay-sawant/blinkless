# Anchor positioning

One-canvas subset of css-anchor-position-1. The engine places one absolute or fixed box against another box on the same drawing list. There is no scrollport and no JavaScript.

## Contract

Input is HTML plus CSS. Output stays `layout.DisplayList`. Paint Y grows downward. One CSS pixel is 0.75 points (`layout/displaylist.go`). The positioned box moves its border box and its paint ops. The anchor box does not move.

Only `position: absolute` and `position: fixed` enter anchor placement. `position: static` keeps flow position. `position: sticky` never enters anchor placement.

## Properties

Pinned upstream is `webref-ed-css` revision `1f2ec8f74a80c14066b4c7d6822cee59f69fa03b`, spec `css-anchor-position`.

| Property | Accepted |
|---|---|
| `anchor-name` | `none` or one or more dashed idents such as `--menu`. Invalid keeps previous. `none` clears. |
| `anchor-scope` | `none`, `all`, or a name list. `none` is initial and hides nothing. `all` hides every name on that element from descendants outside it. A name list hides only those names. Lookup only, no containing block. |
| `position-anchor` | `auto` or one dashed ident. Unknown keeps earlier value. `normal`, `none` and `match-parent` are rejected. |
| `position-area` | `none` or a `position-area` keyword. `none` keeps `left` and `top` offsets. Single keywords `top`, `bottom`, `left`, `right` and `center` move. Exotic multi-token areas are stored but keep offsets. |
| `position-try` | Shorthand for order plus fallbacks, including fallbacks only. CSS-wide keywords reset both. A following longhand wins in source order. |
| `position-try-fallbacks` | `none`, a `position-area` list, or pinned tactic keywords `flip-block`, `flip-inline`, `flip-start`, `flip-x` and `flip-y`. Dashed idents read `@position-try` areas. Unknown idents are skipped. Inset overrides from `@position-try` are stored but not applied. |
| `position-try-order` | `normal` keeps document order. Size keywords `most-width`, `most-height`, `most-block-size` and `most-inline-size` pick the largest passing `no-overflow` candidate on that axis. |
| `position-visibility` | `always`, `anchor-valid`, `anchor-visible`, `no-overflow`, and combinations. Invalid keeps previous. `always` keeps the base even off canvas. `anchor-valid` falls back when the anchor is missing or out of scope, else keeps ordinary absolute placement and still paints. `no-overflow` rejects candidates outside `Options.Width` and `Options.Height`. |

`anchor()` inside `top`, `left`, `right`, `bottom` or `inset` is out of scope. `position-area` moves the box without it.

## Scope and auto

A name on a box is visible to later boxes unless an ancestor scope separates them. A descendant inside the scope sees the name. A box outside it does not.

`position-anchor: auto` uses the anchor only when exactly one in-scope `anchor-name` exists. When several exist, `auto` does not guess. The box keeps the offsets from ordinary absolute placement. A later explicit `position-anchor` names one.

## Candidate selection

The base `position-area` runs first. With `always`, the base wins and fallbacks are unused. Otherwise each candidate is tested in order. `anchor-valid` rejects missing or out-of-scope anchors. `no-overflow` rejects boxes outside the viewport. `flip-block` and `flip-y` mirror vertical, `flip-inline` and `flip-x` mirror horizontal, `flip-start` mirrors both. The flip applies after the area is known and before the final shift.

## Scroll non-goal

`anchor-visible` is accepted and stored. On this canvas it matches `anchor-valid` because the tree has no scroll offset. Hiding a box when its anchor leaves a scrollport waits on host-scroll work (`plans/v0.0.1/native-window/native-web-view-checklist.md`). That work owns scrolling. Blinkless returns one drawing list.

## Code

- `internal/layout/anchor_position.go` owns parsing, scope lookup, placement, fallbacks, order and visibility.
- `internal/layout/style.go` holds the seven used fields.
- `internal/layout/style_cascade.go` registers `applyAnchorProps` and expands `position-try`.
- `internal/css/at_position_try.go` parses `@position-try` blocks.
- `internal/layout/layout_context.go` calls `resolveAnchorPositions` once before chrome merge.
- Tests: `internal/layout/anchor_position_test.go`, `internal/layout/css_behavior_anchor_test.go`, `internal/layout/css_behavior_anchor_try_test.go`, `internal/css/at_position_try_test.go`.

Gates: `make test`, `make golden`, `make lint`, `make claim-scan`, `make catalog-check`, `make matrix-check`.
