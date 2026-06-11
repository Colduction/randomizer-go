package randomizer_test

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/colduction/randomizer-go"
)

type fixedProvider uint64

func (fp fixedProvider) Sum(b []byte) []byte {
	return binary.LittleEndian.AppendUint64(b, uint64(fp))
}

func (fixedProvider) Sum32() uint32 {
	return 0xabcdef01
}

func (fp fixedProvider) Sum64() uint64 {
	return uint64(fp)
}

type oneShotSource struct {
	calls atomic.Uint32
}

func (oss *oneShotSource) Uint64() uint64 {
	if oss.calls.Add(1) != 1 {
		panic("Uint64 called more than once")
	}
	return 0x0123456789abcdef
}

func TestProviderSetProviderRoutesGenerators(t *testing.T) {
	const value fixedProvider = 0x0123456789abcdef

	previous := randomizer.SetProvider(value)
	defer randomizer.SetProvider(previous)

	if got := randomizer.Uint[uint64](); got != uint64(value) {
		t.Fatalf("Uint[uint64]() = 0x%016x, want 0x%016x", got, uint64(value))
	}
	if got := randomizer.Word.Hex(16, false); got != "fedcba9876543210" {
		t.Fatalf("Word.Hex(16, false) = %q, want %q", got, "fedcba9876543210")
	}
	if got := randomizer.Network.VLANID(); got != 0x12 {
		t.Fatalf("Network.VLANID() = 0x%x, want 0x12", got)
	}
}

func TestProviderSetProviderNilRestoresDefault(t *testing.T) {
	previous := randomizer.SetProvider(fixedProvider(1))
	defer randomizer.SetProvider(previous)

	if got := randomizer.SetProvider(nil); got != fixedProvider(1) {
		t.Fatalf("SetProvider(nil) previous provider = %v, want %v", got, fixedProvider(1))
	}
	if got := randomizer.SetProvider(previous); got != randomizer.DefaultProvider {
		t.Fatalf("SetProvider(previous) replaced %T, want DefaultProvider", got)
	}
}

func TestProviderSetProviderConcurrent(t *testing.T) {
	previous := randomizer.SetProvider(fixedProvider(1))
	defer randomizer.SetProvider(previous)

	const goroutines = 32
	var group sync.WaitGroup
	group.Add(goroutines)
	for i := range goroutines {
		go func() {
			defer group.Done()
			provider := fixedProvider(i + 1)
			for range 128 {
				randomizer.SetProvider(provider)
				_ = randomizer.Uint[uint64]()
			}
		}()
	}
	group.Wait()
}

func TestUint64ProviderSupportsMathRand(t *testing.T) {
	const seed int64 = 42

	first := randomizer.NewUint64Provider(rand.New(rand.NewSource(seed)))
	second := randomizer.NewUint64Provider(rand.New(rand.NewSource(seed)))
	if first == nil || second == nil {
		t.Fatal("NewUint64Provider returned nil")
	}

	for range 16 {
		if got, want := first.Sum64(), second.Sum64(); got != want {
			t.Fatalf("matching math/rand sources produced 0x%016x and 0x%016x", got, want)
		}
	}
}

func TestUint64ProviderConcurrent(t *testing.T) {
	const (
		goroutines = 32
		values     = 128
	)

	source := new(oneShotSource)
	provider := randomizer.NewUint64Provider(source)
	if provider == nil {
		t.Fatal("NewUint64Provider returned nil")
	}

	results := make(chan uint64, goroutines*values)
	var group sync.WaitGroup
	group.Add(goroutines)
	for range goroutines {
		go func() {
			defer group.Done()
			for range values {
				results <- provider.Sum64()
			}
		}()
	}
	group.Wait()
	close(results)

	seen := make(map[uint64]struct{}, goroutines*values)
	for value := range results {
		if _, exists := seen[value]; exists {
			t.Fatalf("Sum64 returned duplicate value 0x%016x", value)
		}
		seen[value] = struct{}{}
	}
	if got := source.calls.Load(); got != 1 {
		t.Fatalf("source Uint64 calls = %d, want 1", got)
	}
}

func TestReaderProviderValues(t *testing.T) {
	data := []byte{
		0x01, 0x02, 0x03, 0x04,
		0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c,
		0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14,
	}
	provider := randomizer.NewReaderProvider(bytes.NewReader(data))
	if provider == nil {
		t.Fatal("NewReaderProvider returned nil")
	}

	if got := provider.Sum32(); got != 0x04030201 {
		t.Fatalf("Sum32() = 0x%08x, want 0x04030201", got)
	}
	if got := provider.Sum64(); got != 0x0c0b0a0908070605 {
		t.Fatalf("Sum64() = 0x%016x, want 0x0c0b0a0908070605", got)
	}

	got := provider.Sum([]byte{0xaa, 0xbb})
	want := []byte{0xaa, 0xbb, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14}
	if !bytes.Equal(got, want) {
		t.Fatalf("Sum() = %x, want %x", got, want)
	}
}

func TestReaderProviderSupportsCryptoRand(t *testing.T) {
	const goroutines = 32

	provider := randomizer.NewReaderProvider(cryptorand.Reader)
	if provider == nil {
		t.Fatal("NewReaderProvider returned nil")
	}

	var group sync.WaitGroup
	group.Add(goroutines)
	for range goroutines {
		go func() {
			defer group.Done()
			if got := provider.Sum(nil); len(got) != 8 {
				t.Errorf("Sum(nil) length = %d, want 8", len(got))
			}
		}()
	}
	group.Wait()
}

func TestProviderConstructorsRejectNil(t *testing.T) {
	if got := randomizer.NewUint64Provider(nil); got != nil {
		t.Fatalf("NewUint64Provider(nil) = %v, want nil", got)
	}
	if got := randomizer.NewReaderProvider(nil); got != nil {
		t.Fatalf("NewReaderProvider(nil) = %v, want nil", got)
	}

}
