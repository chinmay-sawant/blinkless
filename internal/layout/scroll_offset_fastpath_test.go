package layout

import "testing"

// TestTestScrollOffsetFastPathSkipsLock pins the production lookup: an engine
// without a test config (every production engine) reads a zero offset with no
// shared state to lock. The offsets live on the per-engine TestConfig, so the
// former process-wide map and its lock no longer exist.
func TestTestScrollOffsetFastPathSkipsLock(t *testing.T) {
	t.Parallel()

	var cfg *TestConfig

	if got := cfg.scrollOffsetOf(&box{}); got != (ScrollOffset{}) {
		t.Fatalf("offset without a config = %+v, want zero", got)
	}

	eng := &engine{scale: 1}
	if got := eng.scrollRegionViewport(&box{}).At; got != (ScrollOffset{}) {
		t.Fatalf("production viewport offset = %+v, want zero", got)
	}
}

func TestTestScrollOffsetStillWorks(t *testing.T) {
	t.Parallel()

	region := &box{}
	cfg := &TestConfig{}

	cfg.setScrollOffset(region, ScrollOffset{X: 3, Y: 4})

	if got := cfg.scrollOffsetOf(region); got != (ScrollOffset{X: 3, Y: 4}) {
		t.Fatalf("offset = %+v", got)
	}

	// A second config never sees the first one's offsets.
	if got := (&TestConfig{}).scrollOffsetOf(region); got != (ScrollOffset{}) {
		t.Fatalf("offset leaked across configs: %+v", got)
	}

	cfg.setScrollOffset(region, ScrollOffset{})

	if got := cfg.scrollOffsetOf(region); got != (ScrollOffset{}) || len(cfg.scrollOffsets) != 0 {
		t.Fatalf("offset after unset = %+v entries=%d", got, len(cfg.scrollOffsets))
	}
}
