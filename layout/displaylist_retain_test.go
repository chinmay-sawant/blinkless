package layout_test

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"strings"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

// retainSource is a page with one element a host can patch.
const retainSource = `<div id="box" style="width:120px;height:40px">Hello</div>`

// TestRelayoutElementNeedsRetainedState is the opt-in gate for the patch
// path. A display laid out the default way keeps no retained layout state,
// so the engine has no box tree to re-place the element inside and refuses
// the patch. A display laid out with Options.RetainState set patches, and
// the drawing list it carries is the same list the default display carries.
func TestRelayoutElementNeedsRetainedState(t *testing.T) {
	t.Parallel()

	tree, err := html.Parse([]byte(retainSource))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")

	plain := layOf(t, styled)
	retained := layRetained(t, styled)

	compareDisplays(t, retained, plain)

	if !tree.SetText("box", "Hello world") {
		t.Fatal("SetText reported no element")
	}

	_, err = layout.RelayoutElement(t.Context(), styled, plain, "box", layout.Options{})
	if !errors.Is(err, layout.ErrNoRetainedState) {
		t.Fatalf("error = %v, want %v", err, layout.ErrNoRetainedState)
	}

	patch, err := layout.RelayoutElement(t.Context(), styled, retained, "box", layout.Options{})
	if err != nil {
		t.Fatalf("relayout element: %v", err)
	}

	if len(patch.Ops) == 0 {
		t.Fatal("patch carries no operations")
	}

	if got := patchText(patch); got != "Hello world" {
		t.Fatalf("patch text = %q, want %q", got, "Hello world")
	}
}

// TestRetainedStateNeedsTheHostToAsk pins the other half of the gate: the
// retained state is not free, and it is not taken by accident. A display laid
// out with the option off holds nothing the patch path can use, and a
// display laid out with it on holds the placement the patch path needs.
func TestRetainedStateNeedsTheHostToAsk(t *testing.T) {
	t.Parallel()

	tree, err := html.Parse([]byte(retainSource))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")

	plain := layOf(t, styled)

	if !tree.SetText("box", "Hello world") {
		t.Fatal("SetText reported no element")
	}

	// The default display refuses the patch, and the refusal names the
	// missing state, so a host can tell it apart from a missing element.
	_, err = layout.RelayoutElement(t.Context(), styled, plain, "box", layout.Options{})
	if !errors.Is(err, layout.ErrNoRetainedState) {
		t.Fatalf("error = %v, want %v", err, layout.ErrNoRetainedState)
	}

	if errors.Is(err, layout.ErrNoElement) {
		t.Fatalf("error = %v, want it to differ from %v", err, layout.ErrNoElement)
	}
}

// retainPage returns a page with the given number of cards. Each card holds
// a heading, a paragraph, and a list of three items, so a card is seven
// elements. The parser adds the html, body, and style elements around them.
func retainPage(cards int) string {
	var page strings.Builder

	page.WriteString(`<style>.card{border:1px solid #333333;padding:4px;margin:2px}</style>`)

	for index := range cards {
		fmt.Fprintf(&page,
			`<div class="card" id="card-%d"><h3>Card %d</h3><p>Body %d</p><ul><li>One</li><li>Two</li><li>Three</li></ul></div>`,
			index, index, index)
	}

	return page.String()
}

// retainStyled parses and applies a page of the given number of cards.
func retainStyled(tb testing.TB, cards int) *css.Document {
	tb.Helper()

	tree, err := html.Parse([]byte(retainPage(cards)))
	if err != nil {
		tb.Fatalf("parse: %v", err)
	}

	styled, err := css.Apply(tb.Context(), tree, css.Options{WidthPx: 800, HeightPx: 600, Media: "screen"})
	if err != nil {
		tb.Fatalf("apply: %v", err)
	}

	return styled
}

// retainedBytes returns the live heap bytes the value build returns keeps
// alive. It collects before and after the build, so the number is what the
// caller still holds, not what the build allocated along the way.
func retainedBytes(build func() any) int64 {
	runtime.GC()

	var before runtime.MemStats

	runtime.ReadMemStats(&before)

	kept := build()

	runtime.GC()

	var after runtime.MemStats

	runtime.ReadMemStats(&after)

	runtime.KeepAlive(kept)

	if after.HeapAlloc < before.HeapAlloc {
		return 0
	}

	grown := after.HeapAlloc - before.HeapAlloc
	if grown > math.MaxInt64 {
		return math.MaxInt64
	}

	return int64(grown)
}

// BenchmarkDisplayRetainedBytes measures the heap a laid-out page keeps
// alive with the retained layout state on and off. The off number is what a
// host that only draws the page pays. The on number is what a host that
// patches an element pays on top of it.
//
// The measurement reads the process heap, so run it alone:
//
//	go test ./layout -run '^$' -bench BenchmarkDisplayRetainedBytes -benchtime=5x
func BenchmarkDisplayRetainedBytes(b *testing.B) {
	for _, cards := range []int{4, 40, 100} {
		b.Run(fmt.Sprintf("cards-%d", cards), func(b *testing.B) {
			styled := retainStyled(b, cards)

			// Warm the font registry and the style caches, so the first
			// measured layout does not carry their one-time cost.
			warm, err := layout.DisplayList(b.Context(), styled)
			if err != nil {
				b.Fatalf("display: %v", err)
			}

			runtime.KeepAlive(warm)

			b.Run("plain", func(b *testing.B) {
				var total int64

				for range b.N {
					total += retainedBytes(func() any {
						display, err := layout.DisplayList(b.Context(), styled)
						if err != nil {
							b.Fatalf("display: %v", err)
						}

						b.ReportMetric(float64(len(display.Boxes)), "boxes/page")

						return display
					})
				}

				b.ReportMetric(float64(total)/float64(b.N), "retained-B/page")
			})

			b.Run("retained", func(b *testing.B) {
				var total int64

				for range b.N {
					total += retainedBytes(func() any {
						display, err := layout.DisplayListOptions(b.Context(), styled, layout.Options{RetainState: true})
						if err != nil {
							b.Fatalf("display: %v", err)
						}

						b.ReportMetric(float64(len(display.Boxes)), "boxes/page")

						return display
					})
				}

				b.ReportMetric(float64(total)/float64(b.N), "retained-B/page")
			})
		})
	}
}
