package html

import (
	"testing"
	"time"
)

// TestFindMissUsesReadLock holds the index read lock: a miss for an absent id
// must answer without waiting for the write lock or walking the tree.
func TestFindMissUsesReadLock(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte(`<div id="a"><p id="b">x</p></div>`))
	if err != nil {
		t.Fatal(err)
	}

	idx := elementIndexFor(doc)
	idx.mu.RLock()

	done := make(chan bool, 1)

	go func() {
		_, ok := doc.Find("missing")
		done <- ok
	}()

	select {
	case ok := <-done:
		if ok {
			t.Error("found a missing id")
		}
	case <-time.After(2 * time.Second):
		idx.mu.RUnlock()
		t.Fatal("negative Find blocked on the index write lock")
	}

	idx.mu.RUnlock()
}

// TestFindMissStaysCorrectAfterEdits covers the cases that invalidate the
// complete table: an id rename and a dropped subtree with a later duplicate.
func TestFindMissStaysCorrectAfterEdits(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte(`<div id="box"><p id="dup">one</p></div><p id="dup">two</p>`))
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := doc.Find("renamed"); ok {
		t.Fatal("unexpected hit")
	}

	if !doc.SetAttribute("box", "id", "renamed") {
		t.Fatal("SetAttribute failed")
	}

	if _, ok := doc.Find("renamed"); !ok {
		t.Fatal("renamed id not found after rename")
	}

	if _, ok := doc.Find("box"); ok {
		t.Fatal("old id still found")
	}

	// Dropping the first dup must surface the later duplicate.
	if !doc.SetText("renamed", "gone") {
		t.Fatal("SetText failed")
	}

	el, ok := doc.Find("dup")
	if !ok || el.Text != "two" {
		t.Fatalf("dup after drop = %+v, %v", el, ok)
	}
}
