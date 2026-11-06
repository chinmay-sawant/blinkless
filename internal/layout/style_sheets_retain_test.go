package layout

import (
	"runtime"
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

// TestSharedBundleRetainsSheets proves the bundle pins the exact sheet set it
// was built from. The bundle key is pointer identity plus rule count, so
// without retention a freed sheet address recycled by a different sheet would
// hit a stale bundle and resolve silently wrong styles. Retention plus capped
// eviction closes that window: an entry cannot outlive its addresses.
func TestSharedBundleRetainsSheets(t *testing.T) {
	// Sequential on purpose: the shared bundle map is process-global with a
	// bounded size, so parallel tests compiling other sheet sets could evict
	// the entry between two calls and make a pointer comparison flaky.
	first, err := css.Parse(`.a { color: red; }`)
	if err != nil {
		t.Fatal(err)
	}
	sheets := []*css.Stylesheet{first}
	bundle := sharedCompiledFor(sheets)
	if bundle == nil {
		t.Fatal("no bundle for one sheet")
	}
	if len(bundle.sheets) != 1 || bundle.sheets[0] != first {
		t.Fatalf("bundle retains %d sheets, want the exact input sheet", len(bundle.sheets))
	}
}

// TestSharedBundleStaysCorrectAcrossGCChurn lays out two different sheet sets
// with allocation churn and garbage collections between them, then asserts
// both still resolve their own rules. Before retention this pattern
// intermittently resolved the second layout with the first bundle.
func TestSharedBundleStaysCorrectAcrossGCChurn(t *testing.T) {
	// Sequential on purpose: see above. The churn plus forced GC makes a
	// use-after-free style collision likely without retention; with the
	// bundle pinning its sheets the lookups below are deterministic.
	red, err := css.Parse(`.tag { color: red; }`)
	if err != nil {
		t.Fatal(err)
	}
	blue, err := css.Parse(`.tag { color: blue; }`)
	if err != nil {
		t.Fatal(err)
	}
	redBundle := sharedCompiledFor([]*css.Stylesheet{red})
	blueBundle := sharedCompiledFor([]*css.Stylesheet{blue})
	if redBundle == blueBundle {
		t.Fatal("distinct sheet sets share one bundle")
	}
	red, blue = nil, nil
	for range 64 {
		_ = make([]byte, 1<<16)
	}
	runtime.GC()
	if got := sharedCompiledFor(redBundle.sheets); got != redBundle {
		t.Fatal("red bundle lost after churn")
	}
	if got := sharedCompiledFor(blueBundle.sheets); got != blueBundle {
		t.Fatal("blue bundle lost after churn")
	}
}
