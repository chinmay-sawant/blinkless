package layout

import "testing"

// TestXformValueReadsWithoutBinding checks the pointer-receiver accessor used
// by the replay scan: it returns the stored matrix, reports the zero matrix
// when there is no payload, and never attaches the shared empty extra.
func TestXformValueReadsWithoutBinding(t *testing.T) {
	t.Parallel()

	var operation Op

	if got := operation.XformValue(); got != (Matrix2D{}) {
		t.Fatalf("XformValue on a nil payload = %+v, want the zero matrix", got)
	}

	if got := operation.Transform(); !got.IsIdentity() {
		t.Fatalf("Transform on a nil payload = %+v, want identity", got)
	}

	operation.setXform(Matrix2D{A: 2, D: 3, E: 7, F: 11})

	want := Matrix2D{A: 2, D: 3, E: 7, F: 11}
	if got := operation.XformValue(); got != want {
		t.Fatalf("XformValue = %+v, want %+v", got, want)
	}

	if got := operation.Transform(); got != want {
		t.Fatalf("Transform = %+v, want %+v", got, want)
	}

	var shared Op

	shared.bindEmptyExtra()

	if shared.opExtra != emptyExtra {
		t.Fatal("bindEmptyExtra did not attach the shared extra")
	}

	if got := shared.XformValue(); got != (Matrix2D{}) {
		t.Fatalf("XformValue on the shared extra = %+v, want the zero matrix", got)
	}

	if shared.opExtra != emptyExtra {
		t.Fatal("XformValue replaced the shared extra")
	}
}
