package experiments

import (
	"fmt"
	"math/rand"
	"slices"
	"testing"
	"time"

	"github.com/dlm/blqsort"
)

// BenchmarkCrossover measures where blqsort starts to beat slices.Sort on
// random int64 data, and whether the answer depends on the benchmark method:
//
//   - same:  every sort in a batch gets a copy of the same random input, as
//     when a benchmark sorts one input over and over. Branch predictors can
//     partly learn a small, repeated input, which favors branchy code.
//   - fresh: every sort gets a different random input.
//
// Each timed batch sorts about 1M elements in total (many sorts at small n),
// so timer overhead does not distort small sizes. Reported as ns/sort.
func BenchmarkCrossover(b *testing.B) {
	batchSortBench(b, []int{100, 300, 1000, 3000, 10_000, 30_000, 100_000}, true)
}

// BenchmarkThreshold checks whether blqsort's switch between its small path
// (parts of up to smallPart+1 = 1025 elements go straight to smallSort) and
// its large partition shows up as a step: sizes cluster around 1025/1026.
// Fresh inputs only; slices.Sort has no such switch and serves as the
// reference curve.
func BenchmarkThreshold(b *testing.B) {
	batchSortBench(b, []int{500, 800, 1000, 1024, 1025, 1026, 1100, 1500, 2000, 3000}, false)
}

// batchSortBench sorts batches of about 1M elements in total (many sorts at
// small n, so timer overhead doesn't distort small sizes), reported as
// ns/sort. With withSame it also runs the "same" variant (every sort gets a
// copy of one input) next to "fresh" (every sort gets a different input).
func batchSortBench(b *testing.B, sizes []int, withSame bool) {
	const batchElems = 1 << 20
	sorts := []struct {
		name string
		sort func([]int64)
	}{{"std", slices.Sort[[]int64]}, {"blq", blqsort.Sort[int64]}}
	for _, n := range sizes {
		k := max(1, batchElems/n) // sorts per batch
		rng := rand.New(rand.NewSource(int64(n)))
		fresh := make([]int64, k*n)
		for i := range fresh {
			fresh[i] = rng.Int63()
		}
		inputs := []struct {
			name string
			data []int64
		}{{"fresh", fresh}}
		if withSame {
			same := make([]int64, k*n)
			for i := range k {
				copy(same[i*n:(i+1)*n], fresh[:n])
			}
			inputs = []struct {
				name string
				data []int64
			}{{"same", same}, {"fresh", fresh}}
		}
		work := make([]int64, k*n)
		for _, in := range inputs {
			for _, s := range sorts {
				b.Run(fmt.Sprintf("n=%d/%s/%s", n, in.name, s.name), func(b *testing.B) {
					var total time.Duration
					batches := 0
					for b.Loop() {
						b.StopTimer()
						copy(work, in.data)
						b.StartTimer()
						t0 := time.Now()
						for i := 0; i < k; i++ {
							s.sort(work[i*n : (i+1)*n])
						}
						total += time.Since(t0)
						batches++
					}
					b.ReportMetric(float64(total.Nanoseconds())/float64(batches*k), "ns/sort")
				})
			}
		}
	}
}
