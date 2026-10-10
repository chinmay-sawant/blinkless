package layout

import (
	"sort"
	"testing"
)

// Border-image repeat tiling behavior tests. Every property below is asserted
// through emitted ops (cell counts and used geometry), never through a stored
// style string. Lengths are written in CSS px and converted with pxToPt
// (1px = 0.75pt). Reference browser: Chrome 143.0.7499.40, which tiles the
// edge middle-slice along each edge for repeat (clipping a trailing partial
// tile) while stretch maps one slice across the whole edge.

// TestBehaviorBorderImageRepeatTilesEdges is border-image-repeat: repeat on a
// 100px by 60px box with a 10px border and slice 2 of an 8px image. The edge
// middle slice is 4px wide and 2px tall, so each top/bottom tile paints
// 10px by 10px (7.5pt by 7.5pt aspect-kept) over the 100px (75pt) edge span:
// 5 tiles per horizontal edge and 3 per vertical edge, 20 cells total against
// 8 under stretch. Corners stay identical to the stretch frame.
func TestBehaviorBorderImageRepeatTilesEdges(t *testing.T) {
	t.Parallel()

	img := tinyPNG(8, 8)
	htmlSrc := func(repeat string) string {
		return `<html style="margin:0"><body style="margin:0">` +
			`<div id="bi" style="width:100px;height:60px;border:10px solid #000;` +
			`border-image-source:url(border.png);border-image-slice:2;border-image-repeat:` + repeat + `"></div>` +
			`</body></html>`
	}

	stretchRes := layoutHTMLWithImages(t, htmlSrc("stretch"), img, "border.png")
	repeatRes := layoutHTMLWithImages(t, htmlSrc("repeat"), img, "border.png")
	roundRes := layoutHTMLWithImages(t, htmlSrc("round"), img, "border.png")

	stretched := behaviorBgImages(opsOfKind(stretchRes, OpImage))
	repeated := behaviorBgImages(opsOfKind(repeatRes, OpImage))
	rounded := behaviorBgImages(opsOfKind(roundRes, OpImage))

	if len(stretched) != 8 {
		t.Fatalf("stretch border-image cells = %d, want 8", len(stretched))
	}

	if len(repeated) != 20 {
		t.Fatalf("repeat border-image cells = %d, want 20 (4 corners + 5 + 5 + 3 + 3 tiles)", len(repeated))
	}

	if len(rounded) != len(repeated) {
		t.Fatalf("round border-image cells = %d, want %d shared with repeat", len(rounded), len(repeated))
	}

	borderBox := boxByID(t, repeatRes, "bi")
	thick := pxToPt(10)
	topBand := behaviorBordImgBand(repeated, borderBox.y, thick, true)
	bottomBand := behaviorBordImgBand(repeated, borderBox.y+borderBox.height-thick, thick, true)
	leftBand := behaviorBordImgBand(repeated, borderBox.x, thick, false)
	rightBand := behaviorBordImgBand(repeated, borderBox.x+borderBox.w-thick, thick, false)

	if len(topBand) != 5 || len(bottomBand) != 5 {
		t.Fatalf("horizontal edge tiles = top %d bottom %d, want 5 each", len(topBand), len(bottomBand))
	}

	if len(leftBand) != 3 || len(rightBand) != 3 {
		t.Fatalf("vertical edge tiles = left %d right %d, want 3 each", len(leftBand), len(rightBand))
	}

	// Tiles abut along the edge and exactly cover the span between corners.
	behaviorBordImgAssertAbutting(t, topBand, borderBox.x+thick, borderBox.x+borderBox.w-thick, true, "top")
	behaviorBordImgAssertAbutting(t, leftBand, borderBox.y+thick, borderBox.y+borderBox.height-thick, false, "left")
	behaviorBordImgAssertTileGeometry(t, topBand, leftBand, thick)
	behaviorBordImgAssertCornersKept(t, stretched, repeated, borderBox, thick)
}

// behaviorBordImgAssertTileGeometry asserts full tiles carry the whole middle slice:
// 4px by 2px on top/bottom, 2px by 4px on the sides.
func behaviorBordImgAssertTileGeometry(t *testing.T, topBand, leftBand []Op, thick float64) {
	t.Helper()

	// Full tiles carry the whole middle slice: 4px by 2px on top/bottom,
	// 2px by 4px on the sides. The 75pt span is an exact multiple of the
	// 15pt tile, so every tile here is full.
	for _, tile := range topBand {
		behaviorBordImgAssertTile(t, tile, pxToPt(20), thick, 4, 2, "top")
	}

	for _, tile := range leftBand {
		behaviorBordImgAssertTile(t, tile, thick, pxToPt(20), 2, 4, "left")
	}
}

// behaviorBordImgAssertTile asserts one full tile has the wanted size and payload.
func behaviorBordImgAssertTile(t *testing.T, tile Op, wantW, wantH float64, wantIw, wantIh int, tag string) {
	t.Helper()

	if !near(tile.W, wantW) || !near(tile.H, wantH) {
		t.Errorf("%s tile = %.4fpt x %.4fpt, want %.2fpx x %.2fpx",
			tag, tile.W/ptPerCSSPx, tile.H/ptPerCSSPx, wantW/ptPerCSSPx, wantH/ptPerCSSPx)
	}

	if tile.ImgW != wantIw || tile.ImgH != wantIh {
		t.Errorf("%s tile payload = %dx%d, want %dx%d", tag, tile.ImgW, tile.ImgH, wantIw, wantIh)
	}

	if len(tile.Image) == 0 {
		t.Errorf("%s tile carries no image payload", tag)
	}
}

// behaviorBordImgAssertCornersKept asserts every stretch corner survives under repeat
// with identical geometry and payload bytes.
func behaviorBordImgAssertCornersKept(t *testing.T, stretched, repeated []Op, borderBox *box, thick float64) {
	t.Helper()

	for _, corner := range stretched {
		if !behaviorBordImgIsCorner(corner, borderBox, thick) {
			continue
		}

		match := behaviorBordImgFindCell(repeated, corner)

		if match == nil {
			t.Errorf("stretch corner at (%.4fpt, %.4fpt) missing under repeat", corner.X, corner.Y)

			continue
		}

		if match.ImgW != corner.ImgW || match.ImgH != corner.ImgH ||
			string(match.Image) != string(corner.Image) {
			t.Errorf("repeat corner at (%.4fpt, %.4fpt) payload differs from stretch",
				corner.X, corner.Y)
		}
	}
}

// TestBehaviorBorderImageRepeatClipsPartialTile is border-image-repeat:
// repeat on a 95px wide box: the 71.25pt edge span holds 4 full 15pt tiles
// plus an 11.25pt trailing tile whose source is clipped to the leading 3px
// of the 4px middle slice. Reference: Chrome 143.0.7499.40 clips rather than
// squeezes the last tile.
func TestBehaviorBorderImageRepeatClipsPartialTile(t *testing.T) {
	t.Parallel()

	img := tinyPNG(8, 8)
	res := layoutHTMLWithImages(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="bi" style="width:95px;height:60px;border:10px solid #000;`+
		`border-image-source:url(border.png);border-image-slice:2;border-image-repeat:repeat"></div>`+
		`</body></html>`, img, "border.png")

	cells := behaviorBgImages(opsOfKind(res, OpImage))
	borderBox := boxByID(t, res, "bi")
	thick := pxToPt(10)
	topBand := behaviorBordImgBand(cells, borderBox.y, thick, true)

	if len(topBand) != 5 {
		t.Fatalf("top edge tiles = %d, want 5 (4 full + 1 clipped)", len(topBand))
	}

	last := topBand[len(topBand)-1]

	if !near(last.W, pxToPt(15)) {
		t.Errorf("trailing tile width = %.4fpt, want 15px clipped from 20px",
			last.W/ptPerCSSPx)
	}

	if last.ImgW != 3 || last.ImgH != 2 {
		t.Errorf("trailing tile payload = %dx%d, want 3x2 clipped slice", last.ImgW, last.ImgH)
	}

	if !near(last.X+last.W, borderBox.x+borderBox.w-thick) {
		t.Errorf("trailing tile ends at %.4fpt, want the right corner edge at %.4fpt",
			last.X+last.W, borderBox.x+borderBox.w-thick)
	}
}

// behaviorBordImgBand returns the edge tiles of one band sorted along the edge:
// horizontal bands match Y and thickness H, vertical bands match X and
// thickness W. Corner cells (which share the band origin) are excluded.
func behaviorBordImgBand(cells []Op, origin, thick float64, horizontal bool) []Op {
	band := make([]Op, 0, len(cells))

	for _, cell := range cells {
		candidate := cell

		if horizontal {
			if !near(candidate.Y, origin) || !near(candidate.H, thick) {
				continue
			}
		} else {
			if !near(candidate.X, origin) || !near(candidate.W, thick) {
				continue
			}
		}

		band = append(band, candidate)
	}

	// Corners share the band row or column but span the full thickness in
	// both axes; tiles span further along the edge than across it.
	kept := behaviorBordImgExcludeCorners(band, thick, horizontal)

	sort.Slice(kept, func(i, j int) bool {
		if horizontal {
			return kept[i].X < kept[j].X
		}

		return kept[i].Y < kept[j].Y
	})

	return kept
}

// behaviorBordImgExcludeCorners drops corner cells from a band: corners span the full
// thickness in both axes while tiles span further along the edge.
func behaviorBordImgExcludeCorners(band []Op, thick float64, horizontal bool) []Op {
	kept := make([]Op, 0, len(band))

	for _, cell := range band {
		if horizontal && near(cell.W, thick) {
			continue
		}

		if !horizontal && near(cell.H, thick) {
			continue
		}

		kept = append(kept, cell)
	}

	return kept
}

// behaviorBordImgAssertAbutting checks tiles start at spanStart, each starts where the
// previous ends, and the last ends at spanEnd.
func behaviorBordImgAssertAbutting(
	t *testing.T, tiles []Op, spanStart, spanEnd float64, horizontal bool, tag string,
) {
	t.Helper()

	if len(tiles) == 0 {
		t.Fatalf("%s edge has no tiles", tag)
	}

	pos := spanStart

	for idx, tile := range tiles {
		start := tile.X
		size := tile.W

		if !horizontal {
			start = tile.Y
			size = tile.H
		}

		if !near(start, pos) {
			t.Fatalf("%s tile %d starts at %.4fpt, want %.4fpt", tag, idx, start, pos)
		}

		if size <= 0 {
			t.Fatalf("%s tile %d has non-positive size %.4fpt", tag, idx, size)
		}

		pos += size
	}

	if !near(pos, spanEnd) {
		t.Errorf("%s tiles end at %.4fpt, want %.4fpt (corner to corner)",
			tag, pos, spanEnd)
	}
}

// behaviorBordImgIsCorner reports whether a cell sits at one of the four frame corners.
func behaviorBordImgIsCorner(cell Op, borderBox *box, thick float64) bool {
	atLeft := near(cell.X, borderBox.x)
	atRight := near(cell.X+cell.W, borderBox.x+borderBox.w)
	atTop := near(cell.Y, borderBox.y)
	atBottom := near(cell.Y+cell.H, borderBox.y+borderBox.height)

	return (atLeft || atRight) && (atTop || atBottom) &&
		near(cell.W, thick) && near(cell.H, thick)
}

// behaviorBordImgFindCell returns the cell in cells with near-equal geometry, or nil.
func behaviorBordImgFindCell(cells []Op, want Op) *Op {
	for _, cell := range cells {
		candidate := cell

		if near(candidate.X, want.X) && near(candidate.Y, want.Y) &&
			near(candidate.W, want.W) && near(candidate.H, want.H) {
			return &candidate
		}
	}

	return nil
}
