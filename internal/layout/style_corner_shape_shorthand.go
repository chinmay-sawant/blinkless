package layout

import (
	"strings"
)

// Four-corner and top-level corner shorthands: checklist Phase 1 rows
// `corner-shape` and `corner`.
//
// `corner-shape` takes one to four <corner-shape-value> tokens. The four
// values assign to the top, right, bottom, and left SIDES, not to corners
// in border-radius order (CSS Borders 4 section 3.8.2). A missing left
// copies right, a missing bottom copies top, and a missing right copies
// top. One value sets all four corners. A side value sets both corners on
// that side.
//
// `corner` takes a radius piece, a shape piece, or both in any order
// (<'border-radius'> || <'corner-shape'>). It sets all four radii and all
// four shapes. A missing piece resets that piece to the initial value
// (zero radius, round shape), the same way a border shorthand resets
// omitted parts. The radius piece goes through the existing setter; no
// second radius store lives here.
//
// Shape parsing and storage are style_corner_shape.go's: parseCornerShapeValue
// maps round to 1, squircle to 2, square to +Inf, bevel to 0, scoop to -1,
// notch to -Inf, and superellipse() to its number (infinity/-infinity at
// the ends). SetCornerShapeByName writes one corner's K plus its keyword.
// This file only expands sides and splits the `corner` pieces, then calls
// those.
//
// An invalid value owns its property name but stores nothing, so the
// previous declaration stays. Validation runs before any write: a bad
// token anywhere rejects the whole declaration, never half of it.
func applyCornerShapeShorthand(style *ResolvedStyle, prop, value string, fsize float64) bool {
	switch prop {
	case propCornerShape:
		applyCornerShapeSides(style, value)
	case propCornerShorthand:
		applyCornerTopLevel(style, value, fsize)
	default:
		return false
	}

	return true
}

// cornerShapeSide is one parsed side value: the superellipse K plus the
// canonical keyword for the declared-form tests.
type cornerShapeSide struct {
	k       float64
	keyword string
}

// applyCornerShapeSides stores one `corner-shape` declaration. Anything
// but one to four valid shape tokens stores nothing.
func applyCornerShapeSides(style *ResolvedStyle, value string) {
	sides, ok := expandCornerShapeSides(value)
	if !ok {
		return
	}

	writeCornerShapeSides(style, sides)
}

// writeCornerShapeSides assigns the four side values to corners in side
// order top, right, bottom, left. Each side writes both of its corners:
// top writes top-left and top-right, right writes top-right and
// bottom-right, bottom writes bottom-right and bottom-left, left writes
// bottom-left and top-left. A corner named by two sides keeps the side
// later in top, right, bottom, left order, so the final corners read
// top-left = left, top-right = right, bottom-right = bottom,
// bottom-left = left. That is not the border-radius corner order, where
// four values land directly on top-left, top-right, bottom-right,
// bottom-left. The checklist records this tie-break because the draft
// paragraph does not spell it out.
func writeCornerShapeSides(style *ResolvedStyle, sides [cornerShapeQuadValues]cornerShapeSide) {
	top, right, bottom, left := sides[0], sides[1], sides[2], sides[3]

	SetCornerShapeByName(style, cornerNameTopLeft, top.k, top.keyword)
	SetCornerShapeByName(style, cornerNameTopRight, top.k, top.keyword)

	SetCornerShapeByName(style, cornerNameTopRight, right.k, right.keyword)
	SetCornerShapeByName(style, cornerNameBottomRight, right.k, right.keyword)

	SetCornerShapeByName(style, cornerNameBottomRight, bottom.k, bottom.keyword)
	SetCornerShapeByName(style, cornerNameBottomLeft, bottom.k, bottom.keyword)

	SetCornerShapeByName(style, cornerNameBottomLeft, left.k, left.keyword)
	SetCornerShapeByName(style, cornerNameTopLeft, left.k, left.keyword)
}

// expandCornerShapeSides parses one `corner-shape` value into its four
// side values in top, right, bottom, left order. Missing sides copy per
// the draft: one value sets all four, two values set top then right with
// bottom copying top and left copying right, three values set top, right,
// bottom with left copying right. Tokens split paren-aware (nextSpaceToken
// keeps superellipse() with inner spaces whole). ok is false for zero
// tokens, more than four, or any token that is not a shape value.
func expandCornerShapeSides(value string) ([cornerShapeQuadValues]cornerShapeSide, bool) {
	var toks []string

	for start := 0; ; {
		token, next, has := nextSpaceToken(value, start)
		if !has {
			break
		}

		toks = append(toks, token)
		start = next
	}

	return expandCornerShapeSideTokens(toks)
}

// expandCornerShapeSideTokens parses already-split shape tokens into the
// four side values. One token sets all four sides; otherwise the missing
// sides copy per expandCornerShapeSides. ok is false unless one to four
// tokens are present and every token parses as a shape value.
func expandCornerShapeSideTokens(toks []string) ([cornerShapeQuadValues]cornerShapeSide, bool) {
	var empty [cornerShapeQuadValues]cornerShapeSide

	if len(toks) == 0 || len(toks) > cornerShapeQuadValues {
		return empty, false
	}

	parsed := make([]cornerShapeSide, 0, len(toks))

	for _, token := range toks {
		kValue, keyword, valid := parseCornerShapeValue(token)
		if !valid {
			return empty, false
		}

		parsed = append(parsed, cornerShapeSide{k: kValue, keyword: keyword})
	}

	sides := [cornerShapeQuadValues]cornerShapeSide{parsed[0], parsed[0], parsed[0], parsed[0]}

	if len(parsed) > 1 {
		sides[1], sides[3] = parsed[1], parsed[1]
	}

	if len(parsed) > cornerShapePairValues {
		sides[cornerShapePairValues] = parsed[cornerShapePairValues]
	}

	if len(parsed) > cornerShapeTripleValues {
		sides[cornerShapeTripleValues] = parsed[cornerShapeTripleValues]
	}

	return sides, true
}

// applyCornerTopLevel stores one `corner` declaration: a radius piece, a
// shape piece of one to four tokens, or both. Either piece missing resets
// that piece to the initial value on all four corners. A token that is
// neither a shape value nor part of a valid radius rejects the whole
// declaration, as do more than four shape tokens.
func applyCornerTopLevel(style *ResolvedStyle, value string, fsize float64) {
	radiusPiece, shapeToks, ok := splitCornerTopLevelPieces(value)
	if !ok || len(shapeToks) > cornerShapeQuadValues {
		return
	}

	sides, shapesOK := expandCornerShapeSideTokens(shapeToks)
	if len(shapeToks) > 0 && !shapesOK {
		return
	}

	if radiusPiece != "" && !borderRadiusValueAccepted(radiusPiece) {
		return
	}

	if radiusPiece == "" {
		setBorderRadius(style, "0", fsize)
	} else {
		setBorderRadius(style, radiusPiece, fsize)
	}

	if len(shapeToks) == 0 {
		resetCornerShapeRound(style)
	} else {
		writeCornerShapeSides(style, sides)
	}
}

// resetCornerShapeRound restores the initial shape on all four corners:
// round, K = 1.
func resetCornerShapeRound(style *ResolvedStyle) {
	SetCornerShapeByName(style, cornerNameTopLeft, CornerShapeRoundK, cornerShapeKeywordRound)
	SetCornerShapeByName(style, cornerNameTopRight, CornerShapeRoundK, cornerShapeKeywordRound)
	SetCornerShapeByName(style, cornerNameBottomRight, CornerShapeRoundK, cornerShapeKeywordRound)
	SetCornerShapeByName(style, cornerNameBottomLeft, CornerShapeRoundK, cornerShapeKeywordRound)
}

// splitCornerTopLevelPieces partitions a `corner` value into its radius
// piece (rejoined with single spaces, original order kept) and its shape
// tokens. Every token lands in exactly one piece: tokens that parse as a
// shape value go to the shape piece, everything else, including "/" and
// junk, stays in the radius piece so radius validation rejects it as a
// whole. ok is false when the value holds no token at all.
func splitCornerTopLevelPieces(value string) (string, []string, bool) {
	var radiusToks, shapeToks []string

	for start := 0; ; {
		token, next, has := nextSpaceToken(value, start)
		if !has {
			break
		}

		if _, _, valid := parseCornerShapeValue(token); valid {
			shapeToks = append(shapeToks, token)
		} else {
			radiusToks = append(radiusToks, token)
		}

		start = next
	}

	if len(radiusToks) == 0 && len(shapeToks) == 0 {
		return "", nil, false
	}

	return strings.Join(radiusToks, " "), shapeToks, true
}

// cornerShapeSidesValueAccepted mirrors expandCornerShapeSides for the
// cascade gate (supportedDeclaration) and the @supports probe, so a bad
// token never wins the cascade over an earlier valid declaration.
func cornerShapeSidesValueAccepted(value string) bool {
	_, ok := expandCornerShapeSides(value)

	return ok
}

// cornerTopLevelValueAccepted mirrors applyCornerTopLevel for the cascade
// gate and the @supports probe.
func cornerTopLevelValueAccepted(value string) bool {
	radiusPiece, shapeToks, ok := splitCornerTopLevelPieces(value)
	if !ok || len(shapeToks) > cornerShapeQuadValues {
		return false
	}

	if _, shapesOK := expandCornerShapeSideTokens(shapeToks); len(shapeToks) > 0 && !shapesOK {
		return false
	}

	if radiusPiece == "" {
		return true
	}

	return borderRadiusValueAccepted(radiusPiece)
}
