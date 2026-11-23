package layout

import "testing"

func TestExpandOneRepeatNonASCIIPrefix(t *testing.T) {
	t.Parallel()

	got, ok := expandOneRepeat("[İstanbul] REPEAT(2, 1fr)")
	if !ok || got != "[İstanbul] 1fr 1fr" {
		t.Fatalf("expandOneRepeat = %q, %v", got, ok)
	}
}
