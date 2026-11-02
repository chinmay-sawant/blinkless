package layout_test

import "testing"

func TestDisplayBackdropFilterCarriesHostEffect(t *testing.T) {
	t.Parallel()

	display := displayOf(t, `<div style="position:absolute;left:16px;top:12px;width:160px;height:80px;`+
		`border-radius:16px;backdrop-filter:blur(12px) saturate(1.25);background:rgba(255,255,255,.2)"></div>`)

	for _, index := range display.Order {
		if index < 0 || index >= len(display.Ops) {
			continue
		}

		paintOp := &display.Ops[index]
		if !paintOp.IsGroupBegin() || paintOp.Group() == nil || paintOp.Group().BackdropFilter == "" {
			continue
		}

		if got := paintOp.Group().BackdropFilter; got != "blur(12px) saturate(1.25)" {
			t.Fatalf("backdrop-filter = %q, want blur(12px) saturate(1.25)", got)
		}

		if paintOp.W <= 0 || paintOp.H <= 0 {
			t.Fatalf("backdrop-filter group box = %.2fx%.2f, want non-empty", paintOp.W, paintOp.H)
		}

		return
	}

	t.Fatal("backdrop-filter was not preserved on a display group")
}

func TestBackdropFilterKeepsContainingZIndex(t *testing.T) {
	t.Parallel()

	display := displayOf(t, `<div style="position:fixed;left:0;top:0;width:320px;height:240px;z-index:60">`+
		`<div style="position:absolute;left:16px;top:12px;width:160px;height:80px;`+
		`backdrop-filter:blur(12px);background:rgba(255,255,255,.2)"></div></div>`)

	for _, paintOp := range display.Ops {
		if !paintOp.IsGroupBegin() || paintOp.Group() == nil || paintOp.Group().BackdropFilter == "" {
			continue
		}

		if paintOp.ZIndex != 60 || !paintOp.ZIndexSet {
			t.Fatalf("backdrop-filter group z-index = %d (set %t), want inherited 60", paintOp.ZIndex, paintOp.ZIndexSet)
		}

		return
	}

	t.Fatal("backdrop-filter group marker was not emitted")
}
