package randomizer_test

import (
	"net"
	"testing"

	"github.com/colduction/randomizer-go"
)

var (
	benchIP  net.IP
	benchMAC net.HardwareAddr
)

func TestNetworkIPv4Addr(t *testing.T) {
	ip := randomizer.Network.IPv4Addr()
	if len(ip) != net.IPv4len {
		t.Fatalf("IPv4Addr length = %d, want %d", len(ip), net.IPv4len)
	}
	if ip.To4() == nil {
		t.Fatal("IPv4Addr returned non-IPv4 address")
	}
}

func TestNetworkIPv6Addr(t *testing.T) {
	ip := randomizer.Network.IPv6Addr()
	if len(ip) != net.IPv6len {
		t.Fatalf("IPv6Addr length = %d, want %d", len(ip), net.IPv6len)
	}
	if ip.To16() == nil {
		t.Fatal("IPv6Addr returned non-IPv6 address")
	}
}

func TestNetworkMACAddrBits(t *testing.T) {
	cases := []struct {
		local     bool
		multicast bool
	}{
		{local: false, multicast: false},
		{local: true, multicast: false},
		{local: false, multicast: true},
		{local: true, multicast: true},
	}
	for _, tc := range cases {
		mac := randomizer.Network.MACAddr(tc.local, tc.multicast)
		if len(mac) != 6 {
			t.Fatalf("MACAddr length = %d, want 6", len(mac))
		}
		gotLocal := (mac[0] & 0x02) != 0
		if gotLocal != tc.local {
			t.Fatalf("MACAddr local bit = %t, want %t (mac=%v)", gotLocal, tc.local, mac)
		}
		gotMulticast := (mac[0] & 0x01) != 0
		if gotMulticast != tc.multicast {
			t.Fatalf("MACAddr multicast bit = %t, want %t (mac=%v)", gotMulticast, tc.multicast, mac)
		}
	}
}

func TestNetworkIPv6UnicastPrefixes(t *testing.T) {
	global := randomizer.Network.IPv6UnicastAddr(randomizer.GlobalType)
	if len(global) != net.IPv6len {
		t.Fatalf("Global unicast length = %d, want %d", len(global), net.IPv6len)
	}
	if global[0]&0xE0 != 0x20 {
		t.Fatalf("Global unicast prefix mismatch: first byte=0x%02X", global[0])
	}

	linkLocal := randomizer.Network.IPv6UnicastAddr(randomizer.LinkLocalType)
	if linkLocal[0] != 0xFE || (linkLocal[1]&0xC0) != 0x80 {
		t.Fatalf("Link-local prefix mismatch: first two bytes=0x%02X 0x%02X", linkLocal[0], linkLocal[1])
	}

	siteLocal := randomizer.Network.IPv6UnicastAddr(randomizer.SiteLocalType)
	if siteLocal[0] != 0xFE || (siteLocal[1]&0xC0) != 0xC0 {
		t.Fatalf("Site-local prefix mismatch: first two bytes=0x%02X 0x%02X", siteLocal[0], siteLocal[1])
	}

	uniqueLocal := randomizer.Network.IPv6UnicastAddr(randomizer.UniqueLocalType)
	if uniqueLocal[0] != 0xFD {
		t.Fatalf("Unique-local prefix mismatch: first byte=0x%02X", uniqueLocal[0])
	}

	privateLocal := randomizer.Network.IPv6UnicastAddr(randomizer.PrivateType)
	if privateLocal[0] != 0xFD {
		t.Fatalf("PrivateType prefix mismatch: first byte=0x%02X", privateLocal[0])
	}
}

func TestNetworkIPv6MulticastScope(t *testing.T) {
	scopes := []randomizer.MulticastScope{
		randomizer.InterfaceLocalScope,
		randomizer.LinkLocalScope,
		randomizer.AdminLocalScope,
		randomizer.SiteLocalScope,
		randomizer.OrgLocalScope,
		randomizer.GlobalScope,
	}
	for _, scope := range scopes {
		ip := randomizer.Network.IPv6MulticastAddr(scope)
		if len(ip) != net.IPv6len {
			t.Fatalf("IPv6MulticastAddr length = %d, want %d", len(ip), net.IPv6len)
		}
		if ip[0] != 0xFF {
			t.Fatalf("IPv6MulticastAddr prefix byte = 0x%02X, want 0xFF", ip[0])
		}
		if ip[1]&0x0F != uint8(scope) {
			t.Fatalf("IPv6MulticastAddr scope nibble = 0x%X, want 0x%X", ip[1]&0x0F, uint8(scope))
		}
		if ip[1]&0xF0 != 0x00 {
			t.Fatalf("IPv6MulticastAddr flags nibble = 0x%X, want 0x0", ip[1]>>4)
		}
	}
}

func BenchmarkNetworkIPv4Addr(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchIP = randomizer.Network.IPv4Addr()
	}
}

func BenchmarkNetworkIPv6Addr(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchIP = randomizer.Network.IPv6Addr()
	}
}

func BenchmarkNetworkMACAddr(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchMAC = randomizer.Network.MACAddr(true, true)
	}
}

func BenchmarkNetworkIPv6UnicastAddr(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchIP = randomizer.Network.IPv6UnicastAddr(randomizer.GlobalType)
	}
}

func BenchmarkNetworkIPv6MulticastAddr(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchIP = randomizer.Network.IPv6MulticastAddr(randomizer.GlobalScope)
	}
}

func TestNetworkPort(t *testing.T) {
	cases := []struct {
		portRange randomizer.PortRange
		min, max  uint16
	}{
		{randomizer.AnyPort, 0, 65535},
		{randomizer.PrivilegedPort, 1, 1023},
		{randomizer.RegisteredPort, 1024, 49151},
		{randomizer.EphemeralPort, 49152, 65535},
	}
	for _, tc := range cases {
		for range 1000 {
			p := randomizer.Network.Port(tc.portRange)
			if p < tc.min || p > tc.max {
				t.Fatalf("Port(%v) = %d, want [%d, %d]", tc.portRange, p, tc.min, tc.max)
			}
		}
	}
}

func TestNetworkVLANID(t *testing.T) {
	for range 1000 {
		v := randomizer.Network.VLANID()
		if v > 4095 {
			t.Fatalf("VLANID = %d, want [0, 4095]", v)
		}
	}
}

func TestNetworkUUIDv4Version(t *testing.T) {
	uuid := randomizer.Network.UUIDv4()
	if uuid[6]>>4 != 0x4 {
		t.Fatalf("UUIDv4 version nibble = 0x%X, want 0x4", uuid[6]>>4)
	}
	if uuid[8]>>6 != 0x2 {
		t.Fatalf("UUIDv4 variant bits = 0x%X, want 0x2 (10xx)", uuid[8]>>6)
	}
}

func TestNetworkUUIDv4StringFormat(t *testing.T) {
	s := randomizer.Network.UUIDv4String()
	if len(s) != 36 {
		t.Fatalf("UUIDv4String length = %d, want 36", len(s))
	}
	for _, pos := range []int{8, 13, 18, 23} {
		if s[pos] != '-' {
			t.Fatalf("UUIDv4String[%d] = %q, want '-'", pos, s[pos])
		}
	}
	if s[14] != '4' {
		t.Fatalf("UUIDv4String version char = %q, want '4'", s[14])
	}
	variantNibble := s[19]
	if variantNibble != '8' && variantNibble != '9' && variantNibble != 'a' && variantNibble != 'b' {
		t.Fatalf("UUIDv4String variant char = %q, want one of '8','9','a','b'", variantNibble)
	}
}

func TestNetworkIPv4CIDR(t *testing.T) {
	cases := []uint8{0, 8, 16, 24, 32}
	for _, prefix := range cases {
		ipNet := randomizer.Network.IPv4CIDR(prefix)
		if ipNet == nil {
			t.Fatalf("IPv4CIDR(%d) returned nil", prefix)
		}
		if len(ipNet.IP) != net.IPv4len {
			t.Fatalf("IPv4CIDR(%d) IP length = %d, want %d", prefix, len(ipNet.IP), net.IPv4len)
		}
		ones, bits := ipNet.Mask.Size()
		if ones != int(prefix) || bits != 32 {
			t.Fatalf("IPv4CIDR(%d) mask ones=%d bits=%d", prefix, ones, bits)
		}
		for i, b := range ipNet.IP {
			if b&^ipNet.Mask[i] != 0 {
				t.Fatalf("IPv4CIDR(%d) host bits not zeroed in IP byte %d", prefix, i)
			}
		}
	}
}

func TestNetworkIPv6CIDR(t *testing.T) {
	cases := []uint8{0, 32, 48, 64, 128}
	for _, prefix := range cases {
		ipNet := randomizer.Network.IPv6CIDR(prefix)
		if ipNet == nil {
			t.Fatalf("IPv6CIDR(%d) returned nil", prefix)
		}
		if len(ipNet.IP) != net.IPv6len {
			t.Fatalf("IPv6CIDR(%d) IP length = %d, want %d", prefix, len(ipNet.IP), net.IPv6len)
		}
		ones, bits := ipNet.Mask.Size()
		if ones != int(prefix) || bits != 128 {
			t.Fatalf("IPv6CIDR(%d) mask ones=%d bits=%d", prefix, ones, bits)
		}
		for i, b := range ipNet.IP {
			if b&^ipNet.Mask[i] != 0 {
				t.Fatalf("IPv6CIDR(%d) host bits not zeroed in IP byte %d", prefix, i)
			}
		}
	}
}

func TestNetworkIPv4AddrInCIDR(t *testing.T) {
	_, ipNet, _ := net.ParseCIDR("192.168.1.0/24")
	for range 100 {
		ip := randomizer.Network.IPv4AddrInCIDR(ipNet)
		if ip == nil {
			t.Fatal("IPv4AddrInCIDR returned nil")
		}
		if !ipNet.Contains(ip) {
			t.Fatalf("IPv4AddrInCIDR returned %v, not in %v", ip, ipNet)
		}
	}
}

func TestNetworkIPv6AddrInCIDR(t *testing.T) {
	_, ipNet, _ := net.ParseCIDR("2001:db8::/32")
	for range 100 {
		ip := randomizer.Network.IPv6AddrInCIDR(ipNet)
		if ip == nil {
			t.Fatal("IPv6AddrInCIDR returned nil")
		}
		if !ipNet.Contains(ip) {
			t.Fatalf("IPv6AddrInCIDR returned %v, not in %v", ip, ipNet)
		}
	}
}

func TestNetworkEUI64Bits(t *testing.T) {
	eui := randomizer.Network.EUI64()
	if len(eui) != 8 {
		t.Fatalf("EUI64 length = %d, want 8", len(eui))
	}
	if eui[0]&0x02 == 0 {
		t.Fatal("EUI64 U/L bit not set (should be locally administered)")
	}
	if eui[0]&0x01 != 0 {
		t.Fatal("EUI64 I/G bit set (should be unicast)")
	}
}

func TestNetworkEUI64FromMAC(t *testing.T) {
	mac := net.HardwareAddr{0x00, 0x1A, 0x2B, 0x3C, 0x4D, 0x5E}
	eui := randomizer.Network.EUI64FromMAC(mac)
	if len(eui) != 8 {
		t.Fatalf("EUI64FromMAC length = %d, want 8", len(eui))
	}
	if eui[0] != mac[0]^0x02 || eui[1] != mac[1] || eui[2] != mac[2] {
		t.Fatalf("EUI64FromMAC OUI mismatch: got %X %X %X", eui[0], eui[1], eui[2])
	}
	if eui[3] != 0xFF || eui[4] != 0xFE {
		t.Fatalf("EUI64FromMAC FFFE bytes = 0x%02X 0x%02X, want 0xFF 0xFE", eui[3], eui[4])
	}
	if eui[5] != mac[3] || eui[6] != mac[4] || eui[7] != mac[5] {
		t.Fatalf("EUI64FromMAC NIC octets mismatch")
	}
	if randomizer.Network.EUI64FromMAC(net.HardwareAddr{0x00}) != nil {
		t.Fatal("EUI64FromMAC with invalid MAC should return nil")
	}
}

var (
	benchPort  uint16
	benchUUID  [16]byte
	benchUUIDS string
	benchIPNet *net.IPNet
)

func BenchmarkNetworkPort(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchPort = randomizer.Network.Port(randomizer.AnyPort)
	}
}

func BenchmarkNetworkEphemeralPort(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchPort = randomizer.Network.Port(randomizer.EphemeralPort)
	}
}

func BenchmarkNetworkUUIDv4(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchUUID = randomizer.Network.UUIDv4()
	}
}

func BenchmarkNetworkUUIDv4String(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchUUIDS = randomizer.Network.UUIDv4String()
	}
}

func BenchmarkNetworkIPv4CIDR(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchIPNet = randomizer.Network.IPv4CIDR(24)
	}
}

func BenchmarkNetworkEUI64(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchMAC = randomizer.Network.EUI64()
	}
}
