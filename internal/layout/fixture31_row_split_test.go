package layout

import (
	"math"
	"strings"
	"testing"
)

// TestFixture31SingleLineRowMovesWhole locks the fixture-31 row-split fix: a
// single-line .row whose line cannot fit the page must be exported as a
// KeepTogether range covering its border box, so a paginator moves the row
// whole to the next page top instead of leaving its top band on one page and
// snapping its text to the next. The sticky bar must not be kept: sticky
// stays in its natural fragment.
func TestFixture31SingleLineRowMovesWhole(t *testing.T) {
	t.Parallel()

	root, sheets := loadFixture(t, "fixture-31-sticky-top.html")

	// A4 content box with the PDF writer's default 10mm margins. The writer
	// lays blinkless out at this viewport and paginates at the same height.
	const (
		pageW = 595.28 - 2*28.3465
		pageH = 841.89 - 2*28.3465
	)

	res, err := Layout(root, Options{
		Width: pageW, Height: pageH, Sheets: sheets, Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	row29 := fixture31RowWithText(t, res, "Row 29")
	if row29.y >= pageH || row29.y+row29.height <= pageH {
		t.Fatalf("fixture no longer exercises the split: row 29 [%.2f %.2f] vs page boundary %.2f",
			row29.y, row29.y+row29.height, pageH)
	}

	_, keep := PageBreaks(res, pageH)

	if !keepCoversRange(keep, row29.y, row29.y+row29.height) {
		t.Fatalf("row 29 keep missing: keep=%v row=[%.2f %.2f] pageH=%.2f",
			keep, row29.y, row29.y+row29.height, pageH)
	}

	// Every .row that straddles a page boundary in this viewport must be
	// exported whole, so no row can lose its top edge or leave an empty
	// leading band on any page.
	for _, row := range fixture31Rows(t, res) {
		if fixture31Straddles(row, res.Height, pageH) && !keepCoversRange(keep, row.y, row.y+row.height) {
			t.Errorf("straddling row [%.2f %.2f] has no keep range: keep=%v",
				row.y, row.y+row.height, keep)
		}
	}

	// The sticky bar is positioned: keeping it would fight the paginator's
	// sticky pass, so it must not appear in the keep list.
	fixture31AssertStickyNotKept(t, res, keep)
}

// fixture31AssertStickyNotKept fails when the sticky bar appears in the keep
// list.
func fixture31AssertStickyNotKept(t *testing.T, res *Result, keep [][2]float64) {
	t.Helper()

	sticky := fixture31ClassBox(t, res, "sticky-bar")

	for _, kr := range keep {
		if math.Abs(kr[0]-sticky.y) < layoutCoordEpsilon &&
			math.Abs(kr[1]-(sticky.y+sticky.height)) < layoutCoordEpsilon {
			t.Errorf("sticky bar exported as keep range %v", kr)
		}
	}
}

// fixture31Rows returns the .row boxes in document order.
func fixture31Rows(t *testing.T, res *Result) []*box {
	t.Helper()

	var rows []*box

	var walk func(*box)

	walk = func(boxNode *box) {
		if boxNode == nil {
			return
		}

		if fixture31Class(boxNode) == "row" {
			rows = append(rows, boxNode)
		}

		for _, child := range boxNode.children {
			walk(child)
		}
	}
	walk(res.root)

	if len(rows) < 30 {
		t.Fatalf("found %d .row boxes, want the fixture's 35", len(rows))
	}

	return rows
}

// fixture31RowWithText returns the .row box whose ops include text starting
// with prefix.
func fixture31RowWithText(t *testing.T, res *Result, prefix string) *box {
	t.Helper()

	for _, row := range fixture31Rows(t, res) {
		for idx := row.opStart; idx <= row.opEnd && idx >= 0 && idx < len(res.Ops); idx++ {
			if res.Ops[idx].Kind == OpText && strings.HasPrefix(res.Ops[idx].Text, prefix) {
				return row
			}
		}
	}

	t.Fatalf("no .row containing text %q", prefix)

	return nil
}

// fixture31ClassBox returns the first box whose class attribute equals class.
func fixture31ClassBox(t *testing.T, res *Result, class string) *box {
	t.Helper()

	var found *box

	var walk func(*box)

	walk = func(boxNode *box) {
		if found != nil || boxNode == nil {
			return
		}

		if fixture31Class(boxNode) == class {
			found = boxNode

			return
		}

		for _, child := range boxNode.children {
			walk(child)
		}
	}
	walk(res.root)

	if found == nil {
		t.Fatalf("no box with class %q", class)
	}

	return found
}

func fixture31Class(boxNode *box) string {
	if boxNode == nil || boxNode.node == nil {
		return ""
	}

	return boxNode.node.Attribute("class")
}

// fixture31Straddles reports whether the box crosses any page boundary.
func fixture31Straddles(boxNode *box, canvasH, pageH float64) bool {
	if boxNode == nil || pageH <= 0 {
		return false
	}

	for boundary := pageH; boundary < canvasH; boundary += pageH {
		if boxNode.y < boundary && boxNode.y+boxNode.height > boundary {
			return true
		}
	}

	return false
}

// keepCoversRange reports whether keep holds an exact [start, end] range.
func keepCoversRange(keep [][2]float64, start, end float64) bool {
	for _, kr := range keep {
		if math.Abs(kr[0]-start) < layoutCoordEpsilon && math.Abs(kr[1]-end) < layoutCoordEpsilon {
			return true
		}
	}

	return false
}
