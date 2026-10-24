package layout

import (
	"bytes"
	"image/png"
	"math"
	"strconv"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// svgPaintLayout lays out one inline SVG page and returns the result and the
// <svg> node the retained box tree placed.
func svgPaintLayout(t *testing.T, source string) (*Result, *html.Node) {
	t.Helper()

	root := mustParse(t, source)

	res, err := Layout(root, Options{Width: testViewport, Height: 800, Background: true})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	boxNode := res.boxForID("icon")
	if boxNode == nil || boxNode.node == nil {
		t.Fatal("inline svg has no placed box")
	}

	return res, boxNode.node
}

// svgPaintOp returns the SVG raster operation of the laid-out element.
func svgPaintOp(t *testing.T, res *Result) Op {
	t.Helper()

	index := svgImageOpIndex(res.Ops, res.boxForID("icon"))
	if index < 0 {
		t.Fatal("inline svg has no image operation")
	}

	return res.Ops[index]
}

// svgPaintRaster decodes an image operation's payload and its logical pixel
// size.
func svgPaintRaster(t *testing.T, op Op) ([]byte, int, int) {
	t.Helper()

	data, width, height := op.ImageBytes()
	if len(data) == 0 || width <= 0 || height <= 0 {
		t.Fatalf("image payload = %d bytes, %dx%d, want a raster", len(data), width, height)
	}

	return data, width, height
}

// svgPaintPixel samples one point from the logical SVG coordinate space. The
// raster may be supersampled, so the point scales with the decoded bounds.
func svgPaintPixel(
	t *testing.T, data []byte, logicalW, logicalH int, pixelX, pixelY float64,
) (uint8, uint8, uint8, uint8) {
	t.Helper()

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode raster: %v", err)
	}

	bounds := img.Bounds()
	px := bounds.Min.X + int(math.Floor(pixelX*float64(bounds.Dx())/float64(logicalW)))
	py := bounds.Min.Y + int(math.Floor(pixelY*float64(bounds.Dy())/float64(logicalH)))

	red, green, blue, alpha := img.At(px, py).RGBA()

	// RGBA returns 16-bit channels; the shift keeps the high 8 bits, the
	// value a PNG decoder would report.
	//nolint:gosec // the high byte of a 16-bit channel is in range
	return uint8(red >> 8), uint8(green >> 8), uint8(blue >> 8), uint8(alpha >> 8)
}

// svgPaintWantColor fails when a sampled pixel is not the expected opaque
// color. The tolerance absorbs the rasterizer's antialiasing.
func svgPaintWantColor(
	t *testing.T, label string, gotR, gotG, gotB, gotA uint8, wantR, wantG, wantB uint8,
) {
	t.Helper()

	const tolerance = 8

	closeTo := func(got, want uint8) bool {
		return int(got) >= int(want)-tolerance && int(got) <= int(want)+tolerance
	}

	if !closeTo(gotR, wantR) || !closeTo(gotG, wantG) || !closeTo(gotB, wantB) || gotA < 250 {
		t.Fatalf("%s = rgba(%d, %d, %d, %d), want opaque (%d, %d, %d)",
			label, gotR, gotG, gotB, gotA, wantR, wantG, wantB)
	}
}

// TestRestyleSVGPaintFillChangesRaster overrides the root fill of an inline
// SVG whose shape has no paint of its own. The shape inherits the new fill and
// the raster pixels change.
func TestRestyleSVGPaintFillChangesRaster(t *testing.T) {
	t.Parallel()

	res, node := svgPaintLayout(t, `<html><body style="margin:0">`+
		`<svg id="icon" style="display:block" width="64" height="28" viewBox="0 0 64 28" xmlns="http://www.w3.org/2000/svg">`+
		`<rect x="2" y="2" width="60" height="24"/>`+
		`</svg></body></html>`)

	oldData, _, _ := svgPaintRaster(t, svgPaintOp(t, res))

	edits, err := RestyleSVGPaint(t.Context(), res, []SVGPaintRequest{{
		Node: node,
		Overrides: []SVGPaintOverride{{
			ElementID: "icon",
			Fill:      "#ff0000",
			HasFill:   true,
		}},
	}}, Options{Width: testViewport, Height: 800})
	if err != nil {
		t.Fatalf("RestyleSVGPaint: %v", err)
	}

	if len(edits) != 1 {
		t.Fatalf("edits = %d, want 1", len(edits))
	}

	newData, width, height := svgPaintRaster(t, edits[0].NewImage)
	if bytes.Equal(oldData, newData) {
		t.Fatal("fill override left the raster unchanged")
	}

	red, green, blue, alpha := svgPaintPixel(t, newData, width, height, 32, 14)
	svgPaintWantColor(t, "restyled fill", red, green, blue, alpha, 255, 0, 0)
}

// TestRestyleSVGPaintNoOverridesLeavesDisplay pins the no-op contract: no
// overrides means no serialization, no raster work, and the same operations.
//
//nolint:cyclop // the fixture compares every field a no-op must keep
func TestRestyleSVGPaintNoOverridesLeavesDisplay(t *testing.T) {
	t.Parallel()

	res, node := svgPaintLayout(t, `<html><body style="margin:0">`+
		`<svg id="icon" style="display:block" width="64" height="28" viewBox="0 0 64 28" xmlns="http://www.w3.org/2000/svg">`+
		`<rect x="2" y="2" width="60" height="24" style="fill:#00ff00"/>`+
		`</svg></body></html>`)

	before := append([]Op(nil), res.Ops...)
	oldData, _, _ := svgPaintRaster(t, svgPaintOp(t, res))

	edits, err := RestyleSVGPaint(t.Context(), res, []SVGPaintRequest{{
		Node:      node,
		Overrides: nil,
	}}, Options{Width: testViewport, Height: 800})
	if err != nil {
		t.Fatalf("RestyleSVGPaint: %v", err)
	}

	if edits != nil {
		t.Fatalf("edits = %d, want nil", len(edits))
	}

	if len(res.Ops) != len(before) {
		t.Fatalf("operation count = %d, want %d", len(res.Ops), len(before))
	}

	for index := range before {
		if before[index].Kind != res.Ops[index].Kind ||
			before[index].ID != res.Ops[index].ID ||
			before[index].X != res.Ops[index].X ||
			before[index].Y != res.Ops[index].Y ||
			before[index].W != res.Ops[index].W ||
			before[index].H != res.Ops[index].H {
			t.Fatalf("operation %d changed: %+v -> %+v", index, before[index], res.Ops[index])
		}
	}

	newData, _, _ := svgPaintRaster(t, svgPaintOp(t, res))
	if !bytes.Equal(oldData, newData) {
		t.Fatal("no-override call changed the raster")
	}
}

// TestRestyleSVGPaintRootFillInherits checks the inheritance rule: a shape
// with no paint follows the root override, and a shape with its own paint does
// not.
func TestRestyleSVGPaintRootFillInherits(t *testing.T) {
	t.Parallel()

	res, node := svgPaintLayout(t, `<html><body style="margin:0">`+
		`<svg id="icon" style="display:block" width="80" height="30" viewBox="0 0 80 30" xmlns="http://www.w3.org/2000/svg">`+
		`<rect x="2" y="2" width="36" height="26"/>`+
		`<rect x="42" y="2" width="36" height="26" style="fill:#0000ff"/>`+
		`</svg></body></html>`)

	edits, err := RestyleSVGPaint(t.Context(), res, []SVGPaintRequest{{
		Node: node,
		Overrides: []SVGPaintOverride{{
			ElementID: "icon",
			Fill:      "#ff0000",
			HasFill:   true,
		}},
	}}, Options{Width: testViewport, Height: 800})
	if err != nil {
		t.Fatalf("RestyleSVGPaint: %v", err)
	}

	if len(edits) != 1 {
		t.Fatalf("edits = %d, want 1", len(edits))
	}

	data, width, height := svgPaintRaster(t, edits[0].NewImage)

	red, green, blue, alpha := svgPaintPixel(t, data, width, height, 20, 15)
	svgPaintWantColor(t, "inherited fill", red, green, blue, alpha, 255, 0, 0)

	red, green, blue, alpha = svgPaintPixel(t, data, width, height, 60, 15)
	svgPaintWantColor(t, "own fill", red, green, blue, alpha, 0, 0, 255)
}

// TestRestyleSVGPaintPartOverride changes one child by SVG id and leaves its
// sibling alone.
func TestRestyleSVGPaintPartOverride(t *testing.T) {
	t.Parallel()

	res, node := svgPaintLayout(t, `<html><body style="margin:0">`+
		`<svg id="icon" style="display:block" width="80" height="30" viewBox="0 0 80 30" xmlns="http://www.w3.org/2000/svg">`+
		`<rect id="left" x="2" y="2" width="36" height="26" style="fill:#ff0000"/>`+
		`<rect id="right" x="42" y="2" width="36" height="26" style="fill:#0000ff"/>`+
		`</svg></body></html>`)

	edits, err := RestyleSVGPaint(t.Context(), res, []SVGPaintRequest{{
		Node: node,
		Overrides: []SVGPaintOverride{{
			ElementID: "icon",
			PartID:    "right",
			Fill:      "#00ff00",
			HasFill:   true,
		}},
	}}, Options{Width: testViewport, Height: 800})
	if err != nil {
		t.Fatalf("RestyleSVGPaint: %v", err)
	}

	if len(edits) != 1 {
		t.Fatalf("edits = %d, want 1", len(edits))
	}

	data, width, height := svgPaintRaster(t, edits[0].NewImage)

	red, green, blue, alpha := svgPaintPixel(t, data, width, height, 20, 15)
	svgPaintWantColor(t, "left fill", red, green, blue, alpha, 255, 0, 0)

	red, green, blue, alpha = svgPaintPixel(t, data, width, height, 60, 15)
	svgPaintWantColor(t, "right fill", red, green, blue, alpha, 0, 255, 0)
}

// TestRestyleSVGPaintStrokeWidthChangesRaster widens a stroke so pixels just
// outside the old outline become painted.
func TestRestyleSVGPaintStrokeWidthChangesRaster(t *testing.T) {
	t.Parallel()

	res, node := svgPaintLayout(t, `<html><body style="margin:0">`+
		`<svg id="icon" style="display:block;stroke-width:1" width="64" height="28" viewBox="0 0 64 28" `+
		`xmlns="http://www.w3.org/2000/svg">`+
		`<rect x="10" y="10" width="44" height="8" style="fill:none;stroke:#ff0000"/>`+
		`</svg></body></html>`)

	oldData, oldW, oldH := svgPaintRaster(t, svgPaintOp(t, res))

	if _, _, _, alpha := svgPaintPixel(t, oldData, oldW, oldH, 32, 7); alpha != 0 {
		t.Fatalf("old stroke alpha at 3px above the edge = %d, want 0", alpha)
	}

	edits, err := RestyleSVGPaint(t.Context(), res, []SVGPaintRequest{{
		Node: node,
		Overrides: []SVGPaintOverride{{
			ElementID:      "icon",
			StrokeWidth:    8,
			HasStrokeWidth: true,
		}},
	}}, Options{Width: testViewport, Height: 800})
	if err != nil {
		t.Fatalf("RestyleSVGPaint: %v", err)
	}

	if len(edits) != 1 {
		t.Fatalf("edits = %d, want 1", len(edits))
	}

	data, width, height := svgPaintRaster(t, edits[0].NewImage)

	red, green, blue, alpha := svgPaintPixel(t, data, width, height, 32, 7)
	svgPaintWantColor(t, "widened stroke", red, green, blue, alpha, 255, 0, 0)
}

// TestRestyleSVGPaintPresentationAttributes checks the authored attribute
// paths: fill="inherit" follows the root override, while fill="#0000ff" is the
// shape's own paint and stays blue.
func TestRestyleSVGPaintPresentationAttributes(t *testing.T) {
	t.Parallel()

	res, node := svgPaintLayout(t, `<html><body style="margin:0">`+
		`<svg id="icon" style="display:block" width="120" height="30" viewBox="0 0 120 30" `+
		`xmlns="http://www.w3.org/2000/svg">`+
		`<rect x="2" y="2" width="36" height="26"/>`+
		`<rect x="42" y="2" width="36" height="26" fill="inherit"/>`+
		`<rect x="82" y="2" width="36" height="26" fill="#0000ff"/>`+
		`</svg></body></html>`)

	edits, err := RestyleSVGPaint(t.Context(), res, []SVGPaintRequest{{
		Node: node,
		Overrides: []SVGPaintOverride{{
			ElementID: "icon",
			Fill:      "#ff0000",
			HasFill:   true,
		}},
	}}, Options{Width: testViewport, Height: 800})
	if err != nil {
		t.Fatalf("RestyleSVGPaint: %v", err)
	}

	if len(edits) != 1 {
		t.Fatalf("edits = %d, want 1", len(edits))
	}

	data, width, height := svgPaintRaster(t, edits[0].NewImage)

	red, green, blue, alpha := svgPaintPixel(t, data, width, height, 20, 15)
	svgPaintWantColor(t, "no attribute", red, green, blue, alpha, 255, 0, 0)

	red, green, blue, alpha = svgPaintPixel(t, data, width, height, 60, 15)
	svgPaintWantColor(t, "fill inherit", red, green, blue, alpha, 255, 0, 0)

	red, green, blue, alpha = svgPaintPixel(t, data, width, height, 100, 15)
	svgPaintWantColor(t, "own attribute", red, green, blue, alpha, 0, 0, 255)
}

// TestRestyleSVGPaintFillNoneKeepsOwnChildPaint checks that fill:none on the
// root does not zero the fill-opacity a child with its own fill inherits. The
// cascade stores fill:none at opacity zero; the restyle pass remembers the
// no-fill value separately so the child stays opaque.
func TestRestyleSVGPaintFillNoneKeepsOwnChildPaint(t *testing.T) {
	t.Parallel()

	res, node := svgPaintLayout(t, `<html><body style="margin:0">`+
		`<svg id="icon" style="display:block" width="80" height="30" viewBox="0 0 80 30" xmlns="http://www.w3.org/2000/svg">`+
		`<rect x="2" y="2" width="36" height="26"/>`+
		`<rect x="42" y="2" width="36" height="26" style="fill:#0000ff"/>`+
		`</svg></body></html>`)

	edits, err := RestyleSVGPaint(t.Context(), res, []SVGPaintRequest{{
		Node: node,
		Overrides: []SVGPaintOverride{{
			ElementID: "icon",
			Fill:      "none",
			HasFill:   true,
		}},
	}}, Options{Width: testViewport, Height: 800})
	if err != nil {
		t.Fatalf("RestyleSVGPaint: %v", err)
	}

	if len(edits) != 1 {
		t.Fatalf("edits = %d, want 1", len(edits))
	}

	data, width, height := svgPaintRaster(t, edits[0].NewImage)

	_, _, _, alpha := svgPaintPixel(t, data, width, height, 20, 15) //nolint:dogsled // only alpha matters
	if alpha != 0 {
		t.Fatalf("inherited none fill alpha = %d, want 0", alpha)
	}

	red, green, blue, alpha := svgPaintPixel(t, data, width, height, 60, 15)
	svgPaintWantColor(t, "own fill", red, green, blue, alpha, 0, 0, 255)
}

// TestRestyleSVGPaintFillFromNoneBecomesOpaque checks that a root that used to
// be fill:none paints at full opacity after a color override. The cascade's
// zero opacity for the old fill is not a declared fill-opacity.
func TestRestyleSVGPaintFillFromNoneBecomesOpaque(t *testing.T) {
	t.Parallel()

	res, node := svgPaintLayout(t, `<html><body style="margin:0">`+
		`<svg id="icon" style="display:block;fill:none" width="64" height="28" viewBox="0 0 64 28" `+
		`xmlns="http://www.w3.org/2000/svg">`+
		`<rect x="2" y="2" width="60" height="24"/>`+
		`</svg></body></html>`)

	edits, err := RestyleSVGPaint(t.Context(), res, []SVGPaintRequest{{
		Node: node,
		Overrides: []SVGPaintOverride{{
			ElementID: "icon",
			Fill:      "#ff0000",
			HasFill:   true,
		}},
	}}, Options{Width: testViewport, Height: 800})
	if err != nil {
		t.Fatalf("RestyleSVGPaint: %v", err)
	}

	if len(edits) != 1 {
		t.Fatalf("edits = %d, want 1", len(edits))
	}

	data, width, height := svgPaintRaster(t, edits[0].NewImage)

	red, green, blue, alpha := svgPaintPixel(t, data, width, height, 32, 14)
	svgPaintWantColor(t, "fill from none", red, green, blue, alpha, 255, 0, 0)
}

// svgPaintInlineIconSource is a paragraph that holds one inline-level <svg>,
// so the inline collector flattens it into a replaced item instead of building
// a block box for it. The paragraph carries a transform and a z-index, and the
// icon isolates, so the raster op reaches the restyle with a baked transform,
// a stacking level, and a blend group to preserve.
func svgPaintInlineIconSource(inner string) string {
	return `<html><body style="margin:0">` +
		`<p style="margin:0;position:relative;z-index:3;transform:translate(5px,0)">label ` +
		`<svg id="icon" width="64" height="28" viewBox="0 0 64 28" style="isolation:isolate"` +
		` xmlns="http://www.w3.org/2000/svg">` +
		inner +
		`</svg> tail</p></body></html>`
}

// svgPaintWantSameImageShape pins the payload-replacement rule: only the
// encoded raster and its pixel bounds may change. The operation keeps its
// identity, rectangle, stacking level, transform, and blend group so the
// splice path sees the same op it painted before.
//
//nolint:cyclop // one helper checks every preserved field
func svgPaintWantSameImageShape(t *testing.T, oldOp, newOp Op) {
	t.Helper()

	if oldOp.ID == 0 || oldOp.ID != newOp.ID {
		t.Fatalf("image op id = %d, want %d", newOp.ID, oldOp.ID)
	}

	if oldOp.Kind != newOp.Kind {
		t.Fatalf("image op kind = %v, want %v", newOp.Kind, oldOp.Kind)
	}

	if oldOp.X != newOp.X || oldOp.Y != newOp.Y || oldOp.W != newOp.W || oldOp.H != newOp.H {
		t.Fatalf("image rect = (%g, %g, %g, %g), want (%g, %g, %g, %g)",
			newOp.X, newOp.Y, newOp.W, newOp.H, oldOp.X, oldOp.Y, oldOp.W, oldOp.H)
	}

	if oldOp.ZIndex != newOp.ZIndex || oldOp.ZIndexSet != newOp.ZIndexSet {
		t.Fatalf("image z-index = %d set=%t, want %d set=%t",
			newOp.ZIndex, newOp.ZIndexSet, oldOp.ZIndex, oldOp.ZIndexSet)
	}

	if oldOp.XformSet != newOp.XformSet || oldOp.Transform() != newOp.Transform() {
		t.Fatalf("image transform = %+v set=%t, want %+v set=%t",
			newOp.Transform(), newOp.XformSet, oldOp.Transform(), oldOp.XformSet)
	}

	if oldOp.Group() != newOp.Group() {
		t.Fatalf("image group = %p, want %p", newOp.Group(), oldOp.Group())
	}

	if oldOp.BlendModeName() != newOp.BlendModeName() {
		t.Fatalf("image blend mode = %q, want %q", newOp.BlendModeName(), oldOp.BlendModeName())
	}
}

// TestRestyleSVGPaintInlineIconRestyles checks the inline-level case end to
// end: the paragraph keeps the SVG box, the box owns the raster op, and a root
// fill reaches a shape with no paint of its own.
//
//nolint:funlen // one fixture proves discovery, payload replacement, and op identity
func TestRestyleSVGPaintInlineIconRestyles(t *testing.T) {
	t.Parallel()

	res, node := svgPaintLayout(t, svgPaintInlineIconSource(
		`<rect x="2" y="2" width="60" height="24"/>`,
	))

	boxNode := res.boxForID("icon")
	if boxNode.opEnd < boxNode.opStart {
		t.Fatalf("inline svg box range = [%d, %d], want the raster op",
			boxNode.opStart, boxNode.opEnd)
	}

	oldOp := svgPaintOp(t, res)
	oldData, _, _ := svgPaintRaster(t, oldOp)

	if !oldOp.XformSet {
		t.Fatal("inline svg op has no baked transform")
	}

	if oldOp.Group() == nil {
		t.Fatal("inline svg op has no blend group")
	}

	edits, err := RestyleSVGPaint(t.Context(), res, []SVGPaintRequest{{
		Node: node,
		Overrides: []SVGPaintOverride{{
			ElementID: "icon",
			Fill:      "#ff0000",
			HasFill:   true,
		}},
	}}, Options{Width: testViewport, Height: 800})
	if err != nil {
		t.Fatalf("RestyleSVGPaint: %v", err)
	}

	if len(edits) != 1 {
		t.Fatalf("edits = %d, want 1", len(edits))
	}

	svgPaintWantSameImageShape(t, edits[0].OldImage, edits[0].NewImage)

	newData, width, height := svgPaintRaster(t, edits[0].NewImage)
	if bytes.Equal(oldData, newData) {
		t.Fatal("fill override left the inline raster unchanged")
	}

	red, green, blue, alpha := svgPaintPixel(t, newData, width, height, 32, 14)
	svgPaintWantColor(t, "inline fill", red, green, blue, alpha, 255, 0, 0)

	// The retained op and the box payload track the replacement together, so a
	// second restyle finds the same image operation.
	iconBox := res.boxForID("icon")

	index := svgImageOpIndex(res.Ops, iconBox)
	if index < 0 {
		t.Fatal("restyled inline svg lost its image operation")
	}

	if index != iconBox.opEnd {
		t.Fatalf("image operation = %d, want the box's last op %d", index, iconBox.opEnd)
	}

	if data, _, _ := res.Ops[index].ImageBytes(); !bytes.Equal(data, newData) {
		t.Fatal("patched operation kept the old payload")
	}
}

// TestRestyleSVGPaintInlineOwnFillStays checks inheritance for an inline icon:
// a shape that sets its own fill keeps it when the root is overridden.
func TestRestyleSVGPaintInlineOwnFillStays(t *testing.T) {
	t.Parallel()

	res, node := svgPaintLayout(t, `<html><body style="margin:0">`+
		`<p style="margin:0">`+
		`<svg id="icon" width="80" height="30" viewBox="0 0 80 30" xmlns="http://www.w3.org/2000/svg">`+
		`<rect x="2" y="2" width="36" height="26"/>`+
		`<rect x="42" y="2" width="36" height="26" style="fill:#0000ff"/>`+
		`</svg></p></body></html>`)

	edits, err := RestyleSVGPaint(t.Context(), res, []SVGPaintRequest{{
		Node: node,
		Overrides: []SVGPaintOverride{{
			ElementID: "icon",
			Fill:      "#ff0000",
			HasFill:   true,
		}},
	}}, Options{Width: testViewport, Height: 800})
	if err != nil {
		t.Fatalf("RestyleSVGPaint: %v", err)
	}

	if len(edits) != 1 {
		t.Fatalf("edits = %d, want 1", len(edits))
	}

	data, width, height := svgPaintRaster(t, edits[0].NewImage)

	red, green, blue, alpha := svgPaintPixel(t, data, width, height, 20, 15)
	svgPaintWantColor(t, "inherited inline fill", red, green, blue, alpha, 255, 0, 0)

	red, green, blue, alpha = svgPaintPixel(t, data, width, height, 60, 15)
	svgPaintWantColor(t, "own inline fill", red, green, blue, alpha, 0, 0, 255)
}

// TestRestyleSVGPaintInlineRasterCache checks the serialized-string cache: a
// second call with the same paint reuses the stored raster byte for byte and
// does not rasterize again, while a different paint misses and rasterizes.
//
//nolint:cyclop,funlen // one fixture walks a hit, a miss, and the bound
func TestRestyleSVGPaintInlineRasterCache(t *testing.T) {
	t.Parallel()

	res, node := svgPaintLayout(t, svgPaintInlineIconSource(
		`<rect x="2" y="2" width="60" height="24"/>`,
	))

	request := func(fill string) []SVGPaintRequest {
		return []SVGPaintRequest{{
			Node: node,
			Overrides: []SVGPaintOverride{{
				ElementID: "icon",
				Fill:      fill,
				HasFill:   true,
			}},
		}}
	}

	first, err := RestyleSVGPaint(t.Context(), res, request("#ff0000"),
		Options{Width: testViewport, Height: 800})
	if err != nil {
		t.Fatalf("RestyleSVGPaint: %v", err)
	}

	if len(first) != 1 {
		t.Fatalf("first edits = %d, want 1", len(first))
	}

	cache := res.svgRasters
	if cache == nil {
		t.Fatal("restyle left no raster cache on the layout")
	}

	if cache.hits != 0 || cache.misses != 1 {
		t.Fatalf("cache after first call = %d hits, %d misses, want 0 and 1",
			cache.hits, cache.misses)
	}

	second, err := RestyleSVGPaint(t.Context(), res, request("#ff0000"),
		Options{Width: testViewport, Height: 800})
	if err != nil {
		t.Fatalf("RestyleSVGPaint (repeat): %v", err)
	}

	if len(second) != 1 {
		t.Fatalf("second edits = %d, want 1", len(second))
	}

	if cache.hits != 1 || cache.misses != 1 {
		t.Fatalf("cache after repeat = %d hits, %d misses, want 1 and 1",
			cache.hits, cache.misses)
	}

	firstData, _, _ := svgPaintRaster(t, first[0].NewImage)
	secondData, _, _ := svgPaintRaster(t, second[0].NewImage)

	if !bytes.Equal(firstData, secondData) {
		t.Fatal("cached repeat changed the raster payload")
	}

	// A hit returns the stored slice, so the payload is the same bytes, not a
	// second encode that happens to match.
	if &firstData[0] != &secondData[0] {
		t.Fatal("cache hit re-encoded instead of reusing the stored raster")
	}

	third, err := RestyleSVGPaint(t.Context(), res, request("#00ff00"),
		Options{Width: testViewport, Height: 800})
	if err != nil {
		t.Fatalf("RestyleSVGPaint (new paint): %v", err)
	}

	if len(third) != 1 {
		t.Fatalf("third edits = %d, want 1", len(third))
	}

	if cache.hits != 1 || cache.misses != 2 {
		t.Fatalf("cache after new paint = %d hits, %d misses, want 1 and 2",
			cache.hits, cache.misses)
	}

	if len(cache.entries) > svgRasterCacheMaxEntries {
		t.Fatalf("cache entries = %d, want at most %d",
			len(cache.entries), svgRasterCacheMaxEntries)
	}
}

// TestRestyleSVGPaintInlineWithoutOverridesLeavesDisplay pins the no-op
// contract for an inline icon: a request that changes no paint property does
// no raster work and leaves the operations untouched.
//
//nolint:cyclop // the fixture compares every field a no-op must keep
func TestRestyleSVGPaintInlineWithoutOverridesLeavesDisplay(t *testing.T) {
	t.Parallel()

	res, node := svgPaintLayout(t, svgPaintInlineIconSource(
		`<rect x="2" y="2" width="60" height="24" style="fill:#00ff00"/>`,
	))

	before := append([]Op(nil), res.Ops...)
	oldData, _, _ := svgPaintRaster(t, svgPaintOp(t, res))

	edits, err := RestyleSVGPaint(t.Context(), res, []SVGPaintRequest{{
		Node:      node,
		Overrides: nil,
	}}, Options{Width: testViewport, Height: 800})
	if err != nil {
		t.Fatalf("RestyleSVGPaint: %v", err)
	}

	if edits != nil {
		t.Fatalf("edits = %d, want nil", len(edits))
	}

	if len(res.Ops) != len(before) {
		t.Fatalf("operation count = %d, want %d", len(res.Ops), len(before))
	}

	for index := range before {
		if before[index].Kind != res.Ops[index].Kind ||
			before[index].ID != res.Ops[index].ID ||
			before[index].X != res.Ops[index].X ||
			before[index].Y != res.Ops[index].Y ||
			before[index].W != res.Ops[index].W ||
			before[index].H != res.Ops[index].H {
			t.Fatalf("operation %d changed: %+v -> %+v", index, before[index], res.Ops[index])
		}
	}

	if res.svgRasters != nil {
		t.Fatal("a request that changes no paint property touched the raster cache")
	}

	newData, _, _ := svgPaintRaster(t, svgPaintOp(t, res))
	if !bytes.Equal(oldData, newData) {
		t.Fatal("no-override call changed the inline raster")
	}
}

// TestSVGRasterCacheBounds pins the cache limits: the entry count is capped
// oldest-first and a raster above the pixel budget is not stored at all.
func TestSVGRasterCacheBounds(t *testing.T) {
	t.Parallel()

	cache := &svgRasterCache{}

	for index := range svgRasterCacheMaxEntries + 3 {
		cache.put(strconv.Itoa(index), []byte{1}, 1, 1)
	}

	if len(cache.entries) != svgRasterCacheMaxEntries {
		t.Fatalf("cache entries = %d, want %d", len(cache.entries), svgRasterCacheMaxEntries)
	}

	if _, _, _, ok := cache.get("0"); ok {
		t.Fatal("oldest entry survived eviction")
	}

	if _, _, _, ok := cache.get("3"); !ok {
		t.Fatal("newest entries were evicted instead of the oldest")
	}

	before := len(cache.entries)

	cache.put("oversized", []byte{1}, 4096, 4096)

	if len(cache.entries) != before {
		t.Fatalf("oversized raster changed the entry count: %d -> %d",
			before, len(cache.entries))
	}

	if _, _, _, ok := cache.get("oversized"); ok {
		t.Fatal("raster above the pixel budget was cached")
	}

	if cache.pixels > svgRasterCacheMaxPixels {
		t.Fatalf("cache pixels = %d, want at most %d", cache.pixels, svgRasterCacheMaxPixels)
	}
}
