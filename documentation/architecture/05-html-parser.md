# HTML parser: WHATWG tokenizer and insertion-mode tree builder

## 1. Responsibility & position in the pipeline

`internal/html` is the parse stage. The pipeline order is load, parse, CSS
cascade, then layout as a drawing list (`layout.DisplayList`). The package
turns the decoded bytes of one document (`<body>` string, or raw UTF-8
document bytes) into an in-memory DOM tree that the downstream stages walk:

- `internal/css` **matches selectors against the tree** (`Match`,
  `internal/css/match.go:95`; namespace-aware type matching at
  `match.go:276-290`; the synthetic root is special-cased by `isRootElement`,
  `match.go:539`);
- `internal/layout` **resolves styles per node and builds the box tree and
  drawing list** (`Layout`, `internal/layout/layout.go:1111`);
- `internal/convert/prepare` **is the main document path** (`Document`,
  `internal/convert/prepare/prepare.go:210`; the parse call is at
  `prepare.go:236`);
- `internal/pubstate` **carries the tree across the public/internal
  boundary** (`Styled`, `internal/pubstate/state.go:13-23`);
- the public packages wrap or copy it: `html.Parse`
  (`html/parse.go:36-48`), `css.Apply` (`css/css.go:97`), `markup.Parse`
  (`markup/markup.go:59-65`).

The package doc (`html.go:1-6`) describes the scope:

> *"implements a tokenizer and tree builder for the HTML subset blinkless
> accepts: tags, attributes, text, comments, doctype, self-closing and void
> elements. The tokenizer follows the WHATWG tokenizer's recovery rules for
> unfinished comments, tags, declarations, and quoted attribute values.
> Script/style contents are kept as raw text and stripped at the layout
> stage."*

The parser implements the WHATWG tokenizer state machines and the
insertion-mode tree-construction skeleton, but it is **not a full browser
parser**: select and template states, fragment parsing, script-data escaped
states, and quirks-mode layout effects are not implemented. The measured gaps
are listed in §10.

Two entry points cover the internal callers:

- `Parse(source string) (*Node, error)` (`html.go:154-161`) - parse a Go
  string; used by `markup.Parse` (`markup/markup.go:60`) and the public `html`
  wrapper (`html/parse.go:39`).
- `ParseDocument(body []byte) (*Node, error)` (`html.go:166-170`) - raw
  document bytes; strips a leading UTF-8 BOM (mirroring `load.IsHTML`,
  `internal/load/load.go:303`).

Both return an error, but no input currently produces one: malformed markup
is recovered (§3.8) and over-deep nesting is capped (`tree.go:11`), never
reported.

The package holds the **shared DOM representation in the codebase**;
`markup.Node` (`markup/markup.go:47-55`) is a detached public copy, not a
second parser. The `Node` shape is a de-facto cross-cutting contract.

## 2. Package / file map

| File | Responsibility | Lines |
|------|----------------|-------|
| `internal/html/html.go` | Node model; input preprocessing; comment, tag, attribute, and raw-text tokenizer state machines; `Parse` / `ParseDocument` | 922 |
| `internal/html/doctype.go` | Structured `Doctype` token; WHATWG doctype tokenizer states; quirks / limited-quirks / no-quirks classification | 564 |
| `internal/html/foreign.go` | `Namespace` and `Attr`; SVG element and attribute adjustment tables; foreign attribute namespaces; MathML/SVG integration points; breakout tags | 274 |
| `internal/html/tree.go` | Insertion-mode tree builder; implicit html/head/body; scope rules; implied end tags; foreign-content dispatch; resource caps | 1566 |
| `internal/html/formatting.go` | Active formatting list, Noah's Ark clause, reconstruction, adoption agency | 451 |
| `internal/html/tables.go` | Table insertion modes (in table, caption, column group, table body, row, cell) and foster parenting | 603 |
| `internal/html/entities.go` | Context-aware character-reference decoder (text and attribute rules) | 252 |
| `internal/html/doc.go` | Package doc indirection | 2 |
| `internal/html/html_test.go` | Unit tests (same package, so tokenizer internals are tested directly) | 1938 |
| `internal/html/conformance_test.go` | Corpus runner: manifest, category planning, report, baseline evidence | 791 |
| `internal/html/conformance_tokenizer_test.go` | Tokenizer category: expected-stream parsing and comparison | 589 |
| `internal/html/conformance_tree_test.go` | Tree-construction category: `.dat` parsing and tree-dump comparison | 806 |
| `internal/html/fuzz_test.go` | `FuzzParseHTML`: no-panic fuzzing | 31 |

Total: 8,789 lines (roughly half test code). The package is split by
responsibility, not length, and every file stays under the ~2,000-line soft
limit (see `AGENTS.md`, "Code structure").

## 3. Key types, functions & entry points

### 3.1 Node model (`html.go:19-133`)

```go
type NodeType int          // html.go:19

const (
    NodeUnknown NodeType = iota  // 0: zero-constructed nodes must not be elements
    ElementNode
    TextNode
    CommentNode
    DoctypeNode
)

type Node struct {         // html.go:32
    Type      NodeType
    Name      string             // lowercased for HTML; adjusted case for foreign
    Namespace Namespace          // HTML (zero value), SVG, or MathML
    Attrs     map[string]string  // keys lowercased / adjusted
    AttrList  []Attr             // ordered attrs, with foreign namespaces
    Text      string             // text / comment / doctype content
    Doctype   Doctype            // structured doctype fields
    Mode      DocumentMode       // document mode; only meaningful on the root
    Children  []*Node
    Parent    *Node
}
```

Key methods:

| Member | Line | Purpose |
|--------|------|---------|
| `(*Node) Attribute(name string) string` | 49 | Case-insensitive attribute lookup: exact key first, lowercased fallback for HTML-style callers. Missing attr -> `""`. |
| `(*Node) FirstChild(name string) *Node` | 64 | First **element** child with the given name, or nil. |
| `(*Node) TextContent() string` | 75 | Concatenated descendant text (comments/doctype contribute nothing). |
| `(*Node) Walk(f func(*Node))` | 84 | Pre-order (document-order) recursive walk. The main iteration primitive used by every consumer. |
| `(*Node) WalkUntil(f func(*Node) bool) bool` | 94 | Pre-order walk with early stop; reports whether the full tree was visited. |
| `(*Node) FindFirst(pred func(*Node) bool) *Node` | 109 | First pre-order match, or nil (used by the public `html.Document.Find`, `html/parse.go:58`). |
| `(*Node) TextContentOf(name string) string` | 127 | Text content of the *first* element descendant with that name. |

Foreign attributes keep their adjusted names (`viewBox`) or qualified names
(`xlink:href`) as map keys, and the namespace is preserved in `AttrList`
(§3.5).

### 3.2 Public entry points

| Function | Line | Purpose |
|----------|------|---------|
| `Parse(source string) (*Node, error)` | 154 | Preprocess, scan, and build a tree with a synthetic root named `#document`. Streaming: `scanTokens` emits tokens to `builder.appendToken`, so no whole-token slice is retained. |
| `ParseDocument(body []byte) (*Node, error)` | 166 | Bytes -> tree; strips a leading UTF-8 BOM (`html.go:167`). |

### 3.3 Tokenizer internals

Input preprocessing (`preprocessInput`, `html.go:209-250`) applies the WHATWG
input-stream rules: CRLF and CR become LF, and invalid UTF-8 byte sequences
become U+FFFD. NUL bytes stay in the stream and are handled per state.
`scanTokens` runs preprocessing once before the scan loop (`html.go:271`).
When the source has no CR and is valid UTF-8, the original string is returned
unchanged (`html.go:210-212`).

| Symbol | Line | Purpose |
|--------|------|---------|
| `type tokenKind` / `tokDoctype, tokStart, tokEnd, tokText, tokComment` | 175-181 | Token classification. |
| `type token struct { kind; data; attrs []string; selfClosing bool; doctype Doctype }` | 183-189 | Token payload. `attrs` is an **interleaved name,value slice** (not a map) to avoid per-token map allocation. |
| `type tokenSink func(token)` | 192 | Push-style token consumer (streaming). |
| `tokenize(src) ([]token, error)` | 196-204 | Test-only collector: buffers all tokens via the sink. The error return is always nil. |
| `scanTokens(src, emit)` | 270-314 | Main scanner loop. Dispatches on `<`; a bare `<` becomes text. |
| `scanBang(src, pos, emit)` | 318-328 | `<!-- comment -->`, case-insensitive `<!doctype ...>`, or a bogus comment. |
| `scanBogusComment(src, from, initial, emit)` | 332-343 | Consumes to `>` or EOF as comment data; used for `<!bogus` and `<?...?>` (the latter via `scanTokens`, 302-303). |
| `scanComment(src, pos, emit)` | 360-465 | Comment state machine (states at 346-355) with unfinished-comment recovery. |
| `scanEndTag(src, pos, emit)` | 470-518 | `</name>`; attributes on end tags are parsed and ignored (509-517); non-letter names become bogus comments. |
| `scanTagAttributes(src, i)` | 539-725 | Attribute state machine (states at 521-533). Returns interleaved attrs, the self-closing flag, and the index after `>`. |
| `decodeAttributeReference(src, i)` | 742-748 | Attribute-context character reference; an ambiguous or historically flushed reference decodes to a bare `&`. |
| `scanStartTag(src, pos, emit)` | 753-808 | Start tags plus raw-text content capture; `plaintext` swallows the rest (799-805). |
| `rawTextMode(name)` | 823-834 | Content mode by element: RCDATA (`title`, `textarea`), RAWTEXT (`style`, `xmp`, `iframe`, `noframes`), script data (`script`). `noscript`/`noembed` are deliberately absent (scripting disabled). |
| `scanRawText(src, from, name)` | 841-914 | Raw content up to the real closing tag; a partial end tag at EOF is dropped or re-emitted as text per the eof-in-tag rules (881-889). |
| `replaceNUL(s)` | 258-264 | Maps literal NUL to U+FFFD in names, values, comments, and raw text. |

Character references live in `entities.go`: `UnescapeEntities` for text
context (`entities.go:25-58`), `decodeCharRefAt` with the attribute rules
(`entities.go:67-129`), the numeric end state with Windows-1252 C1 mapping
(`entities.go:149-185`, `200-240`), and two named references added after the
standard library's table snapshot (`entities.go:134-143`).

### 3.4 Doctype token and document mode (`doctype.go`)

`Doctype` (`doctype.go:12-19`) is the structured token content: `Name`,
`PublicID`/`HasPublicID`, `SystemID`/`HasSystemID`, `ForceQuirks`. The raw
declaration text stays on `Node.Text`; the structured fields travel on the
token and the node.

`scanDoctype` (`doctype.go:218-553`) implements the full WHATWG doctype state
machine; the state constants are listed at `doctype.go:192-211`. EOF in any
state but bogus forces quirks (`doctype.go:544-548`).

`DocumentMode` (`doctype.go:21-32`): `NoQuirks` (zero value),
`LimitedQuirks`, `Quirks`; `String` names them for tests and diagnostics
(`doctype.go:35-46`).

`classifyDocumentMode` (`doctype.go:143-179`) applies the standard's
"initial" insertion-mode tables: force-quirks or a non-`html` name means
quirks; exact and prefix public/system identifier tables (`doctype.go:52-138`)
decide the rest; the HTML 4.01 frameset/transitional identifiers depend on
whether the system identifier is missing or empty (`doctype.go:165-176`). The mode is set
in `processInitial` (`tree.go:193-198`), defaulted to quirks at EOF
(`finish`, `tree.go:107-111`), and copied to `root.Mode` (`tree.go:135`).

The mode crosses the public boundary as `pubstate.Styled.Mode`
(`internal/pubstate/state.go:16`), populated by `css.Apply`
(`css/css.go:29-38`). Quirks-mode *layout* effects are not applied; the mode
is exposed as data (plan row HTML-MODE-01).

### 3.5 Foreign content: namespaces and adjustments (`foreign.go`)

`Namespace` (`foreign.go:8-19`) and `Attr` (`foreign.go:21-28`) are the
representation added by HTML-FOREIGN-01. The standard's tables are
transcribed in full:

- SVG element-name adjustments (`foreign.go:32-70`), e.g. `clippath` ->
  `clipPath`;
- SVG attribute-name adjustments (`foreign.go:74-133`), e.g. `viewbox` ->
  `viewBox`;
- foreign attribute namespaces (`foreign.go:144-156`): `xlink:*`, `xml:*`,
  `xmlns`, `xmlns:xlink`.

`adjustForeignElementName` (`foreign.go:160-168`) and `adjustAttributeName`
(`foreign.go:174-200`, including MathML `definitionurl` -> `definitionURL`)
apply them during insertion.

Integration points: `isMathMLTextIntegrationPoint` (`foreign.go:204-215`),
`isHTMLIntegrationPoint` (`foreign.go:220-241`). Breakout tags:
`foreignBreakoutTags` (`foreign.go:245-254`) and `isForeignBreakout`
(`foreign.go:259-274`; `font` breaks out only with `color`, `face`, or
`size`).

The tree side is `foreignToken` (`tree.go:843-886`), `inForeignStartContext`
(`tree.go:888-903`), `currentIsForeignText` (`tree.go:909-916`),
`popForeignBreakout` (`tree.go:918-927`), and `closeForeignElement`
(`tree.go:1213-1238`).

### 3.6 Tree builder internals (`tree.go`)

`insertionMode` (`tree.go:20-36`): initial, before html, before head, in
head, after head, in body, in text, in table, in caption, in column group,
in table body, in row, in cell, after body, after after body.

`treeBuilder` (`tree.go:39-52`) carries the root, the open-element stack, the
current mode, `textReturn` (mode to restore after a raw-text element), the
`head` and `form` pointers, the document mode, `ignoreNextLF`,
`fosterParenting`, pending table text, and the active-formatting list.

Dispatch and flow:

| Symbol | Line | Purpose |
|--------|------|---------|
| `appendToken(tok)` | 88-98 | Flushes pending table text for non-text tokens, then reprocesses a token up to `maxReprocess` (15) times across mode switches. |
| `finish()` | 102-140 | The EOF mode chain: initial -> before html -> before head -> in head -> after head, inserting omitted `html`, `head`, and `body`; EOF in a raw-text element pops it and reprocesses under `textReturn` (126-133). |
| `processToken(tok)` | 144-183 | Foreign-content check (`foreignToken`, 843) then mode dispatch. |
| `processInitial` | 187-221 | Doctype classification; leading whitespace dropped; non-whitespace starts the body chain. |
| `processBeforeHTML` / `processBeforeHead` / `processInHead` / `processAfterHead` | 223 / 259 / 300 / 375 | Implicit `html`/`head`/`body`; head-content routing (335-343); the in-head `template` end tag pops through the open template (352-360); `html`/`body` attribute merging (1151-1160); head content seen after `</head>` is pushed back through `processHeadContent` (1135-1148). |
| `processInText` | 430-445 | Raw-text element content; end tag pops and restores `textReturn`. |
| `processInBody` / `processInBodyStart` / `processInBodyEnd` | 447 / 477 / 671 | Paragraph, list, and definition closing (697-722); headings (502-511, 723-731); `pre`/`listing`/`textarea` LF swallow (457-460, 512-518, 610-616); button-in-scope closing (545-554); formatting start tags reconstruct and push (555-591); void elements reconstruct (592-600); `<image>` -> `img` (606-609); `select`/`option` legacy closing (634-638); `template` end tag (763-766); `table` enters the table modes (647-655); formatting end tags go to the adoption agency (674-678). |
| `processAfterBody` / `processAfterAfterBody` | 785 / 814 | Post-body comments and whitespace; stray content re-enters the body. |

Insertion helpers: `insertHTMLElement` (932), `insertNode` (994),
`insertForeignElement` (1039), `insertChildAt` (1021), `appropriatePlace`
(947), `appropriatePlaceFor` (953, the adoption agency's adjusted location),
`isFosterTarget` (978), `applyAttributes` (1110), `appendTextToken`
(1052), `appendCommentTo` (1044).

**Self-closing rules by namespace** (`insertNode`, `tree.go:994-1017`): the
self-closing flag is a parse error that is ignored on ordinary HTML elements;
void HTML elements never take content; foreign elements honor the flag
(`(ns == NamespaceHTML && isVoidElement(name)) || (ns != NamespaceHTML && selfClosing)`,
1010). `isVoidElement` is at `tree.go:1558-1566`.

Scope and implied end tags: the default, button, and list-item scope maps end
at MathML text integration points and HTML integration points through the
`integrationPointStops` sentinel (`tree.go:1242-1247`); table scope omits it.
Maps: `tree.go:1249-1305` (`select` is now a scope stop). Helpers:
`findInScope` (1307), `hasInScope` (1322), `hasInScopeAny` (1326),
`isScopeBoundary` (1345), `generateImpliedEndTags` (1362), `popUntilName`
(1373), `popUntilAnyName` (1384), `closePElement` (1403),
`closeOpenListItem` (1418), `closeOpenDefinitionItem` (1435).
`closeHTMLElement` (1192) applies the in-body "any other end tag" rule: a
matching element is popped with implied end tags, but a special non-matching
element makes the end tag a no-op. `closeForeignElement` (1213).

**Formatting elements and the adoption agency** live in `formatting.go`:
`formattingTags` (`formatting.go:12-16`), the Noah's Ark clause
(`pushActiveFormatting`, 49-69), list helpers (35-114), reconstruction
(`reconstructActiveFormatting`, 185-214), the adoption agency algorithm
(`adoptionAgency`, 235-366), and the stack/tree surgery helpers (368-451).
Markers are pushed and cleared in `tree.go` (`insertMarker`, 64-66;
`clearActiveFormattingToMarker`, 70-80) and by caption and cell close
(`tables.go:235-240`, `477-491`). The plan row HTML-FORMAT-01 is closed.

Select: `legacyOptionAutoClose` (`tree.go:1478-1497`) closes `option` and
`optgroup`; the full select insertion modes are still owned by
HTML-CONTEXT-01 (comment at `tree.go:1474`).

### 3.7 Table insertion modes (`tables.go`)

Table scope is `html`, `table`, `template` (`tables.go:13`). The modes:

| Symbol | Line | Purpose |
|--------|------|---------|
| `processInTable` | 17-56 | Text is collected (`pendingTableText`); structure tags dispatch to `processInTableStart` (61-116); anything else goes through `fosterInBody` (120-126). |
| `flushPendingTableText` | 131-160 | Pure whitespace is inserted in place; text mixed with non-whitespace is foster-parented around the table. |
| `processInCaption` / `closeCaption` | 194 / 235 | Caption content uses the in-body rules; close clears formatting to the caption marker. |
| `processInColumnGroup` | 244-304 | `col` insertion, whitespace handling, and leaving the column group. |
| `processInTableBody` | 308-363 | Row and section transitions. |
| `processInRow` | 367-431 | Cells and row transitions. |
| `processInCell` / `closeCell` | 435 / 477 | Cell content uses the in-body rules; close clears formatting to the cell marker. |

Stack and scope helpers: `clearStackToTableContext` (497),
`clearStackToTableBodyContext` (511), `clearStackToTableRowContext` (526),
`hasTableBodyInScope` (540), `hasCellInTableScope` (551).
`resetInsertionMode` (558-606) resets the mode after a table closes; the
template and fragment branches are omitted until those features exist
(`tables.go:555-556`).

Foster parenting reuses the in-body rules: `appropriatePlace`
(`tree.go:947-974`) inserts before the open table when `fosterParenting` is
set and the current node is a foster target (`table`, `tbody`, `tfoot`,
`thead`, `tr`; `tree.go:978-989`).

### 3.8 Recovery model (no sentinel errors)

Wave B replaced the six `errUnterminated*` sentinels and the fatal tokenizer
paths. `errors` is no longer imported by the package. The recovery rules are
the WHATWG ones:

- unfinished comments emit the data gathered at EOF (`html.go:462-464`);
- EOF in a tag name drops the token (`scanEndTag`, 493-495;
  `scanStartTag`, 761-763);
- EOF inside a tag drops the token (`scanTagAttributes` returns `ok=false`,
  724);
- `<!bogus` and `<?...?>` become bogus comments (`scanBang`, 318-328;
  `scanTokens`, 302-303);
- doctype EOF forces quirks unless the state is bogus
  (`doctype.go:544-548`);
- raw-text EOF keeps the text and drops or re-emits a partial end tag
  (`scanRawText`, 881-889, 911-913);
- EOF while a raw-text element is open pops it and reprocesses under
  `textReturn` (`finish`, `tree.go:126-133`);
- NUL bytes become U+FFFD in names, values, comments, and raw text
  (`replaceNUL`, `html.go:258-264`); in body text, NUL is dropped for HTML
  content and replaced for foreign text (`appendTextToken`,
  `tree.go:1052-1057`).

Resource caps replace error returns: `maxElementDepth = 1024` drops elements
deeper than the cap so recursive walks stay bounded (`tree.go:8-11`,
`994-997`), and `maxReprocess = 8` bounds mode-transition ping-pong
(`tree.go:13-15`).

## 4. Data & control flow

### 4.1 Main document path

```text
internal/convert/prepare/prepare.go:210
    prepare.Document(ctx, loader, page, ...)
        -> loader.Load(ctx, page, loadPage)        // bytes + charset gate (UTF-8/ASCII only)
        -> html.ParseDocument(res.Body)            // prepare.go:236; BOM strip + Parse
        -> root *html.Node                         // synthetic "#document" root
        -> Resources.CollectSheets(ctx, root, ...) // <style>, <link>, inline style attrs
        -> Sheets []*css.Stylesheet
        -> Prepared{Root, Sheets, Registry, ...}
```

- **Charset seam:** `internal/load` enforces UTF-8/ASCII before the parser
  sees bytes (`checkDocumentCharset`, `internal/load/load.go:968-984`; the
  `<meta charset>` fallback scan is `metaCharset`, `load.go:1016`).
- **BOM mirror:** `ParseDocument` strips `\ufeff` (`html.go:167`), the same
  way `load.IsHTML` recognizes inline HTML (`load.go:303`).

### 4.2 Tokenizer -> tree flow (inside `Parse`)

```text
Parse(source)                                 html.go:154
  -> preprocessInput(source)                  html.go:209 (CRLF/CR -> LF, bad UTF-8 -> U+FFFD)
  -> scanTokens(source, builder.appendToken)  html.go:270
       text runs / comments / doctype / bogus declarations / PI-as-comment /
       end tags / start tags (raw-text content captured here)
  -> builder.finish()                         tree.go:102 (EOF mode chain)
  -> root Node "#document"
```

`appendToken` (`tree.go:88`) feeds each token through `processToken`
(`tree.go:136`); a handler can ask for reprocessing when it switches modes.
Tree building and scanning are interleaved in one pass: no production path
materializes a token slice (`tokenize`, `html.go:196`, exists for tests
only).

### 4.3 Consumer flows (how the tree is walked downstream)

- **CSS matching** - `internal/css/match.go`: type selectors are
  case-sensitive for foreign nodes and case-insensitive for HTML
  (`match.go:276-290`); structural pseudo-classes walk parent/sibling
  pointers; `:has` walks descendants (`internal/css/has.go`). The synthetic
  `#document` root never matches (`isRootElement`, `match.go:539`).
- **Style resolution and layout** - `internal/layout`: `Layout`
  (`internal/layout/layout.go:1111`) walks the tree to build boxes and the
  display list; per-node styles are keyed by `*html.Node` identity.
- **Public boundary** - `html.Document` (`html/parse.go:23-25`) hides the
  engine node; `css.Apply` (`css/css.go:97`) consumes it; `pubstate` readers
  (`internal/pubstate/state.go:31-49`) hand the internal tree to the public
  packages without leaking internal types in exported signatures.
- **Detached copies** - `markup.Parse` copies the tree and drops parent
  pointers (`markup/markup.go:44-55`, `68-96`).

## 5. Cross-package dependencies

### 5.1 What `internal/html` imports

The import list is `strings` and `unicode/utf8` (`html.go:13-16`), `strings`
(`doctype.go:6`, `foreign.go:6`, `tree.go:6`, `tables.go:10`), and `html`
(aliased `stdhtml`) plus `strings` (`entities.go:3-6`). There are no internal
imports and no third-party imports, so the package is a dependency leaf: the
import graph cannot cycle through it.

### 5.2 Who imports `internal/html` (consumers)

| Package | Why it needs the tree |
|---------|----------------------|
| `internal/css` | Selector matching against nodes (`match.go`, `has.go`). |
| `internal/layout` | Style resolution, box building, drawing list (`layout.go`, plus the many `layout_*` files). |
| `internal/convert/prepare` | Main parse entry (`prepare.go:210`, `236`) and stylesheet collection (`styles.go`). |
| `internal/pubstate` | Public/internal bridge types and readers (`state.go:13-49`). |
| root `css` | Public cascade over a public `html.Document` (`css/css.go:97`). |
| root `html` | Public `Document` and `Find` over the engine node (`html/parse.go:36-70`). |
| `markup` | Detached copy for callers that want an owned tree (`markup/markup.go:59-96`). |

### 5.3 Import-direction rule

`internal/html` sits at the **bottom of the dependency graph**:

```text
public html / css / markup / layout
                 |
        internal/pubstate, internal/convert/prepare
                 |
        internal/css ──┐
        internal/layout ─┼──> internal/html   (leaf; only Go stdlib)
        internal/pubstate ┘
```

The `ponytail` note at `html.go:8` records the migration constraint:

> *"ponytail: custom Node tree (Parent/Attrs/void); migrate to x/net/html only
> if layout/css rewritten, not free delete."*

Replacing this tree with `x/net/html` would be a cross-cutting rewrite of the
consumers, not a local swap.

## 6. Design decisions & trade-offs

### 6.1 WHATWG tokenizer and insertion modes instead of `x/net/html`

- **Why:** the project policy keeps the dependency surface at two direct
  modules, and the product scope is authored HTML (reports, templates), not
  arbitrary websites. Implementing the standard's tokenizer states and
  insertion-mode skeleton buys recovery behavior and a conformance corpus
  without a new dependency.
- **Cost:** the tree-construction coverage is partial. Select/template
  states, fragment parsing, script-data escaped states, and some ruby and
  frameset recovery rules are missing; §10 lists the measured gaps.
- **Migration note:** the ponytail note at `html.go:8` still applies.

### 6.2 Streaming single pass (sink callback)

`scanTokens` emits tokens through `tokenSink` as they are recognized, and
`appendToken` builds the tree in the same loop. Production documents never
materialize a token slice (the `tokenize` collector at `html.go:196` is
test-only), so memory stays bounded by tree size, not token count.

### 6.3 Recovery plus resource caps, not error returns

Malformed markup follows the standard's recovery rules (§3.8); hostile input
is bounded by the depth and reprocess caps instead of failing. `Parse` and
`ParseDocument` keep an `error` return, but the current implementation never
produces one.

### 6.4 Document mode is data

The doctype is tokenized into structured fields and classified into
no-quirks / limited-quirks / quirks (`doctype.go:143-179`). The mode is
carried to consumers (`pubstate.Styled.Mode`, `css/css.go:29-38`); the
layout-stage quirks effects (for example line-height quirks) are not
implemented.

### 6.5 Foreign namespaces and self-closing rules

SVG and MathML are represented with `Namespace`, adjusted names, and
namespaced attributes (`foreign.go`), and foreign content can break back
into HTML (`tree.go:753-837`). The self-closing flag is honored only for
foreign elements; on ordinary HTML elements it is a parse error that is
ignored, and void elements never take content (`tree.go:994-1017`). The
conformance harness compares the flag directly at the tokenizer level
(`conformance_tokenizer_test.go:482-484`); the tree level validates its
effect.

### 6.6 Character references: context-aware decoder

The decoder distinguishes text context (`UnescapeEntities`,
`entities.go:25-58`) from attribute context (`decodeCharRefAt`,
`entities.go:67-129`), where ambiguous ampersands and named references
without a semicolon before `=` or an alphanumeric are flushed literally for
historical reasons. Numeric references follow the spec end state: zero,
values above U+10FFFF, and surrogates become U+FFFD, and the C1 range uses
the Windows-1252 mapping (`entities.go:149-240`). Two named references added
after the standard library's table snapshot are filled in
(`entities.go:134-143`). Decoding happens once, at parse time; downstream
code never re-decodes.

### 6.7 Performance micro-decisions

- `preprocessInput` returns the original string when it contains no CR and
  is valid UTF-8 (`html.go:210-212`), avoiding a copy for the common case.
- Token attrs use an interleaved `[]string` (two slots per pair) instead of a
  map; the map is built only in `applyAttributes` (`tree.go:1016-1036`).
- `scanRawText` searches byte-wise for `<` and only builds the text it keeps
  (`html.go:841-914`).
- `appendTextToken` merges adjacent text nodes with a single pre-sized
  builder (`tree.go:958-1010`).

## 7. Notable patterns & invariants

1. **One shared DOM.** `*html.Node` is the tree the engine walks;
   `markup.Node` is an explicit detached copy. `pubstate` exists so the
   public packages can share the tree without exporting internal types
   (`internal/pubstate/state.go:1-11`).

2. **Streaming scanner plus callback tree build.** No intermediate token
   list in production paths; `tokenize` exists purely for tests.

3. **Adjacent text merging.** Consecutive text tokens merge into one
   `TextNode` (`tree.go:1052-1104`), keeping the tree small and `TextContent`
   deterministic.

4. **First-wins duplicate attributes.** `applyAttributes` keeps the first
   value of a duplicated attribute (`tree.go:1121-1129`).

5. **Lowercasing at the boundary.** HTML element and attribute names are
   lowercased once by the tokenizer; foreign names are case-adjusted per the
   standard's tables. `Node.Attribute` still handles uppercase callers.

6. **Raw-text elements are content, not markup.** `script`/`style`/
   `title`/`textarea`/`xmp`/`iframe`/`noframes` contents are captured
   verbatim as text (with RCDATA decoding for `title`/`textarea`), so `<`
   inside them never opens elements. Hiding them from rendering is delegated
   to UA `display:none` styles in layout (`internal/layout/style_values.go:1837-1846`).

7. **Recovery, not errors.** Malformed input produces a usable tree; the
   resource caps (`maxElementDepth`, `maxReprocess`) are the only hard stops.

8. **Document mode is root data.** `root.Mode` is the only place the mode
   lives in the tree; `Node.Mode` on other nodes is meaningless.

9. **Extension points.** The mode handlers (`tree.go:144-183`), the scope
   maps (`tree.go:1249-1305`), `impliedEndTags` (`tree.go:1270-1273`), and
   the foreign tables (`foreign.go`) are the vocabularies a new construct
   touches.

## 8. Security considerations

The parser is the first trust boundary for *markup*, and the design leans on
**structural inertness plus downstream rendering gating**:

- **No script execution by construction.** Script content is raw text
  (`html.go:823-834`) and the layout UA sheet sets `display: none`
  (`internal/layout/style_values.go:1837-1839`). JavaScript-related flags are
  unknown options and no code path evaluates scripts
  (`documentation/compatibility-matrix.md:921`, `documentation/deferred.md:72`;
  `documentation/THREAT-MODEL.md:14-17`).
- **No form submission path.** POST only via explicit `--post` flags, and no
  cookies are auto-forwarded (`documentation/compatibility-matrix.md:1179`).
- **Deterministic, non-crashing parsing.** Recovery replaces fatal errors;
  `maxElementDepth` bounds recursion; `TestParseUsableTreeNoPanic`
  (`html_test.go:1781`) and `FuzzParseHTML` (`fuzz_test.go:9-30`) lock in the
  no-panic invariant.
- **Charset is enforced before parse.** Only UTF-8/ASCII reaches the parser
  (`load.checkDocumentCharset`, `internal/load/load.go:968-984`).
- **Attribute values are decoded, not executed.** Entity decoding happens at
  parse time (`tree.go:1110-1130`); there is no mechanism to turn attribute
  content into behavior.
- **CDATA is inert.** `<![CDATA[...]]>` is consumed as a bogus comment
  (`html.go:318-328`), so it cannot inject markup.

The model: the parser produces a *safe, inert data structure*; dangerous
HTML features are structurally impossible to execute, not filtered late.

## 9. Testing & verification

### 9.1 Unit tests

All unit tests are **same-package** (`//nolint:all` at `html_test.go:1`), so
tokenizer internals (`tokenize`, `tokenKind`, `scanDoctype`) are tested
directly. Helpers: `mustParse` (`html_test.go:13`), `treeString` (25),
`assertChildren` (56).

Tokenizer and preprocessing tests (`html_test.go:82-791`):
`TestTokenizeAttributes` (82), `TestTokenizeWhitespaceAroundEquals` (111),
`TestTokenizeGreaterThanInQuotedValue` (137), `TestTokenizeComments` (163),
`TestTokenizeDoctype` (204), `TestTokenizeDoctypeIdentifiers` (268),
`TestTokenizeDoctypeIdentifiersMalformed` (304),
`TestDocumentModeClassification` (363), `TestTokenizeDeclarationsAndPI`
(422), `TestTokenizeRawText` (452),
`TestTokenizeRawTextClosesOnlyRealEndTag` (499),
`TestTokenizeNormalizesNewlines` (530), `TestTokenizeNullHandling` (553),
`TestParseReplacesInvalidUTF8` (611),
`TestTokenizeAttributeCharacterReferences` (619),
`TestUnescapeEntitiesNumericEdgeCases` (658), `TestParseTextContexts` (679),
`TestTokenizeBareLessThanIsText` (703), `TestTokenizeRecoversUnterminated`
(737), `TestParseMatchesCollectedTokenBuilder` (791),
`TestUnescapeEntitiesInText` (976).

Foreign-content tests (`html_test.go:831-974`):
`TestParseForeignNamespaces` (831), `TestParseForeignIntegrationPointSVG`
(855), `TestParseForeignIntegrationPointMathML` (872),
`TestParseForeignAttributesHTML` (898), `TestParseForeignAttributesSVG`
(914), `TestParseForeignSelfClosingAndBreakout` (938).

Tree-builder tests (`html_test.go:996-1921`): nesting (996), parent
pointers (1015), void elements (1032), auto-close table (1069), `p` (1101),
lists (1112), table sections (1127), foster parenting (1142), table end
tags (1177), table scope through foreign content (1209), misnested
formatting (1226), `html`/`head`/`body` merging (1283), head-to-body
transition (1347), implicit document structure (1360),
paragraph/list/definition closing (1392), text merging (1446), duplicate
attributes (1487), self-closing (1513), doctype node (1543), comments
(1565), raw-text tree shape (1584), malformed input (1624), stray text and
tags (1661), unclosed tables (1714), EOF recovery (1736), the no-panic
invariant (1781), pre-order walk (1802), `TextContentOf` (1826),
`ParseDocument` (1843), BOM stripping (1859), deep nesting (1880), and the
`NodeUnknown` zero value (1921).

`FuzzParseHTML` (`fuzz_test.go:9-30`) parses arbitrary strings up to 64 KiB
and requires that `html.Parse` never panics.

### 9.2 Corpus conformance harness

`TestHTMLConformance` (`internal/html/conformance_test.go:54-78`) runs the
pinned `html5lib/html5lib-tests` corpus vendored under
`testdata/html-conformance/`. The manifest
(`testdata/html-conformance/manifest.json`) fixes the contract: upstream
revision `9329e64694e7835d0dcff9811e22856ef6ad16f9`, MIT license,
`scripting: false`, the `tokenizer`, `tokenizer-local`, `tree-construction`,
and `tree-construction-local` categories, and the unsupported buckets. The
local categories hold repo-authored cases: `local/rawtext-entities.test` (6
tokenizer cases) and `local/tables-local.dat` (12 tree cases pinning the
table wrappers, adjacent cells, foster-parented content, and the table
end-tag chain).

Comparison rules (from the runner doc comment, `conformance_test.go:15-53`):

- tokenizer cases compare the full token stream (coalesced character runs):
  kind, tag names, attribute maps, self-closing flag, doctype
  name/public/system/force-quirks, and exact text data
  (`conformance_tokenizer_test.go:440-459`);
- tree cases compare node kinds, parent/child order, namespaces, exact text
  and comment data, attribute name/value pairs (with foreign namespaces), and
  the doctype node (`conformance_tree_test.go:593-632`);
- fields the engine cannot represent are never silently skipped: a case
  whose only remaining difference is a missing engine field is counted
  `unsupported` with a reason (template contents, processing instructions,
  document fragments, tokenizer initial states other than the Data state,
  XML-violation coercions, lone surrogates); a case that also differs in
  representable behavior is counted `failed`.

Parser mismatches are baseline evidence, not test failures. The test fails
only on harness problems. `HTML_CONFORMANCE_STRICT=1` turns any failed case
into a test failure (`conformance_test.go:73-77`). Detail depth for the
failure log is `HTML_CONFORMANCE_DETAIL` (default 10,
`conformance_test.go:779-791`).

Evidence is written under the gitignored `temps/html-conformance/`:
`engine-baseline.json` (counts, non-passed case ids, reasons;
`conformance_test.go:662-733`) and `failures.txt` (capped detailed diffs).
An absent corpus or manifest skips with an explicit zero-count message
instead of failing the package (`conformance_test.go:57-65`).

### 9.3 Measured corpus status (2026-10-09 20:33)

Command: `go test ./internal/html -run TestHTMLConformance -count=1`.
Baseline: `temps/html-conformance/engine-baseline.json`, byte-identical
across two consecutive runs (`cmp` exit 0). The plan ledger records the same
counts (GATE-02, wave D update); re-run the command for the current numbers.

| Category | Total | Passed | Failed | Skipped | Unsupported |
|----------|-------|--------|--------|---------|-------------|
| `tokenizer` | 7036 | 6686 | 0 | 0 | 350 |
| `tokenizer-local` | 6 | 6 | 0 | 0 | 0 |
| `tree-construction` | 1792 | 1225 | 331 | 8 | 228 |
| `tree-construction-local` | 12 | 12 | 0 | 0 | 0 |

Tokenizer: zero failures. The 350 unsupported cases are non-Data initial
states (342 runs: CDATA 56, PLAINTEXT 52, RAWTEXT 71, RCDATA 74, script data
89), lone-surrogate inputs (4), and XML-violation coercions (4).

Tree construction: the plan ledger's HTML-02a row recorded 841 cases passing
when the insertion modes landed; the table modes (HTML-05a) and the adoption
agency (HTML-FORMAT-01) then measure 1225 of 1792. `adoption01.dat`,
`adoption02.dat`, `tricky01.dat`, and `tables01.dat` are fully passing. The
remaining non-passes break down as:

- **script-data handling** (`tests16.dat`, 40; 36 are `<!--<script` escaped
  sequences and the rest are `noscript`/`noembed` edges; related records in
  `scriptdata01.dat` 9, `plain-text-unsafe.dat` 11, and `noscript01.dat` 8):
  the tokenizer closes at the first `</script` even inside a `<!--<script`
  sequence; the standard's script-data escaped and double-escaped states are
  not implemented.
- **template** (`template.dat`, 74 failed plus 36 unsupported): template
  contents are not modeled; `template` is routed as head content
  (`tree.go:335-343`) but has no separate content tree.
- **ruby** (`tests19.dat` 53, `ruby.dat` 16): `rp`/`rt`/`rb`/`rtc` implied
  end tags are incomplete.
- **CDATA in foreign content** (`tests21.dat`, 21; also part of
  `domjs-unsafe.dat` 14): `<![CDATA[...]]>` becomes a bogus comment
  everywhere (`html.go:318-328`); in SVG/MathML the standard wants a text
  node.
- **select** (24 failed inputs contain `<select`, mostly `webkit02.dat`):
  the select insertion modes are not implemented
  (`tree.go:1474-1497`, HTML-CONTEXT-01).
- **frameset** (`tests6.dat`, 14): `frameset` is not in the vocabulary, so
  frameset documents do not build the expected trees.
- **tables/foster parenting**: closed. `tables01.dat` is fully passing, the
  12 local table cases pass, and plan row HTML-05a is closed. One MathML
  `annotation-xml` integration-point case (`tests20.dat#59`) is the only
  remaining tests20 failure.
- **fragments**: all 192 `#document-fragment` records are unsupported
  because `Parse` has no context element (`conformance_tree_test.go:49-50`);
  plan row HTML-FRAGMENT-01 is open.

The 8 skipped cases are records with only a scripting-on expected document,
and the 228 unsupported cases are the 192 fragments plus 36
template-contents-only records. The full per-case list is in
`temps/html-conformance/engine-baseline.json`.

Traceability: tokenizer rows HTML-03a, HTML-03b, HTML-04a, and HTML-INPUT-01
are closed; tree rows HTML-02a, HTML-MODE-01, HTML-FOREIGN-01, HTML-05a, and
HTML-FORMAT-01 are closed; HTML-CONTEXT-01 (select/template) and
HTML-FRAGMENT-01 remain open in
`plans/0.0.1/html-css-json-compatibility-checklist.md`.

### 9.4 Cross-package validation

The parser is also exercised indirectly by consumers that parse HTML
fixtures: `internal/css/namespace_match_test.go:10`
(`TestImplicitBodyChildSelector` proves implicit `body > p`),
`css/mode_test.go:13` (`TestApplyCarriesDocumentMode` proves the doctype
mode reaches `css.Apply`), and the layout tests that call `html.Parse` and
check drawing-list output (for example `layout/displaylist_test.go`, run by
`make golden` via `TestDisplay`).

## 10. Known limitations, deferred items & open questions

- **Partial tree construction.** The parser implements the tokenizer states,
  the insertion-mode skeleton, and the adoption agency, but not the full
  standard: select and template states, fragment parsing, script-data
  escaped states, and parts of ruby and frameset recovery are missing. The
  measured gaps and counts are in §9.3. This is tracked work
  (HTML-CONTEXT-01, HTML-FRAGMENT-01), not a hidden regression.
- **Quirks mode is exposed but not applied.** `root.Mode` carries
  no-quirks / limited-quirks / quirks (`doctype.go:143-179`) to
  `pubstate.Styled.Mode`; quirks-mode layout effects are not implemented
  (HTML-MODE-01).
- **Tokenizer initial states.** Only the Data state is implemented;
  non-Data initial states (342 corpus runs) are counted unsupported rather
  than approximated.
- **XML-violation coercions and lone surrogates.** Four XML-violation cases
  and four lone-surrogate inputs are counted unsupported: Go strings cannot
  carry lone surrogates, and the infoset-coercion variant is not
  implemented.
- **CDATA in foreign content.** `<![CDATA[...]]>` is a bogus comment
  everywhere (`html.go:318-328`); in HTML content that matches the standard,
  in SVG/MathML it does not (21 measured failures).
- **No fragment parsing API.** `Parse` and `ParseDocument` take no context
  element, so 192 `#document-fragment` corpus records are unsupported.
- **Entity scope.** The decoder covers the standard's named table through
  the standard library plus two post-snapshot names; unknown named
  references pass through as literal text rather than being flagged.
- **No DOM APIs.** No `getElementById`-style lookup, no tree mutation
  beyond parse-time construction; `Walk`/`WalkUntil`/`FindFirst`/
  `TextContentOf` are the traversal surface (the public `html.Document.Find`
  wraps `FindFirst`, `html/parse.go:53-70`).
- **Unknown tags** are accepted structurally and render via UA defaults; the
  compatibility matrix is the normative contract for what is styled
  (`documentation/compatibility-matrix.md`).
- **Migration question (tracked).** The ponytail note at `html.go:8`:
  adopt `x/net/html` only if `layout`/`css` are rewritten; not a free
  delete.

## 11. Related documents

- Pipeline overview: [`../architecture.md`](../architecture.md)
- Fidelity tiers & degrade rules: [`../fidelity.md`](../fidelity.md)
- Security model & local-file ACL: [`../THREAT-MODEL.md`](../THREAT-MODEL.md)
- Support matrix (per-element / per-property): [`../compatibility-matrix.md`](../compatibility-matrix.md)
- Deferred / not-planned items: [`../deferred.md`](../deferred.md)
- Fonts & shaping (feeds text layout that consumes this tree): [`../fonts.md`](../fonts.md)
- Library API (how callers supply HTML bytes): [`../library-api.md`](../library-api.md)
- Live parser plan ledger: [`../../plans/0.0.1/html-css-json-compatibility-checklist.md`](../../plans/0.0.1/html-css-json-compatibility-checklist.md)

Sibling architecture deep-dives (same directory):

- [01-entrypoints-cli.md](01-entrypoints-cli.md) - `cmd/*` entrypoints
- [02-library-api.md](02-library-api.md) - public API
- [03-settings.md](03-settings.md) - `internal/settings` dotted config
- [04-load.md](04-load.md) - `internal/load` (the seam that feeds this parser)
- [06-css.md](06-css.md) - selector matching *against this tree*
- [07-layout.md](07-layout.md) - style resolution + box building over this tree
- [08-convert-pipeline.md](08-convert-pipeline.md) - prepare and the image pipeline
- [10-imageout-svg.md](10-imageout-svg.md) - raster output and SVG images
