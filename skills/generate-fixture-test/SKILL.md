---
name: generate-fixture-test
description: >
  Create a drawing-list regression test for an existing blinkless fixture.
  Use when a user asks to pin a fixture's operations, compare a fixture
  against the current engine, or generate fixture test cases from the golden
  corpus. Measure every expected value directly from the engine or a
  Chromium reference; never copy coordinates from another fixture. Not for
  diagnosing a wrong-looking fixture, which belongs to diagnose-fixture-picture.
---

# Generate a fixture test

Create one evidence-backed Go test for one existing golden fixture. The test
must pin the drawing-list operations the current engine produces for that
fixture's HTML, measured from the engine itself (or a Chromium reference for
cross-checks). It must also pin a small set of authored features measured from
that exact fixture.

This skill is for test generation and the supporting drawing-list serializer
coverage needed by those tests. It is not a layout redesign, a layout fix, or
a visual parity investigation.

## Hard boundaries

- Work on one fixture at a time unless the user explicitly requests a
  batch.
- Do not run any Git command. This includes read-only commands such as status,
  diff, log, and ls-files. The user may inspect or publish the changes later.
- There is no committed reference PDF. Treat the measured drawing list (a
  saved JSON dump or a recorded op table) as the reference artifact, and never
  overwrite a golden fixture.
- Never invent a coordinate and never copy a coordinate, font, or operation
  count from another fixture.
- Do not widen a tolerance to make a failing assertion pass.
- Do not mark a plan row complete until its exact proof command has passed.

## The contract

The reference pair is:

```text
testdata/golden/fixture-NN-<slug>.html
the drawing list the current engine produces for it
```

There is no committed PDF. Capture the engine's drawing list once as a
reference snapshot (for example with
`go run ./bindings/wasm -fixture testdata/golden/<fixture>.html -out /tmp/<fixture>.json`),
then write a Go test that rebuilds the styled document and asserts the ops.
The public surface is `layout.DisplayList` / `layout.DisplayListOptions`; the
fixture-backed tests under `internal/layout` show the package-local style.

Coordinates use the drawing-list space:

- Operation coordinates are canvas points, y down.
- Text and bullet `Y` is the baseline.
- Boxes are CSS pixels, y down, origin at the top left.
- Distances are points, where 72 points equal one inch.
- The JSON dump and `layout.Display` expose the same `Ops` in source order and
  `Order` in paint order.

The fixture-specific assertions pin representative authored features. They do
not need to repeat every text string when the full operation record is already
compared. Choose anchors that make a layout change easy to understand.

## Phase 1: discover the current implementation

Read the local knowledge-base entry point first:

```text
knowledge-base/wiki/index.md
```

Then inspect the current source. Reuse existing helpers before adding another
one:

```text
layout/displaylist.go
layout/displaylist_test.go
internal/layout/fixture_bugs_test.go
internal/layout/spacing_fix_test.go
bindings/wasm/drawing_list.go
```

Confirm these facts from source, not memory:

1. `html.Parse` reads the fixture HTML and `css.Apply` styles it.
2. `layout.DisplayList` / `DisplayListOptions` returns the `Display` with
   `Ops`, `Order`, `Boxes`, and the canvas size.
3. The op accessors (`LinkURI`, `ImageBytes`, `Opacity`, `Transform`, ...) are
   nil-safe and are the supported way to read rare payloads.
4. The existing fixture tests show where the package-local helpers live.

If a shared helper does not exist, add the smallest package-local
infrastructure needed before writing the fixture test. Keep fixture-specific
assertions in the test file, not in the engine.

## Phase 2: resolve the exact fixture pair

Start from the fixture the user names. Check that the HTML exists by matching
the complete basename, not only the numeric ID. This matters for duplicate IDs
such as the two fixture-29 files.

For example:

```text
Fixture: testdata/golden/fixture-01-simple-invoice.html
```

Read the HTML before choosing anchors. Record which visible features it
actually authors:

- text groups and their semantic role
- tables and repeated rows
- borders and rules
- filled backgrounds
- images and their source paths
- links, if link ops are in the requested scope
- header and footer companions, if the fixture has them

Do not call a styled text block an image. Do not add an image assertion when
the HTML has no image. Do not assume that the next fixture has the same canvas
size, margins, rows, or feature mix.

If the basename has no unique matching HTML, stop and report the ambiguity.
Resolve the fixture before measuring anything.

## Phase 3: measure the reference drawing list

Dump the exact fixture's drawing list before writing Go assertions:

```bash
go run ./bindings/wasm -fixture testdata/golden/fixture-NN-<slug>.html \
  -out /tmp/fixture-NN-<slug>.json
```

The dump is the source of truth for the test values. Record, per selected
operation:

```text
kind, x, y, width/height, size, color, font, text
```

For every fixture, also record:

- op count by kind (text, line, fillRect, strokeRect, image)
- canvas width and height
- whether the output contains authored images or fills
- the paint order of the anchors you choose

Select anchors by geometry and meaning:

- one upper-canvas title or heading
- one middle-canvas item or table value
- one lower-canvas or footer value
- a right-aligned total when present
- a rule or border when present
- an image placement when the HTML authors an image

For a tall fixture, use anchors from the top, middle, and bottom of the
canvas. Prefer unique strings near the named anchor.

Do not copy a location from the HTML's CSS, a screenshot, a Chromium PDF, or
another fixture. CSS coordinates, screenshot coordinates, and Chromium PDF
coordinates use different origins and units. The engine dump is the value to
pin.

If the JSON dump cannot be produced, do not guess. Report the missing tool and
use a focused Go probe that prints the same ops.

## Phase 4: write the fixture test

Use the naming pattern:

```text
layout/display_fixture_NN_<slug>_test.go
TestDisplayFixtureNN<PascalCaseSlug>
```

Use the shared shape from `references/test-template.md`:

1. Parse the fixture HTML and apply CSS.
2. Call `layout.DisplayList`.
3. Pin measured authored text, lines, fills, images, op counts, and canvas size.
4. Assert op geometry and paint order for the chosen anchors.

Keep the test parallel-safe. Use `t.Parallel()` when the existing package
allows it. Do not write generated artifacts into `testdata/` from the test;
use a temporary directory.

Use the fixture's own measured values. For example, a fixture-01 test may pin
the title at one location and a table row at another, but those numbers must
not appear in fixture-02's test. Every coordinate belongs to the fixture that
was measured.

The fixture test should fail for both kinds of drift:

- An engine change moves or drops a pinned op.
- A fixture edit changes the authored features the test pins.

The independent pins prevent a stale but self-consistent dump from being the
only oracle. Keep the pins tied to features actually authored by the HTML.

## Phase 5: cover the serializer before scaling out

When this work adds or changes the drawing-list JSON projection
(`bindings/wasm/drawing_list.go`), synthetic tests must cover the op kinds the
serializer claims to support. At minimum, cover:

- every `layout.DisplayOp*` kind, including noop and group boundaries
- text, bullet, fill, stroke, line, image, link, and gridRun payloads
- font and image resource tables with content-derived IDs
- paint order and box projection

A simple fixture may use only text, fills, and lines. That proves the fixture
path, not every branch of a reusable serializer. Add synthetic coverage before
relying on an untested branch for later fixtures.

If the serializer intentionally supports only the ops this repository emits,
state that in its comments and the plan. Do not silently imply broader support
than the contract claims.

## Phase 6: validate the exact final tree

Run formatting first. An empty `gofmt -d` result is required for touched Go
files.

For a single fixture, run the focused proofs with a writable temporary Go
cache:

```bash
review_cache=/tmp/blinkless-fixture-test-cache
GOCACHE="$review_cache" go test ./layout/ -run 'TestDisplayFixtureNN<Slug>$' -count=1
```

Then run the affected packages together:

```bash
GOCACHE="$review_cache" go test ./layout/ ./internal/layout -count=1
```

For a batch or a shared serializer change, finish with the repository gates in
the order required by `AGENTS.md`:

```bash
make test
make golden
make claim-scan
make lint
```

Read the exit status of every command. A cached earlier result does not prove a
later source tree. Re-run the focused fixture test after the last source or
test edit.

Do not run Git commands during this workflow. Report the files inspected and
the validation results without claiming a clean or dirty Git state.

## Phase 7: close the ledger

If the user asked for plan tracking, update the matching phase file only after
the exact fixture proof passes. Keep the row tied to the actual test name and
actual file name. Update the parent ledger and the local knowledge-base when
the plan or behavior is completed, following the repository instructions.

Do not mark later fixtures complete because the shared helper passed on one
fixture. Each fixture needs its own dump, measured coordinates, test case,
and proof command.

## Completion checklist

The task is complete only when all applicable answers are yes:

- The fixture HTML was resolved by complete basename.
- The exact fixture's drawing list was dumped and measured.
- Every location in the test came from that drawing list.
- Anchors cover the top, middle, and bottom regions.
- Authored images, fills, rules, and text features are represented correctly.
- Engine output matches the pinned record.
- Synthetic serializer branches used by the rollout have tests.
- Formatting and focused tests pass on the final tree.
- Full repository gates pass when the scope requires them.
- The plan row and knowledge-base are synchronized when requested by repo
  policy.

## Report shape

```text
Fixture: testdata/golden/<html> ↔ drawing list
Test: layout/<test-file>:<line>
Measured: canvas size, op counts, anchors, authored drawings
Proof: commands and exit results
Remaining: unimplemented fixtures or serializer branches
Git: not inspected or run
```

## Related skills

- `diagnose-golden-fixture` for structural golden failures.
- `diagnose-fixture-picture` for a wrong-looking fixture.
- `phase-wise-checklist` for evidence-backed plan ledgers.
