---
name: chrome-flex-pdf-closure
description: Close the complete 40-case Chromium Flexbox interaction inventory in blinkless by converting static inputs, proving the right engine and output behavior for each case, and recording coverage evidence. Use when agents are assigned individual Chrome Flexbox cases, case groups, or the whole inventory.
---

# Close the Chrome Flexbox interaction inventory

Use this skill to move every case in the 40-case Chrome Flexbox inventory from
scaffold to evidence-backed completion. The proof target depends on the
manifest's `goTarget`. The engine emits a drawing list, not a PDF or page PNG,
so the proof boundaries are direct layout tests and Chromium-reference
measurements. A passing direct layout test does not prove that the drawing
list preserves the result.

Read these files before editing:

- `knowledge-base/wiki/index.md`
- `plans/0.0.1/html-css-json-compatibility-checklist.md`
- `test/chrome/README.md`
- `test/chrome/manifest.json`
- the Chromium source file named by the case
- the existing test helper in the package you will change

## Work boundaries

The case assignment is the write boundary. Do not edit another agent's case,
shared test helper, production layout code, or manifest row unless the
assignment includes it. If a shared fix is needed, report the failing input,
the expected geometry, and the owning package instead of taking that file.

Agents may work in parallel only when their write sets are disjoint. Keep
fixture files, layout tests, drawing-list tests, and coverage metadata in
separate ownership groups.

Do not run Git commands. Do not build Chromium. Do not copy Blink C++ test
assertions into Go. Reuse the HTML and CSS behavior, then write an assertion
at a Go-owned boundary.

## The four completion layers

Apply all four layers to every assigned case. Do not mark one case complete
because a representative case in the same property family passes.

### 1. Static case input

Replace the generated scaffold with a reviewed static HTML file under
`test/chrome/cases/`.

The file must have:

- a valid doctype;
- the exact CSS interaction under test;
- stable labels or geometry markers that make the expected result observable;
- a short expected-behavior comment;
- the Chromium source path comment;
- no browser-only JavaScript or generated assertion matrix.

Keep the case's `id`, `fixture`, source path, category, and `goTarget` aligned
with `test/chrome/manifest.json`. The manifest status remains `scaffold` until
the required evidence exists.

Run the Chrome inventory validator after changing the input:

```sh
python3 scripts/generate_chrome_flex_cases.py
go test ./test/chrome
```

If the generator rewrites the file, update the generator's case template or
case table instead of leaving a hand edit that regeneration will erase.

Completion condition: the file is a reviewed static input, its expected
behavior is named, and the validator passes.

### 2. Engine behavior

Add or update the focused test at the boundary named by the case's `goTarget`.
Use an independent expected result from the Chromium behavior, CSS example,
or worked specification case. Do not recompute the expected value from the
implementation.

Use these boundaries:

- `layout-unit`: focused `internal/layout` geometry assertions;
- `chrome-reference`: a named Chromium result compared with the Go result;
- `golden-fixture`: drawing-list assertions against a `testdata/golden` fixture
  (public `layout.DisplayList` or a focused `internal/layout` test).

The assertion should cover the layout decision named by the case:

- sizing: used widths or heights after basis, grow, shrink, min, and max;
- alignment: item positions relative to the container, including exact center
  or free-space distribution;
- flow: line assignment, axis positions, reverse order, wrapping, and
  `align-content` placement;
- print fragmentation: the page and line geometry that must survive a break.

A test that only checks that an item moved away from the origin is too weak.
For centering, compare the item center with the container center. For wrapping,
check both line membership and the cross-axis offset. For reverse flow, check
the positions of named items, not only their extracted text order.

Run the focused package test first:

```sh
go test ./internal/layout -run '<test name>' -count=1
```

Completion condition: the expected behavior fails against the old behavior or
would catch a regression, passes with the current implementation, and the test
names the case or interaction it proves.

### 3. Product output

Run the same static HTML through the product output path required by the case:
the public drawing list (`layout.DisplayList`, or its JSON projection from
`bindings/wasm`) and the Chromium runner for reference cases. There is no PDF
writer and no page rasterizer.

Every output case must prove the checks that apply:

- drawing list: op kinds, ordered geometry, text content, fonts, and image
  payloads that prove the case's expected placement or sizing;
- Chromium reference: the same static input, matching selected geometry or
  pixels, and an explicit tolerance or unsupported-feature decision.

Do not treat a successful conversion as proof of the interaction. Do not
treat `make golden` as proof of centering, item width, or line placement by
itself; it runs the public `TestDisplay` suite in `./layout`. Add a geometry
assertion for the behavior under test.

For the print-fragmentation case, use a golden fixture and pin the op geometry
that must survive a break. Add a reviewed Chromium reference crop only when
op checks cannot detect the regression.

Run the focused output test before the full gates:

```sh
go test ./layout/ -run '<test name>' -count=1
```

Use the equivalent focused command in `internal/layout` or the reference
runner when the case has another output target.

Completion condition: the selected product output proves the case's expected
behavior, not merely that conversion completed successfully.

### 4. Coverage record

Record the evidence for every assigned case in the canonical 0.0.1 checklist
and update the matching knowledge-base page in the same work session.

Use the case allocation already defined by the plan:

- `layout-unit`: direct box geometry;
- `chrome-reference`: the same static input compared with Chromium geometry
  or pixels;
- `golden-fixture`: drawing-list geometry and semantic checks.

Do not create a pairwise matrix of every CSS property. The inventory is the
40 named cases in `test/chrome/manifest.json`. Close each case individually.
Add a named case when two declarations share a layout decision or exercise a
distinct branch. The coverage record must make missing families visible,
including direction plus alignment, automatic sizing plus alignment, wrapping
plus gaps, percentage sizes, intrinsic sizing, replaced elements, and print
fragmentation.

Only close a checklist row after its named test and validation command pass.
Do not change a row to `[x]` from intent.

Completion condition: every assigned case has an agreed status, the plan,
manifest, test name, and evidence agree about what it proves, and remaining
interaction families are explicit.

## Delegation modes

When several agents use this skill, partition the 40 manifest case IDs first.
Assign one mode and a disjoint file set to each agent. A family label such as
"sizing" is not an ownership boundary by itself because cases in one family
may share files.

| Mode | Owns | Returns |
| --- | --- | --- |
| Static input | one or more assigned HTML cases and their generator entries | case IDs, source paths, expected behavior, validator output |
| Layout unit | assigned `internal/layout` tests and only the required production fix | test names, expected geometry, red/green commands, changed files |
| Drawing-list integration | assigned `layout`/`internal/layout` test or golden fixture | op count, semantic text, geometry proof, focused test output |
| Reference comparison | assigned Chromium-reference cases and comparison notes | browser result, Go result, tolerance, unsupported-feature decision |
| Coverage record | assigned plan and knowledge-base rows | closed rows, evidence links, and remaining interaction families |

Each agent must return a short handoff:

```text
Cases: <ids>
Files: <paths>
Behavior: <one sentence per case>
Proof: <commands and pass results>
Remaining: <explicit gaps>
```

## Whole-inventory completion

The inventory is complete only when all 40 manifest cases have one of these
evidence-backed outcomes:

- completed, with all four layers closed;
- explicitly unsupported, with a source-backed reason, a safe skip or rewrite
  note, and coverage recorded;
- blocked by a named engine gap, with a failing test and an owner.

An agent may not mark an unworked case complete because another case uses the
same CSS property. Representative tests prove a shared branch. They do not
close every case that mentions that property.

## Final verification

After all assigned cases are integrated, run the repository gates in this
order:

```sh
make test
make golden
make claim-scan
make lint
```

The final report must distinguish:

- the 40 case IDs and their outcomes;
- static inputs completed;
- direct layout cases completed;
- drawing-list or Chromium-reference cases completed;
- print-fragmentation coverage completed;
- interaction families still missing.

Do not call the whole inventory complete when only representative layout tests
have passed.
