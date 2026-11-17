package layout

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
	"github.com/chinmay-sawant/blinkless/internal/svg"
)

// Stylesheet SVG paint properties: marker and its longhands, stop-color,
// stop-opacity, paint-order, and path-length. A stylesheet value must beat
// the presentation attribute. paint-order and path-length are serialized
// only: the pinned canvas rasterizer reads neither, so their tests assert
// the XML and the catalog rows stay unsupported.

const svgPaintDefs = behaviorSvgMarkersArrowDefs +
	`<defs><linearGradient id="g" x1="0%" y1="0%" x2="100%" y2="0%"><stop id="s1" offset="0" stop-color="#ff0000"/>` +
	`<stop offset="1" stop-color="#0000ff"/></linearGradient></defs>`

// svgPaintRender serializes one inline svg under the given stylesheet and
// returns the XML and the decoded raster.
func svgPaintRender(t *testing.T, sheetText, inner string) (string, image.Image) {
	t.Helper()

	root := mustParse(t, `<html><body style="margin:0">`+
		`<svg width="64" height="40" viewBox="0 0 64 40" xmlns="http://www.w3.org/2000/svg">`+
		svgPaintDefs+inner+`</svg></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, sheetText)}, "print", testViewport, 800)
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

	xml := eng.serializeInlineSVG(svgNode)

	pngBytes, _, _, err := svg.Rasterize(xml, 256)
	if err != nil {
		t.Fatalf("rasterize: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	return string(xml), img
}

func svgPaintPixels(t *testing.T, sheetText, inner string) int {
	t.Helper()

	_, img := svgPaintRender(t, sheetText, inner)
	n := 0
	b := img.Bounds()

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a>>8 >= 128 {
				n++
			}
		}
	}

	return n
}

const (
	svgPaintTwoSeg  = `<path id="p" d="M8 34 L32 8 L56 34" fill="none" stroke="#000000" stroke-width="3"/>`
	svgPaintLine    = `<path id="p" d="M8 30 H56" fill="none" stroke="#000000" stroke-width="3"/>`
	svgPaintGrad    = `<rect x="2" y="2" width="60" height="24" fill="url(#g)"/>`
	svgPaintBaseCSS = `body{margin:0}`
)

func TestBehaviorSvgPresentMarker(t *testing.T) {
	t.Parallel()

	with := svgPaintPixels(t, svgPaintBaseCSS+`#p{marker:url(#ah)}`, svgPaintTwoSeg)
	none := svgPaintPixels(t, svgPaintBaseCSS+`#p{marker:none}`, svgPaintTwoSeg)

	if with <= none {
		t.Errorf("marker url painted = %d, none = %d, want url greater", with, none)
	}

	// A longhand declared after the shorthand keeps its own value.
	xml, _ := svgPaintRender(t, svgPaintBaseCSS+`#p{marker:url(#ah);marker-mid:none}`, svgPaintTwoSeg)
	if !strings.Contains(xml, `marker-start="url(#ah)"`) || strings.Contains(xml, "marker-mid=") {
		t.Errorf("longhand after shorthand lost its value: %s", xml)
	}

	// An invalid token is not stored, so the presentation attribute stays.
	attr := strings.Replace(svgPaintTwoSeg, `<path `, `<path marker-end="url(#ah)" `, 1)
	xml, _ = svgPaintRender(t, svgPaintBaseCSS+`#p{marker:bogus}`, attr)
	if !strings.Contains(xml, `marker-end="url(#ah)"`) {
		t.Errorf("invalid marker dropped the attribute: %s", xml)
	}
}

// The canvas draws a second marker on the wrong layer, so a shape with two or
// three markers is written once per marker. Every vertex must get its arrow.
func TestBehaviorSvgPresentMarkerAllThreeVertices(t *testing.T) {
	t.Parallel()

	// The three markers sit at different vertices, so each region must have
	// ink that the no-marker path lacks: start (8,34), mid (32,8), end (56,34).
	_, none := svgPaintRender(t, svgPaintBaseCSS+`#p{marker:none}`, svgPaintTwoSeg)
	_, all := svgPaintRender(t, svgPaintBaseCSS+`#p{marker:url(#ah)}`, svgPaintTwoSeg)

	for name, at := range map[string][2]int{"start": {8, 34}, "mid": {32, 8}, "end": {56, 34}} {
		if svgPaintRegionDiff(none, all, at[0], at[1]) == 0 {
			t.Errorf("no marker ink around the %s vertex", name)
		}
	}

	// Nothing may land at the origin, where the canvas put a misplaced copy.
	if svgPaintRegionDiff(none, all, 0, 0) != 0 {
		t.Error("stray marker ink at the top-left corner")
	}

	xml, _ := svgPaintRender(t, svgPaintBaseCSS+`#p{marker:url(#ah)}`, svgPaintTwoSeg)
	if strings.Count(xml, `id="p"`) != 1 || strings.Count(xml, `stroke-dasharray="0 100000"`) != 2 {
		t.Errorf("want one painted path and two marker copies, got %s", xml)
	}
}

// svgPaintRegionDiff counts differing pixels in a 14 by 14 viewBox-unit window
// around (cx, cy). The test svg is 64 by 40 units; the raster is scaled.
func svgPaintRegionDiff(a, b image.Image, cx, cy int) int {
	scale := float64(a.Bounds().Dx()) / 64
	n := 0

	for y := int(float64(cy-7) * scale); y < int(float64(cy+7)*scale); y++ {
		for x := int(float64(cx-7) * scale); x < int(float64(cx+7)*scale); x++ {
			if x < 0 || y < 0 || x >= a.Bounds().Dx() || y >= a.Bounds().Dy() {
				continue
			}

			if a.At(x, y) != b.At(x, y) {
				n++
			}
		}
	}

	return n
}

func TestBehaviorSvgPresentMarkerEnd(t *testing.T) {
	t.Parallel()

	with := svgPaintPixels(t, svgPaintBaseCSS+`#p{marker-end:url(#ah)}`, svgPaintLine)
	none := svgPaintPixels(t, svgPaintBaseCSS+`#p{marker-end:none}`, svgPaintLine)

	if with <= none {
		t.Errorf("marker-end url painted = %d, none = %d, want url greater", with, none)
	}

	attr := strings.Replace(svgPaintLine, `<path `, `<path marker-end="url(#ah)" `, 1)
	cleared := svgPaintPixels(t, svgPaintBaseCSS+`#p{marker-end:none}`, attr)

	if cleared != none {
		t.Errorf("stylesheet none left the attribute marker: painted = %d, want %d", cleared, none)
	}
}

func TestBehaviorSvgPresentMarkerStart(t *testing.T) {
	t.Parallel()

	with := svgPaintPixels(t, svgPaintBaseCSS+`#p{marker-start:url(#ah)}`, svgPaintLine)
	none := svgPaintPixels(t, svgPaintBaseCSS+`#p{marker-start:none}`, svgPaintLine)

	if with <= none {
		t.Errorf("marker-start url painted = %d, none = %d, want url greater", with, none)
	}

	xml, _ := svgPaintRender(t, svgPaintBaseCSS+`#p{marker-start:url(#ah)}`, svgPaintLine)
	if strings.Contains(xml, "marker-end=") || strings.Contains(xml, "marker-mid=") {
		t.Errorf("marker-start bled onto other longhands: %s", xml)
	}
}

func TestBehaviorSvgPresentMarkerMid(t *testing.T) {
	t.Parallel()

	with := svgPaintPixels(t, svgPaintBaseCSS+`#p{marker-mid:url(#ah)}`, svgPaintTwoSeg)
	none := svgPaintPixels(t, svgPaintBaseCSS+`#p{marker-mid:none}`, svgPaintTwoSeg)

	if with <= none {
		t.Errorf("marker-mid url painted = %d, none = %d, want url greater", with, none)
	}
}

func TestBehaviorSvgPresentStopColor(t *testing.T) {
	t.Parallel()

	_, red := svgPaintRender(t, svgPaintBaseCSS+`#s1{stop-color:#ff0000}`, svgPaintGrad)
	_, green := svgPaintRender(t, svgPaintBaseCSS+`#s1{stop-color:#00ff00}`, svgPaintGrad)

	if svgBakeDiffPixels(t, red, green) == 0 {
		t.Error("stylesheet stop-color did not change the gradient")
	}

	// CSS beats the stop-color attribute: the attribute is #ff0000 in defs.
	attrWin, _ := svgPaintRender(t, svgPaintBaseCSS+`#s1{stop-color:#00ff00}`, svgPaintGrad)
	if !strings.Contains(attrWin, `stop-color="#00ff00"`) || strings.Count(attrWin, `id="s1"`) != 1 ||
		strings.Contains(attrWin, `id="s1" offset="0" stop-color="#ff0000"`) {
		t.Errorf("attribute won over the stylesheet: %s", attrWin)
	}

	// A named color becomes hex so the canvas parser does not turn it black.
	named, _ := svgPaintRender(t, svgPaintBaseCSS+`#s1{stop-color:lime}`, svgPaintGrad)
	if !strings.Contains(named, `stop-color="#00ff00"`) {
		t.Errorf("named stop-color not converted to hex: %s", named)
	}

	// A stop-color on a rect does not bake.
	onRect, _ := svgPaintRender(t, svgPaintBaseCSS+`rect{stop-color:#00ff00}`, svgPaintGrad)
	if _, afterRect, found := strings.Cut(onRect, `<rect`); found && strings.Contains(afterRect, "stop-color") {
		t.Errorf("stop-color baked onto a rect: %s", onRect)
	}
}

func TestBehaviorSvgPresentStopOpacity(t *testing.T) {
	t.Parallel()

	_, opaque := svgPaintRender(t, svgPaintBaseCSS, svgPaintGrad)
	_, faded := svgPaintRender(t, svgPaintBaseCSS+`#s1{stop-opacity:0.2}`, svgPaintGrad)

	if svgBakeDiffPixels(t, opaque, faded) == 0 {
		t.Error("stylesheet stop-opacity did not fade the gradient")
	}

	// An out-of-range value is not stored.
	xml, _ := svgPaintRender(t, svgPaintBaseCSS+`#s1{stop-opacity:3}`, svgPaintGrad)
	if strings.Contains(xml, "stop-opacity") {
		t.Errorf("out-of-range stop-opacity was baked: %s", xml)
	}
}

func TestBehaviorSvgPresentPaintOrder(t *testing.T) {
	t.Parallel()

	rect := `<rect id="r" x="8" y="8" width="40" height="20" fill="#ff0000" stroke="#0000ff" stroke-width="8"%s/>`
	plain := strings.Replace(rect, "%s", "", 1)

	// Inner half of the left stroke: x from 8 to 12, y in the middle. The
	// default order paints the stroke over the fill (blue). stroke first
	// lets the fill cover that half (red).
	inner := func(css, inner string) (int, int) {
		_, img := svgPaintRender(t, svgPaintBaseCSS+css, inner)
		scale := float64(img.Bounds().Dx()) / 64
		red, _, blue, _ := img.At(int(10*scale), int(18*scale)).RGBA()

		return int(red >> 8), int(blue >> 8)
	}

	r, b := inner(``, plain)
	if b < 200 || r > 60 {
		t.Fatalf("default order inner stroke pixel = (%d,_,%d), want blue", r, b)
	}

	r, b = inner(`#r{paint-order:stroke}`, plain)
	if r < 200 || b > 60 {
		t.Errorf("paint-order:stroke inner stroke pixel = (%d,_,%d), want red fill over the stroke", r, b)
	}

	// The outer half stays blue in both orders.
	_, img := svgPaintRender(t, svgPaintBaseCSS+`#r{paint-order:stroke}`, plain)
	scale := float64(img.Bounds().Dx()) / 64

	if r, _, b, _ := img.At(int(6*scale), int(18*scale)).RGBA(); b>>8 < 200 || r>>8 > 60 {
		t.Errorf("paint-order:stroke outer stroke pixel = (%d,_,%d), want blue", r>>8, b>>8)
	}

	// An explicit fill-first order and normal match the default.
	for _, css := range []string{`#r{paint-order:normal}`, `#r{paint-order:fill stroke}`} {
		if r, b := inner(css, plain); b < 200 || r > 60 {
			t.Errorf("%s inner stroke pixel = (%d,_,%d), want blue like the default", css, r, b)
		}
	}

	// The stylesheet beats an attribute, and a no-stroke shape is untouched.
	attr := strings.Replace(rect, "%s", ` paint-order="fill"`, 1)
	if r, b := inner(`#r{paint-order:stroke}`, attr); r < 200 || b > 60 {
		t.Errorf("attribute won over the stylesheet: pixel = (%d,_,%d)", r, b)
	}

	xml, _ := svgPaintRender(t, svgPaintBaseCSS+`#r{paint-order:stroke stroke}`, plain)
	if strings.Contains(xml, "paint-order") {
		t.Errorf("repeated token was baked: %s", xml)
	}
}

func TestBehaviorSvgPresentPathLength(t *testing.T) {
	t.Parallel()

	dash := `<path id="d" d="M4 20 H60" fill="none" stroke="#000000" stroke-width="3" stroke-dasharray="%s"/>`
	line := func(css, dashes string) image.Image {
		_, img := svgPaintRender(t, svgPaintBaseCSS+css, fmt.Sprintf(dash, dashes))

		return img
	}

	xml, _ := svgPaintRender(t, svgPaintBaseCSS+`#d{path-length:100}`, fmt.Sprintf(dash, "6 3"))
	if !strings.Contains(xml, `pathLength="100"`) {
		t.Errorf("stylesheet path-length not serialized: %s", xml)
	}

	// The line is 56 units. path-length:112 halves every dash length, so it
	// must match a hand-written "3 1.5" pattern and differ from "6 3".
	scaled := line(`#d{path-length:112}`, "6 3")
	byHand := line(``, "3 1.5")
	unscaled := line(``, "6 3")

	if svgBakeDiffPixels(t, scaled, byHand) != 0 {
		t.Error("path-length:112 did not rescale the dashes to the hand-written pattern")
	}

	if svgBakeDiffPixels(t, scaled, unscaled) == 0 {
		t.Error("path-length:112 left the dashes unscaled")
	}

	// A path-length equal to the real length changes nothing.
	if svgBakeDiffPixels(t, line(`#d{path-length:56}`, "6 3"), unscaled) != 0 {
		t.Error("path-length equal to the real length changed the dashes")
	}

	// none and a negative length are not stored, so nothing rescales.
	for _, css := range []string{`#d{path-length:none}`, `#d{path-length:-5}`} {
		if svgBakeDiffPixels(t, line(css, "6 3"), unscaled) != 0 {
			t.Errorf("%s changed the dashes", css)
		}
	}

	// The stylesheet beats a pathLength attribute.
	attr := strings.Replace(fmt.Sprintf(dash, "6 3"), `<path `, `<path pathLength="28" `, 1)
	_, img := svgPaintRender(t, svgPaintBaseCSS+`#d{path-length:112}`, attr)

	if svgBakeDiffPixels(t, img, byHand) != 0 {
		t.Error("attribute pathLength won over the stylesheet")
	}
}

// vector-effect: non-scaling-stroke keeps the stroke width in device units.
// The svg below maps a 256 by 160 viewBox onto 64 by 40 device units, a scale
// of 0.25, so a stroke-width of 6 paints 1.5 units wide by default and 6 units
// wide with non-scaling-stroke. Chrome 143 ships only none and
// non-scaling-stroke.
func TestBehaviorSvgPresentVectorEffect(t *testing.T) {
	t.Parallel()

	render := func(sheetText, attr string) (int, string) {
		root := mustParse(t, `<html><body style="margin:0"><svg width="64" height="40" viewBox="0 0 256 160" `+
			`xmlns="http://www.w3.org/2000/svg"><line id="l" x1="40" y1="80" x2="216" y2="80" fill="none" `+
			`stroke="#000000" stroke-width="6"`+attr+`/></svg></body></html>`)
		styles := resolveStyles(root, []*css.Stylesheet{sheet(t, sheetText)}, "print", testViewport, 800)
		eng := &engine{styles: styles, scale: 1}

		var svgNode *html.Node

		root.Walk(func(node *html.Node) {
			if svgNode == nil && node.Type == html.ElementNode && node.Name == "svg" {
				svgNode = node
			}
		})

		xml := eng.serializeInlineSVG(svgNode)

		pngBytes, _, _, err := svg.Rasterize(xml, 256)
		if err != nil {
			t.Fatalf("rasterize: %v", err)
		}

		img, err := png.Decode(bytes.NewReader(pngBytes))
		if err != nil {
			t.Fatalf("decode: %v", err)
		}

		n := 0
		b := img.Bounds()

		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if _, _, _, a := img.At(x, y).RGBA(); a>>8 >= 128 {
					n++
				}
			}
		}

		return n, string(xml)
	}

	plain, _ := render(`#l{vector-effect:none}`, "")
	fixed, xml := render(`#l{vector-effect:non-scaling-stroke}`, "")

	if fixed <= plain*3 {
		t.Errorf("non-scaling-stroke painted %d, none painted %d, want roughly 4x the pixels", fixed, plain)
	}

	if !strings.Contains(xml, `vector-effect="non-scaling-stroke"`) {
		t.Errorf("stylesheet vector-effect not serialized: %s", xml)
	}

	// The stylesheet beats the attribute, and an unknown token is not stored.
	byAttr, _ := render(``, ` vector-effect="none"`)
	overridden, _ := render(`#l{vector-effect:non-scaling-stroke}`, ` vector-effect="none"`)

	if overridden != fixed || byAttr != plain {
		t.Errorf("attribute won over the stylesheet: overridden=%d fixed=%d attr=%d plain=%d",
			overridden, fixed, byAttr, plain)
	}

	if bogus, _ := render(`#l{vector-effect:bogus}`, ""); bogus != plain {
		t.Errorf("an unknown token changed the stroke: %d vs %d", bogus, plain)
	}
}

// flood-color and flood-opacity act on feFlood. The pinned canvas has no
// filter support, so internal/svg turns a filter that holds one feFlood into
// a flat rect over the filter region (the box plus 10 percent on each side).
func TestBehaviorSvgPresentFloodColorAndOpacity(t *testing.T) {
	t.Parallel()

	filter := `<defs><filter id="f"><feFlood id="fl" flood-color="#ff0000"%s/></filter></defs>` +
		`<rect x="16" y="8" width="32" height="16" fill="#0000ff" filter="url(#f)"/>`
	render := func(sheetText, extra string) image.Image {
		_, img := svgPaintRender(t, svgPaintBaseCSS+sheetText, strings.Replace(filter, "%s", extra, 1))

		return img
	}
	pixel := func(img image.Image, x, y int) (int, int, int, int) {
		scale := float64(img.Bounds().Dx()) / 64
		r, g, b, a := img.At(int(float64(x)*scale), int(float64(y)*scale)).RGBA()

		return int(r >> 8), int(g >> 8), int(b >> 8), int(a >> 8)
	}

	red := render(``, "")
	if r, g, b, a := pixel(red, 30, 16); r < 200 || g > 60 || b > 60 || a < 250 {
		t.Errorf("attribute flood pixel = (%d,%d,%d,%d), want opaque red, not the blue rect", r, g, b, a)
	}

	// The filter region is the box plus 10 percent: x 12.8 to 51.2.
	if _, _, _, a := pixel(red, 14, 16); a < 250 {
		t.Errorf("pixel inside the filter margin has alpha %d, want flooded", a)
	}

	if _, _, _, a := pixel(red, 6, 16); a != 0 {
		t.Errorf("pixel outside the filter region has alpha %d, want empty", a)
	}

	green := render(`#fl{flood-color:#00ff00}`, "")
	if r, g, _, _ := pixel(green, 30, 16); g < 200 || r > 60 {
		t.Errorf("stylesheet flood-color pixel = (%d,%d,...), want green over the attribute", r, g)
	}

	if svgBakeDiffPixels(t, red, green) == 0 {
		t.Error("two flood colors rendered the same")
	}

	named := render(`#fl{flood-color:lime}`, "")
	if svgBakeDiffPixels(t, green, named) != 0 {
		t.Error("a named flood color did not become the same hex as #00ff00")
	}

	faded := render(`#fl{flood-opacity:0.2}`, "")
	if _, _, _, a := pixel(faded, 30, 16); a < 40 || a > 70 {
		t.Errorf("flood-opacity:0.2 alpha = %d, want about 51", a)
	}

	if svgBakeDiffPixels(t, red, faded) == 0 {
		t.Error("flood-opacity did not change the picture")
	}

	// A flood-color on a rect does not bake, and an out-of-range opacity is refused.
	xml, _ := svgPaintRender(t, svgPaintBaseCSS+`rect{flood-color:#00ff00}`, strings.Replace(filter, "%s", "", 1))
	if _, after, found := strings.Cut(xml, "<rect"); found && strings.Contains(after, "flood-color") {
		t.Errorf("flood-color baked onto a rect: %s", xml)
	}

	if bad := render(`#fl{flood-opacity:3}`, ""); svgBakeDiffPixels(t, red, bad) != 0 {
		t.Error("an out-of-range flood-opacity changed the picture")
	}
}
