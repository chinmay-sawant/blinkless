//nolint:wsl,cyclop,funlen,varnamelen // at-rule fixtures
package css_test

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
	"github.com/chinmay-sawant/blinkless/internal/layout"
)

// TestPositionTryAtRule proves the atrule row: @position-try stores a
// fallback position-area, position-try-fallbacks: --above uses that stored
// area, and an unknown ident is skipped. One stylesheet fixture carries the
// two @position-try blocks and the two targets.
//
// Pinned spec: css-anchor-position-1 @position-try rule
// (https://drafts.csswg.org/css-anchor-position-1/#fallback-rule).
func TestPositionTryAtRule(t *testing.T) {
	t.Parallel()

	sheet, err := css.Parse(`
		@position-try --above { position-area: top; }
		@position-try --below { position-area: bottom; top: 5px; }
		#anchor { anchor-name: --tip; width: 100px; height: 50px; }
		#tip { position: absolute; position-anchor: --tip; position-area: bottom;
			position-try-fallbacks: --above, --missing; width: 60px; height: 20px; }
		#tip-unknown { position: absolute; position-anchor: --tip; position-area: bottom;
			position-try-fallbacks: --missing; width: 60px; height: 20px; }
	`)
	if err != nil {
		t.Fatalf("css.Parse: %v", err)
	}

	if len(sheet.PositionTries) != 2 {
		t.Fatalf("PositionTries = %d, want 2", len(sheet.PositionTries))
	}

	above := sheet.PositionTry("--above")
	if above == nil || above.Area != "top" {
		t.Fatalf("@position-try --above = %+v, want area top", above)
	}

	below := sheet.PositionTry("--below")
	if below == nil || below.Area != "bottom" || below.Top != "5px" {
		t.Fatalf("@position-try --below = %+v, want area bottom top 5px", below)
	}

	if sheet.PositionTry("--missing") != nil {
		t.Fatal("@position-try --missing found, want nil (unknown ident)")
	}

	root, err := html.Parse(`<html><body>` +
		`<div id="anchor">a</div><div id="tip">t</div><div id="tip-unknown">u</div>` +
		`</body></html>`)
	if err != nil {
		t.Fatalf("html.Parse: %v", err)
	}

	styles, err := layout.ResolveStyles(t.Context(), root, layout.Options{
		Width: 500, Height: 800, Sheets: []*css.Stylesheet{sheet},
	})
	if err != nil {
		t.Fatalf("layout.ResolveStyles: %v", err)
	}

	tipStyle := styleForID(t, root, styles, "tip")
	if len(tipStyle.PositionTryFallbacks) != 2 ||
		tipStyle.PositionTryFallbacks[0] != "--above" ||
		tipStyle.PositionTryFallbacks[1] != "--missing" {
		t.Errorf("tip fallbacks = %q, want [--above --missing] (unknown kept for skip at use)",
			tipStyle.PositionTryFallbacks)
	}

	if tipStyle.PositionArea != "bottom" {
		t.Errorf("tip area = %q, want bottom (base kept, fallback wins at use)", tipStyle.PositionArea)
	}

	unknownStyle := styleForID(t, root, styles, "tip-unknown")
	if len(unknownStyle.PositionTryFallbacks) != 1 || unknownStyle.PositionTryFallbacks[0] != "--missing" {
		t.Errorf("tip-unknown fallbacks = %q, want [--missing]", unknownStyle.PositionTryFallbacks)
	}

	// The layout pass resolves --above to the stored top area and skips
	// --missing; covered end to end by the tactic DisplayList test in
	// internal/layout. This fixture proves the atrule storage it reads.
}

func styleForID(
	t *testing.T, root *html.Node, styles map[*html.Node]*layout.ResolvedStyle, id string,
) *layout.ResolvedStyle {
	t.Helper()

	var found *html.Node

	root.Walk(func(n *html.Node) {
		if found == nil && n.Attribute("id") == id {
			found = n
		}
	})

	if found == nil {
		t.Fatalf("no node with id %q", id)
	}

	sty := styles[found]
	if sty == nil {
		t.Fatalf("no style for id %q", id)
	}

	return sty
}
