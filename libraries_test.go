package experiments

import (
	"fmt"
	"math/rand"
	"slices"
	"testing"

	"github.com/jfcg/sorty/v2"
	"github.com/shawnsmithdev/zermelo/v2"
	"github.com/shawnsmithdev/zermelo/v2/floats"

	"github.com/dlm/blqsort"
)

// Single-threaded comparisons: sorty is concurrent by default (MaxGor
// goroutines); it is run with MaxGor = 1 here, like the others.

type intSorter struct {
	name string
	sort func(data, scratch []int64)
}

type floatSorter struct {
	name string
	sort func(data, scratch []float64)
}

func singleThreadedSorty(ar any) {
	old := sorty.MaxGor
	sorty.MaxGor = 1
	sorty.SortSlice(ar)
	sorty.MaxGor = old
}

var intLibs = []intSorter{
	{"slices.Sort", func(d, _ []int64) { slices.Sort(d) }},
	{"blqsort.Sort", func(d, _ []int64) { blqsort.Sort(d) }},
	{"zermelo.Sort", func(d, _ []int64) { zermelo.Sort(d) }},            // radix, allocates a buffer
	{"zermelo.SortBYOB", func(d, s []int64) { zermelo.SortBYOB(d, s) }}, // radix, caller's buffer
	{"sorty.SortSlice", func(d, _ []int64) { singleThreadedSorty(d) }},
}

var floatLibs = []floatSorter{
	{"slices.Sort", func(d, _ []float64) { slices.Sort(d) }},
	{"blqsort.Sort", func(d, _ []float64) { blqsort.Sort(d) }},
	{"zermelo.SortFloats", func(d, _ []float64) { floats.SortFloats(d) }},
	{"zermelo.SortFloatsBYOB", func(d, s []float64) { floats.SortFloatsBYOB(d, s) }},
	{"sorty.SortSlice", func(d, _ []float64) { singleThreadedSorty(d) }},
}

// libInputs are int64 inputs: random data plus the structured shapes from
// the stdlib benchmarks and the pattern experiments.
func libInputs(n int) []struct {
	name string
	data []int64
} {
	rng := rand.New(rand.NewSource(42))
	mk := func(f func(i int) int64) []int64 {
		d := make([]int64, n)
		for i := range d {
			d[i] = f(i)
		}
		return d
	}
	return []struct {
		name string
		data []int64
	}{
		{"random", mk(func(int) int64 { return rng.Int63() })},
		{"fewunique", mk(func(int) int64 { return rng.Int63n(10) })},
		{"sorted", mk(func(i int) int64 { return int64(i) })},
		{"reversed", mk(func(i int) int64 { return int64(n - i) })},
		{"mod8", mk(func(i int) int64 { return int64(i % 8) })},
		{"xor", mk(func(i int) int64 { return int64(i ^ 0xcccc) })},
	}
}

func TestLibrariesCorrect(t *testing.T) {
	for _, n := range []int{0, 1, 100, 5000, 100000} {
		for _, in := range libInputs(n) {
			want := slices.Clone(in.data)
			slices.Sort(want)
			for _, l := range intLibs {
				got := slices.Clone(in.data)
				l.sort(got, make([]int64, n))
				if !slices.Equal(got, want) {
					t.Fatalf("%s %s n=%d: wrong result", l.name, in.name, n)
				}
			}
		}
		rng := rand.New(rand.NewSource(1))
		f := make([]float64, n)
		for i := range f {
			f[i] = rng.NormFloat64()
		}
		want := slices.Clone(f)
		slices.Sort(want)
		for _, l := range floatLibs {
			got := slices.Clone(f)
			l.sort(got, make([]float64, n))
			if !slices.Equal(got, want) {
				t.Fatalf("%s float64 n=%d: wrong result", l.name, n)
			}
		}
	}
}

// BenchmarkLibraries: random int64 and float64 at several sizes.
func BenchmarkLibraries(b *testing.B) {
	for _, n := range []int{1000, 100_000, 1_000_000, 10_000_000} {
		rng := rand.New(rand.NewSource(42))
		ints, fl := make([]int64, n), make([]float64, n)
		for i := range ints {
			ints[i] = rng.Int63()
			fl[i] = rng.Float64()
		}
		iscratch, fscratch := make([]int64, n), make([]float64, n)
		for _, l := range intLibs {
			b.Run(fmt.Sprintf("int64/n=%d/%s", n, l.name), func(b *testing.B) {
				b.ReportAllocs()
				benchSort(b, ints, func(d []int64) { l.sort(d, iscratch) })
			})
		}
		for _, l := range floatLibs {
			b.Run(fmt.Sprintf("float64/n=%d/%s", n, l.name), func(b *testing.B) {
				b.ReportAllocs()
				benchSort(b, fl, func(d []float64) { l.sort(d, fscratch) })
			})
		}
	}
}

// BenchmarkLibraryShapes: int64 inputs of different shapes at 1M.
func BenchmarkLibraryShapes(b *testing.B) {
	const n = 1_000_000
	scratch := make([]int64, n)
	for _, in := range libInputs(n) {
		for _, l := range intLibs {
			b.Run(in.name+"/"+l.name, func(b *testing.B) {
				benchSort(b, in.data, func(d []int64) { l.sort(d, scratch) })
			})
		}
	}
}
