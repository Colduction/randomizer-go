package randomizer

import (
	"fmt"
	"math/big"
	"math/bits"
	"math/rand/v2"
	"runtime"
	"slices"
	"testing"
)

func entropyFill(g *Generator, dst []uint64, bound uint64, batched bool) {
	if bound <= 1 {
		clear(dst)
		return
	}
	s := openProvider(g.source())
	defer s.close()
	r := entropyRecycler{span: 1}
	if !batched {
		quotient, threshold := bits.Div64(1, 0, bound)
		for i := range dst {
			dst[i] = r.uniform(bound, quotient, threshold, s.provider)
		}
		return
	}
	spec := newBatchSpec(bound)
	for i := 0; i < len(dst); {
		take := min(int(spec.k), len(dst)-i)
		product := uint64(1)
		for range take {
			product *= bound
		}
		quotient, threshold := bits.Div64(1, 0, product)
		for {
			original := r.pop(s.provider)
			x := original
			for j := range take {
				dst[i+j], x = bits.Mul64(x, bound)
			}
			leftover, tail := bits.Mul64(quotient, x)
			if tail >= threshold {
				r.merge(leftover, quotient)
				break
			}
			used, _ := bits.Mul64(original, product*quotient)
			r.merge(original-used, threshold)
		}
		i += take
	}
}

func entropyShuffle(g *Generator, values []int) {
	if len(values) < 2 {
		return
	}
	s := openProvider(g.source())
	defer s.close()
	shuffleRecycled(values, s.provider)
}

func entropyDirectShuffle(g *Generator, values []int) {
	s := openProvider(g.source())
	defer s.close()
	shuffleDirect(values, s.provider)
}

type entropyCountProvider struct {
	provider
	calls uint64
}

func (p *entropyCountProvider) Uint64() uint64 {
	p.calls++
	return p.provider.Uint64()
}

func TestEntropyRecyclerReducedUniform(t *testing.T) {
	for width := uint(2); width <= 8; width++ {
		base := uint64(1) << width
		mask := base - 1
		for bound := uint64(2); bound < base; bound++ {
			quotient, threshold := base/bound, base%bound
			accepted := make([]bool, bound*quotient)
			rejected := make([]bool, threshold)
			for x := range base {
				out, low := x*bound>>width, x*bound&mask
				leftover, tail := quotient*low>>width, quotient*low&mask
				if tail >= threshold {
					index := out*quotient + leftover
					if index >= uint64(len(accepted)) || accepted[index] {
						t.Fatalf("width %d bound %d repeats or exceeds accepted pair %d", width, bound, index)
					}
					accepted[index] = true
				} else {
					leftover = x - (x * (bound * quotient) >> width)
					if leftover >= threshold || rejected[leftover] {
						t.Fatalf("width %d bound %d repeats or exceeds rejection state %d", width, bound, leftover)
					}
					rejected[leftover] = true
				}
			}
			for _, seen := range append(accepted, rejected...) {
				if !seen {
					t.Fatalf("width %d bound %d omits an accepted pair or rejection state", width, bound)
				}
			}
		}
	}
}

func TestEntropyRecyclerReducedBatch(t *testing.T) {
	for _, bounds := range [][]uint64{{2, 3}, {3, 5}, {2, 3, 7}, {3, 3, 3}, {2, 2, 2, 2}} {
		for width := uint(6); width <= 8; width++ {
			base := uint64(1) << width
			mask := base - 1
			product := uint64(1)
			for _, bound := range bounds {
				product *= bound
			}
			quotient, threshold := base/product, base%product
			accepted := make([]bool, product*quotient)
			rejected := make([]bool, threshold)
			for original := range base {
				x, index := original, uint64(0)
				for _, bound := range bounds {
					out := x * bound >> width
					x = x * bound & mask
					index = index*bound + out
				}
				leftover, tail := quotient*x>>width, quotient*x&mask
				if tail >= threshold {
					index = index*quotient + leftover
					if index >= uint64(len(accepted)) || accepted[index] {
						t.Fatalf("width %d bounds %v repeat accepted tuple %d", width, bounds, index)
					}
					accepted[index] = true
				} else {
					leftover = original - (original * (product * quotient) >> width)
					if leftover >= threshold || rejected[leftover] {
						t.Fatalf("width %d bounds %v repeat rejection state %d", width, bounds, leftover)
					}
					rejected[leftover] = true
				}
			}
			for _, seen := range append(accepted, rejected...) {
				if !seen {
					t.Fatalf("width %d bounds %v omit a tuple or rejection state", width, bounds)
				}
			}
		}
	}
}

func TestEntropyRecyclerReducedMerge(t *testing.T) {
	for width := uint(2); width <= 5; width++ {
		base := uint64(1) << width
		mask := base - 1
		for span := uint64(1); span < base; span++ {
			for incomingSpan := uint64(1); incomingSpan < base; incomingSpan++ {
				product := span * incomingSpan
				spanHi, spanLo := product>>width, product&mask
				seen := make([]bool, product)
				for value := uint64(0); value < span; value++ {
					for incoming := uint64(0); incoming < incomingSpan; incoming++ {
						merged := value + incoming*span
						hi, lo := merged>>width, merged&mask
						var index uint64
						if hi == spanHi {
							if lo >= spanLo {
								t.Fatalf("partial state exceeds its range at width %d", width)
							}
							index = spanHi*base + lo
						} else {
							if hi >= spanHi {
								t.Fatalf("full-word state exceeds its range at width %d", width)
							}
							index = hi*base + lo
						}
						if seen[index] {
							t.Fatalf("merge repeats a state at width %d", width)
						}
						seen[index] = true
					}
				}
				for _, present := range seen {
					if !present {
						t.Fatalf("merge omits a state at width %d", width)
					}
				}
			}
		}
	}
}

func TestEntropyRecyclerMerge64(t *testing.T) {
	spans := []uint64{1, 2, 3, 1<<32 - 1, 1 << 32, 1<<32 + 1, 1 << 63, 1<<63 + 1, ^uint64(0)}
	for _, span := range spans {
		for _, incomingSpan := range spans {
			for _, value := range []uint64{0, span / 2, span - 1} {
				for _, incoming := range []uint64{0, incomingSpan / 2, incomingSpan - 1} {
					r := entropyRecycler{value: value, span: span}
					r.merge(incoming, incomingSpan)
					merged := new(big.Int).Mul(new(big.Int).SetUint64(incoming), new(big.Int).SetUint64(span))
					merged.Add(merged, new(big.Int).SetUint64(value))
					product := new(big.Int).Mul(new(big.Int).SetUint64(span), new(big.Int).SetUint64(incomingSpan))
					wantValue := new(big.Int).Rsh(new(big.Int).Set(merged), 64).Uint64()
					wantSpan := new(big.Int).Rsh(new(big.Int).Set(product), 64).Uint64()
					wantPending := wantValue != wantSpan
					if !wantPending {
						wantValue, wantSpan = merged.Uint64(), product.Uint64()
					}
					if r.value != wantValue || r.span != wantSpan || r.pending != wantPending || r.span == 0 || r.value >= r.span {
						t.Fatalf("merge (%d,%d), (%d,%d) = %+v; want (%d,%d), pending %t", value, span, incoming, incomingSpan, r, wantValue, wantSpan, wantPending)
					}
					if r.pending && r.word != merged.Uint64() {
						t.Fatal("pushed-back word differs from exact low word")
					}
				}
			}
		}
	}
}

func TestEntropyRecyclerRejectionAndPushback(t *testing.T) {
	source := &samplingScript{words: []uint64{0, ^uint64(0), 0}}
	r := entropyRecycler{span: 1}
	quotient, threshold := bits.Div64(1, 0, 3)
	if out := r.uniform(3, quotient, threshold, source); out != 2 || source.next != 2 {
		t.Fatalf("forced rejection returns %d after %d source words", out, source.next)
	}
	quotient, threshold = bits.Div64(1, 0, 2)
	if out := r.uniform(2, quotient, threshold, source); out != 0 || !r.pending || source.next != 3 {
		t.Fatalf("second output %d does not push back a word: %+v, calls %d", out, r, source.next)
	}
	if out := r.uniform(2, quotient, threshold, source); out > 1 || source.next != 3 {
		t.Fatalf("pushback returns %d after %d source words", out, source.next)
	}
}

func TestEntropyRecyclerOperations(t *testing.T) {
	for _, bound := range []uint64{0, 1, 2, 3, 10, 1000, 1<<32 + 1, 1<<63 + 1, ^uint64(0)} {
		for _, batched := range []bool{false, true} {
			first, second := NewSeeded([32]byte{8}), NewSeeded([32]byte{8})
			for _, count := range []int{0, 1, 17, 256} {
				a, b := make([]uint64, count), make([]uint64, count)
				entropyFill(first, a, bound, batched)
				entropyFill(second, b, bound, batched)
				if !slices.Equal(a, b) {
					t.Fatalf("bound %d batch %t is not reproducible", bound, batched)
				}
				for _, value := range a {
					if bound > 0 && value >= bound || bound == 0 && value != 0 {
						t.Fatalf("bound %d batch %t returns %d", bound, batched, value)
					}
				}
			}
		}
	}
	for _, count := range []int{0, 1, 2, 17, 256, 4096} {
		values := make([]int, count)
		for i := range values {
			values[i] = i
		}
		entropyShuffle(NewSeeded([32]byte{9}), values)
		slices.Sort(values)
		for i, value := range values {
			if value != i {
				t.Fatalf("shuffle length %d loses element %d", count, i)
			}
		}
	}
	g := newGenerator(&samplingScript{})
	var values [3]uint64
	entropyFill(g, values[:], 1, false)
	entropyFill(g, values[:], 1, true)
	entropyShuffle(g, nil)
}

func TestEntropyRecyclerReducesSourceWords(t *testing.T) {
	for _, batched := range []bool{false, true} {
		source := &entropyCountProvider{provider: FromSource(rand.NewPCG(1, 2)).provider}
		g := newGenerator(source)
		var values [4096]uint64
		entropyFill(g, values[:], 1000, batched)
		if source.calls >= uint64(len(values)) {
			t.Fatalf("batch %t consumes %d source words for %d outputs", batched, source.calls, len(values))
		}
	}
}

func BenchmarkEntropyUniform(b *testing.B) {
	for _, mode := range []string{"fast", "secure"} {
		b.Run(mode, func(b *testing.B) {
			for _, bound := range []uint64{10, 1000, 1000003, 1<<63 + 1} {
				b.Run(fmt.Sprint(bound), func(b *testing.B) {
					for _, algorithm := range []string{"production", "recycle43", "recycle44"} {
						b.Run(algorithm, func(b *testing.B) {
							g := New()
							if mode == "secure" {
								g = NewSecure()
							}
							values := make([]uint64, 1024)
							counted := &entropyCountProvider{provider: FromSource(rand.NewPCG(1, 2)).provider}
							counter := newGenerator(counted)
							for range 32 {
								switch algorithm {
								case "production":
									counter.FillUintInterval(values, uint64(0), bound)
								case "recycle43":
									entropyFill(counter, values, bound, false)
								case "recycle44":
									entropyFill(counter, values, bound, true)
								}
							}
							b.ReportAllocs()
							for b.Loop() {
								switch algorithm {
								case "production":
									g.FillUintInterval(values, uint64(0), bound)
								case "recycle43":
									entropyFill(g, values, bound, false)
								case "recycle44":
									entropyFill(g, values, bound, true)
								}
							}
							b.ReportMetric(float64(counted.calls)/32, "source-words/op")
							runtime.KeepAlive(values)
						})
					}
				})
			}
		})
	}
}

func BenchmarkEntropyShuffle(b *testing.B) {
	for _, mode := range []string{"fast", "secure"} {
		b.Run(mode, func(b *testing.B) {
			for _, count := range []int{16, 256, 4096} {
				b.Run(fmt.Sprint(count), func(b *testing.B) {
					for _, algorithm := range []string{"direct", "recycle43"} {
						b.Run(algorithm, func(b *testing.B) {
							g := New()
							if mode == "secure" {
								g = NewSecure()
							}
							values := make([]int, count)
							for i := range values {
								values[i] = i
							}
							counted := &entropyCountProvider{provider: FromSource(rand.NewPCG(1, 2)).provider}
							counter := newGenerator(counted)
							for range 32 {
								if algorithm == "direct" {
									entropyDirectShuffle(counter, values)
								} else {
									entropyShuffle(counter, values)
								}
							}
							b.ReportAllocs()
							for b.Loop() {
								if algorithm == "direct" {
									entropyDirectShuffle(g, values)
								} else {
									entropyShuffle(g, values)
								}
							}
							b.ReportMetric(float64(counted.calls)/32, "source-words/op")
							runtime.KeepAlive(values)
						})
					}
				})
			}
		})
	}
}
