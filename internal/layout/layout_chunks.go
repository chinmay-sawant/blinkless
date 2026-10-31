package layout

import (
	"sync"

	"github.com/chinmay-sawant/blinkless/internal/html"
)

// Release returns resources associated with Result back to pools.
func (r *Result) Release() {
	if r == nil {
		return
	}

	for _, chunk := range r.styleChunks {
		putStyleChunk(chunk)
	}

	r.styleChunks = nil

	for _, chunk := range r.boxChunks {
		putBoxChunk(chunk)
	}

	r.boxChunks = nil
}

const defaultBoxChunkSize = 64

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

//nolint:gochecknoglobals // global pool for contiguous resolved style chunks
var styleChunkPool = sync.Pool{
	New: func() any {
		b := make([]ResolvedStyle, 0, defaultBoxChunkSize)

		return &b
	},
}

//nolint:forcetypeassert // pool stores *[]ResolvedStyle
func getStyleChunk() *[]ResolvedStyle {
	chunk := styleChunkPool.Get().(*[]ResolvedStyle)
	*chunk = (*chunk)[:0]

	return chunk
}

func putStyleChunk(chunk *[]ResolvedStyle) {
	if chunk == nil {
		return
	}

	clear(*chunk)
	*chunk = (*chunk)[:0]
	styleChunkPool.Put(chunk)
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
		for i := mark.chunkIdx + 1; i < len(e.styleChunks); i++ {
			clear(*e.styleChunks[i])
			*e.styleChunks[i] = (*e.styleChunks[i])[:0]
			e.spareChunks = append(e.spareChunks, e.styleChunks[i])
		}

		e.styleChunks = e.styleChunks[:mark.chunkIdx+1]
		chunk := e.styleChunks[mark.chunkIdx]

		if mark.itemIdx < len(*chunk) {
			clear((*chunk)[mark.itemIdx:])
			*chunk = (*chunk)[:mark.itemIdx]
		}
	}
}

func (e *engine) allocStyle(base *ResolvedStyle) *ResolvedStyle {
	if len(e.styleChunks) == 0 || len(*e.styleChunks[len(e.styleChunks)-1]) == cap(*e.styleChunks[len(e.styleChunks)-1]) {
		if len(e.spareChunks) > 0 {
			last := len(e.spareChunks) - 1
			spare := e.spareChunks[last]
			*spare = (*spare)[:0]
			e.spareChunks = e.spareChunks[:last]
			e.styleChunks = append(e.styleChunks, spare)
		} else {
			e.styleChunks = append(e.styleChunks, getStyleChunk())
		}
	}

	last := len(e.styleChunks) - 1
	chunk := e.styleChunks[last]

	if base != nil {
		*chunk = append(*chunk, *base)
	} else {
		*chunk = append(*chunk, zeroResolvedStyle)
	}

	return &(*chunk)[len(*chunk)-1]
}
