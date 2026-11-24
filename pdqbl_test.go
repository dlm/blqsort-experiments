package experiments

import (
	"math"
	"math/rand"
	"slices"
	"testing"
)

var pdqVariants = []struct {
	name string
	sort func([]int)
}{{"pdq-branchless", PdqSortBranchless[int]}, {"pdq-go", PdqSortGo[int]}}

func TestPdqVariantsBM(t *testing.T) {
	rng := rand.New(rand.NewSource(81))
	for _, n := range []int{13, 50, 63, 64, 65, 127, 128, 129, 200, 1000, 1025, 10000, 100000} {
		for m := 1; m < 2*n; m *= 2 {
			for dist := range 5 {
				for mode := range 6 {
					in := bmInput(rng, n, m, dist, mode)
					want := slices.Clone(in)
					slices.Sort(want)
					for _, v := range pdqVariants {
						got := slices.Clone(in)
						v.sort(got)
						if !slices.Equal(got, want) {
							t.Fatalf("%s n=%d m=%d dist=%d mode=%d: wrong result", v.name, n, m, dist, mode)
						}
					}
				}
			}
		}
	}
}

func TestPdqVariantsRandomSizes(t *testing.T) {
	rng := rand.New(rand.NewSource(82))
	for range 3000 {
		n := rng.Intn(5000)
		k := 1 + rng.Intn(n+1)
		in := make([]int, n)
		for i := range in {
			in[i] = rng.Intn(k)
		}
		want := slices.Clone(in)
		slices.Sort(want)
		for _, v := range pdqVariants {
			got := slices.Clone(in)
			v.sort(got)
			if !slices.Equal(got, want) {
				t.Fatalf("%s n=%d k=%d: wrong result", v.name, n, k)
			}
		}
	}
}

func TestPdqVariantsFloatsStrings(t *testing.T) {
	rng := rand.New(rand.NewSource(83))
	for _, n := range []int{100, 1000, 100000} {
		f := make([]float64, n)
		for i := range f {
			switch rng.Intn(20) {
			case 0:
				f[i] = math.NaN()
			case 1:
				f[i] = math.Inf(rng.Intn(2)*2 - 1)
			default:
				f[i] = rng.NormFloat64()
			}
		}
		want := slices.Clone(f)
		slices.Sort(want)
		got := slices.Clone(f)
		PdqSortBranchless(got)
		if !nanEqual(got, want) {
			t.Fatalf("float64 n=%d: mismatch with slices.Sort", n)
		}
		s := randStrings(rng, n, 3, "")
		ws := slices.Clone(s)
		slices.Sort(ws)
		PdqSortBranchless(s)
		if !slices.Equal(s, ws) {
			t.Fatalf("strings n=%d: wrong result", n)
		}
	}
}

func FuzzPdqBranchless(f *testing.F) {
	f.Add([]byte{3, 1, 2})
	f.Fuzz(func(t *testing.T, b []byte) {
		in := make([]int8, len(b))
		for i, c := range b {
			in[i] = int8(c)
		}
		want := slices.Clone(in)
		slices.Sort(want)
		PdqSortBranchless(in)
		if !slices.Equal(in, want) {
			t.Fatalf("got %v want %v", in, want)
		}
	})
}
