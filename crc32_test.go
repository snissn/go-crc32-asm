package crc32asm

import (
	"bytes"
	"encoding"
	"hash"
	"hash/crc32"
	"testing"
)

var benchSink uint32

func requireBinaryMarshaler(t *testing.T, name string, h hash.Hash32) encoding.BinaryMarshaler {
	t.Helper()
	marshaler, ok := h.(encoding.BinaryMarshaler)
	if !ok {
		t.Fatalf("%s hash does not implement encoding.BinaryMarshaler", name)
	}
	return marshaler
}

func requireBinaryUnmarshaler(t *testing.T, name string, h hash.Hash32) encoding.BinaryUnmarshaler {
	t.Helper()
	unmarshaler, ok := h.(encoding.BinaryUnmarshaler)
	if !ok {
		t.Fatalf("%s hash does not implement encoding.BinaryUnmarshaler", name)
	}
	return unmarshaler
}

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

func TestChecksumCastagnoli(t *testing.T) {
	tab := crc32.MakeTable(crc32.Castagnoli)
	for n := 0; n <= 1<<20; n = nextSize(n) {
		data := makeInput(n)
		got := ChecksumCastagnoli(data)
		want := crc32.Checksum(data, tab)
		if got != want {
			t.Fatalf("len=%d got=%08x want=%08x", n, got, want)
		}
	}
}

func TestStdlibCompatibleConstantsAndTables(t *testing.T) {
	if Size != crc32.Size {
		t.Fatalf("Size=%d want %d", Size, crc32.Size)
	}
	if IEEE != crc32.IEEE {
		t.Fatalf("IEEE=%08x want %08x", IEEE, crc32.IEEE)
	}
	if Castagnoli != crc32.Castagnoli {
		t.Fatalf("Castagnoli=%08x want %08x", Castagnoli, crc32.Castagnoli)
	}
	if Koopman != crc32.Koopman {
		t.Fatalf("Koopman=%08x want %08x", Koopman, crc32.Koopman)
	}
	if IEEETable != MakeTable(IEEE) {
		t.Fatalf("MakeTable(IEEE) did not return IEEETable")
	}
	if MakeTable(Castagnoli) != crc32.MakeTable(crc32.Castagnoli) {
		t.Fatalf("MakeTable(Castagnoli) did not match stdlib Castagnoli table")
	}
}

func TestStdlibCompatibleChecksumAndUpdate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		poly    uint32
		stdTab  *crc32.Table
		fastTab *Table
	}{
		{name: "IEEE", poly: crc32.IEEE, stdTab: crc32.MakeTable(crc32.IEEE), fastTab: MakeTable(IEEE)},
		{name: "Castagnoli", poly: crc32.Castagnoli, stdTab: crc32.MakeTable(crc32.Castagnoli), fastTab: MakeTable(Castagnoli)},
		{name: "Koopman", poly: crc32.Koopman, stdTab: crc32.MakeTable(crc32.Koopman), fastTab: MakeTable(Koopman)},
		{name: "Custom", poly: 0xd5828281, stdTab: crc32.MakeTable(0xd5828281), fastTab: MakeTable(0xd5828281)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for n := 0; n <= 1<<20; n = nextSize(n) {
				data := makeInput(n)
				got := Checksum(data, tc.fastTab)
				want := crc32.Checksum(data, tc.stdTab)
				if got != want {
					t.Fatalf("Checksum len=%d got=%08x want=%08x", n, got, want)
				}

				for _, split := range []int{0, 1, 2, 3, 7, 8, 31, 64, 1024, 65536} {
					if split > len(data) {
						continue
					}
					got = Update(Update(0, tc.fastTab, data[:split]), tc.fastTab, data[split:])
					want = crc32.Update(crc32.Update(0, tc.stdTab, data[:split]), tc.stdTab, data[split:])
					if got != want {
						t.Fatalf("Update len=%d split=%d got=%08x want=%08x", n, split, got, want)
					}
				}
			}
		})
	}
}

func TestStdlibCompatibleHash32(t *testing.T) {
	for _, tc := range []struct {
		name    string
		stdHash hash.Hash32
		fast    hash.Hash32
	}{
		{name: "IEEE", stdHash: crc32.NewIEEE(), fast: NewIEEE()},
		{name: "Castagnoli", stdHash: crc32.New(crc32.MakeTable(crc32.Castagnoli)), fast: New(MakeTable(Castagnoli))},
		{name: "Koopman", stdHash: crc32.New(crc32.MakeTable(crc32.Koopman)), fast: New(MakeTable(Koopman))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := makeInput(1<<20 + 333)
			for _, chunkLen := range []int{1, 3, 7, 64, 4096, 65536} {
				tc.stdHash.Reset()
				tc.fast.Reset()
				for off := 0; off < len(data); {
					end := off + chunkLen
					if end > len(data) {
						end = len(data)
					}
					if n, err := tc.stdHash.Write(data[off:end]); n != end-off || err != nil {
						t.Fatalf("stdlib Write n=%d err=%v", n, err)
					}
					if n, err := tc.fast.Write(data[off:end]); n != end-off || err != nil {
						t.Fatalf("fast Write n=%d err=%v", n, err)
					}
					off = end
				}
				if got, want := tc.fast.Sum32(), tc.stdHash.Sum32(); got != want {
					t.Fatalf("chunkLen=%d Sum32 got=%08x want=%08x", chunkLen, got, want)
				}
				if got, want := tc.fast.Sum([]byte("prefix")), tc.stdHash.Sum([]byte("prefix")); !bytes.Equal(got, want) {
					t.Fatalf("chunkLen=%d Sum got=%x want=%x", chunkLen, got, want)
				}
				if tc.fast.Size() != tc.stdHash.Size() {
					t.Fatalf("Size=%d want %d", tc.fast.Size(), tc.stdHash.Size())
				}
				if tc.fast.BlockSize() != tc.stdHash.BlockSize() {
					t.Fatalf("BlockSize=%d want %d", tc.fast.BlockSize(), tc.stdHash.BlockSize())
				}
			}
		})
	}
}

func TestStdlibCompatibleHashBinaryState(t *testing.T) {
	data := makeInput(65536)
	fast := NewIEEE()
	std := crc32.NewIEEE()
	_, _ = fast.Write(data[:12345])
	_, _ = std.Write(data[:12345])

	fastMarshaler := requireBinaryMarshaler(t, "fast", fast)
	stdMarshaler := requireBinaryMarshaler(t, "stdlib", std)
	fastState, err := fastMarshaler.MarshalBinary()
	if err != nil {
		t.Fatalf("fast MarshalBinary: %v", err)
	}
	stdState, err := stdMarshaler.MarshalBinary()
	if err != nil {
		t.Fatalf("stdlib MarshalBinary: %v", err)
	}
	if !bytes.Equal(fastState, stdState) {
		t.Fatalf("MarshalBinary mismatch got=%x want=%x", fastState, stdState)
	}

	restored := NewIEEE()
	if err := requireBinaryUnmarshaler(t, "restored", restored).UnmarshalBinary(fastState); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	_, _ = restored.Write(data[12345:])
	if got, want := restored.Sum32(), crc32.ChecksumIEEE(data); got != want {
		t.Fatalf("restored Sum32 got=%08x want=%08x", got, want)
	}

	cloneHashForTest(t, fast, data[12345:], crc32.ChecksumIEEE(data))
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

func TestCombineCastagnoli(t *testing.T) {
	tab := crc32.MakeTable(crc32.Castagnoli)
	for _, split := range []int{0, 1, 2, 3, 7, 8, 31, 64, 1024, 65536} {
		data := makeInput(split + 12345)
		left := crc32.Checksum(data[:split], tab)
		right := crc32.Checksum(data[split:], tab)
		got := combineCRC(Castagnoli, left, right, int64(len(data)-split))
		want := crc32.Checksum(data, tab)
		if got != want {
			t.Fatalf("split=%d got=%08x want=%08x", split, got, want)
		}
		got = combineCastagnoliCached(left, right, int64(len(data)-split))
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

func BenchmarkChecksumCastagnoli(b *testing.B) {
	stdTab := crc32.MakeTable(crc32.Castagnoli)
	asmTab := MakeTable(Castagnoli)
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
		b.Run(size.name+"/asm_direct", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				benchSink ^= ChecksumCastagnoli(data)
			}
		})
		b.Run(size.name+"/asm_table", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				benchSink ^= Checksum(data, asmTab)
			}
		})
		b.Run(size.name+"/stdlib", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				benchSink ^= crc32.Checksum(data, stdTab)
			}
		})
	}
}

func BenchmarkUpdateNonZero(b *testing.B) {
	stdCastagnoliTab := crc32.MakeTable(crc32.Castagnoli)
	asmCastagnoliTab := MakeTable(Castagnoli)
	prefix := makeInput(4096)
	ieeeSeed := crc32.ChecksumIEEE(prefix)
	castagnoliSeed := crc32.Checksum(prefix, stdCastagnoliTab)

	for _, size := range []struct {
		name string
		n    int
	}{
		{name: "64B", n: 64},
		{name: "4KiB", n: 4 << 10},
		{name: "64KiB", n: 64 << 10},
		{name: "256KiB", n: 256 << 10},
		{name: "512KiB", n: 512 << 10},
		{name: "1MiB", n: 1 << 20},
	} {
		data := makeInput(size.n)
		b.Run(size.name+"/IEEE/asm", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				benchSink ^= Update(ieeeSeed, IEEETable, data)
			}
		})
		b.Run(size.name+"/IEEE/stdlib", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				benchSink ^= crc32.Update(ieeeSeed, crc32.IEEETable, data)
			}
		})
		b.Run(size.name+"/Castagnoli/asm", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				benchSink ^= Update(castagnoliSeed, asmCastagnoliTab, data)
			}
		})
		b.Run(size.name+"/Castagnoli/stdlib", func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				benchSink ^= crc32.Update(castagnoliSeed, stdCastagnoliTab, data)
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
