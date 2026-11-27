package layout_test

import (
	"bytes"
	"context"
	"strconv"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

func buildCullTestDocument(tb testing.TB, rowCount int) *css.Document {
	tb.Helper()

	var buf bytes.Buffer
	buf.WriteString(`<!DOCTYPE html><html><head><style>` +
		`html,body{margin:0;font-family:sans-serif;color:#1c1915;background:#f4f1ea}` +
		`#head{height:40px;background:#1c1915;color:#ffffff}` +
		`.row{display:flex;box-sizing:border-box;height:40px;gap:12px;padding:8px 16px;border-bottom:1px solid #ddd6c7}` +
		`.num{width:64px}.title{width:200px;font-weight:bold}.body{color:#4a463c}` +
		`</style></head><body><div id="head">Cull test</div>`)

	for i := 1; i <= rowCount; i++ {
		s := strconv.Itoa(i)
		buf.WriteString(`<div id="row-` + s + `" class="row"><span class="num">` + s +
			`</span><span class="title">Item ` + s + `</span><span class="body">Row payload ` + s + `</span></div>`)
	}

	buf.WriteString(`</body></html>`)

	doc, err := html.Parse(buf.Bytes())
	if err != nil {
		tb.Fatalf("html.Parse failed: %v", err)
	}

	styled, err := css.Apply(context.Background(), doc, css.Options{
		WidthPx:  800,
		HeightPx: 600,
		Media:    "screen",
	})
	if err != nil {
		tb.Fatalf("css.Apply failed: %v", err)
	}

	return styled
}

func TestDisplayListCullWindow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	styled := buildCullTestDocument(t, 200)

	full, err := layout.DisplayListOptions(ctx, styled, layout.Options{})
	if err != nil {
		t.Fatalf("full DisplayListOptions failed: %v", err)
	}
	defer full.Release()

	culled, err := layout.DisplayListOptions(ctx, styled, layout.Options{
		CullWindow: &layout.CullWindow{TopPx: 0, BottomPx: 600},
	})
	if err != nil {
		t.Fatalf("culled DisplayListOptions failed: %v", err)
	}
	defer culled.Release()

	if culled.Height != full.Height {
		t.Fatalf("culled Height = %d, want %d", culled.Height, full.Height)
	}

	if len(culled.Ops) >= len(full.Ops)/4 {
		t.Fatalf("culled emitted %d ops, full emitted %d; expected significant culling", len(culled.Ops), len(full.Ops))
	}

	foundVisible := false
	foundOffscreen := false
	for _, op := range culled.Ops {
		if op.Text == "Item 1" {
			foundVisible = true
		}
		if op.Text == "Item 150" {
			foundOffscreen = true
		}
	}

	if !foundVisible {
		t.Fatal("visible row Item 1 missing from culled ops")
	}
	if foundOffscreen {
		t.Fatal("off-screen row Item 150 should have been culled")
	}
}

func BenchmarkDisplayListCullWindow1000(b *testing.B) {
	ctx := context.Background()
	styled := buildCullTestDocument(b, 1000)

	b.Run("Full", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			d, err := layout.DisplayListOptions(ctx, styled, layout.Options{})
			if err != nil {
				b.Fatal(err)
			}
			d.Release()
		}
	})

	b.Run("CulledViewport600px", func(b *testing.B) {
		b.ReportAllocs()
		opts := layout.Options{
			CullWindow: &layout.CullWindow{TopPx: 0, BottomPx: 600},
		}
		for b.Loop() {
			d, err := layout.DisplayListOptions(ctx, styled, opts)
			if err != nil {
				b.Fatal(err)
			}
			d.Release()
		}
	})
}
