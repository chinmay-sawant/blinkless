# 0.0.1 - HTML, CSS, and JSON compatibility

> **Parent:** [0.0.1 migration ledger](phase-wise-checklist.md). This checklist owns the compatibility work identified in the repository review.
> **Status:** planned. All implementation and validation rows remain open.
> **Estimated effort:** six implementation phases. Parser repair is the largest phase; estimate its duration after the Phase 2 corpus baseline.
> **Source review:** 2026-10-09. Browser comparisons below were measured on 2026-10-08.

---

## Overview

Make the HTML tree and CSS results agree with the standard for the inputs the engine supports. Then describe those capabilities in JSON and export a drawing list that a consumer can replay.

The output remains `layout.DisplayList`. It carries operations, paint order, element boxes, and coordinate conversions. See [layout/displaylist.go:99](../../layout/displaylist.go). HTML parsing, CSS support, and drawing-list output are separate contracts. Each needs its own proof.

The tokenizer reads HTML characters and produces tags, attributes, and text. The tree builder puts those pieces into the document's parent/child structure. The CSS cascade chooses the winning declaration for each property. A feature query such as `@supports` asks whether a property and value work. A corpus is a fixed collection of test inputs and expected results. A namespace identifies whether an element belongs to HTML, SVG, or MathML.

This document is the only checklist for the findings below. The parent ledger keeps the earlier writer-removal work. Existing broad documentation cleanup in that ledger must use the behavior and coverage results recorded here.

Planning is the work authorized for this session. Implementation starts in a later session. `[ ]` means unimplemented or unproven. `[x]` requires passing evidence from the final source for that change. `[~]` requires a reason, an owner, and a next gate. A historical passing test does not close a new row.

## Executive summary

| Phase | Result | Main proof |
|---|---|---|
| 1 | Invalid CSS preserves valid fallbacks; feature queries report usable support | Declaration and public box-geometry tests |
| 2 | A pinned parser corpus measures the actual gap | Exact tree comparisons and explicit pass/fail/skip counts |
| 3 | Parser rules repair the measured gaps | Corpus cases plus CSS selector and layout regressions |
| 4 | CSS capability JSON matches code and behavior | Catalog checks and linked behavior tests |
| 5 | WASM returns a versioned, usable drawing-list JSON payload | Native contract tests and an actual WASM consumer |
| 6 | Stored CSS features gain behavior or an honest support status | Line-layout comparisons and final gates |

### Evidence behind the plan

The probes used the public Go API and Google Chrome 143.0.7499.40. They establish specific differences, not an overall compatibility percentage. Phase 2 must make the parser comparisons repeatable.

| Finding | Observed result | Source checked on 2026-10-09 |
|---|---|---|
| CSS-01: `width:80px; width:bogus` | Blinkless produced a 200px box in a 200px viewport; Chrome retained 80px | [style_cascade.go:1195](../../internal/layout/style_cascade.go) accepts every declaration value |
| CSS-02: `@supports(display:bogus)` | Blinkless applied the block; Chrome reported false | [style_cascade.go:1626](../../internal/layout/style_cascade.go) asks property handlers; [style_properties.go:58](../../internal/layout/style_properties.go) reports ownership even when a value is rejected |
| HTML-01: `<div/><span>child</span>` | Blinkless closed the div; Chrome put the span inside it | [html.go:271](../../internal/html/html.go) closes every self-closing token |
| HTML-02: `<p>one<div>two</div>three` | Blinkless kept all text in the paragraph; Chrome ended it before the div | [html.go:409](../../internal/html/html.go) has limited automatic closing rules |
| HTML-03: `&amp;` in a style element | Blinkless decoded it to `&`; Chrome preserved it | [html.go:217](../../internal/html/html.go) decodes all text tokens |
| HTML-04: unfinished comment | Blinkless returned an error; Chrome recovered | [html.go:534](../../internal/html/html.go) returns an unterminated-comment error |
| HTML-05: table wrappers and omitted cell ends | Tests expect direct table-to-row children and a new cell outside the row | [html_test.go:472](../../internal/html/html_test.go), [html.go:300](../../internal/html/html.go) |
| JSON-01: CSS coverage catalog | Six expected catalog files under `plans/0.2.6/catalog/` are absent | [css-catalog-map.py:60](../../scripts/css-catalog-map.py), [compatibility-matrix.md:6](../../documentation/compatibility-matrix.md) still cites the old 407/818 count |
| JSON-02: property discovery | The scanner reads selected files, while dispatch also calls separate property groups | [css-catalog-map.py:142](../../scripts/css-catalog-map.py), [style_cascade.go:1497](../../internal/layout/style_cascade.go) |
| JSON-03: WASM output | Requests accept PNG/JPEG names; output is JSON containing only an operation count and dimensions | [contract.go:101](../../bindings/wasm/contract.go), [contract.go:160](../../bindings/wasm/contract.go), [manifest.json:3](../../testdata/wasm/manifest.json) |
| CSS-03: line wrapping | `text-wrap-style` is stored; source search found no line-layout reader | [style_text_props.go:25](../../internal/layout/style_text_props.go), [inline.go](../../internal/layout/inline.go) |

The existing parser, feature-query, and Chrome-manifest tests passed during the review. The manifest test validates case metadata and fixture presence. Its 40 `completed` entries are not a measured browser pass rate. See [manifest_test.go:44](../../test/chrome/manifest_test.go).

Reference rules: [HTML parsing](https://html.spec.whatwg.org/multipage/parsing.html), [CSS declaration support](https://www.w3.org/TR/css-conditional-3/#support-definition), and the [html5lib parser corpus](https://github.com/html5lib/html5lib-tests). Full browser rendering parity requires more evidence than passing parser cases.

## Phase 1: CSS values and fallback correctness

### 1.1 Declaration handling

- [ ] **CSS-01a:** In `internal/layout/style_cascade_test.go`, add a regression where `width:80px; width:bogus` yields an 80px box. Include an invalid higher-specificity declaration and invalid `!important` declaration.
- [ ] **CSS-01b:** In `internal/layout/style_cascade.go` and the owning property handlers, reject invalid ordinary values before they replace a valid declaration. Verify stylesheet and inline inputs through `layout.DisplayList`.
- [ ] **CSS-01c:** Preserve custom-property rules while changing validation. In `internal/layout/`, test valid `var()` substitution, missing variables with fallbacks, and values that become invalid only after substitution. Those cases must follow their own standard rules rather than revive an earlier declaration incorrectly.
- [ ] **CSS-01d:** In `internal/layout/style_cascade_test.go`, verify shorthand resets, longhand order, inheritance, layers, and vendor aliases still select the right values after the change.

### 1.2 Feature queries

- [ ] **CSS-02a:** In `internal/css/atrules_test.go`, add positive and negative property/value cases, including `display:block`, `display:bogus`, unknown properties, nested conditions, and negation.
- [ ] **CSS-02b:** Make `engineSupportsProperty` in `internal/layout/style_cascade.go` validate the value using the same acceptance rules as declarations. Handler ownership alone must not imply support.
- [ ] **CSS-02c:** Add a public `layout/` regression where `@supports(display:bogus)` leaves the base 11px width intact, while a valid supported query applies its override.
- [ ] **GATE-01:** Run `go test -p 2 -parallel 2 ./internal/layout -run 'TestCascade' -count=1`, `go test -p 2 -parallel 2 ./internal/css -run 'TestSupports' -count=1`, and the new public regressions. Record commands, test names, and exit codes here.

## Phase 2: Repeatable HTML conformance measurements

### 2.1 Corpus and comparison runner

- [ ] **HTML-BASE-01:** Add pinned parser test data under `testdata/html-conformance/`. Record the upstream revision, license and attribution, case identifiers, tokenizer/tree-construction categories, and selected scripting mode.
- [ ] **HTML-BASE-02:** Add a corpus runner in `internal/html/` that compares exact text, node kinds, parent/child order, attributes, namespaces, and document mode where the case specifies them. Missing engine fields must appear as failures or explicitly unsupported categories.
- [ ] **HTML-BASE-03:** Add one canonical comparison tool under `scripts/`. Run the same inputs in available Chromium, WebKit, and Gecko references. Record browser versions. Count unavailable browsers as skipped comparisons; preserve significant text and whitespace.
- [ ] **HTML-BASE-04:** Record a baseline by category with total, passed, failed, and skipped cases. Keep generated diffs and browser logs under ignored `temps/html-conformance/`; record commands and counts beside this row when it closes.
- [ ] **HTML-BASE-05:** Turn HTML-01 through HTML-05 into permanent corpus cases and inspect each affected assertion in `internal/html/html_test.go`. Identify tests that currently require behavior different from the standard.
- [ ] **GATE-02:** Run the pinned corpus and comparison tool twice with the same inputs. Case identifiers and counts must agree. Record failures as baseline evidence, and list the parser categories that Phase 3 must repair.

## Phase 3: HTML tokenizer and tree construction

### 3.1 Tokenizer rules

- [ ] **HTML-03a:** In `internal/html/`, separate ordinary text, raw style/script text, and title/textarea text that decodes character references. Prove entity handling in each context with corpus cases.
- [ ] **HTML-03b:** Add context-aware attribute character-reference handling in `internal/html/entities.go` and attribute scanning. Prove ambiguous ampersands and references without semicolons against the corpus.
- [ ] **HTML-04a:** Replace fatal errors for recoverable malformed input with the specified token/EOF behavior in `internal/html/`. Prove unfinished comments, tags, declarations, and quoted attributes; retain deliberate resource-limit failures.
- [ ] **HTML-INPUT-01:** Implement the corpus-backed UTF-8 input preprocessing rules in `internal/html/`, including newline handling and the specified handling of null characters. Keep BOM and depth-limit regression tests.

### 3.2 Tree-building rules

- [ ] **HTML-01a:** In `internal/html/`, ignore the self-closing flag on ordinary non-void HTML elements. Keep void-element behavior and apply foreign-element rules by namespace.
- [ ] **HTML-02a:** Add the parser states that insert omitted `html`, `head`, and `body` elements and close paragraphs, list items, and definition items in the correct contexts. Prove tree shape and selectors such as `body > p`.
- [ ] **HTML-05a:** Add table states that insert required wrappers, keep adjacent cells in their row, and move misplaced table content to the standard location. Prove both exact trees and table box geometry.
- [ ] **HTML-FORMAT-01:** Repair misnested formatting elements in `internal/html/` using the specified active-formatting rules. Prove cases such as `<b><i>one</b>two</i>` and their resulting styled text runs.
- [ ] **HTML-CONTEXT-01:** Implement the remaining corpus-required select and template states in `internal/html/`. Keep template content separate from ordinary rendered children and verify downstream traversal.
- [ ] **HTML-FOREIGN-01:** Extend the node representation in `internal/html/` for SVG/MathML namespaces, attribute adjustments, and transitions back into HTML content. Check CSS matching and layout callers after the representation change.
- [ ] **HTML-MODE-01:** Record doctype-derived document mode in `internal/html/` and carry it to the style/layout boundary. Test no-quirks, limited-quirks, and quirks classification; document separately any unimplemented layout effects.
- [ ] **HTML-FRAGMENT-01:** Add internal context-aware fragment parsing in `internal/html/`. Prove table, select, raw-text, and foreign-element contexts before exposing any new public entry point.
- [ ] **HTML-INTEGRATION-01:** Update the assertions identified in Phase 2 to the standard trees. Verify `html.Parse`, detached `markup.Parse` copies, selector matching in `internal/css/`, and placement in `layout/` agree with the new structure.
- [ ] **GATE-03:** Run all in-scope pinned tokenizer and tree-construction cases plus `go test -p 2 -parallel 2 ./internal/html -count=1` and targeted integration regressions. Record every exclusion. Describe results as UTF-8 parsing with scripting disabled unless broader input handling is proved.

Full HTML parsing compliance is a separate claim. The public parser takes UTF-8 bytes, and `internal/load/load.go:968` rejects other charsets. A UTF-8 corpus pass does not prove browser byte-encoding detection or every HTML parsing rule.

## Phase 4: CSS capability JSON and truthful coverage

### 4.1 Catalog and evidence

- [ ] **CAT-01:** Define a versioned catalog schema under `testdata/css/catalog/`. Each row needs a property name, aliases, accepted value forms, limitations, source path, behavior-test references, and a status: implemented, partial, unsupported, or intentionally ignored.
- [ ] **CAT-02:** Pin the upstream property inventory and its source revision under `testdata/css/catalog/`. Preserve source metadata so draft, vendor, SVG, and browser-UI properties do not become one unexplained coverage denominator.
- [ ] **CAT-03:** Update the canonical `scripts/css-catalog-map.py` to use that catalog and discover every registered property group, constants used as property names, and vendor mappings. A handler's presence must not automatically promote its status.
- [ ] **CAT-04:** Add catalog checks for duplicate names, missing handlers, invalid statuses, missing source/test references, and summary-count drift. Wire the read-only check into the Makefile and CI.
- [ ] **CAT-05:** Audit each advertised supported value against a consumer and a behavior test. Mark storage-only features partial or unsupported, with a named missing behavior. Start with CSS-03 and `box-decoration-break`.
- [ ] **CAT-06:** Generate the property tables in `documentation/compatibility-matrix.md` from the current catalog. Replace the absent 0.2.6 catalog reference and historical 407/818 count with current measured totals and explicit limitations.
- [ ] **CAT-07:** In `test/chrome/manifest.json` and `manifest_test.go`, connect completed cases to named behavior tests or measured browser evidence. An inventory check alone must not close a behavior case.
- [ ] **GATE-04:** Run `python3 scripts/css-catalog-map.py --check` and the new catalog tests. Regenerate the documented tables, rerun the check, and record totals and exit codes here.

## Phase 5: Usable drawing-list JSON from WASM

### 5.1 Request and output contracts

- [ ] **WASM-01:** Define the versioned drawing-list JSON schema in `documentation/library-api.md`. Specify operation kinds, paint order, element boxes, units, text baselines, transforms, opacity, blend-group boundaries, font references, and image payload references.
- [ ] **WASM-02:** Add an explicit serializer for the public `layout.Display` data. Give font/image resources stable per-result identifiers and preserve payloads needed by a consumer. Avoid serializing internal pointers or relying on hidden embedded fields.
- [ ] **WASM-03:** In `bindings/wasm/contract.go`, accept and default to the drawing-list mode. Remove or explicitly reject obsolete image-mode and unused writer fields, while keeping JSON size checks and stable error responses.
- [ ] **WASM-04:** Replace the count-only payload in `bindings/wasm/contract.go` with the full versioned result. The array of operations, its paint order, and resource references must agree with the same native `layout.DisplayList` call.
- [ ] **WASM-05:** Update `testdata/wasm/manifest.json`, its README, and `scripts/check-wasm-contract.sh` to describe JSON drawing-list output. Remove PDF/PNG/JPEG output assertions from this fixture contract.
- [ ] **WASM-06:** Add native contract tests in `bindings/wasm/` covering filled/stroked boxes, lines, text, images, links, groups, paint order, units, resource references, and malformed requests. Include output-size enforcement after serialization.
- [ ] **WASM-07:** Add a WASM consumer check under `scripts/` that decodes a built result and reads actual operations and referenced resources. Compare it with the native result for the same fixture; an operation count alone cannot pass.
- [ ] **GATE-05:** Run `go test -p 2 -parallel 2 ./bindings/wasm -count=1`, `make wasm-test`, and the actual consumer check. Record the runtime, artifact path, schema version, and exit codes here.

## Phase 6: Complete CSS behavior and close the compatibility work

### 6.1 Line wrapping and remaining advertised behavior

- [ ] **CSS-03a:** Add line-layout cases in `internal/layout/` for `text-wrap-style:balance`. Assert line breaks or line widths against a recorded browser reference, including a case whose balanced result differs from normal wrapping.
- [ ] **CSS-03b:** Make `text-wrap-style:balance` affect line placement in `internal/layout/inline.go` or a focused same-package helper. Verify normal wrapping, nowrap, explicit breaks, mixed inline styles, and narrow widths remain correct.
- [ ] **CSS-03c:** For `pretty`, `stable`, and other advertised forms, implement a tested result or record the unsupported behavior in the Phase 4 catalog. Feature queries must follow that decision.
- [ ] **CSS-REVIEW-01:** Compare the existing Flexbox, Grid, logical-property, and text fixtures with browser box geometry. Record per-case results, browser versions, and measurement tolerances.
- [ ] **CSS-REVIEW-02:** Add one implementation row here for each confirmed defect from CSS-REVIEW-01. Each row needs the affected file, expected result, and a permanent regression. Complete those rows before expanding the property inventory.

### 6.2 Final evidence and documentation

- [ ] **GATE-06a:** On the final implementation tree, run `make build`, `make test`, `make golden`, `make claim-scan`, and `make lint` once after the targeted checks. Record each exit code. Repeat a gate only after a relevant edit, failure, or unresolved concern.
- [ ] **GATE-06b:** Run the pinned parser corpus, catalog check, browser comparisons, and WASM consumer on that same tree. Record versions, case totals, skips, and remaining unsupported categories.
- [ ] **DOC-01:** Update `documentation/architecture/05-html-parser.md`, `06-css.md`, `compatibility-matrix.md`, `library-api.md`, and `test/chrome/README.md` from the final evidence. Keep parser compliance, CSS coverage, and rendering comparisons as separate claims.
- [ ] **KB-01:** Update `knowledge-base/wiki/concepts/html-css-json-compatibility.md`, its index, and its operation log in the same change as each shipped behavior. Point to this ledger and its source/test evidence.
- [ ] **GATE-06c:** Audit the final explanation with the Feynman and unslop skills. Record the pass count and close only rows whose behavior and proof have both landed.

## Dependencies

Phase 1 can start immediately. Phase 2 establishes the failures and exact comparison format before Phase 3 changes parser behavior. Phase 4 uses the acceptance rules from Phase 1 and the corrected trees from Phase 3. Phase 5 uses the existing public display-list contract and can proceed after Phase 1. Phase 6 updates the catalog after the remaining CSS behavior is settled.

| Work | Prerequisite |
|---|---|
| Parser repairs | Pinned corpus baseline and affected-test inventory |
| CSS support statuses | Value acceptance plus actual consumer/test evidence |
| WASM payload | Documented operation/resource schema |
| Browser parity statements | Recorded per-case comparisons with named versions |
| Final completion | All applicable gates pass on the final source |

Implementation must keep the current direct-dependency policy and the default pure-Go build. A dependency change needs the user sign-off required by `AGENTS.md`. A browser is a comparison tool, not a runtime dependency of the engine.

Use the capped Makefile targets for full suites. For each implementation wave, run the smallest relevant regressions while editing, then `make test` and `make lint` before closing its phase. Parser, CSS, or layout changes also require `make golden`. Store gate evidence beside the closing row. This documentation-only planning change does not run implementation gates.

Commit and publication remain separate actions requiring user authorization. Do not run Git commands while executing this plan unless that authorization has been given.

## Planning review

The planning change passed the document structure, local-link, checklist-status, and em-dash checks on 2026-10-09. The Feynman explanation audit took two passes. Implementation rows remain open, and no implementation tests or lint ran for this documentation-only change.
