// Branchless block partition for pdqsort: a Go port of partition_right_branchless
// from Orson Peters' pdqsort (https://github.com/orlp/pdqsort, zlib license),
// itself derived from "BlockQuicksort: How Branch Mispredictions don't affect
// Quicksort" by Stefan Edelkamp and Armin Weiss. pdqsort is Copyright (c) 2021
// Orson Peters, zlib license; see THIRD_PARTY_LICENSES.md.
//
// It has the interface and contract of the stdlib pdqsort partition
// ($GOROOT/src/slices/zsortordered.go): elements < pivot end up left of
// newpivot, elements >= pivot right of it. Unlike blqsort it does not copy
// every element; blocks of 64 elements from each end are scanned without
// branches, recording the offsets of misplaced elements, which are then
// swapped in pairs.

package experiments

import "cmp"

const blockSize = 64 // offsets fit in a uint8; must be a power of two

func partitionBlock[T cmp.Ordered](data []T, a, b, pivot int) (newpivot int, alreadyPartitioned bool) {
	data[a], data[pivot] = data[pivot], data[a]
	piv := data[a]
	i, j := a+1, b-1 // inclusive bounds of the unpartitioned elements

	for i <= j && data[i] < piv {
		i++
	}
	for i <= j && !(data[j] < piv) {
		j--
	}
	if i > j {
		data[j], data[a] = data[a], data[j]
		return j, true
	}
	data[i], data[j] = data[j], data[i]

	// [first, last) is unpartitioned from here on.
	first, last := i+1, j
	var offL, offR [blockSize]uint8
	baseL, baseR := first, last
	numL, numR, startL, startR := 0, 0, 0, 0

	for first < last {
		// Refill whichever offset block is empty; split the unknown elements
		// between the two if both are.
		unknown := last - first
		leftSplit, rightSplit := 0, 0
		if numL == 0 {
			if numR == 0 {
				leftSplit = unknown / 2
			} else {
				leftSplit = unknown
			}
		}
		if numR == 0 {
			rightSplit = unknown - leftSplit
		}

		// Left block: record offsets of elements >= pivot (wrong side).
		if leftSplit >= blockSize {
			for k := range blockSize {
				offL[numL&(blockSize-1)] = uint8(k)
				numL += b2i(!(data[first] < piv))
				first++
			}
		} else {
			for k := range leftSplit {
				offL[numL&(blockSize-1)] = uint8(k)
				numL += b2i(!(data[first] < piv))
				first++
			}
		}

		// Right block: record offsets (from baseR, counting 1..) of elements < pivot.
		if rightSplit >= blockSize {
			for k := 1; k <= blockSize; k++ {
				last--
				offR[numR&(blockSize-1)] = uint8(k)
				numR += b2i(data[last] < piv)
			}
		} else {
			for k := 1; k <= rightSplit; k++ {
				last--
				offR[numR&(blockSize-1)] = uint8(k)
				numR += b2i(data[last] < piv)
			}
		}

		num := min(numL, numR)
		swapOffsets(data, baseL, baseR, offL[startL:startL+num], offR[startR:startR+num], numL == numR)
		numL -= num
		numR -= num
		startL += num
		startR += num
		if numL == 0 {
			startL = 0
			baseL = first
		}
		if numR == 0 {
			startR = 0
			baseR = last
		}
	}

	// Some misplaced elements of one block remain; swap them to the boundary.
	if numL > 0 {
		for numL > 0 {
			numL--
			last--
			l := baseL + int(offL[startL+numL])
			data[l], data[last] = data[last], data[l]
		}
		first = last
	}
	if numR > 0 {
		for numR > 0 {
			numR--
			r := baseR - int(offR[startR+numR])
			data[r], data[first] = data[first], data[r]
			first++
		}
	}

	newpivot = first - 1
	data[a], data[newpivot] = data[newpivot], piv
	return newpivot, false
}

// swapOffsets exchanges the misplaced elements recorded in offL (relative to
// first) and offR (relative to last, counting back). With equal counts it
// uses plain swaps, which keeps pdqsort O(n) on descending input; otherwise a
// cyclic permutation that needs fewer moves.
func swapOffsets[T any](data []T, first, last int, offL, offR []uint8, useSwaps bool) {
	num := min(len(offL), len(offR))
	if useSwaps {
		for k := range num {
			l, r := first+int(offL[k]), last-int(offR[k])
			data[l], data[r] = data[r], data[l]
		}
		return
	}
	if num == 0 {
		return
	}
	l, r := first+int(offL[0]), last-int(offR[0])
	tmp := data[l]
	data[l] = data[r]
	for k := 1; k < num; k++ {
		l = first + int(offL[k])
		data[r] = data[l]
		r = last - int(offR[k])
		data[l] = data[r]
	}
	data[r] = tmp
}

// PdqSortBranchless is the stdlib pdqsort with partitionBlock. NaNs are
// moved to the front first, like blqsort.Sort (and slices.Sort's ordering).
func PdqSortBranchless[T cmp.Ordered](data []T) {
	data = moveNaNsFirst(data)
	pdqsortPDQBL(data, 0, len(data), depthLimit(len(data)))
}

// PdqSortGo is the same generated pdqsort with the stdlib (branchy) partition,
// as a control: it should match slices.Sort.
func PdqSortGo[T cmp.Ordered](data []T) {
	data = moveNaNsFirst(data)
	pdqsortPDQ(data, 0, len(data), depthLimit(len(data)))
}
