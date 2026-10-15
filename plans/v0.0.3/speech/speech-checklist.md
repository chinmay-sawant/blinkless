# v0.0.3 - Speech

> **Parent:** `plans/v0.0.3/README.md` - index of these ledgers.
> **Status:** not started. Baseline rows record what the code already does. Implementation rows are open.
> **Estimated effort:** 19 CSS property names across 8 phases. No aliases in this set.

---

## Overview

Speech properties say how an element would be spoken: cues, pauses, rests, and voice settings. Blinkless does not speak. This ledger stores those values on the element in the drawing list so a host can.

The result is still `layout.Display` from `layout.DisplayList` (`layout/displaylist.go:129-138`). There is no page PNG, no PDF, no JavaScript, and no audio device.

`documentation/deferred.md:114` calls these 19 names a permanent print non-goal. `documentation/compatibility-matrix.md:1126` is the same list, under `A_speech_aural`. This file is the planning home. Do not edit `deferred.md` in the planning session. Phase 5 points that row here.

`css.Options.Media` does not accept `speech`. That stays true. Aural values are stored when they win the cascade for screen or print. A rule inside `@media speech` does not apply while the document is screen or print (`internal/css/media.go:72-73` requires the media type word to equal the document media).

## Executive summary

| Property | Catalog line | Status | Pinned syntax |
|---|---:|---|---|
| `cue-before` | 6366 | unsupported | `<url> <decibel>? | none` |
| `cue-after` | 6350 | unsupported | `<url> <decibel>? | none` |
| `cue` | 6334 | unsupported | `<'cue-before'> <'cue-after'>?` |
| `pause-before` | 12519 | unsupported | `<time [0s,inf]> | none | x-weak | weak | medium | strong | x-strong` |
| `pause-after` | 12503 | unsupported | `<time [0s,inf]> | none | x-weak | weak | medium | strong | x-strong` |
| `pause` | 12487 | unsupported | `<'pause-before'> <'pause-after'>?` |
| `rest-before` | 13101 | unsupported | `<time [0s,inf]> | none | x-weak | weak | medium | strong | x-strong` |
| `rest-after` | 13085 | unsupported | `<time [0s,inf]> | none | x-weak | weak | medium | strong | x-strong` |
| `rest` | 13069 | unsupported | `<'rest-before'> <'rest-after'>?` |
| `speak` | 14668 | unsupported | `auto | never | always` |
| `speak-as` | 14684 | unsupported | `normal | spell-out || digits || [ literal-punctuation | no-punctuation ]` |
| `voice-family` | 17221 | unsupported | `[ <voice-family-name> | <generic-voice> ]# | preserve` |
| `voice-balance` | 17189 | unsupported | `<number> | left | center | right | leftwards | rightwards` |
| `voice-duration` | 17205 | unsupported | `auto | <time [0s,inf]>` |
| `voice-pitch` | 17237 | unsupported | `<frequency [0Hz,inf]> && absolute | [ [ x-low | low | medium | high | x-high ] || [ <frequency [0Hz,inf]> | <semitones> | <percentage> ] ]` |
| `voice-range` | 17253 | unsupported | `<frequency [0Hz,inf]> && absolute | [ [ x-low | low | medium | high | x-high ] || [ <frequency [0Hz,inf]> | <semitones> | <percentage> ] ]` |
| `voice-rate` | 17269 | unsupported | `[ normal | x-slow | slow | medium | fast | x-fast ] || <percentage [0,inf]>` |
| `voice-stress` | 17285 | unsupported | `normal | strong | moderate | none | reduced` |
| `voice-volume` | 17301 | unsupported | `silent | [ [ x-soft | soft | medium | loud | x-loud ] || <decibel> ]` |

`inf` stands in for the infinity sign in the catalog. All 19 rows were read on 2026-10-15 from `testdata/css/catalog/properties.json` (`updated` `2026-10-13`, `properties.json:4`). Each is `unsupported`, with empty `handlers` and empty `tests`. A search of `*.go` found no `voice-family`, `speak-as`, or `cue-before` string. Header totals: 785 properties, 389 implemented, 19 partial, 369 unsupported, 8 intentionally ignored (`properties.json:11-18`).

## Phase 0: lock the contract and the baseline

### 0.1 What the code does today

- [x] The public pipeline is `html.Parse` (`html/parse.go:36`), `css.Apply` (`css/css.go:98`), and `layout.DisplayList` (`layout/displaylist.go:137`). `css.Options` is `WidthPx`, `HeightPx`, `Media`, `Extra`, `Focus`, `Hover`, and `Active` (`css/css.go:69-78`). `Media: "speech"` returns `ErrBadMedia` (`css/css.go:147-155`, `css/css_test.go:58-60`).
- [x] `layout.Box` has no speech fields (`layout/layout.go:10-21`). WASM `drawBoxJSON` copies id, tag, action, text, and the border box (`bindings/wasm/drawing_list.go:194-203`).
- [x] Unlisted names pass `supportedDeclaration` (`internal/layout/style_value_accept.go:135-138`) and are then dropped by `applyIgnoredGroup` (`internal/layout/style_properties.go:1854-1859`) after the cascade keeps them (`internal/layout/style_cascade.go:693-695`). `@supports` is false because no group returns true (`internal/layout/style_cascade.go:1915-1920`).
- [x] `@media speech` does not match a screen or print document. The type word must equal the document media (`internal/css/media.go:72-73`).
- [x] `internal/layout/layout.go` is 2142 lines (`scripts/file-size-allowlist.txt:9`). Do not add code there.
- [x] GitHub CI runs only `make test` and `make lint` (`.github/workflows/ci.yml:19-26`).

### 0.2 Contract to implement

- [ ] Store one speech record on the element. The host reads it from `layout.Box.Speech`. Times are milliseconds. Decibels stay numbers. Keywords stay keywords. URLs stay URL text.
- [ ] Do not fetch cue URLs. Do not route them through `layout.Options.Images` (`layout/layout.go:26-32`).
- [ ] Do not add `speech` to `css.Options.Media`. Declarations that are not hidden behind `@media speech` are the ones this ledger stores.
- [ ] Do not add a synthesizer. Phase 7 is that non-goal.
- [ ] New code lives in `internal/layout/speech_style.go` and `internal/layout/speech_payload.go`. Register `applySpeechProps` as one entry in `styleGroups` (`internal/layout/style_cascade.go:1642`). One call from `declarationValueAccepted` (`internal/layout/style_value_accept.go:139`). Do not add a switch body to `style_cascade.go` (1985 lines) or `style_value_accept.go` (1753 lines).
- [ ] A name is done only when `layout.DisplayList` shows the value and a test fails if it is removed (`testdata/css/catalog/properties.json:9`).

**Gate:** no synthesizer work and no media-type change are part of the open rows below.

## Phase 1: style fields and interning

Add fields on `ResolvedStyle` in `internal/layout/style.go`. Parse in `internal/layout/speech_style.go`. Run `go generate` from `internal/layout/style.go:2`. `TestStyleInternGeneratedFieldsComplete` fails if a field is missing (`internal/layout/style_intern_gen_test.go:14-27`). Shorthands do not get their own fields.

### Cue

- [ ] `cue-before` (`properties.json:6366`, syntax `<url> <decibel>? | none`). Store the used value on `ResolvedStyle`. Store `none`, or the URL text plus an optional decibel. Do not fetch the URL and do not pass it to `layout.Options.Images` (`layout/layout.go:26-32`). The public-box test uses `url("beep.wav")` and fails if that URL is missing. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `cue-after` (`properties.json:6350`, syntax `<url> <decibel>? | none`). Store the used value on `ResolvedStyle`. Same storage as cue-before, on the after field. The test uses a different URL so a swapped before and after fails. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `cue` (`properties.json:6334`, syntax `<'cue-before'> <'cue-after'>?`). No field of its own. One value writes cue-before and cue-after. Two values write before then after. The test uses two different URLs.
### Pause

- [ ] `pause-before` (`properties.json:12519`, syntax `<time [0s,inf]> | none | x-weak | weak | medium | strong | x-strong`). Store the used value on `ResolvedStyle`. Store `none`, a non-negative time in milliseconds, or one of those strength keywords. A negative time is rejected. The test uses `strong` and `200ms` in two cases. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `pause-after` (`properties.json:12503`, syntax `<time [0s,inf]> | none | x-weak | weak | medium | strong | x-strong`). Store the used value on `ResolvedStyle`. Same as pause-before, on the after field. The test uses `weak` so it cannot pass by copying the before value. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `pause` (`properties.json:12487`, syntax `<'pause-before'> <'pause-after'>?`). No field of its own. One value writes both sides. Two values write before then after. The test uses `strong weak`.
### Rest

- [ ] `rest-before` (`properties.json:13101`, syntax `<time [0s,inf]> | none | x-weak | weak | medium | strong | x-strong`). Store the used value on `ResolvedStyle`. Same value set as pause-before, stored on the rest-before field. The test uses `medium`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `rest-after` (`properties.json:13085`, syntax `<time [0s,inf]> | none | x-weak | weak | medium | strong | x-strong`). Store the used value on `ResolvedStyle`. The test uses `x-strong`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `rest` (`properties.json:13069`, syntax `<'rest-before'> <'rest-after'>?`). No field of its own. One value writes both sides. Two values write before then after. The test uses `none 100ms`.
### Speak

- [ ] `speak` (`properties.json:14668`, syntax `auto | never | always`). Store the used value on `ResolvedStyle`. The test uses `never` and fails if the public field is missing or `auto`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `speak-as` (`properties.json:14684`, syntax `normal | spell-out || digits || [ literal-punctuation | no-punctuation ]`). Store the used value on `ResolvedStyle`. Store the keywords the syntax allows together. The test uses `spell-out digits` and fails if either keyword is dropped. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
### Voice

- [ ] `voice-family` (`properties.json:17221`, syntax `[ <voice-family-name> | <generic-voice> ]# | preserve`). Store the used value on `ResolvedStyle`. Read `<generic-voice>` and `<voice-family-name>` from the pinned spec before accepting a name. Store `preserve` or the list. The test uses `preserve` and one family name taken from that production. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `voice-balance` (`properties.json:17189`, syntax `<number> | left | center | right | leftwards | rightwards`). Store the used value on `ResolvedStyle`. The test uses `left` and a number case. Do not collapse `left` to 0 if 0 is also a legal number. Keep the keyword. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `voice-duration` (`properties.json:17205`, syntax `auto | <time [0s,inf]>`). Store the used value on `ResolvedStyle`. Store `auto` or milliseconds. The test uses `2s` and fails if the field is `auto` or 0. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `voice-pitch` (`properties.json:17237`, syntax `<frequency [0Hz,inf]> && absolute | [ [ x-low | low | medium | high | x-high ] || [ <frequency [0Hz,inf]> | <semitones> | <percentage> ] ]`). Store the used value on `ResolvedStyle`. Store the keyword, the absolute flag, and the frequency, semitone, or percentage piece. The test uses `high` and fails if it is stored as `medium`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `voice-range` (`properties.json:17253`, syntax `<frequency [0Hz,inf]> && absolute | [ [ x-low | low | medium | high | x-high ] || [ <frequency [0Hz,inf]> | <semitones> | <percentage> ] ]`). Store the used value on `ResolvedStyle`. Same shape as voice-pitch, on the range field. The test uses `x-low` so it cannot pass by copying pitch. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `voice-rate` (`properties.json:17269`, syntax `[ normal | x-slow | slow | medium | fast | x-fast ] || <percentage [0,inf]>`). Store the used value on `ResolvedStyle`. The test uses `fast` and a percentage case. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `voice-stress` (`properties.json:17285`, syntax `normal | strong | moderate | none | reduced`). Store the used value on `ResolvedStyle`. The test uses `strong`. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.
- [ ] `voice-volume` (`properties.json:17301`, syntax `silent | [ [ x-soft | soft | medium | loud | x-loud ] || <decibel> ]`). Store the used value on `ResolvedStyle`. The test uses `silent`. Do not store `silent` as 0 decibel. Proof: `go test ./internal/layout -run TestStyleInternGenerated -count=1`.

## Phase 2: value acceptance and @supports

Put the grammar in `internal/layout/speech_style.go` and call it from `declarationValueAccepted` (`internal/layout/style_value_accept.go:139`). `engineSupportsProperty` (`internal/layout/style_cascade.go:1895`) must return the same answer. A bad token loses the cascade (`style_cascade.go:693-695`) and does not satisfy `@supports`. CSS-wide keywords stay accepted (`style_value_accept.go:102-104`).

### Cue

- [ ] `cue-before` (`properties.json:6366`). Accept `<url> <decibel>? | none`. Store `none`, or the URL text plus an optional decibel. Do not fetch the URL and do not pass it to `layout.Options.Images` (`layout/layout.go:26-32`). The public-box test uses `url("beep.wav")` and fails if that URL is missing. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
- [ ] `cue-after` (`properties.json:6350`). Accept `<url> <decibel>? | none`. Same storage as cue-before, on the after field. The test uses a different URL so a swapped before and after fails. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
- [ ] `cue` (`properties.json:6334`). Accept `<'cue-before'> <'cue-after'>?`. One value writes cue-before and cue-after. Two values write before then after. The test uses two different URLs. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
- [ ] **Expansion.** `cue` writes the longhands in its note. A CSS-wide keyword resets them. A later longhand wins. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
### Pause

- [ ] `pause-before` (`properties.json:12519`). Accept `<time [0s,inf]> | none | x-weak | weak | medium | strong | x-strong`. Store `none`, a non-negative time in milliseconds, or one of those strength keywords. A negative time is rejected. The test uses `strong` and `200ms` in two cases. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
- [ ] `pause-after` (`properties.json:12503`). Accept `<time [0s,inf]> | none | x-weak | weak | medium | strong | x-strong`. Same as pause-before, on the after field. The test uses `weak` so it cannot pass by copying the before value. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
- [ ] `pause` (`properties.json:12487`). Accept `<'pause-before'> <'pause-after'>?`. One value writes both sides. Two values write before then after. The test uses `strong weak`. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
- [ ] **Expansion.** `pause` writes the longhands in its note. A CSS-wide keyword resets them. A later longhand wins. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
### Rest

- [ ] `rest-before` (`properties.json:13101`). Accept `<time [0s,inf]> | none | x-weak | weak | medium | strong | x-strong`. Same value set as pause-before, stored on the rest-before field. The test uses `medium`. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
- [ ] `rest-after` (`properties.json:13085`). Accept `<time [0s,inf]> | none | x-weak | weak | medium | strong | x-strong`. The test uses `x-strong`. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
- [ ] `rest` (`properties.json:13069`). Accept `<'rest-before'> <'rest-after'>?`. One value writes both sides. Two values write before then after. The test uses `none 100ms`. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
- [ ] **Expansion.** `rest` writes the longhands in its note. A CSS-wide keyword resets them. A later longhand wins. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
### Speak

- [ ] `speak` (`properties.json:14668`). Accept `auto | never | always`. The test uses `never` and fails if the public field is missing or `auto`. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
- [ ] `speak-as` (`properties.json:14684`). Accept `normal | spell-out || digits || [ literal-punctuation | no-punctuation ]`. Store the keywords the syntax allows together. The test uses `spell-out digits` and fails if either keyword is dropped. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
### Voice

- [ ] `voice-family` (`properties.json:17221`). Accept `[ <voice-family-name> | <generic-voice> ]# | preserve`. Read `<generic-voice>` and `<voice-family-name>` from the pinned spec before accepting a name. Store `preserve` or the list. The test uses `preserve` and one family name taken from that production. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
- [ ] `voice-balance` (`properties.json:17189`). Accept `<number> | left | center | right | leftwards | rightwards`. The test uses `left` and a number case. Do not collapse `left` to 0 if 0 is also a legal number. Keep the keyword. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
- [ ] `voice-duration` (`properties.json:17205`). Accept `auto | <time [0s,inf]>`. Store `auto` or milliseconds. The test uses `2s` and fails if the field is `auto` or 0. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
- [ ] `voice-pitch` (`properties.json:17237`). Accept `<frequency [0Hz,inf]> && absolute | [ [ x-low | low | medium | high | x-high ] || [ <frequency [0Hz,inf]> | <semitones> | <percentage> ] ]`. Store the keyword, the absolute flag, and the frequency, semitone, or percentage piece. The test uses `high` and fails if it is stored as `medium`. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
- [ ] `voice-range` (`properties.json:17253`). Accept `<frequency [0Hz,inf]> && absolute | [ [ x-low | low | medium | high | x-high ] || [ <frequency [0Hz,inf]> | <semitones> | <percentage> ] ]`. Same shape as voice-pitch, on the range field. The test uses `x-low` so it cannot pass by copying pitch. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
- [ ] `voice-rate` (`properties.json:17269`). Accept `[ normal | x-slow | slow | medium | fast | x-fast ] || <percentage [0,inf]>`. The test uses `fast` and a percentage case. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
- [ ] `voice-stress` (`properties.json:17285`). Accept `normal | strong | moderate | none | reduced`. The test uses `strong`. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.
- [ ] `voice-volume` (`properties.json:17301`). Accept `silent | [ [ x-soft | soft | medium | loud | x-loud ] || <decibel> ]`. The test uses `silent`. Do not store `silent` as 0 decibel. Proof: `go test ./internal/layout -run TestSpeechAccept -count=1`.

## Phase 3: drawing-list payload and WASM JSON

`internal/layout/speech_payload.go` copies values through `PlacedElement` (`internal/layout/placed.go:7`) and `boxesFrom` (`layout/layout.go:35`) onto `layout.Box.Speech`. WASM adds omitempty fields on `drawBoxJSON` (`bindings/wasm/drawing_list.go:194`) inside `newDrawingList` (`drawing_list.go:296-309`). Schema stays `blinkless.drawinglist/1` (`drawing_list.go:20-23`). Do not edit `internal/layout/layout.go`.

- [ ] `cue-before` (`properties.json:6366`) is a field the host reads on `layout.Box.Speech`. Store `none`, or the URL text plus an optional decibel. Do not fetch the URL and do not pass it to `layout.Options.Images` (`layout/layout.go:26-32`). The public-box test uses `url("beep.wav")` and fails if that URL is missing. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.
- [ ] `cue-after` (`properties.json:6350`) is a field the host reads on `layout.Box.Speech`. Same storage as cue-before, on the after field. The test uses a different URL so a swapped before and after fails. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.
- [ ] `cue` (`properties.json:6334`) has no JSON key. The host reads the longhands. One value writes cue-before and cue-after. Two values write before then after. The test uses two different URLs. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.
- [ ] `pause-before` (`properties.json:12519`) is a field the host reads on `layout.Box.Speech`. Store `none`, a non-negative time in milliseconds, or one of those strength keywords. A negative time is rejected. The test uses `strong` and `200ms` in two cases. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.
- [ ] `pause-after` (`properties.json:12503`) is a field the host reads on `layout.Box.Speech`. Same as pause-before, on the after field. The test uses `weak` so it cannot pass by copying the before value. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.
- [ ] `pause` (`properties.json:12487`) has no JSON key. The host reads the longhands. One value writes both sides. Two values write before then after. The test uses `strong weak`. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.
- [ ] `rest-before` (`properties.json:13101`) is a field the host reads on `layout.Box.Speech`. Same value set as pause-before, stored on the rest-before field. The test uses `medium`. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.
- [ ] `rest-after` (`properties.json:13085`) is a field the host reads on `layout.Box.Speech`. The test uses `x-strong`. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.
- [ ] `rest` (`properties.json:13069`) has no JSON key. The host reads the longhands. One value writes both sides. Two values write before then after. The test uses `none 100ms`. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.
- [ ] `speak` (`properties.json:14668`) is a field the host reads on `layout.Box.Speech`. The test uses `never` and fails if the public field is missing or `auto`. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.
- [ ] `speak-as` (`properties.json:14684`) is a field the host reads on `layout.Box.Speech`. Store the keywords the syntax allows together. The test uses `spell-out digits` and fails if either keyword is dropped. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.
- [ ] `voice-family` (`properties.json:17221`) is a field the host reads on `layout.Box.Speech`. Read `<generic-voice>` and `<voice-family-name>` from the pinned spec before accepting a name. Store `preserve` or the list. The test uses `preserve` and one family name taken from that production. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.
- [ ] `voice-balance` (`properties.json:17189`) is a field the host reads on `layout.Box.Speech`. The test uses `left` and a number case. Do not collapse `left` to 0 if 0 is also a legal number. Keep the keyword. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.
- [ ] `voice-duration` (`properties.json:17205`) is a field the host reads on `layout.Box.Speech`. Store `auto` or milliseconds. The test uses `2s` and fails if the field is `auto` or 0. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.
- [ ] `voice-pitch` (`properties.json:17237`) is a field the host reads on `layout.Box.Speech`. Store the keyword, the absolute flag, and the frequency, semitone, or percentage piece. The test uses `high` and fails if it is stored as `medium`. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.
- [ ] `voice-range` (`properties.json:17253`) is a field the host reads on `layout.Box.Speech`. Same shape as voice-pitch, on the range field. The test uses `x-low` so it cannot pass by copying pitch. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.
- [ ] `voice-rate` (`properties.json:17269`) is a field the host reads on `layout.Box.Speech`. The test uses `fast` and a percentage case. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.
- [ ] `voice-stress` (`properties.json:17285`) is a field the host reads on `layout.Box.Speech`. The test uses `strong`. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.
- [ ] `voice-volume` (`properties.json:17301`) is a field the host reads on `layout.Box.Speech`. The test uses `silent`. Do not store `silent` as 0 decibel. Proof: `go test ./bindings/wasm -run TestDrawingListSpeech -count=1`.

## Phase 4: behavior tests

The test that closes a row calls `layout.DisplayList` (`layout/displaylist.go:137`, the pattern at `layout/displaylist_test.go:34`) and reads `layout.Box.Speech`. `layoutHTML` (`internal/layout/layout_test.go:41`) does not close the row.

- [ ] `cue-before` (`properties.json:6366`). Fail if this value is dropped before `layout.Box.Speech`. Store `none`, or the URL text plus an optional decibel. Do not fetch the URL and do not pass it to `layout.Options.Images` (`layout/layout.go:26-32`). The public-box test uses `url("beep.wav")` and fails if that URL is missing. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.
- [ ] `cue-after` (`properties.json:6350`). Fail if this value is dropped before `layout.Box.Speech`. Same storage as cue-before, on the after field. The test uses a different URL so a swapped before and after fails. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.
- [ ] `cue` (`properties.json:6334`). Fail if this value is dropped before `layout.Box.Speech`. One value writes cue-before and cue-after. Two values write before then after. The test uses two different URLs. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.
- [ ] `pause-before` (`properties.json:12519`). Fail if this value is dropped before `layout.Box.Speech`. Store `none`, a non-negative time in milliseconds, or one of those strength keywords. A negative time is rejected. The test uses `strong` and `200ms` in two cases. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.
- [ ] `pause-after` (`properties.json:12503`). Fail if this value is dropped before `layout.Box.Speech`. Same as pause-before, on the after field. The test uses `weak` so it cannot pass by copying the before value. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.
- [ ] `pause` (`properties.json:12487`). Fail if this value is dropped before `layout.Box.Speech`. One value writes both sides. Two values write before then after. The test uses `strong weak`. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.
- [ ] `rest-before` (`properties.json:13101`). Fail if this value is dropped before `layout.Box.Speech`. Same value set as pause-before, stored on the rest-before field. The test uses `medium`. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.
- [ ] `rest-after` (`properties.json:13085`). Fail if this value is dropped before `layout.Box.Speech`. The test uses `x-strong`. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.
- [ ] `rest` (`properties.json:13069`). Fail if this value is dropped before `layout.Box.Speech`. One value writes both sides. Two values write before then after. The test uses `none 100ms`. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.
- [ ] `speak` (`properties.json:14668`). Fail if this value is dropped before `layout.Box.Speech`. The test uses `never` and fails if the public field is missing or `auto`. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.
- [ ] `speak-as` (`properties.json:14684`). Fail if this value is dropped before `layout.Box.Speech`. Store the keywords the syntax allows together. The test uses `spell-out digits` and fails if either keyword is dropped. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.
- [ ] `voice-family` (`properties.json:17221`). Fail if this value is dropped before `layout.Box.Speech`. Read `<generic-voice>` and `<voice-family-name>` from the pinned spec before accepting a name. Store `preserve` or the list. The test uses `preserve` and one family name taken from that production. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.
- [ ] `voice-balance` (`properties.json:17189`). Fail if this value is dropped before `layout.Box.Speech`. The test uses `left` and a number case. Do not collapse `left` to 0 if 0 is also a legal number. Keep the keyword. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.
- [ ] `voice-duration` (`properties.json:17205`). Fail if this value is dropped before `layout.Box.Speech`. Store `auto` or milliseconds. The test uses `2s` and fails if the field is `auto` or 0. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.
- [ ] `voice-pitch` (`properties.json:17237`). Fail if this value is dropped before `layout.Box.Speech`. Store the keyword, the absolute flag, and the frequency, semitone, or percentage piece. The test uses `high` and fails if it is stored as `medium`. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.
- [ ] `voice-range` (`properties.json:17253`). Fail if this value is dropped before `layout.Box.Speech`. Same shape as voice-pitch, on the range field. The test uses `x-low` so it cannot pass by copying pitch. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.
- [ ] `voice-rate` (`properties.json:17269`). Fail if this value is dropped before `layout.Box.Speech`. The test uses `fast` and a percentage case. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.
- [ ] `voice-stress` (`properties.json:17285`). Fail if this value is dropped before `layout.Box.Speech`. The test uses `strong`. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.
- [ ] `voice-volume` (`properties.json:17301`). Fail if this value is dropped before `layout.Box.Speech`. The test uses `silent`. Do not store `silent` as 0 decibel. Proof: `go test ./layout -run TestDisplaySpeech -count=1`.

## Phase 5: catalog, matrix, and the deferred pointer

Status follows the public-box test. The catalog rule is `testdata/css/catalog/properties.json:9`. Handler discovery walks `styleGroups` (`scripts/css-catalog-map.py:336-370`).

- [ ] `cue-before` (`properties.json:6366`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.
- [ ] `cue-after` (`properties.json:6350`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.
- [ ] `cue` (`properties.json:6334`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.
- [ ] `pause-before` (`properties.json:12519`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.
- [ ] `pause-after` (`properties.json:12503`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.
- [ ] `pause` (`properties.json:12487`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.
- [ ] `rest-before` (`properties.json:13101`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.
- [ ] `rest-after` (`properties.json:13085`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.
- [ ] `rest` (`properties.json:13069`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.
- [ ] `speak` (`properties.json:14668`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.
- [ ] `speak-as` (`properties.json:14684`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.
- [ ] `voice-family` (`properties.json:17221`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.
- [ ] `voice-balance` (`properties.json:17189`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.
- [ ] `voice-duration` (`properties.json:17205`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.
- [ ] `voice-pitch` (`properties.json:17237`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.
- [ ] `voice-range` (`properties.json:17253`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.
- [ ] `voice-rate` (`properties.json:17269`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.
- [ ] `voice-stress` (`properties.json:17285`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.
- [ ] `voice-volume` (`properties.json:17301`). Set the handler to `applySpeechProps` and the test to the phase 4 test. Proof: `make catalog-check`.

### Deferred pointer

- [ ] Point `documentation/deferred.md:114` at `plans/v0.0.3/speech/speech-checklist.md`. Remove "permanent non-goal" for these 19 names. Say the values are stored and not spoken.
- [ ] Refresh `documentation/compatibility-matrix.md:1126` the same way. Regenerate the property section with `python3 scripts/css-catalog-map.py --matrix`.
- [ ] Update the local knowledge-base page in the same session as the code. This planning session does not write that page.

**Gate:** `make catalog-check` and `make matrix-check` exit 0. Record the new totals. The planning baseline was 389 implemented, 19 partial, 369 unsupported, 8 intentionally ignored (`properties.json:12-17`).


## Phase 6: closure gates

These stay open until the commands exit 0. During the work, run one package: `go test ./internal/layout -run '<Name>' -count=1`, `go test ./layout -run '<Name>' -count=1`, or `go test ./bindings/wasm -run '<Name>' -count=1`. Full gates go through the Makefile. Do not run bare `go test ./...`.

- [ ] `make test` exits 0.
- [ ] `make golden` exits 0. That target runs `TestDisplay` in `./layout`. It does not rasterize a page.
- [ ] `make lint` exits 0. It runs the file-size check. `internal/layout/layout.go` stays 2142 lines unless a reviewed allowlist edit says otherwise (`scripts/file-size-allowlist.txt:9`).
- [ ] `make claim-scan` exits 0. `claim-scan` reads `doc.go`, `README.md`, and `documentation/*.md` (`Makefile:104-114`). Do not claim a page PNG, a PDF, a scrollport, or a speech synthesizer.
- [ ] `make catalog-check` exits 0.
- [ ] `make matrix-check` exits 0.
- [ ] GitHub CI (`.github/workflows/ci.yml:1`, steps at `19-26`) runs only `make test` and `make lint`. The other gates are local.

**Gate:** every command above exited 0, and the output was recorded on the row.

## Phase 7: speech synthesis

- [~] Do not add a synthesizer, an audio device, or a cue player. Reason: this tree returns a drawing list (`layout/displaylist.go:129-138`) and rejects `Media: "speech"` (`css/css.go:147-155`). There is nothing here that can play `beep.wav`. Owner: a future host that reads `Box.Speech` and speaks it. Next gate: a host plan names those fields and a synthesizer outside this module, and a test shows the host received the cue URL and the voice keywords. Until then Blinkless only stores the values.

## Dependencies

- Cascade acceptance stays the shared `supportedDeclaration` rule (`internal/layout/style_value_accept.go:83-87`).
- `@media speech` stays unmatched for screen and print (`internal/css/media.go:72-73`). This ledger does not change that.
- Cue URLs are not images. The image callback stays for pictures (`layout/layout.go:26-32`).
- The native view plan does not own speech (`plans/v0.0.1/native-window/native-web-view-checklist.md`). Synthesis waits for a host plan that does.
- No new direct module. The allowlist stays `github.com/go-text/typesetting` and `github.com/tdewolff/canvas`.

