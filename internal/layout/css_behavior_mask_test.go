package layout

import (
	"bytes"
	"image/png"
	"math"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

const maskTestPosition = "25% 50%"

// TestBehaviorMaskLayerPropertiesAttachToAllElementPaint pins the group-level
// mask contract against Chrome 143.0.7499.40. A mask owns the element's
// background, border, and descendant text as one paint group.
func TestBehaviorMaskLayerPropertiesAttachToAllElementPaint(t *testing.T) {
	t.Parallel()

	maskPNG := tinyPNG(8, 6)
	res := layoutHTMLWithImages(t, `<html><body style="margin:0">`+
		`<div id="masked" style="width:80px;height:40px;background:#f00;border:2px solid #000;`+
		`mask-image:url(mask.png);mask-mode:luminance;mask-composite:exclude;`+
		`mask-position:25% 50%;mask-repeat:no-repeat;mask-size:contain;`+
		`mask-origin:padding-box;mask-clip:content-box">`+
		`<span>masked text</span><img src="child.png" width="16" height="12">`+
		`</div><div>sibling</div></body></html>`, maskPNG, "")

	group := onlyMaskGroup(t, res)
	if !group.Isolate || group.Mode != "" {
		t.Fatalf("mask group mode/isolation = %q/%v, want empty/true", group.Mode, group.Isolate)
	}

	if len(group.Mask.Layers) != 1 {
		t.Fatalf("mask layer count = %d, want 1", len(group.Mask.Layers))
	}

	layer := group.Mask.Layers[0]
	wantLayer := maskStyleValues{
		image:     "url(mask.png)",
		mode:      maskModeLuminance,
		position:  maskTestPosition,
		size:      "contain",
		repeat:    maskRepeatNoRepeat,
		origin:    "padding-box",
		clip:      "content-box",
		composite: "exclude",
	}

	if got := maskStyleValuesFromLayer(layer); got != wantLayer {
		t.Fatalf("mask layer properties = %+v, want %+v", got, wantLayer)
	}

	if !bytes.Equal(layer.ImageBytes, maskPNG) || layer.PixelWidth != 8 || layer.PixelHeight != 6 {
		t.Fatalf("mask image = %dx%d, %d bytes, want 8x6 and original payload",
			layer.PixelWidth, layer.PixelHeight, len(layer.ImageBytes))
	}

	counts := countMaskPaintOperations(res.Ops, group)

	assertMaskPaintCounts(t, counts)
}

func assertMaskPaintCounts(t *testing.T, counts maskPaintOperationCounts) {
	t.Helper()

	if counts.fills == 0 || counts.strokes == 0 || counts.text == 0 || counts.images == 0 ||
		counts.siblingText == 0 || counts.begins != 1 || counts.ends != 1 {
		t.Fatalf("group paint counts fill=%d stroke=%d text=%d image=%d sibling=%d markers=%d/%d",
			counts.fills, counts.strokes, counts.text, counts.images,
			counts.siblingText, counts.begins, counts.ends)
	}
}

func TestBehaviorMaskGroupsNestAndRepeatLayerValues(t *testing.T) {
	t.Parallel()

	maskPNG := tinyPNG(4, 4)
	res := layoutHTMLWithImages(t, `<html><body style="margin:0">`+
		`<div style="mask-image:url(outer.png),url(inner.png);mask-mode:luminance;`+
		`mask-position:25% 50%;mask-repeat:no-repeat;mask-size:cover;mask-origin:padding-box;`+
		`mask-clip:content-box;mask-composite:subtract,exclude">`+
		`outer <span style="mask-image:url(child.png);mask-mode:alpha">inner text</span>`+
		`</div></body></html>`, maskPNG, "")

	outer, inner := findNestedMaskGroups(res.Ops)

	if outer == nil || inner == nil || inner.Parent != outer {
		t.Fatalf("nested mask groups outer=%p inner=%p parent=%p, want inner parent=outer", outer, inner,
			func() *BlendGroup {
				if inner == nil {
					return nil
				}

				return inner.Parent
			}())
	}

	if got := outer.Mask.Layers[1].Mode; got != maskModeLuminance {
		t.Errorf("repeated mask-mode = %q, want luminance", got)
	}

	if got := outer.Mask.Layers[1].Position; got != maskTestPosition {
		t.Errorf("repeated mask-position = %q, want 25%% 50%%", got)
	}

	if got := outer.Mask.Layers[1].Composite; got != "exclude" {
		t.Errorf("second mask-composite = %q, want exclude", got)
	}
}

func TestBehaviorSVGMaskTypeFlowsIntoMaskImagePayload(t *testing.T) {
	t.Parallel()

	sourceSVG := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="8" height="2" viewBox="0 0 8 2">
<defs><mask id="shape" style="mask-type:luminance">
<rect width="4" height="2" fill="black"/><rect x="4" width="4" height="2" fill="white"/>
</mask></defs></svg>`)
	res := layoutHTMLWithImages(t, `<html><body><div style="width:40px;height:20px;`+
		`mask-image:url(mask.svg#shape);mask-mode:match-source"></div></body></html>`, sourceSVG, "mask.svg")

	group := onlyMaskGroup(t, res)

	layer := group.Mask.Layers[0]
	if !layer.Preprocessed || layer.Mode != "match-source" {
		t.Fatalf("SVG mask preprocessing/mode = %v/%q, want true/match-source", layer.Preprocessed, layer.Mode)
	}

	decoded, err := png.Decode(bytes.NewReader(layer.ImageBytes))
	if err != nil {
		t.Fatalf("decode preprocessed SVG mask: %v", err)
	}

	if decoded.Bounds().Dx() == 0 || decoded.Bounds().Dy() == 0 || layer.PixelWidth != 8 || layer.PixelHeight != 2 {
		t.Fatalf("SVG mask output bounds=%v logical=%dx%d", decoded.Bounds(), layer.PixelWidth, layer.PixelHeight)
	}
}

func TestBehaviorMaskVendorAliasUsesCSSSourceOrder(t *testing.T) {
	t.Parallel()

	cases := []struct {
		style string
		want  string
	}{
		{style: "mask-image:url(standard.png);-webkit-mask-image:url(prefixed.png)", want: "url(prefixed.png)"},
		{style: "-webkit-mask-image:url(prefixed.png);mask-image:url(standard.png)", want: "url(standard.png)"},
		{
			style: "mask:url(standard.png) center / cover no-repeat;-webkit-mask:url(prefixed.png) right / contain no-repeat",
			want:  "url(prefixed.png)",
		},
		{
			style: "-webkit-mask:url(prefixed.png) right / contain no-repeat;mask:url(standard.png) center / cover no-repeat",
			want:  "url(standard.png)",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.want, func(t *testing.T) {
			t.Parallel()
			res := layoutHTMLWithImages(t, `<html><body><div style="width:20px;height:20px;`+
				testCase.style+`"></div></body></html>`, tinyPNG(2, 2), "")

			group := onlyMaskGroup(t, res)
			if got := group.Mask.Layers[0].Source; got != testCase.want {
				t.Fatalf("winning mask-image = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestBehaviorMaskShorthandCreatesElementGroup(t *testing.T) {
	t.Parallel()

	maskPNG := tinyPNG(8, 6)
	res := layoutHTMLWithImages(t, `<html><body><div style="width:80px;height:40px;`+
		`mask:url(mask.png) center / cover no-repeat alpha padding-box exclude, `+
		`url(second.png) 25% 50% / contain repeat luminance content-box subtract">masked</div></body></html>`,
		maskPNG, "")

	group := onlyMaskGroup(t, res)
	if len(group.Mask.Layers) != 2 {
		t.Fatalf("shorthand mask layers = %d, want 2", len(group.Mask.Layers))
	}

	want := []maskStyleValues{
		{
			image: "url(mask.png)", mode: maskModeAlpha, position: "center", size: "cover",
			repeat: maskRepeatNoRepeat, origin: "padding-box", clip: "padding-box", composite: "exclude",
		},
		{
			image: "url(second.png)", mode: maskModeLuminance, position: maskTestPosition, size: "contain",
			repeat: "repeat", origin: "content-box", clip: "content-box", composite: "subtract",
		},
	}
	assertMaskGroupLayers(t, group, want)

	if !bytes.Equal(group.Mask.Layers[0].ImageBytes, maskPNG) {
		t.Fatalf("first shorthand mask image payload differs from input")
	}
}

func TestBehaviorMaskVendorAliasesReachMaskGroup(t *testing.T) {
	t.Parallel()

	maskPNG := tinyPNG(8, 6)

	for _, testCase := range maskAliasCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assertVendorMaskGroupCase(t, maskPNG, testCase)
		})
	}
}

type maskAliasCase struct {
	name          string
	style         string
	wantImage     string
	wantImages    []string
	wantMode      string
	wantPos       string
	wantSize      string
	wantRepeat    string
	wantOrigin    string
	wantClip      string
	wantComposite []string
}

func maskAliasCases() []maskAliasCase {
	return []maskAliasCase{
		{
			name:      "mask shorthand",
			style:     `-webkit-mask:url(mask.png) right / contain no-repeat alpha content-box`,
			wantImage: "url(mask.png)", wantMode: "alpha", wantPos: "right", wantSize: "contain",
			wantRepeat: "no-repeat", wantOrigin: "content-box", wantClip: "content-box",
			wantComposite: []string{"add"},
		},
		{
			name:      "mask clip",
			style:     "mask-image:url(mask.png);-webkit-mask-clip:content-box",
			wantImage: "url(mask.png)", wantMode: "alpha", wantPos: "0% 0%", wantSize: "auto",
			wantRepeat: "repeat", wantOrigin: "border-box", wantClip: "content-box",
			wantComposite: []string{"add"},
		},
		{
			name: "mask composite operators",
			style: "-webkit-mask-image:url(mask.png),url(second.png),url(third.png),url(fourth.png);" +
				"-webkit-mask-composite:source-over,source-in,source-out,xor",
			wantImage: "url(mask.png)", wantMode: "alpha", wantPos: "0% 0%", wantSize: "auto",
			wantImages: []string{"url(mask.png)", "url(second.png)", "url(third.png)", "url(fourth.png)"},
			wantRepeat: "repeat", wantOrigin: "border-box", wantClip: "border-box",
			wantComposite: []string{"add", "intersect", "subtract", "exclude"},
		},
		{
			name:      "mask image",
			style:     "-webkit-mask-image:url(mask.png)",
			wantImage: "url(mask.png)", wantMode: "alpha", wantPos: "0% 0%", wantSize: "auto",
			wantRepeat: "repeat", wantOrigin: "border-box", wantClip: "border-box",
			wantComposite: []string{"add"},
		},
		{
			name:      "mask origin",
			style:     "mask-image:url(mask.png);-webkit-mask-origin:padding-box",
			wantImage: "url(mask.png)", wantMode: "alpha", wantPos: "0% 0%", wantSize: "auto",
			wantRepeat: "repeat", wantOrigin: "padding-box", wantClip: "border-box",
			wantComposite: []string{"add"},
		},
		{
			name:      "mask position",
			style:     "mask-image:url(mask.png);-webkit-mask-position:25% 50%",
			wantImage: "url(mask.png)", wantMode: "alpha", wantPos: "25% 50%", wantSize: "auto",
			wantRepeat: "repeat", wantOrigin: "border-box", wantClip: "border-box",
			wantComposite: []string{"add"},
		},
		{
			name:      "mask repeat",
			style:     "mask-image:url(mask.png);-webkit-mask-repeat:no-repeat",
			wantImage: "url(mask.png)", wantMode: "alpha", wantPos: "0% 0%", wantSize: "auto",
			wantRepeat: "no-repeat", wantOrigin: "border-box", wantClip: "border-box",
			wantComposite: []string{"add"},
		},
		{
			name:      "mask size",
			style:     "mask-image:url(mask.png);-webkit-mask-size:contain",
			wantImage: "url(mask.png)", wantMode: "alpha", wantPos: "0% 0%", wantSize: "contain",
			wantRepeat: "repeat", wantOrigin: "border-box", wantClip: "border-box",
			wantComposite: []string{"add"},
		},
	}
}

func assertVendorMaskGroupCase(t *testing.T, maskPNG []byte, testCase maskAliasCase) {
	t.Helper()

	res := layoutHTMLWithImages(t, `<html><body><div style="width:40px;height:20px;`+
		testCase.style+`"></div></body></html>`, maskPNG, "")
	group := onlyMaskGroup(t, res)
	want := maskAliasCaseValues(testCase)
	assertMaskGroupLayers(t, group, want)
}

func maskAliasCaseValues(testCase maskAliasCase) []maskStyleValues {
	want := make([]maskStyleValues, len(testCase.wantComposite))

	for idx, composite := range testCase.wantComposite {
		image := testCase.wantImage
		if len(testCase.wantImages) > idx {
			image = testCase.wantImages[idx]
		}

		want[idx] = maskStyleValues{
			image: image, mode: testCase.wantMode,
			position: testCase.wantPos, size: testCase.wantSize,
			repeat: testCase.wantRepeat, origin: testCase.wantOrigin,
			clip: testCase.wantClip, composite: composite,
		}
	}

	return want
}

func TestBehaviorMaskMatchSourceUsesImageAlpha(t *testing.T) {
	t.Parallel()

	res := layoutHTMLWithImages(t, `<html><body><div style="width:20px;height:20px;`+
		`mask-image:url(mask.png);mask-mode:match-source"></div></body></html>`, tinyPNG(2, 2), "mask.png")
	if got := onlyMaskGroup(t, res).Mask.Layers[0].Mode; got != "alpha" {
		t.Fatalf("raster image match-source mode = %q, want alpha", got)
	}
}

func TestBehaviorMaskSupportsMatchesAcceptedValues(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>`+
		`<div id="url">url</div><div id="gradient">gradient</div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		#url { mask-image:none }
		@supports (mask-image:url(mask.png)) { #url { mask-image:url(valid.png) } }
		#gradient { mask-image:none }
		@supports (mask-image:linear-gradient(red, blue)) {
			#gradient { mask-image:linear-gradient(red, blue) }
		}
		@supports (mask-image:image-set(url(b.png) 1x)) {
			#gradient { mask-image:image-set(url(b.png) 1x) }
		}
		@supports (mask-image:image-set()) { #gradient { mask-image:url(empty-set.png) } }
		@supports (mask-image:not-an-image()) { #gradient { mask-image:url(rejected.png) } }
		@supports (-webkit-mask-composite:source-out) { #gradient { -webkit-mask-composite:source-out } }
	`)}, "print", testViewport, 800)

	urlFeature := styles[findElementByID(t, root, "url")]
	if urlFeature.MaskImage != "url(valid.png)" {
		t.Fatalf("@supports url() mask-image = %q, want url(valid.png)", urlFeature.MaskImage)
	}

	gradient := styles[findElementByID(t, root, "gradient")]
	if gradient.MaskImage != "image-set(url(b.png) 1x)" || gradient.MaskComposite != "subtract" {
		t.Fatalf("@supports mask result = image:%q composite:%q", gradient.MaskImage, gradient.MaskComposite)
	}
}

func TestBehaviorMaskInvalidValuesDoNotReplaceEarlierDeclarations(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div id="feature"></div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		#feature { mask-image:url(earlier.png); mask-size:cover }
		#feature { mask-image:not-an-image(); mask-size:-1px }
	`)}, "print", testViewport, 800)

	feature := styles[findElementByName(root, "div")]
	if feature.MaskImage != "url(earlier.png)" || feature.MaskSize != "cover" {
		t.Fatalf("invalid mask declarations won: image=%q size=%q", feature.MaskImage, feature.MaskSize)
	}
}

func TestBehaviorMaskNoneDoesNotCreateMaskGroup(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div style="mask:none">shorthand</div>`+
		`<div style="mask-image:none">longhand</div></body></html>`)
	for idx := range res.Ops {
		if group := res.Ops[idx].Group(); group != nil && group.Mask != nil {
			t.Fatalf("mask:none created a mask group: %+v", group)
		}
	}
}

func TestBehaviorMaskGradientBytesOnGroup(t *testing.T) {
	t.Parallel()

	assertGradientMaskRaster(t, "linear-gradient(red, blue)")
}

//nolint:wsl // the image provider and the layout result stay in one test
func TestBehaviorMaskImageSetPicks1x(t *testing.T) {
	t.Parallel()

	wide := tinyPNG(4, 2)
	square := tinyPNG(2, 2)
	root := mustParse(t, `<html><body><div style="width:20px;height:10px;`+
		`mask-image:image-set(url(a.png) 2x, url(b.png) 1x)"></div></body></html>`)
	res, err := Layout(root, Options{
		Width: testViewport, Height: 800, Background: true,
		Images: func(src string) ([]byte, error) {
			switch src {
			case "a.png":
				return wide, nil
			case "b.png":
				return square, nil
			default:
				t.Errorf("image src %q", src)

				return nil, nil
			}
		},
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	layer := onlyMaskGroup(t, res).Mask.Layers[0]
	if layer.Source != "url(b.png)" {
		t.Fatalf("image-set source = %q, want url(b.png)", layer.Source)
	}

	if layer.PixelWidth != 2 || layer.PixelHeight != 2 {
		t.Fatalf("image-set pixels = %d x %d, want the 1x candidate 2 x 2", layer.PixelWidth, layer.PixelHeight)
	}

	if !bytes.Equal(layer.ImageBytes, square) {
		t.Fatal("image-set bytes are not the 1x candidate")
	}
}

//nolint:wsl // the group read follows the layout call
func TestBehaviorMaskImageSetGradientCandidate(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><div style="width:20px;height:10px;`+
		`mask-image:image-set(linear-gradient(red, blue) 1x)"></div></body></html>`)
	layer := onlyMaskGroup(t, res).Mask.Layers[0]
	if layer.Source != "linear-gradient(red, blue)" {
		t.Fatalf("gradient image-set source = %q", layer.Source)
	}

	if len(layer.ImageBytes) < 4 || layer.ImageBytes[0] != 0x89 {
		t.Fatal("chosen gradient did not rasterize to a PNG")
	}
}

//nolint:wsl // the style read follows resolveStyles
func TestBehaviorMaskImageSetEmptyDoesNotReplace(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div id="feature"></div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		#feature { mask-image:url(earlier.png) }
		#feature { mask-image:image-set() }
	`)}, "print", testViewport, 800)
	feature := styles[findElementByName(root, "div")]
	if feature.MaskImage != "url(earlier.png)" {
		t.Fatalf("empty image-set replaced mask-image with %q", feature.MaskImage)
	}
}

func TestBehaviorMaskRepeatingGradientBytes(t *testing.T) {
	t.Parallel()

	// parseLinearGradient accepts the repeating-linear-gradient prefix, so this
	// layer is a border-box PNG. The declaration is not dropped in favor of an
	// earlier url().
	assertGradientMaskRaster(t, "repeating-linear-gradient(red, blue)")
}

func onlyMaskGroup(t *testing.T, res *Result) *BlendGroup {
	t.Helper()

	var found *BlendGroup

	for idx := range res.Ops {
		group := res.Ops[idx].Group()
		if group == nil || group.Mask == nil {
			continue
		}

		if found != nil && found != group {
			t.Fatal("found more than one mask group")
		}

		found = group
	}

	if found == nil {
		t.Fatal("no mask group in display list")
	}

	return found
}

type maskPaintOperationCounts struct {
	fills       int
	strokes     int
	text        int
	images      int
	siblingText int
	begins      int
	ends        int
}

func countMaskPaintOperations(ops []Op, group *BlendGroup) maskPaintOperationCounts {
	var counts maskPaintOperationCounts

	for idx := range ops {
		operation := &ops[idx]
		countMaskGroupOperation(operation, group, &counts)
		countSiblingText(operation, group, &counts)
		countMaskBoundary(operation, group, &counts)
	}

	return counts
}

func countMaskGroupOperation(operation *Op, group *BlendGroup, counts *maskPaintOperationCounts) {
	if operation.Group() != group {
		return
	}

	switch operation.Kind {
	case OpFillRect:
		counts.fills++
	case OpStrokeRect, OpLine:
		counts.strokes++
	case OpText:
		if strings.Contains(operation.Text, "masked") || strings.Contains(operation.Text, "text") {
			counts.text++
		}
	case OpImage:
		counts.images++
	case OpUnknown, OpLinkURI, OpBullet, OpGridRun, opKindNoop:
		return
	}
}

func countSiblingText(operation *Op, group *BlendGroup, counts *maskPaintOperationCounts) {
	if operation.Kind == OpText && operation.Text == "sibling" && operation.Group() != group {
		counts.siblingText++
	}
}

func countMaskBoundary(operation *Op, group *BlendGroup, counts *maskPaintOperationCounts) {
	if operation.Group() != group {
		return
	}

	if operation.IsGroupBegin() {
		counts.begins++
	} else if operation.IsGroupEnd() {
		counts.ends++
	}
}

func findNestedMaskGroups(ops []Op) (*BlendGroup, *BlendGroup) {
	var outer, inner *BlendGroup

	for idx := range ops {
		group := ops[idx].Group()
		if group == nil || group.Mask == nil {
			continue
		}

		if len(group.Mask.Layers) == 2 {
			outer = group
		} else if len(group.Mask.Layers) == 1 {
			inner = group
		}
	}

	return outer, inner
}

func assertMaskGroupLayers(t *testing.T, group *BlendGroup, want []maskStyleValues) {
	t.Helper()

	if len(group.Mask.Layers) != len(want) {
		t.Fatalf("mask layer count = %d, want %d", len(group.Mask.Layers), len(want))
	}

	for idx, layer := range group.Mask.Layers {
		if got := maskStyleValuesFromLayer(layer); got != want[idx] {
			t.Errorf("mask layer %d properties = %+v, want %+v", idx, got, want[idx])
		}
	}
}

func maskStyleValuesFromLayer(layer MaskLayer) maskStyleValues {
	return maskStyleValues{
		image:     layer.Source,
		mode:      layer.Mode,
		position:  layer.Position,
		size:      layer.Size,
		repeat:    layer.Repeat,
		origin:    layer.Origin,
		clip:      layer.Clip,
		composite: layer.Composite,
	}
}

func layoutGradientMaskBox(t *testing.T, maskImage string) *Result {
	t.Helper()

	style := "width:20px;height:10px;background-color:#00ff00"
	if maskImage != "" {
		style += ";mask-image:" + maskImage
	}

	return layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="masked" style="`+style+`"></div></body></html>`)
}

//nolint:cyclop // one helper checks bytes, size, and the unmasked twin
func assertGradientMaskRaster(t *testing.T, gradient string) {
	t.Helper()

	res := layoutGradientMaskBox(t, gradient)
	plain := layoutGradientMaskBox(t, "")
	group := onlyMaskGroup(t, res)

	if len(group.Mask.Layers) != 1 {
		t.Fatalf("mask layer count = %d, want 1", len(group.Mask.Layers))
	}

	layer := group.Mask.Layers[0]
	if len(layer.ImageBytes) == 0 || !bytes.HasPrefix(layer.ImageBytes, []byte{0x89, 'P', 'N', 'G'}) {
		t.Fatalf("gradient mask bytes = %d, want a non-empty PNG", len(layer.ImageBytes))
	}

	masked := boxByID(t, res, "masked")
	wantW := int(math.Ceil(masked.w))
	wantH := int(math.Ceil(masked.height))
	// 20px is 15pt and 10px is 7.5pt, so those ceils are 15 and 8.
	fromPxW := int(math.Ceil(pxToPt(20)))
	fromPxH := int(math.Ceil(pxToPt(10)))
	pixelsInRange := layer.PixelWidth > 0 && layer.PixelWidth <= 512 &&
		layer.PixelHeight > 0 && layer.PixelHeight <= 512

	if layer.PixelWidth != wantW || layer.PixelHeight != wantH || !pixelsInRange {
		t.Fatalf("mask pixels = %dx%d, want ceil(border box) %dx%d with both in 1..512",
			layer.PixelWidth, layer.PixelHeight, wantW, wantH)
	}

	if wantW != fromPxW || wantH != fromPxH {
		t.Fatalf("border box ceil = %dx%d, want %dx%d for 20px by 10px with no padding",
			wantW, wantH, fromPxW, fromPxH)
	}

	if !strings.Contains(layer.Source, gradient) {
		t.Fatalf("mask source = %q, want it to contain %q", layer.Source, gradient)
	}

	if layer.Preprocessed {
		t.Fatal("gradient mask Preprocessed = true, want false")
	}

	fills := len(opsOfKind(res, OpFillRect))
	plainFills := len(opsOfKind(plain, OpFillRect))

	if fills != plainFills {
		t.Fatalf("OpFillRect count = %d, twin = %d; gradient must not add a fill", fills, plainFills)
	}

	images := len(opsOfKind(res, OpImage))
	plainImages := len(opsOfKind(plain, OpImage))

	if images != plainImages {
		t.Fatalf("OpImage count = %d, twin = %d; gradient must not add an image op", images, plainImages)
	}

	if behaviorBordersFindFill(res.Ops, masked.x, masked.y, masked.w, masked.height, 0, 1, 0) == nil {
		t.Fatal("green background fill missing")
	}
}
