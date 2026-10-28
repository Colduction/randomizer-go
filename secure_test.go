package randomizer

import (
	"fmt"
	"math"
	"sync"
	"testing"
	"unsafe"
)

type secureInt8 int8
type secureInt16 int16
type secureInt32 int32
type secureInt64 int64
type secureInt int
type secureUint8 uint8
type secureUint16 uint16
type secureUint32 uint32
type secureUint64 uint64
type secureUint uint
type secureUintptr uintptr

func TestSecureIntegerFills(t *testing.T) {
	g := NewSecure()
	testSecureIntegerFill(t, g.FillInt[int8])
	testSecureIntegerFill(t, g.FillInt[int16])
	testSecureIntegerFill(t, g.FillInt[int32])
	testSecureIntegerFill(t, g.FillInt[int64])
	testSecureIntegerFill(t, g.FillInt[int])
	testSecureIntegerFill(t, g.FillUint[uint8])
	testSecureIntegerFill(t, g.FillUint[uint16])
	testSecureIntegerFill(t, g.FillUint[uint32])
	testSecureIntegerFill(t, g.FillUint[uint64])
	testSecureIntegerFill(t, g.FillUint[uint])
	testSecureIntegerFill(t, g.FillUint[uintptr])
	testSecureIntegerFill(t, g.FillInt[secureInt8])
	testSecureIntegerFill(t, g.FillInt[secureInt16])
	testSecureIntegerFill(t, g.FillInt[secureInt32])
	testSecureIntegerFill(t, g.FillInt[secureInt64])
	testSecureIntegerFill(t, g.FillInt[secureInt])
	testSecureIntegerFill(t, g.FillUint[secureUint8])
	testSecureIntegerFill(t, g.FillUint[secureUint16])
	testSecureIntegerFill(t, g.FillUint[secureUint32])
	testSecureIntegerFill(t, g.FillUint[secureUint64])
	testSecureIntegerFill(t, g.FillUint[secureUint])
	testSecureIntegerFill(t, g.FillUint[secureUintptr])
}

func testSecureIntegerFill[T Integers](t *testing.T, fill func([]T)) {
	t.Helper()
	var value T
	t.Run(fmt.Sprintf("%T", value), func(t *testing.T) {
		width := int(unsafe.Sizeof(value))
		words := []uint64{
			0, 1, 0x7f, 0x80, 0xff, 0x7fff, 0x8000, 0xffff,
			0x7fffffff, 0x80000000, 0xffffffff, 0x7fffffffffffffff,
			0x8000000000000000, 0xffffffffffffffff, 0xfedcba9876543210,
		}
		decoded := make([]T, len(words))
		decodeIntegers(decoded, secureWordBytes(words, width))
		for i, word := range words {
			if decoded[i] != T(word) {
				t.Fatalf("decoded[%d]=%v, want %v", i, decoded[i], T(word))
			}
		}
		chunk := 256 / width
		for _, length := range []int{0, 1, chunk - 1, chunk, chunk + 1, 2*chunk + 3} {
			storage := make([]T, length+2)
			storage[0], storage[len(storage)-1] = T(0x55), T(0x55)
			fill(storage[1 : len(storage)-1])
			if storage[0] != T(0x55) || storage[len(storage)-1] != T(0x55) {
				t.Fatalf("length=%d changed neighboring storage", length)
			}
		}
		var storage [257]T
		if allocations := testing.AllocsPerRun(100, func() { fill(storage[:]) }); allocations != 0 {
			t.Fatalf("secure integer fill allocations=%v, want 0", allocations)
		}
	})
}

func TestSecureFloatFills(t *testing.T) {
	g := NewSecure()
	testSecureFloatFill(t, g.FillFloat32)
	testSecureFloatFill(t, g.FillFloat64)
}

func testSecureFloatFill[T ~float32 | ~float64](t *testing.T, fill func([]T)) {
	t.Helper()
	var value T
	t.Run(fmt.Sprintf("%T", value), func(t *testing.T) {
		width := int(unsafe.Sizeof(value))
		words := []uint64{0, 255, 256, 1 << 31, math.MaxUint32}
		want := []T{0, 0, T(inv24), 0.5, T(1 - inv24)}
		if width == 8 {
			words = []uint64{0, 1<<11 - 1, 1 << 11, 1 << 63, math.MaxUint64}
			want = []T{0, 0, T(inv53), 0.5, T(1 - inv53)}
		}
		decoded := make([]T, len(words))
		decodeFloats(decoded, secureWordBytes(words, width))
		for i := range want {
			if decoded[i] != want[i] {
				t.Fatalf("decoded[%d]=%v, want %v", i, decoded[i], want[i])
			}
		}
		chunk := 256 / width
		for _, length := range []int{0, 1, chunk - 1, chunk, chunk + 1, 2*chunk + 3} {
			storage := make([]T, length+2)
			storage[0], storage[len(storage)-1] = -1, -1
			fill(storage[1 : len(storage)-1])
			if storage[0] != -1 || storage[len(storage)-1] != -1 {
				t.Fatalf("length=%d changed neighboring storage", length)
			}
			for i, number := range storage[1 : len(storage)-1] {
				if !(number >= 0 && number < 1) {
					t.Fatalf("length=%d value[%d]=%v outside [0,1)", length, i, number)
				}
			}
		}
		var storage [257]T
		if allocations := testing.AllocsPerRun(100, func() { fill(storage[:]) }); allocations != 0 {
			t.Fatalf("secure float fill allocations=%v, want 0", allocations)
		}
	})
}

func secureWordBytes(words []uint64, width int) []byte {
	data := make([]byte, len(words)*width)
	for i, word := range words {
		for j := range width {
			data[i*width+j] = byte(word >> (j * 8))
		}
	}
	return data
}

func TestSecureFillConcurrent(t *testing.T) {
	g := NewSecure()
	var group sync.WaitGroup
	for range 16 {
		group.Go(func() {
			var (
				integers [257]uint64
				floats   [257]float64
			)
			for range 32 {
				g.FillUint(integers[:])
				g.FillFloat64(floats[:])
				for _, number := range floats {
					if !(number >= 0 && number < 1) {
						t.Errorf("concurrent float %v outside [0,1)", number)
					}
				}
			}
		})
	}
	group.Wait()
}
