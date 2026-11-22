package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

// groupedHoverSource lays out three sibling cards, the first inside a paint
// group. The hover rule changes text color, which the paint-only fast path
// does not take, so the patch goes through the engine splice.
func groupedHoverSource(group string) string {
	return `<style>.g{display:flex;gap:4px;padding:4px}` +
		`.i{width:84px;height:22px;box-sizing:border-box;background:#eee;color:#222}` +
		`#a{` + group + `}#a:hover{color:#c00}</style><div class="g">` +
		`<div id="a" class="i">a</div><div id="b" class="i">b</div><div id="c" class="i">c</div></div>`
}

// paintOps lists the operations that paint, without group markers, so a
// patched display can be compared with a full layout that wraps the same
// paint in a differently nested set of redundant groups.
func paintOps(d *layout.Display) []layout.DisplayOp {
	out := make([]layout.DisplayOp, 0, len(d.Ops))
	for i := range d.Ops {
		if d.Ops[i].IsGroupBegin() || d.Ops[i].IsGroupEnd() {
			continue
		}
		out = append(out, d.Ops[i])
	}

	return out
}

// checkGroupStructure requires balanced, properly nested group markers and a
// paint order that is a permutation of the operations.
func checkGroupStructure(t *testing.T, step int, d *layout.Display) {
	t.Helper()

	depth := 0
	for i := range d.Ops {
		switch {
		case d.Ops[i].IsGroupBegin():
			depth++
		case d.Ops[i].IsGroupEnd():
			depth--
			if depth < 0 {
				t.Fatalf("step %d: group end without a begin at op %d", step, i)
			}
		}
	}
	if depth != 0 {
		t.Fatalf("step %d: %d group begins are never closed", step, depth)
	}
	seen := make([]bool, len(d.Ops))
	for _, idx := range d.Order {
		if idx < 0 || idx >= len(d.Ops) || seen[idx] {
			t.Fatalf("step %d: paint order %v is not a permutation of %d ops", step, d.Order, len(d.Ops))
		}
		seen[idx] = true
	}
}

// TestRelayoutElementsGroupedRootKeepsOpCount chains hover enters and leaves
// the way a page does, patching the display the previous patch returned. A
// root that owns a paint group must replace its group markers: the op count
// stays where the first patch left it instead of growing a pair per hover,
// the markers stay balanced, the paint order stays valid, and the painted
// operations equal a full layout's.
func TestRelayoutElementsGroupedRootKeepsOpCount(t *testing.T) {
	t.Parallel()

	for _, group := range []string{
		"backdrop-filter:blur(4px)",
		"opacity:.9",
		"filter:blur(1px)",
		"isolation:isolate;position:relative;z-index:1",
		"mix-blend-mode:multiply",
	} {
		t.Run(group, func(t *testing.T) {
			t.Parallel()

			resting := applyAt(t, groupedHoverSource(group), 320, 200, "", "", "")
			hovered := restyleHover(t, resting, "a")
			unhovered := restyleHover(t, hovered, "")

			current := layRetained(t, resting)
			steady := -1
			for step := range 12 {
				next := hovered
				if step%2 == 1 {
					next = unhovered
				}
				patched, err := layout.RelayoutElements(
					t.Context(), next, current, []string{"a"}, layout.Options{RetainState: true})
				if err != nil {
					t.Fatalf("step %d: patch refused: %v", step, err)
				}
				checkGroupStructure(t, step, patched)
				if steady < 0 {
					steady = len(patched.Ops)
				} else if len(patched.Ops) != steady {
					t.Fatalf("step %d: %d ops, want the steady %d", step, len(patched.Ops), steady)
				}
				got, want := paintOps(patched), paintOps(layRetained(t, next))
				if len(got) != len(want) {
					t.Fatalf("step %d: %d painted ops, a full layout has %d", step, len(got), len(want))
				}
				for i := range got {
					if got[i].Kind != want[i].Kind || got[i].X != want[i].X || got[i].Y != want[i].Y ||
						got[i].W != want[i].W || got[i].H != want[i].H || got[i].Text != want[i].Text ||
						got[i].R != want[i].R || got[i].G != want[i].G || got[i].B != want[i].B {
						t.Fatalf("step %d: painted op %d differs from a full layout:\n got  %+v\n want %+v", step, i, got[i], want[i])
					}
				}
				current = patched
			}
		})
	}
}

// TestRelayoutElementGroupedRootKeepsOpCount is the single element form of
// the same gate.
func TestRelayoutElementGroupedRootKeepsOpCount(t *testing.T) {
	t.Parallel()

	for _, group := range []string{"backdrop-filter:blur(4px)", "opacity:.9", "mix-blend-mode:multiply"} {
		t.Run(group, func(t *testing.T) {
			t.Parallel()

			resting := applyAt(t, groupedHoverSource(group), 320, 200, "", "", "")
			hovered := restyleHover(t, resting, "a")
			unhovered := restyleHover(t, hovered, "")

			current := layRetained(t, resting)
			steady := -1
			for step := range 10 {
				next := hovered
				if step%2 == 1 {
					next = unhovered
				}
				patch, err := layout.RelayoutElement(t.Context(), next, current, "a", layout.Options{RetainState: true})
				if err != nil {
					t.Fatalf("step %d: patch refused: %v", step, err)
				}
				checkGroupStructure(t, step, patch.Display)
				if steady < 0 {
					steady = len(patch.Display.Ops)
				} else if len(patch.Display.Ops) != steady {
					t.Fatalf("step %d: %d ops, want the steady %d", step, len(patch.Display.Ops), steady)
				}
				current = patch.Display
			}
		})
	}
}

// A child gaining its own group must leave its parent's markers in place.
func TestRelayoutElementsKeepsParentGroup(t *testing.T) {
	t.Parallel()

	const source = `<style>#wrap{mix-blend-mode:multiply}#a{height:30px;background:red}` +
		`#a:hover{isolation:isolate}</style><div id="wrap"><div id="a">a</div></div>`

	resting := applyAt(t, source, 320, 200, "", "", "")
	hovered := restyleHover(t, resting, "a")

	patched, err := layout.RelayoutElements(
		t.Context(), hovered, layRetained(t, resting), []string{"a"}, layout.Options{RetainState: true})
	if err != nil {
		t.Fatal(err)
	}

	checkGroupStructure(t, 0, patched)
	for i := range patched.Ops {
		op := &patched.Ops[i]
		if op.IsGroupBegin() && op.Group().Mode == "multiply" {
			return
		}
	}

	t.Fatal("patch removed the parent's multiply group")
}

// TestRelayoutElementWithOverridesGroupedContainerKeepsOpCount verifies that
// animating a layout override inside a grouped containing block replaces the
// container's group markers instead of adding two markers per tick.
func TestRelayoutElementWithOverridesGroupedContainerKeepsOpCount(t *testing.T) {
	t.Parallel()

	for _, group := range []string{
		"backdrop-filter:blur(4px)",
		"isolation:isolate",
	} {
		t.Run(group, func(t *testing.T) {
			t.Parallel()

			source := `<style>#wrap{width:240px;height:60px;` + group + `;background:#eee}` +
				`#child{width:60px;height:20px;background:#336}</style>` +
				`<div id="wrap"><div id="child">x</div></div>`

			doc := applyAt(t, source, 320, 200, "", "", "")
			current := layRetained(t, doc)
			steady := len(current.Ops)

			for step := range 10 {
				margin := "10px"
				if step%2 == 1 {
					margin = "24px"
				}

				patch, err := layout.RelayoutElementWithOverrides(
					t.Context(), doc, current, "child",
					[]layout.Override{{Property: "margin-left", Value: margin}},
					layout.Options{RetainState: true},
				)
				if err != nil {
					t.Fatalf("step %d: override relayout refused: %v", step, err)
				}

				checkGroupStructure(t, step, patch.Display)

				if len(patch.Display.Ops) != steady {
					t.Fatalf("step %d: %d ops, want steady %d", step, len(patch.Display.Ops), steady)
				}

				current = patch.Display
			}
		})
	}
}

// TestRelayoutElementsRemovingGroupStripsMarkers verifies that hover-leave
// removing backdrop-filter or isolation:isolate strips the old group begin and
// end markers cleanly instead of stranding an empty wrapper.
func TestRelayoutElementsRemovingGroupStripsMarkers(t *testing.T) {
	t.Parallel()

	for _, group := range []string{
		"backdrop-filter:blur(4px)",
		"isolation:isolate",
	} {
		t.Run(group, func(t *testing.T) {
			t.Parallel()

			source := `<style>.g{display:flex;gap:4px}` +
				`.i{width:84px;height:24px;background:#eee;color:#222}` +
				`#a:hover{` + group + `;color:#c00}</style>` +
				`<div class="g"><div id="a" class="i">a</div><div id="b" class="i">b</div></div>`

			resting := applyAt(t, source, 320, 200, "", "", "")
			hovered := restyleHover(t, resting, "a")
			unhovered := restyleHover(t, hovered, "")

			baseDisplay := layRetained(t, resting)
			baseOps := len(baseDisplay.Ops)

			entered, err := layout.RelayoutElements(
				t.Context(), hovered, baseDisplay, []string{"a"}, layout.Options{RetainState: true})
			if err != nil {
				t.Fatalf("hover enter refused: %v", err)
			}

			checkGroupStructure(t, 0, entered)

			if len(entered.Ops) != baseOps+2 {
				t.Fatalf("hover enter ops = %d, want %d (base + 2 markers)", len(entered.Ops), baseOps+2)
			}

			left, err := layout.RelayoutElements(
				t.Context(), unhovered, entered, []string{"a"}, layout.Options{RetainState: true})
			if err != nil {
				t.Fatalf("hover leave refused: %v", err)
			}

			checkGroupStructure(t, 1, left)

			if len(left.Ops) != baseOps {
				t.Fatalf("hover leave ops = %d, want %d (stale group markers leaked)", len(left.Ops), baseOps)
			}

			for i := range left.Ops {
				if left.Ops[i].IsGroupBegin() || left.Ops[i].IsGroupEnd() {
					t.Fatalf("hover leave left stale group marker at op %d", i)
				}
			}
		})
	}
}
