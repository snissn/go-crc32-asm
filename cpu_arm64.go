//go:build arm64 && !darwin

package crc32asm

import "golang.org/x/sys/cpu"

func hasARM64CRC32() bool {
	return cpu.ARM64.HasCRC32
}
