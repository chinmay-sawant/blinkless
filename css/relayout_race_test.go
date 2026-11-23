package css_test

import (
	"encoding/base64"
	"sync"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
)

// TestRelayoutConcurrentSharesSheetCache reproduces the concurrent map
// read/write on the SheetCache shared by every Relayout result. Run with
// -race; the @media-gated @import forces cache writes on first activation.
func TestRelayoutConcurrentSharesSheetCache(t *testing.T) {
	t.Parallel()

	body := base64.StdEncoding.EncodeToString([]byte("#bar { color: #123456 }"))
	source := `<style>@import url("data:text/css;base64,` + body +
		`") (min-width:400px);</style><style>#bar{color:red}</style><div id="bar" class="a">x</div>`

	styled, err := css.Apply(t.Context(), mustDoc(t, source),
		css.Options{WidthPx: 200, HeightPx: 100, Media: "screen"})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup

	for i := range 16 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for j := range 20 {
				width := 200
				if (i+j)%2 == 0 {
					width = 800
				}

				if _, err := css.Relayout(t.Context(), styled, width, 600, "", "", ""); err != nil {
					t.Error(err)

					return
				}
			}
		}()
	}

	wg.Wait()
}

// TestHoverChangedScopesConcurrentSharedDocument reproduces the lazy
// Node.EnsureAttrs map materialization racing across goroutines.
func TestHoverChangedScopesConcurrentSharedDocument(t *testing.T) {
	t.Parallel()

	source := `<style>.a:hover{color:red}</style><div id="x"><p class="a" title="t" data-k="v">hi</p><p class="a" lang="en">yo</p></div>`

	styled, err := css.Apply(t.Context(), mustDoc(t, source),
		css.Options{WidthPx: 200, HeightPx: 100, Media: "screen", Hover: "x"})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup

	for range 16 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			styled.HoverChangedScopes("")
		}()
	}

	wg.Wait()
}
