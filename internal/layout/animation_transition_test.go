package layout

import "testing"

func TestCSSTimeMillisecondsRejectsOverflow(t *testing.T) {
	t.Parallel()

	if got, ok := cssTimeMilliseconds("250ms", false); !ok || got != 250 {
		t.Fatalf("cssTimeMilliseconds(250ms) = (%g, %t), want (250, true)", got, ok)
	}

	for _, raw := range []string{"1e308s", "-1e308s"} {
		if got, ok := cssTimeMilliseconds(raw, true); ok {
			t.Errorf("cssTimeMilliseconds(%q) = (%g, true), want overflow rejection", raw, got)
		}
	}
}
