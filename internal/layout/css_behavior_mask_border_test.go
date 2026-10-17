package layout

import (
	"bytes"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// Shared test image tokens. goconst counts repeated string literals across
// the package, so the mask-border fixtures name each once.
const (
	maskBorderTestSource  = "url(border.png)"
	maskBorderTestEarlier = "url(earlier.png)"
)

// A mask-border is a mask for the host, not a painted border image. Layout
// records the frame on the element group and emits no OpImage cells for it.
// Slice insets are source pixels in top, right, bottom, left order; width and
// outset are points. Reference: Chrome 155 computed values for the same
// declarations.

func TestBehaviorMaskBorderDescriptorNotPaint(t *testing.T) {
	t.Parallel()

	borderPNG := tinyPNG(8, 6)
	res := layoutHTMLWithImages(t, `<html style="margin:0"><body style="margin:0">`+
		`<div style="width:80px;height:40px;background:#0f0;`+
		`mask-border-source:url(border.png);mask-border-slice:1 2 3 4 fill;`+
		`mask-border-width:10px;mask-border-outset:10px;`+
		`mask-border-repeat:repeat;mask-border-mode:luminance"></div></body></html>`,
		borderPNG, "border.png")
	plain := layoutHTMLWithImages(t, `<html style="margin:0"><body style="margin:0">`+
		`<div style="width:80px;height:40px;background:#0f0"></div></body></html>`,
		borderPNG, "border.png")

	group := onlyMaskBorderGroup(t, res)
	if group.Mask != nil {
		t.Fatalf("mask-border-only element gained a first-wave Mask: %+v", group.Mask)
	}

	assertMaskBorderSourcePayload(t, group.MaskBorder, borderPNG)
	assertMaskBorderScaledFrame(t, group.MaskBorder)
	maskBorderAssertNoCells(t, res, plain)
}

// assertMaskBorderSourcePayload pins the source token, slice insets, and fill
// flag of the rich no-paint fixture.
func assertMaskBorderSourcePayload(t *testing.T, frame *MaskBorder, png []byte) {
	t.Helper()

	if frame.Source != maskBorderTestSource || frame.PixelWidth != 8 || frame.PixelHeight != 6 ||
		!bytes.Equal(frame.ImageBytes, png) {
		t.Fatalf("frame source/payload = %q, %d bytes, %dx%d; want url(border.png) and the 8x6 payload",
			frame.Source, len(frame.ImageBytes), frame.PixelWidth, frame.PixelHeight)
	}

	if frame.Slice != [4]float64{1, 2, 3, 4} || !frame.SliceFill {
		t.Fatalf("frame slice/fill = %v/%v, want [1 2 3 4] and true", frame.Slice, frame.SliceFill)
	}
}

// assertMaskBorderScaledFrame pins the width, outset, repeat, and mode of the
// rich no-paint fixture. 10px is 7.5pt.
func assertMaskBorderScaledFrame(t *testing.T, frame *MaskBorder) {
	t.Helper()

	if frame.Width != [4]float64{7.5, 7.5, 7.5, 7.5} || frame.Outset != [4]float64{7.5, 7.5, 7.5, 7.5} {
		t.Fatalf("frame width/outset = %v/%v, want 7.5pt on every side", frame.Width, frame.Outset)
	}

	if frame.Repeat != "repeat" || frame.Mode != maskModeLuminance {
		t.Fatalf("frame repeat/mode = %q/%q, want repeat/luminance", frame.Repeat, frame.Mode)
	}
}

// maskBorderAssertNoCells proves the frame paints nothing: the twin document
// differs only by the mask-border declarations, so both the OpImage count and
// the border-image cell count must match, and the twin has no cells at all.
func maskBorderAssertNoCells(t *testing.T, res, plain *Result) {
	t.Helper()

	if got, want := len(opsOfKind(res, OpImage)), len(opsOfKind(plain, OpImage)); got != want {
		t.Fatalf("OpImage count = %d, twin = %d; a mask-border must not add image ops", got, want)
	}

	cells := len(behaviorBgImages(opsOfKind(res, OpImage)))
	plainCells := len(behaviorBgImages(opsOfKind(plain, OpImage)))

	if cells != plainCells || cells != 0 {
		t.Fatalf("border-image cells = %d, twin = %d, want 0 in both", cells, plainCells)
	}
}

// TestBehaviorMaskBorderAliasReachesGroup drives each of the six
// -webkit-mask-box-image* aliases and checks the descriptor field that alias
// names.
func TestBehaviorMaskBorderAliasReachesGroup(t *testing.T) {
	t.Parallel()

	borderPNG := tinyPNG(8, 6)

	for _, testCase := range maskBorderAliasCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTMLWithImages(t, `<html><body><div style="width:40px;height:20px;border:10px solid #000;`+
				testCase.style+`"></div></body></html>`, borderPNG, "border.png")
			assertMaskBorderAliasWant(t, onlyMaskBorderGroup(t, res).MaskBorder, testCase.want)
		})
	}
}

// maskBorderAliasCases lists the six -webkit-mask-box-image* aliases, one
// declaration per alias, and the whole frame each alias is expected to reach.
// The alias div carries a 10px border, so an auto mask-border-width resolves
// to 7.5pt.
func maskBorderAliasCases() []maskBorderAliasCase {
	return []maskBorderAliasCase{
		{
			name:  "shorthand",
			style: "-webkit-mask-box-image:url(border.png) 1 2 3 4 fill / 10px / 5px repeat alpha",
			want: maskBorderAliasWant{
				source: maskBorderTestSource, slice: [4]float64{1, 2, 3, 4}, fill: true,
				width: [4]float64{7.5, 7.5, 7.5, 7.5}, outset: [4]float64{3.75, 3.75, 3.75, 3.75},
				repeat: "repeat", mode: maskModeAlpha,
			},
		},
		{
			name:  "source",
			style: "mask-border-slice:1 2 3 4;-webkit-mask-box-image-source:url(border.png)",
			want: maskBorderAliasWant{
				source: maskBorderTestSource, slice: [4]float64{1, 2, 3, 4},
				width: maskBorderAliasAutoWidth(), repeat: fxStretch, mode: maskModeAlpha,
			},
		},
		{
			name:  "slice",
			style: "mask-border-source:url(border.png);-webkit-mask-box-image-slice:1 2 3 4 fill",
			want: maskBorderAliasWant{
				source: maskBorderTestSource, slice: [4]float64{1, 2, 3, 4}, fill: true,
				width: maskBorderAliasAutoWidth(), repeat: fxStretch, mode: maskModeAlpha,
			},
		},
		{
			name:  "width",
			style: "mask-border-source:url(border.png);-webkit-mask-box-image-width:10px",
			want: maskBorderAliasWant{
				source: maskBorderTestSource, width: maskBorderAliasAutoWidth(),
				repeat: fxStretch, mode: maskModeAlpha,
			},
		},
		{
			name:  "outset",
			style: "mask-border-source:url(border.png);-webkit-mask-box-image-outset:10px",
			want: maskBorderAliasWant{
				source: maskBorderTestSource, width: maskBorderAliasAutoWidth(),
				outset: [4]float64{7.5, 7.5, 7.5, 7.5}, repeat: fxStretch, mode: maskModeAlpha,
			},
		},
		{
			name:  "repeat",
			style: "mask-border-source:url(border.png);-webkit-mask-box-image-repeat:space round",
			want: maskBorderAliasWant{
				source: maskBorderTestSource, width: maskBorderAliasAutoWidth(),
				repeat: "space round", mode: maskModeAlpha,
			},
		},
	}
}

type maskBorderAliasCase struct {
	name  string
	style string
	want  maskBorderAliasWant
}

// maskBorderAliasWant is the whole frame one alias case expects. The zero
// slice and outset are the property initials.
type maskBorderAliasWant struct {
	source string
	slice  [4]float64
	width  [4]float64
	outset [4]float64
	repeat string
	mode   string
	fill   bool
}

// maskBorderAliasAutoWidth is the used auto width: the alias div's 10px
// border is 7.5pt on every side.
func maskBorderAliasAutoWidth() [4]float64 {
	return [4]float64{7.5, 7.5, 7.5, 7.5}
}

func assertMaskBorderAliasWant(t *testing.T, frame *MaskBorder, want maskBorderAliasWant) {
	t.Helper()

	got := maskBorderAliasWant{
		source: frame.Source,
		slice:  frame.Slice,
		width:  frame.Width,
		outset: frame.Outset,
		repeat: frame.Repeat,
		mode:   frame.Mode,
		fill:   frame.SliceFill,
	}

	if got != want {
		t.Fatalf("frame = %+v, want %+v", got, want)
	}
}

// TestBehaviorMaskBorderVendorAliasUsesCSSSourceOrder pins alias/standard
// source order in both declaration orders, for the longhand and the shorthand.
func TestBehaviorMaskBorderVendorAliasUsesCSSSourceOrder(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		style string
		want  string
	}{
		{
			name:  "prefixed longhand comes later",
			style: "mask-border-source:url(standard.png);-webkit-mask-box-image-source:url(prefixed.png)",
			want:  "url(prefixed.png)",
		},
		{
			name:  "standard longhand comes later",
			style: "-webkit-mask-box-image-source:url(prefixed.png);mask-border-source:url(standard.png)",
			want:  "url(standard.png)",
		},
		{
			name:  "prefixed shorthand comes later",
			style: "mask-border:url(standard.png);-webkit-mask-box-image:url(prefixed.png)",
			want:  "url(prefixed.png)",
		},
		{
			name:  "standard shorthand comes later",
			style: "-webkit-mask-box-image:url(prefixed.png);mask-border:url(standard.png)",
			want:  "url(standard.png)",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTMLWithImages(t, `<html><body><div style="width:20px;height:20px;`+
				testCase.style+`"></div></body></html>`, tinyPNG(2, 2), "")
			frame := onlyMaskBorderGroup(t, res).MaskBorder

			if frame.Source != testCase.want {
				t.Fatalf("winning mask-border-source = %q, want %q", frame.Source, testCase.want)
			}
		})
	}
}

// TestBehaviorMaskAndMaskBorderCoexist pins that the first-wave mask and the
// mask-border frame share the group without clobbering each other.
func TestBehaviorMaskAndMaskBorderCoexist(t *testing.T) {
	t.Parallel()

	res := layoutHTMLWithImages(t, `<html><body><div style="width:40px;height:20px;`+
		`mask-image:url(mask.png);mask-border-source:url(border.png)"></div></body></html>`,
		tinyPNG(2, 2), "")

	var found *BlendGroup

	for idx := range res.Ops {
		group := res.Ops[idx].Group()
		if group == nil || group.Mask == nil || group.MaskBorder == nil {
			continue
		}

		if found != nil && found != group {
			t.Fatal("found more than one group carrying both mask frames")
		}

		found = group
	}

	if found == nil {
		t.Fatal("no group carries both mask frames")
	}

	if got := found.MaskBorder.Source; got != maskBorderTestSource {
		t.Fatalf("mask-border source = %q, want url(border.png)", got)
	}

	if got := found.Mask.Layers[0].Source; got != "url(mask.png)" {
		t.Fatalf("mask layer source = %q, want url(mask.png)", got)
	}
}

// TestBehaviorMaskBorderInvalidValuesKeepEarlier proves the acceptance gate:
// an invalid longhand, a gradient source, and a bogus prefixed shorthand do
// not replace an earlier valid declaration.
func TestBehaviorMaskBorderInvalidValuesKeepEarlier(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><div id="feature"></div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		#feature { mask-border-source:url(earlier.png); mask-border-slice:2 fill;
			mask-border-width:4px; mask-border-outset:1px }
		#feature { mask-border-source:not-an-image(); mask-border-slice:-1;
			mask-border-width:-2px; mask-border-outset:-1px }
		#feature { mask-border-source:linear-gradient(red, blue); mask-border-repeat:bogus; mask-border-mode:bogus }
		#feature { -webkit-mask-box-image:not-an-image 30 }
	`)}, "print", testViewport, 800)

	feature := styles[findElementByName(root, "div")]
	if feature.MaskBorderSource != maskBorderTestEarlier {
		t.Fatalf("invalid declarations replaced mask-border-source with %q", feature.MaskBorderSource)
	}

	if feature.MaskBorderSlice != "2 fill" || feature.MaskBorderWidth != "4px" || feature.MaskBorderOutset != "1px" {
		t.Fatalf("invalid declarations replaced slice/width/outset with %q/%q/%q",
			feature.MaskBorderSlice, feature.MaskBorderWidth, feature.MaskBorderOutset)
	}

	if feature.MaskBorderRepeat != "stretch" || feature.MaskBorderMode != maskModeAlpha {
		t.Fatalf("invalid declarations replaced repeat/mode with %q/%q",
			feature.MaskBorderRepeat, feature.MaskBorderMode)
	}
}

// TestBehaviorMaskBorderSupportsAcceptedValues pins the @supports probe
// against the same acceptance rule as the cascade: the shorthand body with a
// prefix matches, a negative slice does not, and a prefixed source matches.
func TestBehaviorMaskBorderSupportsAcceptedValues(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>`+
		`<div id="shorthand">s</div><div id="slice">sl</div>`+
		`<div id="alias">a</div><div id="bogus">b</div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		#shorthand { mask-border-source:none }
		@supports (mask-border: url(border.png) 30 fill / 10px / 10px stretch alpha) {
			#shorthand { mask-border-source:url(ok.png) }
		}
		#slice { mask-border-source:url(keep.png) }
		@supports (mask-border-slice: -1) { #slice { mask-border-source:url(rejected.png) } }
		#alias { mask-border-source:none }
		@supports (-webkit-mask-box-image-source: url(prefixed.png)) {
			#alias { mask-border-source:url(alias.png) }
		}
		#bogus { mask-border-source:url(keep2.png) }
		@supports (-webkit-mask-box-image-slice: -5) { #bogus { mask-border-source:url(rejected2.png) } }
	`)}, "print", testViewport, 800)

	if got := styles[findElementByID(t, root, "shorthand")].MaskBorderSource; got != "url(ok.png)" {
		t.Fatalf("shorthand @supports source = %q, want url(ok.png)", got)
	}

	if got := styles[findElementByID(t, root, "slice")].MaskBorderSource; got != "url(keep.png)" {
		t.Fatalf("negative slice satisfied @supports; source = %q, want url(keep.png)", got)
	}

	if got := styles[findElementByID(t, root, "alias")].MaskBorderSource; got != "url(alias.png)" {
		t.Fatalf("prefixed source @supports result = %q, want url(alias.png)", got)
	}

	if got := styles[findElementByID(t, root, "bogus")].MaskBorderSource; got != "url(keep2.png)" {
		t.Fatalf("prefixed bogus slice satisfied @supports; source = %q, want url(keep2.png)", got)
	}
}

// onlyMaskBorderGroup returns the one group carrying a mask-border frame.
func onlyMaskBorderGroup(t *testing.T, res *Result) *BlendGroup {
	t.Helper()

	var found *BlendGroup

	for idx := range res.Ops {
		group := res.Ops[idx].Group()
		if group == nil || group.MaskBorder == nil {
			continue
		}

		if found != nil && found != group {
			t.Fatal("found more than one mask-border group")
		}

		found = group
	}

	if found == nil {
		t.Fatal("no mask-border group in display list")
	}

	return found
}
