package layout

import (
	"sync"
	"sync/atomic"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// Retain increments the reference counter of r.
func (r *Result) Retain() {
	if r == nil {
		return
	}

	atomic.AddInt32(&r.refs, 1)
}

// DisownChunks clears the shared style chunk slices on r without returning
// them to pools, transferring ownership of those chunks to a clone. Box
// chunks stay owned by r because CloneResult allocates fresh box chunks for
// every cloned box.
func (r *Result) DisownChunks() {
	if r == nil {
		return
	}

	r.styleChunks = nil
	r.storeChunks = nil
}

// adoptRelayoutEngine transfers the live box and style chunks built by a
// winning subtree relayout engine onto prev so they recycle when prev is
// released, and immediately returns unused spare chunks and scratch caches.
func adoptRelayoutEngine(prev *Result, eng *engine) {
	if eng == nil {
		return
	}

	if prev != nil {
		adoptEngineOwnedChunks(prev, eng)
	} else {
		releaseEngineOwnedChunks(eng)
	}

	eng.boxChunks = nil
	eng.styleChunks = nil

	for _, chunk := range eng.spareBoxChunks {
		putBoxChunk(chunk)
	}

	eng.spareBoxChunks = nil

	for _, chunk := range eng.spareChunks {
		putStyleChunk(chunk)
	}

	eng.spareChunks = nil
	eng.releaseScratch()
}

func adoptEngineOwnedChunks(prev *Result, eng *engine) {
	prev.boxChunks = append(prev.boxChunks, eng.boxChunks...)

	if !eng.isSubtree {
		prev.styleChunks = append(prev.styleChunks, eng.styleChunks...)
		prev.styleChunks = append(prev.styleChunks, eng.overrideStyleChunks...)
	} else {
		adoptSubtreeFlexOverrides(prev, eng)
	}

	eng.overrideStyleChunks = nil
}

func adoptSubtreeFlexOverrides(prev *Result, eng *engine) {
	if len(eng.flexStyleOverrides) == 0 || len(prev.flexOverrides) >= flexOverrideMaxEntries {
		return
	}

	merged := make(map[flexStyleOverrideKey]*ResolvedStyle, len(prev.flexOverrides)+len(eng.flexStyleOverrides))
	for k, v := range prev.flexOverrides {
		merged[k] = v
	}

	for k, v := range eng.flexStyleOverrides {
		if len(merged) >= flexOverrideMaxEntries {
			break
		}

		merged[k] = v
	}

	prev.flexOverrides = merged
}

func releaseEngineOwnedChunks(eng *engine) {
	for _, chunk := range eng.boxChunks {
		putBoxChunk(chunk)
	}

	for _, chunk := range eng.styleChunks {
		putStyleChunk(chunk)
	}
}

// discardRelayoutEngine returns every chunk and scratch cache owned by an
// aborted subtree relayout engine back to its pool.
func discardRelayoutEngine(eng *engine) {
	adoptRelayoutEngine(nil, eng)
}

// Release returns resources associated with Result back to pools
// when the reference count reaches zero.
func (r *Result) Release() {
	if r == nil {
		return
	}

	if atomic.AddInt32(&r.refs, -1) > 0 {
		return
	}

	// The flattened box list was rented at exact capacity in finalizeResult
	// and assembleNodeResult. The public display copied box values out of it
	// before Release, and Release only runs on the non-retained path, so no
	// live display aliases this backing: hand it back.
	returnBoxPtrs(r.boxes)
	r.boxes = nil

	if r.ownsOps && r.Ops != nil {
		returnOpSlice(r.Ops)
		r.Ops = nil
		r.ownsOps = false
	}

	for _, chunk := range r.styleChunks {
		putStyleChunk(chunk)
	}

	r.styleChunks = nil

	for _, chunk := range r.boxChunks {
		putBoxChunk(chunk)
	}

	r.boxChunks = nil

	for i := range r.storeChunks {
		putStyleStoreChunk(&r.storeChunks[i])
	}

	r.storeChunks = nil
}

const (
	defaultBoxChunkSize = 64
	// defaultStyleChunkSize is smaller than defaultBoxChunkSize because one
	// ResolvedStyle is about 5.8 KB: a 64-entry style chunk is about 375 KB.
	// allocStyle uses a chunk for flex and grid override styles, including the
	// private aspect-ratio copy.
	defaultStyleChunkSize = 16
	// maxStyleChunksRetained bounds the style free list below the 4 MiB test
	// bound in TestStyleChunkPoolMemoryBound. With Declared and Triggers stored
	// lazily behind pointers, 40 chunks stay well below the 4 MiB cap while
	// retaining more backing across successive full layouts.
	maxStyleChunksRetained = 40
)

//nolint:gochecknoglobals // global pool for contiguous box chunks
var boxChunkPool = sync.Pool{
	New: func() any {
		b := make([]box, 0, defaultBoxChunkSize)

		return &b
	},
}

//nolint:forcetypeassert // pool stores *[]box
func getBoxChunk() *[]box {
	chunk := boxChunkPool.Get().(*[]box)
	*chunk = (*chunk)[:0]

	return chunk
}

func putBoxChunk(chunk *[]box) {
	if chunk == nil {
		return
	}

	clear(*chunk)
	*chunk = (*chunk)[:0]
	boxChunkPool.Put(chunk)
}

type boxMark struct {
	chunkIdx int
	itemIdx  int
}

func (e *engine) markBoxes() boxMark {
	if len(e.boxChunks) == 0 {
		return boxMark{chunkIdx: 0, itemIdx: 0}
	}

	last := len(e.boxChunks) - 1

	return boxMark{chunkIdx: last, itemIdx: len(*e.boxChunks[last])}
}

func (e *engine) resetBoxes(mark boxMark) {
	if len(e.boxChunks) == 0 {
		return
	}

	if mark.chunkIdx < len(e.boxChunks) {
		for i := mark.chunkIdx + 1; i < len(e.boxChunks); i++ {
			clear(*e.boxChunks[i])
			*e.boxChunks[i] = (*e.boxChunks[i])[:0]
			e.spareBoxChunks = append(e.spareBoxChunks, e.boxChunks[i])
		}

		e.boxChunks = e.boxChunks[:mark.chunkIdx+1]
		chunk := e.boxChunks[mark.chunkIdx]

		if mark.itemIdx < len(*chunk) {
			clear((*chunk)[mark.itemIdx:])
			*chunk = (*chunk)[:mark.itemIdx]
		}
	}
}

func (e *engine) allocBox(node *html.Node, sty *ResolvedStyle, kind boxKind, posX, posY float64) *box {
	if len(e.boxChunks) == 0 || len(*e.boxChunks[len(e.boxChunks)-1]) == cap(*e.boxChunks[len(e.boxChunks)-1]) {
		if len(e.spareBoxChunks) > 0 {
			last := len(e.spareBoxChunks) - 1
			spare := e.spareBoxChunks[last]
			*spare = (*spare)[:0]
			e.spareBoxChunks = e.spareBoxChunks[:last]
			e.boxChunks = append(e.boxChunks, spare)
		} else {
			e.boxChunks = append(e.boxChunks, getBoxChunk())
		}
	}

	last := len(e.boxChunks) - 1
	chunk := e.boxChunks[last]
	*chunk = append(*chunk, box{ //nolint:exhaustruct // intentional zero fields
		node:  node,
		style: sty,
		kind:  kind,
		x:     posX,
		y:     posY,
	})

	return &(*chunk)[len(*chunk)-1]
}

// styleChunkFreeList is a bounded free list of resolved-style chunks. A
// buffered channel keeps the bound exact: sync.Pool drops entries at garbage
// collection, so a counter over its contents would drift above the live
// count and stop pooling for the rest of the process. A return that finds
// the list full is dropped, which keeps one layout run from pinning memory
// for later small ones.
type styleChunkFreeList struct {
	free chan *[]ResolvedStyle
}

func newStyleChunkFreeList(limit int) styleChunkFreeList {
	return styleChunkFreeList{free: make(chan *[]ResolvedStyle, limit)}
}

// get returns a zero-length chunk, reusing a retained one when available.
func (p *styleChunkFreeList) get() *[]ResolvedStyle {
	select {
	case chunk := <-p.free:
		*chunk = (*chunk)[:0]

		return chunk
	default:
	}

	chunk := make([]ResolvedStyle, 0, defaultStyleChunkSize)

	return &chunk
}

const subtreeStyleChunkSize = 1

// put clears chunk and retains it when the list has room. Only full-size
// chunks enter the pool so oversized or tiny subtree backings do not distort
// pool capacity.
func (p *styleChunkFreeList) put(chunk *[]ResolvedStyle) {
	if chunk == nil || cap(*chunk) != defaultStyleChunkSize {
		return
	}

	clear((*chunk)[:cap(*chunk)])
	*chunk = (*chunk)[:0]

	select {
	case p.free <- chunk:
	default:
	}
}

// retained reports how many chunks the free list holds.
func (p *styleChunkFreeList) retained() int {
	return len(p.free)
}

//nolint:gochecknoglobals // global free list for contiguous resolved style chunks
var styleChunkPool = newStyleChunkFreeList(maxStyleChunksRetained)

func getStyleChunk() *[]ResolvedStyle {
	return styleChunkPool.get()
}

func putStyleChunk(chunk *[]ResolvedStyle) {
	styleChunkPool.put(chunk)
}

type styleMark struct {
	chunkIdx int
	itemIdx  int
}

func (e *engine) markStyles() styleMark {
	if len(e.styleChunks) == 0 {
		return styleMark{chunkIdx: 0, itemIdx: 0}
	}

	last := len(e.styleChunks) - 1

	return styleMark{chunkIdx: last, itemIdx: len(*e.styleChunks[last])}
}

func (e *engine) resetStyles(mark styleMark) {
	if len(e.styleChunks) == 0 {
		return
	}

	if mark.chunkIdx < len(e.styleChunks) {
		rewound := false

		for i := mark.chunkIdx + 1; i < len(e.styleChunks); i++ {
			clear(*e.styleChunks[i])
			*e.styleChunks[i] = (*e.styleChunks[i])[:0]
			e.spareChunks = append(e.spareChunks, e.styleChunks[i])
			rewound = true
		}

		e.styleChunks = e.styleChunks[:mark.chunkIdx+1]
		chunk := e.styleChunks[mark.chunkIdx]

		if mark.itemIdx < len(*chunk) {
			clear((*chunk)[mark.itemIdx:])
			*chunk = (*chunk)[:mark.itemIdx]
			rewound = true
		}

		if rewound && len(e.anonFlexStyles) > 0 {
			clear(e.anonFlexStyles)
		}
	}
}

func (e *engine) allocStyle(base *ResolvedStyle) *ResolvedStyle {
	e.nextStyleChunk(&e.styleChunks)

	chunk := e.styleChunks[len(e.styleChunks)-1]

	if base != nil {
		*chunk = append(*chunk, *base)
	} else {
		*chunk = append(*chunk, zeroResolvedStyle)
	}

	return &(*chunk)[len(*chunk)-1]
}

// nextStyleChunk makes sure chunks ends in a chunk with room for one more
// record, reusing a spare chunk before renting a fresh one.
func (e *engine) nextStyleChunk(chunks *[]*[]ResolvedStyle) {
	if len(*chunks) > 0 {
		lastChunk := (*chunks)[len(*chunks)-1]
		if len(*lastChunk) < cap(*lastChunk) {
			return
		}
	}

	if len(e.spareChunks) > 0 {
		last := len(e.spareChunks) - 1
		spare := e.spareChunks[last]
		*spare = (*spare)[:0]
		e.spareChunks = e.spareChunks[:last]

		*chunks = append(*chunks, spare)

		return
	}

	if e.isSubtree {
		small := make([]ResolvedStyle, 0, subtreeStyleChunkSize)
		*chunks = append(*chunks, &small)

		return
	}

	*chunks = append(*chunks, getStyleChunk())
}
