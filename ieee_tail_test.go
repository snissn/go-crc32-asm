package crc32asm

import (
	"fmt"
	"hash/crc32"
	"testing"
)

// Exercise every possible accelerated-stripe remainder, including unaligned
// input addresses, through the public checksum and continuation entry points.
func TestIEEETailContinuation(t *testing.T) {
	lengths := make([]int, 0, 1600)
	for n := 0; n <= 1536; n++ {
		lengths = append(lengths, n)
	}
	for _, base := range []int{64 << 10, 1 << 20} {
		for _, tail := range []int{0, 1, 17, 127, 255, 511} {
			lengths = append(lengths, base+tail)
		}
	}
	for _, n := range lengths {
		data := makeInput(n + 15)
		for _, offset := range []int{0, 1, 15} {
			view := data[offset : offset+n]
			want := crc32.ChecksumIEEE(view)
			if got := ChecksumIEEE(view); got != want {
				t.Fatalf("ChecksumIEEE len=%d offset=%d got=%08x want=%08x", n, offset, got, want)
			}
			if got := Checksum(view, IEEETable); got != want {
				t.Fatalf("Checksum len=%d offset=%d got=%08x want=%08x", n, offset, got, want)
			}
			for _, seed := range []uint32{0, 0xdeadbeef} {
				wantSeeded := crc32.Update(seed, crc32.IEEETable, view)
				if got := Update(seed, IEEETable, view); got != wantSeeded {
					t.Fatalf("Update len=%d offset=%d seed=%08x got=%08x want=%08x", n, offset, seed, got, wantSeeded)
				}
			}
		}
	}
}

func BenchmarkChecksumIEEETails(b *testing.B) {
	for _, n := range []int{64, 128, 139, 255, 256, 257, 511, 512, 513, 65536, 65553, 1048576, 1048593} {
		b.Run(fmt.Sprintf("%dB", n), func(b *testing.B) {
			data := makeInput(n)
			b.SetBytes(int64(n))
			b.ReportAllocs()
			b.ResetTimer()
			var sum uint32
			for i := 0; i < b.N; i++ {
				sum ^= ChecksumIEEE(data)
			}
			benchSink = sum
		})
	}
}
