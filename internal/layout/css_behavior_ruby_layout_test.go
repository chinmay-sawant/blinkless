package layout

import (
	"strings"
	"testing"
)

// Ruby layout behavior tests: every property is asserted through USED
// annotation geometry (text op positions and sizes in points), never through
// stored style strings. Reference browser for every case:
// Chrome 143.0.7499.40. Spec: https://drafts.csswg.org/css-ruby-1/
//
// Shared helpers (layoutHTML, opsOfKind, near) come from other files in this
// same package and are reused here, never redeclared.
//
// Documented approximations: inter-character paints inline at half size
// instead of beside the base; ruby-merge:merge concatenates a whole <ruby>
// into one group segment; a segment may split across lines; rp fallback text
// is dropped; annotations nested below a direct rt/rtc child stay flat.

// rubyLayoutTextOp returns the first text op containing substr.
func rubyLayoutTextOp(t *testing.T, res *Result, substr string) Op {
	t.Helper()

	for _, op := range opsOfKind(res, OpText) {
		if strings.Contains(op.Text, substr) {
			return op
		}
	}

	t.Fatalf("no text op containing %q (ops %q)", substr, rubyLayoutOpTexts(res))

	return Op{}
}

// rubyLayoutOpTexts joins painted texts for failure messages.
func rubyLayoutOpTexts(res *Result) string {
	var out strings.Builder

	for _, op := range opsOfKind(res, OpText) {
		out.WriteString(op.Text)
		out.WriteString("|")
	}

	return out.String()
}

// TestBehaviorRubyLayoutStacksAnnotationOverBase: the default ruby-position
// (alternate, first level behaves as over) stacks a half-size annotation
// above its base. Chrome paints <rt> over the base by default.
func TestBehaviorRubyLayoutStacksAnnotationOverBase(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt"><ruby>base<rt>anno</rt></ruby></p></body></html>`)

	base, annot := rubyLayoutTextOp(t, res, "base"), rubyLayoutTextOp(t, res, "anno")

	if !(annot.Size < base.Size) {
		t.Errorf("annotation size %.4fpt should be below base size %.4fpt", annot.Size, base.Size)
	}

	if !near(annot.Size, base.Size*rubyAnnotationScale) {
		t.Errorf("annotation size %.4fpt vs base %.4fpt, want half-size ratio %.2f",
			annot.Size, base.Size, rubyAnnotationScale)
	}

	if !(annot.Y < base.Y-1) {
		t.Errorf("annotation Y %.4fpt should sit above base Y %.4fpt", annot.Y, base.Y)
	}
}

// TestBehaviorRubyLayoutPositionUnderStacksBelow: ruby-position:under on the ruby
// container stacks the annotation below the base. Chrome paints <rt> under
// the base for this value.
func TestBehaviorRubyLayoutPositionUnderStacksBelow(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt"><ruby style="ruby-position:under">base<rt>anno</rt></ruby></p>`+
		`</body></html>`)

	base, annot := rubyLayoutTextOp(t, res, "base"), rubyLayoutTextOp(t, res, "anno")

	if !(annot.Y > base.Y+1) {
		t.Errorf("annotation Y %.4fpt should sit below base Y %.4fpt", annot.Y, base.Y)
	}

	if !(annot.Size < base.Size) {
		t.Errorf("annotation size %.4fpt should stay below base size %.4fpt", annot.Size, base.Size)
	}
}

// TestBehaviorRubyLayoutAlignStartVsCenter: a narrow annotation pins to the base start
// edge under ruby-align:start but centers under ruby-align:center. Chrome
// centers short kana over kanji by default.
func TestBehaviorRubyLayoutAlignStartVsCenter(t *testing.T) {
	t.Parallel()

	start := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt"><ruby style="ruby-align:start">base<rt>x</rt></ruby></p>`+
		`</body></html>`)
	center := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt"><ruby style="ruby-align:center">base<rt>x</rt></ruby></p>`+
		`</body></html>`)

	startBase, startAnnot := rubyLayoutTextOp(t, start, "base"), rubyLayoutTextOp(t, start, "x")
	centerBase, centerAnnot := rubyLayoutTextOp(t, center, "base"), rubyLayoutTextOp(t, center, "x")

	if !near(startBase.X, centerBase.X) {
		t.Errorf("base X = %.4fpt vs %.4fpt, want identical line start", startBase.X, centerBase.X)
	}

	if !near(startAnnot.X, startBase.X) {
		t.Errorf("start annotation X %.4fpt should pin to base X %.4fpt", startAnnot.X, startBase.X)
	}

	if !(centerAnnot.X > startAnnot.X+0.5) {
		t.Errorf("center annotation X %.4fpt should sit right of start X %.4fpt",
			centerAnnot.X, startAnnot.X)
	}
}

// TestBehaviorRubyLayoutOverhangSpacesWidensSegment: a wide annotation spills over
// its follower under ruby-overhang:auto but contains the follower under
// spaces (legacy none). Chrome lets long kana spill by default.
func TestBehaviorRubyLayoutOverhangSpacesWidensSegment(t *testing.T) {
	t.Parallel()

	auto := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt"><ruby>B<rt>wideanno</rt></ruby>`+
		`<span style="font-weight:700">Y</span></p></body></html>`)
	spaces := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt"><ruby style="ruby-overhang:spaces">B<rt>wideanno</rt></ruby>`+
		`<span style="font-weight:700">Y</span></p></body></html>`)

	autoMark, spacesMark := rubyLayoutTextOp(t, auto, "Y"), rubyLayoutTextOp(t, spaces, "Y")
	autoAnnot := rubyLayoutTextOp(t, auto, "wideanno")
	autoBase := rubyLayoutTextOp(t, auto, "B")

	if !(autoAnnot.W > autoBase.W) {
		t.Errorf("annotation width %.4fpt should exceed base width %.4fpt", autoAnnot.W, autoBase.W)
	}

	if !(spacesMark.X > autoMark.X+1) {
		t.Errorf("spaces marker X %.4fpt should sit right of auto marker X %.4fpt",
			spacesMark.X, autoMark.X)
	}
}

// TestBehaviorRubyLayoutMergeNarrowsSharedSpace: with two columns whose first
// annotation is wider than its base, separate sizes each column to its widest
// member while merge fits the wider of the summed runs, so the follower sits
// further left under merge. Chrome shares annotation space for group ruby.
func TestBehaviorRubyLayoutMergeNarrowsSharedSpace(t *testing.T) {
	t.Parallel()

	separate := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">`+
		`<ruby style="ruby-merge:separate;ruby-overhang:spaces">A<rt>wide</rt>B<rt>x</rt></ruby>`+
		`<span style="font-weight:700">Y</span></p></body></html>`)
	merged := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt">`+
		`<ruby style="ruby-merge:merge;ruby-overhang:spaces">A<rt>wide</rt>B<rt>x</rt></ruby>`+
		`<span style="font-weight:700">Y</span></p></body></html>`)

	separateMark, mergedMark := rubyLayoutTextOp(t, separate, "Y"), rubyLayoutTextOp(t, merged, "Y")

	if !(separateMark.X > mergedMark.X+1) {
		t.Errorf("separate marker X %.4fpt should sit right of merged marker X %.4fpt",
			separateMark.X, mergedMark.X)
	}
}
