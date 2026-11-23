# Settings system and errors

`internal/settings` is the wkhtmltopdf-compatible settings model and typed structs. The layout pipeline reads only part of it: load policy, font paths, and CSS media selection. PDF version, PDF profile, copies, outlines, headers, footers, and TOC fields still exist on the structs, but nothing in this tree consumes them. There is no PDF writer.

## 1. Responsibility & position in the pipeline

The package is the single source of truth for user-facing knobs: page geometry, margins, headers/footers, TOC behaviour, load policy (proxy, ACL), web behaviour, and image settings, with wkhtmltopdf-compatible defaults.

Its consumers today:

```text
settings.PdfGlobal            -> internal/fonts          (font path scan, registry)
settings.LoadGlobal/LoadPage  -> internal/load           (loader policy)
settings.LoadPage             -> internal/convert/prepare (document load)
settings.MediaType / DefaultPdfGlobal / LoadPage -> css   (sheet collection)
settings.LoadGlobal/LoadPage  -> bindings/wasm           (data: image loader)
```

Nothing in the engine reads argv. The structs are the typed payload the consumers above read.

A second, smaller package is in scope: `internal/errs` owns the shared nil-context sentinel `ErrNilContext` and a parked, deprecated `ErrNilCommand` (`internal/errs/errs.go:21-26`). `ErrNilLoader` moved to `internal/load` (`load.go:71`). `errs` imports only `errors`.

## 2. Package / file map

| File | Responsibility | Approx. lines |
|------|----------------|---------------|
| `internal/settings/settings.go` | Typed model and parsers: `PdfGlobal`, `PdfObject`, `ImageGlobal`, sub-structs (`Web`, `LoadGlobal`, `LoadPage`, `HeaderFooter`, `TableOfContent`, `Margin`, `Size`, `CropSettings`), enums (`Orientation`, `LoadErrorHandling`, `MediaType`), `ParsePDFVersion` / `ParsePDFProfile`, `HeaderFor` / `FooterFor`, wkhtml-compatible defaults | 523 |
| `internal/settings/clone.go` | `ClonePdfGlobal`, `ClonePdfObject`, `CloneImageGlobal`, `CloneHeaderFooter`; copies slices and maps | 65 |
| `internal/settings/object_roles.go` | `StampCover`, `StampTOC` object flags | 28 |
| `internal/settings/pagesize.go` | Static ISO/ANSI page-size table; `ParsePageSize` | 63 |
| `internal/settings/unitreal.go` | `UnitReal` scalar with unit suffix; `Points`/`Mm` conversion; `ErrInvalidUnitReal` | 105 |
| `internal/settings/httperror.go` | `HttpStatusError` (HTTP status + URL) and `HttpErrorCode` (404 -> 2, 401 -> 3, else 1) | 41 |
| `internal/settings/doc.go` | Package doc | 20 |
| tests | `settings_test.go` (274), `clone_test.go` (93), `object_roles_test.go` (81), `options_test.go` (61) | 509 |
| `internal/errs/errs.go` | Shared sentinels | 27 |

## 3. Key types, functions & entry points

### 3.1 The three settings structs

| Type | Purpose | File:line |
|------|---------|-----------|
| `PdfGlobal` | Global settings: page geometry, orientation, `PdfVersion` / `PdfProfile`, grayscale, copies/collate, outline, title, margins, header/footer, TOC, background, load policy, font paths | `settings.go:358` |
| `PdfObject` | One page/cover/toc object: `Page`, link flags, outline inclusion, object header/footer overrides (`HeaderSet`/`FooterSet`), `LoadPage`, `Web`, `UseOutline` | `settings.go:427` |
| `ImageGlobal` | Image-mode settings: width/height/quality, smart width, crop, format, transparency, `Web`, `LoadGlobal` | `settings.go:488` |

Supporting sub-structs (all in `settings.go`):

| Type | Purpose | File:line |
|------|---------|-----------|
| `Margin` | Four page margins in millimetres | `settings.go:238` |
| `Size` | Custom page dimensions in mm; 0 = unset | `settings.go:251` |
| `Web` | Web behaviour: `Images`, `PrintMediaType`/`MediaType`, `SimplifyDOM(+Profile)`, `PrintLinkUnderline` | `settings.go:260` |
| `LoadGlobal` | Shared load policy: `Proxy`, `Allow` prefixes, `EnableLocalFileAccess`, network-policy fields | `settings.go:278` |
| `LoadPage` | Per-page load policy: zoom, block-local-access, load-error handling, auth, headers, media, timeout, `InlineHTML`/`InlineBase` | `settings.go:293` |
| `HeaderFooter` | Text/HTML header and footer: font, left/right/center, line, spacing, `HTMLURL`, `Replace` | `settings.go:311` |
| `TableOfContent` | TOC settings: font scale, indentation, dotted lines, caption, links, XSL | `settings.go:333` |
| `CropSettings` | Image crop box (Left/Top/Width/Height, -1 = unset) | `settings.go:502` |

### 3.2 Enums and parsers

| Symbol | Purpose | File:line |
|--------|---------|-----------|
| `Orientation` | Portrait/landscape enum | `settings.go:169` |
| `LoadErrorHandling` | abort/skip/ignore enum | `settings.go:188` |
| `MediaType` | `MediaUnset` (zero), `MediaScreen`, `MediaPrint`; the zero value stringifies as `ignore` | `settings.go:212` |
| `ParsePDFVersion` | `""`/`1.4`/`1.7`/`2.0` | `settings.go:142` |
| `ParsePDFProfile` | Aliases (`a3a-ua1`, `a4-ua2`, ...) to canonical tokens (`PDF/A-3a+PDF/UA-1`, ...) | `settings.go:64` |
| `IsPDFA3` / `IsPDFA4` / `IsPDFUA1` / `IsPDFUA2` / `IsPDFUA` | Profile class checks | `settings.go:118-138` |

### 3.3 Defaults (wkhtmltopdf `pdfsettings.cc` / `imagesettings.cc` compatible)

| Function | Notable defaults | File:line |
|----------|------------------|-----------|
| `DefaultMargins()` | 10 mm all sides | `settings.go:246` |
| `DefaultHeaderFooter()` | Arial, font size 12, spacing 0 | `settings.go:324` |
| `DefaultTableOfContent()` | font scale 0.8, indentation `1em`, dotted lines true, caption "Table of Contents" | `settings.go:344` |
| `DefaultPdfGlobal()` | A4 portrait, 1 copy collated, outline depth 4, compression on, smart shrinking on, background on, images on, margins 10 mm, resolve-relative-links on | `settings.go:402` |
| `DefaultPdfObject()` | external/local links on, outline flags on, block-local-file-access true, load-error abort | `settings.go:463` |
| `DefaultLoadPage()` | `BlockLocalFileAccess: true`, `LoadErrorHandling: LoadErrorAbort` | `settings.go:479` |
| `DefaultImageGlobal()` | width 1024, quality 94, smart width on, crop = (-1,-1,-1,-1) | `settings.go:510` |

### 3.5 Units, page sizes, HTTP errors

| Symbol | Purpose | File:line |
|--------|---------|-----------|
| `UnitReal` | Scalar + unit suffix (`mm cm m in pt px em rem ex ch %`) | `unitreal.go:21` |
| `ParseUnitReal(raw, impliedUnit)` | Suffix detection + `strconv.ParseFloat`; `ErrInvalidUnitReal` on failure | `unitreal.go:31` |
| `(u UnitReal) Points()` | PDF points at 96 px/in; `%` and font-relative units return `ok=false` | `unitreal.go:69` |
| `(u UnitReal) Mm()` | Millimetres | `unitreal.go:99` |
| `pageSizes` / `ParsePageSize(name)` | Fixed 23-entry ISO/ANSI table in points; empty -> A4 | `pagesize.go:19` / `pagesize.go:51` |
| `HttpStatusError` | Load failure carrying status + URL; produced by `internal/load` | `httperror.go:16` |
| `HttpErrorCode(status)` | 404 -> 2, 401 -> 3, else 1 | `httperror.go:27` |
| `(e *HttpStatusError) HttpErrorCode()` | Satisfies the exit-code interface check | `httperror.go:39` |

### 3.6 Object role stamps

| Symbol | Purpose | File |
|--------|---------|------|
| `StampCover` | `IsCover`, outline exclusion, empty header/footer override | `object_roles.go:5` |
| `StampTOC` | TOC object flags (`IsTableOfContent`, outline flags false) | `object_roles.go:19` |

`HeaderFor` / `FooterFor` (`settings.go:445` / `settings.go:454`) return the object override or the global header/footer.

### 3.7 Sentinel errors

`internal/errs` now holds two sentinels (`internal/errs/errs.go`):

| Symbol | Purpose | File:line |
|--------|---------|-----------|
| `ErrNilContext` | Shared nil-context guard; consumers include load, prepare, render, and layout | `errs.go:23` |
| `ErrNilCommand` | Parked and deprecated; no consumer in this tree | `errs.go:26` |

`ErrNilLoader` lives in `internal/load` (`load.go:71`).

## 4. Data & control flow

### 4.1 CSS sheet collection (the main production path)

`css.Apply` (`css/css.go:97`) is the caller that exercises the most settings:

1. `mediaType` maps `Options.Media` to `settings.MediaScreen` / `settings.MediaPrint` (`css.go:146-155`).
2. `collect` builds `settings.DefaultPdfGlobal()`, sets `Web.MediaType`, and constructs a loader from `global.Load` (`css.go:166-172`).
3. It passes `settings.LoadPage{MediaType: media}` into `prepare.NewResourceContext` (`css.go:175-176`).
4. `prepare.Document` forwards that `LoadPage` to `loader.Load` (`prepare.go:210-222`).

### 4.2 Font registry

`internal/fonts.RegistryFromGlobal(settings.PdfGlobal)` reads `FontPaths` and `UseSystemFonts` to build a registry (`internal/fonts/registry.go:440`). `LogFontRegistryScan` emits the scan notice (`registry.go:33`). The layout tests are the callers today.

### 4.3 Browser adapter

`bindings/wasm/resources.go` builds a loader from a zero-value `settings.LoadGlobal` (`resources.go:38`) and fetches `data:` images with a zero-value `settings.LoadPage` (`resources.go:45`).

### 4.4 No dotted Set/Get surface

The dotted `Set`/`Get` surface, its descriptor tables, and the accepted-but-inert key tables were removed. Callers read and write the typed structs directly.

## 5. Cross-package dependencies

### 5.1 What `internal/settings` imports

Stdlib only: `errors`, `fmt`, `strings`, `strconv`, `math`, `maps`, `slices`. No internal imports. `internal/errs` imports only `errors`.

### 5.2 Who depends on them

| Package | Direction of use |
|---------|------------------|
| `internal/load` | Imports `settings`; `NewLoaderWithError(settings.LoadGlobal)`, `Load(..., settings.LoadPage)`, emits `settings.HttpStatusError` |
| `internal/convert/prepare` | Imports `settings`; `Document(..., settings.LoadPage, ...)` |
| `css` (public) | Imports `settings`; media type, `DefaultPdfGlobal`, `LoadPage` for sheet collection |
| `internal/fonts` | Imports `settings`; `RegistryFromGlobal(settings.PdfGlobal)` |
| `bindings/wasm` | Imports `settings`; zero-value load policy for `data:` images |
| `internal/errs` | Shared nil-context sentinel used by load, prepare, render, and layout |

### 5.3 Import-direction rule

Settings types flow down into the engine and never back up. `internal/settings` imports no internal package, so it cannot depend on load, css, or layout. Callers use the typed structs.

## 6. Design decisions & trade-offs

### 6.1 wkhtmltopdf work-alike surface (compat first)

The defaults and HTTP exit-code mapping are copied from wkhtmltopdf (`pdfsettings.cc`, `imagesettings.cc`, `utilities.cc`, `loadsettings.cc`, cited in doc comments). The trade-off: more fields than the engine consumes, which Policy A contains.

### 6.2 Policy A: settings honesty

- Only options with an engine consumer get typed fields. Consumers today are load, prepare, css, and fonts.
- Inert wkhtml keys (`dpi`, `javascript`, `plugins`, `log-level`, `js-delay`, `user-style-sheet`, `produce-forms`, `default-encoding`, and more) have no field and no accepted-key table. They are never promoted to typed fields without a consumer.
- Dual storage is collapsed: `Grayscale` is the sole color bit; page geometry is the `PageSize` name plus the `Size` width/height pair; `Background` has one home on `PdfGlobal`.
- PDF-only knobs (`PdfVersion`, `PdfProfile`, `Copies`, `Outline`, `TOC`, header/footer) remain typed fields, but no engine in this tree reads them.

Trade-off: honesty (no field without a consumer) against script compatibility with wkhtmltopdf invocations.

### 6.3 Typed structs only

There is no dotted-key `Set`/`Get` surface. Callers write the typed struct fields directly, so adding a setting touches the struct and its default, with no key table to keep in step.

### 6.4 Millimetres as the internal length unit

Margins and custom page sizes are stored as float millimetres and parsed through `ParseUnitReal` + `UnitReal.Mm()`. Page names resolve to points through the static table. Conversion to points happens at the engine boundary.

### 6.5 Value semantics and cloning

Settings structs are plain value types except for map and slice fields. `clone.go` copies those fields (`ClonePdfGlobal` at `clone.go:11`, and siblings). `clone_test.go` pins the behavior.

### 6.6 Errors as a shared leaf vocabulary

`internal/errs` keeps `ErrNilContext` in one place so `errors.Is` works across packages (`internal/errs/errs.go:21-23`). `HttpStatusError` stays in `settings` because it carries data (status, URL) and keeps the exit-code mapping next to the vocabulary that defines it.

## 7. Notable patterns & invariants

- Header/footer inheritance: object overrides apply only when `HeaderSet`/`FooterSet` is true; otherwise `HeaderFor`/`FooterFor` fall back to global (`settings.go:445-461`).
- `Background` is the sole body-background bit on `PdfGlobal`; `Web` has no mirror.
- `MediaType` zero value is `MediaUnset`; callers fall back to their own default media (`settings.go:217`).
- Defaults mirror `pdfsettings.cc` for consumed fields.
- The HTTP exit-code mapping stays available on the error type even though no CLI consumes it: `(*HttpStatusError).HttpErrorCode` (`httperror.go:39`).

## 8. Security considerations

The settings layer is where security-relevant defaults and ACL policy originate:

- Local file access is denied by default. `DefaultLoadPage()` sets `BlockLocalFileAccess: true` (`settings.go:479-484`), and `DefaultPdfGlobal().Load.EnableLocalFileAccess` is false. The gate is `EnableLocalFileAccess && !BlockLocalFileAccess` in `load.fileAccessAllowed` (`load.go:1442-1449`), with `Allow` prefixes as the only widening path.
- `Allow` prefixes feed `AccessController` (`load.go:286`, `Allowed` at `load.go:297`).
- Scripts are never executed and forms are never created; there are no inert wkhtml key fields.
- Proxy config must be an absolute `http`/`https` URL (`parseProxy`, `load.go:959-975`).
- `HttpStatusError` carries only status and URL and is produced by `internal/load` (`load.go:1605-1627`).
- Timeouts and caps are typed in `LoadPage.Timeout` and enforced by `internal/load`, not by this package.

## 9. Testing & verification

The package is validated by table-driven unit tests:

| Test | What it verifies |
|------|------------------|
| `TestDefaultPdfGlobalSnapshot`, `TestDefaultPdfObjectSnapshot`, `TestDefaultLoadPageSnapshot`, `TestDefaultImageGlobalNoQuietLogLevel` | wkhtmltopdf-compatible defaults |
| `TestParseUnitReal`, `TestUnitRealPoints`, `TestParsePageSize` | unit and page-size parsers |
| `TestEnumStringReportsUnknownForInvalid`, `TestMediaTypeZeroIsUnset` | enum string edges |
| `TestHttpErrorCode` | 404 -> 2, 401 -> 3, 500 -> 1 |
| `TestHeaderForFooterForInherit` | object override wins, global fallback otherwise |
| `TestParsePDFVersion` | version parsing |
| `TestPdfGlobalFieldSnapshotViaClone` (`options_test.go`) | clone snapshot of typed fields |
| `clone_test.go`, `object_roles_test.go` | clone independence and role stamps |

## 10. Known limitations, deferred items & open questions

- Inert wkhtml keys have no accepted-and-ignored table any more: the dotted `Set`/`Get` surface was removed, so there is nothing to drift. Policy A still requires an engine consumer before any field is added.
- `size.pagesize` uses one canonical field: `PdfGlobal.PageSize` stores the name, `Size` stores width/height.
- `Quiet` lives on `PdfGlobal`, not `ImageGlobal`.
- PDF version, profile, copies, outline, header/footer, and TOC keys parse and round-trip but have no engine consumer in this tree.
- The HTTP exit-code mapping is status-only; wkhtmltopdf has finer codes that are not modeled.
- Deferred feature context: `documentation/deferred.md`; per-feature support: `documentation/compatibility-matrix.md`.

## 11. Related documents

- Siblings in this directory: [01-entrypoints-cli.md](01-entrypoints-cli.md), [02-library-api.md](02-library-api.md), [04-load.md](04-load.md), [05-html-parser.md](05-html-parser.md), [06-css.md](06-css.md), [07-layout.md](07-layout.md), [08-convert-pipeline.md](08-convert-pipeline.md), [10-imageout-svg.md](10-imageout-svg.md), [README.md](README.md).
- Top level: [../architecture.md](../architecture.md), [../library-api.md](../library-api.md), [../fidelity.md](../fidelity.md), [../THREAT-MODEL.md](../THREAT-MODEL.md), [../compatibility-matrix.md](../compatibility-matrix.md), [../deferred.md](../deferred.md).
