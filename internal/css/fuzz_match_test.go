package css_test

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

const fuzzMatchDoc = `<div id="root" class="a b" data-x="1"><ul><li class="a">one</li>` +
	`<li class="b" lang="en-US">two</li><li>three<span>s</span></li></ul>` +
	`<p>text</p><input type="checkbox" checked><a href="#x">link</a></div>`

// FuzzMatchSelector parses an arbitrary selector list and matches every
// accepted selector against every node of a fixed tree. It must not panic or
// hang, and Match must agree with itself on a second call.
func FuzzMatchSelector(f *testing.F) {
	seeds := []string{
		"div", ".a", "#root", "ul > li", "li + li", "li ~ li", "div li",
		"li:nth-child(2n+1)", "li:first-child", "li:last-of-type", "li:not(.a)",
		"div:has(> p)", "div:has(.a, .b) li", "[data-x]", "[data-x=\"1\" i]",
		"[lang|=en]", "input:checked", "a:hover", ":is(li, p) > span", ":where(.a)",
		"*", "li:nth-last-child(odd)", "ul > li:only-child", ":root",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	doc, err := html.Parse(fuzzMatchDoc)
	if err != nil {
		f.Fatalf("parse fixture: %v", err)
	}

	var nodes []*html.Node

	doc.Walk(func(n *html.Node) {
		if n.Type == html.ElementNode {
			nodes = append(nodes, n)
		}
	})

	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 4<<10 {
			return
		}

		sels, ok := css.ParseSelectors(raw)
		if !ok {
			return
		}

		for _, sel := range sels {
			for _, node := range nodes {
				first := css.Match(sel, node)
				if second := css.Match(sel, node); first != second {
					t.Fatalf("Match(%q, <%s>) not deterministic", raw, node.Name)
				}
			}
		}
	})
}
