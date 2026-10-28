package randomizer

import (
	"net"
	"testing"
)

func assertMaskMatchesSize(t *testing.T, mask net.IPMask) {
	t.Helper()
	_, width := mask.Size()
	want := width == 32 || width == 128
	addresses := []net.IP{net.IPv4(10, 1, 2, 3), net.IP{10, 1, 2, 3}}
	if len(mask) == net.IPv6len {
		addresses = []net.IP{net.ParseIP("2001:db8::1234"), net.ParseIP("::ffff:10.1.2.3")}
	}
	for _, address := range addresses {
		var dst [16]byte
		for i := range dst {
			dst[i] = 0xa5
		}
		network := &net.IPNet{IP: address, Mask: mask}
		got := Network.IPInCIDR(dst[:0], network)
		if (got != nil) != want {
			t.Fatalf("mask %x address %v accepted=%t; [net.IPMask.Size] width=%d", []byte(mask), address, got != nil, width)
		}
		if got != nil {
			if !network.Contains(got) {
				t.Fatalf("mask %x generates address %v outside %v", []byte(mask), got, network)
			}
			continue
		}
		for _, value := range dst {
			if value != 0xa5 {
				t.Fatalf("invalid mask %x changes destination", []byte(mask))
			}
		}
	}
}

func TestCIDRMaskAllPrefixesAndSingleBitFlips(t *testing.T) {
	for _, width := range []int{32, 128} {
		for prefix := 0; prefix <= width; prefix++ {
			mask := net.CIDRMask(prefix, width)
			assertMaskMatchesSize(t, mask)
			for bit := range width {
				mask[bit/8] ^= 0x80 >> (bit & 7)
				assertMaskMatchesSize(t, mask)
				mask[bit/8] ^= 0x80 >> (bit & 7)
			}
		}
	}
}

func TestCIDRMaskReducedByteExhaustive(t *testing.T) {
	for _, length := range []int{net.IPv4len, net.IPv6len} {
		for position := range length {
			mask := make(net.IPMask, length)
			for i := range position {
				mask[i] = 0xff
			}
			for value := range 256 {
				mask[position] = byte(value)
				assertMaskMatchesSize(t, mask)
			}
		}
	}
}

func TestCIDRMaskInvalidLengths(t *testing.T) {
	for _, length := range []int{0, 1, 2, 3, 5, 8, 12, 15, 17, 32} {
		for _, value := range []byte{0, 0xf0, 0xff} {
			mask := make(net.IPMask, length)
			for i := range mask {
				mask[i] = value
			}
			assertMaskMatchesSize(t, mask)
		}
	}
}
