package alg

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
