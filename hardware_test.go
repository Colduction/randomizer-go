package randomizer_test

import (
	"bytes"
	"net"
	"testing"

	"github.com/colduction/randomizer-go"
)

func TestNetworkHardware(t *testing.T) {
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
		mac := randomizer.Network.Hardware(nil, randomizer.HardwareMAC, randomizer.HardwareOptions{
			Local:     tc.local,
			Multicast: tc.multicast,
		})
		if len(mac) != 6 {
			t.Fatalf("Hardware(HardwareMAC) length = %d, want 6", len(mac))
		}
		if got := mac[0]&0x02 != 0; got != tc.local {
			t.Fatalf("HardwareMAC local bit = %t, want %t", got, tc.local)
		}
		if got := mac[0]&0x01 != 0; got != tc.multicast {
			t.Fatalf("HardwareMAC multicast bit = %t, want %t", got, tc.multicast)
		}
	}

	var buf [8]byte
	got := randomizer.Network.Hardware(buf[:0], randomizer.HardwareEUI64, randomizer.HardwareOptions{})
	if len(got) != 8 || &got[0] != &buf[0] || got[0]&0x03 != 0x02 {
		t.Fatalf("Hardware(HardwareEUI64) = %x, want local unicast in caller buffer", got)
	}
}

func TestNetworkHardwareOUI(t *testing.T) {
	oui := [3]byte{0x02, 0x1a, 0x3f}
	mac := randomizer.Network.Hardware(nil, randomizer.HardwareMACOUI, randomizer.HardwareOptions{OUI: oui})
	if len(mac) != 6 || mac[0] != oui[0] || mac[1] != oui[1] || mac[2] != oui[2] {
		t.Fatalf("Hardware(HardwareMACOUI) = %x, want prefix %x", mac, oui)
	}

	for range 1000 {
		real := randomizer.Network.Hardware(nil, randomizer.HardwareMACRealOUI, randomizer.HardwareOptions{})
		if len(real) != 6 || real[0]&0x03 != 0 {
			t.Fatalf("Hardware(HardwareMACRealOUI) = %x, want universal unicast", real)
		}
		if real[3] == 0x9e && real[4] == 0x8b && real[5] <= 0x3f {
			t.Fatalf("Hardware(HardwareMACRealOUI) = %x uses a Bluetooth inquiry LAP", real)
		}
	}

}

func TestNetworkHardwarePrefix(t *testing.T) {
	prefix := []byte{0x70, 0xb3, 0xd5, 0xab, 0xcd, 0xef}
	cases := []struct {
		bits uint8
		want []byte
		mask []byte
	}{
		{24, prefix[:3], []byte{0xff, 0xff, 0xff}},
		{28, prefix[:4], []byte{0xff, 0xff, 0xff, 0xf0}},
		{36, prefix[:5], []byte{0xff, 0xff, 0xff, 0xff, 0xf0}},
		{48, prefix, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
		{1, prefix[:1], []byte{0x80}},
	}
	for _, tc := range cases {
		options := randomizer.HardwareOptions{Prefix: prefix, PrefixBits: tc.bits}
		mac := randomizer.Network.Hardware(nil, randomizer.HardwareMACPrefix, options)
		if len(mac) != 6 {
			t.Fatalf("Hardware(HardwareMACPrefix, %d) = %x, want 6 bytes", tc.bits, mac)
		}
		for i, m := range tc.mask {
			if mac[i]&m != tc.want[i]&m {
				t.Fatalf("Hardware(HardwareMACPrefix, %d) = %x, want prefix %x", tc.bits, mac, tc.want)
			}
		}
	}
	for _, options := range []randomizer.HardwareOptions{
		{Prefix: prefix, PrefixBits: 0},
		{Prefix: prefix, PrefixBits: 49},
		{Prefix: prefix[:2], PrefixBits: 24},
	} {
		if got := randomizer.Network.Hardware(nil, randomizer.HardwareMACPrefix, options); got != nil {
			t.Fatalf("Hardware(HardwareMACPrefix) with invalid options = %x, want nil", got)
		}
	}
}

func TestNetworkHardwareSLAP(t *testing.T) {
	for range 100 {
		aai := randomizer.Network.Hardware(nil, randomizer.HardwareMACAAI, randomizer.HardwareOptions{})
		if len(aai) != 6 || aai[0]&0x0f != 0x02 {
			t.Fatalf("Hardware(HardwareMACAAI) = %x, want second hex digit 2", aai)
		}
		sai := randomizer.Network.Hardware(nil, randomizer.HardwareMACSAI, randomizer.HardwareOptions{})
		if len(sai) != 6 || sai[0]&0x0f != 0x0e {
			t.Fatalf("Hardware(HardwareMACSAI) = %x, want second hex digit E", sai)
		}
	}
	cid := [3]byte{0x0a, 0x1b, 0x2c}
	eli := randomizer.Network.Hardware(nil, randomizer.HardwareMACELI, randomizer.HardwareOptions{CID: cid})
	if len(eli) != 6 || eli[0] != cid[0] || eli[1] != cid[1] || eli[2] != cid[2] {
		t.Fatalf("Hardware(HardwareMACELI) = %x, want CID prefix %x", eli, cid)
	}
	if got := randomizer.Network.Hardware(nil, randomizer.HardwareMACELI, randomizer.HardwareOptions{CID: [3]byte{0x00, 0x1b, 0x2c}}); got != nil {
		t.Fatalf("Hardware(HardwareMACELI) with non-CID = %x, want nil", got)
	}
}

func TestNetworkHardwareEUI64RealOUI(t *testing.T) {
	for range 1000 {
		eui := randomizer.Network.Hardware(nil, randomizer.HardwareEUI64RealOUI, randomizer.HardwareOptions{})
		if len(eui) != 8 || eui[0]&0x03 != 0 {
			t.Fatalf("Hardware(HardwareEUI64RealOUI) = %x, want universal unicast EUI-64", eui)
		}
		if eui[3] == 0xff && eui[4] >= 0xfe {
			t.Fatalf("Hardware(HardwareEUI64RealOUI) = %x uses an encapsulation extension", eui)
		}
	}

}

func TestNetworkHardwareEUI64FromMAC(t *testing.T) {
	mac := net.HardwareAddr{0x00, 0x1a, 0x2b, 0x3c, 0x4d, 0x5e}
	eui := randomizer.Network.Hardware(nil, randomizer.HardwareEUI64FromMAC, randomizer.HardwareOptions{MAC: mac})
	if !bytes.Equal(eui, []byte{0x00, 0x1a, 0x2b, 0xff, 0xfe, 0x3c, 0x4d, 0x5e}) {
		t.Fatalf("Hardware(HardwareEUI64FromMAC) = %x, want IEEE EUI-64 with U/L kept", eui)
	}
	modified := randomizer.Network.Hardware(nil, randomizer.HardwareModifiedEUI64FromMAC, randomizer.HardwareOptions{MAC: mac})
	if !bytes.Equal(modified, []byte{0x02, 0x1a, 0x2b, 0xff, 0xfe, 0x3c, 0x4d, 0x5e}) {
		t.Fatalf("Hardware(HardwareModifiedEUI64FromMAC) = %x, want U/L inverted", modified)
	}
	for _, kind := range []randomizer.HardwareKind{randomizer.HardwareEUI64FromMAC, randomizer.HardwareModifiedEUI64FromMAC} {
		if got := randomizer.Network.Hardware(nil, kind, randomizer.HardwareOptions{MAC: net.HardwareAddr{0x00}}); got != nil {
			t.Fatalf("Hardware(%d) invalid MAC = %x, want nil", kind, got)
		}
	}
}

func TestNetworkHardwareBluetooth(t *testing.T) {
	for range 1000 {
		static := randomizer.Network.Hardware(nil, randomizer.HardwareBluetoothStatic, randomizer.HardwareOptions{})
		if len(static) != 6 || static[0]&0xc0 != 0xc0 {
			t.Fatalf("Hardware(HardwareBluetoothStatic) = %x, want top bits 11", static)
		}
		nrpa := randomizer.Network.Hardware(nil, randomizer.HardwareBluetoothNRPA, randomizer.HardwareOptions{})
		if len(nrpa) != 6 || nrpa[0]&0xc0 != 0x00 {
			t.Fatalf("Hardware(HardwareBluetoothNRPA) = %x, want top bits 00", nrpa)
		}
		rpa := randomizer.Network.Hardware(nil, randomizer.HardwareBluetoothRPA, randomizer.HardwareOptions{})
		if len(rpa) != 6 || rpa[0]&0xc0 != 0x40 {
			t.Fatalf("Hardware(HardwareBluetoothRPA) = %x, want top bits 01", rpa)
		}
	}
	if got := randomizer.Network.Hardware(nil, randomizer.HardwareBluetoothRPA, randomizer.HardwareOptions{IRK: []byte{1, 2, 3}}); got != nil {
		t.Fatalf("Hardware(HardwareBluetoothRPA) with short IRK = %x, want nil", got)
	}
}

func TestNetworkHardwareBluetoothRejection(t *testing.T) {
	const ones46 = 1<<46 - 1
	cases := []struct {
		kind randomizer.HardwareKind
		seq  []uint64
		want []byte
	}{
		{randomizer.HardwareBluetoothStatic, []uint64{0, ones46, 5}, []byte{0xc0, 0, 0, 0, 0, 5}},
		{randomizer.HardwareBluetoothNRPA, []uint64{0, ones46, 5}, []byte{0, 0, 0, 0, 0, 5}},
		{randomizer.HardwareBluetoothRPA, []uint64{0, 1<<22 - 1, 0x308194}, []byte{0x70, 0x81, 0x94, 0x0d, 0xfb, 0xaa}},
	}
	irk := []byte{0xec, 0x02, 0x34, 0xa3, 0x57, 0xc8, 0xad, 0x05, 0x34, 0x10, 0x10, 0xa6, 0x0a, 0x39, 0x7d, 0x9b}
	for _, tc := range cases {
		g := randomizer.FromSource(newSeqProvider(t, tc.seq...))
		got := g.Network.Hardware(nil, tc.kind, randomizer.HardwareOptions{IRK: irk})
		if !bytes.Equal(got, tc.want) {
			t.Fatalf("Hardware(%d) = %x, want %x", tc.kind, got, tc.want)
		}
	}
}

func BenchmarkNetworkHardwareMAC(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchMAC = randomizer.Network.Hardware(nil, randomizer.HardwareMAC, randomizer.HardwareOptions{
			Local:     true,
			Multicast: false,
		})
	}
}

func BenchmarkNetworkHardwareMACBuffered(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchMAC = randomizer.Network.Hardware(benchBuffer[:0], randomizer.HardwareMAC, randomizer.HardwareOptions{
			Local:     true,
			Multicast: false,
		})
	}
}

func BenchmarkNetworkHardwareRealOUI(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchMAC = randomizer.Network.Hardware(benchBuffer[:0], randomizer.HardwareMACRealOUI, randomizer.HardwareOptions{})
	}
}

func BenchmarkNetworkHardwareBluetoothStatic(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchMAC = randomizer.Network.Hardware(benchBuffer[:0], randomizer.HardwareBluetoothStatic, randomizer.HardwareOptions{})
	}
}

func BenchmarkNetworkHardwareBluetoothRPA(b *testing.B) {
	options := randomizer.HardwareOptions{IRK: []byte("0123456789abcdef")}
	b.ReportAllocs()
	for b.Loop() {
		benchMAC = randomizer.Network.Hardware(benchBuffer[:0], randomizer.HardwareBluetoothRPA, options)
	}
}
