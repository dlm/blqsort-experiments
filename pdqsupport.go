// Copyright 2009 The Go Authors. BSD license, see THIRD_PARTY_LICENSES.md.

// Helpers needed by the generated copies of the stdlib pdqsort
// (copied from $GOROOT/src/slices/sort.go, BSD license).

package experiments

import "math/bits"

type sortedHint int // hint for pdqsort when choosing the pivot

const (
	unknownHint sortedHint = iota
	increasingHint
	decreasingHint
)

// xorshift paper: https://www.jstatsoft.org/article/view/v008i14/xorshift.pdf
type xorshift uint64

func (r *xorshift) Next() uint64 {
	*r ^= *r << 13
	*r ^= *r >> 7
	*r ^= *r << 17
	return uint64(*r)
}

func nextPowerOfTwo(length int) uint {
	return 1 << bits.Len(uint(length))
}
