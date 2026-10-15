package css_test

import (
	"testing"

	"github.com/chinmay-sawant/blinkless/internal/css"
)

func FuzzEvalMath(f *testing.F) {
	f.Add("1px + 2px")
	f.Add("calc(100% / 3)")
	f.Add("((((1px))))")
	f.Add(longUnary())

	f.Fuzz(func(_ *testing.T, expr string) {
		if len(expr) > 2048 {
			return
		}

		_, _ = css.EvalMath(expr, css.MathEnv{})
	})
}

func longUnary() string {
	signs := make([]byte, 128)
	for i := range signs {
		signs[i] = '+'
	}

	return string(signs) + "4px"
}
