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

func svgPartialRender(t *testing.T, sheetText, svgTag string) (string, image.Image) {
	t.Helper()

	root := mustParse(t, `<html><body style="margin:0">`+svgTag+`</body></html>`)
	var sheets []*css.Stylesheet
	if sheetText != "" {
		sheets = []*css.Stylesheet{sheet(t, sheetText)}
	}
	styles := resolveStyles(root, sheets, "print", testViewport, 800)
	eng := &engine{styles: styles, scale: 1}

	var svgNode *html.Node
	root.Walk(func(node *html.Node) {
		if svgNode == nil && node.Type == html.ElementNode && node.Name == "svg" {
			svgNode = node
		}
	})
	if svgNode == nil {
		t.Fatal("svg node missing")
	}

	xmlBytes := eng.serializeInlineSVG(svgNode)
	pngBytes, _, _, err := svg.Rasterize(xmlBytes, 256)
	if err != nil {
		t.Fatalf("rasterize failed: %v\nxml: %s", err, xmlBytes)
	}

	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("png decode failed: %v", err)
	}

	return string(xmlBytes), img
}

func TestBehaviorSvgPartialPaintOrderInherited(t *testing.T) {
	t.Parallel()

	shape := `<rect id="r" x="16" y="16" width="88" height="48" fill="#c0392b" stroke="#1f2937" stroke-width="12"/>`
	_, normalImg := svgPartialRender(t, "",
		`<svg width="120" height="80" viewBox="0 0 120 80">`+shape+`</svg>`)
	xml, strokeFirstImg := svgPartialRender(t, "",
		`<svg width="120" height="80" viewBox="0 0 120 80" style="paint-order:stroke fill">`+shape+`</svg>`)

	if !strings.Contains(xml, `paint-order="stroke fill"`) {
		t.Fatalf("inherited paint-order missing from serialized XML: %s", xml)
	}

	if diff := svgBakeDiffPixels(t, normalImg, strokeFirstImg); diff == 0 {
		t.Error("inherited paint-order:stroke fill raster identical to default, want stroke behind fill")
	}

	// Child's own presentation attribute beats an inherited value on <svg>.
	childAttr := `<rect id="r" x="16" y="16" width="88" height="48" fill="#c0392b" stroke="#1f2937" stroke-width="12" paint-order="fill"/>`
	_, childWinsImg := svgPartialRender(t, "",
		`<svg width="120" height="80" viewBox="0 0 120 80" style="paint-order:stroke fill">`+childAttr+`</svg>`)
	if diff := svgBakeDiffPixels(t, normalImg, childWinsImg); diff != 0 {
		t.Errorf("child paint-order=\"fill\" attribute differed from default by %d pixels, want attribute to beat inherited rule", diff)
	}
}

func TestBehaviorSvgPartialStopColorAndOpacityUnitlessGradient(t *testing.T) {
	t.Parallel()

	mkGrad := func(secondStopStyle string) string {
		styleAttr := ""
		if secondStopStyle != "" {
			styleAttr = ` style="` + secondStopStyle + `"`
		}
		return `<svg width="120" height="80" viewBox="0 0 120 80">` +
			`<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="0">` +
			`<stop offset="0" stop-color="#3b82f6"/>` +
			`<stop offset="1" stop-color="#1f2937"` + styleAttr + `/>` +
			`</linearGradient></defs>` +
			`<rect x="10" y="10" width="100" height="60" fill="url(#g)"/></svg>`
	}

	baseXML, baseImg := svgPartialRender(t, "", mkGrad(""))
	if !strings.Contains(baseXML, `x1="0%"`) || !strings.Contains(baseXML, `x2="100%"`) {
		t.Fatalf("unitless objectBoundingBox linearGradient coords not normalized to %%: %s", baseXML)
	}

	// Midpoint must be a smooth blend (more blue than the right edge, not flat).
	b := baseImg.Bounds()
	scaleX := float64(b.Dx()) / 120
	scaleY := float64(b.Dy()) / 80
	_, _, midBlue, _ := baseImg.At(int(60*scaleX), int(40*scaleY)).RGBA()
	_, _, rightBlue, _ := baseImg.At(int(105*scaleX), int(40*scaleY)).RGBA()
	if midBlue <= rightBlue+2000 {
		t.Errorf("gradient not smoothly blending: midBlue=%d, rightBlue=%d", midBlue>>8, rightBlue>>8)
	}

	// Inline stop-color overrides the stop-color presentation attribute (hex and named).
	hexXML, hexImg := svgPartialRender(t, "", mkGrad("stop-color:#c0392b"))
	if !strings.Contains(hexXML, `stop-color="#c0392b"`) || strings.Contains(hexXML, `stop-color="#1f2937"`) {
		t.Errorf("inline stop-color did not override attribute: %s", hexXML)
	}
	if diff := svgBakeDiffPixels(t, baseImg, hexImg); diff == 0 {
		t.Error("inline stop-color:#c0392b did not change gradient pixels")
	}

	namedXML, namedImg := svgPartialRender(t, "", mkGrad("stop-color:red"))
	if !strings.Contains(namedXML, `stop-color="#ff0000"`) {
		t.Errorf("inline named stop-color:red not converted to hex: %s", namedXML)
	}
	if diff := svgBakeDiffPixels(t, baseImg, namedImg); diff == 0 {
		t.Error("inline stop-color:red did not change gradient pixels")
	}

	// Inline stop-opacity fades the second stop.
	_, fadedImg := svgPartialRender(t, "", mkGrad("stop-color:#c0392b;stop-opacity:40%"))
	if diff := svgBakeDiffPixels(t, hexImg, fadedImg); diff == 0 {
		t.Error("stop-opacity:40% did not fade gradient pixels")
	}
}

func TestBehaviorSvgPartialStrokeLinejoinAndMiterlimit(t *testing.T) {
	t.Parallel()

	// Sharp peak: angle ~35 deg, miter ratio ~3.3 (clipped by limit 1, kept by limit 10).
	sharp := `<path id="p" d="M35 72 L60 12 L85 72" fill="none" stroke="#1f2937" stroke-width="10" stroke-linejoin="miter"/>`
	sharpNoJoin := `<path id="p" d="M35 72 L60 12 L85 72" fill="none" stroke="#1f2937" stroke-width="10"/>`

	_, miterImg := svgPartialRender(t, "",
		`<svg width="120" height="80" viewBox="0 0 120 80">`+sharp+`</svg>`)

	// CSS stroke-linejoin on the element overrides stroke-linejoin="miter" attribute.
	roundXML, roundImg := svgPartialRender(t, `#p{stroke-linejoin:round}`,
		`<svg width="120" height="80" viewBox="0 0 120 80">`+sharp+`</svg>`)
	if !strings.Contains(roundXML, `stroke-linejoin="round"`) {
		t.Errorf("stylesheet stroke-linejoin:round did not override attribute: %s", roundXML)
	}
	if diff := svgBakeDiffPixels(t, miterImg, roundImg); diff == 0 {
		t.Error("stylesheet stroke-linejoin:round identical to miter")
	}

	// Inherited stroke-linejoin on <svg> applies to child without stroke-linejoin attr.
	_, inhRoundImg := svgPartialRender(t, "",
		`<svg width="120" height="80" viewBox="0 0 120 80" style="stroke-linejoin:round">`+sharpNoJoin+`</svg>`)
	if diff := svgBakeDiffPixels(t, miterImg, inhRoundImg); diff == 0 {
		t.Error("inherited stroke-linejoin:round identical to miter")
	}

	// stroke-miterlimit must be emitted BEFORE stroke-linejoin="miter" so canvas applies it.
	limitXML, limitImg := svgPartialRender(t, "",
		`<svg width="120" height="80" viewBox="0 0 120 80" style="stroke-miterlimit:1">`+sharp+`</svg>`)
	miterIdx := strings.LastIndex(limitXML, `stroke-miterlimit="1"`)
	joinIdx := strings.LastIndex(limitXML, `stroke-linejoin="miter"`)
	if miterIdx < 0 || joinIdx < 0 || miterIdx > joinIdx {
		t.Errorf("stroke-miterlimit must precede stroke-linejoin in XML: %s", limitXML)
	}
	if diff := svgBakeDiffPixels(t, miterImg, limitImg); diff == 0 {
		t.Error("inherited stroke-miterlimit:1 with stroke-linejoin=\"miter\" attr did not clip peak")
	}

	// Bare stroke-miterlimit without explicit stroke-linejoin attribute also clips peak.
	_, bareLimitImg := svgPartialRender(t, `#p{stroke-miterlimit:1}`,
		`<svg width="120" height="80" viewBox="0 0 120 80">`+sharpNoJoin+`</svg>`)
	if diff := svgBakeDiffPixels(t, miterImg, bareLimitImg); diff == 0 {
		t.Error("bare stroke-miterlimit:1 without stroke-linejoin attr did not clip peak")
	}
}

func TestBehaviorSvgPartialFillRuleInheritedAndStylesheet(t *testing.T) {
	t.Parallel()

	// Two nested same-winding subpaths: evenodd carves the inner hole.
	holed := `<path id="p" d="M10 10 L110 10 L110 70 L10 70 Z M35 25 L85 25 L85 55 L35 55 Z" fill="#c0392b"/>`
	_, nonzeroImg := svgPartialRender(t, "",
		`<svg width="120" height="80" viewBox="0 0 120 80">`+holed+`</svg>`)

	inhXML, inhImg := svgPartialRender(t, "",
		`<svg width="120" height="80" viewBox="0 0 120 80" style="fill-rule:evenodd">`+holed+`</svg>`)
	if !strings.Contains(inhXML, `fill-rule="evenodd"`) {
		t.Errorf("inherited fill-rule:evenodd not baked onto path: %s", inhXML)
	}
	if diff := svgBakeDiffPixels(t, nonzeroImg, inhImg); diff == 0 {
		t.Error("inherited fill-rule:evenodd did not carve hole")
	}

	sheetXML, sheetImg := svgPartialRender(t, `#p{fill-rule:evenodd}`,
		`<svg width="120" height="80" viewBox="0 0 120 80">`+holed+`</svg>`)
	if !strings.Contains(sheetXML, `fill-rule="evenodd"`) {
		t.Errorf("stylesheet fill-rule:evenodd not baked onto path: %s", sheetXML)
	}
	if diff := svgBakeDiffPixels(t, nonzeroImg, sheetImg); diff == 0 {
		t.Error("stylesheet fill-rule:evenodd did not carve hole")
	}
}

func TestBehaviorSvgPartialColorInterpolationLinearRGB(t *testing.T) {
	t.Parallel()

	mkSVG := func(interp string) string {
		style := ""
		if interp != "" {
			style = ` style="color-interpolation:` + interp + `"`
		}
		return `<svg width="120" height="80" viewBox="0 0 120 80">` +
			`<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="0"` + style + `>` +
			`<stop offset="0" stop-color="#c0392b"/>` +
			`<stop offset="1" stop-color="#3b82f6"/>` +
			`</linearGradient></defs>` +
			`<rect x="10" y="10" width="100" height="60" fill="url(#g)"/></svg>`
	}

	_, srgbImg := svgPartialRender(t, "", mkSVG("sRGB"))
	linXML, linImg := svgPartialRender(t, "", mkSVG("linearRGB"))

	if !strings.Contains(linXML, `color-interpolation="linearRGB"`) {
		t.Errorf("linearRGB color-interpolation not serialized: %s", linXML)
	}
	if strings.Count(linXML, "<stop ") <= 2 {
		t.Errorf("linearRGB gradient stops were not subdivided: %s", linXML)
	}

	if diff := svgBakeDiffPixels(t, srgbImg, linImg); diff == 0 {
		t.Error("color-interpolation:linearRGB gradient raster identical to sRGB")
	}
}

func TestBehaviorSvgPartialColorInterpolationFiltersLighting(t *testing.T) {
	t.Parallel()

	mkSVG := func(interp string) string {
		return `<svg width="120" height="80" viewBox="0 0 120 80">` +
			`<defs><filter id="f" x="0" y="0" width="1" height="1">` +
			`<feDiffuseLighting in="SourceGraphic" lighting-color="#ff8040" style="color-interpolation-filters:` + interp + `">` +
			`<feDistantLight azimuth="45" elevation="30"/>` +
			`</feDiffuseLighting></filter></defs>` +
			`<rect x="10" y="10" width="100" height="60" fill="#3b82f6" filter="url(#f)"/></svg>`
	}

	srgbXML, srgbImg := svgPartialRender(t, "", mkSVG("sRGB"))
	linXML, linImg := svgPartialRender(t, "", mkSVG("linearRGB"))

	if !strings.Contains(srgbXML, `color-interpolation-filters="sRGB"`) {
		t.Errorf("sRGB color-interpolation-filters not serialized: %s", srgbXML)
	}
	if !strings.Contains(linXML, `color-interpolation-filters="linearRGB"`) {
		t.Errorf("linearRGB color-interpolation-filters not serialized: %s", linXML)
	}

	if diff := svgBakeDiffPixels(t, srgbImg, linImg); diff == 0 {
		t.Error("color-interpolation-filters:linearRGB lighting raster identical to sRGB")
	}
}
