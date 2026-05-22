# go-crc32-asm

Fast CRC-32 checksums for Go storage engines.

This package mirrors the public shape of Go's `hash/crc32` package:
`MakeTable`, `Checksum`, `ChecksumIEEE`, `Update`, `New`, and `NewIEEE`.
It also adds `ChecksumCastagnoli` for CRC-32C.

```go
package main

import crc32asm "github.com/snissn/go-crc32-asm"

func checksum(block []byte) uint32 {
	return crc32asm.ChecksumIEEE(block)
}
```

The mirrored API includes:

- Constants: `Size`, `IEEE`, `Castagnoli`, and `Koopman`.
- Table API: `Table`, `IEEETable`, and `MakeTable`.
- Checksum API: `Checksum`, `ChecksumIEEE`, and `Update`.
- Streaming API: `New` and `NewIEEE` return `hash.Hash32` values with standard
  `Write`, `Sum32`, `Sum`, `Reset`, `Size`, and `BlockSize` behavior.

Canonical CRC-32/IEEE and CRC-32C/Castagnoli tables get fast paths. `Koopman`
and custom tables stay bit-compatible with `hash/crc32` through the fallback
path.

## Fast Paths

On `arm64`, CRC-32/IEEE uses `PMULL` folding over 12 adjacent 16-byte vectors
for large buffers, then reduces with ARM CRC32 instructions. CPUs with SHA3 use
`EOR3` in the fold.

On `arm64`, CRC-32C/Castagnoli uses Castagnoli-specific `PMULL` folding over
12 adjacent 16-byte vectors for large buffers, then reduces with ARM CRC32C
instructions. CPUs with SHA3 use `EOR3` in the fold.

On `amd64`, CRC-32/IEEE uses carry-less multiply folding:

- `AVX512F` + `AVX512BW` + `AVX512VL` + `VPCLMULQDQ`: eight 64-byte lanes.
- `AVX2` + `VPCLMULQDQ`: eight 32-byte lanes.
- `PCLMULQDQ` + `SSE4.1`: eight 16-byte lanes.

On `amd64`, CRC-32C/Castagnoli uses Castagnoli-specific carry-less multiply
folding with the same `PCLMULQDQ`/`VPCLMULQDQ` feature tiers as the IEEE path.

## Fallback Decisions

The package keeps stdlib delegation where the delegated path is already the
fast path:

- `Koopman` and custom tables use `hash/crc32`.
- Tiny writes below the architecture thresholds route directly to `hash/crc32`
  to avoid extra wrapper overhead.
- `arm64` seeded CRC-32/IEEE and CRC-32C/Castagnoli use the `PMULL` paths plus
  CRC combine from 576 B upward when PMULL is available.

## Benchmarks

Run the focused checksum benchmarks:

```sh
go test -run '^$' -bench 'BenchmarkChecksumIEEE|BenchmarkChecksumCastagnoli' -benchtime=1s -count=3
```

Run seeded `Update` benchmarks:

```sh
go test -run '^$' -bench BenchmarkUpdateNonZero -benchtime=1s -count=3
```

Run streaming `hash.Hash32` write benchmarks:

```sh
go test -run '^$' -bench BenchmarkHashWrite -benchtime=1s -count=3
```

Run the broader hash tournament:

```sh
go test -run '^$' -bench BenchmarkGomapHashTournamentSlice -benchtime=1s -count=3
```

The storage-oriented tournament sizes are 64 KiB, 256 KiB, 512 KiB, and 1 MiB.

Recent Apple M3 Castagnoli comparison for this ARM64 PMULL path against the
previous ARM64 Castagnoli path:

```text
ChecksumCastagnoli/64KiB/package_direct    -57.46% sec/op  +135.07% B/s
ChecksumCastagnoli/64KiB/package_table     -56.18% sec/op  +127.99% B/s
ChecksumCastagnoli/256KiB/package_direct   -57.05% sec/op  +132.85% B/s
ChecksumCastagnoli/256KiB/package_table    -56.59% sec/op  +130.35% B/s
ChecksumCastagnoli/512KiB/package_direct   -39.63% sec/op   +65.75% B/s
ChecksumCastagnoli/512KiB/package_table    -48.89% sec/op   +95.67% B/s
ChecksumCastagnoli/1MiB/package_direct     -16.29% sec/op   +19.46% B/s
ChecksumCastagnoli/1MiB/package_table      -14.05% sec/op   +16.35% B/s
geomean                                    -45.56% sec/op   +83.68% B/s
```

## Current Results

Representative one-shot checksum throughput:

| Machine | Function | Throughput |
|---|---|---:|
| Apple M3 | `ChecksumIEEE` | ~57-59 GB/s |
| Apple M3 | `ChecksumCastagnoli` | ~57-59 GB/s at 64-256 KiB; ~35-55 GB/s at 512 KiB-1 MiB in the latest run |
| Apple M3 | Go stdlib CRC-32/IEEE or CRC-32C | ~10 GB/s |
| Intel i5-11400F | `ChecksumIEEE` | ~54-65 GB/s |
| Intel i5-11400F | `ChecksumCastagnoli` | ~55-65 GB/s on large buffers |
| Intel i5-11400F | Go stdlib CRC-32/IEEE | ~23-24 GB/s |
| Intel i5-11400F | Go stdlib CRC-32C | ~29-30 GB/s |

Representative nonzero-seed `Update` throughput at 64 KiB to 1 MiB:

| Machine | Function | Throughput |
|---|---|---:|
| Apple M3 | CRC-32/IEEE `Update` | ~56-60 GB/s |
| Apple M3 | CRC-32C/Castagnoli `Update` | ~56-60 GB/s |
| Intel i5-11400F | CRC-32/IEEE `Update` | ~54-64 GB/s |
| Intel i5-11400F | CRC-32C/Castagnoli `Update` | ~55-65 GB/s on large buffers |

Streaming `hash.Hash32` writes use the same `Update` paths. Large writes keep
the fast-path behavior; tiny writes are mostly call overhead.

Representative nonzero-seed `Update` throughput at 4 KiB:

| Machine | Function | Package | Go stdlib |
|---|---|---:|---:|
| Apple M3 | CRC-32/IEEE `Update` | ~30 GB/s | ~10 GB/s |
| Apple M3 | CRC-32C/Castagnoli `Update` | ~28 GB/s | ~10 GB/s |
| Intel i5-11400F | CRC-32/IEEE `Update` | ~52-54 GB/s | ~23-24 GB/s |
| Intel i5-11400F | CRC-32C/Castagnoli `Update` | ~29 GB/s | ~29-30 GB/s |

The tournament conclusion from these runs is:

- CRC-32/IEEE is a large win for this package on both tested machines.
- CRC-32C/Castagnoli is now also a large win on Apple M3 and on the tested Intel
  host for large buffers.
- `XXH3_64` and `FarmHash64` remain strong non-CRC content-hash competitors
  when a CRC polynomial is not required.

## Compatibility

CRC-32/IEEE and CRC-32C/Castagnoli are different 32-bit CRC polynomials. They
return different checksums for the same bytes and are not disk-format
compatible with each other.

`New` and `NewIEEE` return `hash.Hash32` values matching Go stdlib behavior,
including `Sum`, `Sum32`, `Reset`, binary marshal/unmarshal, binary append, and
clone support.
