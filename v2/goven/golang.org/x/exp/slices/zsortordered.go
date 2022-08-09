package slices

import "github.com/specgen-io/specgen-golang/v2/goven/golang.org/x/exp/constraints"

func insertionSortOrdered[E constraints.Ordered](data []E, a, b int) {
	for i := a + 1; i < b; i++ {
		for j := i; j > a && (data[j] < data[j-1]); j-- {
			data[j], data[j-1] = data[j-1], data[j]
		}
	}
}

func siftDownOrdered[E constraints.Ordered](data []E, lo, hi, first int) {
	root := lo
	for {
		child := 2*root + 1
		if child >= hi {
			break
		}
		if child+1 < hi && (data[first+child] < data[first+child+1]) {
			child++
		}
		if !(data[first+root] < data[first+child]) {
			return
		}
		data[first+root], data[first+child] = data[first+child], data[first+root]
		root = child
	}
}

func heapSortOrdered[E constraints.Ordered](data []E, a, b int) {
	first := a
	lo := 0
	hi := b - a

	for i := (hi - 1) / 2; i >= 0; i-- {
		siftDownOrdered(data, i, hi, first)
	}

	for i := hi - 1; i >= 0; i-- {
		data[first], data[first+i] = data[first+i], data[first]
		siftDownOrdered(data, lo, i, first)
	}
}

func pdqsortOrdered[E constraints.Ordered](data []E, a, b, limit int) {
	const maxInsertion = 12

	var (
		wasBalanced	= true
		wasPartitioned	= true
	)

	for {
		length := b - a

		if length <= maxInsertion {
			insertionSortOrdered(data, a, b)
			return
		}

		if limit == 0 {
			heapSortOrdered(data, a, b)
			return
		}

		if !wasBalanced {
			breakPatternsOrdered(data, a, b)
			limit--
		}

		pivot, hint := choosePivotOrdered(data, a, b)
		if hint == decreasingHint {
			reverseRangeOrdered(data, a, b)

			pivot = (b - 1) - (pivot - a)
			hint = increasingHint
		}

		if wasBalanced && wasPartitioned && hint == increasingHint {
			if partialInsertionSortOrdered(data, a, b) {
				return
			}
		}

		if a > 0 && !(data[a-1] < data[pivot]) {
			mid := partitionEqualOrdered(data, a, b, pivot)
			a = mid
			continue
		}

		mid, alreadyPartitioned := partitionOrdered(data, a, b, pivot)
		wasPartitioned = alreadyPartitioned

		leftLen, rightLen := mid-a, b-mid
		balanceThreshold := length / 8
		if leftLen < rightLen {
			wasBalanced = leftLen >= balanceThreshold
			pdqsortOrdered(data, a, mid, limit)
			a = mid + 1
		} else {
			wasBalanced = rightLen >= balanceThreshold
			pdqsortOrdered(data, mid+1, b, limit)
			b = mid
		}
	}
}

func partitionOrdered[E constraints.Ordered](data []E, a, b, pivot int) (newpivot int, alreadyPartitioned bool) {
	data[a], data[pivot] = data[pivot], data[a]
	i, j := a+1, b-1

	for i <= j && (data[i] < data[a]) {
		i++
	}
	for i <= j && !(data[j] < data[a]) {
		j--
	}
	if i > j {
		data[j], data[a] = data[a], data[j]
		return j, true
	}
	data[i], data[j] = data[j], data[i]
	i++
	j--

	for {
		for i <= j && (data[i] < data[a]) {
			i++
		}
		for i <= j && !(data[j] < data[a]) {
			j--
		}
		if i > j {
			break
		}
		data[i], data[j] = data[j], data[i]
		i++
		j--
	}
	data[j], data[a] = data[a], data[j]
	return j, false
}

func partitionEqualOrdered[E constraints.Ordered](data []E, a, b, pivot int) (newpivot int) {
	data[a], data[pivot] = data[pivot], data[a]
	i, j := a+1, b-1

	for {
		for i <= j && !(data[a] < data[i]) {
			i++
		}
		for i <= j && (data[a] < data[j]) {
			j--
		}
		if i > j {
			break
		}
		data[i], data[j] = data[j], data[i]
		i++
		j--
	}
	return i
}

func partialInsertionSortOrdered[E constraints.Ordered](data []E, a, b int) bool {
	const (
		maxSteps		= 5
		shortestShifting	= 50
	)
	i := a + 1
	for j := 0; j < maxSteps; j++ {
		for i < b && !(data[i] < data[i-1]) {
			i++
		}

		if i == b {
			return true
		}

		if b-a < shortestShifting {
			return false
		}

		data[i], data[i-1] = data[i-1], data[i]

		if i-a >= 2 {
			for j := i - 1; j >= 1; j-- {
				if !(data[j] < data[j-1]) {
					break
				}
				data[j], data[j-1] = data[j-1], data[j]
			}
		}

		if b-i >= 2 {
			for j := i + 1; j < b; j++ {
				if !(data[j] < data[j-1]) {
					break
				}
				data[j], data[j-1] = data[j-1], data[j]
			}
		}
	}
	return false
}

func breakPatternsOrdered[E constraints.Ordered](data []E, a, b int) {
	length := b - a
	if length >= 8 {
		random := xorshift(length)
		modulus := nextPowerOfTwo(length)

		for idx := a + (length/4)*2 - 1; idx <= a+(length/4)*2+1; idx++ {
			other := int(uint(random.Next()) & (modulus - 1))
			if other >= length {
				other -= length
			}
			data[idx], data[a+other] = data[a+other], data[idx]
		}
	}
}

func choosePivotOrdered[E constraints.Ordered](data []E, a, b int) (pivot int, hint sortedHint) {
	const (
		shortestNinther	= 50
		maxSwaps	= 4 * 3
	)

	l := b - a

	var (
		swaps	int
		i	= a + l/4*1
		j	= a + l/4*2
		k	= a + l/4*3
	)

	if l >= 8 {
		if l >= shortestNinther {

			i = medianAdjacentOrdered(data, i, &swaps)
			j = medianAdjacentOrdered(data, j, &swaps)
			k = medianAdjacentOrdered(data, k, &swaps)
		}

		j = medianOrdered(data, i, j, k, &swaps)
	}

	switch swaps {
	case 0:
		return j, increasingHint
	case maxSwaps:
		return j, decreasingHint
	default:
		return j, unknownHint
	}
}

func order2Ordered[E constraints.Ordered](data []E, a, b int, swaps *int) (int, int) {
	if data[b] < data[a] {
		*swaps++
		return b, a
	}
	return a, b
}

func medianOrdered[E constraints.Ordered](data []E, a, b, c int, swaps *int) int {
	a, b = order2Ordered(data, a, b, swaps)
	b, c = order2Ordered(data, b, c, swaps)
	a, b = order2Ordered(data, a, b, swaps)
	return b
}

func medianAdjacentOrdered[E constraints.Ordered](data []E, a int, swaps *int) int {
	return medianOrdered(data, a-1, a, a+1, swaps)
}

func reverseRangeOrdered[E constraints.Ordered](data []E, a, b int) {
	i := a
	j := b - 1
	for i < j {
		data[i], data[j] = data[j], data[i]
		i++
		j--
	}
}

func swapRangeOrdered[E constraints.Ordered](data []E, a, b, n int) {
	for i := 0; i < n; i++ {
		data[a+i], data[b+i] = data[b+i], data[a+i]
	}
}

func stableOrdered[E constraints.Ordered](data []E, n int) {
	blockSize := 20
	a, b := 0, blockSize
	for b <= n {
		insertionSortOrdered(data, a, b)
		a = b
		b += blockSize
	}
	insertionSortOrdered(data, a, n)

	for blockSize < n {
		a, b = 0, 2*blockSize
		for b <= n {
			symMergeOrdered(data, a, a+blockSize, b)
			a = b
			b += 2 * blockSize
		}
		if m := a + blockSize; m < n {
			symMergeOrdered(data, a, m, n)
		}
		blockSize *= 2
	}
}

func symMergeOrdered[E constraints.Ordered](data []E, a, m, b int) {

	if m-a == 1 {

		i := m
		j := b
		for i < j {
			h := int(uint(i+j) >> 1)
			if data[h] < data[a] {
				i = h + 1
			} else {
				j = h
			}
		}

		for k := a; k < i-1; k++ {
			data[k], data[k+1] = data[k+1], data[k]
		}
		return
	}

	if b-m == 1 {

		i := a
		j := m
		for i < j {
			h := int(uint(i+j) >> 1)
			if !(data[m] < data[h]) {
				i = h + 1
			} else {
				j = h
			}
		}

		for k := m; k > i; k-- {
			data[k], data[k-1] = data[k-1], data[k]
		}
		return
	}

	mid := int(uint(a+b) >> 1)
	n := mid + m
	var start, r int
	if m > mid {
		start = n - b
		r = mid
	} else {
		start = a
		r = m
	}
	p := n - 1

	for start < r {
		c := int(uint(start+r) >> 1)
		if !(data[p-c] < data[c]) {
			start = c + 1
		} else {
			r = c
		}
	}

	end := n - start
	if start < m && m < end {
		rotateOrdered(data, start, m, end)
	}
	if a < start && start < mid {
		symMergeOrdered(data, a, start, mid)
	}
	if mid < end && end < b {
		symMergeOrdered(data, mid, end, b)
	}
}

func rotateOrdered[E constraints.Ordered](data []E, a, m, b int) {
	i := m - a
	j := b - m

	for i != j {
		if i > j {
			swapRangeOrdered(data, m-i, m, j)
			i -= j
		} else {
			swapRangeOrdered(data, m-i, m+j-i, i)
			j -= i
		}
	}

	swapRangeOrdered(data, m-i, m, i)
}
