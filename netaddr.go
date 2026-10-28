package randomizer

import (
	"encoding/binary"
	"net"
	"net/netip"
)

// Addr returns a random address of the selected kind as a [netip.Addr].
// It allocates nothing.
func (g NetworkGenerator) Addr(kind IPKind) netip.Addr {
	var buf [net.IPv6len]byte
	b := buf[:ipLen(kind)]
	if g.provider == nil {
		fillIP(b, kind, fastProvider{})
	} else {
		g.fillIP(b, kind)
	}
	if len(b) == net.IPv4len {
		return netip.AddrFrom4([net.IPv4len]byte(b))
	}
	return netip.AddrFrom16(buf)
}

// Prefix returns a random network prefix as a [netip.Prefix].
// It clamps IPv4 prefix lengths to [0, 32] and IPv6 prefix lengths to [0, 128]
// and allocates nothing.
func (g NetworkGenerator) Prefix(kind CIDRKind, bits uint8) netip.Prefix {
	bits, n := prefixLen(kind, bits)
	var buf [net.IPv6len]byte
	b := buf[:n]
	s := openProvider(g.provider)
	defer s.close()
	fillPrefix(b, kind, bits, s.provider)
	if n == net.IPv4len {
		return netip.PrefixFrom(netip.AddrFrom4([net.IPv4len]byte(b)), int(bits))
	}
	return netip.PrefixFrom(netip.AddrFrom16(buf), int(bits))
}

// AddrIn returns a uniform random address inside the prefix. It returns the zero
// [netip.Addr] when the prefix is invalid and allocates nothing.
func (g NetworkGenerator) AddrIn(prefix netip.Prefix) netip.Addr {
	if !prefix.IsValid() {
		return netip.Addr{}
	}
	bits := prefix.Bits()
	addr := prefix.Addr()
	if bits == addr.BitLen() {
		return addr
	}
	s := openProvider(g.provider)
	defer s.close()
	if addr.Is4() {
		base := addr.As4()
		var out [net.IPv4len]byte
		putIPv4(out[:], hostBits32(binary.BigEndian.Uint32(base[:]), ^uint32(0)<<(32-bits), uint32(s.Uint64())))
		return netip.AddrFrom4(out)
	}
	base := addr.As16()
	var hiMask, loMask, randomHi uint64
	if bits <= 64 {
		hiMask = ^uint64(0) << (64 - bits)
		if bits < 64 {
			randomHi = s.Uint64()
		}
	} else {
		hiMask, loMask = ^uint64(0), ^uint64(0)<<(128-bits)
	}
	var out [net.IPv6len]byte
	binary.BigEndian.PutUint64(out[:], hostBits64(binary.BigEndian.Uint64(base[:]), hiMask, randomHi))
	binary.BigEndian.PutUint64(out[8:], hostBits64(binary.BigEndian.Uint64(base[8:]), loMask, s.Uint64()))
	return netip.AddrFrom16(out)
}
