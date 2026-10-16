//nolint:wsl,cyclop,funlen,varnamelen,nlreturn,lll,gocognit,dupword,gocyclo // behavior fixtures
package layout

import (
	"fmt"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// TestBehaviorPositionAreaPlacesAboveAnchor proves core placement: an
// absolute target with position-area:top sits above its in-scope anchor
// with X aligned to the anchor start edge. The anchor does not move and
// the target paint ops move with the box.
func TestBehaviorPositionAreaPlacesAboveAnchor(t *testing.T) {
	t.Parallel()

	src := `<html><body style="margin:0">` +
		`<div id="anchor" style="anchor-name:--a;margin:100px 0 0 50px;width:100px;height:50px;background:#ff0000">anchor</div>` +
		`<div id="target" style="position:absolute;left:0;top:0;position-anchor:--a;position-area:top;width:80px;height:30px;background:#00ff00">target</div>` +
		`</body></html>`
	res := layoutHTML(t, src)

	anchor := boxByID(t, res, "anchor")
	target := boxByID(t, res, "target")

	if !(target.y < anchor.y) {
		t.Fatalf("target y=%v want < anchor y=%v (above anchor)", target.y, anchor.y)
	}

	if !near(target.x, anchor.x) {
		t.Fatalf("target x=%v want anchor x=%v (start-edge aligned)", target.x, anchor.x)
	}

	// Anchor keeps its flow position: 50px left, 100px top at 0.75pt per px.
	if !near(anchor.x, 50*0.75) {
		t.Fatalf("anchor x=%v want %v", anchor.x, 50*0.75)
	}

	if !near(anchor.y, 100*0.75) {
		t.Fatalf("anchor y=%v want %v", anchor.y, 100*0.75)
	}

	// Target sits directly above: anchor top minus target height.
	wantY := anchor.y - target.height
	if !near(target.y, wantY) {
		t.Fatalf("target y=%v want anchor.y-height=%v", target.y, wantY)
	}

	// Paint ops move with the box: the target background fill sits at the
	// target border box, not at the pre-anchor left/top offsets.
	found := false

	for _, op := range res.Ops[target.opStart : target.opEnd+1] {
		if op.Kind != OpFillRect {
			continue
		}

		if near(op.X, target.x) && near(op.Y, target.y) {
			found = true

			break
		}
	}

	if !found {
		t.Fatalf("no target fill at box x=%v y=%v in op range [%d,%d]", target.x, target.y, target.opStart, target.opEnd)
	}
}

// TestBehaviorPositionAreaNone is position-area:none on an absolute box
// leaves the left/top offsets alone. The box matches a document that omits
// position-area. A sibling document with position-area:top moves above the
// anchor, proving the harness detects placement.
func TestBehaviorPositionAreaNone(t *testing.T) {
	t.Parallel()

	base := `<html style="margin:0"><body style="margin:0">` +
		`<div id="anchor" style="anchor-name:--a;width:100px;height:50px;margin-top:100px"></div>` +
		`<div id="target" style="position:absolute;left:10px;top:10px;width:50px;height:20px;position-anchor:--a%s"></div>` +
		`</body></html>`
	omitted := layoutHTML(t, fmt.Sprintf(base, ""))
	noneRes := layoutHTML(t, fmt.Sprintf(base, ";position-area:none"))

	omBox := boxByID(t, omitted, "target")
	noneBox := boxByID(t, noneRes, "target")

	if !near(omBox.x, noneBox.x) || !near(omBox.y, noneBox.y) {
		t.Errorf("none vs omitted = (%.4fpt, %.4fpt) vs (%.4fpt, %.4fpt), want identical",
			noneBox.x, noneBox.y, omBox.x, omBox.y)
	}
	if !near(omBox.x, pxToPt(10)) || !near(omBox.y, pxToPt(10)) {
		t.Errorf("omitted offsets = (%.4fpt, %.4fpt) = (%.2fpx, %.2fpx), want (10px, 10px)",
			omBox.x, omBox.y, omBox.x/ptPerCSSPx, omBox.y/ptPerCSSPx)
	}

	topRes := layoutHTML(t, fmt.Sprintf(base, ";position-area:top"))
	topBox := boxByID(t, topRes, "target")
	anchorBox := boxByID(t, topRes, "anchor")

	if !(topBox.y < anchorBox.y) {
		t.Errorf("top y=%.4fpt should be above anchor y=%.4fpt",
			topBox.y, anchorBox.y)
	}
	if !near(topBox.x, anchorBox.x) {
		t.Errorf("top x=%.4fpt should align to anchor x=%.4fpt",
			topBox.x, anchorBox.x)
	}
}

// TestBehaviorPositionAreaOutOfScope is anchor-scope: an anchor inside a
// scope does not move a target outside that subtree. The scoped-out target
// keeps its left/top offsets, matching a document with no anchor name. A
// sibling document without the scope moves above the anchor.
func TestBehaviorPositionAreaOutOfScope(t *testing.T) {
	t.Parallel()

	scoped := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="scope" style="anchor-scope:--a">`+
		`<div id="anchor" style="anchor-name:--a;width:100px;height:50px;margin-top:100px"></div>`+
		`</div>`+
		`<div id="target" style="position:absolute;left:10px;top:10px;width:50px;height:20px;position-anchor:--a;position-area:top"></div>`+
		`</body></html>`)
	missing := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="scope">`+
		`<div id="anchor" style="width:100px;height:50px;margin-top:100px"></div>`+
		`</div>`+
		`<div id="target" style="position:absolute;left:10px;top:10px;width:50px;height:20px;position-anchor:--a;position-area:top"></div>`+
		`</body></html>`)
	inScope := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="anchor" style="anchor-name:--a;width:100px;height:50px;margin-top:100px"></div>`+
		`<div id="target" style="position:absolute;left:10px;top:10px;width:50px;height:20px;position-anchor:--a;position-area:top"></div>`+
		`</body></html>`)

	scopedBox := boxByID(t, scoped, "target")
	missingBox := boxByID(t, missing, "target")

	if !near(scopedBox.x, missingBox.x) || !near(scopedBox.y, missingBox.y) {
		t.Errorf("scoped-out vs missing = (%.4fpt, %.4fpt) vs (%.4fpt, %.4fpt), want identical",
			scopedBox.x, scopedBox.y, missingBox.x, missingBox.y)
	}
	if !near(scopedBox.x, pxToPt(10)) || !near(scopedBox.y, pxToPt(10)) {
		t.Errorf("scoped-out offsets = (%.4fpt, %.4fpt) = (%.2fpx, %.2fpx), want (10px, 10px)",
			scopedBox.x, scopedBox.y, scopedBox.x/ptPerCSSPx, scopedBox.y/ptPerCSSPx)
	}

	movedBox := boxByID(t, inScope, "target")
	anchorBox := boxByID(t, inScope, "anchor")
	if !(movedBox.y < anchorBox.y) {
		t.Errorf("in-scope top y=%.4fpt should be above anchor y=%.4fpt",
			movedBox.y, anchorBox.y)
	}
}

// TestBehaviorPositionAreaStatic is position-area on an in-flow box: a
// static target keeps its flow Y with or without the anchor properties.
func TestBehaviorPositionAreaStatic(t *testing.T) {
	t.Parallel()

	withArea := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="anchor" style="anchor-name:--a;width:100px;height:50px"></div>`+
		`<div id="target" style="width:50px;height:20px;position-anchor:--a;position-area:top"></div>`+
		`</body></html>`)
	withoutArea := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="anchor" style="anchor-name:--a;width:100px;height:50px"></div>`+
		`<div id="target" style="width:50px;height:20px"></div>`+
		`</body></html>`)

	withBox := boxByID(t, withArea, "target")
	withoutBox := boxByID(t, withoutArea, "target")

	if !near(withBox.y, withoutBox.y) {
		t.Errorf("static with-area y=%.4fpt vs without y=%.4fpt, want identical flow Y",
			withBox.y, withoutBox.y)
	}
	if !near(withBox.x, withoutBox.x) {
		t.Errorf("static with-area x=%.4fpt vs without x=%.4fpt, want identical flow X",
			withBox.x, withoutBox.x)
	}
}

// TestBehaviorPositionTryAlwaysKeepsBase: the first fallback is tried only
// when the base fails visibility. With visibility always the base area wins
// and the fallback is unused.
func TestBehaviorPositionTryAlwaysKeepsBase(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="anchor" style="anchor-name:--foo;width:100px;height:20px;margin:100px 0 0 50px"></div>`+
		`<div id="tip" style="position:absolute;position-anchor:--foo;position-area:top;`+
		`width:60px;height:30px;position-try-fallbacks:bottom;position-visibility:always"></div>`+
		`</body></html>`)

	anchor := boxByID(t, res, "anchor")
	tip := boxByID(t, res, "tip")

	if len(tip.style.PositionTryFallbacks) != 1 || tip.style.PositionTryFallbacks[0] != "bottom" {
		t.Fatalf("fallbacks = %v, want [bottom]", tip.style.PositionTryFallbacks)
	}

	// Base top sits above the anchor; fallback bottom would sit below it.
	wantY := anchor.y - tip.height
	fallbackY := anchor.y + anchor.height

	if !near(tip.y, wantY) {
		t.Errorf("tip.y = %.4fpt (%.2fpx), want base top %.4fpt; fallback bottom would be %.4fpt",
			tip.y, tip.y/ptPerCSSPx, wantY, fallbackY)
	}

	if !near(tip.x, anchor.x) {
		t.Errorf("tip.x = %.4fpt, want anchor start edge %.4fpt", tip.x, anchor.x)
	}
}

// TestBehaviorPositionTryOrderNormalKeepsDocOrder: normal keeps fallback
// document order, so the base wins when it passes.
func TestBehaviorPositionTryOrderNormalKeepsDocOrder(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="anchor" style="anchor-name:--foo;width:50px;height:20px;margin:100px 0 0 50px"></div>`+
		`<div id="tip" style="position:absolute;position-anchor:--foo;position-area:left;`+
		`width:20px;height:20px;position-try-fallbacks:left,right;`+
		`position-try-order:normal;position-visibility:no-overflow"></div>`+
		`</body></html>`)

	anchor := boxByID(t, res, "anchor")
	tip := boxByID(t, res, "tip")

	wantX := anchor.x - tip.w
	if !near(tip.x, wantX) {
		t.Errorf("tip.x = %.4fpt (%.2fpx), want base left %.4fpt",
			tip.x, tip.x/ptPerCSSPx, wantX)
	}
}

// TestBehaviorPositionTryOrderMostInlineSizePicksWide: size keywords pick
// the largest candidate on that axis among passing no-overflow candidates.
// The anchor sits left of center, so the right area is wider than the left.
func TestBehaviorPositionTryOrderMostInlineSizePicksWide(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="anchor" style="anchor-name:--foo;width:50px;height:20px;margin:100px 0 0 50px"></div>`+
		`<div id="tip" style="position:absolute;position-anchor:--foo;position-area:left;`+
		`width:20px;height:20px;position-try-fallbacks:left,right;`+
		`position-try-order:most-inline-size;position-visibility:no-overflow"></div>`+
		`</body></html>`)

	anchor := boxByID(t, res, "anchor")
	tip := boxByID(t, res, "tip")

	if tip.style.PositionTryOrder != "most-inline-size" {
		t.Fatalf("order = %q, want most-inline-size", tip.style.PositionTryOrder)
	}

	wantX := anchor.x + anchor.w
	if !near(tip.x, wantX) {
		t.Errorf("tip.x = %.4fpt (%.2fpx), want wide right %.4fpt",
			tip.x, tip.x/ptPerCSSPx, wantX)
	}
}

// TestBehaviorPositionTryFallbacksStored: none, a list of position-area
// values, and try-tactic keywords are accepted, while an invalid value
// leaves the previous value.
func TestBehaviorPositionTryFallbacksStored(t *testing.T) {
	t.Parallel()

	s := sheet(t, `
.tip { position-try-fallbacks: top; }
.tactics { position-try-fallbacks: flip-block, flip-x flip-y, --my-try, top; }
.cleared { position-try-fallbacks: none; }
.bad { position-try-fallbacks: bogus; }
`)
	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="area" class="tip" style="position:absolute"></div>`+
		`<div id="mix" class="tip tactics" style="position:absolute"></div>`+
		`<div id="none" class="tip cleared" style="position:absolute"></div>`+
		`<div id="keep" class="tip bad" style="position:absolute"></div>`+
		`</body></html>`, s)

	area := boxByID(t, res, "area")
	if len(area.style.PositionTryFallbacks) != 1 || area.style.PositionTryFallbacks[0] != "top" {
		t.Errorf("area fallbacks = %v, want [top]", area.style.PositionTryFallbacks)
	}

	mix := boxByID(t, res, "mix")
	wantMix := []string{"flip-block", "flip-x flip-y", "--my-try", "top"}
	if len(mix.style.PositionTryFallbacks) != len(wantMix) {
		t.Fatalf("mixed fallbacks = %v, want %v", mix.style.PositionTryFallbacks, wantMix)
	}

	for i, want := range wantMix {
		if mix.style.PositionTryFallbacks[i] != want {
			t.Errorf("mixed[%d] = %q, want %q", i, mix.style.PositionTryFallbacks[i], want)
		}
	}

	none := boxByID(t, res, "none")
	if len(none.style.PositionTryFallbacks) != 0 {
		t.Errorf("none fallbacks = %v, want empty", none.style.PositionTryFallbacks)
	}

	keep := boxByID(t, res, "keep")
	if len(keep.style.PositionTryFallbacks) != 1 || keep.style.PositionTryFallbacks[0] != "top" {
		t.Errorf("invalid fallbacks = %v, want previous [top]", keep.style.PositionTryFallbacks)
	}
}

// TestBehaviorPositionTryOrderStored: normal and the four try-size keywords
// are accepted; an invalid value keeps the previous order.
func TestBehaviorPositionTryOrderStored(t *testing.T) {
	t.Parallel()

	s := sheet(t, `
.base { position-try-order: most-inline-size; }
.bad { position-try-order: bogus; }
`)
	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="good" class="base" style="position:absolute"></div>`+
		`<div id="keep" class="base bad" style="position:absolute"></div>`+
		`<div id="initial" style="position:absolute"></div>`+
		`</body></html>`, s)

	good := boxByID(t, res, "good")
	if good.style.PositionTryOrder != "most-inline-size" {
		t.Errorf("order = %q, want most-inline-size", good.style.PositionTryOrder)
	}

	keep := boxByID(t, res, "keep")
	if keep.style.PositionTryOrder != "most-inline-size" {
		t.Errorf("invalid order = %q, want previous most-inline-size", keep.style.PositionTryOrder)
	}

	initial := boxByID(t, res, "initial")
	if initial.style.PositionTryOrder != "" {
		t.Errorf("initial order = %q, want empty (normal behavior)", initial.style.PositionTryOrder)
	}
}

// TestBehaviorPositionTryShorthandExpands: position-try expands into order
// plus fallbacks, including fallbacks-only, and CSS-wide keywords reset both.
func TestBehaviorPositionTryShorthandExpands(t *testing.T) {
	t.Parallel()

	s := sheet(t, `
.only { position-try: flip-block; }
.both { position-try: most-width flip-block, top; }
.wide { position-try: initial; }
`)
	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="only" class="only" style="position:absolute"></div>`+
		`<div id="both" class="both" style="position:absolute"></div>`+
		`<div id="wide" class="both wide" style="position:absolute"></div>`+
		`</body></html>`, s)

	only := boxByID(t, res, "only")
	if only.style.PositionTryOrder != "normal" {
		t.Errorf("fallbacks-only order = %q, want normal", only.style.PositionTryOrder)
	}

	if len(only.style.PositionTryFallbacks) != 1 || only.style.PositionTryFallbacks[0] != "flip-block" {
		t.Errorf("fallbacks-only = %v, want [flip-block]", only.style.PositionTryFallbacks)
	}

	both := boxByID(t, res, "both")
	if both.style.PositionTryOrder != "most-width" {
		t.Errorf("both order = %q, want most-width", both.style.PositionTryOrder)
	}

	if len(both.style.PositionTryFallbacks) != 2 {
		t.Fatalf("both fallbacks = %v, want [flip-block top]", both.style.PositionTryFallbacks)
	}

	wide := boxByID(t, res, "wide")
	if wide.style.PositionTryOrder != "" || len(wide.style.PositionTryFallbacks) != 0 {
		t.Errorf("CSS-wide reset = order %q fallbacks %v, want empty and empty",
			wide.style.PositionTryOrder, wide.style.PositionTryFallbacks)
	}
}

// TestBehaviorPositionTryShorthandCascadeOrder: a following longhand
// overrides the shorthand, and a following shorthand resets an earlier
// longhand.
func TestBehaviorPositionTryShorthandCascadeOrder(t *testing.T) {
	t.Parallel()

	longhandWins := sheet(t, `
.tip { position-try: flip-block; }
.tip { position-try-order: most-width; }
`)
	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="tip" class="tip" style="position:absolute"></div>`+
		`</body></html>`, longhandWins)
	tip := boxByID(t, res, "tip")

	if tip.style.PositionTryOrder != "most-width" {
		t.Errorf("longhand-after order = %q, want most-width", tip.style.PositionTryOrder)
	}

	if len(tip.style.PositionTryFallbacks) != 1 || tip.style.PositionTryFallbacks[0] != "flip-block" {
		t.Errorf("longhand-after fallbacks = %v, want [flip-block]", tip.style.PositionTryFallbacks)
	}

	shorthandWins := sheet(t, `
.tip { position-try-order: most-width; }
.tip { position-try: flip-block; }
`)
	res2 := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="tip" class="tip" style="position:absolute"></div>`+
		`</body></html>`, shorthandWins)
	tip2 := boxByID(t, res2, "tip")

	if tip2.style.PositionTryOrder != "normal" {
		t.Errorf("shorthand-after order = %q, want normal reset", tip2.style.PositionTryOrder)
	}

	if len(tip2.style.PositionTryFallbacks) != 1 || tip2.style.PositionTryFallbacks[0] != "flip-block" {
		t.Errorf("shorthand-after fallbacks = %v, want [flip-block]", tip2.style.PositionTryFallbacks)
	}
}

// TestBehaviorPositionVisibility covers visibility storage, always keeping
// the off-viewport base with its ops, and anchor-valid keeping ordinary
// placement for missing or out-of-scope anchors.
func TestBehaviorPositionVisibility(t *testing.T) {
	t.Parallel()

	t.Run("store", func(t *testing.T) {
		t.Parallel()

		for _, tok := range []string{"always", "anchor-valid", "anchor-visible", "no-overflow", "anchor-valid no-overflow"} {
			if _, ok := parseAnchorVisibility(tok); !ok {
				t.Errorf("parseAnchorVisibility(%q) = invalid, want valid", tok)
			}
		}

		for _, tok := range []string{"", "bogus", "anchor-valids", "anchor-invalid", "always anchor-valid", "anchor-valid anchor-valid"} {
			if _, ok := parseAnchorVisibility(tok); ok {
				t.Errorf("parseAnchorVisibility(%q) = valid, want invalid", tok)
			}
		}

		sty := initialStyle()
		if sty.PositionVisibility != "" {
			t.Fatalf("initial visibility = %q, want empty (anchor-valid behavior)", sty.PositionVisibility)
		}

		if !applyAnchorProps(&sty, "position-visibility", "anchor-valid", 0, nil, nil, false) {
			t.Fatalf("applyAnchorProps did not own position-visibility")
		}
		if sty.PositionVisibility != "anchor-valid" {
			t.Fatalf("visibility = %q, want anchor-valid", sty.PositionVisibility)
		}

		// Invalid must not replace the earlier valid value.
		if supportedDeclaration("position-visibility", "bogus") {
			t.Fatal("supportedDeclaration(bogus) = true, want false")
		}
		applyAnchorProps(&sty, "position-visibility", "bogus", 0, nil, nil, false)
		if sty.PositionVisibility != "anchor-valid" {
			t.Errorf("invalid replaced valid: visibility = %q, want anchor-valid", sty.PositionVisibility)
		}
	})

	t.Run("always-keeps-above", func(t *testing.T) {
		t.Parallel()

		res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
			`<div id="anchor" style="anchor-name:--a;width:100px;height:50px;background-color:#eeeeee"></div>`+
			`<div id="tip" style="position:absolute;position-anchor:--a;position-area:top;`+
			`position-visibility:always;width:50px;height:40px;left:0;top:0;background-color:#ff0000"></div>`+
			`</body></html>`)

		anchor := boxByID(t, res, "anchor")
		tip := boxByID(t, res, "tip")

		if !near(tip.x, anchor.x) {
			t.Errorf("tip.x = %.4fpt, anchor.x = %.4fpt, want aligned start edge", tip.x, anchor.x)
		}
		if !(tip.y < anchor.y) {
			t.Errorf("tip.y = %.4fpt, anchor.y = %.4fpt, want tip above anchor", tip.y, anchor.y)
		}
		if !(tip.y < 0) {
			t.Errorf("tip.y = %.4fpt, want negative (hanging off viewport top)", tip.y)
		}
		if len(res.Ops) == 0 {
			t.Fatalf("no ops, want tip fill retained")
		}
		found := false
		for _, op := range res.Ops {
			if op.Kind == OpFillRect && near(op.X, tip.x) && near(op.Y, tip.y) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tip fill at (%.4fpt, %.4fpt) missing from Ops, want retained off-viewport base", tip.x, tip.y)
		}
	})

	t.Run("always-keeps-past-height", func(t *testing.T) {
		t.Parallel()

		src := `<html style="margin:0"><body style="margin:0">` +
			`<div style="height:90px"></div>` +
			`<div id="anchor" style="anchor-name:--a;width:100px;height:20px;background-color:#eeeeee"></div>` +
			`<div id="tip" style="position:absolute;position-anchor:--a;position-area:bottom;` +
			`position-visibility:always;width:50px;height:40px;left:0;top:0;background-color:#ff0000"></div>` +
			`</body></html>`
		root, err := html.Parse(src)
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		viewportH := pxToPt(100)
		res, err := Layout(root, Options{Width: pxToPt(400), Height: viewportH, Background: true})
		if err != nil {
			t.Fatalf("Layout: %v", err)
		}

		tip := boxByID(t, res, "tip")
		if !(tip.y > viewportH) {
			t.Errorf("tip.y = %.4fpt, viewportH = %.4fpt, want past height", tip.y, viewportH)
		}
		found := false
		for _, op := range res.Ops {
			if op.Kind == OpFillRect && near(op.X, tip.x) && near(op.Y, tip.y) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tip fill at (%.4fpt, %.4fpt) missing from Ops, want retained past height", tip.x, tip.y)
		}
	})

	t.Run("anchor-valid-dangling", func(t *testing.T) {
		t.Parallel()

		res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
			`<div id="tip" style="position:absolute;position-anchor:--missing;position-area:top;`+
			`position-visibility:anchor-valid;left:16px;top:32px;width:50px;height:40px;background-color:#ff0000"></div>`+
			`</body></html>`)

		tip := boxByID(t, res, "tip")
		if !near(tip.x, pxToPt(16)) || !near(tip.y, pxToPt(32)) {
			t.Errorf("tip = (%.4fpt, %.4fpt), want (16px, 32px) ordinary absolute",
				tip.x, tip.y)
		}
		found := false
		for _, op := range res.Ops {
			if op.Kind == OpFillRect && near(op.X, tip.x) && near(op.Y, tip.y) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("dangling tip fill at (%.4fpt, %.4fpt) missing, want still painted", tip.x, tip.y)
		}
	})

	t.Run("anchor-valid-out-of-scope", func(t *testing.T) {
		t.Parallel()

		res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
			`<div id="scope" style="anchor-scope:--a">`+
			`<div id="anchor" style="anchor-name:--a;width:100px;height:50px"></div>`+
			`</div>`+
			`<div id="tip" style="position:absolute;position-anchor:--a;position-area:top;`+
			`position-visibility:anchor-valid;left:16px;top:32px;width:50px;height:40px;background-color:#ff0000"></div>`+
			`</body></html>`)

		tip := boxByID(t, res, "tip")
		if !near(tip.x, pxToPt(16)) || !near(tip.y, pxToPt(32)) {
			t.Errorf("scoped-out tip = (%.4fpt, %.4fpt), want (16px, 32px) ordinary absolute",
				tip.x, tip.y)
		}
		found := false
		for _, op := range res.Ops {
			if op.Kind == OpFillRect && near(op.X, tip.x) && near(op.Y, tip.y) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("scoped-out tip fill missing, want still painted")
		}
	})
}

// TestBehaviorPositionVisibilityNoOverflow covers no-overflow fallback
// inside the layout viewport, and anchor-visible matching anchor-valid
// while no scroll offset exists.
func TestBehaviorPositionVisibilityNoOverflow(t *testing.T) {
	t.Parallel()

	t.Run("NoOverflowFallsBackInsideViewport", func(t *testing.T) {
		t.Parallel()

		res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
			`<div id="anchor" style="anchor-name:--a;width:100px;height:40px;background-color:red"></div>`+
			`<div id="tip" style="position:absolute;position-anchor:--a;position-area:top;`+
			`position-try-fallbacks:right;position-visibility:no-overflow;`+
			`width:60px;height:20px;background-color:blue"></div>`+
			`</body></html>`)

		anchor := boxByID(t, res, "anchor")
		tip := boxByID(t, res, "tip")

		// Base candidate "top" would sit above the canvas
		// (anchor.y - tip.height < 0) so no-overflow rejects it. The
		// fallback "right" sits beside the anchor inside the viewport and wins.
		wantX := anchor.x + anchor.w
		wantY := anchor.y

		if !near(tip.x, wantX) || !near(tip.y, wantY) {
			t.Errorf("tip = (%.4fpt, %.4fpt) = (%.2fpx, %.2fpx), want beside anchor (%.2fpx, %.2fpx)",
				tip.x, tip.y, tip.x/ptPerCSSPx, tip.y/ptPerCSSPx, wantX/ptPerCSSPx, wantY/ptPerCSSPx)
		}

		if tip.height <= 0 {
			t.Errorf("tip.height = %.4fpt, want positive", tip.height)
		}
	})

	t.Run("AnchorVisibleMatchesAnchorValid", func(t *testing.T) {
		t.Parallel()

		// This tree has no scroll offset and blinkless returns one drawing
		// list. Until a scroll offset exists, anchor-visible is stored and
		// treated like anchor-valid on this canvas. Next gate: a test with
		// a non-zero scroll offset hides the box when the anchor lies
		// outside that port.
		doc := func(vis string) string {
			return `<html style="margin:0"><body style="margin:0">` +
				`<div id="anchor" style="anchor-name:--a;width:100px;height:40px;background-color:red"></div>` +
				`<div id="tip" style="position:absolute;position-anchor:--a;position-area:bottom;` +
				`position-visibility:` + vis + `;width:60px;height:20px;background-color:blue"></div>` +
				`</body></html>`
		}

		resValid := layoutHTML(t, doc("anchor-valid"))
		resVisible := layoutHTML(t, doc("anchor-visible"))

		tipValid := boxByID(t, resValid, "tip")
		tipVisible := boxByID(t, resVisible, "tip")

		if tipValid.style.PositionVisibility != "anchor-valid" {
			t.Errorf("anchor-valid stored = %q, want %q", tipValid.style.PositionVisibility, "anchor-valid")
		}

		if tipVisible.style.PositionVisibility != "anchor-visible" {
			t.Errorf("anchor-visible stored = %q, want %q", tipVisible.style.PositionVisibility, "anchor-visible")
		}

		if !near(tipValid.x, tipVisible.x) || !near(tipValid.y, tipVisible.y) {
			t.Errorf("anchor-visible tip = (%.4fpt, %.4fpt), anchor-valid tip = (%.4fpt, %.4fpt), want identical",
				tipVisible.x, tipVisible.y, tipValid.x, tipValid.y)
		}

		if tipVisible.opEnd < tipVisible.opStart {
			t.Errorf("anchor-visible tip emitted no ops (box hidden), want paint ops kept on this canvas")
		}
	})
}
