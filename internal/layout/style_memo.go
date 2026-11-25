package layout

import (
	"slices"
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// styleResolutionMemo caches resolved element styles for repeated declaration
// shapes. The 500-page report resolves 66,007 elements into 15 distinct
// ResolvedStyle records, so without the memo every element pays a full cascade
// plus the intern fingerprint/equality scan only to discover that the record
// already exists.
//
// The key captures every input the element resolution reads:
//
//   - parent style pointer: inheritance and custom-property resolution read
//     the parent record;
//   - matched rule pointers plus specificity: the exact matched-declaration
//     identity, so :nth-child, sibling combinators, :has, media gates and
//     container gates are reflected in which rules are present;
//   - node name: the UA sheet dispatches on the tag;
//   - inline style attribute: parsed only on a miss;
//   - rem base: rem lengths depend on the root font size;
//   - the --print-link-underline outcome for anchors: applied after the
//     cascade, so it cannot be derived from the declarations.
//
// A hit skips cascadeRaw, the whole raw-to-used application, and the intern
// fingerprint/equality scan, and returns the pointer already interned by
// styleStore. A miss stores a copy of the matched-rule sequence and compares
// it field by field on later lookups, so different declaration shapes that
// land in one map cell cannot share a record.
//
// The memo never crosses a resolution pass: it lives in styleContext, and
// container re-cascade passes (ctx.containers non-nil) do not use it because
// their gate outcomes are measured, not static.
type styleResolutionMemo struct {
	entries map[styleMemoKey][]*styleMemoEntry
	// disabled is the differential-test switch; container passes also skip
	// the memo without setting it.
	disabled bool
}

// styleMemoKey is the comparable part of the memo key. Elements that share it
// still need their matched-rule sequences compared before sharing a record.
type styleMemoKey struct {
	parent        *ResolvedStyle
	nodeName      string
	inline        string
	remBase       float64
	linkUnderline bool
}

// styleMemoEntry is one cached resolution: the exact matched-rule sequence
// that produced style, kept so equality is decided by the rules, never by the
// map hash alone.
type styleMemoEntry struct {
	hits  []ruleHit
	style *ResolvedStyle
}

// usable reports whether this pass may consult the memo for an element.
func (m *styleResolutionMemo) usable(parent *ResolvedStyle) bool {
	return !m.disabled && parent != nil
}

// lookup returns the cached style for key when the stored matched-rule
// sequence equals hits.
func (m *styleResolutionMemo) lookup(key styleMemoKey, hits []ruleHit) *ResolvedStyle {
	for _, entry := range m.entries[key] {
		if ruleHitsEqual(entry.hits, hits) {
			return entry.style
		}
	}

	return nil
}

// insert stores style under key and a private copy of hits.
func (m *styleResolutionMemo) insert(key styleMemoKey, hits []ruleHit, style *ResolvedStyle) {
	if m.entries == nil {
		m.entries = make(map[styleMemoKey][]*styleMemoEntry)
	}

	m.entries[key] = append(m.entries[key], &styleMemoEntry{
		hits:  slices.Clone(hits),
		style: style,
	})
}

// ruleHitsEqual reports whether two matched-rule sequences are identical.
// ruleHit holds a rule pointer and three specificity ints, so field
// comparison is the exact identity check the memo key needs.
func ruleHitsEqual(left, right []ruleHit) bool {
	if len(left) != len(right) {
		return false
	}

	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}

	return true
}

// resolveElementStyleMemo resolves one element through the memo when the pass
// allows it and returns the interned style pointer.
//
// The root element is excluded: it has no parent, and its resolution is the
// only writer of ctx.remBase, so caching it would skip that side effect.
func resolveElementStyleMemo(
	node *html.Node, ctx *styleContext, parent *ResolvedStyle, store *styleStore,
) *ResolvedStyle {
	if ctx == nil || ctx.containers != nil || node.Name == "html" || !ctx.memo.usable(parent) {
		resolveElementStyle(node, ctx, parent, &store.candidate)

		return store.append(store.candidate)
	}

	// The matched-rule list is needed for the key, so it is computed before
	// the cascade instead of inside it.
	hits := ctx.matchedRules(node, "")
	key := styleMemoKey{
		parent:        parent,
		nodeName:      node.Name,
		inline:        node.Attribute("style"),
		remBase:       ctx.remBase,
		linkUnderline: linkUnderlinePolicy(ctx, node),
	}

	if cached := ctx.memo.lookup(key, hits); cached != nil {
		return cached
	}

	if key.inline != "" {
		if style := tryInlinePxBoxStyle(node, ctx, parent, store, key, hits); style != nil {
			return style
		}
	}

	resolveElementStyleWithHits(node, ctx, parent, &store.candidate, hits)
	style := store.append(store.candidate)

	ctx.memo.insert(key, hits, style)

	return style
}

type inlinePxBox struct {
	left, top, right, bottom, width, height                   float64
	hasLeft, hasTop, hasRight, hasBottom, hasWidth, hasHeight bool
}

// tryInlinePxBoxStyle resolves elements whose inline style only sets plain px
// box geometry (left/top/right/bottom/width/height) by memoizing the base
// stylesheet cascade once and cloning it with the parsed px offsets.
func tryInlinePxBoxStyle(
	node *html.Node, ctx *styleContext, parent *ResolvedStyle, store *styleStore,
	key styleMemoKey, hits []ruleHit,
) *ResolvedStyle {
	if hitsHaveImportantBoxProp(hits) {
		return nil
	}

	box, ok := parseInlinePxBox(key.inline)
	if !ok {
		return nil
	}

	baseKey := key
	baseKey.inline = ""

	base := ctx.memo.lookup(baseKey, hits)
	if base == nil {
		resolveElementStyleWithHitsInline(node, ctx, parent, &store.candidate, hits, "")
		base = store.append(store.candidate)
		ctx.memo.insert(baseKey, hits, base)
	}

	if base.Triggers != nil || base.Declared != nil {
		return nil
	}

	store.candidate = *base
	if box.hasWidth {
		store.candidate.Width = box.width
		store.candidate.WidthPercent = -1
	}

	if box.hasHeight {
		store.candidate.Height = box.height
		store.candidate.HeightPercent = -1
	}

	if box.hasLeft || box.hasTop || box.hasRight || box.hasBottom {
		pos := *base.positioning()
		if box.hasLeft {
			pos.Left = box.left
			pos.LeftAuto = false
		}
		if box.hasTop {
			pos.Top = box.top
			pos.TopAuto = false
		}
		if box.hasRight {
			pos.Right = box.right
			pos.RightAuto = false
		}
		if box.hasBottom {
			pos.Bottom = box.bottom
			pos.BottomAuto = false
		}
		store.candidate.positioningGroup = &pos
	}

	style := store.append(store.candidate)
	ctx.memo.insert(key, hits, style)

	return style
}

func hitsHaveImportantBoxProp(hits []ruleHit) bool {
	for i := range hits {
		for j := range hits[i].rule.Decls {
			d := &hits[i].rule.Decls[j]
			if !d.Important {
				continue
			}
			switch d.Prop {
			case "left", "top", "right", "bottom", "width", "height", "inset",
				"inset-inline", "inset-inline-start", "inset-inline-end",
				"inset-block", "inset-block-start", "inset-block-end",
				"inline-size", "block-size":
				return true
			}
		}
	}

	return false
}

func parseInlinePxBox(s string) (inlinePxBox, bool) {
	const pxToPt = 0.75

	var out inlinePxBox

	seen := false
	i := 0
	n := len(s)

	for i < n {
		for i < n && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' || s[i] == ';') {
			i++
		}
		if i >= n {
			break
		}

		pStart := i
		for i < n && s[i] != ':' && s[i] != ';' {
			i++
		}
		if i >= n || s[i] != ':' {
			return inlinePxBox{}, false
		}
		prop := trimASCII(s[pStart:i])
		i++ // skip ':'

		vStart := i
		for i < n && s[i] != ';' {
			i++
		}
		val := trimASCII(s[vStart:i])
		if i < n {
			i++ // skip ';'
		}

		px, ok := parseSimplePx(val)
		if !ok {
			return inlinePxBox{}, false
		}

		pt := px * pxToPt
		switch prop {
		case "left":
			out.left, out.hasLeft = pt, true
		case "top":
			out.top, out.hasTop = pt, true
		case "right":
			out.right, out.hasRight = pt, true
		case "bottom":
			out.bottom, out.hasBottom = pt, true
		case "width":
			if pt < 0 {
				return inlinePxBox{}, false
			}
			out.width, out.hasWidth = pt, true
		case "height":
			if pt < 0 {
				return inlinePxBox{}, false
			}
			out.height, out.hasHeight = pt, true
		default:
			return inlinePxBox{}, false
		}
		seen = true
	}

	return out, seen
}

func trimASCII(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func parseSimplePx(s string) (float64, bool) {
	if s == "0" {
		return 0, true
	}
	if len(s) < 3 || s[len(s)-2] != 'p' || s[len(s)-1] != 'x' {
		return 0, false
	}
	num := s[:len(s)-2]
	neg := false
	idx := 0
	if num[0] == '-' {
		neg = true
		idx = 1
	} else if num[0] == '+' {
		idx = 1
	}
	if idx >= len(num) {
		return 0, false
	}

	var intPart float64
	var fracPart float64
	fracScale := 1.0
	inFrac := false
	digits := 0

	for ; idx < len(num); idx++ {
		c := num[idx]
		if c >= '0' && c <= '9' {
			digits++
			d := float64(c - '0')
			if inFrac {
				fracScale *= 10
				fracPart += d / fracScale
			} else {
				intPart = intPart*10 + d
			}
			continue
		}
		if c == '.' && !inFrac {
			inFrac = true
			continue
		}
		return 0, false
	}

	if digits == 0 {
		return 0, false
	}

	val := intPart + fracPart
	if neg {
		val = -val
	}
	return val, true
}

// linkUnderlinePolicy reports whether resolveElementStyle overrides
// text-decoration for this node under --print-link-underline. The policy
// runs after the cascade, so its outcome is part of the memo key.
func linkUnderlinePolicy(ctx *styleContext, node *html.Node) bool {
	return ctx.printLinkUnderline &&
		node.Name == "a" &&
		strings.TrimSpace(node.Attribute("href")) != ""
}
