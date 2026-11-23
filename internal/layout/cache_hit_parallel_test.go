package layout

import (
	"sync"
	"testing"
)

// TestVerdictHitsParallelCounted hammers cache hits from many goroutines
// under -race and checks the atomic hit counters lose no increments.
func TestVerdictHitsParallelCounted(t *testing.T) { //nolint:paralleltest // global caches and counters
	resetAcceptVerdictForTest()
	resetCascadeCachesForTest()
	supportedDeclaration("width", "10px")
	engineSupportsProperty("display", "grid")

	const workers, per = 8, 500

	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range per {
				supportedDeclaration("width", "10px")
				engineSupportsProperty("display", "grid")
			}
		}()
	}

	wg.Wait()

	if _, hits := acceptVerdictStatsForTest(); hits < workers*per {
		t.Fatalf("accept hits = %d, want >= %d", hits, workers*per)
	}

	if _, hits := supportsProbeStatsForTest(); hits < workers*per {
		t.Fatalf("supports hits = %d, want >= %d", hits, workers*per)
	}
}

func BenchmarkSupportedDeclarationHitParallel(b *testing.B) {
	supportedDeclaration("width", "10px")
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			supportedDeclaration("width", "10px")
		}
	})
}
