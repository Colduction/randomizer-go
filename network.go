package randomizer

import (
	"net"
	"unsafe"
)

type network struct{}

// Network provides random network addresses, identifiers, and port values
// using the active [Provider] selected by [SetProvider].
var Network network

func fillRandomBytes(out []byte, provider Provider) {
	var (
		i int
		n = len(out)
	)
	for ; i+8 <= n; i += 8 {
		*(*uint64)(unsafe.Pointer(&out[i])) = provider.Sum64()
	}
	if i < n {
		x := provider.Sum64()
		for j := i; j < n; j++ {
			out[j] = byte(x)
			x >>= 8
		}
	}
}

// UnicastType identifies the kind of IPv6 unicast address generated through
// [Network].
type UnicastType uint8

const (
	// GlobalType identifies a global IPv6 unicast address.
	GlobalType UnicastType = iota + 1
	// LinkLocalType identifies a link-local IPv6 unicast address.
	LinkLocalType
	// SiteLocalType identifies a site-local IPv6 unicast address.
	SiteLocalType
	// UniqueLocalType identifies a unique-local IPv6 unicast address.
	UniqueLocalType
	// PrivateType is an alias for UniqueLocalType.
	PrivateType UnicastType = UniqueLocalType
)

// MulticastScope identifies the IPv6 multicast scope generated through
// [Network].
type MulticastScope uint8

const (
	// InterfaceLocalScope identifies interface-local multicast scope.
	InterfaceLocalScope MulticastScope = 0x1
	// LinkLocalScope identifies link-local multicast scope.
	LinkLocalScope MulticastScope = 0x2
	// AdminLocalScope identifies admin-local multicast scope.
	AdminLocalScope MulticastScope = 0x4
	// SiteLocalScope identifies site-local multicast scope.
	SiteLocalScope MulticastScope = 0x5
	// OrgLocalScope identifies organization-local multicast scope.
	OrgLocalScope MulticastScope = 0x8
	// GlobalScope identifies global multicast scope.
	GlobalScope MulticastScope = 0xE
)

// IPv4Addr returns a random 4-byte [net.IP] IPv4 address.
func (network) IPv4Addr() net.IP {
	b := make(net.IP, net.IPv4len)
	fillRandomBytes(b, currentProvider())
	return b
}

// IPv6Addr returns a random 16-byte [net.IP] IPv6 address.
func (network) IPv6Addr() net.IP {
	b := make(net.IP, net.IPv6len)
	fillRandomBytes(b, currentProvider())
	return b
}

// MACAddr returns a random 6-byte [net.HardwareAddr].
// local sets the U/L bit (locally administered); multicast sets the I/G bit.
func (network) MACAddr(local, multicast bool) net.HardwareAddr {
	b := make(net.HardwareAddr, 6)
	fillRandomBytes(b, currentProvider())
	if local {
		b[0] = b[0] | 0x02
	} else {
		b[0] = b[0] &^ 0x02
	}
	if multicast {
		b[0] = b[0] | 0x01
	} else {
		b[0] = b[0] &^ 0x01
	}
	return net.HardwareAddr(b)
}

// IPv6UnicastAddr returns a random IPv6 unicast address of the given
// [UnicastType].
func (network) IPv6UnicastAddr(unicastType UnicastType) net.IP {
	b := make(net.IP, net.IPv6len)
	fillRandomBytes(b, currentProvider())
	switch unicastType {
	case GlobalType:
		b[0] = (b[0] & 0x1F) | 0x20
	case LinkLocalType:
		b[0] = 0xFE
		b[1] = (b[1] & 0x3F) | 0x80
	case SiteLocalType:
		b[0] = 0xFE
		b[1] = (b[1] & 0x3F) | 0xC0
	case UniqueLocalType:
		b[0] = 0xFD
	}
	return b
}

// IPv6MulticastAddr returns a random IPv6 multicast address with the given
// [MulticastScope].
func (network) IPv6MulticastAddr(scope MulticastScope) net.IP {
	b := make(net.IP, net.IPv6len)
	fillRandomBytes(b, currentProvider())
	b[0] = 0xFF
	b[1] = uint8(scope) & 0x0F
	return b
}

// PortRange identifies a range of port numbers as defined by IANA and RFC 6335.
type PortRange uint8

const (
	// AnyPort identifies the full port range [0, 65535].
	AnyPort PortRange = iota
	// PrivilegedPort identifies the IANA well-known port range [1, 1023].
	PrivilegedPort
	// RegisteredPort identifies the IANA registered port range [1024, 49151].
	RegisteredPort
	// EphemeralPort identifies the dynamic and private port range [49152, 65535].
	EphemeralPort
)

// Port returns a random port number within the given [PortRange].
func (network) Port(portRange PortRange) uint16 {
	provider := currentProvider()
	switch portRange {
	case PrivilegedPort:
		return uint16(uniformUint64n(1023, provider)) + 1
	case RegisteredPort:
		return uint16(uniformUint64n(48128, provider)) + 1024
	case EphemeralPort:
		return uint16(provider.Sum64()>>50) + 49152
	default:
		return uint16(provider.Sum64() >> 48)
	}
}

// VLANID returns a random 12-bit IEEE 802.1Q VLAN ID in [0, 4095].
// See https://standards.ieee.org/ieee/802.1Q/10323/
func (network) VLANID() uint16 {
	return uint16(currentProvider().Sum64() >> 52)
}

// UUIDv4 returns a random RFC 4122 version-4 UUID as a [16]byte value.
func (network) UUIDv4() [16]byte {
	var b [16]byte
	provider := currentProvider()
	*(*uint64)(unsafe.Pointer(&b[0])) = provider.Sum64()
	*(*uint64)(unsafe.Pointer(&b[8])) = provider.Sum64()
	b[6] = (b[6] & 0x0F) | 0x40 // version 4:          0100 xxxx
	b[8] = (b[8] & 0x3F) | 0x80 // RFC 4122 variant:   10xx xxxx
	return b
}

// uuidHexEncode encodes src into dst as lowercase hex pairs; len(dst) must equal 2*len(src).
func uuidHexEncode(dst []byte, src []byte) {
	for i, v := range src {
		dst[i<<1] = lhexdict[v>>4]
		dst[i<<1|1] = lhexdict[v&0x0F]
	}
}

// UUIDv4String returns a random RFC 4122 version-4 UUID as a 36-character
// lowercase hexadecimal string in the standard 8-4-4-4-12 form.
func (n network) UUIDv4String() string {
	uuid := n.UUIDv4()
	b := make([]byte, 36)
	uuidHexEncode(b[0:8], uuid[0:4])
	b[8] = '-'
	uuidHexEncode(b[9:13], uuid[4:6])
	b[13] = '-'
	uuidHexEncode(b[14:18], uuid[6:8])
	b[18] = '-'
	uuidHexEncode(b[19:23], uuid[8:10])
	b[23] = '-'
	uuidHexEncode(b[24:36], uuid[10:16])
	return unsafe.String(unsafe.SliceData(b), 36)
}

// IPv4CIDR returns a random IPv4 [net.IPNet] with the prefix length clamped to
// [0, 32].
func (network) IPv4CIDR(prefixLen uint8) *net.IPNet {
	if prefixLen > 32 {
		prefixLen = 32
	}
	ab := make([]byte, net.IPv4len+net.IPv4len)
	*(*uint32)(unsafe.Pointer(&ab[0])) = uint32(currentProvider().Sum64())
	mask := net.CIDRMask(int(prefixLen), 32)
	ab[0] &= mask[0]
	ab[1] &= mask[1]
	ab[2] &= mask[2]
	ab[3] &= mask[3]
	return &net.IPNet{IP: net.IP(ab[:4]), Mask: mask}
}

// IPv6CIDR returns a random IPv6 [net.IPNet] with the prefix length clamped to
// [0, 128].
func (network) IPv6CIDR(prefixLen uint8) *net.IPNet {
	if prefixLen > 128 {
		prefixLen = 128
	}
	b := make([]byte, net.IPv6len)
	provider := currentProvider()
	*(*uint64)(unsafe.Pointer(&b[0])) = provider.Sum64()
	*(*uint64)(unsafe.Pointer(&b[8])) = provider.Sum64()
	mask := net.CIDRMask(int(prefixLen), 128)
	for i := range b {
		b[i] &= mask[i]
	}
	return &net.IPNet{IP: net.IP(b), Mask: mask}
}

// IPv4AddrInCIDR returns a random [net.IP] host address within ipNet.
// It returns nil if ipNet is not a valid IPv4 network.
func (network) IPv4AddrInCIDR(ipNet *net.IPNet) net.IP {
	ip4 := ipNet.IP.To4()
	mask := ipNet.Mask
	if ip4 == nil || len(mask) != net.IPv4len {
		return nil
	}
	b := make(net.IP, net.IPv4len)
	x := uint32(currentProvider().Sum64())
	b[0] = ip4[0] | (byte(x) &^ mask[0])
	b[1] = ip4[1] | (byte(x>>8) &^ mask[1])
	b[2] = ip4[2] | (byte(x>>16) &^ mask[2])
	b[3] = ip4[3] | (byte(x>>24) &^ mask[3])
	return b
}

// IPv6AddrInCIDR returns a random [net.IP] host address within ipNet.
// It returns nil if ipNet is not a valid IPv6 network.
func (network) IPv6AddrInCIDR(ipNet *net.IPNet) net.IP {
	ip6 := ipNet.IP.To16()
	mask := ipNet.Mask
	if ip6 == nil || len(mask) != net.IPv6len {
		return nil
	}
	var (
		b        = make(net.IP, net.IPv6len)
		provider = currentProvider()
		x0       = provider.Sum64()
		x1       = provider.Sum64()
	)
	for i := range 8 {
		b[i] = ip6[i] | (byte(x0>>(uint(i)<<3)) &^ mask[i])
	}
	for i := 8; i < 16; i++ {
		b[i] = ip6[i] | (byte(x1>>(uint(i-8)<<3)) &^ mask[i])
	}
	return b
}

// EUI64 returns a random 8-byte locally-administered unicast
// [net.HardwareAddr].
// See https://standards.ieee.org/content/dam/ieee-standards/standards/web/documents/tutorials/eui.pdf
func (network) EUI64() net.HardwareAddr {
	b := make(net.HardwareAddr, 8)
	*(*uint64)(unsafe.Pointer(&b[0])) = currentProvider().Sum64()
	b[0] |= 0x02
	b[0] &^= 0x01
	return b
}

// EUI64FromMAC returns the [net.HardwareAddr] EUI-64 identifier derived from
// mac by inserting
// 0xFF 0xFE and flipping the U/L bit as specified in RFC 4291 appendix A.
// It returns nil if mac is not exactly 6 bytes.
func (network) EUI64FromMAC(mac net.HardwareAddr) net.HardwareAddr {
	if len(mac) != 6 {
		return nil
	}
	b := make(net.HardwareAddr, 8)
	b[0] = mac[0] ^ 0x02
	b[1] = mac[1]
	b[2] = mac[2]
	b[3] = 0xFF
	b[4] = 0xFE
	b[5] = mac[3]
	b[6] = mac[4]
	b[7] = mac[5]
	return b
}
