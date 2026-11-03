package css

import (
	icss "github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// HoverChangedScopes returns the IDs of subtrees containing elements whose
// hover-dependent selector matches changed from previous to d's hover state.
// It includes inherited changes by keeping each matched element's subtree.
// Rules inside conditional groups are included conservatively.
//
// A changed element without an ID uses its nearest ID-bearing ancestor. False
// means no such scope exists, or d is nil; the caller must lay out the page.
// An empty result with true means no selector match changed.
func (d *Document) HoverChangedScopes(previous string) ([]string, bool) {
	if d == nil || d.root == nil {
		return nil, false
	}

	old := d.state
	old.Hover = previous
	selectors := hoverSelectors(d.sheets)
	scopes := make(map[*html.Node]bool)
	complete := true

	var walk func(*html.Node)

	walk = func(node *html.Node) {
		if !complete {
			return
		}

		if node.Type == html.ElementNode && hoverMatchChanged(node, selectors, old, d.state) {
			scope := hoverIDScope(node)
			if scope == nil {
				complete = false

				return
			}

			scopes[scope] = true
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(d.root)

	if !complete {
		return nil, false
	}

	return outerHoverScopes(d.root, scopes), true
}

func hoverIDScope(node *html.Node) *html.Node {
	for node != nil && node.Attribute("id") == "" {
		node = node.Parent
	}

	return node
}

func hoverSelectors(sheets []*icss.Stylesheet) []icss.Selector {
	var selectors []icss.Selector

	for _, sheet := range sheets {
		if sheet == nil {
			continue
		}

		for _, rule := range sheet.Rules {
			for _, selector := range rule.Selectors {
				if icss.SelectorUsesHover(selector) {
					selectors = append(selectors, selector)
				}
			}
		}
	}

	return selectors
}

func hoverMatchChanged(node *html.Node, selectors []icss.Selector, old, next icss.MatchState) bool {
	for _, selector := range selectors {
		pseudo := selector.Parts[len(selector.Parts)-1].PseudoElement
		if pseudo != "" {
			if old.MatchesPseudo(selector, node, pseudo) != next.MatchesPseudo(selector, node, pseudo) {
				return true
			}
		} else if old.Matches(selector, node) != next.Matches(selector, node) {
			return true
		}
	}

	return false
}

// Emit tree order and drop scopes already covered by an ancestor's patch.
func outerHoverScopes(root *html.Node, scopes map[*html.Node]bool) []string {
	var ids []string

	var walk func(*html.Node)

	walk = func(node *html.Node) {
		if scopes[node] {
			ids = append(ids, node.Attribute("id"))

			return
		}

		for _, child := range node.Children {
			walk(child)
		}
	}

	walk(root)

	return ids
}
