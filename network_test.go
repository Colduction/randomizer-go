package randomizer_test

import (
	"bytes"
	"net"
	"net/netip"
	"slices"
	"testing"

	"github.com/colduction/randomizer-go"
)

var (
	benchIP     net.IP
	benchMAC    net.HardwareAddr
	benchPort   uint64
	benchIPNet  *net.IPNet
	benchValue  uint64
	benchBuffer [64]byte
)

func TestNetworkAllocs(t *testing.T) {
	if allocs := testing.AllocsPerRun(1000, func() {
		var buf [16]byte
		ip := randomizer.Network.IP(buf[:0], randomizer.IPv6Any)
		if len(ip) != net.IPv6len {
			t.Fatal("bad IPv6 length")
		}
	}); allocs != 0 {
		t.Fatalf("IP with caller buffer allocs/op = %v, want 0", allocs)
	}
	if allocs := testing.AllocsPerRun(1000, func() {
		var buf [8]byte
		mac := randomizer.Network.Hardware(buf[:0], randomizer.HardwareEUI64, randomizer.HardwareOptions{})
		if len(mac) != 8 {
			t.Fatal("bad EUI64 length")
		}
	}); allocs != 0 {
		t.Fatalf("Hardware with caller buffer allocs/op = %v, want 0", allocs)
	}
	kinds := []randomizer.HardwareKind{
		randomizer.HardwareMACRealOUI, randomizer.HardwareMACAAI, randomizer.HardwareMACSAI,
		randomizer.HardwareEUI64RealOUI, randomizer.HardwareBluetoothStatic,
		randomizer.HardwareBluetoothNRPA, randomizer.HardwareBluetoothRPA,
	}
	for _, kind := range kinds {
		if allocs := testing.AllocsPerRun(1000, func() {
			var buf [8]byte
			if randomizer.Network.Hardware(buf[:0], kind, randomizer.HardwareOptions{}) == nil {
				t.Fatal("nil address")
			}
		}); allocs != 0 {
			t.Fatalf("Hardware(%d) with caller buffer allocs/op = %v, want 0", kind, allocs)
		}
	}
	if allocs := testing.AllocsPerRun(1000, func() {
		var buf [8]byte
		options := randomizer.HardwareOptions{IRK: []byte("0123456789abcdef")}
		if randomizer.Network.Hardware(buf[:0], randomizer.HardwareBluetoothRPA, options) == nil {
			t.Fatal("nil address")
		}
	}); allocs != 0 {
		t.Fatalf("Hardware(RPA with IRK) steady-state allocs/op = %v, want 0", allocs)
	}
	if allocs := testing.AllocsPerRun(1000, func() {
		benchValue = randomizer.Network.Value(randomizer.ASN)
	}); allocs != 0 {
		t.Fatalf("Value allocs/op = %v, want 0", allocs)
	}
	if allocs := testing.AllocsPerRun(1000, func() {
		benchIPNet = randomizer.Network.CIDR(randomizer.IPv6ULAPrefix, 0)
	}); allocs > 1 {
		t.Fatalf("CIDR(IPv6ULAPrefix) allocs/op = %v, want <= 1", allocs)
	}
	if allocs := testing.AllocsPerRun(1000, func() {
		var buf [16]byte
		mac := net.HardwareAddr{0, 1, 2, 3, 4, 5}
		if randomizer.Network.IPv6EUI64(buf[:0], randomizer.IPv6LinkLocal, mac) == nil {
			t.Fatal("nil address")
		}
	}); allocs != 0 {
		t.Fatalf("IPv6EUI64 with caller buffer allocs/op = %v, want 0", allocs)
	}
}

func TestNetworkHardwareAliasing(t *testing.T) {
	original := []byte{0x00, 0x1a, 0x2b, 0x3c, 0x4d, 0x5e}
	for source := 0; source <= 12; source++ {
		for destination := 0; destination <= 12; destination++ {
			for _, kind := range []randomizer.HardwareKind{randomizer.HardwareEUI64FromMAC, randomizer.HardwareModifiedEUI64FromMAC} {
				var buf [32]byte
				copy(buf[source:], original)
				mac := net.HardwareAddr(buf[source : source+6])
				got := randomizer.Network.Hardware(buf[destination:destination], kind, randomizer.HardwareOptions{MAC: mac})
				want := []byte{0, 0x1a, 0x2b, 0xff, 0xfe, 0x3c, 0x4d, 0x5e}
				if kind == randomizer.HardwareModifiedEUI64FromMAC {
					want[0] = 2
				}
				if !slices.Equal(got, want) {
					t.Fatalf("Hardware alias %d/%d kind %d = %x, want %x", source, destination, kind, got, want)
				}
			}
			for bits := uint8(1); bits <= 48; bits++ {
				var buf [32]byte
				copy(buf[source:], original)
				options := randomizer.HardwareOptions{Prefix: buf[source : source+6], PrefixBits: bits}
				got := randomizer.Network.Hardware(buf[destination:destination], randomizer.HardwareMACPrefix, options)
				for bit := uint8(0); bit < bits; bit++ {
					mask := byte(0x80 >> (bit & 7))
					if got[bit/8]&mask != original[bit/8]&mask {
						t.Fatalf("Hardware prefix alias %d/%d bits %d = %x, want prefix %x", source, destination, bits, got, original)
					}
				}
			}
		}
	}
}

func TestNetworkIPv6EUI64Aliasing(t *testing.T) {
	for _, original := range [][]byte{{0, 0x1a, 0x2b, 0x3c, 0x4d, 0x5e}, {0, 0x1a, 0x2b, 0x11, 0x22, 0x3c, 0x4d, 0x5e}} {
		want := randomizer.Network.IPv6EUI64(nil, randomizer.IPv6LinkLocal, original)
		for source := 0; source <= 12; source++ {
			for destination := 0; destination <= 12; destination++ {
				var buf [40]byte
				copy(buf[source:], original)
				got := randomizer.Network.IPv6EUI64(buf[destination:destination], randomizer.IPv6LinkLocal, buf[source:source+len(original)])
				if !slices.Equal(got, want) {
					t.Fatalf("IPv6EUI64 alias %d/%d = %v, want %v", source, destination, got, want)
				}
			}
		}
	}
}

func TestNetworkReservedEUI64Public(t *testing.T) {
	g := randomizer.FromSource(newSeqProvider(t))
	for _, iid := range []uint64{0, 0xfdffffffffffff80, 0xfdffffffffffffff, 0xffffffffffffff80, 0xffffffffffffffff} {
		var mac [8]byte
		for i := range mac {
			mac[i] = byte(iid >> (56 - i*8))
		}
		mac[0] ^= 2
		dst := bytes.Repeat([]byte{0xa5}, 16)
		if got := g.Network.IPv6EUI64(dst[:0], randomizer.IPv6Public, mac[:]); got != nil || !bytes.Equal(dst, bytes.Repeat([]byte{0xa5}, 16)) {
			t.Fatalf("IPv6Public accepts reserved IID %016x or changes destination", iid)
		}
	}
}

func TestNetworkIPInCIDRAliasing(t *testing.T) {
	g := randomizer.FromSource(fixedProvider(0x0123456789abcdef))
	for _, raw := range []string{"10.1.2.0/24", "2001:db8::/32", "2001:db8::/64", "2001:db8::/65", "2001:db8::/120"} {
		_, network, _ := net.ParseCIDR(raw)
		want := g.Network.IPInCIDR(nil, network)
		for offset := 0; offset <= 12; offset++ {
			var buf [40]byte
			copy(buf[:], network.IP)
			aliased := &net.IPNet{IP: buf[:len(network.IP)], Mask: network.Mask}
			if got := g.Network.IPInCIDR(buf[offset:offset], aliased); !slices.Equal(got, want) {
				t.Fatalf("IPInCIDR IP alias %s/%d = %v, want %v", raw, offset, got, want)
			}
			copy(buf[:], network.Mask)
			aliased = &net.IPNet{IP: network.IP, Mask: buf[:len(network.Mask)]}
			if got := g.Network.IPInCIDR(buf[offset:offset], aliased); !slices.Equal(got, want) {
				t.Fatalf("IPInCIDR mask alias %s/%d = %v, want %v", raw, offset, got, want)
			}
		}
	}
}

func TestNetworkFixedResultsConsumeNoEntropy(t *testing.T) {
	g := randomizer.FromSource(newSeqProvider(t))
	for _, kind := range []randomizer.CIDRKind{randomizer.IPv4CIDR, randomizer.IPv6CIDR} {
		if prefix := g.Network.Prefix(kind, 0); prefix.Bits() != 0 || !prefix.Addr().IsUnspecified() {
			t.Fatalf("Prefix(%d, 0) = %v", kind, prefix)
		}
		if network := g.Network.CIDR(kind, 0); !network.IP.IsUnspecified() {
			t.Fatalf("CIDR(%d, 0) = %v", kind, network)
		}
	}
	for _, raw := range []string{"203.0.113.7/32", "2001:db8::1/128"} {
		prefix := netip.MustParsePrefix(raw)
		if got := g.Network.AddrIn(prefix); got != prefix.Addr() {
			t.Fatalf("AddrIn(%v) = %v", prefix, got)
		}
		_, network, _ := net.ParseCIDR(raw)
		if got := g.Network.IPInCIDR(nil, network); !got.Equal(network.IP) {
			t.Fatalf("IPInCIDR(%v) = %v", network, got)
		}
	}
	mac := net.HardwareAddr{0, 1, 2, 3, 4, 5}
	if got := g.Network.Hardware(nil, randomizer.HardwareMACPrefix, randomizer.HardwareOptions{Prefix: mac, PrefixBits: 48}); !bytes.Equal(got, mac) {
		t.Fatalf("fixed hardware prefix = %x, want %x", got, mac)
	}
	if got := g.Network.IPv6EUI64(nil, randomizer.IPv6LinkLocal, mac); got == nil {
		t.Fatal("fixed link-local EUI-64 failed")
	}
}

func TestNetworkIPInCIDRInvalidMasks(t *testing.T) {
	g := randomizer.FromSource(newSeqProvider(t))
	cases := []*net.IPNet{
		{IP: net.IPv4(10, 1, 2, 3), Mask: net.IPMask{255, 0, 255, 0}},
		{IP: net.ParseIP("2001:db8::1"), Mask: net.IPMask{255, 0, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{IP: net.IP{1, 2, 3}, Mask: net.CIDRMask(24, 32)},
		{IP: net.ParseIP("2001:db8::1"), Mask: net.CIDRMask(24, 32)},
	}
	for _, network := range cases {
		dst := bytes.Repeat([]byte{0xa5}, 16)
		if got := g.Network.IPInCIDR(dst[:0], network); got != nil || !bytes.Equal(dst, bytes.Repeat([]byte{0xa5}, 16)) {
			t.Fatalf("invalid network %v accepted or destination changed", network)
		}
	}
}

func FuzzNetworkAddrIn(f *testing.F) {
	f.Add(uint64(0x20010db800000000), uint64(1), uint8(64), false)
	f.Add(uint64(0), uint64(0x0a010203), uint8(31), true)
	f.Fuzz(func(t *testing.T, hi, lo uint64, bits uint8, ipv4 bool) {
		var data [16]byte
		for i := range 8 {
			data[i], data[i+8] = byte(hi>>(56-i*8)), byte(lo>>(56-i*8))
		}
		addr := netip.AddrFrom16(data)
		if ipv4 {
			addr = netip.AddrFrom4([4]byte(data[12:]))
		}
		prefix := netip.PrefixFrom(addr, int(bits)%(addr.BitLen()+1))
		got := randomizer.Network.AddrIn(prefix)
		if !prefix.Contains(got) || got.Is4() != addr.Is4() {
			t.Fatalf("AddrIn(%v) = %v", prefix, got)
		}
	})
}
