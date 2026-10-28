package randomizer_test

import (
	"net"
	"testing"

	"github.com/colduction/randomizer-go"
)

func TestNetworkCIDR(t *testing.T) {
	cases := []struct {
		kind       randomizer.CIDRKind
		prefix     uint8
		ipLen      int
		maskBits   int
		maskLength int
	}{
		{randomizer.IPv4CIDR, 24, net.IPv4len, 24, 32},
		{randomizer.IPv4CIDR, 13, net.IPv4len, 13, 32},
		{randomizer.IPv4CIDR, 0, net.IPv4len, 0, 32},
		{randomizer.IPv4CIDR, 40, net.IPv4len, 32, 32},
		{randomizer.IPv6CIDR, 48, net.IPv6len, 48, 128},
		{randomizer.IPv6CIDR, 61, net.IPv6len, 61, 128},
		{randomizer.IPv6CIDR, 200, net.IPv6len, 128, 128},
	}
	for _, tc := range cases {
		for range 20 {
			ipNet := randomizer.Network.CIDR(tc.kind, tc.prefix)
			if ipNet == nil || len(ipNet.IP) != tc.ipLen {
				t.Fatalf("CIDR(%v, %d) = %v, want IP len %d", tc.kind, tc.prefix, ipNet, tc.ipLen)
			}
			ones, bits := ipNet.Mask.Size()
			if ones != tc.maskBits || bits != tc.maskLength {
				t.Fatalf("CIDR(%v, %d) mask = %d/%d, want %d/%d", tc.kind, tc.prefix, ones, bits, tc.maskBits, tc.maskLength)
			}
			for i, b := range ipNet.IP {
				if b&^ipNet.Mask[i] != 0 {
					t.Fatalf("CIDR(%v, %d) host bit set in byte %d", tc.kind, tc.prefix, i)
				}
			}
		}
	}
}

func TestNetworkIPv6ULAPrefix(t *testing.T) {
	ipNet := randomizer.Network.CIDR(randomizer.IPv6ULAPrefix, 0)
	if ipNet == nil || len(ipNet.IP) != net.IPv6len || ipNet.IP[0] != 0xfd || ipNet.IP.To4() != nil {
		t.Fatalf("CIDR(IPv6ULAPrefix) = %v, want fd00::/8 IPv6", ipNet)
	}
	for i := 6; i < net.IPv6len; i++ {
		if ipNet.IP[i] != 0 {
			t.Fatalf("CIDR(IPv6ULAPrefix) host bit set in byte %d", i)
		}
	}
	ones, bits := ipNet.Mask.Size()
	if ones != 48 || bits != 128 {
		t.Fatalf("CIDR(IPv6ULAPrefix) mask = %d/%d, want 48/128", ones, bits)
	}
}

func TestNetworkIPInCIDR(t *testing.T) {
	_, ip4Net, _ := net.ParseCIDR("192.168.1.0/24")
	_, ip6Net, _ := net.ParseCIDR("2001:db8::/32")
	for range 100 {
		var buf [16]byte
		ip4 := randomizer.Network.IPInCIDR(buf[:0], ip4Net)
		if ip4 == nil || !ip4Net.Contains(ip4) || &ip4[0] != &buf[0] {
			t.Fatalf("IPInCIDR IPv4 = %v, want address inside %v using caller buffer", ip4, ip4Net)
		}
		ip6 := randomizer.Network.IPInCIDR(buf[:0], ip6Net)
		if ip6 == nil || !ip6Net.Contains(ip6) || &ip6[0] != &buf[0] {
			t.Fatalf("IPInCIDR IPv6 = %v, want address inside %v using caller buffer", ip6, ip6Net)
		}
	}
	if got := randomizer.Network.IPInCIDR(nil, nil); got != nil {
		t.Fatalf("IPInCIDR(nil) = %v, want nil", got)
	}
	if got := randomizer.Network.IPInCIDR(nil, &net.IPNet{IP: net.IPv4(1, 2, 3, 4), Mask: net.CIDRMask(8, 16)}); got != nil {
		t.Fatalf("IPInCIDR with 2-byte mask = %v, want nil", got)
	}
}

func TestNetworkIPInCIDRUnmaskedBase(t *testing.T) {
	cases := []*net.IPNet{
		{IP: net.IPv4(10, 1, 2, 3).To4(), Mask: net.CIDRMask(24, 32)},
		{IP: net.ParseIP("2001:db8::1"), Mask: net.CIDRMask(64, 128)},
	}
	for _, ipNet := range cases {
		var set, clear bool
		for range 200 {
			ip := randomizer.Network.IPInCIDR(nil, ipNet)
			if !ipNet.Contains(ip) {
				t.Fatalf("IPInCIDR(%v) = %v outside network", ipNet, ip)
			}
			if ip[len(ip)-1]&1 != 0 {
				set = true
			} else {
				clear = true
			}
		}
		if !set || !clear {
			t.Fatalf("IPInCIDR(%v) never varied the lowest host bit", ipNet)
		}
	}
}

func BenchmarkNetworkCIDR(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchIPNet = randomizer.Network.CIDR(randomizer.IPv6CIDR, 48)
	}
}

func BenchmarkNetworkIPInCIDR6(b *testing.B) {
	_, ipNet, _ := net.ParseCIDR("2001:db8::/32")
	b.ReportAllocs()
	for b.Loop() {
		benchIP = randomizer.Network.IPInCIDR(benchBuffer[:0], ipNet)
	}
}

func TestNetworkIPInCIDRMappedIPv4(t *testing.T) {
	g := randomizer.FromSource(fixedProvider(0x0123456789abcdef))
	for _, address := range []net.IP{net.IP{10, 1, 2, 3}, net.ParseIP("::ffff:10.1.2.3")} {
		for _, mask := range []net.IPMask{net.CIDRMask(0, 32), net.CIDRMask(24, 32), net.CIDRMask(0, 128), net.CIDRMask(80, 128), net.CIDRMask(120, 128), net.CIDRMask(128, 128)} {
			network := &net.IPNet{IP: address, Mask: mask}
			got := g.Network.IPInCIDR(nil, network)
			if len(got) != net.IPv4len || !network.Contains(got) {
				t.Fatalf("IPInCIDR(%v) = %v, want 4-byte IPv4 inside network", network, got)
			}
		}
	}
}
