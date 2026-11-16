# v0.1.2 - Print-noop properties

> Update 2026-11-16: the ten scroll, view, and pointer timeline names now take
> comma lists and name the scroll container they read, so their catalog rows are
> `implemented`. Any sentence below that says one name per kind, a one-axis
> store, or a page-only scroller describes the state before that change. The
> proof is in [../validation.md](../validation.md).

> **Parent:** `plans/v0.1.2/README.md`
> **Status:** 75 names parked. The 27 timeline, trigger, interest, and interactivity names have an engine row `[x]` and a host row `[~]`.
> **Estimated effort:** Leave 75 names parked until a separate product adds a scrollport, a speech reader, a pointer, or a host timeline. This ledger records that wait.

---

## Overview

Blinkless returns one drawing list from `layout.DisplayList` (`layout/displaylist.go:257`). A drawing list is the boxes and paint operations for the page. The matrix calls this set print-noop because a static drawing list has no scrollport, speaker, pointer, or clock to apply these declarations. The engine has no scrollport (the rectangle a person scrolls), no pointer (mouse or touch), no caret (the text cursor), no speech reader, and no clock of its own (`layout/layout.go:96`, `layout/timeline.go:11-13`).

Animation and transition metadata are already on the public box for a host to sample (`layout/layout.go:22-23`, `layout/layout.go:95-97`, `layout/layout.go:117-118`). A host is the program outside this module that shows the page, plays sound, or moves a clock. This file does not replan animation or transition.

Section 5.4 of `documentation/compatibility-matrix.md` lists 114 names in five print-noop families: 102 unsupported, five implemented `scroll-margin` properties, and seven intentionally ignored (`documentation/compatibility-matrix.md:1123-1134`). This ledger is those 102 unsupported names. The catalog header on 2026-11-02 is 785 properties, 555 implemented, 4 partial, 218 unsupported, and 8 intentionally ignored (`testdata/css/catalog/properties.json:4`, `properties.json:11-18`). The 102 are a subset of the 218. Each of them has status `unsupported` and an empty handlers list.

A name with no style-group owner is still accepted by the cascade. The cascade is the step that picks the winning declaration. A handler is the function that stores one property. An unlisted name passes `declarationValueAccepted` (`internal/layout/style_value_accept.go:236-239`, default at `938-939`) and is then discarded by `applyIgnoredGroup` (`internal/layout/style_cascade.go:1974-1980`, `internal/layout/style_properties.go:1876-1880`).

Seven of the 102 already have storage even though the catalog row is still unsupported: `scroll-timeline`, `scroll-timeline-name`, `scroll-timeline-axis`, `view-timeline`, `view-timeline-name`, `view-timeline-axis`, and `view-timeline-inset`. `applyTimelineProps` writes them (`internal/layout/style_cascade.go:1747`, `internal/layout/timeline_style.go:356-369`) and `layout.Box.Timelines` exports them (`layout/layout.go:29-32`). The export carries the name, the axis, and view insets. It does not carry progress.

`[~]` means deferred on purpose. Each row names a reason, the owner (this ledger), and the next gate. The next gate is a product change: a scrollport, a speech reader, a pointer, or a host timeline. It is not a patch that only stores a keyword in this module.

This file is the current parked ledger. It supersedes the open rows in `plans/v0.0.3/speech/speech-checklist.md` and `plans/v0.0.3/scroll/scroll-checklist.md` for the 102 names below, including the view-timeline and pointer-timeline rows in the scroll file. Those two files are not edited. Their open rows that ask this module to store speech values, snap keywords, padding, overscroll, scrollbar chrome, or pointer timelines are no longer the plan. Open rows in the old scroll file that only concern the five physical `scroll-margin` properties are outside this ledger.

## Executive summary

| Phase | Headings | What they wait for |
|---|---:|---|
| 1 speech | 19 | A speech reader |
| 2 scroll and overscroll | 36 | A scrollport. The three `scroll-timeline` names also wait for a host timeline to turn the exported name into progress |
| 3 pointer, caret, form | 18 | A pointer, including caret and form chrome |
| 4 motion and view timelines | 13 | A host that places a box on a motion path, a scrollport for `overflow-anchor`, and a host timeline for view progress and `timeline-scope` |
| 5 event and pointer timelines | 16 | A pointer or an event, plus a host timeline |
| Total | 102 | 27 of them are engine-done (rows `[x]`); their host halves stay `[~]` |

The five physical scroll-margin properties and eight intentionally ignored cousins are phase 0. They are not part of the 102.

## Phase 0: already implemented cousins

`scroll-margin`, `scroll-margin-top`, `scroll-margin-right`, `scroll-margin-bottom`, and `scroll-margin-left` are implemented. `applyScrollProps` stores a non-negative length on each physical side (`internal/layout/scroll.go:28-32`, `internal/layout/scroll.go:49-76`) and expands the shorthand in top, right, bottom, left order (`internal/layout/scroll.go:87-120`). A negative length is rejected (`internal/layout/scroll.go:123-133`). The fields are `ScrollMarginTop`, `ScrollMarginRight`, `ScrollMarginBottom`, and `ScrollMarginLeft` (`internal/layout/style.go:445-448`). The group is registered in `styleGroups` (`internal/layout/style_cascade.go:1740`). `applySVGPresentationProps` lists the same five names and returns them to `applyLeftoversProps`, which declines them, so the scroll group still runs (`internal/layout/style_paint_props.go:124-127`, `internal/layout/scroll.go:42-47`). The snap area is the border box outset by those lengths (`internal/layout/scroll.go:196-198`). Production scroll offset stays 0 (`internal/layout/scroll.go:136-139`). This ledger does not add implementation rows for these five names.

The catalog marks seven cousins in section 5.4 as intentionally ignored, plus `page`, which is the eighth intentionally ignored name in the catalog summary. Intentionally ignored means the catalog records a deliberate drop, not an unfinished patch. Each of these stays out of the work rows.

`appearance` is intentionally ignored (`testdata/css/catalog/properties.json:2344`). `appearance: none` on a range input does not change the painted thumb (`internal/layout/css_behavior_interaction_test.go:302-318`).

`caret-color` is intentionally ignored (`testdata/css/catalog/properties.json:5608`). The engine paints no caret, so `caret-color: #ff0000` does not change paint (`internal/layout/css_behavior_interaction_test.go:271-282`).

`cursor` is intentionally ignored (`testdata/css/catalog/properties.json:8141`). `cursor: pointer` does not change paint (`internal/layout/css_behavior_interaction_test.go:241-253`).

`pointer-events` is intentionally ignored (`testdata/css/catalog/properties.json:14753`). There is no hit testing, so `pointer-events: none` does not change paint (`internal/layout/css_behavior_interaction_test.go:224-238`).

`resize` is intentionally ignored (`testdata/css/catalog/properties.json:15166`). `resize: both` does not change the box or the paint (`internal/layout/css_behavior_interaction_test.go:285-298`).

`touch-action` is intentionally ignored (`testdata/css/catalog/properties.json:19110`). No Go file stores the name, and the interaction tests do not list it among the paint no-ops (`internal/layout/css_behavior_interaction_test.go:19-22`).

`user-select` is intentionally ignored (`testdata/css/catalog/properties.json:19602`). There is no selection model, so `user-select: none` does not change paint (`internal/layout/css_behavior_interaction_test.go:256-268`).

`page` is intentionally ignored (`testdata/css/catalog/properties.json:14456`). `applyTableBreakGroup` stores the name on `ResolvedStyle.PageName` (`internal/layout/style_properties.go:1623-1624`, `internal/layout/style_properties.go:1746-1770`, field at `internal/layout/style.go:335-337`). `page: chapter` inherits (`internal/layout/page_named_test.go:12-24`). The public result is still one continuous drawing list, so the name does not select a page box.

## Phase 1: speech

A speech reader is a program that says the words aloud. This module returns boxes and paint. It has no speaker. Each name below waits for that product. The old speech checklist's open rows are superseded by this phase.

### cue

No handler stores `cue`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:8093`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `cue: url("beep.wav") url("ding.wav")` and demands `beep.wav` play before the element's words and `ding.wav` play after them. One value, `cue: url("beep.wav")`, demands that same cue on both sides. The URL stays a sound address. It is not loaded through the image callback.

- [~] Reason: no speech reader, and no handler stores `cue`. Owner: this ledger. Next gate: a speech product that plays `beep.wav` before the words and `ding.wav` after them for `cue: url("beep.wav") url("ding.wav")`.

### cue-after

No handler stores `cue-after`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:8109`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `cue-after: url("ding.wav") -3dB` and demands `ding.wav` play 3 decibels quieter after the element's words. `cue-after: none` demands silence in that slot. A test that swaps this URL with the before-cue URL fails.

- [~] Reason: no speech reader, and no handler stores `cue-after`. Owner: this ledger. Next gate: a speech product that plays `ding.wav` at -3 dB after the words when the declaration is `cue-after: url("ding.wav") -3dB`.

### cue-before

No handler stores `cue-before`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:8125`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `cue-before: url("beep.wav")` and demands `beep.wav` play before the element's words, with no extra decibel attached. `cue-before: none` demands no leading sound.

- [~] Reason: no speech reader, and no handler stores `cue-before`. Owner: this ledger. Next gate: a speech product that plays `beep.wav` before the words when the declaration is `cue-before: url("beep.wav")`.

### pause

No handler stores `pause`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:14519`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `pause: strong weak` and demands a strong pause before the words and a weak pause after them. One value, `pause: 200ms`, demands a 200 millisecond pause on both sides.

- [~] Reason: no speech reader, and no handler stores `pause`. Owner: this ledger. Next gate: a speech product that inserts a strong pause before the words and a weak pause after them for `pause: strong weak`.

### pause-after

No handler stores `pause-after`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:14535`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `pause-after: weak` and demands a weak pause after the words, which fails if the before-pause value is copied into this side. `pause-after: 150ms` demands a 150 millisecond pause after the words. A negative time is out of grammar.

- [~] Reason: no speech reader, and no handler stores `pause-after`. Owner: this ledger. Next gate: a speech product that pauses weakly after the words when the declaration is `pause-after: weak`.

### pause-before

No handler stores `pause-before`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:14551`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `pause-before: 200ms` and demands 200 milliseconds of pause before the words. `pause-before: strong` demands the strong pause step instead of a timed wait. `pause-before: -200ms` is out of grammar because the time starts at 0s.

- [~] Reason: no speech reader, and no handler stores `pause-before`. Owner: this ledger. Next gate: a speech product that waits 200 milliseconds before speaking the element when the declaration is `pause-before: 200ms`.

### rest

No handler stores `rest`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:15187`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `rest: none 100ms` and demands no rest before the words and a 100 millisecond rest after them. A rest is silence inside the utterance. A test fails if those values are applied as `pause`.

- [~] Reason: no speech reader, and no handler stores `rest`. Owner: this ledger. Next gate: a speech product that keeps the utterance silent for 100 milliseconds after the words, and silent for no extra time before them, when the declaration is `rest: none 100ms`.

### rest-after

No handler stores `rest-after`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:15203`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `rest-after: x-strong` and demands the strongest rest step after the words. `rest-after: 80ms` demands an 80 millisecond rest in that slot. Copying `rest-before` into this side fails the test.

- [~] Reason: no speech reader, and no handler stores `rest-after`. Owner: this ledger. Next gate: a speech product that inserts an x-strong rest after the words when the declaration is `rest-after: x-strong`.

### rest-before

No handler stores `rest-before`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:15219`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `rest-before: medium` and demands a medium rest before the words. The same keywords and non-negative times as `pause-before` are legal here, and the test fails if the value is stored on the pause side.

- [~] Reason: no speech reader, and no handler stores `rest-before`. Owner: this ledger. Next gate: a speech product that inserts a medium rest before the words when the declaration is `rest-before: medium`.

### speak

No handler stores `speak`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:17103`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `speak: never` and demands the element's words be skipped. `speak: always` demands they be spoken even when the usual rule would skip them. `speak: auto` follows that usual rule. The test fails if `never` is treated as `auto`.

- [~] Reason: no speech reader, and no handler stores `speak`. Owner: this ledger. Next gate: a speech product that skips the element's words when the declaration is `speak: never`.

### speak-as

No handler stores `speak-as`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:17119`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `speak-as: spell-out digits` and demands each letter be spelled and each digit be spoken alone. Dropping either keyword fails the test. `speak-as: literal-punctuation` demands punctuation be spoken as words.

- [~] Reason: no speech reader, and no handler stores `speak-as`. Owner: this ledger. Next gate: a speech product that spells letters and speaks digits one by one when the declaration is `speak-as: spell-out digits`.

### voice-balance

No handler stores `voice-balance`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:19938`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `voice-balance: left` and demands the voice sit hard on the left. A second case, `voice-balance: -50`, demands a numeric pan of -50. The keyword `left` stays a keyword, so a test fails if it is collapsed to the number 0.

- [~] Reason: no speech reader, and no handler stores `voice-balance`. Owner: this ledger. Next gate: a speech product that pans the voice hard left for `voice-balance: left` and pans it by the number -50 for `voice-balance: -50`.

### voice-duration

No handler stores `voice-duration`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:19954`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `voice-duration: 2s` and demands the utterance be fit into 2 seconds. `voice-duration: auto` lets `voice-rate` decide the length. The test fails if `2s` is treated as `auto` or as 0 seconds.

- [~] Reason: no speech reader, and no handler stores `voice-duration`. Owner: this ledger. Next gate: a speech product that fits the utterance into 2 seconds when the declaration is `voice-duration: 2s`.

### voice-family

No handler stores `voice-family`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:19970`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `voice-family: preserve` and demands the voice already in use stay in use. A second case, `voice-family: female`, demands the generic female voice. The catalog grammar is a list of voice names or generics, or the single keyword `preserve`.

- [~] Reason: no speech reader, and no handler stores `voice-family`. Owner: this ledger. Next gate: a speech product that keeps the current voice for `voice-family: preserve` and switches to the generic female voice for `voice-family: female`.

### voice-pitch

No handler stores `voice-pitch`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:19986`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `voice-pitch: high` and demands a pitch one step above medium. `voice-pitch: 440Hz absolute` demands 440 Hz with the absolute flag, so the inherited pitch is not added. The test fails if `high` is treated as `medium`.

- [~] Reason: no speech reader, and no handler stores `voice-pitch`. Owner: this ledger. Next gate: a speech product that raises pitch to the high step for `voice-pitch: high`, and sets 440 Hz absolute for `voice-pitch: 440Hz absolute`.

### voice-range

No handler stores `voice-range`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:20002`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `voice-range: x-low` and demands the narrowest pitch range, which fails if the pitch value is copied onto the range. `voice-range: 50%` demands the range shrink to half. The grammar matches `voice-pitch`, on this other field.

- [~] Reason: no speech reader, and no handler stores `voice-range`. Owner: this ledger. Next gate: a speech product that uses the x-low pitch range when the declaration is `voice-range: x-low`.

### voice-rate

No handler stores `voice-rate`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:20018`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `voice-rate: fast 150%` and demands the fast step combined with 150 percent of the normal rate. `voice-rate: x-slow` demands the slowest keyword with no percentage attached. The test fails if the percentage is dropped.

- [~] Reason: no speech reader, and no handler stores `voice-rate`. Owner: this ledger. Next gate: a speech product that speaks at the fast step and 150 percent rate when the declaration is `voice-rate: fast 150%`.

### voice-stress

No handler stores `voice-stress`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:20034`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `voice-stress: strong` and demands stronger stress on the words. `voice-stress: reduced` demands less stress. `voice-stress: none` demands flat stress. The test fails if `strong` is treated as `normal`.

- [~] Reason: no speech reader, and no handler stores `voice-stress`. Owner: this ledger. Next gate: a speech product that stresses the words strongly when the declaration is `voice-stress: strong`.

### voice-volume

No handler stores `voice-volume`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:20050`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future speech test sets `voice-volume: silent` and demands no sound, which fails if `silent` is treated as 0 dB. `voice-volume: x-loud 6dB` demands the loudest keyword plus 6 decibels.

- [~] Reason: no speech reader, and no handler stores `voice-volume`. Owner: this ledger. Next gate: a speech product that stays silent for `voice-volume: silent` and speaks at x-loud plus 6 dB for `voice-volume: x-loud 6dB`.

## Phase 2: scroll and overscroll

A scrollport is the rectangle a person scrolls. The snap window is the part of that rectangle a target lines up with. `scroll-padding` shrinks the snap window. horizontal-tb is the usual mode, lines running left to right. vertical-rl stacks those lines from right to left. The block direction is the direction lines stack. The inline direction is the direction a line runs. Production layout keeps the scroll offset at 0 (`internal/layout/scroll.go:136-139`). Each name below waits for a scrollport. The three `scroll-timeline` names already export a name and an axis, and they still wait for a host timeline to compute progress. Open scroll-checklist rows for these names are superseded here. The five physical scroll-margin properties stay in phase 0.

### overscroll-behavior

No handler stores `overscroll-behavior`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:14079`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `overscroll-behavior: contain none`. The first keyword applies to the x axis and the second to the y axis, so a gesture that hits the x edge stays inside this scrollport, and a gesture that hits the y edge does not chain and does not show an overscroll effect. One keyword, `overscroll-behavior: chain`, applies `chain` to both axes and passes leftover movement to the parent.

- [~] Reason: no scrollport, and no handler stores `overscroll-behavior`. Owner: this ledger. Next gate: a scrollport that keeps x inside this scroller and applies `none` on y when the declaration is `overscroll-behavior: contain none`.

### overscroll-behavior-block

No handler stores `overscroll-behavior-block`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:14095`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `overscroll-behavior-block: contain`. In horizontal-tb the block axis is y, the axis `applyLogicalMarginBlockPair` writes as top and bottom (`internal/layout/style_logical_box.go:112-114`), so the leftover y movement stays in this scrollport. In `vertical-rl` the block axis is x (`style_logical_box.go:106-108`), and the same keyword contains x.

- [~] Reason: no scrollport, and no handler stores `overscroll-behavior-block`. Owner: this ledger. Next gate: a scrollport that contains leftover movement on the block axis for `overscroll-behavior-block: contain`, y in horizontal-tb and x in `vertical-rl`.

### overscroll-behavior-inline

No handler stores `overscroll-behavior-inline`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:14111`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `overscroll-behavior-inline: chain` in horizontal-tb and demands leftover inline movement pass to the parent on x. In `vertical-rl` the same declaration chains on y, following the writing-mode split in `applyLogicalMargin` (`internal/layout/style_logical_box.go:8-15`).

- [~] Reason: no scrollport, and no handler stores `overscroll-behavior-inline`. Owner: this ledger. Next gate: a scrollport that chains leftover inline movement to the parent for `overscroll-behavior-inline: chain`.

### overscroll-behavior-x

No handler stores `overscroll-behavior-x`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:14127`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `overscroll-behavior-x: contain` and demands a gesture that hits the left or right edge stay inside this scrollport. `overscroll-behavior-x: none` demands no chain and no overscroll effect on x. The test fails if the y axis changes with it.

- [~] Reason: no scrollport, and no handler stores `overscroll-behavior-x`. Owner: this ledger. Next gate: a scrollport that contains leftover x movement when the declaration is `overscroll-behavior-x: contain`.

### overscroll-behavior-y

No handler stores `overscroll-behavior-y`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:14143`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `overscroll-behavior-y: none` and demands a gesture that hits the top or bottom edge neither chain nor show an overscroll effect. The value is `none` so a test cannot pass by copying a `contain` value from x.

- [~] Reason: no scrollport, and no handler stores `overscroll-behavior-y`. Owner: this ledger. Next gate: a scrollport that neither chains nor shows an overscroll effect on y when the declaration is `overscroll-behavior-y: none`.

### scroll-axis-lock

No handler stores `scroll-axis-lock`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16235`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-axis-lock: none` and demands a diagonal drag change both x and y. `scroll-axis-lock: auto` may lock the gesture to the axis that moved first. The static paint offset stays 0 until that scrollport exists (`internal/layout/scroll.go:136-139`).

- [~] Reason: no scrollport, and no handler stores `scroll-axis-lock`. Owner: this ledger. Next gate: a scrollport that lets a diagonal drag move both axes when the declaration is `scroll-axis-lock: none`.

### scroll-behavior

No handler stores `scroll-behavior`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16251`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-behavior: smooth` and demands a jump to a target element animate the scroll offset over time. `scroll-behavior: auto` demands the offset jump in one step. This module has no clock of its own (`layout/layout.go:96`) and the laid-out offset stays 0 (`internal/layout/scroll.go:136-139`).

- [~] Reason: no scrollport, and no handler stores `scroll-behavior`. Owner: this ledger. Next gate: a scrollport that animates the offset to the target when the declaration is `scroll-behavior: smooth`.

### scroll-initial-target

No handler stores `scroll-initial-target`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16267`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-initial-target: nearest` on a heading and demands the first scroll offset bring that heading into the scroll window. `scroll-initial-target: none` demands the offset stay 0. The test fails if `nearest` leaves the heading below the window.

- [~] Reason: no scrollport, and no handler stores `scroll-initial-target`. Owner: this ledger. Next gate: a scrollport whose first offset brings the nearest `scroll-initial-target: nearest` element into view.

### scroll-margin-block

No handler stores `scroll-margin-block`. `applyScrollProps` claims only `scroll-margin` and the four physical sides (`internal/layout/scroll.go:53-72`), and the catalog row is unsupported with an empty handlers list (`testdata/css/catalog/properties.json:16306`).

What to expect: A future scrollport test sets `scroll-margin-block: 40px 10px` in horizontal-tb and demands the snap area extend 40px past the top border and 10px past the bottom border, the block pair `applyLogicalMarginBlockPair` writes (`internal/layout/style_logical_box.go:112-114`). One length, `scroll-margin-block: 40px`, demands 40px on both block edges. The snap area is the border box outset by those lengths (`internal/layout/scroll.go:196-198`).

- [~] Reason: no scrollport, and no handler stores `scroll-margin-block`. Owner: this ledger. Next gate: a scrollport whose snap area uses top 40px and bottom 10px in horizontal-tb for `scroll-margin-block: 40px 10px`.

### scroll-margin-block-end

No handler stores `scroll-margin-block-end`. `applyScrollProps` claims only `scroll-margin` and the four physical sides (`internal/layout/scroll.go:53-72`), and the catalog row is unsupported with an empty handlers list (`testdata/css/catalog/properties.json:16322`).

What to expect: A future scrollport test sets `scroll-margin-block-end: 20px` in horizontal-tb and demands the snap area extend 20px past the bottom border. In `vertical-rl` that 20px sits on the left, the block-end side (`internal/layout/style_logical_box.go:34-35`).

- [~] Reason: no scrollport, and no handler stores `scroll-margin-block-end`. Owner: this ledger. Next gate: a scrollport whose snap area extends 20px past the block-end edge for `scroll-margin-block-end: 20px`.

### scroll-margin-block-start

No handler stores `scroll-margin-block-start`. `applyScrollProps` claims only `scroll-margin` and the four physical sides (`internal/layout/scroll.go:53-72`), and the catalog row is unsupported with an empty handlers list (`testdata/css/catalog/properties.json:16338`).

What to expect: A future scrollport test sets `scroll-margin-block-start: 40px` in horizontal-tb and demands the snap area extend 40px past the top border, the side `applyLogicalMarginBlockPair` writes for block start (`internal/layout/style_logical_box.go:112-114`). In `vertical-rl` that 40px sits on the right (`style_logical_box.go:32-33`).

- [~] Reason: no scrollport, and no handler stores `scroll-margin-block-start`. Owner: this ledger. Next gate: a scrollport whose snap area extends 40px past the block-start edge for `scroll-margin-block-start: 40px`.

### scroll-margin-inline

No handler stores `scroll-margin-inline`. `applyScrollProps` claims only `scroll-margin` and the four physical sides (`internal/layout/scroll.go:53-72`), and the catalog row is unsupported with an empty handlers list (`testdata/css/catalog/properties.json:16377`).

What to expect: A future scrollport test sets `scroll-margin-inline: 30px 5px` in horizontal-tb with `direction: ltr` and demands 30px on the left and 5px on the right. With `direction: rtl` those lengths swap, matching `applyLogicalMargin` (`internal/layout/style_logical_box.go:82-87`). One length sets both inline edges.

- [~] Reason: no scrollport, and no handler stores `scroll-margin-inline`. Owner: this ledger. Next gate: a scrollport whose snap area uses left 30px and right 5px in horizontal-tb ltr for `scroll-margin-inline: 30px 5px`.

### scroll-margin-inline-end

No handler stores `scroll-margin-inline-end`. `applyScrollProps` claims only `scroll-margin` and the four physical sides (`internal/layout/scroll.go:53-72`), and the catalog row is unsupported with an empty handlers list (`testdata/css/catalog/properties.json:16393`).

What to expect: A future scrollport test sets `scroll-margin-inline-end: 5px` in horizontal-tb ltr and demands 5px past the right border. In rtl that 5px sits on the left, the opposite side from inline-start (`internal/layout/style_logical_box.go:82-87`). The length differs from the inline-start case so a copied value fails.

- [~] Reason: no scrollport, and no handler stores `scroll-margin-inline-end`. Owner: this ledger. Next gate: a scrollport whose snap area extends 5px past the inline-end edge for `scroll-margin-inline-end: 5px`.

### scroll-margin-inline-start

No handler stores `scroll-margin-inline-start`. `applyScrollProps` claims only `scroll-margin` and the four physical sides (`internal/layout/scroll.go:53-72`), and the catalog row is unsupported with an empty handlers list (`testdata/css/catalog/properties.json:16409`).

What to expect: A future scrollport test sets `scroll-margin-inline-start: 30px` in horizontal-tb ltr and demands 30px past the left border. In rtl the same declaration outsets the right border (`internal/layout/style_logical_box.go:82-84`).

- [~] Reason: no scrollport, and no handler stores `scroll-margin-inline-start`. Owner: this ledger. Next gate: a scrollport whose snap area extends 30px past the inline-start edge for `scroll-margin-inline-start: 30px`, the left edge in ltr and the right edge in rtl.

### scroll-marker-group

No handler stores `scroll-marker-group`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16494`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-marker-group: before links` and demands a group of link markers placed before the scroller. Dropping either keyword fails the test. `scroll-marker-group: after tabs` demands tab markers after the scroller. The display list grows those marker boxes only when that scrollport product exists.

- [~] Reason: no scrollport, and no handler stores `scroll-marker-group`. Owner: this ledger. Next gate: a scrollport that builds a before-links marker group for `scroll-marker-group: before links`.

### scroll-padding

No handler stores `scroll-padding`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16510`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-padding: 20px 10% auto 8px` and demands the snap window inset by 20px on top, 10 percent of the scrollport width on the right, the `auto` keyword on the bottom, and 8px on the left. `auto` computes to a 0 inset and stays the keyword `auto` in the specified value. A negative length rejects the whole declaration.

- [~] Reason: no scrollport, and no handler stores `scroll-padding`. Owner: this ledger. Next gate: a scrollport whose snap window is inset 20px, 10% of the width, `auto`, and 8px for `scroll-padding: 20px 10% auto 8px`.

### scroll-padding-block

No handler stores `scroll-padding-block`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16526`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-padding-block: 12px auto` in horizontal-tb and demands a 12px inset at the block start (top) and the keyword `auto` at the block end (bottom). One value sets both block edges. The writing-mode map is the same pair as block scroll-margin (`internal/layout/style_logical_box.go:105-114`).

- [~] Reason: no scrollport, and no handler stores `scroll-padding-block`. Owner: this ledger. Next gate: a scrollport whose block snap insets are 12px then `auto` for `scroll-padding-block: 12px auto` in horizontal-tb.

### scroll-padding-block-end

No handler stores `scroll-padding-block-end`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16542`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-padding-block-end: 6px` in horizontal-tb and demands the snap window's bottom edge move inward by 6px. In `vertical-rl` that inset is on the left.

- [~] Reason: no scrollport, and no handler stores `scroll-padding-block-end`. Owner: this ledger. Next gate: a scrollport whose block-end snap inset is 6px for `scroll-padding-block-end: 6px`.

### scroll-padding-block-start

No handler stores `scroll-padding-block-start`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16558`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-padding-block-start: 12px` in horizontal-tb and demands the snap window's top edge move inward by 12px, the block-start side (`internal/layout/style_logical_box.go:112-114`).

- [~] Reason: no scrollport, and no handler stores `scroll-padding-block-start`. Owner: this ledger. Next gate: a scrollport whose block-start snap inset is 12px for `scroll-padding-block-start: 12px`.

### scroll-padding-bottom

No handler stores `scroll-padding-bottom`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16574`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-padding-bottom: auto` and demands a used inset of 0 while the specified value stays the keyword `auto`. `scroll-padding-bottom: 0px` is a different specified value with the same used inset. The test fails if `auto` is dropped.

- [~] Reason: no scrollport, and no handler stores `scroll-padding-bottom`. Owner: this ledger. Next gate: a scrollport that treats `scroll-padding-bottom: auto` as a 0 used inset and keeps the keyword `auto`.

### scroll-padding-inline

No handler stores `scroll-padding-inline`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16590`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-padding-inline: 8px 4px` in horizontal-tb ltr and demands an 8px inset on the left and a 4px inset on the right. In rtl the 8px moves to the right. One value sets both inline edges.

- [~] Reason: no scrollport, and no handler stores `scroll-padding-inline`. Owner: this ledger. Next gate: a scrollport whose inline snap insets are 8px then 4px for `scroll-padding-inline: 8px 4px` in horizontal-tb ltr.

### scroll-padding-inline-end

No handler stores `scroll-padding-inline-end`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16606`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-padding-inline-end: 4px` in horizontal-tb ltr and demands the snap window's right edge move inward by 4px. The length differs from inline-start so a copied value fails. In rtl the 4px is on the left.

- [~] Reason: no scrollport, and no handler stores `scroll-padding-inline-end`. Owner: this ledger. Next gate: a scrollport whose inline-end snap inset is 4px for `scroll-padding-inline-end: 4px`.

### scroll-padding-inline-start

No handler stores `scroll-padding-inline-start`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16622`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-padding-inline-start: 8px` with `direction: rtl` in horizontal-tb and demands the snap window's right edge move inward by 8px. In ltr the same declaration insets the left edge (`internal/layout/style_logical_box.go:82-87`).

- [~] Reason: no scrollport, and no handler stores `scroll-padding-inline-start`. Owner: this ledger. Next gate: a scrollport whose inline-start snap inset is 8px on the right in rtl for `scroll-padding-inline-start: 8px`.

### scroll-padding-left

No handler stores `scroll-padding-left`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16638`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-padding-left: 8px` and demands the snap window's left edge move inward by 8px. A percentage case is covered by `scroll-padding-right`. This side is the physical left edge in every writing mode.

- [~] Reason: no scrollport, and no handler stores `scroll-padding-left`. Owner: this ledger. Next gate: a scrollport whose left snap inset is 8px for `scroll-padding-left: 8px`.

### scroll-padding-right

No handler stores `scroll-padding-right`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16654`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-padding-right: 10%` and demands the snap window's right edge move inward by 10 percent of the scrollport width. The test fails if 10% is treated as 10px.

- [~] Reason: no scrollport, and no handler stores `scroll-padding-right`. Owner: this ledger. Next gate: a scrollport whose right snap inset is 10 percent of its width for `scroll-padding-right: 10%`.

### scroll-padding-top

No handler stores `scroll-padding-top`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16670`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-padding-top: 20px` and demands the snap window's top edge move inward by 20px. `scroll-padding-top: auto` is a separate case that computes to 0 and keeps the keyword.

- [~] Reason: no scrollport, and no handler stores `scroll-padding-top`. Owner: this ledger. Next gate: a scrollport whose top snap inset is 20px for `scroll-padding-top: 20px`.

### scroll-snap-align

No handler stores `scroll-snap-align`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16686`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-snap-align: start center` and demands the target's block-start edge and inline center line up with the snap window. One keyword, `scroll-snap-align: end`, sets both axes to `end`. `scroll.go` already says choosing that snap offset is a scrolling pass this engine does not have (`internal/layout/scroll.go:20-26`).

- [~] Reason: no scrollport, and no handler stores `scroll-snap-align`. Owner: this ledger. Next gate: a scrollport that aligns a target's block-start edge and inline center for `scroll-snap-align: start center`.

### scroll-snap-stop

No handler stores `scroll-snap-stop`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16702`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-snap-stop: always` and demands a fast scroll stop on this target instead of passing it. `scroll-snap-stop: normal` may pass the target when the scroll is fast.

- [~] Reason: no scrollport, and no handler stores `scroll-snap-stop`. Owner: this ledger. Next gate: a scrollport that stops on this target during a fast scroll when the declaration is `scroll-snap-stop: always`.

### scroll-snap-type

No handler stores `scroll-snap-type`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16718`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-snap-type: y mandatory` and demands the scrollport rest on a snap position on the y axis. `scroll-snap-type: x proximity` snaps on x only when the offset is already near a target. `scroll-snap-type: none` demands no snapping. Choosing that offset is the pass `scroll.go` leaves out (`internal/layout/scroll.go:20-26`).

- [~] Reason: no scrollport, and no handler stores `scroll-snap-type`. Owner: this ledger. Next gate: a scrollport that must rest on a y snap position for `scroll-snap-type: y mandatory`.

### scroll-target-group

No handler stores `scroll-target-group`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16734`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scroll-target-group: auto` and demands this element and its descendants share one snap group. `scroll-target-group: none` demands each target snap on its own. The test fails if `auto` leaves the descendants ungrouped.

- [~] Reason: no scrollport, and no handler stores `scroll-target-group`. Owner: this ledger. Next gate: a scrollport that groups descendant snap targets together for `scroll-target-group: auto`.

### scroll-timeline

`applyTimelineProps` expands `scroll-timeline` into one name and one axis (`internal/layout/timeline_style.go:302-325`, handler at `359-369`). A missing axis resets to `block` (`timeline_style.go:201-213`), which exports as `y` in horizontal writing mode (`timeline_style.go:190-196`). `layout.Box.Timelines` carries kind `scroll` and an empty scroller id (`layout/timeline.go:15-16`, `layout/timeline.go:22-31`). The catalog row is still unsupported with an empty handlers list (`testdata/css/catalog/properties.json:16750`).

What to expect: A future host-timeline test sets `scroll-timeline: --page y` and demands progress for `--page` run from 0 at the start of the page scroll to 1 at the end, on the y axis. `TestBoxTimelines` already checks the exported name, kind `scroll`, and axis `y` (`layout/timeline_test.go:42-51`). Progress is the missing piece, because the engine owns no clock and no scroll offset (`layout/timeline.go:11-13`).

- [x] Engine: `scroll-timeline` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxScrollAndViewTimelineCascade` (`layout/timeline_scope_test.go:166`), passing on 2026-11-16 with `go test ./layout -run TestBoxScrollAndViewTimelineCascade -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: progress from scroll offset, view position, or pointer position happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `scroll-timeline` on the scroll page.

### scroll-timeline-axis

`applyTimelineAxisProp` writes one axis keyword to `ResolvedStyle.ScrollTimelineAxis` (`internal/layout/timeline_style.go:474-475`, field at `internal/layout/style.go:619`). The catalog row is still unsupported with an empty handlers list (`testdata/css/catalog/properties.json:16766`).

What to expect: A future host-timeline test sets `scroll-timeline-axis: x` beside `scroll-timeline-name: --page` and demands the `--page` progress move with horizontal scroll. The initial axis is `block` (`internal/layout/style.go:835`), which is y in horizontal-tb, so a test fails if `x` is left as `block`. A comma list of axes is outside the one-axis store.

- [x] Engine: `scroll-timeline-axis` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxScrollAndViewTimelineCascade` (`layout/timeline_scope_test.go:166`), passing on 2026-11-16 with `go test ./layout -run TestBoxScrollAndViewTimelineCascade -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: progress from scroll offset, view position, or pointer position happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `scroll-timeline-axis` on the scroll page.

### scroll-timeline-name

`applyTimelineNameProp` writes one dashed name to `ResolvedStyle.ScrollTimelineName` (`internal/layout/timeline_style.go:440-441`, field at `internal/layout/style.go:618`) and, since 2026-11-16, accepts a comma list of names (`parseTimelineName` in `internal/layout/timeline_style.go`). The catalog row is still unsupported with an empty handlers list (`testdata/css/catalog/properties.json:16782`).

What to expect: A future host-timeline test sets `scroll-timeline-name: --page` and demands a scroll timeline of that name on the initial block axis, y in horizontal-tb. `scroll-timeline-name: none` demands no scroll timeline on the box. `scroll-timeline-name: --a, --b` now exports two scroll timelines in that order (`layout/timeline_list_test.go:31`). A future product that wants two names has to pass that list through.

- [x] Engine: `scroll-timeline-name` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxScrollAndViewTimelineCascade` (`layout/timeline_scope_test.go:166`), passing on 2026-11-16 with `go test ./layout -run TestBoxScrollAndViewTimelineCascade -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: progress from scroll offset, view position, or pointer position happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `scroll-timeline-name` on the scroll page.

### scrollbar-color

No handler stores `scrollbar-color`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16798`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scrollbar-color: red blue` and demands a red thumb on a blue track. Dropping either color fails the test. `scrollbar-color: auto` demands the platform colors. The display list does not paint that chrome until the scrollport product does.

- [~] Reason: no scrollport, and no handler stores `scrollbar-color`. Owner: this ledger. Next gate: a scrollport that paints a red thumb on a blue track for `scrollbar-color: red blue`.

### scrollbar-gutter

No handler stores `scrollbar-gutter`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16814`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scrollbar-gutter: stable both-edges` and demands the content box stay inset on both inline edges by the gutter, including when the bar is not visible, so the line length does not jump. `scrollbar-gutter: auto` demands no reserved gutter. `stable` alone reserves the one edge where the bar would appear.

- [~] Reason: no scrollport, and no handler stores `scrollbar-gutter`. Owner: this ledger. Next gate: a scrollport that reserves gutter space on both inline edges for `scrollbar-gutter: stable both-edges`.

### scrollbar-width

No handler stores `scrollbar-width`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:16830`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `scrollbar-width: thin` and demands a thin bar. `scrollbar-width: none` demands no bar and no gutter that comes from the width. `scrollbar-width: auto` demands the platform thickness. Static paint of the content box stays as laid out until that scrollport exists.

- [~] Reason: no scrollport, and no handler stores `scrollbar-width`. Owner: this ledger. Next gate: a scrollport that uses a thin bar for `scrollbar-width: thin` and no bar for `scrollbar-width: none`.

## Phase 3: pointer, caret, form

A pointer is the mouse or touch point. A caret is the text cursor. This module has neither, and it has no form chrome that changes size, masking, or window position. `caret-color` stays in phase 0.

### caret

No handler stores `caret`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:5576`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `caret: underscore manual` and demands an underscore caret that moves only when the host steps it. `caret: bar auto` demands a thin bar that blinks on the host caret clock. `caret-color` stays in phase 0 and is not a work row here.

- [~] Reason: no pointer, caret, or form chrome, and no handler stores `caret`. Owner: this ledger. Next gate: a pointer product that draws an underscore caret and steps it only on request for `caret: underscore manual`.

### caret-animation

No handler stores `caret-animation`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:5592`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `caret-animation: manual` and demands the caret stay still until the host steps it. `caret-animation: auto` demands the host blink the caret on its own clock. The test fails if `manual` blinks by itself.

- [~] Reason: no pointer, caret, or form chrome, and no handler stores `caret-animation`. Owner: this ledger. Next gate: a pointer product that blinks the caret only when asked for `caret-animation: manual`.

### caret-shape

No handler stores `caret-shape`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:5629`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `caret-shape: underscore` and demands the caret be an underline. `caret-shape: block` demands a block that covers the character cell. `caret-shape: bar` demands a thin vertical bar. `caret-shape: auto` leaves the shape to the host.

- [~] Reason: no pointer, caret, or form chrome, and no handler stores `caret-shape`. Owner: this ledger. Next gate: a pointer product that draws an underline caret for `caret-shape: underscore`.

### field-sizing

No handler stores `field-sizing`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:8485`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `field-sizing: content` on an input with `width: 40px` and the value `hello`, and demands the used width grow with that text past the empty control. `field-sizing: fixed` demands the used width stay 40px.

- [~] Reason: no pointer, caret, or form chrome, and no handler stores `field-sizing`. Owner: this ledger. Next gate: a pointer product that grows the control with its text for `field-sizing: content` and keeps `width: 40px` for `field-sizing: fixed`.

### input-security

No handler stores `input-security`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:10915`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `input-security: none` on `<input type="password">` and demands the characters be visible. `input-security: auto` on that control demands the characters stay masked. The test fails if `none` still masks them.

- [~] Reason: no pointer, caret, or form chrome, and no handler stores `input-security`. Owner: this ledger. Next gate: a pointer product that shows password characters for `input-security: none` and masks them for `input-security: auto`.

### interactivity

Until 2026-11-16 no handler stored `interactivity`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:11103`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `interactivity: inert` and demands pointer hits and focus skip the element and its descendants. `interactivity: auto` demands normal hits and focus. The test fails if a click on a child of an inert element still focuses it.

- [x] Engine: `interactivity` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxInteractivity` (`layout/interest_test.go:112`), passing on 2026-11-16 with `go test ./layout -run TestBoxInteractivity -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: skipping pointer hits and focus on an inert subtree happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `interactivity` on the scroll page.

### interest-delay

Until 2026-11-16 no handler stored `interest-delay`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:11119`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `interest-delay: 200ms 500ms` and demands interest start 200 milliseconds after the pointer enters and end 500 milliseconds after it leaves. One value, `interest-delay: 200ms`, demands 200 milliseconds on both sides. The catalog grammar is one or two `interest-delay-start` values.

- [x] Engine: `interest-delay` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxInterestDelayShorthand` (`layout/interest_test.go:17`), passing on 2026-11-16 with `go test ./layout -run TestBoxInterestDelayShorthand -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: the hover timer happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `interest-delay` on the scroll page.

### interest-delay-end

Until 2026-11-16 no handler stored `interest-delay-end`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:11135`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `interest-delay-end: 500ms` and demands interest clear 500 milliseconds after the pointer leaves. `interest-delay-end: normal` demands the host's usual end delay. The test fails if this value changes the start delay.

- [x] Engine: `interest-delay-end` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxInterestDelayLonghands` (`layout/interest_test.go:53`), passing on 2026-11-16 with `go test ./layout -run TestBoxInterestDelayLonghands -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: the hover timer happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `interest-delay-end` on the scroll page.

### interest-delay-start

Until 2026-11-16 no handler stored `interest-delay-start`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:11151`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `interest-delay-start: 200ms` and demands interest begin 200 milliseconds after the pointer enters, not on the first frame of the hover. `interest-delay-start: normal` demands the host's usual start delay.

- [x] Engine: `interest-delay-start` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxInterestDelayLonghands` (`layout/interest_test.go:53`), passing on 2026-11-16 with `go test ./layout -run TestBoxInterestDelayLonghands -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: the hover timer happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `interest-delay-start` on the scroll page.

### nav-down

No handler stores `nav-down`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:13137`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `nav-down: #next` and demands the Down key move focus to the element with id `next`. `nav-down: #next current` searches inside the current frame. `nav-down: auto` picks the geometric neighbor.

- [~] Reason: no pointer, caret, or form chrome, and no handler stores `nav-down`. Owner: this ledger. Next gate: a pointer product that moves focus to `#next` on the Down key for `nav-down: #next`.

### nav-left

No handler stores `nav-left`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:13153`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `nav-left: #side` and demands the Left key move focus to the element with id `side`. The target id differs from the other three nav longhands so a copied value fails.

- [~] Reason: no pointer, caret, or form chrome, and no handler stores `nav-left`. Owner: this ledger. Next gate: a pointer product that moves focus to `#side` on the Left key for `nav-left: #side`.

### nav-right

No handler stores `nav-right`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:13169`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `nav-right: #more root` and demands the Right key search for id `more` from the root. `nav-right: auto` picks the geometric neighbor to the right.

- [~] Reason: no pointer, caret, or form chrome, and no handler stores `nav-right`. Owner: this ledger. Next gate: a pointer product that searches from the root for `#more` on the Right key for `nav-right: #more root`.

### nav-up

No handler stores `nav-up`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:13185`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `nav-up: #banner` and demands the Up key move focus to the element with id `banner`. `nav-up: auto` picks the geometric neighbor above.

- [~] Reason: no pointer, caret, or form chrome, and no handler stores `nav-up`. Owner: this ledger. Next gate: a pointer product that moves focus to `#banner` on the Up key for `nav-up: #banner`.

### slider-orientation

No handler stores `slider-orientation`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:17039`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `slider-orientation: bottom-to-top` on a range control and demands 0 at the bottom and the high value at the top. `slider-orientation: left-to-right` demands 0 on the left. `slider-orientation: auto` follows the writing mode.

- [~] Reason: no pointer, caret, or form chrome, and no handler stores `slider-orientation`. Owner: this ledger. Next gate: a pointer product that lays a range control from bottom to top for `slider-orientation: bottom-to-top`.

### spatial-navigation-action

No handler stores `spatial-navigation-action`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:17055`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `spatial-navigation-action: scroll` and demands an arrow key scroll the port and leave focus where it is. `spatial-navigation-action: focus` demands the key move focus. `spatial-navigation-action: auto` lets the host choose.

- [~] Reason: no pointer, caret, or form chrome, and no handler stores `spatial-navigation-action`. Owner: this ledger. Next gate: a pointer product that scrolls on an arrow key and leaves focus put for `spatial-navigation-action: scroll`.

### spatial-navigation-contain

No handler stores `spatial-navigation-contain`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:17071`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `spatial-navigation-contain: contain` on a dialog and demands Up, Down, Left, and Right stay inside that dialog. `spatial-navigation-contain: auto` may leave the dialog for a neighbor outside it.

- [~] Reason: no pointer, caret, or form chrome, and no handler stores `spatial-navigation-contain`. Owner: this ledger. Next gate: a pointer product that keeps spatial navigation inside the element for `spatial-navigation-contain: contain`.

### spatial-navigation-function

No handler stores `spatial-navigation-function`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:17087`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `spatial-navigation-function: grid` and demands the next target be chosen by row and column. `spatial-navigation-function: normal` chooses by geometric distance. The test fails if `grid` picks the nearest box that sits off the row.

- [~] Reason: no pointer, caret, or form chrome, and no handler stores `spatial-navigation-function`. Owner: this ledger. Next gate: a pointer product that picks the next target by row and column for `spatial-navigation-function: grid`.

### window-drag

No handler stores `window-drag`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:20255`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `window-drag: move` on a title bar and demands a drag on that element move the host window. `window-drag: none` demands the drag leave the window where it is.

- [~] Reason: no pointer, caret, or form chrome, and no handler stores `window-drag`. Owner: this ledger. Next gate: a pointer product that moves the host window on a drag for `window-drag: move`.

## Phase 4: motion and view timelines

A motion path moves a box along a curve. A view timeline measures how far an element has crossed the scroll window. View-timeline names and insets are already exported. Progress, `timeline-scope`, scroll anchoring, and motion placement are not. `will-change` is a hint to the host.

### offset

No handler stores `offset`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:13270`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future motion test sets `offset: path("M 0 0 L 100 0") 50% / center` and demands the border box sit 50px along that path, with the center of the box on the path point. The shorthand writes path, distance, and anchor. `layout.DisplayList` today returns the static border box (`layout/displaylist.go:257`).

- [~] Reason: no motion-path placement, and no handler stores `offset`. Owner: this ledger. Next gate: a host motion placement that puts the border box 50px along `path("M 0 0 L 100 0")` with its center on the path for `offset: path("M 0 0 L 100 0") 50% / center`.

### offset-anchor

No handler stores `offset-anchor`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:13286`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future motion test sets `offset-anchor: top left` and demands the box's top-left corner sit on the path point. `offset-anchor: center` demands the center sit on that point. `offset-anchor: auto` uses the default anchor. The test fails if `top left` and `center` land on the same corner.

- [~] Reason: no motion-path placement, and no handler stores `offset-anchor`. Owner: this ledger. Next gate: a host motion placement that pins the box's top-left corner to the path point for `offset-anchor: top left`.

### offset-distance

No handler stores `offset-distance`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:13302`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future motion test sets `offset-distance: 50%` with `offset-path: path("M 0 0 L 100 0")` and demands the box sit halfway along that path. `offset-distance: 0px` demands the box sit at the path start. The test fails if the box stays at the static layout position.

- [~] Reason: no motion-path placement, and no handler stores `offset-distance`. Owner: this ledger. Next gate: a host motion placement that puts the box halfway along its path for `offset-distance: 50%`.

### offset-path

No handler stores `offset-path`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:13318`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future motion test sets `offset-path: path("M 0 0 L 100 0")` and demands the box travel that line. `offset-path: none` demands the box stay at its static position. A ray or a shape is the other legal path form in the catalog grammar `none | <offset-path> || <coord-box>`.

- [~] Reason: no motion-path placement, and no handler stores `offset-path`. Owner: this ledger. Next gate: a host motion placement that moves the box along `path("M 0 0 L 100 0")` for `offset-path: path("M 0 0 L 100 0")`.

### offset-position

No handler stores `offset-position`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:13334`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future motion test sets `offset-position: left top` and demands the path start at the box's left-top position when the path is `none`. `offset-position: normal` demands the usual static position. `offset-position: auto` takes the position from the offset path.

- [~] Reason: no motion-path placement, and no handler stores `offset-position`. Owner: this ledger. Next gate: a host motion placement that starts from the left-top position for `offset-position: left top`.

### offset-rotate

No handler stores `offset-rotate`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:13350`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future motion test sets `offset-rotate: auto 90deg` and demands the box point along the path and then turn another 90 degrees. `offset-rotate: reverse` demands the path direction plus 180 degrees. `offset-rotate: 0deg` demands no extra turn. The test fails if `auto 90deg` leaves the box at the path's unrotated angle.

- [~] Reason: no motion-path placement, and no handler stores `offset-rotate`. Owner: this ledger. Next gate: a host motion placement that points the box along the path and then turns it 90 degrees for `offset-rotate: auto 90deg`.

### overflow-anchor

No handler stores `overflow-anchor`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:13663`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future scrollport test sets `overflow-anchor: none` on a block above the visible window and demands the scroll offset stay put when that block grows taller. `overflow-anchor: auto` demands the offset shift so the content the reader was looking at stays in view.

- [~] Reason: no scrollport shifts the offset when content above the window grows, and no handler stores `overflow-anchor`. Owner: this ledger. Next gate: a scrollport that leaves the offset put when content above the window grows, for `overflow-anchor: none`.

### timeline-scope

Until 2026-11-16 no handler stored `timeline-scope`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:18915`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future host-timeline test sets `timeline-scope: --card` and demands descendants resolve `animation-timeline: --card` to this element's timeline. `timeline-scope: all` demands every name declared here be visible to descendants. `timeline-scope: none` demands descendants not see those names. The export comment says `timeline-scope` is not implemented, so a name that needs it stays unresolved (`layout/timeline.go:16-17`).

- [x] Engine: `timeline-scope` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxTimelineScope` (`layout/timeline_scope_test.go:21`), passing on 2026-11-16 with `go test ./layout -run TestBoxTimelineScope -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: lookup of a dashed name across the scope happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `timeline-scope` on the scroll page.

### view-timeline

`applyTimelineProps` expands `view-timeline` into one name, one axis, and one or two insets (`internal/layout/timeline_style.go:328-353`, handler at `359-369`). A missing axis stays `block` (`internal/layout/style.go:836`). `layout.Box.Timelines` exports kind `view` with the subject id and an empty scroller id (`layout/timeline.go:27-31`). The catalog row is still unsupported with an empty handlers list (`testdata/css/catalog/properties.json:19671`).

What to expect: A future host-timeline test sets `view-timeline: --card block 20px 10px` and demands progress 0 when the subject's border box, outset by the 20px start inset, first meets the scroll window, and progress 1 when the 10px end inset has passed. `TestBoxTimelines` already checks the name `--card`, kind `view`, axis `y` in horizontal writing mode, the subject id, and insets 20 and 10 CSS pixels (`layout/timeline_test.go:54-70`). Progress needs a scrollport.

- [x] Engine: `view-timeline` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxScrollAndViewTimelineCascade` (`layout/timeline_scope_test.go:166`), passing on 2026-11-16 with `go test ./layout -run TestBoxScrollAndViewTimelineCascade -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: progress from scroll offset, view position, or pointer position happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `view-timeline` on the scroll page.

### view-timeline-axis

`applyTimelineAxisProp` writes one axis keyword to `ResolvedStyle.ViewTimelineAxis` (`internal/layout/timeline_style.go:476-477`, field at `internal/layout/style.go:621`). The catalog row is still unsupported with an empty handlers list (`testdata/css/catalog/properties.json:19687`).

What to expect: A future host-timeline test sets `view-timeline-axis: inline` beside `view-timeline-name: --card` and demands `--card` progress follow the inline axis, x in horizontal-tb and y in vertical writing modes (`internal/layout/timeline_style.go:185-196`). The initial axis is `block` (`internal/layout/style.go:836`). A second axis in one declaration is rejected.

- [x] Engine: `view-timeline-axis` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxScrollAndViewTimelineCascade` (`layout/timeline_scope_test.go:166`), passing on 2026-11-16 with `go test ./layout -run TestBoxScrollAndViewTimelineCascade -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: progress from scroll offset, view position, or pointer position happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `view-timeline-axis` on the scroll page.

### view-timeline-inset

`applyTimelineProps` stores one or two insets on `ViewTimelineInsetStart` and `ViewTimelineInsetEnd` (`internal/layout/style.go:623-624`, `internal/layout/timeline_style.go:484`). The public box reports them in CSS pixels (`layout/timeline.go:33-35`). The catalog row is still unsupported with an empty handlers list (`testdata/css/catalog/properties.json:19703`).

What to expect: A future host-timeline test sets `view-timeline-inset: 20px 10px` and demands the view range start 20px before the subject and end 10px after it. One value sets both edges (`internal/layout/timeline_style.go:270-285`). `view-timeline-inset: -10px 20px` is legal and starts the range 10px outside the scroll window (`timeline_style.go:291-293`). `TestBoxTimelines` already reads 20 and 10 CSS pixels (`layout/timeline_test.go:69-70`). Progress across that range needs a scrollport.

- [x] Engine: `view-timeline-inset` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxScrollAndViewTimelineCascade` (`layout/timeline_scope_test.go:166`), passing on 2026-11-16 with `go test ./layout -run TestBoxScrollAndViewTimelineCascade -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: progress from scroll offset, view position, or pointer position happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `view-timeline-inset` on the scroll page.

### view-timeline-name

`applyTimelineNameProp` writes one dashed name to `ResolvedStyle.ViewTimelineName` (`internal/layout/timeline_style.go:442-443`, field at `internal/layout/style.go:620`) and, since 2026-11-16, accepts a comma list of names (`parseTimelineName` in `internal/layout/timeline_style.go`). The catalog row is still unsupported with an empty handlers list (`testdata/css/catalog/properties.json:19719`).

What to expect: A future host-timeline test sets `view-timeline-name: --card` and demands a view timeline of that name whose subject is this element (`layout/timeline.go:27-28`). `view-timeline-name: none` demands no view timeline. A comma list is rejected (`internal/layout/timeline_style.go:144-146`).

- [x] Engine: `view-timeline-name` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxScrollAndViewTimelineCascade` (`layout/timeline_scope_test.go:166`), passing on 2026-11-16 with `go test ./layout -run TestBoxScrollAndViewTimelineCascade -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: progress from scroll offset, view position, or pointer position happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `view-timeline-name` on the scroll page.

### will-change

No handler stores `will-change`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:20239`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future host test sets `will-change: transform` and demands the host keep a transform layer ready for that element. The border box from `layout.DisplayList` stays where layout put it. `will-change: auto` demands no extra layer. `will-change: scroll-position` demands a scroll layer, which still needs a scrollport.

- [~] Reason: no host layer hint is exported, and no handler stores `will-change`. Owner: this ledger. Next gate: a host that prepares a transform layer for `will-change: transform` while the display-list border box stays put.

## Phase 5: event and pointer timelines

These names wait for a click, a key, or a pointer position, then arm a host timeline. This module has no event loop and no pointer. `animation-trigger` is already in the animation export and is not replanned here. Pointer timelines were open rows in the old scroll checklist. Those rows are superseded here.

### event-trigger

Until 2026-11-16 no handler stored `event-trigger`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:8437`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future host-timeline test sets `event-trigger: --press click` and demands a click on the element activate the trigger named `--press`. `event-trigger: none` demands no event trigger. The shorthand pairs a name with a source. This module has no event loop.

- [x] Engine: `event-trigger` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxEventTriggerShorthand` (`layout/event_trigger_test.go:11`), passing on 2026-11-16 with `go test ./layout -run TestBoxEventTriggerShorthand -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: watching click and key events and arming the trigger happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `event-trigger` on the scroll page.

### event-trigger-name

Until 2026-11-16 no handler stored `event-trigger-name`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:8453`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future host-timeline test sets `event-trigger-name: --press` and demands the trigger answer to the name `--press`. `event-trigger-name: none` demands no name. A second name in the list, `event-trigger-name: --press, --key`, demands both names.

- [x] Engine: `event-trigger-name` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxEventTriggerLonghands` (`layout/event_trigger_test.go:66`), passing on 2026-11-16 with `go test ./layout -run TestBoxEventTriggerLonghands -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: watching click and key events and arming the trigger happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `event-trigger-name` on the scroll page.

### event-trigger-source

Until 2026-11-16 no handler stored `event-trigger-source`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:8469`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future host-timeline test sets `event-trigger-source: click / keydown` and demands the trigger start on `click` and end on `keydown`. The catalog grammar puts the end events after the slash. `event-trigger-source: none` demands no events. The test fails if a click does not start the trigger.

- [x] Engine: `event-trigger-source` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxEventTriggerLonghands` (`layout/event_trigger_test.go:66`), passing on 2026-11-16 with `go test ./layout -run TestBoxEventTriggerLonghands -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: watching click and key events and arming the trigger happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `event-trigger-source` on the scroll page.

### image-animation

No handler stores `image-animation`. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:10703`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future host-timeline test sets `image-animation: paused` on an animated image and demands the image stay on its current frame while the host clock runs. `image-animation: running` demands the frames advance with that clock. `image-animation: stopped` demands the first frame. This module does not advance time (`layout/layout.go:96`).

- [~] Reason: this module does not step image frames, and no handler stores `image-animation`. Owner: this ledger. Next gate: a host clock that holds an animated image on its current frame for `image-animation: paused` and advances frames for `image-animation: running`.

### pointer-timeline

Until 2026-11-16 no handler stored `pointer-timeline`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:14774`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `pointer-timeline: --ptr x` and demands progress for `--ptr` follow the pointer's x position from 0 at the left of the element to 1 at the right. The shorthand writes the name and an optional axis. This module has no pointer, so it exports no pointer progress.

- [x] Engine: `pointer-timeline` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxPointerTimeline` (`layout/timeline_scope_test.go:88`), passing on 2026-11-16 with `go test ./layout -run TestBoxPointerTimeline -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: progress from scroll offset, view position, or pointer position happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `pointer-timeline` on the scroll page.

### pointer-timeline-axis

Until 2026-11-16 no handler stored `pointer-timeline-axis`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:14790`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `pointer-timeline-axis: inline` beside `pointer-timeline-name: --ptr` and demands progress follow the inline axis, x in horizontal-tb. `pointer-timeline-axis: y` demands vertical pointer movement. The test fails if `inline` is read as the block axis.

- [x] Engine: `pointer-timeline-axis` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxPointerTimeline` (`layout/timeline_scope_test.go:88`), passing on 2026-11-16 with `go test ./layout -run TestBoxPointerTimeline -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: progress from scroll offset, view position, or pointer position happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `pointer-timeline-axis` on the scroll page.

### pointer-timeline-name

Until 2026-11-16 no handler stored `pointer-timeline-name`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:14806`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future pointer test sets `pointer-timeline-name: --ptr` and demands a pointer timeline of that name. `pointer-timeline-name: none` demands no pointer timeline. A list of names demands one progress track per name. No Go file stores this name today.

- [x] Engine: `pointer-timeline-name` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxPointerTimeline` (`layout/timeline_scope_test.go:88`), passing on 2026-11-16 with `go test ./layout -run TestBoxPointerTimeline -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: progress from scroll offset, view position, or pointer position happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `pointer-timeline-name` on the scroll page.

### timeline-trigger

Until 2026-11-16 no handler stored `timeline-trigger`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:18931`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future host-timeline test sets `timeline-trigger: --play scroll() 0% 100%` and demands the trigger named `--play` arm while the anonymous scroll timeline is between 0% and 100%. The part after an optional slash would be the active range. `timeline-trigger: none` demands no trigger.

- [x] Engine: `timeline-trigger` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxTimelineTriggerShorthand` (`layout/trigger_test.go:21`), passing on 2026-11-16 with `go test ./layout -run TestBoxTimelineTriggerShorthand -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: comparing timeline progress with the activation and active ranges happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `timeline-trigger` on the scroll page.

### timeline-trigger-activation-range

Until 2026-11-16 no handler stored `timeline-trigger-activation-range`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:18947`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future host-timeline test sets `timeline-trigger-activation-range: entry 0% entry 100%` and demands the trigger arm at the start of the entry range and finish arming at the end of that range. One value sets the start and leaves the end at its initial value.

- [x] Engine: `timeline-trigger-activation-range` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxTimelineTriggerRangeShorthands` (`layout/trigger_test.go:197`), passing on 2026-11-16 with `go test ./layout -run TestBoxTimelineTriggerRangeShorthands -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: comparing timeline progress with the activation and active ranges happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `timeline-trigger-activation-range` on the scroll page.

### timeline-trigger-activation-range-end

Until 2026-11-16 no handler stored `timeline-trigger-activation-range-end`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:18963`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future host-timeline test sets `timeline-trigger-activation-range-end: entry 100%` and demands arming finish at the end of the entry range. `timeline-trigger-activation-range-end: normal` demands the range's own end. The test fails if this value changes the start.

- [x] Engine: `timeline-trigger-activation-range-end` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxTimelineTriggerLonghands` (`layout/trigger_test.go:99`), passing on 2026-11-16 with `go test ./layout -run TestBoxTimelineTriggerLonghands -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: comparing timeline progress with the activation and active ranges happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `timeline-trigger-activation-range-end` on the scroll page.

### timeline-trigger-activation-range-start

Until 2026-11-16 no handler stored `timeline-trigger-activation-range-start`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:18979`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future host-timeline test sets `timeline-trigger-activation-range-start: entry 0%` and demands arming begin at the start of the entry range. `timeline-trigger-activation-range-start: 25%` demands arming begin at 25 percent of the timeline. The test fails if 25% is read as 25px.

- [x] Engine: `timeline-trigger-activation-range-start` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxTimelineTriggerLonghands` (`layout/trigger_test.go:99`), passing on 2026-11-16 with `go test ./layout -run TestBoxTimelineTriggerLonghands -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: comparing timeline progress with the activation and active ranges happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `timeline-trigger-activation-range-start` on the scroll page.

### timeline-trigger-active-range

Until 2026-11-16 no handler stored `timeline-trigger-active-range`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:18995`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future host-timeline test sets `timeline-trigger-active-range: contain 0% contain 100%` and demands the trigger stay active from the start of the contain range through the end of it. This range is the interval where the trigger is on, after the activation range has armed it.

- [x] Engine: `timeline-trigger-active-range` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxTimelineTriggerRangeShorthands` (`layout/trigger_test.go:197`), passing on 2026-11-16 with `go test ./layout -run TestBoxTimelineTriggerRangeShorthands -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: comparing timeline progress with the activation and active ranges happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `timeline-trigger-active-range` on the scroll page.

### timeline-trigger-active-range-end

Until 2026-11-16 no handler stored `timeline-trigger-active-range-end`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:19011`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future host-timeline test sets `timeline-trigger-active-range-end: contain 100%` and demands the trigger turn off at the end of the contain range. `timeline-trigger-active-range-end: auto` demands the active end match the activation end. `normal` demands the range's own end.

- [x] Engine: `timeline-trigger-active-range-end` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxTimelineTriggerLonghands` (`layout/trigger_test.go:99`), passing on 2026-11-16 with `go test ./layout -run TestBoxTimelineTriggerLonghands -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: comparing timeline progress with the activation and active ranges happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `timeline-trigger-active-range-end` on the scroll page.

### timeline-trigger-active-range-start

Until 2026-11-16 no handler stored `timeline-trigger-active-range-start`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:19027`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future host-timeline test sets `timeline-trigger-active-range-start: contain 0%` and demands the trigger turn on at the start of the contain range. `timeline-trigger-active-range-start: auto` matches the activation start. The test fails if this start is copied from the activation-range start when the declarations differ.

- [x] Engine: `timeline-trigger-active-range-start` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxTimelineTriggerLonghands` (`layout/trigger_test.go:99`), passing on 2026-11-16 with `go test ./layout -run TestBoxTimelineTriggerLonghands -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: comparing timeline progress with the activation and active ranges happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `timeline-trigger-active-range-start` on the scroll page.

### timeline-trigger-name

Until 2026-11-16 no handler stored `timeline-trigger-name`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:19043`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future host-timeline test sets `timeline-trigger-name: --play` and demands the trigger answer to `--play`. `timeline-trigger-name: none` demands no trigger name. A list, `timeline-trigger-name: --play, --pause`, demands both names.

- [x] Engine: `timeline-trigger-name` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxTimelineTriggerNameAndSource` (`layout/trigger_test.go:143`), passing on 2026-11-16 with `go test ./layout -run TestBoxTimelineTriggerNameAndSource -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: comparing timeline progress with the activation and active ranges happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `timeline-trigger-name` on the scroll page.

### timeline-trigger-source

Until 2026-11-16 no handler stored `timeline-trigger-source`. It is stored now, and the catalog row is the other agent\'s to flip. The catalog row is unsupported and its handlers list is empty (`testdata/css/catalog/properties.json:19059`), and `applyStyleProp` finishes in `applyIgnoredGroup` (`internal/layout/style_cascade.go:1980`, `internal/layout/style_properties.go:1877`).

What to expect: A future host-timeline test sets `timeline-trigger-source: --page` and demands the trigger read the named timeline `--page`. `timeline-trigger-source: scroll()` demands the anonymous scroll timeline. `timeline-trigger-source: view()` demands the anonymous view timeline. `timeline-trigger-source: auto` picks the element's own timeline. `none` attaches the trigger to no timeline.

- [x] Engine: `timeline-trigger-source` is parsed, validated, cascaded, and exported on `layout.Box`. Proof: `TestBoxTimelineTriggerNameAndSource` (`layout/trigger_test.go:143`), passing on 2026-11-16 with `go test ./layout -run TestBoxTimelineTriggerNameAndSource -count=1 -p 1`. Page: `testdata/css/v012/scroll/page.html`.
- [~] Host: comparing timeline progress with the activation and active ranges happens in OwnFrame, not here (`../ownframe/examples/blinkless-scroll`). Reason: the engine owns no clock, scroll offset, or pointer. Owner: OwnFrame. Next gate: that example exercises `timeline-trigger-source` on the scroll page.

## Dependencies

- The public result stays `layout.DisplayList` (`layout/displaylist.go:257`). No row in this file adds a speaker, a scrollport, a pointer, or a clock inside this module.
- Animation and transition metadata stay on `layout.Box.Animations` and `layout.Box.Transitions` (`layout/layout.go:22-23`). This ledger does not replan them.
- Physical scroll-margin stays in `applyScrollProps` (`internal/layout/scroll.go:49`, registered at `internal/layout/style_cascade.go:1740`). The six logical scroll-margin names in phase 2 do not write those fields.
- Named scroll and view timelines stay on `layout.Box.Timelines` (`layout/timeline.go:19`). Progress still needs a host and a scrollport (`layout/timeline.go:11-13`).
- `timeline-scope` is the lookup limit named next to that export (`layout/timeline.go:16-17`). Phase 4 owns the parked row.
- An unlisted name is accepted and then discarded (`internal/layout/style_value_accept.go:938-939`, `internal/layout/style_properties.go:1877`). A future product replaces that drop for the name it implements.
- The parent index is `plans/v0.1.2/README.md`. The v0.0.3 speech and scroll checklists stay untouched. This file supersedes their open rows for these 102 names.
- No new direct module. The allowlist stays `github.com/go-text/typesetting` and `github.com/tdewolff/canvas`.
