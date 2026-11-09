package fonts

import (
	"sync"
	"testing"
)

func TestScaleCacheSharesIdenticalRequests(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatalf("LoadDefaultFaces: %v", err)
	}

	face := faces.Regular
	first := SharedScaledFace(face, 16)
	second := SharedScaledFace(face, 16)

	if first == nil || second == nil {
		t.Fatal("SharedScaledFace returned nil for a valid face and size")
	}

	if first != second {
		t.Fatal("identical face and size must return the same scaled instance")
	}

	want := face.GlyphAdvancePoints('A', 16)

	if got := first.Advance('A'); got != want {
		t.Fatalf("Advance('A') = %g, want %g", got, want)
	}

	if got := second.Advance('A'); got != want {
		t.Fatalf("second Advance('A') = %g, want %g", got, want)
	}
}

func TestScaleCacheDistinctSizesGetDistinctInstances(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatalf("LoadDefaultFaces: %v", err)
	}

	small := SharedScaledFace(faces.Regular, 12)
	large := SharedScaledFace(faces.Regular, 20)

	if small == nil || large == nil {
		t.Fatal("SharedScaledFace returned nil for valid sizes")
	}

	if small == large {
		t.Fatal("different sizes must return distinct scaled instances")
	}

	smallAdv := small.Advance('A')
	largeAdv := large.Advance('A')

	if smallAdv <= 0 || largeAdv <= 0 {
		t.Fatalf("advances = %g, %g, want both positive", smallAdv, largeAdv)
	}

	if largeAdv <= smallAdv {
		t.Fatalf("large advance = %g, want greater than small %g", largeAdv, smallAdv)
	}
}

func TestShareAcrossLayoutsReusesFaceAndScaledInstance(t *testing.T) {
	t.Parallel()

	firstFaces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatalf("LoadDefaultFaces: %v", err)
	}

	secondFaces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatalf("LoadDefaultFaces: %v", err)
	}

	if firstFaces.Regular != secondFaces.Regular {
		t.Fatal("two layouts must see the same Regular face pointer")
	}

	first := SharedScaledFace(firstFaces.Regular, 14)
	second := SharedScaledFace(secondFaces.Regular, 14)

	if first != second {
		t.Fatal("two layouts with identical requests must share the scaled instance")
	}

	bold := SharedScaledFace(firstFaces.Bold, 14)
	if bold == first {
		t.Fatal("different weight must return a distinct scaled instance")
	}

	italic := SharedScaledFace(firstFaces.Italic, 14)
	if italic == first || italic == bold {
		t.Fatal("different style must return a distinct scaled instance")
	}
}

func TestScaleCacheConcurrentUseIsRaceClean(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatalf("LoadDefaultFaces: %v", err)
	}

	face := faces.Regular

	const (
		workers = 8
		rounds  = 200
	)

	var group sync.WaitGroup

	for worker := range workers {
		group.Add(1)

		go func(worker int) {
			defer group.Done()

			size := 12.0
			if worker%2 == 1 {
				size = 18.0
			}

			want := SharedScaledFace(face, size)

			for round := range rounds {
				got := SharedScaledFace(face, size)
				if got != want {
					t.Errorf("worker %d round %d: shared instance changed", worker, round)

					return
				}

				first := got.Advance('e')
				second := got.Advance('e')

				if first != second {
					t.Errorf("worker %d round %d: Advance = %g then %g", worker, round, first, second)

					return
				}

				if first <= 0 {
					t.Errorf("worker %d round %d: Advance = %g, want positive", worker, round, first)

					return
				}
			}
		}(worker)
	}

	group.Wait()
}
