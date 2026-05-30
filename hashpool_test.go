package randomizer_test

import (
	"testing"

	"github.com/colduction/randomizer-go"
)

func TestHashPoolNewHashPoolZero(t *testing.T) {
	p := randomizer.NewHashPool(0)
	if p != nil {
		t.Fatal("NewHashPool(0) should return nil")
	}
	// Ensure nil receiver methods are safe.
	_ = p.Sum64()
	_ = p.Sum32()
	if got := p.Sum(nil); len(got) != 8 {
		t.Fatalf("nil pool Sum(nil) length = %d, want 8", len(got))
	}
}

func TestHashPoolSumAppends8Bytes(t *testing.T) {
	p := randomizer.NewHashPool(4)
	if p == nil {
		t.Fatal("NewHashPool(4) returned nil")
	}
	in := []byte{1, 2, 3}
	out := p.Sum(in)
	if len(out) != len(in)+8 {
		t.Fatalf("Sum length = %d, want %d", len(out), len(in)+8)
	}
	if out[0] != 1 || out[1] != 2 || out[2] != 3 {
		t.Fatalf("Sum prefix mutated: got %v", out[:3])
	}
}

func TestHashPoolSumVaries(t *testing.T) {
	p := randomizer.NewHashPool(2)
	if p == nil {
		t.Fatal("NewHashPool(2) returned nil")
	}

	first64 := p.Sum64()
	var different64 bool
	for range 1024 {
		if p.Sum64() != first64 {
			different64 = true
			break
		}
	}
	if !different64 {
		t.Fatal("Sum64 appears constant")
	}

	first32 := p.Sum32()
	var different32 bool
	for range 1024 {
		if p.Sum32() != first32 {
			different32 = true
			break
		}
	}
	if !different32 {
		t.Fatal("Sum32 appears constant")
	}
}
