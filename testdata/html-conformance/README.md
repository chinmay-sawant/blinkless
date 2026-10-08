# HTML conformance corpus

Pinned copy of the html5lib parser test data that `internal/html` measures
against. The engine runner reads `manifest.json`; this file explains what is
pinned, what was left out, and how to refresh it.

## Source

- Repository: https://github.com/html5lib/html5lib-tests
- Pinned revision: `9329e64694e7835d0dcff9811e22856ef6ad16f9` (2026-06-22)
- Tarball URL: https://codeload.github.com/html5lib/html5lib-tests/tar.gz/9329e64694e7835d0dcff9811e22856ef6ad16f9
- Tarball SHA-256: `37d2a0619856234e0631cfb08cf77f32420cba07ad4f3b3871ea991126cc5b57`
- License: MIT. The full text is in `LICENSE`; copyright is 2006-2013
  James Graham, Geoffrey Sneddon, and other contributors.

## Why this revision and not current master

Upstream removed the tree-construction tests from this repository. Commit
`224991ec10db04f056a89eed8b0bd8695fd2950e` ("Tree construction tests have
moved to WPT", 2026-06-26) deleted them, and the upstream README now points
to https://github.com/web-platform-tests/wpt/tree/master/html/syntax/parsing.
Current master `c777c408b61078ea2eb4acefc2535f54dbc8b28a` (2026-10-01) still
carries the tokenizer tests but no tree-construction data.

`9329e646` is the revision just before that deletion, so it is the newest
revision that contains both categories and lets `manifest.json` name a
single revision for the whole corpus.

Trade-off: tokenizer changes landed after 2026-06-22 are not vendored. That
includes tokenizer tests for processing instructions added on 2026-10-01.

## What is vendored

| Path | Files | Contents |
|------|-------|----------|
| `tokenizer/*.test` | 14 | JSON tokenizer tests |
| `tree-construction/*.dat` | 57 | Tree-construction tests, document and fragment |
| `LICENSE` | 1 | Upstream MIT license text |

Not vendored, on purpose: `encoding/`, `serializer/`, `lint/`, `lint_lib/`,
the upstream `README.md` files, and `tree-construction/scripted/*.dat`.

## Selection rules

`manifest.json` sets `"scripting": false`. The `.dat` format marks script
mode per test:

- A case with a `#script-on` line runs only with scripting enabled. These
  are excluded: 8 cases.
- A case with a `#script-off` line runs only with scripting disabled. These
  are included: 27 cases.
- A case with neither marker runs in both modes. These are included.
- `tree-construction/scripted/*.dat` is excluded entirely because that
  directory is for scripting enabled.

Selected total: 1784 of 1792 tree-construction cases. Tokenizer tests do not
carry a script mode.

## Case counts

Tokenizer, counted with `json.load` on every `.test` file:

- 14 files, 6810 test objects, 7036 runs after expanding `initialStates`.
- 4 of the objects live under the `xmlViolationTests` key of
  `tokenizer/xmlViolation.test`; the other 6806 use the `tests` key.

Tree construction, counted by exact `#data` line starts:

- 57 files, 1792 cases.
- 1784 run with scripting disabled (the selection above).
- 192 cases carry a `#document-fragment` section.

## What the engine cannot express yet

`manifest.json` lists these as `unsupported`. Counts overlap (a fragment
case can also need namespaces), so the numbers are not a partition.

- 192 fragment cases: `html.Parse` and `html.ParseDocument` take no context
  element.
- 236 cases whose expected tree uses `svg ` or `math ` element namespace
  designators: `html.Node` has no namespace field.
- 10 cases whose expected tree uses `xlink `, `xml `, or `xmlns ` attribute
  designators: `html.Node.Attrs` is a flat `map[string]string`.
- 111 cases in `template.dat` whose expected tree contains a `content`
  node: template contents are not modeled.
- 166 tokenizer cases with non-default `initialStates` (392 runs): the
  tokenizer always starts in Data state.
- 32 tokenizer cases with `lastStartTag`: no way to seed the last start
  tag.
- 4 `xmlViolationTests` cases: they expect the spec's infoset-coercion
  tweaks, which the engine does not implement.

## How the corpus is measured

- Engine side: `internal/html` runs the corpus in `TestHTMLConformance`.
- Browser side: `scripts/html-conformance-compare.py` feeds each
  tree-construction input to available browsers, dumps a normalized DOM
  tree, and diffs it against the expected tree in the `.dat` file. Browser
  versions, logs, and diffs go under `temps/html-conformance/` (ignored by
  git).
- Browsers expose no tokenizer API, so tokenizer inputs are marked not
  browser-comparable and are only measured by the engine runner.

## Refresh procedure

1. Resolve the target revision with curl, never git:

   ```
   curl -sS https://api.github.com/repos/html5lib/html5lib-tests/commits/master
   ```

2. Download and verify the tarball:

   ```
   curl -sSL -o /tmp/html5lib-tests.tar.gz \
     https://codeload.github.com/html5lib/html5lib-tests/tar.gz/<revision>
   sha256sum /tmp/html5lib-tests.tar.gz
   ```

3. Extract only the contracted paths:

   ```
   tar -xzf /tmp/html5lib-tests.tar.gz -C testdata/html-conformance \
     --strip-components=1 --no-wildcards-match-slash --wildcards \
     '*/tokenizer/*.test' '*/tree-construction/*.dat' '*/LICENSE'
   ```

4. Update `manifest.json` (`upstream.revision`, `unsupported`) and the
   counts in this file. If upstream master has stopped carrying
   tree-construction data, pin the last revision that has both categories
   and say so here.
