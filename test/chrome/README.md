# Chromium Flexbox interaction cases

This directory records 40 high-value Flexbox behavior targets selected from
the local Chromium source checkout. The checkout lives in the ignored
`chromium/` directory at the repository root.

The fixture files directly under `cases/` are static HTML inputs.
`manifest.json` records the behavior target, the expected result, the current
status, and the evidence behind that status. The cases are not a claim that
the Go engine matches Chrome on all 40 inputs.

The manifest is `schema: 2`. Schema 2 adds the `evidence` and `reason` fields;
the case fields from schema 1 are unchanged.

## Status and evidence

Every case carries a `status`:

- `completed`: verified. The case names `evidence` that resolves.
- `scaffold`: the fixture exists, but no behavior test or browser measurement
  is linked yet.
- `blocked`: a known blocker or a measured divergence keeps the case open.
- `unsupported`: the engine does not support the behavior and the case is not
  planned.

The manifest records 26 `completed` and 14 `blocked` cases; no case is
`scaffold` or `unsupported`.

`manifest_test.go` enforces the rule:

- A `completed` case must carry `evidence`.
- Every other status must carry a short `reason`.
- A browser evidence `verdict` of `fail` cannot be marked `completed`.
- Go-test evidence must name a function that exists in the file it cites.
- Browser evidence must name a file that exists and records the case id and
  browser version.

An inventory check alone cannot close a case. Either a named Go test or a
measured browser comparison has to back it.

### Evidence kinds

Go test:

```json
"evidence": {
  "file": "internal/layout/flex_chrome_cases_01_02_test.go",
  "kind": "go-test",
  "test": "TestChromeFlexCase01LegacyAlgorithm"
}
```

Browser measurement:

```json
"evidence": {
  "browser": "Chrome/143.0.7499.40",
  "case": "flex-gap-002-ltr",
  "file": "temps/css-review/geometry-report.md",
  "kind": "browser",
  "verdict": "pass"
}
```

`file` is repository-relative. For a browser measurement, `case` is the case
id used inside the evidence document and `browser` is the measured version.
The browser evidence lives under the gitignored `temps/` directory:
`temps/chrome-cases/geometry-report.md` is the 2026-10-09 run, and
`temps/css-review/geometry-report.md` is the earlier 2026-10-08 run that the
`wpt-align-items-stretch` and `wpt-gap-002-ltr` cases still cite. The browser
check skips when `temps/` is absent (for example on CI); the go-test check
always runs.

## Running the checks

```sh
go test ./test/chrome -count=1
```

The test checks the 40 unique ids, the status vocabulary, fixture presence and
doctype, the `Port status:` marker inside each fixture, evidence resolution,
and source paths when the `chromium/` checkout is present.

## Adding a case

1. Add `cases/case-<nn>-<id>.html`. The file must start with `<!doctype html>`
   and carry a `Port status: <status>. <reason>` comment that matches the
   manifest entry.
2. Add the manifest entry with `id`, `title`, `source`, `combination`,
   `category`, `goTarget`, `expected`, `kind`, `fixture`, `status`, and either
   `evidence` for a `completed` case or `reason` for every other status.
3. Bump `caseCount` to the number of cases and update the expected count in
   `manifest_test.go`.
4. Run `go test ./test/chrome -count=1`.

`scripts/generate_chrome_flex_cases.py` generated the original scaffolds. On a
rerun it preserves each case's `status`, keeps `evidence` on `completed` cases
(refusing to regenerate one without it), and keeps the existing `reason` (or
fills a default) for every other status. It does not invent new evidence, so
update `manifest.json` by hand when a case gains a measurement.

## Notes

- The PDFs under `pdf/` and `cases/pdf/` are stale inspection artifacts from
  the removed writer pipeline. No test regenerates them, and
  `make chrome-cases-pdf` is a disabled stub.
- Chromium C++ assertions are not copied into Go. They inspect Blink
  fragments, constraint spaces, lifecycle state, or scroll state. The porting
  cases use box geometry or browser measurements instead.
- The inventory has 24 `layout-unit` targets, 15 that need a Chrome reference
  or a larger rewrite, and one print-fragmentation target.
- CAT-07 in `plans/0.0.1/html-css-json-compatibility-checklist.md` owns the
  evidence linkage.
