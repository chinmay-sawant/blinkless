package layout

// Page-break keyword shared by style resolution. The PDF page splitter that
// consumed it is gone. The keyword still lands on the box so later passes can
// read it.
const (
	pageBreakAvoid  = "avoid"
	pageBreakAlways = "always"
)

// layoutCoordEpsilon is the coordinate tolerance for same-band tests.
const layoutCoordEpsilon = 0.01

func opRadii(op *Op) [4]float64 {
	if op.RadiusTopLeft == 0 && op.RadiusTopRight == 0 && op.RadiusBottomRight == 0 && op.RadiusBottomLeft == 0 {
		return [4]float64{op.Radius, op.Radius, op.Radius, op.Radius}
	}

	return [4]float64{op.RadiusTopLeft, op.RadiusTopRight, op.RadiusBottomRight, op.RadiusBottomLeft}
}

func isDashedOrDottedStyle(style string) bool {
	return style == borderStyleDashed || style == borderStyleDotted
}

func looksLikeDashSegmentLength(segLen, strokeWidth float64) bool {
	if segLen <= 0 {
		return false
	}

	maxSeg := strokeWidth * 3
	if maxSeg < 0.5 {
		maxSeg = 0.5
	}

	maxSeg += 0.5
	if strokeWidth <= 0 {
		maxSeg = 3.5
	}

	return segLen <= maxSeg
}

func nearLayout(a, b float64) bool {
	return absFloat(a-b) < layoutCoordEpsilon
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}

	return v
}
