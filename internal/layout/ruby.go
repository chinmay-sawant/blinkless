// Ruby annotation geometry helpers (CSS Ruby 1, draft).
//
// The engine flattens ruby markup today: <ruby> base text and <rt>
// annotation text land in one inline text run on one baseline (see the gap
// test in css_behavior_ruby_test.go). Full interlinear layout needs a ruby
// formatting context in the inline collector plus cascade storage for the
// four properties below, neither of which exists yet.
//
// This file therefore holds only pure geometry: given measured base and
// annotation widths and heights in points, where does the annotation go?
// Each helper takes the already-parsed used value (never a stored string)
// and returns coordinates in points. Reference browser for every case:
// Chrome 143.0.7499.40. Spec: https://drafts.csswg.org/css-ruby-1/
//
// Used values and initials used here:
//
//	ruby-align: start | center | space-between | space-around, initial space-around
//	ruby-merge: separate | merge | auto, initial separate
//	ruby-overhang: auto | spaces (legacy alias none), initial auto
//	ruby-position: [ alternate || [ over | under ] ] | inter-character, initial alternate
package layout

import "strings"

// Canonical ruby used values kept here so geometry helpers and tests share
// one spelling. Parsing itself is not wired to the cascade yet (see the seam
// note on rubyCascadeSeam).
const (
	rubyAlignStart        = "start"
	rubyAlignCenter       = "center"
	rubyAlignSpaceBetween = "space-between"
	rubyAlignSpaceAround  = "space-around"

	rubyMergeSeparate = "separate"
	rubyMergeMerge    = "merge"
	rubyMergeAuto     = "auto"

	rubyOverhangAuto   = "auto"
	rubyOverhangSpaces = "spaces"

	rubyPositionOver           = "over"
	rubyPositionUnder          = "under"
	rubyPositionAlternate      = "alternate"
	rubyPositionInterCharacter = "inter-character"
)

// Integration seam (kept as a comment: the declarations reach
// applySVGPresentationProps (internal/layout/style_paint_props.go:94-96) and
// fall into applyLeftoversProps (internal/layout/style_leftovers.go:34-35)
// which returns false, so the cascade drops them). Wiring needs a storage
// choice (CustomProps avoids regenerating style_intern_gen.go) plus a new
// style group registered in styleGroups
// (internal/layout/style_cascade.go:1490-1522), plus annotation stacking in
// the inline collector (internal/layout/inline_collect.go:191-202) and
// emission (internal/layout/inline.go:1191).

// parseRubyAlign normalizes a ruby-align value to its canonical used value.
// Unknown values return false so the caller keeps the initial space-around.
func parseRubyAlign(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case rubyAlignStart:
		return rubyAlignStart, true
	case rubyAlignCenter:
		return rubyAlignCenter, true
	case rubyAlignSpaceBetween:
		return rubyAlignSpaceBetween, true
	case rubyAlignSpaceAround:
		return rubyAlignSpaceAround, true
	default:
		return "", false
	}
}

// parseRubyMerge normalizes a ruby-merge value. Only the spec trio is
// accepted; anything else (including the old draft word collapse) returns
// false so the caller keeps the initial separate.
func parseRubyMerge(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case rubyMergeSeparate:
		return rubyMergeSeparate, true
	case rubyMergeMerge:
		return rubyMergeMerge, true
	case rubyMergeAuto:
		return rubyMergeAuto, true
	default:
		return "", false
	}
}

// parseRubyOverhang normalizes a ruby-overhang value. The legacy alias none
// maps to spaces per the spec; unknown values return false (initial auto).
func parseRubyOverhang(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case rubyOverhangAuto:
		return rubyOverhangAuto, true
	case rubyOverhangSpaces:
		return rubyOverhangSpaces, true
	case "none":
		return rubyOverhangSpaces, true
	default:
		return "", false
	}
}

// parseRubyPosition normalizes a ruby-position value to one of over, under,
// alternate, or inter-character. The grammar is
// [ alternate || [ over | under ] ] | inter-character; inter-character is
// exclusive, alternate may combine with at most one of over/under. Unknown
// values return false (initial alternate).
func parseRubyPosition(value string) (string, bool) {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(value)))
	if len(fields) == 0 || len(fields) > 2 {
		return "", false
	}

	if len(fields) == 1 {
		return parseRubyPositionSingle(fields[0])
	}

	return parseRubyPositionPair(fields)
}

// parseRubyPositionSingle normalizes a lone position keyword.
func parseRubyPositionSingle(field string) (string, bool) {
	switch field {
	case rubyPositionOver:
		return rubyPositionOver, true
	case rubyPositionUnder:
		return rubyPositionUnder, true
	case rubyPositionAlternate:
		return rubyPositionAlternate, true
	case rubyPositionInterCharacter:
		return rubyPositionInterCharacter, true
	default:
		return "", false
	}
}

// parseRubyPositionPair normalizes "alternate over|under" in either order,
// rejecting repeats and bare side pairs.
func parseRubyPositionPair(fields []string) (string, bool) {
	seenAlternate, seenSide := rubyPositionPairFlags(fields)

	if !seenAlternate || !seenSide {
		return "", false
	}

	for _, field := range fields {
		if field == rubyPositionUnder {
			return rubyPositionUnder, true
		}
	}

	return rubyPositionOver, true
}

// rubyPositionPairFlags scans a two-field position for the alternate marker
// and a side marker, rejecting repeats and unknown words.
func rubyPositionPairFlags(fields []string) (bool, bool) {
	seenAlternate, seenSide := false, false

	for _, field := range fields {
		switch field {
		case rubyPositionAlternate:
			if seenAlternate {
				return false, false
			}

			seenAlternate = true
		case rubyPositionOver, rubyPositionUnder:
			if seenSide {
				return false, false
			}

			seenSide = true
		default:
			return false, false
		}
	}

	return seenAlternate, seenSide
}

// rubyPositionIsUnder reports whether an interlinear annotation stacks below
// the base. Alternate alone behaves as over for the first level, so only a
// used value carrying under counts. Inter-character annotations sit to the
// side, never below.
func rubyPositionIsUnder(position string) bool {
	return strings.ToLower(strings.TrimSpace(position)) == rubyPositionUnder
}

// rubyPositionIsInterCharacter reports side placement (Taiwanese bopomofo
// style): the annotation sits to the right of the base in horizontal text,
// not stacked above or below.
func rubyPositionIsInterCharacter(position string) bool {
	return strings.ToLower(strings.TrimSpace(position)) == rubyPositionInterCharacter
}

// rubyCenterDivisor splits the leftover space evenly on both sides.
const rubyCenterDivisor = 2

// rubyAlignOffset returns the inline-axis offset of an annotation within its
// ruby column, in points. Column width is max(baseW, annotW) under separate
// sizing; the narrower side gets extra space distributed by ruby-align.
// When the annotation fills or overflows the column there is no extra space
// and every value returns 0. Start pins to the start edge; center and the two
// distribute values center a lone narrow annotation (for CJK text the
// distribute values would justify between characters, but a single short
// Latin annotation has no justification opportunity and centers, per the
// spec note on text-justify: ruby).
func rubyAlignOffset(baseW, annotW float64, align string) float64 {
	if annotW >= baseW {
		return 0
	}

	switch strings.ToLower(strings.TrimSpace(align)) {
	case rubyAlignStart:
		return 0
	default:
		return (baseW - annotW) / rubyCenterDivisor
	}
}

// rubyAnnotationStackOffset returns the block-axis offset from the base
// container top edge to the annotation container top edge, in points.
// Containers stack outward with no intervening space: over/alternate sits
// directly above (-annotH), under sits directly below (+baseH),
// inter-character sits beside the base (0 vertical offset).
func rubyAnnotationStackOffset(baseH, annotH float64, position string) float64 {
	pos := strings.ToLower(strings.TrimSpace(position))
	if pos == rubyPositionInterCharacter {
		return 0
	}

	if pos == rubyPositionUnder {
		return baseH
	}

	return -annotH
}

// rubyOverhangExpand returns how much wider (in points) a ruby segment must
// grow to avoid overlapping adjacent text. Auto allows overhang so nothing
// grows; spaces forbids it so the base side grows to fit a wider annotation.
func rubyOverhangExpand(baseW, annotW float64, overhang string) float64 {
	if strings.ToLower(strings.TrimSpace(overhang)) == rubyOverhangAuto {
		return 0
	}

	if annotW > baseW {
		return annotW - baseW
	}

	return 0
}

// rubyMergeSegmentWidth returns the total inline size of one ruby segment in
// points. Separate sizes each column to its widest member and sums them
// (mono ruby). Merge concatenates all annotations into one span, so the
// segment fits the wider of the whole base run and the whole annotation run
// (group ruby). Auto renders as separate when every annotation fits its base
// and as merge otherwise (the simplest jukugo algorithm named by the spec).
func rubyMergeSegmentWidth(baseWs, annotWs []float64, merge string) float64 {
	switch strings.ToLower(strings.TrimSpace(merge)) {
	case rubyMergeMerge:
		return rubyMergedSpanWidth(baseWs, annotWs)
	case rubyMergeAuto:
		if rubyAnnotationsFitBases(baseWs, annotWs) {
			return rubySeparateSegmentWidth(baseWs, annotWs)
		}

		return rubyMergedSpanWidth(baseWs, annotWs)
	default:
		return rubySeparateSegmentWidth(baseWs, annotWs)
	}
}

// rubySeparateSegmentWidth sums per-column maxima.
func rubySeparateSegmentWidth(baseWs, annotWs []float64) float64 {
	total := 0.0
	count := len(baseWs)

	if len(annotWs) < count {
		count = len(annotWs)
	}

	for i := range count {
		if annotWs[i] > baseWs[i] {
			total += annotWs[i]
		} else {
			total += baseWs[i]
		}
	}

	return total
}

// rubyMergedSpanWidth fits the wider of the summed runs.
func rubyMergedSpanWidth(baseWs, annotWs []float64) float64 {
	baseTotal, annotTotal := 0.0, 0.0

	for _, w := range baseWs {
		baseTotal += w
	}

	for _, w := range annotWs {
		annotTotal += w
	}

	if annotTotal > baseTotal {
		return annotTotal
	}

	return baseTotal
}

// rubyAnnotationsFitBases reports whether every annotation fits its base.
func rubyAnnotationsFitBases(baseWs, annotWs []float64) bool {
	count := len(baseWs)

	if len(annotWs) < count {
		count = len(annotWs)
	}

	for i := range count {
		if annotWs[i] > baseWs[i] {
			return false
		}
	}

	return true
}
