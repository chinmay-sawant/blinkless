package layout

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

// Part A corner-shape behavior tests (Phase 3, first half).
//
// Every test checks the drawing list, never a page PNG, following
// css_behavior_borders_test.go patterns: one 120px by 60px red box, lengths in
// CSS px converted with pxToPt, helpers from layout_test.go and
// css_review_02_test.go (layoutHTML, boxByID, near, pxToPt, ptPerCSSPx).
//
// Reference is the CSS Borders 4 draft corner-shape section (fetched
// 2026-10-15, https://drafts.csswg.org/css-borders-4/#typedef-corner-shape-value):
// round K=1, squircle K=2, square K=infinity, bevel K=0, scoop K=-1,
// notch K=-infinity, plus superellipse(<number>). There is no Chrome parity
// claim for these names; the catalog records no browser result.
//
// The OpPath kind (corner_path.go), the per-corner K store on ResolvedStyle
// (style_corner_shape.go), and the painter (corner_shape_paint.go) are all
// landed, so the assertions below run for real.

// cornerShapeAItemID is the id of the single test box in cornerShapeADoc.
const cornerShapeAItemID = "item"

// cornerShapeADoc lays out one 120px by 60px red box with a 16px corner box
// plus the given extra declaration (a corner-shape decl or empty).
func cornerShapeADoc(extraDecl string) string {
	return `<html style="margin:0"><body style="margin:0">` +
		`<div id="` + cornerShapeAItemID + `" style="width:120px;height:60px;background-color:#f00;border-radius:16px;` +
		extraDecl + `"></div>` +
		`</body></html>`
}

// cornerShapeALayout runs cornerShapeADoc through the engine.
func cornerShapeALayout(t *testing.T, extraDecl string) *Result {
	t.Helper()

	return layoutHTML(t, cornerShapeADoc(extraDecl))
}

// cornerShapeABoxOps returns the ops owned by the test box, using its
// recorded op range. Same rule as behaviorBordersBoxOps in
// css_behavior_borders_test.go, restated here so this file stands alone.
func cornerShapeABoxOps(t *testing.T, res *Result) []Op {
	t.Helper()

	elementBox := boxByID(t, res, cornerShapeAItemID)
	if elementBox.opEnd < elementBox.opStart {
		return nil
	}

	start, end := elementBox.opStart, elementBox.opEnd
	if end >= len(res.Ops) {
		end = len(res.Ops) - 1
	}

	return res.Ops[start : end+1]
}

// cornerShapeAPathOps selects OpPath ops from ops.
func cornerShapeAPathOps(ops []Op) []Op {
	var out []Op

	for _, paintOp := range ops {
		if paintOp.Kind == OpPath {
			out = append(out, paintOp)
		}
	}

	return out
}

// cornerShapeAFindRedFill returns the first red OpFillRect in ops, or nil.
// Unlike redFillOp in radius_test.go it does not fail when the background
// was replaced by a path, which is the expected bevel outcome.
func cornerShapeAFindRedFill(ops []Op) *Op {
	for idx := range ops {
		if ops[idx].Kind == OpFillRect && ops[idx].R > 0.9 {
			candidate := ops[idx]

			return &candidate
		}
	}

	return nil
}

// cornerShapeAHasRadiiFill reports whether any fill in ops carries an
// elliptical corner radius on any corner (either axis).
func cornerShapeAHasRadiiFill(ops []Op) bool {
	for idx := range ops {
		if ops[idx].Kind == OpFillRect && cornerShapeAOpHasRadii(&ops[idx]) {
			return true
		}
	}

	return false
}

// cornerShapeAOpHasRadii reports whether one fill op carries a non-zero
// radius on any corner or the uniform pair.
func cornerShapeAOpHasRadii(paintOp *Op) bool {
	radii := [10]float64{
		paintOp.RadiusTopLeft, paintOp.RadiusTopRight,
		paintOp.RadiusBottomRight, paintOp.RadiusBottomLeft,
		paintOp.RadiusTopLeftY, paintOp.RadiusTopRightY,
		paintOp.RadiusBottomRightY, paintOp.RadiusBottomLeftY,
		paintOp.Radius, paintOp.RadiusY,
	}

	for _, radius := range radii {
		if !near(radius, 0) {
			return true
		}
	}

	return false
}

// cornerShapeAZeroID returns a copy of paintOp with ID cleared so two layouts
// can be compared without their per-run operation identities.
func cornerShapeAZeroID(paintOp Op) Op {
	paintOp.ID = 0

	return paintOp
}

// cornerShapeAEqualIgnoringID reports whether two ops carry the same drawing
// payload after clearing IDs. reflect.DeepEqual follows the opExtra pointer,
// so the corner path payload participates without this file naming its type.
// Comparing payloads directly is the "compare path commands, not a
// screenshot" rule from the checklist.
func cornerShapeAEqualIgnoringID(a, b Op) bool {
	return reflect.DeepEqual(cornerShapeAZeroID(a), cornerShapeAZeroID(b))
}

// cornerShapeAFormatPath renders a path op for failure messages.
func cornerShapeAFormatPath(paintOp Op) string {
	path, ok := paintOp.CornerPathValue()
	if !ok {
		return fmt.Sprintf("kind=%d no corner path payload: %+v", uint8(paintOp.Kind), paintOp)
	}

	var builder strings.Builder

	fmt.Fprintf(&builder, "stroke=%v commands=%d [", path.Stroke, len(path.Cmds))

	for _, command := range path.Cmds {
		fmt.Fprintf(&builder, " %s", cornerShapeAFormatCommand(command))
	}

	builder.WriteString(" ]")

	return builder.String()
}

// cornerShapeAFormatCommand renders one path command for failure messages.
func cornerShapeAFormatCommand(command PathCmd) string {
	switch command.Verb {
	case PathMoveTo:
		return fmt.Sprintf("moveTo(%.3f,%.3f)", command.Pts[0].X, command.Pts[0].Y)
	case PathLineTo:
		return fmt.Sprintf("lineTo(%.3f,%.3f)", command.Pts[0].X, command.Pts[0].Y)
	case PathClose:
		return "close"
	case PathQuadTo:
		return fmt.Sprintf("quadTo(%.3f,%.3f %.3f,%.3f)",
			command.Pts[0].X, command.Pts[0].Y, command.Pts[1].X, command.Pts[1].Y)
	case PathCubicTo:
		return fmt.Sprintf("cubicTo(%.3f,%.3f %.3f,%.3f %.3f,%.3f)",
			command.Pts[0].X, command.Pts[0].Y, command.Pts[1].X, command.Pts[1].Y,
			command.Pts[2].X, command.Pts[2].Y)
	default:
		return fmt.Sprintf("verb(%d)", command.Verb)
	}
}

// cornerShapeAStyleKs reads the four corner K values in TL, TR, BR, BL order
// from the canonical store.
func cornerShapeAStyleKs(sty *ResolvedStyle) [4]float64 {
	if sty == nil {
		return [4]float64{}
	}

	return [4]float64{
		sty.CornerShapeTopLeft, sty.CornerShapeTopRight,
		sty.CornerShapeBottomRight, sty.CornerShapeBottomLeft,
	}
}

// cornerShapeARequirePathSupport returns the result and the box ops for the
// decl, failing when the declaration emits no path-kind op at all: the
// painter is wired, so a path-less decl is a regression, not a skip.
func cornerShapeARequirePathSupport(t *testing.T, decl string) (*Result, []Op) {
	t.Helper()

	res := cornerShapeALayout(t, decl)
	boxOps := cornerShapeABoxOps(t, res)

	if len(cornerShapeAPathOps(boxOps)) == 0 {
		t.Fatalf("%q emitted no OpPath for the box, want a shaped path", decl)
	}

	return res, boxOps
}

// cornerShapeAAssertNoRadiiFill fails when the box still owns an elliptical
// radii fill, which Phase 2 forbids for a non-round box: the background must
// move to OpPath instead of a radiused OpFillRect.
func cornerShapeAAssertNoRadiiFill(t *testing.T, boxOps []Op, tag string) {
	t.Helper()

	if cornerShapeAHasRadiiFill(boxOps) {
		fill := cornerShapeAFindRedFill(boxOps)
		detail := "red fill missing but some fill carries radii"

		if fill != nil {
			detail = fmt.Sprintf("fill radii TL=%.3f TR=%.3f BR=%.3f BL=%.3f",
				fill.RadiusTopLeft, fill.RadiusTopRight, fill.RadiusBottomRight, fill.RadiusBottomLeft)
		}

		t.Fatalf("%s: box still owns an elliptical radii fill (%s), want OpPath only", tag, detail)
	}
}

// TestCornerShapeARoundMatchesRadius is corner-shape:round on a 16px radius:
// it must match the op geometry of border-radius:16px alone. Round is K=1,
// the initial value, and keeps today's radii with zero OpPath (checklist
// Phase 0 and Phase 2).
func TestCornerShapeARoundMatchesRadius(t *testing.T) {
	t.Parallel()

	plain := cornerShapeALayout(t, "")
	round := cornerShapeALayout(t, "corner-shape:round;")

	if len(cornerShapeAPathOps(plain.Ops)) != 0 {
		t.Fatalf("radius-only box emitted %d path-kind ops, want 0 (baseline must stay radii)",
			len(cornerShapeAPathOps(plain.Ops)))
	}

	if len(cornerShapeAPathOps(round.Ops)) != 0 {
		t.Fatalf("corner-shape:round emitted %d path-kind ops, want 0 (round keeps radii)",
			len(cornerShapeAPathOps(round.Ops)))
	}

	plainFill := cornerShapeAFindRedFill(plain.Ops)
	roundFill := cornerShapeAFindRedFill(round.Ops)

	if plainFill == nil || roundFill == nil {
		t.Fatalf("red fills = plain %v round %v, want one each", plainFill != nil, roundFill != nil)
	}

	for name, pair := range map[string][2]float64{
		"RadiusTopLeft":     {roundFill.RadiusTopLeft, plainFill.RadiusTopLeft},
		"RadiusTopRight":    {roundFill.RadiusTopRight, plainFill.RadiusTopRight},
		"RadiusBottomRight": {roundFill.RadiusBottomRight, plainFill.RadiusBottomRight},
		"RadiusBottomLeft":  {roundFill.RadiusBottomLeft, plainFill.RadiusBottomLeft},
	} {
		if !near(pair[0], pair[1]) {
			t.Errorf("round fill %s = %.4fpt, radius-only = %.4fpt, want equal", name, pair[0], pair[1])
		}
	}

	if !near(roundFill.RadiusTopLeft, pxToPt(16)) {
		t.Errorf("round fill RadiusTopLeft = %.4fpt (%.2fpx), want 16px",
			roundFill.RadiusTopLeft, roundFill.RadiusTopLeft/ptPerCSSPx)
	}

	if !cornerShapeAEqualIgnoringID(*plainFill, *roundFill) {
		t.Errorf("round fill differs from radius-only fill:\nplain %+v\nround %+v", *plainFill, *roundFill)
	}
}

// TestCornerShapeABevelEmitsPath is corner-shape:bevel (K=0): the box must
// emit OpPath and must not keep an elliptical radii fill for that background
// (checklist Phase 2 gate).
func TestCornerShapeABevelEmitsPath(t *testing.T) {
	t.Parallel()

	_, boxOps := cornerShapeARequirePathSupport(t, "corner-shape:bevel;")

	paths := cornerShapeAPathOps(boxOps)

	cornerShapeAAssertNoRadiiFill(t, boxOps, "corner-shape:bevel")

	t.Logf("bevel path: %s", cornerShapeAFormatPath(paths[0]))
}

// TestCornerShapeAScoopConcave is corner-shape:scoop (K=-1): it emits a path
// whose corner is concave relative to round. Round has no path (it keeps
// radii), so concavity is shown by comparing path commands: scoop commands
// differ from the bevel (K=0, straight cut) diagonal, proving a different K
// drove the curve.
func TestCornerShapeAScoopConcave(t *testing.T) {
	t.Parallel()

	_, scoopBoxOps := cornerShapeARequirePathSupport(t, "corner-shape:scoop;")

	scoopPaths := cornerShapeAPathOps(scoopBoxOps)

	cornerShapeAAssertNoRadiiFill(t, scoopBoxOps, "corner-shape:scoop")

	bevelRes := cornerShapeALayout(t, "corner-shape:bevel;")
	bevelPaths := cornerShapeAPathOps(cornerShapeABoxOps(t, bevelRes))

	if len(bevelPaths) == 0 {
		t.Fatalf("scoop paints a path but bevel does not; the concave comparison needs both")
	}

	if cornerShapeAEqualIgnoringID(scoopPaths[0], bevelPaths[0]) {
		t.Errorf("scoop path equals bevel path (%s), want concave (K=-1) vs straight (K=0) commands to differ",
			cornerShapeAFormatPath(scoopPaths[0]))
	}

	// Both corners share the 16px corner box, so the outer bounds match; only
	// the inner curve differs. Bounds equality keeps this check about the
	// curve, not about a moved box.
	if !near(scoopPaths[0].X, bevelPaths[0].X) || !near(scoopPaths[0].Y, bevelPaths[0].Y) ||
		!near(scoopPaths[0].W, bevelPaths[0].W) || !near(scoopPaths[0].H, bevelPaths[0].H) {
		t.Errorf("scoop bounds (%.3f,%.3f %.3fx%.3f) != bevel bounds (%.3f,%.3f %.3fx%.3f), want same outer box",
			scoopPaths[0].X, scoopPaths[0].Y, scoopPaths[0].W, scoopPaths[0].H,
			bevelPaths[0].X, bevelPaths[0].Y, bevelPaths[0].W, bevelPaths[0].H)
	}

	t.Logf("scoop path: %s", cornerShapeAFormatPath(scoopPaths[0]))
	t.Logf("bevel path: %s", cornerShapeAFormatPath(bevelPaths[0]))
}

// TestCornerShapeANotchSquareSquircleDistinct is corner-shape:notch, square,
// and squircle: each emits a path and each stores a different K
// (notch K=-infinity, square K=+infinity, squircle K=2, per the Borders 4
// draft). Different K values must produce different path commands.
func TestCornerShapeANotchSquareSquircleDistinct(t *testing.T) {
	t.Parallel()

	decls := map[string]string{
		"notch":    "corner-shape:notch;",
		"square":   "corner-shape:square;",
		"squircle": "corner-shape:squircle;",
	}

	paths := map[string]Op{}
	kValues := map[string]float64{}

	for name, decl := range decls {
		res, boxOps := cornerShapeARequirePathSupport(t, decl)

		pathOps := cornerShapeAPathOps(boxOps)

		cornerShapeAAssertNoRadiiFill(t, boxOps, "corner-shape:"+name)

		paths[name] = pathOps[0]

		// All four corners share the shorthand value, so TL carries it.
		sty := boxByID(t, res, cornerShapeAItemID).style
		cornerKs := cornerShapeAStyleKs(sty)
		kValues[name] = cornerKs[0]
	}

	names := []string{"notch", "square", "squircle"}
	cornerShapeAAssertDistinctPaths(t, names, paths)
	cornerShapeAAssertDistinctKs(t, names, kValues)
}

// cornerShapeAAssertDistinctPaths fails when any two named corners emit equal
// path commands.
func cornerShapeAAssertDistinctPaths(t *testing.T, names []string, paths map[string]Op) {
	t.Helper()

	for i := range names {
		for j := i + 1; j < len(names); j++ {
			first, second := names[i], names[j]
			if cornerShapeAEqualIgnoringID(paths[first], paths[second]) {
				t.Errorf("%s path equals %s path (%s), want different K to emit different commands",
					first, second, cornerShapeAFormatPath(paths[first]))
			}
		}
	}
}

// cornerShapeAAssertDistinctKs fails when any two named corners store the
// same K (infinities compared with their sign).
func cornerShapeAAssertDistinctKs(t *testing.T, names []string, kValues map[string]float64) {
	t.Helper()

	for i := range names {
		for j := i + 1; j < len(names); j++ {
			first, second := names[i], names[j]
			firstK, secondK := kValues[first], kValues[second]

			same := firstK == secondK ||
				(math.IsInf(firstK, 0) && math.IsInf(secondK, 0) && math.Signbit(firstK) == math.Signbit(secondK))
			if same {
				t.Errorf("stored K %s=%v equals %s=%v, want distinct (notch=-inf square=+inf squircle=2)",
					first, firstK, second, secondK)
			}
		}
	}
}

// TestCornerShapeASuperellipseMatchesKeywords is superellipse() equivalence:
// superellipse(1) matches round, superellipse(0) matches bevel, and
// superellipse(-1) matches scoop (keyword to K table in the checklist).
func TestCornerShapeASuperellipseMatchesKeywords(t *testing.T) {
	t.Parallel()

	t.Run("one matches round", func(t *testing.T) {
		t.Parallel()

		cornerShapeAAssertSuperellipseOneMatchesRound(t)
	})

	t.Run("zero matches bevel", func(t *testing.T) {
		t.Parallel()

		cornerShapeAAssertSuperellipseZeroMatchesBevel(t)
	})

	t.Run("minus one matches scoop", func(t *testing.T) {
		t.Parallel()

		cornerShapeAAssertSuperellipseMinusOneMatchesScoop(t)
	})
}

// cornerShapeAAssertSuperellipseOneMatchesRound checks that superellipse(1)
// keeps radii and paints no path, equal to round.
func cornerShapeAAssertSuperellipseOneMatchesRound(t *testing.T) {
	t.Helper()

	round := cornerShapeALayout(t, "corner-shape:round;")
	one := cornerShapeALayout(t, "corner-shape:superellipse(1);")

	if len(cornerShapeAPathOps(round.Ops)) != 0 || len(cornerShapeAPathOps(one.Ops)) != 0 {
		t.Fatalf("round/superellipse(1) emitted paths %d/%d, want 0 each (K=1 keeps radii)",
			len(cornerShapeAPathOps(round.Ops)), len(cornerShapeAPathOps(one.Ops)))
	}

	roundFill := cornerShapeAFindRedFill(round.Ops)
	oneFill := cornerShapeAFindRedFill(one.Ops)

	if roundFill == nil || oneFill == nil {
		t.Fatalf("red fills = round %v superellipse(1) %v, want one each", roundFill != nil, oneFill != nil)
	}

	if !cornerShapeAEqualIgnoringID(*roundFill, *oneFill) {
		t.Errorf("superellipse(1) fill differs from round fill:\nround %+v\none %+v", *roundFill, *oneFill)
	}
}

// cornerShapeAAssertSuperellipseZeroMatchesBevel checks that superellipse(0)
// paints the same path as bevel.
func cornerShapeAAssertSuperellipseZeroMatchesBevel(t *testing.T) {
	t.Helper()

	bevelRes := cornerShapeALayout(t, "corner-shape:bevel;")
	bevelPaths := cornerShapeAPathOps(cornerShapeABoxOps(t, bevelRes))

	if len(bevelPaths) == 0 {
		t.Fatalf("bevel path not in the box range")
	}

	zeroRes := cornerShapeALayout(t, "corner-shape:superellipse(0);")
	zeroPaths := cornerShapeAPathOps(cornerShapeABoxOps(t, zeroRes))

	if len(zeroPaths) == 0 {
		t.Fatalf("corner-shape:bevel paints a path but superellipse(0) paints none, want equal (K=0)")
	}

	if !cornerShapeAEqualIgnoringID(bevelPaths[0], zeroPaths[0]) {
		t.Errorf("superellipse(0) path differs from bevel path:\nbevel %s\nzero %s",
			cornerShapeAFormatPath(bevelPaths[0]), cornerShapeAFormatPath(zeroPaths[0]))
	}
}

// cornerShapeAAssertSuperellipseMinusOneMatchesScoop checks that
// superellipse(-1) paints the same path as scoop.
func cornerShapeAAssertSuperellipseMinusOneMatchesScoop(t *testing.T) {
	t.Helper()

	scoopRes := cornerShapeALayout(t, "corner-shape:scoop;")
	scoopPaths := cornerShapeAPathOps(cornerShapeABoxOps(t, scoopRes))

	if len(scoopPaths) == 0 {
		t.Fatalf("scoop path not in the box range")
	}

	negRes := cornerShapeALayout(t, "corner-shape:superellipse(-1);")
	negPaths := cornerShapeAPathOps(cornerShapeABoxOps(t, negRes))

	if len(negPaths) == 0 {
		t.Fatalf("corner-shape:scoop paints a path but superellipse(-1) paints none, want equal (K=-1)")
	}

	if !cornerShapeAEqualIgnoringID(scoopPaths[0], negPaths[0]) {
		t.Errorf("superellipse(-1) path differs from scoop path:\nscoop %s\nneg %s",
			cornerShapeAFormatPath(scoopPaths[0]), cornerShapeAFormatPath(negPaths[0]))
	}
}

// cornerShapeASingleCornerCase is the shared body for the four single-corner
// shape longhands. prop names the declaration (for example
// "corner-top-left-shape:bevel"), wantIdx is the corner that must change
// (0=TL 1=TR 2=BR 3=BL), and cornerName names it in failures.
//
// Mix rule (checklist Phase 3): the other three corners stay round. The Phase
// 2 painter cannot mix one path corner with three elliptical corners, so all
// four corners go to OpPath and the other three K values stay 1. If a future
// painter learns to mix, the box would keep a radii fill for the three round
// corners plus a path for the shaped corner; this helper accepts that branch
// too and says which branch it took, so the test asserts whichever strategy
// the tree implements.
func cornerShapeASingleCornerCase(t *testing.T, prop string, wantIdx int, cornerName string) {
	t.Helper()

	res, boxOps := cornerShapeARequirePathSupport(t, prop)

	paths := cornerShapeAPathOps(boxOps)

	if cornerShapeAHasRadiiFill(boxOps) {
		t.Logf("%s: mixing painter (three round corners stay on radii fields)", prop)
		cornerShapeAAssertMixKeepsRoundRadii(t, prop, wantIdx, cornerName, boxOps)
	} else {
		t.Logf("%s: no-mix painter (all four corners go OpPath, other three K==1)", prop)
		cornerShapeAAssertNoRadiiFill(t, boxOps, prop)
	}

	cornerShapeAAssertDiffersFromAllBevel(t, prop, cornerName, paths[0])
	cornerShapeAAssertSingleCornerKs(t, prop, wantIdx, cornerName, boxByID(t, res, cornerShapeAItemID).style)
}

// cornerShapeAAssertMixKeepsRoundRadii checks the future-mixer branch: every
// unshaped corner keeps the 16px radii on the surviving fill.
func cornerShapeAAssertMixKeepsRoundRadii(
	t *testing.T, prop string, wantIdx int, cornerName string, boxOps []Op,
) {
	t.Helper()

	fill := cornerShapeAFindRedFill(boxOps)
	if fill == nil {
		t.Fatalf("%s: radii fill reported but red fill missing", prop)
	}

	radii := [4]float64{fill.RadiusTopLeft, fill.RadiusTopRight, fill.RadiusBottomRight, fill.RadiusBottomLeft}

	for idx, radius := range radii {
		if idx == wantIdx {
			continue
		}

		if !near(radius, pxToPt(16)) {
			t.Errorf("%s: unshaped corner %d radius = %.4fpt (%.2fpx), want 16px (only %s changes)",
				prop, idx, radius, radius/ptPerCSSPx, cornerName)
		}
	}
}

// cornerShapeAAssertDiffersFromAllBevel checks that the single-corner path
// differs from the all-corners bevel path: the other three corners are round
// (K=1) here but bevel (K=0) there. Equality would mean the single longhand
// rewrote every corner.
func cornerShapeAAssertDiffersFromAllBevel(t *testing.T, prop, cornerName string, pathOp Op) {
	t.Helper()

	allBevel := cornerShapeALayout(t, "corner-shape:bevel;")
	allBevelPaths := cornerShapeAPathOps(cornerShapeABoxOps(t, allBevel))

	if len(allBevelPaths) > 0 && cornerShapeAEqualIgnoringID(pathOp, allBevelPaths[0]) {
		t.Errorf("%s path equals all-corners bevel path (%s), want only %s to change",
			prop, cornerShapeAFormatPath(pathOp), cornerName)
	}
}

// cornerShapeAAssertSingleCornerKs checks the stored K values: the shaped
// corner is bevel (K=0) and the other three are round (K=1).
func cornerShapeAAssertSingleCornerKs(
	t *testing.T, prop string, wantIdx int, cornerName string, sty *ResolvedStyle,
) {
	t.Helper()

	kValues := cornerShapeAStyleKs(sty)

	for idx, kValue := range kValues {
		if idx == wantIdx {
			if kValue != 0 {
				t.Errorf("%s: corner %d K = %v, want 0 (bevel)", prop, idx, kValue)
			}
		} else if kValue != 1 {
			t.Errorf("%s: corner %d K = %v, want 1 (round; only %s changes)", prop, idx, kValue, cornerName)
		}
	}
}

// TestCornerShapeASingleTopLeftBevel is corner-top-left-shape:bevel: only the
// top-left corner changes. See cornerShapeASingleCornerCase for the mix rule.
func TestCornerShapeASingleTopLeftBevel(t *testing.T) {
	t.Parallel()

	cornerShapeASingleCornerCase(t, "corner-top-left-shape:bevel", 0, "top-left")
}

// TestCornerShapeASingleTopRightBevel is corner-top-right-shape:bevel: only
// the top-right corner changes, under the same mix rule as top-left.
func TestCornerShapeASingleTopRightBevel(t *testing.T) {
	t.Parallel()

	cornerShapeASingleCornerCase(t, "corner-top-right-shape:bevel", 1, "top-right")
}

// TestCornerShapeASingleBottomRightBevel is corner-bottom-right-shape:bevel:
// only the bottom-right corner changes, under the same mix rule as top-left.
func TestCornerShapeASingleBottomRightBevel(t *testing.T) {
	t.Parallel()

	cornerShapeASingleCornerCase(t, "corner-bottom-right-shape:bevel", 2, "bottom-right")
}

// TestCornerShapeASingleBottomLeftBevel is corner-bottom-left-shape:bevel:
// only the bottom-left corner changes, under the same mix rule as top-left.
func TestCornerShapeASingleBottomLeftBevel(t *testing.T) {
	t.Parallel()

	cornerShapeASingleCornerCase(t, "corner-bottom-left-shape:bevel", 3, "bottom-left")
}
