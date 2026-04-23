package randomizer

import (
	"hash/maphash"
	"sync"
	"sync/atomic"
)

// hashPool pairs a sync.Pool of maphash.Hash objects (for callers that need
// one via Get/Put) with an atomic SplitMix64 counter used as the package's
// primary lock-free PRNG.
type hashPool struct {
	pool  sync.Pool
	state atomic.Uint64
}

const splitMixGamma uint64 = 0x9e3779b97f4a7c15

func splitMix64(x uint64) uint64 {
	z := x
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// NewHashPool creates a new hashPool. Any positive size enables the pool;
// zero or negative returns nil. Hash objects are allocated on demand by
// sync.Pool and recycled automatically by the GC.
func NewHashPool(size int) *hashPool {
	if size <= 0 {
		return nil
	}
	p := &hashPool{
		pool: sync.Pool{
			New: func() any {
				h := new(maphash.Hash)
				h.SetSeed(maphash.MakeSeed())
				return h
			},
		},
	}
	seed := maphash.Bytes(maphash.MakeSeed(), nil)
	if seed == 0 {
		seed = splitMixGamma
	}
	p.state.Store(seed)
	return p
}

// Get retrieves a maphash.Hash from the pool. The caller must call Put to
// return it after use.
func (p *hashPool) Get() *maphash.Hash {
	if p == nil {
		h := new(maphash.Hash)
		h.SetSeed(maphash.MakeSeed())
		return h
	}
	return p.pool.Get().(*maphash.Hash)
}

// Put returns a maphash.Hash to the pool for reuse.
func (p *hashPool) Put(h *maphash.Hash) {
	if p == nil || h == nil {
		return
	}
	h.Reset()
	p.pool.Put(h)
}

func (p *hashPool) next64() uint64 {
	if p == nil {
		return splitMix64(maphash.Bytes(maphash.MakeSeed(), nil) + splitMixGamma)
	}
	return splitMix64(p.state.Add(splitMixGamma))
}

// Sum appends 8 random bytes to b and returns the extended slice.
func (p *hashPool) Sum(b []byte) []byte {
	x := p.next64()
	return append(b,
		byte(x>>0),
		byte(x>>8),
		byte(x>>16),
		byte(x>>24),
		byte(x>>32),
		byte(x>>40),
		byte(x>>48),
		byte(x>>56))
}

// Sum32 generates a random 32-bit number using the hashPool.
func (p *hashPool) Sum32() uint32 {
	return uint32(p.next64() >> 32)
}

// Sum64 generates a random 64-bit number using the hashPool.
func (p *hashPool) Sum64() uint64 {
	return p.next64()
}

// DefaultHashPool is a globally accessible hashPool with a preallocated size.
var DefaultHashPool = NewHashPool(64)
