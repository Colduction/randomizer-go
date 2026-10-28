package randomizer_test

import (
	"math"
	"slices"
	"testing"

	"github.com/colduction/randomizer-go"
)

type signedOctet int8
type unsignedOctet uint8
type entropyForbidden struct{}

func (entropyForbidden) Uint64() uint64 { panic("unexpected entropy consumption") }

func TestNumericExtremaAndNamedTypes(t *testing.T) {
	g := randomizer.NewSeeded([32]byte{7})
	for _, bounds := range [][2]int64{
		{math.MinInt64, math.MaxInt64}, {math.MinInt64, 1}, {-1, math.MaxInt64}, {math.MaxInt64, math.MinInt64}, {-2, 2},
	} {
		low, high := min(bounds[0], bounds[1]), max(bounds[0], bounds[1])
		var values [257]int64
		g.FillIntInterval(values[:], bounds[0], bounds[1])
		for _, got := range values {
			if got < low || got >= high {
				t.Fatalf("FillIntInterval(%v) = %d", bounds, got)
			}
		}
		for range 128 {
			if got := g.IntInterval(bounds[0], bounds[1]); got < low || got >= high {
				t.Fatalf("IntInterval(%v) = %d", bounds, got)
			}
		}
	}
	for _, bounds := range [][2]uint64{
		{0, math.MaxUint64}, {0, 1<<63 + 1}, {math.MaxUint64, 1}, {math.MaxUint64 - 1, math.MaxUint64}, {0, 1 << 63},
	} {
		var values [257]uint64
		g.FillUintInterval(values[:], bounds[0], bounds[1])
		for _, got := range values {
			if got < min(bounds[0], bounds[1]) || got >= max(bounds[0], bounds[1]) {
				t.Fatalf("FillUintInterval(%v) = %d", bounds, got)
			}
		}
	}
	for range 128 {
		if got := g.IntInterval(signedOctet(-128), signedOctet(127)); got == 127 {
			t.Fatalf("named signed interval = %d", got)
		}
		if got := g.UintInterval(unsignedOctet(1), unsignedOctet(255)); got == 0 || got == 255 {
			t.Fatalf("named unsigned interval = %d", got)
		}
	}
	var signed [257]signedOctet
	var unsigned [257]unsignedOctet
	g.FillIntInterval(signed[:], signedOctet(-128), signedOctet(127))
	g.FillUintInterval(unsigned[:], unsignedOctet(1), unsignedOctet(255))
	if slices.Contains(signed[:], signedOctet(127)) || slices.Contains(unsigned[:], unsignedOctet(0)) || slices.Contains(unsigned[:], unsignedOctet(255)) {
		t.Fatal("narrow bulk interval escaped bounds")
	}
	_ = g.Int[int16]()
	_ = g.Int[int32]()
	_ = g.Int[int]()
	_ = g.Uint[uint16]()
	_ = g.Uint[uint32]()
	_ = g.Uint[uint]()
	_ = g.Uint[uintptr]()
}

func TestNumericConstantAndEmptyConsumeNoEntropy(t *testing.T) {
	g := randomizer.FromSource(entropyForbidden{})
	var signed [33]int64
	var unsigned [33]uint64
	g.FillIntInterval(signed[:], int64(-9), int64(-9))
	g.FillUintInterval(unsigned[:], uint64(7), uint64(8))
	for i := range signed {
		if signed[i] != -9 || unsigned[i] != 7 {
			t.Fatal("constant fill differs")
		}
	}
	if g.IntInterval(int64(-9), int64(-8)) != -9 || g.UintInterval(uint64(99), uint64(99)) != 99 {
		t.Fatal("constant scalar differs")
	}
	g.FillInt([]int64(nil))
	g.FillUint([]uint64(nil))
	g.FillIntInterval([]int64(nil), -2, 2)
	g.FillUintInterval([]uint64(nil), 1, 100)
	g.FillFloat32(nil)
	g.FillFloat64(nil)
}

func TestPackedNumericFill(t *testing.T) {
	g := randomizer.FromSource(fixedProvider(0x0123456789abcdef))
	var octets [9]uint8
	g.FillUint(octets[:])
	want := []uint8{0xef, 0xcd, 0xab, 0x89, 0x67, 0x45, 0x23, 0x01, 0xef}
	if !slices.Equal(octets[:], want) {
		t.Fatalf("packed octets = %x", octets)
	}
	var signed [9]int8
	g.FillInt(signed[:])
	for i := range signed {
		if signed[i] != int8(want[i]) {
			t.Fatalf("signed octet %d = %d", i, signed[i])
		}
	}
	var words [5]uint16
	g.FillUint(words[:])
	if !slices.Equal(words[:], []uint16{0xcdef, 0x89ab, 0x4567, 0x0123, 0xcdef}) {
		t.Fatalf("packed uint16 = %x", words)
	}
	var halves [3]uint32
	g.FillUint(halves[:])
	if !slices.Equal(halves[:], []uint32{0x89abcdef, 0x01234567, 0x89abcdef}) {
		t.Fatalf("packed uint32 = %x", halves)
	}
}

func TestFloatExactEndpoints(t *testing.T) {
	for _, tc := range []struct {
		word uint64
		want float64
	}{{0, 0}, {math.MaxUint64, 1 - 1.0/(1<<53)}} {
		g := randomizer.FromSource(fixedProvider(tc.word))
		if got := g.Float64(); got != tc.want {
			t.Fatalf("Float64 = %.18g, want %.18g", got, tc.want)
		}
		var values [17]float64
		g.FillFloat64(values[:])
		for _, got := range values {
			if got != tc.want {
				t.Fatalf("FillFloat64 = %.18g, want %.18g", got, tc.want)
			}
		}
	}
	g := randomizer.FromSource(newSeqProvider(t, 0, math.MaxUint64))
	if got := g.Float32(); got != 0 {
		t.Fatalf("Float32 minimum = %g", got)
	}
	if got := g.Float32(); got != 1-1.0/(1<<24) {
		t.Fatalf("Float32 maximum = %g", got)
	}
	var values [33]float32
	randomizer.FillFloat32(values[:])
	for _, got := range values {
		if got < 0 || got >= 1 {
			t.Fatalf("FillFloat32 out of range: %g", got)
		}
	}
}

func TestNumericBulkAllocs(t *testing.T) {
	for _, g := range []*randomizer.Generator{randomizer.New(), randomizer.NewSeeded([32]byte{}), randomizer.NewSecure()} {
		var ints [257]int64
		var uints [257]uint64
		var f32 [257]float32
		var f64 [257]float64
		if got := testing.AllocsPerRun(100, func() {
			g.FillInt(ints[:])
			g.FillUint(uints[:])
			g.FillIntInterval(ints[:], int64(-1000), int64(1000))
			g.FillUintInterval(uints[:], uint64(7), uint64(1007))
			g.FillFloat32(f32[:])
			g.FillFloat64(f64[:])
		}); got != 0 {
			t.Fatalf("numeric bulk allocations = %g", got)
		}
	}
}

func FuzzNumericIntervals(f *testing.F) {
	f.Add(int64(math.MinInt64), int64(math.MaxInt64), uint16(257))
	f.Add(int64(-9), int64(-8), uint16(33))
	f.Add(int64(0), int64(0), uint16(0))
	f.Fuzz(func(t *testing.T, low, high int64, length uint16) {
		g := randomizer.NewSeeded([32]byte{1})
		values := make([]int64, int(length)%513)
		g.FillIntInterval(values, low, high)
		if low > high {
			low, high = high, low
		}
		for _, got := range values {
			if low == high && got != low || low != high && (got < low || got >= high) {
				t.Fatalf("%d outside [%d,%d)", got, low, high)
			}
		}
	})
}
