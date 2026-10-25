package layout

import (
	"sync"
	"testing"
)

// TestRestampResetLeavesSharedExtraReadOnly proves resetWindowTransforms does
// not write the global emptyExtra. Every op that never carried a rare payload
// points at that singleton, including ops in other displays that a replay
// goroutine may read through Transform(). The reader owns a separate Op value,
// so the shared extra is the only memory the two goroutines have in common.
func TestRestampResetLeavesSharedExtraReadOnly(t *testing.T) {
	t.Parallel()

	const iterations = 1000

	writer := []Op{
		{Kind: OpFillRect, X: 0, Y: 0, W: 10, H: 10},
		{Kind: OpFillRect, X: 10, Y: 0, W: 10, H: 10},
	}
	for idx := range writer {
		writer[idx].bindEmptyExtra()
	}

	var reader Op

	reader.bindEmptyExtra()

	want := reader.Transform()
	start := make(chan struct{})

	var group sync.WaitGroup

	group.Add(2)

	go func() {
		defer group.Done()

		<-start

		for range iterations {
			resetWindowTransforms(writer, OpWindow{Start: 0, End: len(writer) - 1})
		}
	}()

	go func() {
		defer group.Done()

		<-start

		for range iterations {
			if got := reader.Transform(); got != want {
				t.Errorf("shared extra Transform = %+v, want the unchanged %+v", got, want)

				return
			}
		}
	}()

	close(start)
	group.Wait()
}
