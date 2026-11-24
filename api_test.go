package experiments

import (
	"math/rand"
	"slices"
	"testing"

	"github.com/dlm/blqsort"
)

func lessRec(a, b rec) bool     { return a.id < b.id }
func keyRec(r rec) int32        { return r.id }
func keyRec64(r rec64) int64    { return r.id }
func lessRec64(a, b rec64) bool { return a.id < b.id }

func TestAPIVariants(t *testing.T) {
	rng := rand.New(rand.NewSource(13))
	for _, n := range []int{0, 1, 2, 18, 19, 1025, 5000, 100000, 1 << 20} {
		for _, keys := range []int{1, n/4 + 1, 1 << 30} {
			checkSort(t, genRecs(rng, n, keys, mkRec), cmpRec, fRec, []variant[rec]{
				{"less", func(x []rec) { blqsort.SortLess(x, lessRec) }},
				{"bykey", func(x []rec) { blqsort.SortByKey(x, keyRec) }},
				{"bykey-nc", func(x []rec) { SortByKeyNoCache(x, keyRec) }},
			})
			checkSort(t, genRecs(rng, n, keys, mkRec64), k64, f64, []variant[rec64]{
				{"less64", func(x []rec64) { blqsort.SortLess(x, lessRec64) }},
				{"bykey64", func(x []rec64) { blqsort.SortByKey(x, keyRec64) }},
			})
		}
	}
}

func BenchmarkAPI(b *testing.B) {
	const n = 1_000_000
	rng := rand.New(rand.NewSource(42))
	d := genRecs(rng, n, 1<<31, mkRec)
	b.Run("rec8/std-SortFunc", func(b *testing.B) { benchSortFunc(b, d, slices.SortFunc[[]rec], cmpRec) })
	b.Run("rec8/std-SortFunc-BL", func(b *testing.B) { benchSortFunc(b, d, slices.SortFunc[[]rec], cmpRecBL) })
	b.Run("rec8/blq-SortFunc-BL", func(b *testing.B) { benchSortFunc(b, d, blqsort.SortFunc[rec], cmpRecBL) })
	b.Run("rec8/blq-Less", func(b *testing.B) { benchSort(b, d, func(x []rec) { blqsort.SortLess(x, lessRec) }) })
	b.Run("rec8/blq-ByKey-nocache", func(b *testing.B) { benchSort(b, d, func(x []rec) { SortByKeyNoCache(x, keyRec) }) })
	b.Run("rec8/blq-ByKey", func(b *testing.B) { benchSort(b, d, func(x []rec) { blqsort.SortByKey(x, keyRec) }) })
	b.Run("rec8/blq-specialized", func(b *testing.B) { benchSort(b, d, SortRec8) })

	d64 := genRecs(rng, n, 1<<62, mkRec64)
	b.Run("rec64/std-SortFunc", func(b *testing.B) { benchSortFunc(b, d64, slices.SortFunc[[]rec64], k64) })
	b.Run("rec64/blq-ByKey", func(b *testing.B) { benchSort(b, d64, func(x []rec64) { blqsort.SortByKey(x, keyRec64) }) })
	b.Run("rec64/pdq-specialized", func(b *testing.B) { benchSort(b, d64, pdq(pdqsortPRec64)) })
}
