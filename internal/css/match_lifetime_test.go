package css

import (
	"runtime"
	"testing"
	"weak"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

//nolint:paralleltest // checks the process-wide selector memos across garbage collection
func TestSelectorMemosDoNotKeepDroppedTreesAlive(t *testing.T) {
	resetMatchMemoForTest()
	defer resetMatchMemoForTest()
	probe := func() weak.Pointer[html.Node] {
		parent, kids := evictProbeParent()
		siblingEntryFor(parent)
		getCachedLoweredAttr(kids[0], "data-label", "UPPERCASE")
		return weak.Make(parent)
	}()
	for range 3 {
		runtime.GC()
	}
	if probe.Value() != nil {
		t.Fatal("selector memos retained a discarded tree")
	}
}
