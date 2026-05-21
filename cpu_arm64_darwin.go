//go:build arm64 && darwin

package crc32asm

func hasARM64CRC32() bool {
	return true
}

func hasARM64PMULL() bool {
	return true
}

func hasARM64SHA3() bool {
	return true
}
