package layout_test

import (
	"errors"
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

// TestRelayoutElementsAbortedBatchLeavesPrevIntact verifies that when a batch
// patches its first element (via paint restyle or full subtree relayout) and
// aborts on a later unknown ID, prev's ops, boxes, and retained styles stay
// untouched and a subsequent valid patch on prev still succeeds cleanly.
func TestRelayoutElementsAbortedBatchLeavesPrevIntact(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		css  string
	}{
		{
			name: "paint restyle then unknown id",
			css:  `.card{width:120px;height:32px;background:#eeeeee;color:#111111}#c1:hover{background:#224488}`,
		},
		{
			name: "subtree relayout then unknown id",
			css:  `.card{width:120px;height:32px;background:#eeeeee;color:#111111}#c1:hover{color:#cc0000}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			source := `<style>` + tc.css + `</style>` +
				`<div id="c1" class="card">one</div><div id="c2" class="card">two</div>`

			resting := applyAt(t, source, 320, 200, "", "", "")
			hovered := restyleHover(t, resting, "c1")

			prev := layRetained(t, resting)
			origOps := append([]layout.DisplayOp(nil), prev.Ops...)
			origBoxes := append([]layout.Box(nil), prev.Boxes...)

			if _, err := layout.RelayoutElements(
				t.Context(), hovered, prev, []string{"c1", "missing"}, layout.Options{RetainState: true},
			); !errors.Is(err, layout.ErrNoElement) {
				t.Fatalf("aborted batch err = %v, want ErrNoElement", err)
			}

			if len(prev.Ops) != len(origOps) {
				t.Fatalf("prev ops len changed from %d to %d", len(origOps), len(prev.Ops))
			}

			for i := range origOps {
				if prev.Ops[i].Kind != origOps[i].Kind ||
					prev.Ops[i].R != origOps[i].R || prev.Ops[i].G != origOps[i].G ||
					prev.Ops[i].B != origOps[i].B || prev.Ops[i].Alpha != origOps[i].Alpha {
					t.Fatalf("aborted batch mutated prev.Ops[%d]: got %+v, want %+v", i, prev.Ops[i], origOps[i])
				}
			}

			if len(prev.Boxes) != len(origBoxes) {
				t.Fatalf("prev boxes len changed from %d to %d", len(origBoxes), len(prev.Boxes))
			}

			// A follow-up valid batch on prev must still observe prev's original
			// retained styles (e.g., paint restyle must still see the resting
			// background color on #c1 rather than the aborted hovered color).
			patched, err := layout.RelayoutElements(
				t.Context(), hovered, prev, []string{"c1"}, layout.Options{RetainState: true},
			)
			if err != nil {
				t.Fatalf("follow-up valid batch after abort failed: %v", err)
			}

			want := layRetained(t, hovered)
			if len(patched.Ops) != len(want.Ops) {
				t.Fatalf("follow-up ops = %d, want %d", len(patched.Ops), len(want.Ops))
			}

			for i := range want.Ops {
				if patched.Ops[i].Kind != want.Ops[i].Kind ||
					patched.Ops[i].R != want.Ops[i].R || patched.Ops[i].G != want.Ops[i].G ||
					patched.Ops[i].B != want.Ops[i].B || patched.Ops[i].Alpha != want.Ops[i].Alpha {
					t.Fatalf("follow-up op %d differs from full layout:\n got  %+v\n want %+v",
						i, patched.Ops[i], want.Ops[i])
				}
			}

			prev.Release()
			patched.Release()
			want.Release()
		})
	}
}
