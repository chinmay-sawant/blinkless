package layout

import (
	"errors"
	"math"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"weak"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	ilayout "github.com/chinmay-sawant/blinkless/internal/layout"
)

//nolint:paralleltest // exercises the shared fingerprint cache across garbage collection
func TestFingerprintCacheDoesNotRetainReleasedDocument(t *testing.T) {
	docProbe, resultProbe := func() (weak.Pointer[css.Document], weak.Pointer[ilayout.Result]) {
		tree := fastTree(t, `<div id="box" style="width:120px;height:40px">Hello</div>`)
		styled := fastStyled(t, tree)
		display := fastRetained(t, styled)
		resultProbe := weak.Make(display.result)
		patch, err := RelayoutElement(t.Context(), styled, display, "box", Options{})
		if err != nil {
			t.Fatal(err)
		}
		docProbe := weak.Make(styled)
		display.Release()
		patch.Display.Release()
		return docProbe, resultProbe
	}()
	for range 3 {
		runtime.GC()
	}
	if docProbe.Value() != nil {
		t.Fatal("fingerprint memo retained a document after its display was released")
	}
	if resultProbe.Value() != nil {
		t.Fatal("fingerprint memo retained a layout result after its display was released")
	}
}

//nolint:paralleltest // checks measurement and patch caches across garbage collection
func TestReleasedLayoutResultIsCollectible(t *testing.T) {
	probe := func() weak.Pointer[ilayout.Result] {
		tree := fastTree(t, `<div id="box" style="display:flex;width:120px;height:40px">Hello</div>`)
		styled := fastStyled(t, tree)
		display := fastRetained(t, styled)
		probe := weak.Make(display.result)
		display.Release()
		return probe
	}()
	for range 3 {
		runtime.GC()
	}
	if probe.Value() != nil {
		t.Fatal("a cache retained a released layout result")
	}
}

// fastTree parses source into a document the tests can edit in place.
func fastTree(t *testing.T, source string) *html.Document {
	t.Helper()

	tree, err := html.Parse([]byte(source))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	return tree
}

// fastStyled applies tree at a fixed viewport with no state.
func fastStyled(t *testing.T, tree *html.Document) *css.Document {
	t.Helper()

	styled, err := css.Apply(t.Context(), tree, css.Options{WidthPx: 400, HeightPx: 300, Media: "screen"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	return styled
}

// fastRetained lays styled out with the retained state the patch path needs.
func fastRetained(t *testing.T, styled *css.Document) *Display {
	t.Helper()

	display, err := DisplayListOptions(t.Context(), styled, Options{RetainState: true})
	if err != nil {
		t.Fatalf("display: %v", err)
	}

	return display
}

// fastPatchText returns the text of the patch's first text operation.
func fastPatchText(patch *ElementPatch) string {
	for i := range patch.Ops {
		if patch.Ops[i].Kind == DisplayOpText {
			return patch.Ops[i].Text
		}
	}

	return ""
}

// fastChildSize returns the point size of the first text run containing want.
func fastChildSize(t *testing.T, ops []DisplayOp, want string) float64 {
	t.Helper()

	for i := range ops {
		if ops[i].Kind == DisplayOpText && strings.Contains(ops[i].Text, want) {
			return ops[i].Size
		}
	}

	t.Fatalf("no text run contains %q", want)

	return 0
}

// TestFastNoopIdenticalReturnsPrev proves the identical-value fast path: a
// second patch with no edit in between hands the same display back, the
// engine counter stays flat, and the operations and boxes are untouched.
func TestFastNoopIdenticalReturnsPrev(t *testing.T) { //nolint:paralleltest // reads process-wide engine counters
	tree := fastTree(t, `<div id="box" style="width:120px;height:40px">Hello</div>`)
	styled := fastStyled(t, tree)
	display := fastRetained(t, styled)

	warm, err := RelayoutElement(t.Context(), styled, display, "box", Options{})
	if err != nil {
		t.Fatalf("warm relayout: %v", err)
	}

	beforeEngine := atomic.LoadInt64(&relayoutEngineCalls)
	beforeHits := atomic.LoadInt64(&relayoutFastHits)
	opsLen := len(warm.Display.Ops)
	boxesLen := len(warm.Display.Boxes)

	second, err := RelayoutElement(t.Context(), styled, warm.Display, "box", Options{})
	if err != nil {
		t.Fatalf("no-op relayout: %v", err)
	}

	if second.Display != warm.Display {
		t.Fatal("no-op patch did not hand the same display back")
	}

	if got := atomic.LoadInt64(&relayoutEngineCalls); got != beforeEngine {
		t.Fatalf("engine calls = %d, want %d (no engine work)", got, beforeEngine)
	}

	if got := atomic.LoadInt64(&relayoutFastHits); got != beforeHits+1 {
		t.Fatalf("fast hits = %d, want %d", got, beforeHits+1)
	}

	if len(second.Display.Ops) != opsLen || len(second.Display.Boxes) != boxesLen {
		t.Fatal("no-op patch changed the operations or the boxes")
	}
}

// TestFastIgnoredAttributeNoop proves the ignored-attribute fast path: editing
// title, which no selector names, keeps the fingerprint, so the patch hands
// the same display back with no engine work.
func TestFastIgnoredAttributeNoop(t *testing.T) { //nolint:paralleltest // reads process-wide engine counters
	tree := fastTree(t, `<div id="box" style="width:120px;height:40px">Hello</div>`)
	styled := fastStyled(t, tree)
	display := fastRetained(t, styled)

	warm, err := RelayoutElement(t.Context(), styled, display, "box", Options{})
	if err != nil {
		t.Fatalf("warm relayout: %v", err)
	}

	if !tree.SetAttribute("box", "title", "tooltip") {
		t.Fatal("SetAttribute reported no element")
	}

	beforeEngine := atomic.LoadInt64(&relayoutEngineCalls)
	beforeHits := atomic.LoadInt64(&relayoutFastHits)

	second, err := RelayoutElement(t.Context(), styled, warm.Display, "box", Options{})
	if err != nil {
		t.Fatalf("ignored-attribute relayout: %v", err)
	}

	if second.Display != warm.Display {
		t.Fatal("ignored-attribute patch did not hand the same display back")
	}

	if got := atomic.LoadInt64(&relayoutEngineCalls); got != beforeEngine {
		t.Fatalf("engine calls = %d, want %d (no engine work)", got, beforeEngine)
	}

	if got := atomic.LoadInt64(&relayoutFastHits); got != beforeHits+1 {
		t.Fatalf("fast hits = %d, want %d", got, beforeHits+1)
	}
}

// TestFastFontSkipTextFree proves the text-free phase skip: a patch of an
// element without text skips the default font load, keeps its bounds, and
// still carries paint.
func TestFastFontSkipTextFree(t *testing.T) { //nolint:paralleltest // reads process-wide engine counters
	tree := fastTree(t, `<div id="box" style="width:120px;height:40px;background-color:#ff0000"></div>`)
	styled := fastStyled(t, tree)
	display := fastRetained(t, styled)

	beforeSkips := atomic.LoadInt64(&relayoutFontSkips)

	patch, err := RelayoutElement(t.Context(), styled, display, "box", Options{})
	if err != nil {
		t.Fatalf("relayout element: %v", err)
	}

	if got := atomic.LoadInt64(&relayoutFontSkips); got != beforeSkips+1 {
		t.Fatalf("font skips = %d, want %d", got, beforeSkips+1)
	}

	if patch.Display == display {
		t.Fatal("real patch handed the same display back")
	}

	if patch.Old.Empty() || patch.Old != patch.New {
		t.Fatalf("bounds old=%v new=%v, want equal and non-empty", patch.Old, patch.New)
	}

	if len(patch.Ops) == 0 {
		t.Fatal("patch carries no operations")
	}
}

// TestFastNoopRealChangeStillRelayouts is the negative of the no-op path: new
// text misses the fingerprint, runs the engine, and carries the new text.
func TestFastNoopRealChangeStillRelayouts(t *testing.T) {
	t.Parallel()

	tree := fastTree(t, `<div id="box" style="width:120px;height:40px">Hello</div>`)
	styled := fastStyled(t, tree)
	display := fastRetained(t, styled)

	if !tree.SetText("box", "Hello world") {
		t.Fatal("SetText reported no element")
	}

	patch, err := RelayoutElement(t.Context(), styled, display, "box", Options{})
	if err != nil {
		t.Fatalf("relayout element: %v", err)
	}

	if patch.Display == display {
		t.Fatal("real change returned the same display")
	}

	if got := fastPatchText(patch); got != "Hello world" {
		t.Fatalf("patch text = %q, want %q", got, "Hello world")
	}
}

// TestFastIgnoredAttributeNegative proves a relevant attribute still
// relayouts: data-action reaches the boxes, so the patch carries the new
// action in a new display.
func TestFastIgnoredAttributeNegative(t *testing.T) {
	t.Parallel()

	tree := fastTree(t, `<div id="box" style="width:120px;height:40px">Hello</div>`)
	styled := fastStyled(t, tree)
	display := fastRetained(t, styled)

	if !tree.SetAttribute("box", "data-action", "save") {
		t.Fatal("SetAttribute reported no element")
	}

	patch, err := RelayoutElement(t.Context(), styled, display, "box", Options{})
	if err != nil {
		t.Fatalf("relayout element: %v", err)
	}

	if patch.Display == display {
		t.Fatal("relevant attribute change returned the same display")
	}

	if len(patch.Boxes) == 0 || patch.Boxes[0].ID != "box" || patch.Boxes[0].Action != "save" {
		t.Fatalf("patched boxes = %+v, want the box with action save", patch.Boxes)
	}
}

// TestFastFontSkipTextChange is the negative of the font skip: an element with
// text still shapes it after an edit.
func TestFastFontSkipTextChange(t *testing.T) {
	t.Parallel()

	tree := fastTree(t, `<div id="box" style="width:120px;height:40px">Hello</div>`)
	styled := fastStyled(t, tree)
	display := fastRetained(t, styled)

	if !tree.SetText("box", "Hello world") {
		t.Fatal("SetText reported no element")
	}

	patch, err := RelayoutElement(t.Context(), styled, display, "box", Options{})
	if err != nil {
		t.Fatalf("relayout element: %v", err)
	}

	if got := fastPatchText(patch); got != "Hello world" {
		t.Fatalf("patch text = %q, want %q", got, "Hello world")
	}
}

// TestFastInertOverrideObjectPosition proves the inert-override fast path
// instead of erroring: object-position cannot paint a plain div, so the patch
// hands the same display back even though the subtree holds a float the
// engine would refuse.
func TestFastInertOverrideObjectPosition(t *testing.T) {
	t.Parallel()

	tree := fastTree(t, `<div id="wrap" style="width:200px;height:100px">`+
		`<div id="box" style="float:left;width:50px;height:20px">F</div></div>`)
	styled := fastStyled(t, tree)
	display := fastRetained(t, styled)

	patch, err := RelayoutElementWithOverrides(t.Context(), styled, display, "box",
		[]Override{{Property: "object-position", Value: "center"}}, Options{})
	if err != nil {
		t.Fatalf("inert override: %v", err)
	}

	if patch.Display != display {
		t.Fatal("inert override did not hand the same display back")
	}
}

// TestFastInertOverrideRefusedStillErrors proves the refusal stays: a width
// override on the same float subtree reaches the engine and is refused, and
// the display is left untouched.
func TestFastInertOverrideRefusedStillErrors(t *testing.T) {
	t.Parallel()

	tree := fastTree(t, `<div id="wrap" style="width:200px;height:100px">`+
		`<div id="box" style="float:left;width:50px;height:20px">F</div></div>`)
	styled := fastStyled(t, tree)
	display := fastRetained(t, styled)
	beforeOps := slices.Clone(display.Ops)
	beforeBoxes := slices.Clone(display.Boxes)

	_, err := RelayoutElementWithOverrides(t.Context(), styled, display, "box",
		[]Override{{Property: "width", Value: "180px"}}, Options{})
	if err == nil {
		t.Fatal("refused override succeeded, want an error")
	}

	if !reflect.DeepEqual(display.Ops, beforeOps) {
		t.Fatal("a refused call changed the display operations")
	}

	if !reflect.DeepEqual(display.Boxes, beforeBoxes) {
		t.Fatal("a refused call changed the display boxes")
	}
}

// TestFastInertOverrideLetterSpacingNegative proves spacing overrides still
// relayout an element with text: the patch lands in a new display.
func TestFastInertOverrideLetterSpacingNegative(t *testing.T) {
	t.Parallel()

	tree := fastTree(t, `<div id="box" style="width:200px;height:40px;font-size:12px">Hi</div>`)
	styled := fastStyled(t, tree)
	display := fastRetained(t, styled)

	patch, err := RelayoutElementWithOverrides(t.Context(), styled, display, "box",
		[]Override{{Property: "letter-spacing", Value: "5px"}}, Options{})
	if err != nil {
		t.Fatalf("spacing override: %v", err)
	}

	if patch.Display == display {
		t.Fatal("spacing change on text returned the same display")
	}
}

// TestFastInertOverrideFontSize proves the no-text no-relative fast path: a
// font-size override on a fixed empty box changes nothing, so the patch hands
// the same display back.
func TestFastInertOverrideFontSize(t *testing.T) {
	t.Parallel()

	tree := fastTree(t, `<div id="box" style="width:100px;height:40px"></div>`)
	styled := fastStyled(t, tree)
	display := fastRetained(t, styled)

	patch, err := RelayoutElementWithOverrides(t.Context(), styled, display, "box",
		[]Override{{Property: "font-size", Value: "24px"}}, Options{})
	if err != nil {
		t.Fatalf("inert font-size override: %v", err)
	}

	if patch.Display != display {
		t.Fatal("inert font-size override did not hand the same display back")
	}
}

// TestFastInertOverrideFontSizeNegative proves the same override still
// relayouts an element with text: the child run grows in a new display.
func TestFastInertOverrideFontSizeNegative(t *testing.T) {
	t.Parallel()

	tree := fastTree(t,
		`<div id="card" style="width:200px;height:40px;font-size:12px">Base <span id="child">Child</span></div>`)
	styled := fastStyled(t, tree)
	display := fastRetained(t, styled)

	patch, err := RelayoutElementWithOverrides(t.Context(), styled, display, "card",
		[]Override{{Property: "font-size", Value: "24px"}}, Options{})
	if err != nil {
		t.Fatalf("font-size override: %v", err)
	}

	if patch.Display == display {
		t.Fatal("font-size change on text returned the same display")
	}

	if got := fastChildSize(t, patch.Ops, "Child"); math.Abs(got-18) > 0.1 {
		t.Fatalf("child text size = %gpt, want 18pt", got)
	}
}

// TestFastRefusalPreserved pins the size-change refusal: a width override that
// escapes the containing block is refused and the display is untouched.
func TestFastRefusalPreserved(t *testing.T) {
	t.Parallel()

	tree := fastTree(t, `<div id="card" style="width:200px;font-size:12px">`+
		`one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen</div>`)
	styled := fastStyled(t, tree)
	display := fastRetained(t, styled)
	beforeOps := slices.Clone(display.Ops)
	beforeBoxes := slices.Clone(display.Boxes)

	_, err := RelayoutElementWithOverrides(t.Context(), styled, display, "card",
		[]Override{{Property: "width", Value: "40px"}}, Options{})
	if !errors.Is(err, ErrOverrideReflow) {
		t.Fatalf("error = %v, want %v", err, ErrOverrideReflow)
	}

	if !reflect.DeepEqual(display.Ops, beforeOps) {
		t.Fatal("a refused call changed the display operations")
	}

	if !reflect.DeepEqual(display.Boxes, beforeBoxes) {
		t.Fatal("a refused call changed the display boxes")
	}
}
