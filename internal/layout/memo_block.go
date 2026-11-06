package layout

import (
	"math"
	"sync"
)

// Block-container intrinsic memo (designed, not wired).
//
// What it memos: one repeated block-container intrinsic height measurement
// at an available width. Block and flex intrinsic passes rebuild the same
// subtree once to measure and once to emit, and shrink-to-fit, table, grid,
// and flex remeasures repeat that work at identical widths. Grid already
// memos this shape per run (see grid_memo.go); this memo is the block
// counterpart, kept in its own file so the key, bounds, and adopt contract
// can be reviewed without touching call sites.
//
// Key: (style fingerprint, content fingerprint, available width).
//   - styleFP pins the resolved style that sizes the box. At wiring time
//     compute it with styleInternFingerprint(e.stylePtr(node)): equal styles
//     share a fingerprint and any width-affecting field change (width,
//     min/max, padding, border, box-sizing, font, white-space, and related
//     fields) changes it. Never substitute a partial hash of a few fields.
//   - contentFP pins the subtree the measurement flows. The caller supplies
//     it (for example a per-run subtree generation counter bumped on any
//     child, text, image, or descendant-style change, or a hash over child
//     node identities plus text and image keys). Any content change that can
//     move intrinsic size must change contentFP.
//   - widthBits pins the available width through blockIntrinsicWidthBits,
//     which mirrors gridMeasureBits: negative zero maps to zero so equal
//     widths share a key, and non-finite or negative widths are uncacheable.
//
// Invalidation: every key component change is a miss by construction, so a
// style, content, or width change never reads a stale entry. There is no
// cross-run state: the memo instance lives for one Layout run and is dropped
// with the engine (same lifetime as gridMeasureCache), and Clear drops all
// entries within a run (style recascade or subtree mutation when fingerprints
// are generation-based). Uncacheable widths bypass both directions.
//
// Bounds: at most blockIntrinsicMemoCap entries. A new key past the cap
// evicts the oldest inserted key (FIFO). Re-storing an existing key updates
// the value in place and never grows the memo or moves the eviction order.
// FIFO is deliberate: a single Layout run has no hot set worth tracking, and
// a fixed queue keeps every operation O(1) amortized without allocator churn.
//
// Hit verification: a hit returns the stored border-box height for the key.
// This standalone memo stores geometry only. If the skipped build consumes
// engine counters (stickySeq, nextGroupID, op IDs), the wired result must
// carry those deltas and the lookup must replay them, exactly as
// gridMeasureResult does. Do not wire this memo until that counter audit is
// done for the chosen call site.
//
// Adopt contract (where the lookup and store calls go, not yet wired):
//  1. Compute styleFP and contentFP, then call Lookup before the noEmit
//     intrinsic build in the block measure helper (the measure-then-emit and
//     shrink-to-fit remeasure paths are the intended first call sites).
//  2. On a miss, run the existing noEmit build unchanged, then call Store
//     only when the run has no error (eng.err == nil), mirroring
//     measureGridItemPrefH.
//  3. The caller guarantees: all inputs besides the key are loop-invariant at
//     the call site (scale, font set, and image-clamp ambients such as
//     imgMaxW, plus the percentage-height basis that grid_memo.go pins as
//     cbBits, are either fixed for the run or folded into contentFP before
//     wiring); widths passed in are finite and non-negative; stored heights
//     are finite and non-negative; noEmit and scratch-state save/restore
//     still run on a miss exactly as today.
//  4. Keep one memo per Layout run (fresh instance per engine), never share
//     it across runs, and keep bypass available as the differential-test
//     switch until cached and uncached layouts are proved identical.
//
// Concurrency: the engine is single-threaded per Layout run, but the memo
// carries its own mutex so differential and stress tests can share one
// instance across goroutines. Every method is safe for concurrent use and
// nil-receiver safe; a nil memo always misses and never stores.

// blockIntrinsicMemoCap bounds the memo to one small fixed queue per run.
// A pathological document can hold thousands of distinct block widths, so an
// unbounded map would grow with input size; 256 entries cover the repeated
// remeasure working set (same containers measured at the same widths) while
// keeping the per-run footprint flat.
const blockIntrinsicMemoCap = 256

// blockIntrinsicKey identifies one block-container intrinsic measurement.
// styleFP pins the resolved style fingerprint, contentFP pins the measured
// subtree identity, and widthBits pins the available width.
type blockIntrinsicKey struct {
	styleFP   uint64
	contentFP uint64
	widthBits uint64
}

// blockIntrinsicResult is the cached outcome of one measurement: the
// border-box height the skipped noEmit build would have produced.
type blockIntrinsicResult struct {
	height float64
}

// blockIntrinsicMemo is a bounded, mutex-guarded map from measurement key
// to result. The zero value is usable except for the cap, which Lookup and
// Store default to blockIntrinsicMemoCap; prefer the constructors.
type blockIntrinsicMemo struct {
	mu      sync.Mutex
	entries map[blockIntrinsicKey]blockIntrinsicResult
	order   []blockIntrinsicKey
	max     int
	bypass  bool
}

// newBlockIntrinsicMemo returns a memo with the default bound.
func newBlockIntrinsicMemo() *blockIntrinsicMemo {
	return &blockIntrinsicMemo{max: blockIntrinsicMemoCap}
}

// newBlockIntrinsicMemoWithCap returns a memo bounded to maxEntries keys.
// Non-positive values fall back to blockIntrinsicMemoCap. Tests use a small
// cap to prove eviction without inserting hundreds of entries.
func newBlockIntrinsicMemoWithCap(maxEntries int) *blockIntrinsicMemo {
	if maxEntries <= 0 {
		maxEntries = blockIntrinsicMemoCap
	}

	return &blockIntrinsicMemo{max: maxEntries}
}

// blockIntrinsicWidthBits normalizes an available width for map keys.
// Negative zero maps to zero so equal widths share a key. NaN, infinite, and
// negative widths are uncacheable and report false.
func blockIntrinsicWidthBits(width float64) (uint64, bool) {
	if math.IsNaN(width) || math.IsInf(width, 0) || width < 0 {
		return 0, false
	}

	if width == 0 {
		return 0, true
	}

	return math.Float64bits(width), true
}

// makeBlockIntrinsicKey builds the memo key for a measurement. ok is false
// when the width is uncacheable, and that measurement must run uncached.
func makeBlockIntrinsicKey(styleFP, contentFP uint64, width float64) (key blockIntrinsicKey, ok bool) {
	bits, valid := blockIntrinsicWidthBits(width)
	if !valid {
		return blockIntrinsicKey{}, false //nolint:exhaustruct // zero key means uncacheable
	}

	return blockIntrinsicKey{styleFP: styleFP, contentFP: contentFP, widthBits: bits}, true
}

// Lookup returns the cached height when the same style, content, and width
// measured before. A nil memo, a bypassed memo, an uncacheable width, or an
// unknown key reports a miss.
func (m *blockIntrinsicMemo) Lookup(styleFP, contentFP uint64, width float64) (float64, bool) {
	if m == nil || m.bypass {
		return 0, false
	}

	key, valid := makeBlockIntrinsicKey(styleFP, contentFP, width)
	if !valid {
		return 0, false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.entries == nil {
		return 0, false
	}

	cached, found := m.entries[key]
	if !found {
		return 0, false
	}

	return cached.height, true
}

// Store records one measurement. It is a no-op on a nil or bypassed memo,
// for an uncacheable width, or for a non-finite or negative height, so bad
// inputs never poison later lookups. Re-storing a known key updates the value
// without growing the memo; a new key past the bound evicts the oldest key.
func (m *blockIntrinsicMemo) Store(styleFP, contentFP uint64, width, height float64) {
	if m == nil || m.bypass {
		return
	}

	if math.IsNaN(height) || math.IsInf(height, 0) || height < 0 {
		return
	}

	key, valid := makeBlockIntrinsicKey(styleFP, contentFP, width)
	if !valid {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.entries == nil {
		m.entries = make(map[blockIntrinsicKey]blockIntrinsicResult)
	}

	if _, exists := m.entries[key]; exists {
		m.entries[key] = blockIntrinsicResult{height: height}
		return
	}

	max := m.max
	if max <= 0 {
		max = blockIntrinsicMemoCap
		m.max = max
	}

	if len(m.order) >= max {
		oldest := m.order[0]
		delete(m.entries, oldest)
		m.order = m.order[1:]
	}

	m.entries[key] = blockIntrinsicResult{height: height}
	m.order = append(m.order, key)
}

// Len reports the number of cached entries.
func (m *blockIntrinsicMemo) Len() int {
	if m == nil {
		return 0
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.entries)
}

// Clear drops every entry and resets the eviction order.
func (m *blockIntrinsicMemo) Clear() {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries = nil
	m.order = nil
}
