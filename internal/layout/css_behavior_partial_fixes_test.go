package layout

import (
	"math"
	"testing"
)

func TestBorderImageGradientSourceAndShorthand(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		style string
	}{
		{
			name:  "shorthand linear-gradient",
			style: "border:12px solid #ccc;border-image:linear-gradient(#c0392b, #2980b9) 10",
		},
		{
			name:  "longhand border-image-source linear-gradient",
			style: "border:12px solid #ccc;border-image-slice:10;border-image-source:linear-gradient(#c0392b, #2980b9)",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			html := `<html><body><div style="width:120px;height:60px;` + tc.style + `"></div></body></html>`
			doc := layoutHTML(t, html)
			imgOps := 0

			for _, op := range doc.Ops {
				if data, _, _ := op.ImageBytes(); op.Kind == OpImage && len(data) > 0 {
					imgOps++
				}
			}

			if imgOps != 8 {
				t.Fatalf("%s: expected 8 gradient border-image OpImage ops, got %d (total ops=%d)", tc.name, imgOps, len(doc.Ops))
			}
		})
	}
}

func TestVisibilityHiddenSuppressesBoxChrome(t *testing.T) {
	t.Parallel()

	html := `<html><body><div style="width:150px;height:30px;background:#c96;border:2px solid #333;visibility:hidden"></div></body></html>`
	doc := layoutHTML(t, html)

	for _, op := range doc.Ops {
		if op.Kind == OpFillRect || op.Kind == OpStrokeRect || op.Kind == OpLine {
			t.Fatalf("visibility:hidden painted chrome op %+v", op)
		}
	}
}

func TestGapTwoValueShorthand(t *testing.T) {
	t.Parallel()

	var sty ResolvedStyle
	ctx := &styleContext{viewportW: 400}
	if !applyGapProps(&sty, "gap", "8px 16px", 12, ctx) {
		t.Fatal("applyGapProps did not handle gap")
	}

	if math.Abs(sty.flex().RowGap-6) > 0.05 {
		t.Errorf("RowGap = %v pt, want ~6 pt (8px)", sty.flex().RowGap)
	}

	if math.Abs(sty.flex().ColumnGap-12) > 0.05 {
		t.Errorf("ColumnGap = %v pt, want ~12 pt (16px)", sty.flex().ColumnGap)
	}
}

func TestApplyScalePercentage(t *testing.T) {
	t.Parallel()

	var sty ResolvedStyle
	sty.editEffects().Transform = Scale(1, 1)
	if !ApplyScale(&sty, "40%") {
		t.Fatal("ApplyScale rejected percentage value 40%")
	}

	if !sty.effects().HasTransform {
		t.Fatal("HasTransform = false, want true")
	}

	m := sty.effects().Transform
	if math.Abs(m.A-0.4) > 1e-6 || math.Abs(m.D-0.4) > 1e-6 {
		t.Errorf("Transform scale = (%v, %v), want (0.4, 0.4)", m.A, m.D)
	}
}

func TestBorderAndOutlineColorInherit(t *testing.T) {
	t.Parallel()

	var parent ResolvedStyle
	parent.editBorders().BorderTop.Color = [3]float64{1, 0, 0}
	parent.editBorders().BorderRight.Color = [3]float64{0, 1, 0}
	parent.editBorders().BorderBottom.Color = [3]float64{0, 0, 1}
	parent.editBorders().BorderLeft.Color = [3]float64{1, 1, 0}
	parent.editBorders().OutlineColor = [3]float64{1, 0, 1}
	parent.editBorders().OutlineColorSet = true

	var child ResolvedStyle
	for _, prop := range []string{
		"border-top-color", "border-right-color", "border-bottom-color", "border-left-color", "outline-color",
	} {
		if !applyBorderGroup(&child, prop, "inherit", 12, &styleContext{}, &parent, true) {
			t.Fatalf("applyBorderGroup did not handle %s", prop)
		}
	}

	if child.borders().BorderTop.Color != parent.borders().BorderTop.Color {
		t.Errorf("BorderTop.Color = %v, want %v", child.borders().BorderTop.Color, parent.borders().BorderTop.Color)
	}
	if child.borders().BorderRight.Color != parent.borders().BorderRight.Color {
		t.Errorf("BorderRight.Color = %v, want %v", child.borders().BorderRight.Color, parent.borders().BorderRight.Color)
	}
	if child.borders().BorderBottom.Color != parent.borders().BorderBottom.Color {
		t.Errorf("BorderBottom.Color = %v, want %v", child.borders().BorderBottom.Color, parent.borders().BorderBottom.Color)
	}
	if child.borders().BorderLeft.Color != parent.borders().BorderLeft.Color {
		t.Errorf("BorderLeft.Color = %v, want %v", child.borders().BorderLeft.Color, parent.borders().BorderLeft.Color)
	}
	if !child.borders().OutlineColorSet || child.borders().OutlineColor != parent.borders().OutlineColor {
		t.Errorf("OutlineColor = (%v, set=%v), want (%v, true)", child.borders().OutlineColor, child.borders().OutlineColorSet, parent.borders().OutlineColor)
	}
}

func TestGridColumnFlowAndAutoColumns(t *testing.T) {
	t.Parallel()

	html := `<html><body style="margin:0"><div style="display:grid;width:280px;grid-auto-flow:column;grid-auto-columns:40px">` +
		`<div id="a" style="height:16px"></div><div id="b" style="height:16px"></div>` +
		`</div></body></html>`
	doc := layoutHTML(t, html)
	a := boxByID(t, doc, "a")
	b := boxByID(t, doc, "b")

	if !near(a.w, pxToPt(40)) {
		t.Errorf("a.w = %.2fpt, want 40px (%.2fpt)", a.w, pxToPt(40))
	}

	if !near(b.x, a.x+pxToPt(40)) || !near(b.y, a.y) {
		t.Errorf("b at (%.2f, %.2f), want right of a (%.2f, %.2f)", b.x, b.y, a.x+pxToPt(40), a.y)
	}
}

func TestGridRowPinnedLeavesAutoCursorAtRowZero(t *testing.T) {
	t.Parallel()

	html := `<html><body style="margin:0"><div style="display:grid;width:300px;grid-template-columns:1fr 1fr 1fr">` +
		`<div id="a" style="height:16px;grid-row-start:2"></div>` +
		`<div id="b" style="height:16px"></div>` +
		`</div></body></html>`
	doc := layoutHTML(t, html)
	a := boxByID(t, doc, "a")
	b := boxByID(t, doc, "b")

	if b.y >= a.y {
		t.Errorf("auto-placed b.y=%.2f should be above row-2 pinned a.y=%.2f", b.y, a.y)
	}
}

func TestColorSchemeDarkFormControls(t *testing.T) {
	t.Parallel()

	lightDoc := layoutHTML(t, `<html><body style="margin:0"><button id="b">Go</button><input id="i" type="text" value="hi"></body></html>`)
	darkDoc := layoutHTML(t, `<html><body style="margin:0"><button id="b" style="color-scheme:dark">Go</button><input id="i" type="text" value="hi" style="color-scheme:dark"></body></html>`)
	overrideDoc := layoutHTML(t, `<html><body style="margin:0"><button id="b" style="color-scheme:dark;background-color:#ff0000;color:#00ff00">Go</button></body></html>`)

	lightBtn := boxByID(t, lightDoc, "b").style
	darkBtn := boxByID(t, darkDoc, "b").style
	overrideBtn := boxByID(t, overrideDoc, "b").style

	if !near(lightBtn.BGColor[0], 0xef/255.0) {
		t.Fatalf("light button bg = %v, want #efefef", lightBtn.BGColor)
	}
	if !near(darkBtn.BGColor[0], 0x3b/255.0) || darkBtn.BGColor[3] != 1 {
		t.Fatalf("dark button bg = %v, want opaque #3b3b3b", darkBtn.BGColor)
	}
	if !near(darkBtn.Color[0], 0xe8/255.0) {
		t.Fatalf("dark button color = %v, want #e8e8e8", darkBtn.Color)
	}
	if !near(overrideBtn.BGColor[0], 1) || !near(overrideBtn.BGColor[1], 0) {
		t.Fatalf("author override button bg = %v, want #ff0000", overrideBtn.BGColor)
	}
	if !near(overrideBtn.Color[1], 1) {
		t.Fatalf("author override button color = %v, want #00ff00", overrideBtn.Color)
	}

	lightInp := boxByID(t, lightDoc, "i").style
	darkInp := boxByID(t, darkDoc, "i").style
	if lightInp.BGColor[3] != 0 {
		t.Fatalf("light input bg = %v, want transparent", lightInp.BGColor)
	}
	if !near(darkInp.BGColor[0], 0x2b/255.0) || darkInp.BGColor[3] != 1 {
		t.Fatalf("dark input bg = %v, want opaque #2b2b2b", darkInp.BGColor)
	}
	if !near(darkInp.Color[0], 0xe8/255.0) {
		t.Fatalf("dark input color = %v, want #e8e8e8", darkInp.Color)
	}
}

func TestTransformBoxBorderBoxVsContentBox(t *testing.T) {
	t.Parallel()

	doc := func(tb string) *Result {
		return layoutHTML(t, `<html><body style="margin:0">`+
			`<div id="t" style="width:40px;height:20px;padding:12px;border:6px solid #333;`+
			`background:#3b82f6;transform:rotate(30deg);transform-origin:0 0;transform-box:`+tb+`"></div>`+
			`</body></html>`)
	}

	cb := doc("content-box")
	bb := doc("border-box")

	findXform := func(res *Result) Matrix2D {
		for _, op := range res.Ops {
			if op.Kind == OpFillRect && op.XformSet {
				return op.Transform()
			}
		}
		t.Fatal("no transformed OpFillRect found")
		return IdentityMatrix()
	}

	mCB := findXform(cb)
	mBB := findXform(bb)
	if near(mCB.E, mBB.E) && near(mCB.F, mBB.F) {
		t.Fatalf("content-box (%v,%v) and border-box (%v,%v) baked identical origins", mCB.E, mCB.F, mBB.E, mBB.F)
	}
}

func TestPositionVisibilityAnchorValidOverridesNoOverflow(t *testing.T) {
	t.Parallel()

	doc := func(vis string) *Result {
		return layoutHTML(t, `<html><body style="margin:0">`+
			`<div style="position:relative;width:120px;height:60px;border:1px solid #ccc">`+
			`<div style="position:absolute;top:4px;left:20px;width:40px;height:16px;anchor-name:--a;background:#ddd">A</div>`+
			`<div id="tip" style="position:absolute;position-anchor:--a;position-area:top;width:40px;height:20px;background:#ef4444;position-visibility:`+vis+`">T</div>`+
			`</div></body></html>`)
	}

	hidden := doc("no-overflow")
	shown := doc("anchor-valid")

	hasRedFill := func(res *Result) bool {
		for _, op := range res.Ops {
			if op.Kind == OpFillRect && op.R > 0.8 && op.G < 0.4 && op.B < 0.4 {
				return true
			}
		}
		return false
	}

	if hasRedFill(hidden) {
		t.Fatalf("no-overflow should hide overflowing anchored tooltip")
	}
	if !hasRedFill(shown) {
		t.Fatalf("anchor-valid should keep overflowing anchored tooltip visible")
	}
}

func TestGridTemplateAreasSingleVsOffset(t *testing.T) {
	t.Parallel()

	ctrl := layoutHTML(t, `<html><body style="margin:0">`+
		`<div style="display:grid;width:120px;grid-template-columns:1fr 1fr;grid-template-areas:'. x'">`+
		`<div id="x" style="grid-area:x;height:20px;background:#3b82f6"></div></div></body></html>`)
	targ := layoutHTML(t, `<html><body style="margin:0">`+
		`<div style="display:grid;width:120px;grid-template-columns:1fr 1fr;grid-template-areas:'x'">`+
		`<div id="x" style="grid-area:x;height:20px;background:#3b82f6"></div></div></body></html>`)

	xCtrl := boxByID(t, ctrl, "x")
	xTarg := boxByID(t, targ, "x")
	if xCtrl.x <= xTarg.x {
		t.Fatalf("control '. x' box x=%.2f should be right of target 'x' box x=%.2f", xCtrl.x, xTarg.x)
	}
}
