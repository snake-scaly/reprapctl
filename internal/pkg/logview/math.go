package logview

import "cmp"

func Clamp[T cmp.Ordered](v, min, max T) T {
	switch {
	case min > max:
		panic("logview/math.Clamp: want min <= max")
	case v < min:
		return min
	case v > max:
		return max
	default:
		return v
	}
}

// BinarySearch finds the largest index between 0 and size (inclusive) for which metric is less or equal target.
// Metric must be a monotonically non-decreasing function of index.
func BinarySearch[T cmp.Ordered](size int, target T, metric func(int) T) (i int, m T) {
	first, firstMetric := 0, metric(0)
	last, lastMetric := size, metric(size)

	if target < firstMetric {
		return first, firstMetric
	}
	if lastMetric <= target {
		return last, lastMetric
	}

	for last-first > 1 {
		mid := (first + last) / 2
		midMetric := metric(mid)
		if midMetric <= target {
			first, firstMetric = mid, midMetric
		} else {
			last, lastMetric = mid, midMetric
		}
	}

	return first, firstMetric
}
