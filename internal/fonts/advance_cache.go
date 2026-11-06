package fonts

import (
	"math"
	"sync"
)

// maxAdvUnitsCacheEntries bounds the per-Font advance-units cache. A trade
// price table reuses a dozen distinct runes; a large CJK document can exceed
// the cap, in which case new runes compute without storing so the map cannot
// grow with the document.
const maxAdvUnitsCacheEntries = 2048

// initialAdvUnitsCacheEntries sizes a fresh per-Font advance-units map. A
// trade price table reuses a dozen distinct runes; the map grows no further
// than maxAdvUnitsCacheEntries.
const initialAdvUnitsCacheEntries = 64

// maxScaledFaces bounds the shared scaled-face cache. A document with a
// dozen sizes stays far below the cap, while a hostile size sweep stops
// storing instead of pinning one entry per distinct size.
const maxScaledFaces = 256

// maxScaledPointsEntries bounds one ScaledFace points-advance map. It mirrors
// maxAdvUnitsCacheEntries so a large CJK document cannot grow the map with
// the document.
const maxScaledPointsEntries = 2048

// initialScaledPointsEntries sizes a fresh ScaledFace points-advance map.
const initialScaledPointsEntries = 64

// cachedAdvanceUnits returns the cached advance width in font units for r.
func (f *Font) cachedAdvanceUnits(runeValue rune) (float64, bool) {
	f.advMu.RLock()
	defer f.advMu.RUnlock()

	if len(f.advUnits) == 0 {
		return 0, false
	}

	units, ok := f.advUnits[runeValue]

	return units, ok
}

// storeAdvanceUnits records the advance width in font units for r unless the
// cache is full.
func (f *Font) storeAdvanceUnits(runeValue rune, units float64) {
	f.advMu.Lock()
	defer f.advMu.Unlock()

	if f.advUnits == nil {
		f.advUnits = make(map[rune]float64, initialAdvUnitsCacheEntries)
	}

	if len(f.advUnits) >= maxAdvUnitsCacheEntries {
		return
	}

	f.advUnits[runeValue] = units
}

// advanceUnitsUncached computes the advance width in font units for r with a
// single cmap lookup, matching Advance.
func (f *Font) advanceUnitsUncached(runeValue rune) float64 {
	if len(f.advance) == 0 {
		return 0
	}

	g := f.GlyphID(runeValue)
	if int(g) >= len(f.advance) {
		return float64(f.advance[0])
	}

	return float64(f.advance[g])
}

// advanceUnits returns the advance width in font units for r, serving
// repeated runes from the per-Font cache.
func (f *Font) advanceUnits(runeValue rune) float64 {
	if units, ok := f.cachedAdvanceUnits(runeValue); ok {
		return units
	}

	units := f.advanceUnitsUncached(runeValue)
	f.storeAdvanceUnits(runeValue, units)

	return units
}

// scaleCacheKey identifies one shared scaled face: the resolved face pointer
// plus the requested size bits. The face pointer already encodes family,
// weight, and style because FaceSet resolution returns a distinct *Font per
// style variant and Instance returns a distinct *Font per variation set.
type scaleCacheKey struct {
	face     *Font
	sizeBits uint64
}

// ScaledFace is one face at one size with a shared points-advance cache. Face
// and Size are immutable after creation. Advance is safe for concurrent use,
// and SharedScaledFace shares each instance across layouts.
type ScaledFace struct {
	face   *Font
	size   float64
	mu     sync.RWMutex
	points map[rune]float64
}

// Face returns the underlying face.
func (s *ScaledFace) Face() *Font {
	if s == nil {
		return nil
	}

	return s.face
}

// Size returns the requested size in points.
func (s *ScaledFace) Size() float64 {
	if s == nil {
		return 0
	}

	return s.size
}

// Advance returns the advance width in points for r at the face size, serving
// repeated runes from the shared points cache.
func (s *ScaledFace) Advance(runeValue rune) float64 {
	if s == nil || s.face == nil {
		return 0
	}

	s.mu.RLock()

	if len(s.points) > 0 {
		if value, ok := s.points[runeValue]; ok {
			s.mu.RUnlock()

			return value
		}
	}

	s.mu.RUnlock()

	value := s.face.GlyphAdvancePoints(runeValue, s.size)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.points == nil {
		s.points = make(map[rune]float64, initialScaledPointsEntries)
	}

	if existing, ok := s.points[runeValue]; ok {
		return existing
	}

	if len(s.points) >= maxScaledPointsEntries {
		return value
	}

	s.points[runeValue] = value

	return value
}

// scaledFaceCache is the cross-layout store of ScaledFace values. The RWMutex
// guards every map access so concurrent layouts share without racing.
type scaledFaceCache struct {
	mu      sync.RWMutex
	entries map[scaleCacheKey]*ScaledFace
}

//nolint:gochecknoglobals // cross-layout scaled-face cache, guarded by RWMutex.
var sharedScaledFaces = &scaledFaceCache{ //nolint:exhaustruct // mutex zero value is intentional.
	entries: make(map[scaleCacheKey]*ScaledFace),
}

// SharedScaledFace returns the shared scaled face for one resolved face at one
// size. Identical requests return the same pointer, so their points-advance
// caches survive across layouts. A nil face or a non-positive or non-finite
// size bypasses the cache with a transient instance. When the cache is full,
// the call also returns a transient instance that still measures correctly.
func SharedScaledFace(face *Font, size float64) *ScaledFace {
	if face == nil || !isValidScaleSize(size) {
		return &ScaledFace{ //nolint:exhaustruct // transient, never stored.
			face: face,
			size: size,
		}
	}

	key := scaleCacheKey{face: face, sizeBits: math.Float64bits(size)}

	sharedScaledFaces.mu.RLock()

	if cached, ok := sharedScaledFaces.entries[key]; ok {
		sharedScaledFaces.mu.RUnlock()

		return cached
	}

	sharedScaledFaces.mu.RUnlock()

	sharedScaledFaces.mu.Lock()
	defer sharedScaledFaces.mu.Unlock()

	if cached, ok := sharedScaledFaces.entries[key]; ok {
		return cached
	}

	if len(sharedScaledFaces.entries) >= maxScaledFaces {
		return &ScaledFace{ //nolint:exhaustruct // cache full, never stored.
			face: face,
			size: size,
		}
	}

	scaled := &ScaledFace{ //nolint:exhaustruct // points map grows on first Advance.
		face: face,
		size: size,
	}

	sharedScaledFaces.entries[key] = scaled

	return scaled
}

// isValidScaleSize reports whether size can key the shared cache: finite and
// positive, matching the layout zoom contract.
func isValidScaleSize(size float64) bool {
	if math.IsNaN(size) || math.IsInf(size, 0) {
		return false
	}

	return size > 0
}
