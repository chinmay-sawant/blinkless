## Summary

GitHub Actions now runs `make test` and `make lint` and nothing else. The tag-versus-source version check is gone, so a `v*` tag is no longer rejected when it disagrees with `bindings/python/pyproject.toml` or `bindings/c/include/blinkless.h`. The Python package and the PyPI publish workflow are removed with that check.

## Motivation / context

- Plans: no new plan file. The live contract is `AGENTS.md` and `CONTRIBUTING.md`.
- Issues: no matching GitHub issue exists. The request was to simplify CI after the version mismatch and to drop the Python release path.

## Changes

### GitHub Actions

- `.github/workflows/ci.yml` is one job: checkout, Go 1.26, `make test`, install golangci-lint v1.64.8, `make lint`.
- `.github/workflows/release.yml` on a `v*` tag runs those same two commands, then creates or updates the GitHub Release. It does not call `scripts/check_versions.sh`.
- `.github/workflows/publish-pypi.yml` is deleted.

### Python package

- `bindings/python` is deleted, including `pyproject.toml`, the ctypes loader, tests, and the invoice example.
- `scripts/check_versions.sh` and `scripts/build_cshared_for_wheel.sh` are deleted.
- Makefile targets `check-versions`, `python-binding-test`, and `python-benchmarks` are removed.
- `bindings/c`, `bindings/wasm`, and the local `make c-shared` target stay. CI does not build them.

### Docs that stated the old contract

- `AGENTS.md`, `CONTRIBUTING.md`, `CHANGELOG.md` (intro and Unreleased only), `skills/release-note/SKILL.md`, `documentation/deferred.md`, `documentation/architecture/01-entrypoints-cli.md`, `documentation/compatibility-matrix.md`, `output/README.md`, and `testdata/golden/benchmarks/README.md`.
- Older `CHANGELOG.md` release sections and the closed checklists under `plans/` still describe the Python package as it existed then.

## Impact

| Area | Impact |
|------|--------|
| Performance | CI does less work. The engine is unchanged. |
| Memory | None in the library. |
| Behavior / correctness | Go layout output is unchanged. Publishing a `v*` tag no longer fails on a version-file mismatch. |
| API / CLI | The PyPI package `blinkless` and `bindings/python` are gone. `bindings/c` and `bindings/wasm` stay. |
| Dependencies | No `go.mod` change. |
| Binary size / build time | CI no longer builds the c-shared library, WASM, or the race job. |

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| `bindings/python` and PyPI wheels | Call the Go module `github.com/chinmay-sawant/blinkless` instead. |
| `make check-versions` and `scripts/check_versions.sh` | Do not run them. A tag is not checked against a version file. |
| `make python-binding-test` and `make python-benchmarks` | Those targets are gone. |

## Test plan

- [ ] `make test` not run. This session was asked not to run the heavy suite.
- [ ] `make lint` not run, for the same reason.
- [ ] `make build` not run. No new Go API.
- [ ] `make golden` not run. Layout and paint code were not edited.
- [ ] `make claim-scan` not run. Documentation sentences about the Python package were edited.

### Commands

```sh
bash scripts/pr-diff-stat.sh master
```

GitHub CI on this PR is the check: `make test` and `make lint`.

## Screenshots / sample output

```
no visual surface changed
```

## Related issues

- None. No GitHub issue is linked.

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled. There is no ticket ID.
- [x] Filled body committed under `plans/PR/pr-ci-cleanup.md`

## Follow-ups (out of scope)

- Python helper scripts stay: `scripts/css-catalog-map.py` and the Chrome harness. They are not the package.
- Closed rows under `plans/` still mention `bindings/python` as history.
- `RELEASE.md` still lists local gates beyond the two CI commands.

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
- [ ] Diff-stat-by-extension table pasted at the bottom

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.in` | 1 | 0 | 7 |
| `.md` | 10 | 61 | 104 |
| `.py` | 11 | 0 | 1849 |
| `.sh` | 3 | 0 | 81 |
| `.toml` | 1 | 0 | 100 |
| `.typed` | 1 | 0 | 1 |
| `.yml` | 4 | 14 | 395 |
| No extension | 2 | 8 | 39 |
| **Total** | **33** | **83** | **2576** |
