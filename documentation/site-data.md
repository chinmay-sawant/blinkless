# Site data

The site under `frontend/` reads two generated JSON files.
`scripts/export-frontend-data` writes both from the engine and the CSS
catalog. The site build is outside the engine build: `make test` and
`make lint` do not touch it, and the built output under `docs/` is committed
separately.

## Inputs

- `testdata/css/catalog/properties.json` is the CSS catalog source. The
  exporter keeps only rows whose status is `implemented` or `partial`
  (`scripts/export-frontend-data/main.go:317-319`).
- `specimenHTML` is a fixed document declared in the exporter
  (`scripts/export-frontend-data/main.go:65`). It is parsed with `html.Parse`,
  styled with `css.Apply` at 420 CSS pixels wide, and laid out with
  `layout.DisplayList` (`scripts/export-frontend-data/main.go:105-141`).

## Outputs

| File | Contents |
|------|----------|
| `frontend/src/data/specimen.json` | One `layout.DisplayList` for the fixed document: the HTML, canvas size, pixel-per-point factor, and rounded operations (`scripts/export-frontend-data/main.go:131-141`) |
| `frontend/src/data/catalog.json` | The implemented and partial catalog rows: name, status, values, note, and `implementation_scope` (`scripts/export-frontend-data/main.go:287-293`, `:306-340`) |

Both files are written with mode 0644 (`scripts/export-frontend-data/main.go:30-32`,
`:371`).

## Regeneration

```sh
go run ./scripts/export-frontend-data
```

The command is deterministic. Run it twice and the second run must leave no
diff. Commit the two JSON files with the change that alters the engine output
or the catalog.

Rules encoded in the exporter:

- `implementation_scope` is carried from the catalog row to the site entry
  (`scripts/export-frontend-data/main.go:295-304`, `:323-329`). Commit
  `1c68b28` added the field so the 43 Blinkless+OwnFrame rows keep their badge
  and count when the catalog is regenerated.
- Every non-ASCII rune becomes a `\uXXXX` escape. A rune above U+FFFF uses a
  UTF-16 surrogate pair (`scripts/export-frontend-data/main.go:381-399`). The
  committed JSON is ASCII-only, so regeneration stays diff-free.
- A catalog note is the first limitation line, with the pinned-upstream marker
  removed, cut at 180 bytes with an ellipsis
  (`scripts/export-frontend-data/main.go:41`, `:343-360`).
- Pixel coordinates round to one decimal
  (`scripts/export-frontend-data/main.go:38-39`, `:278`).

## Building the site

`frontend/package.json` runs `tsc && vite build`. `frontend/vite.config.ts:9`
writes the output to `../docs`. The built site is not part of the engine
build or the CI test gates. `make lint-frontend` is a no-op echo target
(`Makefile:72-73`).

## Verification

There is no Makefile target for the export. The check is a clean second run:

```sh
go run ./scripts/export-frontend-data
go run ./scripts/export-frontend-data
git diff --exit-code frontend/src/data
```

A clean exit from `git diff` means the exporter reproduced the committed
files. Rebuild `docs/` when the site is part of the change.
