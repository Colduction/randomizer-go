package randomizer

import (
	"bytes"
	"encoding/binary"
	"math/bits"
	"net"
	"net/netip"
	"testing"
)

func TestRealOUIsTable(t *testing.T) {
	if len(realOUIs) < 128 {
		t.Fatalf("realOUIs has %d entries, want at least 128", len(realOUIs))
	}
	for i, oui := range realOUIs {
		if oui[0]&0x03 != 0 {
			t.Fatalf("realOUIs[%d] = %x is not universal unicast", i, oui)
		}
		if i > 0 && bytes.Compare(realOUIs[i-1][:], oui[:]) >= 0 {
			t.Fatalf("realOUIs[%d] = %x is not sorted after %x", i, oui, realOUIs[i-1])
		}
	}
}

func TestIsPublicIPv6(t *testing.T) {
	cases := map[string]bool{
		"2001::1": false, "2001:2::1": false, "2001:1ff::1": false, "2001:200::1": true,
		"2001:db8::1": false, "2001:db9::1": true, "2002::1": false, "3fff::1": false,
		"3fff:fff::1": false, "3fff:1000::1": true, "5f00::1": false, "2620:4f:8000::1": false,
		"2620:4f:8001::1": true, "2a00::1": true, "2606:4700::1": true,
	}
	for s, want := range cases {
		hi := uint64(0)
		a := netip.MustParseAddr(s).As16()
		for _, b := range a[:8] {
			hi = hi<<8 | uint64(b)
		}
		if got := isPublicIPv6(hi); got != want {
			t.Fatalf("isPublicIPv6(%s) = %t, want %t", s, got, want)
		}
	}
}

func TestIsPublicASN(t *testing.T) {
	cases := map[uint32]bool{
		0: false, 1: true, 23455: true, 23456: false, 23457: true, 64495: true, 64496: false,
		65535: false, 65552: false, 100000: false, 131071: false, 131072: true,
		4199999999: true, 4200000000: false, 4294967295: false,
	}
	for a, want := range cases {
		if got := isPublicASN(a); got != want {
			t.Fatalf("isPublicASN(%d) = %t, want %t", a, got, want)
		}
	}
}

func TestBluetoothHashVector(t *testing.T) {
	irk := []byte{0xec, 0x02, 0x34, 0xa3, 0x57, 0xc8, 0xad, 0x05, 0x34, 0x10, 0x10, 0xa6, 0x0a, 0x39, 0x7d, 0x9b}
	if got := bluetoothHash(irk, 0x708194); got != 0x0dfbaa {
		t.Fatalf("bluetoothHash = %06x, want 0dfbaa", got)
	}
}

func TestBatchSpecs(t *testing.T) {
	for n := uint64(1); n <= 256; n++ {
		spec := batchSpecs[n]
		prod := uint64(1)
		for range spec.k {
			prod *= n
		}
		if spec.k == 0 || prod != spec.prod {
			t.Fatalf("batchSpecs[%d] = %+v, want n^k == prod", n, spec)
		}
		if n > 1 && -spec.prod%spec.prod > 1<<60 {
			t.Fatalf("batchSpecs[%d] rejects more than 1/16 of words", n)
		}
		if hi, _ := bits.Mul64(spec.prod, n); n > 1 && hi == 0 && -(spec.prod*n)%(spec.prod*n) <= 1<<60 {
			t.Fatalf("batchSpecs[%d] could use k+1", n)
		}
	}
	if batchSpecs[9].k < 18 || batchSpecs[61].k < 10 || batchSpecs[25].k < 13 {
		t.Fatalf("batches smaller than expected: %+v %+v %+v", batchSpecs[9], batchSpecs[61], batchSpecs[25])
	}
}

func generatorAt(index, bound uint64) *Generator {
	word := ^uint64(0)
	if index < bound-1 {
		var rem uint64
		word, rem = bits.Div64(index+1, 0, bound)
		if rem == 0 {
			word--
		}
	}
	return FromReader(bytes.NewReader(binary.LittleEndian.AppendUint64(nil, word)))
}

func TestNetworkJointHardwareChoices(t *testing.T) {
	for i, oui := range realOUIs {
		const macExtensions = 1<<24 - 64
		for _, rank := range []uint64{0, 0x9e8aff, 0x9e8b00, macExtensions - 1} {
			g := generatorAt(uint64(i)*macExtensions+rank, uint64(len(realOUIs))*macExtensions)
			mac := g.Network.Hardware(nil, HardwareMACRealOUI, HardwareOptions{})
			lap := uint32(mac[3])<<16 | uint32(mac[4])<<8 | uint32(mac[5])
			want := uint32(rank)
			if want >= 0x9e8b00 {
				want += 64
			}
			if !bytes.Equal(mac[:3], oui[:]) || lap != want {
				t.Fatalf("MAC choice %d/%d = %x, want OUI %x and LAP %06x", i, rank, mac, oui, want)
			}
		}
		const euiExtensions = 1<<40 - 2<<24
		for _, rank := range []uint64{0, 0x0102030405, euiExtensions - 1} {
			g := generatorAt(uint64(i)*euiExtensions+rank, uint64(len(realOUIs))*euiExtensions)
			eui := g.Network.Hardware(nil, HardwareEUI64RealOUI, HardwareOptions{})
			extension := uint64(eui[3])<<32 | uint64(eui[4])<<24 | uint64(eui[5])<<16 | uint64(eui[6])<<8 | uint64(eui[7])
			if !bytes.Equal(eui[:3], oui[:]) || extension != rank {
				t.Fatalf("EUI choice %d/%d = %x, want OUI %x and extension %010x", i, rank, eui, oui, rank)
			}
		}
	}
}

func TestNetworkPrivateBlockChoices(t *testing.T) {
	for block := range uint64(3) {
		for _, host := range []uint64{0, 0x123456, 1<<24 - 1} {
			g := generatorAt(block<<24|host, 3<<24)
			ip := g.Network.IP(nil, IPv4Private)
			want := [4]byte{10, byte(host >> 16), byte(host >> 8), byte(host)}
			switch block {
			case 1:
				want[0], want[1] = 172, 16+(byte(host>>16)&15)
			case 2:
				want[0], want[1] = 192, 168
			}
			if !bytes.Equal(ip, want[:]) {
				t.Fatalf("private choice %d/%x = %v, want %v", block, host, ip, net.IP(want[:]))
			}
		}
	}
}

func TestNetworkPublicIPv4RegistryBoundaries(t *testing.T) {
	prefixes := []string{
		"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16",
		"172.16.0.0/12", "192.0.0.0/24", "192.0.2.0/24", "192.31.196.0/24", "192.52.193.0/24",
		"192.88.99.0/24", "192.168.0.0/16", "192.175.48.0/24", "198.18.0.0/15",
		"198.51.100.0/24", "203.0.113.0/24", "224.0.0.0/4", "240.0.0.0/4",
	}
	for _, raw := range prefixes {
		prefix := netip.MustParsePrefix(raw)
		base := prefix.Addr().As4()
		first := binary.BigEndian.Uint32(base[:])
		last := first | ^uint32(0)>>prefix.Bits()
		if isPublicIPv4(first) || isPublicIPv4(last) {
			t.Fatalf("public classifier accepts an endpoint of %s", raw)
		}
		for _, neighbor := range []uint32{first - 1, last + 1} {
			var b [4]byte
			binary.BigEndian.PutUint32(b[:], neighbor)
			addr := netip.AddrFrom4(b)
			want := true
			for _, excluded := range prefixes {
				if netip.MustParsePrefix(excluded).Contains(addr) {
					want = false
				}
			}
			if isPublicIPv4(neighbor) != want {
				t.Fatalf("public classifier at %v = %t, want %t", addr, !want, want)
			}
		}
	}
}

func TestTelecomJointMEIDChoices(t *testing.T) {
	for region := range uint64(0x60) {
		for _, serial := range []uint64{0, 0x123456789abc, 1<<48 - 1} {
			index := region<<48 | serial
			g := generatorAt(index, 0x60<<48)
			got := g.Telecom.String(MEID, TelecomOptions{})
			var want [14]byte
			r := byte(region) + 0xa0
			want[0], want[1] = uhexdict[r>>4], uhexdict[r&15]
			for i := range 12 {
				want[i+2] = uhexdict[(serial>>(i*4))&15]
			}
			if got != string(want[:]) {
				t.Fatalf("MEID choice %x/%x = %q, want %q", region, serial, got, want)
			}
		}
	}
}
