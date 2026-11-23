package css

import (
	"fmt"
	"testing"
)

func TestResolveCustomPropsEmbeddedVar(t *testing.T) {
	t.Parallel()

	got := ResolveCustomProps(map[string]string{
		"--accent": "red",
		"--u":      "4px",
		"--border": "1px solid var(--accent)",
		"--w":      "calc(var(--u) * 2)",
		"--c1":     "var(--c2)",
		"--c2":     "var(--c1)",
	}, nil)

	if got["--border"] != "1px solid red" {
		t.Errorf("--border = %q", got["--border"])
	}

	if got["--w"] != "calc(4px * 2)" {
		t.Errorf("--w = %q", got["--w"])
	}

	if got["--c1"] != "" {
		t.Errorf("cycle = %q, want empty", got["--c1"])
	}
}

func TestResolveCustomPropsLongChainCapped(t *testing.T) {
	t.Parallel()

	const n = 1000

	declared := map[string]string{fmt.Sprintf("--v%d", n): "end"}
	for i := range n {
		declared[fmt.Sprintf("--v%d", i)] = fmt.Sprintf("var(--v%d)", i+1)
	}

	for range 5 {
		got := ResolveCustomProps(declared, nil)

		if got["--v0"] != "" {
			t.Fatalf("--v0 = %q, want invalid beyond the depth cap", got["--v0"])
		}

		// Short chains still resolve, independent of iteration order.
		if got[fmt.Sprintf("--v%d", n-3)] != "end" {
			t.Fatalf("short tail = %q, want end", got[fmt.Sprintf("--v%d", n-3)])
		}
	}
}

func TestCustomPropertyNamesCaseSensitive(t *testing.T) {
	t.Parallel()

	decls := ParseInline("--colorPrimary: red; --colorprimary: blue; COLOR: green")
	if len(decls) != 3 || decls[0].Prop != "--colorPrimary" || decls[1].Prop != "--colorprimary" || decls[2].Prop != "color" {
		t.Fatalf("decls = %#v", decls)
	}

	got := ResolveCustomProps(map[string]string{
		"--colorPrimary": "red",
		"--colorprimary": "blue",
		"--a":            "var(--colorPrimary)",
		"--b":            "var(--colorprimary)",
	}, nil)

	if got["--a"] != "red" || got["--b"] != "blue" {
		t.Fatalf("a=%q b=%q", got["--a"], got["--b"])
	}
}
