package layout

// Motion path consumer. The cascade stores the six properties as text (see
// declared_motion.go). Once the box has a position, withMotionPath turns them
// into one matrix and folds it into the box transform, so the display list
// moves the element the same way an element with `transform` moves.
//
// Reference box: a box carries no pointer to its containing block, so path
// coordinates, a ray start, and the position keywords resolve against the
// element's own border box, and the path origin is the box's top-left corner.
// A browser uses the containing block there. Order of use: offset first, then
// the transform the element already has.

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

	boxW, boxH := boxNode.w, boxNode.height

	var startX, startY float64
	if spec.posSet {
		startX, startY = spec.pos.x.at(boxW), spec.pos.y.at(boxH)
	}

	path, ok := spec.shape.buildAt(boxW, boxH, sty.FontSize, startX, startY)
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

	abs := Translate(boxNode.x+pathX, boxNode.y+pathY).Mul(RotateDeg(angle)).Mul(Translate(-anchorX, -anchorY))

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
