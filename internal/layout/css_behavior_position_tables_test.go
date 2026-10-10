package layout

import (
	"strings"
	"testing"
)

// This file holds the positioning and table behavior tests: every property
// below is asserted through a USED value (a used box offset, a used paint
// order, a used column gap), never through a stored style string. The engine
// lays out in points and 1 CSS pixel = 0.75pt (ptPerCSSPx in
// css_review_02_test.go), so every expectation is written in CSS pixels and
// converted with pxToPt before comparing, which is how Chrome reports it.
// Reference browser for every case: Chrome 143.0.7499.40.

// TestBehaviorPositionRelativeKeepsFlow is position: a relative box with
// top/left offsets shifts visually but keeps its static flow space, so the
// next sibling stays where static flow put it. Reference: Chrome
// 143.0.7499.40, two 50px blocks, first shifted by top 20px left 30px, second
// still starts at 50px.
func TestBehaviorPositionRelativeKeepsFlow(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="s1" style="position:relative;top:20px;left:30px;width:50px;height:50px"></div>`+
		`<div id="s2" style="width:50px;height:50px"></div>`+
		`</body></html>`)

	shifted := boxByID(t, res, "s1")
	next := boxByID(t, res, "s2")

	if !near(shifted.x, pxToPt(30)) || !near(shifted.y, pxToPt(20)) {
		t.Errorf("s1 = (%.4fpt, %.4fpt) = (%.2fpx, %.2fpx), want (30px, 20px)",
			shifted.x, shifted.y, shifted.x/ptPerCSSPx, shifted.y/ptPerCSSPx)
	}

	if !near(next.y, pxToPt(50)) {
		t.Errorf("s2.y = %.4fpt (%.2fpx), want 50px (static flow kept)",
			next.y, next.y/ptPerCSSPx)
	}
}

// TestBehaviorTopShiftsRelativeBox is top: top:20px on a relative box moves
// its used y down 20px against the same box with top:auto at y 0. Reference:
// Chrome 143.0.7499.40, 50px block, top 20px lands its border top at 20px.
func TestBehaviorTopShiftsRelativeBox(t *testing.T) {
	t.Parallel()

	with := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="t1" style="position:relative;top:20px;width:50px;height:50px"></div>`+
		`</body></html>`)
	without := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="t1" style="position:relative;width:50px;height:50px"></div>`+
		`</body></html>`)

	got := boxByID(t, with, "t1")
	base := boxByID(t, without, "t1")

	if !near(base.y, 0) {
		t.Fatalf("auto-top base y = %.4fpt, want 0", base.y)
	}

	if !near(got.y-base.y, pxToPt(20)) {
		t.Errorf("top delta = %.4fpt (%.2fpx), want 20px",
			got.y-base.y, (got.y-base.y)/ptPerCSSPx)
	}
}

// TestBehaviorLeftShiftsRelativeBox is left: left:30px on a relative box
// moves its used x right 30px against the same box with left:auto at x 0.
// Reference: Chrome 143.0.7499.40, 50px block, left 30px lands its border
// left at 30px.
func TestBehaviorLeftShiftsRelativeBox(t *testing.T) {
	t.Parallel()

	with := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="l1" style="position:relative;left:30px;width:50px;height:50px"></div>`+
		`</body></html>`)
	without := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="l1" style="position:relative;width:50px;height:50px"></div>`+
		`</body></html>`)

	got := boxByID(t, with, "l1")
	base := boxByID(t, without, "l1")

	if !near(base.x, 0) {
		t.Fatalf("auto-left base x = %.4fpt, want 0", base.x)
	}

	if !near(got.x-base.x, pxToPt(30)) {
		t.Errorf("left delta = %.4fpt (%.2fpx), want 30px",
			got.x-base.x, (got.x-base.x)/ptPerCSSPx)
	}
}

// TestBehaviorInsetAnchorsAbsoluteBox is inset: inset:16px on an absolute box
// with a definite size pins its used top-left 16px inside the relative
// parent's padding box. Reference: Chrome 143.0.7499.40, 200px relative
// parent, 50x40px absolute child lands at (16, 16)px.
func TestBehaviorInsetAnchorsAbsoluteBox(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="par" style="position:relative;width:200px;height:200px">`+
		`<div id="kid" style="position:absolute;inset:16px;width:50px;height:40px"></div>`+
		`</div></body></html>`)

	parent := boxByID(t, res, "par")
	kid := boxByID(t, res, "kid")

	if !near(kid.x-parent.x, pxToPt(16)) || !near(kid.y-parent.y, pxToPt(16)) {
		t.Errorf("kid offset = (%.4fpt, %.4fpt) = (%.2fpx, %.2fpx), want (16px, 16px)",
			kid.x-parent.x, kid.y-parent.y,
			(kid.x-parent.x)/ptPerCSSPx, (kid.y-parent.y)/ptPerCSSPx)
	}

	if !near(kid.w, pxToPt(50)) || !near(kid.height, pxToPt(40)) {
		t.Errorf("kid size = %.4fpt x %.4fpt, want 50px x 40px", kid.w, kid.height)
	}
}

// TestBehaviorZIndexOrdersOverlappingPaint is z-index: of two fully
// overlapping absolute boxes the higher z-index paints later under the shared
// PaintOrder policy (see paint_order.go), and swapping the values swaps the
// order. Raw emission stays in document order; the ZIndex stamp on each op
// decides. Reference: Chrome 143.0.7499.40, two 100px squares at (0, 0), z 1
// under z 2.
func TestBehaviorZIndexOrdersOverlappingPaint(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		redZ     string
		blueZ    string
		redOnTop bool
	}{
		{name: "red on top", redZ: "2", blueZ: "1", redOnTop: true},
		{name: "blue on top", redZ: "1", blueZ: "2", redOnTop: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
				`<div id="par" style="position:relative;width:200px;height:200px">`+
				`<div id="red" style="position:absolute;top:0;left:0;width:100px;height:100px;`+
				`background-color:#ff0000;z-index:`+testCase.redZ+`"></div>`+
				`<div id="blue" style="position:absolute;top:0;left:0;width:100px;height:100px;`+
				`background-color:#0000ff;z-index:`+testCase.blueZ+`"></div>`+
				`</div></body></html>`)

			redIdx, blueIdx := behaviorPosTblSquareIndexes(t, res.Ops)

			// PaintOrder resolves the ZIndex stamps into paint sequence.
			rank := make(map[int]int, len(res.Ops))
			for rankPos, opIdx := range PaintOrder(res.Ops) {
				rank[opIdx] = rankPos
			}

			redRank, blueRank := rank[redIdx], rank[blueIdx]

			if testCase.redOnTop && !(redRank > blueRank) {
				t.Errorf("red z 2 must paint after blue z 1: red=%d blue=%d", redRank, blueRank)
			}

			if !testCase.redOnTop && !(blueRank > redRank) {
				t.Errorf("blue z 2 must paint after red z 1: blue=%d red=%d", blueRank, redRank)
			}
		})
	}
}

// behaviorPosTblSquareColor names the primary color of a square fill op.
func behaviorPosTblSquareColor(paintOp Op) string {
	switch {
	case paintOp.R > 0.9 && paintOp.G < 0.1 && paintOp.B < 0.1:
		return "red"
	case paintOp.B > 0.9 && paintOp.R < 0.1 && paintOp.G < 0.1:
		return "blue"
	default:
		return ""
	}
}

// behaviorPosTblSquareIndexes returns the op indexes of the red and blue
// 100px squares, failing when either is missing.
func behaviorPosTblSquareIndexes(t *testing.T, ops []Op) (int, int) {
	t.Helper()

	redIdx, blueIdx := -1, -1

	for idx, paintOp := range ops {
		if paintOp.Kind != OpFillRect || !near(paintOp.W, pxToPt(100)) || !near(paintOp.H, pxToPt(100)) {
			continue
		}

		switch behaviorPosTblSquareColor(paintOp) {
		case "red":
			redIdx = idx
		case "blue":
			blueIdx = idx
		}
	}

	if redIdx < 0 || blueIdx < 0 {
		t.Fatalf("missing square rects: red=%d blue=%d", redIdx, blueIdx)
	}

	return redIdx, blueIdx
}

// TestBehaviorFloatLeftSharesBandWithSibling is float: a left float and a
// right float share one y band with the left box west of the right box.
// Reference: Chrome 143.0.7499.40, 60px left float and 80px right float both
// start at the container top.
func TestBehaviorFloatLeftSharesBandWithSibling(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0;width:400px">`+
		`<div id="logo" style="float:left;width:60px;height:100px"></div>`+
		`<div id="meta" style="float:right;width:80px;height:60px"></div>`+
		`</body></html>`)

	logo := boxByID(t, res, "logo")
	meta := boxByID(t, res, "meta")

	if !(logo.x < meta.x) {
		t.Errorf("logo x=%.4fpt should be left of meta x=%.4fpt", logo.x, meta.x)
	}

	if !near(logo.y, meta.y) {
		t.Errorf("floats should share a band: logo y=%.4fpt meta y=%.4fpt", logo.y, meta.y)
	}

	if !near(logo.height, pxToPt(100)) || !near(meta.height, pxToPt(60)) {
		t.Errorf("float heights = %.4fpt and %.4fpt, want 100px and 60px",
			logo.height, meta.height)
	}
}

// TestBehaviorClearDropsBelowFloats is clear: a clear:both block starts at or
// below the bottom of a 100px float. Reference: Chrome 143.0.7499.40, 100px
// left float, clearing block border top at 100px.
func TestBehaviorClearDropsBelowFloats(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="cff" style="float:left;width:60px;height:100px"></div>`+
		`<div id="cbelow" style="clear:both;height:20px">below</div>`+
		`</body></html>`)

	floating := boxByID(t, res, "cff")
	cleared := boxByID(t, res, "cbelow")

	bottom := floating.y + floating.height

	if cleared.y+0.01 < bottom {
		t.Errorf("clear y=%.4fpt (%.2fpx) should be at or below float bottom %.4fpt (100px)",
			cleared.y, cleared.y/ptPerCSSPx, bottom)
	}
}

// TestBehaviorOverflowVisibleKeepsTallChild is overflow: a 200x50px visible
// container keeps its used 50px height and keeps every child line live, while
// the same container with overflow:hidden deactivates the lines past the clip
// bottom. Reference: Chrome 143.0.7499.40, five 18px lines (90px) in a 50px
// box: visible paints all five, hidden clips the lines starting past 50px.
func TestBehaviorOverflowVisibleKeepsTallChild(t *testing.T) {
	t.Parallel()

	inner := `<div id="tall" style="font-size:16px">` +
		`<p id="v1" style="margin:0">visline one</p>` +
		`<p id="v2" style="margin:0">visline two</p>` +
		`<p id="v3" style="margin:0">visline three</p>` +
		`<p id="v4" style="margin:0">visline four</p>` +
		`<p id="v5" style="margin:0">visline five</p>` +
		`</div>`
	visible := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="clip" style="width:200px;height:50px;overflow:visible">`+inner+`</div></body></html>`)
	hidden := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="clip" style="width:200px;height:50px;overflow:hidden">`+inner+`</div></body></html>`)

	clip := boxByID(t, visible, "clip")

	if !near(clip.w, pxToPt(200)) || !near(clip.height, pxToPt(50)) {
		t.Fatalf("clip used box = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want 200px x 50px",
			clip.w, clip.height, clip.w/ptPerCSSPx, clip.height/ptPerCSSPx)
	}

	visLive, visDead := behaviorPosTblCountVislines(visible)
	hidLive, hidDead := behaviorPosTblCountVislines(hidden)

	if visLive == 0 {
		t.Errorf("visible container painted no live visline text, want all three lines live")
	}

	if visDead != 0 {
		t.Errorf("visible container deactivated %d visline ops, want 0", visDead)
	}

	if hidDead == 0 {
		t.Errorf("hidden container deactivated no visline ops, want lines past 50px removed")
	}

	if hidLive >= visLive {
		t.Errorf("hidden live=%d should be below visible live=%d", hidLive, visLive)
	}
}

// behaviorPosTblCountVislines counts live and deactivated visline text ops.
func behaviorPosTblCountVislines(res *Result) (int, int) {
	live, dead := 0, 0

	for _, paintOp := range res.Ops {
		if !strings.Contains(paintOp.Text, "visline") {
			continue
		}

		if paintOp.Kind == opKindNoop {
			dead++
		} else if paintOp.Kind == OpText {
			live++
		}
	}

	return live, dead
}

// TestBehaviorVisibilityHiddenKeepsGeometry is visibility: hidden keeps the
// used box (100x50px at the same x/y) while painting no text for it.
// Reference: Chrome 143.0.7499.40, 100x50px block: hidden reserves the same
// space as visible but draws nothing.
func TestBehaviorVisibilityHiddenKeepsGeometry(t *testing.T) {
	t.Parallel()

	shown := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="vbox" style="width:100px;height:50px">hideme words</div>`+
		`</body></html>`)
	hidden := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="vbox" style="width:100px;height:50px;visibility:hidden">hideme words</div>`+
		`</body></html>`)

	showBox := boxByID(t, shown, "vbox")
	hideBox := boxByID(t, hidden, "vbox")

	for _, field := range []struct {
		name string
		a, b float64
	}{
		{"x", showBox.x, hideBox.x},
		{"y", showBox.y, hideBox.y},
		{"w", showBox.w, hideBox.w},
		{"height", showBox.height, hideBox.height},
	} {
		if !near(field.a, field.b) {
			t.Errorf("hidden %s = %.4fpt, visible = %.4fpt, want identical geometry",
				field.name, field.b, field.a)
		}
	}

	live := behaviorPosTblCountHideme

	if live(shown) == 0 {
		t.Errorf("visible box painted no hideme text, want at least one live op")
	}

	if live(hidden) != 0 {
		t.Errorf("hidden box painted %d live hideme ops, want 0", live(hidden))
	}
}

// TestBehaviorTableLayoutFixedEqualShare is table-layout: a fixed 200px
// table splits into two equal 100px columns whatever the content, so the
// second cell starts 100px right of the first. Reference: Chrome
// 143.0.7499.40, fixed 200px two-column table, second column at 100px.
func TestBehaviorTableLayoutFixedEqualShare(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<table style="table-layout:fixed;width:200px;border-collapse:collapse;border-spacing:0">`+
		`<tr><td id="f1" style="padding:0">a</td>`+
		`<td id="f2" style="padding:0">supercalifragilisticexpialidocious</td></tr>`+
		`</table></body></html>`)

	first := boxByID(t, res, "f1")
	second := boxByID(t, res, "f2")

	gapPx := (second.x - first.x) / ptPerCSSPx

	if gapPx < 90 || gapPx > 110 {
		t.Errorf("fixed column gap = %.4fpt (%.2fpx), want about 100px equal share",
			second.x-first.x, gapPx)
	}
}

// TestBehaviorBorderCollapseMergesAdjacency is border-collapse: with 2px
// cell borders the separate model paints each cell's own four border lines
// while collapse merges the adjoining borders into one grid run for the row.
// The used cell boxes match (see probe notes in the test), so the property
// is pinned through paint ops. Reference: Chrome 143.0.7499.40, two
// 2px-bordered cells: separate draws doubled interior borders, collapse one
// shared 2px line.
func TestBehaviorBorderCollapseMergesAdjacency(t *testing.T) {
	t.Parallel()

	rows := `<tr><td id="c1" style="padding:0;border:2px solid black">a</td>` +
		`<td id="c2" style="padding:0;border:2px solid black">b</td></tr>`
	separate := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<table style="border-collapse:separate;border-spacing:0">`+rows+`</table></body></html>`)
	collapsed := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<table style="border-collapse:collapse;border-spacing:0">`+rows+`</table></body></html>`)

	sepLines := opsOfKind(separate, OpLine)
	colLines := opsOfKind(collapsed, OpLine)
	colGrids := opsOfKind(collapsed, OpGridRun)

	// Two cells times four sides each.
	if len(sepLines) < 8 {
		t.Errorf("separate border lines = %d, want at least 8 (4 per cell)", len(sepLines))
	}

	if len(colLines) != 0 {
		t.Errorf("collapse line ops = %d, want 0 (merged into grid runs)", len(colLines))
	}

	if len(colGrids) != 1 {
		t.Fatalf("collapse grid runs = %d, want 1 for the single row", len(colGrids))
	}

	// The merged run spans both used cells.
	first := boxByID(t, collapsed, "c1")
	second := boxByID(t, collapsed, "c2")

	if !near(colGrids[0].W, second.x+second.w-first.x) {
		t.Errorf("grid run w = %.4fpt (%.2fpx), want both cells %.4fpt",
			colGrids[0].W, colGrids[0].W/ptPerCSSPx, second.x+second.w-first.x)
	}
}

// TestBehaviorBorderSpacingWidensColumns is border-spacing: 10px of table
// spacing pushes the second column 10px further right than spacing 0.
// Reference: Chrome 143.0.7499.40, two-cell table, gap grows by exactly 10px.
func TestBehaviorBorderSpacingWidensColumns(t *testing.T) {
	t.Parallel()

	rows := `<tr><td id="s1" style="padding:0">A</td><td id="s2" style="padding:0">B</td></tr>`
	flat := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<table style="border-spacing:0">`+rows+`</table></body></html>`)
	wide := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<table style="border-spacing:10px">`+rows+`</table></body></html>`)

	gap0 := boxByID(t, flat, "s2").x - boxByID(t, flat, "s1").x
	gap10 := boxByID(t, wide, "s2").x - boxByID(t, wide, "s1").x

	deltaPx := (gap10 - gap0) / ptPerCSSPx

	if deltaPx < 9 || deltaPx > 11 {
		t.Errorf("spacing delta = %.4fpt (%.2fpx), want 10px",
			gap10-gap0, deltaPx)
	}
}

// TestBehaviorCaptionSideBottomBelowTable is caption-side: top (the default)
// paints the caption above the first cell while bottom moves it below the
// cell row. Reference: Chrome 143.0.7499.40, captioned table: top caption
// ends above row 1, bottom caption starts below it.
func TestBehaviorCaptionSideBottomBelowTable(t *testing.T) {
	t.Parallel()

	top := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<table><caption id="cap">cap words</caption>`+
		`<tr><td id="cell">body words</td></tr></table></body></html>`)
	bottom := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<table style="caption-side:bottom"><caption id="cap">cap words</caption>`+
		`<tr><td id="cell">body words</td></tr></table></body></html>`)

	topCap := boxByID(t, top, "cap")
	topCell := boxByID(t, top, "cell")

	if !(topCap.y < topCell.y) {
		t.Errorf("top caption y=%.4fpt should be above cell y=%.4fpt",
			topCap.y, topCell.y)
	}

	bottomCap := boxByID(t, bottom, "cap")
	bottomCell := boxByID(t, bottom, "cell")

	if !(bottomCap.y > bottomCell.y) {
		t.Errorf("bottom caption y=%.4fpt should be below cell y=%.4fpt",
			bottomCap.y, bottomCell.y)
	}
}

// TestBehaviorEmptyCellsHideOmitsBackground is empty-cells: an empty cell
// with a set background paints its fill under show but omits it under hide.
// Reference: Chrome 143.0.7499.40, two-cell table with one empty
// background-colored cell: show paints two cell fills, hide paints one.
func TestBehaviorEmptyCellsHideOmitsBackground(t *testing.T) {
	t.Parallel()

	rows := `<tr><td style="padding:0 20px;background-color:#eeeeee"></td>` +
		`<td style="padding:0 20px;background-color:#eeeeee">full</td></tr>`
	shown := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<table style="empty-cells:show;border-spacing:0">`+rows+`</table></body></html>`)
	hidden := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<table style="empty-cells:hide;border-spacing:0">`+rows+`</table></body></html>`)

	fills := behaviorPosTblCountLightFills

	showFills, hideFills := fills(shown), fills(hidden)

	if !(showFills > hideFills) {
		t.Errorf("show fills=%d should exceed hide fills=%d (empty cell omitted)",
			showFills, hideFills)
	}
}

// behaviorPosTblCountHideme counts live hideme text ops in a result.
func behaviorPosTblCountHideme(res *Result) int {
	count := 0

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpText && strings.Contains(paintOp.Text, "hideme") {
			count++
		}
	}

	return count
}

// behaviorPosTblCountLightFills counts large light-gray fills in a result.
func behaviorPosTblCountLightFills(res *Result) int {
	count := 0

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpFillRect && paintOp.R > 0.8 && paintOp.G > 0.8 && paintOp.B > 0.8 &&
			paintOp.R < 1 && paintOp.W > 1 && paintOp.H > 1 {
			count++
		}
	}

	return count
}
