# v0.0.2 security release

0.0.2 is the security release after the published `v0.0.1` renderer cut.
The note covers `master` from `6939166` through `9792e54` (2026-10-15):
the resource caps, the loader checks, and the removal of the Python package.

| File | What it is |
|------|------------|
| [release-note/v0.0.2.md](release-note/v0.0.2.md) | The note. What changed, what is still uncapped, and what does not ship. |
| [release-note/commits.md](release-note/commits.md) | One line for each commit after `v0.0.1`. |
| [PR/release-v0.0.2.md](PR/release-v0.0.2.md) | GitHub Release body. |
| [PR/pr-release-v0.0.2.md](PR/pr-release-v0.0.2.md) | Pull request body. |

The GitHub Release tag is `v0.0.2`. `RELEASE.md` names this cut 0.0.2.
`Makefile` `BINDINGS_VERSION` and `WASM_VERSION` stay `0.0.1`. There is no
`VERSION` file and no Python package.
