# Contributing to blinkless

Thanks for helping improve this pure-Go HTML and CSS layout engine.

This document covers **how to contribute** (setup, tests, PRs, layout QA).
Product design, fidelity tiers, and the support matrix live under
[`documentation/`](documentation/README.md). Implementation ledgers live under
[`plans/`](plans/README.md).

---

## Ground rules

1. **Pure Go, no CGO** for the main path. Direct third-party modules stay on the
   allowlist enforced by `internal/fonts.TestDirectModuleAllowlist` (OpenType
   shaping and SVG raster via `github.com/go-text/typesetting` and
   `github.com/tdewolff/canvas`; see `Makefile` / `go.mod`).
2. **Generic engine fixes**: prefer CSS/print semantics over site-specific
   (e.g. MediaWiki class) hacks. Operator policy belongs in `internal/settings`
   and the public `css.Options` / `layout.Options`, not cascade overrides for
   one skin.
3. **Honesty**: update `documentation/compatibility-matrix.md` and
   `documentation/fidelity.md` when behavior claims change. `make claim-scan`
   polices the wording in `doc.go`, `README.md`, and `documentation/`.
4. **Stable drawing list**: the same HTML, CSS, and fonts should produce the
   same operations. Do not treat the leftover rasters under `output/` as
   golden masters; `make golden` is the drawing-list gate.

---

## Development setup

```sh
git clone https://github.com/chinmay-sawant/blinkless.git
cd blinkless
make test
make build
```

| Target | Purpose |
|--------|---------|
| `make test` | Full `go test ./...` with capped concurrency (`-p 2 -parallel 2`) so ~8 GiB machines do not thrash |
| `make test-unit` | Same caps, every package except `internal/convert` (pair with `make golden` for layout or convert work) |
| `make test-quick` | `make test` plus `-short` (skips long perf-budget tests) |
| `make test-serial` | `-p 1 -parallel 1` when even capped runs freeze the desktop |
| `make test-race` | `-race` on the hot packages (`./internal/layout`, `./internal/load`, `./internal/fonts`), same concurrency caps |
| `make lint` | `golangci-lint run` (pinned v1.64.8, all linters via `.golangci.yml`), then `make size-check` (file-size ledger) |
| `make build` | `CGO_ENABLED=0 go build ./...`; compiles every package and writes no binary |
| `make golden` | Public drawing-list tests (`TestDisplay*`) in `./layout`, capped parallelism |
| `make samples` | Alias for `make golden`; writes nothing (there is no page raster) |
| `make fmt` | `gofmt -w .` |

Raise concurrency only when you have RAM: `make test TEST_P=4 TEST_PARALLEL=4`.
Do not run bare `go test ./...` on a many-core laptop with ~8 GiB; that is what freezes the machine.

Minimum: Go 1.26 or newer; `go.mod` pins the toolchain (`go1.26.4`).

---

## Branch and PR workflow

1. Branch from **`master`** (default integration branch).
2. Keep commits focused; prefer conventional-style messages:
   `fix(layout): ...`, `feat(css): ...`, `docs: ...`, `test: ...`.
3. Open a PR **to `master`** with:
   - Self-assign (`--assignee @me`)
   - At least one label (`bug`, `enhancement`, `documentation`, ...)
   - Body based on [`skills/PR/PR_TEMPLATE.md`](skills/PR/PR_TEMPLATE.md)
   - Filled body copy under `plans/PR/pr-<slug>.md`
4. Do **not** merge your own PR unless maintainers agree.

```sh
gh pr create \
  --base master \
  --head "$(git branch --show-current)" \
  --title "fix(layout): short imperative description" \
  --body-file plans/PR/pr-<slug>.md \
  --assignee "@me" \
  --label bug \
  --label enhancement
```

Issue templates: [`skills/PR/ISSUE_TEMPLATE.md`](skills/PR/ISSUE_TEMPLATE.md).

---

## Where to change code

| Concern | Package |
|---------|---------|
| Load / HTTP / ACL | `internal/load` |
| HTML parse | `internal/html` (public `html`) |
| CSS cascade | `internal/css` (public `css`) |
| Layout, floats, tables, page breaks | `internal/layout` (public `layout`) |
| Fonts / shaping | `internal/fonts` |
| SVG / image-op fallback | `internal/svg`, image ops in `internal/layout` |
| Settings | `internal/settings` |
| End-to-end convert | `internal/convert` (`prepare`, `render`) |
| Public library API | `html`, `css`, `layout` |

Pipeline: **load -> html.Parse -> css.Apply -> internal/layout drawing list**.
The public `html`, `css`, and `layout` packages stop at the drawing list:
no page raster and no PDF. The list is the output.

---

## Testing expectations

### Required for layout/print changes

```sh
go test ./internal/layout -count=1
make golden
make lint
```

Add or extend unit tests next to the code you touch
(e.g. `internal/layout/*_test.go`). Prefer **regression tests** that fail
before the fix for pagination, tables, floats, and avoid-packing bugs.

### Visual QA (layout regressions)

Drawing-list tests do **not** catch text overlap, underline noise, or table
chrome. For print/layout work:

1. Run `make golden` (public drawing-list tests, `TestDisplay*` in `./layout`).
2. Run the focused `internal/layout` test for the behavior you touched
   (`go test ./internal/layout -run '<TestName>' -count=1`).
3. For flexbox behavior, check the browser-backed cases under
   [`test/chrome/`](test/chrome/README.md); their evidence lives in
   `test/chrome/evidence/`.

`make samples` is an alias for `make golden` and writes nothing, so there is
no page raster to inspect.

### Dangerous patterns (layout)

- **Document-global Y shifts** of all ops below a baseline - easily
  interleaves body paragraphs. Prefer scoped shifts instead.
- **Site-specific CSS** injected for one website skin; put operator policy in
  `internal/settings`, not cascade overrides for one skin.
- **Weakening** `page-break-inside: avoid` without tests for blank-page
  cascades on dense lists.

---

## Cutting a release

CI on PRs and ordinary branch pushes does **not** publish anything. A release
starts when a **`v*` git tag** is pushed (workflow
[`.github/workflows/release.yml`](.github/workflows/release.yml)).

1. Bump `[project].version` in
   [`bindings/python/pyproject.toml`](bindings/python/pyproject.toml) and
   `BLINKLESS_VERSION` in
   [`bindings/c/include/blinkless.h`](bindings/c/include/blinkless.h) to the
   new semver (for example `0.2.6`). `make check-versions` gates the two
   sources against each other.
2. Move notes under `## Unreleased` in [`CHANGELOG.md`](CHANGELOG.md) into a
   dated section for that version.
3. Merge the release prep to the default branch.
4. Tag and push (the tag must match both version sources, with a `v` prefix):

```sh
git tag v0.2.6
git push origin v0.2.6
```

The release workflow then:

- Refuses to publish when the tag does not match the committed version
  sources (`scripts/check_versions.sh`; there is no `VERSION` file)
- Runs the [`RELEASE.md`](RELEASE.md) hard gates: `make build`,
  `make test-quick`, `make golden`, `make lint`, `make claim-scan`
- Creates or updates the GitHub Release for that tag with generated notes.
  No binaries are attached.

Python wheels publish from
[`.github/workflows/publish-pypi.yml`](.github/workflows/publish-pypi.yml).

Creating a GitHub Release in the UI with a **new** `v*` tag also pushes the tag
and runs the same workflow. The workflow does **not** invent tags on its own.

---

## Documentation updates

When your change affects user-visible behavior:

| Change type | Update |
|-------------|--------|
| CSS / element support | `documentation/compatibility-matrix.md` |
| Fidelity claims / tiers | `documentation/fidelity.md` |
| Fonts / shaping | `documentation/fonts.md` |
| Library API | `documentation/library-api.md` |
| Settings keys | `documentation/architecture/03-settings.md` |
| Performance numbers / benches | `testdata/golden/benchmarks/README.md` |
| Deferred / not-planned features | `documentation/deferred.md` |
| Security / ACL | `documentation/THREAT-MODEL.md`, `documentation/integration-security.md` |
| User-facing release notes | `CHANGELOG.md` (Unreleased or next version) |

Implementation plans under `plans/` are for maintainers/agents; keep them in
sync only when you close a phase item.

---

## Coding notes

- Run `gofmt` on all Go you touch (`make fmt` / `make lint`).
- Match existing package style; keep modules deep (small public surface).
- Do not commit secrets, large unrelated binaries, or personal paths.
- Do not add generated artifacts: `output/` holds leftovers from the removed
  pipeline and nothing in the tree rebuilds them (see `output/README.md`).

---

## Reporting bugs

Include:

- Commit hash (and the binding version from `bindings/python/pyproject.toml`
  or `bindings/c/include/blinkless.h` when you use a binding)
- Minimal HTML (or fixture path) and the exact library call
- Expected vs actual; for layout bugs, a drawing-list diff or the matching
  browser case under `test/chrome/` helps
- OS and Go version

Security issues: prefer a private report if the bug is load/SSRF related; see
[`documentation/THREAT-MODEL.md`](documentation/THREAT-MODEL.md).

---

## License

By contributing, you agree that your contributions are licensed under the
project's [MIT License](LICENSE).
