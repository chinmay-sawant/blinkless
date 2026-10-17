package layout

import (
	"math"
	"strings"
	"testing"
)

// fixture21Viewport returns the page content box gowkhtmltopdf lays a
// default A4 document out in: the A4 sheet minus the default 10 mm margins,
// in points. Break metadata is canvas Y, so the bounds only pick which page
// boundaries a default render hits.
func fixture21Viewport() (float64, float64) {
	const (
		a4W      = 595.28
		a4H      = 841.89
		marginPt = 10.0 * 72.0 / 25.4
	)

	return a4W - 2*marginPt, a4H - 2*marginPt
}

// TestFixture21SectionHeadingKeepsFollowingContent: in the detailed report,
// the section-5 h1 falls at the bottom of page 2 with its meta line, the 5.1
// h2, and the first requirement row before the page boundary (orphaned
// heading group). The break export must emit a keep range that starts at the
// section heading and covers enough of its following content to straddle the
// page-2/3 boundary while fitting one page, so a paginator honoring
// Display.KeepTogether moves the whole group to the top of page 3 and leaves
// page 2 ending with the section-4 invoice block. Reference: wkhtmltopdf
// starts section 5 at the top of page 3.
func TestFixture21SectionHeadingKeepsFollowingContent(t *testing.T) { //nolint:cyclop // fixture assertions
	t.Parallel()

	root, sheets := loadFixture(t, "fixture-21-detailed-report.html")

	contentW, contentH := fixture21Viewport()

	res, err := Layout(root, Options{
		Width: contentW, Height: contentH, Sheets: sheets, Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	sec5 := boxByHeadingText(res.root, "5. Platform requirements status")
	if sec5 == nil {
		t.Fatal("fixture-21 section-5 heading box not found")
	}

	_, keep := PageBreaks(res, 0)

	group, found := keepStartingAt(keep, sec5.y)
	if !found {
		t.Fatalf("no keep range starts at the section-5 heading top y=%.2f; keeps=%v", sec5.y, keep)
	}

	if span := group[1] - group[0]; span > contentH {
		t.Fatalf("section-5 keep span = %.2fpt, want one-page span <= %.2fpt", span, contentH)
	}

	startPage := int(group[0] / contentH)
	endPage := int((group[1] - 0.001) / contentH)

	if startPage != 1 || endPage != 2 {
		t.Fatalf("section-5 keep [%.2f .. %.2f] spans pages %d..%d, want the page-2/3 boundary",
			group[0], group[1], startPage, endPage)
	}

	// A paginator moves a straddling keep that fits one page to the next page
	// top. The whole group must clear page 2 and the document must stay at the
	// three pages the section-4 break already produces.
	delta := float64(startPage+1)*contentH - group[0]

	for idx := range res.Ops {
		paintOp := &res.Ops[idx]
		if paintOp.Y < group[0] || paintOp.Y > group[1] {
			continue
		}

		if shifted := paintOp.Y + delta; shifted < float64(endPage)*contentH-0.01 {
			t.Fatalf("op %d %q stays on page 2 after the group shift: y=%.2f -> %.2f",
				idx, paintOp.Text, paintOp.Y, shifted)
		}
	}

	if end := res.Height + delta; end > float64(endPage+1)*contentH {
		t.Fatalf("shifted document end = %.2fpt, want three pages (<= %.2fpt)",
			end, float64(endPage+1)*contentH)
	}
}

// boxByHeadingText returns the first heading box whose text contains want.
func boxByHeadingText(boxNode *box, want string) *box {
	if boxNode == nil {
		return nil
	}

	if isHeadingBox(boxNode) && boxNode.node != nil && strings.Contains(boxNode.node.TextContent(), want) {
		return boxNode
	}

	for _, child := range boxNode.children {
		if found := boxByHeadingText(child, want); found != nil {
			return found
		}
	}

	return nil
}

// keepStartingAt returns the keep range whose start matches y, if any.
func keepStartingAt(keep [][2]float64, y float64) ([2]float64, bool) {
	for _, candidate := range keep {
		if math.Abs(candidate[0]-y) <= 0.01 {
			return candidate, true
		}
	}

	return [2]float64{}, false
}

// TestPageBreaksHeadingKeepsFollowingBlock: the keep export keeps a heading
// with its following block, folds following headings and intro lead-ins into
// one group, skips op-less break markers, and stops at the first content
// block that closes the run.
func TestPageBreaksHeadingKeepsFollowingBlock(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body>`+
		`<h1 id="a">Section</h1>`+
		`<p id="lead">Lead line</p>`+
		`<h2 id="b">Subsection</h2>`+
		`<div class="marker" style="page-break-before:always"></div>`+
		`<p id="content">Following content</p>`+
		`<p id="tail">Tail</p>`+
		`</body></html>`)

	_, keep := PageBreaks(res, 0)

	h1 := boxByID(t, res, "a")
	if group, found := keepStartingAt(keep, h1.y); !found {
		t.Fatalf("h1 keep missing; keeps=%v", keep)
	} else if want := boxByID(t, res, "content"); math.Abs(group[1]-(want.y+want.height)) > 0.01 {
		t.Errorf("h1 keep end = %.2f, want first content block bottom %.2f",
			group[1], want.y+want.height)
	}

	h2 := boxByID(t, res, "b")
	if group, found := keepStartingAt(keep, h2.y); !found {
		t.Fatalf("h2 keep missing; keeps=%v", keep)
	} else if want := boxByID(t, res, "content"); math.Abs(group[1]-(want.y+want.height)) > 0.01 {
		t.Errorf("h2 keep end = %.2f, want first content block bottom %.2f",
			group[1], want.y+want.height)
	}
}
