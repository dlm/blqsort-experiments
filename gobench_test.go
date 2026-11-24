package experiments

import (
	"slices"
	"strconv"
	"testing"

	"github.com/dlm/blqsort"
)

// Equivalents of the stdlib sort benchmarks ($GOROOT/src/sort/sort_test.go),
// run through slices.Sort and blqsort.Sort.
func BenchmarkGoStdlib(b *testing.B) {
	ints := map[string]func(i, n int) int{
		"SortInt1K":          func(i, n int) int { return i ^ 0x2cc },
		"SortInt1K_Sorted":   func(i, n int) int { return i },
		"SortInt1K_Reversed": func(i, n int) int { return n - i },
		"SortInt1K_Mod8":     func(i, n int) int { return i % 8 },
	}
	for _, name := range slices.Sorted(func(yield func(string) bool) {
		for k := range ints {
			if !yield(k) {
				return
			}
		}
	}) {
		d := make([]int, 1<<10)
		for i := range d {
			d[i] = ints[name](i, len(d))
		}
		b.Run(name+"/std", func(b *testing.B) { benchSort(b, d, slices.Sort[[]int]) })
		b.Run(name+"/blq", func(b *testing.B) { benchSort(b, d, blqsort.Sort[int]) })
		b.Run(name+"/pdqbl", func(b *testing.B) { benchSort(b, d, PdqSortBranchless[int]) })
	}
	d64 := make([]int, 1<<16)
	for i := range d64 {
		d64[i] = i ^ 0xcccc
	}
	b.Run("SortInt64K/std", func(b *testing.B) { benchSort(b, d64, slices.Sort[[]int]) })
	b.Run("SortInt64K/blq", func(b *testing.B) { benchSort(b, d64, blqsort.Sort[int]) })
	b.Run("SortInt64K/pdqbl", func(b *testing.B) { benchSort(b, d64, PdqSortBranchless[int]) })
	s := make([]string, 1<<10)
	for i := range s {
		s[i] = strconv.Itoa(i ^ 0x2cc)
	}
	b.Run("SortString1K/std", func(b *testing.B) { benchSort(b, s, slices.Sort[[]string]) })
	b.Run("SortString1K/blq", func(b *testing.B) { benchSort(b, s, blqsort.Sort[string]) })
	b.Run("SortString1K/pdqbl", func(b *testing.B) { benchSort(b, s, PdqSortBranchless[string]) })
}
