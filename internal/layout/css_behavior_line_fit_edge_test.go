package layout

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBehaviorLineFitEdgeTextDropsPadding is line-fit-edge. leading keeps
// today's line box: half-leading plus the span's padding and border. text
// leaves that padding and border out, so the paragraph line box is shorter
// by the vertical padding.
//
// Measured from testdata/css/v012/line-fit-edge/page.html through layoutHTML.
// The 16px paragraphs have no margin. Each span has padding 8px. leading's
// line box is 25.5pt (34px). text's line box is 13.5pt (18px). The drop is
// 12pt (16px): inlineChromeTop and inlineChromeBottom each add 8px (6pt),
// and text adds neither.
func TestBehaviorLineFitEdgeTextDropsPadding(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "testdata", "css", "v012", "line-fit-edge", "page.html")

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	res := layoutHTML(t, string(raw))
	leading := boxByID(t, res, "leading")
	text := boxByID(t, res, "text")
	drop := leading.height - text.height

	if !near(leading.height, pxToPt(34)) {
		t.Errorf("leading line box = %.4fpt (%.2fpx), want 34px (25.5pt)",
			leading.height, leading.height/ptPerCSSPx)
	}

	if !near(text.height, pxToPt(18)) {
		t.Errorf("text line box = %.4fpt (%.2fpx), want 18px (13.5pt)",
			text.height, text.height/ptPerCSSPx)
	}

	if !near(drop, pxToPt(16)) {
		t.Errorf("line box drop = %.4fpt (%.2fpx), want 16px (12pt) of padding",
			drop, drop/ptPerCSSPx)
	}
}
