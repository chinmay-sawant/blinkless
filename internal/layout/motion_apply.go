package layout

// Motion path consumer. The cascade stores the six properties as text (see
// declared_motion.go). Once the box has a position, withMotionPath turns them
// into one matrix and folds it into the box transform, so the display list
// moves the element the same way an element with `transform` moves.
//
// Reference box: path coordinates start at the top-left corner of the
// containing block (motionReference), and offset-position and the shape
// functions resolve against its size. offset-anchor still resolves against the
// element's own border box.
// A box no stamp pass has reached has no parent yet and uses its own top-left
// corner. Order of use: offset first, then the transform the element already
// has.

// effectiveMotion merges the shorthand and the longhands. A longhand wins over
// the shorthand whatever their source order: the cascade keeps the text, not
// the order.
func effectiveMotion(sty *ResolvedStyle, fsize float64) motionSpec {
	var spec motionSpec

	if text := sty.Declared.Offset; text != "" {
		spec, _ = parseOffsetShorthand(text, fsize)
	}

	overlayMotionPath(&spec, &sty.Declared, fsize)
	overlayMotionPosition(&spec, &sty.Declared, fsize)

	return spec
}

// overlayMotionPath lets the path, distance and rotate longhands override spec.
func overlayMotionPath(spec *motionSpec, declared *declaredStyle, fsize float64) {
	if text := declared.OffsetPath; text != "" {
		shape, none, ok := parseMotionPathTokens(splitTop(text))
		if ok {
			spec.hasPath, spec.shape, spec.pathCleard = !none, shape, none
		}
	}

	if text := declared.OffsetDistance; text != "" {
		if length, ok := parseMotionLen(text, fsize); ok {
			spec.dist, spec.distSet = length, true
		}
	}

	if text := declared.OffsetRotate; text != "" {
		if rot, ok := parseMotionRotate(splitTop(text)); ok {
			spec.rot, spec.rotSet = rot, true
		}
	}
}

// overlayMotionPosition lets the position and anchor longhands override spec.
func overlayMotionPosition(spec *motionSpec, declared *declaredStyle, fsize float64) {
	if text := declared.OffsetPosition; text != "" {
		if pos, set, ok := parseMotionPositionValue(splitTop(text), fsize); ok {
			spec.pos, spec.posSet = pos, set
		}
	}

	if text := declared.OffsetAnchor; text != "" {
		if pos, set, ok := parseAnchorValue(splitTop(text), fsize); ok {
			spec.anchor, spec.anchorSet = pos, set
		}
	}
}

// motionActive reports whether the style declares an offset path.
func motionActive(sty *ResolvedStyle) bool {
	if sty.Declared.OffsetPath == "" && sty.Declared.Offset == "" {
		return false
	}

	return effectiveMotion(sty, motionNormFontSize).hasPath
}

// motionAfterDeclared turns the box transform on when an offset path is
// declared, so the stamp pass visits the box. It runs after each declared
// property is stored.
func motionAfterDeclared(style *ResolvedStyle, prop string) {
	if declaredGroupOf(prop) == declaredGroupMotion && motionActive(style) {
		style.HasTransform = true
	}
}

// motionHeading is the rotation, in degrees, of the element at a point whose
// path heading is given.
func motionHeading(rot motionRotate, rotSet bool, heading float64) float64 {
	if !rotSet {
		return heading
	}

	deg := rot.angle
	if rot.auto {
		deg += heading
		if rot.reverse {
			deg += degPerHalfTurn
		}
	}

	return deg
}

// motionLocalMatrix returns the offset transform of a box, built so that
// BakeOrigin around the box's transform-origin gives T(P) R(a) T(-A): the
// anchor A lands on path point P and turns by the heading.
func motionLocalMatrix(sty *ResolvedStyle, boxNode *box) (Matrix2D, bool) {
	if sty == nil || boxNode == nil || !motionActive(sty) {
		return IdentityMatrix(), false
	}

	spec := effectiveMotion(sty, sty.FontSize)
	if !spec.hasPath {
		return IdentityMatrix(), false
	}

	refX, refY, refW, refH := motionReference(boxNode)
	boxW, boxH := boxNode.w, boxNode.height

	var startX, startY float64
	if spec.posSet {
		startX, startY = spec.pos.x.at(refW), spec.pos.y.at(refH)
	}

	path, ok := spec.shape.buildAt(refW, refH, sty.FontSize, startX, startY)
	if !ok {
		return IdentityMatrix(), false
	}

	var dist float64
	if spec.distSet {
		dist = spec.dist.resolve(path.length())
	}

	pathX, pathY, heading := path.at(dist)
	angle := motionHeading(spec.rot, spec.rotSet, heading)

	originX, originY := resolveBorderBoxOrigin(sty.TransformOrigin, boxNode)
	anchorX, anchorY := originX, originY

	if spec.anchorSet {
		anchorX, anchorY = boxNode.x+spec.anchor.x.at(boxW), boxNode.y+spec.anchor.y.at(boxH)
	}

	abs := Translate(refX+pathX, refY+pathY).Mul(RotateDeg(angle)).Mul(Translate(-anchorX, -anchorY))

	return Translate(-originX, -originY).Mul(abs).Mul(Translate(originX, originY)), true
}

// withMotionPath folds the offset transform into tform.
func withMotionPath(sty *ResolvedStyle, boxNode *box, tform Matrix2D) Matrix2D {
	if sty.Declared.OffsetPath == "" && sty.Declared.Offset == "" {
		return tform
	}

	if m, ok := motionLocalMatrix(sty, boxNode); ok {
		return m.Mul(tform)
	}

	return tform
}

// motionReference returns the containing block that path coordinates and
// offset-position resolve against: its top-left corner and its size, in canvas
// points. An absolute or fixed box uses the padding box of its nearest
// positioned ancestor. Any other box uses the content box of its parent.
// Without a parent, or without a positioned ancestor for an absolute box, the
// box's own border box stands in.
func motionReference(boxNode *box) (float64, float64, float64, float64) {
	own := func() (float64, float64, float64, float64) { return boxNode.x, boxNode.y, boxNode.w, boxNode.height }

	parent := boxNode.motionParent
	if parent == nil {
		return own()
	}

	if sty := boxNode.style; sty != nil && (sty.Position == positionAbsolute || sty.Position == positionFixed) {
		for anc := parent; anc != nil; anc = anc.motionParent {
			if anc.style != nil && anc.style.Position != "" && anc.style.Position != positionStatic {
				return motionInset(anc, 0)
			}
		}

		return own()
	}

	return motionInset(parent, 1)
}

// motionInset returns the padding box (withPadding 0) or content box
// (withPadding 1) of one box.
func motionInset(anc *box, withPadding float64) (float64, float64, float64, float64) {
	sty := anc.style
	if sty == nil {
		return anc.x, anc.y, anc.w, anc.height
	}

	left := sty.BorderLeft.Width + withPadding*sty.PaddingLeft
	top := sty.BorderTop.Width + withPadding*sty.PaddingTop
	right := sty.BorderRight.Width + withPadding*sty.PaddingRight
	bottom := sty.BorderBottom.Width + withPadding*sty.PaddingBottom

	return anc.x + left, anc.y + top, max(anc.w-left-right, 0), max(anc.height-top-bottom, 0)
}
