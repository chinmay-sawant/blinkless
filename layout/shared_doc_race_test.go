package layout_test

import (
	"sync"
	"testing"

	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/layout"
)

// TestDisplayListConcurrentSharedDocument lays out one shared document from
// many goroutines. Small attribute lists stay map-free after parsing, and
// read-only passes must not materialize Node.Attrs lazily (data race).
func TestDisplayListConcurrentSharedDocument(t *testing.T) {
	t.Parallel()

	const source = `<div id="a" title="t" lang="en"><p class="x" data-k="v">hi</p>` +
		`<svg width="20" height="20" viewBox="0 0 20 20"><rect width="10" height="10" fill="red"/></svg></div>`

	tree, err := html.Parse([]byte(source))
	if err != nil {
		t.Fatal(err)
	}

	styled := mustApply(t, tree, 400, 300, "", "", "")

	var wg sync.WaitGroup

	for range 8 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if _, err := layout.DisplayListOptions(t.Context(), styled, layout.Options{RetainState: true}); err != nil {
				t.Error(err)
			}
		}()
	}

	wg.Wait()
}
