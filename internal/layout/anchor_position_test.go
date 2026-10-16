//nolint:cyclop,funlen,lll,goconst // scope proofs
package layout

import (
	"slices"
	"testing"
)

// TestAnchorNameScope stores anchor-name and anchor-scope and proves scope
// lookup through placement: a descendant inside the scope moves against the
// anchor, a box outside it keeps its offsets.
func TestAnchorNameScope(t *testing.T) {
	t.Parallel()

	// Two boxes carry different names.
	res := layoutHTML(t, `<html><body>`+
		`<div id="a" style="anchor-name: --a"></div>`+
		`<div id="b" style="anchor-name: --b"></div>`+
		`</body></html>`)
	aNames := boxByID(t, res, "a").style.AnchorNames
	bNames := boxByID(t, res, "b").style.AnchorNames

	if !slices.Equal(aNames, []string{"--a"}) {
		t.Fatalf("a anchor-name = %q, want [--a]", aNames)
	}

	if !slices.Equal(bNames, []string{"--b"}) {
		t.Fatalf("b anchor-name = %q, want [--b]", bNames)
	}

	// Comma list stores both names.
	multi := layoutHTML(t, `<html><body><div id="m" style="anchor-name: --a, --b"></div></body></html>`)
	if got := boxByID(t, multi, "m").style.AnchorNames; !slices.Equal(got, []string{"--a", "--b"}) {
		t.Fatalf("multi anchor-name = %q, want [--a --b]", got)
	}

	// Missing leaves the initial (empty).
	missing := layoutHTML(t, `<html><body><div id="p"></div></body></html>`)
	if got := boxByID(t, missing, "p").style.AnchorNames; len(got) != 0 {
		t.Fatalf("missing anchor-name = %q, want empty", got)
	}

	// Invalid leaves the previous valid value: the later bogus rule loses.
	invalidSheet := sheet(t, `#a { anchor-name: --a; } #a { anchor-name: bogus; }`)
	invalidRes := layoutHTML(t, `<html><body><div id="a"></div></body></html>`, invalidSheet)

	if got := boxByID(t, invalidRes, "a").style.AnchorNames; !slices.Equal(got, []string{"--a"}) {
		t.Fatalf("invalid anchor-name kept = %q, want [--a]", got)
	}

	// A lone invalid declaration keeps the initial.
	loneInvalid := layoutHTML(t, `<html><body><div id="q" style="anchor-name: bogus"></div></body></html>`)
	if got := boxByID(t, loneInvalid, "q").style.AnchorNames; len(got) != 0 {
		t.Fatalf("lone invalid anchor-name = %q, want empty", got)
	}

	// none clears the list: later none wins over an earlier name.
	noneSheet := sheet(t, `#c { anchor-name: --a; } #c { anchor-name: none; }`)
	noneRes := layoutHTML(t, `<html><body><div id="c"></div></body></html>`, noneSheet)

	if got := boxByID(t, noneRes, "c").style.AnchorNames; len(got) != 0 {
		t.Fatalf("anchor-name none = %q, want empty", got)
	}

	// anchor-scope storage: none, all, and a name list.
	scopeRes := layoutHTML(t, `<html><body>`+
		`<div id="snone" style="anchor-scope: none"></div>`+
		`<div id="sall" style="anchor-scope: all"></div>`+
		`<div id="slist" style="anchor-scope: --a, --b"></div>`+
		`<div id="splain"></div>`+
		`</body></html>`)

	if got := boxByID(t, scopeRes, "snone").style.AnchorScope; got != "none" {
		t.Fatalf("scope none = %q, want none", got)
	}

	if got := boxByID(t, scopeRes, "sall").style.AnchorScope; got != "all" {
		t.Fatalf("scope all = %q, want all", got)
	}

	if got := boxByID(t, scopeRes, "slist").style.AnchorScope; got != "--a --b" {
		t.Fatalf("scope list = %q, want --a --b", got)
	}

	if got := boxByID(t, scopeRes, "splain").style.AnchorScope; got != "" {
		t.Fatalf("scope initial = %q, want empty", got)
	}

	// Invalid scope leaves the previous value.
	badScopeSheet := sheet(t, `#b { anchor-scope: --a; } #b { anchor-scope: bogus; }`)
	badScopeRes := layoutHTML(t, `<html><body><div id="b"></div></body></html>`, badScopeSheet)

	if got := boxByID(t, badScopeRes, "b").style.AnchorScope; got != "--a" {
		t.Fatalf("invalid scope kept = %q, want --a", got)
	}

	// Same-element scope: all hides from outside, inside still moves.
	targetStyle := `position:absolute;left:10px;top:10px;width:50px;height:20px;position-anchor:--a;position-area:bottom`
	sameRes := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="scope" style="anchor-name: --a; anchor-scope: all;width:100px;height:50px">`+
		`<div id="inner" style="`+targetStyle+`"></div></div>`+
		`<div id="outside" style="`+targetStyle+`"></div>`+
		`</body></html>`)
	sameAnchor := boxByID(t, sameRes, "scope")
	sameInner := boxByID(t, sameRes, "inner")
	sameOutside := boxByID(t, sameRes, "outside")

	if !near(sameInner.y, sameAnchor.y+sameAnchor.height) {
		t.Fatalf("inner y=%.4fpt, want anchor bottom %.4fpt", sameInner.y, sameAnchor.y+sameAnchor.height)
	}

	if !near(sameOutside.x, pxToPt(10)) || !near(sameOutside.y, pxToPt(10)) {
		t.Fatalf("outside = (%.4fpt, %.4fpt), want kept (10px, 10px) offsets",
			sameOutside.x, sameOutside.y)
	}

	// Name list hides only those names.
	listRes := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="scope" style="anchor-name: --a, --b; anchor-scope: --a;width:100px;height:50px"></div>`+
		`<div id="hidden" style="`+targetStyle+`"></div>`+
		`<div id="shown" style="position:absolute;left:10px;top:10px;width:50px;height:20px;position-anchor:--b;position-area:bottom"></div>`+
		`</body></html>`)
	listAnchor := boxByID(t, listRes, "scope")
	listHidden := boxByID(t, listRes, "hidden")
	listShown := boxByID(t, listRes, "shown")

	if !near(listHidden.x, pxToPt(10)) || !near(listHidden.y, pxToPt(10)) {
		t.Fatalf("scoped-out --a = (%.4fpt, %.4fpt), want kept offsets",
			listHidden.x, listHidden.y)
	}

	if !near(listShown.y, listAnchor.y+listAnchor.height) {
		t.Fatalf("unscoped --b y=%.4fpt, want anchor bottom %.4fpt",
			listShown.y, listAnchor.y+listAnchor.height)
	}

	// Ancestor scope: name declared inside is visible inside, hidden outside.
	wrapRes := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="wrap" style="anchor-scope: --a">`+
		`<div id="anchor" style="anchor-name: --a;width:100px;height:50px"></div>`+
		`<div id="inwrap" style="`+targetStyle+`"></div></div>`+
		`<div id="outwrap" style="`+targetStyle+`"></div>`+
		`</body></html>`)
	wrapAnchor := boxByID(t, wrapRes, "anchor")
	wrapIn := boxByID(t, wrapRes, "inwrap")
	wrapOut := boxByID(t, wrapRes, "outwrap")

	if !near(wrapIn.y, wrapAnchor.y+wrapAnchor.height) {
		t.Fatalf("inwrap y=%.4fpt, want anchor bottom %.4fpt",
			wrapIn.y, wrapAnchor.y+wrapAnchor.height)
	}

	if !near(wrapOut.x, pxToPt(10)) || !near(wrapOut.y, pxToPt(10)) {
		t.Fatalf("outwrap = (%.4fpt, %.4fpt), want kept offsets",
			wrapOut.x, wrapOut.y)
	}

	// none does not hide: both sides move.
	noneScopeRes := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="scope" style="anchor-name: --a; anchor-scope: none;width:100px;height:50px"></div>`+
		`<div id="tip" style="`+targetStyle+`"></div>`+
		`</body></html>`)
	noneAnchor := boxByID(t, noneScopeRes, "scope")
	noneTip := boxByID(t, noneScopeRes, "tip")

	if !near(noneTip.y, noneAnchor.y+noneAnchor.height) {
		t.Fatalf("none-scope tip y=%.4fpt, want anchor bottom %.4fpt",
			noneTip.y, noneAnchor.y+noneAnchor.height)
	}

	// Auto with no anchor-name anywhere keeps offsets.
	emptyRes := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="x"><div id="y" style="position:absolute;left:10px;top:10px;width:50px;height:20px;position-anchor:auto;position-area:bottom"></div></div>`+
		`</body></html>`)
	emptyTip := boxByID(t, emptyRes, "y")

	if !near(emptyTip.x, pxToPt(10)) || !near(emptyTip.y, pxToPt(10)) {
		t.Fatalf("auto with no anchors = (%.4fpt, %.4fpt), want kept offsets",
			emptyTip.x, emptyTip.y)
	}
}

// TestAnchorScopeLookup covers the name map and the position-anchor
// reference in both directions: a descendant inside an anchor-scope moves
// against the name, a box outside it does not. Auto resolves only with
// exactly one in-scope name.
func TestAnchorScopeLookup(t *testing.T) {
	t.Parallel()

	src := `<html style="margin:0"><body style="margin:0">` +
		`<div id="scope" style="anchor-scope: --tip">` +
		`<div id="anchor" style="anchor-name: --tip;width:100px;height:50px"></div>` +
		`<div id="inner" style="position:absolute;left:10px;top:10px;width:50px;height:20px;position-anchor:--tip;position-area:bottom"></div>` +
		`</div>` +
		`<div id="outer" style="position:absolute;left:10px;top:10px;width:50px;height:20px;position-anchor:--tip;position-area:bottom"></div>` +
		`</body></html>`

	res := layoutHTML(t, src)
	if res == nil {
		t.Fatal("Layout returned nil")
	}

	anchor := boxByID(t, res, "anchor")
	inner := boxByID(t, res, "inner")
	outer := boxByID(t, res, "outer")

	if got := boxByID(t, res, "anchor").style.AnchorNames; !slices.Equal(got, []string{"--tip"}) {
		t.Fatalf("anchor-name used value = %v, want [--tip]", got)
	}

	if got := inner.style.PositionAnchor; got != "--tip" {
		t.Fatalf("position-anchor used value = %q, want %q", got, "--tip")
	}

	if !near(inner.y, anchor.y+anchor.height) {
		t.Fatalf("inner y=%.4fpt, want anchor bottom %.4fpt", inner.y, anchor.y+anchor.height)
	}

	if !near(outer.x, pxToPt(10)) || !near(outer.y, pxToPt(10)) {
		t.Fatalf("outer = (%.4fpt, %.4fpt), want kept (10px, 10px) offsets", outer.x, outer.y)
	}

	// Auto with exactly one in-scope name resolves to that anchor.
	autoRes := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="anchor" style="anchor-name: --solo;width:100px;height:50px"></div>`+
		`<div id="target" style="position:absolute;left:10px;top:10px;width:50px;height:20px;position-anchor:auto;position-area:bottom"></div>`+
		`</body></html>`)
	autoAnchor := boxByID(t, autoRes, "anchor")
	autoTarget := boxByID(t, autoRes, "target")

	if !near(autoTarget.y, autoAnchor.y+autoAnchor.height) {
		t.Fatalf("auto y=%.4fpt, want anchor bottom %.4fpt",
			autoTarget.y, autoAnchor.y+autoAnchor.height)
	}

	// Auto with two in-scope names does not guess.
	twoRes := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="first" style="anchor-name: --one;width:100px;height:50px"></div>`+
		`<div id="second" style="anchor-name: --two;width:100px;height:50px"></div>`+
		`<div id="target" style="position:absolute;left:10px;top:10px;width:50px;height:20px;position-anchor:auto;position-area:bottom"></div>`+
		`</body></html>`)
	twoTarget := boxByID(t, twoRes, "target")

	if !near(twoTarget.x, pxToPt(10)) || !near(twoTarget.y, pxToPt(10)) {
		t.Fatalf("two-name auto = (%.4fpt, %.4fpt), want kept offsets",
			twoTarget.x, twoTarget.y)
	}

	// The cascade gate agrees with the applier: invalid values are rejected
	// before winner selection, valid ones pass.
	for _, sample := range []struct {
		prop  string
		value string
		want  bool
	}{
		{"anchor-name", "--a", true},
		{"anchor-name", "--a, --b", true},
		{"anchor-name", "none", true},
		{"anchor-name", "bogus", false},
		{"anchor-name", "--a, bogus", false},
		{"anchor-scope", "all", true},
		{"anchor-scope", "none", true},
		{"anchor-scope", "--a", true},
		{"anchor-scope", "bogus", false},
		{"position-anchor", "auto", true},
		{"position-anchor", "--a", true},
		{"position-anchor", "bogus", false},
		{"position-anchor", "none", false},
	} {
		if got := supportedDeclaration(sample.prop, sample.value); got != sample.want {
			t.Errorf("supportedDeclaration(%q, %q) = %v, want %v",
				sample.prop, sample.value, got, sample.want)
		}
	}
}

// TestAnchorInvalidLeavesPrevious checks that an invalid declaration keeps
// the previous used value through the shared apply path.
func TestAnchorInvalidLeavesPrevious(t *testing.T) {
	t.Parallel()

	var style ResolvedStyle

	ctx := &styleContext{}

	applyStyleProp(&style, "anchor-name", "--kept", 12, ctx, nil, false)
	applyStyleProp(&style, "anchor-name", "bogus", 12, ctx, nil, false)

	if len(style.AnchorNames) != 1 || style.AnchorNames[0] != "--kept" {
		t.Fatalf("anchor-name after invalid = %v, want [--kept]", style.AnchorNames)
	}

	applyStyleProp(&style, "position-anchor", "--kept", 12, ctx, nil, false)
	applyStyleProp(&style, "position-anchor", "bogus", 12, ctx, nil, false)

	if style.PositionAnchor != "--kept" {
		t.Fatalf("position-anchor after invalid = %q, want %q", style.PositionAnchor, "--kept")
	}

	applyStyleProp(&style, "position-anchor", "auto", 12, ctx, nil, false)

	if style.PositionAnchor != "auto" {
		t.Fatalf("position-anchor auto = %q, want auto", style.PositionAnchor)
	}

	applyStyleProp(&style, "anchor-scope", "--kept", 12, ctx, nil, false)
	applyStyleProp(&style, "anchor-scope", "bogus", 12, ctx, nil, false)

	if style.AnchorScope != "--kept" {
		t.Fatalf("anchor-scope after invalid = %q, want --kept", style.AnchorScope)
	}
}
