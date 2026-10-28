package randomizer

import "errors"

// ErrInvalidSample indicates a negative population or a destination longer than the population.
var ErrInvalidSample = errors.New("invalid sample size")

// Shuffle permutes the slice in place, giving every permutation equal probability.
// It uses the default source and constant auxiliary space without heap allocations.
// Concurrent callers must use separate slice storage.
func Shuffle[T any](values []T) {
	(*Generator)(nil).Shuffle(values)
}

// Shuffle permutes the slice in place, giving every permutation equal probability.
// It uses constant auxiliary space without heap allocations.
// Concurrent callers must use separate slice storage.
// A nil receiver uses the default source.
func (g *Generator) Shuffle[T any](values []T) {
	if len(values) < 2 {
		return
	}
	p := openProvider(g.source())
	defer p.close()
	shuffleValues(values, p.provider)
}

func shuffleValues[T any](values []T, p provider) {
	switch p.(type) {
	case fastProvider, cryptoProvider, *uint64Stream, *readerStream:
		shuffleRecycled(values, p)
	default:
		shuffleDirect(values, p)
	}
}

func shuffleDirect[T any](values []T, p provider) {
	for i := len(values) - 1; i > 0; i-- {
		j := int(boundedFrom(p.Uint64(), uint64(i+1), p))
		values[i], values[j] = values[j], values[i]
	}
}

// SampleIndices fills the destination with distinct indices from [0, population),
// giving every ordered sample equal probability and using the default source.
// It returns [ErrInvalidSample] without changing the destination or consuming
// randomness when the population is negative or smaller than the destination.
// Auxiliary storage is proportional to the sample size, independent of the population.
func SampleIndices(dst []int, population int) error {
	return (*Generator)(nil).SampleIndices(dst, population)
}

// SampleIndices fills the destination with distinct indices from [0, population),
// giving every ordered sample equal probability.
// It returns [ErrInvalidSample] without changing the destination or consuming
// randomness when the population is negative or smaller than the destination.
// Empty and full samples use constant space without heap allocations. Other samples
// use storage proportional to the sample size, independent of the population.
// A nil receiver uses the default source. Concurrent callers must use separate
// destination storage. A source panic may leave a partially filled destination.
func (g *Generator) SampleIndices(dst []int, population int) error {
	if population < 0 || len(dst) > population {
		return ErrInvalidSample
	}
	if len(dst) == 0 {
		return nil
	}
	if population == 1 {
		dst[0] = 0
		return nil
	}
	p := openProvider(g.source())
	defer p.close()
	if len(dst) == population {
		for i := range dst {
			dst[i] = i
		}
		shuffleValues(dst, p.provider)
	} else if len(dst) <= 16 {
		sampleFloyd(dst, population, p.provider)
	} else {
		sampleSparse(dst, population, p.provider)
	}
	return nil
}

func sampleFloyd(dst []int, population int, p provider) {
	for i := range dst {
		j := population - len(dst) + i
		t := int(boundedFrom(p.Uint64(), uint64(j+1), p))
		for previous := range i {
			if dst[previous] == t {
				dst[previous] = j
				break
			}
		}
		dst[i] = t
	}
}

func sampleSparse(dst []int, population int, p provider) {
	remap := make(map[int]int, len(dst))
	for i := range dst {
		remaining := population - i
		j := int(boundedFrom(p.Uint64(), uint64(remaining), p))
		value, ok := remap[j]
		if !ok {
			value = j
		}
		last := remaining - 1
		replacement, ok := remap[last]
		if !ok {
			replacement = last
		}
		if j != last {
			remap[j] = replacement
		}
		delete(remap, last)
		dst[i] = value
	}
}
