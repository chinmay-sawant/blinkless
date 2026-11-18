package layout

import (
	"sync"
	"testing"
)

// TestLineHeightAlternatingStylesDoNotAllocate covers the regular/bold pattern
// in list rows: both shared entries are warm, but every call changes the last
// style and therefore needs the shared lookup.
//
//nolint:paralleltest // isolates shared caches and measures process-wide allocations
func TestLineHeightAlternatingStylesDoNotAllocate(t *testing.T) {
	faces, err := loadHotFaces(t)
	if err != nil {
		t.Fatal(err)
	}
	clearLineHeightCache()
	lastLineHeight.Store(nil)
	t.Cleanup(func() {
		clearLineHeightCache()
		lastLineHeight.Store(nil)
	})

	eng := &engine{font: faces.Regular, faces: faces, scale: 1}
	regular := hotStyle(16, "", "")
	bold := regular
	bold.FontWeight = fontWeightBoldValue
	if eng.faceFor(&regular) == eng.faceFor(&bold) {
		t.Fatal("fixture did not resolve distinct regular and bold faces")
	}
	wantRegular := eng.lineHeightOf(&regular)
	wantBold := eng.lineHeightOf(&bold)
	var gotRegular, gotBold float64
	allocs := testing.AllocsPerRun(100, func() {
		gotRegular = eng.lineHeightOf(&regular)
		gotBold = eng.lineHeightOf(&bold)
	})
	if gotRegular != wantRegular || gotBold != wantBold {
		t.Fatal("warm alternating styles changed the measured line heights")
	}
	if allocs != 0 {
		t.Fatalf("warm regular/bold pair allocated %.1f objects, want 0", allocs)
	}
}

// TestLineHeightSharedEntriesStayImmutable requires replacement and removal
// from the bounded map to leave previously published entries readable.
//
//nolint:paralleltest // replaces entries in the shared line-height cache
func TestLineHeightSharedEntriesStayImmutable(t *testing.T) {
	faces, err := loadHotFaces(t)
	if err != nil {
		t.Fatal(err)
	}
	clearLineHeightCache()
	lastLineHeight.Store(nil)
	t.Cleanup(func() {
		clearLineHeightCache()
		lastLineHeight.Store(nil)
	})
	eng := &engine{font: faces.Regular, faces: faces, scale: 1}
	style := hotStyle(16, "", "")
	want := eng.lineHeightOf(&style)
	entry, found := lookupLineHeight(eng.faceFor(&style), style.FontSize, eng.scale)
	if !found || entry == nil {
		t.Fatal("initial line height was not retained")
	}
	replacement := &lastLineHeightEntry{key: entry.key, height: want + 1}
	storeLineHeight(replacement)
	got, found := lookupLineHeight(entry.key.face, style.FontSize, eng.scale)
	if !found || got != replacement || entry.height != want {
		t.Fatal("replacing the map entry changed an already published height")
	}
	clearLineHeightCache()
	if entry.height != want || lastLineHeight.Load() != entry {
		t.Fatal("removing the map entry invalidated the atomic cache's retained entry")
	}
}

// TestLineHeightConcurrentStyleLookupsMatchWarmValues exercises shared entry
// publication while every layout keeps its own mutable face caches.
//
//nolint:paralleltest // controls shared cache setup before concurrent readers
func TestLineHeightConcurrentStyleLookupsMatchWarmValues(t *testing.T) {
	faces, err := loadHotFaces(t)
	if err != nil {
		t.Fatal(err)
	}
	styles := [2]ResolvedStyle{hotStyle(16, "", ""), hotStyle(16, "", "")}
	styles[1].FontWeight = fontWeightBoldValue
	reference := &engine{font: faces.Regular, faces: faces, scale: 1}
	want := [2]float64{reference.lineHeightOf(&styles[0]), reference.lineHeightOf(&styles[1])}
	const workers = 8
	results := make(chan bool, workers)
	var done sync.WaitGroup
	for range workers {
		done.Add(1)
		go func() {
			defer done.Done()
			eng := &engine{font: faces.Regular, faces: faces, scale: 1}
			for index := range 100 {
				style := index % len(styles)
				if eng.lineHeightOf(&styles[style]) != want[style] {
					results <- false
					return
				}
			}
			results <- true
		}()
	}
	done.Wait()
	close(results)
	for matched := range results {
		if !matched {
			t.Fatal("concurrent style lookup changed a warmed line height")
		}
	}
}
