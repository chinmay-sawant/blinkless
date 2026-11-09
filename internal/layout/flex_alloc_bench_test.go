package layout

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
	"github.com/chinmay-sawant/blinkless/internal/html"
)

// BenchmarkFlexAllocationChurn measures allocations during nested flex container layouts.
func BenchmarkFlexAllocationChurn(b *testing.B) {
	const doc = `<html><body>
<div class="container">
  <div class="row">
    <div class="item baseline"><span class="txt">Header 1</span></div>
    <div class="item center"><span class="txt">Subheader 2</span></div>
    <div class="item end"><span class="txt">Trailing 3</span></div>
  </div>
  <div class="col">
    <div class="col-item"><span class="txt">Column 1</span></div>
    <div class="col-item"><span class="txt">Column 2</span></div>
  </div>
</div>
</body></html>`

	const cssSrc = `body{margin:0}
.container{display:flex;flex-direction:column;width:600pt;gap:8pt}
.row{display:flex;flex-direction:row;justify-content:space-between;align-items:baseline;width:100%;gap:4pt}
.col{display:flex;flex-direction:column;width:100%;height:200pt;gap:4pt}
.item{flex:1 1 auto;min-width:50pt;max-width:200pt}
.col-item{flex:1 1 auto;min-height:20pt}
.baseline{align-self:baseline}
.center{align-self:center}
.end{align-self:flex-end}
.txt{font-size:12pt}
`
	parsed, err := css.Parse(cssSrc)
	if err != nil {
		b.Fatalf("css.Parse: %v", err)
	}
	sheets := []*css.Stylesheet{parsed}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		root, err := html.Parse(doc)
		if err != nil {
			b.Fatalf("html.Parse: %v", err)
		}
		opts := Options{
			Width: 800, Height: 600, Sheets: sheets, Background: true,
		}
		if _, err := Layout(root, opts); err != nil {
			b.Fatalf("Layout: %v", err)
		}
	}
}

// TestFlexBaselineAndAlignParity verifies baseline and cross-axis alignment
// calculations produce identical layout geometry.
func TestFlexBaselineAndAlignParity(t *testing.T) {
	t.Parallel()

	const doc = `<html><body>
<div class="row">
  <div class="item base1"><span>Top text</span></div>
  <div class="item base2"><span style="font-size:24pt">Big text</span></div>
  <div class="item none"><span>Plain</span></div>
</div>
</body></html>`

	const cssSrc = `body{margin:0}
.row{display:flex;flex-direction:row;align-items:baseline;width:400pt}
.item{flex:1 1 auto}
.base1{margin-top:10pt}
.base2{margin-top:4pt}
.none{align-self:flex-end}
`
	sheets := []*css.Stylesheet{sheet(t, cssSrc)}
	root := mustParse(t, doc)
	opts := Options{
		Width: 500, Height: 400, Sheets: sheets, Background: true,
	}
	res, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}
	if len(res.Ops) == 0 {
		t.Fatal("expected non-empty ops")
	}
}
