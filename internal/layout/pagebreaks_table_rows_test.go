package layout

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// auditPageContentH is the A4 content height the audit fixture PDFs are
// generated with: 595.28x841.89pt page, 12mm margins.
const auditPageContentH = 841.89 - 2*12*72/25.4

// TestPageBreaksKeepTableRowsIntact: a table row must never straddle a page
// boundary. Browsers (and the native engine) never split a row: a straddling
// row moves whole to the next page top. Before rows were exported as keep
// ranges, the writer split row 15 of fixture-60 (top 751.21, bottom 796.13,
// boundary 773.86) and painted an empty leading fragment at the page bottom:
// cell tops and the Effect box border with no text, while page 2 started with
// the row's top padding and border eaten. Native output ends page 1 cleanly
// after row 14.
//
// The three implemented-props audits are where the defect shows (60 page 1
// row 15, 61 page 1 rows 15-16, 62 page 1 row 16). The assertion is generic:
// every non-empty row of every table that straddles a page boundary has a
// KeepTogether range spanning the row. A row a forced break cuts is exempt
// (the forced break wins and the exporter drops its range).
func TestPageBreaksKeepTableRowsIntact(t *testing.T) {
	t.Parallel()

	fixtures := []string{
		"fixture-60-implemented-props-a.html",
		"fixture-61-implemented-props-b.html",
		"fixture-62-implemented-props-c.html",
		"fixture-64-next-72-props.html",
	}

	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			t.Parallel()

			res := layoutAuditFixture(t, fixture)

			forced, keep := PageBreaks(res, 0)

			bands := tableRowBands(tablesOf(res))
			if len(bands) == 0 {
				t.Fatal("no table rows found")
			}

			rows, straddles, covered := auditRowCoverage(bands, forced, keep, auditPageContentH)

			if straddles == 0 {
				t.Fatalf("%d rows, none straddle a %.2f boundary: fixture no longer exercises the defect",
					rows, auditPageContentH)
			}

			if covered != straddles {
				t.Fatalf("%d of %d straddling rows lack a KeepTogether range; %d rows total",
					straddles-covered, straddles, rows)
			}
		})
	}
}

// TestPageBreaksKeepCoversTableTextLineBoxes: the paginator checks a text op
// for a page-boundary crossing as baseline Y plus recorded line height H,
// which pokes below the row border box. A row keep ending at the border box
// would let the paginator snap the text alone and leave the row chrome
// behind, so the keep end must cover the row's text line boxes.
func TestPageBreaksKeepCoversTableTextLineBoxes(t *testing.T) { //nolint:cyclop // row coverage assertions
	t.Parallel()

	res := layoutHTML(t, `<html><body>`+
		`<table style="border-collapse:collapse;width:360px">`+
		`<thead><tr><th>Head</th></tr></thead>`+
		`<tbody>`+
		`<tr><td style="padding:6px">alpha bravo charlie delta echo foxtrot golf hotel india juliet kilo lima</td></tr>`+
		`<tr><td>second row text</td></tr>`+
		`</tbody></table>`+
		`</body></html>`)

	_, keep := PageBreaks(res, 0)

	checked := 0

	for _, tbl := range tablesOf(res) {
		for _, cell := range tbl.children {
			if cell.kind != boxKindCell || cell.opStart < 0 || cell.opStart > cell.opEnd {
				continue
			}

			for idx := cell.opStart; idx <= cell.opEnd && idx < len(res.Ops); idx++ {
				paintOp := &res.Ops[idx]
				if paintOp.Kind != OpText && paintOp.Kind != OpBullet {
					continue
				}

				bottom := paintOp.Y + paintOp.H
				if paintOp.H <= 0 {
					bottom = paintOp.Y + paintOp.Size*textLineHeightFactor
				}

				if !keepCovers(keep, paintOp.Y, bottom) {
					t.Fatalf("text op %q [%.2f..%.2f] not inside a keep range; keeps = %v",
						paintOp.Text, paintOp.Y, bottom, keep)
				}

				checked++
			}
		}
	}

	if checked == 0 {
		t.Fatal("no table text ops checked")
	}
}

// tableRowBands returns [top, bottom] canvas bands for every non-empty table
// row, derived from cell boxes rather than from break export: each local cell
// carries its starting row's top (y) and height (rowBoxH).
func tableRowBands(tables []*box) [][2]float64 {
	var bands [][2]float64

	for _, tbl := range tables {
		found := make(map[int][2]float64)

		for _, cell := range tbl.children {
			if cell.kind != boxKindCell || cell.row < 0 {
				continue
			}

			if _, ok := found[cell.row]; !ok {
				found[cell.row] = [2]float64{cell.y, cell.y + cell.rowBoxH}
			}
		}

		for _, band := range found {
			if band[1]-band[0] > 0.01 {
				bands = append(bands, band)
			}
		}
	}

	return bands
}

// tablesOf collects the table boxes in document order.
func tablesOf(res *Result) []*box {
	var tables []*box

	var walk func(boxNode *box)
	walk = func(boxNode *box) {
		if boxNode == nil {
			return
		}

		if boxNode.kind == boxKindTable {
			tables = append(tables, boxNode)
		}

		for _, child := range boxNode.children {
			walk(child)
		}
	}
	walk(res.root)

	return tables
}

// auditRowCoverage counts non-empty rows, rows that straddle a page boundary
// without a forced break cutting them, and straddling rows covered by a keep
// range that spans the whole row.
func auditRowCoverage(bands [][2]float64, forced []float64, keep [][2]float64, contentH float64) (int, int, int) {
	rows, straddles, covered := 0, 0, 0

	for _, band := range bands {
		rows++

		if int(band[0]/contentH) == int((band[1]-1e-6)/contentH) {
			continue
		}

		// A forced break inside the row splits it no matter what, so the
		// exporter drops its keep; that row is not this test's contract.
		if cutByForced(band, forced) {
			continue
		}

		straddles++

		if keepCovers(keep, band[0], band[1]) {
			covered++
		}
	}

	return rows, straddles, covered
}

// cutByForced reports whether a forced break cuts the band (its start above
// the break, its end on or below it).
func cutByForced(band [2]float64, forced []float64) bool {
	const eps = 0.01

	for _, breakY := range forced {
		if band[0] < breakY-eps && band[1] >= breakY-eps {
			return true
		}
	}

	return false
}

func keepCovers(keep [][2]float64, top, bottom float64) bool {
	const eps = 0.01

	for _, k := range keep {
		if k[0] <= top+eps && k[1] >= bottom-eps {
			return true
		}
	}

	return false
}

// layoutAuditFixture lays out one audit fixture at the A4 content box the PDF
// writer uses (contentW x contentH points), with its <style> blocks applied.
func layoutAuditFixture(t *testing.T, name string) *Result {
	t.Helper()

	b, err := os.ReadFile("../../testdata/golden/" + name)
	if err != nil {
		t.Fatal(err)
	}

	root, err := html.Parse(string(b))
	if err != nil {
		t.Fatal(err)
	}

	var sheets []*css.Stylesheet

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Name == "style" {
			var cssSheet string

			for _, c := range node.Children {
				if c.Type == html.TextNode {
					cssSheet += c.Text
				}
			}

			if sh, err := css.Parse(cssSheet); err == nil {
				sheets = append(sheets, sh)
			}
		}

		for _, c := range node.Children {
			walk(c)
		}
	}
	walk(root)

	res, err := Layout(root, Options{
		Width: 595.28 - 2*12*72/25.4, Height: auditPageContentH, Sheets: sheets,
		Background: true, Media: "print",
		Images: auditFixtureImages,
	})
	if err != nil {
		t.Fatal(err)
	}

	return res
}

// errFixtureImageNotFound reports an audit fixture image source that cannot
// be resolved under testdata/golden.
var errFixtureImageNotFound = errors.New("image not found")

// auditFixtureImages resolves the audit fixtures' document-relative image
// sources (logo.png, assets/*.png) against testdata/golden.
func auditFixtureImages(src string) ([]byte, error) {
	base := filepath.Base(src)

	for _, path := range []string{
		filepath.Join("../../testdata/golden", base),
		filepath.Join("../../testdata/golden/assets", base),
	} {
		if b, err := os.ReadFile(path); err == nil {
			return b, nil
		}
	}

	return nil, errFixtureImageNotFound
}
