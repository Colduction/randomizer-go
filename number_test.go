package randomizer_test

import (
	"testing"

	"github.com/colduction/randomizer-go"
)

var (
	benchInt64  int64
	benchUint64 uint64
	benchF32    float32
	benchF64    float64
)

func TestNumberFloatRanges(t *testing.T) {
	for range 10000 {
		f32 := randomizer.Float32()
		if !(f32 >= 0 && f32 < 1) {
			t.Fatalf("Float32 out of range [0,1): %v", f32)
		}
		f64 := randomizer.Float64()
		if !(f64 >= 0 && f64 < 1) {
			t.Fatalf("Float64 out of range [0,1): %v", f64)
		}
	}
}

func TestNumberIntIntervalRange(t *testing.T) {
	min, max := int64(-1000), int64(1000)
	for range 10000 {
		v := randomizer.IntInterval(min, max)
		if v < min || v >= max {
			t.Fatalf("IntInterval out of range [%d,%d): %d", min, max, v)
		}
	}
}

func TestNumberIntIntervalSwappedBounds(t *testing.T) {
	min, max := int64(1000), int64(-1000)
	for range 10000 {
		v := randomizer.IntInterval(min, max)
		if v < max || v >= min {
			t.Fatalf("IntInterval with swapped bounds out of range [%d,%d): %d", max, min, v)
		}
	}
}

func TestNumberIntIntervalEqualBounds(t *testing.T) {
	if got := randomizer.IntInterval(int64(42), int64(42)); got != 42 {
		t.Fatalf("IntInterval equal bounds = %d, want 42", got)
	}
}

func TestNumberUintIntervalRange(t *testing.T) {
	min, max := uint64(10), uint64(100000)
	for range 10000 {
		v := randomizer.UintInterval(min, max)
		if v < min || v >= max {
			t.Fatalf("UintInterval out of range [%d,%d): %d", min, max, v)
		}
	}
}

func TestNumberUintIntervalSwappedBounds(t *testing.T) {
	min, max := uint64(100000), uint64(10)
	for range 10000 {
		v := randomizer.UintInterval(min, max)
		if v < max || v >= min {
			t.Fatalf("UintInterval with swapped bounds out of range [%d,%d): %d", max, min, v)
		}
	}
}

func TestNumberUintIntervalEqualBounds(t *testing.T) {
	if got := randomizer.UintInterval(uint64(7), uint64(7)); got != 7 {
		t.Fatalf("UintInterval equal bounds = %d, want 7", got)
	}
}

func BenchmarkNumberInt(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchInt64 = randomizer.Int[int64]()
	}
}

func BenchmarkNumberIntInterval(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchInt64 = randomizer.IntInterval(int64(-100000), int64(100000))
	}
}

func BenchmarkNumberUint(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchUint64 = randomizer.Uint[uint64]()
	}
}

func BenchmarkNumberUintInterval(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchUint64 = randomizer.UintInterval(uint64(100), uint64(1000000))
	}
}

func BenchmarkNumberFloat32(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchF32 = randomizer.Float32()
	}
}

func BenchmarkNumberFloat64(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchF64 = randomizer.Float64()
	}
}
