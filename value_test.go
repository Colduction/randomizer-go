package randomizer_test

import (
	"testing"

	"github.com/colduction/randomizer-go"
)

func TestNetworkValueRanges(t *testing.T) {
	cases := []struct {
		kind     randomizer.ValueKind
		min, max uint64
	}{
		{randomizer.AnyPort, 0, 65535},
		{randomizer.PrivilegedPort, 1, 1023},
		{randomizer.RegisteredPort, 1024, 49151},
		{randomizer.EphemeralPort, 49152, 65535},
		{randomizer.VLANID, 0, 4095},
		{randomizer.VNI, 0, 0xffffff},
		{randomizer.FlowLabel, 0, 0xfffff},
		{randomizer.MPLSLabel, 0, 0xfffff},
		{randomizer.VLANIDUnreserved, 1, 4094},
		{randomizer.MPLSLabelUnreserved, 16, 0xfffff},
		{randomizer.ASNPublic16, 1, 64495},
		{randomizer.ASNPrivate16, 64512, 65534},
		{randomizer.ASNPrivate32, 4200000000, 4294967294},
		{randomizer.ASNPublic, 1, 4199999999},
	}
	for _, tc := range cases {
		for range 1000 {
			v := randomizer.Network.Value(tc.kind)
			if v < tc.min || v > tc.max {
				t.Fatalf("Value(%v) = %d, want [%d, %d]", tc.kind, v, tc.min, tc.max)
			}
		}
	}
}

func isReservedASN(a uint64) bool {
	return a == 0 || a == 23456 || (a >= 64496 && a <= 131071) || a >= 4200000000
}

func TestNetworkValueASNKinds(t *testing.T) {
	for range 10000 {
		if a := randomizer.Network.Value(randomizer.ASNPublic); isReservedASN(a) {
			t.Fatalf("Value(ASNPublic) = %d is reserved", a)
		}
		if a := randomizer.Network.Value(randomizer.ASNPublic16); a == 23456 || a > 64495 || a == 0 {
			t.Fatalf("Value(ASNPublic16) = %d is reserved", a)
		}
	}
	g := randomizer.FromSource(newSeqProvider(t, 23456<<32, 100))
	if got := g.Network.Value(randomizer.ASNPublic); got != 100 {
		t.Fatalf("Value(ASNPublic) after rejection = %d, want 100", got)
	}
}

func TestNetworkValueFixedProvider(t *testing.T) {
	const value fixedProvider = 0x0123456789abcdef
	g := randomizer.FromSource(value)

	if got, want := g.Network.Value(randomizer.VLANID), uint64(0x12); got != want {
		t.Fatalf("Value(VLANID) = 0x%x, want 0x%x", got, want)
	}
	if got, want := g.Network.Value(randomizer.ASN), uint64(value)>>32; got != want {
		t.Fatalf("Value(ASN) = 0x%x, want 0x%x", got, want)
	}
	if got, want := g.Network.Value(randomizer.VNI), uint64(value)>>40; got != want {
		t.Fatalf("Value(VNI) = 0x%x, want 0x%x", got, want)
	}
	if got, want := g.Network.Value(randomizer.FlowLabel), uint64(value)>>44; got != want {
		t.Fatalf("Value(FlowLabel) = 0x%x, want 0x%x", got, want)
	}
	if got, want := g.Network.Value(randomizer.MPLSLabel), uint64(value)>>44; got != want {
		t.Fatalf("Value(MPLSLabel) = 0x%x, want 0x%x", got, want)
	}
	if got, want := g.Network.Value(randomizer.IPv6InterfaceID), uint64(value)&^(uint64(0x03)<<56); got != want {
		t.Fatalf("Value(IPv6InterfaceID) = 0x%x, want 0x%x", got, want)
	}
}

func BenchmarkNetworkValuePort(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchPort = randomizer.Network.Value(randomizer.AnyPort)
	}
}

func BenchmarkNetworkValueRegisteredPort(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchPort = randomizer.Network.Value(randomizer.RegisteredPort)
	}
}

func BenchmarkNetworkValueASN(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchValue = randomizer.Network.Value(randomizer.ASN)
	}
}

func BenchmarkNetworkValueASNPublic(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchValue = randomizer.Network.Value(randomizer.ASNPublic)
	}
}
