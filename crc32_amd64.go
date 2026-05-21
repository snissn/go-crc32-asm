//go:build amd64

package crc32asm

import "golang.org/x/sys/cpu"

const pclmul8Threshold = 128

func crc32IEEEPCLMUL8(crc uint32, p []byte) uint32
func crc32IEEEVPCLMUL256(crc uint32, p []byte) uint32
func crc32IEEEVPCLMUL512(crc uint32, p []byte) uint32

func checksumIEEE(data []byte) uint32 {
	if !cpu.X86.HasPCLMULQDQ || !cpu.X86.HasSSE41 || len(data) < pclmul8Threshold {
		return checksumIEEEFallback(data)
	}

	stripeLen := 128
	useVPCLMUL256 := cpu.X86.HasAVX2 && cpu.X86.HasAVX512VPCLMULQDQ && len(data) >= 256
	useVPCLMUL512 := cpu.X86.HasAVX512F && cpu.X86.HasAVX512BW && cpu.X86.HasAVX512VL && cpu.X86.HasAVX512VPCLMULQDQ && len(data) >= 512
	if useVPCLMUL512 {
		stripeLen = 512
	} else if useVPCLMUL256 {
		stripeLen = 256
	}

	prefixLen := len(data) / stripeLen * stripeLen
	var crc uint32
	if useVPCLMUL512 {
		crc = ^crc32IEEEVPCLMUL512(^uint32(0), data[:prefixLen])
	} else if useVPCLMUL256 {
		crc = ^crc32IEEEVPCLMUL256(^uint32(0), data[:prefixLen])
	} else {
		crc = ^crc32IEEEPCLMUL8(^uint32(0), data[:prefixLen])
	}
	if prefixLen != len(data) {
		crc = combineIEEECached(crc, checksumIEEEFallback(data[prefixLen:]), int64(len(data)-prefixLen))
	}
	return crc
}

func checksumCastagnoli(data []byte) uint32 {
	return checksumCastagnoliFallback(data)
}
