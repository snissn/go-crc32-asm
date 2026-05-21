package crc32asm

import (
	"bytes"
	"hash/crc32"
	"testing"
)

var benchSink uint32

func TestChecksumIEEE(t *testing.T) {
	for n := 0; n <= 1<<20; n = nextSize(n) {
		data := makeInput(n)
		got := ChecksumIEEE(data)
		want := crc32.ChecksumIEEE(data)
		if got != want {
			t.Fatalf("len=%d got=%08x want=%08x", n, got, want)
		}
	}
}

func TestChecksumIEEEEdges(t *testing.T) {
	for _, n := range []int{
		191, 192, 193,
		575, 576, 577,
		767, 768, 769,
		1023, 1024, 1025,
		64 << 10, 256 << 10, 512 << 10, 1 << 20,
	} {
		data := makeInput(n + 15)
		for offset := 0; offset < 16; offset++ {
			view := data[offset : offset+n]
			got := ChecksumIEEE(view)
			want := crc32.ChecksumIEEE(view)
			if got != want {
				t.Fatalf("len=%d offset=%d got=%08x want=%08x", n, offset, got, want)
			}
		}
	}
}

func TestCombineIEEE(t *testing.T) {
	for _, split := range []int{0, 1, 2, 3, 7, 8, 31, 64, 1024, 65536} {
		data := makeInput(split + 12345)
		left := crc32.ChecksumIEEE(data[:split])
		right := crc32.ChecksumIEEE(data[split:])
		got := combineIEEE(left, right, int64(len(data)-split))
		want := crc32.ChecksumIEEE(data)
		if got != want {
			t.Fatalf("split=%d got=%08x want=%08x", split, got, want)
		}
		got = combineIEEECached(left, right, int64(len(data)-split))
		if got != want {
			t.Fatalf("cached split=%d got=%08x want=%08x", split, got, want)
		}
	}
}

func BenchmarkChecksumIEEE(b *testing.B) {
	for _, size := range []struct {
		name string
		n    int
	}{
		{name: "64KiB", n: 64 << 10},
		{name: "256KiB", n: 256 << 10},
		{name: "512KiB", n: 512 << 10},
		{name: "1MiB", n: 1 << 20},
	} {
		data := makeInput(size.n)
		b.Run(size.name+"/asm", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				benchSink ^= ChecksumIEEE(data)
			}
		})
		b.Run(size.name+"/stdlib", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				benchSink ^= crc32.ChecksumIEEE(data)
			}
		})
	}
}

func makeInput(n int) []byte {
	var buf bytes.Buffer
	x := uint64(0x9e3779b97f4a7c15)
	for buf.Len() < n {
		x ^= x << 7
		x ^= x >> 9
		x ^= x << 8
		buf.WriteByte(byte(x))
	}
	return buf.Bytes()
}

func nextSize(n int) int {
	switch {
	case n == 0:
		return 1
	case n < 64:
		return n + 1
	case n < 1024:
		return n + 31
	case n < 64<<10:
		return n + 4093
	default:
		return n + 65521
	}
}
