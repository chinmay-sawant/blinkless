package layout

// worldTransform3DForBox returns one box's transform in canvas coordinates.
// Preserve-3d chains use these world-space matrices so each box keeps its own
// transform origin while the parent and child matrices compose.
func worldTransform3DForBox(boxNode *box, viewport svgViewport) matrix3D {
	if boxNode == nil || boxNode.style == nil {
		return identity3D()
	}

	style := boxNode.style
	local2D := IdentityMatrix()
	if style.HasTransform {
		local2D = withPercentTranslate(style, boxNode)
	}
	local3D := identity3D()
	if style.HasTransform3D {
		local3D = baked3DForBox(boxNode, viewport)
	}

	originX, originY := resolveTransformOriginWithBox(
		style.TransformOrigin, boxNode, style.TransformBox, viewport,
	)
	local := matrix3DFromAffine(local2D).mul(local3D)
	return translate3D(originX, originY, 0).mul(local).mul(translate3D(-originX, -originY, 0))
}

func matrix3DFromAffine(matrix Matrix2D) matrix3D {
	return matrix3D{
		matrix.A, matrix.C, 0, matrix.E,
		matrix.B, matrix.D, 0, matrix.F,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// stampPreserveTransformRange applies the accumulated preserve-3d projection
// to either a box's exclusive operations or its complete subtree. A flat
// boundary owns the latter because it flattens its descendants onto one plane.
func stampPreserveTransformRange(
	boxNode *box, worldChain matrix3D, parentAccum Matrix2D, ops []Op,
	covered []bool, exclusive bool, viewport svgViewport,
) {
	if boxNode == nil || boxNode.style == nil || !boxOwnsOps(boxNode) || len(ops) == 0 {
		return
	}

	originX, originY := resolveTransformOriginWithBox(
		boxNode.style.TransformOrigin, boxNode, boxNode.style.TransformBox, viewport,
	)
	dist, vanishX, vanishY := perspectiveFlattenArgs(boxNode, worldChain, originX, originY)
	start, end := boxNode.opStart, min(boxNode.opEnd, len(ops)-1)
	if start < 0 || start > end {
		return
	}

	var projection ProjectiveTransform
	var affine Matrix2D
	if dist > 0 {
		projection = projectiveTransform3DWorld(worldChain, vanishX, vanishY, dist)
		projection = composeProjectiveAffine(projection, parentAccum)
	} else {
		affine = parentAccum.Mul(matrix2DFromWorld3D(worldChain))
	}

	for idx := start; idx <= end; idx++ {
		if exclusive && covered[idx] {
			continue
		}

		op := &ops[idx]
		if op.Kind == OpKindNoop || op.Kind == OpUnknown {
			continue
		}

		if dist > 0 {
			if op.Kind == OpImage {
				mapping := projection
				if !exclusive {
					mapping = composeProjectiveInputAffine(mapping, op.Transform())
				}
				setPreserveImageTransform(op, mapping)

				continue
			}

			if !exclusive {
				if inner, ok := op.ProjectiveTransformValue(); ok {
					projectionForOp := composeProjectiveTransforms(projection, inner)
					op.setProjectiveTransform(projectionForOp)
					op.XformSet = false

					continue
				}
				if op.XformSet {
					projectionForOp := composeProjectiveInputAffine(projection, op.Transform())
					op.setProjectiveTransform(projectionForOp)
					op.XformSet = false

					continue
				}
			}

			op.setProjectiveTransform(projection)
			op.XformSet = false

			continue
		}

		transform := affine
		if !exclusive {
			if op.XformSet {
				transform = transform.Mul(op.Transform())
			}
			if inner, ok := op.ProjectiveTransformValue(); ok {
				if op.Kind == OpImage {
					setPreserveImageTransform(op, composeProjectiveAffine(inner, affine))
				} else {
					op.setProjectiveTransform(composeProjectiveAffine(inner, affine))
					op.XformSet = false
				}

				continue
			}
		}

		op.setXform(transform)
		op.XformSet = true
	}
}

func matrix2DFromWorld3D(world matrix3D) Matrix2D {
	return Matrix2D{
		A: world[0], B: world[4], C: world[1], D: world[5], E: world[3], F: world[7],
	}
}

func projectiveTransform3DWorld(world matrix3D, vanishX, vanishY, dist float64) ProjectiveTransform {
	wX, wY, w0 := -world[8], -world[9], dist-world[11]

	return ProjectiveTransform{
		A: vanishX*wX + dist*world[0],
		B: vanishY*wX + dist*world[4],
		C: vanishX*wY + dist*world[1],
		D: vanishY*wY + dist*world[5],
		E: vanishX*w0 + dist*(world[3]-vanishX),
		F: vanishY*w0 + dist*(world[7]-vanishY),
		G: wX, H: wY, I: w0,
	}
}

func composeProjectiveInputAffine(projective ProjectiveTransform, affine Matrix2D) ProjectiveTransform {
	return composeProjectiveTransforms(projective, ProjectiveTransform{
		A: affine.A, B: affine.B, C: affine.C, D: affine.D, E: affine.E, F: affine.F, I: 1,
	})
}

func composeProjectiveTransforms(outer, inner ProjectiveTransform) ProjectiveTransform {
	return ProjectiveTransform{
		A: outer.A*inner.A + outer.C*inner.B + outer.E*inner.G,
		B: outer.B*inner.A + outer.D*inner.B + outer.F*inner.G,
		C: outer.A*inner.C + outer.C*inner.D + outer.E*inner.H,
		D: outer.B*inner.C + outer.D*inner.D + outer.F*inner.H,
		E: outer.A*inner.E + outer.C*inner.F + outer.E*inner.I,
		F: outer.B*inner.E + outer.D*inner.F + outer.F*inner.I,
		G: outer.G*inner.A + outer.H*inner.B + outer.I*inner.G,
		H: outer.G*inner.C + outer.H*inner.D + outer.I*inner.H,
		I: outer.G*inner.E + outer.H*inner.F + outer.I*inner.I,
	}
}

func setPreserveImageTransform(op *Op, projective ProjectiveTransform) {
	cx, cy := op.X+op.W/2, op.Y+op.H/2
	x, y := projective.Apply(cx, cy)
	jacobian := projective.Jacobian(cx, cy)
	jacobian.E = x - jacobian.A*cx - jacobian.C*cy
	jacobian.F = y - jacobian.B*cx - jacobian.D*cy
	op.setXform(jacobian)
	op.XformSet = true
	op.clearProjectiveTransform()
}
