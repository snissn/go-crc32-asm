//go:build arm64 && !darwin

package crc32asm

import "golang.org/x/sys/cpu"

func hasARM64CRC32() bool {
	return cpu.ARM64.HasCRC32
}

func hasARM64PMULL() bool {
	return cpu.ARM64.HasPMULL
}

func hasARM64SHA3() bool {
	return cpu.ARM64.HasSHA3
}
