//nolint:cyclop,varnamelen,dupword // behavior fixtures: long table tests, one-char ids, repeated filler
package layout

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// Named flows, exclusions, masonry tolerance, and wrap breaks (v0.0.3
// regions ledger). Every case proves a drawing-list outcome on the one
// canvas this engine returns; nothing here allocates Result.Pages.

// goldenFixture reads one named testdata/golden fixture.
func goldenFixture(t *testing.T, name string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}

	return string(raw)
}

// layoutGoldenFixture lays out a golden fixture with its <style> sheets
// applied (layoutHTML does not collect document stylesheets).
func layoutGoldenFixture(t *testing.T, name string) *Result {
	t.Helper()

	root := mustParse(t, goldenFixture(t, name))

	var sheets []*css.Stylesheet

	root.Walk(func(node *html.Node) {
		if node.Name != "style" {
			return
		}

		var text string

		for _, child := range node.Children {
			if child.Type == html.TextNode {
				text += child.Text
			}
		}

		if parsed, err := css.Parse(text); err == nil {
			sheets = append(sheets, parsed)
		}
	})

	res, err := Layout(root, Options{
		Width: testViewport, Height: 800, Sheets: sheets, Background: true,
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	return res
}

// textOpContaining returns the first text op whose Text contains needle.
func textOpContaining(t *testing.T, res *Result, needle string) Op {
	t.Helper()

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, needle) {
			return op
		}
	}

	t.Fatalf("no text op contains %q", needle)

	return Op{}
}

// TestBehaviorNamedFlowPropertyStorage pins the named-flow, tolerance,
// exclusion, and wrap-break used values and the cascade gates: invalid values
// do not replace an earlier valid declaration.
//
//nolint:funlen // one table of stored used values plus the cascade gate cases
func TestBehaviorNamedFlowPropertyStorage(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div id="a" style="flow-into:article content;flow-from:region-a;region-fragment:break;`+
		`flow-tolerance:8px;wrap-flow:both;wrap-through:none;`+
		`wrap-before:line;wrap-after:avoid-line;wrap-inside:avoid"></div>`+
		`<div id="b" style="flow-into:story element"></div>`+
		`<div id="baregion" style="flow-from:story"></div>`+
		`<div id="c" style="flow-into:none;flow-from:none;region-fragment:auto;`+
		`flow-tolerance:infinite;wrap-flow:auto;wrap-through:wrap;`+
		`wrap-before:auto;wrap-after:auto;wrap-inside:auto"></div>`+
		`<div id="d" style="flow-into:first;flow-into:2bad!ident"></div>`+
		`</body></html>`)

	a := boxByID(t, res, "a")
	if a.style.FlowInto != "article" || a.style.FlowIntoElement {
		t.Errorf("flow-into:article content stored %q element=%v, want article/content",
			a.style.FlowInto, a.style.FlowIntoElement)
	}

	if a.style.FlowFrom != "region-a" {
		t.Errorf("flow-from stored %q, want region-a", a.style.FlowFrom)
	}

	if a.style.RegionFragment != regionFragmentBreak {
		t.Errorf("region-fragment stored %q, want break", a.style.RegionFragment)
	}

	if !near(a.style.FlowTolerancePt, 6) || a.style.FlowToleranceInfinite {
		t.Errorf("flow-tolerance:8px stored pt=%v infinite=%v, want 6pt finite",
			a.style.FlowTolerancePt, a.style.FlowToleranceInfinite)
	}

	if a.style.WrapFlow != wrapFlowBoth || a.style.WrapThrough != wrapThroughNone {
		t.Errorf("wrap-flow/wrap-through stored %q/%q, want both/none",
			a.style.WrapFlow, a.style.WrapThrough)
	}

	if a.style.WrapBefore != wrapBreakLine || a.style.WrapAfter != wrapBreakAvoidLine ||
		a.style.WrapInside != wrapBreakAvoid {
		t.Errorf("wrap-before/after/inside stored %q/%q/%q, want line/avoid-line/avoid",
			a.style.WrapBefore, a.style.WrapAfter, a.style.WrapInside)
	}

	b := boxByID(t, res, "b")
	if b.style.FlowInto != "" || b.style.FlowIntoElement {
		t.Errorf("element source built inside its region kept flow-into %q element=%v, want the override cleared",
			b.style.FlowInto, b.style.FlowIntoElement)
	}

	if ident, element, ok := parseFlowInto("story element"); !ok || ident != "story" || !element {
		t.Errorf("parseFlowInto(story element) = %q/%v/%v, want story/true/true", ident, element, ok)
	}

	if _, _, ok := parseFlowInto("2bad!ident"); ok {
		t.Errorf("parseFlowInto accepted an invalid ident")
	}

	c := boxByID(t, res, "c")
	if c.style.FlowInto != "" || c.style.FlowFrom != "" || c.style.RegionFragment != "" ||
		c.style.WrapFlow != "" || c.style.WrapThrough != "" || c.style.WrapBefore != "" ||
		!c.style.FlowToleranceInfinite || c.style.FlowTolerancePt != 0 {
		t.Errorf("none/auto resets stored %+v", c.style)
	}

	if c.style.WrapAfter != "" || c.style.WrapInside != "" {
		t.Errorf("wrap auto stored %q/%q, want empty", c.style.WrapAfter, c.style.WrapInside)
	}

	d := boxByID(t, res, "d")
	if d.style.FlowInto != "first" || d.style.FlowIntoElement {
		t.Errorf("invalid flow-into replaced the valid one: stored %q element=%v",
			d.style.FlowInto, d.style.FlowIntoElement)
	}
}

// TestBehaviorNamedFlowMovesTextIntoSecondBox pins Phase 1: flow-into:
// <ident> content takes the source's children out of normal flow and
// flow-from lays them out inside the region's content box. none/none is a
// no-op and no page slice is allocated.
func TestBehaviorNamedFlowMovesTextIntoSecondBox(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0;font:10pt/12pt monospace">`+
		`<div id="src" style="flow-into:article content">flow source words</div>`+
		`<div id="b" style="flow-from:article;border:1pt solid black;padding:2pt"></div>`+
		`</body></html>`)

	op := textOpContaining(t, res, "flow source words")
	b := boxByID(t, res, "b")

	if op.Y < b.y-0.01 || op.Y > b.y+b.height+0.01 {
		t.Errorf("named-flow text Y = %.2f, want inside region B [%.2f, %.2f]",
			op.Y, b.y, b.y+b.height)
	}

	if n := strings.Count(strings.Join(collectText(res), "\n"), "flow source words"); n != 1 {
		t.Errorf("named-flow text painted %d times, want 1", n)
	}

	src := boxByID(t, res, "src")
	if src.height > 0.5 {
		t.Errorf("source box height = %.2f, want empty (children moved)", src.height)
	}

	if len(res.Pages) != 0 {
		t.Errorf("Result.Pages len = %d, want 0 (one canvas)", len(res.Pages))
	}

	plain := layoutHTML(t, `<html style="margin:0"><body style="margin:0;font:10pt/12pt monospace">`+
		`<div id="src">flow source words</div>`+
		`<div id="b"></div>`+
		`</body></html>`)

	none := layoutHTML(t, `<html style="margin:0"><body style="margin:0;font:10pt/12pt monospace">`+
		`<div id="src" style="flow-into:none">flow source words</div>`+
		`<div id="b" style="flow-from:none"></div>`+
		`</body></html>`)

	behaviorColumnsAssertSameOps(t, plain, none)
}

// collectText concatenates every text op's payload for duplicate detection.
func collectText(res *Result) []string {
	out := make([]string, 0, len(res.Ops))

	for _, op := range res.Ops {
		if op.Kind == OpText {
			out = append(out, op.Text)
		}
	}

	return out
}

// TestBehaviorNamedFlowNoRegionDropsSourceContent pins the no-region case: an
// ident with a source and no region drops that source's children instead of
// inventing a box or a page.
func TestBehaviorNamedFlowNoRegionDropsSourceContent(t *testing.T) {
	t.Parallel()

	res := layoutGoldenFixture(t, "fixture-65-named-flow-no-region.html")

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "unplaced source words") {
			t.Fatalf("no-region source painted %q", op.Text)
		}
	}

	src := boxByID(t, res, "src")
	if src.height > 0.5 {
		t.Errorf("no-region source height = %.2f, want empty", src.height)
	}

	if len(res.Pages) != 0 {
		t.Errorf("Result.Pages len = %d, want 0", len(res.Pages))
	}
}

// TestBehaviorNamedFlowElementMovesWholeSourceBox pins the element keyword:
// the source's own box (border included) paints inside the region instead of
// its old flow position.
func TestBehaviorNamedFlowElementMovesWholeSourceBox(t *testing.T) {
	t.Parallel()

	res := layoutGoldenFixture(t, "fixture-66-named-flow-element-move.html")

	b := boxByID(t, res, "b")
	src := boxByID(t, res, "src")

	borderOp := firstStrokeInRange(t, res, src)

	if borderOp.Y < b.y-0.01 || borderOp.Y > b.y+b.height+0.01 {
		t.Errorf("source border op Y = %.2f, want inside region B [%.2f, %.2f]",
			borderOp.Y, b.y, b.y+b.height)
	}

	if src.y < b.y-0.01 || src.y > b.y+b.height+0.01 {
		t.Errorf("source box y = %.2f, want inside region B [%.2f, %.2f]",
			src.y, b.y, b.y+b.height)
	}
}

// firstStrokeInRange returns the first border op (a stroke rect or line)
// owned by boxNode's op range.
func firstStrokeInRange(t *testing.T, res *Result, boxNode *box) Op {
	t.Helper()

	if boxNode.opEnd < boxNode.opStart {
		t.Fatalf("box has no ops")
	}

	for idx := boxNode.opStart; idx <= boxNode.opEnd && idx < len(res.Ops); idx++ {
		if res.Ops[idx].Kind == OpStrokeRect || res.Ops[idx].Kind == OpLine {
			return res.Ops[idx]
		}
	}

	t.Fatalf("box has no stroke op")

	return Op{}
}

// TestBehaviorNamedFlowContinuesInNextRegion pins Phase 2 chaining: the first
// region fills its content box and the remainder continues in the second
// region, both on the one canvas.
func TestBehaviorNamedFlowContinuesInNextRegion(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0;font:10pt/12pt monospace;width:200pt">`+
		`<div id="src" style="flow-into:article content">`+
		`<p id="p" style="margin:0">alpha bravo charlie delta echo foxtrot golf hotel india `+
		`juliet kilo lima mike november oscar papa quebec romeo sierra tango uniform victor `+
		`whiskey xray yankee zulu</p></div>`+
		`<div id="b" style="flow-from:article;height:30pt"></div>`+
		`<div id="c" style="flow-from:article"></div>`+
		`</body></html>`)

	b := boxByID(t, res, "b")
	c := boxByID(t, res, "c")

	firstInB, laterInC := false, false
	count := 0

	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}

		count++

		if op.Y >= b.y-0.01 && op.Y <= b.y+30+0.01 {
			firstInB = true
		}

		if op.Y >= c.y-0.01 && op.Y <= c.y+c.height+0.01 {
			laterInC = true
		}
	}

	if !firstInB {
		t.Errorf("no line paints inside region B [%.2f, %.2f]", b.y, b.y+30)
	}

	if !laterInC {
		t.Errorf("no line continues inside region C [%.2f, %.2f]", c.y, c.y+c.height)
	}

	if count < 4 {
		t.Errorf("text op count = %d, want the whole paragraph laid out", count)
	}

	if len(res.Pages) != 0 {
		t.Errorf("Result.Pages len = %d, want 0", len(res.Pages))
	}
}

// TestBehaviorNamedFlowRegionFragmentAutoVsBreak pins the region-fragment
// pair: auto slices one block child across B and C, break moves it whole.
func TestBehaviorNamedFlowRegionFragmentAutoVsBreak(t *testing.T) {
	t.Parallel()

	auto := layoutGoldenFixture(t, "fixture-67-named-flow-region-fragment-auto.html")
	broken := layoutGoldenFixture(t, "fixture-68-named-flow-region-fragment-break.html")

	autoB := boxByID(t, auto, "b")
	autoC := boxByID(t, auto, "c")

	autoInB, autoInC := false, false

	for _, op := range auto.Ops {
		if op.Kind != OpText {
			continue
		}

		if op.Y >= autoB.y && op.Y <= autoB.y+autoB.height {
			autoInB = true
		}

		if op.Y >= autoC.y && op.Y <= autoC.y+autoC.height {
			autoInC = true
		}
	}

	if !autoInB || !autoInC {
		t.Errorf("region-fragment auto split in B=%v C=%v, want both", autoInB, autoInC)
	}

	breakB := boxByID(t, broken, "b")
	breakC := boxByID(t, broken, "c")
	breakInB, breakInC := false, false

	for _, op := range broken.Ops {
		if op.Kind != OpText {
			continue
		}

		if op.Y >= breakB.y && op.Y <= breakB.y+breakB.height {
			breakInB = true
		}

		if op.Y >= breakC.y && op.Y <= breakC.y+breakC.height {
			breakInC = true
		}
	}

	if breakInB {
		t.Errorf("region-fragment break left a paragraph line inside region B")
	}

	if !breakInC {
		t.Errorf("region-fragment break moved no line into region C")
	}
}

// TestBehaviorFlowToleranceChangesMasonryColumn pins flow-tolerance in the
// masonry packer: infinite keeps the shortest-column choice, a finite
// tolerance skips a column when the item would stick out past the next one.
func TestBehaviorFlowToleranceChangesMasonryColumn(t *testing.T) {
	t.Parallel()

	doc := func(tolerance string) string {
		return `<html style="margin:0"><body style="margin:0;font:10pt/12pt monospace">` +
			`<div style="display:grid;grid-template-rows:masonry;grid-template-columns:100pt 100pt;` +
			`column-gap:10pt;width:210pt;flow-tolerance:` + tolerance + `">` +
			`<div id="a" style="height:20pt">item a</div>` +
			`<div id="b" style="height:60pt">item b</div>` +
			`</div></body></html>`
	}

	infinite := layoutHTML(t, doc("infinite"))
	zero := layoutHTML(t, doc("0px"))

	infiniteB := textOpContaining(t, infinite, "item b")
	zeroB := textOpContaining(t, zero, "item b")

	if infiniteB.X < 100 {
		t.Errorf("flow-tolerance infinite put item b at X = %.2f, want the second column", infiniteB.X)
	}

	if zeroB.X > 10 {
		t.Errorf("flow-tolerance 0px put item b at X = %.2f, want the first column", zeroB.X)
	}

	if near(infiniteB.X, zeroB.X) {
		t.Errorf("flow-tolerance did not change item b's column: X = %.2f in both", zeroB.X)
	}
}

// TestBehaviorWrapFlowBothShortensLine pins Phase 3: wrap-flow: both makes a
// sized box an exclusion, so the following paragraph's line is pushed and
// shortened while the box itself stays put. The one-interval line model
// takes the side with more room; the two-box case bounds the line on both
// sides.
func TestBehaviorWrapFlowBothShortensLine(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0;font:10pt/12pt monospace;width:200pt">`+
		`<div id="exc" style="wrap-flow:both;width:60pt;height:30pt"></div>`+
		`<p id="p" style="margin:0">alpha bravo charlie delta echo foxtrot golf hotel india juliet</p>`+
		`</body></html>`)

	exc := boxByID(t, res, "exc")
	line := textOpContaining(t, res, "alpha")

	if line.X <= 1 {
		t.Errorf("wrap-flow both left the line at X = %.2f, want pushed past the exclusion", line.X)
	}

	if line.W >= 190 {
		t.Errorf("wrap-flow both line width = %.2f, want shorter than the 200pt content box", line.W)
	}

	if line.X+line.W > 200.01 {
		t.Errorf("wrap-flow both line right edge = %.2f, want inside the content box", line.X+line.W)
	}

	if !near(exc.y, 0) || !near(exc.x, 0) {
		t.Errorf("exclusion box moved to (%.2f, %.2f), want to stay put", exc.x, exc.y)
	}

	two := layoutHTML(t, `<html style="margin:0"><body style="margin:0;font:10pt/12pt monospace;width:200pt">`+
		`<div id="left" style="wrap-flow:both;width:60pt;height:30pt"></div>`+
		`<div id="right" style="wrap-flow:both;margin-left:140pt;width:60pt;height:30pt"></div>`+
		`<p id="p" style="margin:0">alpha bravo charlie delta echo foxtrot golf hotel india juliet</p>`+
		`</body></html>`)

	bounded := textOpContaining(t, two, "alpha")

	if bounded.X <= 1 {
		t.Errorf("two-exclusion line X = %.2f, want pushed off the content start", bounded.X)
	}

	if bounded.X+bounded.W >= 200 {
		t.Errorf("two-exclusion line right edge = %.2f, want before the content end", bounded.X+bounded.W)
	}
}

// TestBehaviorWrapFlowKeywords pins one fixture per wrap-flow keyword:
// start/end shorten one side, minimum/maximum pick the smaller/larger free
// gap, and clear drops the following line below the exclusion.
//
//nolint:funlen // one subtest per keyword fixture
func TestBehaviorWrapFlowKeywords(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		fixture string
		check   func(t *testing.T, exc *box, line Op)
	}{
		{
			fixture: "fixture-69-wrap-flow-start.html",
			check: func(t *testing.T, exc *box, line Op) {
				t.Helper()

				if line.X+line.W > exc.x+0.5 {
					t.Errorf("start: line right edge %.2f, want at the exclusion left edge %.2f",
						line.X+line.W, exc.x)
				}
			},
		},
		{
			fixture: "fixture-70-wrap-flow-end.html",
			check: func(t *testing.T, exc *box, line Op) {
				t.Helper()

				if line.X < exc.x+exc.w-0.5 {
					t.Errorf("end: line X %.2f, want after the exclusion right edge %.2f",
						line.X, exc.x+exc.w)
				}
			},
		},
		{
			fixture: "fixture-71-wrap-flow-minimum.html",
			check: func(t *testing.T, exc *box, line Op) {
				t.Helper()

				if line.X < exc.x+exc.w-0.5 {
					t.Errorf("minimum: line X %.2f, want after the exclusion right edge %.2f",
						line.X, exc.x+exc.w)
				}
			},
		},
		{
			fixture: "fixture-72-wrap-flow-maximum.html",
			check: func(t *testing.T, exc *box, line Op) {
				t.Helper()

				if line.X+line.W > exc.x+0.5 {
					t.Errorf("maximum: line right edge %.2f, want at the exclusion left edge %.2f",
						line.X+line.W, exc.x)
				}
			},
		},
		{
			fixture: "fixture-73-wrap-flow-clear.html",
			check: func(t *testing.T, exc *box, line Op) {
				t.Helper()

				if line.Y < exc.y+exc.height-0.5 {
					t.Errorf("clear: line baseline %.2f, want at or below the exclusion bottom %.2f",
						line.Y, exc.y+exc.height)
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.fixture, func(t *testing.T) {
			t.Parallel()

			res := layoutGoldenFixture(t, testCase.fixture)
			testCase.check(t, boxByID(t, res, "exc"), textOpContaining(t, res, "alpha"))
		})
	}
}

// TestBehaviorWrapThroughNoneIgnoresExclusion pins wrap-through: none: the
// element's lines use the full content width even while an earlier wrap-flow
// exclusion overlaps them, while the control paragraph is shortened.
func TestBehaviorWrapThroughNoneIgnoresExclusion(t *testing.T) {
	t.Parallel()

	doc := func(wrapThrough string) string {
		return `<html style="margin:0"><body style="margin:0;font:10pt/12pt monospace;width:200pt">` +
			`<div id="exc" style="wrap-flow:end;width:60pt;height:30pt"></div>` +
			`<p id="p" style="margin:0;` + wrapThrough + `">alpha bravo charlie delta echo foxtrot golf</p>` +
			`</body></html>`
	}

	control := textOpContaining(t, layoutHTML(t, doc("")), "alpha")
	bypass := textOpContaining(t, layoutHTML(t, doc("wrap-through:none")), "alpha")

	if control.X < 59 {
		t.Errorf("control line X = %.2f, want pushed past the exclusion", control.X)
	}

	if bypass.X > 1 {
		t.Errorf("wrap-through none line X = %.2f, want the full content width", bypass.X)
	}

	if bypass.X >= control.X {
		t.Errorf("wrap-through none did not return the line to the content start")
	}
}

// TestBehaviorFlowToleranceNonMasonryGridIgnored pins that a non-masonry grid
// ignores flow-tolerance: the two fixture grids paint identical item geometry.
func TestBehaviorFlowToleranceNonMasonryGridIgnored(t *testing.T) {
	t.Parallel()

	res := layoutGoldenFixture(t, "fixture-76-flow-tolerance-non-masonry-grid.html")

	oneA := textOpContaining(t, res, "grid one A")
	twoA := textOpContaining(t, res, "grid two A")
	oneB := textOpContaining(t, res, "grid one B")
	twoB := textOpContaining(t, res, "grid two B")

	if !near(oneA.X, twoA.X) || !near(oneB.X, twoB.X) {
		t.Errorf("item X differs: A (%.2f vs %.2f), B (%.2f vs %.2f)",
			oneA.X, twoA.X, oneB.X, twoB.X)
	}

	// The grids stack, so compare the item-to-item delta inside each grid.
	if !near(oneB.Y-oneA.Y, twoB.Y-twoA.Y) {
		t.Errorf("item row delta differs: %.2f vs %.2f", oneB.Y-oneA.Y, twoB.Y-twoA.Y)
	}
}

// TestBehaviorWrapBeforeLineBreaks pins Phase 4 line breaks: wrap-before:
// line starts the element's text on the next line, wrap-after: line moves the
// following text down, and auto does neither.
func TestBehaviorWrapBeforeLineBreaks(t *testing.T) {
	t.Parallel()

	doc := func(span string) string {
		return `<html style="margin:0"><body style="margin:0;font:10pt/12pt monospace;width:200pt">` +
			`<p id="p" style="margin:0">one two <span id="s" style="` + span + `">alpha</span> three</p>` +
			`</body></html>`
	}

	auto := layoutHTML(t, doc(""))
	before := layoutHTML(t, doc("wrap-before:line"))
	after := layoutHTML(t, doc("wrap-after:line"))

	autoOne := textOpContaining(t, auto, "one two ")
	autoAlpha := textOpContaining(t, auto, "alpha")
	autoThree := textOpContaining(t, auto, "three")

	if !near(autoOne.Y, autoAlpha.Y) || !near(autoAlpha.Y, autoThree.Y) {
		t.Errorf("auto lines differ: %.2f/%.2f/%.2f, want one line",
			autoOne.Y, autoAlpha.Y, autoThree.Y)
	}

	beforeAlpha := textOpContaining(t, before, "alpha")
	if beforeAlpha.Y <= autoAlpha.Y {
		t.Errorf("wrap-before line alpha Y = %.2f, want below the auto line %.2f",
			beforeAlpha.Y, autoAlpha.Y)
	}

	if !near(beforeAlpha.Y, autoAlpha.Y+12) {
		t.Errorf("wrap-before line alpha Y = %.2f, want one line (12pt) below %.2f",
			beforeAlpha.Y, autoAlpha.Y)
	}

	afterAlpha := textOpContaining(t, after, "alpha")
	afterThree := textOpContaining(t, after, "three")

	if afterThree.Y <= afterAlpha.Y {
		t.Errorf("wrap-after line three Y = %.2f, want below alpha %.2f",
			afterThree.Y, afterAlpha.Y)
	}
}

// TestBehaviorWrapInsideAvoidFitsAndOverflows pins the wrap-inside pair:
// avoid keeps a fitting phrase on one line and lets an over-wide phrase wrap.
func TestBehaviorWrapInsideAvoidFitsAndOverflows(t *testing.T) {
	t.Parallel()

	fits := layoutGoldenFixture(t, "fixture-74-wrap-inside-avoid-fits.html")
	overflow := layoutGoldenFixture(t, "fixture-75-wrap-inside-avoid-overflows.html")

	if lines := distinctTextYs(fits, "short", "phrase"); lines != 1 {
		t.Errorf("wrap-inside avoid fitting phrase spans %d lines, want 1", lines)
	}

	if lines := distinctTextYs(overflow, "this", "deliberately", "anyway"); lines < 2 {
		t.Errorf("wrap-inside avoid over-wide phrase spans %d lines, want more than 1", lines)
	}
}

// distinctTextYs counts the distinct baselines of text ops whose payload
// contains any needle (every text op when no needle is given).
func distinctTextYs(res *Result, needles ...string) int {
	seen := map[float64]bool{}

	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}

		if len(needles) == 0 {
			seen[math.Round(op.Y*100)/100] = true

			continue
		}

		for _, needle := range needles {
			if strings.Contains(op.Text, needle) {
				seen[math.Round(op.Y*100)/100] = true

				break
			}
		}
	}

	return len(seen)
}

// TestBehaviorWrapAvoidSkipsSoftWrap pins avoid/avoid-line: a short boundary
// word is glued across a soft wrap within one em of slack, so the paragraph
// stays one line; long content still wraps and never overflows without bound.
func TestBehaviorWrapAvoidSkipsSoftWrap(t *testing.T) {
	t.Parallel()

	doc := func(span string) string {
		return `<html style="margin:0"><body style="margin:0;font:10pt/12pt monospace;width:200pt">` +
			`<p id="p" style="margin:0">aaaa aaaa aaaa aaaa aaaa aaaa aa ` +
			`<span id="s" style="` + span + `">x</span></p>` +
			`</body></html>`
	}

	plain := layoutHTML(t, doc(""))
	avoid := layoutHTML(t, doc("wrap-before:avoid"))

	plainFirst := textOpContaining(t, plain, "aaaa")
	plainX := textOpContaining(t, plain, "x")

	if plainX.Y <= plainFirst.Y {
		t.Fatalf("control kept the boundary word on line 1 (%.2f), want it pushed to line 2", plainX.Y)
	}

	avoidFirst := textOpContaining(t, avoid, "aaaa")
	avoidX := textOpContaining(t, avoid, "x")

	if !near(avoidX.Y, avoidFirst.Y) {
		t.Errorf("wrap-before avoid boundary Y = %.2f, want glued on line 1 (%.2f)",
			avoidX.Y, avoidFirst.Y)
	}

	long := layoutHTML(t, `<html style="margin:0"><body style="margin:0;font:10pt/12pt monospace;width:200pt">`+
		`<p id="p" style="margin:0">aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa aaaa `+
		`<span style="wrap-before:avoid">longboundaryword that cannot fit</span></p>`+
		`</body></html>`)

	longFirst := textOpContaining(t, long, "aaaa")
	longBoundary := textOpContaining(t, long, "longboundaryword")

	if longBoundary.Y <= longFirst.Y {
		t.Errorf("over-wide boundary Y = %.2f, want it wrapped to a later line (%.2f)",
			longBoundary.Y, longFirst.Y)
	}

	if lines := distinctTextYs(avoid); lines != 1 {
		t.Errorf("fitting avoid fixture line count = %d, want 1", lines)
	}

	if lines := distinctTextYs(long); lines < 2 {
		t.Errorf("overflow avoid fixture line count = %d, want more than 1", lines)
	}
}

// TestBehaviorWrapBeforeFlexBreaksLine pins wrap-before: flex: the item
// starts a new flex line even when it would fit, while a nowrap container
// ignores the value.
func TestBehaviorWrapBeforeFlexBreaksLine(t *testing.T) {
	t.Parallel()

	doc := func(wrap, itemB string) string {
		return `<html style="margin:0"><body style="margin:0;font:10pt/12pt monospace;width:200pt">` +
			`<div style="display:flex;flex-wrap:` + wrap + `;width:200pt">` +
			`<div id="a" style="width:60pt">item a</div>` +
			`<div id="b" style="width:60pt;` + itemB + `">item b</div>` +
			`<div id="c" style="width:60pt">item c</div>` +
			`</div></body></html>`
	}

	plain := layoutHTML(t, doc("wrap", ""))
	forced := layoutHTML(t, doc("wrap", "wrap-before:flex"))
	nowrap := layoutHTML(t, doc("nowrap", "wrap-before:flex"))

	plainA := textOpContaining(t, plain, "item a")
	plainB := textOpContaining(t, plain, "item b")

	if !near(plainA.Y, plainB.Y) {
		t.Fatalf("control flex line differs: a %.2f b %.2f", plainA.Y, plainB.Y)
	}

	forcedA := textOpContaining(t, forced, "item a")
	forcedB := textOpContaining(t, forced, "item b")
	forcedC := textOpContaining(t, forced, "item c")

	if forcedB.Y <= forcedA.Y {
		t.Errorf("wrap-before flex b Y = %.2f, want a new flex line below a %.2f",
			forcedB.Y, forcedA.Y)
	}

	if !near(forcedB.Y, forcedC.Y) {
		t.Errorf("wrap-before flex moved c too: b %.2f c %.2f, want c on b's line",
			forcedB.Y, forcedC.Y)
	}

	nowrapA := textOpContaining(t, nowrap, "item a")
	nowrapB := textOpContaining(t, nowrap, "item b")

	if !near(nowrapA.Y, nowrapB.Y) {
		t.Errorf("nowrap flex b Y = %.2f, want the value ignored (a %.2f)",
			nowrapB.Y, nowrapA.Y)
	}
}

// TestBehaviorWrapBreakNotPageBreak pins the Phase 4 gate: wrap-before: line
// is a line break, not a page break. The box stays above Options.Height when
// its content is short, and its Y grows by about one line, not by a page.
func TestBehaviorWrapBreakNotPageBreak(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html style="margin:0"><body style="margin:0;font:10pt/12pt monospace;width:200pt">`+
		`<p id="p" style="margin:0">short lead `+
		`<span id="s" style="wrap-before:line">next line words</span></p>`+
		`</body></html>`)

	lead := textOpContaining(t, res, "short lead")
	next := textOpContaining(t, res, "next line words")

	if next.Y >= 800 {
		t.Errorf("wrap-before line Y = %.2f, want below the page boundary (no page jump)", next.Y)
	}

	if delta := next.Y - lead.Y; delta < 11 || delta > 13 {
		t.Errorf("wrap-before line Y delta = %.2f, want about one 12pt line", delta)
	}

	if len(res.Pages) != 0 {
		t.Errorf("Result.Pages len = %d, want 0 (wrap breaks do not assign pages)", len(res.Pages))
	}
}
