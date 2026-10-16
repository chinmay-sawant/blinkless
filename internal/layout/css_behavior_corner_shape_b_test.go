package layout

import (
	"math"
	"testing"
)

// Part B corner-shape behavior tests (Phase 3, second half): the pair
// longhands, the logical map, the radius-plus-shape shorthands, and the
// invalid-value rule.
//
// Every test checks the drawing list, never a page PNG, following
// css_behavior_borders_test.go patterns: one 120px by 60px red box, lengths in
// CSS px converted with pxToPt, helpers from layout_test.go and
// css_review_02_test.go (layoutHTML, boxByID, near, pxToPt, ptPerCSSPx).
//
// Reference is the CSS Borders 4 draft corner-shape section (fetched
// 2026-10-15, https://drafts.csswg.org/css-borders-4/#typedef-corner-shape-value):
// round K=1, squircle K=2, square K=infinity, bevel K=0, scoop K=-1,
// notch K=-infinity. There is no Chrome parity claim for these names; the
// catalog records no browser result.
//
// The OpPath kind (corner_path.go), the per-corner K store on ResolvedStyle
// (style_corner_shape.go), and the painter (corner_shape_paint.go) are all
// landed, so the assertions below run for real.

// cornerShapeBItemID is the id of the single test box in the B documents.
const cornerShapeBItemID = "item"

// cornerShapeBCornerNames names the four corners in style K order:
// TL, TR, BR, BL.
var cornerShapeBCornerNames = [4]string{ //nolint:gochecknoglobals // test table
	"top-left", "top-right", "bottom-right", "bottom-left",
}

// cornerShapeBDoc lays out one 120px by 60px red box with a 16px corner box
// plus the given extra declaration (a corner-shape decl or empty). Shape-only
// longhands need the radius baseline so the painter has a corner box.
func cornerShapeBDoc(extraDecl string) string {
	return `<html style="margin:0"><body style="margin:0">` +
		`<div id="` + cornerShapeBItemID + `" style="width:120px;height:60px;background-color:#f00;border-radius:16px;` +
		extraDecl + `"></div>` +
		`</body></html>`
}

// cornerShapeBDocBare is cornerShapeBDoc without the radius baseline, for the
// shorthands that set a radius themselves (corner, corner-top, ...). The
// declaration under test must be the only radius source.
func cornerShapeBDocBare(extraDecl string) string {
	return `<html style="margin:0"><body style="margin:0">` +
		`<div id="` + cornerShapeBItemID + `" style="width:120px;height:60px;background-color:#f00;` +
		extraDecl + `"></div>` +
		`</body></html>`
}

// cornerShapeBLayoutDoc runs one document through the engine.
func cornerShapeBLayoutDoc(t *testing.T, doc string) *Result {
	t.Helper()

	return layoutHTML(t, doc)
}

// cornerShapeBBoxOps returns the ops owned by the test box, using its
// recorded op range. Same rule as behaviorBordersBoxOps in
// css_behavior_borders_test.go, restated here so this file stands alone.
func cornerShapeBBoxOps(t *testing.T, res *Result) []Op {
	t.Helper()

	elementBox := boxByID(t, res, cornerShapeBItemID)
	if elementBox.opEnd < elementBox.opStart {
		return nil
	}

	start, end := elementBox.opStart, elementBox.opEnd
	if end >= len(res.Ops) {
		end = len(res.Ops) - 1
	}

	return res.Ops[start : end+1]
}

// cornerShapeBPathOps selects OpPath ops from ops.
func cornerShapeBPathOps(ops []Op) []Op {
	var out []Op

	for _, paintOp := range ops {
		if paintOp.Kind == OpPath {
			out = append(out, paintOp)
		}
	}

	return out
}

// cornerShapeBBoxStyle returns the resolved style of the box under test.
func cornerShapeBBoxStyle(t *testing.T, res *Result) *ResolvedStyle {
	t.Helper()

	return boxByID(t, res, cornerShapeBItemID).style
}

// cornerShapeBStyleKs reads the four corner K values in TL, TR, BR, BL order
// from the canonical store.
func cornerShapeBStyleKs(sty *ResolvedStyle) [4]float64 {
	if sty == nil {
		return [4]float64{}
	}

	return [4]float64{
		sty.CornerShapeTopLeft, sty.CornerShapeTopRight,
		sty.CornerShapeBottomRight, sty.CornerShapeBottomLeft,
	}
}

// cornerShapeBBoxPath lays out doc and returns the result plus the box's
// first path-kind op, failing when the declaration emits no path at all
// (a dropped property or painter regression, not a skip).
func cornerShapeBBoxPath(t *testing.T, doc, tag string) (*Result, Op) {
	t.Helper()

	res := cornerShapeBLayoutDoc(t, doc)
	paths := cornerShapeBPathOps(cornerShapeBBoxOps(t, res))

	if len(paths) == 0 {
		t.Fatalf("%s: emitted no OpPath for the box, want a shaped path", tag)
	}

	return res, paths[0]
}

// cornerShapeBControlPath lays out a control document and returns the box's
// first path-kind op, failing when the control paints no path: the test
// declaration already proved paths work, so a pathless control means
// inconsistent wiring.
func cornerShapeBControlPath(t *testing.T, doc, tag, controlTag string) Op {
	t.Helper()

	res := cornerShapeBLayoutDoc(t, doc)
	paths := cornerShapeBPathOps(cornerShapeBBoxOps(t, res))

	if len(paths) == 0 {
		t.Fatalf("%s: control %s paints no path while the test decl does; wiring is inconsistent", tag, controlTag)
	}

	return paths[0]
}

// cornerShapeBRequireDifferentPath fails when the test path equals a control
// path that must differ (a control with a different K on some corner).
func cornerShapeBRequireDifferentPath(t *testing.T, controlDoc, tag, controlTag string, got Op) {
	t.Helper()

	control := cornerShapeBControlPath(t, controlDoc, tag, controlTag)

	if cornerShapeAEqualIgnoringID(got, control) {
		t.Errorf("%s: path equals %s path (%s), want a different corner K to emit different commands",
			tag, controlTag, cornerShapeAFormatPath(got))
	}
}

// cornerShapeBRequireEqualPath fails when the test path differs from a control
// path that must match (an equivalent declaration through another name).
func cornerShapeBRequireEqualPath(t *testing.T, controlDoc, tag, controlTag string, got Op) {
	t.Helper()

	control := cornerShapeBControlPath(t, controlDoc, tag, controlTag)

	if !cornerShapeAEqualIgnoringID(got, control) {
		t.Errorf("%s: path differs from %s path:\ngot     %s\ncontrol %s",
			tag, controlTag, cornerShapeAFormatPath(got), cornerShapeAFormatPath(control))
	}
}

// cornerShapeBAssertKs compares the stored corner K values in TL, TR, BR, BL
// order against want. Infinities (notch, square) compare by sign.
func cornerShapeBAssertKs(t *testing.T, sty *ResolvedStyle, want [4]float64, tag string) {
	t.Helper()

	got := cornerShapeBStyleKs(sty)

	for idx := range 4 {
		if math.IsInf(want[idx], 0) {
			if !math.IsInf(got[idx], 0) || math.Signbit(got[idx]) != math.Signbit(want[idx]) {
				t.Errorf("%s: %s K = %v, want %v", tag, cornerShapeBCornerNames[idx], got[idx], want[idx])
			}

			continue
		}

		if got[idx] != want[idx] {
			t.Errorf("%s: %s K = %v, want %v", tag, cornerShapeBCornerNames[idx], got[idx], want[idx])
		}
	}
}

// cornerShapeBAssertStyleRadii compares the four stored border radii in TL,
// TR, BR, BL order against want (points). These fields exist today, so this
// asserts the radius piece of the corner* shorthands directly.
func cornerShapeBAssertStyleRadii(t *testing.T, sty *ResolvedStyle, want [4]float64, tag string) {
	t.Helper()

	got := [4]float64{
		sty.BorderRadiusTopLeft, sty.BorderRadiusTopRight,
		sty.BorderRadiusBottomRight, sty.BorderRadiusBottomLeft,
	}

	for idx := range 4 {
		if !near(got[idx], want[idx]) {
			t.Errorf("%s: %s radius = %.4fpt (%.2fpx), want %.2fpx",
				tag, cornerShapeBCornerNames[idx], got[idx], got[idx]/ptPerCSSPx, want[idx]/ptPerCSSPx)
		}
	}
}

// cornerShapeBPairCase is the shared body for one-or-two-value pair tests: the
// pair declaration must paint a path whose commands differ from every control
// that shapes a strict subset of its corners (proving each named corner took
// its own K), and the K store must hold wantKs.
func cornerShapeBPairCase(t *testing.T, decl string, wantKs [4]float64, subsetControls []string, tag string) {
	t.Helper()

	res, got := cornerShapeBBoxPath(t, cornerShapeBDoc(decl+";"), tag)

	cornerShapeBAssertKs(t, cornerShapeBBoxStyle(t, res), wantKs, tag)

	for _, control := range subsetControls {
		cornerShapeBRequireDifferentPath(t, cornerShapeBDoc(control+";"), tag, control, got)
	}

	t.Logf("%s: path %s", tag, cornerShapeAFormatPath(got))
}

// TestCornerShapeBTopPairBevelScoop is corner-top-shape:bevel scoop: left
// then right, so top-left is bevel (K=0) and top-right is scoop (K=-1)
// (checklist Phase 1). The other two corners stay round (K=1).
func TestCornerShapeBTopPairBevelScoop(t *testing.T) {
	t.Parallel()

	cornerShapeBPairCase(t, "corner-top-shape:bevel scoop", [4]float64{0, -1, 1, 1},
		[]string{
			"corner-shape:bevel",
			"corner-shape:scoop",
			"corner-top-left-shape:bevel",
			"corner-top-right-shape:scoop",
		}, "corner-top-shape:bevel scoop")
}

// TestCornerShapeBBottomPairScoopBevel is corner-bottom-shape:scoop bevel:
// left then right, so bottom-left is scoop (K=-1) and bottom-right is bevel
// (K=0). The top corners stay round.
func TestCornerShapeBBottomPairScoopBevel(t *testing.T) {
	t.Parallel()

	cornerShapeBPairCase(t, "corner-bottom-shape:scoop bevel", [4]float64{1, 1, 0, -1},
		[]string{
			"corner-shape:bevel",
			"corner-shape:scoop",
			"corner-bottom-left-shape:scoop",
			"corner-bottom-right-shape:bevel",
		}, "corner-bottom-shape:scoop bevel")
}

// TestCornerShapeBLeftPairNotchScoop is corner-left-shape:notch scoop: top
// then bottom, so top-left is notch (K=-infinity) and bottom-left is scoop
// (K=-1). The right corners stay round.
func TestCornerShapeBLeftPairNotchScoop(t *testing.T) {
	t.Parallel()

	cornerShapeBPairCase(t, "corner-left-shape:notch scoop", [4]float64{math.Inf(-1), 1, 1, -1},
		[]string{
			"corner-shape:notch",
			"corner-shape:scoop",
			"corner-top-left-shape:notch",
			"corner-bottom-left-shape:scoop",
		}, "corner-left-shape:notch scoop")
}

// TestCornerShapeBRightPairSquircleSquare is corner-right-shape:squircle
// square: top then bottom, so top-right is squircle (K=2) and bottom-right is
// square (K=+infinity). The left corners stay round.
func TestCornerShapeBRightPairSquircleSquare(t *testing.T) {
	t.Parallel()

	cornerShapeBPairCase(t, "corner-right-shape:squircle square", [4]float64{1, 2, math.Inf(1), 1},
		[]string{
			"corner-shape:squircle",
			"corner-shape:square",
			"corner-top-right-shape:squircle",
			"corner-bottom-right-shape:square",
		}, "corner-right-shape:squircle square")
}

// cornerShapeBLogicalPairCase is the shared body for the logical pair tests:
// the pair declaration must paint, the K store must hold wantKs, the path must
// differ from each single-corner control (proving both corners of the pair
// were set, not just the first), and the path must equal the physical pair
// declaration for the same two corners on the fixed LTR map.
func cornerShapeBLogicalPairCase(
	t *testing.T, decl string, wantKs [4]float64, singleControls []string, physicalPair, tag string,
) {
	t.Helper()

	res, got := cornerShapeBBoxPath(t, cornerShapeBDoc(decl+";"), tag)

	cornerShapeBAssertKs(t, cornerShapeBBoxStyle(t, res), wantKs, tag)

	for _, control := range singleControls {
		cornerShapeBRequireDifferentPath(t, cornerShapeBDoc(control+";"), tag, control, got)
	}

	cornerShapeBRequireEqualPath(t, cornerShapeBDoc(physicalPair+";"), tag, physicalPair, got)
}

// TestCornerShapeBBlockStartPairNotch is corner-block-start-shape:notch:
// start then end on the inline axis, which is top-left then top-right on the
// fixed map (checklist Phase 1). Same path as corner-top-shape:notch notch.
func TestCornerShapeBBlockStartPairNotch(t *testing.T) {
	t.Parallel()

	cornerShapeBLogicalPairCase(t, "corner-block-start-shape:notch",
		[4]float64{math.Inf(-1), math.Inf(-1), 1, 1},
		[]string{"corner-top-left-shape:notch", "corner-top-right-shape:notch"},
		"corner-top-shape:notch notch", "corner-block-start-shape:notch")
}

// TestCornerShapeBBlockEndPairNotch is corner-block-end-shape:notch: end-start
// then end-end, which is bottom-left then bottom-right on the fixed map. Same
// path as corner-bottom-shape:notch notch.
func TestCornerShapeBBlockEndPairNotch(t *testing.T) {
	t.Parallel()

	cornerShapeBLogicalPairCase(t, "corner-block-end-shape:notch",
		[4]float64{1, 1, math.Inf(-1), math.Inf(-1)},
		[]string{"corner-bottom-left-shape:notch", "corner-bottom-right-shape:notch"},
		"corner-bottom-shape:notch notch", "corner-block-end-shape:notch")
}

// TestCornerShapeBInlineStartPairNotch is corner-inline-start-shape:notch:
// start-start then end-start on the block axis, which is top-left then
// bottom-left on the fixed map. Same path as corner-left-shape:notch notch.
func TestCornerShapeBInlineStartPairNotch(t *testing.T) {
	t.Parallel()

	cornerShapeBLogicalPairCase(t, "corner-inline-start-shape:notch",
		[4]float64{math.Inf(-1), 1, 1, math.Inf(-1)},
		[]string{"corner-top-left-shape:notch", "corner-bottom-left-shape:notch"},
		"corner-left-shape:notch notch", "corner-inline-start-shape:notch")
}

// TestCornerShapeBInlineEndPairNotch is corner-inline-end-shape:notch:
// start-end then end-end, which is top-right then bottom-right on the fixed
// map. Same path as corner-right-shape:notch notch.
func TestCornerShapeBInlineEndPairNotch(t *testing.T) {
	t.Parallel()

	cornerShapeBLogicalPairCase(t, "corner-inline-end-shape:notch",
		[4]float64{1, math.Inf(-1), math.Inf(-1), 1},
		[]string{"corner-top-right-shape:notch", "corner-bottom-right-shape:notch"},
		"corner-right-shape:notch notch", "corner-inline-end-shape:notch")
}

// TestCornerShapeBLogicalSinglesMatchPhysical is the logical single-corner
// equivalence: in horizontal left-to-right flow corner-start-start-shape is
// top-left, start-end is top-right, end-start is bottom-left, and end-end is
// bottom-right. The radius precedent is TestBehaviorBorderStartStartRadiusMapsTopLeft
// (css_behavior_borders_rest_test.go:405), which pins the same fixed map for
// border radii. Each logical declaration must paint the identical path as its
// physical twin and store the identical K.
func TestCornerShapeBLogicalSinglesMatchPhysical(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		logical  string
		physical string
		wantIdx  int
	}{
		{
			name:     "start-start matches top-left",
			logical:  "corner-start-start-shape:bevel",
			physical: "corner-top-left-shape:bevel",
			wantIdx:  0,
		},
		{
			name:     "start-end matches top-right",
			logical:  "corner-start-end-shape:bevel",
			physical: "corner-top-right-shape:bevel",
			wantIdx:  1,
		},
		{
			name:     "end-start matches bottom-left",
			logical:  "corner-end-start-shape:bevel",
			physical: "corner-bottom-left-shape:bevel",
			wantIdx:  3,
		},
		{
			name:     "end-end matches bottom-right",
			logical:  "corner-end-end-shape:bevel",
			physical: "corner-bottom-right-shape:bevel",
			wantIdx:  2,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cornerShapeBLogicalSingleCase(t, testCase.logical, testCase.physical, testCase.wantIdx)
		})
	}
}

// cornerShapeBLogicalSingleCase checks one logical/physical twin pair: equal
// painted paths, equal stored K, and only the wantIdx corner shaped.
func cornerShapeBLogicalSingleCase(t *testing.T, logical, physical string, wantIdx int) {
	t.Helper()

	res, got := cornerShapeBBoxPath(t, cornerShapeBDoc(logical+";"), logical)

	cornerShapeBRequireEqualPath(t, cornerShapeBDoc(physical+";"), logical, physical, got)

	logicalKs := cornerShapeBStyleKs(cornerShapeBBoxStyle(t, res))

	physicalRes := cornerShapeBLayoutDoc(t, cornerShapeBDoc(physical+";"))
	physicalKs := cornerShapeBStyleKs(cornerShapeBBoxStyle(t, physicalRes))

	if logicalKs != physicalKs {
		t.Errorf("%s: logical Ks %v != physical Ks %v, want identical store", logical, logicalKs, physicalKs)
	}

	want := [4]float64{1, 1, 1, 1}
	want[wantIdx] = 0

	if logicalKs != want {
		t.Errorf("%s: Ks %v, want only corner %d bevel (K=0), rest round (K=1)", logical, logicalKs, wantIdx)
	}
}

// TestCornerShapeBCornerShorthandRadiusShape is corner:16px bevel: the radius
// piece calls the existing setter on all four corners and the shape piece
// sets K=0 everywhere (checklist Phase 1). The painted path must equal the
// longhand combo border-radius:16px with corner-shape:bevel.
func TestCornerShapeBCornerShorthandRadiusShape(t *testing.T) {
	t.Parallel()

	const tag = "corner:16px bevel"

	res, got := cornerShapeBBoxPath(t, cornerShapeBDocBare(tag+";"), tag)

	r16 := pxToPt(16)
	cornerShapeBAssertStyleRadii(t, cornerShapeBBoxStyle(t, res),
		[4]float64{r16, r16, r16, r16}, tag)
	cornerShapeBAssertKs(t, cornerShapeBBoxStyle(t, res), [4]float64{0, 0, 0, 0}, tag)

	cornerShapeBRequireEqualPath(t, cornerShapeBDoc("corner-shape:bevel;"),
		tag, "border-radius:16px corner-shape:bevel", got)

	t.Logf("%s: path %s", tag, cornerShapeAFormatPath(got))
}

// cornerShapeBSideCase describes one side-shorthand row: the test shorthand,
// the equivalent radius-plus-shape longhand control, and the two corners (TL,
// TR, BR, BL order) the side owns on the fixed map.
type cornerShapeBSideCase struct {
	name        string
	testDecl    string
	controlDecl string
	corners     [2]int
}

// cornerShapeBSideCases lists the four physical sides plus the four logical
// sides on the fixed LTR map: block-start is top, block-end is bottom,
// inline-start is left, inline-end is right.
func cornerShapeBSideCases() []cornerShapeBSideCase {
	return []cornerShapeBSideCase{
		{
			name:        "top",
			testDecl:    "corner-top:16px scoop;",
			controlDecl: "border-top-radius:16px;corner-top-shape:scoop;",
			corners:     [2]int{0, 1},
		},
		{
			name:        "right",
			testDecl:    "corner-right:16px scoop;",
			controlDecl: "border-right-radius:16px;corner-right-shape:scoop;",
			corners:     [2]int{1, 2},
		},
		{
			name:        "bottom",
			testDecl:    "corner-bottom:16px scoop;",
			controlDecl: "border-bottom-radius:16px;corner-bottom-shape:scoop;",
			corners:     [2]int{3, 2},
		},
		{
			name:        "left",
			testDecl:    "corner-left:16px scoop;",
			controlDecl: "border-left-radius:16px;corner-left-shape:scoop;",
			corners:     [2]int{0, 3},
		},
		{
			name:        "block-start",
			testDecl:    "corner-block-start:16px scoop;",
			controlDecl: "border-block-start-radius:16px;corner-block-start-shape:scoop;",
			corners:     [2]int{0, 1},
		},
		{
			name:        "block-end",
			testDecl:    "corner-block-end:16px scoop;",
			controlDecl: "border-block-end-radius:16px;corner-block-end-shape:scoop;",
			corners:     [2]int{3, 2},
		},
		{
			name:        "inline-start",
			testDecl:    "corner-inline-start:16px scoop;",
			controlDecl: "border-inline-start-radius:16px;corner-inline-start-shape:scoop;",
			corners:     [2]int{0, 3},
		},
		{
			name:        "inline-end",
			testDecl:    "corner-inline-end:16px scoop;",
			controlDecl: "border-inline-end-radius:16px;corner-inline-end-shape:scoop;",
			corners:     [2]int{1, 2},
		},
	}
}

// TestCornerShapeBSideShorthands is corner-top:16px scoop and its side
// equivalents: the shorthand sets the side radii and the side shapes (K=-1)
// and leaves the other two corners at the initial radius (0) and round (K=1).
// Each shorthand must paint the identical path as its radius-plus-shape
// longhand control.
func TestCornerShapeBSideShorthands(t *testing.T) {
	t.Parallel()

	for _, testCase := range cornerShapeBSideCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cornerShapeBCheckSideCase(t, testCase)
		})
	}
}

// cornerShapeBCheckSideCase checks one side-shorthand row.
func cornerShapeBCheckSideCase(t *testing.T, testCase cornerShapeBSideCase) {
	t.Helper()

	res, got := cornerShapeBBoxPath(t, cornerShapeBDocBare(testCase.testDecl), testCase.testDecl)

	var wantRadii, wantKs [4]float64

	for idx := range 4 {
		if idx == testCase.corners[0] || idx == testCase.corners[1] {
			wantRadii[idx] = pxToPt(16)
			wantKs[idx] = -1
		} else {
			wantKs[idx] = 1
		}
	}

	sty := cornerShapeBBoxStyle(t, res)
	cornerShapeBAssertStyleRadii(t, sty, wantRadii, testCase.testDecl)
	cornerShapeBAssertKs(t, sty, wantKs, testCase.testDecl)

	cornerShapeBRequireEqualPath(t, cornerShapeBDocBare(testCase.controlDecl),
		testCase.testDecl, testCase.controlDecl, got)
}

// cornerShapeBSingleShorthandCase describes one single-corner shorthand row:
// the test shorthand, the equivalent radius-plus-shape longhand control, and
// the corner index (TL, TR, BR, BL order) it owns on the fixed map.
type cornerShapeBSingleShorthandCase struct {
	name        string
	idx         int
	testDecl    string
	controlDecl string
}

// cornerShapeBSingleShorthandCases lists the four physical corners plus the
// four logical corners on the fixed LTR map.
func cornerShapeBSingleShorthandCases() []cornerShapeBSingleShorthandCase {
	return []cornerShapeBSingleShorthandCase{
		{
			name:        "top-left",
			idx:         0,
			testDecl:    "corner-top-left:16px notch;",
			controlDecl: "border-top-left-radius:16px;corner-top-left-shape:notch;",
		},
		{
			name:        "top-right",
			idx:         1,
			testDecl:    "corner-top-right:16px notch;",
			controlDecl: "border-top-right-radius:16px;corner-top-right-shape:notch;",
		},
		{
			name:        "bottom-right",
			idx:         2,
			testDecl:    "corner-bottom-right:16px notch;",
			controlDecl: "border-bottom-right-radius:16px;corner-bottom-right-shape:notch;",
		},
		{
			name:        "bottom-left",
			idx:         3,
			testDecl:    "corner-bottom-left:16px notch;",
			controlDecl: "border-bottom-left-radius:16px;corner-bottom-left-shape:notch;",
		},
		{
			name:        "start-start",
			idx:         0,
			testDecl:    "corner-start-start:16px notch;",
			controlDecl: "border-start-start-radius:16px;corner-start-start-shape:notch;",
		},
		{
			name:        "start-end",
			idx:         1,
			testDecl:    "corner-start-end:16px notch;",
			controlDecl: "border-start-end-radius:16px;corner-start-end-shape:notch;",
		},
		{
			name:        "end-start",
			idx:         3,
			testDecl:    "corner-end-start:16px notch;",
			controlDecl: "border-end-start-radius:16px;corner-end-start-shape:notch;",
		},
		{
			name:        "end-end",
			idx:         2,
			testDecl:    "corner-end-end:16px notch;",
			controlDecl: "border-end-end-radius:16px;corner-end-end-shape:notch;",
		},
	}
}

// cornerShapeBPhysicalShorthandFor maps a logical single-corner shorthand to
// its physical twin on the fixed map, for the cross-check that the shorthand
// logical map matches the longhand one.
func cornerShapeBPhysicalShorthandFor(logical string) string {
	switch logical {
	case "start-start":
		return "top-left"
	case "start-end":
		return "top-right"
	case "end-start":
		return "bottom-left"
	case "end-end":
		return "bottom-right"
	default:
		return ""
	}
}

// TestCornerShapeBSingleCornerShorthands is corner-top-left:16px notch and its
// corner equivalents: the shorthand sets only that corner's radius (16px) and
// that corner's K (-infinity) and leaves the other three corners at the
// initial radius (0) and round (K=1). Each shorthand must paint the identical
// path as its radius-plus-shape longhand control, and each logical shorthand
// must match its physical twin.
func TestCornerShapeBSingleCornerShorthands(t *testing.T) {
	t.Parallel()

	byName := map[string]cornerShapeBSingleShorthandCase{}
	for _, testCase := range cornerShapeBSingleShorthandCases() {
		byName[testCase.name] = testCase
	}

	for _, testCase := range cornerShapeBSingleShorthandCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cornerShapeBCheckSingleShorthandCase(t, testCase, byName)
		})
	}
}

// cornerShapeBCheckSingleShorthandCase checks one single-corner shorthand row,
// including its logical twin when the row is logical.
func cornerShapeBCheckSingleShorthandCase(
	t *testing.T, testCase cornerShapeBSingleShorthandCase, byName map[string]cornerShapeBSingleShorthandCase,
) {
	t.Helper()

	res, got := cornerShapeBBoxPath(t, cornerShapeBDocBare(testCase.testDecl), testCase.testDecl)

	var wantRadii, wantKs [4]float64

	for idx := range 4 {
		if idx == testCase.idx {
			wantRadii[idx] = pxToPt(16)
			wantKs[idx] = math.Inf(-1)
		} else {
			wantKs[idx] = 1
		}
	}

	sty := cornerShapeBBoxStyle(t, res)
	cornerShapeBAssertStyleRadii(t, sty, wantRadii, testCase.testDecl)
	cornerShapeBAssertKs(t, sty, wantKs, testCase.testDecl)

	cornerShapeBRequireEqualPath(t, cornerShapeBDocBare(testCase.controlDecl),
		testCase.testDecl, testCase.controlDecl, got)

	physical := cornerShapeBPhysicalShorthandFor(testCase.name)
	if physical == "" {
		return
	}

	twin := byName[physical]

	cornerShapeBRequireEqualPath(t, cornerShapeBDocBare(twin.testDecl),
		testCase.testDecl, twin.testDecl, got)
}

// TestCornerShapeBBadTokenKeepsBevel is the invalid-value rule (checklist
// Phase 1): invalid values leave the previous declaration, so a bad shape
// token after a valid bevel must not replace it. Both the single longhand
// and the side pair keep their bevel paths and their K values verbatim.
func TestCornerShapeBBadTokenKeepsBevel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		before string
		after  string
		wantKs [4]float64
	}{
		{
			name:   "single longhand keeps bevel",
			before: "corner-top-left-shape:bevel;",
			after:  "corner-top-left-shape:bevel;corner-top-left-shape:bogus;",
			wantKs: [4]float64{0, 1, 1, 1},
		},
		{
			name:   "side pair keeps bevel scoop",
			before: "corner-top-shape:bevel scoop;",
			after:  "corner-top-shape:bevel scoop;corner-top-shape:bogus;",
			wantKs: [4]float64{0, -1, 1, 1},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cornerShapeBBadTokenCase(t, testCase.name, testCase.before, testCase.after, testCase.wantKs)
		})
	}
}

// cornerShapeBBadTokenCase checks one bogus-token row: the bogus-suffixed
// declaration paints the same path and stores the same Ks as the valid one.
func cornerShapeBBadTokenCase(t *testing.T, name, before, after string, wantKs [4]float64) {
	t.Helper()

	afterRes, afterPath := cornerShapeBBoxPath(t, cornerShapeBDoc(after), after)

	beforeRes := cornerShapeBLayoutDoc(t, cornerShapeBDoc(before))
	beforePaths := cornerShapeBPathOps(cornerShapeBBoxOps(t, beforeRes))

	if len(beforePaths) == 0 {
		t.Fatalf("%s: valid declaration %q paints no path while the bogus-suffixed one does", name, before)
	}

	if !cornerShapeAEqualIgnoringID(afterPath, beforePaths[0]) {
		t.Errorf("%s: bogus-suffixed path differs from the valid path:\nafter  %s\nbefore %s",
			name, cornerShapeAFormatPath(afterPath), cornerShapeAFormatPath(beforePaths[0]))
	}

	afterKs := cornerShapeBStyleKs(cornerShapeBBoxStyle(t, afterRes))
	beforeKs := cornerShapeBStyleKs(cornerShapeBBoxStyle(t, beforeRes))

	if afterKs != beforeKs {
		t.Errorf("%s: Ks after bogus %v != Ks before %v, want the earlier bevel kept",
			name, afterKs, beforeKs)
	}

	if afterKs != wantKs {
		t.Errorf("%s: Ks %v, want %v", name, afterKs, wantKs)
	}
}
