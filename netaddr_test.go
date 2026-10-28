package randomizer_test

import (
	"net/netip"
	"testing"

	"github.com/colduction/randomizer-go"
)

var (
	benchAddr   netip.Addr
	benchPrefix netip.Prefix
)

func TestNetworkAddr(t *testing.T) {
	cases := []struct {
		kind   randomizer.IPKind
		prefix netip.Prefix
	}{
		{randomizer.IPv4Any, netip.MustParsePrefix("0.0.0.0/0")},
		{randomizer.IPv4Private, netip.MustParsePrefix("0.0.0.0/0")},
		{randomizer.IPv4Public, netip.MustParsePrefix("0.0.0.0/0")},
		{randomizer.IPv4LinkLocal, netip.MustParsePrefix("169.254.0.0/16")},
		{randomizer.IPv4Documentation, netip.MustParsePrefix("0.0.0.0/0")},
		{randomizer.IPv6Any, netip.MustParsePrefix("::/0")},
		{randomizer.IPv6Global, netip.MustParsePrefix("2000::/3")},
		{randomizer.IPv6Public, netip.MustParsePrefix("2000::/3")},
		{randomizer.IPv6LinkLocal, netip.MustParsePrefix("fe80::/64")},
		{randomizer.IPv6UniqueLocal, netip.MustParsePrefix("fd00::/8")},
		{randomizer.IPv6Documentation, netip.MustParsePrefix("2001:db8::/32")},
		{randomizer.IPv6LinkLocalMulticast, netip.MustParsePrefix("ff12::/16")},
	}
	for _, tc := range cases {
		for range 100 {
			addr := randomizer.Network.Addr(tc.kind)
			if !addr.IsValid() || addr.Is4() != tc.prefix.Addr().Is4() || !tc.prefix.Contains(addr) {
				t.Fatalf("Addr(%v) = %v, want inside %v", tc.kind, addr, tc.prefix)
			}
		}
	}
	if addr := randomizer.Network.Addr(randomizer.IPv4Private); !addr.IsPrivate() {
		t.Fatalf("Addr(IPv4Private) = %v, want private", addr)
	}
}

func TestNetworkPrefix(t *testing.T) {
	cases := []struct {
		kind randomizer.CIDRKind
		bits uint8
		want int
		is4  bool
	}{
		{randomizer.IPv4CIDR, 24, 24, true},
		{randomizer.IPv4CIDR, 0, 0, true},
		{randomizer.IPv4CIDR, 40, 32, true},
		{randomizer.IPv6CIDR, 61, 61, false},
		{randomizer.IPv6CIDR, 200, 128, false},
		{randomizer.IPv6ULAPrefix, 0, 48, false},
	}
	for _, tc := range cases {
		for range 20 {
			prefix := randomizer.Network.Prefix(tc.kind, tc.bits)
			if !prefix.IsValid() || prefix.Bits() != tc.want || prefix.Addr().Is4() != tc.is4 || prefix.Masked() != prefix {
				t.Fatalf("Prefix(%v, %d) = %v, want masked /%d", tc.kind, tc.bits, prefix, tc.want)
			}
			if tc.kind == randomizer.IPv6ULAPrefix && prefix.Addr().As16()[0] != 0xfd {
				t.Fatalf("Prefix(IPv6ULAPrefix) = %v, want fd00::/8", prefix)
			}
		}
	}
}

func TestNetworkAddrIn(t *testing.T) {
	prefixes := []string{
		"192.168.1.0/24", "10.0.0.0/8", "0.0.0.0/0", "203.0.113.7/32",
		"2001:db8::/32", "2001:db8::/64", "fd00::/7", "::/0", "2001:db8::1/128",
		"::ffff:10.0.0.0/104",
	}
	for _, s := range prefixes {
		prefix := netip.MustParsePrefix(s)
		for range 100 {
			addr := randomizer.Network.AddrIn(prefix)
			if !addr.IsValid() || !prefix.Contains(addr) || addr.Is4() != prefix.Addr().Is4() {
				t.Fatalf("AddrIn(%v) = %v, want address inside", prefix, addr)
			}
		}
	}
	for _, s := range []string{"10.1.2.3/24", "2001:db8::1/64"} {
		prefix := netip.MustParsePrefix(s)
		var set, clear bool
		for range 200 {
			addr := randomizer.Network.AddrIn(prefix)
			last := addr.AsSlice()
			if last[len(last)-1]&1 != 0 {
				set = true
			} else {
				clear = true
			}
		}
		if !set || !clear {
			t.Fatalf("AddrIn(%v) never varied the lowest host bit", prefix)
		}
	}
	if got := randomizer.Network.AddrIn(netip.Prefix{}); got.IsValid() {
		t.Fatalf("AddrIn(invalid) = %v, want zero Addr", got)
	}
}

func TestNetworkNetipAllocs(t *testing.T) {
	prefix := netip.MustParsePrefix("2001:db8::/48")
	if allocs := testing.AllocsPerRun(1000, func() {
		benchAddr = randomizer.Network.Addr(randomizer.IPv6Public)
	}); allocs != 0 {
		t.Fatalf("Addr allocs/op = %v, want 0", allocs)
	}
	if allocs := testing.AllocsPerRun(1000, func() {
		benchPrefix = randomizer.Network.Prefix(randomizer.IPv6CIDR, 48)
	}); allocs != 0 {
		t.Fatalf("Prefix allocs/op = %v, want 0", allocs)
	}
	if allocs := testing.AllocsPerRun(1000, func() {
		benchAddr = randomizer.Network.AddrIn(prefix)
	}); allocs != 0 {
		t.Fatalf("AddrIn allocs/op = %v, want 0", allocs)
	}
}

func BenchmarkNetworkAddrIPv4(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchAddr = randomizer.Network.Addr(randomizer.IPv4Any)
	}
}

func BenchmarkNetworkAddrIPv6(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchAddr = randomizer.Network.Addr(randomizer.IPv6Global)
	}
}

func BenchmarkNetworkPrefixIPv6(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchPrefix = randomizer.Network.Prefix(randomizer.IPv6CIDR, 48)
	}
}

func BenchmarkNetworkAddrIn(b *testing.B) {
	prefix := netip.MustParsePrefix("2001:db8::/48")
	b.ReportAllocs()
	for b.Loop() {
		benchAddr = randomizer.Network.AddrIn(prefix)
	}
}
