package randomizer_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/colduction/randomizer-go"
)

func ExampleNetwork_addr() {
	addr := randomizer.Network.Addr(randomizer.IPv6Public)
	prefix := randomizer.Network.Prefix(randomizer.IPv6CIDR, 56)
	host := randomizer.Network.AddrIn(prefix)
	fmt.Println(addr.Is6(), prefix.Contains(host))
}

func ExampleNetwork_hardware() {
	static := randomizer.Network.Hardware(nil, randomizer.HardwareBluetoothStatic, randomizer.HardwareOptions{})
	aai := randomizer.Network.Hardware(nil, randomizer.HardwareMACAAI, randomizer.HardwareOptions{})
	eli := randomizer.Network.Hardware(nil, randomizer.HardwareMACELI, randomizer.HardwareOptions{CID: [3]byte{0x0a, 0x1b, 0x2c}})
	fmt.Println(static[0]&0xc0 == 0xc0, aai[0]&0x0f == 0x02, net.HardwareAddr(eli[:3]))
}

func ExampleNetwork_iPv6EUI64() {
	mac := net.HardwareAddr{0x00, 0x1a, 0x2b, 0x3c, 0x4d, 0x5e}
	ip := randomizer.Network.IPv6EUI64(nil, randomizer.IPv6LinkLocal, mac)
	fmt.Println(ip)
}

func ExampleTelecom_string() {
	imei := randomizer.Telecom.String(randomizer.IMEI, randomizer.TelecomOptions{TAC: "35123456"})
	iccid := randomizer.Telecom.String(randomizer.ICCID, randomizer.TelecomOptions{IIN: "8901", Length: 20})
	fmt.Println(len(imei), imei[:8], len(iccid), iccid[:4])
}

func ExampleWord_custom() {
	s := randomizer.Word.Custom("ABC", 6)
	fmt.Println(len(s))
}

func ExampleGenerator() {
	fast := randomizer.New()
	fmt.Println(fast.Network.Addr(randomizer.IPv4Private).IsPrivate())
	g := randomizer.NewSecure()
	fmt.Println(g.Word.String(randomizer.Base64URLAlphabet, 32))
	seeded := randomizer.NewSeeded([32]byte{1})
	values := []int{1, 2, 3, 4}
	seeded.Shuffle(values)
	fmt.Println(values)
}

func TestExampleBehavior(t *testing.T) {
	prefix := randomizer.Network.Prefix(randomizer.IPv6CIDR, 56)
	if !prefix.Contains(randomizer.Network.AddrIn(prefix)) {
		t.Fatal("generated host is outside prefix")
	}
	mac := net.HardwareAddr{0x00, 0x1a, 0x2b, 0x3c, 0x4d, 0x5e}
	if got := randomizer.Network.IPv6EUI64(nil, randomizer.IPv6LinkLocal, mac).String(); got != "fe80::21a:2bff:fe3c:4d5e" {
		t.Fatalf("EUI-64 example = %s", got)
	}
	imei := randomizer.Telecom.String(randomizer.IMEI, randomizer.TelecomOptions{TAC: "35123456"})
	iccid := randomizer.Telecom.String(randomizer.ICCID, randomizer.TelecomOptions{IIN: "8901", Length: 20})
	if len(imei) != 15 || imei[:8] != "35123456" || len(iccid) != 20 || iccid[:4] != "8901" {
		t.Fatal("telecom example produced invalid lengths/prefixes")
	}
	if len(randomizer.Word.Custom("ABC", 6)) != 6 || len(randomizer.NewSecure().Word.String(randomizer.Base64URLAlphabet, 32)) != 32 {
		t.Fatal("word example produced invalid length")
	}
}
