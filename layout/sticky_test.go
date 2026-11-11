package layout_test

import (
	"math"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

func stickyDisplay(t *testing.T, source string) *layout.Display {
	t.Helper()
	doc, err := html.Parse([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	styled, err := css.Apply(t.Context(), doc, css.Options{WidthPx: 200, HeightPx: 100, Media: "screen"})
	if err != nil {
		t.Fatal(err)
	}
	display, err := layout.DisplayList(t.Context(), styled)
	if err != nil {
		t.Fatal(err)
	}
	return display
}

func TestStickyViewportActivationReleaseAndReturn(t *testing.T) {
	t.Parallel()
	source := `<style>body{margin:0}#before{height:80px}#region{height:300px}` +
		`#head{position:sticky;top:0;height:40px;background:red;z-index:2}#after{height:1000px}</style>` +
		`<div id="before"></div><div id="region"><div id="head"><button id="hit">Go</button></div></div>` +
		`<div id="after"></div>`
	display := stickyDisplay(t, source)
	head := findBox(t, display.Boxes, "head")
	if head.Y != 80 || head.Sticky == nil || !head.Sticky.Viewport || head.Sticky.Bounds != [4]float64{0, 80, 200, 300} {
		t.Fatalf("sticky constraint %+v head %+v", head.Sticky, head)
	}
	if !layout.HasViewportSticky(display) {
		t.Fatal("missing viewport contract")
	}
	original := findBox(t, display.Boxes, "hit")
	cases := []struct{ scroll, want float64 }{
		{0, 80}, {79, 80}, {80, 80}, {100, 100}, {340, 340}, {360, 340}, {500, 340}, {0, 80},
	}
	for _, test := range cases {
		view := layout.StickyView(display, 0, test.scroll, 200, 100)
		current := findBox(t, view.Boxes, "head")
		if current.Y != test.want {
			t.Fatalf("scroll %g: Y %g want %g", test.scroll, current.Y, test.want)
		}
		child := findBox(t, view.Boxes, "hit")
		if child.Y-original.Y != test.want-80 {
			t.Fatal("hit geometry did not move with sticky paint")
		}
		moved := 0
		for index, operation := range view.Ops {
			if operation.StickyID == 0 || operation.Fixed {
				continue
			}
			base := display.Ops[index]
			if test.want != 80 {
				if !operation.XformSet || math.Abs(operation.XformValue().F-(test.want-80)*display.PointsPerPixel) > 0.001 {
					t.Fatalf("sticky paint offset %+v", operation.XformValue())
				}
				if base.XformSet {
					t.Fatal("scroll mutated original operation")
				}
				moved++
			}
		}
		if test.want != 80 && moved == 0 {
			t.Fatal("no sticky operations moved")
		}
	}
	if findBox(t, display.Boxes, "head").Y != 80 {
		t.Fatal("scroll mutated source boxes")
	}
}

func TestStickyViewportBoundariesAndUnsupportedScrollSpaces(t *testing.T) {
	t.Parallel()
	styles := []string{
		"overflow:hidden", "overflow:auto", "overflow:scroll", "transform:translateX(1px)",
		"position:fixed", "position:sticky;top:0",
	}
	for _, style := range styles {
		t.Run(style, func(t *testing.T) {
			t.Parallel()

			source := `<style>body{margin:0}</style><div style="height:300px;` + style + `">` +
				`<div id="head" style="position:sticky;top:0;height:40px;background:red"></div></div>`
			display := stickyDisplay(t, source)
			head := findBox(t, display.Boxes, "head")
			if head.Sticky == nil || head.Sticky.Viewport {
				t.Fatalf("nested scroll-space constraint %+v", head.Sticky)
			}
			view := layout.StickyView(display, 0, 100, 200, 100)
			if findBox(t, view.Boxes, "head").Y != head.Y {
				t.Fatal("unsupported nested scroll space followed viewport")
			}
		})
	}
	source := `<style>body{margin:0}</style>` +
		`<div style="height:300px;border:5px solid blue;padding:10px;overflow:clip">` +
		`<div id="head" style="position:sticky;top:5px;height:40px;background:red"></div></div>`
	display := stickyDisplay(t, source)
	head := findBox(t, display.Boxes, "head")
	if !head.Sticky.Viewport {
		t.Fatal("overflow:clip incorrectly created a scroll container")
	}
	view := layout.StickyView(display, 0, 100, 200, 100)
	if got := findBox(t, view.Boxes, "head").Y; got != 105 {
		t.Fatalf("inset Y %g", got)
	}
	if head.Sticky.Bounds[1] != 15 {
		t.Fatalf("containing block ignored border/padding: %+v", head.Sticky.Bounds)
	}
}

func TestStickyViewHorizontalBottomAndFixedDescendant(t *testing.T) {
	t.Parallel()
	display := &layout.Display{Width: 600, Height: 600, PointsPerPixel: 0.75, PixelPerPoint: 4.0 / 3.0,
		Boxes: []layout.Box{
			{ID: "region", ParentIndex: -1, W: 600, H: 600},
			{
				ID: "head", ParentIndex: 0, X: 80, Y: 180, W: 40, H: 40, OperationIDs: []uint64{1, 2},
				Sticky: &layout.StickyConstraint{
					Insets: [4]float64{}, Viewport: true,
					Set: [4]bool{false, false, true, true}, Bounds: [4]float64{0, 0, 600, 600},
				},
			},
			{ID: "fixed", ParentIndex: 1, X: 2, Y: 3, W: 10, H: 10, Fixed: true, OperationIDs: []uint64{2}},
		},
		Ops: []layout.DisplayOp{
			{ID: 1, Kind: layout.DisplayOpFillRect, X: 60, Y: 135, W: 30, H: 30},
			{ID: 2, Kind: layout.DisplayOpFillRect, X: 1.5, Y: 2.25, W: 7.5, H: 7.5, Fixed: true},
		},
	}
	view := layout.StickyView(display, 100, 0, 200, 100)
	head := findBox(t, view.Boxes, "head")
	if head.X != 100 || head.Y != 60 {
		t.Fatalf("left/bottom sticky %+v", head)
	}
	if findBox(t, view.Boxes, "fixed").Y != 3 || view.Ops[1].XformSet {
		t.Fatal("fixed descendant translated with sticky parent")
	}
	if layout.StickyView(display, math.NaN(), 0, 200, 100) != display {
		t.Fatal("invalid viewport changed geometry")
	}
}
