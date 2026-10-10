package layout

import "testing"

// This file holds the box-model and sizing behavior tests: every property
// below is asserted through a USED value (a measured border-box size or a
// measured offset between two boxes), never through a stored style string.
// The engine lays out in points and 1 CSS pixel = 0.75pt (ptPerCSSPx in
// css_review_02_test.go), so every expectation is written in CSS pixels and
// converted with pxToPt before comparing, which is how Chrome reports it.
// Reference browser for every case: Chrome 143.0.7499.40.

// behaviorBoxModelCheckUsedSize asserts the used border-box size of one
// element. wantWidthPx / wantHeightPx are CSS pixels.
func behaviorBoxModelCheckUsedSize(t *testing.T, res *Result, elementID string, wantWidthPx, wantHeightPx float64) {
	t.Helper()

	elementBox := boxByID(t, res, elementID)

	if !near(elementBox.w, pxToPt(wantWidthPx)) || !near(elementBox.height, pxToPt(wantHeightPx)) {
		t.Errorf("#%s used border box = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want %.2fpx x %.2fpx",
			elementID, elementBox.w, elementBox.height,
			elementBox.w/ptPerCSSPx, elementBox.height/ptPerCSSPx,
			wantWidthPx, wantHeightPx)
	}
}

// behaviorBoxModelCheckOffset asserts the used position of #innerID relative to
// #refID. wantDxPx / wantDyPx are CSS pixels; pass 0 for an axis the test does
// not constrain.
func behaviorBoxModelCheckOffset(
	t *testing.T, res *Result, innerID, refID string, wantDxPx, wantDyPx float64,
) {
	t.Helper()

	inner := boxByID(t, res, innerID)
	ref := boxByID(t, res, refID)

	if !near(inner.x-ref.x, pxToPt(wantDxPx)) || !near(inner.y-ref.y, pxToPt(wantDyPx)) {
		t.Errorf("#%s used offset from #%s = (%.4fpt, %.4fpt) = (%.2fpx, %.2fpx), want (%.2fpx, %.2fpx)",
			innerID, refID, inner.x-ref.x, inner.y-ref.y,
			(inner.x-ref.x)/ptPerCSSPx, (inner.y-ref.y)/ptPerCSSPx,
			wantDxPx, wantDyPx)
	}
}

// TestBehaviorWidthContentBoxUsedSize is CAT-05 width: under the initial
// content-box sizing a declared width is the CONTENT width, so an empty
// paddingless div declared 120px in a 400px wrapper lays out at a used border
// box of 120px, flush with the wrapper content edge.
// Reference: Chrome 143.0.7499.40, 400px wrapper, div width 120px.
func TestBehaviorWidthContentBoxUsedSize(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;height:200px">`+
		`<div id="item" style="width:120px;height:40px"></div>`+
		`</div></body></html>`)

	behaviorBoxModelCheckUsedSize(t, res, "item", 120, 40)
	behaviorBoxModelCheckOffset(t, res, "item", "outer", 0, 0)

	// The wrapper keeps its own declared content width, so the child really is
	// narrower than its containing block and not silently stretched to it.
	behaviorBoxModelCheckUsedSize(t, res, "outer", 400, 200)
}

// TestBehaviorHeightContentBoxUsedSize is CAT-05 height: under content-box
// sizing a declared height is the CONTENT height. An empty div declared 60px
// tall lays out at a used border box of 60px even though it has no in-flow
// content at all.
// Reference: Chrome 143.0.7499.40, 500px viewport, div height 60px.
func TestBehaviorHeightContentBoxUsedSize(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;height:200px">`+
		`<div id="item" style="height:60px"></div>`+
		`</div></body></html>`)

	behaviorBoxModelCheckUsedSize(t, res, "item", 400, 60)
}

// TestBehaviorBoxSizingBorderBoxIncludesPadding is CAT-05 box-sizing: with
// border-box the declared 100px x 50px is the whole border box, so the padding
// and border come out of it and the content box shrinks to 70px wide; with
// the initial content-box the same declarations give a 130px wide border box
// around a 100px content box. The auto-width children measure the content-box
// WIDTH each declaration leaves (100px vs 70px). Their used height is 0, not
// the parent content height: an empty div with auto height has no content, so
// both the engine and Chrome lay it out 0px tall.
// DEVIATION 2026-10-10: earlier revision expected the empty children at 50px
// and 40px tall; actual used height is 0px for both (verified test output).
// Reference: Chrome 143.0.7499.40, width 100px, height 50px, padding 0 10px,
// border 5px.
func TestBehaviorBoxSizingBorderBoxIncludesPadding(t *testing.T) {
	t.Parallel()

	const chrome = `width:100px;padding:0 10px;border:5px solid #000;height:50px`

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="contentBox" style="box-sizing:content-box;`+chrome+`">`+
		`<div id="cbChild"></div></div>`+
		`<div id="borderBox" style="box-sizing:border-box;`+chrome+`">`+
		`<div id="bbChild"></div></div>`+
		`</body></html>`)

	// 100px content + 10px + 10px padding + 5px + 5px border = 130px, and
	// 50px content + 5px + 5px border = 60px.
	behaviorBoxModelCheckUsedSize(t, res, "contentBox", 130, 60)
	behaviorBoxModelCheckUsedSize(t, res, "borderBox", 100, 50)

	// The auto-width children measure the content-box width each declaration
	// leaves. Both are empty with auto height, so their used height is 0
	// (see the test doc comment); only the widths probe the content box.
	behaviorBoxModelCheckUsedSize(t, res, "cbChild", 100, 0)
	behaviorBoxModelCheckUsedSize(t, res, "bbChild", 70, 0)

	// Both content boxes start 15px in: 5px border plus 10px padding.
	behaviorBoxModelCheckOffset(t, res, "cbChild", "contentBox", 15, 5)
	behaviorBoxModelCheckOffset(t, res, "bbChild", "borderBox", 15, 5)

	// The used widths differ by exactly the horizontal chrome: 20px padding
	// plus 10px border.
	delta := boxByID(t, res, "contentBox").w - boxByID(t, res, "borderBox").w
	if !near(delta, pxToPt(30)) {
		t.Errorf("content-box minus border-box used width = %.4fpt (%.2fpx), want 30px = 20px padding + 10px border",
			delta, delta/ptPerCSSPx)
	}
}

// TestBehaviorMinWidthClampsUsedWidth is CAT-05 min-width: a box declared 50px
// wide with min-width 120px is laid out at the 120px floor, while a box already
// wider than its min-width keeps its declared width.
// Reference: Chrome 143.0.7499.40, 400px wrapper.
func TestBehaviorMinWidthClampsUsedWidth(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		style  string
		wantPx float64
	}{
		{"clamped up to the floor", "width:50px;min-width:120px;height:30px", 120},
		{"declared width already wins", "width:200px;min-width:120px;height:30px", 200},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
				`<div id="outer" style="width:400px">`+
				`<div id="item" style="`+testCase.style+`"></div>`+
				`</div></body></html>`)

			behaviorBoxModelCheckUsedSize(t, res, "item", testCase.wantPx, 30)
		})
	}
}

// TestBehaviorMaxWidthClampsUsedWidth is CAT-05 max-width: a box declared 300px
// wide in a 400px wrapper with max-width 180px is laid out at the 180px
// ceiling, while a narrower declaration is untouched.
// Reference: Chrome 143.0.7499.40, 400px wrapper.
func TestBehaviorMaxWidthClampsUsedWidth(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		style  string
		wantPx float64
	}{
		{"clamped down to the ceiling", "width:300px;max-width:180px;height:30px", 180},
		{"declared width under the cap", "width:80px;max-width:180px;height:30px", 80},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
				`<div id="outer" style="width:400px">`+
				`<div id="item" style="`+testCase.style+`"></div>`+
				`</div></body></html>`)

			behaviorBoxModelCheckUsedSize(t, res, "item", testCase.wantPx, 30)
		})
	}
}

// TestBehaviorMinHeightClampsUsedHeight is CAT-05 min-height: a box declared
// 20px tall with min-height 90px is laid out at the 90px floor; a taller
// declaration is untouched. The clamp is on the used border box, so a
// fixed 90px height wins over a 20px declaration.
// Reference: Chrome 143.0.7499.40, 500px viewport.
func TestBehaviorMinHeightClampsUsedHeight(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		style  string
		wantPx float64
	}{
		{"clamped up to the floor", "width:200px;height:20px;min-height:90px", 90},
		{"declared height already wins", "width:200px;height:150px;min-height:90px", 150},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
				`<div id="item" style="`+testCase.style+`"></div>`+
				`</body></html>`)

			behaviorBoxModelCheckUsedSize(t, res, "item", 200, testCase.wantPx)
		})
	}
}

// TestBehaviorMaxHeightClampsUsedHeight is CAT-05 max-height: a box declared
// 200px tall with max-height 60px is laid out at the 60px ceiling and its
// content overflows rather than growing the box; a 40px declaration is
// untouched.
// Reference: Chrome 143.0.7499.40, 500px viewport.
func TestBehaviorMaxHeightClampsUsedHeight(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		style  string
		wantPx float64
	}{
		{"clamped down to the ceiling", "width:200px;height:200px;max-height:60px", 60},
		{"declared height under the cap", "width:200px;height:40px;max-height:60px", 40},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
				`<div id="item" style="`+testCase.style+`"></div>`+
				`</body></html>`)

			behaviorBoxModelCheckUsedSize(t, res, "item", 200, testCase.wantPx)
		})
	}
}

// TestBehaviorMarginTopOffsetsSibling is CAT-05 margin-top: the 24px top
// margin on the first in-flow child of the borderless, paddingless #outer
// collapses THROUGH the parent chain (CSS 2.1 parent/first-child collapse),
// so it moves the body box down 24px instead of opening space inside #outer.
// The first box stays flush with the parent content edge, the sibling stacks
// directly below it with a 0px gap, and the 24px shows up as the body offset.
// DEVIATION 2026-10-10: earlier revision expected the margin to pad the inside
// of #outer (first at 24px, sibling 64px below first, 24px gap); actual used
// geometry is first at (0, 0), sibling 40px below first (one 40px box height),
// gap 0px, body shifted 24px (verified test output, root_margin.go
// absorbRootBodyTopMargin plus pushCollapsedTopOverrides).
// Reference: Chrome 143.0.7499.40, 400px wrapper.
func TestBehaviorMarginTopOffsetsSibling(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body id="body" style="margin:0">`+
		`<div id="outer" style="width:400px">`+
		`<div id="first" style="margin-top:24px;height:40px"></div>`+
		`<div id="second" style="height:30px"></div>`+
		`</div></body></html>`)

	first := boxByID(t, res, "first")
	second := boxByID(t, res, "second")

	behaviorBoxModelCheckUsedSize(t, res, "first", 400, 40)
	behaviorBoxModelCheckUsedSize(t, res, "second", 400, 30)
	behaviorBoxModelCheckOffset(t, res, "second", "first", 0, 40)

	// The collapsed margin leaves the first box flush with its parent.
	behaviorBoxModelCheckOffset(t, res, "first", "outer", 0, 0)

	// No margin applies between the siblings, so the gap is 0.
	if !near(second.y-(first.y+first.height), pxToPt(0)) {
		t.Errorf("sibling gap = %.4fpt (%.2fpx), want 0px, the top margin collapsed through the parent",
			second.y-(first.y+first.height), (second.y-(first.y+first.height))/ptPerCSSPx)
	}

	// The collapsed 24px margin moved the whole body down instead.
	behaviorBoxModelCheckOffset(t, res, "outer", "body", 0, 0)

	body := boxByID(t, res, "body")

	if !near(body.y, pxToPt(24)) {
		t.Errorf("body y = %.4fpt (%.2fpx), want 24px collapsed top margin",
			body.y, body.y/ptPerCSSPx)
	}
}

// TestBehaviorMarginLeftOffsetsSibling is CAT-05 margin-left: a 30px left
// margin shifts a 100px wide block 30px right of the content edge. Both divs
// are block-level, so the following sibling does NOT start at the first box's
// right edge: it stacks on the next line at the parent content edge, one box
// height (20px) below the first box's top, back at x offset 0.
// DEVIATION 2026-10-10: earlier revision expected the sibling 130px right of
// the first box's left edge at the same height (inline-style side-by-side);
// actual used offset is (-30px, 20px) with the sibling's left edge back at the
// parent content edge (verified test output).
// Reference: Chrome 143.0.7499.40, 400px wrapper.
func TestBehaviorMarginLeftOffsetsSibling(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px">`+
		`<div id="first" style="margin-left:30px;width:100px;height:20px"></div>`+
		`<div id="second" style="height:20px"></div>`+
		`</div></body></html>`)

	first := boxByID(t, res, "first")
	second := boxByID(t, res, "second")
	outer := boxByID(t, res, "outer")

	behaviorBoxModelCheckUsedSize(t, res, "first", 100, 20)
	behaviorBoxModelCheckOffset(t, res, "first", "outer", 30, 0)
	behaviorBoxModelCheckOffset(t, res, "second", "first", -30, 20)

	// The block-level sibling starts a new line at the parent content edge,
	// not at the first box's right edge.
	if !near(second.x, outer.x) {
		t.Errorf("sibling x = %.4fpt (%.2fpx), want the parent content edge %.2fpx",
			second.x, second.x/ptPerCSSPx, outer.x/ptPerCSSPx)
	}

	if !near(second.y-(first.y+first.height), pxToPt(0)) {
		t.Errorf("sibling block gap = %.4fpt (%.2fpx), want 0px for stacked blocks",
			second.y-(first.y+first.height), (second.y-(first.y+first.height))/ptPerCSSPx)
	}
}

// TestBehaviorPaddingLeftOffsetsContent is CAT-05 padding-left: the content of
// a padded box starts at its content edge. A wrapper with a 4px left border
// and 20px left padding puts its child's left border edge 24px inside, and the
// 400px content-box declaration makes the wrapper's used border box 424px.
// Reference: Chrome 143.0.7499.40, width 400px, padding-left 20px,
// border-left 4px.
func TestBehaviorPaddingLeftOffsetsContent(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="box-sizing:content-box;width:400px;`+
		`padding-left:20px;border-left:4px solid #000;height:120px">`+
		`<div id="inner" style="width:100px;height:20px"></div>`+
		`</div></body></html>`)

	behaviorBoxModelCheckUsedSize(t, res, "outer", 424, 120)
	behaviorBoxModelCheckOffset(t, res, "inner", "outer", 24, 0)
	behaviorBoxModelCheckUsedSize(t, res, "inner", 100, 20)
}

// TestBehaviorPaddingTopOffsetsContent is CAT-05 padding-top: the content of a
// padded box starts below its padding and border. A wrapper with a 6px top
// border and 16px top padding puts its child's top border edge 22px inside,
// which is a measured coordinate, not the stored declaration.
// Reference: Chrome 143.0.7499.40, padding-top 16px, border-top 6px.
func TestBehaviorPaddingTopOffsetsContent(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;`+
		`padding-top:16px;border-top:6px solid #000;height:120px">`+
		`<div id="inner" style="width:100px;height:20px"></div>`+
		`</div></body></html>`)

	behaviorBoxModelCheckUsedSize(t, res, "outer", 400, 142)
	behaviorBoxModelCheckOffset(t, res, "inner", "outer", 0, 22)
	behaviorBoxModelCheckUsedSize(t, res, "inner", 100, 20)
}

// TestBehaviorMinInlineSizeClampsUsedWidth is CAT-05 min-inline-size: in the
// horizontal writing mode the logical minimum maps onto the physical inline
// axis, so width 40px with min-inline-size 150px is laid out at a used width
// of 150px, and an already wider declaration is left alone.
// Reference: Chrome 143.0.7499.40, 400px wrapper, horizontal-tb writing mode.
func TestBehaviorMinInlineSizeClampsUsedWidth(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		style  string
		wantPx float64
	}{
		{"clamped up to the floor", "width:40px;min-inline-size:150px;height:30px", 150},
		{"declared width already wins", "width:260px;min-inline-size:150px;height:30px", 260},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
				`<div id="outer" style="width:400px">`+
				`<div id="item" style="`+testCase.style+`"></div>`+
				`</div></body></html>`)

			behaviorBoxModelCheckUsedSize(t, res, "item", testCase.wantPx, 30)
		})
	}
}

// TestBehaviorMaxInlineSizeClampsUsedWidth is CAT-05 max-inline-size: the
// logical maximum maps onto the physical inline axis in the horizontal writing
// mode, so width 250px with max-inline-size 90px is laid out at a used width
// of 90px.
// Reference: Chrome 143.0.7499.40, 400px wrapper, horizontal-tb writing mode.
func TestBehaviorMaxInlineSizeClampsUsedWidth(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		style  string
		wantPx float64
	}{
		{"clamped down to the ceiling", "width:250px;max-inline-size:90px;height:30px", 90},
		{"declared width under the cap", "width:60px;max-inline-size:90px;height:30px", 60},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
				`<div id="outer" style="width:400px">`+
				`<div id="item" style="`+testCase.style+`"></div>`+
				`</div></body></html>`)

			behaviorBoxModelCheckUsedSize(t, res, "item", testCase.wantPx, 30)
		})
	}
}

// TestBehaviorMinBlockSizeClampsUsedHeight is CAT-05 min-block-size: in the
// horizontal writing mode the logical minimum maps onto the physical block
// axis, so height 30px with min-block-size 110px is laid out at a used height
// of 110px.
// Reference: Chrome 143.0.7499.40, 500px viewport, horizontal-tb writing mode.
func TestBehaviorMinBlockSizeClampsUsedHeight(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		style  string
		wantPx float64
	}{
		{"clamped up to the floor", "width:200px;height:30px;min-block-size:110px", 110},
		{"declared height already wins", "width:200px;height:190px;min-block-size:110px", 190},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
				`<div id="item" style="`+testCase.style+`"></div>`+
				`</body></html>`)

			behaviorBoxModelCheckUsedSize(t, res, "item", 200, testCase.wantPx)
		})
	}
}

// TestBehaviorMaxBlockSizeClampsUsedHeight is CAT-05 max-block-size: the
// logical maximum maps onto the physical block axis in the horizontal writing
// mode, so height 180px with max-block-size 70px is laid out at a used height
// of 70px.
// Reference: Chrome 143.0.7499.40, 500px viewport, horizontal-tb writing mode.
func TestBehaviorMaxBlockSizeClampsUsedHeight(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		style  string
		wantPx float64
	}{
		{"clamped down to the ceiling", "width:200px;height:180px;max-block-size:70px", 70},
		{"declared height under the cap", "width:200px;height:40px;max-block-size:70px", 40},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
				`<div id="item" style="`+testCase.style+`"></div>`+
				`</body></html>`)

			behaviorBoxModelCheckUsedSize(t, res, "item", 200, testCase.wantPx)
		})
	}
}

// TestBehaviorAspectRatioDerivesHeightFromWidth is CAT-05 aspect-ratio: with an
// auto height the ratio fixes the used height from the used width. A 160px
// wide box with aspect-ratio 2/1 is laid out 80px tall, and a 120px wide box
// with aspect-ratio 1/2 is laid out 240px tall, both with no content. The
// colon form "4 : 1" is NOT a valid ratio (ratios use a slash), so the engine
// rejects it and the empty box keeps its auto height of 0px.
// DEVIATION 2026-10-10: earlier revision expected the colon form at 200px x
// 50px; the parser in style_aspect_ratio_props.go only splits on "/", so the
// declaration is dropped and the used size is 200px x 0px (verified output).
// Reference: Chrome 143.0.7499.40, 500px viewport.
func TestBehaviorAspectRatioDerivesHeightFromWidth(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		style   string
		wantWPx float64
		wantHPx float64
	}{
		{"wide ratio halves the height", "width:160px;aspect-ratio:2/1", 160, 80},
		{"tall ratio doubles the height", "width:120px;aspect-ratio:1/2", 120, 240},
		{"colon form is not a ratio, auto height stays zero", "width:200px;aspect-ratio:4 : 1", 200, 0},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
				`<div id="item" style="`+testCase.style+`"></div>`+
				`</body></html>`)

			behaviorBoxModelCheckUsedSize(t, res, "item", testCase.wantWPx, testCase.wantHPx)
		})
	}
}
