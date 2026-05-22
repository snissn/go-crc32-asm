//go:build amd64

package crc32asm

import "golang.org/x/sys/cpu"

const pclmul8Threshold = 128
const castagnoli4WayThreshold = 128 << 10

func crc32IEEEPCLMUL8(crc uint32, p []byte) uint32
func crc32IEEEVPCLMUL256(crc uint32, p []byte) uint32
func crc32IEEEVPCLMUL512(crc uint32, p []byte) uint32
func crc32Castagnoli4Way(p []byte, chunkLen uintptr) (uint32, uint32, uint32, uint32)

func useIEEEFallback(crc uint32, n int) bool {
	return !cpu.X86.HasPCLMULQDQ || !cpu.X86.HasSSE41 || n < pclmul8Threshold
}

func useCastagnoliFallback(crc uint32, n int) bool {
	return !cpu.X86.HasSSE42 || n < castagnoli4WayThreshold
}

func checksumIEEE(data []byte) uint32 {
	if !cpu.X86.HasPCLMULQDQ || !cpu.X86.HasSSE41 || len(data) < pclmul8Threshold {
		return checksumIEEEFallback(data)
	}

	stripeLen := 128
	useVPCLMUL256 := cpu.X86.HasAVX2 && cpu.X86.HasAVX512F && cpu.X86.HasAVX512VL && cpu.X86.HasAVX512VPCLMULQDQ && len(data) >= 256
	useVPCLMUL512 := cpu.X86.HasAVX512F && cpu.X86.HasAVX512BW && cpu.X86.HasAVX512VL && cpu.X86.HasAVX512VPCLMULQDQ && len(data) >= 512
	if useVPCLMUL512 {
		stripeLen = 512
	} else if useVPCLMUL256 {
		stripeLen = 256
	}

	prefixLen := len(data) / stripeLen * stripeLen
	if prefixLen == 0 {
		return checksumIEEEFallback(data)
	}
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
	if !cpu.X86.HasSSE42 || len(data) < castagnoli4WayThreshold {
		return checksumCastagnoliFallback(data)
	}
	chunkLen := len(data) / 4
	chunkLen &^= 7
	if chunkLen == 0 {
		return checksumCastagnoliFallback(data)
	}

	covered := chunkLen * 4
	c0, c1, c2, c3 := crc32Castagnoli4Way(data[:covered], uintptr(chunkLen))
	crc := combineCastagnoliCached(c0, c1, int64(chunkLen))
	crc = combineCastagnoliCached(crc, c2, int64(chunkLen))
	crc = combineCastagnoliCached(crc, c3, int64(chunkLen))
	if covered != len(data) {
		crc = updateCastagnoliFallback(crc, data[covered:])
	}
	return crc
}

func updateIEEEFast(crc uint32, p []byte) uint32 {
	if !cpu.X86.HasPCLMULQDQ || !cpu.X86.HasSSE41 || len(p) < pclmul8Threshold {
		return updateIEEEFallback(crc, p)
	}

	stripeLen := 128
	useVPCLMUL256 := cpu.X86.HasAVX2 && cpu.X86.HasAVX512F && cpu.X86.HasAVX512VL && cpu.X86.HasAVX512VPCLMULQDQ && len(p) >= 256
	useVPCLMUL512 := cpu.X86.HasAVX512F && cpu.X86.HasAVX512BW && cpu.X86.HasAVX512VL && cpu.X86.HasAVX512VPCLMULQDQ && len(p) >= 512
	if useVPCLMUL512 {
		stripeLen = 512
	} else if useVPCLMUL256 {
		stripeLen = 256
	}

	prefixLen := len(p) / stripeLen * stripeLen
	if prefixLen == 0 {
		return updateIEEEFallback(crc, p)
	}
	state := ^crc
	if useVPCLMUL512 {
		state = crc32IEEEVPCLMUL512(state, p[:prefixLen])
	} else if useVPCLMUL256 {
		state = crc32IEEEVPCLMUL256(state, p[:prefixLen])
	} else {
		state = crc32IEEEPCLMUL8(state, p[:prefixLen])
	}

	crc = ^state
	if prefixLen != len(p) {
		crc = updateIEEEFallback(crc, p[prefixLen:])
	}
	return crc
}

func updateCastagnoliFast(crc uint32, p []byte) uint32 {
	if !cpu.X86.HasSSE42 || len(p) < castagnoli4WayThreshold {
		return updateCastagnoliFallback(crc, p)
	}
	return combineCastagnoliCached(crc, checksumCastagnoli(p), int64(len(p)))
}
