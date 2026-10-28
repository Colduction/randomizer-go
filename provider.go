package randomizer

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"io"
	"math/rand/v2"
	"sync"
)

type provider interface {
	Read([]byte) (int, error)
	Uint32() uint32
	Uint64() uint64
}

type fastProvider struct{}

func (fastProvider) Read(out []byte) (int, error) {
	fillFastBytes(out)
	return len(out), nil
}

func (fastProvider) Uint32() uint32 { return uint32(rand.Uint64() >> 32) }

func (fastProvider) Uint64() uint64 { return rand.Uint64() }

type cryptoProvider struct{}

func (cryptoProvider) Read(out []byte) (int, error) { return cryptorand.Read(out) }

func (cryptoProvider) Uint32() uint32 {
	var word [4]byte
	_, _ = cryptorand.Read(word[:])
	return binary.LittleEndian.Uint32(word[:4])
}

func (cryptoProvider) Uint64() uint64 {
	var word [8]byte
	_, _ = cryptorand.Read(word[:])
	return binary.LittleEndian.Uint64(word[:])
}

type uint64Stream struct {
	source rand.Source
}

func (s *uint64Stream) Read(out []byte) (int, error) {
	fillRandomBytes(out, s)
	return len(out), nil
}

func (s *uint64Stream) Uint32() uint32 { return uint32(s.source.Uint64() >> 32) }

func (s *uint64Stream) Uint64() uint64 { return s.source.Uint64() }

type uint64Provider struct {
	stream uint64Stream
	mu     sync.Mutex
}

func newUint64Provider(source rand.Source) *uint64Provider {
	return &uint64Provider{stream: uint64Stream{source: source}}
}

func (p *uint64Provider) Read(out []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stream.Read(out)
}

func (p *uint64Provider) Uint32() uint32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stream.Uint32()
}

func (p *uint64Provider) Uint64() uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stream.Uint64()
}

type readerStream struct {
	reader io.Reader
	word   [8]byte
}

func (s *readerStream) Read(out []byte) (int, error) { return io.ReadFull(s.reader, out) }

func (s *readerStream) Uint32() uint32 {
	if _, err := s.Read(s.word[:4]); err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint32(s.word[:4])
}

func (s *readerStream) Uint64() uint64 {
	if _, err := s.Read(s.word[:]); err != nil {
		panic(err)
	}
	return binary.LittleEndian.Uint64(s.word[:])
}

type readerProvider struct {
	stream readerStream
	mu     sync.Mutex
}

func newReaderProvider(reader io.Reader) *readerProvider {
	return &readerProvider{stream: readerStream{reader: reader}}
}

func (p *readerProvider) Read(out []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stream.Read(out)
}

func (p *readerProvider) Uint32() uint32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stream.Uint32()
}

func (p *readerProvider) Uint64() uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stream.Uint64()
}

type providerSession struct {
	provider
	mu *sync.Mutex
}

func openProvider(p provider) providerSession {
	if p == nil {
		return providerSession{provider: fastProvider{}}
	}
	return openSharedProvider(p)
}

func openSharedProvider(p provider) providerSession {
	switch p := p.(type) {
	case *uint64Provider:
		p.mu.Lock()
		return providerSession{provider: &p.stream, mu: &p.mu}
	case *readerProvider:
		p.mu.Lock()
		return providerSession{provider: &p.stream, mu: &p.mu}
	default:
		return providerSession{provider: p}
	}
}

func (s providerSession) close() {
	if s.mu != nil {
		s.mu.Unlock()
	}
}

func providerOrDefault(p provider) provider {
	if p == nil {
		return fastProvider{}
	}
	return p
}

func putTail(out []byte, x uint64) {
	for i := range out {
		out[i] = byte(x)
		x >>= 8
	}
}

func fillFastBytes(out []byte) {
	i := 0
	for ; i+8 <= len(out); i += 8 {
		binary.LittleEndian.PutUint64(out[i:i+8], rand.Uint64())
	}
	if i < len(out) {
		putTail(out[i:], rand.Uint64())
	}
}

func fillRandomBytes(out []byte, p provider) {
	switch p.(type) {
	case fastProvider:
		fillFastBytes(out)
		return
	case cryptoProvider:
		_, _ = cryptorand.Read(out)
		return
	}
	i := 0
	for ; i+8 <= len(out); i += 8 {
		binary.LittleEndian.PutUint64(out[i:i+8], p.Uint64())
	}
	if i < len(out) {
		putTail(out[i:], p.Uint64())
	}
}

// Read fills the destination using the package's fast random source.
// Use [Generator.Read] for a selected source or [NewSecure] for secrets.
func Read(out []byte) (int, error) {
	fillFastBytes(out)
	return len(out), nil
}
