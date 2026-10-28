# Current performance

Measured on 2026-10-28 using Go 1.27.3 on Windows/AMD64, with an AMD Ryzen 9
7950X: 16 physical cores and 32 logical processors. `CGO_ENABLED=1`; `GOFLAGS`
and `GOWORK` are empty.

Timings below are medians of six 100 ms benchmark samples with `-cpu=1`.
They exclude profiling and race instrumentation. Setup, generator construction,
and caller-buffer allocation occur outside the timed loops unless construction
is the operation being measured. Local timings vary with hardware and system
load; they do not establish application latency or production capacity.

## Construction

All five entry points return `*Generator`. These benchmarks retain the returned
pointer, so the generator escapes to the heap. Input construction is included
for the custom-source rows; reusing an existing input has a different cost.

| Operation            | Input                         |   Median | B/op | Allocations/op |
| -------------------- | ----------------------------- | -------: | ---: | -------------: |
| `New()`              | Fast package source           | 13.54 ns |   80 |              1 |
| `NewSecure()`        | Cryptographic source          | 13.61 ns |   80 |              1 |
| `NewSeeded(seed)`    | ChaCha8, 32-byte seed         | 100.3 ns |  424 |              3 |
| `FromSource(source)` | Fresh `rand.NewPCG(1, 2)`     | 27.68 ns |  120 |              3 |
| `FromReader(reader)` | Fresh `rand.NewChaCha8(seed)` | 97.41 ns |  432 |              3 |

Construct once and share the generator. Mutable custom sources and readers are
synchronized within that generator. Construction consumes no random output.
Generation and buffer costs are measured separately below.

## Numbers

Package-level functions use the fast source. All operations in this table use
zero bytes and zero heap allocations per call.

| Operation                                |   Median |
| ---------------------------------------- | -------: |
| `Int[int64]`                             | 3.748 ns |
| `Uint[uint64]`                           | 3.510 ns |
| `IntInterval[int64]`, `[-100000,100000)` | 5.032 ns |
| `UintInterval[uint64]`, `[100,1000000)`  | 4.397 ns |
| `Float32`                                | 3.688 ns |
| `Float64`                                | 4.259 ns |

Bulk fills write into caller-owned storage. Each row measures one complete
batch and uses zero heap allocations.

| Operation                      | Source         | Batch size |   Median |
| ------------------------------ | -------------- | ---------: | -------: |
| `FillUintInterval`, `[0,1000)` | Fast           | 256 values | 353.8 ns |
| `FillUintInterval`, `[0,1000)` | Seeded ChaCha8 | 256 values | 386.4 ns |
| `FillUintInterval`, `[0,1000)` | Secure         | 256 values | 2.925 us |
| `FillUint[uint8]`              | Secure         | 256 values | 233.1 ns |
| `FillFloat64`                  | Secure         | 256 values | 1.535 us |

Secure integer and float fills use a 256-byte stack buffer within each operation.
Built-in synchronized sources hold one lock for each bulk operation.

## Words

These measurements use `Word.Append` with the alphanumeric alphabet and
sufficient caller-owned capacity. Both modes use zero heap allocations.

|      Length | Independent symbols | `NoAdjacentRepeats` |
| ----------: | ------------------: | ------------------: |
|    16 bytes |            21.99 ns |            32.91 ns |
|   256 bytes |            256.3 ns |            389.1 ns |
| 4,096 bytes |            3.960 us |            5.951 us |

`Word.WithMode(NoAdjacentRepeats).Custom("AABCXYZ9", 256)` measures 3.741 us,
256 B/op, and one allocation for its returned string. Weighted constrained
sampling uses cumulative weights, excluding the preceding byte's weight interval.
Built-in unique alphabets use batched bounded sampling; power-of-two independent
alphabets use bit extraction.

## Sources, shuffle, and weighted choice

Each row uses a source-bound generator. Output buffers and weighted tables are
prepared outside the timed loops. All entries use zero heap allocations per
operation.

| Operation                     |     Fast | Seeded ChaCha8 |   Secure |
| ----------------------------- | -------: | -------------: | -------: |
| Read 256 bytes                | 136.6 ns |       137.2 ns | 174.2 ns |
| Shuffle 256 items             | 1.181 us |       975.8 ns | 2.703 us |
| Weighted choice, 256 outcomes | 22.77 ns |       18.76 ns | 122.0 ns |

The same workloads through `FromReader` use zero heap allocations:

| Reader input         | Read 256 bytes | Shuffle 256 items | Weighted choice, 256 outcomes |
| -------------------- | -------------: | ----------------: | ----------------------------: |
| ChaCha8              |       116.2 ns |          1.051 us |                      27.26 ns |
| `crypto/rand.Reader` |       168.7 ns |          2.343 us |                     122.95 ns |

The seeded constructor uses the same synchronized source adapter as
`FromSource(rand.NewChaCha8(seed))`. Reader and source adapters retain their
respective byte- and word-consumption contracts. `NewSecure` has dedicated
cryptographic paths, including buffered integer and float fills.

The weighted-choice workload assigns weight `i+1` to outcome `i`.
`NewWeighted` with 4,096 outcomes measures 20.380 us, 131,136 B/op, and five
allocations. Its immutable alias table supports allocation-free repeated draws.

Built-in shuffle sources use Fisher-Yates with per-operation entropy recycling.
Shared seeded generators serialize access to their mutable stream. Give
concurrent writes separate output storage. Parallel benchmark ns/op represents
aggregate throughput, not per-call latency.

## Ordered samples

`SampleIndices` fills caller-owned storage with distinct indices in random order.
The fast source is used here.

| Sample size | Population |    Median |   B/op | Allocations/op |
| ----------: | ---------: | --------: | -----: | -------------: |
|           8 |  1,000,000 |  50.98 ns |      0 |              0 |
|          64 |  1,000,000 |  1.735 us |  2,344 |              3 |
|         512 |      1,024 | 17.022 us | 18,472 |              3 |
|       1,024 |      1,024 |  4.199 us |      0 |              0 |

Samples of at most sixteen elements use ordered Floyd sampling. Larger partial
samples use a sparse permutation map whose storage depends on sample size.
Full samples initialize and shuffle the destination with constant auxiliary space.

## Network and telecom

The fast source and caller-owned buffers are used. Every row uses zero heap
allocations.

| Operation                                 |    Median |
| ----------------------------------------- | --------: |
| Buffered `IPv4Any`                        |  7.058 ns |
| Buffered `IPv6Global`                     | 13.980 ns |
| `netip.Addr`, `IPv4Any`                   |  7.674 ns |
| `netip.Addr`, `IPv6Global`                | 15.425 ns |
| IPv6 host in `2001:db8::/32`, `net.IPNet` | 13.285 ns |
| Buffered MAC                              |  8.395 ns |
| Append IMEI                               | 41.880 ns |
| Bluetooth RPA, one warm key               | 30.770 ns |

Alternating two Bluetooth RPA keys measures 93.03 ns, 544 B/op, and two
allocations. Sixteen alternating keys measure 92.36 ns with the same allocation
budget. The cipher cache retains one key; changing keys constructs AES schedules.

## Validation

The Windows/AMD64 suite passes with Go 1.27.3, including race detection and
parallel benchmarks under the race detector at one and sixteen workers.
`go vet`, Staticcheck 2026.2.2, and govulncheck 1.9.0 pass with no findings.
Windows/386 tests also pass. Five fuzz targets covering words, weighted mass,
ordered samples, numeric intervals, and network prefixes pass bounded one-second
runs with four workers each. Examples compile; ordinary tests assert their
behavior. The documentation audit finds no unresolved Go doc references.

## Reproduce

Run from the repository root in PowerShell:

```powershell
New-Item -ItemType Directory -Force .profiles/current | Out-Null
go test -count=1 ./...
go test -c -o .profiles/current/randomizer.test.exe .

$bench = '^Benchmark(Number|Network|ProviderRead|Telecom|SourceWorkloads|NumericBulk|NumericPacked|SampleIndices|Shuffle|WeightedIndex|WeightedConstruction|HardwareRPAKeys|WordModes|WordCustomNoAdjacent|GeneratorConstruction)'
& ./.profiles/current/randomizer.test.exe '-test.run=^$' "-test.bench=$bench" '-test.benchmem' '-test.benchtime=100ms' '-test.count=6' '-test.cpu=1' |
    Out-File .profiles/current/benchmarks.txt -Encoding utf8
benchstat .profiles/current/benchmarks.txt

& ./.profiles/current/randomizer.test.exe '-test.run=^$' '-test.bench=^Benchmark(SourceWorkloads|NumericBulk|SampleIndices|WeightedConstruction)$' '-test.benchtime=100ms' '-test.cpu=1' '-test.cpuprofile=.profiles/current/cpu.pprof'
& ./.profiles/current/randomizer.test.exe '-test.run=^$' '-test.bench=^Benchmark(SourceWorkloads|NumericPacked|SampleIndices|WeightedConstruction|HardwareRPAKeys)$' '-test.benchtime=100ms' '-test.cpu=1' '-test.memprofilerate=1' '-test.memprofile=.profiles/current/allocations.pprof'
& ./.profiles/current/randomizer.test.exe '-test.run=^$' '-test.bench=^BenchmarkWeightedRetained$' '-test.benchtime=200ms' '-test.cpu=1' '-test.memprofilerate=1' '-test.memprofile=.profiles/current/heap.pprof'
& ./.profiles/current/randomizer.test.exe '-test.run=^$' '-test.bench=^BenchmarkParallelSources$/^Seeded$' '-test.benchtime=1s' '-test.cpu=16' '-test.mutexprofile=.profiles/current/mutex.pprof'
& ./.profiles/current/randomizer.test.exe '-test.run=^$' '-test.bench=^BenchmarkParallelSources$/^Seeded$' '-test.benchtime=1s' '-test.cpu=16' '-test.blockprofile=.profiles/current/block.pprof'

go tool pprof -top .profiles/current/randomizer.test.exe .profiles/current/cpu.pprof
go tool pprof -alloc_space .profiles/current/randomizer.test.exe .profiles/current/allocations.pprof
go tool pprof -inuse_space .profiles/current/randomizer.test.exe .profiles/current/heap.pprof
go tool pprof -top .profiles/current/randomizer.test.exe .profiles/current/mutex.pprof
```

Retain the matching executable with its profiles. Record the toolchain, platform,
CPU, payload size, source mode, and worker count when collecting measurements.
