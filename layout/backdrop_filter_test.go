package layout_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

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

// TestBehaviorBackdropFilterKeepsAuthorString proves the drawing list keeps
// the author filter list and does not paint backdrop pixels. blur(1.5em)
// saturate(150%) is stored unchanged, in that order. none and an empty value
// do not open a backdrop group. A later bad function does not replace an
// earlier valid list. z-index on the group stays the value the element set.
func TestBehaviorBackdropFilterKeepsAuthorString(t *testing.T) {
	t.Parallel()

	const author = "blur(1.5em) saturate(150%)"

	t.Run("fixture", func(t *testing.T) {
		t.Parallel()

		raw, err := os.ReadFile(filepath.Join("..", "testdata", "css", "v012", "backdrop-filter", "page.html"))
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}

		groups := backdropGroupBegins(displayOf(t, string(raw)))
		if len(groups) != 1 {
			t.Fatalf("backdrop groups = %d, want 1 (none must not open a group)", len(groups))
		}

		got := groups[0].Group().BackdropFilter
		if got != author {
			t.Fatalf("BackdropFilter = %q, want %q", got, author)
		}

		if groups[0].W <= 0 || groups[0].H <= 0 {
			t.Fatalf("backdrop group box = %.2fx%.2f, want non-empty", groups[0].W, groups[0].H)
		}

		if groups[0].ZIndex != 4 || !groups[0].ZIndexSet {
			t.Fatalf("backdrop group z-index = %d (set %t), want 4", groups[0].ZIndex, groups[0].ZIndexSet)
		}
	})

	t.Run("reverse order kept", func(t *testing.T) {
		t.Parallel()

		const reversed = "saturate(150%) blur(1.5em)"

		display := displayOf(t, `<div style="position:absolute;left:16px;top:12px;width:160px;height:80px;`+
			`backdrop-filter:`+reversed+`;background:rgba(255,255,255,.2)"></div>`)

		groups := backdropGroupBegins(display)
		if len(groups) != 1 {
			t.Fatalf("backdrop groups = %d, want 1", len(groups))
		}

		if got := groups[0].Group().BackdropFilter; got != reversed {
			t.Fatalf("BackdropFilter = %q, want %q", got, reversed)
		}
	})

	t.Run("bad token keeps earlier list", func(t *testing.T) {
		t.Parallel()

		display := displayOf(t, `<div style="position:absolute;left:16px;top:12px;width:160px;height:80px;z-index:4;`+
			`backdrop-filter:`+author+`;backdrop-filter:blur(bogus);background:rgba(255,255,255,.2)"></div>`)

		groups := backdropGroupBegins(display)
		if len(groups) != 1 {
			t.Fatalf("backdrop groups = %d, want 1", len(groups))
		}

		if got := groups[0].Group().BackdropFilter; got != author {
			t.Fatalf("BackdropFilter = %q, want earlier %q", got, author)
		}

		if groups[0].ZIndex != 4 || !groups[0].ZIndexSet {
			t.Fatalf("backdrop group z-index = %d (set %t), want 4", groups[0].ZIndex, groups[0].ZIndexSet)
		}
	})

	t.Run("none", func(t *testing.T) {
		t.Parallel()

		display := displayOf(t, `<div style="position:absolute;left:16px;top:12px;width:160px;height:80px;`+
			`backdrop-filter:none;background:rgba(255,255,255,.2)"></div>`)

		if groups := backdropGroupBegins(display); len(groups) != 0 {
			t.Fatalf("backdrop-filter:none opened %d group(s)", len(groups))
		}
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		display := displayOf(t, `<div style="position:absolute;left:16px;top:12px;width:160px;height:80px;`+
			`backdrop-filter:;background:rgba(255,255,255,.2)"></div>`)

		if groups := backdropGroupBegins(display); len(groups) != 0 {
			t.Fatalf("empty backdrop-filter opened %d group(s)", len(groups))
		}
	})
}

func backdropGroupBegins(display *layout.Display) []*layout.DisplayOp {
	begins := make([]*layout.DisplayOp, 0, len(display.Order))

	for _, index := range display.Order {
		if index < 0 || index >= len(display.Ops) {
			continue
		}

		paintOp := &display.Ops[index]
		if !paintOp.IsGroupBegin() || paintOp.Group() == nil || paintOp.Group().BackdropFilter == "" {
			continue
		}

		begins = append(begins, paintOp)
	}

	return begins
}
