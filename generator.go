package randomizer

import (
	"io"
	"math/rand/v2"
)

// Generator binds number, word, network, and telecom generation to one source.
// Its zero value uses the fast package source. Instances support shared concurrent
// use, provided callers do not mutate the exported handles or share output storage.
// Identical seeds and sequential calls reproduce results within one version;
// concurrent ordering and sequences across versions are unspecified.
type Generator struct {

	// Network generates network values using this generator's source.
	Network NetworkGenerator

	// Telecom generates telecom identifiers using this generator's source.
	Telecom TelecomGenerator

	provider provider
	// Word generates byte strings using this generator's source.
	Word WordGenerator
}

// New returns a generator using the package's fast random source.
// Use [NewSecure] for secrets or [NewSeeded] for reproducible sequential calls.
func New() *Generator {
	return newGenerator(nil)
}

func newGenerator(provider provider) *Generator {
	return &Generator{
		Word:     WordGenerator{provider: provider},
		Network:  NetworkGenerator{provider: provider},
		Telecom:  TelecomGenerator{provider: provider},
		provider: provider,
	}
}

// NewSecure returns a generator backed by [crypto/rand].
// It never substitutes a noncryptographic source if entropy acquisition fails.
func NewSecure() *Generator {
	return newGenerator(cryptoProvider{})
}

// NewSeeded returns a synchronized, reproducible ChaCha8 generator.
// A seed, including an all-zero seed, is used exactly as supplied.
// Use [NewSecure] for security-sensitive values.
func NewSeeded(seed [32]byte) *Generator {
	return newGenerator(newUint64Provider(rand.NewChaCha8(seed)))
}

// FromSource returns a generator backed by the supplied [math/rand/v2.Source].
// Construction consumes no random words and preserves the source's algorithm and seed.
// Access is synchronized, with one lock for each complete generation operation.
// Wrap a mutable source once and use it only through the returned generator or
// copied generation handles. Concurrent calls require separate output storage.
// A nil source interface panics immediately. Typed nil sources are invalid.
func FromSource(source rand.Source) *Generator {
	if source == nil {
		panic("randomizer: nil source")
	}
	return newGenerator(newUint64Provider(source))
}

// FromReader returns a generator backed by the supplied [io.Reader].
// Access is synchronized and uses [io.ReadFull] semantics.
// Wrap a mutable reader once and use it only through the returned generator or
// copied generation handles. Concurrent calls require separate output storage.
// A nil reader interface panics immediately. Typed nil readers are invalid.
// [Generator.Read] returns read counts and errors; other generation methods panic
// on read failure and may leave partial output.
func FromReader(reader io.Reader) *Generator {
	if reader == nil {
		panic("randomizer: nil reader")
	}
	return newGenerator(newReaderProvider(reader))
}

func (g *Generator) source() provider {
	if g == nil {
		return fastProvider{}
	}
	return providerOrDefault(g.provider)
}

// Read fills the destination using the bound source and returns its count and
// error. A source error may leave a partially filled destination.
func (g *Generator) Read(out []byte) (int, error) {
	return g.source().Read(out)
}
