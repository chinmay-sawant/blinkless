# 0.0.1 - Parser and selector residuals

> **Parent:** `plans/v0.0.1/security/security-checklist.md` - security slice. This file records a 2026-10-15 rescan. It does not repeat a closed row from ledgers 01 through 05.
> **Status:** open. Nothing in this file is implemented. The scan checked `internal/html` and `internal/css` after the 65 closed controls.
> **Estimated effort:** one focused change per row, each with a regression that fails before the cap and passes after it.

---

## Overview

These rows are ways a document the converter already holds can still make parsing or selector matching do much more work than the closed caps allow. They are not web requests. Public `html.Parse` staying uncapped, the `:has` budget, the 32 MiB stylesheet cap, math depth, and the HTML depth and attribute caps stay in ledgers 01 and 03 and are not listed again.

## Phase 1: HTML parser

Proof for the phase, once a row is implemented: `go test ./internal/html`.

- [ ] Foreign raw-text reparse walks the captured text again (`internal/html/tree.go:135-140`, flag set at `internal/html/foreign.go:332-333`). A `<style>`, `<script>`, or similar tag inside SVG is ordinary foreign content, so the tokenizer consumes the run and then `scanTokens` calls `appendToken` on that run. A nested copy sets the flag again. The element-depth cap and the reprocess cap of 8 do not wrap this callback. Next gate: cap the reparse depth and add a nested `<svg><style>` regression that finishes.
- [ ] Foster-parented text is merged by appending to a Go string (`internal/html/tree.go:1373-1381`). Each one-character text token copied in front of a table walks the parent's children and copies the growing text. A long run of those tokens makes parse time grow with the square of the token count. Next gate: merge into a builder or cap the fostered text, with a regression.
- [ ] The active formatting list only drops a fourth element with the same attributes (`internal/html/formatting.go:63-68`). Distinct elements, such as `<b id="1">` then `<b id="2">`, all stay on the list, and later tags reconstruct that list into the tree (`internal/html/formatting.go:206`). The element-depth cap limits one chain. It does not limit how many distinct formatting elements the document can keep. Next gate: cap the list length and add a regression.

## Phase 2: Selectors and colors

Proof for the phase, once a row is implemented: `go test ./internal/css`.

- [ ] General-sibling matching rescans from the first child (`internal/css/match.go:253-260` and `internal/css/match.go:597-614`). `previousElementSibling` walks the parent's children from the start on every step. One `zzz ~ span` rule tested on each of many sibling spans does about N cubed child-list visits. The `:has` visit budget does not cover this walk. Next gate: remember the previous element sibling, or charge the walk to a budget, and add a wide-sibling regression.
- [ ] `:not`, `:is`, and `:where` start a new match budget (`internal/css/match.go:547-564`). Each argument calls `Match`, which allocates a fresh 1,048,576-visit budget. One compound of many `:not(:has(*))` or `:is(:has(*))` therefore multiplies the per-match cap by the argument count. The closed `:has` row shares one budget only inside one `MatchState`, and it rejects `:has` nested inside `:has`. Next gate: pass the caller's budget into these pseudos and add a regression that would have exceeded one budget.
- [ ] Nested `var()` color fallbacks recurse with no depth cap (`internal/css/values.go:195-197` and `internal/css/color_modern.go:477-479`). `ParseColor` and `ParseColorWide` call themselves on the fallback. The custom-property cycle bound of 16 and the math depth of 64 do not apply here. A declaration inside the 32 MiB sheet cap can recurse until the stack dies. Next gate: share a depth cap with math parsing and add a nested-fallback regression.
