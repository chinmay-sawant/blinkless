package layout

import (
	"strconv"
	"testing"
)

// Footnote behavior tests for footnote-display and footnote-policy (CSS
// Generated Content for Paged Media 3, section 2,
// https://drafts.csswg.org/css-gcpm-3/#footnotes). Chrome implements no
// footnote area, so the spec is the reference for every case below, not a
// Chrome version.
//
// What these tests prove, all through used values: footnote elements created
// by float: footnote are collected in document order, numbered through the
// footnote counter (honoring counter-reset and counter-increment), carry
// parsed footnote-display and footnote-policy used values, keep their in-flow
// placed geometry, and group by policy for the document-end fallback.
// Placement fallback, stated explicitly: footnote content stays in normal
// flow; FootnoteFallbackArea reports the strip below the last laid-out box,
// and a per-page footnote area is the follow-up seam in footnote.go.

// TestBehaviorFootnoteCollectsInDocumentOrder numbers three footnote elements
// in document order with used display values: initial block for the two plain
// ones, inline from the author sheet for the classed one, compact from the
// inline style for the last one. Marker text follows the GCPM 3 default
// footnote-call content, counter(footnote).
func TestBehaviorFootnoteCollectsInDocumentOrder(t *testing.T) {
	t.Parallel()

	src := `<html><body>` +
		`<p>one<span id="f1" style="float:footnote">first note</span></p>` +
		`<p>two<span id="f2" class="fn">second note</span></p>` +
		`<p>three<span id="f3" style="float:footnote;footnote-display:compact">third note</span></p>` +
		`</body></html>`

	root := mustParse(t, src)
	items := CollectFootnotes(root, sheet(t, `.fn { float: footnote; footnote-display: inline }`))

	if len(items) != 3 {
		t.Fatalf("collected %d footnotes, want 3", len(items))
	}

	wantNumbers := []int{1, 2, 3}
	wantMarkers := []string{"1", "2", "3"}
	wantTexts := []string{"first note", "second note", "third note"}
	wantDisplays := []string{"block", "inline", "compact"}

	for idx, item := range items {
		if item.Number != wantNumbers[idx] {
			t.Errorf("footnote %d number = %d, want %d", idx, item.Number, wantNumbers[idx])
		}

		if item.Marker != wantMarkers[idx] {
			t.Errorf("footnote %d marker = %q, want %q", idx, item.Marker, wantMarkers[idx])
		}

		if item.Text != wantTexts[idx] {
			t.Errorf("footnote %d text = %q, want %q", idx, item.Text, wantTexts[idx])
		}

		if item.Display != wantDisplays[idx] {
			t.Errorf("footnote %d display = %q, want %q", idx, item.Display, wantDisplays[idx])
		}

		if item.Policy != "auto" {
			t.Errorf("footnote %d policy = %q, want initial auto", idx, item.Policy)
		}
	}
}

// TestBehaviorFootnoteInvalidValuesFallBackToInitial pins the used values for
// unknown footnote-display and footnote-policy keywords: block and auto, the
// GCPM 3 initial values. It also pins case-insensitive float: footnote
// detection, matching the cascade's case folding.
func TestBehaviorFootnoteInvalidValuesFallBackToInitial(t *testing.T) {
	t.Parallel()

	src := `<html><body>` +
		`<p>a<span style="float:footnote;footnote-display:whatever;footnote-policy:sometimes">odd</span></p>` +
		`<p>b<span style="FLOAT:FOOTNOTE">loud</span></p>` +
		`</body></html>`

	items := CollectFootnotes(mustParse(t, src))

	if len(items) != 2 {
		t.Fatalf("collected %d footnotes, want 2", len(items))
	}

	if items[0].Display != "block" || items[0].Policy != "auto" {
		t.Errorf("invalid keywords give display %q policy %q, want block/auto",
			items[0].Display, items[0].Policy)
	}

	if items[1].Number != 2 || items[1].Text != "loud" {
		t.Errorf("uppercase float:footnote gives number %d text %q, want 2/loud",
			items[1].Number, items[1].Text)
	}
}

// TestBehaviorFootnoteCounterResetAndIncrement wires marker numbering to the
// footnote counter: counter-reset on the wrapper restarts numbering inside
// that subtree only, and a per-element counter-increment step moves the
// counter by that step. The footnote after the wrapper numbers from the
// untouched outer counter, proving reset scope does not leak.
func TestBehaviorFootnoteCounterResetAndIncrement(t *testing.T) {
	t.Parallel()

	src := `<html><body>` +
		`<div style="counter-reset:footnote 4">` +
		`<span style="float:footnote">a</span>` +
		`<span style="float:footnote;counter-increment:footnote 2">b</span>` +
		`<span style="float:footnote">c</span></div>` +
		`<span style="float:footnote">d</span>` +
		`</body></html>`

	items := CollectFootnotes(mustParse(t, src))

	want := []int{5, 7, 8, 1}

	if len(items) != len(want) {
		t.Fatalf("collected %d footnotes, want %d", len(items), len(want))
	}

	for idx, item := range items {
		if item.Number != want[idx] {
			t.Errorf("footnote %d number = %d, want %d", idx, item.Number, want[idx])
		}

		if item.Marker != strconv.Itoa(want[idx]) {
			t.Errorf("footnote %d marker = %q, want %q", idx, item.Marker, strconv.Itoa(want[idx]))
		}
	}
}

// TestBehaviorFootnotePolicyGrouping pins the feasible footnote-policy
// subset: block-policy footnotes merge into one fallback group, while line
// and auto policies keep one group per footnote.
func TestBehaviorFootnotePolicyGrouping(t *testing.T) {
	t.Parallel()

	src := `<html><body>` +
		`<span style="float:footnote">a</span>` +
		`<span style="float:footnote;footnote-policy:block">b</span>` +
		`<span style="float:footnote;footnote-policy:block">c</span>` +
		`<span style="float:footnote;footnote-policy:line">d</span>` +
		`</body></html>`

	groups := GroupFootnotesByPolicy(CollectFootnotes(mustParse(t, src)))

	wantSizes := []int{1, 2, 1}

	if len(groups) != len(wantSizes) {
		t.Fatalf("grouped into %d groups, want %d", len(groups), len(wantSizes))
	}

	for idx, group := range groups {
		if len(group) != wantSizes[idx] {
			t.Errorf("group %d holds %d footnotes, want %d", idx, len(group), wantSizes[idx])
		}
	}

	if len(GroupFootnotesByPolicy(nil)) != 0 {
		t.Errorf("empty footnote list groups into %d, want 0", len(GroupFootnotesByPolicy(nil)))
	}
}

// TestBehaviorFootnoteFallbackGeometryKeepsContentInFlow lays out two
// block-level footnote elements and asserts their used placed geometry: both
// keep non-empty in-flow boxes in document order, and the fallback area
// starts at or below the lowest footnote bottom while spanning the full
// content width with no reserved height. Block-level elements are used
// because inline spans produce no dedicated box in the box tree, only line
// runs; geometry is reported where a box exists.
func TestBehaviorFootnoteFallbackGeometryKeepsContentInFlow(t *testing.T) {
	t.Parallel()

	src := `<html><body style="margin:0">` +
		`<p>text</p><div id="f1" style="float:footnote">note one</div>` +
		`<p>more</p><div id="f2" style="float:footnote;footnote-display:inline">note two</div>` +
		`</body></html>`

	root := mustParse(t, src)

	res, err := Layout(root, Options{
		Width: testViewport, Height: 800, Background: true,
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	items := CollectFootnotes(root)
	if len(items) != 2 {
		t.Fatalf("collected %d footnotes, want 2", len(items))
	}

	first, okFirst := FootnoteGeometry(res, items[0])
	second, okSecond := FootnoteGeometry(res, items[1])

	if !okFirst || !okSecond {
		t.Fatalf("footnote geometry found = %v/%v, want true/true", okFirst, okSecond)
	}

	assertFootnoteBoxNonEmpty(t, "first", first)
	assertFootnoteBoxNonEmpty(t, "second", second)

	if second.Y < first.Y {
		t.Errorf("second footnote y = %.2fpt, first = %.2fpt, want document order",
			second.Y, first.Y)
	}

	assertFallbackArea(t, res, FootnoteFallbackArea(res), second.Y+second.H)
}

// assertFootnoteBoxNonEmpty pins that a collected footnote kept a non-empty
// in-flow box.
func assertFootnoteBoxNonEmpty(t *testing.T, name string, geom FootnoteArea) {
	t.Helper()

	if geom.W <= 0 || geom.H <= 0 {
		t.Errorf("%s footnote box = %.2fpt x %.2fpt, want non-empty in-flow box",
			name, geom.W, geom.H)
	}
}

// assertFallbackArea pins the document-end fallback strip: full content width,
// starting at or below the lowest footnote bottom, with no reserved height.
func assertFallbackArea(t *testing.T, res *Result, area FootnoteArea, lowestBottom float64) {
	t.Helper()

	if !near(area.W, res.Width) {
		t.Errorf("fallback area width = %.2fpt, want content width %.2fpt", area.W, res.Width)
	}

	if area.Y < lowestBottom {
		t.Errorf("fallback area y = %.2fpt, want at or below lowest footnote bottom %.2fpt",
			area.Y, lowestBottom)
	}

	if area.H != 0 {
		t.Errorf("fallback area height = %.2fpt, want 0 (nothing moved yet)", area.H)
	}
}

// TestBehaviorFootnoteDisplayNoneSkipped pins that a float: footnote element
// under display: none contributes no footnote and no counter step, since the
// engine builds no box for it.
func TestBehaviorFootnoteDisplayNoneSkipped(t *testing.T) {
	t.Parallel()

	src := `<html><body><p>` +
		`<span style="float:footnote">kept</span>` +
		`<span style="display:none;float:footnote">hidden</span>` +
		`<span style="float:footnote">also kept</span>` +
		`</p></body></html>`

	items := CollectFootnotes(mustParse(t, src))

	if len(items) != 2 {
		t.Fatalf("collected %d footnotes, want 2", len(items))
	}

	if items[0].Number != 1 || items[0].Text != "kept" {
		t.Errorf("first footnote = %d/%q, want 1/kept", items[0].Number, items[0].Text)
	}

	if items[1].Number != 2 || items[1].Text != "also kept" {
		t.Errorf("second footnote = %d/%q, want 2/also kept", items[1].Number, items[1].Text)
	}
}
