package layout_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

func buildList1000HTML(rowCount int) []byte {
	var b strings.Builder
	b.Grow(rowCount * 140)
	b.WriteString(`<!DOCTYPE html><html><head><style>
html,body{margin:0;font-family:sans-serif;color:#1c1915;background:#f4f1ea}
#head{height:40px;background:#1c1915;color:#fff}
.row{display:flex;box-sizing:border-box;height:40px;gap:12px;padding:8px 16px;border-bottom:1px solid #ddd6c7;background-color:#f4f1ea}
.row.selected{background-color:#e5dfd2}
.num{width:64px}.title{width:200px;font-weight:bold}.body{color:#4a463c}
</style></head><body><div id="head">Patch vs Rebuild</div>`)
	for i := 1; i <= rowCount; i++ {
		num := strconv.Itoa(i)
		b.WriteString(`<div id="row-`)
		b.WriteString(num)
		b.WriteString(`" class="row"><span class="num">`)
		b.WriteString(num)
		b.WriteString(`</span><span class="title">Item `)
		b.WriteString(num)
		b.WriteString(`</span><span id="body-`)
		b.WriteString(num)
		b.WriteString(`" class="body">Initial body `)
		b.WriteString(num)
		b.WriteString(`</span></div>`)
	}
	b.WriteString(`</body></html>`)
	return []byte(b.String())
}

func setupList1000(ctx context.Context, b *testing.B) (*html.Document, *css.Document, *layout.Display) {
	b.Helper()
	tree, err := html.Parse(buildList1000HTML(1000))
	if err != nil {
		b.Fatalf("html.Parse: %v", err)
	}
	styled, err := css.Apply(ctx, tree, css.Options{WidthPx: 800, HeightPx: 600, Media: "screen"})
	if err != nil {
		b.Fatalf("css.Apply: %v", err)
	}
	disp, err := layout.DisplayListOptions(ctx, styled, layout.Options{RetainState: true})
	if err != nil {
		b.Fatalf("DisplayListOptions: %v", err)
	}
	return tree, styled, disp
}

func BenchmarkRelayoutElements1000Rows(b *testing.B) {
	ctx := b.Context()
	opts := layout.Options{RetainState: true}

	b.Run("Text1Row", func(b *testing.B) {
		tree, styled, disp := setupList1000(ctx, b)
		defer func() { disp.Release() }()
		ids := []string{"body-4"}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			if i%2 == 0 {
				tree.SetText("body-4", "changed body A")
			} else {
				tree.SetText("body-4", "changed body B")
			}
			next, err := layout.RelayoutElements(ctx, styled, disp, ids, opts)
			if err != nil {
				b.Fatalf("RelayoutElements: %v", err)
			}
			disp.Release()
			disp = next
		}
	})

	b.Run("TextLastRow", func(b *testing.B) {
		tree, styled, disp := setupList1000(ctx, b)
		defer func() { disp.Release() }()
		ids := []string{"body-1000"}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			if i%2 == 0 {
				tree.SetText("body-1000", "changed body A")
			} else {
				tree.SetText("body-1000", "changed body B")
			}
			next, err := layout.RelayoutElements(ctx, styled, disp, ids, opts)
			if err != nil {
				b.Fatalf("RelayoutElements: %v", err)
			}
			disp.Release()
			disp = next
		}
	})

	b.Run("Text10Rows", func(b *testing.B) {
		tree, styled, disp := setupList1000(ctx, b)
		defer func() { disp.Release() }()
		ids := make([]string, 10)
		for j := range ids {
			ids[j] = fmt.Sprintf("body-%d", j+1)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			text := "changed body A"
			if i%2 != 0 {
				text = "changed body B"
			}
			for _, id := range ids {
				tree.SetText(id, text)
			}
			next, err := layout.RelayoutElements(ctx, styled, disp, ids, opts)
			if err != nil {
				b.Fatalf("RelayoutElements: %v", err)
			}
			disp.Release()
			disp = next
		}
	})

	b.Run("BgRestyle1Row", func(b *testing.B) {
		tree, styled, disp := setupList1000(ctx, b)
		defer func() { disp.Release() }()
		ids := []string{"row-4"}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			if i%2 == 0 {
				tree.SetAttribute("row-4", "class", "row selected")
			} else {
				tree.SetAttribute("row-4", "class", "row")
			}
			next, err := layout.RelayoutElements(ctx, styled, disp, ids, opts)
			if err != nil {
				b.Fatalf("RelayoutElements: %v", err)
			}
			disp.Release()
			disp = next
		}
	})
}
