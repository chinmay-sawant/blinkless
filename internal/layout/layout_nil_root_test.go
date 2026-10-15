package layout

import (
	"strings"
	"testing"
)

// TestLayoutNilRoot rejects a nil document without panicking. The defined
// outcome is a descriptive error and a nil Result.
func TestLayoutNilRoot(t *testing.T) {
	t.Parallel()

	opts := Options{Width: 500, Height: 800}

	res, err := Layout(nil, opts)
	if err == nil {
		t.Fatal("Layout(nil) succeeded, want descriptive error")
	}

	if !strings.Contains(err.Error(), "nil root") {
		t.Fatalf("Layout(nil) error = %q, want it to mention nil root", err)
	}

	if res != nil {
		t.Fatalf("Layout(nil) result = %+v, want nil", res)
	}
}

// TestLayoutContextNilRoot covers the context/workspace entry points sharing
// layoutContextWithStyles so they stay nil-safe together.
func TestLayoutContextNilRoot(t *testing.T) {
	t.Parallel()

	opts := Options{Width: 500, Height: 800}

	if res, err := LayoutContext(t.Context(), nil, opts); err == nil || res != nil {
		t.Fatalf("LayoutContext(nil) = (%v, %v), want (nil, error)", res, err)
	}

	if res, err := WithWorkspace(t.Context(), nil, opts, nil); err == nil || res != nil {
		t.Fatalf("WithWorkspace(nil) = (%v, %v), want (nil, error)", res, err)
	}

	if res, err := ContextWithStyles(t.Context(), nil, opts, nil); err == nil || res != nil {
		t.Fatalf("ContextWithStyles(nil) = (%v, %v), want (nil, error)", res, err)
	}
}
