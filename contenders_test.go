package experiments

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"slices"
	"sort"
	"testing"

	"github.com/dlm/blqsort"
)

var contenders = []struct {
	name   string
	ints   func([]int64)
	floats func([]float64)
}{
	{"std", slices.Sort[[]int64], slices.Sort[[]float64]},
	{"pdq-go", PdqSortGo[int64], PdqSortGo[float64]},
	{"pdq-branchless", PdqSortBranchless[int64], PdqSortBranchless[float64]},
	{"blq", blqsort.Sort[int64], blqsort.Sort[float64]},
}

func BenchmarkContenders(b *testing.B) {
	for _, n := range []int{1000, 100_000, 1_000_000, 10_000_000} {
		rng := rand.New(rand.NewSource(42))
		ints, fl := make([]int64, n), make([]float64, n)
		for i := range ints {
			ints[i] = rng.Int63()
			fl[i] = rng.Float64()
		}
		few := make([]int64, n)
		for i := range few {
			few[i] = rng.Int63n(10)
		}
		for _, c := range contenders {
			b.Run(fmt.Sprintf("int64/n=%d/%s", n, c.name), func(b *testing.B) { benchSort(b, ints, c.ints) })
			b.Run(fmt.Sprintf("float64/n=%d/%s", n, c.name), func(b *testing.B) { benchSort(b, fl, c.floats) })
			b.Run(fmt.Sprintf("fewunique/n=%d/%s", n, c.name), func(b *testing.B) { benchSort(b, few, c.ints) })
		}
	}
}

// TestPatternContenders: the Bentley-McIlroy matrix for each contender vs
// slices.Sort. Opt-in: BLQ_PATTERNS=1, pinned.
func TestPatternContenders(t *testing.T) {
	if os.Getenv("BLQ_PATTERNS") == "" {
		t.Skip("set BLQ_PATTERNS=1")
	}
	sorts := map[string]func([]int){"pdq-branchless": PdqSortBranchless[int], "blq": blqsort.Sort[int], "pdq-go": PdqSortGo[int]}
	rng := rand.New(rand.NewSource(51))
	type agg struct {
		logs      []float64
		slower    int
		worst     float64
		worstDesc string
	}
	res := map[string]map[int]*agg{}
	for name := range sorts {
		res[name] = map[int]*agg{}
	}
	for _, n := range []int{100, 1000, 100000} {
		for name := range sorts {
			res[name][n] = &agg{worst: math.Inf(1)}
		}
		for _, m := range []int{1, 8, 64, n / 2, n} {
			for dist := range 5 {
				for mode := range 6 {
					in := bmInput(rng, n, m, dist, mode)
					std := minTime(in, slices.Sort[[]int])
					for name, f := range sorts {
						r := float64(std) / float64(minTime(in, f))
						a := res[name][n]
						a.logs = append(a.logs, math.Log(r))
						if r < 0.95 {
							a.slower++
						}
						if r < a.worst {
							a.worst, a.worstDesc = r, fmt.Sprintf("m=%d dist=%d mode=%d", m, dist, mode)
						}
					}
				}
			}
		}
	}
	names := []string{"pdq-go", "pdq-branchless", "blq"}
	for _, name := range names {
		for _, n := range []int{100, 1000, 100000} {
			a := res[name][n]
			sum := 0.0
			for _, l := range a.logs {
				sum += l
			}
			s := slices.Clone(a.logs)
			sort.Float64s(s)
			t.Logf("%-15s n=%-6d geomean %.2fx  median %.2fx  slower(>5%%) %3d/%d  worst %.2fx (%s)",
				name, n, math.Exp(sum/float64(len(a.logs))), math.Exp(s[len(s)/2]), a.slower, len(a.logs), a.worst, a.worstDesc)
		}
	}
}
