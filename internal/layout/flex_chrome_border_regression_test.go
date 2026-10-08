package layout

import (
	"fmt"
	"math"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// flexStraddleContentH is the A4 landscape Chrome Flex content height
// (595.28pt page height minus two 12mm margins).
const flexStraddleContentH = 527.24

// flexStraddleFixture builds the section 4 frame shape behind an
// inline 423.6pt spacer: one avoid-inside bordered tile holding four 300x30
// justified flex rows, followed by a second bordered tile with a 200x200 SVG
// that constrains to a 100x100 replaced item. The spacer places the first
// tile top at y=423.6, so the second (center) row straddles the first page
// boundary at 527.24pt for the Chrome Flex pagination regression.
func flexStraddleFixture(t *testing.T) *Result {
	t.Helper()

	root := mustParse(t, `<html><body>`+
		`<div class="spacer"></div>`+
		`<section class="ex"><h2>4. justify</h2><p class="note">note four</p>`+
		`<div class="case"><div class="item">A</div><div class="item">B</div></div>`+
		`<div class="case center"><div class="item">A</div><div class="item">B</div></div>`+
		`<div class="case end"><div class="item">A</div><div class="item">B</div></div>`+
		`<div class="case between"><div class="item">A</div><div class="item">B</div></div>`+
		`</section>`+
		`<section class="ex"><h2>10. minimum width</h2><p class="note">note ten</p>`+
		`<div class="constrained-flex"><svg class="svg-item" width="200" height="200">`+
		`<rect width="200" height="200" fill="#0a0"/></svg></div>`+
		`</section>`+
		`</body></html>`)

	styleSheet := sheet(t, `
html, body { margin: 0; padding: 0; font: 11pt/1.4 sans-serif }
.spacer { height: 423.6pt }
.ex { page-break-inside: avoid; overflow: hidden; border: 1px solid #bbb; padding: 8pt; margin: 0 0 12pt }
.ex h2 { font-size: 12pt; margin: 0 0 2pt }
.ex .note { color: #666; font-size: 9pt; margin: 0 0 8pt }
.case { display: flex; width: 300pt; height: 30pt; margin: 0 0 5pt; border: 1pt solid #ddd }
.item { width: 40pt; height: 30pt; font-size: 8pt }
.center { justify-content: center }
.end { justify-content: flex-end }
.between { justify-content: space-between }
.constrained-flex { display: flex; width: 10pt }
.constrained-flex .svg-item { max-height: 100pt }
`)

	const pageW, pageH, margin = 841.89, 595.28, 34.02

	res, err := Layout(root, Options{
		Width: pageW - 2*margin, Height: pageH - 2*margin, Background: true,
		Sheets: []*css.Stylesheet{styleSheet}, Media: "print",
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	return res
}

// flexFrameEdges holds the painted border edges of one uniform-width frame:
// the horizontal top/bottom lines, the vertical rails (possibly split into
// page fragments per side, tracked by their union span), and how many of
// those lines were left with zero width by pagination.
type flexFrameEdges struct {
	topY, bottomY         float64
	leftTop, leftBottom   float64
	rightTop, rightBottom float64
	horizontals           int
	leftRails, rightRails int
	zeroWidth             int
}

// frameEdgesOf collects the container's own border lines from its op range.
// Membership is geometric (lines run on the box's own left/right/top/bottom
// edges), so nested row frames inside a tile range are not picked up. A rail
// split at a page boundary contributes two fragments to its side's span.
func frameEdgesOf(res *Result, boxNode *box) flexFrameEdges {
	edges := flexFrameEdges{
		topY: math.Inf(1), bottomY: math.Inf(-1),
		leftTop: math.Inf(1), leftBottom: math.Inf(-1),
		rightTop: math.Inf(1), rightBottom: math.Inf(-1),
	}

	start, end := frameLineRange(res, boxNode)

	for idx := start; idx <= end; idx++ {
		lineOp := res.Ops[idx]
		if lineOp.Kind == OpLine {
			edges.addLine(lineOp, boxNode)
		}
	}

	if edges.horizontals == 0 {
		edges.topY, edges.bottomY = 0, 0
	}

	if edges.leftRails == 0 {
		edges.leftTop, edges.leftBottom = 0, 0
	}

	if edges.rightRails == 0 {
		edges.rightTop, edges.rightBottom = 0, 0
	}

	return edges
}

// frameLineRange returns the op index range to scan for a box, clamped to the
// live op list. An empty range returns start > end.
func frameLineRange(res *Result, boxNode *box) (int, int) {
	if boxNode == nil || boxNode.opStart < 0 || len(res.Ops) == 0 {
		return 0, -1
	}

	end := boxNode.opEnd
	if end >= len(res.Ops) {
		end = len(res.Ops) - 1
	}

	return boxNode.opStart, end
}

// addLine files one border line as a horizontal edge or a vertical rail.
func (e *flexFrameEdges) addLine(lineOp Op, boxNode *box) {
	switch {
	case isFrameHorizontalLine(lineOp, boxNode):
		e.horizontals++
		e.topY = math.Min(e.topY, lineOp.Y)
		e.bottomY = math.Max(e.bottomY, lineOp.Y)
	case isFrameVerticalLine(lineOp, boxNode):
		if near(lineOp.X, boxNode.x) {
			e.leftRails++
			e.leftTop = math.Min(e.leftTop, lineOp.Y)
			e.leftBottom = math.Max(e.leftBottom, lineOp.Y+lineOp.H)
		} else {
			e.rightRails++
			e.rightTop = math.Min(e.rightTop, lineOp.Y)
			e.rightBottom = math.Max(e.rightBottom, lineOp.Y+lineOp.H)
		}
	default:
		return
	}

	if lineOp.Width <= 0 {
		e.zeroWidth++
	}
}

// isFrameHorizontalLine reports a horizontal line on the box's top or bottom
// edge with the box's own width.
func isFrameHorizontalLine(lineOp Op, boxNode *box) bool {
	return lineOp.H == 0 && lineOp.W > 0 &&
		near(lineOp.X, boxNode.x) && near(lineOp.W, boxNode.w)
}

// isFrameVerticalLine reports a vertical line on the box's left or right edge.
func isFrameVerticalLine(lineOp Op, boxNode *box) bool {
	return lineOp.W == 0 && lineOp.H > 0 &&
		(near(lineOp.X, boxNode.x) || near(lineOp.X, boxNode.x+boxNode.w))
}

// assertContiguousFlexFrame fails with the measured numbers when any painted
// edge of the box's frame sits away from the box's own rect, or when an edge
// line lost its stroke width during pagination. A fragmented frame passes
// when each side's fragments span the box edge to edge.
func assertContiguousFlexFrame(t *testing.T, res *Result, label string, boxNode *box) {
	t.Helper()

	edges := frameEdgesOf(res, boxNode)
	boxBottom := boxNode.y + boxNode.height
	where := fmt.Sprintf("%s (x=%.2f y=%.2f w=%.2f h=%.2f)", label, boxNode.x, boxNode.y, boxNode.w, boxNode.height)

	if edges.horizontals < 2 || edges.leftRails < 1 || edges.rightRails < 1 {
		t.Errorf("%s: collected %d horizontal and %d+%d vertical border lines, want at least 2 and 1+1"+
			" (frame=[%.2f,%.2f] left=[%.2f,%.2f] right=[%.2f,%.2f])",
			where, edges.horizontals, edges.leftRails, edges.rightRails,
			edges.topY, edges.bottomY, edges.leftTop, edges.leftBottom, edges.rightTop, edges.rightBottom)

		return
	}

	if edges.zeroWidth > 0 {
		t.Errorf("%s: %d border line(s) painted with zero stroke width "+
			"(frame=[%.2f,%.2f] left=[%.2f,%.2f] right=[%.2f,%.2f])",
			where, edges.zeroWidth, edges.topY, edges.bottomY,
			edges.leftTop, edges.leftBottom, edges.rightTop, edges.rightBottom)
	}

	if !near(edges.topY, boxNode.y) {
		t.Errorf("%s: top border at y=%.2f, want box top y=%.2f (off by %.2f)",
			where, edges.topY, boxNode.y, edges.topY-boxNode.y)
	}

	if !near(edges.bottomY, boxBottom) {
		t.Errorf("%s: bottom border at y=%.2f, want box bottom y=%.2f (off by %.2f)",
			where, edges.bottomY, boxBottom, edges.bottomY-boxBottom)
	}

	assertFrameRailSpan(t, where, "left", edges.leftTop, edges.leftBottom, boxNode.y, boxBottom)
	assertFrameRailSpan(t, where, "right", edges.rightTop, edges.rightBottom, boxNode.y, boxBottom)
}

// assertFrameRailSpan requires one side's rail fragments (one per page when
// the box is fragmented) to span the box from top to bottom.
func assertFrameRailSpan(t *testing.T, where, side string, railTop, railBottom, boxTop, boxBottom float64) {
	t.Helper()

	if !near(railTop, boxTop) || !near(railBottom, boxBottom) {
		t.Errorf("%s: %s rails span y=%.2f..%.2f but the box is y=%.2f..%.2f (gap %.2f, overshoot %.2f)",
			where, side, railTop, railBottom, boxTop, boxBottom,
			railTop-boxTop, railBottom-boxBottom)
	}
}

// TestFlexContainerBordersStayContiguousAcrossPagination reproduces the
// malformed frames from the section 4 and section 10 cases: a 300x30
// justify-content:center flex row whose horizontal borders move with the
// snapped text while its side rails stay behind (then get stretched), and an
// avoid-inside tile whose frame deskews from its box. Layout alone is
// contiguous; paginateOps introduces the box/chrome Y mismatch and
// stretchPaginatedChrome then stretches the rails along the stale box rect.
func flexSnapTileFixture(t *testing.T, spacerPt string) *Result {
	t.Helper()

	root := mustParse(t, `<html><body>`+
		`<div class="spacer"></div>`+
		`<section class="ex"><h2>snap tile</h2><p class="note">note text</p>`+
		`<div class="constrained-flex"><svg class="svg-item" width="200" height="200">`+
		`<rect width="200" height="200" fill="#0a0"/></svg></div>`+
		`</section>`+
		`<div class="tail"></div>`+
		`</body></html>`)

	styleSheet := sheet(t, fmt.Sprintf(`
html, body { margin: 0; padding: 0; font: 11pt/1.4 sans-serif }
.spacer { height: %s }
.ex { page-break-inside: avoid; overflow: hidden; border: 1px solid #bbb; padding: 8pt; margin: 0 0 12pt }
.ex h2 { font-size: 12pt; margin: 0 0 2pt }
.ex .note { color: #666; font-size: 9pt; margin: 0 0 8pt }
.constrained-flex { display: flex; width: 10pt }
.constrained-flex .svg-item { max-height: 100pt }
.tail { height: 40pt; background: #0aa }
`, spacerPt))

	const pageW, pageH, margin = 841.89, 595.28, 34.02

	res, err := Layout(root, Options{
		Width: pageW - 2*margin, Height: pageH - 2*margin, Background: true,
		Sheets: []*css.Stylesheet{styleSheet}, Media: "print",
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	return res
}

// TestAvoidTileFrameStaysContiguousWhenImageSnaps pins the section 10 frame:
// the tile's replaced image crosses the page boundary and must move to the
// next page whole, but the tile frame must stay one rect around its box. The
// pre-fix defect leaves the top rule below the box top and the bottom rule
// above the box bottom, with the rails stretched past both.
func firstImageOp(t *testing.T, res *Result, tile *box) Op {
	t.Helper()

	for opIdx := tile.opStart; opIdx <= tile.opEnd && opIdx < len(res.Ops); opIdx++ {
		if res.Ops[opIdx].Kind == OpImage {
			return res.Ops[opIdx]
		}
	}

	t.Fatalf("tile [%d,%d] holds no OpImage", tile.opStart, tile.opEnd)

	return Op{} // unreachable after Fatalf
}

// TestFlexWrapCrossingLineMovesWhole pins the wrapped-row case: the second
// wrapped flex line crosses the page boundary with fills over it but text
// inside the page. CSS Flexbox 10.1 (multi-line row container) moves a line
// that does not fit wholly to the next page; the engine must not split the
// item backgrounds at the boundary while their text stays behind.
func assertWholeFlexLineFills(t *testing.T, res *Result, items []*box, rowOf []int) {
	t.Helper()

	for itemIdx, itemBox := range items {
		fragments := itemFillFragments(res, itemBox)
		whole := false
		total := 0.0

		for _, fragment := range fragments {
			total += fragment.H

			if near(fragment.Y, itemBox.y) && near(fragment.H, itemBox.height) {
				whole = true
			}
		}

		if !whole {
			t.Errorf("item %d (line %d, x=%.2f y=%.2f h=%.2f): "+
				"fill fragments %s total %.2f, want one unsplit %.2fpt fill at y=%.2f",
				itemIdx, rowOf[itemIdx], itemBox.x, itemBox.y, itemBox.height,
				fillFragmentSummary(fragments), total, itemBox.height, itemBox.y)
		}
	}
}

// itemFillFragments collects the background fills painted at the item's x and
// width from the item's own row downward, whatever page fragments
// splitCrossingRects produced. Earlier rows share the item's x and width, so
// fragments above the item box top are other rows and are excluded.
func itemFillFragments(res *Result, itemBox *box) []Op {
	var fragments []Op

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpFillRect && near(paintOp.X, itemBox.x) &&
			near(paintOp.W, itemBox.w) && paintOp.H > 0 &&
			paintOp.Y >= itemBox.y-layoutCoordEpsilon {
			fragments = append(fragments, paintOp)
		}
	}

	return fragments
}

// fillFragmentSummary renders fragment Y/H pairs for a failure message.
func fillFragmentSummary(fragments []Op) string {
	out := ""

	for fragmentIdx, fragment := range fragments {
		if fragmentIdx > 0 {
			out += " + "
		}

		out += fmt.Sprintf("(y=%.2f h=%.2f)", fragment.Y, fragment.H)
	}

	if out == "" {
		return "(none)"
	}

	return out
}
