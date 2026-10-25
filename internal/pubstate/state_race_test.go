// Package pubstate_test checks the registration from outside the package so
// it can import the public html and css packages that register the readers.
package pubstate_test

import (
	"sync"
	"testing"

	"github.com/chinmay-sawant/blinkless/css"
	"github.com/chinmay-sawant/blinkless/html"
	"github.com/chinmay-sawant/blinkless/internal/pubstate"
)

const (
	writerCount = 4
	readerCount = 4
	iterations  = 25
)

// TestConcurrentApplyAndReads runs the registration call sites in html.Parse
// and css.Apply beside readers of pubstate.Root and pubstate.StyledOf. It is a
// race regression test: registration used to happen on every Parse and Apply,
// so a layout goroutine reading StyledOf raced with a render goroutine
// applying CSS. Run it with -race.
func TestConcurrentApplyAndReads(t *testing.T) {
	t.Parallel()

	source := []byte(`<!DOCTYPE html><style>p { color: red }</style><p id="x">hello`)
	sharedHTML, sharedStyled := preparedDocuments(t, source)
	start := make(chan struct{})

	var group sync.WaitGroup

	group.Add(writerCount)

	for range writerCount {
		go func() {
			defer group.Done()

			<-start
			parseAndApply(t, source)
		}()
	}

	group.Add(readerCount)

	for range readerCount {
		go func() {
			defer group.Done()

			<-start
			readShared(t, sharedHTML, sharedStyled)
		}()
	}

	close(start)
	group.Wait()
}

// preparedDocuments builds the immutable pair the reader goroutines inspect.
func preparedDocuments(t *testing.T, source []byte) (*html.Document, *css.Document) {
	t.Helper()

	doc, err := html.Parse(source)
	if err != nil {
		t.Fatalf("html.Parse: %v", err)
	}

	styled, err := css.Apply(t.Context(), doc, css.Options{WidthPx: 400, HeightPx: 200})
	if err != nil {
		t.Fatalf("css.Apply: %v", err)
	}

	if pubstate.Root(doc) == nil {
		t.Fatal("pubstate.Root: no parsed tree")
	}

	if _, ok := pubstate.StyledOf(styled); !ok {
		t.Fatal("pubstate.StyledOf: no cascade result")
	}

	return doc, styled
}

// parseAndApply is the writer side of the race: every iteration still calls
// the two public entry points that used to rewrite the readers.
func parseAndApply(t *testing.T, source []byte) {
	t.Helper()

	for range iterations {
		doc, err := html.Parse(source)
		if err != nil {
			t.Errorf("html.Parse: %v", err)

			return
		}

		styled, err := css.Apply(t.Context(), doc, css.Options{WidthPx: 400, HeightPx: 200})
		if err != nil {
			t.Errorf("css.Apply: %v", err)

			return
		}

		if pubstate.Root(doc) == nil {
			t.Error("pubstate.Root: no parsed tree")

			return
		}

		if _, ok := pubstate.StyledOf(styled); !ok {
			t.Error("pubstate.StyledOf: no cascade result")

			return
		}
	}
}

// readShared is the reader side of the race: it reads the registered readers
// while the writers parse and apply.
func readShared(t *testing.T, doc *html.Document, styled *css.Document) {
	t.Helper()

	for range iterations {
		if pubstate.Root(doc) == nil {
			t.Error("pubstate.Root: no parsed tree")

			return
		}

		if _, ok := pubstate.StyledOf(styled); !ok {
			t.Error("pubstate.StyledOf: no cascade result")

			return
		}
	}
}
