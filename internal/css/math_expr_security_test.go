package css

import (
	"strings"
	"testing"
)

func TestEvalMathLongUnaryChain(t *testing.T) {
	t.Parallel()

	expr := "calc(" + strings.Repeat("- + ", 100001) + "1)"
	value, ok := EvalMath(expr, MathEnv{})

	if !ok || value != -1 {
		t.Fatalf("long unary chain = %v, %v; want -1, true", value, ok)
	}
}

// TestEvalMathRejectsNonFinite proves overflow and undefined results degrade
// to "no value" instead of poisoning layout geometry with Inf or NaN.
func TestEvalMathRejectsNonFinite(t *testing.T) {
	t.Parallel()

	env := MathEnv{FontSizePx: 16, PercentPx: 8, ViewportW: 800, ViewportH: 600}

	for _, expr := range []string{
		"calc(1e308 * 1e308)",
		"calc(1e308 + 1e308)",
		"calc(0 / 1 - 0 / 1 + 1e308 * 1e308)",
	} {
		if _, ok := EvalMath(expr, env); ok {
			t.Fatalf("EvalMath(%q) accepted a non-finite result", expr)
		}
	}
}

// TestEvalMathDepthCap proves deeply nested math cannot drive deep parser
// recursion.
func TestEvalMathDepthCap(t *testing.T) {
	t.Parallel()

	env := MathEnv{FontSizePx: 16, PercentPx: 8, ViewportW: 800, ViewportH: 600}

	deep := "calc(" + strings.Repeat("(", maxMathDepth+10) + "1" +
		strings.Repeat(")", maxMathDepth+10) + ")"

	if _, ok := EvalMath(deep, env); ok {
		t.Fatal("EvalMath accepted nesting past the depth cap")
	}

	shallow := "calc((1 + 2) * 3)"

	value, ok := EvalMath(shallow, env)
	if !ok || value != 9 {
		t.Fatalf("EvalMath(%q) = %v, %v; want 9, true", shallow, value, ok)
	}
}
