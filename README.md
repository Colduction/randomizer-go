# randomizer-go

[![Go Reference](https://pkg.go.dev/badge/github.com/colduction/randomizer-go.svg)](https://pkg.go.dev/github.com/colduction/randomizer-go)
[![Go version](https://img.shields.io/github/go-mod/go-version/colduction/randomizer-go)](./go.mod)
[![Dependencies: none](https://img.shields.io/badge/dependencies-none-brightgreen)](./go.mod)
[![GitHub License](https://img.shields.io/github/license/Colduction/randomizer-go)](./LICENSE)

**randomizer-go** generates random numbers, words, network addresses, telecom identifiers, permutations, and weighted samples. It offers caller-owned buffers, isolated generator instances, reproducible seeds, and an explicit cryptographically secure mode.

Package helpers use Go's public `math/rand/v2` source. Sampling algorithms avoid range-reduction bias when their source supplies uniform random words. Use `NewSecure` for secrets; the fast and seeded modes are pseudorandom generators rather than promises of physical true randomness.

## Features

- **Numbers**: generic signed and unsigned integers, bounded intervals, floats in `[0, 1)`, and six bulk fill operations.
- **Words**: independent symbols by default, weighted custom byte dictionaries, explicit no-adjacent-repeat mode, and append APIs.
- **Sampling**: in-place shuffle, ordered samples without replacement, and reusable exact integer-weighted distributions.
- **Network**: IPv4, IPv6, `netip` addresses and prefixes, hardware and Bluetooth identifiers, ports, VLANs, ASNs, VNI, flow labels, and MPLS labels.
- **Telecom**: IMEI, IMEISV, ICCID, and MEID with format constraints and applicable check digits.
- **Sources**: a fast package source, secure OS-backed generation, seeded ChaCha8, and direct constructors for custom sources and readers.

See [PERFORMANCE.md](PERFORMANCE.md) for current benchmark results, CPU and memory profiles, and reproducible profiling commands.

## Installation

```bash
go get github.com/colduction/randomizer-go@latest
```

## Quick Start

```go
package main

import (
    "fmt"

    "github.com/colduction/randomizer-go"
)

func main() {
    g := randomizer.New()
    fmt.Println(g.Int[int64]())
    fmt.Println(g.IntInterval(int64(1), int64(100)))
    fmt.Println(g.Word.String(randomizer.AlphaNumericAlphabet, 24))
    fmt.Println(g.Network.Addr(randomizer.IPv6Public))
    fmt.Println(g.Telecom.String(randomizer.IMEI, randomizer.TelecomOptions{}))

    secure := randomizer.NewSecure()
    fmt.Println(secure.Word.String(randomizer.Base64URLAlphabet, 32))

    var raw [32]byte
    _, _ = secure.Read(raw[:])
}
```

## Choose a Generator

Choose the source once and use the generator's numeric methods and `Word`, `Network`, and `Telecom` fields.

| Purpose                                    | Constructor       |
| ------------------------------------------ | ----------------- |
| General random data                        | `New()`           |
| Secret tokens and cryptographic randomness | `NewSecure()`     |
| Reproducible ChaCha8 output                | `NewSeeded(seed)` |

```go
fast := randomizer.New()
secure := randomizer.NewSecure()
seeded := randomizer.NewSeeded([32]byte{1, 2, 3})

n := seeded.Uint[uint64]()
address := secure.Network.Addr(randomizer.IPv6Any)
word := fast.Word.String(randomizer.HexLowerAlphabet, 16)
```

The zero `Generator` uses the fast package source. Package functions and the package `Word`, `Network`, and `Telecom` handles use that fixed source; constructing an instance does not change other callers. Share one generator when multiple callers need the same synchronized stream.

Generators permit concurrent use. Keep generator fields unchanged while sharing a generator. Equal seeds and identical sequential calls reproduce results within one package version; concurrent scheduling and output sequences across versions are unspecified. A caller-supplied all-zero seed is used unchanged and is not a secure seed.

## Numbers

Package functions use the fast source; instance methods use the selected source.

```go
n := randomizer.Int[int64]()
u := randomizer.Uint[uint64]()
bounded := randomizer.IntInterval(int64(-50), int64(50))
ubounded := randomizer.UintInterval(uint64(1), uint64(1000))
f32 := randomizer.Float32()
f64 := randomizer.Float64()
```

Intervals are `[min, max)`. Reversed bounds are swapped; equal bounds return the bound without consuming randomness. Signed intervals support the full signed endpoint range. Floats occupy uniform 24-bit and 53-bit grids in `[0, 1)`.

Bounded draws use multiplication with rejection to avoid modulo bias, following [Lemire's nearly divisionless method](https://arxiv.org/abs/1805.10941).

### Bulk Fills

All six operations fill existing storage. Their package forms and generator methods have the same names. Built-in synchronized sources lock once per batch, rather than once per output.

```go
g := randomizer.NewSeeded([32]byte{42})
signed := make([]int64, 256)
unsigned := make([]uint64, 256)
floats32 := make([]float32, 256)
floats64 := make([]float64, 256)

g.FillInt(signed)
g.FillUint(unsigned)
g.FillIntInterval(signed, -50, 50)
g.FillUintInterval(unsigned, 1, 1000)
g.FillFloat32(floats32)
g.FillFloat64(floats64)
```

Empty buffers consume no random words. Equal interval bounds fill a constant. Bounded batches use [Brackett-Rozinsky and Lemire's batched ranged sampling](https://arxiv.org/abs/2408.06213); power-of-two ranges extract disjoint groups of bits.

## Words

`IndependentSymbols` is the default: each byte is sampled with replacement. Repeated adjacent bytes are valid. `String` and `Bytes` allocate their output; `Append` reuses caller capacity. Lengths count bytes, and custom dictionaries are not interpreted as Unicode characters.

```go
s := randomizer.Word.String(randomizer.DecimalAlphabet, 12)
b := randomizer.Word.Bytes(randomizer.Base64URLAlphabet, 32)

buf := make([]byte, 0, 32)
buf = randomizer.Word.Append(buf, randomizer.AlphaNumericAlphabet, 32)

noRepeats := randomizer.Word.WithMode(randomizer.NoAdjacentRepeats)
constrained := noRepeats.String(randomizer.AlphaNumericAlphabet, 32)
```

`WithMode` returns a copy without changing the original handle or its source. Unknown modes select `IndependentSymbols`; unknown alphabets select `DecimalAlphabet`.

Built-in alphabets:

| Constant               | Bytes                                    |
| ---------------------- | ---------------------------------------- |
| `DecimalAlphabet`      | `0-9`                                    |
| `HexLowerAlphabet`     | `0-9a-f`                                 |
| `HexUpperAlphabet`     | `0-9A-F`                                 |
| `OctalAlphabet`        | `0-7`                                    |
| `LowerAlphabet`        | `a-z`                                    |
| `UpperAlphabet`        | `A-Z`                                    |
| `AlphaAlphabet`        | `a-zA-Z`                                 |
| `AlphaNumericAlphabet` | `0-9a-zA-Z`                              |
| `Base32Alphabet`       | RFC 4648 base32 without padding          |
| `Base64URLAlphabet`    | RFC 4648 URL-safe base64 without padding |

Power-of-two alphabets extract independent bit groups. Other built-in alphabets use fixed-length multiplication batches with exact rejection. Under `NoAdjacentRepeats`, unique dictionaries sample the other positions directly; weighted dictionaries use cumulative weights with the preceding byte's entire weight removed.

### Custom Dictionaries

Every dictionary position has equal weight, so duplicate bytes increase their probability.

```go
s := randomizer.Word.Custom("AABCXYZ9", 16)
s2 := randomizer.Word.CustomFromBytes([]byte("AABCXYZ9"), 16)
b := randomizer.Word.CustomBytes([]byte("AABCXYZ9"), 16)
b2 := randomizer.Word.CustomBytesFromString("AABCXYZ9", 16)

dst := []byte("id:")
dst = randomizer.Word.AppendCustom(dst, "AABCXYZ9", 16)
dst = randomizer.Word.AppendCustomFromBytes(dst[:3], []byte("AABCXYZ9"), 16)
```

With `"AABC"`, independent draws give `A` probability `1/2` and `B` and `C` probability `1/4` each. Under `NoAdjacentRepeats`, the first byte uses those weights; after `A`, `B` and `C` are equally likely; after `B`, `A` has probability `2/3` and `C` has probability `1/3`. That mode intentionally introduces dependence. Its restriction applies within the newly generated segment, without considering the destination prefix.

Nonpositive lengths and empty dictionaries return `""`, `nil`, or an unchanged append destination. Under `NoAdjacentRepeats`, requests longer than one byte also fail when the dictionary has only one distinct byte. Independent mode accepts constant dictionaries. Byte dictionaries are not retained; append destinations may overlap them, and sampling uses the original dictionary contents.

## Shuffle and Samples Without Replacement

```go
g := randomizer.NewSeeded([32]byte{7})
values := []string{"red", "green", "blue"}
g.Shuffle(values)

var selected [5]int
if err := g.SampleIndices(selected[:], 1000000); err != nil {
    panic(err)
}
```

`Shuffle` uses an unbiased in-place Fisher-Yates permutation with no auxiliary allocation. Nil and singleton slices need no random words. `SampleIndices` fills the supplied slice with distinct indices in `[0, population)`. Its output is an ordered random sample, not a sorted list; every ordered sample has equal probability.

Shuffle recycles unused randomness within each operation using the exact widening-multiplication method of [Draper and Saad](https://www.cs.cmu.edu/~fsaad/assets/papers/2026-DraperEtAl-SODA.pdf). Recycler state is not retained across operations. Current measurements and profiling commands are recorded in [PERFORMANCE.md](PERFORMANCE.md).

A negative population or a destination longer than the population returns `ErrInvalidSample` without changing the destination or consuming randomness. Empty samples succeed. Full samples permute the destination in place; partial samples use at most storage proportional to the sample size, independent of the population. Small samples use Floyd's ordered construction, described in [Bentley and Floyd's original paper](https://doi.org/10.1145/30401.315746). Package-level `Shuffle` and `SampleIndices` use the fast source.

## Weighted Sampling

```go
choices, err := randomizer.NewWeighted([]uint64{1, 3, 0})
if err != nil {
    panic(err)
}
g := randomizer.NewSeeded([32]byte{9})
index := choices.Index(g)
```

The example chooses indices `0` and `1` with probabilities `1/4` and `3/4`; index `2` is never chosen. `NewWeighted` copies the distribution into an immutable integer alias table. Construction takes linear time and storage, and draws take expected constant time without allocating, following the [Walker-Vose alias method](https://doi.org/10.1109/32.92917) with integer thresholds and overflow-safe arithmetic.

Empty or all-zero weights return `ErrInvalidWeights`. A total exceeding `math.MaxUint64` returns `ErrWeightOverflow`; totals equal to that limit are supported, including when intermediate scaled weights need more than 64 bits. Individual zero weights are valid. `Index(nil)` uses the fast source, and a nil or zero-value `Weighted` returns `-1`. The table can be shared concurrently; callers must use separate destination storage for mutable sampling operations.

## Network

Network APIs use small kind enums. Use `g.Network` to select an instance source; the examples below use the package fast source. Address slices reuse caller capacity, and the `netip` methods avoid output allocations. Bluetooth cipher creation and key changes can allocate.

### IP

```go
ip4 := randomizer.Network.IP(nil, randomizer.IPv4Any)
public4 := randomizer.Network.IP(nil, randomizer.IPv4Public)

var ipBuf [16]byte
ip6 := randomizer.Network.IP(ipBuf[:0], randomizer.IPv6Public)
mc6 := randomizer.Network.IP(ipBuf[:0], randomizer.IPv6LinkLocalMulticast)

addr := randomizer.Network.Addr(randomizer.IPv6UniqueLocal) // netip.Addr, zero allocations
```

`IPKind` values:

| Family         | Constants                                                                                                                                                                               |
| -------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| IPv4           | `IPv4Any`, `IPv4Private`, `IPv4LinkLocal`, `IPv4Multicast`, `IPv4Public`, `IPv4Loopback`, `IPv4CGNAT`, `IPv4Documentation`                                                              |
| IPv6 unicast   | `IPv6Any`, `IPv6Global`, `IPv6Public`, `IPv6LinkLocal`, `IPv6SiteLocal`, `IPv6UniqueLocal`, `IPv6Private`, `IPv6Documentation`                                                          |
| IPv6 multicast | `IPv6InterfaceLocalMulticast`, `IPv6LinkLocalMulticast`, `IPv6RealmLocalMulticast`, `IPv6AdminLocalMulticast`, `IPv6SiteLocalMulticast`, `IPv6OrgLocalMulticast`, `IPv6GlobalMulticast` |

Validity rules applied by the generators:

- `IPv4Public` excludes the special-purpose, private, documentation, multicast, and reserved ranges encoded in the package, following RFC 6890. It does not consult live routing tables.
- `IPv4LinkLocal` stays within `169.254.1.0`–`169.254.254.255` (RFC 3927 reserves the first and last /24).
- `IPv4Documentation` picks one of the three RFC 5737 blocks with equal probability; `IPv4CGNAT` is `100.64.0.0/10` (RFC 6598).
- `IPv6LinkLocal` is a strict `fe80::/64` prefix with a random interface identifier (RFC 4291 §2.5.6).
- `IPv6Public` is `2000::/3` minus the IANA IPv6 Special-Purpose Address Registry blocks (`2001::/23`, `2001:db8::/32`, `2002::/16`, `2620:4f:8000::/48`, `3fff::/20`, `5f00::/16`) and never uses the subnet-router or RFC 2526 reserved anycast interface identifiers.
- IPv6 multicast kinds set the transient flag (`T = 1`), the form required for non-IANA groups, so the second byte is `0x1s` where `s` is the scope.

`IPv6EUI64` builds a SLAAC-style address: a random prefix of the selected kind (`IPv6LinkLocal`, `IPv6Global`, `IPv6Public`, `IPv6UniqueLocal`, `IPv6SiteLocal`, or `IPv6Documentation`) with the RFC 4291 modified EUI-64 of a 6-byte MAC address or an 8-byte EUI-64.

```go
mac := net.HardwareAddr{0x00, 0x1a, 0x2b, 0x3c, 0x4d, 0x5e}
ll := randomizer.Network.IPv6EUI64(nil, randomizer.IPv6LinkLocal, mac) // fe80::21a:2bff:fe3c:4d5e
```

### Hardware

```go
mac := randomizer.Network.Hardware(nil, randomizer.HardwareMAC, randomizer.HardwareOptions{Local: true})
ouiMAC := randomizer.Network.Hardware(nil, randomizer.HardwareMACOUI, randomizer.HardwareOptions{OUI: [3]byte{0x3c, 0x22, 0xfb}})
realOUI := randomizer.Network.Hardware(nil, randomizer.HardwareMACRealOUI, randomizer.HardwareOptions{})

// IEEE MA-M (28-bit) assignment prefix.
mam := randomizer.Network.Hardware(nil, randomizer.HardwareMACPrefix, randomizer.HardwareOptions{
	Prefix: []byte{0x70, 0xb3, 0xd5, 0xa0}, PrefixBits: 28,
})

// IEEE 802c Structured Local Address Plan quadrants.
aai := randomizer.Network.Hardware(nil, randomizer.HardwareMACAAI, randomizer.HardwareOptions{})
eli := randomizer.Network.Hardware(nil, randomizer.HardwareMACELI, randomizer.HardwareOptions{CID: [3]byte{0x0a, 0x1b, 0x2c}})
sai := randomizer.Network.Hardware(nil, randomizer.HardwareMACSAI, randomizer.HardwareOptions{})

eui64 := randomizer.Network.Hardware(nil, randomizer.HardwareEUI64RealOUI, randomizer.HardwareOptions{})
fromMAC := randomizer.Network.Hardware(nil, randomizer.HardwareEUI64FromMAC, randomizer.HardwareOptions{MAC: mac})
iid := randomizer.Network.Hardware(nil, randomizer.HardwareModifiedEUI64FromMAC, randomizer.HardwareOptions{MAC: mac})

// Bluetooth device addresses.
irk := []byte("0123456789abcdef") // 16-byte Identity Resolving Key
public := randomizer.Network.Hardware(nil, randomizer.HardwareBluetoothPublic, randomizer.HardwareOptions{})
static := randomizer.Network.Hardware(nil, randomizer.HardwareBluetoothStatic, randomizer.HardwareOptions{})
nrpa := randomizer.Network.Hardware(nil, randomizer.HardwareBluetoothNRPA, randomizer.HardwareOptions{})
rpa := randomizer.Network.Hardware(nil, randomizer.HardwareBluetoothRPA, randomizer.HardwareOptions{IRK: irk})
```

`HardwareKind` values:

| Constant                       | Result                                                                                                                                |
| ------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------- |
| `HardwareMAC`                  | 6 random bytes; `Local` and `Multicast` set the U/L and I/G bits                                                                      |
| `HardwareMACOUI`               | `OUI` followed by 3 random bytes                                                                                                      |
| `HardwareMACRealOUI`           | An IEEE-registered OUI of a well-known vendor and a random extension outside the Bluetooth inquiry LAP range                          |
| `HardwareMACPrefix`            | The first `PrefixBits` (1–48) bits of `Prefix`, then random bits; covers MA-L, MA-M, MA-S, and CID assignments                        |
| `HardwareMACAAI`               | IEEE 802c Administratively Assigned Identifier (local unicast, second hex digit `2`)                                                  |
| `HardwareMACELI`               | IEEE 802c Extended Local Identifier from the 24-bit Company ID in `CID` (second hex digit `A`); `nil` unless `CID` has that form      |
| `HardwareMACSAI`               | IEEE 802c Standard Assigned Identifier (local unicast, second hex digit `E`)                                                          |
| `HardwareEUI64`                | 8 random bytes with the U/L bit set and the I/G bit clear                                                                             |
| `HardwareEUI64RealOUI`         | An IEEE-registered OUI with a 40-bit extension outside the `ff:fe` and `ff:ff` encapsulation values                                   |
| `HardwareEUI64FromMAC`         | The IEEE EUI-64 encapsulation of `MAC`: OUI, `ff fe`, extension, U/L bit kept                                                         |
| `HardwareModifiedEUI64FromMAC` | The RFC 4291 modified EUI-64 of `MAC`, the IPv6 interface identifier form with the U/L bit inverted                                   |
| `HardwareBluetoothPublic`      | Same as `HardwareMACRealOUI`                                                                                                          |
| `HardwareBluetoothStatic`      | Static random address: top two bits `11`, 46 random bits that are neither all zero nor all one                                        |
| `HardwareBluetoothNRPA`        | Non-resolvable private address: top two bits `00`, 46 random bits that are neither all zero nor all one                               |
| `HardwareBluetoothRPA`         | Resolvable private address: 22-bit `prand` under top bits `01`, then `hash = ah(IRK, prand)` (AES-128) when `IRK` is set, else random |

Bluetooth addresses are returned most-significant byte first, the order used in `XX:XX:XX:XX:XX:XX` notation. Resolvable private addresses with an `IRK` are computed with `crypto/aes`; the cipher for the most recent key is cached, and steady-state calls reuse cipher and scratch storage.

### Values

```go
port := randomizer.Network.Value(randomizer.RegisteredPort)
vlan := randomizer.Network.Value(randomizer.VLANIDUnreserved)
asn := randomizer.Network.Value(randomizer.ASNPublic)
private := randomizer.Network.Value(randomizer.ASNPrivate16)
vni := randomizer.Network.Value(randomizer.VNI)
flow := randomizer.Network.Value(randomizer.FlowLabel)
mpls := randomizer.Network.Value(randomizer.MPLSLabelUnreserved)
iid := randomizer.Network.Value(randomizer.IPv6InterfaceID)
```

`ValueKind` values:

| Constant              | Range                                                                                                                               |
| --------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| `AnyPort`             | `[0, 65535]`                                                                                                                        |
| `PrivilegedPort`      | `[1, 1023]`                                                                                                                         |
| `RegisteredPort`      | `[1024, 49151]`                                                                                                                     |
| `EphemeralPort`       | `[49152, 65535]`                                                                                                                    |
| `VLANID`              | `[0, 4095]`                                                                                                                         |
| `VLANIDUnreserved`    | `[1, 4094]` (802.1Q reserves 0 and 4095)                                                                                            |
| `ASN`                 | `[0, 4294967295]`                                                                                                                   |
| `ASNPublic`           | Assignable 32-bit ASN: excludes 0, 23456, 64496–131071, and 4200000000–4294967295 (RFC 5398, 6793, 6996, 7300, 7607, IANA reserved) |
| `ASNPublic16`         | `[1, 64495]` other than 23456                                                                                                       |
| `ASNPrivate16`        | `[64512, 65534]` (RFC 6996)                                                                                                         |
| `ASNPrivate32`        | `[4200000000, 4294967294]` (RFC 6996)                                                                                               |
| `VNI`                 | `[0, 16777215]`                                                                                                                     |
| `FlowLabel`           | `[0, 1048575]`                                                                                                                      |
| `MPLSLabel`           | `[0, 1048575]`                                                                                                                      |
| `MPLSLabelUnreserved` | `[16, 1048575]` (RFC 7274 special-purpose labels 0–15 excluded)                                                                     |
| `IPv6InterfaceID`     | 64-bit interface identifier with U/L and I/G bits clear                                                                             |

### CIDR and Hosts

```go
net4 := randomizer.Network.CIDR(randomizer.IPv4CIDR, 24)
net6 := randomizer.Network.CIDR(randomizer.IPv6CIDR, 48)
ula := randomizer.Network.CIDR(randomizer.IPv6ULAPrefix, 0)

host4 := randomizer.Network.IPInCIDR(nil, net4)

var hostBuf [16]byte
host6 := randomizer.Network.IPInCIDR(hostBuf[:0], net6)

// netip equivalents, zero allocations.
prefix := randomizer.Network.Prefix(randomizer.IPv6CIDR, 56)
host := randomizer.Network.AddrIn(prefix)
```

`CIDR` and `Prefix` clamp IPv4 prefix lengths to `[0, 32]` and IPv6 prefix lengths to `[0, 128]`. `IPv6ULAPrefix` always returns a random RFC 4193 `fd00::/8` `/48` prefix. `IPInCIDR` and `AddrIn` draw host bits uniformly even when the network address carries host bits; `AddrIn` returns the zero `netip.Addr` for an invalid prefix.

## Telecom

`Telecom.String` allocates; `Telecom.Append` writes into caller-owned capacity. Use `g.Telecom` for an instance source. Invalid options make `String` return `""` and `Append` return `dst` unchanged.

```go
imei := randomizer.Telecom.String(randomizer.IMEI, randomizer.TelecomOptions{})
withTAC := randomizer.Telecom.String(randomizer.IMEI, randomizer.TelecomOptions{TAC: "35123456"})
test := randomizer.Telecom.String(randomizer.IMEI, randomizer.TelecomOptions{Test: true})
imeisv := randomizer.Telecom.String(randomizer.IMEISV, randomizer.TelecomOptions{SVN: "07"})
iccid := randomizer.Telecom.String(randomizer.ICCID, randomizer.TelecomOptions{IIN: "8901", Length: 20})
meid := randomizer.Telecom.String(randomizer.MEID, randomizer.TelecomOptions{})
```

| Kind     | Result                                                                                                                               |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| `IMEI`   | 15 digits: 8-digit TAC (a GSMA TS.06 Reporting Body Identifier, or `00` with `Test`), 6-digit serial, Luhn check digit               |
| `IMEISV` | 16 digits: the 14 IMEI digits and a 2-digit software version in `[00, 98]` (3GPP TS 23.003 reserves 99)                              |
| `ICCID`  | ITU-T E.118: `89`, an assigned E.164 country code (or `IIN`), account digits, Luhn check digit; 19 digits by default, `Length` 18–20 |
| `MEID`   | 14 uppercase hexadecimal digits with a regional code in `[A0, FF]`                                                                   |

## Custom Sources and Readers

Custom inputs produce a generator directly:

```go
import "math/rand/v2"

custom := randomizer.FromSource(rand.NewPCG(1, 2))
reader := randomizer.FromReader(rand.NewChaCha8([32]byte{4}))

value := custom.Uint[uint64]()
word := reader.Word.String(randomizer.AlphaNumericAlphabet, 24)
```

`FromSource` accepts the standard [`math/rand/v2.Source`](https://pkg.go.dev/math/rand/v2#Source) interface, so a custom source needs only `Uint64() uint64`. It consumes that source's actual stream. `FromReader` accepts an `io.Reader` and uses [`io.ReadFull`](https://pkg.go.dev/io#ReadFull) semantics. Both constructors return `*Generator` and synchronize mutable input, holding one lock for each bulk operation.

Wrap each mutable input once, then share the resulting generator or its handles. Callers must relinquish unsynchronized access to the input. Independently wrapping the same unsafe source or reader creates independent locks and is unsupported.

A nil interface passed to either constructor panics immediately. Typed-nil inputs are invalid and may panic when used. Only `New()` and zero-value generators select the fast default without an explicit input.

`New()` and package helpers use public [`math/rand/v2`](https://pkg.go.dev/math/rand/v2) functions. `NewSeeded` initializes synchronized ChaCha8 from all 32 supplied seed bytes. `NewSecure` uses [`crypto/rand`](https://pkg.go.dev/crypto/rand) directly and never falls back to a weaker source. Use `NewSecure` for security-sensitive output.

`Generator.Read` returns the number of bytes filled and any full-read error, and may leave partial output. Scalar and structured generation panic when the reader cannot supply a word. Bulk fills and append operations may already have written part of their destination when a source panics.

## OUI Table

`HardwareMACRealOUI`, `HardwareEUI64RealOUI`, and `HardwareBluetoothPublic` select from `oui_table.go`, a generated list of universal unicast OUIs of well-known vendors from the IEEE MA-L registry.

```bash
go generate ./...
```

The generator in `internal/ouigen` downloads the [IEEE MA-L registry](https://standards-oui.ieee.org/oui/oui.csv), keeps the lowest three assignments for each allowlisted vendor, and fails if a vendor no longer matches so registry changes can be reviewed.

## Thread Safety

Package generation and all generator constructors support concurrent calls. Share a generator without replacing its exported handles, and give concurrent writes separate destination storage. `FromSource` and `FromReader` serialize wrapped mutable state. Wrap each unsafe input once and share that generator. Weighted tables are immutable after construction. No call changes another generator's selected source.

## References

- Daniel Lemire, [Fast Random Integer Generation in an Interval](https://arxiv.org/abs/1805.10941), ACM TOMACS 29(1), 2019.
- Nevin Brackett-Rozinsky and Daniel Lemire, [Batched Ranged Random Integer Generation](https://arxiv.org/abs/2408.06213), Software: Practice and Experience 55(1), 2025; first published online in 2024.
- Jon Bentley and Bob Floyd, [A Sample of Brilliance](https://doi.org/10.1145/30401.315746), Communications of the ACM 30(9), 1987.
- Michael D. Vose, [A Linear Algorithm for Generating Random Numbers with a Given Distribution](https://doi.org/10.1109/32.92917), IEEE Transactions on Software Engineering 17(9), 1991.
- Thomas L. Draper and Feras A. Saad, [Efficient Online Random Sampling via Randomness Recycling](https://www.cs.cmu.edu/~fsaad/assets/papers/2026-DraperEtAl-SODA.pdf), SODA 2026.
- Russ Cox and Filippo Valsorda, [Secure Randomness in Go 1.22](https://go.dev/blog/chacha8rand), and the [C2SP ChaCha8Rand specification](https://c2sp.org/chacha8rand).
- IEEE Std 802c-2017; RFC 8948, 4291, and 9542; Bluetooth Core Specification device-address and `ah` sections.
- RFC 6890, 5737, 6598, 3927, and 2526; RFC 7274; RFC 5398, 6793, 6996, 7300, and 7607; IANA special-purpose registries.
- GSMA TS.06, 3GPP TS 23.003, ITU-T E.118, and 3GPP2 S.R0048 for telecom formats.

## License

This project is licensed under the terms of the [MIT License](LICENSE).
