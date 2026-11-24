package experiments

import (
	"math/rand"
	"os"
	"slices"
	"testing"

	"github.com/dlm/blqsort"
)

// sortedRuns returns n random ints arranged as sorted runs of length l,
// alternating ascending/descending when alt is set.
func sortedRuns(rng *rand.Rand, n, l int, alt bool) []int {
	d := make([]int, n)
	for i := range d {
		d[i] = rng.Int()
	}
	for s, k := 0, 0; s < n; s, k = s+l, k+1 {
		r := d[s:min(s+l, n)]
		slices.Sort(r)
		if alt && k%2 == 1 {
			slices.Reverse(r)
		}
	}
	return d
}

// TestRunCrossover finds where pdqsort's adaptivity beats blqsort.
// Opt-in: BLQ_PATTERNS=1, pinned.
func TestRunCrossover(t *testing.T) {
	if os.Getenv("BLQ_PATTERNS") == "" {
		t.Skip("set BLQ_PATTERNS=1")
	}
	rng := rand.New(rand.NewSource(61))
	for _, n := range []int{1000, 100000} {
		for _, alt := range []bool{false, true} {
			for l := 2; l <= n; l *= 4 {
				in := sortedRuns(rng, n, l, alt)
				std := minTime(in, slices.Sort[[]int])
				blq := minTime(in, blqsort.Sort[int])
				t.Logf("n=%-6d alt=%-5v run=%-6d std %10v  blq %10v  speedup %.2fx", n, alt, l, std, blq, float64(std)/float64(blq))
			}
		}
	}
}
