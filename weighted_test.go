package randomizer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"
	"runtime"
	"sync"
	"testing"
)

func assertWeightedMass(t testing.TB, weights []uint64, table *Weighted) {
	t.Helper()
	if len(table.thresholds) != len(weights) || len(table.aliases) != len(weights) {
		t.Fatalf("table lengths differ from weight count %d", len(weights))
	}
	mass := make([]big.Int, len(weights))
	for column, threshold := range table.thresholds {
		alias := table.aliases[column]
		if threshold > table.total || alias < 0 || alias >= len(weights) {
			t.Fatalf("invalid column %d: threshold=%d alias=%d total=%d", column, threshold, alias, table.total)
		}
		mass[column].Add(&mass[column], new(big.Int).SetUint64(threshold))
		mass[alias].Add(&mass[alias], new(big.Int).SetUint64(table.total-threshold))
	}
	count := new(big.Int).SetUint64(uint64(len(weights)))
	for i, weight := range weights {
		want := new(big.Int).Mul(new(big.Int).SetUint64(weight), count)
		if mass[i].Cmp(want) != 0 {
			t.Fatalf("outcome %d mass=%s want=%s for %v", i, &mass[i], want, weights)
		}
	}
}

func TestWeightedValidation(t *testing.T) {
	for _, weights := range [][]uint64{nil, {}, {0}, {0, 0, 0}} {
		if table, err := NewWeighted(weights); table != nil || !errors.Is(err, ErrInvalidWeights) {
			t.Fatalf("weights %v produced %v, %v", weights, table, err)
		}
	}
	for _, weights := range [][]uint64{{math.MaxUint64, 1}, {1 << 63, 1 << 63}, {math.MaxUint64, 0, math.MaxUint64}} {
		if table, err := NewWeighted(weights); table != nil || !errors.Is(err, ErrWeightOverflow) {
			t.Fatalf("weights %v produced %v, %v", weights, table, err)
		}
	}
	var empty Weighted
	if got := empty.Index(nil); got != -1 {
		t.Fatalf("zero table index = %d", got)
	}
	if got := (*Weighted)(nil).Index(nil); got != -1 {
		t.Fatalf("nil table index = %d", got)
	}
}

func TestWeightedExactMass(t *testing.T) {
	for _, weights := range [][]uint64{
		{1},
		{math.MaxUint64},
		{1, 1, 1},
		{0, 0, 1, 0},
		{2, 1, 1},
		{math.MaxUint64 - 1, 1},
		{math.MaxUint64 - 2, 0, 1, 1},
		{1 << 63, 1 << 62, 1<<62 - 1},
		{1, math.MaxUint64 - 1},
	} {
		table, err := NewWeighted(weights)
		if err != nil {
			t.Fatalf("weights %v: %v", weights, err)
		}
		assertWeightedMass(t, weights, table)
	}
	for size := 1; size <= 5; size++ {
		weights := make([]uint64, size)
		var visit func(int)
		visit = func(i int) {
			if i < len(weights) {
				for weight := range uint64(4) {
					weights[i] = weight
					visit(i + 1)
				}
				return
			}
			table, err := NewWeighted(weights)
			if errors.Is(err, ErrInvalidWeights) {
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			assertWeightedMass(t, weights, table)
		}
		visit(0)
	}
}

func TestWeightedExhaustiveDraws(t *testing.T) {
	for _, weights := range [][]uint64{{1}, {0, 1, 0}, {1, 1, 1}, {2, 1, 1}, {0, 7, 1, 0, 2}, {3, 0, 5, 1, 4, 0}} {
		table, err := NewWeighted(weights)
		if err != nil {
			t.Fatal(err)
		}
		counts := make([]uint64, len(weights))
		for column := range weights {
			for coin := range table.total {
				source := &samplingScript{words: []uint64{
					wordForChoice(uint64(column), uint64(len(weights))),
					wordForChoice(coin, table.total),
				}}
				index := table.Index(newGenerator(source))
				if index < 0 || index >= len(weights) {
					t.Fatalf("draw index = %d for %v", index, weights)
				}
				counts[index]++
			}
		}
		for i, weight := range weights {
			if want := uint64(len(weights)) * weight; counts[i] != want {
				t.Fatalf("weights %v index %d count=%d want=%d", weights, i, counts[i], want)
			}
		}
	}
}

func TestWeightedOwnsStorage(t *testing.T) {
	weights := []uint64{0, 1, 0}
	table, err := NewWeighted(weights)
	if err != nil {
		t.Fatal(err)
	}
	weights[0], weights[1], weights[2] = 9, 0, 9
	for range 100 {
		if got := table.Index(nil); got != 1 {
			t.Fatalf("input mutation changed draw to %d", got)
		}
	}
	assertWeightedMass(t, []uint64{0, 1, 0}, table)
}

func TestWeightedReproducible(t *testing.T) {
	table, err := NewWeighted([]uint64{1, 3, 5, 0, 7})
	if err != nil {
		t.Fatal(err)
	}
	first := NewSeeded([32]byte{42})
	second := NewSeeded([32]byte{42})
	for range 1000 {
		if a, b := table.Index(first), table.Index(second); a != b {
			t.Fatalf("matching seeded draws = %d and %d", a, b)
		}
	}
}

func TestWeightedForcedRejection(t *testing.T) {
	table, err := NewWeighted([]uint64{1, 2, 7})
	if err != nil {
		t.Fatal(err)
	}
	source := &samplingScript{words: []uint64{0, wordForChoice(0, 3), 0, math.MaxUint64}}
	if got := table.Index(newGenerator(source)); got != 2 || source.next != 4 {
		t.Fatalf("rejection draw gives %d after %d words", got, source.next)
	}
}

func TestWeightedSharedGenerator(t *testing.T) {
	table, err := NewWeighted([]uint64{0, 1, 3, 0, 9})
	if err != nil {
		t.Fatal(err)
	}
	g := NewSeeded([32]byte{17})
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			for range 1024 {
				if index := table.Index(g); index != 1 && index != 2 && index != 4 {
					t.Errorf("concurrent draw returned %d", index)
					return
				}
			}
		})
	}
	workers.Wait()
}

func TestWeightedAllocs(t *testing.T) {
	table, err := NewWeighted([]uint64{0, 1, 3, 7})
	if err != nil {
		t.Fatal(err)
	}
	if got := testing.AllocsPerRun(1000, func() { table.Index(nil) }); got != 0 {
		t.Fatalf("weighted draws allocate %v times", got)
	}
	single, err := NewWeighted([]uint64{7})
	if err != nil {
		t.Fatal(err)
	}
	if got := single.Index(newGenerator(&samplingScript{})); got != 0 {
		t.Fatalf("singleton draw = %d", got)
	}
}

func FuzzWeightedMass(f *testing.F) {
	f.Add([]byte{1, 2, 3})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 1})
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 256 {
			return
		}
		weights := make([]uint64, (len(data)+7)/8)
		for i := range weights {
			var encoded [8]byte
			copy(encoded[:], data[i*8:min((i+1)*8, len(data))])
			weights[i] = binary.LittleEndian.Uint64(encoded[:])
		}
		var sum big.Int
		for _, weight := range weights {
			sum.Add(&sum, new(big.Int).SetUint64(weight))
		}
		table, err := NewWeighted(weights)
		if sum.Sign() == 0 {
			if !errors.Is(err, ErrInvalidWeights) {
				t.Fatalf("zero total error = %v", err)
			}
		} else if sum.BitLen() > 64 {
			if !errors.Is(err, ErrWeightOverflow) {
				t.Fatalf("overflow error = %v", err)
			}
		} else {
			if err != nil {
				t.Fatal(err)
			}
			assertWeightedMass(t, weights, table)
		}
	})
}

func BenchmarkWeightedConstruction(b *testing.B) {
	for _, size := range []int{1, 16, 256, 4096} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			weights := make([]uint64, size)
			for i := range weights {
				weights[i] = uint64(i%7 + 1)
			}
			b.ReportAllocs()
			for b.Loop() {
				table, err := NewWeighted(weights)
				if err != nil {
					b.Fatal(err)
				}
				runtime.KeepAlive(table)
			}
		})
	}
}

func BenchmarkWeightedIndex(b *testing.B) {
	for _, size := range []int{1, 16, 256, 4096} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			weights := make([]uint64, size)
			for i := range weights {
				weights[i] = uint64(i % 7)
			}
			weights[0] = 1
			table, err := NewWeighted(weights)
			if err != nil {
				b.Fatal(err)
			}
			for _, mode := range []string{"default", "seeded", "secure"} {
				b.Run(mode, func(b *testing.B) {
					var g *Generator
					if mode == "seeded" {
						g = NewSeeded([32]byte{1})
					} else if mode == "secure" {
						g = NewSecure()
					}
					var sink int
					b.ReportAllocs()
					for b.Loop() {
						sink = table.Index(g)
					}
					runtime.KeepAlive(sink)
				})
			}
		})
	}
}
