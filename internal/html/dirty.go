package html

import (
	"maps"
	"slices"
	"sync"
)

// DirtySet is the monotonic dirty record for one layout run. It holds the
// ids of elements whose style or layout may be stale, plus one flag for a
// structural change (tag replace, insert, or remove) that a bare id cannot
// describe. Marking only adds: entries and the flag survive until Clear runs
// after the run adopts them, so concurrent patches cannot lose each other.
type DirtySet struct {
	mu         sync.RWMutex
	ids        map[string]struct{}
	structural bool
}

// NewDirtySet returns an empty DirtySet.
func NewDirtySet() *DirtySet {
	return &DirtySet{ids: make(map[string]struct{})}
}

// Mark records id as dirty. An empty id is ignored. Calling Mark twice for
// the same id keeps one entry.
func (s *DirtySet) Mark(id string) {
	if s == nil || id == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ids == nil {
		s.ids = make(map[string]struct{})
	}

	s.ids[id] = struct{}{}
}

// MarkStructural records a structural change: a tag replace, an insert, or a
// remove. The flag stays set until Clear, even when the edited element keeps
// its id, because the shape of the subtree changed.
func (s *DirtySet) MarkStructural() {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.structural = true
}

// Clear drops every dirty id and the structural flag after the layout run
// adopts them. The set is empty afterwards and ready for the next run.
func (s *DirtySet) Clear() {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	clear(s.ids)
	s.structural = false
}

// Snapshot returns the dirty ids in sorted order. The order is deterministic
// so tests and the layout run see the same sequence for the same edits.
func (s *DirtySet) Snapshot() []string {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Sorted(maps.Keys(s.ids))
}

// Contains reports whether id is dirty.
func (s *DirtySet) Contains(id string) bool {
	if s == nil || id == "" {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.ids[id]

	return ok
}

// Len returns the number of dirty ids.
func (s *DirtySet) Len() int {
	if s == nil {
		return 0
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.ids)
}

// Structural reports whether a structural change was recorded since the last
// Clear.
func (s *DirtySet) Structural() bool {
	if s == nil {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.structural
}

// IsEmpty reports whether nothing is dirty and no structural change is
// recorded.
func (s *DirtySet) IsEmpty() bool {
	if s == nil {
		return true
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.ids) == 0 && !s.structural
}
