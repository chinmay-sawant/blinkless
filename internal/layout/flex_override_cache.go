package layout

import (
	"math"
	"sync"
	"sync/atomic"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// flexOverrideMaxEntries bounds one run's flex item style override cache.
// Past the bound a lookup falls back to a fresh copy, so a document with many
// distinct forced sizes cannot grow the map with its item count.
const flexOverrideMaxEntries = 256

// flexOverrideMapInitCap sizes a fresh flex override map so one container's
// repeated items do not grow it entry by entry.
const flexOverrideMapInitCap = 16

// flexStyleOverrideFlags records which force steps produced one override.
// The row and logical-axis paths force the main size first and the cross size
// second; the column path forces the cross size first and the measured cross
// width second, so the column bit keeps the two meanings of the key's length
// fields apart.
type flexStyleOverrideFlags uint8

const (
	// flexOverrideStretch forced the cross height after the main width.
	flexOverrideStretch flexStyleOverrideFlags = 1 << iota
	// flexOverrideClearBasis cleared the flex basis with the main width.
	flexOverrideClearBasis
	// flexOverrideColumn marks a column container's cross-size override.
	flexOverrideColumn
	// flexOverrideCrossWidth forced the measured cross width after the cross
	// height.
	flexOverrideCrossWidth
)

// flexStyleOverrideKey identifies one finished flex item style override. base
// pins the interned resolved style, mainBits and crossBits are the
// math.Float64bits of the forced lengths in application order, and flags
// record the force steps. The engine's scale is fixed for the run and the
// cache dies with the engine, so the scale needs no key field of its own.
type flexStyleOverrideKey struct {
	base      *ResolvedStyle
	mainBits  uint64
	crossBits uint64
	flags     flexStyleOverrideFlags
}

// flexOverrideBypass disables the override cache. Tests only: in-package
// tests flip it to prove a cached and an uncached run produce identical
// output. The hit and miss counters record cache traffic for the same tests.
//
//nolint:gochecknoglobals // test-visible memo state mirrors flexRowMemoBypass in flex.go
var (
	flexOverrideBypass atomic.Bool
	flexOverrideHits   atomic.Int64
	flexOverrideMisses atomic.Int64
)

// flexRowOverrideKey builds the cache key for a row or logical-axis item.
// mainBits is the forced main width, crossBits the forced cross height when
// forceStretch is set. valid is false for a nil base, which keeps the old
// allocation path because the force functions need a style record.
func flexRowOverrideKey(
	base *ResolvedStyle, forceStretch bool, targetCross, availW float64,
) (flexStyleOverrideKey, bool) {
	if base == nil {
		return flexStyleOverrideKey{}, false //nolint:exhaustruct // zero key means uncacheable
	}

	flags := flexOverrideClearBasis
	crossBits := uint64(0)

	if forceStretch {
		flags |= flexOverrideStretch
		crossBits = math.Float64bits(targetCross)
	}

	return flexStyleOverrideKey{
		base:      base,
		mainBits:  math.Float64bits(availW),
		crossBits: crossBits,
		flags:     flags,
	}, true
}

// flexColumnOverrideKey builds the cache key for a column item. mainBits is
// the forced cross height and crossBits the measured cross width when
// forceCrossWidth is set.
func flexColumnOverrideKey(
	base *ResolvedStyle, height, crossW float64, forceCrossWidth bool,
) (flexStyleOverrideKey, bool) {
	if base == nil {
		return flexStyleOverrideKey{}, false //nolint:exhaustruct // zero key means uncacheable
	}

	flags := flexOverrideColumn
	crossBits := uint64(0)

	if forceCrossWidth {
		flags |= flexOverrideCrossWidth
		crossBits = math.Float64bits(crossW)
	}

	return flexStyleOverrideKey{
		base:      base,
		mainBits:  math.Float64bits(height),
		crossBits: crossBits,
		flags:     flags,
	}, true
}

// flexBaseStyleStable reports whether style is the node's interned cascade
// style. Override and synthetic styles are allocated in resettable style
// chunks, so their address can later hold another record; keying the override
// cache on such a pointer would serve a stale override for a different base.
func (e *engine) flexBaseStyleStable(node *html.Node, style *ResolvedStyle) bool {
	return e.styles[node] == style
}

//nolint:gochecknoglobals // global pool for reusable flex style override maps
var flexOverrideMapPool = sync.Pool{
	New: func() any {
		return make(map[flexStyleOverrideKey]*ResolvedStyle, flexOverrideMapInitCap)
	},
}

// allocFlexOverrideStyle allocates a ResolvedStyle from a chunk that survives
// until finalizeResult adopts it, ensuring speculative measure scopes that
// unwind e.styleChunks cannot invalidate cached overrides.
func (e *engine) allocFlexOverrideStyle(base *ResolvedStyle) *ResolvedStyle {
	e.nextStyleChunk(&e.overrideStyleChunks)

	chunk := e.overrideStyleChunks[len(e.overrideStyleChunks)-1]

	if base != nil {
		*chunk = append(*chunk, *base)
	} else {
		*chunk = append(*chunk, zeroResolvedStyle)
	}

	return &(*chunk)[len(*chunk)-1]
}

// flexRowStyleOverride returns the immutable override of base used to build a
// row or logical-axis flex item: the main width forced to availW and, when
// forceStretch, the cross height forced to targetCross. Repeated items that
// share the interned base style and the same forced sizes share one override
// instead of copying the 5872-byte record per item. The caller must not
// mutate the returned style.
func (e *engine) flexRowStyleOverride(
	base *ResolvedStyle, stableBase bool, forceStretch bool, targetCross, availW float64,
) *ResolvedStyle {
	key, cacheable := flexRowOverrideKey(base, forceStretch, targetCross, availW)
	if !cacheable || !stableBase {
		return e.allocFlexRowStyleOverride(base, forceStretch, targetCross, availW)
	}

	if !flexOverrideBypass.Load() {
		if e.flexStyleOverrides != nil {
			if cached, ok := e.flexStyleOverrides[key]; ok {
				flexOverrideHits.Add(1)

				return cached
			}
		}
	}

	// The override is owned by the cache and outlives the measure scopes that
	// reclaim allocStyle chunks, so it cannot come from e.styleChunks: a
	// resetStyles later in the run would clear and reuse that memory. Instead,
	// it comes from overrideStyleChunks, which survives until finalizeResult
	// merges them into Result.styleChunks.
	override := e.allocFlexOverrideStyle(base)
	e.forceFlexItemMainSize(override, availW)

	if forceStretch {
		e.forceFlexItemCrossSize(override, targetCross)
	}

	flexOverrideMisses.Add(1)
	e.storeFlexStyleOverride(key, override)

	return override
}

// flexColumnStyleOverride returns the immutable override of base used to
// build a column flex item: the cross height forced to height and, when
// forceCrossWidth, the measured cross width forced to crossW.
func (e *engine) flexColumnStyleOverride(
	base *ResolvedStyle, stableBase bool, height, crossW float64, forceCrossWidth bool,
) *ResolvedStyle {
	key, cacheable := flexColumnOverrideKey(base, height, crossW, forceCrossWidth)
	if !cacheable || !stableBase {
		return e.allocFlexColumnStyleOverride(base, height, crossW, forceCrossWidth)
	}

	if !flexOverrideBypass.Load() {
		if e.flexStyleOverrides != nil {
			if cached, ok := e.flexStyleOverrides[key]; ok {
				flexOverrideHits.Add(1)

				return cached
			}
		}
	}

	override := e.allocFlexOverrideStyle(base)
	e.forceFlexItemCrossSize(override, height)

	if forceCrossWidth {
		e.forceFlexItemCrossWidth(override, crossW)
	}

	flexOverrideMisses.Add(1)
	e.storeFlexStyleOverride(key, override)

	return override
}

// allocFlexRowStyleOverride is the uncached row path, kept for a nil base.
func (e *engine) allocFlexRowStyleOverride(
	base *ResolvedStyle, forceStretch bool, targetCross, availW float64,
) *ResolvedStyle {
	override := e.allocStyle(base)
	e.forceFlexItemMainSize(override, availW)

	if forceStretch {
		e.forceFlexItemCrossSize(override, targetCross)
	}

	return override
}

// allocFlexColumnStyleOverride is the uncached column path, kept for a nil
// base.
func (e *engine) allocFlexColumnStyleOverride(
	base *ResolvedStyle, height, crossW float64, forceCrossWidth bool,
) *ResolvedStyle {
	override := e.allocStyle(base)
	e.forceFlexItemCrossSize(override, height)

	if forceCrossWidth {
		e.forceFlexItemCrossWidth(override, crossW)
	}

	return override
}

// storeFlexStyleOverride records one finished override for the rest of the
// run. Past the bound the caller keeps the fresh copy and the map stops
// growing, so the document falls back to the old allocation path.
func (e *engine) storeFlexStyleOverride(key flexStyleOverrideKey, override *ResolvedStyle) {
	if flexOverrideBypass.Load() {
		return
	}

	if e.flexStyleOverrides == nil {
		if m, ok := flexOverrideMapPool.Get().(map[flexStyleOverrideKey]*ResolvedStyle); ok {
			clear(m)
			e.flexStyleOverrides = m
		} else {
			e.flexStyleOverrides = make(map[flexStyleOverrideKey]*ResolvedStyle, flexOverrideMapInitCap)
		}
	} else if len(e.flexStyleOverrides) >= flexOverrideMaxEntries {
		return
	}

	e.flexStyleOverrides[key] = override
}
