package layout

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

func widthOverride(value string) []StyleOverride {
	return []StyleOverride{{Property: "width", Value: value}}
}

func opsSnapshot(res *Result) [][8]float64 {
	out := make([][8]float64, len(res.Ops))
	for i := range res.Ops {
		o := &res.Ops[i]
		out[i] = [8]float64{float64(o.Kind), o.X, o.Y, o.W, o.H, o.R, o.G, o.B}
	}

	return out
}

func compareOverrideResults(t *testing.T, step int, cached, fresh *Result) {
	t.Helper()

	got, want := opsSnapshot(cached), opsSnapshot(fresh)
	if len(got) != len(want) {
		t.Fatalf("step %d: %d ops with the cache, %d without", step, len(got), len(want))
	}

	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("step %d: op %d = %v with the cache, %v without", step, i, got[i], want[i])
		}
	}

	for _, id := range []string{"row", "card", "tail"} {
		a, b := relayoutBoxByID(t, cached, id), relayoutBoxByID(t, fresh, id)
		if a.x != b.x || a.y != b.y || a.w != b.w || a.height != b.height {
			t.Fatalf("step %d: box %q differs between the cached and the fresh pass", step, id)
		}
	}
}

// TestOverrideBaseMatchesFreshCascade runs a layout animation twice, once
// reusing the base styles between frames and once with the cache cleared
// before every frame. Every frame must produce the same boxes and operations,
// and the retained styles must be the base styles after each one.
func TestOverrideBaseMatchesFreshCascade(t *testing.T) {
	t.Parallel()

	opts := overrideOptions(t)

	rootCached, rootFresh := mustParse(t, overrideFlexHTML), mustParse(t, overrideFlexHTML)

	cached, err := Layout(rootCached, opts)
	if err != nil {
		t.Fatal(err)
	}

	fresh, err := Layout(rootFresh, opts)
	if err != nil {
		t.Fatal(err)
	}

	cardCached, cardFresh := relayoutNodeByID(t, rootCached, "card"), relayoutNodeByID(t, rootFresh, "card")
	baseWidth := cached.styles[cardCached].Width

	hits := 0
	overrideFrames := []string{"120px", "140px", "180px", "150px", "100px", "170px"}

	for step, value := range overrideFrames {
		if cached.overrideBase != nil {
			hits++
		}

		if _, err := RelayoutSubtreeWithOverrides(t.Context(), cached, cardCached, widthOverride(value), opts); err != nil {
			t.Fatalf("step %d cached: %v", step, err)
		}

		fresh.overrideBase = nil

		if _, err := RelayoutSubtreeWithOverrides(t.Context(), fresh, cardFresh, widthOverride(value), opts); err != nil {
			t.Fatalf("step %d fresh: %v", step, err)
		}

		compareOverrideResults(t, step, cached, fresh)

		if cached.styles[cardCached].Width != baseWidth {
			t.Fatalf("step %d: the retained width is %gpt, want the base %gpt", step,
				cached.styles[cardCached].Width, baseWidth)
		}
		if relayoutBoxByID(t, cached, "card").style != cached.styles[cardCached] {
			t.Fatalf("step %d: the card box and the retained map disagree about the base style", step)
		}
	}

	if hits != len(overrideFrames)-1 {
		t.Fatalf("the cache was present before %d of %d later frames", hits, len(overrideFrames)-1)
	}
}

// TestOverrideBaseMissesWhenInputsMove pins the guards: a different style
// input, a different target, or a style another patch replaced since the
// cache was kept must each miss, so the pass runs the cascade.
func TestOverrideBaseMissesWhenInputsMove(t *testing.T) {
	t.Parallel()

	opts := overrideOptions(t)
	root := mustParse(t, overrideFlexHTML)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatal(err)
	}

	card := relayoutNodeByID(t, root, "card")

	if _, err := RelayoutSubtreeWithOverrides(t.Context(), prev, card, widthOverride("140px"), opts); err != nil {
		t.Fatal(err)
	}

	if prev.overrideBase == nil {
		t.Fatal("a committed override pass kept no base styles")
	}

	target, err := resolveOverrideTarget(prev, card)
	if err != nil {
		t.Fatal(err)
	}

	parent := relayoutParentStyle(target)
	rootNode := target.box.node

	keep := *prev.overrideBase
	restore := func() { clone := keep; prev.overrideBase = &clone }

	hit := func() bool { restore(); return takeOverrideBase(prev, rootNode, parent, opts) != nil }

	if !hit() {
		t.Fatal("unchanged inputs missed the cache")
	}

	if prev.overrideBase != nil {
		t.Fatal("a hit must take the map out of the cache")
	}

	moved := opts
	moved.State.Hover = "row"

	restore()

	if takeOverrideBase(prev, rootNode, parent, moved) != nil {
		t.Fatal("a different hover state reused the cache")
	}

	restore()

	if takeOverrideBase(prev, card, parent, opts) != nil {
		t.Fatal("a different target reused the cache")
	}

	restore()

	other := *parent
	if takeOverrideBase(prev, rootNode, &other, opts) != nil {
		t.Fatal("a different parent style reused the cache")
	}

	// Another patch replaced the card's style in the retained map.
	restore()

	replaced := *prev.styles[card]
	prev.styles[card] = &replaced

	if takeOverrideBase(prev, rootNode, parent, opts) != nil {
		t.Fatal("a restyled node reused the cache")
	}

	// A node that has no cached style (the tree gained a child).
	restore()

	prev.styles[card] = keep.styles[card]
	rootNode.Children = append(rootNode.Children, &html.Node{Type: html.ElementNode, Name: "span"})

	defer func() { rootNode.Children = rootNode.Children[:len(rootNode.Children)-1] }()

	if takeOverrideBase(prev, rootNode, parent, opts) != nil {
		t.Fatal("a node without a cached style reused the cache")
	}
}

// TestCloneResultDropsOverrideBase keeps the cache out of clones, which
// patch independently from the display they copied.
func TestCloneResultDropsOverrideBase(t *testing.T) {
	t.Parallel()

	opts := overrideOptions(t)
	root := mustParse(t, overrideFlexHTML)

	prev, err := Layout(root, opts)
	if err != nil {
		t.Fatal(err)
	}

	card := relayoutNodeByID(t, root, "card")
	if _, err := RelayoutSubtreeWithOverrides(t.Context(), prev, card, widthOverride("140px"), opts); err != nil {
		t.Fatal(err)
	}

	if prev.overrideBase == nil {
		t.Fatal("the pass kept no base styles")
	}

	if CloneResult(prev).overrideBase != nil {
		t.Fatal("a clone shares the source's override cache")
	}
}
