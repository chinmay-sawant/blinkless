package layout

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// chapterPageName is the named page the inheritance tests reuse.
const chapterPageName = "chapter"

func TestPageNameInherits(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div class="outer"><p class="inner">a</p><p class="own">b</p></div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.outer { page: chapter }
		.own { page: cover }
	`)}, "print", testViewport, 800)

	inner := styleByClass(t, styles, "inner")
	if inner.PageName != chapterPageName {
		t.Fatalf("inner PageName = %q, want chapter (used-value inherit)", inner.PageName)
	}

	own := styleByClass(t, styles, "own")
	if own.PageName != "cover" {
		t.Fatalf("own PageName = %q, want cover", own.PageName)
	}

	outer := styleByClass(t, styles, "outer")
	if outer.PageName != chapterPageName {
		t.Fatalf("outer PageName = %q, want chapter", outer.PageName)
	}
}

