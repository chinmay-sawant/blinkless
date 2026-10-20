package layout

import (
	"cmp"
	"slices"
	"strings"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// ruleSel names one selector of one rule. The selector index places it in the
// bucket of its rightmost compound, so a node never sees the same selector
// twice and never walks a rule that cannot match. sheet keeps the candidate
// list sortable back into the source order the linear walk used, which keeps
// equal-specificity, equal-order ties (orders restart per sheet) unchanged.
type ruleSel struct {
	sheet int
	rule  *css.Rule
	sel   int
}

// ruleIndex narrows the rules one style pass must test. host holds selectors
// that end on the element itself; pseudo holds selectors that end on
// ::before or ::after, keyed by that name. content records that a pseudo rule
// declares content, which lets the generated-content walk skip pseudo
// matching when no rule can generate content at all.
type ruleIndex struct {
	host    ruleBucket
	pseudo  map[string]*ruleBucket
	content bool
}

// ruleBucket groups selectors by the rightmost compound's strongest key: an
// id, else the first class, else the tag, else the universal bucket. Grouping
// by one key is enough: every key a selector lists must be present for it to
// match, so the bucket of any one of them finds it.
type ruleBucket struct {
	byID      map[string][]ruleSel
	byClass   map[string][]ruleSel
	byTag     map[string][]ruleSel
	universal []ruleSel
}

// newRuleIndex walks every sheet once and buckets each selector.
func newRuleIndex(sheets []*css.Stylesheet) *ruleIndex {
	index := &ruleIndex{host: ruleBucket{}, pseudo: nil, content: false}
	for si, sheet := range sheets {
		if sheet == nil {
			continue
		}

		for r := range sheet.Rules {
			rule := &sheet.Rules[r]
			for s := range rule.Selectors {
				sel := rule.Selectors[s]
				if len(sel.Parts) == 0 {
					continue
				}

				last := sel.Parts[len(sel.Parts)-1]
				bucket := index.bucket(last.PseudoElement)
				bucket.add(ruleSel{sheet: si, rule: rule, sel: s}, last)
				if last.PseudoElement != "" && declaresContent(rule) {
					index.content = true
				}
			}
		}
	}

	return index
}

// bucket returns the selector bucket for one pseudo-element name, creating
// the map on first use.
func (idx *ruleIndex) bucket(pseudo string) *ruleBucket {
	if pseudo == "" {
		return &idx.host
	}

	if idx.pseudo == nil {
		idx.pseudo = map[string]*ruleBucket{}
	}

	b := idx.pseudo[pseudo]
	if b == nil {
		b = &ruleBucket{}
		idx.pseudo[pseudo] = b
	}

	return b
}

// add places one selector in the bucket its rightmost compound names.
func (b *ruleBucket) add(entry ruleSel, part css.SelectorPart) {
	switch {
	case part.ID != "":
		if b.byID == nil {
			b.byID = map[string][]ruleSel{}
		}

		b.byID[part.ID] = append(b.byID[part.ID], entry)
	case len(part.Classes) > 0:
		if b.byClass == nil {
			b.byClass = map[string][]ruleSel{}
		}

		name := part.Classes[0]
		b.byClass[name] = append(b.byClass[name], entry)
	case part.Tag != "" && part.Tag != "*":
		if b.byTag == nil {
			b.byTag = map[string][]ruleSel{}
		}

		name := strings.ToLower(part.Tag)
		b.byTag[name] = append(b.byTag[name], entry)
	default:
		b.universal = append(b.universal, entry)
	}
}

// declaresContent reports whether one rule can generate pseudo content.
func declaresContent(rule *css.Rule) bool {
	for _, d := range rule.Decls {
		if strings.EqualFold(d.Prop, "content") {
			return true
		}
	}

	return false
}

// candidates appends the selectors that can match node with pseudo. The
// result may hold selectors whose other parts do not match; the caller still
// runs the full matcher. Order follows the id, class, tag, universal buckets,
// not source order; cascade priority comes from specificity and rule order
// fields, never from slice position.
func (idx *ruleIndex) candidates(dst []ruleSel, node *html.Node, pseudo string) []ruleSel {
	if idx == nil || node == nil {
		return dst
	}

	b := &idx.host
	if pseudo != "" {
		b = idx.pseudo[pseudo]
		if b == nil {
			return dst
		}
	}

	if id := node.Attribute("id"); id != "" {
		dst = append(dst, b.byID[id]...)
	}

	forEachClass(node.Attribute("class"), func(name string) {
		dst = append(dst, b.byClass[name]...)
	})

	if len(b.byTag) > 0 {
		dst = append(dst, b.byTag[strings.ToLower(node.Name)]...)
	}

	return append(dst, b.universal...)
}

// sortCandidates restores the sheet, rule, and selector order the linear walk
// used. Rule orders restart in every sheet, so the sheet ordinal comes first;
// without it a cross-sheet equal-order tie could flip.
func sortCandidates(cands []ruleSel) {
	slices.SortFunc(cands, func(a, b ruleSel) int {
		if a.sheet != b.sheet {
			return cmp.Compare(a.sheet, b.sheet)
		}

		if a.rule.Order != b.rule.Order {
			return cmp.Compare(a.rule.Order, b.rule.Order)
		}

		return cmp.Compare(a.sel, b.sel)
	})
}

// forEachClass calls fn for every class token in an HTML class attribute.
func forEachClass(value string, fn func(string)) {
	for value != "" {
		value = strings.TrimLeft(value, asciiSpace)

		if value == "" {
			return
		}

		end := strings.IndexAny(value, asciiSpace)
		if end < 0 {
			fn(value)

			return
		}

		fn(value[:end])

		value = value[end:]
	}
}

const asciiSpace = " \t\r\n\f"
