package slices

func insertionSortLessFunc[E any](data []E, a, b int, less func(a, b E) bool) {
	for i := a + 1; i < b; i++ {
		for j := i; j > a && less(data[j], data[j-1]); j-- {
			data[j], data[j-1] = data[j-1], data[j]
		}
	}
}

func siftDownLessFunc[E any](data []E, lo, hi, first int, less func(a, b E) bool) {
	root := lo
	for {
		child := 2*root + 1
		if child >= hi {
			break
		}
		if child+1 < hi && less(data[first+child], data[first+child+1]) {
			child++
		}
		if !less(data[first+root], data[first+child]) {
			return
		}
		data[first+root], data[first+child] = data[first+child], data[first+root]
		root = child
	}
}

func heapSortLessFunc[E any](data []E, a, b int, less func(a, b E) bool) {
	first := a
	lo := 0
	hi := b - a

	for i := (hi - 1) / 2; i >= 0; i-- {
		siftDownLessFunc(data, i, hi, first, less)
	}

	for i := hi - 1; i >= 0; i-- {
		data[first], data[first+i] = data[first+i], data[first]
		siftDownLessFunc(data, lo, i, first, less)
	}
}

func pdqsortLessFunc[E any](data []E, a, b, limit int, less func(a, b E) bool) {
	const maxInsertion = 12

	var (
		wasBalanced	= true
		wasPartitioned	= true
	)

	for {
		length := b - a

		if length <= maxInsertion {
			insertionSortLessFunc(data, a, b, less)
			return
		}

		if limit == 0 {
			heapSortLessFunc(data, a, b, less)
			return
		}

		if !wasBalanced {
			breakPatternsLessFunc(data, a, b, less)
			limit--
		}

		pivot, hint := choosePivotLessFunc(data, a, b, less)
		if hint == decreasingHint {
			reverseRangeLessFunc(data, a, b, less)

			pivot = (b - 1) - (pivot - a)
			hint = increasingHint
		}

		if wasBalanced && wasPartitioned && hint == increasingHint {
			if partialInsertionSortLessFunc(data, a, b, less) {
				return
			}
		}

		if a > 0 && !less(data[a-1], data[pivot]) {
			mid := partitionEqualLessFunc(data, a, b, pivot, less)
			a = mid
			continue
		}

		mid, alreadyPartitioned := partitionLessFunc(data, a, b, pivot, less)
		wasPartitioned = alreadyPartitioned

		leftLen, rightLen := mid-a, b-mid
		balanceThreshold := length / 8
		if leftLen < rightLen {
			wasBalanced = leftLen >= balanceThreshold
			pdqsortLessFunc(data, a, mid, limit, less)
			a = mid + 1
		} else {
			wasBalanced = rightLen >= balanceThreshold
			pdqsortLessFunc(data, mid+1, b, limit, less)
			b = mid
		}
	}
}

func partitionLessFunc[E any](data []E, a, b, pivot int, less func(a, b E) bool) (newpivot int, alreadyPartitioned bool) {
	data[a], data[pivot] = data[pivot], data[a]
	i, j := a+1, b-1

	for i <= j && less(data[i], data[a]) {
		i++
	}
	for i <= j && !less(data[j], data[a]) {
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
		for i <= j && less(data[i], data[a]) {
			i++
		}
		for i <= j && !less(data[j], data[a]) {
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

func partitionEqualLessFunc[E any](data []E, a, b, pivot int, less func(a, b E) bool) (newpivot int) {
	data[a], data[pivot] = data[pivot], data[a]
	i, j := a+1, b-1

	for {
		for i <= j && !less(data[a], data[i]) {
			i++
		}
		for i <= j && less(data[a], data[j]) {
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

func partialInsertionSortLessFunc[E any](data []E, a, b int, less func(a, b E) bool) bool {
	const (
		maxSteps		= 5
		shortestShifting	= 50
	)
	i := a + 1
	for j := 0; j < maxSteps; j++ {
		for i < b && !less(data[i], data[i-1]) {
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
				if !less(data[j], data[j-1]) {
					break
				}
				data[j], data[j-1] = data[j-1], data[j]
			}
		}

		if b-i >= 2 {
			for j := i + 1; j < b; j++ {
				if !less(data[j], data[j-1]) {
					break
				}
				data[j], data[j-1] = data[j-1], data[j]
			}
		}
	}
	return false
}

func breakPatternsLessFunc[E any](data []E, a, b int, less func(a, b E) bool) {
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

func choosePivotLessFunc[E any](data []E, a, b int, less func(a, b E) bool) (pivot int, hint sortedHint) {
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

			i = medianAdjacentLessFunc(data, i, &swaps, less)
			j = medianAdjacentLessFunc(data, j, &swaps, less)
			k = medianAdjacentLessFunc(data, k, &swaps, less)
		}

		j = medianLessFunc(data, i, j, k, &swaps, less)
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

func order2LessFunc[E any](data []E, a, b int, swaps *int, less func(a, b E) bool) (int, int) {
	if less(data[b], data[a]) {
		*swaps++
		return b, a
	}
	return a, b
}

func medianLessFunc[E any](data []E, a, b, c int, swaps *int, less func(a, b E) bool) int {
	a, b = order2LessFunc(data, a, b, swaps, less)
	b, c = order2LessFunc(data, b, c, swaps, less)
	a, b = order2LessFunc(data, a, b, swaps, less)
	return b
}

func medianAdjacentLessFunc[E any](data []E, a int, swaps *int, less func(a, b E) bool) int {
	return medianLessFunc(data, a-1, a, a+1, swaps, less)
}

func reverseRangeLessFunc[E any](data []E, a, b int, less func(a, b E) bool) {
	i := a
	j := b - 1
	for i < j {
		data[i], data[j] = data[j], data[i]
		i++
		j--
	}
}

func swapRangeLessFunc[E any](data []E, a, b, n int, less func(a, b E) bool) {
	for i := 0; i < n; i++ {
		data[a+i], data[b+i] = data[b+i], data[a+i]
	}
}

func stableLessFunc[E any](data []E, n int, less func(a, b E) bool) {
	blockSize := 20
	a, b := 0, blockSize
	for b <= n {
		insertionSortLessFunc(data, a, b, less)
		a = b
		b += blockSize
	}
	insertionSortLessFunc(data, a, n, less)

	for blockSize < n {
		a, b = 0, 2*blockSize
		for b <= n {
			symMergeLessFunc(data, a, a+blockSize, b, less)
			a = b
			b += 2 * blockSize
		}
		if m := a + blockSize; m < n {
			symMergeLessFunc(data, a, m, n, less)
		}
		blockSize *= 2
	}
}

func symMergeLessFunc[E any](data []E, a, m, b int, less func(a, b E) bool) {

	if m-a == 1 {

		i := m
		j := b
		for i < j {
			h := int(uint(i+j) >> 1)
			if less(data[h], data[a]) {
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
			if !less(data[m], data[h]) {
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
		if !less(data[p-c], data[c]) {
			start = c + 1
		} else {
			r = c
		}
	}

	end := n - start
	if start < m && m < end {
		rotateLessFunc(data, start, m, end, less)
	}
	if a < start && start < mid {
		symMergeLessFunc(data, a, start, mid, less)
	}
	if mid < end && end < b {
		symMergeLessFunc(data, mid, end, b, less)
	}
}

func rotateLessFunc[E any](data []E, a, m, b int, less func(a, b E) bool) {
	i := m - a
	j := b - m

	for i != j {
		if i > j {
			swapRangeLessFunc(data, m-i, m, j, less)
			i -= j
		} else {
			swapRangeLessFunc(data, m-i, m+j-i, i, less)
			j -= i
		}
	}

	swapRangeLessFunc(data, m-i, m, i, less)
}
