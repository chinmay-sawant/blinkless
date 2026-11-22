# feat(fonts): fall back to installed system fonts for CJK text

Filled copy of [`skills/PR/PR_TEMPLATE.md`](../../skills/PR/PR_TEMPLATE.md).

- Base: `master`
- Head: `feature/native-platform-cjk-font-fallback`

---

## Summary

Han, kana, Hangul, and fullwidth characters that no CSS family, bundled face, or opt-in registry face covers now resolve to an installed operating-system face instead of a missing-glyph box. The fallback is on by default and reads a short per-OS list of font files, lazily, only when such a character appears.

---

## Motivation / context

- The bundled faces (Liberation, DejaVu) carry no CJK glyphs, and system font discovery is opt-in, so Japanese text typed into an ownframe field rendered as boxes on a machine that has Japanese fonts installed.
- Windows ships its Japanese faces as `.ttc` collections, which the font scan did not read, and `C:\Windows\Fonts` was deliberately left out of the default scan paths.
- Issues: none.

---

## Changes

### Fonts (`internal/fonts`)

- `system_fallback.go`: per-OS candidate discovery. Windows reads the font folder under `%WINDIR%` for Yu Gothic, Meiryo, MS Gothic, BIZ UD Gothic, Microsoft YaHei and JhengHei, SimSun, and Malgun Gothic, with bold variants. macOS reads Hiragino, PingFang, Apple SD Gothic Neo, and Arial Unicode. Linux and Android scan the font roots by file name for TrueType CJK families. WSL also reads `/mnt/c/Windows/Fonts`.
- It serves only CJK and fullwidth code points (`systemFallbackRune`), so a Latin document never reads a font file.
- Each file is read and parsed once, on the first lookup that needs it, and each answer, including a miss, is cached per code point.
- `ttc.go`: `SplitTTC` rebuilds each member of a TrueType collection as a standalone SFNT, so a shaper that loads one face per byte slice keeps working. CFF/`OTTO` faces are still skipped, so Noto Sans CJK, which ships as CFF on most Linux distributions, is not used there.

### Layout (`internal/layout`)

- `registryGlyphFallback` consults the system fallback after the opt-in registry. It runs only when nothing else covers the rune.
- The layout tests pin the bundled faces (`TestMain`), so glyph advances and op counts do not depend on the fonts installed on the test machine.

### Public API (`layout`)

- `layout.SetSystemFontFallback(bool)` and `layout.SystemFontFallbackEnabled()`, plus `BLINKLESS_SYSTEM_FONTS=off` to start with it disabled.

### Docs

- `documentation/fonts.md` and `CHANGELOG.md` describe the fallback, its per-OS file list, and its limits.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | No cost for text without CJK. The first CJK character reads one font file (about 10 MB for Yu Gothic) and parses it; measured at 0.17 s including the file reads on the test machine. Later lookups are map hits. |
| **Memory** | A loaded system face stays resident for the life of the process. |
| **Behavior / correctness** | CJK text that used to render as boxes now renders with the installed face. Output now depends on the fonts installed on the machine; a host with no CJK font still shows boxes. Pin or bundle fonts with `@font-face` when a document must render the same everywhere. |
| **API / CLI** | Two new functions in `layout`; one environment variable. |
| **Dependencies** | None. |
| **Binary size / build time** | No change; no font is bundled. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| Documents with CJK text can lay out differently on a machine that has a CJK font | Call `layout.SetSystemFontFallback(false)` or set `BLINKLESS_SYSTEM_FONTS=off` to keep the old behavior. |

---

## Test plan

- [x] `make test` (`go test ./...`)
- [x] `make lint` (golangci-lint and the file-size gate)
- [x] `make claim-scan`
- [ ] `make golden`: not run; no fixture or paint behavior changed because the layout tests pin the bundled faces.

### Commands

```sh
go test ./internal/fonts ./internal/layout ./layout
make lint
make claim-scan
```

Verified end to end on Windows 11 with Microsoft Japanese IME: `examples/login` in ownframe now shows the typed and committed text as real glyphs where it showed boxes before. `TestSystemFallbackHostCJK` and `TestSystemFallbackFaceForCJK` run against the host's real fonts and skip where none exist. The macOS and Linux candidate lists follow platform conventions and are not exercised on those hosts.

---

## Screenshots / sample output

```
no visual surface changed in this repository
```

---

## Related issues

- Relates to ownframe's native-platform work (Windows IME), which exposed the missing glyphs.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled
- [x] Filled body committed under `plans/PR/pr-system-cjk-font-fallback.md`

---

## Follow-ups (out of scope)

- ownframe bumps its blinkless pin (`go.mod` and `examples/go.mod`) once this is merged and tagged.
- A bundled CJK face (for example Noto Sans JP) would make rendering identical everywhere, at several MB.
- A per-glyph language preference for ambiguous Han characters.

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 9 | 907 | 4 |
| `.md` | 3 | 165 | 3 |
| **Total** | **12** | **1072** | **7** |
