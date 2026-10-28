package randomizer

import (
	"errors"
	"fmt"
	"math/bits"
	"math/rand/v2"
	"runtime"
	"slices"
	"sync"
	"testing"
)

type samplingScript struct {
	words []uint64
	next  int
}

func (s *samplingScript) Read(dst []byte) (int, error) {
	fillRandomBytes(dst, s)
	return len(dst), nil
}

func (s *samplingScript) Uint32() uint32 {
	return uint32(s.Uint64() >> 32)
}

func (s *samplingScript) Uint64() uint64 {
	if s.next == len(s.words) {
		panic("unexpected random draw")
	}
	word := s.words[s.next]
	s.next++
	return word
}

func wordForChoice(choice, bound uint64) uint64 {
	word, _ := bits.Div64(choice, 0, bound)
	return word + 2
}

func forEachSamplingPath(bounds []uint64, visit func([]uint64)) {
	words := make([]uint64, len(bounds))
	var walk func(int)
	walk = func(i int) {
		if i == len(bounds) {
			visit(words)
			return
		}
		for choice := range bounds[i] {
			words[i] = wordForChoice(choice, bounds[i])
			walk(i + 1)
		}
	}
	walk(0)
}

func validSample(values []int, population int) bool {
	for i, value := range values {
		if value < 0 || value >= population {
			return false
		}
		if slices.Contains(values[:i], value) {
			return false
		}
	}
	return true
}

func TestShuffleExhaustive(t *testing.T) {
	for n := 0; n <= 6; n++ {
		var bounds []uint64
		for bound := n; bound > 1; bound-- {
			bounds = append(bounds, uint64(bound))
		}
		counts := make(map[string]int)
		forEachSamplingPath(bounds, func(words []uint64) {
			values := make([]int, n)
			for i := range values {
				values[i] = i
			}
			g := newGenerator(&samplingScript{words: words})
			g.Shuffle(values)
			if !validSample(values, n) {
				t.Fatalf("shuffle of %d produced %v", n, values)
			}
			counts[fmt.Sprint(values)]++
		})
		want := 1
		for i := 2; i <= n; i++ {
			want *= i
		}
		if len(counts) != want {
			t.Fatalf("shuffle of %d produced %d permutations, want %d", n, len(counts), want)
		}
		for output, count := range counts {
			if count != 1 {
				t.Fatalf("shuffle of %d produced %s %d times", n, output, count)
			}
		}
	}
}

func TestSampleIndicesExhaustive(t *testing.T) {
	for population := 0; population <= 6; population++ {
		for k := 0; k <= population; k++ {
			var bounds []uint64
			if k == population {
				for bound := population; bound > 1; bound-- {
					bounds = append(bounds, uint64(bound))
				}
			} else {
				for i := range k {
					bounds = append(bounds, uint64(population-k+i+1))
				}
			}
			counts := make(map[string]int)
			forEachSamplingPath(bounds, func(words []uint64) {
				dst := make([]int, k)
				g := newGenerator(&samplingScript{words: words})
				if err := g.SampleIndices(dst, population); err != nil {
					t.Fatal(err)
				}
				if !validSample(dst, population) {
					t.Fatalf("sample %d of %d produced %v", k, population, dst)
				}
				counts[fmt.Sprint(dst)]++
			})
			want := 1
			for i := range k {
				want *= population - i
			}
			if len(counts) != want {
				t.Fatalf("sample %d of %d produced %d ordered samples, want %d", k, population, len(counts), want)
			}
			for output, count := range counts {
				if count != 1 {
					t.Fatalf("sample %d of %d produced %s %d times", k, population, output, count)
				}
			}
		}
	}
}

func TestSampleSparseExhaustive(t *testing.T) {
	for population := 1; population <= 6; population++ {
		for k := 1; k <= population; k++ {
			bounds := make([]uint64, k)
			for i := range bounds {
				bounds[i] = uint64(population - i)
			}
			counts := make(map[string]int)
			forEachSamplingPath(bounds, func(words []uint64) {
				dst := make([]int, k)
				sampleSparse(dst, population, &samplingScript{words: words})
				if !validSample(dst, population) {
					t.Fatalf("sparse sample %d of %d produced %v", k, population, dst)
				}
				counts[fmt.Sprint(dst)]++
			})
			want := 1
			for i := range k {
				want *= population - i
			}
			if len(counts) != want {
				t.Fatalf("sparse sample %d of %d produced %d ordered samples, want %d", k, population, len(counts), want)
			}
			for output, count := range counts {
				if count != 1 {
					t.Fatalf("sparse sample %d of %d produced %s %d times", k, population, output, count)
				}
			}
		}
	}
}

func TestSampleIndicesValidation(t *testing.T) {
	g := newGenerator(&samplingScript{})
	for _, tc := range []struct {
		population int
		values     []int
	}{
		{-1, nil},
		{-1, []int{9}},
		{0, []int{9}},
		{1, []int{9, 8}},
	} {
		before := slices.Clone(tc.values)
		if err := g.SampleIndices(tc.values, tc.population); !errors.Is(err, ErrInvalidSample) {
			t.Fatalf("sample length %d of %d error = %v", len(tc.values), tc.population, err)
		}
		if !slices.Equal(tc.values, before) {
			t.Fatalf("invalid sample changed %v to %v", before, tc.values)
		}
	}
	if err := g.SampleIndices(nil, 0); err != nil {
		t.Fatal(err)
	}
	if err := g.SampleIndices(nil, int(^uint(0)>>1)); err != nil {
		t.Fatal(err)
	}
	values := []int{9}
	if err := g.SampleIndices(values, 1); err != nil || values[0] != 0 {
		t.Fatalf("singleton sample = %v, %v", values, err)
	}
	g.Shuffle([]int{1})
	g.Shuffle([]int(nil))
}

func TestSampleIndicesBoundaries(t *testing.T) {
	g := NewSeeded([32]byte{42})
	for _, tc := range []struct {
		population int
		count      int
	}{
		{17, 16},
		{18, 17},
		{128, 127},
		{128, 128},
		{int(^uint(0) >> 1), 8},
		{int(^uint(0) >> 1), 17},
	} {
		values := make([]int, tc.count)
		if err := g.SampleIndices(values, tc.population); err != nil {
			t.Fatal(err)
		}
		if !validSample(values, tc.population) {
			t.Fatalf("sample %d of %d produced %v", tc.count, tc.population, values)
		}
	}
	values := make([]int, 17)
	if err := SampleIndices(values, 100); err != nil || !validSample(values, 100) {
		t.Fatalf("default sample = %v, %v", values, err)
	}
	Shuffle(values)
	if !validSample(values, 100) {
		t.Fatalf("default shuffle = %v", values)
	}
}

func TestSamplingReproducible(t *testing.T) {
	first := NewSeeded([32]byte{1, 2, 3})
	second := NewSeeded([32]byte{1, 2, 3})
	for _, count := range []int{1, 8, 17, 64} {
		a := make([]int, count)
		b := make([]int, count)
		if err := first.SampleIndices(a, 64); err != nil {
			t.Fatal(err)
		}
		if err := second.SampleIndices(b, 64); err != nil {
			t.Fatal(err)
		}
		first.Shuffle(a)
		second.Shuffle(b)
		if !slices.Equal(a, b) {
			t.Fatalf("matching seeded calls produced %v and %v", a, b)
		}
	}
}

func TestSamplingSharedGenerator(t *testing.T) {
	g := NewSeeded([32]byte{7})
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			var values [17]int
			for range 128 {
				if err := g.SampleIndices(values[:], 64); err != nil {
					t.Error(err)
					return
				}
				g.Shuffle(values[:])
				if !validSample(values[:], 64) {
					t.Errorf("concurrent sample = %v", values)
					return
				}
			}
		})
	}
	workers.Wait()
}

func TestSamplingSourceFailure(t *testing.T) {
	source := &samplingScript{words: []uint64{wordForChoice(0, 3)}}
	g := newGenerator(source)
	values := []int{-1, -1}
	defer func() {
		if recover() == nil {
			t.Error("exhausted source did not panic")
		}
		if values[0] != 0 || values[1] != -1 {
			t.Errorf("partially filled destination = %v", values)
		}
	}()
	_ = g.SampleIndices(values, 4)
}

func TestSamplingForcedRejection(t *testing.T) {
	source := &samplingScript{words: []uint64{0, wordForChoice(1, 3), wordForChoice(0, 2)}}
	values := []int{0, 1, 2}
	newGenerator(source).Shuffle(values)
	if !slices.Equal(values, []int{2, 0, 1}) || source.next != 3 {
		t.Fatalf("shuffle rejection gives %v after %d draws", values, source.next)
	}
	source = &samplingScript{words: []uint64{0, wordForChoice(0, 3), wordForChoice(0, 4)}}
	values = []int{-1, -1}
	if err := newGenerator(source).SampleIndices(values, 4); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(values, []int{3, 0}) || source.next != 3 {
		t.Fatalf("sample rejection/collision gives %v after %d draws", values, source.next)
	}
}

func TestSamplingAllocs(t *testing.T) {
	var values [256]int
	for _, count := range []int{0, 8, 16, 256} {
		if got := testing.AllocsPerRun(100, func() {
			if err := SampleIndices(values[:count], 256); err != nil {
				panic(err)
			}
		}); got != 0 {
			t.Fatalf("sample length %d allocates %v times", count, got)
		}
	}
	if got := testing.AllocsPerRun(100, func() { Shuffle(values[:]) }); got != 0 {
		t.Fatalf("shuffle allocates %v times", got)
	}
}

func FuzzSampleIndices(f *testing.F) {
	f.Add([]byte{1, 2, 3}, 64)
	f.Add(make([]byte, 17), 128)
	f.Add([]byte{9}, -1)
	f.Fuzz(func(t *testing.T, data []byte, population int) {
		if len(data) > 256 {
			return
		}
		population %= 1_000_001
		dst := make([]int, len(data))
		for i := range dst {
			dst[i] = -1
		}
		g := FromSource(rand.NewPCG(uint64(len(data)), uint64(population)))
		err := g.SampleIndices(dst, population)
		if population < 0 || len(dst) > population {
			if !errors.Is(err, ErrInvalidSample) {
				t.Fatalf("invalid input error = %v", err)
			}
			for _, value := range dst {
				if value != -1 {
					t.Fatal("invalid sample changed destination")
				}
			}
		} else if err != nil || !validSample(dst, population) {
			t.Fatalf("sample = %v, error = %v", dst, err)
		}
	})
}

func BenchmarkShuffle(b *testing.B) {
	for _, size := range []int{16, 256, 4096} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			for _, mode := range []string{"default", "seeded", "secure"} {
				b.Run(mode, func(b *testing.B) {
					var g *Generator
					if mode == "seeded" {
						g = NewSeeded([32]byte{1})
					} else if mode == "secure" {
						g = NewSecure()
					}
					values := make([]int, size)
					for i := range values {
						values[i] = i
					}
					b.ReportAllocs()
					for b.Loop() {
						g.Shuffle(values)
					}
					runtime.KeepAlive(values)
				})
			}
		})
	}
}

func BenchmarkSampleIndices(b *testing.B) {
	for _, tc := range []struct {
		name       string
		population int
		count      int
	}{
		{"small", 1_000_000, 8},
		{"sparse", 1_000_000, 64},
		{"dense", 1024, 512},
		{"full", 1024, 1024},
	} {
		b.Run(tc.name, func(b *testing.B) {
			dst := make([]int, tc.count)
			b.ReportAllocs()
			for b.Loop() {
				if err := SampleIndices(dst, tc.population); err != nil {
					b.Fatal(err)
				}
			}
			runtime.KeepAlive(dst)
		})
	}
}
