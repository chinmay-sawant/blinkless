# Settings system and errors

`internal/settings` is the wkhtmltopdf-compatible settings model and the dotted-name Set/Get surface. The layout pipeline reads only part of it: load policy, font paths, and CSS media selection. PDF version, PDF profile, copies, outlines, headers, footers, and TOC keys still parse and round-trip through `Set`/`Get`, but nothing in this tree consumes them. There is no PDF writer.

## 1. Responsibility & position in the pipeline

The package is the single source of truth for user-facing knobs: page geometry, margins, headers/footers, TOC behaviour, load policy (proxy, ACL), web behaviour, image settings, and the accepted-but-inert wkhtmltopdf keys. It owns the wkhtmltopdf dotted-name vocabulary (`margin.top`, `load.jsdelay`, `web.background`, `toc.captiontext`).

Its consumers today:

```text
settings.PdfGlobal            -> internal/fonts          (font path scan, registry)
settings.LoadGlobal/LoadPage  -> internal/load           (loader policy)
settings.LoadPage             -> internal/convert/prepare (document load)
settings.MediaType / DefaultPdfGlobal / LoadPage -> css   (sheet collection)
settings.LoadGlobal/LoadPage  -> bindings/wasm           (data: image loader)
```

Nothing in the engine reads argv. The dotted `Set`/`Get` surface has no production caller in this tree; the package tests exercise it. The structs are the typed payload the consumers above read.

A second, smaller package is in scope: `internal/errs` owns the shared nil-context sentinel `ErrNilContext` and a parked, deprecated `ErrNilCommand` (`internal/errs/errs.go:21-26`). `ErrNilLoader` moved to `internal/load` (`load.go:60`). `errs` imports only `errors`.

## 2. Package / file map

| File | Responsibility | Approx. lines |
|------|----------------|---------------|
| `internal/settings/settings.go` | Typed model and parsers: `PdfGlobal`, `PdfObject`, `ImageGlobal`, sub-structs (`Web`, `LoadGlobal`, `LoadPage`, `HeaderFooter`, `TableOfContent`, `Margin`, `Size`, `CropSettings`, `PostItem`), enums (`Orientation`, `LoadErrorHandling`, `MediaType`), `ParsePDFVersion` / `ParsePDFProfile`, `ResolveMedia` / `ResolvePDFMedia` / `ResolveImageMedia` / `ResolveImages`, wkhtml-compatible defaults | 695 |
| `internal/settings/reflect.go` | Descriptor engine: `field`/`keyTable`, dotted-key `Set`, type-coercing setters, ignored-key tables, `ApplyImageKey` | 1046 |
| `internal/settings/getters.go` | `Get` methods and canonical string formatting | 37 |
| `internal/settings/clone.go` | `ClonePdfGlobal`, `ClonePdfObject`, `CloneImageGlobal`, `CloneHeaderFooter`; copies slices and maps | 74 |
| `internal/settings/object_roles.go` | `StampCover`, `StampTOC` object flags | 28 |
| `internal/settings/pagesize.go` | Static ISO/ANSI page-size table; `ParsePageSize` | 64 |
| `internal/settings/unitreal.go` | `UnitReal` scalar with unit suffix; `Points`/`Mm` conversion; `ErrInvalidUnitReal` | 106 |
| `internal/settings/httperror.go` | `HttpStatusError` (HTTP status + URL) and `HttpErrorCode` (404 -> 2, 401 -> 3, else 1) | 41 |
| `internal/settings/doc.go` | Package doc | 22 |
| tests | `settings_test.go` (1067), `clone_test.go` (97), `object_roles_test.go` (81), `options_test.go` (79), `reflect_parity_test.go` (27) | 1351 |
| `internal/errs/errs.go` | Shared sentinels | 27 |

## 3. Key types, functions & entry points

### 3.1 The three settings structs

| Type | Purpose | File:line |
|------|---------|-----------|
| `PdfGlobal` | Global settings: page geometry, orientation, `PdfVersion` / `PdfProfile`, grayscale, copies/collate, outline, title, margins, header/footer, TOC, background, load policy, font paths, `Ignored` sink | `settings.go:503` |
| `PdfObject` | One page/cover/toc object: `Page`, link flags, outline inclusion, object header/footer overrides (`HeaderSet`/`FooterSet`), `LoadPage`, `Web`, `UseOutline`, `Ignored` | `settings.go:575` |
| `ImageGlobal` | Image-mode settings: width/height/quality, smart width, crop, format, transparency, `Web`, `LoadGlobal`, `Ignored` | `settings.go:658` |

Supporting sub-structs (all in `settings.go`):

| Type | Purpose | File:line |
|------|---------|-----------|
| `Margin` | Four page margins in millimetres | `settings.go:362` |
| `Size` | Custom page dimensions in mm; 0 = unset | `settings.go:388` |
| `Web` | Web behaviour: `Images`, `PrintMediaType`/`MediaType`, `SimplifyDOM(+Profile)`, `PrintLinkUnderline` | `settings.go:397` |
| `LoadGlobal` | Shared load policy: `Proxy`, `Allow` prefixes, `EnableLocalFileAccess`, network-policy fields | `settings.go:416` |
| `LoadPage` | Per-page load policy: zoom, block-local-access, load-error handling, auth, headers, cookies, POST, media, timeout, `InlineHTML`/`InlineBase` | `settings.go:431` |
| `HeaderFooter` | Text/HTML header and footer: font, left/right/center, line, spacing, `HTMLURL`, `Replace` | `settings.go:457` |
| `TableOfContent` | TOC settings: font scale, indentation, dotted lines, caption, links, XSL | `settings.go:479` |
| `PostItem` | One urlencoded form field for POST loads | `settings.go:451` |
| `CropSettings` | Image crop box (Left/Top/Width/Height, -1 = unset) | `settings.go:674` |

### 3.2 Enums and parsers

| Symbol | Purpose | File:line |
|--------|---------|-----------|
| `Orientation` + `ParseOrientation` | portrait/landscape, case-insensitive | `settings.go:166` / `settings.go:185` |
| `LoadErrorHandling` + `ParseLoadErrorHandling` | abort/skip/ignore | `settings.go:197` / `settings.go:219` |
| `MediaType` + `ResolveMedia(base, global Web, obj *Web)` | screen/print/ignore resolution; print-media-type override wins, then object, then global, then base | `settings.go:234` / `settings.go:268` |
| `ResolvePDFMedia` / `ResolveImageMedia` / `ResolveImages` | Mode-specific media and image-enable folding; no production caller today | `settings.go:292` / `settings.go:312` / `settings.go:347` |
| `ParsePDFVersion` | `""`/`1.4`/`1.7`/`2.0` | `settings.go:142` |
| `ParsePDFProfile` | Aliases (`a3a-ua1`, `a4-ua2`, ...) to canonical tokens (`PDF/A-3a+PDF/UA-1`, ...) | `settings.go:64` |
| `IsPDFA3` / `IsPDFA4` / `IsPDFUA1` / `IsPDFUA2` / `IsPDFUA` | Profile class checks | `settings.go:117-140` |

### 3.3 Defaults (wkhtmltopdf `pdfsettings.cc` / `imagesettings.cc` compatible)

| Function | Notable defaults | File:line |
|----------|------------------|-----------|
| `DefaultMargins()` | 10 mm all sides | `settings.go:383` |
| `DefaultHeaderFooter()` | Arial, font size 12, spacing 0 | `settings.go:470` |
| `DefaultTableOfContent()` | font scale 0.8, indentation `1em`, dotted lines true, caption "Table of Contents" | `settings.go:490` |
| `DefaultPdfGlobal()` | A4 portrait, 1 copy collated, outline depth 4, compression on, smart shrinking on, background on, images on, margins 10 mm, resolve-relative-links on | `settings.go:550` |
| `DefaultPdfObject()` | external/local links on, outline flags on, block-local-file-access true, load-error abort | `settings.go:633` |
| `DefaultLoadPage()` | `BlockLocalFileAccess: true`, `LoadErrorHandling: LoadErrorAbort` | `settings.go:649` |
| `DefaultImageGlobal()` | width 1024, quality 94, smart width on, crop = (-1,-1,-1,-1) | `settings.go:682` |

### 3.4 The descriptor engine (`reflect.go`)

| Symbol | Purpose | File:line |
|--------|---------|-----------|
| `field[T]` | One dotted key's apply/get pair | `reflect.go:108` |
| `keyTable[T]` | One descriptor table per settings type (`globalKeys`, `objectKeys`, `imageKeys`) | `reflect.go:114`, built at `reflect.go:987` |
| `sub[T, S]` / `subTable` | Adapt a sub-struct descriptor to its container | `reflect.go:118` / `reflect.go:509` |
| `setForKey` / `getForKey` | Lookup: descriptor table -> known-ignored set -> error / not found | `reflect.go:127` / `reflect.go:146` |
| `ignoredGlobalKeySet` / `ignoredObjectKeySet` | Inert wkhtml keys (dpi, javascript, js-delay, log-level, ...) | `reflect.go:166` / `reflect.go:200` |
| `normalizeDots` | Lowercase and trim the key | `reflect.go:235` |
| `setBool` / `setFloat` / `setInt` / `setIntRange` / `setFloatMin` / `setString(Default)` | Type-coercing setters | `reflect.go:239`, `:256`, `:269`, `:294`, `:314`, `:331`, `:339` |
| `setGrayscaleFromColorMode` | Maps `colormode` strings onto the shared `Grayscale` bool | `reflect.go:395` |
| `setUnitMm` | Parse a unit real, convert to mm, store | `reflect.go:431` |
| `buildKeyTables` | Builds all three tables once at package init | `reflect.go:987` |
| `(g *PdfGlobal) Set` / `(o *PdfObject) Set` / `(g *ImageGlobal) Set` | Dotted-key entry points | `reflect.go:1014` / `:1020` / `:1025` |
| `ApplyImageKey` | Routes image keys; `background`/`web.background` alias to `PdfGlobal.Background` | `reflect.go:1033` |
| `register*` functions | Table wiring: global, document, version/profile, load, geometry, header/footer, TOC, web, load-page, object, image keys | `reflect.go:519-930` |

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

`HeaderFor` / `FooterFor` (`settings.go:595` / `settings.go:604`) return the object override or the global header/footer. `ValidateRenderableObjects` (`settings.go:618`) rejects an object list with no page source.

### 3.7 Sentinel errors

`internal/errs` now holds two sentinels (`internal/errs/errs.go`):

| Symbol | Purpose | File:line |
|--------|---------|-----------|
| `ErrNilContext` | Shared nil-context guard; consumers include load, prepare, render, and layout | `errs.go:23` |
| `ErrNilCommand` | Parked and deprecated; no consumer in this tree | `errs.go:26` |

`ErrNilLoader` lives in `internal/load` (`load.go:60`).

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

### 4.4 The dotted Set/Get surface

`Set` normalizes the key, consults the descriptor table, then the known-ignored set, and errors on truly unknown keys (`reflect.go:127-160`). `Get` returns the canonical string form; accepted ignored keys return the last `Set` value (`getters.go:13-25`). No production code calls either today; `settings_test.go` and `reflect_parity_test.go` are the consumers.

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

Settings types flow down into the engine and never back up. `internal/settings` imports no internal package, so it cannot depend on load, css, or layout. The dotted vocabulary lives only here; callers use the typed structs.

## 6. Design decisions & trade-offs

### 6.1 wkhtmltopdf work-alike surface (compat first)

The dotted keys, defaults, and HTTP exit-code mapping are copied from wkhtmltopdf (`pdfsettings.cc`, `imagesettings.cc`, `utilities.cc`, `loadsettings.cc`, cited in doc comments). The trade-off: a larger key vocabulary than the engine consumes, which Policy A contains.

### 6.2 Policy A: settings honesty

- Only options with an engine consumer get typed fields. Consumers today are load, prepare, css, and fonts.
- Inert wkhtml keys (`dpi`, `javascript`, `plugins`, `log-level`, `js-delay`, `user-style-sheet`, `produce-forms`, `default-encoding`, and more) are accepted into `Ignored` so existing invocations do not hard-fail. They are never promoted to typed fields without a consumer.
- Dual storage is collapsed: `Grayscale` is the sole color bit (`colormode` and `grayscale` both write it); page geometry is the `PageSize` name plus the `Size` width/height pair; `background` and `web.background` share one field.
- PDF-only knobs (`PdfVersion`, `PdfProfile`, `Copies`, `Outline`, `TOC`, header/footer) remain parseable and round-trippable, but no engine in this tree reads them.

Trade-off: script compatibility (accept and ignore) against honesty (reject what is not honored). Typos still fail loudly; known inert keys pass quietly.

### 6.3 Descriptor tables over reflection

The dotted surface is driven by explicit typed descriptor tables built once by `buildKeyTables()` (`reflect.go:987`), not by `reflect` traversal. Setters are closures over typed fields, and `TestGlobalKeyDescriptorsHaveSetAndGetSides` (`reflect_parity_test.go:5`) checks that every key has both halves.

Trade-off: adding a setting touches the struct, a setter, and a register function.

### 6.4 Millimetres as the internal length unit

Margins and custom page sizes are stored as float millimetres and parsed through `ParseUnitReal` + `UnitReal.Mm()`. Page names resolve to points through the static table. Conversion to points happens at the engine boundary.

### 6.5 Value semantics and cloning

Settings structs are plain value types except for map and slice fields. `clone.go` copies those fields (`ClonePdfGlobal` at `clone.go:11`, and siblings). `clone_test.go` pins the behavior.

### 6.6 Errors as a shared leaf vocabulary

`internal/errs` keeps `ErrNilContext` in one place so `errors.Is` works across packages (`internal/errs/errs.go:21-23`). `HttpStatusError` stays in `settings` because it carries data (status, URL) and keeps the exit-code mapping next to the vocabulary that defines it.

## 7. Notable patterns & invariants

- Dotted-key normalization: `normalizeDots` lowercases and trims (`reflect.go:235`).
- Boolean coercion: `setBool` accepts `""/true/1/yes/on` and `false/0/no/off` (`reflect.go:239`).
- `Ignored` round-trip: an accepted inert key is stored verbatim and returned by `Get` with `ok=true` (`getForKey`, `reflect.go:146`).
- Unknown keys fail loudly: `setForKey` errors instead of ignoring (`reflect.go:127`).
- Header/footer inheritance: object overrides apply only when `HeaderSet`/`FooterSet` is true; otherwise `HeaderFor`/`FooterFor` fall back to global (`settings.go:595-610`).
- `web.background` alias: both `background` and `web.background` write `PdfGlobal.Background` (`reflect.go:593`); on objects the key is inert (`ignoredObjectKeySet`, `reflect.go:217`).
- Media resolution precedence: print-media-type override, then object media type, then global, then the mode base (`ResolveMedia`, `settings.go:268`).
- Defaults mirror `pdfsettings.cc` for consumed fields; `DefaultPdfGlobal` leaves `Ignored` nil.
- Immutable-by-convention tables: `ignoredGlobalKeySet` (`reflect.go:166`) and `ignoredObjectKeySet` (`reflect.go:200`).
- The HTTP exit-code mapping stays available on the error type even though no CLI consumes it: `(*HttpStatusError).HttpErrorCode` (`httperror.go:39`).

## 8. Security considerations

The settings layer is where security-relevant defaults and ACL policy originate:

- Local file access is denied by default. `DefaultLoadPage()` sets `BlockLocalFileAccess: true` (`settings.go:649-654`), and `DefaultPdfGlobal().Load.EnableLocalFileAccess` is false. The gate is `EnableLocalFileAccess && !BlockLocalFileAccess` in `load.fileAccessAllowed` (`load.go:1287-1294`), with `Allow` prefixes as the only widening path.
- `Allow` prefixes feed `AccessController` (`load.go:252`, `Allowed` at `load.go:263`).
- Inert-key acceptance is a compat surface, not an attack surface: scripts are never executed and forms are never created. `Set` validates the key vocabulary, so a misspelled key errors.
- Proxy config must be an absolute `http`/`https` URL (`parseProxy`, `load.go:843-860`).
- `HttpStatusError` carries only status and URL and is produced by `internal/load` (`load.go:1439-1461`).
- Timeouts and caps are typed in `LoadPage.Timeout` and enforced by `internal/load`, not by this package.

## 9. Testing & verification

The package is validated by table-driven unit tests:

| Test | What it verifies |
|------|------------------|
| `TestDefaultPdfGlobalSnapshot`, `TestDefaultPdfObjectSnapshot`, `TestDefaultLoadPageSnapshot`, `TestDefaultImageGlobalNoQuietLogLevel` | wkhtmltopdf-compatible defaults |
| `TestGlobalSetDottedKeys`, `TestObjectSetDottedKeys`, `TestImageSet` | dotted key application |
| `TestGlobalSetIgnoredKeys`, `TestObjectSetIgnoredKeys` | Policy A acceptance and round-trip |
| `TestGlobalSetUnknownKey` | unknown keys error |
| `TestCopiesSetterRange`, `TestLoadTimeoutZoomSetterRange`, `TestFiniteSetters`, `TestMarginSetterAutoAndSides`, `TestImageQualitySetterRange` | setter range and coercion rules |
| `TestParseUnitReal`, `TestUnitRealPoints`, `TestParsePageSize`, `TestParseEnums` | parsers |
| `TestColorModeSetGrayscale`, `TestMediaTypeZeroIsUnset`, `TestMarginEdgeUnknown` | collapsed storage and enum edges |
| `TestHttpErrorCode` | 404 -> 2, 401 -> 3, 500 -> 1 |
| `TestHeaderForFooterForInherit` | object override wins, global fallback otherwise |
| `TestGlobalGetSetRoundTripAndIgnored`, `TestKeyTableSetGetParity`, `TestGlobalKeyDescriptorsHaveSetAndGetSides` | Get round-trip and table parity |
| `TestBackgroundSingleFieldNoWebMirror`, `TestApplyImageKeyBackgroundAlias`, `TestResolveMedia`, `TestResolveImages` | aliases and media/image resolution |
| `TestParsePDFVersion`, `TestGlobalPdfVersionSetting`, `TestGlobalPdfProfileSetting` | version/profile parsing and round-trip |
| `TestPdfGlobalFieldSnapshotViaClone`, `TestPdfGlobalSetCoversCompatibilityKeys` (`options_test.go`) | clone snapshot and compatibility keys |
| `clone_test.go`, `object_roles_test.go` | clone independence and role stamps |

## 10. Known limitations, deferred items & open questions

- Ignored keys are a growing compat debt surface. Every accepted-but-inert key (`ignoredGlobalKeySet`, `reflect.go:166`; `ignoredObjectKeySet`, `reflect.go:200`) behaves differently from wkhtmltopdf without saying so. Policy A requires an engine consumer before any promotion.
- `default-encoding` is accepted then ignored (`reflect.go:177`). The engine is UTF-8/ASCII only at the load charset gate.
- `web.background` on objects is inert (`ignoredObjectKeySet`, `reflect.go:217`); paint background is global-only.
- `load.proxy` is object-ignored (`reflect.go:211`); only `LoadGlobal.Proxy` is wired.
- `size.pagesize` uses one canonical field: `PdfGlobal.PageSize` stores the name, `Size` stores width/height.
- `fontpath` and `allow` are append-only setters; there is no way to clear them through the dotted surface.
- `Quiet` lives on `PdfGlobal`, not `ImageGlobal`.
- PDF version, profile, copies, outline, header/footer, and TOC keys parse and round-trip but have no engine consumer in this tree.
- The HTTP exit-code mapping is status-only; wkhtmltopdf has finer codes that are not modeled.
- Deferred feature context: `documentation/deferred.md`; per-feature support: `documentation/compatibility-matrix.md`.

## 11. Related documents

- Siblings in this directory: [01-entrypoints-cli.md](01-entrypoints-cli.md), [02-library-api.md](02-library-api.md), [04-load.md](04-load.md), [05-html-parser.md](05-html-parser.md), [06-css.md](06-css.md), [07-layout.md](07-layout.md), [08-convert-pipeline.md](08-convert-pipeline.md), [10-imageout-svg.md](10-imageout-svg.md), [README.md](README.md).
- Top level: [../architecture.md](../architecture.md), [../library-api.md](../library-api.md), [../fidelity.md](../fidelity.md), [../THREAT-MODEL.md](../THREAT-MODEL.md), [../compatibility-matrix.md](../compatibility-matrix.md), [../deferred.md](../deferred.md).
