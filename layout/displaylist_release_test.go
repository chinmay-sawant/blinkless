package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

func TestDisplayReleaseNilAndEmpty(t *testing.T) {
	t.Parallel()

	var nilDisplay *layout.Display
	nilDisplay.Release()

	emptyDisplay := &layout.Display{}
	emptyDisplay.Release()
	emptyDisplay.Release() // idempotent
}

func TestDisplayReleaseRetainedLifecycle(t *testing.T) {
	t.Parallel()

	tree, err := html.Parse([]byte(`<div id="box" style="width:100px;height:50px">Test</div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled, err := css.Apply(t.Context(), tree, css.Options{WidthPx: 400, HeightPx: 300, Media: "screen"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	display, err := layout.DisplayListOptions(t.Context(), styled, layout.Options{RetainState: true})
	if err != nil {
		t.Fatalf("display: %v", err)
	}

	// Release once
	display.Release()

	// Release again should be idempotent and not panic
	display.Release()
}

func TestDisplayReleasePatchedLifecycle(t *testing.T) {
	t.Parallel()

	tree, err := html.Parse([]byte(`<div id="box" style="width:100px;height:50px">Original</div>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled, err := css.Apply(t.Context(), tree, css.Options{WidthPx: 400, HeightPx: 300, Media: "screen"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	d1, err := layout.DisplayListOptions(t.Context(), styled, layout.Options{RetainState: true})
	if err != nil {
		t.Fatalf("display: %v", err)
	}

	if !tree.SetText("box", "Updated") {
		t.Fatal("SetText failed")
	}

	patch, err := layout.RelayoutElement(t.Context(), styled, d1, "box", layout.Options{})
	if err != nil {
		t.Fatalf("relayout: %v", err)
	}
	d2 := patch.Display

	// Release d1 first; d2 should remain valid and not panic or corrupt.
	d1.Release()

	// Release d2 should release the underlying Result back to pool.
	d2.Release()
	d2.Release()
}
