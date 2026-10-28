package randomizer

import (
	"bytes"
	"math/bits"
	"testing"
)

type wordSequence struct {
	t      *testing.T
	values []uint64
	i      int
}

func (s *wordSequence) Uint64() uint64 {
	if s.i == len(s.values) {
		s.t.Fatal("word sampler exhausted scripted source")
	}
	x := s.values[s.i]
	s.i++
	return x
}

func wordForRank(rank, bound uint64) uint64 {
	x, _ := bits.Div64(rank, 1<<63, bound)
	return x
}

func TestWordWeightedTransitionsExhaustive(t *testing.T) {
	characters := [...]byte{0, 127, 255}
	for a := 1; a <= 4; a++ {
		for b := 1; b <= 4; b++ {
			for c := 1; c <= 4; c++ {
				weights := [...]int{a, b, c}
				var dictionary []byte
				for i, weight := range weights {
					dictionary = append(dictionary, bytes.Repeat([]byte{characters[i]}, weight)...)
				}
				for previous, excluded := range weights {
					var firstRank uint64
					for _, weight := range weights[:previous] {
						firstRank += uint64(weight)
					}
					bound := len(dictionary) - excluded
					var counts [256]int
					for rank := range bound {
						source := &wordSequence{t: t, values: []uint64{
							wordForRank(firstRank, uint64(len(dictionary))),
							wordForRank(uint64(rank), uint64(bound)),
						}}
						w := WordGenerator{provider: FromSource(source).provider, mode: NoAdjacentRepeats}
						out := w.CustomBytes(dictionary, 2)
						if out[0] != characters[previous] || out[1] == out[0] || source.i != 2 {
							t.Fatalf("weights=%v previous=%d rank=%d output=%x source calls=%d", weights, previous, rank, out, source.i)
						}
						counts[out[1]]++
					}
					for i, weight := range weights {
						want := weight
						if i == previous {
							want = 0
						}
						if counts[characters[i]] != want {
							t.Fatalf("weights=%v previous=%d byte=%d count=%d want=%d", weights, previous, characters[i], counts[characters[i]], want)
						}
					}
				}
			}
		}
	}
}

func TestWordIndependentTuplesExhaustive(t *testing.T) {
	for n := 2; n <= 16; n++ {
		dictionary := make([]byte, n)
		for i := range dictionary {
			dictionary[i] = byte(i)
		}
		bound := n * n * n
		seen := make(map[string]bool, bound)
		for rank := range bound {
			x := uint64(rank)
			if n&(n-1) != 0 {
				x = wordForRank(uint64(rank), uint64(bound))
			}
			source := &wordSequence{t: t, values: []uint64{x}}
			w := WordGenerator{provider: FromSource(source).provider}
			out := w.CustomBytes(dictionary, 3)
			if seen[string(out)] || source.i != 1 {
				t.Fatalf("base=%d rank=%d duplicate tuple=%x or source calls=%d", n, rank, out, source.i)
			}
			seen[string(out)] = true
		}
		if len(seen) != bound {
			t.Fatalf("base=%d tuples=%d want=%d", n, len(seen), bound)
		}
	}
}

func TestWordForcedRejection(t *testing.T) {
	for _, tc := range []struct {
		dictionary string
		mode       WordMode
		values     []uint64
		want       string
	}{
		{"ABC", IndependentSymbols, []uint64{0, ^uint64(0)}, "CC"},
		{"ABCD", NoAdjacentRepeats, []uint64{^uint64(0), 0, ^uint64(0)}, "DCD"},
		{"AABC", NoAdjacentRepeats, []uint64{wordForRank(2, 4), 0, ^uint64(0)}, "BC"},
	} {
		source := &wordSequence{t: t, values: tc.values}
		w := WordGenerator{provider: FromSource(source).provider, mode: tc.mode}
		if got := w.Custom(tc.dictionary, len(tc.want)); got != tc.want || source.i != len(tc.values) {
			t.Fatalf("mode=%v dictionary=%q output=%q calls=%d, want %q/%d", tc.mode, tc.dictionary, got, source.i, tc.want, len(tc.values))
		}
	}
}

func TestWordExtremeSkewDoesNotRetry(t *testing.T) {
	dictionary := append(bytes.Repeat([]byte{'A'}, 100000), 'B')
	source := &wordSequence{t: t, values: []uint64{^uint64(0)}}
	w := WordGenerator{provider: FromSource(source).provider, mode: NoAdjacentRepeats}
	out := w.CustomBytes(dictionary, 4096)
	for i, c := range out {
		want := byte('B')
		if i%2 != 0 {
			want = 'A'
		}
		if c != want {
			t.Fatalf("byte %d = %q, want %q", i, c, want)
		}
	}
	if source.i != 1 {
		t.Fatalf("source calls=%d, want 1", source.i)
	}
}

func TestWordPowerOfTwoUsesAllLanes(t *testing.T) {
	source := &wordSequence{t: t, values: []uint64{0xfedcba9876543210}}
	w := WordGenerator{provider: FromSource(source).provider}
	if got := w.String(HexLowerAlphabet, 16); got != "0123456789abcdef" || source.i != 1 {
		t.Fatalf("hex lanes=%q calls=%d", got, source.i)
	}
}

func TestWordConstantDictionaryNeedsNoEntropy(t *testing.T) {
	for _, mode := range []WordMode{IndependentSymbols, NoAdjacentRepeats} {
		for _, dictionary := range []string{"A", "AA", string(bytes.Repeat([]byte{'A'}, 300))} {
			source := &wordSequence{t: t}
			w := WordGenerator{provider: FromSource(source).provider, mode: mode}
			if got := w.Custom(dictionary, 1); got != "A" {
				t.Fatalf("mode=%v size=%d constant word=%q", mode, len(dictionary), got)
			}
		}
	}
}
