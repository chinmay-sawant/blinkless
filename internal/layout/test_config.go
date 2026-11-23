package layout

import "context"

// TestConfig switches individual layout fast paths and caches off for the runs
// that carry it. It replaces process-wide test switches: a config rides one
// context (WithTestConfig) or one in-package Options value, so parallel tests
// with different settings never see each other's. Production layout carries no
// config and every accessor below reads the zero value.
type TestConfig struct {
	// FlexRowMemoBypass disables the flex row base-width memo.
	FlexRowMemoBypass bool
	// SimpleFlexHeightBypass disables the simple flex row height fast path.
	SimpleFlexHeightBypass bool
	// MulticolMemoBypass disables the multicol child height memo.
	MulticolMemoBypass bool
	// PaintRestyleDisabled turns the paint-only relayout fast path off so a
	// test can produce the slow-path result for the same input.
	PaintRestyleDisabled bool
	// MaxSharedTextWidthEntries caps the shared string width cache. Zero keeps
	// the default bound.
	MaxSharedTextWidthEntries int
	// MaxSharedRuneWidthEntries caps the shared rune width cache. Zero keeps
	// the default bound.
	MaxSharedRuneWidthEntries int

	// scrollOffsets carries one scroll offset per scrollable region box (see
	// scroll_runtime.go). It is unexported: only in-package tests hold box
	// pointers.
	scrollOffsets map[*box]ScrollOffset
}

type testConfigKey struct{}

// WithTestConfig returns a context whose layout runs use cfg. A nil ctx is
// treated as context.Background.
func WithTestConfig(ctx context.Context, cfg TestConfig) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, testConfigKey{}, &cfg)
}

// testConfigFrom returns the config ctx carries, or nil when it carries none.
func testConfigFrom(ctx context.Context) *TestConfig {
	if ctx == nil {
		return nil
	}

	cfg, _ := ctx.Value(testConfigKey{}).(*TestConfig)

	return cfg
}

// resolveTestConfig prefers an in-package Options config over a context one.
func resolveTestConfig(ctx context.Context, opts Options) *TestConfig {
	if opts.testCfg != nil {
		return opts.testCfg
	}

	return testConfigFrom(ctx)
}

func (c *TestConfig) flexRowMemoBypass() bool { return c != nil && c.FlexRowMemoBypass }

func (c *TestConfig) simpleFlexHeightBypass() bool { return c != nil && c.SimpleFlexHeightBypass }

func (c *TestConfig) multicolMemoBypass() bool { return c != nil && c.MulticolMemoBypass }

func (c *TestConfig) paintRestyleDisabled() bool { return c != nil && c.PaintRestyleDisabled }

// sharedTextBound returns the effective string width cache bound.
func (c *TestConfig) sharedTextBound() int {
	if c == nil || c.MaxSharedTextWidthEntries <= 0 {
		return maxTextWidthCacheEntries
	}

	return c.MaxSharedTextWidthEntries
}

// sharedRuneBound returns the effective rune width cache bound.
func (c *TestConfig) sharedRuneBound() int {
	if c == nil || c.MaxSharedRuneWidthEntries <= 0 {
		return maxRuneWidthCacheEntries
	}

	return c.MaxSharedRuneWidthEntries
}

// scrollOffsetOf returns the offset one region box carries, or zero when the
// config sets none. Production carries no config, so every region reads as
// zero and static paint is unchanged.
func (c *TestConfig) scrollOffsetOf(boxNode *box) ScrollOffset {
	if c == nil || boxNode == nil {
		return ScrollOffset{X: 0, Y: 0}
	}

	return c.scrollOffsets[boxNode]
}

// setScrollOffset stores the offset one scrollable region carries. A zero
// offset deletes the entry. Tests call it before handing the config to an
// engine; the map is not safe for concurrent use.
func (c *TestConfig) setScrollOffset(boxNode *box, off ScrollOffset) {
	if boxNode == nil {
		return
	}

	if off.X == 0 && off.Y == 0 {
		delete(c.scrollOffsets, boxNode)

		return
	}

	if c.scrollOffsets == nil {
		c.scrollOffsets = make(map[*box]ScrollOffset)
	}

	c.scrollOffsets[boxNode] = off
}
