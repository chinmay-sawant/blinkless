# v0.0.1 - CSS masking first wave checklist

> **Parent:** [HTML, CSS, and JSON compatibility](../html-css-json-compatibility-checklist.md) - shared catalog and behavior-test rules.
> **Status:** proposal. The 18-property baseline is verified. Implementation has not started.
> **Estimated effort:** set after Phase 0 selects the mask representation and supported source types.

---

## Overview

Add static CSS masking for controlled HTML and CSS templates. A mask controls how much of an element shows through: alpha mode uses source transparency, while luminance mode uses source brightness. The first wave targets 18 names in the CSS property catalog, the repo's status table: 10 standard properties and 8 WebKit-prefixed properties.

Blinkless returns a drawing list, an ordered list of paint operations for a renderer to replay. A mask must apply to the element and its children as one painted group, including text, borders, backgrounds, and child paint. This matches the W3C mask rendering model ([CSS Masking, sections 1-2](https://www.w3.org/TR/css-masking-1/#module-interactions)). The `maskClipPathOps` helper changes image operations only (`internal/layout/clip_path.go:467-485`), so that helper alone is not enough for CSS masks.

The drawing list already has group boundary operations for blend and isolation (`layout/layout.go:364-376`, `internal/layout/blend_group.go:14-47`). Phase 0 must decide whether that representation can carry mask behavior or whether the display-list contract needs a new form. The decision must include the Go API (`layout/displaylist.go:99-106,134-139`) and the WebAssembly (WASM) JSON output (`bindings/wasm/contract.go:123-128,165-174`).

## Executive summary

The target names are:

| Standard property | Matching WebKit-prefixed property |
|---|---|
| `mask` | `-webkit-mask` |
| `mask-clip` | `-webkit-mask-clip` |
| `mask-composite` | `-webkit-mask-composite` |
| `mask-image` | `-webkit-mask-image` |
| `mask-mode` | No alias in this target |
| `mask-origin` | `-webkit-mask-origin` |
| `mask-position` | `-webkit-mask-position` |
| `mask-repeat` | `-webkit-mask-repeat` |
| `mask-size` | `-webkit-mask-size` |
| `mask-type` | No alias in this target |

All 18 names are currently `unsupported` with no handler or behavior test in the catalog (`testdata/css/catalog/properties.json:247-442,10186-10442`). This wave excludes the visual behavior of `mask-border*` and the `-webkit-mask-box-image*` family. The `mask` shorthand still has to reset the `mask-border` properties to their initial values as the spec requires ([CSS Masking section 7.9](https://www.w3.org/TR/css-masking-1/#propdef-mask)). Property names are the target count; each property's catalog status must still match the value and paint behavior that passes tests.

The current compatibility matrix still lists `clip-path` as unsupported in its old Phase 83 summary, although section 2 records it as implemented and names a behavior test (`documentation/compatibility-matrix.md:167,953-967`). Reconcile that stale family summary when updating compatibility docs for this work.

## Phase 0: lock the contract and baseline

- [x] Verify the 18 selected catalog rows are `unsupported` and have no handlers or tests. Proof on 2026-10-13: catalog row inspection found all 18 with empty handler and test lists; `make catalog-check` passed with 785 properties, 389 implemented, 1 partial, 387 unsupported, and 8 intentionally ignored; `make matrix-check` passed.
- [ ] Add test pages compared with a named browser for masks on elements with text, border, background, image, and child paint. Record browser version, viewport, device scale, and expected pixels or drawing-list result.
- [ ] Add fixtures for nested masks, multiple mask layers (comma-separated mask images), alpha (source transparency) and luminance (source brightness), invalid declarations, and prefixed plus unprefixed declarations in both orders.
- [ ] Choose the first supported `mask-image` source types from the current loader and image decoding paths. Record unsupported source types as explicit limitations.
- [ ] Trace a real `mask-type` use through an SVG mask source. If this engine path has no consumer, remove `mask-type` from this wave before implementation and keep its catalog row unsupported.
- [ ] Choose the element-group representation. Specify paint order, nested groups, clipping, blend/isolation interaction, and whether the Go or WASM drawing-list format changes.

**Gate:** the fixture set, supported source types, `mask-type` decision, and display-list representation are written down. Do not start property handlers before this gate passes.

## Phase 1: represent and apply an element mask

- [ ] Add the selected mask state to the style and paint model. Keep mask state separate from background clipping and `clip-path` (`internal/layout/style.go:317-335`).
- [ ] Carry a mask group around all paint emitted by the element. Cover text, inline decorations, backgrounds, borders, and images; do not limit the effect to `OpImage`.
- [ ] Preserve nested element groups and existing blend/isolation boundaries. Add focused tests for paint order and group ownership.
- [ ] If the public display-list format changes, update the Go reader and WASM schema/consumer together. Add a replay test for the new operation before changing the schema version.

**Gate:** fixtures prove the mask affects every paint type in scope, preserves unaffected siblings, and composes with nested groups. If the display-list contract changed, its Go and WASM consumers replay the same fixture.

## Phase 2: implement the standard properties

### 2.1 Layer source and mode

- [ ] Implement `mask-image` for the Phase 0 source types, including `none`, comma-separated layers, and normal CSS image sizing inputs. A layer is one image in that comma-separated list.
- [ ] Implement `mask-mode` for the selected alpha, luminance, and `match-source` cases. Test pixels with different alpha and color values.
- [ ] Implement `mask-type` only for the SVG `<mask>` source proven in Phase 0. Test the actual source and leave the property unsupported if no consumer exists ([CSS Masking section 9.2](https://www.w3.org/TR/css-masking-1/#mask-type)).

### 2.2 Layer geometry

- [ ] Implement `mask-position` and resolve percentages and lengths against the mask positioning area, the box used to position each mask image.
- [ ] Implement `mask-size`, including the selected `cover`, `contain`, and explicit-size cases.
- [ ] Implement `mask-repeat` for the selected repeat, repeat-x, repeat-y, and no-repeat cases.
- [ ] Implement `mask-origin` and `mask-clip` using the selected border, padding, and content boxes. Test that each box changes the visible mask boundary.

### 2.3 Layer composition and shorthand

- [ ] Implement `mask-composite` in layer order. Test each supported standard combination rule against pixel references.
- [ ] Implement the `mask` shorthand, one declaration that sets several mask properties, including its layer parsing, resets of omitted component properties, and reset of `mask-border` properties to initial values. Test shorthand and component-property ordering in both directions ([CSS Masking section 7.9](https://www.w3.org/TR/css-masking-1/#propdef-mask)).
- [ ] Validate declarations through the shared acceptance path. Make `@supports` (the CSS feature query) report true only for values accepted by the property code and consumed by paint (`internal/layout/style_cascade.go:1640-1695`).
- [ ] Add a behavior test for every standard property that asserts its effect on the returned drawing list or rendered test pixels. Do not promote a catalog row from parsing or stored-value tests alone.

**Gate:** all standard properties retained in scope have passing behavior tests for their recorded value subset. Invalid values preserve earlier valid declarations, and `@supports` agrees with declaration acceptance.

## Phase 3: add the WebKit-prefixed forms

- [ ] Map `-webkit-mask`, `-webkit-mask-clip`, `-webkit-mask-image`, `-webkit-mask-origin`, `-webkit-mask-position`, `-webkit-mask-repeat`, and `-webkit-mask-size` to the working standard behavior.
- [ ] Implement `-webkit-mask-composite` with its own tested legacy operator mapping. Do not pass its values through the standard operator parser unchanged.
- [ ] Test declaration order when prefixed and unprefixed forms appear together. Prefix mapping currently happens during property application (`internal/layout/style_cascade.go:1573-1637,1643-1659`), so the fixture must prove the winning declaration follows CSS source order.
- [ ] Test `@supports` for valid and invalid prefixed values through the same path as the cascade (`internal/layout/style_cascade.go:1662-1695`).
- [ ] Add one behavior test per prefixed catalog row. Leave any alias unsupported if its value mapping or paint behavior is not proven.

**Gate:** the eight prefixed rows have browser-reference behavior tests, including the legacy composite values and mixed-prefix cascade cases.

## Phase 4: reconcile catalog, docs, and final evidence

- [ ] Add the matching handler and behavior-test name to each completed property row in `testdata/css/catalog/properties.json:247-442,10186-10442`. Use `partial` when only a tested subset works; use `implemented` only when the advertised values have behavior coverage.
- [ ] Regenerate the compatibility matrix from the catalog. Update its Phase 83 mask-family record and vendor-alias table to distinguish this wave from the still-deferred `mask-border` family.
- [ ] Update `documentation/deferred.md` and the compatibility matrix's stale `clip-path` Phase 83 note. Keep remaining unsupported effects listed accurately.
- [ ] Update the CSS masking knowledge-base page with final source paths, limitations, and evidence.
- [ ] Run `make catalog-check`, `make matrix-check`, `make golden`, `make test`, `make lint`, and `make claim-scan` on the final code tree. If the public display-list or WASM contract changes, also run `make wasm-test` and record its result.

**Gate:** catalog and generated matrix agree; all required final commands pass; documentation states only behavior proven by the fixtures. Record commands and results here before closing this phase.

## Dependencies

- The CSS declaration and `@supports` paths share vendor normalization and style-group dispatch (`internal/layout/style_cascade.go:1573-1695`).
- The current `clip-path` raster helper changes image payloads only (`internal/layout/clip_path.go:467-510`). Element masking needs a group-level paint path.
- Blend and isolation already delimit grouped operations (`layout/layout.go:364-376`, `internal/layout/blend_group.go:14-47`). Their current contract does not itself prove mask support.
- CSS Masking section 7.9 says the `mask` shorthand resets the `mask-border` shorthand and its component properties. This reset behavior stays in scope even though the first wave excludes their visual effect ([W3C CSS Masking](https://www.w3.org/TR/css-masking-1/#propdef-mask)).
- The catalog is authoritative for per-property status. Its current selected rows are `unsupported` (`testdata/css/catalog/properties.json:247-442,10186-10442`); the old Phase 83 prose is historical and needs reconciliation at closeout (`documentation/compatibility-matrix.md:953-967,1053-1072`).
