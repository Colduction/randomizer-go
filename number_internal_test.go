package randomizer

import (
	"math"
	"math/big"
	"math/bits"
	"testing"
)

func acceptedBucketBounds(index, span uint64) (uint64, uint64) {
	divisor := new(big.Int).SetUint64(span)
	threshold := new(big.Int).SetUint64(-span % span)
	numerator := new(big.Int).Lsh(new(big.Int).SetUint64(index), 64)
	numerator.Add(numerator, threshold)
	var start, remainder big.Int
	start.QuoRem(numerator, divisor, &remainder)
	if remainder.Sign() != 0 {
		start.Add(&start, big.NewInt(1))
	}
	numerator.Lsh(new(big.Int).SetUint64(index+1), 64)
	var end big.Int
	end.QuoRem(numerator, divisor, &remainder)
	if remainder.Sign() != 0 {
		end.Add(&end, big.NewInt(1))
	}
	end.Sub(&end, big.NewInt(1))
	return start.Uint64(), end.Uint64()
}

func TestBoundedReducedWordUniformity(t *testing.T) {
	for _, width := range []uint{8, 16} {
		limit := uint64(1) << width
		var spans []uint64
		if width == 8 {
			for span := uint64(1); span < limit; span++ {
				spans = append(spans, span)
			}
		} else {
			spans = []uint64{1, 2, 3, 7, 10, 31, 63, 127, 255, 256, 257, 1000, 32767, 32768, 32769, 65535}
		}
		for _, span := range spans {
			counts := make([]uint64, span)
			threshold := limit % span
			for x := range limit {
				product := x * span
				if product&(limit-1) < threshold {
					continue
				}
				want := product >> width
				if got := boundedFrom(x<<(64-width), span, &samplingScript{}); got != want {
					t.Fatalf("width=%d span=%d x=%d: got %d want %d", width, span, x, got, want)
				}
				counts[want]++
			}
			for index, count := range counts {
				if want := limit / span; count != want {
					t.Fatalf("width=%d span=%d bucket=%d count=%d want=%d", width, span, index, count, want)
				}
			}
		}
	}
}

func TestBoundedFullWordPartitions(t *testing.T) {
	limit := new(big.Int).Lsh(big.NewInt(1), 64)
	for span := uint64(2); span <= 256; span++ {
		wantCount := new(big.Int).Quo(limit, new(big.Int).SetUint64(span))
		for index := range span {
			start, end := acceptedBucketBounds(index, span)
			count := new(big.Int).Sub(new(big.Int).SetUint64(end), new(big.Int).SetUint64(start))
			count.Add(count, big.NewInt(1))
			if count.Cmp(wantCount) != 0 {
				t.Fatalf("span=%d bucket=%d has mass %s want %s", span, index, count, wantCount)
			}
			for _, x := range []uint64{start, start + (end-start)/2, end} {
				if got := boundedFrom(x, span, &samplingScript{}); got != index {
					t.Fatalf("span=%d x=%d gives %d want %d", span, x, got, index)
				}
			}
			if start > 0 {
				x := start - 1
				hi, lo := bits.Mul64(x, span)
				source := &samplingScript{words: []uint64{math.MaxUint64}}
				got := boundedFrom(x, span, source)
				if lo < -span%span {
					if got != span-1 || source.next != 1 {
						t.Fatalf("span=%d x=%d did not reject: result=%d calls=%d", span, x, got, source.next)
					}
				} else if got != hi || source.next != 0 {
					t.Fatalf("span=%d x=%d unexpected rejection or output", span, x)
				}
			}
		}
	}
}

func TestBoundedExtremeSpans(t *testing.T) {
	for _, span := range []uint64{0, 1, 2, 3, 10, 255, 256, 257, 1<<32 - 1, 1 << 32, 1<<32 + 1, 1<<63 - 1, 1 << 63, 1<<63 + 1, math.MaxUint64} {
		source := &samplingScript{words: []uint64{0, math.MaxUint64}}
		got := boundedFrom(0, span, source)
		if span <= 1 || span&(span-1) == 0 {
			if got != 0 || source.next != 0 {
				t.Fatalf("span=%d returned %d with %d draws", span, got, source.next)
			}
		} else if got != span-1 || source.next != 2 {
			t.Fatalf("span=%d rejection result=%d calls=%d", span, got, source.next)
		}
		if span > 0 {
			if got := boundedFrom(math.MaxUint64, span, &samplingScript{}); got != span-1 {
				t.Fatalf("span=%d maximum word result=%d", span, got)
			}
		}
	}
	for _, span := range []uint64{0, 1} {
		if got := uniformUint64n(span, &samplingScript{}); got != 0 {
			t.Fatalf("constant span %d returned %d", span, got)
		}
	}
}

func TestFillBoundedJointPartitions(t *testing.T) {
	for _, span := range []uint64{3, 5, 7, 10} {
		product := uint64(1)
		for take := 1; take <= 3; take++ {
			product *= span
			for tuple := range product {
				want := make([]uint64, take)
				digits := tuple
				for i := len(want) - 1; i >= 0; i-- {
					want[i] = digits % span
					digits /= span
				}
				start, end := acceptedBucketBounds(tuple, product)
				for _, x := range []uint64{start, start + (end-start)/2, end} {
					dst := make([]uint64, take)
					source := &samplingScript{words: []uint64{x}}
					fillBounded(dst, 0, span, source)
					for i := range dst {
						if dst[i] != want[i] {
							t.Fatalf("span=%d x=%d tuple=%d gives %v want %v", span, x, tuple, dst, want)
						}
					}
					if source.next != 1 {
						t.Fatalf("span=%d tuple=%d consumed %d words", span, tuple, source.next)
					}
				}
			}
		}
	}
}

func TestFillBoundedPowerOfTwoJoint(t *testing.T) {
	for _, span := range []uint64{2, 4, 8, 16} {
		width := bits.TrailingZeros64(span)
		product := span * span * span
		for x := range product {
			var dst [3]uint64
			source := &samplingScript{words: []uint64{x}}
			fillBounded(dst[:], 11, span, source)
			for i := range dst {
				if want := uint64(11) + (x>>(i*width))&(span-1); dst[i] != want {
					t.Fatalf("span=%d x=%d position=%d gives %d want %d", span, x, i, dst[i], want)
				}
			}
			if source.next != 1 {
				t.Fatalf("power-of-two batch consumed %d words", source.next)
			}
		}
	}
}

func TestFillBoundedRejectionAndTail(t *testing.T) {
	for _, span := range []uint64{3, 10, 255} {
		count := int(newBatchSpec(span).k) + 3
		dst := make([]uint64, count)
		source := &samplingScript{words: []uint64{0, math.MaxUint64, 0, math.MaxUint64}}
		fillBounded(dst, 100, span, source)
		for i, value := range dst {
			if want := uint64(100) + span - 1; value != want {
				t.Fatalf("span=%d position=%d gives %d want %d", span, i, value, want)
			}
		}
		if source.next != 4 {
			t.Fatalf("span=%d batch/tail consumed %d words", span, source.next)
		}
	}
	var dst [4]uint64
	fillBounded(dst[:], math.MaxUint64, 1, &samplingScript{})
	for _, value := range dst {
		if value != math.MaxUint64 {
			t.Fatalf("constant full-width output = %d", value)
		}
	}
	for _, span := range []uint64{1<<63 - 1, 1<<63 + 1, math.MaxUint64} {
		source := &samplingScript{words: []uint64{0, math.MaxUint64, 0, math.MaxUint64, 0, math.MaxUint64, 0, math.MaxUint64}}
		fillBounded(dst[:], 0, span, source)
		for _, value := range dst {
			if value != span-1 {
				t.Fatalf("extreme span=%d result=%d", span, value)
			}
		}
		if source.next != 8 {
			t.Fatalf("extreme span=%d consumed %d words", span, source.next)
		}
	}
	fillBounded([]uint64(nil), 0, 10, &samplingScript{})
}
