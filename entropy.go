package randomizer

import "math/bits"

type entropyRecycler struct {
	value   uint64
	span    uint64
	word    uint64
	pending bool
}

func (r *entropyRecycler) pop(p provider) uint64 {
	if r.pending {
		r.pending = false
		return r.word
	}
	return p.Uint64()
}

func (r *entropyRecycler) merge(value, span uint64) {
	hi, lo := bits.Mul64(value, r.span)
	lo, carry := bits.Add64(lo, r.value, 0)
	hi += carry
	spanHi, spanLo := bits.Mul64(r.span, span)
	if hi == spanHi {
		r.value, r.span = lo, spanLo
	} else {
		r.value, r.span = hi, spanHi
		r.word, r.pending = lo, true
	}
}

func (r *entropyRecycler) uniform(bound, quotient, threshold uint64, p provider) uint64 {
	for {
		x := r.pop(p)
		out, low := bits.Mul64(x, bound)
		leftover, tail := bits.Mul64(quotient, low)
		if tail >= threshold {
			r.merge(leftover, quotient)
			return out
		}
		used, _ := bits.Mul64(x, bound*quotient)
		r.merge(x-used, threshold)
	}
}

func shuffleRecycled[T any](values []T, p provider) {
	r := entropyRecycler{span: 1}
	for i := len(values) - 1; i > 0; i-- {
		bound := uint64(i + 1)
		quotient, threshold := bits.Div64(1, 0, bound)
		j := int(r.uniform(bound, quotient, threshold, p))
		values[i], values[j] = values[j], values[i]
	}
}
