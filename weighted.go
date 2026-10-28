package randomizer

import (
	"errors"
	"math/bits"
)

var (
	// ErrInvalidWeights indicates empty weights or weights whose sum is zero.
	ErrInvalidWeights = errors.New("weights must contain positive mass")

	// ErrWeightOverflow indicates that the total weight exceeds the unsigned 64-bit range.
	ErrWeightOverflow = errors.New("total weight exceeds uint64 range")
)

// Weighted is an immutable discrete distribution whose outcome probabilities
// are proportional to nonnegative integer weights.
// Construction takes linear time and storage; draws take expected constant time
// and allocate no storage. It is safe for concurrent draws when the supplied
// [Generator] supports concurrent use. The zero value has no outcomes.
type Weighted struct {
	thresholds []uint64
	aliases    []int
	total      uint64
}

// NewWeighted constructs an exact integer alias table without retaining the input.
// Zero individual weights are valid and their outcomes are never selected.
// Empty input and an all-zero total return [ErrInvalidWeights].
// A total exceeding the unsigned 64-bit range returns [ErrWeightOverflow].
// Every distribution whose total fits that range is supported, including weights
// whose products with the outcome count exceed the unsigned 64-bit range.
func NewWeighted(weights []uint64) (*Weighted, error) {
	var total uint64
	for _, weight := range weights {
		var carry uint64
		total, carry = bits.Add64(total, weight, 0)
		if carry != 0 {
			return nil, ErrWeightOverflow
		}
	}
	if total == 0 {
		return nil, ErrInvalidWeights
	}
	thresholds := make([]uint64, len(weights))
	aliases := make([]int, len(weights))
	high := make([]uint64, len(weights))
	work := make([]int, len(weights))
	var small int
	large := len(work)
	for i, weight := range weights {
		high[i], thresholds[i] = bits.Mul64(uint64(len(weights)), weight)
		aliases[i] = i
		if high[i] == 0 && thresholds[i] < total {
			work[small] = i
			small++
		} else {
			large--
			work[large] = i
		}
	}
	for small > 0 && large < len(work) {
		small--
		s := work[small]
		l := work[large]
		large++
		aliases[s] = l
		var borrow uint64
		thresholds[l], borrow = bits.Sub64(thresholds[l], total-thresholds[s], 0)
		high[l], _ = bits.Sub64(high[l], 0, borrow)
		if high[l] == 0 && thresholds[l] < total {
			work[small] = l
			small++
		} else {
			large--
			work[large] = l
		}
	}
	for small > 0 {
		small--
		thresholds[work[small]] = total
	}
	for large < len(work) {
		thresholds[work[large]] = total
		large++
	}
	return &Weighted{thresholds: thresholds, aliases: aliases, total: total}, nil
}

// Index returns an outcome index using the supplied generator, or -1 when the
// receiver is nil or has no outcomes. A nil generator selects the default source.
// It draws independent bounded column and threshold values; no floating-point
// rounding can erase a positive outcome probability.
func (w *Weighted) Index(g *Generator) int {
	if w == nil || len(w.thresholds) == 0 {
		return -1
	}
	if len(w.thresholds) == 1 {
		return 0
	}
	p := openProvider(g.source())
	defer p.close()
	column := int(uniformUint64n(uint64(len(w.thresholds)), p.provider))
	threshold := w.thresholds[column]
	if threshold == w.total {
		return column
	}
	if threshold != 0 && uniformUint64n(w.total, p.provider) < threshold {
		return column
	}
	return w.aliases[column]
}
