# CSS subsystem: parse, selectors, cascade

## 1. Responsibility & position in the pipeline

`internal/css` implements the CSS subset blinkless accepts. Its package doc (`internal/css/css.go`) states the contract
precisely:

> Scope: `*`, type, `.class`, `#id`, attribute selectors (`[attr]`, `=`, `~=`,
> `*=`, `^=`, `$=`, `|=`, ASCII `i` flag),
> `:first-child`/`:last-child`/`:nth-child`/`:first-of-type`/`:last-of-type`/
> `:nth-of-type`/`:nth-last-of-type`/`:has()`/`:not()`/`:is()`/`:where()`,
> descendant/child/sibling combinators, `@media` type + size-feature matching,
> `@container` size queries, `!important`, inline style attributes. Unsupported
> constructs degrade without panicking.

The package sits **between the HTML tree and the layout engine** in the pipeline:

```text
internal/load    → internal/html   → internal/css   → internal/layout   → internal/pdf
   (fetch)           (tree)            (this pkg)       (style+cascade     (paint)
                                          │              +box layout)
                                          ├──> internal/convert/prepare   (stylesheet collection, @font-face)
                                          ├──> internal/outline           (exclude-selector matching)
                                          └──> internal/imageout          (image-mode style)
```

It is a **pure parsing/matching/value library**: it never fetches resources,
never resolves page size, and never decides layout. It produces three kinds of
artifacts:

1. **Parsed sheets** (`*css.Stylesheet`) containing ordered rules, `@font-face`
   metadata and `@page` margin/size declarations.
2. **Boolean matching results** (`css.Match`, `css.MatchPseudo`,
   `css.MediaMatches`, `css.ContainerCond.Matches`) evaluated against the
   `internal/html` tree and viewport numbers supplied by the caller.
3. **Value helpers** (`css.ParseLength`, `css.ParseColor`, `css.ParseFontFamily`,
   `css.LengthToPt`, `css.ResolveCustomProps`, ...) that downstream style
   resolution uses to turn raw declaration strings into typed geometry.

Layout owns everything after this: computed styles, inheritance, cascade
*merging*, unit-to-point interpretation inside box contexts, and painting. The
boundary rule is that **css never imports layout** (see §5), so the package can
be reasoned about and fuzzed in isolation.

## 2. Package / file map

All files live under `internal/css/` and belong to `package css`. Approximate
line counts from `wc -l` (2026-09):

| File | Lines | Responsibility |
|------|------:|----------------|
| `css.go` | 971 | Stylesheet/rule/selector/declaration model; top-level parser; at-rule dispatch (`@media`, `@container`, `@page`, `@font-face`, `@import`, skip-others); block/paren scanning; `ParseSelectors`; specificity |
| `selector_parser.go` | 637 | Selector-chain and compound parsing; attribute selectors incl. the ASCII `i` flag; pseudo classification; recursive `:has()`/`:not()`/`:is()`/`:where()` argument parsers (`appendIsWherePseudo`, `isWherePseudo`) |
| `match.go` | 729 | Right-to-left matching (`Match`, `MatchPseudo`, `matchPart`, `matchPseudos`, `matchPseudo` incl. `:is()`/`:where()` via `matchAnySelector`); `:nth-child`/of-type arithmetic; direct sibling scans in `getSiblingInfo` |
| `import.go` | 107 | `@import` prelude parsing into `Stylesheet.Imports` (`parseImportRule`); never fetches |
| `page_margin.go` | 223 | Lite unnamed `@page` margin-box parsing (`PageMarginBoxes`); quoted content strings only |
| `values.go` | 854 | Declaration-block splitting (`ParseInline`); `!important`; length/number/color parsing; `var()` fallback + custom-property resolution; font-family splitting; named-color table |
| `container.go` | 744 | `@container` prelude parsing (`ContainerQuery`), boolean condition tree (`ContainerCond`), size-feature evaluation, length→pt conversion (`LengthToPt`), container-name/shorthand sidecars, `HasContainerRules` |
| `has.go` | 365 | Paren/quote scanning shared by `:has()`, `:not()`, media & container features; strict selector-list parsing; relative selector matching; leftmost-match combinator walk; specificity max-of-arguments helpers |
| `media.go` | 207 | `MediaMatches` evaluation of raw `@media` preludes: types (`print`/`screen`), size features, `orientation`, `not`/`only`, comma OR-lists |
| `css_test.go` | 1198 | Parse/matching/specificity/value/custom-prop tests |
| `has_test.go` | 215 | `:has()`/`:not()` parse+match+specificity tests |
| `is_test.go` | 282 | `:is()`/`:where()` parse/match tests |
| `container_test.go` | 189 | Container query parse/eval tests |
| `nth_type_test.go` | 175 | `:first-of-type`/`:nth-of-type`/`:nth-last-of-type` tests |
| `phase4_bench_test.go` | 153 | Selector/parse benchmarks |
| `attr_iflag_test.go` | 124 | `[attr operator value i]` case-insensitivity flag tests |
| `has_bench_test.go` | 89 | `:has()` benchmarks |
| `media_test.go` | 86 | Media-query evaluation tests |
| `parse_depth_test.go` | 85 | Parse-depth guard regression (`TestParseNestedFunctionalPseudoDepth`) |
| `pseudo_element_drop_test.go` | 74 | Regression: `::before/::after` must not apply to the host |
| `page_margin_test.go` | 70 | `@page` margin-box parsing tests |
| `target_pseudo_test.go` | 55 | Regression: `:target` must not match the bare host |
| `wiki_print_hide_test.go` | 52 | Real-world parser smoke test (Wikipedia print-hide sheets) |
| `sibling_cache_test.go` | 51 | Sibling-cache bound/eviction regression |
| `fuzz_test.go` | 31 | Fuzz guard for `Parse` (skips inputs over 64 KiB) |

The two complementary roles are worth distinguishing early:

- **Parse-time**: `css.go` + `container.go` + `has.go` + `values.go` build
  `Stylesheet`/`Selector`/`ContainerQuery`/`Declaration` values.
- **Match-time**: `css.go` (`Match`, `MatchPseudo`, `matchPart`),
  `has.go` (combinator walk), `media.go` (`MediaMatches`), `container.go`
  (`ContainerCond.Matches`).

## 3. Key types, functions & entry points

### 3.1 Type model (all in internal/css/css.go)

| Type | Location | Purpose |
|------|----------|---------|
| `Stylesheet` | css.go:66 | Parsed sheet: `Rules []Rule`, `FontFaces []FontFace`, `Page *PageStyle`, `Properties []PropertyRule`, `Layers []string`. Rules keep source order |
| `Rule` | css.go:121 | Selector list + declaration block + raw `Media` prelude + `Order` (source-order tiebreak, rebased by callers across sheets) + optional `Container *ContainerQuery`, `Supports *SupportsCondition`, and `Layer` cascade rank |
| `Selector` | css.go:83 | Chain of `SelectorPart` linked by combinators; carries a cached specificity triple (`spec`/`specValid`, computed at parse time) |
| `SelectorPart` | css.go:95 | One compound: `Tag`, `Classes`, `ID`, `Attrs []AttrSelector`, `Pseudos []PseudoClass`, `PseudoElement` ("before"/"after"/""), `Combinator` ("" first part, ">" "+" "~" " ") |
| `AttrSelector` | css.go:108 | `[name]`, `[name=value]`, `~=`, `*=`, `^=`, `$=`, `|=` forms |
| `RelativeSelector` | css.go:116 | Selectors-4 relative selector (`>`, `+`, `~`, descendant) used inside `:has()` |
| `PseudoClass` | css.go:123 | Named pseudo with optional `Arg` (`:nth-child`), `Has []RelativeSelector`, `Not []Selector`, and a pre-parsed integer `nth nthForm` |
| `Declaration` | css.go:134 | `Prop`, `Value`, `Important`: the raw wire form of a `prop: value[!important]` pair |
| `PageStyle` | css.go:59 | `@page` margin/size declarations kept as raw strings, resolved at the PDF boundary |
| `FontFace` | css.go:66 | `@font-face` local subset: `Family` + raw `Src` (consumed by `prepare.ResourceContext.MergeFontFaces`) |

### 3.2 Public entry points (exported functions)

| Function | Location | Purpose |
|----------|----------|---------|
| `Parse(src string) (*Stylesheet, error)` | css.go:143 | Parse a whole stylesheet. Only unbalanced-brace input errors; garbage degrades silently |
| `ParseSelectors(s string) ([]Selector, bool)` | css.go:644 | Strict parse of a comma-separated selector list (used by `--exclude-from-outline`, internal/convert/outline.go:123) |
| `ParseInline(style string) []Declaration` | values.go:9 | Parse a `style=""` attribute value |
| `Match(s Selector, n *html.Node) bool` | css.go:1216 | Does the selector match the element? (right-to-left) |
| `MatchPseudo(sel Selector, n *html.Node, pseudo string) bool` | css.go:1222 | `::before`/`::after` shape matching (host must match the pseudo's compound) |
| `Specificity(s Selector) (a, b, c int)` | css.go:1738 | ID / class·attr·pseudo / type counts; `:has()/:not()` contribute max-of-arguments |
| `MediaMatches(query, mediaType string, widthPt, heightPt float64) bool` | media.go:24 | Evaluate a raw `@media` prelude against conversion media + viewport |
| `ParseLength(val string) (float64, string, bool)` | values.go:108 | Number+unit split; bare numbers default to px; accepts px/pt/pc/in/cm/mm/em/rem/ex/ch/%/vw/vh |
| `LengthToPt(val float64, unit string, basePt float64) (float64, bool)` | container.go:113 | Unit→pt conversion incl. em/rem/ex/ch; `%` and viewport units unsupported (false) |
| `ParseNumber(val string) (float64, bool)` | values.go:163 | Bare number (line-height, font-weight) |
| `ParseColor(val string) (r,g,b int, alpha float64, ok bool)` | values.go:182 | `#rgb`, `#rrggbb`, `#rrggbbaa`, `rgb()/rgba()` int/float/percent, `transparent`, named-color subset, `var()` fallback |
| `ParseFontFamily(value string) []string` | values.go:562 | Comma split + quote trim |
| `ResolveVar(val string, lookup func(string)(string,bool)) string` | values.go:412 | Expand one `var(--name, fallback)` chain (bounded depth) |
| `ResolveCustomProps(declared, inherited map[string]string) map[string]string` | values.go:448 | The single place custom-property policy lives: overlay + memoized var expansion + cycle guard |
| `ParseContainerNameValue(value string) string` | container.go:165 | `container-name: none | <custom-ident>+` |
| `ParseContainerShorthand(value string) (name, ctype string)` | container.go:189 | `container: <name> [ / <type> ]?` |
| `HasContainerRules(sheets []*Stylesheet) bool` | container.go:709 | Fast gate: does any sheet contain `@container` rules? (layout skips its second style pass when false, internal/layout/layout.go:831) |
| `FontFaceURLs(src string) []string` | css.go:392 | Extract `url(...)` from an `@font-face src` (case-preserving) |
| `SupportsMatches(cond *SupportsCondition, supported func(prop, value string) bool) bool` | at_supports.go:238 | Evaluate one parsed `@supports` condition; layout passes `engineSupportsProperty` as the predicate |

### 3.3 Interesting non-exported machinery

| Symbol | Location | Why it exists |
|--------|----------|---------------|
| `parseAtRule` | css.go:262 | Dispatch: `@media`/`@container`/`@supports`/`@layer` parsed into rules; `@property`/`@page`/`@font-face`/`@import` side channels; `@keyframes` and anything unknown **parse-ignored** (static cascade only; there is no animation) |
| `stripComments` | css.go:537 | Removes `/* */`, preserving `\n` so line numbers stay stable |
| `findBlock`/`takeBlock` | css.go:572/614 | Brace finding with quote/paren tracking; the only source of real parse errors (`errUnbalanced`, `errNoBlock` at css.go:526) |
| `splitTopLevel` | css.go:668 | Splits selector lists and declaration blocks on top-level `,`/`;` outside parens/brackets/quotes |
| `splitSelectorChain` | css.go:712 | Breaks a selector into compounds + separators, including the `addDescendantCombinator` whitespace rule |
| `writePseudoLiteral` | css.go:826 | Keeps `:pseudo(...)` and `::before/::after` inside the compound so unsupported pseudos reject the selector instead of degrading to the host |
| `parseCompoundCtx` | css.go:862 | One compound → `SelectorPart`; `insideHas` rejects nested `:has`/pseudo-elements |
| `appendSimplePseudo` | css.go:1086 | Classifies pseudos: matchable (`first-child`, `nth-child`, `link/visited`…), never-match-but-keep (`hover/active/focus/target`, unknown), rejected (`first-line/first-letter`) |
| `parseNthArg`/`matchNth` | css.go:1576/1608 | `:nth-child()` pre-parsed to integer form at parse time; matching is pure arithmetic |
| `leftmostMatch`/`leftmostStep` | has.go:256/284 | Shared right-to-left combinator walk; `Match`, `MatchPseudo` and `:has` relative matching all ride it |
| `computeSpecificity` | css.go:1747 | Selectors-4 specificity incl. `:has()/:not()` max-of-argument contribution |
| `parseDeclarations` | values.go:15 | Block → `[]Declaration`; `validPropName` allowlist (lowercase `-` alnum only); `isImportant` accepts `! important` spacing |
| `namedColorTable` | values.go:594 | Curated CSS2 + common web colors (read-only cache; not full CSS Color 4) |

## 4. Data & control flow

### 4.1 Stylesheet collection (the entry into css for a document)

1. `internal/convert/prepare` `ResourceContext.CollectSheets` walks the HTML tree with
   `root.Walk`, visiting every `style` and `link` element (styles.go:35–48).
2. Inline `<style>` text → `css.Parse` (styles.go:90); `<link rel=stylesheet>`
   is first gated by media/viewport via `linkStylesheet` (styles.go:164, itself
   using `css.MediaMatches`), then fetched through `load.ResourceContext.Fetch`
   under the document's ACL policy, and parsed with `css.Parse` (styles.go:110).
3. A hard rule limit of `maxStylesheetRules = 1_000_000` (styles.go:25) with a
   soft warning at 25 000 rules bounds hostile input.
4. The resulting `[]*css.Stylesheet` plus the internal UA/helper sheets
   (`prepare.SimplifyChromeCSS`, `prepare.SimplifyMediaWikiCSS` parsed at
   simplify.go:49/55) are handed to layout as `opts.Sheets`.

### 4.2 Cascade in the layout consumer (internal/layout/style_cascade.go)

The css package reports *matches and specificity*; layout performs the cascade:

1. `matchedRules(node, pseudoElem)` (style_cascade.go:498) iterates every sheet
   in order through `appendSheetRuleHits` (style_cascade.go:522), which calls
   `css.MediaMatches(rule.Media, ctx.media, ctx.viewportW, ctx.viewportH)`
   (style_cascade.go:535), `containerGateMatches` (style_cascade.go:539, using
   `runic.Container.Cond.Matches(info.inlineSize, info.fontSize)`,
   style_cascade.go:597), and the `@supports` gate
   `css.SupportsMatches(rule.Supports, engineSupportsProperty)`
   (style_cascade.go:543) before selector matching. Media, container, and
   feature-query filtering is layered *on top of* the css package.
2. `appendRuleSelectorHits` (style_cascade.go:556) loops selectors, calls
   `css.Match`/`css.MatchPseudo` and records `css.Specificity` as the
   `ruleHit` triple.
3. `cascadeRaw` (style_cascade.go:623) merges three tiers into one winner map:
   - UA sheet rules (hard-coded `uaRules`, priority `(0,0,0)` order `-1`);
   - author-sheet hits with `(a,b,c)` `r.Order` `r.Layer` `d.Important`;
   - inline style via `css.ParseInline(node.Attribute("style"))` with a
     sentinel specificity `1<<maxIntShift` described as "outranks all normal
     declarations and all sheet important declarations" (style_cascade.go:655).
4. Every declaration passes the shared acceptance gate before it can win:
   `supportedDeclaration` (`style_value_accept.go:57`) checks the value
   against a per-property table (`declarationValueAccepted`,
   `style_value_accept.go:97`). The gate filters sheet declarations at
   `style_cascade.go:647`, inline declarations at `style_cascade.go:658`, and
   pseudo-element declarations at `style_cascade.go:712`, so an invalid later
   declaration cannot replace an earlier valid one. The same predicate decides
   `@supports` value support in `engineSupportsProperty`
   (style_cascade.go:1621): the property must have an apply arm in the
   style-group dispatch and its value must pass `supportedDeclaration`
   (style_cascade.go:1626). The table is partial today; see §10 item 9.
5. `applyCascadeWin` (style_cascade.go:1194) compares
   `(important ⇒ @layer rank ⇒ ids/classes/types ⇒ order)`: any `!important`
   beats any normal value, then `layerBeats` (style_cascade.go:1247) ranks
   layers before specificity. Rank 0 is unlayered and beats every ranked
   layer; among ranked layers, later-declared wins.
6. Custom properties: `mergeCustomProps` (style_cascade.go:68) inherits parent
   values, extracts `--*` declarations, calls
   `css.ResolveCustomProps(declared, parentProps)`, and folds in `@property`
   registrations (`applyRegisteredProps`, style_cascade.go:121);
   `resolveRawVars` (style_cascade.go:170) then expands `var()` in ordinary
   values via `css.ResolveVar` with a lookup into the resolved custom-prop map.
7. Typed conversion happens later in `internal/layout` using `css.ParseLength`/
   `css.ParseColor` (style_properties.go:401, 1245), `css.ParseFontFamily`
   (style_values.go:83), and `css.ParseContainerNameValue`/
   `css.ParseContainerShorthand` (style_container_props.go:31, 84).

### 4.3 Matching walk (inside css, selector → bool)

`Match(sel, n)` (css.go:1216) → `leftmostMatch` (has.go:256):

- The **last part** must match `n` via `matchPart` (css.go:1266): tag
  case-insensitive, `id` exact, every class a whitespace token
  (`hasClassToken`, css.go:1669; non-ASCII whitespace falls back to
  `hasUnicodeClassToken`, css.go:1698), every attribute selector
  (`matchAttrs`/`attrValueMatches`, css.go:1322/1350), every pseudo
  (`matchPseudos`/`matchPseudo`, css.go:1296/1410).
- `matchPseudo` dispatches: `first-child`/`last-child` via element-sibling
  scans (css.go:1489/1509), `nth-child` via `matchNth` (css.go:1608),
  `:has()` via `matchAnyRelative` (css.go:1439), `:not()` via `matchNone`
  (css.go:1450), `link`/`visited` → both mean "a[href]"
  (`isLinkAnchor`, css.go:1479), `root` via `isRootElement` (css.go:1463),
  interactive pseudos → `false` (never match in print).
- Earlier parts step left through the combinator chain (`leftmostStep`,
  has.go:284): `>` child, `+` adjacent, `~` sibling, descendant ancestor.

`:has()` (Selectors 4 relative selectors): `parseRelativeSelectorList`
(has.go:125) parses `>`, `+`, `~`, or descendant-relative argument lists;
`matchRelative` (has.go:180) anchors the argument at the subject and uses
`elementDescendants` (has.go:341) for scoped walks. Nested `:has()` and
pseudo-elements inside `:has()` are rejected at parse time (has.go's
`insideHas` flag, css.go:1033).

### 4.4 Media & container evaluation

- `@media` preludes are stored **raw** on `Rule.Media` (css.go:72) at parse
  time; the viewport is unknown to the parser. Conversion later supplies
  `(mediaType, viewportW, viewportH)` to `MediaMatches` (media.go:24): comma
  lists OR, `not`/`only` supported, size features (`width`, `height`,
  `inline-size`, `block-size` with min-/max- prefix) compare against the
  viewport in points with an em base of 12pt (`defaultMediaFontPt`, media.go:9),
  `orientation` supported, unknown features → `false`.
- `@container` preludes are parsed eagerly into `ContainerQuery{Cond}`
  (container.go:20). `Cond` is a boolean tree of `SizeFeature` comparisons
  (`and`/`or`/`not`, container.go:30). `ContainerCond.Matches` (container.go:48)
  evaluates against a container's inline size and font size, supplied by layout
  after box layout (`findSizeContainer` in layout). Nested `@container` inside
  `@media` is flattened (parseNestedAtRule, css.go:489) and the nested query
  *replaces* (not combines) the outer one.

### 4.5 @font-face and @page side channels

- `@font-face` rules are collected into `Stylesheet.FontFaces`
  (parseFontFaceRule, css.go:275; `parseFontFace`, css.go:371). Conversion
  (`convert/prepare/styles.go:191`) iterates these and calls
  `css.FontFaceURLs` (css.go:392) to fetch each `src` through the document's
  resource policy. Font weight/style are intentionally ignored (css.go:66).
- Unnamed `@page` declarations keep raw `margin`/`size` strings in
  `PageStyle` (css.go:59, parsePageRule css.go:192) so physical units resolve
  at the PDF boundary.

### 4.6 text-wrap-style line placement (wave C)

`text-wrap-style` shows the full handoff from a declaration to line breaking.
The css package parses the value like any other property; layout decides
whether it wins and what it does to line placement.

1. **Acceptance.** `declarationValueAccepted` accepts `auto`, `balance`, and
   `stable` for `text-wrap-style` (style_value_accept.go:145-146), and routes
   `text-wrap` through `textWrapShorthandValueAccepted`
   (style_value_accept.go:147-148). The shorthand takes one or two
   whitespace-separated tokens: at most one mode (`wrap`/`nowrap`) and at most
   one style (`auto`/`balance`/`stable`). A repeated mode, a repeated style,
   and any other token are rejected (style_value_accept.go:293-323). `pretty`
   and `avoid-short-last-line` are rejected too (style_value_accept.go:291-292),
   so a paragraph that asks for either keeps greedy wrapping. The same
   predicate answers `@supports (text-wrap-style: ...)`
   (style_cascade.go:1621-1626).
2. **Storage.** `applyTextGroup` routes the three text-wrap properties to
   `applyTextPropsWave3` (style_properties.go:1362-1369). `setTextWrap` splits
   the shorthand: `wrap`/`nowrap` go to `TextWrapMode`, every other token goes
   to `TextWrapStyle` (style_text_props.go:153-163). The longhand stores its
   lowercase value (style_text_props.go:25-26). The fields live on
   `ResolvedStyle` (style.go:240-242) and inherit (style_cascade.go:279-281).
3. **Line placement.** `balanceCanApply` decides per paragraph whether
   balancing runs: the style must be `balance`, and the block must have no
   line clamp, no first-line indent, and no active left or right float
   (inline_balance.go:94-101; the clamp is computed at inline.go:184-190).
   When it applies, `inline.go` computes the balanced width lazily at the
   start of each forced-break segment, meaning the items between two `<br>`s
   (inline.go:240-242), using `balanceSegmentWidth` (inline_balance.go:105-107)
   and `inlineSegmentEnd` (inline_balance.go:113-121).
4. **Bisection.** `balanceLineWidth` packs a trial copy of the segment at the
   full content width with `countInlineLines` (inline_balance.go:128-164). It
   gives up when the trial packs to fewer than two or more than six lines
   (inline_balance.go:84; `minBalanceLines` and `maxBalanceLines` at
   inline_balance.go:16-17), when the width or epsilon is unusable or the block
   has a first-line indent (inline_balance.go:79), when the trial meets a
   forced break, or when an item cannot fit whole (inline_balance.go:143,
   151-153). The search mirrors Blink's ParagraphLineBreaker: the low end
   starts at 80% of the average normal line width (`balanceMinWidthFactor`,
   inline_balance.go:18,50), the high end is the content width
   (inline_balance.go:55), and the loop halves the range until it finds the
   widest width that keeps the same line count, with a one CSS pixel epsilon
   (inline_balance.go:57-65; `pxToPt(1)*e.scale` at inline_balance.go:106). A
   return of 0 means no narrower width kept the count, so normal wrapping
   stands (inline_balance.go:67-69). The packer then breaks at
   `min(lineW, balanceW)` (inline.go:244-250), while alignment still uses the
   full line width, matching Chrome, which narrows only the breaker
   (inline.go:276-281).

Balance does not apply in these cases:

- `stable` is accepted and wraps greedily like `auto`, so it has no separate
  code path (inline_balance.go:3-5).
- `pretty` and `avoid-short-last-line` never reach line breaking because the
  acceptance gate rejects them (style_value_accept.go:291-292). Chrome breaks
  `pretty` with a score-based algorithm instead (compatibility-matrix.md:444).
- Active floats, line clamp, and text-indent opt out
  (inline_balance.go:94-101). The indent opt-out avoids copying a Chrome
  quirk: Chrome applies the balanced width without subtracting the indent
  (inline_balance.go:40-43).
- A paragraph outside the two-to-six line window is left alone
  (inline_balance.go:84), which covers very long paragraphs.
- The bisection is Blink's fallback path. Blink tries its score-based
  `ScoreLineBreaker` first; the two agree when the greedy re-break balances
  the lines, which covers the pinned cases, but can pick different break sets
  when uneven word widths leave several valid sets (inline_balance.go:31-35).

## 5. Cross-package dependencies

### 5.1 What css imports

| Import | Why |
|--------|-----|
| `blinkless/internal/html` | `Match`, `MatchPseudo` and all matching walkers operate on `*html.Node` (custom allowlisted tree, internal/html/html.go:36). This is the only intra-repo dependency |
| stdlib: `errors`, `strconv`, `strings`, `unicode`, `unicode/utf8` | Parsing, identifier/class scanning, number/color conversion |

The import graph is a **strict lower layer**: `internal/css → internal/html →
stdlib only`. This guarantees no import cycle can ever form with layout,
convert, or pdf, and it makes the package independently testable.

### 5.2 Who depends on css

| Consumer | What it uses |
|----------|--------------|
| `internal/layout` (style_cascade.go, style.go, style_properties.go, transform.go, pseudo_content.go, layout.go) | `Parse`, `Match`, `MatchPseudo`, `Specificity`, `MediaMatches`, `SupportsMatches`, `ParseInline`, `ParseLength`, `LengthToPt`, `ParseColor`, `ParseFontFamily`, `ParseNumber`, `ResolveVar`, `ResolveCustomProps`, `ParseContainerNameValue`, `ParseContainerShorthand`, `HasContainerRules` |
| `internal/convert/prepare` (styles.go, simplify.go, prepare.go) | `Parse` for `<style>`/`<link>`/helper sheets; `MediaMatches` for link media gating; `FontFaces`/`FontFaceURLs` for font merging |
| `internal/convert` (outline.go, toc.go) | `ParseSelectors` for `--exclude-from-outline`; `ParseLength` in TOC geometry |
| `internal/outline` (outline.go) | `css.Selector` matching for outline exclusion |
| `internal/imageout` (imageout.go) | Same pipeline in image mode (load → parse → css → layout → raster) |
| `api.go` / `internal/cli` / `internal/settings` | *None directly*: settings inject sheets/media via layout options |

**Import-direction rule**: nothing below css (html) knows css exists; css never
imports layout, convert, pdf, load, or settings. All viewport/media/container
*context* is passed as arguments, never pulled in.

### 5.3 Latent coupling worth knowing

- `Stylesheet.Order` is owned by the parser (single sheet), but the cascade
  needs a document-global order across many sheets. Layout treats the order as
  per-sheet and relies on sheet iteration order for cross-sheet tiebreaks
  (style_cascade.go:225). Callers who build compound sheet lists must keep
  document order intact (ResourceContext.CollectSheets does).
- `Selector.spec` cache means **mutating a parsed Selector's parts after
  parsing yields stale specificity**; `Specificity` only falls back to a walk
  when `specValid` is false (css.go:1738). Today no caller mutates parsed
  selectors, but the invariant is documented in the field comment (css.go:86).

## 6. Design decisions & trade-offs

1. **Zero third-party CSS dependencies.** The project rule is pure Go, no cgo,
   no third-party HTML/CSS/PDF APIs. Unlike the layout package's one narrow
   exception (OpenType shaping via `github.com/go-text/typesetting`), the CSS
   layer is 100% handwritten: parser, selector engine, cascade helpers, unit
   math. This keeps `CGO_ENABLED=0` trivially satisfiable and the security
   surface small.

2. **Allowlist over completeness.** The package deliberately implements a
   *report* subset: no full Selectors 4, no grid-of-everything, no animations.
   The compatibility matrix
   (documentation/compatibility-matrix.md §2) is the normative contract, and
   `documentation/fidelity.md` frames it: this is Tier 1–2 "leave wkhtmltopdf
   for most jobs", explicitly **not** Chrome-quality arbitrary-web print
   (`fidelity.md` Tier 3 stays banned for a pure-Go PDF engine based on HTML templates without any wrappers).

3. **Degrade, never panic, never misapply.** Two distinct softenings:
   - *Garbage is dropped*: recoverable parse debris is skipped; only
     unbalanced braces error (`errUnbalanced`, `errNoBlock`, css.go:526) and
     even then `ParseNeverPanics` (css_test.go:178) pins the "no panic on
     adversarial input" property.
   - *Unsupported selectors never degrade to the host*. This is the
     single most subtle design rule in the package. `writePseudoLiteral`
     (selector_parser.go:123-129) and `appendSimplePseudo`
     (selector_parser.go:414) deliberately **keep unknown/unmatchable pseudos
     on the compound** so that, e.g., `li:target` cannot silently become `li`
     and apply `:target`'s declarations to every list item (the code comment at
     selector_parser.go:123-128 documents a real regression:
     `p::before{width:120pt}` used to crush wiki body columns, and `li:target`
     would otherwise paint every reflist item blue). Unmatchable pseudos are
     stored, matched as `false`, and thus **suppress** the whole rule.
     `:first-line`/`:first-letter` are rejected outright
     (selector_parser.go:431); `:is()`/`:where()` parse as functional pseudos
     with their argument selector lists (selector_parser.go:397-459) and match
     when any argument matches (`matchAnySelector`, match.go:576).

4. **Parse early, evaluate late.** Media preludes stay raw strings; container
   queries and `:nth-child` arguments are eagerly compiled (integer `nthForm`,
   css.go:1566). The former exists because the viewport is only known at
   conversion; the latter exists because matching is then pure integer math,
   a measurable hot path in the per-element cascade.

5. **Selectors-4 specificity for functional pseudos.** `:has()`/`:not()`
   contribute the specificity of their most specific argument (css.go:1738),
   not a flat class-level count. This matches current CSS behavior and avoids
   the classic `:not()` over/under-matching bugs.

6. **One owner for custom-property policy.** `css.ResolveCustomProps`
   (values.go:448) is documented as "the single place custom-property policy
   lives": inherited overlay + declared values, memoized expansion, cycle stack
   (cycles → empty → invalid). Layout drives it; css defines it. `ParseColor`
   handles `var()` only at the fallback level (values.go:182), a deliberate
   simplification flagged with a `ponytail:` note.

7. **Caller-supplied context keeps css pure.** Matching needs the viewport
   (`MediaMatches`), a container's used size (layout), and a base font size
   (em/rem in `LengthToPt`, container.go:113). All are function arguments.
   This purity is what lets layout do container-query gating in two passes
   (css only *reports* `HasContainerRules`, layout.go:831) and what keeps css
   trivially fuzzable.

8. **Micro-allocations matter in the hot path.** `matchPart`, `hasClasses`
   and `matchAttrs` are called once per (element, candidate-selector) pair;
   the class/word token scanning avoids `strings.Fields` allocations
   (`containsWord`/`hasClassToken`, css.go:1381/1669, with a Unicode fallback
   only on non-ASCII whitespace), the color parser scans channels in place
   (`parseRGBColor`, values.go:267), and `namedColorTable` is a cached global
   (values.go:594), all with explicit `//nolint:cyclop` notes explaining why
   the linear dispatch stays.

9. **Unit math tuned for IEEE float cancellation.** `LengthToPt`
   (container.go:113) deliberately multiplies-then-divides physical units
   (`val * 72 / 25.4`) so `25.4mm` cancels cleanly to `72pt` instead of
   `71.999…`; `px` maps at `0.75` (96 CSS px/in → 72 pt/in). `%` and viewport
   units return `false`, leaving the decision to caller policy (e.g.
   line-height inherits, percentages resolve against the containing block in
   layout).

## 7. Notable patterns & invariants

- **Parse-order stability**: `Rule.Order` is a per-sheet monotonic counter
  owned by `Parse` (parseOneRule, css.go:328); across sheets, callers preserve
  document order and use sheet order as the final tiebreak: the CSS "later
  wins" rule is reproduced exactly by `applyCascadeWin`'s order comparison.
- **At-rules taxonomy**: `@media`/`@container`/`@supports` flatten into the
  rule list with a gate (`Rule.Media`, `Rule.Container`, `Rule.Supports`;
  dispatch css.go:264-270); `@layer` stamps a cascade rank on each rule
  (`Rule.Layer`, at_layer.go:22); `@page`/`@font-face`/`@property` are
  side-channel structs (`PageStyle`, `FontFace`, `Stylesheet.Properties`,
  at_property.go:19); `@import` is recorded in `Stylesheet.Imports` for the
  collection layer to fetch; `@keyframes`, `@charset`, CSS nesting, and unknown
  at-rules are skipped.
- **Inline is strongest**: layout gives `style=""` the sentinel specificity
  `1<<maxIntShift`, stronger than any sheet rule including `!important`
  (style_cascade.go:655). `!important` in inline style is parsed by
  `isImportant` but the sentinel already dominates.
- **Pseudo-element shapes**: `::before/::after` never match the host
  element (css.go:1266 checks `part.PseudoElement != ""`), only through
  `MatchPseudo` used by layout's pseudo-content path (`pseudo_content.go`,
  style_cascade.go:578).
- **Never-match sets**: interactive pseudos (`hover/active/focus/target`) are
  *kept on the compound* but `matchPseudo` returns `false` for them (css.go:1410),
  because a static PDF has no pointer/focus state.
- **Link semantics**: `:link` and `:visited` both mean "an `<a>` with a
  non-empty `href`" (any scheme incl. `#fragments`), since print has no history.
- **Root definition**: `:root` matches the document element while excluding
  the synthetic `#document` wrapper (`isRootElement`, css.go:1463; the HTML
  tree wraps everything under an element named `#document`).
- **Container condition grammar**: `or < and < not < paren/feature` precedence
  with top-level keyword splitting (`splitCondKeyword`, container.go:390),
  range forms `(`name `>` 20em / `20em < name`)` supported as single
  comparisons (rangeFeatureFromTokens, container.go:668), and only
  `width`/`inline-size` features (`matches`, container.go:85).
- **String/paren scanning shared**: `matchingParen`/`skipQuoted`/`takeParen`
  (has.go:12, container.go:442, has.go:49) are the single implementation of
  balanced-paren/quoted scanning reused by `:has()`, `:not()`, media features,
  and container conditions, honoring backslash escapes.
- **Identifier allowlist**: compound tags, ids, classes, attribute names and
  property names all route through `validIdent`/`validPropName`
  (css.go:1190, values.go:56): lowercase `[a-z0-9-]`, no leading digit.
  Property names that fail are dropped with their declaration.

## 8. Security considerations

The css package itself executes no code and fetches nothing, but it sits on the
attack path for hostile HTML, so its posture matters:

- **Parsing is memory-safe by construction**: bounded index scanning,
  quote/paren tracking with sentinel errors, and the `ParseNeverPanics`
  regression test (css_test.go:178). There is no regex engine, no eval, no
  network access anywhere in the package.
- **Resource amplification is capped outside css**: linked-stylesheet and
  `@font-face` fetches are governed by `internal/load`'s ACL and by
  `ResourceContext.CollectSheets`'s rule limits (soft warn 25k, hard cap 1M,
  styles.go:21/33/63); see `documentation/THREAT-MODEL.md`.
- **No CSS-triggered exfiltration**: `url()` is only honored in
  `@font-face src` via `FontFaceURLs` (and only through the loader's policy);
  `background: url(...)` etc. are inert raw strings. Media queries cannot
  probe anything beyond the numbers the caller supplies.
- **The `--allow-local-files` / `--allow` story applies at load, not
  css**; css just consumes whatever sheets the loader delivers.
- Fuzz-relevant surface: `Parse`, `ParseSelectors`, `ParseInline`,
  `ParseLength`, `ParseColor`, `LengthToPt`, `MediaMatches`: all pure string
  → value functions ideal for fuzzing harnesses (see §9).

## 9. Testing & verification

Unit tests live in the package (973 + 161 + 189 + 86 + 74 + 55 + 52 lines of
`*_test.go`):

| Test file | Coverage |
|-----------|----------|
| `css_test.go` | Parse basics; selector lists; comments/garbage; `!important`; media parse; at-rule skipping; `@page`; order & nested-media order; **never-panics**; unbalanced-brace errors; inline parse; compound parsing; `Match`; link/visited; root; attr operators; sibling combinators; `Specificity`; `ParseLength`/`ParseNumber`/`ParseColor`/`ParseFontFamily`; newline-preserving comment strip; `LengthToPt`; custom-property resolution incl. inheritance overlay, cycles, deep chains, self-reference fallback; strict `ParseSelectors` |
| `has_test.go` | `:has()` parse+match, invalid forms, `:has` specificity, `:not()` matching |
| `container_test.go` | `container` shorthand/name parsing, `@container` rule parsing, `Cond.Matches` truth tables, non-size container rejection at eval, `HasContainerRules`, invalid preludes skipped |
| `media_test.go` | type matching, size features (min-/max-/equal), orientation, empty media-type legacy behavior |
| `pseudo_element_drop_test.go` | Regression: `::before/::after` selectors never apply declarations to the host element |
| `target_pseudo_test.go` | Regression: `li:target` does not match a bare `li` (wiki reflist highlight bug class) |
| `wiki_print_hide_test.go` | Real-world Wikipedia print-hide sheet parses and the `.noprint` class semantics survive |

Cross-package validation:

- `internal/layout/*_test.go` and golden `fixtures` (testdata/golden/) exercise
  css end-to-end: `css.Parse` of embedded `<style>`, cascade output, container
  gating (`layout.go:831` + `containerGateMatches`), and pseudo-content via
  `MatchPseudo`. Fixtures 22/29/38 (float), 25/28/32–35 (flex/grid),
  30/37 (orphans/widows) are cited by the compatibility matrix as evidence.
- `internal/layout/inline_balance_test.go` covers the wave C text-wrap path:
  the bisection arithmetic on synthetic items (inline_balance_test.go:95),
  Chrome 143 line-break comparisons for balance, the shorthand, `stable`, and
  forced-break segments (inline_balance_test.go:160, 202, 227), the six-line
  and `nowrap` opt-outs (inline_balance_test.go:271, 303), the full-width
  alignment invariant (inline_balance_test.go:322), and the acceptance gate
  through both `supportedDeclaration` and `engineSupportsProperty`
  (inline_balance_test.go:356).
- `internal/convert/outline.go` `--exclude-from-outline` selectors parse via
  `css.ParseSelectors` and are exercised by outline tests.
- Run via `make test` / `go test ./internal/css/...` (standard repo flow;
  verified by CI workflow .github/workflows/ci.yml).

## 10. Known limitations, deferred items & open questions

Ground truth: documentation/compatibility-matrix.md (the normative allowlist)
and documentation/deferred.md. Confirmed gaps in css itself:

1. **Selectors**: `:first-line`/`:first-letter` rejected
   (selector_parser.go:431). `:has()` forbids nested `:has()` and
   pseudo-elements inside arguments (selector_parser.go:372). No escaped-ident
   unescaping beyond the naive `\` copy.
2. **At-rules**: `@import` is parsed into `Stylesheet.Imports`
   (import.go:10) and fetched by `prepare` under the same loader ACL as
   `<link>`, with media gating and an 8-deep nesting cap (maxImportDepth,
   prepare/styles.go:21; collector at :203, cap check at :215). `@supports`,
   `@layer`, and `@property` are parsed (`at_supports.go`, `at_layer.go`,
   `at_property.go`); CSS nesting, `@charset`, `@keyframes`, and unknown
   at-rules are skipped. Animations/transitions are explicitly out of scope
   (static cascaded values only; `@keyframes` is skip-parsed at css.go:285;
   deferred.md §4).
3. **Values**: `%` and viewport units (`vw`/`vh`) parse but `LengthToPt`
   returns `false` for them (container.go:113): layout decides policy; a
   curated named-color subset, not CSS Color 4 (values.go:594, ponytail note);
   `var()` inside `ParseColor` resolves fallback only (no prop map; layout's
   `ResolveCustomProps` fills that gap, values.go:182 comment).
4. **Media queries**: unknown features → `false` (media.go:158); only
   width/height/inline-size/block-size + orientation; no `resolution`,
   `pointer`, `prefers-*`, no `@media print and (min-width:...)` distinction
   by page box (viewport = page content box).
5. **Container queries**: size-only (`width`/`inline-size`), no style queries,
   no `container-type: style`, range comparisons limited to a single
   three-token form (`rangeFeatureFromTokens`, container.go:668); nested
   `@container` replaces rather than combines the outer query
   (parseNestedAtRule, css.go:489).
6. **Specificity cache staleness risk**: hand-built or post-parse-mutated
   selectors recompute specificity, but mutating a *parsed* selector's parts
   after `setSpec` leaves the stale cache (css.go:86 comment; no current
   caller does this).
7. **Layer rank scope and important ordering**: `!important` is resolved as a
   single global tier over all normal declarations (style_cascade.go:1210); UA
   rules are a fixed lowest tier. `@layer` ranks are numbered per sheet
   (`Stylesheet.layerRank`, at_layer.go:64) and compared as integers before
   specificity (`layerBeats`, style_cascade.go:1247), so layer names that
   appear in multiple sheets rank independently.
8. **Performance ceilings**: selector matching is O(selector parts × related
   nodes) per element with no index (no class/id dispatch table); acceptable
   for report documents, warned at 25k rules in preparation
   (styles.go:63). The `ponytail:` notes (e.g. ContainerCond tree
   simplification, container-name wire form) flag future internal cleanups.
9. **Declaration acceptance is partial (CSS-01b)**: `declarationValueAccepted`
   models about 40 property names (`style_value_accept.go:97`); a property
   without an entry hits the default arm and is accepted
   (`style_value_accept.go:164`), because its applier either takes the value as
   written or drops an invalid one. Wave C added the text-wrap pair:
   `text-wrap-style` accepts `auto`/`balance`/`stable`
   (style_value_accept.go:145-146), and `text-wrap` accepts one or two tokens
   from the mode and style sets, at most one of each
   (style_value_accept.go:147-148, 289-323). An invalid higher-priority value
   for an unmodeled property can still win the cascade, and an
   `@supports (prop: bogus)` query reports true whenever the property has an
   apply arm. Extending the table to the advertised property list is the next
   gate for CSS-01b.

Open questions an architect should keep an eye on:

- Cross-sheet `Order` rebasing: if layout ever needs a single global order,
  `Stylesheet` will need a rebase helper rather than per-sheet counters.

## 11. Related documents

- `../architecture.md`: package map overview (`internal/css` row).
- `../compatibility-matrix.md`: normative allowlist: supported properties
  (§2), selectors, media, flex/grid stages, pagination evidence fixtures.
- `../fidelity.md`: tiers (Tier 1/2 shipped; Tier 3 arbitrary-web banned),
  "print CSS subset" framing, `:nth-child`/attr/siblings shipped note (16.1).
- `../THREAT-MODEL.md`: ACL and network-request policy around stylesheet and
  font-face fetching.
- `../deferred.md`: deferred CSS/JS/SPA items to keep the matrix honest.
- `../fonts.md`: `@font-face`/`font-family` interplay with `FontFaces` and
  `ParseFontFamily`.
- Sibling architecture docs in this directory:
  - `05-html-parser.md`: the `internal/html` tree and tokenizer css matches
    against.
  - `07-layout.md`: style resolution/cascade consumption of this package
    (style_cascade.go, style.go, style_properties.go, transform.go,
    pseudo_content.go).
  - `04-load.md`: the loader whose ACL governs stylesheet/font fetches that
    `ResourceContext.CollectSheets` triggers.
  - `08-convert-pipeline.md`: prepare/simplify/outline consumers
    (`.Sheets`, `FontFaces`, `ParseSelectors`).
  - `10-imageout-svg.md`: image-mode reuse of the same css→layout path.
