package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/layout"
)

func kerningTextOps(t *testing.T, declarations, text string) []layout.DisplayOp {
	t.Helper()

	source := `<style>p{margin:0;width:300px;font-size:24px;` + declarations + `}</style><p>` + text + `</p>`
	d := displayAt(t, source, 400, 120, "", "", "")
	ops := make([]layout.DisplayOp, 0, len(d.Ops))

	for _, op := range d.Ops {
		if op.Kind == layout.DisplayOpText && op.Text != "" {
			ops = append(ops, op)
		}
	}

	if len(ops) == 0 {
		t.Fatal("no text operations")
	}

	return ops
}

func TestDisplayExplicitKerningChangesWidth(t *testing.T) {
	t.Parallel()

	const text = "AVAVAVAV"
	// Measure all variants in one process so equal text can hit the
	// shared cache. A kerning setting must not poison the default width.
	auto := kerningTextOps(t, "", text)[0]

	for _, tc := range []struct {
		name, decl, feature string
		tighter             bool
	}{
		{"normal", "font-kerning:normal", `"kern" 1`, true},
		{"none", "font-kerning:none", `"kern" 0`, false},
		{"feature enables", `font-kerning:none;font-feature-settings:"kern" 1`, `"kern" 1`, true},
		{"feature disables", `font-kerning:normal;font-feature-settings:"kern" 0`, `"kern" 0`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			op := kerningTextOps(t, tc.decl, text)[0]
			if op.FontFeatures() != tc.feature {
				t.Fatalf("features = %q, want %q", op.FontFeatures(), tc.feature)
			}

			if tc.tighter && op.W >= auto.W || !tc.tighter && op.W != auto.W {
				t.Fatalf("width = %g, default width = %g, want narrower = %t", op.W, auto.W, tc.tighter)
			}

			again := kerningTextOps(t, "", text)[0]
			if again.W != auto.W || again.FontFeatures() != "" {
				t.Fatalf("default changed after measuring explicit kerning: width %g, features %q", again.W, again.FontFeatures())
			}
		})
	}
}

func TestDisplayExplicitKerningAffectsLineFit(t *testing.T) {
	t.Parallel()

	const text = "AVAV VAVA"
	auto := kerningTextOps(t, "width:128px", text)
	kerned := kerningTextOps(t, "width:128px;font-kerning:normal", text)

	if auto[0].Y == auto[len(auto)-1].Y {
		t.Fatal("the default width should wrap the two words")
	}

	if kerned[0].Y != kerned[len(kerned)-1].Y {
		t.Fatal("the kerned words should fit on one line")
	}
}
