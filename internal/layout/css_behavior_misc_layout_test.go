package layout

import (
	"math"
	"strings"
	"testing"
)

// Misc layout behavior tests: background logical axes, containment,
// container queries, logical margins and padding, page floats, grid-area,
// and multicol level 2. Every property is asserted through a used value
// (a measured box edge or an emitted op), never through a stored style
// string. Lengths are written in CSS px and converted with pxToPt
// (1px = 0.75pt, ptPerCSSPx in css_review_02_test.go), which is how Chrome
// reports them. Reference browser for every case: Chrome 143.0.7499.40.
// Helpers boxByID, near, pxToPt, opsOfKind, layoutHTMLWithImages, and
// tinyPNG come from css_review_02_test.go and layout_test.go.

// behaviorMiscCheckUsedSize asserts the used border-box size in CSS px.
func behaviorMiscCheckUsedSize(t *testing.T, res *Result, elementID string, wantWPx, wantHPx float64) {
	t.Helper()

	elementBox := boxByID(t, res, elementID)

	if !near(elementBox.w, pxToPt(wantWPx)) || !near(elementBox.height, pxToPt(wantHPx)) {
		t.Errorf("#%s used border box = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want %.2fpx x %.2fpx",
			elementID, elementBox.w, elementBox.height,
			elementBox.w/ptPerCSSPx, elementBox.height/ptPerCSSPx,
			wantWPx, wantHPx)
	}
}

// behaviorMiscCheckOffset asserts the used position of innerID relative to
// refID in CSS px.
func behaviorMiscCheckOffset(t *testing.T, res *Result, innerID, refID string, wantDxPx, wantDyPx float64) {
	t.Helper()

	inner := boxByID(t, res, innerID)
	ref := boxByID(t, res, refID)

	if !near(inner.x-ref.x, pxToPt(wantDxPx)) || !near(inner.y-ref.y, pxToPt(wantDyPx)) {
		t.Errorf("#%s offset from #%s = (%.4fpt, %.4fpt) = (%.2fpx, %.2fpx), want (%.2fpx, %.2fpx)",
			innerID, refID, inner.x-ref.x, inner.y-ref.y,
			(inner.x-ref.x)/ptPerCSSPx, (inner.y-ref.y)/ptPerCSSPx,
			wantDxPx, wantDyPx)
	}
}

// behaviorMiscAssertSameOps pins no-op behavior: two results paint the same
// ops in the same order with identical geometry and text.
func behaviorMiscAssertSameOps(t *testing.T, first, second *Result) {
	t.Helper()

	if len(first.Ops) != len(second.Ops) {
		t.Fatalf("op count = %d vs %d, want identical geometry", len(first.Ops), len(second.Ops))
	}

	for i := range first.Ops {
		x, y := first.Ops[i], second.Ops[i]
		if x.Kind != y.Kind || x.Text != y.Text ||
			!near(x.X, y.X) || !near(x.Y, y.Y) ||
			!near(x.W, y.W) || !near(x.H, y.H) {
			t.Fatalf("op %d differs: %+v vs %+v, want identical geometry", i, x, y)
		}
	}
}

// behaviorMiscBgImages keeps the non-empty background image ops.
func behaviorMiscBgImages(ops []Op) []Op {
	out := make([]Op, 0, len(ops))

	for _, paintOp := range ops {
		if paintOp.Kind != OpImage || !paintOp.IsBackground {
			continue
		}

		if paintOp.W <= 0 || paintOp.H <= 0 {
			continue
		}

		out = append(out, paintOp)
	}

	return out
}

// behaviorMiscLabelYSpan returns the Y span and count of the label text ops
// in a result (the NATO-word paragraphs of the column fixtures).
func behaviorMiscLabelYSpan(t *testing.T, res *Result) (float64, float64, int) {
	t.Helper()

	minY := math.MaxFloat64
	maxY := -math.MaxFloat64
	count := 0

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || strings.TrimSpace(paintOp.Text) == "" {
			continue
		}

		count++

		if paintOp.Y < minY {
			minY = paintOp.Y
		}

		if paintOp.Y > maxY {
			maxY = paintOp.Y
		}
	}

	return minY, maxY, count
}

// TestBehaviorBackgroundPositionBlockOffsetsImage is background-position-block:
// in horizontal-tb the block axis is vertical, so top keeps a no-repeat image
// at the origin and bottom slides it to the far edge.
// Reference: Chrome 143.0.7499.40, 80px by 40px box, 10px by 10px image.
func TestBehaviorBackgroundPositionBlockOffsetsImage(t *testing.T) {
	t.Parallel()

	behaviorBgRunPositionCase(t, "background-position-block", "top", "bottom", "y")
}

// TestBehaviorBackgroundPositionInlineOffsetsImage is background-position-inline:
// in horizontal-tb the inline axis is horizontal, so left keeps a no-repeat
// image at the origin and right slides it to the far edge.
// Reference: Chrome 143.0.7499.40, 80px by 40px box, 10px by 10px image.
func TestBehaviorBackgroundPositionInlineOffsetsImage(t *testing.T) {
	t.Parallel()

	behaviorBgRunPositionCase(t, "background-position-inline", "left", "right", "x")
}

// TestBehaviorBackgroundRepeatBlockTilesHorizontally is background-repeat-block:
// in horizontal-tb the block axis is vertical, so no-repeat keeps one row of
// 8 tiles across the 80px box while the horizontal axis stays at repeat.
// Reference: Chrome 143.0.7499.40, 80px by 40px box, 10px by 10px image.
func TestBehaviorBackgroundRepeatBlockTilesHorizontally(t *testing.T) {
	t.Parallel()

	png := tinyPNG(10, 10)
	res := layoutHTMLWithImages(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:80px;height:40px;background-image:url(bg.png);`+
		`background-size:10px 10px;background-repeat-block:no-repeat"></div>`+
		`</body></html>`, png, "")

	tiles := behaviorMiscBgImages(res.Ops)
	if len(tiles) != 8 {
		t.Fatalf("background-repeat-block:no-repeat tiles = %d, want 8 in one row", len(tiles))
	}

	for _, tile := range tiles[1:] {
		if !near(tile.Y, tiles[0].Y) {
			t.Fatalf("tile y = %.4fpt, want row y %.4fpt", tile.Y, tiles[0].Y)
		}
	}
}

// TestBehaviorBackgroundRepeatInlineTilesVertically is background-repeat-inline:
// in horizontal-tb the inline axis is horizontal, so no-repeat keeps one
// column of 4 tiles down the 40px box while the vertical axis stays at repeat.
// Reference: Chrome 143.0.7499.40, 80px by 40px box, 10px by 10px image.
func TestBehaviorBackgroundRepeatInlineTilesVertically(t *testing.T) {
	t.Parallel()

	png := tinyPNG(10, 10)
	res := layoutHTMLWithImages(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:80px;height:40px;background-image:url(bg.png);`+
		`background-size:10px 10px;background-repeat-inline:no-repeat"></div>`+
		`</body></html>`, png, "")

	tiles := behaviorMiscBgImages(res.Ops)
	if len(tiles) != 4 {
		t.Fatalf("background-repeat-inline:no-repeat tiles = %d, want 4 in one column", len(tiles))
	}

	for _, tile := range tiles[1:] {
		if !near(tile.X, tiles[0].X) {
			t.Fatalf("tile x = %.4fpt, want column x %.4fpt", tile.X, tiles[0].X)
		}
	}
}

// TestBehaviorBackgroundRepeatXTilesVertically is background-repeat-x: no-repeat
// keeps one column of 4 tiles down the 40px box while the vertical axis stays
// at its repeat initial. Reference: Chrome 143.0.7499.40.
func TestBehaviorBackgroundRepeatXTilesVertically(t *testing.T) {
	t.Parallel()

	png := tinyPNG(10, 10)
	res := layoutHTMLWithImages(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:80px;height:40px;background-image:url(bg.png);`+
		`background-size:10px 10px;background-repeat-x:no-repeat"></div>`+
		`</body></html>`, png, "")

	tiles := behaviorMiscBgImages(res.Ops)
	if len(tiles) != 4 {
		t.Fatalf("background-repeat-x:no-repeat tiles = %d, want 4 in one column", len(tiles))
	}

	for _, tile := range tiles[1:] {
		if !near(tile.X, tiles[0].X) {
			t.Fatalf("tile x = %.4fpt, want column x %.4fpt", tile.X, tiles[0].X)
		}
	}
}

// TestBehaviorBackgroundRepeatYTilesHorizontally is background-repeat-y:
// no-repeat keeps one row of 8 tiles across the 80px box while the horizontal
// axis stays at its repeat initial. Reference: Chrome 143.0.7499.40.
func TestBehaviorBackgroundRepeatYTilesHorizontally(t *testing.T) {
	t.Parallel()

	png := tinyPNG(10, 10)
	res := layoutHTMLWithImages(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:80px;height:40px;background-image:url(bg.png);`+
		`background-size:10px 10px;background-repeat-y:no-repeat"></div>`+
		`</body></html>`, png, "")

	tiles := behaviorMiscBgImages(res.Ops)
	if len(tiles) != 8 {
		t.Fatalf("background-repeat-y:no-repeat tiles = %d, want 8 in one row", len(tiles))
	}

	for _, tile := range tiles[1:] {
		if !near(tile.Y, tiles[0].Y) {
			t.Fatalf("tile y = %.4fpt, want row y %.4fpt", tile.Y, tiles[0].Y)
		}
	}
}

// TestBehaviorContainSizeCollapsesToEmpty is contain:size: the block-axis size
// is the as-if-empty size, so a 200px tall child inside a size-contained box
// leaves the box 0px tall and the child overflows it.
// Reference: Chrome 143.0.7499.40 sizes the box as if empty.
func TestBehaviorContainSizeCollapsesToEmpty(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;contain:size">`+
		`<div id="tall" style="height:200px"></div>`+
		`</div></body></html>`)

	outer := boxByID(t, res, "outer")
	tall := boxByID(t, res, "tall")

	if !near(outer.height, 0) {
		t.Errorf("size-contained height = %.4fpt (%.2fpx), want 0px as-if-empty",
			outer.height, outer.height/ptPerCSSPx)
	}

	if !near(tall.height, pxToPt(200)) {
		t.Errorf("child height = %.4fpt (%.2fpx), want 200px still laid out",
			tall.height, tall.height/ptPerCSSPx)
	}
}

// TestBehaviorContainIntrinsicSizeSetsContainedHeight is contain-intrinsic-size:
// with contain:size the single 60px entry mirrors to both axes, so a 200px
// tall child leaves the box 60px tall.
// Reference: Chrome 143.0.7499.40, 60px intrinsic fallback.
func TestBehaviorContainIntrinsicSizeSetsContainedHeight(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;contain:size;contain-intrinsic-size:60px">`+
		`<div id="tall" style="height:200px"></div>`+
		`</div></body></html>`)

	behaviorMiscCheckUsedSize(t, res, "outer", 400, 60)
}

// TestBehaviorContainIntrinsicWidthSetsInlineBlockWidth is contain-intrinsic-width:
// a size-contained inline-block measures its intrinsic inline size instead of
// its descendants, so long text still lays out 100px wide.
// Reference: Chrome 143.0.7499.40, 100px intrinsic width.
func TestBehaviorContainIntrinsicWidthSetsInlineBlockWidth(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<span id="item" style="display:inline-block;contain:size;contain-intrinsic-width:100px">`+
		`long text that would be wider than one hundred pixels in a row</span>`+
		`</body></html>`)

	if w := boxByID(t, res, "item").w; !near(w, pxToPt(100)) {
		t.Errorf("intrinsic width = %.4fpt (%.2fpx), want 100px", w, w/ptPerCSSPx)
	}
}

// TestBehaviorContainIntrinsicHeightSetsContainedHeight is contain-intrinsic-height:
// with contain:size a 200px tall child leaves the box at the 60px intrinsic
// height. Reference: Chrome 143.0.7499.40.
func TestBehaviorContainIntrinsicHeightSetsContainedHeight(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;contain:size;contain-intrinsic-height:60px">`+
		`<div id="tall" style="height:200px"></div>`+
		`</div></body></html>`)

	behaviorMiscCheckUsedSize(t, res, "outer", 400, 60)
}

// TestBehaviorContainIntrinsicBlockSizeSetsContainedHeight is
// contain-intrinsic-block-size: in horizontal-tb the block axis is height, so
// a 200px tall child inside a size-contained box leaves it 70px tall.
// Reference: Chrome 143.0.7499.40, horizontal-tb mapping.
func TestBehaviorContainIntrinsicBlockSizeSetsContainedHeight(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;contain:size;contain-intrinsic-block-size:70px">`+
		`<div id="tall" style="height:200px"></div>`+
		`</div></body></html>`)

	behaviorMiscCheckUsedSize(t, res, "outer", 400, 70)
}

// TestBehaviorContainIntrinsicInlineSizeSetsInlineBlockWidth is
// contain-intrinsic-inline-size: in horizontal-tb the inline axis is width, so
// a size-contained inline-block with long text still lays out 90px wide.
// Reference: Chrome 143.0.7499.40, horizontal-tb mapping.
func TestBehaviorContainIntrinsicInlineSizeSetsInlineBlockWidth(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<span id="item" style="display:inline-block;contain:size;contain-intrinsic-inline-size:90px">`+
		`long text that would be wider than ninety pixels in a row</span>`+
		`</body></html>`)

	if w := boxByID(t, res, "item").w; !near(w, pxToPt(90)) {
		t.Errorf("intrinsic inline size = %.4fpt (%.2fpx), want 90px", w, w/ptPerCSSPx)
	}
}

// TestBehaviorContainerShorthandNoLayoutEffect documents the current behavior
// of container: without an @container query the shorthand only registers a
// size container and paints identical geometry to the unstyled document.
// Unobservable in the drawing list: reported as a no-op, never failing.
// Reference: Chrome 143.0.7499.40 registers card / inline-size the same way.
// GAP: add a matching @container query to observe a style switch.
func TestBehaviorContainerShorthandNoLayoutEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px"><div id="item" style="width:100px;height:20px"></div></div>`+
		`</body></html>`)

	withContainer := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;container:card / inline-size">`+
		`<div id="item" style="width:100px;height:20px"></div></div>`+
		`</body></html>`)

	behaviorMiscAssertSameOps(t, plain, withContainer)
	behaviorMiscCheckUsedSize(t, withContainer, "item", 100, 20)
}

// TestBehaviorContainerNameNoLayoutEffect documents the current behavior of
// container-name: a name alone never establishes a size container (see
// TestContainerQueryRequiresContainment), so it paints identical geometry.
// Unobservable in the drawing list: reported as a no-op, never failing.
// Reference: Chrome 143.0.7499.40.
// GAP: pair the name with container-type and a matching @container query.
func TestBehaviorContainerNameNoLayoutEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px"><div id="item" style="width:100px;height:20px"></div></div>`+
		`</body></html>`)

	named := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;container-name:card">`+
		`<div id="item" style="width:100px;height:20px"></div></div>`+
		`</body></html>`)

	behaviorMiscAssertSameOps(t, plain, named)
	behaviorMiscCheckUsedSize(t, named, "item", 100, 20)
}

// TestBehaviorContainerTypeNoSizeQueryEffect documents the current behavior of
// container-type: registering inline-size alone only enables future @container
// queries and leaves the used boxes untouched.
// Unobservable in the drawing list: reported as a no-op, never failing.
// Reference: Chrome 143.0.7499.40.
// GAP: add a matching @container query to observe a style switch.
func TestBehaviorContainerTypeNoSizeQueryEffect(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px"><div id="item" style="width:100px;height:20px"></div></div>`+
		`</body></html>`)

	typed := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;container-type:inline-size">`+
		`<div id="item" style="width:100px;height:20px"></div></div>`+
		`</body></html>`)

	behaviorMiscAssertSameOps(t, plain, typed)
	behaviorMiscCheckUsedSize(t, typed, "item", 100, 20)
}

// TestBehaviorMarginBlockStartOffsetsSibling is margin-block-start: in
// horizontal-tb it maps to margin-top, so a 24px block-start margin on the
// second child opens a 24px gap after the 40px first child inside a flow-root
// wrapper that keeps the margin from collapsing through the parent.
// Reference: Chrome 143.0.7499.40, 400px wrapper.
func TestBehaviorMarginBlockStartOffsetsSibling(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;display:flow-root">`+
		`<div id="first" style="height:40px"></div>`+
		`<div id="second" style="margin-block-start:24px;height:30px"></div>`+
		`</div></body></html>`)

	behaviorMiscCheckUsedSize(t, res, "first", 400, 40)
	behaviorMiscCheckUsedSize(t, res, "second", 400, 30)
	behaviorMiscCheckOffset(t, res, "second", "first", 0, 64)
}

// TestBehaviorMarginBlockEndOffsetsSibling is margin-block-end: in horizontal-tb
// it maps to margin-bottom, so a 24px block-end margin on the first child
// opens a 24px gap before the second child.
// Reference: Chrome 143.0.7499.40, 400px wrapper.
func TestBehaviorMarginBlockEndOffsetsSibling(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;display:flow-root">`+
		`<div id="first" style="margin-block-end:24px;height:40px"></div>`+
		`<div id="second" style="height:30px"></div>`+
		`</div></body></html>`)

	behaviorMiscCheckUsedSize(t, res, "first", 400, 40)
	behaviorMiscCheckOffset(t, res, "second", "first", 0, 64)
}

// TestBehaviorMarginInlineStartOffsetsSibling is margin-inline-start: in
// horizontal-tb LTR it maps to margin-left, so a 100px block with a 30px
// inline-start margin sits 30px inside the 400px wrapper.
// Reference: Chrome 143.0.7499.40, 400px wrapper.
func TestBehaviorMarginInlineStartOffsetsSibling(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;display:flow-root">`+
		`<div id="item" style="margin-inline-start:30px;width:100px;height:20px"></div>`+
		`</div></body></html>`)

	behaviorMiscCheckUsedSize(t, res, "item", 100, 20)
	behaviorMiscCheckOffset(t, res, "item", "outer", 30, 0)
}

// TestBehaviorMarginInlineEndShrinksAutoWidth is margin-inline-end: in
// horizontal-tb LTR it maps to margin-right, so an auto-width block with a
// 30px inline-end margin in a 400px wrapper lays out 370px wide flush left.
// Reference: Chrome 143.0.7499.40, 400px wrapper.
func TestBehaviorMarginInlineEndShrinksAutoWidth(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;display:flow-root">`+
		`<div id="item" style="margin-inline-end:30px;height:20px"></div>`+
		`</div></body></html>`)

	behaviorMiscCheckUsedSize(t, res, "item", 370, 20)
	behaviorMiscCheckOffset(t, res, "item", "outer", 0, 0)
}

// TestBehaviorMarginTrimTrimsFirstChild is margin-trim:block: the 30px top
// margin of the first child and the 30px bottom margin of the last child are
// zeroed at the container edges, so the first box sits flush and the wrapper
// is 70px tall instead of 100px.
// Reference: Chrome 143.0.7499.40, 400px flow-root wrapper.
func TestBehaviorMarginTrimTrimsFirstChild(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;display:flow-root;margin-trim:block">`+
		`<div id="first" style="margin-top:30px;height:40px"></div>`+
		`<div id="second" style="margin-bottom:30px;height:30px"></div>`+
		`</div></body></html>`)

	behaviorMiscCheckOffset(t, res, "first", "outer", 0, 0)
	behaviorMiscCheckOffset(t, res, "second", "first", 0, 40)
	behaviorMiscCheckUsedSize(t, res, "outer", 400, 70)
}

// TestBehaviorPaddingBlockStartOffsetsContent is padding-block-start: in
// horizontal-tb it maps to padding-top, so 20px of block-start padding puts
// the child 20px below the wrapper border edge.
// Reference: Chrome 143.0.7499.40, 400px wrapper.
func TestBehaviorPaddingBlockStartOffsetsContent(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;padding-block-start:20px;height:120px">`+
		`<div id="inner" style="width:100px;height:20px"></div>`+
		`</div></body></html>`)

	behaviorMiscCheckUsedSize(t, res, "outer", 400, 140)
	behaviorMiscCheckOffset(t, res, "inner", "outer", 0, 20)
}

// TestBehaviorPaddingBlockEndExpandsBox is padding-block-end: in horizontal-tb
// it maps to padding-bottom, so a 100px tall wrapper with 20px of block-end
// padding lays out 120px tall with the child still flush at the top.
// Reference: Chrome 143.0.7499.40.
func TestBehaviorPaddingBlockEndExpandsBox(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;padding-block-end:20px;height:100px">`+
		`<div id="inner" style="width:100px;height:20px"></div>`+
		`</div></body></html>`)

	behaviorMiscCheckUsedSize(t, res, "outer", 400, 120)
	behaviorMiscCheckOffset(t, res, "inner", "outer", 0, 0)
}

// TestBehaviorPaddingInlineStartOffsetsContent is padding-inline-start: in
// horizontal-tb LTR it maps to padding-left, so 20px of inline-start padding
// puts the child 20px inside the 400px content-box wrapper, whose border box
// grows to 420px. Reference: Chrome 143.0.7499.40.
func TestBehaviorPaddingInlineStartOffsetsContent(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;padding-inline-start:20px;height:120px">`+
		`<div id="inner" style="width:100px;height:20px"></div>`+
		`</div></body></html>`)

	behaviorMiscCheckUsedSize(t, res, "outer", 420, 120)
	behaviorMiscCheckOffset(t, res, "inner", "outer", 20, 0)
}

// TestBehaviorPaddingInlineEndExpandsBox is padding-inline-end: in
// horizontal-tb LTR it maps to padding-right, so a 400px content-box wrapper
// with 20px of inline-end padding lays out 420px wide with the child flush
// left. Reference: Chrome 143.0.7499.40.
func TestBehaviorPaddingInlineEndExpandsBox(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px;padding-inline-end:20px;height:120px">`+
		`<div id="inner" style="width:100px;height:20px"></div>`+
		`</div></body></html>`)

	behaviorMiscCheckUsedSize(t, res, "outer", 420, 120)
	behaviorMiscCheckOffset(t, res, "inner", "outer", 0, 0)
}

// TestBehaviorFloatOffsetNudgesFloat is float-offset: a 20px offset nudges a
// left float 20px down the block axis from where it would otherwise sit.
// Reference: Chrome 143.0.7499.40, 400px wrapper, 100px by 40px float.
func TestBehaviorFloatOffsetNudgesFloat(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px">`+
		`<div id="plain" style="float:left;width:100px;height:40px"></div>`+
		`</div></body></html>`)

	nudged := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:400px">`+
		`<div id="plain" style="float:left;width:100px;height:40px;float-offset:20px"></div>`+
		`</div></body></html>`)

	plainFloat := boxByID(t, plain, "plain")
	nudgedFloat := boxByID(t, nudged, "plain")

	if !near(nudgedFloat.y-plainFloat.y, pxToPt(20)) {
		t.Errorf("float-offset dy = %.4fpt (%.2fpx), want 20px down",
			nudgedFloat.y-plainFloat.y, (nudgedFloat.y-plainFloat.y)/ptPerCSSPx)
	}

	if !near(nudgedFloat.x, plainFloat.x) {
		t.Errorf("float-offset x = %.4fpt, want unchanged %.4fpt", nudgedFloat.x, plainFloat.x)
	}
}

// TestBehaviorFloatReferencePinsToPage is float-reference:page: a right float
// inside a 200px wrapper packs against the page content box instead of the
// wrapper, so it sits at the viewport right edge rather than the wrapper
// right edge. Reference: Chrome 143.0.7499.40, 500pt viewport.
func TestBehaviorFloatReferencePinsToPage(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="outer" style="width:200px">`+
		`<div id="fright" style="float:right;width:50px;height:20px;float-reference:page"></div>`+
		`</div></body></html>`)

	outer := boxByID(t, res, "outer")
	floated := boxByID(t, res, "fright")

	// Viewport is testViewport (500pt); page packing puts the 50px float at
	// 500pt minus 37.5pt, while wrapper packing would keep it at 150pt minus
	// 37.5pt. The used x must match the page edge.
	if !near(floated.x, 500-pxToPt(50)) {
		t.Errorf("page float x = %.4fpt (%.2fpx), want viewport right %.2fpx",
			floated.x, floated.x/ptPerCSSPx, (500-pxToPt(50))/ptPerCSSPx)
	}

	if near(floated.x, outer.x+outer.w-floated.w) {
		t.Errorf("page float x %.4fpt matches wrapper packing, want page packing", floated.x)
	}
}

// TestBehaviorGridAreaPlacesNamedItem is grid-area: the named area b places the
// item in the second 100px track at x 100.
// Reference: Chrome 143.0.7499.40, tracks 100px plus 100px, areas 'a b'.
func TestBehaviorGridAreaPlacesNamedItem(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="g" style="display:grid;width:200px;grid-template-columns:100px 100px;grid-template-areas:'a b'">`+
		`<div id="item" style="grid-area:b;height:40px"></div>`+
		`</div></body></html>`)

	behaviorMiscCheckOffset(t, res, "item", "g", 100, 0)
	behaviorMiscCheckUsedSize(t, res, "item", 100, 40)
}

// TestBehaviorColumnHeightOpensSecondRow is column-height: with a definite
// 36pt height and auto wrap, content past one row opens a second multicol row
// instead of stretching a single tall column.
// Reference: Chrome 143.0.7499.40, 200px container, 2 columns.
func TestBehaviorColumnHeightOpensSecondRow(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="mc" style="width:200px;column-count:2;column-gap:8px;column-height:36pt;`+
		`column-fill:auto;font-size:9pt">`+
		`<p style="margin:0 0 1pt 0">Alpha one.</p><p style="margin:0 0 1pt 0">Bravo two.</p>`+
		`<p style="margin:0 0 1pt 0">Charlie three.</p><p style="margin:0 0 1pt 0">Delta four.</p>`+
		`<p style="margin:0 0 1pt 0">Echo five.</p><p style="margin:0 0 1pt 0">Foxtrot six.</p>`+
		`<p style="margin:0 0 1pt 0">Golf seven.</p><p style="margin:0 0 1pt 0">Hotel eight.</p>`+
		`<p style="margin:0 0 1pt 0">India nine.</p><p style="margin:0 0 1pt 0">Juliet ten.</p>`+
		`</div></body></html>`)

	minY, maxY, n := behaviorMiscLabelYSpan(t, res)
	if n < 6 {
		t.Fatalf("column-height produced %d labels, want at least 6", n)
	}

	if maxY-minY < 30 {
		t.Fatalf("column-height Y span = %.4fpt (%.2fpx), want a second row past 36pt",
			maxY-minY, (maxY-minY)/ptPerCSSPx)
	}
}

// TestBehaviorColumnWrapNowrapKeepsSingleRow is column-wrap:nowrap: with a
// definite 36pt height, nowrap keeps one row only, so all labels stay inside
// a single 36pt band instead of opening a second block-direction row.
// Reference: Chrome 143.0.7499.40.
func TestBehaviorColumnWrapNowrapKeepsSingleRow(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="mc" style="width:200px;column-count:2;column-gap:8px;column-height:36pt;`+
		`column-wrap:nowrap;column-fill:auto;font-size:9pt">`+
		`<p style="margin:0 0 1pt 0">Alpha one.</p><p style="margin:0 0 1pt 0">Bravo two.</p>`+
		`<p style="margin:0 0 1pt 0">Charlie three.</p><p style="margin:0 0 1pt 0">Delta four.</p>`+
		`<p style="margin:0 0 1pt 0">Echo five.</p><p style="margin:0 0 1pt 0">Foxtrot six.</p>`+
		`<p style="margin:0 0 1pt 0">Golf seven.</p><p style="margin:0 0 1pt 0">Hotel eight.</p>`+
		`<p style="margin:0 0 1pt 0">India nine.</p><p style="margin:0 0 1pt 0">Juliet ten.</p>`+
		`</div></body></html>`)

	minY, maxY, n := behaviorMiscLabelYSpan(t, res)
	if n < 2 {
		t.Fatalf("column-wrap:nowrap produced %d labels, want at least 2", n)
	}

	if maxY-minY > 40 {
		t.Fatalf("column-wrap:nowrap Y span = %.4fpt (%.2fpx), want one 36pt row",
			maxY-minY, (maxY-minY)/ptPerCSSPx)
	}
}
