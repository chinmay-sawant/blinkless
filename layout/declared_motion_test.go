package layout_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

// motionPage holds one 40x20 box #m, 10px from the left and 30px from the top.
func motionPage(style string) string {
	return `<!DOCTYPE html><html><body style="margin:0"><div style="height:30px"></div>` +
		`<div id="m" style="width:40px;height:20px;background:#f00;margin-left:10px;` + style + `">x</div></body></html>`
}

// motionCenter maps the middle of #m through its own transform, in points.
func motionCenter(t *testing.T, display *layout.Display, anchorX, anchorY float64) (float64, float64) {
	t.Helper()

	b := boxByID(t, display, "m")
	k := display.PointsPerPixel

	return b.Transform.Own.Apply((b.X+b.W*anchorX)*k, (b.Y+b.H*anchorY)*k)
}

func near(t *testing.T, label string, got, want float64) {
	t.Helper()

	if math.Abs(got-want) > 0.01 {
		t.Errorf("%s = %.3f, want %.3f", label, got, want)
	}
}

// TestMotionPathMovesBox: the anchor (box center) lands on the point halfway
// along a 100px line, and the ops of the box carry that matrix.
func TestMotionPathMovesBox(t *testing.T) {
	t.Parallel()

	display := renderDisplay(t, motionPage("offset-path: path('M 0 0 L 100 0'); offset-distance: 50%"), "")
	b := boxByID(t, display, "m")

	if !b.Transform.HasMatrix {
		t.Fatal("offset-path must give the box a transform")
	}

	k := display.PointsPerPixel
	x, y := motionCenter(t, display, 0.5, 0.5)

	near(t, "x", x, (b.X+50)*k)
	near(t, "y", y, b.Y*k)

	moved := 0

	for _, op := range display.Ops {
		if op.XformSet && op.Transform().E != 0 {
			moved++
		}
	}

	if moved == 0 {
		t.Error("no display op carries the motion matrix")
	}

	plain := boxByID(t, renderDisplay(t, motionPage(""), ""), "m")
	if plain.Transform.HasMatrix {
		t.Error("a box without offset-path must keep no transform")
	}
}

// TestMotionPathRotatesWithHeading: a downward path turns the box 90 degrees
// (auto), and offset-rotate changes that turn.
func TestMotionPathRotatesWithHeading(t *testing.T) {
	t.Parallel()

	cases := []struct {
		rotate string
		wantB  float64 // sine of the turn
		wantA  float64 // cosine of the turn
	}{
		{"", 1, 0},
		{"offset-rotate: auto;", 1, 0},
		{"offset-rotate: reverse;", -1, 0},
		{"offset-rotate: 0deg;", 0, 1},
		{"offset-rotate: auto 90deg;", 0, -1},
	}

	for _, tc := range cases {
		display := renderDisplay(t, motionPage(tc.rotate+"offset-path: path('M 0 0 L 0 100'); offset-distance: 10px"), "")
		own := boxByID(t, display, "m").Transform.Own

		near(t, tc.rotate+" sin", own.B, tc.wantB)
		near(t, tc.rotate+" cos", own.A, tc.wantA)
	}
}

// TestMotionDistanceClampAndWrap: an open path stops at its end, a closed
// polygon wraps around.
func TestMotionDistanceClampAndWrap(t *testing.T) {
	t.Parallel()

	open := renderDisplay(t,
		motionPage("offset-rotate: 0deg; offset-path: path('M 0 0 L 100 0'); offset-distance: 500px"), "")
	b := boxByID(t, open, "m")
	x, _ := motionCenter(t, open, 0.5, 0.5)
	near(t, "open clamp x", x, (b.X+100)*open.PointsPerPixel)

	// A 100x100 square has a 400px perimeter, so 450px is 50px past the start.
	closed := renderDisplay(t, motionPage("offset-rotate: 0deg; "+
		"offset-path: polygon(0px 0px, 100px 0px, 100px 100px, 0px 100px); offset-distance: 450px"), "")
	b = boxByID(t, closed, "m")
	x, y := motionCenter(t, closed, 0.5, 0.5)
	near(t, "wrap x", x, (b.X+50)*closed.PointsPerPixel)
	near(t, "wrap y", y, b.Y*closed.PointsPerPixel)
}

// TestMotionRayAndAnchor: a ray runs up from its start; offset-anchor moves
// which point of the box rides the path.
func TestMotionRayAndAnchor(t *testing.T) {
	t.Parallel()

	ray := renderDisplay(t, motionPage("offset-rotate: 0deg; offset-path: ray(90deg); "+
		"offset-distance: 30px; offset-position: 5px 5px"), "")
	b := boxByID(t, ray, "m")
	x, y := motionCenter(t, ray, 0.5, 0.5)
	near(t, "ray x", x, (b.X+35)*ray.PointsPerPixel)
	near(t, "ray y", y, (b.Y+5)*ray.PointsPerPixel)

	corner := renderDisplay(t, motionPage("offset-rotate: 0deg; offset-anchor: left top; "+
		"offset-path: path('M 20 10 L 100 10'); offset-distance: 0px"), "")
	b = boxByID(t, corner, "m")
	x, y = motionCenter(t, corner, 0, 0)
	near(t, "corner x", x, (b.X+20)*corner.PointsPerPixel)
	near(t, "corner y", y, (b.Y+10)*corner.PointsPerPixel)
}

// TestMotionShorthandMatchesLonghands: the offset shorthand expands onto the
// same placement the longhands give.
func TestMotionShorthandMatchesLonghands(t *testing.T) {
	t.Parallel()

	short := renderDisplay(t, motionPage("offset: path('M 0 0 L 0 100') 40px 90deg / left top"), "")
	long := renderDisplay(t, motionPage(
		"offset-path: path('M 0 0 L 0 100'); offset-distance: 40px; offset-rotate: 90deg; offset-anchor: left top"), "")

	a, b := boxByID(t, short, "m").Transform.Own, boxByID(t, long, "m").Transform.Own
	if !boxByID(t, short, "m").Transform.HasMatrix {
		t.Fatal("offset shorthand with a path must move the box")
	}

	for i, pair := range [][2]float64{{a.A, b.A}, {a.B, b.B}, {a.C, b.C}, {a.D, b.D}, {a.E, b.E}, {a.F, b.F}} {
		near(t, fmt.Sprintf("matrix[%d]", i), pair[0], pair[1])
	}
}

// TestMotionComposesWithTransform: the box's own transform acts first, then
// the motion carries the result along the path.
func TestMotionComposesWithTransform(t *testing.T) {
	t.Parallel()

	display := renderDisplay(t, motionPage(
		"transform: translate(20px, 0); offset-rotate: 0deg; offset-path: path('M 0 0 L 100 0'); offset-distance: 50%"), "")
	b := boxByID(t, display, "m")
	x, y := motionCenter(t, display, 0.5, 0.5)

	near(t, "x", x, (b.X+50+20)*display.PointsPerPixel)
	near(t, "y", y, b.Y*display.PointsPerPixel)
}

// TestMotionDeclaredValues: grammar accepted and rejected, no inheritance,
// wide keywords.
func TestMotionDeclaredValues(t *testing.T) {
	t.Parallel()

	good := map[string]string{
		"offset-path":     "path('M 0 0 L 10 10') border-box",
		"offset-distance": "25%",
		"offset-rotate":   "reverse 45deg",
		"offset-anchor":   "right 10px bottom 20%",
		"offset-position": "auto",
		"offset":          "10px 30px ray(45deg) 5px auto / center",
	}

	for prop, value := range good {
		display := renderDisplay(t, motionPage(prop+": "+value), "")
		if boxByID(t, display, "m").Declared[prop] == "" {
			t.Errorf("%s: %s should be accepted", prop, value)
		}
	}

	bad := map[string]string{
		"offset-path":     "path(M 0 0)",
		"offset-distance": "fast",
		"offset-rotate":   "auto reverse",
		"offset-anchor":   "left right",
		"offset-position": "10px 10px 10px 10px 10px",
		"offset":          "10px 20px 30px",
	}

	for prop, value := range bad {
		display := renderDisplay(t, motionPage(prop+": "+value), "")
		if got := boxByID(t, display, "m").Declared[prop]; got != "" {
			t.Errorf("%s: %s should be dropped, got %q", prop, value, got)
		}
	}

	display := renderDisplay(t, `<!DOCTYPE html><html><body>`+
		`<div id="p" style="offset-distance: 10px"><div id="c" style="offset-rotate: auto"><div id="d">x</div></div>`+
		`<p id="e" style="offset-distance: inherit">y</p></div></body></html>`, "")

	if _, ok := boxByID(t, display, "d").Declared["offset-distance"]; ok {
		t.Error("offset-distance must not inherit")
	}

	if got := boxByID(t, display, "e").Declared["offset-distance"]; got != "10px" {
		t.Errorf("inherit gave %q", got)
	}

	display = renderDisplay(t, motionPage("offset-distance: 5px; offset-distance: initial"), "")
	if _, ok := boxByID(t, display, "m").Declared["offset-distance"]; ok {
		t.Error("initial must clear offset-distance")
	}
}
