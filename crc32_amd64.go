//go:build amd64

package crc32asm

import "golang.org/x/sys/cpu"

const pclmul8Threshold = 128

func crc32IEEEPCLMUL8(crc uint32, p []byte) uint32
func crc32IEEEVPCLMUL256(crc uint32, p []byte) uint32

func checksumIEEE(data []byte) uint32 {
	if !cpu.X86.HasPCLMULQDQ || !cpu.X86.HasSSE41 || len(data) < pclmul8Threshold {
		return checksumIEEEFallback(data)
	}

	stripeLen := 128
	useVPCLMUL := cpu.X86.HasAVX2 && cpu.X86.HasAVX512VPCLMULQDQ && len(data) >= 256
	if useVPCLMUL {
		stripeLen = 256
	}

	prefixLen := len(data) / stripeLen * stripeLen
	var crc uint32
	if useVPCLMUL {
		crc = ^crc32IEEEVPCLMUL256(^uint32(0), data[:prefixLen])
	} else {
		crc = ^crc32IEEEPCLMUL8(^uint32(0), data[:prefixLen])
	}
	if prefixLen != len(data) {
		crc = combineIEEECached(crc, checksumIEEEFallback(data[prefixLen:]), int64(len(data)-prefixLen))
	}
	return crc
}
