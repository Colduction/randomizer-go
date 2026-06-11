package randomizer

import "math/bits"

// Integers combines [SignedIntegers] and [UnsignedIntegers].
type Integers interface {
	SignedIntegers | UnsignedIntegers
}

// SignedIntegers identifies all supported signed integer types.
type SignedIntegers interface {
	~int8 | ~int16 | ~int | ~int32 | ~int64
}

// UnsignedIntegers identifies all supported unsigned integer types.
type UnsignedIntegers interface {
	~uint8 | ~uint16 | ~uint | ~uint32 | ~uint64 | ~uintptr
}

// uniformUint64n returns a uniform random value in [0, n) using Lemire's algorithm.
// The common case requires one multiplication; a rejection loop runs with probability < 1/n.
func uniformUint64n(n uint64, provider Provider) uint64 {
	if n == 0 {
		return 0
	}
	var (
		x      uint64 = provider.Sum64()
		hi, lo uint64 = bits.Mul64(x, n)
	)
	if lo < n {
		threshold := (-n) % n
		for lo < threshold {
			x = provider.Sum64()
			hi, lo = bits.Mul64(x, n)
		}
	}
	return hi
}

// Int returns a random integer constrained by [SignedIntegers].
func Int[T SignedIntegers]() T {
	return T(currentProvider().Sum64())
}

// IntInterval returns a random [SignedIntegers] value in [min, max).
// min and max are swapped automatically if min > max.
func IntInterval[T SignedIntegers](min, max T) T {
	if min == max {
		return min
	}
	if min > max {
		min, max = max, min
	}

	// Map signed values to a monotonic unsigned domain.
	const signMask = uint64(1) << 63
	var (
		minU = uint64(int64(min)) ^ signMask
		maxU = uint64(int64(max)) ^ signMask
		span = maxU - minU
	)

	v := uniformUint64n(span, currentProvider())
	return T(int64((minU + v) ^ signMask))
}

// Uint returns a random integer constrained by [UnsignedIntegers].
func Uint[T UnsignedIntegers]() T {
	return T(currentProvider().Sum64())
}

// UintInterval returns a random [UnsignedIntegers] value in [min, max).
// min and max are swapped automatically if min > max.
func UintInterval[T UnsignedIntegers](min, max T) T {
	if min == max {
		return min
	}
	if min > max {
		min, max = max, min
	}
	v := uniformUint64n(uint64(max-min), currentProvider())
	return min + T(v)
}

// Float32 returns a uniformly distributed random float32 in [0, 1) using the
// active [Provider].
func Float32() float32 {
	const inv24 float32 = float32(1.0 / (1 << 24))
	return float32(currentProvider().Sum32()>>8) * inv24
}

// Float64 returns a uniformly distributed random float64 in [0, 1) using the
// active [Provider].
func Float64() float64 {
	const inv53 float64 = float64(1.0 / (1 << 53))
	return float64(currentProvider().Sum64()>>11) * inv53
}
