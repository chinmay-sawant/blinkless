package layout

// flowIndexStorage retains page-index slices so a later rebuild can reuse
// the backing arrays. The PDF pagination pass that filled these indexes is
// gone. Result still carries the fields so clones stay independent.
type flowIndexStorage struct {
	pages  [][]int
	pageOf []int
	pos    []int
	counts []int
}

func (s *flowIndexStorage) reset() {
	*s = flowIndexStorage{} //nolint:exhaustruct // zero value clears the retained arrays
}
