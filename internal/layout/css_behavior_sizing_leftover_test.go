package layout

import (
	"os"
	"testing"
)

// TestBehaviorSizeShorthandSetsUsedBorderBox is the element property size.
// One value is copied onto both axes. Two values set width, then height.
// size:auto on an empty div in a 400px wrapper is 400px wide and 0px tall.
// A token height rejects, or any other bad token, leaves the earlier width
// and height in place. Percentages resolve against the containing block.
// 1px = 0.75pt (ptPerCSSPx in css_review_02_test.go).
func TestBehaviorSizeShorthandSetsUsedBorderBox(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../testdata/css/v012/size/page.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	res := layoutHTML(t, string(raw))

	behaviorBoxModelCheckUsedSize(t, res, "pair", 120, 40)
	behaviorBoxModelCheckUsedSize(t, res, "square", 120, 120)
	behaviorBoxModelCheckUsedSize(t, res, "auto", 400, 0)
	behaviorBoxModelCheckUsedSize(t, res, "pct", 200, 50)
	behaviorBoxModelCheckUsedSize(t, res, "kept", 80, 10)
	behaviorBoxModelCheckUsedSize(t, res, "bad", 90, 30)
}

// TestBehaviorMinSizeClampsUsedBorderBox is the min-size shorthand.
// One length is copied onto both axes. Two lengths set width, then height.
// auto clears both floors. A bad token or a negative length writes nothing,
// so an earlier floor stays. A percentage resolves against the containing
// block on each axis. Lengths are CSS pixels (1px = 0.75pt).
func TestBehaviorMinSizeClampsUsedBorderBox(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../testdata/css/v012/min-size/page.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	fixture := layoutHTML(t, string(raw))
	minSizeAssertBorderBox(t, fixture, "item", 120, 120)

	cases := []struct {
		name  string
		html  string
		wantW float64
		wantH float64
	}{
		{
			name: "two lengths set width then height",
			html: minSizePage(`<div id="outer" style="width:400px">` +
				`<div id="item" style="width:50px;height:20px;min-size:120px 80px"></div>` +
				`</div>`),
			wantW: 120,
			wantH: 80,
		},
		{
			name:  "auto keeps the specified box",
			html:  minSizePage(`<div id="item" style="width:50px;height:20px;min-size:auto"></div>`),
			wantW: 50,
			wantH: 20,
		},
		{
			name:  "bad second token keeps the earlier min-width",
			html:  minSizePage(`<div id="item" style="width:4px;height:20px;min-width:10px;min-size:120px bogus"></div>`),
			wantW: 10,
			wantH: 20,
		},
		{
			name: "percentage resolves against the containing block",
			html: minSizePage(`<div id="outer" style="width:200px;height:100px">` +
				`<div id="item" style="width:20px;height:10px;min-size:50%"></div>` +
				`</div>`),
			wantW: 100,
			wantH: 50,
		},
		{
			name:  "negative length keeps the earlier floor",
			html:  minSizePage(`<div id="item" style="width:4px;height:6px;min-size:40px 30px;min-size:-10px"></div>`),
			wantW: 40,
			wantH: 30,
		},
		{
			name: "negative second token writes neither axis",
			html: minSizePage(`<div id="item" style="width:4px;height:6px;min-width:10px;min-height:12px;` +
				`min-size:90px -8px"></div>`),
			wantW: 10,
			wantH: 12,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := layoutHTML(t, testCase.html)
			minSizeAssertBorderBox(t, res, "item", testCase.wantW, testCase.wantH)
		})
	}
}

func minSizePage(inner string) string {
	return `<html style="margin:0"><body style="margin:0">` + inner + `</body></html>`
}

func minSizeAssertBorderBox(t *testing.T, res *Result, elementID string, wantWidthPx, wantHeightPx float64) {
	t.Helper()

	item := boxByID(t, res, elementID)
	if !near(item.w, pxToPt(wantWidthPx)) || !near(item.height, pxToPt(wantHeightPx)) {
		t.Errorf("#%s used border box = %.4fpt x %.4fpt (%.2fpx x %.2fpx), want %.2fpx x %.2fpx",
			elementID, item.w, item.height, item.w/ptPerCSSPx, item.height/ptPerCSSPx,
			wantWidthPx, wantHeightPx)
	}
}

// TestBehaviorMaxSizeClampsUsedBorderBox is the max-size shorthand. It copies
// one value onto both axes and two values onto max-width then max-height.
// The used border box is CSS pixels. none stores the same unset ceiling the
// longhand setters use (MaxWidth and MaxHeight at -1).
func TestBehaviorMaxSizeClampsUsedBorderBox(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../testdata/css/v012/max-size/page.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	res := layoutHTML(t, string(raw))

	behaviorBoxModelCheckUsedSize(t, res, "item", 180, 60)
	behaviorBoxModelCheckUsedSize(t, res, "square", 100, 100)
	behaviorBoxModelCheckUsedSize(t, res, "open", 300, 200)
	behaviorBoxModelCheckUsedSize(t, res, "kept", 40, 200)
	behaviorBoxModelCheckUsedSize(t, res, "pct", 200, 100)

	open := boxByID(t, res, "open")
	if open.style.MaxWidth != -1 || open.style.MaxHeight != -1 {
		t.Errorf("max-size:none MaxWidth=%v MaxHeight=%v, want -1 and -1",
			open.style.MaxWidth, open.style.MaxHeight)
	}

	kept := boxByID(t, res, "kept")
	if !near(kept.style.MaxWidth, pxToPt(40)) || kept.style.MaxHeight != -1 {
		t.Errorf("bad max-size MaxWidth=%v (want 40px) MaxHeight=%v (want -1)",
			kept.style.MaxWidth, kept.style.MaxHeight)
	}

	item := boxByID(t, res, "item")
	if !near(item.style.MaxWidth, pxToPt(180)) || !near(item.style.MaxHeight, pxToPt(60)) {
		t.Fatalf("max-size 180px 60px MaxWidth=%v MaxHeight=%v, want 180px and 60px",
			item.style.MaxWidth, item.style.MaxHeight)
	}

	var ctx styleContext

	applyStyleProp(item.style, "max-size", "none", item.style.FontSize, &ctx, nil, false)

	if item.style.MaxWidth != -1 || item.style.MaxHeight != -1 ||
		item.style.MaxWidthPercent != -1 || item.style.MaxHeightPercent != -1 {
		t.Errorf("none after max-size MaxWidth=%v MaxHeight=%v percents %v %v, want -1",
			item.style.MaxWidth, item.style.MaxHeight,
			item.style.MaxWidthPercent, item.style.MaxHeightPercent)
	}

	pct := boxByID(t, res, "pct")
	if pct.style.MaxWidthPercent != 50 || pct.style.MaxHeightPercent != 50 ||
		pct.style.MaxWidth != -1 || pct.style.MaxHeight != -1 {
		t.Errorf("max-size 50%% stored width %v/%v%% height %v/%v%%, want percent 50 and absolute -1",
			pct.style.MaxWidth, pct.style.MaxWidthPercent,
			pct.style.MaxHeight, pct.style.MaxHeightPercent)
	}

	applyStyleProp(pct.style, "max-size", "none", pct.style.FontSize, &ctx, nil, false)

	if pct.style.MaxWidth != -1 || pct.style.MaxHeight != -1 ||
		pct.style.MaxWidthPercent != -1 || pct.style.MaxHeightPercent != -1 {
		t.Errorf("none after percentage max-size MaxWidth=%v MaxHeight=%v percents %v %v, want -1",
			pct.style.MaxWidth, pct.style.MaxHeight,
			pct.style.MaxWidthPercent, pct.style.MaxHeightPercent)
	}
}
