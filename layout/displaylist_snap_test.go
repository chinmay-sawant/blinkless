package layout_test

import (
	"errors"
	"math"
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

// strokeWidthsPx returns the positive stroke widths of the stroke operations
// in display, in CSS pixels.
func strokeWidthsPx(display *layout.Display) []float64 {
	var widths []float64

	for _, paintOp := range display.Ops {
		if paintOp.Kind != layout.DisplayOpStrokeRect && paintOp.Kind != layout.DisplayOpLine {
			continue
		}

		if paintOp.Width > 0 {
			widths = append(widths, paintOp.Width/display.PointsPerPixel)
		}
	}

	return widths
}

// borderDisplay lays out a 100x10 box with a solid border of widthPx. A
// uniform solid border without a radius is four OpLine strokes.
func borderDisplay(t *testing.T, widthPx string) *layout.Display {
	t.Helper()

	return displayOf(t, `<div style="width:100px;height:10px;border:`+widthPx+
		` solid #000"></div>`)
}

// TestSnapDisplayToDevicePixelsBorderWidths pins the snap formula on real
// border strokes. A visible stroke keeps at least one device pixel, then is
// floored onto the device grid: max(1, floor(cssPx*dsf+1e-6))/dsf CSS pixels.
// Chrome 143.0.7499.40 agrees at dsf 1 (a 1px border is 1px, a 2px border is
// 2px, a 1pt border is 1px); the fractional rows are the same rule at other
// scale factors. The source display is not mutated.
func TestSnapDisplayToDevicePixelsBorderWidths(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		borderPx string
		dsf      float64
		wantPx   float64
	}{
		{borderPx: "1px", dsf: 1, wantPx: 1},
		{borderPx: "1px", dsf: 1.25, wantPx: 0.8},
		{borderPx: "1px", dsf: 1.5, wantPx: 2.0 / 3.0},
		{borderPx: "1px", dsf: 2, wantPx: 1},
		{borderPx: "2px", dsf: 1, wantPx: 2},
		{borderPx: "2px", dsf: 1.25, wantPx: 1.6},
		{borderPx: "2px", dsf: 2, wantPx: 2},
		{borderPx: "3px", dsf: 1.25, wantPx: 2.4},
	}

	for _, testCase := range testCases {
		display := borderDisplay(t, testCase.borderPx)

		before := strokeWidthsPx(display)
		if len(before) == 0 {
			t.Fatalf("%s: no stroke ops", testCase.borderPx)
		}

		snapped, err := layout.SnapDisplayToDevicePixels(display, testCase.dsf)
		if err != nil {
			t.Fatalf("%s at dsf %v: snap: %v", testCase.borderPx, testCase.dsf, err)
		}

		after := strokeWidthsPx(snapped)
		if len(after) != len(before) {
			t.Fatalf("%s at dsf %v: %d strokes after snap, want %d",
				testCase.borderPx, testCase.dsf, len(after), len(before))
		}

		for _, width := range after {
			if math.Abs(width-testCase.wantPx) > 1e-9 {
				t.Errorf("%s at dsf %v: width %.6fpx, want %.6fpx",
					testCase.borderPx, testCase.dsf, width, testCase.wantPx)
			}
		}

		unchanged := strokeWidthsPx(display)
		for idx, width := range unchanged {
			if width != before[idx] {
				t.Errorf("%s at dsf %v: original stroke %d changed from %.6fpx to %.6fpx",
					testCase.borderPx, testCase.dsf, idx, before[idx], width)
			}
		}
	}
}

// TestSnapDisplayToDevicePixelsTextDecorationWidth: text decorations are
// OpLine strokes too, so an underline is snapped by the same rule. The
// expected value is derived from the original underline width rather than a
// hardcoded face metric.
func TestSnapDisplayToDevicePixelsTextDecorationWidth(t *testing.T) {
	t.Parallel()

	display := displayOf(t, `<p style="font-size:16px;text-decoration:underline">underlined</p>`)

	widths := strokeWidthsPx(display)
	if len(widths) == 0 {
		t.Fatal("no underline stroke op")
	}

	const dsf = 1.5

	snapped, err := layout.SnapDisplayToDevicePixels(display, dsf)
	if err != nil {
		t.Fatalf("snap: %v", err)
	}

	got := strokeWidthsPx(snapped)
	if len(got) != len(widths) {
		t.Fatalf("%d strokes after snap, want %d", len(got), len(widths))
	}

	for idx, width := range got {
		want := wantSnappedPx(widths[idx], dsf)
		if math.Abs(width-want) > 1e-9 {
			t.Errorf("underline %d: %.6fpx, want %.6fpx", idx, width, want)
		}
	}
}

// wantSnappedPx is the snap formula in CSS pixels.
func wantSnappedPx(widthPx, dsf float64) float64 {
	devicePx := math.Floor(widthPx*dsf + 1e-6)
	if devicePx < 1 {
		devicePx = 1
	}

	return devicePx / dsf
}

// TestSnapDisplayToDevicePixelsCopiesAndPreserves: the returned display is a
// copy. Every op except a positive-width stroke is identical, stroke ops
// differ only in Width, and Boxes, Order, canvas size, and unit factors are
// unchanged. Payloads (text, the face pointer, and the link URI) ride along
// untouched.
func TestSnapDisplayToDevicePixelsCopiesAndPreserves(t *testing.T) {
	t.Parallel()

	display := displayOf(t, `
<div style="background:#eee;width:40px;height:10px">plain</div>
<div style="border:1px solid #000;width:40px;height:10px">bordered</div>
<a href="/next" style="text-decoration:underline">link</a>
`)

	opsBefore := append([]layout.DisplayOp(nil), display.Ops...)
	orderBefore := append([]int(nil), display.Order...)
	boxesBefore := append([]layout.Box(nil), display.Boxes...)

	snapped, err := layout.SnapDisplayToDevicePixels(display, 2)
	if err != nil {
		t.Fatalf("snap: %v", err)
	}

	if snapped == display {
		t.Fatal("snap returned the same display pointer")
	}

	assertOpsPreserved(t, display, snapped)
	assertStructurePreserved(t, display, snapped, orderBefore, boxesBefore)
	assertSnapSlicesAreCopies(t, display, snapped)

	for idx := range opsBefore {
		if display.Ops[idx] != opsBefore[idx] {
			t.Errorf("source op %d mutated", idx)
		}
	}
}

// assertOpsPreserved checks that every op is unchanged except a positive-width
// stroke, whose Width is the only field allowed to move.
func assertOpsPreserved(t *testing.T, display, snapped *layout.Display) {
	t.Helper()

	if len(snapped.Ops) != len(display.Ops) {
		t.Fatalf("ops %d, want %d", len(snapped.Ops), len(display.Ops))
	}

	strokes := 0

	for idx := range display.Ops {
		orig, got := display.Ops[idx], snapped.Ops[idx]

		isStroke := (got.Kind == layout.DisplayOpStrokeRect || got.Kind == layout.DisplayOpLine) && orig.Width > 0
		if !isStroke {
			if got != orig {
				t.Errorf("op %d (kind %v) changed: %+v -> %+v", idx, got.Kind, orig, got)
			}

			continue
		}

		strokes++

		want := orig
		want.Width = got.Width

		if got != want {
			t.Errorf("op %d stroke changed beyond Width: %+v -> %+v", idx, want, got)
		}
	}

	if strokes == 0 {
		t.Fatal("no positive-width strokes in fixture")
	}
}

// assertStructurePreserved checks Order, Boxes, canvas size, and unit factors.
func assertStructurePreserved(
	t *testing.T, display, snapped *layout.Display, orderBefore []int, boxesBefore []layout.Box,
) {
	t.Helper()

	if len(snapped.Order) != len(orderBefore) {
		t.Fatalf("order %d, want %d", len(snapped.Order), len(orderBefore))
	}

	for idx, index := range orderBefore {
		if snapped.Order[idx] != index {
			t.Errorf("order %d = %d, want %d", idx, snapped.Order[idx], index)
		}
	}

	if len(snapped.Boxes) != len(boxesBefore) {
		t.Fatalf("boxes %d, want %d", len(snapped.Boxes), len(boxesBefore))
	}

	for idx, box := range boxesBefore {
		if snapped.Boxes[idx] != box {
			t.Errorf("box %d = %+v, want %+v", idx, snapped.Boxes[idx], box)
		}
	}

	assertCanvasAndFactorsPreserved(t, display, snapped)
}

// assertCanvasAndFactorsPreserved checks the scalar display fields.
func assertCanvasAndFactorsPreserved(t *testing.T, display, snapped *layout.Display) {
	t.Helper()

	if snapped.Width != display.Width {
		t.Errorf("canvas width %d, want %d", snapped.Width, display.Width)
	}

	if snapped.Height != display.Height {
		t.Errorf("canvas height %d, want %d", snapped.Height, display.Height)
	}

	if snapped.PointsPerPixel != display.PointsPerPixel {
		t.Errorf("PointsPerPixel %v, want %v", snapped.PointsPerPixel, display.PointsPerPixel)
	}

	if snapped.PixelPerPoint != display.PixelPerPoint {
		t.Errorf("PixelPerPoint %v, want %v", snapped.PixelPerPoint, display.PixelPerPoint)
	}
}

// assertSnapSlicesAreCopies edits the snapped slices and checks the source
// display did not move. A caller that mutates the snapped list must not
// change the display it came from.
func assertSnapSlicesAreCopies(t *testing.T, display, snapped *layout.Display) {
	t.Helper()

	if len(snapped.Ops) > 0 {
		originalX := display.Ops[0].X
		snapped.Ops[0].X += 100

		if display.Ops[0].X != originalX {
			t.Error("snapped.Ops aliases the source slice")
		}
	}

	if len(snapped.Order) > 0 {
		originalOrder := display.Order[0]
		snapped.Order[0] = -1

		if display.Order[0] != originalOrder {
			t.Error("snapped.Order aliases the source slice")
		}
	}

	if len(snapped.Boxes) > 0 {
		originalBoxX := display.Boxes[0].X
		snapped.Boxes[0].X += 100

		if display.Boxes[0].X != originalBoxX {
			t.Error("snapped.Boxes aliases the source slice")
		}
	}
}

// TestSnapDisplayToDevicePixelsLeavesGridRunAlone: collapsed table borders
// travel as OpGridRun segments, and the design keeps those segments exact.
// The run's owner op, including its Width field, is untouched.
func TestSnapDisplayToDevicePixelsLeavesGridRunAlone(t *testing.T) {
	t.Parallel()

	display := displayOf(t, `<table style="border-collapse:collapse">`+
		`<tr><td style="border:1px solid #999">A</td><td style="border:1px solid #999">B</td></tr>`+
		`<tr><td style="border:1px solid #999">C</td><td style="border:1px solid #999">D</td></tr>`+
		`</table>`)

	gridIndex := -1

	for idx, paintOp := range display.Ops {
		if paintOp.Kind == layout.DisplayOpGridRun {
			gridIndex = idx

			break
		}
	}

	if gridIndex < 0 {
		t.Fatal("no OpGridRun in a collapsed table display")
	}

	snapped, err := layout.SnapDisplayToDevicePixels(display, 1.5)
	if err != nil {
		t.Fatalf("snap: %v", err)
	}

	orig, got := display.Ops[gridIndex], snapped.Ops[gridIndex]

	if got != orig {
		t.Errorf("OpGridRun changed: %+v -> %+v", orig, got)
	}

	for segIdx := range orig.Grid.Segs {
		if got.Grid.Segs[segIdx] != orig.Grid.Segs[segIdx] {
			t.Errorf("grid segment %d changed: %+v -> %+v",
				segIdx, orig.Grid.Segs[segIdx], got.Grid.Segs[segIdx])
		}
	}
}

// TestSnapDisplayToDevicePixelsRejectsBadInput covers the error contract:
// a nil display is ErrNilDocument, and a non-finite or non-positive device
// scale factor is ErrBadDeviceScale.
func TestSnapDisplayToDevicePixelsRejectsBadInput(t *testing.T) {
	t.Parallel()

	if _, err := layout.SnapDisplayToDevicePixels(nil, 1); !errors.Is(err, layout.ErrNilDocument) {
		t.Errorf("nil display: %v, want ErrNilDocument", err)
	}

	display := displayOf(t, `<p>scale</p>`)

	for _, dsf := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := layout.SnapDisplayToDevicePixels(display, dsf); !errors.Is(err, layout.ErrBadDeviceScale) {
			t.Errorf("dsf %v: %v, want ErrBadDeviceScale", dsf, err)
		}
	}

	if _, err := layout.SnapDisplayToDevicePixels(display, 1); err != nil {
		t.Errorf("dsf 1: %v, want nil", err)
	}
}
