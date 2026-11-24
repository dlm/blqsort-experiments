package experiments

import (
	"fmt"
	"math/rand"
	"slices"
	"testing"

	"github.com/dlm/blqsort"
)

func randStrings(rng *rand.Rand, n, maxLen int, prefix string) []string {
	s := make([]string, n)
	b := make([]byte, maxLen)
	for i := range s {
		l := 1 + rng.Intn(maxLen)
		for j := range l {
			b[j] = byte('a' + rng.Intn(26))
		}
		s[i] = prefix + string(b[:l])
	}
	return s
}

func TestStrings(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	for _, n := range []int{0, 1, 19, 1025, 5000, 100000} {
		d := randStrings(rng, n, 4, "")
		want := slices.Clone(d)
		slices.Sort(want)
		blqsort.Sort(d)
		if !slices.Equal(d, want) {
			t.Fatalf("n=%d: strings not sorted", n)
		}
	}
}

func BenchmarkStrings(b *testing.B) {
	for _, n := range []int{1000, 1_000_000} {
		for _, pre := range []string{"", "common/prefix/"} {
			d := randStrings(rand.New(rand.NewSource(42)), n, 12, pre)
			name := fmt.Sprintf("n=%d/prefix=%v", n, pre != "")
			b.Run(name+"/std", func(b *testing.B) { benchSort(b, d, slices.Sort[[]string]) })
			b.Run(name+"/blq", func(b *testing.B) { benchSort(b, d, blqsort.Sort[string]) })
		}
	}
}
