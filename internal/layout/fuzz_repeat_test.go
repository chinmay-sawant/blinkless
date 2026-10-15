package layout

import (
	"strings"
	"testing"
)

func FuzzExpandRepeat(f *testing.F) {
	f.Add("repeat(4, 1px)")
	f.Add(strings.Repeat("repeat(63, ", 2) + "1fr))")
	f.Add("none")

	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 1024 {
			return
		}

		got := expandRepeatFunctions(raw)
		if len(got) > maxRepeatBytes && got != raw {
			t.Fatalf("expanded %d bytes past the repeat budget", len(got))
		}
	})
}
