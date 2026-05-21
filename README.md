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
`AVX2` and `VPCLMULQDQ`, it folds eight 32-byte vectors at a time. On older
CPUs with `PCLMULQDQ` and `SSE4.1`, it folds eight 16-byte vectors at a time.

For smaller `arm64` buffers, and for `arm64` CPUs without `PMULL`, the package
keeps a four-stream CRC32-instruction path. Other architectures fall back to
Go's standard library implementation.

## Benchmarks

The focused CRC benchmark compares this package against Go stdlib:

```sh
go test -run '^$' -bench BenchmarkChecksumIEEE -benchtime=1s -count=3
```

`BenchmarkGomapHashTournamentSlice` reuses the gomap hash tournament block
sizes and a small competitor set:

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
`VPCLMULQDQ`/`PCLMULQDQ` paths:

- 64 KiB: about 32 GB/s.
- 256 KiB: about 32 GB/s.
- 512 KiB: about 32 GB/s.
- 1 MiB: about 32-33 GB/s.

That is roughly 1.35x to 1.4x faster than Go's `hash/crc32.ChecksumIEEE` on the
same machine for these sizes.
