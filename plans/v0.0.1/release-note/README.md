# v0.0.1 release note

This folder is the release note for the renderer cut named in
[RELEASE.md](../../../RELEASE.md). It was written on 2026-10-15 from the 54
commits on `master`, `23061d8` through `c085b34` (2026-10-08 to 2026-10-13).

| File | What it is |
|------|------------|
| [v0.0.1.md](v0.0.1.md) | The note. What a caller gets, what was removed, and the limits. |
| [commits.md](commits.md) | One line for each of the 54 commits, in order. |

Nothing here tags a release, edits `CHANGELOG.md`, or moves the binding
version strings. The GitHub Release tag is `v0.0.1`. `RELEASE.md` calls this cut
0.0.1, and the Makefile stamps `BINDINGS_VERSION` and `WASM_VERSION` as
0.0.1. `bindings/python/pyproject.toml` and `BLINKLESS_VERSION` in
`bindings/c/include/blinkless.h` still say `0.2.6`.
