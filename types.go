// Package experiments holds the measurements behind blqsort: pattern
// matrices, comparisons with other sorts and libraries, API variants and
// type specializations. See README.md for how to run them.
package experiments

//go:generate go run ./gen

// Record types for the specialization experiments: sort key id, the rest is
// ballast that has to move (rec has a second key, value).
type rec struct {
	id, value int32
}
type rec16 struct {
	id  int64
	pad [1]int64
}
type rec32 struct {
	id  int64
	pad [3]int64
}
type rec64 struct {
	id  int64
	pad [7]int64
}

// Less is used through a generic method constraint (SortLessMethod).
func (a rec) Less(b rec) bool { return a.id < b.id }
