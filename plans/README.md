# Plans

Current ledger: [v0.0.1 phase-wise checklist](v0.0.1/phase-wise-checklist.md).

That ledger records removal of the PDF writer and retention of the HTML drawing list. Its header reports the audited state (2026-10-10, rechecked after wave H): the writer package, pdfprofile, CLI, and compliance tooling are gone; open rows cover settings keys, binding leftovers, and Makefile and script cleanup. Close rows only after the required proof command exits 0.

Follow-up checklist: [v0.0.1 HTML, CSS, and JSON compatibility](v0.0.1/html-css-json-compatibility-checklist.md). It covers CSS fallback correctness, HTML parser measurements and repairs, the capability catalog, WASM drawing-list JSON, and remaining CSS behavior. Its header reports completion: all rows closed through waves A-F, with the wave G gate recorded on 2026-10-10.

Native view proposal: [native web view checklist](v0.0.1/native-window/native-web-view-checklist.md). It defines the runtime, scrolling, input, navigation, JavaScript, native surface, accessibility, and compatibility work around Blinkless's drawing-list engine. All implementation rows are open.

CSS masking first wave: [CSS masking checklist](v0.0.1/css-masking/css-masking-checklist.md). It targets 10 standard mask properties and 8 WebKit-prefixed forms. The display-list group contract and property behavior are implemented; all final gates passed on 2026-10-13.

v0.0.3 CSS coverage ledgers: [index](v0.0.3/README.md). Thirteen phase checklists, written 2026-10-15, still open. They plan animation, transitions, scroll, speech, SVG presentation, corner shape, interior shapes, row rules, anchors, regions, mask gradient sources, mask borders, and the remaining 3D aliases. No implementation has started. The masking first wave stays the finished mask contract. These ledgers do not replace it.

Security hardening: [security checklist](v0.0.1/security/security-checklist.md). Ledgers 01 through 09 are closed. The rescan rows in 06 through 09 landed on 2026-10-15.

Release note: [v0.0.1 release note](v0.0.1/release-note/v0.0.1.md), with the [54-commit ledger](v0.0.1/release-note/commits.md). It covers `master` from `23061d8` through `c085b34`. Tag `v0.0.1` is `108c2c9`.

Security release: [v0.0.2 release note](v0.0.2/release-note/v0.0.2.md), with the [commits after v0.0.1](v0.0.2/release-note/commits.md). Tag `v0.0.2`. `Makefile` binding stamps stay `0.0.1`.
