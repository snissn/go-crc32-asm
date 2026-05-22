# go-crc32-asm

Fast CRC-32/IEEE checksums for Go.

`ChecksumIEEE` is bit-compatible with Go's `hash/crc32.ChecksumIEEE`, but uses
wide folding paths for large buffers. The target workload is storage-engine
checksums over pages, granules, compressed blocks, and similar contiguous byte
ranges.

```go
package main

import crc32asm "github.com/snissn/go-crc32-asm"

func checksum(block []byte) uint32 {
	return crc32asm.ChecksumIEEE(block)
}
```

The package also mirrors the standard `hash/crc32` API shape:

- Constants: `Size`, `IEEE`, `Castagnoli`, and `Koopman`.
- Table API: `Table`, `IEEETable`, and `MakeTable`.
- Checksum API: `Checksum`, `ChecksumIEEE`, and `Update`.
- Streaming API: `New` and `NewIEEE` return `hash.Hash32` values with standard
  `Write`, `Sum32`, `Sum`, `Reset`, `Size`, and `BlockSize` behavior.

Canonical CRC-32/IEEE calls use the fast path; other polynomials remain
compatible with Go stdlib behavior.

## Implementation

On `arm64`, the large-buffer path follows the same high-level shape as
`libdeflate`: fold 12 adjacent 16-byte vectors with `PMULL`, reduce the folded
state with ARM CRC32 instructions, and use `EOR3` on CPUs with the SHA3
extension.

On `amd64`, the fast path uses carry-less multiply folding:

- `AVX512F` + `AVX512BW` + `AVX512VL` + `VPCLMULQDQ`: eight 64-byte lanes with
  `VPTERNLOGD`.
- `AVX2` + `VPCLMULQDQ`: eight 32-byte lanes.
- `PCLMULQDQ` + `SSE4.1`: eight 16-byte lanes.

Other architectures fall back to Go's standard library implementation.

## Benchmarks

Run the focused CRC benchmark:

```sh
go test -run '^$' -bench BenchmarkChecksumIEEE -benchtime=1s -count=3
```

Run the broader hash tournament:

```sh
go test -run '^$' -bench BenchmarkGomapHashTournamentSlice -benchtime=1s -count=3
```

The tournament uses storage-oriented block sizes:

- 64 KiB
- 256 KiB
- 512 KiB
- 1 MiB

Current local results:

| Machine | Size | `go-crc32-asm` | Go stdlib `ChecksumIEEE` |
|---|---:|---:|---:|
| Apple M3 | 64 KiB | ~57 GB/s | ~10.5 GB/s |
| Apple M3 | 256 KiB | ~59 GB/s | ~10 GB/s |
| Apple M3 | 512 KiB | ~59 GB/s | ~10.5 GB/s |
| Apple M3 | 1 MiB | ~59 GB/s | ~10 GB/s |
| Intel i5-11400F | 64 KiB | ~63-64 GB/s | ~24 GB/s |
| Intel i5-11400F | 256 KiB | ~63-65 GB/s | ~23 GB/s |
| Intel i5-11400F | 512 KiB | ~58-62 GB/s | ~23 GB/s |
| Intel i5-11400F | 1 MiB | ~54-62 GB/s | ~23 GB/s |

On the same Intel host, a local `libdeflate_crc32` C benchmark measured about
62-64 GB/s across these block sizes.

## Tournament Notes

`BenchmarkGomapHashTournamentSlice` also includes Go stdlib CRC-32C/Castagnoli,
`github.com/klauspost/crc32`, `FarmHash64`, and `XXH3_64`.

Representative results from the current runs:

| Machine | Competitor | Throughput |
|---|---|---:|
| Apple M3 | `go-crc32-asm` CRC-32/IEEE | ~57-59 GB/s |
| Apple M3 | `XXH3_64` | ~31 GB/s |
| Apple M3 | `FarmHash64` | ~26-27 GB/s |
| Apple M3 | Go stdlib CRC-32/IEEE or CRC-32C | ~10 GB/s |
| Intel i5-11400F | `go-crc32-asm` CRC-32/IEEE | ~60-64 GB/s |
| Intel i5-11400F | `XXH3_64` | ~60-79 GB/s |
| Intel i5-11400F | Go stdlib CRC-32C/Castagnoli | ~29-30 GB/s |
| Intel i5-11400F | `github.com/klauspost/crc32` CRC-32C/Castagnoli | ~29-30 GB/s |
| Intel i5-11400F | Go stdlib CRC-32/IEEE | ~23-24 GB/s |
| Intel i5-11400F | `github.com/klauspost/crc32` CRC-32/IEEE | ~23-24 GB/s |

## Checksum Compatibility

This package implements CRC-32/IEEE, the same polynomial and output format as
`hash/crc32.ChecksumIEEE`.

CRC-32C/Castagnoli is a different 32-bit CRC polynomial. It returns different
checksums for the same bytes and is not disk-format compatible with CRC-32/IEEE
unless the stored checksum format is allowed to change.
