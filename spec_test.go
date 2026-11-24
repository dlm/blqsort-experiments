package experiments

import (
	"cmp"
	"math/bits"
	"math/rand"
	"slices"
	"testing"

	"github.com/dlm/blqsort"
)

func cmpRec2(a, b rec) int { return cmp.Or(cmp.Compare(a.id, b.id), cmp.Compare(a.value, b.value)) }

func pdq[T any](f func([]T, int, int, int)) func([]T) {
	return func(x []T) { f(x, 0, len(x), bits.Len(uint(len(x)))) }
}

type variant[T any] struct {
	name string
	sort func([]T)
}

// checkSort sorts copies of data with every variant and compares against
// slices.SortFunc under the full ordering (byKey for the check, full for
// detecting lost or duplicated elements).
func checkSort[T any](t *testing.T, data []T, byKey, full func(a, b T) int, vs []variant[T]) {
	t.Helper()
	want := slices.Clone(data)
	slices.SortFunc(want, full)
	for _, v := range vs {
		got := slices.Clone(data)
		v.sort(got)
		if !slices.IsSortedFunc(got, byKey) {
			t.Fatalf("%s n=%d: not sorted", v.name, len(data))
		}
		slices.SortFunc(got, full)
		if !slices.EqualFunc(got, want, func(a, b T) bool { return full(a, b) == 0 }) {
			t.Fatalf("%s n=%d: elements lost or duplicated", v.name, len(data))
		}
	}
}

func genRecs[T any](rng *rand.Rand, n, keys int, mk func(id int64, i int) T) []T {
	s := make([]T, n)
	for i := range s {
		s[i] = mk(rng.Int63n(int64(keys)), i)
	}
	return s
}

func mkRec(id int64, i int) rec     { return rec{int32(id), int32(i)} }
func mkRec16(id int64, i int) rec16 { return rec16{id, [1]int64{int64(i)}} }
func mkRec32(id int64, i int) rec32 { return rec32{id, [3]int64{int64(i)}} }
func mkRec64(id int64, i int) rec64 { return rec64{id, [7]int64{int64(i)}} }

func byID[T interface{ rec16 | rec32 | rec64 }](id func(T) int64, pad0 func(T) int64) (func(a, b T) int, func(a, b T) int) {
	k := func(a, b T) int { return cmp.Compare(id(a), id(b)) }
	return k, func(a, b T) int { return cmp.Or(k(a, b), cmp.Compare(pad0(a), pad0(b))) }
}

var (
	k16, f16 = byID(func(r rec16) int64 { return r.id }, func(r rec16) int64 { return r.pad[0] })
	k32, f32 = byID(func(r rec32) int64 { return r.id }, func(r rec32) int64 { return r.pad[0] })
	k64, f64 = byID(func(r rec64) int64 { return r.id }, func(r rec64) int64 { return r.pad[0] })
	fRec     = func(a, b rec) int { return cmp.Or(cmpRec(a, b), cmp.Compare(a.value, b.value)) }
)

var (
	vRec   = []variant[rec]{{"blq8", SortRec8}, {"pdq8", pdq(pdqsortPRec8)}, {"less", SortLessMethod[rec]}}
	vRec2  = []variant[rec]{{"blq2", SortRec2}, {"pdq2", pdq(pdqsortPRec2)}}
	vRec16 = []variant[rec16]{{"blq16", SortRec16}, {"pdq16", pdq(pdqsortPRec16)}}
	vRec32 = []variant[rec32]{{"blq32", SortRec32}, {"pdq32", pdq(pdqsortPRec32)}}
	vRec64 = []variant[rec64]{{"blq64", SortRec64}, {"pdq64", pdq(pdqsortPRec64)}}
)

func TestSpecialized(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	for _, n := range []int{0, 1, 2, 18, 19, 1025, 5000, 100000, 1 << 20} {
		for _, keys := range []int{1, n/4 + 1, 1 << 30} {
			checkSort(t, genRecs(rng, n, min(keys, 1<<30), mkRec), cmpRec, fRec, vRec)
			r2 := genRecs(rng, n, keys, mkRec)
			for i := range r2 {
				r2[i].value = int32(rng.Intn(100)) // ties on both keys too
			}
			checkSort(t, r2, cmpRec2, cmpRec2, vRec2)
			checkSort(t, genRecs(rng, n, keys, mkRec16), k16, f16, vRec16)
			checkSort(t, genRecs(rng, n, keys, mkRec32), k32, f32, vRec32)
			checkSort(t, genRecs(rng, n, keys, mkRec64), k64, f64, vRec64)
		}
	}
}

func benchVariants[T any](b *testing.B, data []T, c func(a, b T) int, vs []variant[T]) {
	b.Run("std-SortFunc", func(b *testing.B) { benchSortFunc(b, data, slices.SortFunc[[]T], c) })
	for _, v := range vs {
		b.Run(v.name, func(b *testing.B) { benchSort(b, data, v.sort) })
	}
}

func BenchmarkSpecialized(b *testing.B) {
	const n = 1_000_000
	rng := rand.New(rand.NewSource(42))
	b.Run("rec8", func(b *testing.B) {
		vs := append(slices.Clone(vRec), variant[rec]{"blq-SortFunc", func(x []rec) { blqsort.SortFunc(x, cmpRec) }})
		benchVariants(b, genRecs(rng, n, 1<<31, mkRec), cmpRec, vs)
	})
	b.Run("rec8-2key", func(b *testing.B) {
		d := genRecs(rng, n, n/16, mkRec) // ~16 per id, so the second key matters
		for i := range d {
			d[i].value = rng.Int31()
		}
		benchVariants(b, d, cmpRec2, vRec2)
	})
	b.Run("rec16", func(b *testing.B) { benchVariants(b, genRecs(rng, n, 1<<62, mkRec16), k16, vRec16) })
	b.Run("rec32", func(b *testing.B) { benchVariants(b, genRecs(rng, n, 1<<62, mkRec32), k32, vRec32) })
	b.Run("rec64", func(b *testing.B) { benchVariants(b, genRecs(rng, n, 1<<62, mkRec64), k64, vRec64) })
}

// Branchless three-way comparators (cmp.Compare compiles to two branches).
func cmpRecBL(a, b rec) int { return b2i(a.id > b.id) - b2i(a.id < b.id) }

func cmpRec2BL(a, b rec) int {
	return 2*(b2i(a.id > b.id)-b2i(a.id < b.id)) + b2i(a.value > b.value) - b2i(a.value < b.value)
}

func BenchmarkBranchlessCmp(b *testing.B) {
	const n = 1_000_000
	rng := rand.New(rand.NewSource(42))
	d := genRecs(rng, n, 1<<31, mkRec)
	b.Run("rec8/std-SortFunc-BL", func(b *testing.B) { benchSortFunc(b, d, slices.SortFunc[[]rec], cmpRecBL) })
	b.Run("rec8/blq-SortFunc-BL", func(b *testing.B) { benchSortFunc(b, d, blqsort.SortFunc[rec], cmpRecBL) })
	d2 := genRecs(rng, n, n/16, mkRec)
	for i := range d2 {
		d2[i].value = rng.Int31()
	}
	b.Run("rec8-2key/std-SortFunc-BL", func(b *testing.B) { benchSortFunc(b, d2, slices.SortFunc[[]rec], cmpRec2BL) })
	b.Run("rec8-2key/blq2b", func(b *testing.B) { benchSort(b, d2, SortRec2b) })
	b.Run("rec8-2key/pdq2b", func(b *testing.B) { benchSort(b, d2, pdq(pdqsortPRec2b)) })
}

func TestBranchlessCmp(t *testing.T) {
	rng := rand.New(rand.NewSource(12))
	for _, n := range []int{0, 2, 19, 1025, 100000} {
		d := genRecs(rng, n, n/8+1, mkRec)
		for i := range d {
			d[i].value = int32(rng.Intn(50))
		}
		checkSort(t, d, cmpRec2, cmpRec2, []variant[rec]{
			{"blq2b", SortRec2b}, {"pdq2b", pdq(pdqsortPRec2b)},
			{"blqfunc2BL", func(x []rec) { blqsort.SortFunc(x, cmpRec2BL) }},
		})
	}
}
