package randomizer_test

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"math/rand"
	randv2 "math/rand/v2"
	"sync"
	"testing"
	"testing/iotest"

	"github.com/colduction/randomizer-go"
)

type fixedProvider uint64

func (fp fixedProvider) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(uint64(fp) >> ((i % 8) * 8))
	}
	return len(p), nil
}

func (fp fixedProvider) Uint64() uint64 { return uint64(fp) }

type seqProvider struct {
	t    testing.TB
	vals []uint64
	i    int
}

func newSeqProvider(t testing.TB, vals ...uint64) *seqProvider {
	return &seqProvider{t: t, vals: vals}
}

func (sp *seqProvider) Uint64() uint64 {
	if sp.i >= len(sp.vals) {
		sp.t.Fatalf("seqProvider exhausted after %d words", len(sp.vals))
	}
	v := sp.vals[sp.i]
	sp.i++
	return v
}

func (sp *seqProvider) Read(p []byte) (int, error) {
	for i := 0; i < len(p); {
		x := sp.Uint64()
		for range min(8, len(p)-i) {
			p[i] = byte(x)
			x >>= 8
			i++
		}
	}
	return len(p), nil
}

type countingSource struct {
	calls uint64
}

func (s *countingSource) Uint64() uint64 {
	s.calls++
	return s.calls
}

type panicOnceSource struct {
	calls uint64
}

func (s *panicOnceSource) Uint64() uint64 {
	s.calls++
	if s.calls == 1 {
		panic("source failure")
	}
	return s.calls
}

type failingReader struct{}

func (failingReader) Read(dst []byte) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}
	dst[0] = 0xab
	return 1, io.ErrUnexpectedEOF
}

func TestGeneratorRoutesAndIsolatesSources(t *testing.T) {
	const value fixedProvider = 0x0123456789abcdef
	g := randomizer.FromSource(value)
	other := randomizer.FromSource(fixedProvider(99))
	if g.Uint[uint64]() != uint64(value) || other.Uint[uint64]() != 99 {
		t.Fatal("generator sources interfered")
	}
	first := g.Word.String(randomizer.HexLowerAlphabet, 16)
	if second := g.Word.String(randomizer.HexLowerAlphabet, 16); first != second || len(first) != 16 {
		t.Fatalf("fixed words = %q, %q", first, second)
	}
	if got := g.Network.Value(randomizer.VLANID); got != 0x12 {
		t.Fatalf("VLANID = %x", got)
	}
	var buf [12]byte
	n, err := g.Read(buf[:])
	want := []byte{0xef, 0xcd, 0xab, 0x89, 0x67, 0x45, 0x23, 0x01, 0xef, 0xcd, 0xab, 0x89}
	if n != len(buf) || err != nil || !bytes.Equal(buf[:], want) {
		t.Fatalf("Read = %x, %d, %v", buf, n, err)
	}
}

func TestConstructorBindsAllHandles(t *testing.T) {
	source := new(countingSource)
	var data [48]byte
	for i := range 6 {
		binary.LittleEndian.PutUint64(data[i*8:], uint64(i+1))
	}
	reader := bytes.NewReader(data[:])
	for _, tc := range []struct {
		name         string
		generator    *randomizer.Generator
		consumed     func() int
		readConsumed int
	}{
		{"source", randomizer.FromSource(source), func() int { return int(source.calls * 8) }, 48},
		{"reader", randomizer.FromReader(reader), func() int { return len(data) - reader.Len() }, 41},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := tc.generator
			if tc.consumed() != 0 || g.Uint[uint64]() != 1 || tc.consumed() != 8 {
				t.Fatal("construction or numeric binding consumed unexpected input")
			}
			if got := g.Word.String(randomizer.HexLowerAlphabet, 16); got != "2000000000000000" || tc.consumed() != 16 {
				t.Fatalf("bound word = %q after %d input bytes", got, tc.consumed())
			}
			_ = g.Network.Value(randomizer.VLANID)
			if tc.consumed() != 24 {
				t.Fatalf("network handle consumed %d input bytes", tc.consumed())
			}
			if got := g.Telecom.String(randomizer.MEID, randomizer.TelecomOptions{}); got != "A0000000000000" || tc.consumed() != 32 {
				t.Fatalf("bound telecom identifier = %q after %d input bytes", got, tc.consumed())
			}
			var out [9]byte
			if n, err := g.Read(out[:]); n != len(out) || err != nil {
				t.Fatalf("bound read = %d, %v", n, err)
			}
			if binary.LittleEndian.Uint64(out[:8]) != 5 || out[8] != 6 || tc.consumed() != tc.readConsumed {
				t.Fatalf("bound read = %x after %d input bytes", out, tc.consumed())
			}
		})
	}
}

func TestGeneratorZeroAndNil(t *testing.T) {
	var zero randomizer.Generator
	for _, g := range []*randomizer.Generator{nil, &zero, randomizer.New()} {
		var buf [17]byte
		if n, err := g.Read(buf[:]); n != len(buf) || err != nil {
			t.Fatalf("Read = %d, %v", n, err)
		}
		if got := g.UintInterval(uint64(7), uint64(7)); got != 7 {
			t.Fatalf("equal interval = %d", got)
		}
	}
	if len(zero.Word.String(randomizer.DecimalAlphabet, 8)) != 8 || !zero.Network.Addr(randomizer.IPv4Any).Is4() || len(zero.Telecom.String(randomizer.IMEI, randomizer.TelecomOptions{})) != 15 {
		t.Fatal("zero generator handles are unusable")
	}
}

func TestFromSourceFaithful(t *testing.T) {
	const seed int64 = 42
	g := randomizer.FromSource(rand.New(rand.NewSource(seed)))
	want := rand.New(rand.NewSource(seed))
	for range 128 {
		if got, expected := g.Uint[uint64](), want.Uint64(); got != expected {
			t.Fatalf("source adapter = %x, want %x", got, expected)
		}
	}
	source := new(countingSource)
	g = randomizer.FromSource(source)
	if source.calls != 0 {
		t.Fatal("constructor consumed entropy")
	}
	var out [17]byte
	_, _ = g.Read(out[:])
	if source.calls != 3 || binary.LittleEndian.Uint64(out[:8]) != 1 || binary.LittleEndian.Uint64(out[8:16]) != 2 || out[16] != 3 {
		t.Fatalf("bulk source routing = %x after %d calls", out, source.calls)
	}
}

func TestFromSourceConcurrent(t *testing.T) {
	const workers, count = 16, 128
	source := new(countingSource)
	g := randomizer.FromSource(source)
	results := make(chan uint64, workers*count)
	var group sync.WaitGroup
	for range workers {
		group.Go(func() {
			for range count {
				results <- g.Uint[uint64]()
			}
		})
	}
	group.Wait()
	close(results)
	seen := make([]bool, workers*count+1)
	for value := range results {
		if value < 1 || value > workers*count || seen[value] {
			t.Fatalf("serialized source returned %d twice or outside range", value)
		}
		seen[value] = true
	}
	if source.calls != workers*count {
		t.Fatalf("source calls = %d", source.calls)
	}
}

func TestFromSourcePanicUnlock(t *testing.T) {
	for _, tc := range []struct {
		name  string
		draw  func(*randomizer.Generator)
		words uint64
	}{
		{"scalar", func(g *randomizer.Generator) { _ = g.Uint[uint64]() }, 1},
		{"read", func(g *randomizer.Generator) {
			var out [16]byte
			_, _ = g.Read(out[:])
		}, 2},
		{"bulk", func(g *randomizer.Generator) {
			var out [4]uint64
			g.FillUint(out[:])
		}, 4},
		{"word", func(g *randomizer.Generator) {
			var out [16]byte
			g.Word.Append(out[:0], randomizer.HexLowerAlphabet, len(out))
		}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := new(panicOnceSource)
			g := randomizer.FromSource(source)
			func() {
				defer func() {
					if got := recover(); got != "source failure" {
						t.Fatalf("source panic = %v", got)
					}
				}()
				tc.draw(g)
			}()
			tc.draw(g)
			tc.draw(g)
			if got, want := g.Uint[uint64](), 2+2*tc.words; got != want || source.calls != want {
				t.Fatalf("source after panic = %d, calls=%d, want %d", got, source.calls, want)
			}
		})
	}
}

func TestFromReaderValues(t *testing.T) {
	data := make([]byte, 20)
	for i := range data {
		data[i] = byte(i + 1)
	}
	g := randomizer.FromReader(bytes.NewReader(data))
	if got, want := g.Float32(), float32(0x040302)*(1.0/(1<<24)); got != want {
		t.Fatalf("Float32 = %g, want %g", got, want)
	}
	if got := g.Uint[uint64](); got != 0x0c0b0a0908070605 {
		t.Fatalf("Uint64 = %x", got)
	}
	var out [8]byte
	if n, err := g.Read(out[:]); n != len(out) || err != nil || !bytes.Equal(out[:], data[12:]) {
		t.Fatalf("remaining read = %x, %d, %v", out, n, err)
	}
}

func TestFromReaderFullReads(t *testing.T) {
	data := []byte("complete reader input")
	g := randomizer.FromReader(iotest.OneByteReader(bytes.NewReader(data)))
	out := make([]byte, len(data)+1)
	if n, err := g.Read(out[:len(data)]); n != len(data) || err != nil || !bytes.Equal(out[:n], data) {
		t.Fatalf("full read = %x, %d, %v", out[:n], n, err)
	}
	if n, err := g.Read(out[:1]); n != 0 || !errors.Is(err, io.EOF) {
		t.Fatalf("exhausted read = %d, %v", n, err)
	}
	g = randomizer.FromReader(iotest.OneByteReader(bytes.NewReader(data)))
	if n, err := g.Read(out); n != len(data) || !errors.Is(err, io.ErrUnexpectedEOF) || !bytes.Equal(out[:n], data) {
		t.Fatalf("partial full read = %x, %d, %v", out[:n], n, err)
	}
}

func TestFromReaderShortReadAndPanicUnlock(t *testing.T) {
	g := randomizer.FromReader(failingReader{})
	var buf [8]byte
	if n, err := g.Read(buf[:]); n != 1 || !errors.Is(err, io.ErrUnexpectedEOF) || buf[0] != 0xab {
		t.Fatalf("partial read = %d, %v, %x", n, err, buf)
	}
	for _, draw := range []func(){
		func() { _ = g.Uint[uint64]() },
		func() { g.FillUint(buf[:]) },
		func() { g.Word.Append(buf[:0], randomizer.Base64URLAlphabet, len(buf)) },
	} {
		func() {
			defer func() {
				if got := recover(); got != io.ErrUnexpectedEOF {
					t.Fatalf("panic = %v", got)
				}
			}()
			draw()
		}()
	}
	if n, err := g.Read(nil); n != 0 || err != nil {
		t.Fatalf("empty read = %d, %v", n, err)
	}
}

func TestFromReaderConcurrent(t *testing.T) {
	const count = 1024
	data := make([]byte, count*8)
	for i := range count {
		binary.LittleEndian.PutUint64(data[i*8:], uint64(i))
	}
	g := randomizer.FromReader(bytes.NewReader(data))
	results := make(chan uint64, count)
	var group sync.WaitGroup
	for range 16 {
		group.Go(func() {
			var values [64]uint64
			g.FillUint(values[:])
			for _, value := range values {
				results <- value
			}
		})
	}
	group.Wait()
	close(results)
	seen := make([]bool, count)
	for value := range results {
		if value >= count || seen[value] {
			t.Fatalf("reader returned duplicate/out-of-range %d", value)
		}
		seen[value] = true
	}
}

func TestFromConstructorsRejectNil(t *testing.T) {
	for name, constructor := range map[string]func() *randomizer.Generator{
		"source": func() *randomizer.Generator { return randomizer.FromSource(nil) },
		"reader": func() *randomizer.Generator { return randomizer.FromReader(nil) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("nil input did not panic")
				}
			}()
			_ = constructor()
		})
	}
}

func TestTypedNilDoesNotSelectFastSource(t *testing.T) {
	var (
		source *countingSource
		reader *bytes.Reader
	)
	for name, g := range map[string]*randomizer.Generator{
		"source": randomizer.FromSource(source),
		"reader": randomizer.FromReader(reader),
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("typed nil silently selected a usable source")
				}
			}()
			_ = g.Uint[uint64]()
		})
	}
}

func TestSeededGeneratorMatchesChaCha8(t *testing.T) {
	seed := [32]byte{1, 2, 3, 4}
	g := randomizer.NewSeeded(seed)
	want := randv2.NewChaCha8(seed)
	for range 256 {
		if got, expected := g.Uint[uint64](), want.Uint64(); got != expected {
			t.Fatalf("seeded word = %x, want %x", got, expected)
		}
	}
	first, second := randomizer.NewSeeded(seed), randomizer.NewSeeded(seed)
	for range 16 {
		if first.Word.String(randomizer.AlphaNumericAlphabet, 37) != second.Word.String(randomizer.AlphaNumericAlphabet, 37) || first.Network.Addr(randomizer.IPv6Public) != second.Network.Addr(randomizer.IPv6Public) || first.Telecom.String(randomizer.IMEI, randomizer.TelecomOptions{}) != second.Telecom.String(randomizer.IMEI, randomizer.TelecomOptions{}) {
			t.Fatal("same seeded calls differed")
		}
	}
}

func TestGeneratorSharedBuiltins(t *testing.T) {
	for _, g := range []*randomizer.Generator{
		randomizer.New(),
		randomizer.NewSeeded([32]byte{}),
		randomizer.NewSecure(),
		randomizer.FromSource(randv2.NewPCG(1, 2)),
		randomizer.FromReader(randv2.NewChaCha8([32]byte{1})),
		randomizer.FromReader(cryptorand.Reader),
	} {
		words := g.Word
		noAdjacent := words.WithMode(randomizer.NoAdjacentRepeats)
		network := g.Network
		telecom := g.Telecom
		var group sync.WaitGroup
		for range 8 {
			group.Go(func() {
				var values [16]uint64
				var out [64]byte
				for range 32 {
					g.FillUint(values[:])
					g.FillUintInterval(values[:], uint64(3), uint64(997))
					words.Append(out[:0], randomizer.AlphaNumericAlphabet, 64)
					noAdjacent.Append(out[:0], randomizer.DecimalAlphabet, 64)
					network.IP(out[:0], randomizer.IPv6Public)
					telecom.Append(out[:0], randomizer.IMEI, randomizer.TelecomOptions{})
				}
			})
		}
		group.Wait()
	}
}

var benchProviderReadBuf [256]byte

func BenchmarkProviderReadDefault(b *testing.B) {
	buf := benchProviderReadBuf[:]
	b.ReportAllocs()
	for b.Loop() {
		_, _ = randomizer.Read(buf)
	}
}

func BenchmarkProviderReadDefaultParallel(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var buf [256]byte
		for pb.Next() {
			_, _ = randomizer.Read(buf[:])
		}
	})
}
