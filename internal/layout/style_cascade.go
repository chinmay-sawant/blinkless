package layout

import (
	"maps"
	"strconv"
	"strings"
	"sync"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

const (
	borderProperty       = "border"
	borderTopProperty    = "border-top"
	borderBottomProperty = "border-bottom"
	borderLeftProperty   = "border-left"
	borderRightProperty  = "border-right"

	marginProperty  = "margin"
	paddingProperty = "padding"

	textEmphasisProperty         = "text-emphasis"
	textEmphasisStyleProperty    = "text-emphasis-style"
	textEmphasisColorProperty    = "text-emphasis-color"
	textEmphasisPositionProperty = "text-emphasis-position"
	textEmphasisSkipProperty     = "text-emphasis-skip"
	textShadowProperty           = "text-shadow"
	tabSizeProperty              = "tab-size"

	inlineStylePriority = 1 << 30

	defaultRootFontPx = 16

	fontWeightNormalValue = 400
	fontWeightBoldValue   = 700

	cssFontStyleItalic  = "italic"
	cssFontStyleOblique = "oblique"
	cssFontWeightBold   = "bold"

	boxShorthandTwoSides   = 2
	boxShorthandThreeSides = 3
)

// internalCustomPropWriters reports whether raw declares a property whose
// applier stores engine bookkeeping in CustomProps. An element that declares
// no --* property inherits the parent's map; when one of these appliers runs,
// mergeCustomProps must hand it a copy or the parent's stored style would be
// mutated after insertion (styleStore interning shares records).
func internalCustomPropWriters(raw map[string]string) bool {
	for prop := range raw {
		switch prop {
		case textEmphasisProperty, textEmphasisStyleProperty, textEmphasisColorProperty,
			textEmphasisPositionProperty, textEmphasisSkipProperty,
			textShadowProperty, tabSizeProperty,
			rubyAlignPropName, rubyMergePropName, rubyOverhangPropName, rubyPositionPropName,
			footnotePropDisplay, footnotePropPolicy,
			stringSetPropName,
			bookmarkLabelProp, bookmarkLevelProp, bookmarkStateProp:
			return true
		}
	}

	return false
}

// mergeCustomProps inherits parent custom properties and overlays any --*
// declarations from raw, resolving var() chains via css.ResolveCustomProps.
// registered holds @property registrations: a property not declared here
// takes its initial value unless it inherits and the parent supplied one,
// and a non-inherited registered property never inherits the parent's value.
//
// The parent map is shared until an actual write: no declared properties and
// no effective registration returns the parent map itself, and a var-free
// overlay clones once instead of building the resolver work and memo maps.
func mergeCustomProps(parentProps, raw map[string]string, registered map[string]css.PropertyRule) map[string]string {
	declared := declaredCustomProps(raw)

	if len(declared) == 0 {
		return mergeRegisteredOnly(parentProps, raw, registered)
	}

	if len(registered) == 0 && !customValuesNeedResolve(parentProps) && !customValuesNeedResolve(declared) {
		return overlayCustomProps(parentProps, declared)
	}

	resolved := css.ResolveCustomProps(declared, parentProps)
	if len(registered) > 0 {
		applyRegisteredProps(resolved, registered, declared)
	}

	return resolved
}

// mergeRegisteredOnly merges parent custom properties with @property
// registrations when the element declares no custom properties of its own.
// The parent map is shared until an actual write.
func mergeRegisteredOnly(
	parentProps, raw map[string]string, registered map[string]css.PropertyRule,
) map[string]string {
	if len(registered) == 0 {
		if len(parentProps) > 0 && internalCustomPropWriters(raw) {
			return maps.Clone(parentProps)
		}

		return parentProps
	}

	if registeredPropsNoop(parentProps, registered) {
		if len(parentProps) > 0 && internalCustomPropWriters(raw) {
			return maps.Clone(parentProps)
		}

		return parentProps
	}

	out := make(map[string]string, len(parentProps)+len(registered))
	for name, value := range parentProps {
		out[name] = value
	}

	applyRegisteredProps(out, registered, nil)

	return out
}

// overlayCustomProps copies parentProps and overlays the declared custom
// properties.
func overlayCustomProps(parentProps, declared map[string]string) map[string]string {
	out := make(map[string]string, len(parentProps)+len(declared))
	for name, value := range parentProps {
		out[name] = value
	}

	for name, value := range declared {
		out[name] = value
	}

	return out
}

// registeredPropsNoop reports whether folding registered @property rules over
// parentProps changes nothing, so the caller can share the parent map instead
// of cloning it. declared is empty on this path: a registration changes the
// map when it writes an initial value the parent lacks (or that differs) or
// drops an inherited value the parent holds.
func registeredPropsNoop(parentProps map[string]string, registered map[string]css.PropertyRule) bool {
	for name, reg := range registered {
		if reg.Inherits {
			if value, ok := parentProps[name]; ok && strings.TrimSpace(value) != "" {
				continue
			}
		}

		if reg.Initial != "" {
			if value, ok := parentProps[name]; !ok || value != reg.Initial {
				return false
			}

			continue
		}

		if _, ok := parentProps[name]; ok {
			return false
		}
	}

	return true
}

// customValuesNeedResolve reports whether any custom-property value carries a
// var() reference, in which case css.ResolveCustomProps must run its work and
// memo maps with per-key cycle stacks. Values without var() merge by plain
// overlay, so the caller skips those maps with one clone.
func customValuesNeedResolve(values map[string]string) bool {
	for _, value := range values {
		if containsVarFunc(value) {
			return true
		}
	}

	return false
}

// declaredCustomProps collects the --* declarations from raw. It returns nil
// when raw declares none.
func declaredCustomProps(raw map[string]string) map[string]string {
	var declared map[string]string

	for prop, value := range raw {
		if strings.HasPrefix(prop, "--") {
			if declared == nil {
				declared = make(map[string]string)
			}

			declared[prop] = value
		}
	}

	return declared
}

// applyRegisteredProps folds @property registrations into a custom-property
// map. declared lists the properties declared on this element: a declared
// value wins. A registered property that is not declared takes its initial
// value unless it inherits and an inherited value is present; a
// non-inherited registration drops the inherited value first.
func applyRegisteredProps(
	props map[string]string, registered map[string]css.PropertyRule, declared map[string]string,
) {
	for name, reg := range registered {
		if _, ok := declared[name]; ok {
			continue
		}

		if reg.Inherits {
			if value, ok := props[name]; ok && strings.TrimSpace(value) != "" {
				continue
			}
		}

		if reg.Initial != "" {
			props[name] = reg.Initial
		} else {
			delete(props, name)
		}
	}
}

// registeredProperties merges @property registrations from every sheet. A
// later registration of the same name replaces an earlier one.
func registeredProperties(sheets []*css.Stylesheet) map[string]css.PropertyRule {
	var out map[string]css.PropertyRule

	for _, sheet := range sheets {
		if sheet == nil {
			continue
		}

		for _, reg := range sheet.Properties {
			if out == nil {
				out = make(map[string]css.PropertyRule)
			}

			out[reg.Name] = reg
		}
	}

	return out
}

// resolveRawVars expands var() in cascaded property values using customProps.
// Custom property keys (--*) are left unchanged (already resolved in the map).
// When no value contains var(), raw is returned as-is (no map copy).
//
// raw is cascade scratch owned by this pass (cascadeRaw/cascadePseudoRaw
// output or a caller-local map): values carrying var() expand in place
// instead of cloning the whole map when one value references a variable.
//
//nolint:cyclop // hot path; flat scan then expand of var() refs stays readable
func resolveRawVars(raw map[string]string, customProps map[string]string) map[string]string {
	if len(raw) == 0 {
		return raw
	}

	needs := false

	for prop, val := range raw {
		if strings.HasPrefix(prop, "--") {
			continue
		}

		if containsVarFunc(val) {
			needs = true

			break
		}
	}

	if !needs {
		return raw
	}

	lookup := func(name string) (string, bool) {
		if customProps == nil {
			return "", false
		}

		v, ok := customProps[name]

		return v, ok && strings.TrimSpace(v) != ""
	}

	for prop, val := range raw {
		if strings.HasPrefix(prop, "--") {
			continue
		}

		if containsVarFunc(val) {
			raw[prop] = css.ResolveVars(val, lookup)
		}
	}

	return raw
}

// containsVarFunc reports whether s has a CSS var( function (ASCII, case-insensitive)
// without allocating a lowercased copy of s.
func containsVarFunc(s string) bool {
	for i := 0; i+4 <= len(s); i++ {
		if (s[i]|asciiFoldBit) == 'v' && (s[i+1]|asciiFoldBit) == 'a' && (s[i+2]|asciiFoldBit) == 'r' && s[i+3] == '(' {
			return true
		}
	}

	return false
}

// blockifyDisplayForFloat maps specified display to the used value when
// float is left|right (CSS2.1 §9.7). table stays table (floated table
// wrapper); table-cell/row/… and inlines become block.
func blockifyDisplayForFloat(decl string) string {
	switch decl {
	case "inline", "inline-block", "inline-table", "inline-flex", "inline-grid",
		"run-in", "table-row-group", "table-header-group", "table-footer-group",
		"table-row", "table-cell", "table-caption", "table-column",
		"table-column-group", "list-item":
		return displayBlock
	default:
		return decl
	}
}

// selectorMatches reports whether sel matches node, using the pseudo-shape
// matcher when pe is non-empty and the cascade's runtime state.
func (ctx *styleContext) selectorMatches(sel css.Selector, node *html.Node, pe string) bool {
	if pe != "" {
		return ctx.state.MatchesPseudo(sel, node, pe)
	}

	return ctx.state.Matches(sel, node)
}

// containerGateMatches checks the rule's @container query against the nearest
// eligible size container (skipped on passes without container sizes).
func (ctx *styleContext) containerGateMatches(node *html.Node, runic *css.Rule) bool {
	if runic.Container == nil {
		return true
	}

	if ctx.containers == nil {
		return false // pass 1 / pseudo pass without sizes: skip
	}

	info, ok := findSizeContainer(node, runic.Container.Name, ctx.containers)

	return ok && runic.Container.Cond.Matches(info.inlineSize, info.fontSize)
}

// cascadeWinHint is the initial capacity for cascade winner maps. Most elements
// win a small handful of properties (UA + a few author rules).
const cascadeWinHint = 8

// cascadeWin is the winning cascaded declaration for one property: value plus
// the specificity/order bits needed to compare later candidates. important is
// a separate cascade layer (any !important beats any normal); layer is the
// @layer rank (0 = unlayered, which beats every ranked layer).
type cascadeWin struct {
	value               string
	ids, classes, types int
	order               int
	layer               int
	important           bool
}

// maxInlineDeclEntries bounds the interned inline-style cache. Entries are
// pure functions of the attribute text, so eviction only costs a re-parse.
const maxInlineDeclEntries = 1024

// maxSupportsMemoEntries bounds the @supports verdict cache. Verdicts are
// pure functions of (prop, value), so eviction only costs a re-probe.
const maxSupportsMemoEntries = 4096

//nolint:gochecknoglobals // pass-shared caches with test counters
var (
	inlineDeclsMu    sync.RWMutex
	inlineDecls      = make(map[string][]css.Declaration)
	inlineDeclParses uint64

	supportsMemoMu     sync.RWMutex
	supportsMemos      = make(map[string]bool)
	supportsProbeEvals uint64
	supportsProbeHits  uint64
)

// cachedInlineDecls returns the parsed declarations for one inline style
// attribute, parsing once per unique attribute text. The returned slice is
// shared: callers must range over it without mutating it.
func cachedInlineDecls(style string) []css.Declaration {
	inlineDeclsMu.RLock()
	decls, ok := inlineDecls[style]
	inlineDeclsMu.RUnlock()

	if ok {
		return decls
	}

	parsed := css.ParseInline(style)

	inlineDeclsMu.Lock()
	if _, ok := inlineDecls[style]; !ok {
		inlineDeclParses++

		if len(inlineDecls) < maxInlineDeclEntries {
			inlineDecls[style] = parsed
		}
	}
	inlineDeclsMu.Unlock()

	return parsed
}

// resetCascadeCachesForTest clears the inline-declaration intern cache, the
// @supports verdict cache, and their counters. Tests using it must stay
// sequential (no t.Parallel): parallel cascade tests share these caches.
func resetCascadeCachesForTest() {
	inlineDeclsMu.Lock()
	inlineDecls = make(map[string][]css.Declaration)
	inlineDeclParses = 0
	inlineDeclsMu.Unlock()

	supportsMemoMu.Lock()
	supportsMemos = make(map[string]bool)
	supportsProbeEvals = 0
	supportsProbeHits = 0
	supportsMemoMu.Unlock()
}

// inlineDeclParseCountForTest reports actual css.ParseInline calls.
func inlineDeclParseCountForTest() uint64 {
	inlineDeclsMu.RLock()
	defer inlineDeclsMu.RUnlock()

	return inlineDeclParses
}

// supportsProbeStatsForTest reports full @supports evaluations (cache
// misses) and hits.
func supportsProbeStatsForTest() (uint64, uint64) {
	supportsMemoMu.RLock()
	defer supportsMemoMu.RUnlock()

	return supportsProbeEvals, supportsProbeHits
}

// cascadeRaw returns the winning declaration per property for the element
// across UA sheet, the already matched author rules and the inline style
// attribute. Uses one winner map (value+spec+order+important) instead of six
// maps. hits is the matched-rule list, passed in so the caller can reuse it as
// the style memo key.
//
//nolint:cyclop // hot path; three fixed cascade tiers read clearer than one loop
func cascadeRaw( //nolint:funlen // cascade tiers are deliberately visible in one hot-path function
	ctx *styleContext, node *html.Node, hits []ruleHit,
) map[string]string {
	var wins map[string]cascadeWin
	if ctx == nil {
		wins = make(map[string]cascadeWin, cascadeWinHint)
	} else {
		wins = ctx.cascadeWins
		if wins == nil {
			wins = make(map[string]cascadeWin, cascadeWinHint)
			ctx.cascadeWins = wins
		} else {
			clear(wins)
		}
	}

	// UA sheet (lowest priority; specificity 0, order -1)
	for _, d := range uaRules(node.Name) {
		applyCascadeWin(wins, d.Prop, d.Value, 0, 0, 0, -1, 0, false)
	}

	// author sheets in source order (shared matchedRules walk)
	for _, hit := range hits {
		for _, d := range hit.rule.Decls {
			if !supportedDeclaration(d.Prop, d.Value) {
				continue
			}

			applyCascadeDeclaration(wins, d.Prop, d.Value, hit.a, hit.b, hit.c, hit.rule.Order, hit.rule.Layer, d.Important)
		}
	}

	// inline style attribute: outranks all normal declarations and all sheet
	// important declarations (spec 1<<maxIntShift). Elements without a style
	// attribute skip the inline parse entirely; repeated identical attributes
	// share one interned parse (see cachedInlineDecls).
	if styleAttr := strings.TrimSpace(node.Attribute("style")); styleAttr != "" {
		for _, d := range cachedInlineDecls(styleAttr) {
			if !supportedDeclaration(d.Prop, d.Value) {
				continue
			}

			applyCascadeDeclaration(wins, d.Prop, d.Value, inlineStylePriority, 0, 0, inlineStylePriority, 0, d.Important)
		}
	}

	if len(wins) == 0 {
		if ctx != nil && ctx.cascadeProps != nil {
			clear(ctx.cascadeProps)
		}

		return nil
	}

	var out map[string]string
	if ctx == nil {
		out = make(map[string]string, len(wins))
	} else {
		out = ctx.cascadeProps
		if out == nil {
			out = make(map[string]string, len(wins))
			ctx.cascadeProps = out
		} else {
			clear(out)
		}
	}

	for prop, w := range wins {
		out[prop] = w.value
	}

	return out
}

// cascadePseudoRaw returns the winning declarations for a generated
// ::before/::after box. Pseudo-elements have no inline attribute or UA rule of
// their own; their declarations come from the author rules that matched the
// host and pseudo shape. The winner buffer is the shared ctx scratch, but the
// output map stays freshly allocated: callers can hold two pseudo results
// from one context at once, so reusing the output buffer would alias them.
func cascadePseudoRaw(ctx *styleContext, node *html.Node, pseudoElem string) map[string]string {
	if ctx == nil || node == nil {
		return nil
	}

	wins := ctx.cascadeWins
	if wins == nil {
		wins = make(map[string]cascadeWin, cascadeWinHint)
		ctx.cascadeWins = wins
	} else {
		clear(wins)
	}

	for _, hit := range ctx.matchedRules(node, pseudoElem) {
		for _, d := range hit.rule.Decls {
			if !supportedDeclaration(d.Prop, d.Value) {
				continue
			}

			applyCascadeDeclaration(wins, d.Prop, d.Value, hit.a, hit.b, hit.c, hit.rule.Order, hit.rule.Layer, d.Important)
		}
	}

	if len(wins) == 0 {
		return nil
	}

	out := make(map[string]string, len(wins))
	for prop, win := range wins {
		out[prop] = win.value
	}

	return out
}

// applyCascadeDeclaration expands box shorthands before selecting winners.
// A shorthand and a longhand compete per physical property: the declaration
// that wins by specificity and source order must win that side, regardless of
// which form it used. Keeping the shorthand intact until after the cascade
// made an earlier margin-top declaration override a later margin shorthand.
func applyCascadeDeclaration(
	wins map[string]cascadeWin,
	prop, value string,
	ids, classes, types, order, layer int,
	important bool,
) {
	prop, value = remapVendorDeclaration(prop, value)

	if prop == maskShorthandProperty && containsVarFunc(value) {
		for _, longhand := range [...]string{
			"mask-image", "mask-mode", "mask-position", "mask-size",
			"mask-repeat", "mask-origin", "mask-clip", "mask-composite",
		} {
			applyCascadeWin(
				wins, longhand, maskShorthandDeferredPrefix+value,
				ids, classes, types, order, layer, important,
			)
		}

		return
	}

	if cascadeMaskBorderShorthand(wins, prop, value, ids, classes, types, order, layer, important) {
		return
	}

	if expanded, ok := expandShorthandDeclaration(prop, value); ok {
		for _, item := range expanded {
			applyCascadeDeclaration(wins, item.prop, item.val, ids, classes, types, order, layer, important)
		}

		// background-tbd also keeps its own key so Box.Declared reports it.
		if prop == declPropBackgroundTbd {
			applyCascadeWin(wins, prop, value, ids, classes, types, order, layer, important)
		}

		return
	}

	values, ok := expandBoxShorthand(prop, value)
	if !ok {
		applyCascadeWin(wins, prop, value, ids, classes, types, order, layer, important)

		return
	}

	// The longhand names are static per shorthand; concatenating them here
	// used to allocate a fresh property string for every side of every
	// shorthand declaration (6.4 MB per 500-page conversion in the profile).
	for idx, longhand := range boxShorthandLonghands(prop) {
		applyCascadeWin(wins, longhand, values[idx], ids, classes, types, order, layer, important)
	}
}

// expandMaskShorthandDeclaration expands mask layers and resets the mask-border
// family to its initial values, as required by the mask shorthand.
func expandMaskShorthandDeclaration(prop, value string) ([]logicalPropDecl, bool) {
	if prop != maskShorthandProperty {
		return nil, false
	}

	if cssWideKeyword(strings.TrimSpace(value)) {
		return maskShorthandWideDeclarations(strings.TrimSpace(value)), true
	}

	if containsVarFunc(value) {
		return nil, false
	}

	layers, ok := parseMaskShorthand(value)
	if !ok {
		return nil, false
	}

	var image, mode, position, size, repeat, origin, clip, composite strings.Builder

	for idx, layer := range layers {
		if idx > 0 {
			image.WriteString(", ")
			mode.WriteString(", ")
			position.WriteString(", ")
			size.WriteString(", ")
			repeat.WriteString(", ")
			origin.WriteString(", ")
			clip.WriteString(", ")
			composite.WriteString(", ")
		}

		image.WriteString(layer.image)
		mode.WriteString(layer.mode)
		position.WriteString(layer.position)
		size.WriteString(layer.size)
		repeat.WriteString(layer.repeat)
		origin.WriteString(layer.origin)
		clip.WriteString(layer.clip)
		composite.WriteString(layer.composite)
	}

	return []logicalPropDecl{
		{prop: "mask-image", val: image.String()},
		{prop: "mask-mode", val: mode.String()},
		{prop: "mask-position", val: position.String()},
		{prop: "mask-size", val: size.String()},
		{prop: "mask-repeat", val: repeat.String()},
		{prop: "mask-origin", val: origin.String()},
		{prop: "mask-clip", val: clip.String()},
		{prop: "mask-composite", val: composite.String()},
		{prop: "mask-border", val: "initial"},
		{prop: "mask-border-source", val: "none"},
		{prop: "mask-border-mode", val: "alpha"},
		{prop: "mask-border-slice", val: "0"},
		{prop: "mask-border-width", val: "auto"},
		{prop: "mask-border-outset", val: "0"},
		{prop: "mask-border-repeat", val: "stretch"},
	}, true
}

func maskShorthandWideDeclarations(value string) []logicalPropDecl {
	return []logicalPropDecl{
		{prop: "mask-image", val: value},
		{prop: "mask-mode", val: value},
		{prop: "mask-position", val: value},
		{prop: "mask-size", val: value},
		{prop: "mask-repeat", val: value},
		{prop: "mask-origin", val: value},
		{prop: "mask-clip", val: value},
		{prop: "mask-composite", val: value},
		{prop: "mask-border", val: value},
		{prop: "mask-border-source", val: value},
		{prop: "mask-border-mode", val: value},
		{prop: "mask-border-slice", val: value},
		{prop: "mask-border-width", val: value},
		{prop: "mask-border-outset", val: value},
		{prop: "mask-border-repeat", val: value},
	}
}

// boxShorthandLonghands returns the four physical longhand names for a box
// shorthand, in top/right/bottom/left order, or the zero array for any other
// property. Callers reach it only after expandBoxShorthand reported success
// for margin, padding, or border. Literal returns keep the names static
// without package globals.
func boxShorthandLonghands(prop string) [4]string {
	switch prop {
	case marginProperty:
		return [4]string{"margin-top", "margin-right", "margin-bottom", "margin-left"}
	case paddingProperty:
		return [4]string{"padding-top", "padding-right", "padding-bottom", "padding-left"}
	case borderProperty:
		return [4]string{"border-top", "border-right", "border-bottom", "border-left"}
	default:
		return [4]string{}
	}
}

type logicalPropDecl struct {
	prop string
	val  string
}

// expandListStyleDeclaration expands the list-style shorthand into its
// type, position, and image longhands so each part joins the cascade with
// the shorthand's own origin and specificity. Without this, the shorthand
// and a weaker-origin longhand (such as the UA disc on ul) coexist as
// separate raw keys and the later-applied longhand wins regardless of
// origin. Only components present in the value expand; the rest keep
// whatever the cascade decided.
func expandListStyleDeclaration(prop, value string) ([]logicalPropDecl, bool) {
	if prop != "list-style" {
		return nil, false
	}

	var out []logicalPropDecl

	for _, tok := range strings.Fields(value) {
		if typ := parseListStyleType(tok); typ != "" {
			out = append(out, logicalPropDecl{prop: "list-style-type", val: typ})

			break
		}
	}

	for _, tok := range strings.Fields(value) {
		if pos := parseListStylePosition(tok); pos != "" {
			out = append(out, logicalPropDecl{prop: "list-style-position", val: pos})

			break
		}
	}

	if url, ok := firstCSSUrl(value); ok {
		out = append(out, logicalPropDecl{prop: "list-style-image", val: `url("` + url + `")`})
	} else if listStyleImageNone(value) {
		out = append(out, logicalPropDecl{prop: "list-style-image", val: "none"})
	}

	if len(out) == 0 {
		return nil, false
	}

	return out, true
}

// expandFontDeclaration expands the font shorthand into its size,
// line-height, style, weight, and family longhands so each component joins
// the cascade with the shorthand's own origin, specificity, and source
// order. The shorthand used to be applied after every longhand, so an
// earlier `font` beat a later `font-size` regardless of order. Values the
// expansion cannot read (var() references, system fonts) keep the raw key
// and fall back to parseFontShorthand after resolveRawVars.
func expandFontDeclaration(prop, value string) ([]logicalPropDecl, bool) {
	if prop != "font" {
		return nil, false
	}

	parts := strings.Fields(value)
	out := make([]logicalPropDecl, 0, len(parts))

	for idx := range parts {
		tok := parts[idx]

		// font-size with an attached line-height, e.g. 12pt/1.4.
		if strings.Contains(tok, "/") {
			return expandFontSizeToken(out, parts, idx), true
		}

		lower := strings.ToLower(tok)

		if decl, handled := fontPrefixDecl(lower); handled {
			// Style, variant, weight, or stretch keyword positions. normal is
			// the initial value; variant and stretch have no readers.
			if decl.prop != "" {
				out = append(out, decl)
			}

			continue
		}

		if isFontWeightNumber(tok) {
			out = append(out, logicalPropDecl{prop: "font-weight", val: tok})

			continue
		}

		// First token that is not a prefix component starts the required size.
		out = append(out, logicalPropDecl{prop: "font-size", val: tok})

		return appendFontFamilyTail(out, parts, idx), true
	}

	// A valid font shorthand requires a size. Missing or unreadable values
	// stay intact for the post-cascade fallback.
	return nil, false
}

// expandFontSizeToken expands one size token that carries a line-height, e.g.
// 12pt/1.4, plus the family tokens that follow it, onto out.
func expandFontSizeToken(out []logicalPropDecl, parts []string, idx int) []logicalPropDecl {
	size, line, _ := strings.Cut(parts[idx], "/")
	out = append(out, logicalPropDecl{prop: "font-size", val: size})

	if line != "" {
		out = append(out, logicalPropDecl{prop: "line-height", val: line})
	}

	return appendFontFamilyTail(out, parts, idx)
}

// fontPrefixDecl returns the longhand a font shorthand prefix keyword sets.
// handled reports whether lower is a prefix keyword at all; keywords whose
// declaration is empty (normal, variant, and stretch positions) are handled
// but contribute no longhand.
func fontPrefixDecl(lower string) (logicalPropDecl, bool) {
	switch lower {
	case cssFontStyleItalic, cssFontStyleOblique:
		return logicalPropDecl{prop: "font-style", val: lower}, true
	case cssFontWeightBold, "bolder", "lighter":
		return logicalPropDecl{prop: "font-weight", val: lower}, true
	case contentNormal, fontVariantCapsSmall, "condensed", "expanded",
		"semi-condensed", "semi-expanded", "ultra-condensed", "ultra-expanded":
		return logicalPropDecl{}, true //nolint:exhaustruct // intentional empty keyword position
	}

	return logicalPropDecl{}, false //nolint:exhaustruct // intentional empty keyword position
}

// isFontWeightNumber reports whether tok is a numeric font weight 100..900.
func isFontWeightNumber(tok string) bool {
	n, ok := css.ParseNumber(tok)

	return ok && n >= 100 && n <= 900
}

// appendFontFamilyTail appends the tokens after idx as font-family, if any.
func appendFontFamilyTail(out []logicalPropDecl, parts []string, idx int) []logicalPropDecl {
	if idx+1 < len(parts) {
		out = append(out, logicalPropDecl{prop: "font-family", val: strings.Join(parts[idx+1:], " ")})
	}

	return out
}

// expandLogicalBoxDeclaration expands logical margin/padding/inset/border
// declarations to their physical counterpart properties for horizontal-tb.
func expandLogicalBoxDeclaration(prop, value string) ([]logicalPropDecl, bool) {
	if decls, ok := expandLogicalMarginPadding(prop, value); ok {
		return decls, true
	}

	if decls, ok := expandLogicalInset(prop, value); ok {
		return decls, true
	}

	return expandLogicalBorder(prop, value)
}

//nolint:cyclop,funlen // logical margin/padding mapping table
func expandLogicalMarginPadding(prop, value string) ([]logicalPropDecl, bool) {
	if prop == cssPropMarginInline || prop == cssPropMarginBlock ||
		prop == "margin-inline-start" || prop == "margin-inline-end" ||
		prop == "margin-block-start" || prop == "margin-block-end" ||
		prop == cssPropPaddingInline || prop == cssPropPaddingBlock ||
		prop == "padding-inline-start" || prop == "padding-inline-end" ||
		prop == "padding-block-start" || prop == "padding-block-end" {
		// Logical sides depend on the inherited direction and writing mode.
		// Keep them for the resolved-style pass instead of freezing them to
		// physical sides before that context is available.
		return nil, false
	}

	switch prop {
	case cssPropMarginBlock:
		start, end, ok := logicalPair(value)
		if !ok {
			return nil, false
		}

		return []logicalPropDecl{{"margin-top", start}, {"margin-bottom", end}}, true
	case cssPropMarginInline:
		start, end, ok := logicalPair(value)
		if !ok {
			return nil, false
		}

		return []logicalPropDecl{{"margin-left", start}, {"margin-right", end}}, true
	case "margin-block-start":
		return []logicalPropDecl{{"margin-top", value}}, true
	case "margin-block-end":
		return []logicalPropDecl{{"margin-bottom", value}}, true
	case "margin-inline-start":
		return []logicalPropDecl{{"margin-left", value}}, true
	case "margin-inline-end":
		return []logicalPropDecl{{"margin-right", value}}, true
	case cssPropPaddingBlock:
		start, end, ok := logicalPair(value)
		if !ok {
			return nil, false
		}

		return []logicalPropDecl{{"padding-top", start}, {"padding-bottom", end}}, true
	case cssPropPaddingInline:
		start, end, ok := logicalPair(value)
		if !ok {
			return nil, false
		}

		return []logicalPropDecl{{"padding-left", start}, {"padding-right", end}}, true
	case "padding-block-start":
		return []logicalPropDecl{{"padding-top", value}}, true
	case "padding-block-end":
		return []logicalPropDecl{{"padding-bottom", value}}, true
	case "padding-inline-start":
		return []logicalPropDecl{{"padding-left", value}}, true
	case "padding-inline-end":
		return []logicalPropDecl{{"padding-right", value}}, true
	default:
		return nil, false
	}
}

func expandLogicalInset(prop, value string) ([]logicalPropDecl, bool) {
	switch prop {
	case cssPropInsetBlock:
		start, end, ok := logicalPair(value)
		if !ok {
			return nil, false
		}

		return []logicalPropDecl{{"top", start}, {"bottom", end}}, true
	case cssPropInsetInline:
		start, end, ok := logicalPair(value)
		if !ok {
			return nil, false
		}

		return []logicalPropDecl{{"left", start}, {"right", end}}, true
	case cssPropInsetBlockStart:
		return []logicalPropDecl{{"top", value}}, true
	case cssPropInsetBlockEnd:
		return []logicalPropDecl{{"bottom", value}}, true
	case cssPropInsetInlineStart:
		return []logicalPropDecl{{"left", value}}, true
	case cssPropInsetInlineEnd:
		return []logicalPropDecl{{"right", value}}, true
	default:
		return nil, false
	}
}

//nolint:cyclop,funlen // logical border mapping table
func expandLogicalBorder(prop, value string) ([]logicalPropDecl, bool) {
	switch prop {
	case cssPropBorderBlock:
		return []logicalPropDecl{{"border-top", value}, {"border-bottom", value}}, true
	case cssPropBorderInline:
		return []logicalPropDecl{{"border-left", value}, {"border-right", value}}, true
	case cssPropBorderBlockStart:
		return []logicalPropDecl{{"border-top", value}}, true
	case cssPropBorderBlockEnd:
		return []logicalPropDecl{{"border-bottom", value}}, true
	case cssPropBorderInlineStart:
		return []logicalPropDecl{{"border-left", value}}, true
	case cssPropBorderInlineEnd:
		return []logicalPropDecl{{"border-right", value}}, true
	case cssPropBorderBlockColor:
		return []logicalPropDecl{{propBorderTopColor, value}, {propBorderBottomColor, value}}, true
	case cssPropBorderInlineColor:
		return []logicalPropDecl{{propBorderLeftColor, value}, {propBorderRightColor, value}}, true
	case cssPropBorderBlockStartColor:
		return []logicalPropDecl{{propBorderTopColor, value}}, true
	case cssPropBorderBlockEndColor:
		return []logicalPropDecl{{propBorderBottomColor, value}}, true
	case cssPropBorderInlineStartColor:
		return []logicalPropDecl{{propBorderLeftColor, value}}, true
	case cssPropBorderInlineEndColor:
		return []logicalPropDecl{{propBorderRightColor, value}}, true
	case cssPropBorderBlockStyle:
		return []logicalPropDecl{{propBorderTopStyle, value}, {propBorderBottomStyle, value}}, true
	case cssPropBorderInlineStyle:
		return []logicalPropDecl{{propBorderLeftStyle, value}, {propBorderRightStyle, value}}, true
	case cssPropBorderBlockStartStyle:
		return []logicalPropDecl{{propBorderTopStyle, value}}, true
	case cssPropBorderBlockEndStyle:
		return []logicalPropDecl{{propBorderBottomStyle, value}}, true
	case cssPropBorderInlineStartStyle:
		return []logicalPropDecl{{propBorderLeftStyle, value}}, true
	case cssPropBorderInlineEndStyle:
		return []logicalPropDecl{{propBorderRightStyle, value}}, true
	case cssPropBorderBlockWidth:
		return []logicalPropDecl{{propBorderTopWidth, value}, {propBorderBottomWidth, value}}, true
	case cssPropBorderInlineWidth:
		return []logicalPropDecl{{propBorderLeftWidth, value}, {propBorderRightWidth, value}}, true
	case cssPropBorderBlockStartWidth:
		return []logicalPropDecl{{propBorderTopWidth, value}}, true
	case cssPropBorderBlockEndWidth:
		return []logicalPropDecl{{propBorderBottomWidth, value}}, true
	case cssPropBorderInlineStartWidth:
		return []logicalPropDecl{{propBorderLeftWidth, value}}, true
	case cssPropBorderInlineEndWidth:
		return []logicalPropDecl{{propBorderRightWidth, value}}, true
	default:
		return nil, false
	}
}

func expandBoxShorthand(prop, value string) ([4]string, bool) {
	var values [4]string

	switch prop {
	case marginProperty, paddingProperty:
		// 1–4 space-separated sides (CSS box shorthand).
	case borderProperty:
		// border: <width> <style> <color> applies the same value to every side.
		// Expand so it competes with border-top/right/bottom/left longhands
		// (otherwise an earlier border-top can paint after a later border).
		values = [4]string{value, value, value, value}

		return values, true
	default:
		return values, false
	}

	var tokens [4]string
	count := splitSpaceTokens(value, tokens[:])

	if count < 1 || count > len(tokens) {
		return values, false
	}

	switch count {
	case 1:
		values = [4]string{tokens[0], tokens[0], tokens[0], tokens[0]}
	case boxShorthandTwoSides:
		values = [4]string{tokens[0], tokens[1], tokens[0], tokens[1]}
	case boxShorthandThreeSides:
		values = [4]string{tokens[0], tokens[1], tokens[2], tokens[1]}
	default:
		values = tokens
	}

	return values, true
}

// applyCascadeWin folds one declaration into the winner map when its layer,
// specificity, or source order beats the current winner.
func applyCascadeWin(
	wins map[string]cascadeWin,
	prop, value string, ids, classes, types, order, layer int, important bool,
) {
	// prop is already lowercase: sheet and inline declarations are folded by
	// css.parseDeclarations and the UA table is hard-coded lowercase.
	cur, ok := wins[prop]
	if !ok {
		wins[prop] = cascadeWin{
			value: value, ids: ids, classes: classes, types: types,
			order: order, layer: layer, important: important,
		}

		return
	}

	// !important is a higher cascade layer than normal (any origin here).
	if important != cur.important {
		if !important {
			return
		}

		wins[prop] = cascadeWin{
			value: value, ids: ids, classes: classes, types: types,
			order: order, layer: layer, important: true,
		}

		return
	}

	// Layer order: unlayered (0) beats every ranked layer; among ranked
	// layers, the later-declared (higher rank) beats the earlier.
	if layer != cur.layer {
		if !layerBeats(layer, cur.layer) {
			return
		}

		wins[prop] = cascadeWin{
			value: value, ids: ids, classes: classes, types: types,
			order: order, layer: layer, important: important,
		}

		return
	}

	if specificityBeats([4]int{cur.ids, cur.classes, cur.types, 0}, ids, classes, types, order, cur.order) {
		wins[prop] = cascadeWin{
			value: value, ids: ids, classes: classes, types: types,
			order: order, layer: layer, important: important,
		}
	}
}

// layerBeats reports whether the candidate @layer rank outranks the current
// one. Rank 0 is unlayered and beats every ranked layer; among ranked layers,
// later-declared (higher rank) beats earlier.
func layerBeats(candidate, current int) bool {
	if candidate == current {
		return false
	}

	if candidate == 0 {
		return true
	}

	if current == 0 {
		return false
	}

	return candidate > current
}

// specificityBeats reports whether (ids, classes, types) with the given source
// order wins over the current winning specificity/order.
func specificityBeats(cur [4]int, ids, classes, types, order, curOrder int) bool {
	if ids > cur[0] {
		return true
	}

	if ids < cur[0] {
		return false
	}

	if classes > cur[1] {
		return true
	}

	if classes < cur[1] {
		return false
	}

	if types > cur[2] {
		return true
	}

	if types < cur[2] {
		return false
	}

	return order >= curOrder
}

// applyFontProps resolves font-size/family/weight/style/font first, using the
// parent's size for percentages and em, and ctx.remBase for rem.
func applyFontProps(style *ResolvedStyle, raw map[string]string, parentSize float64, ctx *styleContext) {
	remBase := pxToPt(defaultRootFontPx)
	if ctx != nil && ctx.remBase > 0 {
		remBase = ctx.remBase
	}

	applyFontSizeValue(style, raw, parentSize, ctx, remBase)
	applyFontFamilyValue(style, raw)
	applyFontWeightValue(style, raw)
	applyFontStyleValue(style, raw)

	if val, found := raw["font"]; found {
		parseFontShorthand(style, val, remBase)
	}
}

func applyFontSizeValue(
	style *ResolvedStyle, raw map[string]string, parentSize float64, ctx *styleContext, remBase float64,
) {
	val, found := raw["font-size"]
	if !found {
		return
	}

	containing := 0.0
	if ctx != nil {
		containing = ctx.viewportW
	}

	if pt, parsed := clampLength(val, parentSize, containing); parsed {
		style.FontSize = pt

		return
	}

	style.FontSize = fontSize(val, parentSize, remBase)
}

func applyFontFamilyValue(style *ResolvedStyle, raw map[string]string) {
	val, found := raw["font-family"]
	if !found {
		return
	}

	if fam := css.ParseFontFamily(val); len(fam) > 0 {
		style.FontFamily = fam
	}
}

func applyFontWeightValue(style *ResolvedStyle, raw map[string]string) {
	val, found := raw["font-weight"]
	if found {
		style.FontWeight = resolveFontWeight(style.FontWeight, val)
	}
}

func applyFontStyleValue(style *ResolvedStyle, raw map[string]string) {
	val, found := raw["font-style"]
	if found {
		style.FontItalic = val == cssFontStyleItalic || val == cssFontStyleOblique
	}
}

// resolveFontWeight maps a font-weight keyword/number onto a weight value.
func resolveFontWeight(current int, val string) int {
	switch val {
	case contentNormal:
		return fontWeightNormalValue
	case cssFontWeightBold:
		return fontWeightBoldValue
	case "bolder":
		return current + fontWeightStep
	case "lighter":
		return current - fontWeightStep
	default:
		if n, ok := css.ParseNumber(val); ok && n >= 100 && n <= 900 {
			return int(n)
		}
	}

	return current
}

// restShorthandProps are applied before other cascaded properties so a winning
// longhand (e.g. margin-bottom) always overrides its shorthand (margin).
// Package-level to avoid per-node slice/array rebuilds.
var restShorthandProps = [...]string{ //nolint:gochecknoglobals // static apply order
	"display", "writing-mode", "direction", marginProperty, paddingProperty, borderProperty, borderTopProperty,
	borderRightProperty, borderBottomProperty, borderLeftProperty,
	borderWidthKeyword, borderStyleKeyword,
	borderColorKeyword, gapKeyword, flexKeyword, containerKeyword,
	cssPropMarginInline, cssPropMarginBlock, cssPropPaddingInline, cssPropPaddingBlock,
	insetKeyword, cssPropInsetBlock, cssPropInsetInline, "column-rule",
	cssPropBorderBlock, cssPropBorderInline, cssPropBorderBlockStart, cssPropBorderBlockEnd,
	cssPropBorderInlineStart, cssPropBorderInlineEnd, cssPropBorderBlockWidth, cssPropBorderBlockStyle,
	cssPropBorderBlockColor, cssPropBorderInlineWidth, cssPropBorderInlineStyle, cssPropBorderInlineColor,
	"max-size",
}

// restShorthandSet is derived from restShorthandProps so the longhand pass
// exclusion cannot drift from the ordered shorthand pass (display used to be
// missing from the hand-written switch and was applied twice).
var restShorthandSet = func() map[string]struct{} { //nolint:gochecknoglobals // derived from the apply table
	set := make(map[string]struct{}, len(restShorthandProps))

	for _, prop := range restShorthandProps {
		set[prop] = struct{}{}
	}

	return set
}()

// restLonghandStack is the stack scratch for the remaining longhands. Most
// elements declare far fewer than this many; larger declaration sets grow
// the slice on the heap the way any append would.
const restLonghandStack = 32

// applyRestProps resolves every non-font property once the font size is known.
// Shorthands run first in a fixed order. Remaining longhands still run in
// alphabetical order because longhands can overlap: overflow and overflow-x
// both write OverflowX, so the alphabetically later name must apply last for
// output to stay byte-identical. The order is reproduced with an insertion
// sort over a stack buffer, so the pass allocates no per-element key slice
// and does not call sort.Strings.
func applyRestProps(
	style *ResolvedStyle, raw map[string]string, ctx *styleContext,
	parent *ResolvedStyle,
) {
	if len(raw) == 0 {
		return
	}

	fsize := style.FontSize
	hasParent := parent != nil

	for _, prop := range restShorthandProps {
		value, ok := raw[prop]
		if !ok {
			continue
		}

		applyStyleProp(style, prop, value, fsize, ctx, parent, hasParent)
	}

	var keys [restLonghandStack]string

	rest := keys[:0]

	for key := range raw {
		if _, shorthand := restShorthandSet[key]; shorthand {
			continue
		}

		rest = append(rest, key)
	}

	rest = sortRestLonghandProps(rest)

	for _, prop := range rest {
		applyStyleProp(style, prop, raw[prop], fsize, ctx, parent, hasParent)
	}
}

// sortRestLonghandProps insertion-sorts property names into the byte order
// sort.Strings produced. A typical element has about ten remaining longhands,
// so an in-place insertion sort is cheaper than the allocation the old
// per-element slice plus stdlib sort needed.
func sortRestLonghandProps(props []string) []string {
	for i := 1; i < len(props); i++ {
		prop := props[i]
		prev := i - 1

		for prev >= 0 && props[prev] > prop {
			props[prev+1] = props[prev]
			prev--
		}

		props[prev+1] = prop
	}

	return props
}

// styleGroupFn is one property-group handler in the applyStyleProp dispatch.
// Groups return false when they do not own prop.
type styleGroupFn func(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext,
	parent *ResolvedStyle, hasParent bool,
) bool

// styleGroups is the dispatch order for applyStyleProp. It is package-level
// so applyStyleProp does not rebuild the table for every cascaded property.
var styleGroups = [...]styleGroupFn{ //nolint:gochecknoglobals // static dispatch table
	applyDisplayGroup,
	applyPositionGroup,
	applyAnchorProps,
	applyFlexGroup,
	applyMulticolGroup,
	applyRowRuleGroup,
	applyGridGroup,
	applyBoxGroup,
	applyMaxSizeProps,
	applyBorderGroup,
	applyColorGroup,
	applyTextGroup,
	applyTableBreakGroup,
	applyTransformGroup,
	// Re-added support groups (2026-09-12 demotions): stubs registered here,
	// follow-up agents fill the bodies.
	applyContainmentProps,
	applyColorAdjustProps,
	applyFontVariantProps,
	applyFontFeatureProps,
	applyFontSynthesisProps,
	applyFontWidthProps,
	applyFontSizeAdjustProps,
	applyImageAdjustProps,
	applyTextSupportProps,
	applyOverflowLogicalProps,
	applyAspectRatioProps,
	applyTextBoxProps,
	applyLineFitEdgeProps,
	applyTextAutospaceProps,
	applyLinePaddingProps,
	applyHyphenationProps,
	applyInitialLetterProps,
	applyShapeProps,
	applyShapeInsideProps,
	applyFloatPageProps,
	applyMaskProps,
	applyMaskBorderProps,
	applyClipPathProps,
	applyRubyProps,
	applyScrollProps,
	applyNamedFlowProps,
	applyFlowToleranceProps,
	applyExclusionProps,
	applyWrapBreakProps,
	applyAnimationProps,
	applyTransitionProps,
	applyTimelineProps,
	applyBaselineSourceProps,
	applyContinueProps,
	applyLinkParams,
	applyImageRenderingProps,
	applyTextStrokeProps,
	applyInlineSizingProps,
	applySizeShorthandProps,
	applyMinSizeProps,
	applyAllProps,
	applyWordSpaceTransformProps,
}

func normalizeVendorPrefix(prop string) string {
	if !strings.HasPrefix(prop, "-webkit-") {
		return prop
	}

	if normalized, ok := normalizeWebkitMaskProperty(prop); ok {
		return normalized
	}

	return normalizeOtherWebkitProperty(prop)
}

const (
	webkitBoxSizingTarget          = "box-sizing"
	webkitBorderRadiusTarget       = "border-radius"
	webkitBorderTopLeftTarget      = "border-top-left-radius"
	webkitBorderTopRightTarget     = "border-top-right-radius"
	webkitBorderBottomLeftTarget   = "border-bottom-left-radius"
	webkitBorderBottomRightTarget  = "border-bottom-right-radius"
	webkitTransformTarget          = "transform"
	webkitTransformOriginTarget    = "transform-origin"
	webkitBackfaceVisibilityTarget = "backface-visibility"
	webkitPerspectiveTarget        = "perspective"
	webkitPerspectiveOriginTarget  = "perspective-origin"
	webkitTransformStyleTarget     = "transform-style"
	webkitFlexBasisTarget          = "flex-basis"
	webkitFlexDirectionTarget      = "flex-direction"
	webkitFlexGrowTarget           = "flex-grow"
	webkitFlexShrinkTarget         = "flex-shrink"
	webkitFlexWrapTarget           = "flex-wrap"
	webkitJustifyContentTarget     = "justify-content"
	webkitAlignContentTarget       = "align-content"
	webkitAlignItemsTarget         = "align-items"
	webkitAlignSelfTarget          = "align-self"
	webkitOrderTarget              = "order"
	webkitFilterTarget             = "filter"
	webkitBoxAlignProperty         = "-webkit-box-align"
	webkitBoxOrdinalGroupProperty  = "-webkit-box-ordinal-group"
	webkitBoxOrientProperty        = "-webkit-box-orient"
	webkitBoxPackProperty          = "-webkit-box-pack"
)

func normalizeOtherWebkitProperty(prop string) string {
	if normalized, ok := normalizeWebkitBorderProperty(prop); ok {
		return normalized
	}

	if normalized, ok := normalizeWebkitTransformProperty(prop); ok {
		return normalized
	}

	if normalized, ok := normalizeWebkitFlexProperty(prop); ok {
		return normalized
	}

	if normalized, ok := normalizeWebkitFlexAlignmentProperty(prop); ok {
		return normalized
	}

	if normalized, ok := normalizeWebkitLegacyBoxProperty(prop); ok {
		return normalized
	}

	if normalized, ok := normalizeWebkitMiscProperty(prop); ok {
		return normalized
	}

	if normalized, ok := normalizeWebkitAnimationTransitionProperty(prop); ok {
		return normalized
	}

	return prop
}

func normalizeWebkitBorderProperty(prop string) (string, bool) {
	switch prop {
	case "-webkit-box-sizing":
		return webkitBoxSizingTarget, true
	case "-webkit-border-radius":
		return webkitBorderRadiusTarget, true
	case "-webkit-border-top-left-radius":
		return webkitBorderTopLeftTarget, true
	case "-webkit-border-top-right-radius":
		return webkitBorderTopRightTarget, true
	case "-webkit-border-bottom-left-radius":
		return webkitBorderBottomLeftTarget, true
	case "-webkit-border-bottom-right-radius":
		return webkitBorderBottomRightTarget, true
	default:
		return "", false
	}
}

func normalizeWebkitTransformProperty(prop string) (string, bool) {
	switch prop {
	case "-webkit-transform":
		return webkitTransformTarget, true
	case "-webkit-transform-origin":
		return webkitTransformOriginTarget, true
	case "-webkit-backface-visibility":
		return webkitBackfaceVisibilityTarget, true
	case "-webkit-perspective":
		return webkitPerspectiveTarget, true
	case "-webkit-perspective-origin":
		return webkitPerspectiveOriginTarget, true
	case "-webkit-transform-style":
		return webkitTransformStyleTarget, true
	default:
		return "", false
	}
}

func normalizeWebkitFlexProperty(prop string) (string, bool) {
	switch prop {
	case "-webkit-flex":
		return flexKeyword, true
	case "-webkit-flex-basis":
		return webkitFlexBasisTarget, true
	case "-webkit-flex-direction":
		return webkitFlexDirectionTarget, true
	case "-webkit-flex-flow":
		return propFlexFlow, true
	case "-webkit-flex-grow":
		return webkitFlexGrowTarget, true
	case "-webkit-flex-shrink":
		return webkitFlexShrinkTarget, true
	case "-webkit-flex-wrap":
		return webkitFlexWrapTarget, true
	default:
		return "", false
	}
}

func normalizeWebkitFlexAlignmentProperty(prop string) (string, bool) {
	switch prop {
	case "-webkit-justify-content":
		return webkitJustifyContentTarget, true
	case "-webkit-align-content":
		return webkitAlignContentTarget, true
	case "-webkit-align-items":
		return webkitAlignItemsTarget, true
	case "-webkit-align-self":
		return webkitAlignSelfTarget, true
	case "-webkit-order":
		return webkitOrderTarget, true
	default:
		return "", false
	}
}

func normalizeWebkitLegacyBoxProperty(prop string) (string, bool) {
	switch prop {
	case "-webkit-box-shadow":
		return boxShadowProp, true
	case webkitBoxAlignProperty:
		return webkitAlignItemsTarget, true
	case "-webkit-box-flex":
		return webkitFlexGrowTarget, true
	case webkitBoxOrdinalGroupProperty:
		return webkitOrderTarget, true
	case webkitBoxOrientProperty:
		return webkitFlexDirectionTarget, true
	case webkitBoxPackProperty:
		return webkitJustifyContentTarget, true
	default:
		return "", false
	}
}

func normalizeWebkitMiscProperty(prop string) (string, bool) {
	switch prop {
	case "-webkit-filter":
		return webkitFilterTarget, true
	case "-webkit-text-fill-color":
		return blendColor, true
	default:
		if normalized, ok := normalizeWebkitBackgroundClip(prop); ok {
			return normalized, true
		}

		if normalized, ok := normalizeWebkitBackgroundOrigin(prop); ok {
			return normalized, true
		}

		if normalized, ok := normalizeWebkitBackgroundSize(prop); ok {
			return normalized, true
		}

		return "", false
	}
}

func normalizeWebkitMaskProperty(prop string) (string, bool) {
	if target, ok := normalizeWebkitMaskBoxImageAlias(prop); ok {
		return target, true
	}

	return normalizeWebkitMaskSharedProperty(prop)
}

func normalizeWebkitMaskSharedProperty(prop string) (string, bool) {
	switch prop {
	case "-webkit-mask":
		return maskShorthandProperty, true
	case "-webkit-mask-clip":
		return maskClipProperty, true
	case webkitMaskCompositeProperty:
		return maskCompositeProperty, true
	case "-webkit-mask-image":
		return maskImageProperty, true
	case "-webkit-mask-origin":
		return maskOriginProperty, true
	case "-webkit-mask-position":
		return maskPositionProperty, true
	case "-webkit-mask-repeat":
		return maskRepeatProperty, true
	case "-webkit-mask-size":
		return maskSizeProperty, true
	default:
		return prop, false
	}
}

// applyStyleProp routes one cascaded property to the group that owns it.
//
//nolint:wsl // branching on effectiveProp
func applyStyleProp(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext,
	parent *ResolvedStyle, hasParent bool,
) {
	effectiveProp := normalizeVendorPrefix(prop)
	effectiveValue := value
	if effectiveProp != prop {
		effectiveValue = remapWebkitValue(prop, value)
	}

	for _, group := range styleGroups {
		if group(style, effectiveProp, effectiveValue, fsize, ctx, parent, hasParent) {
			return
		}
	}

	applyIgnoredGroup(style, prop, value)
}

// engineSupportsProperty reports whether the engine supports prop: value for
// @supports. The property must have an apply arm in the dispatch table AND the
// value must pass the same acceptance rule as the cascade gate
// (supportedDeclaration). Ownership alone is not support: setDisplayKeyword
// owns display but ignores display:bogus, so that declaration must not
// satisfy the query. Custom properties are supported by definition; their
// values resolve in mergeCustomProps.
//
// The verdict is a pure function of (prop, value), so it is memoized: a
// repeated probe skips both the acceptance parse and the scratch-style
// dispatch instead of rebuilding a full ResolvedStyle per probe.
func engineSupportsProperty(prop, value string) bool {
	if strings.HasPrefix(prop, "--") {
		return true
	}

	key := prop + "\x00" + value

	supportsMemoMu.RLock()
	verdict, ok := supportsMemos[key]
	supportsMemoMu.RUnlock()

	if ok {
		supportsMemoMu.Lock()
		supportsProbeHits++
		supportsMemoMu.Unlock()

		return verdict
	}

	verdict = engineSupportsPropertyUncached(prop, value)

	supportsMemoMu.Lock()
	if _, ok := supportsMemos[key]; !ok {
		if len(supportsMemos) < maxSupportsMemoEntries {
			supportsMemos[key] = verdict
		}

		supportsProbeEvals++
	}
	supportsMemoMu.Unlock()

	return verdict
}

// engineSupportsPropertyUncached runs the @supports probe without consulting
// the verdict memo.
func engineSupportsPropertyUncached(prop, value string) bool {
	if !supportedDeclaration(prop, value) {
		return false
	}

	effectiveProp := normalizeVendorPrefix(prop)

	effectiveValue := value
	if effectiveProp != prop {
		effectiveValue = remapWebkitValue(prop, value)
	}

	var scratch ResolvedStyle

	scratchCtx := styleContext{} //nolint:exhaustruct // zero values model the @supports probe context

	for _, group := range styleGroups {
		if group(&scratch, effectiveProp, effectiveValue, 0, &scratchCtx, nil, false) {
			return true
		}
	}

	return false
}

//nolint:cyclop,wsl,nlreturn,funlen // 2009 box value remaps
func remapWebkitValue(prop, value string) string {
	trimmed := strings.TrimSpace(value)
	low := strings.ToLower(trimmed)
	switch prop {
	case "-webkit-box-align":
		switch low {
		case "start":
			return flexStartKeyword
		case "end":
			return fxFlexEnd
		case "center":
			return fxCenter
		case "stretch":
			return fxStretch
		case baselineName:
			return baselineName
		default:
			return low
		}
	case "-webkit-box-pack":
		switch low {
		case "start":
			return flexStartKeyword
		case "end":
			return fxFlexEnd
		case "center":
			return fxCenter
		case "justify":
			return fxBetween
		default:
			return low
		}
	case "-webkit-box-orient":
		switch low {
		case "horizontal":
			return fxRow
		case "vertical":
			return fxCol
		case "inline-axis":
			return fxRow
		case "block-axis":
			return fxCol
		default:
			return low
		}
	case "-webkit-box-ordinal-group":
		if n, err := strconv.Atoi(trimmed); err == nil {
			if n > 0 {
				n--
			}
			return strconv.Itoa(n)
		}
		return trimmed
	case webkitMaskCompositeProperty:
		return remapWebkitMaskCompositeValue(trimmed)
	default:
		return value
	}
}

// applyDisplayGroup handles display, position-adjacent flow and stacking props.
