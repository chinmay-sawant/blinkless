package layout

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// Gap-close ABC: three properties whose prior pins live in
// internal/layout/css_behavior_gap_close_a_test.go. Each test asserts USED
// drawing-list geometry (op kind plus X/Y/W/H, text, and raster bytes),
// never stored style strings. Reference browser for every case: Chrome
// 143.0.7499.40. Shared helpers layoutHTML, boxByID, near, pxToPt,
// ptPerCSSPx, opsOfKind, mustParse, and sheet come from other files in this
// package and are reused here, never redeclared.

// gapABCSVGTextResult lays out one inline SVG holding a single text element
// with the given extra text style. The SVG path (layout_svg.go) bakes
// text-anchor into an attribute before rasterization.
func gapABCSVGTextResult(t *testing.T, textStyle string) *Result {
	t.Helper()

	return layoutHTML(t, `<html><body style="margin:0">`+
		`<svg width="120" height="30" viewBox="0 0 120 30" xmlns="http://www.w3.org/2000/svg">`+
		`<text x="60" y="20" font-size="16" style="fill:#000000;`+textStyle+`">Hi</text>`+
		`</svg></body></html>`)
}

// gapABCSVGTextSerialized returns the serialized SVG XML for one text style,
// which is the payload the rasterizer consumes.
func gapABCSVGTextSerialized(t *testing.T, textStyle string) string {
	t.Helper()

	root := mustParse(t, `<html><body>`+
		`<svg width="120" height="30" viewBox="0 0 120 30" xmlns="http://www.w3.org/2000/svg">`+
		`<text x="60" y="20" font-size="16" style="fill:#000000;`+textStyle+`">Hi</text>`+
		`</svg></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `body{margin:0}`)}, "print", 500, 800)
	eng := &engine{styles: styles, scale: 1}

	var svgNode *html.Node

	root.Walk(func(node *html.Node) {
		if svgNode == nil && node.Type == html.ElementNode && node.Name == "svg" {
			svgNode = node
		}
	})

	if svgNode == nil {
		t.Fatal("svg missing")
	}

	return string(eng.serializeInlineSVG(svgNode))
}

// TestGapABCTextAnchorMiddleAnchorsSvgText is text-anchor: middle bakes into
// the serialized SVG as a text-anchor attribute on the text element, and the
// rasterizer (which honors middle/end anchoring) paints different pixels than
// the default start anchor. HTML text is unaffected: Chrome 143.0.7499.40
// anchors SVG text only.
func TestGapABCTextAnchorMiddleAnchorsSvgText(t *testing.T) {
	t.Parallel()

	serialized := gapABCSVGTextSerialized(t, "text-anchor:middle")
	if !strings.Contains(serialized, `text-anchor="middle"`) {
		t.Errorf("serialized SVG lacks text-anchor middle, got %s", serialized)
	}

	bogus := gapABCSVGTextSerialized(t, "text-anchor:bogus")
	if strings.Contains(bogus, "text-anchor") {
		t.Errorf("bogus text-anchor baked, want it dropped, got %s", bogus)
	}

	startImgs := opsOfKind(gapABCSVGTextResult(t, ""), OpImage)
	middleImgs := opsOfKind(gapABCSVGTextResult(t, "text-anchor:middle"), OpImage)

	if len(startImgs) != 1 || len(middleImgs) != 1 {
		t.Fatalf("svg image ops = %d / %d, want 1 each", len(startImgs), len(middleImgs))
	}

	if bytes.Equal(startImgs[0].Image, middleImgs[0].Image) {
		t.Error("text-anchor:middle raster bytes match start anchor, want shifted text pixels")
	}

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">anchor sample</p></body></html>`)
	anchored := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-anchor:middle">anchor sample</p></body></html>`)

	behaviorSvgAssertSameOps(t, plain, anchored)
}

// gapABCAlignDoc builds a paragraph carrying one baseline declaration.
func gapABCAlignDoc(decl string) string {
	return `<html><body style="margin:0">` +
		`<p id="p" style="margin:0;font-size:16px;` + decl + `">Ag</p></body></html>`
}

// behaviorABCAlignUsed lays out one probe paragraph and returns its used
// alignment-baseline value.
func behaviorABCAlignUsed(t *testing.T, extra string) string {
	t.Helper()

	return boxByID(t, layoutHTML(t, gapABCAlignDoc(extra)), "p").style.AlignmentBaseline
}

// gapABCTextY returns the baseline Y of the first text op.
func gapABCTextY(t *testing.T, res *Result) float64 {
	t.Helper()

	texts := opsOfKind(res, OpText)
	if len(texts) == 0 {
		t.Fatal("want a text op, got none")
	}

	return texts[0].Y
}

// TestGapABCAlignmentBaselineCentralIdeographic is alignment-baseline: the
// central selector raises the run above alphabetic by half the em box and
// ideographic lowers it by the face descent, while alphabetic paints at the
// default baseline. Reference: Chrome 143.0.7499.40 shifts the text baseline.
func TestGapABCAlignmentBaselineCentralIdeographic(t *testing.T) {
	t.Parallel()

	plainY := gapABCTextY(t, layoutHTML(t, gapABCAlignDoc("")))
	centralY := gapABCTextY(t, layoutHTML(t, gapABCAlignDoc("alignment-baseline:central")))
	ideoY := gapABCTextY(t, layoutHTML(t, gapABCAlignDoc("alignment-baseline:ideographic")))

	if got := behaviorABCAlignUsed(t, "alignment-baseline:central"); got != "central" {
		t.Errorf("alignment-baseline used = %q, want central", got)
	}

	if got := behaviorABCAlignUsed(t, ""); got != "auto" {
		t.Errorf("alignment-baseline initial used = %q, want auto", got)
	}

	if lift := plainY - centralY; lift <= 0.5 || lift > 8 {
		t.Errorf("central lift = %.4fpt (%.2fpx), want a rise above alphabetic",
			lift, lift/ptPerCSSPx)
	}

	if drop := ideoY - plainY; drop <= 0.5 || drop > 8 {
		t.Errorf("ideographic drop = %.4fpt (%.2fpx), want a descent below alphabetic",
			drop, drop/ptPerCSSPx)
	}

	behaviorSvgAssertSameOps(t,
		layoutHTML(t, gapABCAlignDoc("")),
		layoutHTML(t, gapABCAlignDoc("alignment-baseline:alphabetic")))
}

// TestGapABCAlignmentBaselineMiddleHanging is alignment-baseline middle and
// hanging: middle rises by half the x-height and hanging keeps the 5px rise
// at 16px, while a bogus value keeps the default baseline. Reference: Chrome
// 143.0.7499.40 shifts the text baseline.
func TestGapABCAlignmentBaselineMiddleHanging(t *testing.T) {
	t.Parallel()

	plainY := gapABCTextY(t, layoutHTML(t, gapABCAlignDoc("")))
	middleY := gapABCTextY(t, layoutHTML(t, gapABCAlignDoc("alignment-baseline:middle")))
	hangY := gapABCTextY(t, layoutHTML(t, gapABCAlignDoc("alignment-baseline:hanging")))

	if lift := plainY - middleY; lift <= 0.5 || lift > 8 {
		t.Errorf("middle lift = %.4fpt (%.2fpx), want a rise above alphabetic",
			lift, lift/ptPerCSSPx)
	}

	if lift, want := plainY-hangY, pxToPt(5); !near(lift, want) {
		t.Errorf("hanging lift = %.4fpt, want %.4fpt (5px at 16px)", lift, want)
	}

	behaviorSvgAssertSameOps(t,
		layoutHTML(t, gapABCAlignDoc("")),
		layoutHTML(t, gapABCAlignDoc("alignment-baseline:bogus")))
}

// gapABCRedFill returns the first sizable red fill op, which is the
// background of the test box.
func gapABCRedFill(t *testing.T, res *Result) Op {
	t.Helper()

	for _, op := range opsOfKind(res, OpFillRect) {
		if op.R > 0.9 && op.G < 0.1 && op.B < 0.1 && op.W > 10 && op.H > 5 {
			return op
		}
	}

	t.Fatal("no red fill op for the test box")

	return Op{}
}

// TestGapABCClipRectCutsAbsposBox is the deprecated clip: rect() cuts the
// absolutely positioned box to the rect, so the 100px by 50px box paints
// 50px by 20px at the same top-left origin. Static boxes ignore clip and
// clip:auto paints whole, per CSS 2.1 section 11.1.2. Reference: Chrome
// 143.0.7499.40 cuts the absolute box to rect(0px,50px,20px,0px).
func TestGapABCClipRectCutsAbsposBox(t *testing.T) {
	t.Parallel()

	absDiv := func(extra string) string {
		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="a" style="position:absolute;top:10px;left:10px;width:100px;height:50px;` +
			`background-color:#ff0000;` + extra + `"></div>` +
			`</body></html>`
	}

	plain := layoutHTML(t, absDiv(""))
	clipped := layoutHTML(t, absDiv("clip:rect(0px,50px,20px,0px)"))

	if got := boxByID(t, clipped, "a").style.Clip; got == "" {
		t.Error("clip used = empty, want the canonical rect()")
	}

	plainFill, clipFill := gapABCRedFill(t, plain), gapABCRedFill(t, clipped)

	if !near(clipFill.X, plainFill.X) || !near(clipFill.Y, plainFill.Y) {
		t.Errorf("clipped origin = (%.4f, %.4f), want plain origin (%.4f, %.4f)",
			clipFill.X, clipFill.Y, plainFill.X, plainFill.Y)
	}

	if !near(clipFill.W, pxToPt(50)) {
		t.Errorf("clipped width = %.4fpt (%.2fpx), want 50px", clipFill.W, clipFill.W/ptPerCSSPx)
	}

	if !near(clipFill.H, pxToPt(20)) {
		t.Errorf("clipped height = %.4fpt (%.2fpx), want 20px", clipFill.H, clipFill.H/ptPerCSSPx)
	}

	behaviorSvgAssertSameOps(t, plain, layoutHTML(t, absDiv("clip:auto")))

	staticDoc := func(extra string) string {
		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="a" style="width:100px;height:50px;background-color:#ff0000;` + extra + `"></div>` +
			`</body></html>`
	}

	behaviorSvgAssertSameOps(t,
		layoutHTML(t, staticDoc("")),
		layoutHTML(t, staticDoc("clip:rect(0px,50px,20px,0px)")))

	behaviorSvgAssertSameOps(t, plain, layoutHTML(t, absDiv("clip:rect(0px,50px)")))
}
