# go-crc32-asm

Experimental CRC-32/IEEE implementation for Go.

The package exports `ChecksumIEEE`, which is bit-compatible with
`hash/crc32.ChecksumIEEE`.

On `arm64`, the experimental path splits large buffers into four adjacent
chunks, computes four CRC streams with ARM CRC32 instructions, and combines the
partial CRCs. This targets the same serial-dependency problem that optimized C
libraries such as `libdeflate` avoid. On ARM64 machines without CRC32 CPU
support, and on all other architectures including Intel and AMD `amd64`, the
package falls back to Go's standard library implementation.

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
