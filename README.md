# go-crc32-asm

Experimental CRC-32/IEEE implementation for Go.

The package exports `ChecksumIEEE`, which is bit-compatible with
`hash/crc32.ChecksumIEEE`.

On `arm64`, the fast path follows the same large-buffer shape as `libdeflate`:
it folds 12 adjacent 16-byte vectors with `PMULL`, reduces the 128-bit folded
state with ARM CRC32 instructions, and uses `EOR3` on CPUs with the SHA3
extension. That removes the single-stream CRC dependency chain that limits a
plain CRC-instruction loop.

For `amd64`, the fast path uses x86 carry-less multiply folding. On CPUs with
`AVX512F`, `AVX512BW`, `AVX512VL`, and `VPCLMULQDQ`, it folds eight 64-byte
vectors at a time and uses `VPTERNLOGD` for the three-way XOR in the fold. On
CPUs with `AVX2` and `VPCLMULQDQ`, it folds eight 32-byte vectors at a time.
On older CPUs with `PCLMULQDQ` and `SSE4.1`, it folds eight 16-byte vectors at
a time.

For smaller `arm64` buffers, and for `arm64` CPUs without `PMULL`, the package
keeps a four-stream CRC32-instruction path. Other architectures fall back to
Go's standard library implementation.

## Benchmarks

The focused CRC benchmark compares this package against Go stdlib:

```sh
go test -run '^$' -bench BenchmarkChecksumIEEE -benchtime=1s -count=3
```

`BenchmarkGomapHashTournamentSlice` reuses the gomap hash tournament block
sizes and a small competitor set, including Go stdlib CRC-32/IEEE, Go stdlib
CRC-32C/Castagnoli, `github.com/klauspost/crc32`, `FarmHash64`, and `XXH3_64`:

```sh
go test -run '^$' -bench BenchmarkGomapHashTournamentSlice -benchtime=1s -count=3
```

The block sizes are:

- 64 KiB: default lower compression-block threshold.
- 256 KiB: middle case.
- 512 KiB: common one-mark scale for wider rows.
- 1 MiB: default maximum compression block size.

Current Apple M3 result for `BenchmarkChecksumIEEE` after adding the
`PMULL`/`EOR3` path:

- 64 KiB: about 57 GB/s.
- 256 KiB: about 59 GB/s.
- 512 KiB: about 59 GB/s.
- 1 MiB: about 59 GB/s.

That is roughly 5.5x faster than Go's `hash/crc32.ChecksumIEEE` on the same
machine for these sizes, and it beats the non-CRC hash competitors in the gomap
tournament on this hardware.

Current Intel i5-11400F result for `BenchmarkChecksumIEEE` after adding the
`AVX512` `VPCLMULQDQ` path:

- 64 KiB: about 63-64 GB/s.
- 256 KiB: about 63-65 GB/s.
- 512 KiB: about 58-62 GB/s.
- 1 MiB: about 49-61 GB/s in a noisy run.

That is roughly 2.5x to 2.75x faster than Go's `hash/crc32.ChecksumIEEE` on
the same machine for these sizes. It is also in the same performance class as a
local `libdeflate_crc32` C benchmark on the same host, which measured about
62-64 GB/s across these block sizes.

`github.com/klauspost/crc32` comparison:

- On Apple M3, `github.com/klauspost/crc32` does not have the `arm64`
  `PMULL`/`EOR3` path used here. In `BenchmarkGomapHashTournamentSlice`, it
  measured about 2.6-2.7 GB/s for both CRC-32/IEEE and CRC-32C/Castagnoli,
  while this package measured about 57-59 GB/s for CRC-32/IEEE.
- On Intel i5-11400F, `github.com/klauspost/crc32` measured about 23-24 GB/s
  for CRC-32/IEEE and about 29-30 GB/s for CRC-32C/Castagnoli at the gomap
  tournament block sizes. This package measured about 60-64 GB/s for
  CRC-32/IEEE on the same run.

CRC naming note:

- CRC-32/IEEE is the classic Ethernet/gzip/zip/png polynomial. It is the value
  returned by Go's `hash/crc32.ChecksumIEEE`, and it is what this package
  implements.
- CRC-32C/Castagnoli is a different 32-bit CRC polynomial. It returns a
  different checksum for the same bytes. It is often fast on x86 because the
  SSE4.2 `CRC32` instruction computes the Castagnoli polynomial directly.
- The two are not drop-in compatible on disk unless the stored checksum format
  is allowed to change. Both produce 32-bit CRCs, but the polynomial and output
  values differ.
