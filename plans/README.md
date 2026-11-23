# Plans

Current ledger: [v0.0.1 phase-wise checklist](v0.0.1/phase-wise-checklist.md).

That ledger records removal of the PDF writer and retention of the HTML drawing list. Its header reports the audited state (2026-10-10, rechecked after wave H): the writer package, pdfprofile, CLI, and compliance tooling are gone; open rows cover settings keys, binding leftovers, and Makefile and script cleanup. Close rows only after the required proof command exits 0.

Follow-up checklist: [v0.0.1 HTML, CSS, and JSON compatibility](v0.0.1/html-css-json-compatibility-checklist.md). It covers CSS fallback correctness, HTML parser measurements and repairs, the capability catalog, WASM drawing-list JSON, and remaining CSS behavior. Its header reports completion: all rows closed through waves A-F, with the wave G gate recorded on 2026-10-10.

Native view proposal: [native web view checklist](v0.0.1/native-window/native-web-view-checklist.md). It defines the runtime, scrolling, input, navigation, JavaScript, native surface, accessibility, and compatibility work around Blinkless's drawing-list engine. All implementation rows are open.

CSS masking first wave: [CSS masking checklist](v0.0.1/css-masking/css-masking-checklist.md). It targets 10 standard mask properties and 8 WebKit-prefixed forms. The display-list group contract and property behavior are implemented; all final gates passed on 2026-10-13.

v0.0.3 CSS coverage ledgers: [index](v0.0.3/README.md). Thirteen phase checklists were written 2026-10-15. Several are implemented and closed; the index lists the open rows. They cover animation, transitions, scroll, speech, SVG presentation, corner shape, interior shapes, row rules, anchors, regions, mask gradient sources, mask borders, and the remaining 3D aliases. The masking first wave stays the finished mask contract. These ledgers do not replace it.

Security hardening: [security checklist](v0.0.1/security/security-checklist.md). Ledgers 01 through 09 are closed. The rescan rows in 06 through 09 landed on 2026-10-15.

Release note: [v0.0.1 release note](v0.0.1/release-note/v0.0.1.md), with the [54-commit ledger](v0.0.1/release-note/commits.md). It covers `master` from `23061d8` through `c085b34`. Tag `v0.0.1` is `108c2c9`.

Security release: [v0.0.2 release note](v0.0.2/release-note/v0.0.2.md), with the [commits after v0.0.1](v0.0.2/release-note/commits.md). Tag `v0.0.2`. At the tag, the `Makefile` binding stamps stayed `0.0.1`.

Feature release draft: [v0.0.3 release note](v0.0.3/release-note/v0.0.3.md), with the [commits after v0.0.2](v0.0.3/release-note/commits.md). Tag `v0.0.3` follows the release PR merge.

Stabilization release: [v0.1.0 release note](v0.1.0/release-note/v0.1.0.md), with the [26-commit ledger](v0.1.0/release-note/commits.md). Tag `v0.1.0` is the annotated tag object `5dd4571`, peeled to `2e2abfc`.

Patch-path and render-cost release: [v0.1.1 release note](v0.1.1/release-note/v0.1.1.md), with the [39-commit ledger](v0.1.1/release-note/commits.md) and the [agent context pack](v0.1.1/README.md). Tag `v0.1.1` follows the release PR merge. The `0.1.0` CHANGELOG section was backfilled in the `0.1.1` release PR.

Pending CSS for the next cut: [v0.1.2 checklists](v0.1.2/README.md). Nine phase files, quickest first. Ledger 1 is the three `-webkit-background-*` aliases and the four partial rows. Ledgers 6, 8, and 9 are parked. Open rows in the v0.0.3 SVG, scroll, and speech files now point here.

v0.1.6 Go review hardening: [v0.1.6 checklists](v0.1.6/README.md) ([go-review-hardening](v0.1.6/go-review-hardening/README.md)). Four phase files covering cross-checked P0 concurrency/correctness fixes, P1 lock contention/cache bounds/API edge cases, P1 CI race/fuzz/Makefile repair, and P2 lint hygiene/`internal/layout` extraction.

v0.1.6 release note: [v0.1.6 release note](v0.1.6/release/v0.1.6.md). Covers PRs #45 through #49 after tag `v0.1.5`: system CJK font fallback, the 512-byte `ResolvedStyle`, two data-race fixes, CSS correctness fixes, and CI repair. Tag `v0.1.6` is not cut yet.
