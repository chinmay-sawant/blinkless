package layout

import (
	"errors"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

func TestGridRejectsExtremeRowExtent(t *testing.T) {
	t.Parallel()

	for _, rule := range []string{"grid-row: span 1000000000", "grid-row: 1000000000", "grid-row:1 / 1000000000"} {
		root := mustParse(t, `<div class="grid"><div class="item">x</div></div>`)
		sheets := []*css.Stylesheet{sheet(t, ".grid{display:grid}.item{"+rule+"}")}

		if _, err := Layout(root, Options{Width: 500, Sheets: sheets}); !errors.Is(err, errGridExtentExceeded) {
			t.Fatalf("%s: error = %v, want grid budget error", rule, err)
		}
	}
}

func TestGridRejectsCumulativeCellBudget(t *testing.T) {
	t.Parallel()

	if gridExtentWithinBudget(-1, -1, maxGridRows, 32, maxGridCells) {
		t.Fatal("accepted a span exceeding the cell budget")
	}

	if gridExtentWithinBudget(-1, -1, 1, 1, 0) {
		t.Fatal("accepted a cell after exhausting the cumulative budget")
	}
}

func TestGridRepeatNestingStaysBounded(t *testing.T) {
	t.Parallel()

	nested := strings.Repeat("repeat(63, ", 3) + "1px)))"
	got := expandRepeatFunctions(nested)

	if got != nested {
		t.Fatalf("triple nested repeat expanded to %d bytes", len(got))
	}

	flat := expandRepeatFunctions("repeat(4, 1px)")
	if strings.Count(flat, "1px") != 4 {
		t.Fatalf("repeat(4, 1px) = %q", flat)
	}
}

func TestGridOccupationWorkBudget(t *testing.T) {
	t.Parallel()

	occ := newGridOccupation(2)
	occ.workLeft = 2
	occ.freeAt(0, 0, 1000000000, 2)

	if !errors.Is(occ.err, errGridExtentExceeded) {
		t.Fatalf("occupancy error = %v, want grid work budget error", occ.err)
	}

	if len(occ.byRow) > 2 {
		t.Fatalf("allocated %d rows past work budget", len(occ.byRow))
	}
}

// TestTableSpanCap proves arbitrary colspan/rowspan values clamp to the
// table-span budget instead of amplifying grid memory.
func TestTableSpanCap(t *testing.T) {
	t.Parallel()

	if got := tableSpan("999999"); got != maxTableSpan {
		t.Fatalf("tableSpan(999999) = %d, want %d", got, maxTableSpan)
	}

	if got := tableSpan("2"); got != 2 {
		t.Fatalf("tableSpan(2) = %d, want 2", got)
	}

	if got := tableSpan(""); got != 1 {
		t.Fatalf("tableSpan() = %d, want 1", got)
	}
}

// TestValidImageSizeBudget proves extreme header dimensions are rejected
// before any decode or allocation.
func TestValidImageSizeBudget(t *testing.T) {
	t.Parallel()

	for _, size := range [][2]int{{0, 10}, {10, 0}, {-1, 10}, {1 << 30, 1 << 30}, {maxImageDimPixels + 1, 1}} {
		if validImageSize(size[0], size[1]) {
			t.Fatalf("validImageSize(%d, %d) = true, want false", size[0], size[1])
		}
	}

	if !validImageSize(800, 600) {
		t.Fatal("validImageSize(800, 600) = false, want true")
	}

	if _, _, _, ok := imageDims(make([]byte, 24)); ok {
		t.Fatal("imageDims accepted non-image bytes")
	}
}

// TestDangerousSchemeObfuscation proves whitespace/control smuggling cannot
// sneak an executable scheme past the link filter.
func TestDangerousSchemeObfuscation(t *testing.T) {
	t.Parallel()

	for _, href := range []string{
		"javascript:alert(1)",
		"  JaVaScRiPt:alert(1)",
		"java\tscript:alert(1)",
		"java\nscript:alert(1)",
		"java\x00script:alert(1)",
		"vbscript:msgbox(1)",
		"data:text/html,<h1>x</h1>",
		"blob:https://example.com/x",
		"file:///etc/passwd",
	} {
		if !hasDangerousScheme(href) {
			t.Fatalf("hasDangerousScheme(%q) = false, want true", href)
		}

		if isLinkHref(href) {
			t.Fatalf("isLinkHref(%q) = true, want false", href)
		}
	}

	if !isLinkHref("/relative/path") {
		t.Fatal("isLinkHref refused a relative link")
	}
}

// TestAlignContentOffsetsDegenerate proves empty and singleton line sets
// produce finite offsets instead of NaN.
func TestAlignContentOffsetsDegenerate(t *testing.T) {
	t.Parallel()

	for _, align := range []string{"space-between", "space-around", "space-evenly", "center", "flex-start"} {
		for _, count := range []int{0, 1} {
			for _, offset := range alignContentOffsets(align, 100, count) {
				if offset != offset || offset > 1e18 || offset < -1e18 {
					t.Fatalf("alignContentOffsets(%q, 100, %d) produced bad offset %v", align, count, offset)
				}
			}
		}
	}
}

// TestImageRefDecodableNil proves a nil reference is not decodable.
func TestImageRefDecodableNil(t *testing.T) {
	t.Parallel()

	if imageRefDecodable(nil) {
		t.Fatal("imageRefDecodable(nil) = true, want false")
	}
}

// TestHugeDashedEdgeCapsSegments is a dashed border whose edge is far longer
// than a page. The painter steps at least 1pt, so an uncapped edge would
// allocate one segment per step. A few hundred points must still dash.
func TestHugeDashedEdgeCapsSegments(t *testing.T) {
	t.Parallel()

	normal := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<div style="width:400pt;height:30pt;border:1pt dashed #000"></div>`+
		`</body></html>`)
	normalLines := opsOfKind(normal, OpLine)

	if len(normalLines) < 40 || len(normalLines) > 400 {
		t.Fatalf("normal dashed border lines = %d, want dashes on a few hundred points", len(normalLines))
	}

	root := mustParse(t, `<html style="margin:0"><body style="margin:0">`+
		`<div style="width:1000000pt;height:20pt;border:1pt dashed #000"></div>`+
		`</body></html>`)
	res, err := Layout(root, Options{
		Width: testViewport, Height: 800, Background: true,
	})

	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	lines := opsOfKind(res, OpLine)
	if len(lines) < maxDashSegments || len(lines) > maxDashSegments*4 {
		t.Fatalf("huge dashed edge lines = %d, want the %d cap on the long edges", len(lines), maxDashSegments)
	}
}

// TestHugeLetterSpacingDecorationReturns is a dashed, dotted, or wavy
// underline on a nowrap run stretched by letter-spacing. Each step used to
// emit an op across the whole run.
func TestHugeLetterSpacingDecorationReturns(t *testing.T) {
	t.Parallel()

	normal := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
		`<p style="margin:0;font-size:12pt;text-decoration-line:underline;`+
		`text-decoration-style:dashed;text-decoration-skip-ink:none">hello</p>`+
		`</body></html>`)
	if got := len(underlineOps(normal)); got < 2 {
		t.Fatalf("normal dashed underline lines = %d, want dashes", got)
	}

	for _, style := range []string{"dashed", "dotted", "wavy"} {
		res := layoutHTML(t, `<html style="margin:0"><body style="margin:0">`+
			`<p style="margin:0;font-size:12pt;white-space:nowrap;letter-spacing:200000pt;`+
			`text-decoration-line:underline;text-decoration-style:`+style+
			`;text-decoration-skip-ink:none">hi</p>`+
			`</body></html>`)
		lines := underlineOps(res)

		if len(lines) < 100 || len(lines) > maxDecorationSteps*2 {
			t.Fatalf("%s decoration lines = %d, want the %d step cap", style, len(lines), maxDecorationSteps)
		}
	}
}

// TestTinyBorderImageWidthReturns is border-image-repeat with a width so
// small that the tile step would not advance the float cursor.
func TestTinyBorderImageWidthReturns(t *testing.T) {
	t.Parallel()

	img := tinyPNG(8, 8)
	res := layoutHTMLWithImages(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="bi" style="width:100px;height:60px;border:10px solid #000;`+
		`border-image-source:url(border.png);border-image-slice:2;`+
		`border-image-width:0.0000001pt;border-image-repeat:repeat"></div>`+
		`</body></html>`, img, "border.png")
	cells := opsOfKind(res, OpImage)

	if len(cells) == 0 || len(cells) > 12 {
		t.Fatalf("tiny border-image cells = %d, want one stretched frame", len(cells))
	}
}

// TestHugeFontMulticolReturns is column-count:2 with a font taller than the
// page. remain < fontSize*1.2 at every page top used to snap forever.
func TestHugeFontMulticolReturns(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html style="margin:0"><body style="margin:0">`+
		`<div id="mc" style="column-count:2;column-gap:8pt;font-size:2000pt">`+
		`<div>a</div><div>b</div></div></body></html>`)
	res, err := Layout(root, Options{
		Width: testViewport, Height: 400, Background: true,
	})

	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	if res == nil {
		t.Fatal("Layout returned a nil display list")
	}

	if len(res.Ops) > 100000 {
		t.Fatalf("multicol ops = %d, snap loop did not stop", len(res.Ops))
	}

	column := boxByID(t, res, "mc")
	if column.style == nil || column.style.ColumnCount != 2 || column.style.FontSize < 1000 {
		fontSize := 0.0
		cols := 0

		if column.style != nil {
			fontSize = column.style.FontSize
			cols = column.style.ColumnCount
		}

		t.Fatalf("column-count = %d font-size = %.1f, want 2 and a font taller than the page", cols, fontSize)
	}
}
