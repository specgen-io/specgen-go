package slices

import (
	"math/bits"

	"github.com/specgen-io/specgen-golang/v2/goven/golang.org/x/exp/constraints"
)

func Sort[E constraints.Ordered](x []E) {
	n := len(x)
	pdqsortOrdered(x, 0, n, bits.Len(uint(n)))
}

func SortFunc[E any](x []E, less func(a, b E) bool) {
	n := len(x)
	pdqsortLessFunc(x, 0, n, bits.Len(uint(n)), less)
}

func SortStableFunc[E any](x []E, less func(a, b E) bool) {
	stableLessFunc(x, len(x), less)
}

func IsSorted[E constraints.Ordered](x []E) bool {
	for i := len(x) - 1; i > 0; i-- {
		if x[i] < x[i-1] {
			return false
		}
	}
	return true
}

func IsSortedFunc[E any](x []E, less func(a, b E) bool) bool {
	for i := len(x) - 1; i > 0; i-- {
		if less(x[i], x[i-1]) {
			return false
		}
	}
	return true
}

func BinarySearch[E constraints.Ordered](x []E, target E) (int, bool) {

	pos := search(len(x), func(i int) bool { return x[i] >= target })
	if pos >= len(x) || x[pos] != target {
		return pos, false
	} else {
		return pos, true
	}
}

func BinarySearchFunc[E any](x []E, target E, cmp func(E, E) int) (int, bool) {
	pos := search(len(x), func(i int) bool { return cmp(x[i], target) >= 0 })
	if pos >= len(x) || cmp(x[pos], target) != 0 {
		return pos, false
	} else {
		return pos, true
	}
}

func search(n int, f func(int) bool) int {

	i, j := 0, n
	for i < j {
		h := int(uint(i+j) >> 1)

		if !f(h) {
			i = h + 1
		} else {
			j = h
		}
	}

	return i
}

type sortedHint int

const (
	unknownHint	sortedHint	= iota
	increasingHint
	decreasingHint
)

type xorshift uint64

func (r *xorshift) Next() uint64 {
	*r ^= *r << 13
	*r ^= *r >> 17
	*r ^= *r << 5
	return uint64(*r)
}

func nextPowerOfTwo(length int) uint {
	return 1 << bits.Len(uint(length))
}
