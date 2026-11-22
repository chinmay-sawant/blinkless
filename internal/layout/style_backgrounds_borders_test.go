//nolint:usetesting // targeted unit tests for Phase 80
package layout

import (
	"context"
	"testing"
)

func TestBorderSideStyles(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       context.Background(),
		err:       nil,
		work:      0,
		sheets:    nil,
		viewportW: 800,
	}

	s := initialStyle()
	raw := map[string]string{
		"border-top-style":    "dashed",
		"border-right-style":  "dotted",
		"border-bottom-style": "solid",
		"border-left-style":   "none",
	}
	applyRestProps(&s, raw, ctx, nil)

	if s.borders().BorderTop.Style != "dashed" || s.borders().BorderRight.Style != "dotted" ||
		s.borders().BorderBottom.Style != "solid" || s.borders().BorderLeft.Style != "none" {
		t.Errorf("mismatch on border side styles: top=%v, right=%v, bot=%v, left=%v",
			s.borders().BorderTop.Style, s.borders().BorderRight.Style, s.borders().BorderBottom.Style, s.borders().BorderLeft.Style)
	}
}

func TestBackgroundLonghands(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       context.Background(),
		err:       nil,
		work:      0,
		sheets:    nil,
		viewportW: 800,
	}

	s := initialStyle()
	raw := map[string]string{
		"background-image":      "url(header.png)",
		"background-position-x": "center",
		"background-position-y": "top",
		"background-size":       "cover",
		"background-repeat":     "no-repeat",
		"background-clip":       "padding-box",
		"background-origin":     "content-box",
		"background-attachment": "fixed",
	}
	applyRestProps(&s, raw, ctx, nil)

	if s.background().BackgroundImage != "header.png" || s.background().BackgroundPosX != "center" || s.background().BackgroundPosY != "top" {
		t.Errorf("mismatch on bg image/pos: img=%q, posX=%q, posY=%q", s.background().BackgroundImage, s.background().BackgroundPosX, s.background().BackgroundPosY)
	}
	if s.background().BackgroundSize != "cover" || s.background().BackgroundRepeat != "no-repeat" {
		t.Errorf("mismatch on bg size/repeat: size=%q, repeat=%q", s.background().BackgroundSize, s.background().BackgroundRepeat)
	}
	if s.background().BackgroundClip != "padding-box" || s.background().BackgroundOrigin != "content-box" || s.background().BackgroundAttachment != "fixed" {
		t.Errorf("mismatch on bg clip/origin/attachment: clip=%q, origin=%q, attachment=%q",
			s.background().BackgroundClip, s.background().BackgroundOrigin, s.background().BackgroundAttachment)
	}
}

func TestBorderImageProps(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       context.Background(),
		err:       nil,
		work:      0,
		sheets:    nil,
		viewportW: 800,
	}

	s := initialStyle()
	raw := map[string]string{
		"border-image-source": "url(border.png)",
		"border-image-slice":  "30",
		"border-image-width":  "10pt",
		"border-image-outset": "2pt",
		"border-image-repeat": "round",
	}
	applyRestProps(&s, raw, ctx, nil)

	if s.borders().BorderImageSource != "url(border.png)" || s.borders().BorderImageSlice != "30" ||
		s.borders().BorderImageWidth != "10pt" || s.borders().BorderImageOutset != "2pt" || s.borders().BorderImageRepeat != "round" {
		t.Errorf("mismatch on border image props: %+v", s)
	}
}

func TestBorderColorFourValues(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       context.Background(),
		err:       nil,
		work:      0,
		sheets:    nil,
		viewportW: 800,
	}

	s := initialStyle()
	raw := map[string]string{
		"border-color": "#ef4444 #3b82f6 #10b981 #f59e0b",
	}
	applyRestProps(&s, raw, ctx, nil)

	if s.borders().BorderTop.Color[0] < 0.9 || s.borders().BorderRight.Color[2] < 0.9 ||
		s.borders().BorderBottom.Color[1] < 0.7 || s.borders().BorderLeft.Color[0] < 0.9 {
		t.Errorf("mismatch on four-value border-color: top=%v, right=%v, bot=%v, left=%v",
			s.borders().BorderTop.Color, s.borders().BorderRight.Color, s.borders().BorderBottom.Color, s.borders().BorderLeft.Color)
	}
}

// TestTransparentBorderPaintsNothing pins the transparent-border contract:
// layout keeps the declared width, paint emits nothing. Chromium paints
// nothing for `border: 8pt solid transparent` (see
// test/chrome/cases/case-35-wpt-flex-cross-size-border-box.html).
func TestTransparentBorderPaintsNothing(t *testing.T) {
	t.Parallel()

	const borderWidthPt = 8

	ctx := &styleContext{
		ctx:       context.Background(),
		err:       nil,
		work:      0,
		sheets:    nil,
		viewportW: 800,
	}

	resolve := func(props map[string]string) ResolvedStyle {
		s := initialStyle()
		applyRestProps(&s, props, ctx, nil)

		return s
	}

	eng := &engine{scale: 1}

	// All four sides transparent: the block border box and the replaced
	// element emitter stay silent while layout keeps the 8pt width.
	allTransparent := resolve(map[string]string{"border": "8pt solid transparent"})
	if ops := eng.borderOps(&allTransparent, 0, 0, 300, 150); len(ops) != 0 {
		t.Fatalf("transparent border ops = %d, want 0", len(ops))
	}
	eng.emitBorders(allTransparent, 0, 0, 300, 150)
	if len(eng.ops) != 0 {
		t.Fatalf("transparent replaced-element border ops = %d, want 0", len(eng.ops))
	}
	if !near(allTransparent.borders().BorderTop.Width, borderWidthPt) {
		t.Fatalf("transparent border width = %.2fpt, want %dpt", allTransparent.borders().BorderTop.Width, borderWidthPt)
	}

	// Longhand color declarations take the same alpha path.
	longhands := map[string]map[string]string{
		"border-color": {
			"border-width": "8pt", "border-style": "solid", "border-color": "transparent",
		},
		"border-top-color": {
			"border-width": "8pt", "border-style": "solid", "border-top-color": "transparent",
		},
	}
	for name, props := range longhands {
		sty := resolve(props)
		if ops := eng.borderOpsSides(&sty, 0, 0, 300, 150, true, false, false, false); len(ops) != 0 {
			t.Fatalf("%s: transparent top border ops = %d, want 0", name, len(ops))
		}
	}

	// The opaque control still paints so the gates cannot pass vacuously.
	opaque := resolve(map[string]string{"border": "8pt solid black"})
	if ops := eng.borderOps(&opaque, 0, 0, 300, 150); len(ops) == 0 {
		t.Fatal("opaque border ops = 0, want > 0")
	}
}
