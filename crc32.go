package crc32asm

import (
	"hash/crc32"
	"sync"
)

const ieeePolynomial = 0xedb88320

// ChecksumIEEE returns the CRC-32 checksum of data using the IEEE polynomial.
func ChecksumIEEE(data []byte) uint32 {
	return checksumIEEE(data)
}

func checksumIEEEFallback(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}

var zeroAppendOps sync.Map

func combineIEEECached(crc1, crc2 uint32, len2 int64) uint32 {
	if len2 <= 0 {
		return crc1
	}
	op := zeroAppendOperator(len2)
	return gf2MatrixTimes(&op, crc1) ^ crc2
}

func zeroAppendOperator(len2 int64) [32]uint32 {
	if cached, ok := zeroAppendOps.Load(len2); ok {
		return cached.([32]uint32)
	}

	var op [32]uint32
	for n := 0; n < 32; n++ {
		op[n] = combineIEEE(1<<n, 0, len2)
	}
	actual, _ := zeroAppendOps.LoadOrStore(len2, op)
	return actual.([32]uint32)
}

func combineIEEE(crc1, crc2 uint32, len2 int64) uint32 {
	if len2 <= 0 {
		return crc1
	}

	var odd [32]uint32
	var even [32]uint32

	odd[0] = ieeePolynomial
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
