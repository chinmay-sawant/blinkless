package layout

import (
	"runtime"
	"testing"
	"weak"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

//nolint:paralleltest // checks layout cache lifetime across garbage collection
func TestGridIntrinsicsDoNotKeepReleasedTreesAlive(t *testing.T) {
	probe := func() weak.Pointer[html.Node] {
		root := mustParse(t, gridMemoNestedDoc)
		result, err := Layout(root, Options{Width: 600, Height: 400, Sheets: sheetSlice(sheet(t, gridMemoNestedCSS))})
		if err != nil {
			t.Fatal(err)
		}
		result.Release()
		return weak.Make(root)
	}()
	for range 3 {
		runtime.GC()
	}
	if probe.Value() != nil {
		t.Fatal("grid intrinsic memo retained a discarded tree and its style chunks")
	}
}
