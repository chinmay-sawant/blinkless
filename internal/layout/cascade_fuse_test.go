package layout

import (
	"reflect"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// The fuse tests pin the cascade memo work in style_cascade.go and
// style_value_accept.go. Verdict and probe caches are process-global, so every
// test here stays sequential (no t.Parallel): parallel cascade tests share
// these caches and would perturb the counters.

// TestFuseGoldenCascadeVarsPseudosSupports resolves one generated sheet that
// exercises custom-property substitution, generated content, and @supports
// gating, and pins the used values. It is the behavior golden for the fuse
// work: identical input must produce identical output with the memos on.
func TestFuseGoldenCascadeVarsPseudosSupports(t *testing.T) { //nolint:paralleltest // global cascade memos
	root := mustParse(t, `<html><body>`+
		`<div id="a" class="box" style="width: 80px">a</div>`+
		`<div id="b" class="box">b</div></body></html>`)
	sh := sheet(t, `
:root { --w: 120px; --accent: #00ff00 }
.box { width: var(--w); color: var(--accent) }
@supports (display: grid) { .box { display: grid } }
@supports (display: bogus) { .box { width: 4px } }
`)
	styles := resolveStyles(root, []*css.Stylesheet{sh}, "print", testViewport, 800)

	a := styles[findElementByID(t, root, "a")]
	if got := a.Width; !near(got, 60) {
		t.Fatalf("inline width = %.2fpt, want 60 (80px beats var(--w))", got)
	}

	b := styles[findElementByID(t, root, "b")]
	if got := b.Width; !near(got, 90) {
		t.Fatalf("var width = %.2fpt, want 90 (120px from --w)", got)
	}

	if got := b.Color; got != [3]float64{0, 1, 0} {
		t.Fatalf("var color = %v, want green from --accent", got)
	}

	if b.Display != displayGrid {
		t.Fatalf("Display = %q, want grid from valid @supports", b.Display)
	}

	res := layoutHTML(t, `<html><body><p class="x">base</p></body></html>`, sheet(t, `
.x { --accent: #00ff00 }
.x::after { content: "fuse-tail"; color: var(--accent) }
`))

	for _, oper := range res.Ops {
		if oper.Kind != OpText || oper.Text != "fuse-tail" {
			continue
		}

		if !(oper.G > 0.9 && oper.R < 0.1 && oper.B < 0.1) {
			t.Fatalf("::after color = %.3f,%.3f,%.3f, want green from var(--accent)", oper.R, oper.G, oper.B)
		}

		return
	}

	t.Fatal("::after content op not found")
}

// TestFuseAcceptVerdictParsesOnce proves the fused path: one validation parse
// per unique declaration no matter how many elements carry it. The second
// identical call must hit the verdict memo, and neighboring prop/value pairs
// must not alias each other.
func TestFuseAcceptVerdictParsesOnce(t *testing.T) { //nolint:paralleltest // global cascade memos
	resetAcceptVerdictForTest()

	if supportedDeclaration("display", "bogus") {
		t.Fatal("display:bogus accepted, want rejected")
	}

	if supportedDeclaration("display", "bogus") {
		t.Fatal("display:bogus accepted on repeat, want rejected")
	}

	evals, hits := acceptVerdictStatsForTest()
	if evals != 1 || hits != 1 {
		t.Fatalf("accept evals=%d hits=%d, want 1 and 1", evals, hits)
	}

	if !supportedDeclaration("width", "80px") {
		t.Fatal("width:80px rejected, want accepted")
	}

	if !supportedDeclaration("width", "80px") {
		t.Fatal("width:80px rejected on repeat, want accepted")
	}

	evals, hits = acceptVerdictStatsForTest()
	if evals != 2 || hits != 2 {
		t.Fatalf("accept evals=%d hits=%d, want 2 and 2", evals, hits)
	}

	if !supportedDeclaration("margin-top", "auto") {
		t.Fatal("margin-top:auto rejected, want accepted")
	}

	if supportedDeclaration("padding-top", "auto") {
		t.Fatal("padding-top:auto accepted, want rejected (must not alias margin-top:auto)")
	}

	if !supportedDeclaration("display", "block") {
		t.Fatal("display:block rejected, want accepted (must not alias display:bogus)")
	}
}

// TestFuseEmptyStyleSkipsInlineParse resolves elements with no style
// attribute (or a blank one) and requires zero inline parses.
func TestFuseEmptyStyleSkipsInlineParse(t *testing.T) { //nolint:paralleltest // global cascade memos
	resetCascadeCachesForTest()
	before := inlineDeclParseCountForTest()

	root := mustParse(t, `<html><body><div>plain</div><p style="">empty</p><span style="   ">blank</span></body></html>`)
	_ = resolveStyles(root, nil, "print", testViewport, 800)

	if got := inlineDeclParseCountForTest() - before; got != 0 {
		t.Fatalf("inline parses for style-less elements = %d, want 0", got)
	}
}

// TestFuseInlineDeclsInternedPerStyle resolves two elements sharing one
// inline style string and requires a single parse with identical winners.
func TestFuseInlineDeclsInternedPerStyle(t *testing.T) { //nolint:paralleltest // global cascade memos
	resetCascadeCachesForTest()
	before := inlineDeclParseCountForTest()

	root := mustParse(t, `<html><body>`+
		`<div id="i1" style="width: 77px; color: #123456">a</div>`+
		`<div id="i2" style="width: 77px; color: #123456">b</div></body></html>`)
	styles := resolveStyles(root, nil, "print", testViewport, 800)

	if got := inlineDeclParseCountForTest() - before; got != 1 {
		t.Fatalf("inline parses for one shared style string = %d, want 1", got)
	}

	first := styles[findElementByID(t, root, "i1")]
	second := styles[findElementByID(t, root, "i2")]

	if !near(first.Width, 57.75) || !near(second.Width, 57.75) {
		t.Fatalf("shared inline widths = %.2f,%.2f, want 57.75,57.75", first.Width, second.Width)
	}

	if first.Color != second.Color {
		t.Fatalf("shared inline colors differ: %v vs %v", first.Color, second.Color)
	}
}

// TestFuseSupportsProbeMemoized proves the @supports probe runs once per
// unique prop and value: the repeat hits the verdict memo instead of
// rebuilding the scratch style and dispatching every group again.
func TestFuseSupportsProbeMemoized(t *testing.T) { //nolint:paralleltest // global cascade memos
	resetCascadeCachesForTest()

	if !engineSupportsProperty("display", "grid") {
		t.Fatal("supports display:grid rejected, want accepted")
	}

	if !engineSupportsProperty("display", "grid") {
		t.Fatal("supports display:grid rejected on repeat, want accepted")
	}

	if engineSupportsProperty("display", "bogus") {
		t.Fatal("supports display:bogus accepted, want rejected")
	}

	evals, hits := supportsProbeStatsForTest()
	if evals != 2 || hits != 1 {
		t.Fatalf("supports evals=%d hits=%d, want 2 and 1", evals, hits)
	}
}

// TestFuseCustomPropsShareParentMap pins the copy-on-write rule: a child
// that declares no custom properties shares the parent map (no clone), while
// a child that declares one gets its own map (the parent is untouched).
func TestFuseCustomPropsShareParentMap(t *testing.T) { //nolint:paralleltest // global cascade memos
	root := mustParse(t, `<html><body><div id="p"><div id="c" class="tint">x</div>`+
		`<div id="w" style="--x: 1">y</div></div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `:root { --brand: teal } .tint { color: #ff0000 }`)},
		"print", testViewport, 800)

	parent := styles[findElementByID(t, root, "p")]
	child := styles[findElementByID(t, root, "c")]

	if child.CustomProps["--brand"] != "teal" {
		t.Fatalf("child --brand = %q, want teal inherited", child.CustomProps["--brand"])
	}

	if reflect.ValueOf(child.CustomProps).Pointer() != reflect.ValueOf(parent.CustomProps).Pointer() {
		t.Fatal("child custom map is a copy, want the shared parent map until a write")
	}

	writer := styles[findElementByID(t, root, "w")]
	if writer.CustomProps["--x"] != "1" {
		t.Fatalf("writer --x = %q, want 1", writer.CustomProps["--x"])
	}

	if reflect.ValueOf(writer.CustomProps).Pointer() == reflect.ValueOf(parent.CustomProps).Pointer() {
		t.Fatal("declaring child shares the parent custom map, want its own map on write")
	}

	if _, ok := parent.CustomProps["--x"]; ok {
		t.Fatal("parent custom map gained --x, want the declaring child to own its write")
	}
}
