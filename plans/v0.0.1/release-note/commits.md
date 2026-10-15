# Commits on master for 0.0.1

54 commits, oldest first. Range `23061d8`..`c085b34`. Dates are author dates.
The narrative is [v0.0.1.md](v0.0.1.md). A "docs" commit does not change
layout behavior.

## Reset, 2026-10-08

| Commit | Subject | What changed |
|--------|---------|--------------|
| `23061d8` | Init Commit | Imports gowkhtmltopdf v0.2.6 in one root snapshot. Module `github.com/chinmay-sawant/gowkhtmltopdf`. PDF writer, two CLIs, page images, frontend, `VERSION` `0.2.6`. 2,627 files. |
| `1575f92` | Deleting the older plans | Deletes `plans/0.1.0` through `plans/0.2.7` and the old PR tree. 426 files. |
| `4ad5415` | Deleting the older version and the release as well | Deletes `VERSION` and the old `RELEASE.md`. |
| `df8c81b` | Deleting the frontend and the docs | Deletes `frontend/` and the built `docs/` site. `documentation/` markdown stays. 978 files. |
| `365a1c3` | Updated the plans for the PDF writer removal | Adds `plans/0.0.1/phase-wise-checklist.md`. That plan still expected to keep the page image path. `95afa1f` later removes it. |
| `f3a0270` | Cut the PDF writer and rename the engine to blinkless. | Deletes `internal/pdf` and the PDF convert and paint path. Moves fonts to `internal/fonts`. Renames the module. Keeps a page image command. 840 files, −86,137. |
| `95afa1f` | Remove the page PNG and JPEG encoder and keep the drawing list. | Deletes `internal/imageout`, `screen`, `ImageDocument`, `cmd/blinkless`, `internal/cli`. `layout.DisplayList` calls layout directly. `make golden` runs `TestDisplay`. 90 files, −13,659. |
| `fc8f312` | Clear the lint debt left by the PDF writer and page-encoder removal | Deletes unused font and layout symbols left behind. `make lint` had been red. |
| `1078f00` | Fix CI after the frontend and VERSION removals | Drops the frontend Node job. Sets `BINDINGS_VERSION` and `WASM_VERSION` to `0.0.1`. |
| `db416a7` | Align the bindings with the drawing-list engine | Removes `blinkless_html_to_pdf`. `blinkless_html_to_image` reports that image encoding was removed. Python convert helpers raise `RuntimeError`. Header `BLINKLESS_VERSION` stays `0.2.6`. |

## Logo and compatibility, 2026-10-08 to 2026-10-10

| Commit | Subject | What changed |
|--------|---------|--------------|
| `83c5a5e` | feat: add flat cartoon gopher logo | Adds `assets/01-flat-cartoon.png` and the README image. |
| `1c3692a` | docs: add flat cartoon gopher PR record | Adds `plans/PR/pr-flat-cartoon-gopher.md`. |
| `d5a7ff6` | Merge pull request #1 from chinmay-sawant/feature/gopher-logo | Merges the logo. No engine change. |
| `0e12b15` | docs: plan HTML CSS and JSON compatibility work | Adds the compatibility checklist. Rows start open. |
| `a9381bf` | feat: implement HTML, CSS, and JSON compatibility waves A and B | Invalid declarations lose to valid ones. Flex gap, `fr` tracks, normal line height, root margin collapse, flex float enclosure. `PointsPerPixel` and `PixelPerPoint` swapped to match their names. Tokenizer repairs start. Catalog seeded at 93 implemented / 297 partial / 388 unsupported / 7 ignored, 785 rows. WASM schema `blinkless.drawinglist/1`. |
| `c6ff6dc` | feat: complete HTML CSS and JSON compatibility waves C and D | Doctypes and document mode. SVG and MathML namespaces. Foster parenting and the adoption agency. `text-wrap-style: balance`. Checklist in this commit: tokenizer 6,686 pass / 0 fail, tree construction 1,225 of 1,792. Catalog 90 / 301 / 387 / 7. |
| `9bd5d38` | feat: finish HTML CSS and JSON compatibility phase 3 and closeout | `select`, `template` contents, fragment parsing. Checklist: 192 upstream fragment cases, tree construction 1,631 of 1,792. Abspos root and body anchor to the initial containing block. |
| `6f1a31a` | feat: close the parser corpus and fix the blocked Chrome cases | Script-data, foreign CDATA, `noscript`, ruby and frameset recovery. Checklist: tree construction 1,784 of 1,792 with 0 failures. Border lengths convert to points. Six Chrome cases move. Eight stay blocked on device pixels. Deletes the golden API PDF generator. |
| `6892dc5` | feat: add display-list device-pixel snap and medium border default | Adds `layout.SnapDisplayToDevicePixels`. Widthless `border` is `medium` at 3 CSS px. Eight Chrome cases stay blocked. |
| `f0b7965` | chore: reconcile the ledgers and retire PDF-era tooling | Deletes PDF inspect and print scripts and six Makefile targets. Version check reads the Python and C stamps, not a `VERSION` file. |
| `2f5e111` | docs: add the PR body for the compatibility program | Adds `plans/PR/pr-html-css-json-compatibility.md`. |
| `8b64e43` | fix(ci): vendor the tokenizer corpus and commit the straggler evidence | Stops `*.test` ignore from dropping the tokenizer corpus. Commits the `.test` fixtures. |
| `49950d8` | docs: refresh the PR diff table after the CI fix | Rewrites the diff table in the compatibility PR body. |
| `e238055` | docs: refresh the PR diff table | Same PR body, table only. |
| `f0a81a8` | chore(agents): track the catnotify and explain-output skills | Adds `.agents/skills/catnotify/` and `.agents/skills/explain-output/`. |
| `096c382` | docs: refresh the PR diff table | Same PR body, so the skill files show in the stat. |
| `421019e` | docs: plan the CSS behavior coverage tests for the 428 fixture cells | Adds the coverage checklist. Says the reachable implemented ceiling is 391, not 428. No tests yet. |
| `746d0a0` | chore: rename the plan directory to v0.0.1 and update references | `plans/0.0.1` becomes `plans/v0.0.1`. |
| `d918891` | Merge pull request #2 from chinmay-sawant/feat/html-css-json-compatibility-plan | Merges the compatibility program onto master. |

## Native view proposal and CSS coverage, 2026-10-10

| Commit | Subject | What changed |
|--------|---------|--------------|
| `d44c154` | docs: plan the native web view around the drawing-list engine | Adds the native-window checklist. All implementation rows open. |
| `52881c0` | docs: add the PR body for the native window proposal | Adds `plans/PR/pr-native-window-proposal.md`. |
| `6a45869` | Merge pull request #3 from chinmay-sawant/chore/native-window-proposal | Merges the proposal. No runtime. |
| `6d0a863` | feat: add wave A CSS behavior tests and promote 35 catalog rows | Tests only. Catalog 90/301/387/7 to 125/266/387/7. |
| `092a0af` | feat: add wave B CSS behavior tests and promote 31 catalog rows | Tests only. Catalog to 156/235/387/7. |
| `fcd281d` | feat: apply phase 6 and 7 catalog decisions from read-only probes | Docs and catalog only. `background-attachment` and `box-decoration-break` become unsupported. `page` becomes intentionally ignored. Totals 156/232/389/8. |
| `8cfd978` | feat: add wave D behavior tests, harness smoke tests, promote 34 rows | Tests only. Catalog to 190/198/389/8. |
| `646f421` | docs: close the coverage plan with wave D evidence and final gates | Checklist only. The later waves reopen the work. |
| `5c69266` | docs: close the 22 proven phase rows in the coverage plan | Checklist only. |
| `9b9523a` | feat: add wave E behavior tests and promote 137 catalog rows | Tests only. Catalog to 327/61/389/8. |
| `fc7b502` | feat: add wave F behavior tests and promote 5 catalog rows | Subject says 5. The catalog file moves 12 partial rows, to 339/49/389/8. |
| `8533431` | feat: implement wave G engine behaviors and promote 5 catalog rows | Engine: grid `align-content`, `tab-size` on the fast path, tiled `border-image-repeat`, `text-justify`, `white-space-trim`. Catalog to 344/46/387/8. |
| `354841f` | chore: rename waveG test files and symbols to proper names | Renames only. Counts stay. |
| `e2bbb31` | feat: implement wave H engine behaviors and promote 9 catalog rows | SVG stroke properties baked into the inline SVG PNG. `list-style-image` inherits. Catalog file 353/37/387/8. The message's unsupported count of 389 does not match the file. |
| `77c7afe` | feat: wire ruby, clip-rule, string-set, bookmarks, footnotes; promote 5 rows | Ruby align, merge, overhang, and position. `clip-rule: evenodd` on clips. String-set, bookmarks, and footnotes stay unwired as page features. Catalog 358/32/387/8. |
| `d9545aa` | Wave L: CSS behavior coverage to 381 implemented | 23 partial rows become implemented. File is 381/9/387/8. Includes `pretty`, column breaks, dark `color-scheme`, variable fonts, and stored `scroll-margin`. No separate wave I, J, or K commit. |
| `64dc880` | Wave M: 3D, fragmentation and wide-gamut builds to 389 implemented | 2D-projected perspective and backface culling, multicol orphans and widows, `avoid-short-last-line`, wide-gamut color with `dynamic-range-limit`. Catalog 389/1/387/8. The one partial row is `break-inside`. |
| `1cd0136` | Showcase: generated CSS demo pages for all 389 implemented properties | Adds `documentation/css-showcase.html` and `documentation/css-engine-only.html`, plus the generator scripts. No engine change. |
| `6e22288` | docs: add PR body for css behavior coverage tests | Adds `plans/PR/pr-css-behavior-coverage-tests.md`. |
| `5adb219` | chore: rename gap behavior tests to content names | Renames test files. Counts stay 389/1/387/8. |
| `34d9386` | Merge pull request #4 from chinmay-sawant/feature/css-behavior-coverage-tests | Merges the coverage branch. |

## Masks, 2026-10-13

| Commit | Subject | What changed |
|--------|---------|--------------|
| `d719e62` | docs(plan): add CSS masking first-wave checklist | Plan only. 10 standard names and 8 WebKit aliases, all unsupported at that moment. |
| `30971e1` | feat(css): add first-wave mask property support | Mask layers on element groups in the Go list and in WASM JSON. SVG mask fragments can become an alpha PNG. Catalog 389/19/369/8. Status of the 18 names is partial. The host composites pixels. |
| `c8d02ef` | docs(pr): record CSS masking first-wave PR | Adds `plans/PR/pr-css-masking-first-wave.md`. |
| `c085b34` | Merge pull request #5 from chinmay-sawant/feat/css-masking | Merges the mask wave. Tip of `master` for this note. |
