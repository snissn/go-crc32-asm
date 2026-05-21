package crc32asm

import (
	"hash/crc32"
	"testing"

	farm "github.com/dgryski/go-farm"
	"github.com/zeebo/xxh3"
)

var (
	crc32cTable      = crc32.MakeTable(crc32.Castagnoli)
	tournamentSink32 uint32
	tournamentSink64 uint64
)

type tournamentSize struct {
	name string
	n    int
}

var tournamentSizes = []tournamentSize{
	{name: "64KiB_ClickHouseMinCompressBlock", n: 64 << 10},
	{name: "256KiB_Middle", n: 256 << 10},
	{name: "512KiB_WideRowsOneMark", n: 512 << 10},
	{name: "1MiB_ClickHouseMaxCompressBlock", n: 1 << 20},
}

func BenchmarkGomapHashTournamentSlice(b *testing.B) {
	for _, size := range tournamentSizes {
		data := makeInput(size.n)

		b.Run(size.name+"/CRC32_IEEE_AsmPackage", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				tournamentSink32 ^= ChecksumIEEE(data)
			}
		})

		b.Run(size.name+"/CRC32_IEEE_Stdlib", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				tournamentSink32 ^= crc32.ChecksumIEEE(data)
			}
		})

		b.Run(size.name+"/CRC32C_Castagnoli_TreeDB", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				tournamentSink32 ^= crc32.Checksum(data, crc32cTable)
			}
		})

		b.Run(size.name+"/FarmHash64", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				tournamentSink64 ^= farm.Hash64(data)
			}
		})

		b.Run(size.name+"/XXH3_64", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				tournamentSink64 ^= xxh3.Hash(data)
			}
		})
	}
}
