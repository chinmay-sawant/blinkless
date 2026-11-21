package layout

import "math"

// borderBoundarySteps is how many segments trace one quarter of a corner curve.
const borderBoundarySteps = 24

// addBorderBoundary gives every box that declares border-boundary: parent the
// extra border along its parent's rounded corners. Where the child's border box
// reaches a corner curve of the parent, the part of that curve inside the child
// box is stroked with the child's border color and width, inset by half the
// width so the stroke sits inside the parent's edge. A parent with square
// corners, a child that stays clear of the curve, and a child whose border does
// not paint add nothing. The value `display` needs a screen outline the engine
// does not store, so it keeps the rectangular border.
func (e *engine) addBorderBoundary(root *box) {
	if len(e.deferredChrome) == 0 {
		return
	}

	var parents map[*box]*box

	for index := range e.deferredChrome {
		entry := &e.deferredChrome[index]
		if entry.b == nil || entry.b.style == nil || entry.b.style.declared().BorderBoundary != "parent" {
			continue
		}

		if parents == nil {
			parents = boxParents(root)
		}

		if parent := parents[entry.b]; parent != nil {
			extra := e.borderBoundaryOps(entry.b, parent)
			entry.ops = append(entry.ops, extra...)
		}
	}
}

// boxParents maps every box under root to its parent.
func boxParents(root *box) map[*box]*box {
	out := map[*box]*box{}

	var walk func(node *box)

	walk = func(node *box) {
		for _, child := range node.children {
			if child != nil {
				out[child] = node

				walk(child)
			}
		}
	}

	if root != nil {
		walk(root)
	}

	return out
}

// borderBoundaryOps builds the path ops for one child against its parent.
func (e *engine) borderBoundaryOps(child, parent *box) []Op {
	if parent.style == nil {
		return nil
	}

	side := child.style.BorderTop
	if !cornerShapePaintBorderSideUsable(side) {
		return nil
	}

	radii, radiiY := usedBorderRadiiXY(parent.style, parent.w, parent.height)
	width := e.scalePt(borderPaint(side))
	inset := width / 2 //nolint:mnd // stroke is centered on its path

	var ops []Op

	for corner := range radii {
		radX, radY := radii[corner]-inset, radiiY[corner]-inset
		if radX <= 0 || radY <= 0 {
			continue
		}

		for _, run := range borderBoundaryRuns(corner, parent, radX, radY, inset, child) {
			op := Op{ //nolint:exhaustruct // intentional zero fields
				Kind: OpPath, X: child.x, Y: child.y, W: child.w, H: child.height,
				R: side.Color[0], G: side.Color[1], B: side.Color[2], Width: width,
			}
			op.setCornerPath(borderBoundaryCommands(run), true)
			ops = append(ops, op)
		}
	}

	return ops
}

// borderBoundaryRuns traces one parent corner and returns the stretches of it
// that lie inside the child's border box. corner is 0 top-left, 1 top-right,
// 2 bottom-right, 3 bottom-left.
//
//nolint:cyclop // one corner switch, then one trace loop
func borderBoundaryRuns(corner int, parent *box, radX, radY, inset float64, child *box) [][]DisplayPoint {
	centerX, centerY, start := parent.x+inset+radX, parent.y+inset+radY, math.Pi
	right, bottom := parent.x+parent.w-inset, parent.y+parent.height-inset

	switch corner {
	case 1:
		centerX, centerY, start = right-radX, parent.y+inset+radY, 3*math.Pi/2 //nolint:mnd // quarter turns
	case 2: //nolint:mnd // bottom-right
		centerX, centerY, start = right-radX, bottom-radY, 0
	case 3: //nolint:mnd // bottom-left
		centerX, centerY, start = parent.x+inset+radX, bottom-radY, math.Pi/2 //nolint:mnd // quarter turns
	}

	var (
		runs [][]DisplayPoint
		run  []DisplayPoint
	)

	for step := 0; step <= borderBoundarySteps; step++ {
		angle := start + math.Pi/2*float64(step)/borderBoundarySteps
		point := DisplayPoint{X: centerX + radX*math.Cos(angle), Y: centerY + radY*math.Sin(angle)}

		if point.X >= child.x-borderEps && point.X <= child.x+child.w+borderEps &&
			point.Y >= child.y-borderEps && point.Y <= child.y+child.height+borderEps {
			run = append(run, point)

			continue
		}

		if len(run) > 1 {
			runs = append(runs, run)
		}

		run = nil
	}

	if len(run) > 1 {
		runs = append(runs, run)
	}

	return runs
}

// borderBoundaryCommands turns one traced run into an open path.
func borderBoundaryCommands(run []DisplayPoint) []PathCmd {
	cmds := make([]PathCmd, 0, len(run))
	cmds = append(cmds, PathCmd{Verb: PathMoveTo, Pts: [3]DisplayPoint{run[0]}})

	for _, point := range run[1:] {
		cmds = append(cmds, PathCmd{Verb: PathLineTo, Pts: [3]DisplayPoint{point}})
	}

	return cmds
}
