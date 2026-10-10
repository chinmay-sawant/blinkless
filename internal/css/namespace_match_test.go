package css

import (
	"testing"
)

// TestImplicitBodyChildSelector proves that the insertion modes' implicit
// html/head/body wrappers make structural selectors such as body > p match a
// fragment that never wrote the wrappers.
func TestImplicitBodyChildSelector(t *testing.T) {
	t.Parallel()

	root := treeFor(t, `<p id="target">x</p>`)
	paragraph := byID(root, "target")

	if paragraph == nil {
		t.Fatal("target paragraph not found")
	}

	for _, selector := range []string{"body > p", "html > body > p", "html body p"} {
		sels, valid := ParseSelectors(selector)
		if !valid || len(sels) != 1 {
			t.Fatalf("ParseSelectors(%q) failed", selector)
		}

		if !Match(sels[0], paragraph) {
			t.Errorf("selector %q did not match the implicit body child", selector)
		}
	}

	sels, valid := ParseSelectors("head > p")
	if !valid {
		t.Fatal("ParseSelectors(head > p) failed")
	}

	if Match(sels[0], paragraph) {
		t.Error("head > p must not match a body paragraph")
	}
}

// TestForeignTypeSelectorCaseSensitivity proves HTML type selectors stay
// case-insensitive while foreign element names match exactly.
func TestForeignTypeSelectorCaseSensitivity(t *testing.T) {
	t.Parallel()

	root := treeFor(t, `<DIV id="d"></DIV><svg><linearGradient id="g"/></svg>`)
	div := byID(root, "d")
	gradient := byID(root, "g")

	if div == nil || gradient == nil {
		t.Fatal("probe elements not found")
	}

	htmlSel, valid := ParseSelectors("div")
	if !valid || !Match(htmlSel[0], div) {
		t.Error("HTML type selector must match case-insensitively")
	}

	exact, valid := ParseSelectors("linearGradient")
	if !valid || !Match(exact[0], gradient) {
		t.Error("foreign type selector must match the adjusted name exactly")
	}

	lower, valid := ParseSelectors("lineargradient")
	if !valid {
		t.Fatal("ParseSelectors(lineargradient) failed")
	}

	if Match(lower[0], gradient) {
		t.Error("foreign type selector must be case-sensitive")
	}
}
