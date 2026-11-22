//nolint:all // targeted unit tests for Phase 80
package layout

import (
	"context"
	"testing"
)

func TestLogicalCornerRadii(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       context.Background(),
		err:       nil,
		work:      0,
		sheets:    nil,
		viewportW: 800,
	}

	// Horizontal LTR: start-start is TopLeft, start-end is TopRight, end-start is BottomLeft, end-end is BottomRight
	s1 := initialStyle()
	raw1 := map[string]string{
		"border-start-start-radius": "10pt",
		"border-start-end-radius":   "20pt",
		"border-end-start-radius":   "30pt",
		"border-end-end-radius":     "40pt",
	}
	applyRestProps(&s1, raw1, ctx, nil)

	if s1.borders().BorderRadiusTopLeft != 10 || s1.borders().BorderRadiusTopRight != 20 ||
		s1.borders().BorderRadiusBottomLeft != 30 || s1.borders().BorderRadiusBottomRight != 40 {
		t.Errorf("mismatch on logical corner radii: TL=%v, TR=%v, BL=%v, BR=%v",
			s1.borders().BorderRadiusTopLeft, s1.borders().BorderRadiusTopRight, s1.borders().BorderRadiusBottomLeft, s1.borders().BorderRadiusBottomRight)
	}

	// Side radii: block-start is top (TL, TR = 15), inline-end is right (TR, BR = 25 -> TR overwritten with 25)
	s2 := initialStyle()
	raw2a := map[string]string{
		"border-block-start-radius": "15pt",
	}
	applyRestProps(&s2, raw2a, ctx, nil)
	raw2b := map[string]string{
		"border-inline-end-radius": "25pt",
	}
	applyRestProps(&s2, raw2b, ctx, nil)

	// block-start is top (TL, TR = 15), inline-end is right (TR, BR = 25 -> TR overwritten with 25)
	if s2.borders().BorderRadiusTopLeft != 15 || s2.borders().BorderRadiusTopRight != 25 || s2.borders().BorderRadiusBottomRight != 25 {
		t.Errorf("mismatch on side radii: TL=%v, TR=%v, BR=%v",
			s2.borders().BorderRadiusTopLeft, s2.borders().BorderRadiusTopRight, s2.borders().BorderRadiusBottomRight)
	}
}

func TestMarginBreakProperty(t *testing.T) {
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
		"margin-break": "keep",
	}
	applyRestProps(&s, raw, ctx, nil)

	if s.fragmentation().MarginBreak != "keep" {
		t.Errorf("expected MarginBreak 'keep', got %q", s.fragmentation().MarginBreak)
	}
}
