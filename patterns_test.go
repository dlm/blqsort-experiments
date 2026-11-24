package experiments

import (
	"fmt"
	"maps"
	"math"
	"math/rand"
	"os"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/dlm/blqsort"
)

// bmInput builds the Bentley-McIlroy dist/mode inputs (see TestBentleyMcIlroy).
func bmInput(rng *rand.Rand, n, m, dist, mode int) []int {
	data := make([]int, n)
	j, k := 0, 1
	for i := range data {
		switch dist {
		case 0:
			data[i] = i % m
		case 1:
			data[i] = rng.Intn(m)
		case 2:
			data[i] = (i*m + i) % n
		case 3:
			data[i] = min(i, m)
		case 4:
			if rng.Intn(m) != 0 {
				j += 2
				data[i] = j
			} else {
				k += 2
				data[i] = k
			}
		}
	}
	md := make([]int, n)
	switch mode {
	case 0:
		copy(md, data)
	case 1:
		for i := range n {
			md[i] = data[n-i-1]
		}
	case 2:
		for i := 0; i < n/2; i++ {
			md[i] = data[n/2-i-1]
		}
		copy(md[n/2:], data[n/2:])
	case 3:
		copy(md[:n/2], data[:n/2])
		for i := n / 2; i < n; i++ {
			md[i] = data[n-(i-n/2)-1]
		}
	case 4:
		copy(md, data)
		slices.Sort(md)
	case 5:
		for i := range n {
			md[i] = data[i] + i%5
		}
	}
	return md
}

func minTime(master []int, f func([]int)) time.Duration {
	data := make([]int, len(master))
	reps := max(1, 2_000_000/max(1, len(master)))
	best := time.Duration(1 << 62)
	for range 7 {
		var d time.Duration
		for range reps {
			copy(data, master)
			t0 := time.Now()
			f(data)
			d += time.Since(t0)
		}
		best = min(best, d/time.Duration(reps))
	}
	return best
}

// TestPatternTimes compares wall time with slices.Sort over the
// Bentley-McIlroy matrix. Opt-in: BLQ_PATTERNS=1, run pinned to one core.
func TestPatternTimes(t *testing.T) {
	if os.Getenv("BLQ_PATTERNS") == "" {
		t.Skip("set BLQ_PATTERNS=1")
	}
	dists := []string{"sawtooth", "rand", "stagger", "plateau", "shuffle"}
	modes := []string{"copy", "reverse", "reverse1", "reverse2", "sort", "dither"}
	rng := rand.New(rand.NewSource(51))
	type res struct {
		desc     string
		ratio    float64
		std, blq time.Duration
		key      string
	}
	var all []res
	for _, n := range []int{100, 1000, 100000} {
		for _, m := range []int{1, 8, 64, n / 2, n} {
			for dist := range dists {
				for mode := range modes {
					in := bmInput(rng, n, m, dist, mode)
					std := minTime(in, slices.Sort[[]int])
					blq := minTime(in, blqsort.Sort[int])
					all = append(all, res{fmt.Sprintf("n=%d m=%d %s/%s", n, m, dists[dist], modes[mode]), float64(std) / float64(blq), std, blq, fmt.Sprintf("n=%-6d %s", n, modes[mode])})
				}
			}
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ratio < all[j].ratio })
	slower := 0
	for _, r := range all {
		if r.ratio < 0.95 {
			slower++
		}
	}
	t.Logf("%d cases; blq slower (>5%%) in %d; median speedup %.2fx", len(all), slower, all[len(all)/2].ratio)
	for _, r := range all[:min(8, len(all))] {
		t.Logf("%-40s speedup %.2fx  std %v  blq %v", r.desc, r.ratio, r.std, r.blq)
	}
	// Geometric mean speedup per size and mode.
	gm := map[string][2]float64{}
	for _, r := range all {
		v := gm[r.key]
		v[0] += math.Log(r.ratio)
		v[1]++
		gm[r.key] = v
	}
	keys := slices.Sorted(maps.Keys(gm))
	for _, k := range keys {
		t.Logf("geomean %-20s %.2fx", k, math.Exp(gm[k][0]/gm[k][1]))
	}
}
