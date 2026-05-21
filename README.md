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

On `arm64`, CRC-32C/Castagnoli uses four independent ARM CRC32C streams and
combines them with the Castagnoli polynomial.

On `amd64`, CRC-32/IEEE uses carry-less multiply folding:

- `AVX512F` + `AVX512BW` + `AVX512VL` + `VPCLMULQDQ`: eight 64-byte lanes.
- `AVX2` + `VPCLMULQDQ`: eight 32-byte lanes.
- `PCLMULQDQ` + `SSE4.1`: eight 16-byte lanes.

On `amd64`, CRC-32C/Castagnoli uses Go's optimized SSE4.2 CRC32C path.

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

## Current Results

Representative one-shot checksum throughput:

| Machine | Function | Throughput |
|---|---|---:|
| Apple M3 | `ChecksumIEEE` | ~57-59 GB/s |
| Apple M3 | `ChecksumCastagnoli` | ~28-30 GB/s |
| Apple M3 | Go stdlib CRC-32/IEEE or CRC-32C | ~10 GB/s |
| Intel i5-11400F | `ChecksumIEEE` | ~54-65 GB/s |
| Intel i5-11400F | `ChecksumCastagnoli` | ~29-30 GB/s |
| Intel i5-11400F | Go stdlib CRC-32/IEEE | ~23-24 GB/s |
| Intel i5-11400F | Go stdlib CRC-32C | ~29-30 GB/s |

Representative nonzero-seed `Update` throughput at 64 KiB to 1 MiB:

| Machine | Function | Throughput |
|---|---|---:|
| Apple M3 | CRC-32/IEEE `Update` | ~55-60 GB/s |
| Apple M3 | CRC-32C/Castagnoli `Update` | ~28-30 GB/s |
| Intel i5-11400F | CRC-32/IEEE `Update` | ~54-64 GB/s |
| Intel i5-11400F | CRC-32C/Castagnoli `Update` | ~29-30 GB/s |

Streaming `hash.Hash32` writes use the same `Update` paths. Large writes keep
the fast-path behavior; tiny writes are mostly call overhead.

The tournament conclusion from these runs is:

- CRC-32/IEEE is the main win for this package on both tested machines.
- CRC-32C/Castagnoli is a large win on Apple M3 and a tie with stdlib/klauspost
  on the tested Intel host.
- `XXH3_64` and `FarmHash64` remain strong non-CRC content-hash competitors
  when a CRC polynomial is not required.

## Compatibility

CRC-32/IEEE and CRC-32C/Castagnoli are different 32-bit CRC polynomials. They
return different checksums for the same bytes and are not disk-format
compatible with each other.

`New` and `NewIEEE` return `hash.Hash32` values matching Go stdlib behavior,
including `Sum`, `Sum32`, `Reset`, binary marshal/unmarshal, binary append, and
clone support.
