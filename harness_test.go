package experiments

import (
	"cmp"
	"slices"
	"testing"
)

func cmpRec(a, b rec) int { return cmp.Compare(a.id, b.id) }

func nanEqual[T cmp.Ordered](a, b []T) bool {
	return slices.EqualFunc(a, b, func(x, y T) bool { return cmp.Compare(x, y) == 0 })
}

// benchSort times sortFn on fresh copies of master.
func benchSort[T any](b *testing.B, master []T, sortFn func([]T)) {
	data := make([]T, len(master))
	for b.Loop() {
		b.StopTimer()
		copy(data, master)
		b.StartTimer()
		sortFn(data)
	}
}

func benchSortFunc[T any](b *testing.B, master []T, sortFn func([]T, func(a, b T) int), c func(a, b T) int) {
	benchSort(b, master, func(x []T) { sortFn(x, c) })
}
