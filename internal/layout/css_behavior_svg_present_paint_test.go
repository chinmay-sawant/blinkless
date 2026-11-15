package layout

import (
	"bytes"
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
	xml, _ := svgPaintRender(t, svgPaintBaseCSS+`#r{paint-order:stroke}`, strings.Replace(rect, "%s", "", 1))

	if !strings.Contains(xml, `paint-order="stroke"`) {
		t.Errorf("stylesheet paint-order not serialized: %s", xml)
	}

	attr, _ := svgPaintRender(t, svgPaintBaseCSS+`#r{paint-order:stroke}`,
		strings.Replace(rect, "%s", ` paint-order="fill"`, 1))
	if strings.Contains(attr, `paint-order="fill"`) || !strings.Contains(attr, `paint-order="stroke"`) {
		t.Errorf("attribute won over the stylesheet: %s", attr)
	}

	bad, _ := svgPaintRender(t, svgPaintBaseCSS+`#r{paint-order:stroke stroke}`,
		strings.Replace(rect, "%s", "", 1))
	if strings.Contains(bad, "paint-order") {
		t.Errorf("repeated token was baked: %s", bad)
	}
}

func TestBehaviorSvgPresentPathLength(t *testing.T) {
	t.Parallel()

	dash := `<path id="d" d="M4 20 H60" fill="none" stroke="#000000" stroke-width="3" stroke-dasharray="6 3"%s/>`
	xml, _ := svgPaintRender(t, svgPaintBaseCSS+`#d{path-length:100}`, strings.Replace(dash, "%s", "", 1))

	if !strings.Contains(xml, `pathLength="100"`) {
		t.Errorf("stylesheet path-length not serialized: %s", xml)
	}

	withAttr, _ := svgPaintRender(t, svgPaintBaseCSS+`#d{path-length:100}`,
		strings.Replace(dash, "%s", ` pathLength="50"`, 1))
	if strings.Contains(withAttr, `pathLength="50"`) || !strings.Contains(withAttr, `pathLength="100"`) {
		t.Errorf("attribute won over the stylesheet: %s", withAttr)
	}

	neg, _ := svgPaintRender(t, svgPaintBaseCSS+`#d{path-length:-5}`,
		strings.Replace(dash, "%s", "", 1))
	if strings.Contains(neg, "pathLength") {
		t.Errorf("negative path-length was baked: %s", neg)
	}

	// Pinned canvas gap: the raster does not observe pathLength.
	plain := svgPaintPixels(t, svgPaintBaseCSS, strings.Replace(dash, "%s", "", 1))
	long := svgPaintPixels(t, svgPaintBaseCSS+`#d{path-length:100}`, strings.Replace(dash, "%s", "", 1))

	if plain != long {
		t.Logf("canvas now honors pathLength (plain=%d, with=%d): the catalog row can flip", plain, long)
	}
}
