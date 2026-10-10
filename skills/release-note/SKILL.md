---
name: release-note
description: >
  Cut a blinkless release: promote CHANGELOG and RELEASE.md, write GitHub
  release notes from the previous release, drop "unreleased" language from
  the generic user docs, open the chore release PR. Use when the user says
  "release notes", "draft release", "cut 0.x.y", "promote unreleased", or
  runs /release-note.
---

# Release notes

Promote the next semver. Release work always starts at
[`RELEASE.md`](../../RELEASE.md), which holds the version-source table and the
hard-gate order. Do **not** tag or push `v*` until the user asks. Default
integration branch is **`master`**.

There is no `VERSION` file and no `internal/cli` package in this tree. The
release record is `CHANGELOG.md` plus the ledger under `plans/<ver>/`.

## 1. Collect the delta

Cover **every** change since the previous release (commits, merged PRs,
closed issues). Do not invent features that are only in plans.

Write two bodies:

| File | Role |
|------|------|
| `plans/<ver>/PR/release-v<ver>.md` | GitHub Release body, or `plans/PR/release-v<ver>.md` when the version has no ledger dir |
| `plans/<ver>/PR/pr-release-<ver>.md` | PR body from `skills/PR/PR_TEMPLATE.md`, or `plans/PR/pr-release-<ver>.md` |

Honesty that always stays true unless the engine change says otherwise:

- What ships is the **drawing list** from HTML and CSS (`layout.DisplayList`).
- There is no PDF writer and no page rasterizer; do not claim PDF output.
- The bitmap fallback is one image operation re-encoded as PNG when
  orientation or a clip requires it.

## 2. Version files

The version sources are listed in [`RELEASE.md`](../../RELEASE.md); the
bindings carry their own `BINDINGS_VERSION` / `WASM_VERSION` in the Makefile.

| File | What to set |
|------|-------------|
| [`CHANGELOG.md`](../../CHANGELOG.md) | Move `## Unreleased` into a dated `## <ver> (YYYY-MM-DD)` section. Leave an empty `## Unreleased` on top |
| [`RELEASE.md`](../../RELEASE.md) | Update the version line and the gate list if they moved |

Do **not** touch the bindings version stamps unless the user asks.

## 3. Generic docs - drop "unreleased \<ver\>"

Search and rewrite so the new version is **current**, not "on master /
not in the previous release". Keep product honesty: the output is a drawing
list, not a PDF.

**Always touch when the release changes user-visible status:**

| File |
|------|
| [`README.md`](../../README.md) |
| [`documentation/overview.md`](../../documentation/overview.md) |
| [`documentation/getting-started.md`](../../documentation/getting-started.md) |
| [`documentation/library-api.md`](../../documentation/library-api.md) (`go get …@v<ver>`) |
| [`documentation/fidelity.md`](../../documentation/fidelity.md) |
| [`documentation/deferred.md`](../../documentation/deferred.md) (shipped rows say **Shipped in \<ver\>**, not Unreleased) |
| [`documentation/compatibility-matrix.md`](../../documentation/compatibility-matrix.md) |
| [`output/README.md`](../../output/README.md) |

There is no frontend site and no `docs/` build in this tree.

**Index the release notes:**

| File |
|------|
| `plans/<ver>/README.md` |
| [`plans/README.md`](../../plans/README.md) |

## 4. Leave custom / one-off files alone

Do **not** rewrite these unless the user names them:

- Historical PR bodies under `plans/**/PR/` from earlier releases
- `CONTRIBUTING.md` process language (`## Unreleased` as the next-release slot)
- Benchmark snapshot dates that record a **past** measured binary
  (`testdata/golden/benchmarks/`)

## 5. Verify

Run the gates in the order [`RELEASE.md`](../../RELEASE.md) gives:

```sh
go build ./...
make test-quick
make lint
make claim-scan
```

Search leftover "unreleased" on the generic set only:

```sh
rg -n -i 'unreleased' README.md documentation/overview.md \
  documentation/getting-started.md documentation/library-api.md \
  documentation/fidelity.md documentation/deferred.md \
  documentation/compatibility-matrix.md output/README.md
```

`CHANGELOG.md` may keep an empty `## Unreleased` heading.

## 6. Branch, PR, then stop

Branch `chore/release-<ver>` from `master`. Commit. Push only if the user
asked. Open the PR:

```sh
gh pr create \
  --base master \
  --head "$(git branch --show-current)" \
  --title "chore(release): promote v<ver> and drop unreleased language" \
  --body-file plans/<ver>/PR/pr-release-<ver>.md \
  --assignee "@me" \
  --label documentation \
  --label enhancement
```

Use the `plans/PR/` body path when the version has no ledger dir. After
merge (only if the user asks): tag with a `v` prefix (`git tag v<ver>`) and
paste the release body. See
[CONTRIBUTING.md - Cutting a release](../../CONTRIBUTING.md).
