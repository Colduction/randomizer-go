package randomizer_test

import (
	cryptorand "crypto/rand"
	"fmt"
	randv2 "math/rand/v2"
	"runtime"
	"testing"

	"github.com/colduction/randomizer-go"
)

var performanceSources = []struct {
	name string
	new  func() *randomizer.Generator
}{
	{"Fast", randomizer.New},
	{"Seeded", func() *randomizer.Generator { return randomizer.NewSeeded([32]byte{1}) }},
	{"Secure", randomizer.NewSecure},
	{"Reader", func() *randomizer.Generator {
		return randomizer.FromReader(randv2.NewChaCha8([32]byte{1}))
	}},
	{"CryptoReader", func() *randomizer.Generator {
		return randomizer.FromReader(cryptorand.Reader)
	}},
}

var constructedGenerator *randomizer.Generator

func BenchmarkGeneratorConstruction(b *testing.B) {
	for _, tc := range []struct {
		name string
		new  func() *randomizer.Generator
	}{
		{"New", randomizer.New},
		{"NewSecure", randomizer.NewSecure},
		{"NewSeeded", func() *randomizer.Generator { return randomizer.NewSeeded([32]byte{1}) }},
		{"FromSource", func() *randomizer.Generator { return randomizer.FromSource(randv2.NewPCG(1, 2)) }},
		{"FromReader", func() *randomizer.Generator { return randomizer.FromReader(randv2.NewChaCha8([32]byte{1})) }},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				constructedGenerator = tc.new()
			}
		})
	}
}

func BenchmarkNumericBulk(b *testing.B) {
	for _, source := range performanceSources {
		for _, length := range []int{16, 256, 4096} {
			b.Run(fmt.Sprintf("%s/%d/Fill", source.name, length), func(b *testing.B) {
				g := source.new()
				out := make([]uint64, length)
				b.ReportAllocs()
				b.SetBytes(int64(length * 8))
				for b.Loop() {
					g.FillUintInterval(out, uint64(0), uint64(1000))
				}
				runtime.KeepAlive(out)
			})
			b.Run(fmt.Sprintf("%s/%d/Scalar", source.name, length), func(b *testing.B) {
				g := source.new()
				out := make([]uint64, length)
				b.ReportAllocs()
				b.SetBytes(int64(length * 8))
				for b.Loop() {
					for i := range out {
						out[i] = g.UintInterval(uint64(0), uint64(1000))
					}
				}
				runtime.KeepAlive(out)
			})
		}
	}
}

func BenchmarkSourceWorkloads(b *testing.B) {
	for _, source := range performanceSources {
		b.Run(source.name, func(b *testing.B) {
			g := source.new()
			b.Run("Uint64", func(b *testing.B) {
				var sink uint64
				b.ReportAllocs()
				for b.Loop() {
					sink = g.Uint[uint64]()
				}
				runtime.KeepAlive(sink)
			})
			b.Run("Read256", func(b *testing.B) {
				var out [256]byte
				b.ReportAllocs()
				b.SetBytes(int64(len(out)))
				for b.Loop() {
					_, _ = g.Read(out[:])
				}
			})
			b.Run("Word256", func(b *testing.B) {
				var out [256]byte
				b.ReportAllocs()
				for b.Loop() {
					g.Word.Append(out[:0], randomizer.AlphaNumericAlphabet, len(out))
				}
			})
			b.Run("WordNoAdjacent256", func(b *testing.B) {
				var out [256]byte
				words := g.Word.WithMode(randomizer.NoAdjacentRepeats)
				b.ReportAllocs()
				for b.Loop() {
					words.Append(out[:0], randomizer.AlphaNumericAlphabet, len(out))
				}
			})
			b.Run("IPv6Public", func(b *testing.B) {
				var out [16]byte
				b.ReportAllocs()
				for b.Loop() {
					g.Network.IP(out[:0], randomizer.IPv6Public)
				}
			})
			b.Run("IMEI", func(b *testing.B) {
				var out [16]byte
				b.ReportAllocs()
				for b.Loop() {
					g.Telecom.Append(out[:0], randomizer.IMEI, randomizer.TelecomOptions{})
				}
			})
			b.Run("Shuffle256", func(b *testing.B) {
				var out [256]int
				for i := range out {
					out[i] = i
				}
				b.ReportAllocs()
				for b.Loop() {
					g.Shuffle(out[:])
				}
			})
			b.Run("Weighted256", func(b *testing.B) {
				weights := make([]uint64, 256)
				for i := range weights {
					weights[i] = uint64(i + 1)
				}
				w, err := randomizer.NewWeighted(weights)
				if err != nil {
					b.Fatal(err)
				}
				var sink int
				b.ReportAllocs()
				for b.Loop() {
					sink = w.Index(g)
				}
				runtime.KeepAlive(sink)
			})
		})
	}
}

func BenchmarkParallelSources(b *testing.B) {
	for _, source := range performanceSources {
		b.Run(source.name, func(b *testing.B) {
			g := source.new()
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				var out [256]uint64
				for pb.Next() {
					g.FillUintInterval(out[:], uint64(0), uint64(1000))
				}
				runtime.KeepAlive(out)
			})
		})
	}
}

func BenchmarkNumericPacked(b *testing.B) {
	for _, source := range performanceSources[:3] {
		b.Run(source.name, func(b *testing.B) {
			g := source.new()
			b.Run("Uint8/Fill", func(b *testing.B) {
				var out [256]byte
				b.ReportAllocs()
				for b.Loop() {
					g.FillUint(out[:])
				}
			})
			b.Run("Uint8/Scalar", func(b *testing.B) {
				var out [256]byte
				b.ReportAllocs()
				for b.Loop() {
					for i := range out {
						out[i] = g.Uint[uint8]()
					}
				}
				runtime.KeepAlive(out)
			})
			b.Run("Float64/Fill", func(b *testing.B) {
				var out [256]float64
				b.ReportAllocs()
				for b.Loop() {
					g.FillFloat64(out[:])
				}
			})
			b.Run("Float64/Scalar", func(b *testing.B) {
				var out [256]float64
				b.ReportAllocs()
				for b.Loop() {
					for i := range out {
						out[i] = g.Float64()
					}
				}
				runtime.KeepAlive(out)
			})
		})
	}
}

func BenchmarkHardwareRPAKeys(b *testing.B) {
	for _, count := range []int{1, 2, 16} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			keys := make([][16]byte, count)
			for i := range keys {
				keys[i][0] = byte(i + 1)
			}
			var out [6]byte
			var i int
			b.ReportAllocs()
			for b.Loop() {
				randomizer.Network.Hardware(out[:0], randomizer.HardwareBluetoothRPA, randomizer.HardwareOptions{IRK: keys[i%count][:]})
				i++
			}
		})
	}
}

func BenchmarkIIDAlgorithms(b *testing.B) {
	for _, tc := range []struct {
		name       string
		alphabet   randomizer.Alphabet
		dictionary string
	}{
		{"Hex", randomizer.HexLowerAlphabet, "0123456789abcdef"},
		{"AlphaNumeric", randomizer.AlphaNumericAlphabet, "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.Run("Batched", func(b *testing.B) {
				var out [256]byte
				b.ReportAllocs()
				for b.Loop() {
					randomizer.Word.Append(out[:0], tc.alphabet, len(out))
				}
			})
			b.Run("Scalar", func(b *testing.B) {
				var out [256]byte
				b.ReportAllocs()
				for b.Loop() {
					for i := range out {
						out[i] = tc.dictionary[randv2.Uint64N(uint64(len(tc.dictionary)))]
					}
				}
				runtime.KeepAlive(out)
			})
		})
	}
}
