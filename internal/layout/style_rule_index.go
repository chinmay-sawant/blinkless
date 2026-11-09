package layout

import (
	"cmp"
	"slices"
	"strings"
	"unicode"

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
// matching when no rule can generate content at all. contentPseudo records
// the same per pseudo-element name so a ::before lookup skips when only
// ::after rules declare content (and vice versa).
type ruleIndex struct {
	host          ruleBucket
	pseudo        map[string]*ruleBucket
	content       bool
	contentPseudo map[string]bool
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

type keyCounts struct {
	byID      map[string]int
	byClass   map[string]int
	byTag     map[string]int
	universal int
}

// ruleCounts holds the selector counts newRuleIndex sizes its buckets from.
type ruleCounts struct {
	host          keyCounts
	pseudo        map[string]*keyCounts
	contentPseudo map[string]bool
	total         int
	content       bool
}

// newRuleIndex walks every sheet once and buckets each selector.
func newRuleIndex(sheets []*css.Stylesheet) *ruleIndex {
	counts := countRuleSelectors(sheets)
	if counts.total == 0 {
		return new(ruleIndex)
	}

	index := new(ruleIndex)
	index.content = counts.content
	index.contentPseudo = counts.contentPseudo

	allSels := make([]ruleSel, counts.total)
	offset := 0

	initRuleBucket(&index.host, &counts.host, allSels, &offset)

	if len(counts.pseudo) > 0 {
		index.pseudo = make(map[string]*ruleBucket, len(counts.pseudo))

		for pseudo, pseudoCounts := range counts.pseudo {
			bucket := new(ruleBucket)
			initRuleBucket(bucket, pseudoCounts, allSels, &offset)
			index.pseudo[pseudo] = bucket
		}
	}

	populateRuleBuckets(index, sheets)

	return index
}

// countRuleSelectors counts selectors per bucket and key, and records which
// pseudo rules declare content.
func countRuleSelectors(sheets []*css.Stylesheet) *ruleCounts {
	counts := new(ruleCounts)

	for _, sheet := range sheets {
		if sheet == nil {
			continue
		}

		for ruleIndex := range sheet.Rules {
			rule := &sheet.Rules[ruleIndex]

			for selectorIndex := range rule.Selectors {
				sel := rule.Selectors[selectorIndex]
				if len(sel.Parts) == 0 {
					continue
				}

				counts.total++

				countSelector(counts, rule, sel.Parts[len(sel.Parts)-1])
			}
		}
	}

	return counts
}

// pseudoBucket returns the key counts for one pseudo-element, creating the
// bucket on first use.
func (counts *ruleCounts) pseudoBucket(name string) *keyCounts {
	if counts.pseudo == nil {
		counts.pseudo = map[string]*keyCounts{}
	}

	bucket := counts.pseudo[name]
	if bucket == nil {
		bucket = new(keyCounts)
		counts.pseudo[name] = bucket
	}

	return bucket
}

// markContentPseudo records generated content on one pseudo-element.
func (counts *ruleCounts) markContentPseudo(name string) {
	counts.content = true

	if counts.contentPseudo == nil {
		counts.contentPseudo = map[string]bool{}
	}

	counts.contentPseudo[name] = true
}

// countSelector adds one selector to its bucket key counts and records
// generated content on pseudo rules.
func countSelector(counts *ruleCounts, rule *css.Rule, last css.SelectorPart) {
	bucket := &counts.host

	if last.PseudoElement != "" {
		bucket = counts.pseudoBucket(last.PseudoElement)

		if declaresContent(rule) {
			counts.markContentPseudo(last.PseudoElement)
		}
	}

	bucket.count(last)
}

// count adds one rightmost compound to the bucket key counts.
func (c *keyCounts) count(last css.SelectorPart) {
	switch {
	case last.ID != "":
		if c.byID == nil {
			c.byID = map[string]int{}
		}

		c.byID[last.ID]++
	case len(last.Classes) > 0:
		if c.byClass == nil {
			c.byClass = map[string]int{}
		}

		c.byClass[last.Classes[0]]++
	case last.Tag != "" && last.Tag != "*":
		if c.byTag == nil {
			c.byTag = map[string]int{}
		}

		c.byTag[strings.ToLower(last.Tag)]++
	default:
		c.universal++
	}
}

// initRuleBucket reserves one contiguous slice per key in allSels and points
// the bucket at them.
func initRuleBucket(bucket *ruleBucket, counts *keyCounts, allSels []ruleSel, offset *int) {
	if counts == nil {
		return
	}

	bucket.byID = reserveRuleSels(counts.byID, allSels, offset)
	bucket.byClass = reserveRuleSels(counts.byClass, allSels, offset)
	bucket.byTag = reserveRuleSels(counts.byTag, allSels, offset)

	if counts.universal > 0 {
		bucket.universal = allSels[*offset : *offset : *offset+counts.universal]
		*offset += counts.universal
	}
}

// reserveRuleSels hands each key a sub-slice with exactly its count reserved
// in the shared backing array.
func reserveRuleSels(counts map[string]int, allSels []ruleSel, offset *int) map[string][]ruleSel {
	if len(counts) == 0 {
		return nil
	}

	sels := make(map[string][]ruleSel, len(counts))

	for key, count := range counts {
		sels[key] = allSels[*offset : *offset : *offset+count]
		*offset += count
	}

	return sels
}

// populateRuleBuckets fills the preallocated selector slices in source order.
func populateRuleBuckets(index *ruleIndex, sheets []*css.Stylesheet) {
	for sheetIndex, sheet := range sheets {
		if sheet == nil {
			continue
		}

		for ruleIndex := range sheet.Rules {
			rule := &sheet.Rules[ruleIndex]

			for selectorIndex := range rule.Selectors {
				sel := rule.Selectors[selectorIndex]
				if len(sel.Parts) == 0 {
					continue
				}

				last := sel.Parts[len(sel.Parts)-1]
				bucket := index.bucket(last.PseudoElement)
				bucket.add(ruleSel{sheet: sheetIndex, rule: rule, sel: selectorIndex}, last)
			}
		}
	}
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

	bucket := idx.pseudo[pseudo]
	if bucket == nil {
		bucket = new(ruleBucket)
		idx.pseudo[pseudo] = bucket
	}

	return bucket
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

	bucket := &idx.host
	if pseudo != "" {
		bucket = idx.pseudo[pseudo]
		if bucket == nil {
			return dst
		}
	}

	if id := node.Attribute("id"); id != "" {
		dst = append(dst, bucket.byID[id]...)
	}

	forEachClass(node.Attribute("class"), func(name string) {
		dst = append(dst, bucket.byClass[name]...)
	})

	if len(bucket.byTag) > 0 {
		dst = append(dst, bucket.byTag[strings.ToLower(node.Name)]...)
	}

	return append(dst, bucket.universal...)
}

// mayMatchPseudo reports whether any indexed ::before/::after selector for
// pseudo could match node. It mirrors the bucket lookups of candidates
// without building, sorting, or matching the candidate list, so callers can
// skip the generated-content cascade when the walk would find no rule. A
// true result still needs the full matcher (gates and combinators may
// reject); a false result means matchedRules would return no hits.
func (idx *ruleIndex) mayMatchPseudo(node *html.Node, pseudo string) bool {
	if idx == nil || node == nil || pseudo == "" {
		return false
	}

	bucket := idx.pseudo[pseudo]
	if bucket == nil {
		return false
	}

	if id := node.Attribute("id"); id != "" && len(bucket.byID[id]) > 0 {
		return true
	}

	if pseudoBucketMatchesClass(bucket, node) {
		return true
	}

	if len(bucket.byTag) > 0 && len(bucket.byTag[strings.ToLower(node.Name)]) > 0 {
		return true
	}

	return len(bucket.universal) > 0
}

// pseudoBucketMatchesClass reports whether any class on node selects into
// the pseudo bucket. Split from mayMatchPseudo so the bucket probe stays a
// flat sequence of early returns.
func pseudoBucketMatchesClass(bucket *ruleBucket, node *html.Node) bool {
	matched := false

	forEachClass(node.Attribute("class"), func(name string) {
		if !matched && len(bucket.byClass[name]) > 0 {
			matched = true
		}
	})

	return matched
}

// sortCandidates restores the sheet, rule, and selector order the linear walk
// used. Rule orders restart in every sheet, so the sheet ordinal comes first;
// without it a cross-sheet equal-order tie could flip.
func sortCandidates(cands []ruleSel) {
	slices.SortFunc(cands, func(left, right ruleSel) int {
		if left.sheet != right.sheet {
			return cmp.Compare(left.sheet, right.sheet)
		}

		if left.rule.Order != right.rule.Order {
			return cmp.Compare(left.rule.Order, right.rule.Order)
		}

		return cmp.Compare(left.sel, right.sel)
	})
}

// forEachClass calls visit for every class token in an HTML class attribute.
// It splits on the same Unicode whitespace the selector matcher uses, so the
// index never skips a class the matcher would accept.
func forEachClass(value string, visit func(string)) {
	for value != "" {
		value = strings.TrimLeftFunc(value, unicode.IsSpace)

		if value == "" {
			return
		}

		end := strings.IndexFunc(value, unicode.IsSpace)
		if end < 0 {
			visit(value)

			return
		}

		visit(value[:end])

		value = value[end:]
	}
}
