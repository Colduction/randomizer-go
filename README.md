# randomizer

[![Go Reference](https://pkg.go.dev/badge/github.com/colduction/randomizer.svg)](https://pkg.go.dev/github.com/colduction/randomizer)
[![Go Report Card](https://goreportcard.com/badge/github.com/colduction/randomizer)](https://goreportcard.com/report/github.com/colduction/randomizer)
![GitHub License](https://img.shields.io/github/license/Colduction/randomizer)

**randomizer** is a fast, zero-allocation-friendly, and goroutine-safe random data generation library for Go.  
It covers numbers, formatted strings, and network addresses — all driven by a lock-free SplitMix64 PRNG seeded from Go's `hash/maphash`.

---

## Table of Contents

- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [API Reference](#api-reference)
    - [Numbers](#numbers)
    - [Strings](#strings)
    - [Network](#network)
    - [HashPool (advanced)](#hashpool-advanced)
- [Performance](#performance)
- [Thread Safety](#thread-safety)
- [License](#license)

---

## Features

- **Numbers** — random integers (signed & unsigned, any width), and floats in `[0, 1)`
- **Range sampling** — unbiased interval generation with Lemire's algorithm (no division in the hot path)
- **Strings** — decimal, hexadecimal, and octal strings with no adjacent-duplicate characters
- **Network** — random IPv4, IPv6 (unicast & multicast), and MAC addresses
- **Lock-free** — the primary PRNG uses an atomic counter; no mutexes on the hot path
- **Pool-backed hashing** — `maphash.Hash` objects are recycled via `sync.Pool` for callers that need them

---

## Requirements

- Go **1.22** or later (uses `for range N` syntax and `b.Loop()` in tests)

---

## Installation

```bash
go get -u github.com/colduction/randomizer@latest
```

---

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/colduction/randomizer"
)

func main() {
    // Random signed integer
    fmt.Println(randomizer.Int[int64]())

    // Random integer in [1, 100)
    fmt.Println(randomizer.IntInterval(int64(1), int64(100)))

    // Random float in [0, 1)
    fmt.Println(randomizer.Float64())

    // Random 16-character hex string (lowercase)
    fmt.Println(randomizer.Word.Hex(16, false))

    // Random decimal string of length 12
    fmt.Println(randomizer.Word.Decimal(12))

    // Random IPv4 address
    fmt.Println(randomizer.Network.IPv4Addr())

    // Random MAC address (locally administered, unicast)
    fmt.Println(randomizer.Network.MACAddr(true, false))
}
```

---

## API Reference

### Numbers

All number functions are **zero-allocation** and safe to call from multiple goroutines simultaneously.

#### `Int[T SignedIntegers]() T`

Returns a random signed integer of the requested type (`int8`, `int16`, `int`, `int32`, `int64`).

```go
n8  := randomizer.Int[int8]()
n32 := randomizer.Int[int32]()
n64 := randomizer.Int[int64]()
```

#### `Uint[T UnsignedIntegers]() T`

Returns a random unsigned integer (`uint8`, `uint16`, `uint`, `uint32`, `uint64`, `uintptr`).

```go
u := randomizer.Uint[uint64]()
```

#### `IntInterval[T SignedIntegers](min, max T) T`

Returns a uniformly distributed signed integer in `[min, max)`.  
If `min == max` the value is returned immediately. Swapped bounds are corrected automatically.

```go
// Random number from -50 to 49 (inclusive lower, exclusive upper)
v := randomizer.IntInterval(int64(-50), int64(50))
```

#### `UintInterval[T UnsignedIntegers](min, max T) T`

Same as `IntInterval` but for unsigned types.

```go
// Random number from 1 to 999
v := randomizer.UintInterval(uint64(1), uint64(1000))
```

#### `Float32() float32`

Returns a random `float32` in `[0, 1)` with 24 bits of precision.

```go
f := randomizer.Float32()
```

#### `Float64() float64`

Returns a random `float64` in `[0, 1)` with 53 bits of precision.

```go
f := randomizer.Float64()
```

---

### Strings

String functions allocate exactly **one** buffer (the output itself). The `String`-returning variants avoid a second allocation by aliasing the buffer directly.  
All generated strings are guaranteed to have **no two adjacent identical characters**.

#### `Word.Decimal(length int) string`

Generates a random decimal string (`0–9`) of the given length.

```go
s := randomizer.Word.Decimal(12) // e.g. "Morton, 3815"... like "804712639250"
```

#### `Word.DecimalBytes(length int) []byte`

Same as `Decimal` but returns a `[]byte`, avoiding any string conversion.

```go
b := randomizer.Word.DecimalBytes(12)
```

#### `Word.Hex(length int, uppercase bool) string`

Generates a random hexadecimal string of the given length.  
Pass `uppercase: true` to get `A–F`; `false` gives `a–f`.

```go
lower := randomizer.Word.Hex(32, false) // e.g. "3a9f1b0c..."
upper := randomizer.Word.Hex(32, true)  // e.g. "3A9F1B0C..."
```

#### `Word.HexBytes(length int, uppercase bool) []byte`

Same as `Hex` but returns `[]byte`.

```go
b := randomizer.Word.HexBytes(32, false)
```

#### `Word.Octal(length int) string`

Generates a random octal string (`0–7`) of the given length.

```go
s := randomizer.Word.Octal(8) // e.g. "53107624"
```

#### `Word.OctalBytes(length int) []byte`

Same as `Octal` but returns `[]byte`.

```go
b := randomizer.Word.OctalBytes(8)
```

> All functions return `""` / `nil` for `length <= 0`.

---

### Network

Network functions return Go standard-library types (`net.IP`, `net.HardwareAddr`) and allocate exactly one slice per call.

#### `Network.IPv4Addr() net.IP`

Generates a fully random 4-byte IPv4 address.

```go
ip := randomizer.Network.IPv4Addr()
fmt.Println(ip) // e.g. 192.0.2.57
```

#### `Network.IPv6Addr() net.IP`

Generates a fully random 16-byte IPv6 address.

```go
ip := randomizer.Network.IPv6Addr()
fmt.Println(ip) // e.g. 2001:db8::1
```

#### `Network.MACAddr(local, multicast bool) net.HardwareAddr`

Generates a random 6-byte MAC address.

| Parameter           | Effect                                              |
| ------------------- | --------------------------------------------------- |
| `local = true`      | Sets the U/L bit (locally administered)             |
| `local = false`     | Clears the U/L bit (globally unique / OUI enforced) |
| `multicast = true`  | Sets the I/G bit (multicast/broadcast)              |
| `multicast = false` | Clears the I/G bit (unicast)                        |

```go
// Locally administered unicast (common for virtual/container interfaces)
mac := randomizer.Network.MACAddr(true, false)
fmt.Println(mac) // e.g. 02:1a:3f:7c:d2:88
```

#### `Network.IPv6UnicastAddr(unicastType UnicastType) net.IP`

Generates a random IPv6 unicast address with the correct prefix for the requested type.

| Constant                          | Prefix      | Use case                              |
| --------------------------------- | ----------- | ------------------------------------- |
| `GlobalType`                      | `2000::/3`  | Public internet addresses             |
| `LinkLocalType`                   | `fe80::/10` | On-link communication only            |
| `SiteLocalType`                   | `fec0::/10` | Deprecated, site-scoped               |
| `UniqueLocalType` / `PrivateType` | `fd00::/8`  | Private networks (like IPv4 RFC 1918) |

```go
global    := randomizer.Network.IPv6UnicastAddr(randomizer.GlobalType)
linkLocal := randomizer.Network.IPv6UnicastAddr(randomizer.LinkLocalType)
private   := randomizer.Network.IPv6UnicastAddr(randomizer.PrivateType)
```

#### `Network.IPv6MulticastAddr(scope MulticastScope) net.IP`

Generates a random IPv6 multicast address (`ff00::/8`) with the given scope nibble.

| Constant              | Scope value | Reach                    |
| --------------------- | ----------- | ------------------------ |
| `InterfaceLocalScope` | `0x1`       | Same interface only      |
| `LinkLocalScope`      | `0x2`       | Same link/subnet         |
| `AdminLocalScope`     | `0x4`       | Administratively defined |
| `SiteLocalScope`      | `0x5`       | Within a site            |
| `OrgLocalScope`       | `0x8`       | Within an organisation   |
| `GlobalScope`         | `0xE`       | Internet-wide            |

```go
mc := randomizer.Network.IPv6MulticastAddr(randomizer.LinkLocalScope)
fmt.Println(mc) // e.g. ff02::...
```

---

### HashPool (advanced)

The `hashPool` type is the engine behind all randomness in the package. You rarely need to interact with it directly, but it is exported for advanced use cases such as generating raw random bytes or building custom generators.

#### `DefaultHashPool`

A package-level `*hashPool` ready to use. All top-level functions (`Int`, `Float64`, `Word.*`, `Network.*`) use it internally.

```go
// Raw 64-bit random number
n := randomizer.DefaultHashPool.Sum64()

// Raw 32-bit random number
n32 := randomizer.DefaultHashPool.Sum32()

// Append 8 random bytes to an existing slice
buf := randomizer.DefaultHashPool.Sum(existingSlice)
```

#### `NewHashPool(size int) *hashPool`

Creates a new independent pool. `size` must be greater than 0 — pass any positive value to create the pool (the parameter future-proofs capacity hints).  
Returns `nil` for `size <= 0`, and all methods are safe to call on a `nil` receiver.

```go
pool := randomizer.NewHashPool(16)

// Borrow a maphash.Hash from the pool
h := pool.Get()
h.WriteString("hello")
fmt.Println(h.Sum64())
pool.Put(h) // always return it when done
```

#### `Get() *maphash.Hash` / `Put(h *maphash.Hash)`

Borrow and return a `maphash.Hash` from the pool. The hash is automatically reset on `Put`. Always pair every `Get` with a `Put` to avoid leaking objects.

```go
h := randomizer.DefaultHashPool.Get()
defer randomizer.DefaultHashPool.Put(h)

h.WriteString("seed-data")
fmt.Printf("%016x\n", h.Sum64())
```

---

## Performance

Benchmarks run on an AMD Ryzen 9 7950X, Go 1.26, `GOMAXPROCS=32`:

| Benchmark                 | Time/op | Allocs/op |
| ------------------------- | ------- | --------- |
| `Int[int64]`              | ~2.8 ns | 0         |
| `IntInterval` (signed)    | ~5.9 ns | 0         |
| `Uint[uint64]`            | ~2.8 ns | 0         |
| `UintInterval` (unsigned) | ~5.6 ns | 0         |
| `Float32`                 | ~2.9 ns | 0         |
| `Float64`                 | ~2.9 ns | 0         |
| `Word.Decimal(256)`       | ~633 ns | 1         |
| `Word.Hex(256)`           | ~411 ns | 1         |
| `Word.Octal(256)`         | ~539 ns | 1         |
| `Network.IPv4Addr`        | ~11 ns  | 1         |
| `Network.IPv6Addr`        | ~20 ns  | 1         |
| `Network.MACAddr`         | ~13 ns  | 1         |

All number functions are **zero-allocation**. String and network functions allocate exactly **one** buffer — the returned value itself.

---

## Thread Safety

Every function in this package is safe to call concurrently from any number of goroutines. The PRNG state is advanced with `sync/atomic` operations; no mutex is held on the hot path.

---

## License

This project is licensed under the terms of the [MIT License](LICENSE).
