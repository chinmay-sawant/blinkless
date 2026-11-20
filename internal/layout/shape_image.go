package layout

import (
	"image"
	"math"
	"strconv"
	"strings"
)

// shapeMaskRowCap bounds how many scanlines one image mask keeps. A taller
// image is sampled down to this many rows.
const shapeMaskRowCap = 256

// buildShapeOutside resolves a float's shape-outside. A url() shape is read
// from the image's alpha channel; every other shape goes through
// buildShapeExclusion.
func (e *engine) buildShapeOutside(sty ResolvedStyle, fbox *box, margL, margR float64) *shapeExclusion {
	raw := strings.TrimSpace(sty.ShapeOutside)
	if !strings.HasPrefix(raw, "url(") {
		return buildShapeExclusion(sty, fbox, margL, margR, e.scale)
	}

	return e.buildShapeImageMask(sty, fbox, margL, margR, raw)
}

// shapeImageThreshold reads the declared shape-image-threshold. An absent
// declaration is the initial value, 0.
func shapeImageThreshold(sty ResolvedStyle) float64 {
	value, err := strconv.ParseFloat(sty.declared().ShapeImageThreshold, 64)
	if err != nil {
		return 0
	}

	return math.Min(math.Max(value, 0), 1)
}

// shapeURLSource pulls the address out of `url(...)`, with an optional box
// keyword after it, and removes the quotes.
func shapeURLSource(raw string) string {
	end := strings.LastIndexByte(raw, ')')
	if !strings.HasPrefix(raw, "url(") || end < 0 {
		return ""
	}

	return strings.Trim(strings.TrimSpace(raw[len("url("):end]), `"'`)
}

// shapeMaskScan reads one scanline of img and returns the first and last
// column whose alpha is above threshold, or -1 for both when none is.
func shapeMaskScan(img image.Image, srcY int, threshold float64) (int, int) {
	bounds := img.Bounds()
	first, last := -1, -1

	for col := range bounds.Dx() {
		_, _, _, alpha := img.At(bounds.Min.X+col, srcY).RGBA()
		if float64(alpha)/0xffff <= threshold {
			continue
		}

		if first < 0 {
			first = col
		}

		last = col
	}

	return first, last
}

// buildShapeImageMask scans the image's alpha. A pixel is inside the shape
// when its alpha is above the threshold, so threshold 0 counts every faint
// pixel and threshold 1 counts nothing. The image is stretched over the
// float's margin box, and each scanline keeps the leftmost and rightmost
// inside pixel. shape-margin widens each scanline sideways. It does not grow
// the shape up or down. nil keeps the rectangular exclusion.
//
//nolint:cyclop // one guard per way the image can be unusable, then the scan
func (e *engine) buildShapeImageMask(sty ResolvedStyle, fbox *box, margL, margR float64, raw string) *shapeExclusion {
	src := shapeURLSource(raw)
	if src == "" || !e.hasImageResolver() {
		return nil
	}

	ref := e.resolveImage(src)
	if ref == nil {
		return nil
	}

	img, err := decodeImageBytes(ref.data)
	if err != nil || img == nil {
		return nil
	}

	refX := fbox.x - margL
	refW := fbox.w + margL + margR
	refH := fbox.height
	bounds := img.Bounds()

	if refW <= 0 || refH <= 0 || bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return nil
	}

	threshold := shapeImageThreshold(sty)
	margin := resolveShapeMargin(sty, refW, refH, e.scale)
	rowCount := min(bounds.Dy(), shapeMaskRowCap)
	rows := make([]shapeMaskRow, rowCount)

	for row := range rowCount {
		first, last := shapeMaskScan(img, bounds.Min.Y+row*bounds.Dy()/rowCount, threshold)
		if first < 0 {
			continue
		}

		rows[row] = shapeMaskRow{
			left:  refX + refW*float64(first)/float64(bounds.Dx()) - margin,
			right: refX + refW*float64(last+1)/float64(bounds.Dx()) + margin,
			ok:    true,
		}
	}

	return &shapeExclusion{ //nolint:exhaustruct // the mask kind uses only rows and the vertical span
		kind: shapeKindMask, y0: fbox.y, y1: fbox.y + refH, rows: rows,
	}
}

// maskInterval returns the covered stretch of the scanline at canvas y.
func (s *shapeExclusion) maskInterval(y float64) (float64, float64, bool) {
	if len(s.rows) == 0 || y < s.y0 || y >= s.y1 || s.y1 <= s.y0 {
		return 0, 0, false
	}

	row := int((y - s.y0) / (s.y1 - s.y0) * float64(len(s.rows)))
	row = min(max(row, 0), len(s.rows)-1)

	if !s.rows[row].ok {
		return 0, 0, false
	}

	return s.rows[row].left, s.rows[row].right, true
}
