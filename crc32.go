package crc32asm

import (
	"encoding/binary"
	"errors"
	"hash"
	stdcrc32 "hash/crc32"
	"sync"
)

// The size of a CRC-32 checksum in bytes.
const Size = stdcrc32.Size

// Predefined polynomials.
const (
	IEEE       = stdcrc32.IEEE
	Castagnoli = stdcrc32.Castagnoli
	Koopman    = stdcrc32.Koopman
)

const ieeePolynomial = IEEE
const castagnoliPolynomial = Castagnoli

// Table is a 256-word table representing the polynomial for efficient
// processing.
type Table = stdcrc32.Table

// IEEETable is the table for the IEEE polynomial.
var IEEETable = stdcrc32.IEEETable

var castagnoliTable = stdcrc32.MakeTable(Castagnoli)

// MakeTable returns a Table constructed from the specified polynomial.
func MakeTable(poly uint32) *Table {
	switch poly {
	case IEEE:
		return IEEETable
	case Castagnoli:
		return castagnoliTable
	default:
		return stdcrc32.MakeTable(poly)
	}
}

// ChecksumIEEE returns the CRC-32 checksum of data using the IEEE polynomial.
func ChecksumIEEE(data []byte) uint32 {
	return updateIEEE(0, data)
}

// ChecksumCastagnoli returns the CRC-32C checksum of data using Castagnoli's
// polynomial.
func ChecksumCastagnoli(data []byte) uint32 {
	return updateCastagnoli(0, data)
}

// Checksum returns the CRC-32 checksum of data using the polynomial represented
// by tab.
func Checksum(data []byte, tab *Table) uint32 {
	return Update(0, tab, data)
}

// Update returns the result of adding the bytes in p to the crc.
func Update(crc uint32, tab *Table, p []byte) uint32 {
	switch tab {
	case IEEETable:
		return updateIEEE(crc, p)
	case castagnoliTable:
		return updateCastagnoli(crc, p)
	default:
		return stdcrc32.Update(crc, tab, p)
	}
}

// New creates a new hash.Hash32 computing the CRC-32 checksum using the
// polynomial represented by tab.
func New(tab *Table) hash.Hash32 {
	return &digest{tab: tab}
}

// NewIEEE creates a new hash.Hash32 computing the CRC-32 checksum using the
// IEEE polynomial.
func NewIEEE() hash.Hash32 {
	return New(IEEETable)
}

type digest struct {
	crc uint32
	tab *Table
}

func (d *digest) Size() int {
	return Size
}

func (d *digest) BlockSize() int {
	return 1
}

func (d *digest) Reset() {
	d.crc = 0
}

func (d *digest) Write(p []byte) (int, error) {
	d.crc = Update(d.crc, d.tab, p)
	return len(p), nil
}

func (d *digest) Sum32() uint32 {
	return d.crc
}

func (d *digest) Sum(in []byte) []byte {
	s := d.Sum32()
	return append(in, byte(s>>24), byte(s>>16), byte(s>>8), byte(s))
}

func (d *digest) AppendBinary(b []byte) ([]byte, error) {
	b = append(b, "crc\x01"...)
	b = binary.BigEndian.AppendUint32(b, tableSum(d.tab))
	b = binary.BigEndian.AppendUint32(b, d.crc)
	return b, nil
}

func (d *digest) MarshalBinary() ([]byte, error) {
	return d.AppendBinary(make([]byte, 0, len("crc\x01")+4+4))
}

func (d *digest) UnmarshalBinary(b []byte) error {
	if len(b) < len("crc\x01") || string(b[:len("crc\x01")]) != "crc\x01" {
		return errors.New("hash/crc32: invalid hash state identifier")
	}
	if len(b) != len("crc\x01")+4+4 {
		return errors.New("hash/crc32: invalid hash state size")
	}
	if tableSum(d.tab) != binary.BigEndian.Uint32(b[4:8]) {
		return errors.New("hash/crc32: tables do not match")
	}
	d.crc = binary.BigEndian.Uint32(b[8:12])
	return nil
}

func tableSum(t *Table) uint32 {
	var a [1024]byte
	b := a[:0]
	if t != nil {
		for _, x := range t {
			b = binary.BigEndian.AppendUint32(b, x)
		}
	}
	return ChecksumIEEE(b)
}

func checksumIEEEFallback(data []byte) uint32 {
	return stdcrc32.ChecksumIEEE(data)
}

func checksumCastagnoliFallback(data []byte) uint32 {
	return stdcrc32.Checksum(data, castagnoliTable)
}

func updateIEEEFallback(crc uint32, p []byte) uint32 {
	return stdcrc32.Update(crc, IEEETable, p)
}

func updateCastagnoliFallback(crc uint32, p []byte) uint32 {
	return stdcrc32.Update(crc, castagnoliTable, p)
}

func updateIEEE(crc uint32, p []byte) uint32 {
	if crc == 0 {
		return checksumIEEE(p)
	}
	return updateIEEEFast(crc, p)
}

func updateCastagnoli(crc uint32, p []byte) uint32 {
	if crc == 0 {
		return checksumCastagnoli(p)
	}
	return updateCastagnoliFast(crc, p)
}

var zeroAppendOps sync.Map
var castagnoliZeroAppendOps sync.Map

func combineIEEECached(crc1, crc2 uint32, len2 int64) uint32 {
	return combineCached(&zeroAppendOps, ieeePolynomial, crc1, crc2, len2)
}

func combineCastagnoliCached(crc1, crc2 uint32, len2 int64) uint32 {
	return combineCached(&castagnoliZeroAppendOps, castagnoliPolynomial, crc1, crc2, len2)
}

func combineCached(cache *sync.Map, poly uint32, crc1, crc2 uint32, len2 int64) uint32 {
	if len2 <= 0 {
		return crc1
	}
	op := zeroAppendOperator(cache, poly, len2)
	return gf2MatrixTimes(&op, crc1) ^ crc2
}

func zeroAppendOperator(cache *sync.Map, poly uint32, len2 int64) [32]uint32 {
	if cached, ok := cache.Load(len2); ok {
		if op, ok := cached.([32]uint32); ok {
			return op
		}
	}

	var op [32]uint32
	for n := 0; n < 32; n++ {
		op[n] = combineCRC(poly, 1<<n, 0, len2)
	}
	actual, _ := cache.LoadOrStore(len2, op)
	if result, ok := actual.([32]uint32); ok {
		return result
	}
	return op
}

func combineIEEE(crc1, crc2 uint32, len2 int64) uint32 {
	return combineCRC(ieeePolynomial, crc1, crc2, len2)
}

func combineCRC(poly, crc1, crc2 uint32, len2 int64) uint32 {
	if len2 <= 0 {
		return crc1
	}

	var odd [32]uint32
	var even [32]uint32

	odd[0] = poly
	row := uint32(1)
	for n := 1; n < 32; n++ {
		odd[n] = row
		row <<= 1
	}

	gf2MatrixSquare(&even, &odd)
	gf2MatrixSquare(&odd, &even)

	for {
		gf2MatrixSquare(&even, &odd)
		if len2&1 != 0 {
			crc1 = gf2MatrixTimes(&even, crc1)
		}
		len2 >>= 1
		if len2 == 0 {
			break
		}

		gf2MatrixSquare(&odd, &even)
		if len2&1 != 0 {
			crc1 = gf2MatrixTimes(&odd, crc1)
		}
		len2 >>= 1
		if len2 == 0 {
			break
		}
	}

	return crc1 ^ crc2
}

func gf2MatrixTimes(mat *[32]uint32, vec uint32) uint32 {
	var sum uint32
	i := 0
	for vec != 0 {
		if vec&1 != 0 {
			sum ^= mat[i]
		}
		vec >>= 1
		i++
	}
	return sum
}

func gf2MatrixSquare(square, mat *[32]uint32) {
	for n := 0; n < 32; n++ {
		square[n] = gf2MatrixTimes(mat, mat[n])
	}
}
