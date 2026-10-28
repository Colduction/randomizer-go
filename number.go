package randomizer

import (
	"math/bits"
	"math/rand/v2"
	"unsafe"
)

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

const (
	signMask uint64 = 1 << 63

	inv24 float32 = 1.0 / (1 << 24)
	inv53 float64 = 1.0 / (1 << 53)
)

func boundedFrom(x, n uint64, p provider) uint64 {
	if hi, lo := bits.Mul64(x, n); lo >= n {
		return hi
	}
	return boundedSlow(x, n, p)
}

//go:noinline
func boundedSlow(x, n uint64, p provider) uint64 {
	threshold := -n % n
	hi, lo := bits.Mul64(x, n)
	for lo < threshold {
		hi, lo = bits.Mul64(p.Uint64(), n)
	}
	return hi
}

func uniformUint64n(n uint64, p provider) uint64 {
	if n <= 1 {
		return 0
	}
	return boundedFrom(p.Uint64(), n, p)
}

// Int returns a random integer constrained by [SignedIntegers].
func Int[T SignedIntegers]() T {
	return T(rand.Uint64())
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
	var (
		minU = uint64(int64(min)) ^ signMask
		maxU = uint64(int64(max)) ^ signMask
	)
	v := boundedFrom(rand.Uint64(), maxU-minU, fastProvider{})
	return T(int64((minU + v) ^ signMask))
}

// Uint returns a random integer constrained by [UnsignedIntegers].
func Uint[T UnsignedIntegers]() T {
	return T(rand.Uint64())
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
	return min + T(boundedFrom(rand.Uint64(), uint64(max-min), fastProvider{}))
}

// Float32 returns a uniformly distributed random float32 on the 24-bit grid in [0, 1).
func Float32() float32 {
	return float32(rand.Uint64()>>40) * inv24
}

// Float64 returns a uniformly distributed random float64 on the 53-bit grid in [0, 1).
func Float64() float64 {
	return float64(rand.Uint64()>>11) * inv53
}

// Int returns a uniformly distributed signed integer using the bound source.
func (g *Generator) Int[T SignedIntegers]() T {
	return T(g.source().Uint64())
}

// Uint returns a uniformly distributed unsigned integer using the bound source.
func (g *Generator) Uint[T UnsignedIntegers]() T {
	return T(g.source().Uint64())
}

// IntInterval returns a value in [min, max), swapping reversed bounds.
// Equal bounds return that value without consuming randomness.
func (g *Generator) IntInterval[T SignedIntegers](min, max T) T {
	if min == max {
		return min
	}
	if min > max {
		min, max = max, min
	}
	s := openProvider(g.source())
	defer s.close()
	return T(uint64(int64(min)) + uniformUint64n(uint64(int64(max))-uint64(int64(min)), s.provider))
}

// UintInterval returns a value in [min, max), swapping reversed bounds.
// Equal bounds return that value without consuming randomness.
func (g *Generator) UintInterval[T UnsignedIntegers](min, max T) T {
	if min == max {
		return min
	}
	if min > max {
		min, max = max, min
	}
	s := openProvider(g.source())
	defer s.close()
	return min + T(uniformUint64n(uint64(max-min), s.provider))
}

// Float32 returns a value on the uniform 24-bit grid in [0, 1).
func (g *Generator) Float32() float32 {
	return float32(g.source().Uint32()>>8) * inv24
}

// Float64 returns a value on the uniform 53-bit grid in [0, 1).
func (g *Generator) Float64() float64 {
	return float64(g.source().Uint64()>>11) * inv53
}

// FillInt fills caller-owned storage with uniformly distributed signed integers.
func FillInt[T SignedIntegers](dst []T) { (*Generator)(nil).FillInt(dst) }

// FillUint fills caller-owned storage with uniformly distributed unsigned integers.
func FillUint[T UnsignedIntegers](dst []T) { (*Generator)(nil).FillUint(dst) }

// FillIntInterval fills caller-owned storage with values in [min, max).
// Reversed bounds are swapped; equal bounds fill a constant without randomness.
func FillIntInterval[T SignedIntegers](dst []T, min, max T) {
	(*Generator)(nil).FillIntInterval(dst, min, max)
}

// FillUintInterval fills caller-owned storage with values in [min, max).
// Reversed bounds are swapped; equal bounds fill a constant without randomness.
func FillUintInterval[T UnsignedIntegers](dst []T, min, max T) {
	(*Generator)(nil).FillUintInterval(dst, min, max)
}

// FillFloat32 fills caller-owned storage with values on the uniform 24-bit grid in [0, 1).
func FillFloat32(dst []float32) { (*Generator)(nil).FillFloat32(dst) }

// FillFloat64 fills caller-owned storage with values on the uniform 53-bit grid in [0, 1).
func FillFloat64(dst []float64) { (*Generator)(nil).FillFloat64(dst) }

// FillInt fills caller-owned storage with signed integers from the bound source.
// It allocates no heap storage and holds built-in source locks once for the batch.
func (g *Generator) FillInt[T SignedIntegers](dst []T) {
	s := openProvider(g.source())
	defer s.close()
	fillIntegers(dst, s.provider)
}

// FillUint fills caller-owned storage with unsigned integers from the bound source.
// It allocates no heap storage and holds built-in source locks once for the batch.
func (g *Generator) FillUint[T UnsignedIntegers](dst []T) {
	s := openProvider(g.source())
	defer s.close()
	fillIntegers(dst, s.provider)
}

// FillIntInterval fills caller-owned storage with values in [min, max).
// Reversed bounds are swapped. Equal bounds and empty destinations consume no randomness.
// A source failure panics and may leave partial output.
func (g *Generator) FillIntInterval[T SignedIntegers](dst []T, min, max T) {
	if min > max {
		min, max = max, min
	}
	if min == max {
		for i := range dst {
			dst[i] = min
		}
		return
	}
	s := openProvider(g.source())
	defer s.close()
	fillBounded(dst, uint64(int64(min)), uint64(int64(max))-uint64(int64(min)), s.provider)
}

// FillUintInterval fills caller-owned storage with values in [min, max).
// Reversed bounds are swapped. Equal bounds and empty destinations consume no randomness.
// A source failure panics and may leave partial output.
func (g *Generator) FillUintInterval[T UnsignedIntegers](dst []T, min, max T) {
	if min > max {
		min, max = max, min
	}
	if min == max {
		for i := range dst {
			dst[i] = min
		}
		return
	}
	s := openProvider(g.source())
	defer s.close()
	fillBounded(dst, uint64(min), uint64(max-min), s.provider)
}

// FillFloat32 fills caller-owned storage with values on the uniform 24-bit grid in [0, 1).
func (g *Generator) FillFloat32(dst []float32) {
	s := openProvider(g.source())
	defer s.close()
	if _, ok := s.provider.(cryptoProvider); ok {
		fillSecureFloats(dst)
		return
	}
	if _, ok := s.provider.(fastProvider); ok {
		for i := range dst {
			dst[i] = float32(rand.Uint64()>>40) * inv24
		}
		return
	}
	for i := range dst {
		dst[i] = float32(s.Uint32()>>8) * inv24
	}
}

// FillFloat64 fills caller-owned storage with values on the uniform 53-bit grid in [0, 1).
func (g *Generator) FillFloat64(dst []float64) {
	s := openProvider(g.source())
	defer s.close()
	if _, ok := s.provider.(cryptoProvider); ok {
		fillSecureFloats(dst)
		return
	}
	if _, ok := s.provider.(fastProvider); ok {
		for i := range dst {
			dst[i] = float64(rand.Uint64()>>11) * inv53
		}
		return
	}
	for i := range dst {
		dst[i] = float64(s.Uint64()>>11) * inv53
	}
}

func fillIntegers[T Integers](dst []T, p provider) {
	if _, ok := p.(cryptoProvider); ok {
		fillSecureIntegers(dst)
		return
	}
	var value T
	width := uint(unsafe.Sizeof(value) * 8)
	if width < 64 {
		_, fast := p.(fastProvider)
		for i := 0; i < len(dst); {
			var x uint64
			if fast {
				x = rand.Uint64()
			} else {
				x = p.Uint64()
			}
			for range min(int(64/width), len(dst)-i) {
				dst[i] = T(x)
				x >>= width
				i++
			}
		}
		return
	}
	if _, ok := p.(fastProvider); ok {
		for i := range dst {
			dst[i] = T(rand.Uint64())
		}
		return
	}
	for i := range dst {
		dst[i] = T(p.Uint64())
	}
}

func fillBounded[T Integers](dst []T, base, span uint64, p provider) {
	if span <= 1 {
		for i := range dst {
			dst[i] = T(base)
		}
		return
	}
	if span&(span-1) == 0 {
		width := bits.TrailingZeros64(span)
		for i := 0; i < len(dst); {
			x := p.Uint64()
			for range min(64/width, len(dst)-i) {
				dst[i] = T(base + (x & (span - 1)))
				x >>= width
				i++
			}
		}
		return
	}
	spec := newBatchSpec(span)
	fullThreshold := -spec.prod % spec.prod
	for i := 0; i < len(dst); {
		take := min(int(spec.k), len(dst)-i)
		threshold := fullThreshold
		if take != int(spec.k) {
			prod := uint64(1)
			for range take {
				prod *= span
			}
			threshold = -prod % prod
		}
		for {
			x := p.Uint64()
			for j := range take {
				var hi uint64
				hi, x = bits.Mul64(x, span)
				dst[i+j] = T(base + hi)
			}
			if x >= threshold {
				break
			}
		}
		i += take
	}
}
