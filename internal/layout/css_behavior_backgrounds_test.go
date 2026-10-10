package layout

import "testing"

// Background and border-image behavior tests. Every property below is
// asserted through an emitted op or a used value (a measured border-box edge
// or an op field), never through a stored style string. Lengths are written
// in CSS px and converted with pxToPt (1px = 0.75pt, ptPerCSSPx in
// css_review_02_test.go), which is how Chrome reports them. Helpers boxByID,
// near, pxToPt, opsOfKind, layoutHTMLWithImages, and tinyPNG come from
// css_review_02_test.go and layout_test.go in this same package.
// Reference browser for every case: Chrome 143.0.7499.40.

// behaviorBgImages keeps the non-empty background image ops.
func behaviorBgImages(ops []Op) []Op {
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

// behaviorBgFirstImage returns the first background image op, or nil.
func behaviorBgFirstImage(ops []Op) *Op {
	for _, paintOp := range ops {
		if paintOp.Kind == OpImage && paintOp.IsBackground && paintOp.W > 0 && paintOp.H > 0 {
			candidate := paintOp

			return &candidate
		}
	}

	return nil
}

// behaviorBgCorner returns the op with the smallest X among those with the
// smallest Y, which is the top-left cell of a border-image frame.
func behaviorBgCorner(cells []Op) *Op {
	var best *Op

	for _, paintOp := range cells {
		candidate := paintOp

		if best == nil || candidate.Y < best.Y ||
			(candidate.Y == best.Y && candidate.X < best.X) {
			best = &candidate
		}
	}

	return best
}

// behaviorBgRunPositionCase lays out two documents differing only in one
// background-position longhand and asserts the image sits at the origin edge
// for the low value and the far edge for the high value. axis is "x" or "y".
func behaviorBgRunPositionCase(t *testing.T, prop, lowVal, highVal, axis string) {
	t.Helper()

	png := tinyPNG(10, 10)
	htmlSrc := func(pos string) string {
		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="item" style="width:80px;height:40px;background-image:url(bg.png);` +
			`background-repeat:no-repeat;` + prop + `:` + pos + `"></div>` +
			`</body></html>`
	}

	low := layoutHTMLWithImages(t, htmlSrc(lowVal), png, "")
	high := layoutHTMLWithImages(t, htmlSrc(highVal), png, "")

	lowBox := boxByID(t, low, "item")
	highBox := boxByID(t, high, "item")
	lowImg := behaviorBgFirstImage(low.Ops)
	highImg := behaviorBgFirstImage(high.Ops)

	if lowImg == nil || highImg == nil {
		t.Fatalf("background images = %s %v %s %v, want one each",
			lowVal, lowImg != nil, highVal, highImg != nil)
	}

	lowPos, lowOrigin := lowImg.X, lowBox.x
	highPos, highOrigin, highSize, highBoxSize := highImg.X, highBox.x, highImg.W, highBox.w

	if axis == "y" {
		lowPos, lowOrigin = lowImg.Y, lowBox.y
		highPos, highOrigin, highSize, highBoxSize = highImg.Y, highBox.y, highImg.H, highBox.height
	}

	if !near(lowPos, lowOrigin) {
		t.Errorf("%s image at %.4fpt, want box origin %.4fpt", lowVal, lowPos, lowOrigin)
	}

	want := highOrigin + highBoxSize - highSize
	if !near(highPos, want) {
		t.Errorf("%s image at %.4fpt (%.2fpx), want %.2fpx",
			highVal, highPos, highPos/ptPerCSSPx, want/ptPerCSSPx)
	}
}

// TestBehaviorBackgroundPositionXOffsetsImage is background-position-x: right
// slides a no-repeat background image to the right edge of the positioning
// area while left keeps it at the origin. Reference: Chrome 143.0.7499.40,
// 80px by 40px box, 10px by 10px image.
func TestBehaviorBackgroundPositionXOffsetsImage(t *testing.T) {
	t.Parallel()

	behaviorBgRunPositionCase(t, "background-position-x", "left", "right", "x")
}

// TestBehaviorBackgroundPositionYOffsetsImage is background-position-y:
// bottom slides a no-repeat background image to the bottom edge of the
// positioning area while top keeps it at the origin. Reference: Chrome
// 143.0.7499.40, 80px by 40px box, 10px by 10px image.
func TestBehaviorBackgroundPositionYOffsetsImage(t *testing.T) {
	t.Parallel()

	behaviorBgRunPositionCase(t, "background-position-y", "top", "bottom", "y")
}

// TestBehaviorBackgroundSizeScalesImage is background-size: an explicit
// 40px by 20px size scales the painted image op to that size while the
// intrinsic payload stays 10px by 10px. Reference: Chrome 143.0.7499.40.
func TestBehaviorBackgroundSizeScalesImage(t *testing.T) {
	t.Parallel()

	png := tinyPNG(10, 10)
	res := layoutHTMLWithImages(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:80px;height:40px;background-image:url(bg.png);`+
		`background-repeat:no-repeat;background-position:left top;background-size:40px 20px"></div>`+
		`</body></html>`, png, "")

	img := behaviorBgFirstImage(res.Ops)
	if img == nil {
		t.Fatal("no painted background image op")
	}

	if !near(img.W, pxToPt(40)) || !near(img.H, pxToPt(20)) {
		t.Errorf("painted size = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want 40px x 20px",
			img.W, img.H, img.W/ptPerCSSPx, img.H/ptPerCSSPx)
	}

	if img.ImgW != 10 || img.ImgH != 10 {
		t.Errorf("intrinsic payload = %dx%d, want 10x10", img.ImgW, img.ImgH)
	}
}

// TestBehaviorBackgroundRepeatTilesImage is background-repeat: repeat tiles
// the 10px image across the 80px by 40px box (8 by 4 tiles) while no-repeat
// paints one tile. Reference: Chrome 143.0.7499.40.
func TestBehaviorBackgroundRepeatTilesImage(t *testing.T) {
	t.Parallel()

	png := tinyPNG(10, 10)
	htmlSrc := func(repeat string) string {
		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="item" style="width:80px;height:40px;background-image:url(bg.png);` +
			`background-repeat:` + repeat + `"></div>` +
			`</body></html>`
	}

	single := behaviorBgImages(layoutHTMLWithImages(t, htmlSrc("no-repeat"), png, "").Ops)
	tiled := behaviorBgImages(layoutHTMLWithImages(t, htmlSrc("repeat"), png, "").Ops)

	if len(single) != 1 {
		t.Fatalf("no-repeat tiles = %d, want 1", len(single))
	}

	if len(tiled) != 8*4 {
		t.Fatalf("repeat tiles = %d, want 32", len(tiled))
	}

	for _, tile := range tiled {
		if !near(tile.W, single[0].W) || !near(tile.H, single[0].H) {
			t.Fatalf("tile size = %.4fpt x %.4fpt, want %.4fpt x %.4fpt",
				tile.W, tile.H, single[0].W, single[0].H)
		}
	}
}

// TestBehaviorBackgroundOriginInsetsImage is background-origin: with a 10px
// border, padding-box starts the image 10px inside the border box while
// border-box starts it at the border-box origin. Reference: Chrome
// 143.0.7499.40, 80px by 40px content, 10px by 10px image.
func TestBehaviorBackgroundOriginInsetsImage(t *testing.T) {
	t.Parallel()

	png := tinyPNG(10, 10)
	htmlSrc := func(origin string) string {
		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="item" style="width:80px;height:40px;border:10px solid #000;` +
			`background-image:url(bg.png);background-repeat:no-repeat;` +
			`background-position:left top;background-origin:` + origin + `"></div>` +
			`</body></html>`
	}

	borderRes := layoutHTMLWithImages(t, htmlSrc("border-box"), png, "")
	paddingRes := layoutHTMLWithImages(t, htmlSrc("padding-box"), png, "")

	borderBox := boxByID(t, borderRes, "item")
	borderImg := behaviorBgFirstImage(borderRes.Ops)
	paddingImg := behaviorBgFirstImage(paddingRes.Ops)

	if borderImg == nil || paddingImg == nil {
		t.Fatalf("background images = border %v padding %v, want one each",
			borderImg != nil, paddingImg != nil)
	}

	if !near(borderImg.X, borderBox.x) || !near(borderImg.Y, borderBox.y) {
		t.Errorf("border-box origin at (%.4fpt, %.4fpt), want box origin (%.4fpt, %.4fpt)",
			borderImg.X, borderImg.Y, borderBox.x, borderBox.y)
	}

	if !near(paddingImg.X, borderBox.x+pxToPt(10)) || !near(paddingImg.Y, borderBox.y+pxToPt(10)) {
		t.Errorf("padding-box origin at (%.4fpt, %.4fpt), want 10px inside (%.4fpt, %.4fpt)",
			paddingImg.X, paddingImg.Y, borderBox.x+pxToPt(10), borderBox.y+pxToPt(10))
	}
}

// TestBehaviorBackgroundClipClipsImage is background-clip: a 100px by 60px
// image painted over a padded box keeps its full size with border-box but is
// cut to the 80px by 40px content box with content-box. Reference: Chrome
// 143.0.7499.40, 80px by 40px content with 10px padding.
func TestBehaviorBackgroundClipClipsImage(t *testing.T) {
	t.Parallel()

	png := tinyPNG(10, 10)
	htmlSrc := func(clip string) string {
		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="item" style="width:80px;height:40px;padding:10px;` +
			`background-image:url(bg.png);background-repeat:no-repeat;` +
			`background-position:left top;background-size:100px 60px;background-clip:` + clip + `"></div>` +
			`</body></html>`
	}

	borderRes := layoutHTMLWithImages(t, htmlSrc("border-box"), png, "")
	contentRes := layoutHTMLWithImages(t, htmlSrc("content-box"), png, "")

	borderBox := boxByID(t, borderRes, "item")
	borderImg := behaviorBgFirstImage(borderRes.Ops)
	contentImg := behaviorBgFirstImage(contentRes.Ops)

	if borderImg == nil || contentImg == nil {
		t.Fatalf("background images = border %v content %v, want one each",
			borderImg != nil, contentImg != nil)
	}

	if !near(borderImg.W, pxToPt(100)) || !near(borderImg.H, pxToPt(60)) {
		t.Errorf("border-box clip size = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want 100px x 60px",
			borderImg.W, borderImg.H, borderImg.W/ptPerCSSPx, borderImg.H/ptPerCSSPx)
	}

	if !near(contentImg.X, borderBox.x+pxToPt(10)) || !near(contentImg.Y, borderBox.y+pxToPt(10)) {
		t.Errorf("content-box clip origin = (%.4fpt, %.4fpt), want 10px inside the border box",
			contentImg.X, contentImg.Y)
	}

	if !near(contentImg.W, pxToPt(80)) || !near(contentImg.H, pxToPt(40)) {
		t.Errorf("content-box clip size = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want 80px x 40px",
			contentImg.W, contentImg.H, contentImg.W/ptPerCSSPx, contentImg.H/ptPerCSSPx)
	}
}

// TestBehaviorBorderBottomWidthStrokesEdge is border-bottom-width: a 6px
// solid bottom edge paints one horizontal line op 6px wide on the used
// bottom edge. Reference: Chrome 143.0.7499.40.
func TestBehaviorBorderBottomWidthStrokesEdge(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:200px;height:40px;`+
		`border-bottom-width:6px;border-bottom-style:solid"></div>`+
		`</body></html>`)

	item := boxByID(t, res, "item")

	found := false

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpLine || paintOp.H != 0 || paintOp.W <= 0 {
			continue
		}

		if !near(paintOp.Y, item.y+item.height) {
			continue
		}

		found = true

		if !near(paintOp.Width, pxToPt(6)) {
			t.Errorf("bottom edge width = %.4fpt (%.2fpx), want 6px",
				paintOp.Width, paintOp.Width/ptPerCSSPx)
		}
	}

	if !found {
		t.Fatalf("no bottom border op at y %.4fpt", item.y+item.height)
	}
}

// TestBehaviorBorderRightWidthStrokesEdge is border-right-width: a 4px solid
// right edge paints one vertical line op 4px wide on the used right edge.
// Reference: Chrome 143.0.7499.40.
func TestBehaviorBorderRightWidthStrokesEdge(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="item" style="width:200px;height:40px;`+
		`border-right-width:4px;border-right-style:solid"></div>`+
		`</body></html>`)

	item := boxByID(t, res, "item")

	found := false

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpLine || paintOp.W != 0 || paintOp.H <= 0 {
			continue
		}

		if !near(paintOp.X, item.x+item.w) {
			continue
		}

		found = true

		if !near(paintOp.Width, pxToPt(4)) {
			t.Errorf("right edge width = %.4fpt (%.2fpx), want 4px",
				paintOp.Width, paintOp.Width/ptPerCSSPx)
		}
	}

	if !found {
		t.Fatalf("no right border op at x %.4fpt", item.x+item.w)
	}
}

// TestBehaviorBorderImageSourcePaintsSlices is border-image-source: with no
// slice the whole image paints once over the used border box. Reference:
// Chrome 143.0.7499.40, 100px by 60px content with a 10px border.
func TestBehaviorBorderImageSourcePaintsSlices(t *testing.T) {
	t.Parallel()

	img := tinyPNG(12, 12)
	res := layoutHTMLWithImages(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="bi" style="width:100px;height:60px;border:10px solid #000;`+
		`border-image-source:url(border.png)"></div>`+
		`</body></html>`, img, "border.png")

	borderBox := boxByID(t, res, "bi")
	cells := behaviorBgImages(opsOfKind(res, OpImage))

	if len(cells) != 1 {
		t.Fatalf("border-image ops = %d, want 1 unsliced paint", len(cells))
	}

	if !near(cells[0].X, borderBox.x) || !near(cells[0].Y, borderBox.y) ||
		!near(cells[0].W, borderBox.w) || !near(cells[0].H, borderBox.height) {
		t.Errorf("border-image rect = (%.4fpt, %.4fpt) %.4fpt x %.4fpt, want the used border box",
			cells[0].X, cells[0].Y, cells[0].W, cells[0].H)
	}
}

// TestBehaviorBorderImageSliceSelectsGeometry is border-image-slice: slice 3
// cuts 3px corners off the 12px image while slice 4 cuts 4px corners, so the
// top-left cell payload differs while its painted size (the 10px border
// thickness) stays put. Reference: Chrome 143.0.7499.40.
func TestBehaviorBorderImageSliceSelectsGeometry(t *testing.T) {
	t.Parallel()

	img := tinyPNG(12, 12)
	htmlSrc := func(slice string) string {
		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="bi" style="width:100px;height:60px;border:10px solid #000;` +
			`border-image-source:url(border.png);border-image-slice:` + slice + `"></div>` +
			`</body></html>`
	}

	narrow := behaviorBgImages(opsOfKind(layoutHTMLWithImages(t, htmlSrc("3"), img, "border.png"), OpImage))
	wide := behaviorBgImages(opsOfKind(layoutHTMLWithImages(t, htmlSrc("4"), img, "border.png"), OpImage))

	behaviorBgAssertSliceFrame(t, narrow, 3, "slice 3")
	behaviorBgAssertSliceFrame(t, wide, 4, "slice 4")
}

// behaviorBgAssertSliceFrame asserts a border-image frame holds 8 cells whose
// top-left corner carries a wantSlice payload at 10px painted width.
func behaviorBgAssertSliceFrame(t *testing.T, cells []Op, wantSlice int, tag string) {
	t.Helper()

	if len(cells) != 8 {
		t.Fatalf("%s border-image cells = %d, want 8", tag, len(cells))
	}

	corner := behaviorBgCorner(cells)

	if corner == nil {
		t.Fatal("no top-left corner cell in one frame")
	}

	if corner.ImgW != wantSlice || corner.ImgH != wantSlice {
		t.Errorf("%s corner payload = %dx%d, want %dx%d",
			tag, corner.ImgW, corner.ImgH, wantSlice, wantSlice)
	}

	if !near(corner.W, pxToPt(10)) {
		t.Errorf("%s corner painted width = %.4fpt, want 10px", tag, corner.W)
	}
}

// TestBehaviorBorderImageWidthThickensFrame is border-image-width: 5pt
// paints 5pt corners while 10pt paints 10pt corners. Reference: Chrome
// 143.0.7499.40, 100px by 60px box with a 10px border.
func TestBehaviorBorderImageWidthThickensFrame(t *testing.T) {
	t.Parallel()

	img := tinyPNG(8, 8)
	htmlSrc := func(width string) string {
		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="bi" style="width:100px;height:60px;border:10px solid #000;` +
			`border-image-source:url(border.png);border-image-slice:2;border-image-width:` + width + `"></div>` +
			`</body></html>`
	}

	thin := behaviorBgImages(opsOfKind(layoutHTMLWithImages(t, htmlSrc("5pt"), img, "border.png"), OpImage))
	thick := behaviorBgImages(opsOfKind(layoutHTMLWithImages(t, htmlSrc("10pt"), img, "border.png"), OpImage))

	if len(thin) != 8 || len(thick) != 8 {
		t.Fatalf("border-image cells = thin %d thick %d, want 8 each", len(thin), len(thick))
	}

	thinCorner := behaviorBgCorner(thin)
	thickCorner := behaviorBgCorner(thick)

	if thinCorner == nil || thickCorner == nil {
		t.Fatal("no top-left corner cell in one frame")
	}

	if !near(thinCorner.W, 5) || !near(thinCorner.H, 5) {
		t.Errorf("width 5pt corner = %.4fpt x %.4fpt, want 5 x 5", thinCorner.W, thinCorner.H)
	}

	if !near(thickCorner.W, 10) || !near(thickCorner.H, 10) {
		t.Errorf("width 10pt corner = %.4fpt x %.4fpt, want 10 x 10", thickCorner.W, thickCorner.H)
	}
}
